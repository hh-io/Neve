// Package backup 把账本文件镜像进 iCloud 外的 git 仓库并提交/推送,做本地版本历史
// 与异地(私有远程)灾备。
//
// 为什么要"镜像"而不是让 git 直接管 iCloud 目录:账本数据在快捷指令 App 的 iCloud
// 容器里(见项目 data 软链),属 TCC 重点保护区。launchd 沙箱下的进程对该容器
// 禁止 readdir/chdir——git 无论如何都要 chdir 进工作树,于是直接失败。而服务端进程
// 已获该容器的读权限,可用 os.ReadFile 按绝对路径读出内容;因此这里由服务端读出账本
// 内容写进镜像目录,git 只对镜像(非 iCloud、无 TCC 限制)操作,彻底绕开限制。
package backup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// snapshotTimeout 是一次快照(含 push)的总预算。必须有:git 在 launchd 下拿不到 tty,
// 一旦卡在凭据提示或网络黑洞上就永不返回,而 Snapshot 全程持锁——后续每次触发都会
// 多堆一个 goroutine 阻塞在锁上,备份彻底停摆且无任何信号。
const snapshotTimeout = 2 * time.Minute

// gitTermGrace 是取消时留给 git 响应 SIGINT 自行清理(索引锁等)的时间,超时才强杀。
const gitTermGrace = 5 * time.Second

// Snapshotter 串行地把一组账本文件同步进镜像 git 仓库并提交/推送。
type Snapshotter struct {
	dataDir string // 账本源目录(iCloud 软链指向处),按绝对路径读取
	repoDir string // 镜像 git 工作树 + .git,必须在 iCloud 外
	remote  string // git 远程 URL;为空则只做本地快照不推送
	mu      sync.Mutex
}

// New 创建 Snapshotter。repoDir/remote 由调用方从环境变量解析后传入。
func New(dataDir, repoDir, remote string) *Snapshotter {
	return &Snapshotter{dataDir: dataDir, repoDir: repoDir, remote: remote}
}

// Snapshot 同步 files(相对 dataDir 的账本文件清单)进镜像仓库,仅在有变化时提交,
// 配置了 remote 时再推送。整个过程串行,重复触发天然合并为一次提交(无变化即跳过)。
// 返回的错误由调用方记日志;本地提交已落库时即便推送失败也已有版本快照。
func (s *Snapshotter) Snapshot(files []string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), snapshotTimeout)
	defer cancel()

	if err := s.ensureRepo(ctx); err != nil {
		return fmt.Errorf("初始化镜像仓库失败: %w", err)
	}

	desired, err := s.syncFiles(files)
	if err != nil {
		return err
	}
	if err := s.pruneRemoved(ctx, desired); err != nil {
		return err
	}

	if _, err := s.git(ctx, "add", "-A"); err != nil {
		return fmt.Errorf("git add 失败: %w", err)
	}
	// diff --cached 退出码 1 表示有暂存变化,0 表示无变化;无变化则不产生空提交
	if _, err := s.git(ctx, "diff", "--cached", "--quiet"); err == nil {
		return nil
	}

	msg := fmt.Sprintf("backup(%s): %s", reason, time.Now().Format("2006-01-02 15:04:05 -0700"))
	if _, err := s.git(ctx, "commit", "-q", "-m", msg); err != nil {
		return fmt.Errorf("git commit 失败: %w", err)
	}

	if s.remote != "" {
		if _, err := s.git(ctx, "push", "-q", "origin", "HEAD:main"); err != nil {
			// 本地快照已落库;显式上报推送失败供排查,不静默吞掉。
			// 注意有两类失败不会自愈,必须靠调用方告警出来:凭据失效,以及远程被
			// 别处推过导致的 non-fast-forward(非 force push,只会一直失败)。
			return fmt.Errorf("git push 失败(本地快照已保存): %w", err)
		}
	}
	return nil
}

