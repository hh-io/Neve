package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"neve/api"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	// Get data directory from env or use default
	dataDir := os.Getenv("NEVE_DATA_DIR")
	if dataDir == "" {
		// Default to ./data relative to executable
		execPath, err := os.Executable()
		if err != nil {
			log.Fatal("Failed to get executable path:", err)
		}
		dataDir = filepath.Join(filepath.Dir(execPath), "data")

		// If that doesn't exist, try current directory
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			dataDir = "./data"
		}
	}

	// Convert to absolute path
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		log.Fatal("Failed to resolve data directory:", err)
	}
	log.Printf("Using data directory: %s", absDataDir)

	// Initialize server
	server := api.NewServer(absDataDir)

	// Load initial data
	if err := server.Refresh(); err != nil {
		log.Printf("Warning: Failed to load initial data: %v", err)
	}

	// Set up Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	}))
	r.Use(corsMiddleware())

	// API routes
	server.SetupRoutes(r)

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal("Failed to setup static files:", err)
	}

	// Read index.html content for SPA fallback
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		log.Printf("Warning: index.html not found in static files")
		indexHTML = []byte("<html><body><h1>Neve</h1><p>Frontend not built. Run: make build</p></body></html>")
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Serve index.html for root
		if path == "/" {
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
			return
		}

		// Try to serve static file
		fileName := path[1:] // Remove leading slash
		file, err := staticFS.Open(fileName)
		if err != nil {
			// File not found, serve index.html for SPA routing
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
			return
		}
		file.Close()

		// Serve the file with proper content type
		data, err := fs.ReadFile(staticFS, fileName)
		if err != nil {
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
			return
		}

		contentType := getContentType(fileName)
		c.Data(http.StatusOK, contentType, data)
	})

	// Get port from env or use default
	port := os.Getenv("NEVE_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting Neve server on http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func getContentType(fileName string) string {
	switch {
	case strings.HasSuffix(fileName, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(fileName, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(fileName, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(fileName, ".json"):
		return "application/json; charset=utf-8"
	case strings.HasSuffix(fileName, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(fileName, ".png"):
		return "image/png"
	case strings.HasSuffix(fileName, ".jpg"), strings.HasSuffix(fileName, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(fileName, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(fileName, ".woff"):
		return "font/woff"
	case strings.HasSuffix(fileName, ".woff2"):
		return "font/woff2"
	default:
		return "application/octet-stream"
	}
}
