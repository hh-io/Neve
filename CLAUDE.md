# Neve

基于 Beancount 纯文本复式记账的个人财务可视化系统。Go 后端解析 `.bean` 账本并输出统计,Vue 3 前端展示;前端构建产物 embed 进 Go,产出单文件二进制 `./neve`。

## 常用命令

```bash
make dev-server   # 后端 (localhost:8080, NEVE_DATA_DIR=../data)
make dev          # 前端热重载 (localhost:5173, /api 代理到 8080)
make test         # 后端单元测试 (go test -race ./...)
make build        # 前端(lint+typecheck)→ 测试 → 后端,产出 ./neve

# 前端单独校验(make build 已内置,开发时可单跑):
cd web && pnpm run lint        # ESLint flat/recommended
cd web && pnpm run typecheck   # vue-tsc --noEmit
```

前端包管理用 pnpm(`web/pnpm-lock.yaml`)。Go 1.26+,唯一依赖 gin,无数据库。

## 架构与数据流

```
iOS 快捷指令上传账单图片 → POST /api/inbox(Bearer 鉴权,立即 202)
  → server/api/inbox.go 异步:server/ai 拼提示词(账户列表实时取自 main.bean)
    → AI 视觉识别(claude/gemini,原生 HTTP)→ parser 预校验(失败回喂修正一次)
    → 追加 iCloud 的 data/inbox.bean → Refresh → Bark 推送结果
    (失败不落盘,留档 data/failed/<时间戳>/)
  → server/parser/parser.go 解析 main.bean(include 展开)+ 校验
  → server/parser/analytics.go 统计(Refresh 时算好缓存)
  → GET /api/analytics 一次性输出全部数据
  → web/src/composables/useAnalytics.ts 模块级单例 fetch,各 Tab 直接消费
    (无 Router,activeTab + v-show;无 prop 钻透)
```

- `data/` 是 iCloud 软链接,**不入库**(.gitignore);`data.example/` 是入库的演示数据。
- 无感记账入口由环境变量启用:`NEVE_INBOX_TOKEN` + `NEVE_AI_PROVIDER`/`NEVE_AI_API_KEY`
  (+`NEVE_AI_MODEL`、可选 `NEVE_BARK_URL`),缺任一则 `/api/inbox` 返回 404。
  数据备份由 `NEVE_BACKUP_REMOTE`(git 远程 URL)启用,可选 `NEVE_BACKUP_DIR`(镜像位置)。
  部署密钥统一放 gitignore 的 `deploy/local.env`(模板 `local.env.example`),由
  `make install-service` / `make install-tunnel` 渲染注入;Tunnel ingress 只放行
  `/api/inbox`,无鉴权端点不暴露公网。AI 调用走原生 HTTP,**不引入 SDK 依赖**
  (维持后端唯一依赖 gin)。
- 前端 **TypeScript**(`vue-tsc` 校验,契约类型见 `web/src/types/api.ts`),无 UI 库
  (图标用 `@lucide/vue`),无状态管理库(以 composable 模块级单例替代 Pinia:
  `useAnalytics`/`useTheme`/`useToast`/`useBudgets`/`useDebts`),手写 CSS 变量设计系统
  (`web/src/styles/variables.css`,亮/暗双主题;token 体系见该文件头部注释)。

## 正确性约定(改代码前必读)

- **金额是定点数**:后端一律用 `parser.Amount`(分,int64),禁止 float64 累加金额;
  比值/均值才经 `Amount.Yuan()` 转 float64。JSON 序列化为"元"数字,前端按普通数字消费。
- **单币种 CNY**:非 CNY 数据解析时报 `NON_CNY` 错误并跳过。
- **软失败**:脏数据(不平衡/未 open 账户/非法金额日期等)跳过该笔并记入
  `Ledger.Issues`(带文件:行号),随 `/api/analytics` 的 `parseIssues` 展示在
  `IssuesBanner`;仅 main.bean 无法打开才是硬错误。
