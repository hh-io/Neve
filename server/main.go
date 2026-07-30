package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"neve/ai"
	"neve/api"
	"neve/backup"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var staticFiles embed.FS

// fatalLog 专走 stderr。启动期致命错误与 gin Recovery 的 panic 栈同属"进程活不下去"
// 的信号,都该落进 neve.error.log——那个文件的价值就在于非空即真出事。
var fatalLog = log.New(os.Stderr, "", log.LstdFlags)

func main() {
	// 应用日志改走 stdout,与 gin 的访问日志同一条时间线(launchd 把两条流分别落到
	// neve.log / neve.error.log)。默认的 stderr 会让"错误日志"里塞满启动信息和记账
	// 成功记录,真正的失败反被淹没,而它的轮转阈值还比访问日志更小、裁得更快。
	// 分流后 stderr 只剩 gin Recovery 的 panic 栈:neve.error.log 非空即真出事。
	log.SetOutput(os.Stdout)

	// Get data directory from env or use default
	dataDir := os.Getenv("NEVE_DATA_DIR")
	if dataDir == "" {
		// Default to ./data relative to executable
		execPath, err := os.Executable()
		if err != nil {
			fatalLog.Fatalf("boot: 无法获取可执行文件路径: %v", err)
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
		fatalLog.Fatalf("boot: 无法解析数据目录: %v", err)
	}
	log.Printf("boot: 数据目录 %s", absDataDir)

	// Initialize server
	server := api.NewServer(absDataDir)

	// Load initial data
	if err := server.Refresh(); err != nil {
		log.Printf("boot: 初始账本加载失败: %v", err)
	}

	// 无感记账入口:token 与 AI 配置齐备才启用,否则 /api/inbox 返回 404
	inboxEnabled := false
	if inboxToken := os.Getenv("NEVE_INBOX_TOKEN"); inboxToken != "" {
		aiClient, err := ai.NewClientFromEnv()
		if err != nil {
			log.Printf("boot: inbox 未启用: %v", err)
		} else {
			server.EnableInbox(aiClient, inboxToken, os.Getenv("NEVE_BARK_URL"))
			inboxEnabled = true
			log.Printf("boot: inbox 入口已启用 (provider=%s)", aiClient.Provider())
		}
	}

	// 公网入口自检:配了 tunnel 域名且 inbox 已启用才有意义(自检对象就是 inbox 入口)。
	// tunnel 挂掉时本机一切正常,不主动探测就只表现为"没有请求进来"(见 api/health.go)。
	if host := os.Getenv("NEVE_TUNNEL_HOSTNAME"); host != "" && inboxEnabled {
		healthURL := "https://" + host + "/api/inbox"
		server.EnableHealthCheck(healthURL)
		server.StartHealthChecker()
		log.Printf("boot: 公网入口自检已启用 (%s)", healthURL)
	}

	// 数据备份:配置了远程 URL 才启用。服务端把账本镜像进 iCloud 外的 git 仓库并推送,
	// 绕开 launchd 沙箱对 iCloud 容器的 readdir/chdir 限制(见 server/backup 包注释)。
	if remote := os.Getenv("NEVE_BACKUP_REMOTE"); remote != "" {
		repoDir := os.Getenv("NEVE_BACKUP_DIR")
		if repoDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				log.Printf("boot: 备份未启用,无法定位 HOME: %v", err)
				home = ""
			}
			if home != "" {
				repoDir = filepath.Join(home, "Library", "Application Support", "Neve", "data-backup")
			}
		}
		if repoDir != "" {
			server.EnableBackup(backup.New(absDataDir, repoDir, remote))
			server.StartBackupScheduler()
			log.Printf("boot: 数据备份已启用 (repo=%s)", repoDir)
		}
	}

	// Set up Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	// 时间格式对齐标准库 log 的 LstdFlags:两类日志现在同流同文件,格式一致才能按时序 grep
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s %s %s %d %s\n",
			param.TimeStamp.Format("2006/01/02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
		)
	}))

	// API routes
	server.SetupRoutes(r)

	// Serve static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		fatalLog.Fatalf("boot: 静态资源初始化失败: %v", err)
	}

	// Read index.html content for SPA fallback
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		log.Printf("boot: 静态资源缺少 index.html,前端未构建")
		indexHTML = []byte("<html><body><h1>Neve</h1><p>Frontend not built. Run: make build</p></body></html>")
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 未注册的 /api/* 一律 JSON 404,不走 SPA 兜底:
		// 避免经隧道访问 API 路径时把前端页面壳回给公网
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": api.ErrNotFound})
			return
		}

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

	// 不用 r.Run:它走 http.Server 默认值,一个超时都没有。停止读取响应的客户端
	// (手机切后台、隧道断链)会让写操作无限期挂住,连带占住上游的资源。
	// 只是不设 ReadTimeout——它覆盖整个请求体,而 /api/inbox 要在慢速上行里传十几 MB 图片;
	// 防的是慢速握手,交给 ReadHeaderTimeout,体积上限已由 MaxBytesReader 把住。
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 20 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("boot: 服务已启动 http://localhost:%s", port)
	if err := srv.ListenAndServe(); err != nil {
		fatalLog.Fatalf("boot: 服务启动失败: %v", err)
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
	case strings.HasSuffix(fileName, ".webmanifest"):
		return "application/manifest+json; charset=utf-8"
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
