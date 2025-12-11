.PHONY: all build dev clean deps run

# 构建全部
all: deps build

# 安装依赖
deps:
	cd web && pnpm install

# 构建前端并嵌入后端
build: deps
	cd web && pnpm run build
	cd server && go mod tidy && go build -o ../neve .

# 前端开发模式 (热重载)
dev:
	cd web && pnpm run dev

# 运行生产服务器
run: build
	./neve

# 清理构建产物
clean:
	rm -rf server/static
	rm -f neve

# 后端开发模式
dev-server:
	cd server && go run .
