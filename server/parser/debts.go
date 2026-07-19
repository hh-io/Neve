package parser

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// ========== debts.json 配置 ==========

// DebtsConfig 对应 data/debts.json,由前端编辑、api 层持久化。
type DebtsConfig struct {
	Revolving    map[string]RevolvingConfig `json:"revolving"`
	Installments []InstallmentConfig        `json:"installments"`
}

// RevolvingConfig 额度类账单(信用卡/白条/月付等):
// 本期应还取账单日当天结束时的欠款余额快照,自动涵盖上期未还清部分。
type RevolvingConfig struct {
	Name       string `json:"name"` // 为空时回退账户短名
	BillingDay int    `json:"billingDay"`
	DueDay     int    `json:"dueDay"`
}

// InstallmentConfig 分期类账单(房贷/车贷):每月固定金额。
// 月供调整靠追加 schedule 条目而非改历史,旧账期的口径才不会被静默改写。
type InstallmentConfig struct {
	ID       string             `json:"id"`
	Name     string             `json:"name"`
	Account  string             `json:"account"`
	DueDay   int                `json:"dueDay"`
	Schedule []InstallmentPhase `json:"schedule"`
}

// InstallmentPhase 一段生效期的月供金额,自 effectiveFrom 起生效,直到被更晚的条目覆盖。
type InstallmentPhase struct {
	EffectiveFrom string `json:"effectiveFrom"` // "2026-01-01"
	Amount        Amount `json:"amount"`
}

// Validate 返回人话错误列表,为空表示通过。
// 账户是否存在于账本不在此校验:允许先配置后补 open,缺失在报告里标 accountMissing。
func (c *DebtsConfig) Validate() []string {
	var errs []string
	for account, rc := range c.Revolving {
		if accountRoot(account) != "Liabilities" {
			errs = append(errs, fmt.Sprintf("额度类账户 %q 必须以 Liabilities: 开头", account))
		}
		if rc.BillingDay < 1 || rc.BillingDay > 31 {
			errs = append(errs, fmt.Sprintf("账户 %s 的账单日 %d 不在 1-31 范围", account, rc.BillingDay))
		}
		if rc.DueDay < 1 || rc.DueDay > 31 {
			errs = append(errs, fmt.Sprintf("账户 %s 的还款日 %d 不在 1-31 范围", account, rc.DueDay))
		}
	}
	seenIDs := make(map[string]bool)
	for _, ic := range c.Installments {
		label := ic.Name
		if label == "" {
			label = ic.ID
		}
		if ic.ID == "" {
			errs = append(errs, fmt.Sprintf("分期 %q 缺少 id", ic.Name))
		} else if seenIDs[ic.ID] {
			errs = append(errs, fmt.Sprintf("分期 id %q 重复", ic.ID))
		}
		seenIDs[ic.ID] = true
		if accountRoot(ic.Account) != "Liabilities" {
			errs = append(errs, fmt.Sprintf("分期 %s 的账户 %q 必须以 Liabilities: 开头", label, ic.Account))
		}
		if ic.DueDay < 1 || ic.DueDay > 31 {
			errs = append(errs, fmt.Sprintf("分期 %s 的还款日 %d 不在 1-31 范围", label, ic.DueDay))
		}
		if len(ic.Schedule) == 0 {
			errs = append(errs, fmt.Sprintf("分期 %s 的 schedule 不能为空", label))
		}
		for _, ph := range ic.Schedule {
			if _, err := time.Parse("2006-01-02", ph.EffectiveFrom); err != nil {
				errs = append(errs, fmt.Sprintf("分期 %s 的生效日期 %q 非法,应为 YYYY-MM-DD", label, ph.EffectiveFrom))
			}
			if ph.Amount <= 0 {
				errs = append(errs, fmt.Sprintf("分期 %s 的月供金额必须大于 0", label))
			}
		}
	}
	return errs
}

// Normalize 保存前规范化:schedule 按生效日期升序,追加的调整不会乱序落盘。
func (c *DebtsConfig) Normalize() {
	for i := range c.Installments {
		schedule := c.Installments[i].Schedule
		sort.Slice(schedule, func(a, b int) bool {
			return schedule[a].EffectiveFrom < schedule[b].EffectiveFrom
		})
	}
}