// ensureRepo 幂等地初始化镜像仓库并对齐 remote 配置。
func (s *Snapshotter) ensureRepo(ctx context.Context) error {
	// 0700:remote URL 若形如 https://<token>@host,token 会明文存进 .git/config,
	// 只能靠目录权限兜住(改用 SSH remote 则凭据留在 key 里,不落这个仓库)
	if err := os.MkdirAll(s.repoDir, 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(s.repoDir, ".git")); os.IsNotExist(err) {
		if _, err := s.git(ctx, "init", "-q", "-b", "main"); err != nil {
			return err
		}
		// 用仓库级身份,不依赖 launchd 下能否读到全局 gitconfig
		if _, err := s.git(ctx, "config", "user.name", "Neve Backup"); err != nil {
			return err
		}
		if _, err := s.git(ctx, "config", "user.email", "neve-backup@localhost"); err != nil {
			return err
		}
	}
	return s.syncRemote(ctx)
}

// syncRemote 让 origin 与配置的 remote 一致(新增/改动/删除)。
func (s *Snapshotter) syncRemote(ctx context.Context) error {
	out, _ := s.git(ctx, "remote", "get-url", "origin")
	current := strings.TrimSpace(out)
	switch {
	case s.remote == "" && current != "":
		_, err := s.git(ctx, "remote", "remove", "origin")
		return err
	case s.remote != "" && current == "":
		_, err := s.git(ctx, "remote", "add", "origin", s.remote)
		return err
	case s.remote != "" && current != s.remote:
		_, err := s.git(ctx, "remote", "set-url", "origin", s.remote)
		return err
	}
	return nil
}

// syncFiles 把每个源文件内容写进镜像,返回实际写入的相对路径集合(供 pruneRemoved 比对)。
// 读不到的文件(如清单里列了但已删)跳过而非报错——删除交给 pruneRemoved 处理。
func (s *Snapshotter) syncFiles(files []string) (map[string]bool, error) {
	desired := make(map[string]bool, len(files))
	for _, rel := range files {
		rel = filepath.Clean(rel)
		// 防御:清单来自 parser 的 SourceFiles / 已知配置名,不应越界;越界一律跳过
		if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dataDir, rel))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("读取账本文件 %s 失败: %w", rel, err)
		}
		dst := filepath.Join(s.repoDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(dst, data, 0o600); err != nil {
			return nil, err
		}
		desired[rel] = true
	}
	return desired, nil
}

// pruneRemoved 删除镜像里已不在源清单中的已跟踪文件,使快照反映源侧的删除。
func (s *Snapshotter) pruneRemoved(ctx context.Context, desired map[string]bool) error {
	// -z 输出 NUL 分隔的原始路径:默认输出会把非 ASCII 文件名 escape 成 "\346\226..." ,
	// 与 desired 的键对不上,中文名账本会被误判成"已删除"
	out, err := s.git(ctx, "ls-files", "-z")
	if err != nil {
		return fmt.Errorf("git ls-files 失败: %w", err)
	}
	for _, rel := range strings.Split(strings.TrimRight(out, "\x00"), "\x00") {
		if rel == "" || desired[rel] {
			continue
		}
		if err := os.Remove(filepath.Join(s.repoDir, rel)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// credInURLRe 匹配 URL 里的 userinfo 段(https://user:token@host 的 "user:token@")。
var credInURLRe = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.\-]*://)[^/@\s]+@`)

// redactCredentials 抹掉文本里 URL 内嵌的凭据。git 报错常把 remote URL 原样打出来,
// 而错误消息最终会进服务端日志——token 不能跟着落地。
func redactCredentials(s string) string {
	return credInURLRe.ReplaceAllString(s, "${1}***@")
}

// git 在 repoDir 内执行 git 命令。cwd 固定为镜像目录(iCloud 外),不触碰受限容器。
func (s *Snapshotter) git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = s.repoDir
	// 非交互是硬要求:launchd 下无 tty,凭据提示或未知 host key 会让 git 无限等待。
	// 宁可快速失败并告警,也不要挂住(见 snapshotTimeout)。
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes",
	)
	// 先 SIGINT 给 git 机会清理 index.lock,超过宽限期再强杀
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = gitTermGrace

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return stdout.String(), fmt.Errorf("%w: %s", err, redactCredentials(strings.TrimSpace(stderr.String())))
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}