- **AI 输出必须过 parser 预校验才可落盘**:`server/api/inbox.go` 的 `validateCandidate`
  先经 `checkTransactionOnly` 拒绝任何非交易顶层行(open/include/option/散文——parser
  会静默忽略或如实执行它们,AI 补一行 open 即可绕过账户白名单),再在临时目录拼
  "真实 open 指令 + 候选交易"试解析,任何 issue 都拒绝写入并回喂 AI
  修正一次;识别提示词的账户列表由 `server/ai.ExtractAccounts` 从 main.bean **原文**
  提取(保留行尾中文注释,parser 结构化数据会丢注释),不要再手工维护账户清单。
- **交易口径由后端唯一计算**:`classifyTransaction` 输出
  `kind`(expense/income/transfer/opening/mixed)、`category`、`displayAmount`、
  `transferAmount`、`feeAmount`。前端禁止从 postings 推断交易类型/金额
  (`useCategories.ts` 的 `processTransaction` 只派生展示字段)。
  统计按 posting 级聚合:转账本金不计支出,手续费计入;退款(负 Expenses)按净额冲减。
- **balance 断言**会真正核对(断言日期当天开始前的余额,官方 beancount 语义),
  失败报 `BALANCE_FAILED`。
- **负债待还口径**(`server/parser/debts.go` 的 `ComputeDebts`,配置存 `data/debts.json`):
  额度类"本期应还"= 账单日当天结束时的欠款余额快照,**先扣减内嵌免息分期的未出账金额**
  (`RevolvingConfig.Installments`:分期消费记账时全额入负债账户,银行按月出账;
  未出账 = 总额 − 已出账期数×每期金额,尾差落最后一期,月数按 `YYYY-MM` 差值算无进位坑;
  **首期账单月晚于当前账单月的分期不参与扣减**——账单日后新购,本金不在快照里,扣了是双重扣减);
  冲减按账单日后转入该账户的
  **正向 posting**(不限交易 kind,退款/返现也应冲减);分期类"已还"只认 `kind=transfer`。
  账单日/还款日超出当月天数时**顺延至月末**(`clampedDate`,严禁裸 `time.Date` 进位)。
  GET /api/debts 每次用缓存 Ledger 现算,配置变更无需 refresh。