// ========== 计算结果 ==========

// DebtsReport 是 GET /api/debts 的计算结果,金额 JSON 序列化为元。
type DebtsReport struct {
	Summary      DebtsSummary        `json:"summary"`
	Revolving    []RevolvingStatus   `json:"revolving"`
	Installments []InstallmentStatus `json:"installments"`
	// 有欠款但未配置的负债账户,提示用户补配置
	Unconfigured []LiabilityStats `json:"unconfigured"`
}

// DebtsSummary 全局看板:本期总应还、剩余待还、最近一个未结清的最后还款日。
type DebtsSummary struct {
	MonthDue       Amount `json:"monthDue"`
	MonthRemaining Amount `json:"monthRemaining"`
	NextDueDate    string `json:"nextDueDate"` // 空串表示本期已全部结清
	NextDueName    string `json:"nextDueName"`
	NextDueInDays  int    `json:"nextDueInDays"` // 负数 = 已逾期天数
	OverdueCount   int    `json:"overdueCount"`
}

// RevolvingStatus 额度类账户的当期状态。
type RevolvingStatus struct {
	Account        string `json:"account"`
	Name           string `json:"name"`
	AccountMissing bool   `json:"accountMissing"`
	StatementDate  string `json:"statementDate"`
	DueDate        string `json:"dueDate"`
	StatementDue   Amount `json:"statementDue"`
	PaidSince      Amount `json:"paidSince"`
	Remaining      Amount `json:"remaining"`
	CurrentBalance Amount `json:"currentBalance"`
	DaysUntilDue   int    `json:"daysUntilDue"`
	Overdue        bool   `json:"overdue"`
}

// InstallmentStatus 分期类账单的当期状态。
type InstallmentStatus struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Account        string `json:"account"`
	AccountMissing bool   `json:"accountMissing"`
	MonthlyAmount  Amount `json:"monthlyAmount"` // 0 表示本期尚无生效月供(schedule 全在未来)
	DueDate        string `json:"dueDate"`
	Paid           bool   `json:"paid"`
	PaidAmount     Amount `json:"paidAmount"`
	DaysUntilDue   int    `json:"daysUntilDue"`
	Overdue        bool   `json:"overdue"`
	CurrentBalance Amount `json:"currentBalance"`
}

