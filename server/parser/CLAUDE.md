# Neve 解析与统计(`server/parser/`)

资金流向、负债待还、还款计划、净资产分层的口径约定。全局硬约束(定点金额、软失败、
交易分类、时间口径)见根目录 `CLAUDE.md`。

- **资金流向图的三层聚合在后端**(`fundflow.go` 的 `computeFundFlow`,
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
- **负债待还口径**(`debts.go` 的 `ComputeDebts`,配置存 `data/debts.json`):
  额度类"本期应还"= 账单日当天结束时的欠款余额快照,**先扣减内嵌免息分期的未出账金额**
  (`RevolvingConfig.Installments`:分期消费记账时全额入负债账户,银行按月出账;
  未出账 = 总额 − 已出账期数×每期金额,尾差落最后一期,月数按 `YYYY-MM` 差值算无进位坑;
  **首期账单月晚于当前账单月的分期不参与扣减**——账单日后新购,本金不在快照里,扣了是双重扣减);
  冲减按账单日后转入该账户的
  **正向 posting**(不限交易 kind,退款/返现也应冲减);分期类"已还"只认 `kind=transfer`。
  账单日/还款日超出当月天数时**顺延至月末**(`clampedDate`,严禁裸 `time.Date` 进位)。
  GET /api/debts 每次用缓存 Ledger 现算,配置变更无需 refresh。
- **未来还款计划是现金流口径,不是资产负债表口径**(`schedule.go` 的
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
  喂看板的 `due30`/`due90` 走 `SumDueWithin` 汇总,**滚动而非自然月**——按自然月算每到月末
  都会塌成 0。
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
  请求处理的对象上(`Refresh()` 里刚算完那份);已经发布出去的必须当只读,并发契约见
  `server/api/CLAUDE.md`。
  求和遍历 `AccountBalances` 而非 `LiabilityBreakdown`——后者只收余额为负的账户,
  长期负债多还成正余额时会被漏掉,与 `TotalLiabilities` 口径对不上。
