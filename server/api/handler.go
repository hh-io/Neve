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
	dataDir     string
	ledger      *parser.Ledger
	mu          sync.RWMutex
	lastRefresh time.Time
}

// NewServer creates a new API server
func NewServer(dataDir string) *Server {
	return &Server{
		dataDir: dataDir,
	}
}

// Refresh reloads the ledger data
func (s *Server) Refresh() error {
	p := parser.NewParser(s.dataDir)
	ledger, err := p.Parse()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.ledger = ledger
	s.mu.Unlock()

	return nil
}

// SetupRoutes sets up the API routes
func (s *Server) SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/summary", s.handleSummary)
		api.GET("/transactions", s.handleTransactions)
		api.GET("/analytics", s.handleAnalytics)
		api.GET("/accounts", s.handleAccounts)
		api.POST("/refresh", s.handleRefresh)
		api.GET("/budgets", s.handleGetBudgets)
		api.POST("/budgets", s.handleSaveBudgets)
	}
}

func (s *Server) handleSummary(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "data not loaded"})
		return
	}

	analytics := parser.Analyze(s.ledger)
	c.JSON(http.StatusOK, analytics.Summary)
}

func (s *Server) handleTransactions(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "data not loaded"})
		return
	}

	// Return last 100 transactions
	count := 100
	if len(s.ledger.Transactions) < count {
		count = len(s.ledger.Transactions)
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": s.ledger.Transactions[:count],
		"total":        len(s.ledger.Transactions),
	})
}

func (s *Server) handleAnalytics(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "data not loaded"})
		return
	}

	analytics := parser.Analyze(s.ledger)
	c.JSON(http.StatusOK, analytics)
}

func (s *Server) handleAccounts(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.ledger == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "data not loaded"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accounts": s.ledger.Accounts,
		"total":    len(s.ledger.Accounts),
	})
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

	if err := s.Refresh(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": NewAPIError("REFRESH_FAILED", err.Error()),
		})
		return
	}

	s.mu.Lock()
	s.lastRefresh = time.Now()
	s.mu.Unlock()

	analytics := parser.Analyze(s.ledger)
	c.JSON(http.StatusOK, gin.H{
		"message": "data refreshed",
		"summary": analytics.Summary,
	})
}

func (s *Server) handleGetBudgets(c *gin.Context) {
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

	budgetFile := filepath.Join(s.dataDir, "budgets.json")
	if err := os.WriteFile(budgetFile, body, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

// GetDataDir returns absolute data directory path
func GetDataDir(relPath string) string {
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return relPath
	}
	return absPath
}
