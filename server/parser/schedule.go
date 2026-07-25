package parser

import (
	"sort"
	"time"
)

// 未来还款计划:按月展开「已经确定下来」的出账。
//
// 只展开配置里锁定的分期(额度账户内嵌分期 + 固定分期的 schedule),不外推循环账单余额——
// 后者要成立得假设「期间不再消费也不还款」,越往后越离谱。同理也不推演净资产:
// 还款是 transfer(资产↓、负债↓同额),在没有未来收支预测时未来净资产恒等于今天,推了没信息量。
// 这是现金流口径,与 Summary 的资产负债表口径互不影响。

// ScheduleEntry 一笔确定性还款在某个月的出账。
type ScheduleEntry struct {
	Account     string `json:"account"`
	AccountName string `json:"accountName"` // 卡片/账单展示名,免得前端再推一遍短名
	Name        string `json:"name"`        // 分期名
	Source      string `json:"source"`      // "revolving"(额度账户内嵌分期)| "installment"(固定分期)
	Amount      Amount `json:"amount"`
	DueDate     string `json:"dueDate"`  // 该期实际还款日
	LongTerm    bool   `json:"longTerm"` // 账户在 DebtsConfig.LongTermAccounts 里
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
// 纯函数,不需要 Ledger:展开的是配置里的出账计划,不涉及账户余额。
func ComputeSchedule(cfg *DebtsConfig, from time.Time, months int) []ScheduleMonth {
	if months <= 0 {
		return make([]ScheduleMonth, 0)
	}

	loc := from.Location()
	longTerm := make(map[string]bool, len(cfg.LongTermAccounts))
	for _, account := range cfg.LongTermAccounts {
		longTerm[account] = true
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

	// 账单月从 anchor 前一个月开始扫:首桶的钱可能来自上月账单(due 跨月),
	// 不多扫这一个月首桶会漏。nextDueAfter 最多把 due 推后一个月,故一个月足够。
	for i := -1; i < months; i++ {
		m := anchor.AddDate(0, i, 0)

		for account, rc := range cfg.Revolving {
			cardName := rc.Name
			if cardName == "" {
				cardName = getAccountShortName(account)
			}
			// 该账单月出账 → 账单日之后的第一个还款日,与当期口径同一套工具,月末顺延语义一致
			due := nextDueAfter(clampedDate(m.Year(), m.Month(), rc.BillingDay, loc), rc.DueDay)

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
					DueDate:     due.Format("2006-01-02"),
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
			// InstallmentConfig 没有总期数,只有月供分段:窗口内每月照算。房贷这类确实没有
			// 终止期;车贷/装修贷这类还完后要删掉配置条目,否则计划表会继续往后铺。
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
