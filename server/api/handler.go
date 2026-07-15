package api

import (
	"encoding/json"
	"io"
	"net/http"
	"neve/parser"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Server holds the API server state
type Server struct {
	dataDir string
	// mu 保护 analytics/lastRefresh;analytics 在 Refresh 时一次算好,
	// 各端点读同一份缓存,避免每请求重算导致的时间口径不一致
	mu          sync.RWMutex
	analytics   *parser.Analytics
	lastRefresh time.Time
	// budgets.json 的读写不经过账本,单独用一把锁
	budgetMu sync.Mutex
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

	s.mu.Lock()
	s.analytics = analytics
	s.lastRefresh = time.Now()
	s.mu.Unlock()

	return nil
}

// SetupRoutes sets up the API routes
func (s *Server) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/analytics", s.handleAnalytics)
		api.POST("/refresh", s.handleRefresh)
		api.GET("/budgets", s.handleGetBudgets)
		api.POST("/budgets", s.handleSaveBudgets)
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

	// 解析中的脏数据是软失败(体现在 parseIssues),只有账本完全无法加载才算刷新失败
	if err := s.Refresh(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": NewAPIError("REFRESH_FAILED", err.Error()),
		})
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"message":     "data refreshed",
		"summary":     s.analytics.Summary,
		"issueCount":  len(s.analytics.ParseIssues),
		"parseIssues": s.analytics.ParseIssues,
	})
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

	c.JSON(http.StatusOK, gin.H{"message": "saved"})
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
