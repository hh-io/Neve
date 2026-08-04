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
  (维持后端唯一依赖 gin)。`NEVE_TUNNEL_HOSTNAME` 同时注入服务端,启用公网入口自检。
- 前端 **TypeScript**(`vue-tsc` 校验,契约类型见 `web/src/types/api.ts`),无 UI 库
  (图标用 `@lucide/vue`),无状态管理库(以 composable 模块级单例替代 Pinia:
  `useAnalytics`/`useTheme`/`useToast`/`useDebts`),手写 CSS 变量设计系统
  (`web/src/styles/variables.css`,亮/暗双主题;token 体系见该文件头部注释)。

## 正确性约定(改代码前必读)

- **金额是定点数**:后端一律用 `parser.Amount`(分,int64),禁止 float64 累加金额;
  比值/均值才经 `Amount.Yuan()` 转 float64。JSON 序列化为"元"数字,前端按普通数字消费。
- **单币种 CNY**:非 CNY 数据解析时报 `NON_CNY` 错误并跳过。
- **软失败**:脏数据(不平衡/未 open 账户/非法金额日期等)跳过该笔并记入
  `Ledger.Issues`(带文件:行号),随 `/api/analytics` 的 `parseIssues` 展示在
  `IssuesBanner`;仅 main.bean 无法打开才是硬错误。
- **配置文件解析失败不得降级为空**(debts.json;新增配置文件照此办理):软失败策略只适用于账本
  (脏数据跳过一笔,其余照常),配置文件反过来——空配置在界面上与"从未配置过"无法区分,
  用户随手补一条再保存就把原文件**整份覆盖**,而配置没有第二份真源。故
  `loadDebtsConfig` 只把「文件不存在」当正常(回空),
  解析失败一律 5xx 显式报错;写盘路径再加一道 `quarantineCorrupt`——覆盖前若磁盘上那份
  已无法解析,先改名成 `<name>.corrupt-<时间戳>` 留档。`Refresh()` 是唯一例外
  (走 `longTermAccounts()`):账本刷新与配置无关,读不出来就退回全量净资产口径并记日志,
  不让一个坏配置卡死账本。
- **AI 输出必须过 parser 预校验才可落盘**:`server/api/inbox.go` 的 `validateCandidate`
  先经 `checkTransactionOnly` 拒绝任何非交易顶层行(open/include/option/散文——parser
  会静默忽略或如实执行它们,AI 补一行 open 即可绕过账户白名单),再在临时目录拼
  "真实 open 指令 + 候选交易"试解析,任何 issue 都拒绝写入并回喂 AI
  修正一次;识别提示词的账户列表由 `server/ai.ExtractAccounts` 从账本**原文**提取
  (保留行尾中文注释,parser 结构化数据会丢注释),不要再手工维护账户清单。
  **它会逐层展开 include**(去重 + 循环检测,读不到的 include 跳过):这份输出既是提示词里的
  账户白名单、又是 `validateCandidate` 临时账本里唯一的 open 来源,只扫 main.bean 的话,
  账户 open 在子文件里的用户会遇到「AI 看不见该账户 → 照着账本写反而 UNKNOWN_ACCOUNT
  → 两次尝试全废」。
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
  **前端展示层不许对 `expense` 取绝对值**:净额口径下退款多于消费的那天
  `dailyTrend[].expense` 为负(日粒度很常见,一笔价保退款就够),`Math.abs` 把
  「净退回 10.8」翻成「花了 10.8」,误差是金额的两倍——日历格子、概览热力图、
  趋势折线曾同时犯过。日历带符号展示(`formatExpense`,净退回显示 `+N` 并染
  `--income`、底色走 `has-income`),折线直接用原值让它掉到 0 轴以下,热力图只能
  `Math.max(0, …)` 归零(顺序绿标度画不出负值,与资金流向图丢弃负净额分类同一取舍)。
  月粒度那几处 `Math.abs`(`monthExpense`/环比/结余)因整月净退款不现实而无害,别新增。
- **每个统计字段的时间口径是页面契约的一部分**:`expenseByCategory`、`platformRanking`、
  `merchantRanking`、`incomeBreakdown`、`fundFlow` 都是**本月**口径(「收支分析」页整页同期);
  `weekdayDistribution`(星期分布)、`categoryTrends`(近 6 月)、各 `*Trend` 是**跨期**口径,
  归「趋势图表」页。并排的卡片必须同期:混着放等于宣称同期,而卡片副标题是唯一的口径说明,
  改聚合范围时必须连副标题一起改(字段口径见 `Analytics` struct 与 `types/api.ts` 的注释)。
