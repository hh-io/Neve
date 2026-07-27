package parser

import (
	"sort"
	"time"
)

// 未来还款计划:按月展开「已经确定下来」的出账。
//
// 展开四类:额度账户内嵌分期、固定分期的 schedule、已出账未还的账单、已消费但未出账的余额。
// 前两类金额来自配置,后两类要靠 ComputeDebts 先从账本算出来(见 statements 参数),
// 但四类都是**已经确定**的钱——已刷掉或已锁定,只是还没到付款日。
//
// 不做的只有一件事:**预测未来还会消费多少**。所以窗口里每张卡最多只有两笔非分期出账
//(当期账单 + 下期账单),再往后就只剩分期,近月天然高于远月。同理也不推演净资产:还款是 transfer
//(资产↓、负债↓同额),在没有未来收支预测时未来净资产恒等于今天,推了没信息量。
// 这是现金流口径,与 Summary 的资产负债表口径互不影响。

// ScheduleEntry 一笔确定性还款在某个月的出账。
type ScheduleEntry struct {
	Account     string `json:"account"`
	AccountName string `json:"accountName"` // 卡片/账单展示名,免得前端再推一遍短名
	Name        string `json:"name"`        // 分期名
	// "revolving"(额度账户内嵌分期)| "installment"(固定分期)
	// | "statement"(已出账未还的账单)| "unbilled"(已消费未出账,下个账单日出账)
	Source   string `json:"source"`
	Amount   Amount `json:"amount"`
	DueDate  string `json:"dueDate"`  // 该期实际还款日
	LongTerm bool   `json:"longTerm"` // 账户在 DebtsConfig.LongTermAccounts 里
}

// ScheduleMonth 一个自然月的还款计划。Entries 恒为非 nil,让前端拿到 [] 而不是 null。
type ScheduleMonth struct {
	Month      string          `json:"month"` // "2026-08"
	Total      Amount          `json:"total"`
	Cumulative Amount          `json:"cumulative"` // 自首月起的前缀和,回答「到某月为止一共要还多少」
	Entries    []ScheduleEntry `json:"entries"`
}

// scheduleMonths 是计划表展开的月数,一年足以覆盖常见的 12/24 期免息中段。
const scheduleMonths = 12

