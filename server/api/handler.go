package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"neve/ai"
	"neve/backup"
	"neve/parser"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Server holds the API server state
type Server struct {
	dataDir string
	// mu 保护 analytics/ledger/lastRefresh;analytics 在 Refresh 时一次算好,
	// 各端点读同一份缓存,避免每请求重算导致的时间口径不一致
	mu        sync.RWMutex
	analytics *parser.Analytics
	// ledger 随 analytics 一起整体替换,供 /api/debts 只读现算;
	// 拿到旧指针继续算也是一份一致快照,无需额外同步
	ledger      *parser.Ledger
	lastRefresh time.Time
	// budgets.json 的读写不经过账本,单独用一把锁
	budgetMu sync.Mutex
	// debts.json 同理
	debtMu sync.Mutex
	// refreshMu 串行化 /api/refresh:限流检查与 Refresh 之间存在 TOCTOU,
	// 并发请求会同时通过检查并重复解析,靠这把锁 + 拿锁后二次检查兜住
	refreshMu sync.Mutex

	// 无感记账入口(见 inbox.go),EnableInbox 配置后 /api/inbox 才生效
	aiClient     ai.Client
	inboxToken   string
	barkURL      string
	inboxMu      sync.Mutex   // 串行化 inbox.bean 追加
	inboxPending atomic.Int32 // 在途异步识别任务数

	// 数据备份(见 server/backup),EnableBackup 配置后各写入路径成功即异步快照
	backup *backup.Snapshotter
	// 备份失败告警的节流状态;triggerBackup 的 goroutine 可能并发进来
	backupAlertMu   sync.Mutex
	lastBackupAlert time.Time
}

// NewServer creates a new API server
func NewServer(dataDir string) *Server {
	return &Server{
		dataDir: dataDir,
	}
}

// Refresh reloads the ledger data and rebuilds the analytics cache
func (s *Server) Refresh() error {
	p := parser.NewParser(s.dataDir)
	ledger, err := p.Parse()
	if err != nil {
		return err
	}
	analytics := parser.Analyze(ledger)
	// 长期负债清单在 debts.json 而非账本,叠加在 Analyze 之后;
	// 先取 debtMu 读完配置再取 s.mu 写缓存,两把锁不嵌套
	analytics.ApplyLongTermLiabilities(s.longTermAccounts())

	s.mu.Lock()
	s.analytics = analytics
	s.ledger = ledger
	s.lastRefresh = time.Now()
	s.mu.Unlock()

	return nil
}

// EnableBackup 开启数据备份;不调用则所有备份触发点均空操作。
func (s *Server) EnableBackup(snap *backup.Snapshotter) {
	s.backup = snap
}

const (
	// dailyBackupHour 每日兜底快照的时刻(服务器本地时区,TZ 由部署钉死)
	dailyBackupHour = 4
	// backupTickInterval 轮询间隔。这里不用 24h 定时器:它会随进程重启漂移,且机器
	// 睡眠时 monotonic 定时器是否推进依平台而异。改为轮询 + 墙上时钟判定"今天跑过没有",
	// 挂起醒来最迟一个 tick 内补上。取 1h 是因为兜底快照只为捕获手动改动
	// (手动编辑不经写入路径),晚一小时无影响——不需要更细的精度。
	backupTickInterval = time.Hour
	// backupAlertInterval 备份失败告警的最小间隔。失败常是持续性的(凭据失效、
	// non-fast-forward),每次触发都推会把通知刷爆,但完全不推就是备份静默失效。
	backupAlertInterval = 24 * time.Hour
)

// StartBackupScheduler 启动即快照一次(捕获上次运行至今的改动),并每日兜底一次。
// 每日先 Refresh 以纳入手动新增的 include 文件,再快照(文件内容始终读磁盘实时值)。
func (s *Server) StartBackupScheduler() {
	s.triggerBackup("startup")
	go func() {
		t := time.NewTicker(backupTickInterval)
		defer t.Stop()
		// 启动时已快照过,当天不再补一次
		lastDaily := time.Now().Format("2006-01-02")
		for range t.C {
			now := time.Now()
			today := now.Format("2006-01-02")
			if now.Hour() < dailyBackupHour || today == lastDaily {
				continue
			}
			lastDaily = today
			if err := s.Refresh(); err != nil {
				log.Printf("backup: 每日刷新失败: %v", err)
			}
			s.triggerBackup("daily")
		}
	}()
}

// triggerBackup 异步做一次备份;未启用则空操作。账本写入路径成功后调用。
// 护栏:仅在有有效账本(至少含 main.bean)时才快照——否则空/残缺的文件清单会把
// 镜像里已跟踪的 .bean 全当作删除 prune 掉,一次瞬时解析失败就可能清空快照。
func (s *Server) triggerBackup(reason string) {
	if s.backup == nil {
		return
	}
	s.mu.RLock()
	ledger := s.ledger
	s.mu.RUnlock()
	if ledger == nil || len(ledger.SourceFiles) == 0 {
		return
	}
	files := make([]string, 0, len(ledger.SourceFiles)+2)
	files = append(files, ledger.SourceFiles...)
	// 配置文件不经账本 include,按已知名补入(Snapshot 会跳过不存在的)
	files = append(files, "budgets.json", "debts.json")
	go func() {
		if err := s.backup.Snapshot(files, reason); err != nil {
			log.Printf("backup: %v", err)
			s.alertBackupFailure(err)
		}
	}()
}