- **概览与收支分析不给同一个答案**:`expenseByCategory` 同时喂两页,但两页问的问题不同——
  概览是「本月要不要紧张、哪不对劲」,收支分析是「钱具体去哪了」。故构成视图
  (`ExpenseDonut.vue`,环形 + 占比图例)**只留在收支分析页**,概览走
  `ExpenseCategoryBoard.vue`(金额 + 环比 + 上月基数的横条榜)。两页都画环形时,
  收支分析页的头一屏等于白给;而构成比例月月相似,概览拿它答不了「哪不对劲」。
  榜单的环比吃 `CategoryAmount.PrevAmount`(逐类的上月净额,后端算),不是 `categoryTrends`
  ——后者只覆盖 top5,榜单要给每一类都标涨跌。榜里的条**按最大分类归一**表达相对量级
  (按占比会让头部之后的几类几乎等长)。第二行给「上月 ¥X · N 笔」而**不是「占比 X%」**:
  占比正是环形图例那一列,写上去等于又绕回两页重复;而上月金额是环比箭头的基数,
  箭头只说变了多少,基数才让人判断该不该紧张。
- **资金流向图的三层聚合在后端**(`server/parser/fundflow.go` 的 `computeFundFlow`,
  随 `Analytics.FundFlow` 下发,前端只做展示映射):它和同页的 `expenseByCategory` 并排,
  前端自己遍历 `transactions` 就等于把净额、本月、未来日期三条口径各重写一遍,漂一条就露馅
  (原实现丢掉负 Expenses 腿,支出侧比环形图多出整月退款额)。四条不变量:
  ① 退款/收入冲正**按分类等比缩放**该分类的全部链路,总额恒等于 `ExpenseByCategory` /
  `IncomeBreakdown`——不能从某条链路上直接扣,退款到账账户与原消费账户常常不是同一个,
  按链路扣会压出负流量;净额 ≤ 0 的分类整体丢弃(桑基画不出负流量,这与分类卡允许负值是有意差异)。
  ② 资金账户按**付款腿金额占比分摊**(无付款腿则回退收款腿,如工资代扣),取第一条资金腿
  会让归属取决于 posting 书写顺序。③ 分摊尾差落最后一份(`allocateAmount`),否则各链路之和
  与分类合计差几分。④ 节点用**账户全名**做 key,展示名取「在**账本全部资金账户**里不重名的
  最短后缀」——按末段会让 `Assets:Bank:CMB` 与 `Liabilities:CreditCard:CMB` 塌成一个节点;
  消歧范围用全账本而非当月出现的账户,否则同一张卡月月换名。
  节点/链路顺序由后端排稳定,前端 `layoutIterations: 0` 直接消费(开着力导每次 refresh 都重排);
  节点 `depth` 必须按层钉死,否则本月无收入流入的账户(信用卡是常态)会被排到最左列,
  与图例的「收入 | 账户 | 支出」对不上。
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
  账单里,再单独展开就是双重计算;故按 `(账户, 账单月)` 命中当期账单时跳过该期分期
  (**认账单月不认还款日**——账单月是展开循环的变量本身,还款日是它派生的;比派生量等于要求
  `debts.go` 与 `schedule.go` 各自算出同一个日期,账期语义一旦微调就会静默双重计入),顺带让
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
  清单不在账本里,改配置不该触发重解析),该方法幂等,但**就地修改**,只能用在还没发布给
  请求处理的对象上(`Refresh()` 里刚算完那份)。已经放进 `s.analytics` 的必须当只读:
  `handleAnalytics` 锁内只取指针、脱锁才序列化,就地改会与正在编码 JSON 的请求竞争
  (`TestAnalyticsReadWhileDebtsSaved` 用 -race 锁定)。故 `handleSaveDebts` 走
  `WithLongTermLiabilities` 拿副本再换指针,前端 `useDebts.saveDebts` 再静默 `reload()` analytics。
  求和遍历 `AccountBalances` 而非 `LiabilityBreakdown`——后者只收余额为负的账户,
  长期负债多还成正余额时会被漏掉,与 `TotalLiabilities` 口径对不上。
