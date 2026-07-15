# Neve

基于 Beancount 纯文本复式记账的个人财务可视化系统。Go 后端解析 `.bean` 账本并输出统计,Vue 3 前端展示;前端构建产物 embed 进 Go,产出单文件二进制 `./neve`。

## 常用命令

```bash
make dev-server   # 后端 (localhost:8080, NEVE_DATA_DIR=../data)
make dev          # 前端热重载 (localhost:5173, /api 代理到 8080)
make test         # 后端单元测试 (go test -race ./...)
make build        # 前端 → 测试 → 后端,产出 ./neve
```

前端包管理用 pnpm(`web/pnpm-lock.yaml`)。Go 1.25+,唯一依赖 gin,无数据库。

## 架构与数据流

```
iOS 快捷指令 AI 识别账单 → 追加 iCloud 的 data/inbox.bean
  → server/parser/parser.go 解析 main.bean(include 展开)+ 校验
  → server/parser/analytics.go 统计(Refresh 时算好缓存)
  → GET /api/analytics 一次性输出全部数据
  → web/src/App.vue 拉取后经 props 下发各 Tab(无 Router,activeTab + v-show)
```

- `data/` 是 iCloud 软链接,**不入库**(.gitignore);`data.example/` 是入库的演示数据。
- 前端无 TS、无 UI 库、无状态管理库,手写 CSS 变量设计系统(`web/src/styles/variables.css`,三主题)。

## 正确性约定(改代码前必读)

- **金额是定点数**:后端一律用 `parser.Amount`(分,int64),禁止 float64 累加金额;
  比值/均值才经 `Amount.Yuan()` 转 float64。JSON 序列化为"元"数字,前端按普通数字消费。
- **单币种 CNY**:非 CNY 数据解析时报 `NON_CNY` 错误并跳过。
- **软失败**:脏数据(不平衡/未 open 账户/非法金额日期等)跳过该笔并记入
  `Ledger.Issues`(带文件:行号),随 `/api/analytics` 的 `parseIssues` 展示在
  `IssuesBanner`;仅 main.bean 无法打开才是硬错误。
- **交易口径由后端唯一计算**:`classifyTransaction` 输出
  `kind`(expense/income/transfer/opening/mixed)、`category`、`displayAmount`、
  `transferAmount`、`feeAmount`。前端禁止从 postings 推断交易类型/金额
  (`useCategories.js` 的 `processTransaction` 只派生展示字段)。
  统计按 posting 级聚合:转账本金不计支出,手续费计入;退款(负 Expenses)按净额冲减。
- **balance 断言**会真正核对(断言日期当天开始前的余额,官方 beancount 语义),
  失败报 `BALANCE_FAILED`。
- **日期按服务器本地时区**解析与归属,部署机必须 `TZ=Asia/Shanghai`。
  同日交易按文件行序稳定排序。
- **ECharts 颜色**:canvas 不解析 CSS 变量,option 里必须用
  `getThemeColor('--xxx')` 取实际值,并在 computed 中引用 `themeVersion.value`
  以响应主题切换(见 `useThemeColor.js`)。
- **分类中文映射只有一份**:`web/src/composables/useCategories.js` 的 `categoryLabels`。
- 改解析/统计逻辑必须同步更新 `parser_test.go` / `analytics_test.go`
  (`AnalyzeAt` 可注入时钟,fixture 写 `t.TempDir()`)。

## 关键文件

- `server/parser/parser.go` — 解析器(正则)+ 校验 + ParseIssue 收集
- `server/parser/analytics.go` — 统计与交易分类(`AnalyzeAt`)
- `server/parser/amount.go` — 定点金额类型
- `server/api/handler.go` — 路由、analytics 缓存、budgets 原子写
- `web/src/App.vue` — 数据加载、主题、Tab 分发
- `web/src/composables/useCategories.js` — 分类映射 + 交易展示字段
- `shortcut/shortcut_to_call_ai_prompt.md` — iOS 快捷指令的 AI 识别提示词(记账入口)