// ComputeSchedule 自 from 所在月起展开 months 个自然月的确定性还款计划。
// from 按天取用(时分秒忽略)。
//
// statements 是 ComputeDebts 已算好的额度账户当期状态,用于把「已出账未还的账单余额」
// 一并计入现金流;传 nil 则只展开配置里的分期。ComputeSchedule 自己**不碰 Ledger**——
// 账单余额要 balanceAsOf 才能得到,让它吃现成结果,本函数就仍是配置 + 时钟的纯函数。
func ComputeSchedule(cfg *DebtsConfig, statements []RevolvingStatus, from time.Time, months int) []ScheduleMonth {
	if months <= 0 {
		return make([]ScheduleMonth, 0)
	}

	loc := from.Location()
	// 还款日是零点,from 带上时分秒就会把"今天到期"判成已过去而整笔丢掉
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	longTerm := make(map[string]bool, len(cfg.LongTermAccounts))
	for _, account := range cfg.LongTermAccounts {
		longTerm[account] = true
	}

	// 本期账单已经把该卡这一期的内嵌分期含在内(debts.go 的 statementDue 只扣未出账部分),
	// 分期再单独展开就是双重计算。故记下每张卡当期账单的还款日,展开分期时跳过同一天那期,
	// 由账单条目整笔代表——顺带让「账单已还清」时该期自然消失(Remaining 已归 0)。
	billed := make(map[string]string, len(statements))
	for _, st := range statements {
		billed[st.Account] = st.DueDate
	}

	anchor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, loc)
	result := make([]ScheduleMonth, 0, months)
	bucketOf := make(map[string]int, months)
	for i := 0; i < months; i++ {
		key := anchor.AddDate(0, i, 0).Format("2006-01")
		bucketOf[key] = i
		result = append(result, ScheduleMonth{Month: key, Entries: make([]ScheduleEntry, 0)})
	}

	// 归桶键是**还款日所在月**而非账单月:现金流口径问的是钱哪个月流出。账单日 25 /
	// 还款日 10 这类配置 due 落到次月,按账单月归桶会让整张表错位一个月。
	// due 早于 from 的期钱已经该流出了(已还,或逾期——后者归 Summary.MonthRemaining 管),
	// 不属于「未来」计划,过滤掉才与 InstallmentStatus.Paid / Overdue 同一口径。
	add := func(due time.Time, e ScheduleEntry) {
		if due.Before(from) {
			return
		}
		i, ok := bucketOf[due.Format("2006-01")]
		if !ok {
			return
		}
		result[i].Entries = append(result[i].Entries, e)
	}

	// 每张卡今天的欠款按「何时流出」拆成三份,三份相加恒等于 CurrentBalance(总量守恒,
	// 且 PaidSince 只在 Remaining 里扣一次,不会重复冲减):
	//   1. Remaining          —— 已出账未还,本期还款日流出
	//   2. Σ UnbilledAmount   —— 未出账分期,由下面的月循环按期展开
	//   3. 余下的             —— 已消费但未出账的普通消费,下个账单日出账
	// 第 3 份同样不是外推:钱已经刷掉了,只是账单日还没到,它必定原样出现在下期账单上。
	// 真正不做的是「预测未来还会消费多少」。
	for _, st := range statements {
		rc, ok := cfg.Revolving[st.Account]
		if !ok {
			continue
		}
		statement, err := time.ParseInLocation("2006-01-02", st.StatementDate, loc)
		if err != nil {
			continue
		}

		// 已出账未还:逾期那张的 due 早于 from,由 add() 剔除——逾期归 Summary.MonthRemaining 管
		if st.Remaining > 0 {
			if due, err := time.ParseInLocation("2006-01-02", st.DueDate, loc); err == nil {
				add(due, ScheduleEntry{
					Account:     st.Account,
					AccountName: st.Name,
					Name:        "本期账单",
					Source:      "statement",
					Amount:      st.Remaining,
					DueDate:     st.DueDate,
					LongTerm:    longTerm[st.Account],
				})
			}
		}

		// 已消费未出账。分期要按**全量**未出账扣(含首期月晚于本期的:那些本金已在
		// CurrentBalance 里,且下面会作为分期条目逐期展开,不扣就双重计入)
		var unbilledInstallments Amount
		for _, it := range st.Installments {
			unbilledInstallments += it.UnbilledAmount
		}
		pending := st.CurrentBalance - st.Remaining - unbilledInstallments
		if pending <= 0 {
			continue
		}
		// 下个账单日出账 → 其后的第一个还款日。月份推进走月初 +1 月再 clamp,
		// 不让 31 号这类账单日在短月进位
		nextMonth := time.Date(statement.Year(), statement.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, 1, 0)
		due := nextDueAfter(clampedDate(nextMonth.Year(), nextMonth.Month(), rc.BillingDay, loc), rc.DueDay)
		add(due, ScheduleEntry{
			Account:     st.Account,
			AccountName: st.Name,
			Name:        "未出账消费",
			Source:      "unbilled",
			Amount:      pending,
			DueDate:     due.Format("2006-01-02"),
			LongTerm:    longTerm[st.Account],
		})
	}

	// 账单月从 anchor 前一个月开始扫:首桶的钱可能来自上月账单(due 跨月),
	// 不多扫这一个月首桶会漏。nextDueAfter 最多把 due 推后一个月,故一个月足够。
	// 固定分期的 due 就在当月、用不着前扫那一个月,但也不为它单开范围判断——
	// 窗口归属只由 add() 一处裁定,拆成两处早晚漂移。
	for i := -1; i < months; i++ {
		m := anchor.AddDate(0, i, 0)

		for account, rc := range cfg.Revolving {
			cardName := rc.Name
			if cardName == "" {
				cardName = getAccountShortName(account)
			}
			// 该账单月出账 → 账单日之后的第一个还款日,与当期口径同一套工具,月末顺延语义一致
			due := nextDueAfter(clampedDate(m.Year(), m.Month(), rc.BillingDay, loc), rc.DueDay)
			dueKey := due.Format("2006-01-02")
			// 这一期已被本期账单整笔覆盖
			if billed[account] == dueKey {
				continue
			}

			for _, it := range rc.Installments {
				first, err := time.Parse("2006-01", it.FirstBillMonth)
				// 非法首期月本应被 Validate 拦下,手改文件绕过时跳过该条而非让整份计划失败
				if err != nil {
					continue
				}
				period := (m.Year()-first.Year())*12 + int(m.Month()) - int(first.Month()) + 1
				amount := installmentPeriodAmount(it, period)
				if amount <= 0 {
					continue
				}
				add(due, ScheduleEntry{
					Account:     account,
					AccountName: cardName,
					Name:        it.Name,
					Source:      "revolving",
					Amount:      amount,
					DueDate:     dueKey,
					LongTerm:    longTerm[account],
				})
			}
		}

		for _, ic := range cfg.Installments {
			name := ic.Name
			if name == "" {
				name = getAccountShortName(ic.Account)
			}
			due := clampedDate(m.Year(), m.Month(), ic.DueDay, loc)
			// 结清后不再出账。与 ComputeDebts 的当期状态**共用 installmentRemaining**,
			// 口径不会漂移;EndMonth 为空(房贷这类无终止期)则窗口内每月照算
			if _, settled := installmentRemaining(ic.EndMonth, due); settled {
				continue
			}
			// 为 0 表示 schedule 全在该期之后,尚未生效
			amount := effectiveMonthly(ic.Schedule, due, loc)
			if amount <= 0 {
				continue
			}
			add(due, ScheduleEntry{
				Account:     ic.Account,
				AccountName: getAccountShortName(ic.Account),
				Name:        name,
				Source:      "installment",
				Amount:      amount,
				DueDate:     due.Format("2006-01-02"),
				LongTerm:    longTerm[ic.Account],
			})
		}
	}

	var cumulative Amount
	for i := range result {
		entries := result[i].Entries
		// Revolving 是 map,遍历序随机,必须排到确定序才能稳定输出
		sort.Slice(entries, func(a, b int) bool {
			if entries[a].DueDate != entries[b].DueDate {
				return entries[a].DueDate < entries[b].DueDate
			}
			if entries[a].Account != entries[b].Account {
				return entries[a].Account < entries[b].Account
			}
			return entries[a].Name < entries[b].Name
		})

		var total Amount
		for _, e := range entries {
			total += e.Amount
		}
		cumulative += total
		result[i].Total = total
		result[i].Cumulative = cumulative
	}

	return result
}
