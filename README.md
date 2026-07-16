<p align="center">
  <img src="https://img.shields.io/badge/vue-3.5-4FC08D?style=flat-square&logo=vue.js" alt="Vue 3">
  <img src="https://img.shields.io/badge/typescript-5.9-3178C6?style=flat-square&logo=typescript" alt="TypeScript">
  <img src="https://img.shields.io/badge/go-1.25+-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/vite-8.x-646CFF?style=flat-square&logo=vite" alt="Vite">
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat-square" alt="License">
</p>

<h1 align="center">❄️ Neve</h1>

<p align="center">
  <strong>基于 Beancount 复式记账的现代个人财务管理系统</strong>
</p>

<p align="center">
  一键启动 • 数据即文件
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
| 🎨 **Apple 设计语言** | 精心调校的亮/暗双主题 |
| 📈 **丰富的图表分析** | 收支趋势、分类占比、周消费分布、商户排行 |
| ✅ **记账正确性校验** | 借贷平衡、账户定义、balance 断言对账,脏数据显式报错不进统计 |
| 💰 **定点金额运算** | 金额以"分"为单位整数运算,无浮点误差 |
| 🚀 **单文件部署** | 前端嵌入二进制，一个 `./neve` 即刻运行 |
| ☁️ **iCloud 原生集成** | 与 iOS 快捷指令配合，实现移动端无应用记账 |

---

## 🛠️ 技术栈概览

```
┌─────────────────────────────────────────────────────────┐
│                       Frontend                          │
│  Vue 3 + TS  •  Vite 8  •  ECharts  •  @lucide/vue      │
│  设计 token(CSS 变量,双主题)• composable 单例(无 Pinia)│
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
| **Go** | >= 1.25 | [golang.org/dl](https://golang.org/dl/) |
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

### 🧪 测试

```bash
# 后端单元测试（解析器 + 统计逻辑，make build 会自动执行）
make test
```

### 📦 生产构建

```bash
# 一键构建：前端 → 测试 → 后端（嵌入静态文件）
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
├── 📁 data/                     # Beancount 数据目录 (可 iCloud 同步,不入库)
│   ├── main.bean               # 入口文件 (账户定义 + include)
│   ├── balance.bean            # 初始余额 + balance 断言
│   ├── inbox.bean              # 待整理流水 (iOS 快捷指令写入)
│   └── 2025.bean               # 年度归档交易
├── 📁 data.example/             # 演示数据 (入库,结构同 data/)
│
├── 📁 server/                   # 🔧 Go 后端
│   ├── main.go                 # 入口 + HTTP 服务 + 静态文件 embed
│   ├── api/
│   │   └── handler.go          # REST API 处理器 (analytics 缓存)
│   └── parser/
│       ├── amount.go           # 定点金额类型 (分, int64)
│       ├── parser.go           # Beancount 解析器 (校验 + 错误收集)
│       ├── analytics.go        # 数据分析 (净资产/分类/趋势/转账识别)
│       └── *_test.go           # 单元测试
│
└── 📁 web/                      # 🎨 Vue 3 + TypeScript 前端
    └── src/
        ├── App.vue             # 布局壳 + 主题 + Tab 分发
        ├── types/api.ts        # /api/analytics 契约类型 (对照后端 struct)
        ├── components/
        │   ├── tabs/           # 页面标签组件 (Overview/Spending/Trends/Transactions/Accounts)
        │   ├── layout/         # AppSidebar / MobileNav / ThemeSwitcher
        │   ├── common/         # AppToast / IssuesBanner
        │   ├── CategoryTrendChart.vue / WeekdayChart.vue / TransactionCalendar.vue  # 图表
        │   ├── MerchantRanking.vue / PlatformRanking.vue                            # 排行
        │   ├── TransactionList.vue      # 交易列表 (changelog-row)
        │   └── BudgetCard.vue           # 预算卡
        ├── composables/        # 模块级单例 + 工具 (.ts)
        │   ├── useAnalytics.ts # analytics 单例 fetch/refresh (429 处理)
        │   ├── useTheme.ts / useToast.ts / useBudgets.ts  # 主题/Toast/预算单例
        │   ├── useCategories.ts # 分类中文映射 + 交易展示字段
        │   ├── useCategoryIcon.ts / navItems.ts           # lucide 图标映射
        │   ├── useFormatters.ts # 格式化工具
        │   └── useThemeColor.ts # ECharts 主题取色 (getThemeColor + themeVersion)
        └── styles/             # 设计 token 系统 (variables/base/layout/components/mobile)
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
| `GET` | `/api/analytics` | 完整分析数据 (摘要/图表/全量交易/parseIssues/balanceChecks),前端一次拉取全量 |
| `POST` | `/api/refresh` | 重新解析账本并重建缓存 (5 秒限流) |
| `GET` | `/api/budgets` | 获取预算 |
| `POST` | `/api/budgets` | 保存预算 (原子写 budgets.json) |

> 仅支持 CNY 单币种;非 CNY、借贷不平衡、未 open 账户的交易会被跳过并在 `parseIssues` 中报错。

---

## 🚢 部署指南

### macOS 后台服务 (launchd)

服务与日志轮转配置以模板形式放在 `deploy/`(占位符 `@NEVE_ROOT@`/`@HOME@`/`@USER@`),
由 make 按本机路径渲染后安装,仓库中不含硬编码路径:

```bash
# 渲染并安装 launchd 配置到 ~/Library/LaunchAgents (只写文件,不启动)
make install-service
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.neve.server.plist

# 渲染并安装日志轮转配置到 /etc/newsyslog.d (需 sudo)
make install-logrotate

# 查看日志
tail -f ~/Library/Logs/neve.log
```

改动模板后重新执行 `make install-service`,再 `launchctl bootout gui/$(id -u)/com.neve.server`
+ `bootstrap` 重载生效。

> ⚠️ 日期按服务器本地时区归属月份/星期,部署机时区必须为 `Asia/Shanghai`
> (plist 模板已在 EnvironmentVariables 中内置 `TZ=Asia/Shanghai`)。

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
