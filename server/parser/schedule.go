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
	result := make([]ScheduleMonth, 0, maxInt(months, 0))
	if months <= 0 {
		return result
	}

	loc := from.Location()
	longTerm := make(map[string]bool, len(cfg.LongTermAccounts))
	for _, account := range cfg.LongTermAccounts {
		longTerm[account] = true
	}

	anchor := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, loc)
	var cumulative Amount

	for i := 0; i < months; i++ {
		m := anchor.AddDate(0, i, 0)
		entries := make([]ScheduleEntry, 0)

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
				entries = append(entries, ScheduleEntry{
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
			// 房贷类没有终止期,窗口内每月都有条目——现金流确实每月流出;
			// 为 0 表示 schedule 全在该期之后,尚未生效
			amount := effectiveMonthly(ic.Schedule, due, loc)
			if amount <= 0 {
				continue
			}
			entries = append(entries, ScheduleEntry{
				Account:     ic.Account,
				AccountName: getAccountShortName(ic.Account),
				Name:        name,
				Source:      "installment",
				Amount:      amount,
				DueDate:     due.Format("2006-01-02"),
				LongTerm:    longTerm[ic.Account],
			})
		}

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

		result = append(result, ScheduleMonth{
			Month:      m.Format("2006-01"),
			Total:      total,
			Cumulative: cumulative,
			Entries:    entries,
		})
	}

	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