// alertBackupFailure 把备份失败推到手机上。日志在这台机器上没人看,而备份最典型的
// 死法就是悄悄停掉几个月——凭据失效、远程被别处推过,都不会自愈。
func (s *Server) alertBackupFailure(err error) {
	s.backupAlertMu.Lock()
	if time.Since(s.lastBackupAlert) < backupAlertInterval {
		s.backupAlertMu.Unlock()
		return
	}
	s.lastBackupAlert = time.Now()
	s.backupAlertMu.Unlock()

	s.notify("Neve 备份失败", truncateRunes(err.Error(), 300))
}

// SetupRoutes sets up the API routes
func (s *Server) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/analytics", s.handleAnalytics)
		api.POST("/refresh", s.handleRefresh)
		api.GET("/budgets", s.handleGetBudgets)
		api.POST("/budgets", s.handleSaveBudgets)
		api.GET("/debts", s.handleGetDebts)
		api.POST("/debts", s.handleSaveDebts)
		api.POST("/inbox", s.handleInbox)
	}
}

func (s *Server) handleAnalytics(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.analytics == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "data not loaded"})
		return
	}

	c.JSON(http.StatusOK, s.analytics)
}

func (s *Server) handleRefresh(c *gin.Context) {
	// Rate limit: minimum 5 seconds between refreshes
	s.mu.RLock()
	sinceLastRefresh := time.Since(s.lastRefresh)
	s.mu.RUnlock()

	if sinceLastRefresh < 5*time.Second {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      ErrRateLimited,
			"retryAfter": (5*time.Second - sinceLastRefresh).Seconds(),
		})
		return
	}

	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	respondOK := func() {
		s.mu.RLock()
		defer s.mu.RUnlock()
		c.JSON(http.StatusOK, gin.H{
			"message":     "data refreshed",
			"summary":     s.analytics.Summary,
			"issueCount":  len(s.analytics.ParseIssues),
			"parseIssues": s.analytics.ParseIssues,
		})
	}

	// 二次检查:排队等锁期间别人已刷新过,直接返回缓存结果
	s.mu.RLock()
	refreshedWhileWaiting := time.Since(s.lastRefresh) < 5*time.Second
	s.mu.RUnlock()
	if refreshedWhileWaiting {
		respondOK()
		return
	}

	// 解析中的脏数据是软失败(体现在 parseIssues),只有账本完全无法加载才算刷新失败
	if err := s.Refresh(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": NewAPIError("REFRESH_FAILED", err.Error()),
		})
		return
	}

	respondOK()
}

func (s *Server) handleGetBudgets(c *gin.Context) {
	s.budgetMu.Lock()
	defer s.budgetMu.Unlock()

	budgetFile := filepath.Join(s.dataDir, "budgets.json")
	data, err := os.ReadFile(budgetFile)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{}) // 尚未设过预算
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": NewAPIError("BUDGETS_READ_FAILED", "budgets.json 读取失败: "+err.Error()),
		})
		return
	}

	var budgets map[string]float64
	if err := json.Unmarshal(data, &budgets); err != nil {
		// 与 debts 同策略:损坏时不回空对象,否则前端当成"没设过预算",一存就覆盖原文件
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": NewAPIError("BUDGETS_CORRUPT", "budgets.json 无法解析,请修复后重试: "+err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, budgets)
}