// ComputeDebts 以 now 为"现在"计算负债还款报告,便于测试注入固定时钟。不修改 ledger。
func ComputeDebts(ledger *Ledger, cfg *DebtsConfig, now time.Time) *DebtsReport {
	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	// 分类写在交易副本上(与 AnalyzeAt 同口径)。必须用 Ledger 全量交易:
	// opening 交易设定负债初始欠款,Analytics.Transactions 里没有它
	var txs []Transaction
	for _, tx := range ledger.Transactions {
		if tx.Date.After(now) {
			continue
		}
		classifyTransaction(&tx)
		txs = append(txs, tx)
	}

	known := make(map[string]bool, len(ledger.Accounts))
	for _, acc := range ledger.Accounts {
		known[acc.Name] = true
	}

	report := &DebtsReport{
		Revolving:    make([]RevolvingStatus, 0, len(cfg.Revolving)),
		Installments: make([]InstallmentStatus, 0, len(cfg.Installments)),
		Unconfigured: make([]LiabilityStats, 0),
	}

	type dueCandidate struct {
		date time.Time
		name string
	}
	var candidates []dueCandidate

	for account, rc := range cfg.Revolving {
		name := rc.Name
		if name == "" {
			name = getAccountShortName(account)
		}
		statement := latestStatementDate(today, rc.BillingDay)
		due := nextDueAfter(statement, rc.DueDay)

		statementDue := maxAmount(-balanceAsOf(txs, account, statement), 0)
		paidSince := creditsAfter(txs, account, statement, false)
		// 超额还款/大额退款不出负数
		remaining := maxAmount(statementDue-paidSince, 0)
		overdue := remaining > 0 && today.After(due)

		report.Revolving = append(report.Revolving, RevolvingStatus{
			Account:        account,
			Name:           name,
			AccountMissing: !known[account],
			StatementDate:  statement.Format("2006-01-02"),
			DueDate:        due.Format("2006-01-02"),
			StatementDue:   statementDue,
			PaidSince:      paidSince,
			Remaining:      remaining,
			CurrentBalance: maxAmount(-balanceAsOf(txs, account, today), 0),
			DaysUntilDue:   daysBetween(today, due),
			Overdue:        overdue,
		})

		report.Summary.MonthDue += statementDue
		report.Summary.MonthRemaining += remaining
		if overdue {
			report.Summary.OverdueCount++
		}
		if remaining > 0 {
			candidates = append(candidates, dueCandidate{due, name})
		}
	}
	sort.Slice(report.Revolving, func(i, j int) bool {
		if report.Revolving[i].DueDate != report.Revolving[j].DueDate {
			return report.Revolving[i].DueDate < report.Revolving[j].DueDate
		}
		return report.Revolving[i].Account < report.Revolving[j].Account
	})

	for _, ic := range cfg.Installments {
		name := ic.Name
		if name == "" {
			name = getAccountShortName(ic.Account)
		}
		dueDate := clampedDate(today.Year(), today.Month(), ic.DueDay, loc)
		prevMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
		prevDue := clampedDate(prevMonth.Year(), prevMonth.Month(), ic.DueDay, loc)

		monthly := effectiveMonthly(ic.Schedule, dueDate, loc)
		// 已还窗口取 (上月还款日, now]:迟还几天仍归本期,状态从逾期翻成已还
		paidAmount := creditsAfter(txs, ic.Account, prevDue, true)
		paid := paidAmount > 0
		overdue := monthly > 0 && !paid && today.After(dueDate)

		report.Installments = append(report.Installments, InstallmentStatus{
			ID:             ic.ID,
			Name:           name,
			Account:        ic.Account,
			AccountMissing: !known[ic.Account],
			MonthlyAmount:  monthly,
			DueDate:        dueDate.Format("2006-01-02"),
			Paid:           paid,
			PaidAmount:     paidAmount,
			DaysUntilDue:   daysBetween(today, dueDate),
			Overdue:        overdue,
			CurrentBalance: maxAmount(-balanceAsOf(txs, ic.Account, today), 0),
		})

		if monthly > 0 {
			report.Summary.MonthDue += monthly
			if overdue {
				report.Summary.OverdueCount++
			}
			if !paid {
				report.Summary.MonthRemaining += monthly
				candidates = append(candidates, dueCandidate{dueDate, name})
			}
		}
	}

	// 最紧急的还款日:逾期的日期天然早于未来的,直接取最早即可
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].date.Equal(candidates[j].date) {
			return candidates[i].date.Before(candidates[j].date)
		}
		return candidates[i].name < candidates[j].name
	})
	if len(candidates) > 0 {
		next := candidates[0]
		report.Summary.NextDueDate = next.date.Format("2006-01-02")
		report.Summary.NextDueName = next.name
		report.Summary.NextDueInDays = daysBetween(today, next.date)
	}

	// 有欠款但未纳入任何配置的负债账户
	configured := make(map[string]bool, len(cfg.Revolving)+len(cfg.Installments))
	for account := range cfg.Revolving {
		configured[account] = true
	}
	for _, ic := range cfg.Installments {
		configured[ic.Account] = true
	}
	balances := make(map[string]Amount)
	for _, tx := range txs {
		for _, p := range tx.Postings {
			balances[p.Account] += p.Amount
		}
	}
	for account, bal := range balances {
		if accountRoot(account) == "Liabilities" && bal < 0 && !configured[account] {
			report.Unconfigured = append(report.Unconfigured, LiabilityStats{
				Account:  account,
				Name:     getAccountShortName(account),
				Balance:  -bal,
				Currency: ledger.BaseCurrency,
			})
		}
	}
	sort.Slice(report.Unconfigured, func(i, j int) bool {
		if report.Unconfigured[i].Balance != report.Unconfigured[j].Balance {
			return report.Unconfigured[i].Balance > report.Unconfigured[j].Balance
		}
		return report.Unconfigured[i].Account < report.Unconfigured[j].Account
	})

	return report
}

