package backup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitCount 返回镜像仓库的提交数(无提交时为 0)
func commitCount(t *testing.T, repoDir string) int {
	t.Helper()
	out, err := exec.Command("git", "-C", repoDir, "rev-list", "--count", "HEAD").Output()
	if err != nil {
		return 0 // 尚无提交时 HEAD 不存在
	}
	n := 0
	for _, ch := range strings.TrimSpace(string(out)) {
		n = n*10 + int(ch-'0')
	}
	return n
}

// trackedFiles 返回镜像仓库已跟踪文件集合
func trackedFiles(t *testing.T, repoDir string) map[string]bool {
	t.Helper()
	out, err := exec.Command("git", "-C", repoDir, "ls-files").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	set := map[string]bool{}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if f != "" {
			set[f] = true
		}
	}
	return set
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotLifecycle(t *testing.T) {
	dataDir := t.TempDir()
	repoDir := filepath.Join(t.TempDir(), "mirror")

	write(t, filepath.Join(dataDir, "main.bean"), "include \"2025.bean\"\n")
	write(t, filepath.Join(dataDir, "2025.bean"), "2025-01-01 * \"a\" \"b\"\n")
	write(t, filepath.Join(dataDir, "budgets.json"), "{}\n")

	// debts.json 不存在,应被安全跳过
	files := []string{"main.bean", "2025.bean", "budgets.json", "debts.json"}
	s := New(dataDir, repoDir, "") // 无远程,只测本地镜像与提交

	// 1) 首次快照:初始化 + 提交,镜像内容与源一致
	if err := s.Snapshot(files, "test"); err != nil {
		t.Fatalf("首次 Snapshot: %v", err)
	}
	if got := commitCount(t, repoDir); got != 1 {
		t.Fatalf("首次提交后 commit 数 = %d,期望 1", got)
	}
	tracked := trackedFiles(t, repoDir)
	for _, f := range []string{"main.bean", "2025.bean", "budgets.json"} {
		if !tracked[f] {
			t.Errorf("镜像未跟踪 %s(tracked=%v)", f, tracked)
		}
	}
	if tracked["debts.json"] {
		t.Errorf("不存在的 debts.json 不应被跟踪")
	}
	if b, _ := os.ReadFile(filepath.Join(repoDir, "2025.bean")); string(b) != "2025-01-01 * \"a\" \"b\"\n" {
		t.Errorf("镜像 2025.bean 内容不一致: %q", b)
	}

	// 2) 无变化再快照:不产生空提交
	if err := s.Snapshot(files, "test"); err != nil {
		t.Fatalf("无变化 Snapshot: %v", err)
	}
	if got := commitCount(t, repoDir); got != 1 {
		t.Fatalf("无变化后 commit 数 = %d,期望仍为 1", got)
	}

	// 3) 修改源文件:再提交,镜像跟随更新
	write(t, filepath.Join(dataDir, "2025.bean"), "2025-01-01 * \"a\" \"b\"\n2025-01-02 * \"c\" \"d\"\n")
	if err := s.Snapshot(files, "test"); err != nil {
		t.Fatalf("修改后 Snapshot: %v", err)
	}
	if got := commitCount(t, repoDir); got != 2 {
		t.Fatalf("修改后 commit 数 = %d,期望 2", got)
	}

	// 4) 源侧删除文件并移出清单:镜像 prune 掉,反映删除
	if err := os.Remove(filepath.Join(dataDir, "2025.bean")); err != nil {
		t.Fatal(err)
	}
	if err := s.Snapshot([]string{"main.bean", "budgets.json"}, "test"); err != nil {
		t.Fatalf("删除后 Snapshot: %v", err)
	}
	if got := commitCount(t, repoDir); got != 3 {
		t.Fatalf("删除后 commit 数 = %d,期望 3", got)
	}
	if trackedFiles(t, repoDir)["2025.bean"] {
		t.Errorf("2025.bean 已从源删除,镜像仍在跟踪")
	}
	if _, err := os.Stat(filepath.Join(repoDir, "2025.bean")); !os.IsNotExist(err) {
		t.Errorf("2025.bean 应已从镜像工作树移除")
	}
}