- **未来还款计划是现金流口径,不是资产负债表口径**(`server/parser/schedule.go` 的
  `ComputeSchedule`,随 `DebtsReport.Schedule` 一并返回,无独立 endpoint):按月展开未来
  `scheduleMonths` 个自然月的出账。展开四类:额度账户内嵌分期、固定分期 schedule、
  已出账未还的账单(`RevolvingStatus.Remaining`)、已消费但未出账的余额。前两类金额来自配置,
  后两类要 `ComputeDebts` 先从账本算出来,但**四类都是已经确定的钱**(已刷掉或已锁定,
  只是没到付款日),唯一不做的是**预测未来还会消费多少**。故每张卡最多只有两笔非账单分期出账
  (当期账单 + 下期账单),再往后只剩分期,近月天然高于远月(远月不是压力小,是账单还没发生,
  前端口径说明必须讲明这点)。
  **每张卡今天的欠款按「何时流出」拆成三份,三份相加恒等于 `CurrentBalance`**——这是防重复
  计算的不变量,改这里先验算它:①`Remaining`(本期还款日)②`Σ Installments[].UnbilledAmount`
  (由月循环逐期展开)③ 余下的即「已消费未出账」(下个账单日出账)。③ 必须扣**全量**未出账分期
  (含 `FirstBillMonth` 晚于本期的:那些本金已在 `CurrentBalance` 里且下面会逐期展开),
  且 `PaidSince` 只在 ① 里扣一次,别在 ③ 再扣一遍。③ 的归桶日是
  `nextDueAfter(下个账单日, DueDay)`,月份推进走「月初 +1 月再 `clampedDate`」,
  不让 31 号账单日在短月进位。
  **账单条目整笔覆盖该期内嵌分期**:`statementDue` 只扣了*未出账*部分,本期那一期分期已含在
  账单里,再单独展开就是双重计算;故按 `(账户, 还款日)` 命中当期账单时跳过该期分期,顺带让
  「账单已还清」(`Remaining` 归 0)时该期自然消失。
  注:窗口内 ①②③ 之和只在分期剩余期数装得下 `scheduleMonths` 时才与 `CurrentBalance` 相等,
  尾部落在窗口外属预期,不是漏算。同理**不推演未来净资产**:还款是 transfer(资产↓、负债↓同额),没有未来收支预测时未来净资产
  恒等于今天,推了没信息量;更不能把「未到期的分期」从负债里剔除后当净资产看——分期消费记账时
  已全额入负债、对应消费也已如实记在 Expenses 腿上,剔除即虚高(这与长期负债分层性质不同,
  后者是因为对应资产根本不在账本里)。每期金额走 `installmentPeriodAmount`,与
  `revolvingInstallmentStatuses` **共用同一函数**(尾差落最后一期),口径不会漂移;
  日期同样走 `clampedDate`/`nextDueAfter`,月末顺延语义自动一致。
  **归桶键是还款日所在月,不是账单月**——现金流问的是钱哪个月流出,账单日 25/还款日 10 这类配置
  due 落到次月,按账单月归桶整张表会错位一个月;故账单月要从窗口首月**往前多扫一个月**
  (`nextDueAfter` 最多推后一个月,一个月足够),否则首桶漏掉上月账单那笔。
  **due 早于 today 的期一律剔除**:钱已经该流出了(已还,或逾期——后者由 `Summary.MonthRemaining`
  负责),留在「未来计划」里会虚增即期压力;还款日当天仍算未来。这与 `InstallmentStatus.Paid`
  /`Overdue` 同一口径。固定分期的终止期看 `InstallmentConfig.EndMonth`(空 = 无终止期,房贷),
  过了末期月即 `settled`:计划表不再展开、`ComputeDebts` 把月供归 0(顺带让开
  overdue/MonthDue/NextDue),判定与 `ComputeDebts` **共用 `installmentRemaining`**。
  `InstallmentStatus.RemainingPeriods` 是尚未还的期数(本期已还则不含本期,免得月初月末差一期),
  **-1 表示无终止期**,区别于已结清的 0——前端据此决定是否展示「剩 N 期」。
  期数不能从账本反推:负债余额是剩余本金、不含未来利息,除月供不成整数,只能靠配置。
  吃 `DebtsConfig` + 已算好的 `[]RevolvingStatus` + 时钟,**自身仍不碰 Ledger**——账单余额
  要 `balanceAsOf` 才有,让它吃现成结果而非重新解析;`ComputeDebts` 里的调用点因此**必须排在
  `report.Revolving` 填完之后**。`statements` 传 nil 即退化成「只展开配置里的分期」。
- **净资产分层口径**:房贷这类长期负债对应的资产(房产)不在账本里,单边扣减会让净资产
  变成几十年不变的巨额负数。故 `Summary` 除 `netWorth`/`totalLiabilities` 全量口径外,
  另有 `longTermLiabilities`/`shortTermLiabilities`/`netWorthExLongTerm`,
  **概览与账户页默认展示 Ex 口径**(资产负债率同口径),全量降级为补充信息。
  账户清单存 `DebtsConfig.LongTermAccounts`(debts.json 顶层),标记跟着**账户**走:
  已配账期的账户在自己那张卡的编辑态勾选,其余负债账户在待还页底部「其他负债账户」组勾选
  (`LongTermOthers.vue`,勾选即存)。删除账期配置**不清除**该账户的长期标记——账户还在
  账本里就仍是长期负债,只是编辑入口从卡片挪回「其他负债账户」组。
  分层由 `Analytics.ApplyLongTermLiabilities` 叠加在 `Analyze` 之后(**保持 Analyze 纯函数**——
  清单不在账本里,改配置不该触发重解析),该方法幂等,调用点有二:`Refresh()` 与
  `handleSaveDebts()`(后者就地重算缓存,前端 `useDebts.saveDebts` 再静默 `reload()` analytics)。
  求和遍历 `AccountBalances` 而非 `LiabilityBreakdown`——后者只收余额为负的账户,
  长期负债多还成正余额时会被漏掉,与 `TotalLiabilities` 口径对不上。
