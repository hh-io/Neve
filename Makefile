# ==============================================================================
# Neve Makefile
# ==============================================================================
# 项目构建、开发、清理自动化脚本
# 使用方法: make help
# ==============================================================================

# --- 变量定义 ---
BINARY_NAME := neve
WEB_DIR     := web
SERVER_DIR  := server

# Go 构建参数：-s 去除符号表, -w 去除 DWARF 调试信息, 减小二进制体积
GO_LDFLAGS := -s -w

# --- 部署配置 (deploy/ 下的模板渲染后安装到系统位置) ---
DEPLOY_DIR     := deploy
LAUNCH_PLIST   := $(HOME)/Library/LaunchAgents/com.neve.server.plist
NEWSYSLOG_CONF := /etc/newsyslog.d/neve.conf
CLOUDFLARED_CONF := $(HOME)/.cloudflared/config.yml
TUNNEL_PLIST     := $(HOME)/Library/LaunchAgents/com.cloudflared.tunnel.plist
CLOUDFLARED_BIN  ?= $(shell command -v cloudflared)

# 密钥/私有配置来自 deploy/local.env (不入库,见 deploy/local.env.example),缺省渲染为空
-include $(DEPLOY_DIR)/local.env
NEVE_INBOX_TOKEN     ?=
NEVE_AI_PROVIDER     ?=
NEVE_AI_API_KEY      ?=
NEVE_AI_MODEL        ?=
NEVE_BARK_URL        ?=
NEVE_TUNNEL_ID       ?=
NEVE_TUNNEL_HOSTNAME ?=

# 渲染模板占位符: @NEVE_ROOT@ / @HOME@ / @USER@ + local.env 中的私有配置
RENDER := sed -e 's|@NEVE_ROOT@|$(CURDIR)|g' -e 's|@HOME@|$(HOME)|g' -e 's|@USER@|$(USER)|g' \
	-e 's|@NEVE_INBOX_TOKEN@|$(NEVE_INBOX_TOKEN)|g' \
	-e 's|@NEVE_AI_PROVIDER@|$(NEVE_AI_PROVIDER)|g' \
	-e 's|@NEVE_AI_API_KEY@|$(NEVE_AI_API_KEY)|g' \
	-e 's|@NEVE_AI_MODEL@|$(NEVE_AI_MODEL)|g' \
	-e 's|@NEVE_BARK_URL@|$(NEVE_BARK_URL)|g' \
	-e 's|@NEVE_TUNNEL_ID@|$(NEVE_TUNNEL_ID)|g' \
	-e 's|@NEVE_TUNNEL_HOSTNAME@|$(NEVE_TUNNEL_HOSTNAME)|g' \
	-e 's|@CLOUDFLARED_BIN@|$(CLOUDFLARED_BIN)|g'

# --- Makefile 配置 ---
.DEFAULT_GOAL := all
SHELL := /bin/bash

# .PHONY 声明这些目标不是文件，避免与同名文件冲突
.PHONY: all deps build build-web build-server run dev dev-web dev-server test clean help \
        install-service install-logrotate install-tunnel

# ==============================================================================
# 核心构建任务
# ==============================================================================

# 默认目标：安装依赖并构建所有
all: deps build
	@echo "✅ 构建完成！运行 './$(BINARY_NAME)' 或 'make run' 启动服务"

# 安装所有依赖 (前端 + 后端)
deps:
	@echo "📦 安装前端依赖..."
	@cd $(WEB_DIR) && pnpm install --frozen-lockfile
	@echo "📦 整理后端依赖..."
	@cd $(SERVER_DIR) && go mod tidy

# 总构建任务：先构建前端，再构建后端
build: build-web build-server

# 构建前端：先过 ESLint 与类型检查,再生成静态资源到 dist 目录
build-web:
	@echo "🔍 前端 lint 与类型检查..."
	@cd $(WEB_DIR) && pnpm run lint && pnpm run typecheck
	@echo "🔨 构建前端..."
	@cd $(WEB_DIR) && pnpm run build

# 后端单元测试 (解析器 + 统计,含数据竞争检测)
test:
	@echo "🧪 运行后端测试..."
	@cd $(SERVER_DIR) && go test -race ./...

# 构建后端：编译 Go 二进制文件 (依赖前端构建产物用于 embed)
build-server: build-web test
	@echo "🔍 检查代码质量..."
	@cd $(SERVER_DIR) && go fmt ./... && go vet ./...
	@echo "🔨 构建后端二进制..."
	@cd $(SERVER_DIR) && go build -ldflags "$(GO_LDFLAGS)" -o ../$(BINARY_NAME) .
	@echo "✌️ 构建完成,二进制大小: $$(du -h $(BINARY_NAME) | cut -f1)"

# ==============================================================================
# 运行与开发
# ==============================================================================

# 运行生产级二进制文件 (自动触发构建)
run: build
	@echo "🚀 启动 $(BINARY_NAME)..."
	@./$(BINARY_NAME)

# 开发模式：同时启动前后端开发服务器
dev:
	@$(MAKE) -j2 dev-server dev-web

