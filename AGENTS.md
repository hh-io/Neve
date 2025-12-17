# Neve - AI Agent 上下文索引

> 本文档为 AI 编程助手（Cursor、Copilot、Claude 等）设计的技术约束与上下文索引。

---

## 1. 深度技术栈

### 后端 (Go)

| 依赖 | 版本 | 用途 |
|------|------|------|
| Go | 1.21+ | 运行时 |
| `github.com/gin-gonic/gin` | v1.11.0 | HTTP 框架 |
| `embed` (stdlib) | - | 静态文件嵌入 |

**无数据库** - 数据来源为纯文本 `.bean` 文件。

### 前端 (Vue 3)

| 依赖 | 版本 | 用途 |
|------|------|------|
| Vue | ^3.5.25 | 前端框架 |
| Vite | ^7.3.0 | 构建工具 |
| ECharts | ^5.6.0 | 图表库 |
| vue-echarts | ^8.0.1 | Vue 封装 |
| @vitejs/plugin-vue | ^6.0.3 | Vue SFC 编译 |

**无状态管理库** - 使用 Vue 3 `ref`/`reactive` + Composition API。
**无 UI 组件库** - 纯手写 CSS 设计系统。
**无 TypeScript** - 纯 JavaScript。

---

## 2. 目录与路由架构

### 2.1 目录结构

```
Neve/
├── server/                      # Go 后端
│   ├── main.go                  # 入口 + HTTP 服务 + embed 静态文件
│   ├── api/
│   │   └── handler.go           # REST API 处理器 (Server struct)
│   ├── parser/
│   │   ├── parser.go            # Beancount 解析器 (正则)
│   │   └── analytics.go         # 数据分析 (Summary/CategoryAmount/MonthlyData...)
│   └── static/                  # 前端构建产物 (自动生成，勿手改)
│
├── web/                         # Vue 3 前端
│   ├── vite.config.js           # 输出到 ../server/static
│   └── src/
│       ├── main.js              # Vue 入口
│       ├── App.vue              # 主组件 (侧边栏 + 路由分发)
│       ├── components/
│       │   ├── tabs/            # 页面级组件 (作为 Tab 内容)
│       │   │   ├── OverviewTab.vue      # 概览页
│       │   │   ├── SpendingTab.vue      # 支出分析
│       │   │   ├── TrendsTab.vue        # 趋势图表
│       │   │   ├── TransactionsTab.vue  # 交易列表
│       │   │   └── AccountsTab.vue      # 账户管理
│       │   ├── TransactionList.vue      # 交易列表组件 (复用)
│       │   ├── CategoryTrendChart.vue   # 分类趋势图
│       │   ├── WeekdayChart.vue         # 周分布热力图
│       │   ├── BudgetCard.vue           # 预算卡片
│       │   ├── MerchantRanking.vue      # 商户排行
│       │   └── PlatformRanking.vue      # 平台排行
│       ├── composables/
│       │   ├── icons.js         # SVG 图标集合 (inline SVG 字符串)
│       │   ├── useFormatters.js # formatCurrency / formatDate 等
│       │   └── useThemeColor.js # 主题色工具
│       └── styles/
│           └── main.css         # 设计系统 (CSS Variables + 3 主题)
│
└── data/                        # Beancount 数据目录 (可 iCloud 同步)
    ├── main.bean                # 入口文件 (账户 open + include)
    ├── inbox.bean               # iOS 快捷指令写入
    ├── balance.bean             # 余额断言
    └── 2025.bean                # 年度归档
```

### 2.2 前端路由

**SPA 单页应用，无 Vue Router**。路由通过 `activeTab` state 实现：

```javascript
// App.vue
const activeTab = ref('overview');  // 'overview' | 'spending' | 'trends' | 'transactions' | 'accounts'
```

Tab 组件使用 `v-show` 切换（保持状态），非 `v-if`。

### 2.3 API 接口设计

| Method | Endpoint | Handler | 返回类型 |
|--------|----------|---------|----------|
| GET | `/api/summary` | `handleSummary` | `Summary` |
| GET | `/api/analytics` | `handleAnalytics` | `Analytics` (完整数据) |
| GET | `/api/transactions` | `handleTransactions` | `{transactions, total}` |
| GET | `/api/accounts` | `handleAccounts` | `{accounts, total}` |
| POST | `/api/refresh` | `handleRefresh` | `{message, summary}` |
| GET | `/api/budgets` | `handleGetBudgets` | `map[string]float64` |
| POST | `/api/budgets` | `handleSaveBudgets` | `{message}` |

**路由注册位置**: `server/api/handler.go` → `SetupRoutes()`

**前端主要使用 `/api/analytics` 一次性获取所有数据。**

---

## 3. 开发规范与约束 (Critical Rules)

### 3.1 包管理器

