package api

import (
	"net/http"
	"neve/parser"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

// Server holds the API server state
type Server struct {
	dataDir string
	ledger  *parser.Ledger
	mu      sync.RWMutex
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
	if err := s.Refresh(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	analytics := parser.Analyze(s.ledger)
	c.JSON(http.StatusOK, gin.H{
		"message": "data refreshed",
		"summary": analytics.Summary,
	})
}

// GetDataDir returns absolute data directory path
func GetDataDir(relPath string) string {
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return relPath
	}
	return absPath
}