# 前端开发模式
dev-web:
	@echo "🔧 启动前端开发服务器..."
	@cd $(WEB_DIR) && pnpm run dev

# 后端开发模式 (配合 air 或手动重启)
dev-server:
	@echo "🔧 启动后端开发服务器..."
	@cd $(SERVER_DIR) && NEVE_DATA_DIR=../data go run .

# ==============================================================================
# 部署 (macOS launchd + newsyslog)
# ==============================================================================

# 渲染并安装 launchd 服务配置 (只写文件不启动;服务在运行需重载才生效)
install-service:
	@$(RENDER) $(DEPLOY_DIR)/com.neve.server.plist.in > $(LAUNCH_PLIST)
	@chmod 600 $(LAUNCH_PLIST)
	@echo "✅ 已写入 $(LAUNCH_PLIST)"
	@echo "   启动: launchctl bootstrap gui/$$(id -u) $(LAUNCH_PLIST)"
	@echo "   重载: launchctl bootout gui/$$(id -u)/com.neve.server && launchctl bootstrap gui/$$(id -u) $(LAUNCH_PLIST)"

# 渲染并安装 Cloudflare Tunnel 配置 (仅暴露 /api/inbox;需先 cloudflared tunnel create 并在 local.env 填好 ID/域名)
install-tunnel:
	@test -n "$(NEVE_TUNNEL_ID)" || { echo "❌ 请在 $(DEPLOY_DIR)/local.env 设置 NEVE_TUNNEL_ID / NEVE_TUNNEL_HOSTNAME"; exit 1; }
	@test -n "$(CLOUDFLARED_BIN)" || { echo "❌ 未找到 cloudflared,先 brew install cloudflared"; exit 1; }
	@mkdir -p $(HOME)/.cloudflared
	@$(RENDER) $(DEPLOY_DIR)/cloudflared-config.yml.in > $(CLOUDFLARED_CONF)
	@$(RENDER) $(DEPLOY_DIR)/com.cloudflared.tunnel.plist.in > $(TUNNEL_PLIST)
	@echo "✅ 已写入 $(CLOUDFLARED_CONF) 与 $(TUNNEL_PLIST)"
	@echo "   DNS 绑定: cloudflared tunnel route dns $(NEVE_TUNNEL_ID) $(NEVE_TUNNEL_HOSTNAME)"
	@echo "   启动: launchctl bootstrap gui/$$(id -u) $(TUNNEL_PLIST)"
	@echo "   重载: launchctl bootout gui/$$(id -u)/com.cloudflared.tunnel && launchctl bootstrap gui/$$(id -u) $(TUNNEL_PLIST)"

# 渲染并安装日志轮转配置 (需要 sudo)
install-logrotate:
	@$(RENDER) $(DEPLOY_DIR)/neve.newsyslog.conf.in | sudo tee $(NEWSYSLOG_CONF) > /dev/null
	@echo "✅ 已写入 $(NEWSYSLOG_CONF),newsyslog 试运行验证:"
	@sudo newsyslog -nv | grep neve

# ==============================================================================
# 清理与辅助
# ==============================================================================

# 清理所有构建产物
clean:
	@echo "🧹 清理构建产物..."
	@rm -rf $(WEB_DIR)/dist
	@rm -rf $(SERVER_DIR)/static
	@rm -f $(BINARY_NAME)
	@echo "✅ 清理完成"

# 显示帮助信息
help:
	@echo ""
	@echo "Neve Makefile 使用指南"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "  构建命令:"
	@echo "    make all         安装依赖并构建所有 (默认)"
	@echo "    make deps        仅安装前端和后端依赖"
	@echo "    make build       构建前端和后端"
	@echo "    make build-web   仅构建前端"
	@echo "    make build-server 仅构建后端 (会自动先构建前端)"
	@echo ""
	@echo "  运行命令:"
	@echo "    make run         构建并运行生产二进制"
	@echo "    make dev         同时启动前后端开发服务器"
	@echo "    make dev-web     仅启动前端开发服务器 (热重载)"
	@echo "    make dev-server  仅启动后端开发服务器"
	@echo ""
	@echo "  部署命令:"
	@echo "    make install-service    渲染并安装 launchd 服务配置 (含 deploy/local.env 密钥)"
	@echo "    make install-logrotate  渲染并安装日志轮转配置 (需 sudo)"
	@echo "    make install-tunnel     渲染并安装 Cloudflare Tunnel 配置 (仅暴露 /api/inbox)"
	@echo ""
	@echo "  辅助命令:"
	@echo "    make test        运行后端单元测试"
	@echo "    make clean       清理所有构建产物"
	@echo "    make help        显示此帮助信息"
	@echo ""
	@echo "  前端质量 (在 $(WEB_DIR)/ 下用 pnpm 运行):"
	@echo "    pnpm run lint       ESLint 检查"
	@echo "    pnpm run typecheck  vue-tsc 类型检查"
	@echo "    pnpm run format     Prettier 格式化"
	@echo ""
