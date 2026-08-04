# Neve 数据备份(`server/backup/`)

- **数据备份必须由服务端进程做,不能交给独立 launchd/cron 任务**:数据在快捷指令
  App 的 iCloud 容器(`data` 软链指向处),属 macOS TCC 重点保护区。未获授权的
  launchd 进程对该目录 `readdir`/`chdir` 一律 `Operation not permitted`(连 `git add`
  都因 git 要 chdir 进工作树而失败),`stat` 单文件放行但 `open` 读内容也被拒;而
  **服务端进程已获该容器读权限**。故 `server/backup` 采用镜像法:服务端用 `os.ReadFile`
  读账本内容写进 iCloud 外的镜像 git 工作树,git 只对镜像操作(非 iCloud、无 TCC 限制)。
  备份文件清单取自 `Ledger.SourceFiles`(parser 记录实际打开的 main.bean+include 文件,
  单一真源)+ 已知配置名(debts.json);`triggerBackup` 有护栏——账本为空或
  `SourceFiles` 为空时**跳过**,否则空清单会把镜像里已跟踪的 .bean 全 prune 成删除。
  推送用普通 `git push`(非 force),首推需远程为空库。
- **git 子进程必须非交互 + 带超时**:launchd 下无 tty,凭据提示/未知 host key 会让 git
  永久挂住,而 `Snapshot` 全程持锁——后续每次触发都堆一个 goroutine 在锁上,备份彻底
  停摆且无信号。故统一走 `git()`:`GIT_TERMINAL_PROMPT=0` + `ssh -o BatchMode=yes`,
  `snapshotTimeout` 限总时长(取消先 SIGINT 留 `gitTermGrace` 清理 index.lock)。
- 失败必须**推送告警**(`alertBackupFailure`,`backupAlertInterval` 节流):凭据失效与
  non-fast-forward 不会自愈,只写日志等于静默失效。git 报错会带 remote URL,
  出口统一过 `redactCredentials` 抹掉内嵌 token。
- 每日兜底用**墙上时钟轮询**(`backupTickInterval` + 比对日期)而非 24h 定时器:
  后者随进程重启漂移,且睡眠期间 monotonic 定时器是否推进依平台而异。