func (s *Server) handleSaveBudgets(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	// Validate JSON
	var budgets map[string]float64
	if err := json.Unmarshal(body, &budgets); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	s.budgetMu.Lock()
	defer s.budgetMu.Unlock()

	budgetFile := filepath.Join(s.dataDir, "budgets.json")
	quarantineCorrupt(budgetFile, func(b []byte) error {
		var old map[string]float64
		return json.Unmarshal(b, &old)
	})
	if err := atomicWriteFile(budgetFile, body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	s.triggerBackup("budgets")
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

func emptyDebtsConfig() *parser.DebtsConfig {
	return &parser.DebtsConfig{
		LongTermAccounts: []string{},
		Revolving:        map[string]parser.RevolvingConfig{},
		Installments:     []parser.InstallmentConfig{},
	}
}

// loadDebtsConfig 读取 debts.json。文件不存在返回空配置(首次使用);内容损坏返回错误。
//
// 损坏时**不能**降级成空配置:配置只有这一份真源,前端拿到空的会显示"未配置",
// 用户随手补一条再保存就把原文件整份覆盖掉,原配置再也拿不回来。
func (s *Server) loadDebtsConfig() (*parser.DebtsConfig, error) {
	s.debtMu.Lock()
	defer s.debtMu.Unlock()

	data, err := os.ReadFile(filepath.Join(s.dataDir, "debts.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return emptyDebtsConfig(), nil
		}
		return nil, err
	}
	cfg := emptyDebtsConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err // 解析失败时 cfg 可能被写了一半,不能拿来用
	}
	if cfg.Revolving == nil {
		cfg.Revolving = map[string]parser.RevolvingConfig{}
	}
	if cfg.Installments == nil {
		cfg.Installments = []parser.InstallmentConfig{}
	}
	if cfg.LongTermAccounts == nil {
		cfg.LongTermAccounts = []string{}
	}
	// 老文件的额度类条目没有 installments 字段,回显给前端补成 [] 而非 null
	cfg.Normalize()
	return cfg, nil
}

// longTermAccounts 取长期负债清单供 Refresh 叠加分层。配置读不出来时降级为空清单:
// 分层字段退回全量口径(Analyze 已兜底),不至于让账本刷新整个失败。
func (s *Server) longTermAccounts() []string {
	cfg, err := s.loadDebtsConfig()
	if err != nil {
		log.Printf("config: debts.json 无法解析,净资产分层降级为全量口径: %v", err)
		return nil
	}
	return cfg.LongTermAccounts
}

// quarantineCorrupt 在覆盖前把无法解析的旧配置改名留档。
//
// 走到写盘这一步,说明用户是基于界面上看到的内容做的编辑;若界面上那份是读取失败后的
// 降级结果,这次写入就是一次静默的整份覆盖。配置文件没有第二份真源,留档是最后一道防线。
func quarantineCorrupt(path string, parse func([]byte) error) {
	data, err := os.ReadFile(path)
	if err != nil || parse(data) == nil {
		return
	}
	dst := path + ".corrupt-" + time.Now().Format("20060102-150405")
	if err := os.Rename(path, dst); err != nil {
		log.Printf("config: 留档损坏的 %s 失败: %v", filepath.Base(path), err)
		return
	}
	log.Printf("config: %s 无法解析,覆盖前已留档为 %s", filepath.Base(path), filepath.Base(dst))
}

func (s *Server) handleGetDebts(c *gin.Context) {
	s.mu.RLock()
	ledger := s.ledger
	s.mu.RUnlock()
	if ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "data not loaded"})
		return
	}

	cfg, err := s.loadDebtsConfig()
	if err != nil {
		// 不返回空配置:前端会当成"没配过",一次保存就把原文件覆盖
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": NewAPIError("DEBTS_CONFIG_CORRUPT", "debts.json 无法解析,请修复后重试: "+err.Error()),
		})
		return
	}
	// 每次现算:O(交易数) 一次遍历,倒计时永远新鲜,配置变更也无需 /api/refresh
	c.JSON(http.StatusOK, gin.H{
		"config": cfg,
		"report": parser.ComputeDebts(ledger, cfg, time.Now()),
	})
}

func (s *Server) handleSaveDebts(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	var cfg parser.DebtsConfig
	if err := json.Unmarshal(body, &cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": NewAPIError("INVALID_DEBTS_CONFIG", "JSON 解析失败: "+err.Error()),
		})
		return
	}
	if cfg.Revolving == nil {
		cfg.Revolving = map[string]parser.RevolvingConfig{}
	}
	if cfg.Installments == nil {
		cfg.Installments = []parser.InstallmentConfig{}
	}
	if cfg.LongTermAccounts == nil {
		cfg.LongTermAccounts = []string{}
	}
	if errs := cfg.Validate(); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   NewAPIError("INVALID_DEBTS_CONFIG", "配置校验未通过"),
			"details": errs,
		})
		return
	}

	// 落盘规范化后的结构而非原始 body:schedule 排好序,字段顺序稳定
	cfg.Normalize()
	data, err := json.MarshalIndent(&cfg, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode"})
		return
	}

	debtsPath := filepath.Join(s.dataDir, "debts.json")
	s.debtMu.Lock()
	quarantineCorrupt(debtsPath, func(b []byte) error {
		return json.Unmarshal(b, &parser.DebtsConfig{})
	})
	err = atomicWriteFile(debtsPath, data)
	s.debtMu.Unlock()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	s.triggerBackup("debts")

	// 长期负债清单变了会让缓存 analytics 的分层字段过期,就地重算(幂等,一次账户遍历);
	// 账本本身没变,不必重跑 Analyze
	s.mu.Lock()
	if s.analytics != nil {
		s.analytics.ApplyLongTermLiabilities(cfg.LongTermAccounts)
	}
	s.mu.Unlock()

	// 保存后立刻重算,前端一次往返拿到新结果;账本尚未加载时 report 为 null
	s.mu.RLock()
	ledger := s.ledger
	s.mu.RUnlock()
	resp := gin.H{"config": &cfg, "report": nil}
	if ledger != nil {
		resp["report"] = parser.ComputeDebts(ledger, &cfg, time.Now())
	}
	c.JSON(http.StatusOK, resp)
}

// atomicWriteFile 先写同目录临时文件再 rename,避免写入中断损坏目标文件
// (budgets.json / debts.json 共用)
func atomicWriteFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".neve-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // rename 成功后为空操作

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// GetDataDir returns absolute data directory path
func GetDataDir(relPath string) string {
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return relPath
	}
	return absPath
}