```bash
# ✅ 强制使用 pnpm
pnpm install
pnpm run dev
pnpm run build

# ❌ 禁止使用
npm install
yarn add
```

### 3.2 构建命令

```bash
# 开发
make dev-server      # 后端 (localhost:8080)
make dev             # 前端 (localhost:5173, 代理 /api → 8080)

# 生产构建
make build           # 构建前端 → 后端 (embed 静态)
./neve               # 运行单文件
```

### 3.3 代码风格

#### Go 后端

- **格式化**: 运行 `go fmt ./...` 后提交
- **静态分析**: `go vet ./...`
- **错误处理**: 返回 `gin.H{"error": err.Error()}`
- **并发安全**: `Server` struct 使用 `sync.RWMutex` 保护 `ledger`

#### Vue 前端

- **组件范式**: `<script setup>` Composition API
- **响应式**: `ref()` 用于基础类型，`reactive()` 用于对象
- **样式**: 使用 `main.css` 中定义的 CSS Variables
- **图标**: 从 `composables/icons.js` 导入 inline SVG

### 3.4 CSS 主题系统

```css
/* 三套主题 */
.theme-light { /* 默认亮色 */ }
.theme-dark  { /* 暗色 */ }
.theme-geek  { /* Slate/Indigo 极客风 */ }

/* 使用语义化变量 */
var(--bg-primary)      /* 主背景 */
var(--text-primary)    /* 主文字 */
var(--brand-primary)   /* 品牌色 */
var(--income)          /* 收入绿 */
var(--expense)         /* 支出红 */
```

### 3.5 禁止事项

- ❌ 不使用 TypeScript
- ❌ 不使用 CSS 预处理器 (Sass/Less)
- ❌ 不使用 UI 组件库 (Element/Ant Design)
- ❌ 不使用 Tailwind CSS
- ❌ 不引入新的状态管理库 (Pinia/Vuex)
- ❌ 不使用内联样式，统一用 CSS 变量

---

## 4. 业务逻辑上下文

### 4.1 核心数据流

```
┌─────────────────────────────────────────────────────────────────┐
│ iOS 快捷指令 → iCloud Drive → inbox.bean                        │
│                                   ↓                             │
│ Go Parser (parser.go) ─────→ Ledger struct                      │
│                                   ↓                             │
│ Analytics (analytics.go) ──→ Summary + Charts 数据              │
│                                   ↓                             │
│ REST API (handler.go) ─────→ /api/analytics                     │
│                                   ↓                             │
│ Vue Dashboard (App.vue) ───→ 用户可视化界面                      │
└─────────────────────────────────────────────────────────────────┘
```

### 4.2 Beancount 解析规则

**解析器**: `server/parser/parser.go` (使用正则表达式)

```go
// 支持的指令
include "file.bean"                    // 文件包含
2025-01-01 open Assets:Bank:CMBC CNY   // 账户开设
2025-01-01 * "Payee" "Narration" #tag  // 交易
2025-01-01 balance Assets:... 1000 CNY // 余额断言

// 交易 posting
  Assets:Bank:CMBC    -100.00 CNY
  Expenses:Food        100.00 CNY      // 支出为正
  Income:Salary       -5000.00 CNY     // 收入为负
```

### 4.3 分析逻辑 (analytics.go)

| 分析项 | 说明 |
|--------|------|
| `Summary` | 净资产、本月收支 |
| `ExpenseByCategory` | 本月支出分类占比 |
| `MonthlyTrend` | 近 6 个月收支趋势 |
| `DailyTrend` | 近 30 天每日收支 |
| `WeekdayDistribution` | 按星期几聚合消费 |
| `MerchantRanking` | 商户消费排行 Top 10 |
| `PlatformRanking` | 平台消费排行 (by tag) |
| `CategoryTrends` | Top 5 分类月度趋势 |

### 4.4 账户类型判断

```go
Assets:*       → 资产 (正余额)
Liabilities:*  → 负债 (负余额，展示时取反)
Income:*       → 收入 (posting 金额为负)
Expenses:*     → 支出 (posting 金额为正)
Equity:*       → 权益 (初始余额，分析时过滤)
```

---

## 5. 常见任务索引

| 任务 | 修改文件 |
|------|----------|
| 新增 API 端点 | `server/api/handler.go` → `SetupRoutes()` |
| 修改解析规则 | `server/parser/parser.go` |
| 新增分析指标 | `server/parser/analytics.go` → `Analyze()` |
| 新增页面 Tab | `web/src/components/tabs/` + `App.vue` activeTab |
| 修改图表样式 | 对应组件 + `main.css` |
| 添加新图标 | `web/src/composables/icons.js` |
| 修改主题颜色 | `web/src/styles/main.css` `:root` 变量 |

---

## 6. 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `NEVE_DATA_DIR` | `./data` | Beancount 数据目录 |
| `NEVE_PORT` | `8080` | HTTP 服务端口 |
