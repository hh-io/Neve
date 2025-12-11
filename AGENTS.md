# Neve - AI Agent 开发指南

> 本文档专为 AI 编程工具（如 Cursor、Copilot、Claude）设计，帮助快速理解项目结构和开发规范。

## 项目定位

Neve 是一个基于 **Beancount 纯文本复式记账** 的个人财务管理系统：

- **数据层**: `.bean` 文件 (存储在 iCloud，通过 iOS 快捷指令写入)
- **后端**: Go + Gin (解析 .bean 文件，提供 REST API)
- **前端**: Vue 3 + Vite + ECharts (Dashboard 可视化)

## 快速理解

### 数据流

```
用户 → iOS 快捷指令 → iCloud/data/inbox.bean → Go 解析器 → REST API → Vue Dashboard
```

### 核心文件

| 文件                         | 作用                 | 修改频率 |
| ---------------------------- | -------------------- | -------- |
| `server/parser/parser.go`    | 解析 .bean 文件      | 低       |
| `server/parser/analytics.go` | 计算净资产/分类/趋势 | 中       |
| `server/api/handler.go`      | REST API 端点        | 中       |
| `web/src/App.vue`            | Dashboard 主页面     | 高       |
| `web/src/styles/main.css`    | 设计系统 CSS 变量    | 低       |

---

## 开发指令

```bash
# 安装依赖
cd web && pnpm install

# 前端开发 (热重载)
cd web && pnpm run dev

# 前端构建 (输出到 server/static)
cd web && pnpm run build

# 后端开发
cd server && go run .

# 完整构建
make build

# 运行
./neve
# 或
NEVE_DATA_DIR=./data NEVE_PORT=8080 ./neve
```

---

## API 参考

### GET /api/analytics

返回完整的分析数据，前端 Dashboard 主要使用此接口。

```typescript
interface Analytics {
  summary: {
    netWorth: number; // 净资产
    totalAssets: number; // 总资产
    totalLiabilities: number; // 总负债
    monthIncome: number; // 本月收入
    monthExpense: number; // 本月支出
    monthBalance: number; // 本月结余
    lastUpdated: string; // ISO 时间
  };
  expenseByCategory: Array<{
    category: string;
    amount: number;
    percent: number;
  }>;
  accountBalances: Array<{
    account: string;
    balance: number;
    currency: string;
    type: "Assets" | "Liabilities";
  }>;
  monthlyTrend: Array<{
    month: string; // "2025-12"
    income: number;
    expense: number;
    balance: number;
  }>;
  recentTransactions: Transaction[] | null;
}

interface Transaction {
  date: string;
  flag: "*" | "!";
  payee: string;
  narration: string;
  postings: Array<{
    account: string;
    amount: number;
    currency: string;
  }>;
  tags: string[];
  sourceFile: string;
}
```

### POST /api/refresh

重新解析所有 .bean 文件，返回更新后的 summary。

---

## Beancount 语法

### 交易格式

```beancount
YYYY-MM-DD * "Payee" "Narration" #tag
  Account:Name   Amount CURRENCY
  Account:Name   Amount CURRENCY
```

### 场景示例

```beancount
; 支出
2025-12-04 * "Starbucks" "冰美式"
  Expenses:Food:Coffee        35.00 CNY
  Assets:Cash:WeChat         -35.00 CNY

; 收入 (Income 用负数)
2025-12-08 * "张三" "会员费"
  Assets:Cash:WeChat         199.00 CNY
  Income:Membership         -199.00 CNY

; 还款含利息 (3个 posting)
2025-12-10 * "京东金融" "金条还款"
  Assets:Bank:CMBC          -3383.33 CNY
  Liabilities:JD:CLO         3333.33 CNY
  Expenses:Financial:Interest  50.00 CNY
```

### 关键规则

1. **平衡**: 所有 posting 金额之和必须为 0
2. **负债**: 初始化用负数 (欠款)，还款用正数 (减少欠款)
3. **收入**: 必须记为负数
4. **支出**: 必须记为正数

---

## 代码规范

### Go 后端

- 解析器使用正则表达式，位于 `parser/parser.go`
- 分析逻辑在 `parser/analytics.go`
- API 处理器在 `api/handler.go`
- 静态文件通过 `//go:embed` 嵌入

### Vue 前端

- 单文件组件 (SFC)
- CSS 变量定义在 `styles/main.css`
- ECharts 通过 `vue-echarts` 使用
- 设计风格: Apple Minimal + Glassmorphism

---

## 常见任务

### 添加新的 API 端点

1. 编辑 `server/api/handler.go`
2. 在 `SetupRoutes` 中添加路由
3. 实现处理函数

### 添加新的图表

1. 在 `web/src/App.vue` 中添加 computed 属性
2. 使用 ECharts option 格式
3. 添加 `<v-chart>` 组件

### 修改解析规则

1. 编辑 `server/parser/parser.go`
2. 修改对应的正则表达式
3. 重新构建 `make build`

---

## 环境变量

| 变量            | 必需 | 默认值   | 说明           |
| --------------- | ---- | -------- | -------------- |
| `NEVE_DATA_DIR` | 否   | `./data` | .bean 文件目录 |
| `NEVE_PORT`     | 否   | `8080`   | HTTP 端口      |

---

## 目录映射

```
功能 → 文件
────────────────────────────────────
账户定义     → data/main.bean
交易录入     → data/inbox.bean
初始余额     → data/balance.bean
Bean 解析    → server/parser/parser.go
数据分析     → server/parser/analytics.go
REST API     → server/api/handler.go
静态文件服务 → server/main.go
Dashboard UI → web/src/App.vue
样式系统     → web/src/styles/main.css
```