- **数据备份必须由服务端进程做,不能交给独立 launchd/cron 任务**:数据在快捷指令
  App 的 iCloud 容器(`data` 软链指向处),属 macOS TCC 重点保护区。未获授权的
  launchd 进程对该目录 `readdir`/`chdir` 一律 `Operation not permitted`(连 `git add`
  都因 git 要 chdir 进工作树而失败),`stat` 单文件放行但 `open` 读内容也被拒;而
  **服务端进程已获该容器读权限**。故 `server/backup` 采用镜像法:服务端用 `os.ReadFile`
  读账本内容写进 iCloud 外的镜像 git 工作树,git 只对镜像操作(非 iCloud、无 TCC 限制)。
  备份文件清单取自 `Ledger.SourceFiles`(parser 记录实际打开的 main.bean+include 文件,
  单一真源)+ 已知配置名(budgets/debts.json);`triggerBackup` 有护栏——账本为空或
  `SourceFiles` 为空时**跳过**,否则空清单会把镜像里已跟踪的 .bean 全 prune 成删除。
  推送用普通 `git push`(非 force),首推需远程为空库。
  **git 子进程必须非交互 + 带超时**:launchd 下无 tty,凭据提示/未知 host key 会让 git
  永久挂住,而 `Snapshot` 全程持锁——后续每次触发都堆一个 goroutine 在锁上,备份彻底
  停摆且无信号。故统一走 `git()`:`GIT_TERMINAL_PROMPT=0` + `ssh -o BatchMode=yes`,
  `snapshotTimeout` 限总时长(取消先 SIGINT 留 `gitTermGrace` 清理 index.lock)。
  失败必须**推送告警**(`alertBackupFailure`,`backupAlertInterval` 节流):凭据失效与
  non-fast-forward 不会自愈,只写日志等于静默失效。git 报错会带 remote URL,
  出口统一过 `redactCredentials` 抹掉内嵌 token。
  每日兜底用**墙上时钟轮询**(`backupTickInterval` + 比对日期)而非 24h 定时器:
  后者随进程重启漂移,且睡眠期间 monotonic 定时器是否推进依平台而异。
- **日期按服务器本地时区**解析与归属,部署时用 `TZ` 显式钉死记账时区
  (当前 `Asia/Singapore`,见 `deploy/com.neve.server.plist.in`)。
  同日交易按文件行序稳定排序。
- **ECharts 颜色**:canvas 不解析 CSS 变量,option 里必须用
  `getThemeColor('--xxx')` 取实际值,并在 computed 中引用 `themeVersion.value`
  以响应主题切换(见 `useThemeColor.ts`)。图表色板走 `--chart-1..8` /
  `--chart-income` / `--chart-expense` token(热力图为顺序标度,例外保留绿渐变)。
- **前端组件禁止内联 `style="..."`**:颜色/间距/圆角/字号一律走 design token
  (`variables.css` 的 surface/hairline/accent/chart 系列 + `--space-*`/`--radius-*`);
  仅**真正的运行时值**(进度条宽度、数据驱动的 tag 色、交错动画延时)可用 `:style` 绑定。
  组件样式写 `<style scoped>` 或提炼进 `styles/components.css` 共享类
  (如 `.panel`/`.filter-pill`/`.empty-state`)。**scoped 不穿透子组件**:父组件里定义、
  子组件模板复用的类必须放共享层,否则子组件那份渲染成裸元素。
- **分类中文映射只有一份**:`web/src/composables/useCategories.ts` 的 `categoryLabels`。
- 改解析/统计逻辑必须同步更新 `parser_test.go` / `analytics_test.go`
  (`AnalyzeAt` 可注入时钟,fixture 写 `t.TempDir()`)。