// effectiveMonthly 取 effectiveFrom ≤ 本期还款日的最后一条月供。
// 对齐还款日而非 today:1 月 1 日生效的新月供应作用于 1 月 20 日那期,即便今天才 1 月 5 日。
func effectiveMonthly(schedule []InstallmentPhase, dueDate time.Time, loc *time.Location) Amount {
	var best time.Time
	var amount Amount
	for _, ph := range schedule {
		d, err := time.ParseInLocation("2006-01-02", ph.EffectiveFrom, loc)
		// 非法日期本应被 Validate 拦下,手改文件绕过时按未生效跳过,不让整份报告失败
		if err != nil || d.After(dueDate) {
			continue
		}
		if best.IsZero() || !d.Before(best) {
			best = d
			amount = ph.Amount
		}
	}
	return amount
}

// balanceAsOf 账户在 day 当天结束时的余额。快照是纯求和,与同日行序无关。
func balanceAsOf(txs []Transaction, account string, day time.Time) Amount {
	var total Amount
	for _, tx := range txs {
		if tx.Date.After(day) {
			continue
		}
		for _, p := range tx.Postings {
			if p.Account == account {
				total += p.Amount
			}
		}
	}
	return total
}

// creditsAfter 统计 (after, now] 内转入账户的正向 posting 合计(txs 已截止到 now)。
// 额度类不限交易类型:还款、退款、返现都应冲减本期应还,只认 transfer 会漏掉退款;
// 分期类限 kind=transfer,避免记账修正类的正向 posting 被误判为月供已还。
// 正向 posting 金额天然是本金——手续费/利息落在 Expenses 腿上。
func creditsAfter(txs []Transaction, account string, after time.Time, transferOnly bool) Amount {
	var total Amount
	for _, tx := range txs {
		if !tx.Date.After(after) {
			continue
		}
		if transferOnly && tx.Kind != "transfer" {
			continue
		}
		for _, p := range tx.Postings {
			if p.Account == account && p.Amount > 0 {
				total += p.Amount
			}
		}
	}
	return total
}

// ---- 账期日期工具 ----

// clampedDate 返回指定年月 day 号的零点;day 超出当月天数时取月末最后一天。
// 不能直接 time.Date(y, 2, 30):Go 会归一化进位成 3 月 2 日,而账单语义是顺延至月末。
func clampedDate(year int, month time.Month, day int, loc *time.Location) time.Time {
	if last := daysInMonth(year, month, loc); day > last {
		day = last
	}
	return time.Date(year, month, day, 0, 0, 0, 0, loc)
}

// daysInMonth 利用 day=0 归一化为上月最后一天来求当月天数
func daysInMonth(year int, month time.Month, loc *time.Location) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
}

// latestStatementDate 返回最近一个不晚于 today 的账单日:账单一出即为"本期",直到下次出账。
func latestStatementDate(today time.Time, billingDay int) time.Time {
	loc := today.Location()
	cand := clampedDate(today.Year(), today.Month(), billingDay, loc)
	if cand.After(today) {
		prev := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
		cand = clampedDate(prev.Year(), prev.Month(), billingDay, loc)
	}
	return cand
}

// nextDueAfter 返回账单日之后第一个还款日,单调规则天然覆盖同月与跨月(账单 25 还款 10)。
func nextDueAfter(statement time.Time, dueDay int) time.Time {
	loc := statement.Location()
	cand := clampedDate(statement.Year(), statement.Month(), dueDay, loc)
	if !cand.After(statement) {
		next := time.Date(statement.Year(), statement.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, 1, 0)
		cand = clampedDate(next.Year(), next.Month(), dueDay, loc)
	}
	return cand
}

// daysBetween 计算自然日差(两端均为零点),四舍五入抵御夏令时造成的非整 24 小时。
func daysBetween(from, to time.Time) int {
	return int(math.Round(to.Sub(from).Hours() / 24))
}