- **数据备份必须由服务端进程做,不能交给独立 launchd/cron 任务**:数据在快捷指令
  App 的 iCloud 容器(`data` 软链指向处),属 macOS TCC 重点保护区。未获授权的
  launchd 进程对该目录 `readdir`/`chdir` 一律 `Operation not permitted`(连 `git add`
  都因 git 要 chdir 进工作树而失败),`stat` 单文件放行但 `open` 读内容也被拒;而
  **服务端进程已获该容器读权限**。故 `server/backup` 采用镜像法:服务端用 `os.ReadFile`
  读账本内容写进 iCloud 外的镜像 git 工作树,git 只对镜像操作(非 iCloud、无 TCC 限制)。
  备份文件清单取自 `Ledger.SourceFiles`(parser 记录实际打开的 main.bean+include 文件,
  单一真源)+ 已知配置名(debts.json);`triggerBackup` 有护栏——账本为空或
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
- **公网入口故障必须靠主动探测发现,被动等日志发现不了**(`server/api/health.go`):
  记账链路是「快捷指令 → Cloudflare edge → tunnel → 本进程」,前两段挂掉时服务端完全无感
  ——本机进程/端口/账本全正常,现象只是"没有请求进来",与"今天没记账"无法区分;而记账是
  单向 fire-and-forget,用户对成功的唯一感知是 Bark 推送,收不到推送时同样分不清这两者。
  实测 tunnel 曾静默断开 10.5 小时,期间每笔记账都石沉大海(请求没到服务端,连
  `data/failed/` 留档都没有),靠人工翻日志才发现。故 `StartHealthChecker` 定期从公网打
  自己的 `/api/inbox`,**拿 401 当唯一健康信号**:ingress 白名单只放行 `^/api/inbox$`,
  加健康端点就要放宽白名单、扩大暴露面;而无令牌请求由 `handleInbox` 在读 body 前返回 401,
  零副作用不占限流额度,且 401 只可能由本进程产生(edge 侧故障给的是 5xx/1033),
  拿到它就证明链路确实通到了应用层。`TestInboxUnauthorizedIsHealthSignal` 锁定这个契约
  ——鉴权改成 403 之类会让自检把正常入口全判成故障。
  连续 `healthFailThreshold` 次失败才告警(tunnel 重连、边缘节点切换有分钟级窗口,
  单次失败不值得推送),持续故障按 `healthAlertInterval` 节流,恢复时推一次并清零计数。
  三个状态字段只由单个 checker goroutine 读写,故不像备份告警那样需要锁。
- **Tunnel 强制走 HTTP/2 而非默认 QUIC**(`deploy/cloudflared-config.yml.in` 的
  `protocol: http2`):本机 Surge 以 TUN 模式运行,`udp-policy-not-supported-behaviour=reject`
  会把代理不支持的 UDP 直接丢弃,cloudflared 的 QUIC(UDP/7844)拨不出去,只反复报
  `no recent network activity` 并无限重试——即上面那次 10.5 小时静默中断的根因。
  HTTP/2 走 TCP/443 不受代理 UDP 能力影响,换网络环境也稳,对每天几笔图片上传无感知代价。
  注意 `--protocol` 已从 `cloudflared --help` 隐藏(2026.7.3 实测仍生效),升级后连不上先查这里;
  cloudflared 自带的启动 precheck 会打印 `suggested_protocol`,可用来复核。
- **stderr 只留"进程活不下去"的信号**:`main` 里 `log.SetOutput(os.Stdout)` 把应用日志
  与 gin 访问日志合到一条时间线(落 `neve.log`,时间格式统一成 `2006/01/02 15:04:05`),
  stderr 只剩启动期致命错误(走 `fatalLog`)与 Recovery 的 panic 栈(落 `neve.error.log`)。
  **`neve.error.log` 非空即真出事**是这个文件的全部价值,别往里写正常信息——用默认 stderr
  时它塞满了启动信息和"记账成功"的交易明细,真正的失败反被淹没,而 newsyslog 给它的
  轮转阈值(1MB)还比访问日志(5MB)小,信噪比最差的那份反倒裁得最快。新增日志一律
  `组件: 中文描述` 前缀(`boot`/`inbox`/`health`/`backup`/`config`),便于按组件 grep。
