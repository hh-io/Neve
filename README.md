# Neve

个人/家庭记账系统 - 基于 Beancount 格式,采用 Go + Vue 3 构建。

## 特性

- 📊 美观的 Dashboard 界面 (Apple Minimal + Glassmorphism)
- 📱 响应式设计,支持手机和桌面
- 💰 净资产、收支统计、分类分析
- 📈 月度趋势图表
- 🔄 手动刷新数据
- 🔒 Cloudflare Access 认证

## 架构

```
iOS 快捷指令 → iCloud Drive → macOS Go 后端 → Vue 3 Dashboard
                              ↓
                        Cloudflared Tunnel → 公网
```

## 快速开始

```bash
# 安装依赖并构建
make build

# 运行
./neve
# 或指定数据目录
NEVE_DATA_DIR=./data ./neve

# 访问 http://localhost:8080
```

## 开发

```bash
# 前端开发模式 (hot reload)
make dev

# 后端开发模式 (需要先构建前端)
make dev-server
```

## 文件结构

```
├── data/                 # Beancount 数据 (iCloud 同步)
│   ├── main.bean        # 入口文件
│   ├── inbox.bean       # iOS 输入缓冲区
│   ├── balance.bean     # 余额断言
│   └── 2025.bean        # 年度归档
├── server/              # Go 后端
│   ├── main.go
│   ├── parser/          # Bean 解析器
│   └── api/             # REST API
├── web/                 # Vue 3 前端
└── Makefile
```

## API

- `GET /api/summary` - 获取财务摘要
- `GET /api/analytics` - 获取完整分析数据
- `GET /api/transactions` - 获取交易列表
- `GET /api/accounts` - 获取账户列表
- `POST /api/refresh` - 刷新数据

## 配置

环境变量:

- `NEVE_DATA_DIR` - 数据目录路径 (默认: ./data)
- `NEVE_PORT` - 服务端口 (默认: 8080)