## 关键文件

- `server/parser/parser.go` — 解析器(正则)+ 校验 + ParseIssue 收集
- `server/parser/analytics.go` — 统计与交易分类(`AnalyzeAt`)+ 净资产分层(`ApplyLongTermLiabilities`)
- `server/parser/amount.go` — 定点金额类型
- `server/parser/debts.go` — 负债待还计算(`ComputeDebts`,账期/倒计时/剩余期数/schedule 口径)
  + `DebtsConfig`(含 `longTermAccounts`、固定分期的 `endMonth`)
- `server/parser/schedule.go` — 未来还款计划(`ComputeSchedule`,现金流口径,按还款日归月,
  展开确定性分期出账 + 已出账账单 + 已消费未出账)+ 滚动窗口汇总(`SumDueWithin`,
  喂看板的 `due30`/`due90`;**滚动而非自然月**——按自然月算每到月末都会塌成 0)
- `server/api/handler.go` — 路由、analytics 缓存、budgets 原子写
- `server/api/inbox.go` — 无感记账端点(鉴权、异步识别、预校验、留档、Bark 推送)
- `server/backup/backup.go` — 数据备份(账本镜像进 iCloud 外 git 仓库 + 提交/推送)
- `server/ai/` — AI 视觉客户端(claude/gemini 原生 HTTP)+ 提示词模板(prompt.md,
  `{{DATE}}`/`{{ACCOUNTS}}` 运行时注入)
- `web/src/App.vue` — 布局壳、主题、Tab 分发(数据/主题为 composable 单例)
- `web/src/types/api.ts` — `/api/analytics` 契约类型(逐字段对照后端 struct JSON tag)
- `web/src/composables/useAnalytics.ts` — analytics 单例 fetch/refresh(429 处理)/reload(配置变更后静默重取)
- `web/src/composables/useDebts.ts` — 待还配置/报告单例(GET/POST /api/debts)
- `web/src/components/debts/` — 待还条目卡(`RevolvingCard`/`InstallmentCard` 各自带展示/编辑两态,
  一张卡只编辑一个条目;`LongTermOthers` 收拢未配账期账户的长期负债勾选;
  `PaymentSchedule` 展开未来 12 个月还款计划,常驻口径说明不可删——明细里 `statement` 与
  `unbilled` **合并呈现为同一类「信用卡账单」**(同名同色),仅靠标记文案「账单」/「账单 · 预估」
  区分后者金额还会变;两者永不落在同一月(本期与下期账单差一个账单周期),故无需真合并数组)。
  编辑互斥由 `DebtsTab` 的 `editingKey` 控制:保存是「合成全量 config 再 POST」,
  同时开两张卡会互相覆盖。`DebtsTab` 顶部看板走 `due30`/`due90`(现金流口径,直接汇总
  schedule,与表里的数同源),`monthRemaining` 降级为**仅逾期时**显示的告警——
  逾期的钱不在 schedule 里,这是它唯一不可替代的用途;`monthDue` 不再上卡片,
  保留为 API 的当期口径输出(测试与 `NextDue` 逻辑仍以它为基准)。
- `web/src/composables/useDebtValidation.ts` — 保存前本地轻校验,规则镜像 `debts.go` 的
  `Validate()`,只为即时反馈(后端 400 一次只回 details[0]);**后端仍是唯一权威**
- `web/src/composables/useCategories.ts` — 分类映射 + 交易展示字段
- `web/src/composables/useThemeColor.ts` — ECharts 取实色 + `themeVersion` 主题触发
- `web/src/styles/variables.css` — 设计 token(surface 阶梯/发丝线/accent/chart 色板,亮/暗双主题)
- `shortcut/` — iOS 快捷指令搭建说明(不入库);AI 提示词已迁入 `server/ai/prompt.md`,
  快捷指令本身不再携带提示词,只上传图片到 `/api/inbox`