- **日期按服务器本地时区**解析与归属,部署时用 `TZ` 显式钉死记账时区
  (当前 `Asia/Singapore`,见 `deploy/com.neve.server.plist.in`)。
  同日交易按文件行序稳定排序。
  **前端不许把日期字符串喂给 `new Date()`**:交易日期序列化成带偏移的 RFC3339
  (`2026-07-28T00:00:00+08:00`),`new Date()` 会按浏览器时区重新落点,浏览器偏西时
  整体退一天;纯 `YYYY-MM-DD` 串更糟——它按 UTC 解析。日期比较/展示一律截字符串
  (`useCategories.toDateKey`、`useDebtDisplay.shortDate`、`PaymentSchedule.monthLabel`),
  只有需要星期几时才用 `new Date(y, m-1, d)` 按本地零点重建。真实时间戳
  (`summary.lastUpdated`)不在此列,那本来就该按浏览器时区显示。
- **前端(`web/`)的样式/图表/安全区约定见 `web/CLAUDE.md`**(在 `web/` 下工作时自动加载):
  ECharts 取实色、禁止内联 style、交易页两层 sticky、`--safe-*` 安全区、图表祖先链
  `min-width: 0`、分类中文映射的唯一真源。
- 改解析/统计逻辑必须同步更新 `parser_test.go` / `analytics_test.go`
  (`AnalyzeAt` 可注入时钟,fixture 写 `t.TempDir()`)。

## 关键文件

- `server/parser/parser.go` — 解析器(正则)+ 校验 + ParseIssue 收集
- `server/parser/analytics.go` — 统计与交易分类(`AnalyzeAt`)+ 净资产分层(`ApplyLongTermLiabilities`)
- `server/parser/fundflow.go` — 资金流向三层聚合(`computeFundFlow`,本月口径,净额缩放/资金腿分摊/账户消歧/稳定排序)
- `server/parser/amount.go` — 定点金额类型
- `server/parser/debts.go` — 负债待还计算(`ComputeDebts`,账期/倒计时/剩余期数/schedule 口径)
  + `DebtsConfig`(含 `longTermAccounts`、固定分期的 `endMonth`)
- `server/parser/schedule.go` — 未来还款计划(`ComputeSchedule`,现金流口径,按还款日归月,
  展开确定性分期出账 + 已出账账单 + 已消费未出账)+ 滚动窗口汇总(`SumDueWithin`,
  喂看板的 `due30`/`due90`;**滚动而非自然月**——按自然月算每到月末都会塌成 0)
- `server/api/handler.go` — 路由、analytics 缓存、debts.json 原子写
- `server/api/inbox.go` — 无感记账端点(鉴权、异步识别、预校验、留档、Bark 推送)
- `server/api/health.go` — 公网入口自检(定期从公网打自己的 `/api/inbox`,401 判活,
  连续失败 Bark 告警;tunnel 静默断开只能靠它发现)
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
  保留为 API 的当期口径输出(测试与 `NextDue` 逻辑仍以它为基准)。概览页的「未来 30 天待还」
  卡与这块看板**同源同口径**(`due30` + `nextDue`,逾期才露 `monthRemaining`),
  改口径要两处一起改;它也 `onMounted(loadDebts)`,靠单例的 `loaded` 标志与待还页共用一次请求。
- `web/src/composables/useDebtValidation.ts` — 保存前本地轻校验,规则镜像 `debts.go` 的
  `Validate()`,只为即时反馈(后端 400 一次只回 details[0]);**后端仍是唯一权威**
- `web/src/composables/useCategories.ts` — 分类映射 + 交易展示字段(`processTransaction`)
- `web/src/components/ExpenseDonut.vue` — 支出分类环形图 + 占比图例(只在「收支分析」页)
- `web/src/components/ExpenseCategoryBoard.vue` — 概览的支出分类榜(金额 + 环比 + 上月基数,
  消费 `expenseByCategory` 的 `amount`/`prevAmount`)
- `web/src/components/tabs/SpendingTab.vue` — 收支分析页;资金流向桑基图只消费 `fundFlow`,
  自身不聚合(口径见上「资金流向图的三层聚合在后端」)
- `web/src/components/IncomeBreakdownList.vue` — 本月收入来源条形列表(消费 `incomeBreakdown`)
- `web/src/composables/useThemeColor.ts` — ECharts 取实色 + `themeVersion` 主题触发
- `web/src/styles/variables.css` — 设计 token(surface 阶梯/发丝线/accent/chart 色板,亮/暗双主题)
- `shortcut/` — iOS 快捷指令搭建说明(不入库);AI 提示词已迁入 `server/ai/prompt.md`,
  快捷指令本身不再携带提示词,只上传图片到 `/api/inbox`
