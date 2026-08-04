# Neve

基于 Beancount 纯文本复式记账的个人财务可视化系统。Go 后端解析 `.bean` 账本并输出统计,
Vue 3 前端展示;前端构建产物 embed 进 Go,产出单文件二进制 `./neve`。
架构、API、部署步骤见 `README.md`(需要时再读,不常驻上下文)。

## 常用命令

```bash
make dev-server   # 后端 (localhost:8080)
make dev          # 前端热重载 (localhost:5173)
make test         # 后端单元测试 (go test -race ./...)
make build        # 前端(lint+typecheck)→ 测试 → 后端,产出 ./neve

cd web && pnpm run lint        # make build 已内置,开发时可单跑
cd web && pnpm run typecheck
```

## 分区约定(改到哪个目录就读哪份)

| 目录 | 内容 |
|---|---|
| `server/parser/CLAUDE.md` | 资金流向三层聚合、负债待还、未来还款计划、净资产分层 |
| `server/api/CLAUDE.md` | 配置文件不得降级为空、AI 预校验、缓存并发发布、公网入口自检 |
| `server/backup/CLAUDE.md` | iCloud TCC 约束与镜像法、git 非交互 + 超时、失败告警 |
| `deploy/CLAUDE.md` | Tunnel 强制 HTTP/2、日志分流与轮转、`TZ` 钉死 |
| `web/CLAUDE.md` | 样式/图表/安全区、展示口径(负支出、日期字符串、两页分工) |

## 正确性约定(改代码前必读)

- **金额是定点数**:后端一律用 `parser.Amount`(分,int64),禁止 float64 累加金额;
  比值/均值才经 `Amount.Yuan()` 转 float64。JSON 序列化为"元"数字,前端按普通数字消费。
- **单币种 CNY**:非 CNY 数据解析时报 `NON_CNY` 错误并跳过。
- **软失败**:脏数据(不平衡/未 open 账户/非法金额日期等)跳过该笔并记入
  `Ledger.Issues`(带文件:行号),随 `/api/analytics` 的 `parseIssues` 展示在
  `IssuesBanner`;仅 main.bean 无法打开才是硬错误。
  **配置文件(debts.json)反过来——解析失败一律显式报错,不得降级为空**,理由与写盘留档
  见 `server/api/CLAUDE.md`。
- **balance 断言**会真正核对(断言日期当天开始前的余额,官方 beancount 语义),
  失败报 `BALANCE_FAILED`。
- **交易口径由后端唯一计算**:`classifyTransaction` 输出
  `kind`(expense/income/transfer/opening/mixed)、`category`、`displayAmount`、
  `transferAmount`、`feeAmount`。前端禁止从 postings 推断交易类型/金额
  (`useCategories.ts` 的 `processTransaction` 只派生展示字段)。
  统计按 posting 级聚合:转账本金不计支出,手续费计入;退款(负 Expenses)按净额冲减。
  **transfer 分支必须排在 income/mixed 之前**:「两个真实账户间有资金对流」
  (`transferAmt = min(posReal, negReal) > 0`)是转账的结构性强特征——纯收入交易只有正向真实腿、
  纯支出只有负向真实腿,都不会误命中。排在后面时,还款用积分抵掉部分现金那类交易
  (负债 +1721.39 / 资产 -1703.39 / `Income:Rewards` -18)会被 `incNet > 0` 先截成 income,
  而 `debts.go` 的分期已还判定走 `creditsAfter(..., transferOnly=true)` **只认 `kind=transfer`**,
  于是整期误报逾期、`MonthRemaining` 虚增一整笔月供。**改这里的分支顺序等于改分期还款的已还判定**。
  护栏是 `incNet < transferAmt`(Income 腿小于对流本金 = 抵扣/返还,不是独立收入事件),
  挡住「工资到账顺手还款」这类一笔两事被整笔吞成转账而藏掉大额收入;它是结构性严格比较,
  **不要退化成「浮动 5% 以内算已还」之类的比例阈值**——`debts.go` 的 `paid := paidAmount > 0`
  本就不比较金额,加阈值只会把真实的部分还款掩盖成已还清(5% 对房贷 4656 是 232 元容差)。
  被抵扣的那 18 元仍按 posting 进 `IncomeBreakdown`,总账不漏。
  净额口径下 `expense` 可以为负,**前端展示层不许取绝对值**(见 `web/CLAUDE.md`)。
- **每个统计字段的时间口径是页面契约的一部分**:`expenseByCategory`、`platformRanking`、
  `merchantRanking`、`incomeBreakdown`、`fundFlow` 都是**本月**口径(「收支分析」页整页同期);
  `weekdayDistribution`(星期分布)、`categoryTrends`(近 6 月)、各 `*Trend` 是**跨期**口径,
  归「趋势图表」页。并排的卡片必须同期:混着放等于宣称同期,而卡片副标题是唯一的口径说明,
  改聚合范围时必须连副标题一起改(字段口径见 `Analytics` struct 与 `types/api.ts` 的注释)。
- **净资产默认展示 Ex 长期负债口径**(概览与账户页,资产负债率同口径),全量降级为补充信息
  ——房贷对应的房产不在账本里,单边扣减会让净资产变成几十年不变的巨额负数。
  分层实现见 `server/parser/CLAUDE.md`。
- **日期按服务器本地时区**解析与归属,部署时用 `TZ` 显式钉死记账时区。
  同日交易按文件行序稳定排序。前端日期一律截字符串,不喂 `new Date()`(见 `web/CLAUDE.md`)。
- **`data/` 是 iCloud 软链接,不入库**(.gitignore);`data.example/` 是入库的演示数据。
  账本、`debts.json` 都只有这一份真源,写盘路径要留档、要能回滚。
- **AI 调用走原生 HTTP,不引入 SDK 依赖**(维持后端唯一依赖 gin);AI 输出必须过 parser
  预校验才可落盘(见 `server/api/CLAUDE.md`)。
- **新增日志一律 `组件: 中文描述` 前缀**(`boot`/`inbox`/`health`/`backup`/`config`),
  便于按组件 grep;`neve.error.log` 只接致命错误与 panic 栈,别往里写正常信息。
- 改解析/统计逻辑必须同步更新 `parser_test.go` / `analytics_test.go`
  (`AnalyzeAt` 可注入时钟,fixture 写 `t.TempDir()`)。
