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

# --- Makefile 配置 ---
.DEFAULT_GOAL := all
SHELL := /bin/bash

# .PHONY 声明这些目标不是文件，避免与同名文件冲突
.PHONY: all deps build build-web build-server run dev dev-web dev-server test clean help

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

# 构建前端：生成静态资源到 dist 目录
build-web:
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

# 开发模式：启动前端热重载服务器
dev: dev-web

# 前端开发模式
dev-web:
	@echo "🔧 启动前端开发服务器..."
	@cd $(WEB_DIR) && pnpm run dev

# 后端开发模式 (配合 air 或手动重启)
dev-server:
	@echo "🔧 启动后端开发服务器..."
	@cd $(SERVER_DIR) && NEVE_DATA_DIR=../data go run .

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
	@echo "    make dev         启动前端开发服务器 (热重载)"
	@echo "    make dev-server  启动后端开发服务器"
	@echo ""
	@echo "  辅助命令:"
	@echo "    make test        运行后端单元测试"
	@echo "    make clean       清理所有构建产物"
	@echo "    make help        显示此帮助信息"
	@echo ""
