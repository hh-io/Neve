<p align="center">
  <img src="https://img.shields.io/badge/vue-3.5-4FC08D?style=flat-square&logo=vue.js" alt="Vue 3">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/vite-5.x-646CFF?style=flat-square&logo=vite" alt="Vite">
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License">
</p>

<h1 align="center">❄️ Neve</h1>

<p align="center">
  <strong>基于 Beancount 复式记账的现代个人财务管理系统</strong>
</p>

<p align="center">
  一键启动 • 玻璃拟态设计 • 数据即文件
</p>

---

## ✨ 项目概述

**Neve** 是一个轻量级的个人/家庭财务可视化系统，围绕 [Beancount](https://beancount.github.io/) 纯文本账本格式构建。

### 🎯 解决的问题

- 📱 **iOS 快捷指令无缝记账** → 写入 iCloud Drive 的 `.bean` 文件
- 📊 **优雅的数据可视化** → Apple 风格 Dashboard，随时掌握财务状况
- 🔒 **数据所有权** → 纯文本格式，永远不会被平台锁定

### 💎 核心亮点

| 特性 | 描述 |
|------|------|
| 🎨 **Apple 设计语言** | 玻璃拟态 + 精心调校的亮/暗双主题 |
| 📈 **丰富的图表分析** | 收支趋势、分类占比、周消费分布、商户排行 |
| 🚀 **单文件部署** | 前端嵌入二进制，一个 `./neve` 即刻运行 |
| ☁️ **iCloud 原生集成** | 与 iOS 快捷指令配合，实现移动端无应用记账 |

---

## 🛠️ 技术栈概览

```
┌─────────────────────────────────────────────────────────┐
│                       Frontend                          │
│   Vue 3  •  Vite  •  ECharts  •  Glassmorphism CSS     │
├─────────────────────────────────────────────────────────┤
│                       Backend                           │
│   Go  •  Gin  •  embed (静态文件嵌入)                   │
├─────────────────────────────────────────────────────────┤
│                        Data                             │
│   Beancount (.bean 纯文本复式记账格式)                  │
└─────────────────────────────────────────────────────────┘
```

---

## 🚀 快速开始

### 📋 前置环境 (Prerequisites)

| 工具 | 版本要求 | 安装指南 |
|------|----------|----------|
| **Go** | >= 1.21 | [golang.org/dl](https://golang.org/dl/) |
| **Node.js** | >= 18 | [nodejs.org](https://nodejs.org/) |
| **pnpm** | >= 8 | `npm install -g pnpm` |

### ⚙️ 环境变量配置

创建 `.env` 文件或直接 export 环境变量：

```bash
# .env.example
NEVE_DATA_DIR=/path/to/your/beancount/data   # Beancount 数据目录 (默认: ./data)
NEVE_PORT=8080                                # HTTP 服务端口 (默认: 8080)
```

### 📦 安装依赖

```bash
# 克隆仓库
git clone https://github.com/your-username/neve.git
cd neve

# 安装所有依赖（前端 + 后端）
make deps
```

### ▶️ 开发模式

```bash
# 终端 1：启动后端服务
make dev-server

# 终端 2：启动前端热重载 (http://localhost:5173)
make dev
```

### 📦 生产构建

```bash
# 一键构建：前端 → 后端（嵌入静态文件）
make build

# 运行生产版本
./neve
```

访问 http://localhost:8080 即可使用 🎉

---

## 📂 项目架构

```
Neve/
├── 📄 neve                      # 编译后的单文件可执行程序
├── 📄 Makefile                  # 构建自动化脚本
│
├── 📁 data/                     # Beancount 数据目录 (可 iCloud 同步)
│   ├── main.bean               # 入口文件 (账户定义 + include)
│   ├── inbox.bean              # 待整理流水 (iOS 快捷指令写入)
│   └── 2025.bean               # 年度归档交易
│
├── 📁 server/                   # 🔧 Go 后端
│   ├── main.go                 # 入口 + HTTP 服务 + 静态文件 embed
│   ├── api/
│   │   └── handler.go          # REST API 处理器
│   └── parser/
│       ├── parser.go           # Beancount 文件解析器
│       └── analytics.go        # 数据分析 (净资产/分类/趋势)
│
└── 📁 web/                      # 🎨 Vue 3 前端
    └── src/
        ├── App.vue             # 主应用组件
        ├── components/
        │   ├── tabs/           # 页面标签组件
        │   │   ├── OverviewTab.vue      # 概览
        │   │   ├── SpendingTab.vue      # 支出分析
        │   │   ├── TrendsTab.vue        # 趋势图表
        │   │   ├── TransactionsTab.vue  # 交易列表
        │   │   └── AccountsTab.vue      # 账户管理
        │   ├── CategoryTrendChart.vue   # 分类趋势图
        │   ├── WeekdayChart.vue         # 周分布图
        │   └── TransactionList.vue      # 交易列表组件
        ├── composables/
        │   ├── icons.js        # SVG 图标库
        │   └── useFormatters.js # 格式化工具
        └── styles/
            └── main.css        # 设计系统 (玻璃拟态 + 主题)
```

---

## 🖼️ 演示截图

<!-- 
请将截图放置在此处，建议包含：
- Dashboard 概览页（亮/暗主题各一张）
- 支出分类饼图
- 趋势分析图表
-->

> [演示截图 - Dashboard 亮色主题]

> [演示截图 - Dashboard 暗色主题]

> [Gif Demo - 主题切换动画]

---

## 📖 API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/summary` | 财务摘要 (净资产/本月收支) |
| `GET` | `/api/analytics` | 完整分析数据 (图表/列表) |
| `GET` | `/api/transactions` | 获取交易列表 |
| `GET` | `/api/accounts` | 获取账户列表 |
| `POST` | `/api/refresh` | 刷新数据缓存 |

---

## 🚢 部署指南

### macOS 后台服务 (launchd)

```bash
# 安装服务
cp com.neve.server.plist ~/Library/LaunchAgents/
launchctl load ~/Library/LaunchAgents/com.neve.server.plist

# 查看日志
tail -f ~/Library/Logs/neve.log
```

### Cloudflare Tunnel (可选)

```yaml
# ~/.cloudflared/config.yml
ingress:
  - hostname: neve.your-domain.com
    service: http://localhost:8080
  - service: http_status:404
```

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

---

## 📄 开源协议

[MIT License](LICENSE)
