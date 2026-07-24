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
	analytics.ApplyLongTermLiabilities(s.loadDebtsConfig().LongTermAccounts)

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

// StartBackupScheduler 启动即快照一次(捕获上次运行至今的改动),并每日兜底一次。
// 每日先 Refresh 以纳入手动新增的 include 文件,再快照(文件内容始终读磁盘实时值)。
func (s *Server) StartBackupScheduler() {
	s.triggerBackup("startup")
	go func() {
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for range t.C {
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
		}
	}()
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
		// Return empty object if file doesn't exist
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	var budgets map[string]float64
	if err := json.Unmarshal(data, &budgets); err != nil {
		c.JSON(http.StatusOK, gin.H{})
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

	if err := atomicWriteFile(filepath.Join(s.dataDir, "budgets.json"), body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	s.triggerBackup("budgets")
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// loadDebtsConfig 读取 debts.json;文件不存在或 JSON 损坏时降级为空配置(与 budgets 同策略),
// 不阻塞页面展示。
func (s *Server) loadDebtsConfig() *parser.DebtsConfig {
	empty := func() *parser.DebtsConfig {
		return &parser.DebtsConfig{
			LongTermAccounts: []string{},
			Revolving:        map[string]parser.RevolvingConfig{},
			Installments:     []parser.InstallmentConfig{},
		}
	}

	s.debtMu.Lock()
	defer s.debtMu.Unlock()

	data, err := os.ReadFile(filepath.Join(s.dataDir, "debts.json"))
	if err != nil {
		return empty()
	}
	cfg := empty()
	if err := json.Unmarshal(data, cfg); err != nil {
		return empty() // 解析失败时 cfg 可能被写了一半,丢弃重建
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
	return cfg
}

func (s *Server) handleGetDebts(c *gin.Context) {
	s.mu.RLock()
	ledger := s.ledger
	s.mu.RUnlock()
	if ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "data not loaded"})
		return
	}

	cfg := s.loadDebtsConfig()
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

	s.debtMu.Lock()
	err = atomicWriteFile(filepath.Join(s.dataDir, "debts.json"), data)
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

// atomicWriteFile 先写同目录临时文件再 rename,避免写入中断损坏 budgets.json
func atomicWriteFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".budgets-*.tmp")
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
