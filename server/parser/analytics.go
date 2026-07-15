package parser

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Summary holds the financial summary
type Summary struct {
	NetWorth         Amount    `json:"netWorth"`
	TotalAssets      Amount    `json:"totalAssets"`
	TotalLiabilities Amount    `json:"totalLiabilities"`
	MonthIncome      Amount    `json:"monthIncome"`
	MonthExpense     Amount    `json:"monthExpense"`
	MonthBalance     Amount    `json:"monthBalance"`
	TransactionCount int       `json:"transactionCount"`
	TrackingDays     int       `json:"trackingDays"`
	FirstDate        string    `json:"firstDate"`
	LastUpdated      time.Time `json:"lastUpdated"`
}

// CategoryAmount represents expense by category
type CategoryAmount struct {
	Category string  `json:"category"`
	Amount   Amount  `json:"amount"`
	Percent  float64 `json:"percent"`
	Count    int     `json:"count"`
}

// AccountBalance represents an account's balance
type AccountBalance struct {
	Account  string `json:"account"`
	Balance  Amount `json:"balance"`
	Currency string `json:"currency"`
	Type     string `json:"type"`
}

// MonthlyData represents monthly income/expense
type MonthlyData struct {
	Month   string `json:"month"`
	Income  Amount `json:"income"`
	Expense Amount `json:"expense"`
	Balance Amount `json:"balance"`
}

// DailyData represents daily income/expense
type DailyData struct {
	Date    string `json:"date"`
	Income  Amount `json:"income"`
	Expense Amount `json:"expense"`
	Balance Amount `json:"balance"`
}

// WeeklyData represents weekly income/expense
type WeeklyData struct {
	Week      string `json:"week"`      // ISO 周,如 "2026-W03"
	WeekStart string `json:"weekStart"` // 该周周一,如 "2026-01-12"
	Income    Amount `json:"income"`
	Expense   Amount `json:"expense"`
	Balance   Amount `json:"balance"`
}

// TagStats represents spending by tag (platform)
type TagStats struct {
	Tag     string  `json:"tag"`
	Amount  Amount  `json:"amount"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// PayeeStats represents spending by payee (merchant)
type PayeeStats struct {
	Payee  string `json:"payee"`
	Amount Amount `json:"amount"`
	Count  int    `json:"count"`
}

// WeekdayCategoryCount represents count per category for a weekday
type WeekdayCategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
	Amount   Amount `json:"amount"`
}

// WeekdayStats represents spending by day of week
type WeekdayStats struct {
	Weekday           int                    `json:"weekday"` // 0=Sunday, 1=Monday, ...
	Name              string                 `json:"name"`
	Amount            Amount                 `json:"amount"`
	Count             int                    `json:"count"`
	Dates             []string               `json:"dates"`
	CategoryBreakdown []WeekdayCategoryCount `json:"categoryBreakdown"`
}

// MonthlyAmount for category trends
type MonthlyAmount struct {
	Month  string `json:"month"`
	Amount Amount `json:"amount"`
}

// CategoryTrend represents monthly trend for a category
type CategoryTrend struct {
	Category string          `json:"category"`
	Data     []MonthlyAmount `json:"data"`
}

// LiabilityStats represents liability account details
type LiabilityStats struct {
	Account  string `json:"account"`
	Name     string `json:"name"`
	Balance  Amount `json:"balance"`
	Currency string `json:"currency"`
}

// IncomeSource represents income breakdown by source
type IncomeSource struct {
	Source  string  `json:"source"`
	Amount  Amount  `json:"amount"`
	Percent float64 `json:"percent"`
	Count   int     `json:"count"`
}

// Analytics holds all analytical data
type Analytics struct {
	Summary           Summary          `json:"summary"`
	ParseIssues       []ParseIssue     `json:"parseIssues"`
	BalanceChecks     []BalanceCheck   `json:"balanceChecks"`
	ExpenseByCategory []CategoryAmount `json:"expenseByCategory"`
	AccountBalances   []AccountBalance `json:"accountBalances"`
	MonthlyTrend      []MonthlyData    `json:"monthlyTrend"`
	DailyTrend        []DailyData      `json:"dailyTrend"`
	WeeklyTrend       []WeeklyData     `json:"weeklyTrend"`
	// 全量交易(不含 opening),分类/转账字段已由后端计算
	Transactions        []Transaction    `json:"transactions"`
	DailyAverage        float64          `json:"dailyAverage"`
	PlatformRanking     []TagStats       `json:"platformRanking"`
	MerchantRanking     []PayeeStats     `json:"merchantRanking"`
	WeekdayDistribution []WeekdayStats   `json:"weekdayDistribution"`
	CategoryTrends      []CategoryTrend  `json:"categoryTrends"`
	LiabilityBreakdown  []LiabilityStats `json:"liabilityBreakdown"`
	IncomeBreakdown     []IncomeSource   `json:"incomeBreakdown"`
}

// Analyze analyzes the ledger and returns analytics
func Analyze(ledger *Ledger) *Analytics {
	return AnalyzeAt(ledger, time.Now())
}

// AnalyzeAt 以指定时刻为"现在"做统计,便于测试注入固定时钟。
// 不修改 ledger,分类结果写在交易副本上。
func AnalyzeAt(ledger *Ledger, now time.Time) *Analytics {
	analytics := &Analytics{
		Summary:           Summary{LastUpdated: now},
		ParseIssues:       ledger.Issues,
		BalanceChecks:     ledger.BalanceChecks,
		ExpenseByCategory: make([]CategoryAmount, 0),
		AccountBalances:   make([]AccountBalance, 0),
		MonthlyTrend:      make([]MonthlyData, 0),
		Transactions:      make([]Transaction, 0, len(ledger.Transactions)),
	}

	// 分类在交易副本上计算;opening 不进交易列表,未来日期交易进列表但不进统计
	var statsTxs []Transaction
	for _, tx := range ledger.Transactions {
		classifyTransaction(&tx)
		if tx.Kind == "opening" {
			if !tx.Date.After(now) {
				statsTxs = append(statsTxs, tx) // opening 仍参与账户余额
			}
			continue
		}
		analytics.Transactions = append(analytics.Transactions, tx)
		if !tx.Date.After(now) {
			statsTxs = append(statsTxs, tx)
		}
	}

	accountBalances := make(map[string]Amount)
	monthlyData := make(map[string]*MonthlyData)
	dailyData := make(map[string]*DailyData)
	weeklyData := make(map[string]*WeeklyData)
	categoryExpense := make(map[string]struct {
		amount Amount
		count  int
	})

	for _, tx := range statsTxs {
		txMonth := monthKey(tx.Date)
		txDate := tx.Date.Format("2006-01-02")
		weekKey := isoWeekKey(tx.Date)

		if _, ok := monthlyData[txMonth]; !ok {
			monthlyData[txMonth] = &MonthlyData{Month: txMonth}
		}
		if _, ok := dailyData[txDate]; !ok {
			dailyData[txDate] = &DailyData{Date: txDate}
		}
		if _, ok := weeklyData[weekKey]; !ok {
			weeklyData[weekKey] = &WeeklyData{
				Week:      weekKey,
				WeekStart: weekStart(tx.Date).Format("2006-01-02"),
			}
		}

		for _, posting := range tx.Postings {
			accountBalances[posting.Account] += posting.Amount

			// 支出/收入按净额累加:退款记为负数 Expenses posting,应冲减支出而非被忽略
			switch accountRoot(posting.Account) {
			case "Expenses":
				if sameMonth(tx.Date, now) {
					analytics.Summary.MonthExpense += posting.Amount

					category := getExpenseCategory(posting.Account)
					ce := categoryExpense[category]
					ce.amount += posting.Amount
					ce.count++
					categoryExpense[category] = ce
				}
				monthlyData[txMonth].Expense += posting.Amount
				dailyData[txDate].Expense += posting.Amount
				weeklyData[weekKey].Expense += posting.Amount
			case "Income":
				// Income posting 记负数,取反后为正
				if sameMonth(tx.Date, now) {
					analytics.Summary.MonthIncome += -posting.Amount
				}
				monthlyData[txMonth].Income += -posting.Amount
				dailyData[txDate].Income += -posting.Amount
				weeklyData[weekKey].Income += -posting.Amount
			}
		}
	}

	// Net worth and totals
	for account, balance := range accountBalances {
		switch accountRoot(account) {
		case "Assets":
			analytics.Summary.TotalAssets += balance
		case "Liabilities":
			analytics.Summary.TotalLiabilities += -balance // 负债余额为负,展示取反
		}
	}
	analytics.Summary.NetWorth = analytics.Summary.TotalAssets - analytics.Summary.TotalLiabilities
	analytics.Summary.MonthBalance = analytics.Summary.MonthIncome - analytics.Summary.MonthExpense

	// 记账口径:交易总数与记账天数基于全量真实交易(不含 opening、不含未来)
	var firstDate time.Time
	count := 0
	for _, tx := range statsTxs {
		if tx.Kind == "opening" {
			continue
		}
		count++
		if firstDate.IsZero() || tx.Date.Before(firstDate) {
			firstDate = tx.Date
		}
	}
	analytics.Summary.TransactionCount = count
	if !firstDate.IsZero() {
		analytics.Summary.FirstDate = firstDate.Format("2006-01-02")
		analytics.Summary.TrackingDays = int(now.Sub(firstDate).Hours()/24) + 1
	}

	// Expense by category (current month)
	var totalExpense Amount
	for _, ce := range categoryExpense {
		totalExpense += ce.amount
	}
	for category, ce := range categoryExpense {
		percent := 0.0
		if totalExpense > 0 {
			percent = float64(ce.amount) / float64(totalExpense) * 100
		}
		analytics.ExpenseByCategory = append(analytics.ExpenseByCategory, CategoryAmount{
			Category: category,
			Amount:   ce.amount,
			Percent:  percent,
			Count:    ce.count,
		})
	}
	sort.Slice(analytics.ExpenseByCategory, func(i, j int) bool {
		return analytics.ExpenseByCategory[i].Amount > analytics.ExpenseByCategory[j].Amount
	})

	// Account balances (only Assets and Liabilities)
	for account, balance := range accountBalances {
		root := accountRoot(account)
		if root == "Assets" || root == "Liabilities" {
			analytics.AccountBalances = append(analytics.AccountBalances, AccountBalance{
				Account:  account,
				Balance:  balance,
				Currency: ledger.BaseCurrency,
				Type:     root,
			})
		}
	}
	sort.Slice(analytics.AccountBalances, func(i, j int) bool {
		return analytics.AccountBalances[i].Balance > analytics.AccountBalances[j].Balance
	})

	// 月锚点取每月 1 号,避免 AddDate 在月末(如 31 号)回退时跳月
	monthAnchor := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	var last6Months []string
	for i := 5; i >= 0; i-- {
		last6Months = append(last6Months, monthKey(monthAnchor.AddDate(0, -i, 0)))
	}
	for _, month := range last6Months {
		data := monthlyData[month]
		if data == nil {
			data = &MonthlyData{Month: month}
		}
		data.Balance = data.Income - data.Expense
		analytics.MonthlyTrend = append(analytics.MonthlyTrend, *data)
	}

	// Daily trend (last 30 days)
	for i := 29; i >= 0; i-- {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		data := dailyData[day]
		if data == nil {
			data = &DailyData{Date: day}
		}
		data.Balance = data.Income - data.Expense
		analytics.DailyTrend = append(analytics.DailyTrend, *data)
	}

	// Weekly trend (last 8 weeks):按 7 天步进,ISO 周天然不重复
	for i := 7; i >= 0; i-- {
		d := now.AddDate(0, 0, -i*7)
		weekKey := isoWeekKey(d)
		data := weeklyData[weekKey]
		if data == nil {
			data = &WeeklyData{
				Week:      weekKey,
				WeekStart: weekStart(d).Format("2006-01-02"),
			}
		}
		data.Balance = data.Income - data.Expense
		analytics.WeeklyTrend = append(analytics.WeeklyTrend, *data)
	}

	// ========== 排行 / 分布 / 趋势 ==========

	tagSpending := make(map[string]struct {
		amount Amount
		count  int
	})
	payeeSpending := make(map[string]struct {
		amount Amount
		count  int
	})
	weekdaySpending := make(map[int]struct {
		amount     Amount
		count      int
		dates      []string
		categories map[string]struct {
			count  int
			amount Amount
		}
	})
	categoryMonthly := make(map[string]map[string]Amount)
	incomeBySource := make(map[string]struct {
		amount Amount
		count  int
	})

	for _, tx := range statsTxs {
		if tx.Kind == "opening" {
			continue
		}

		var txExpenseAmount Amount
		txCategories := make(map[string]Amount)

		for _, posting := range tx.Postings {
			switch accountRoot(posting.Account) {
			case "Expenses":
				// 净额口径:退款冲减对应分类
				txExpenseAmount += posting.Amount

				category := getExpenseCategory(posting.Account)
				txMonth := monthKey(tx.Date)
				if categoryMonthly[category] == nil {
					categoryMonthly[category] = make(map[string]Amount)
				}
				categoryMonthly[category][txMonth] += posting.Amount
				txCategories[category] += posting.Amount
			case "Income":
				source := getIncomeSource(posting.Account)
				is := incomeBySource[source]
				is.amount += -posting.Amount
				is.count++
				incomeBySource[source] = is
			}
		}

		if txExpenseAmount > 0 {
			weekday := int(tx.Date.Weekday())
			ws := weekdaySpending[weekday]
			ws.amount += txExpenseAmount
			ws.count++

			if ws.categories == nil {
				ws.categories = make(map[string]struct {
					count  int
					amount Amount
				})
			}
			for cat, amt := range txCategories {
				catData := ws.categories[cat]
				catData.count++
				catData.amount += amt
				ws.categories[cat] = catData
			}

			dateStr := tx.Date.Format("01-02")
			found := false
			for _, d := range ws.dates {
				if d == dateStr {
					found = true
					break
				}
			}
			if !found {
				ws.dates = append(ws.dates, dateStr)
			}
			weekdaySpending[weekday] = ws

			for _, tag := range tx.Tags {
				ts := tagSpending[tag]
				ts.amount += txExpenseAmount
				ts.count++
				tagSpending[tag] = ts
			}

			if tx.Payee != "" {
				ps := payeeSpending[tx.Payee]
				ps.amount += txExpenseAmount
				ps.count++
				payeeSpending[tx.Payee] = ps
			}
		}
	}

	// Daily average:本月支出 / 本月已过天数
	analytics.DailyAverage = analytics.Summary.MonthExpense.Yuan() / float64(now.Day())

	// Platform ranking
	var totalTagSpending Amount
	for _, ts := range tagSpending {
		totalTagSpending += ts.amount
	}
	for tag, ts := range tagSpending {
		percent := 0.0
		if totalTagSpending > 0 {
			percent = float64(ts.amount) / float64(totalTagSpending) * 100
		}
		analytics.PlatformRanking = append(analytics.PlatformRanking, TagStats{
			Tag:     tag,
			Amount:  ts.amount,
			Count:   ts.count,
			Percent: percent,
		})
	}
	sort.Slice(analytics.PlatformRanking, func(i, j int) bool {
		return analytics.PlatformRanking[i].Amount > analytics.PlatformRanking[j].Amount
	})
	if len(analytics.PlatformRanking) > 10 {
		analytics.PlatformRanking = analytics.PlatformRanking[:10]
	}

	// Merchant ranking
	for payee, ps := range payeeSpending {
		analytics.MerchantRanking = append(analytics.MerchantRanking, PayeeStats{
			Payee:  payee,
			Amount: ps.amount,
			Count:  ps.count,
		})
	}
	sort.Slice(analytics.MerchantRanking, func(i, j int) bool {
		return analytics.MerchantRanking[i].Amount > analytics.MerchantRanking[j].Amount
	})
	if len(analytics.MerchantRanking) > 10 {
		analytics.MerchantRanking = analytics.MerchantRanking[:10]
	}

	// Weekday distribution
	weekdayNames := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	for i := 0; i < 7; i++ {
		ws := weekdaySpending[i]

		var catBreakdown []WeekdayCategoryCount
		for cat, data := range ws.categories {
			catBreakdown = append(catBreakdown, WeekdayCategoryCount{
				Category: cat,
				Count:    data.count,
				Amount:   data.amount,
			})
		}
		sort.Slice(catBreakdown, func(a, b int) bool {
			return catBreakdown[a].Count > catBreakdown[b].Count
		})

		analytics.WeekdayDistribution = append(analytics.WeekdayDistribution, WeekdayStats{
			Weekday:           i,
			Name:              weekdayNames[i],
			Amount:            ws.amount,
			Count:             ws.count,
			Dates:             ws.dates,
			CategoryBreakdown: catBreakdown,
		})
	}

	// Category trends (top 5 categories, last 6 months)
	var topCategories []string
	for _, cat := range analytics.ExpenseByCategory {
		topCategories = append(topCategories, cat.Category)
		if len(topCategories) >= 5 {
			break
		}
	}
	for _, category := range topCategories {
		trend := CategoryTrend{Category: category}
		monthData := categoryMonthly[category]
		for _, month := range last6Months {
			trend.Data = append(trend.Data, MonthlyAmount{
				Month:  month,
				Amount: monthData[month],
			})
		}
		analytics.CategoryTrends = append(analytics.CategoryTrends, trend)
	}

	// Liability breakdown
	for account, balance := range accountBalances {
		if accountRoot(account) == "Liabilities" && balance < 0 {
			analytics.LiabilityBreakdown = append(analytics.LiabilityBreakdown, LiabilityStats{
				Account:  account,
				Name:     getAccountShortName(account),
				Balance:  -balance,
				Currency: ledger.BaseCurrency,
			})
		}
	}
	sort.Slice(analytics.LiabilityBreakdown, func(i, j int) bool {
		return analytics.LiabilityBreakdown[i].Balance > analytics.LiabilityBreakdown[j].Balance
	})

	// Income breakdown
	var totalIncome Amount
	for _, is := range incomeBySource {
		totalIncome += is.amount
	}
	for source, is := range incomeBySource {
		percent := 0.0
		if totalIncome > 0 {
			percent = float64(is.amount) / float64(totalIncome) * 100
		}
		analytics.IncomeBreakdown = append(analytics.IncomeBreakdown, IncomeSource{
			Source:  source,
			Amount:  is.amount,
			Percent: percent,
			Count:   is.count,
		})
	}
	sort.Slice(analytics.IncomeBreakdown, func(i, j int) bool {
		return analytics.IncomeBreakdown[i].Amount > analytics.IncomeBreakdown[j].Amount
	})

	return analytics
}

// classifyTransaction 计算交易的展示分类。
// 统计口径始终按 posting 级聚合,这里只决定交易在列表中"是什么、显示多少钱"。
func classifyTransaction(tx *Transaction) {
	var posReal, negReal, expNet, incNet Amount
	hasEquity := false
	for _, po := range tx.Postings {
		switch accountRoot(po.Account) {
		case "Equity":
			hasEquity = true
		case "Assets", "Liabilities":
			if po.Amount > 0 {
				posReal += po.Amount
			} else {
				negReal += -po.Amount
			}
		case "Expenses":
			expNet += po.Amount
		case "Income":
			incNet += -po.Amount // Income posting 记负数,取反后为正
		}
	}

	switch {
	case hasEquity:
		tx.Kind = "opening"

	case incNet > 0 && expNet > 0:
		// 收支混合(如税前工资拆税):按净到手展示,统计仍按 posting 分别计入
		tx.Kind = "mixed"
		tx.DisplayAmount = incNet - expNet
		tx.Category = firstCategory(tx, "Income", getIncomeSource)

	case incNet > 0:
		tx.Kind = "income"
		tx.DisplayAmount = incNet
		tx.Category = firstCategory(tx, "Income", getIncomeSource)

	case minAmount(posReal, negReal) > 0:
		// 资金在真实账户(Assets/Liabilities)间对流:转账/还款,本金取对流较小侧,
		// 多出的 Expenses 部分是附带手续费——修复"还款 5000+手续费 5 显示成 5 元支出"
		tx.Kind = "transfer"
		tx.IsTransfer = true
		tx.TransferAmount = minAmount(posReal, negReal)
		tx.FeeAmount = maxAmount(expNet, 0)
		tx.DisplayAmount = tx.TransferAmount
		tx.Category = "Financial"

	default:
		// 普通支出;退款时 expNet 为负,前端按负数金额展示
		tx.Kind = "expense"
		tx.DisplayAmount = expNet
		tx.Category = primaryExpenseCategory(tx)
	}
}

// primaryExpenseCategory 取金额最大的 Expenses posting 的分类
func primaryExpenseCategory(tx *Transaction) string {
	best := ""
	var bestAmount Amount
	for _, po := range tx.Postings {
		if accountRoot(po.Account) != "Expenses" {
			continue
		}
		abs := po.Amount
		if abs < 0 {
			abs = -abs
		}
		if best == "" || abs > bestAmount {
			best = getExpenseCategory(po.Account)
			bestAmount = abs
		}
	}
	if best == "" {
		return "Other"
	}
	return best
}

// firstCategory 取第一个指定类型 posting 的分类
func firstCategory(tx *Transaction, root string, extract func(string) string) string {
	for _, po := range tx.Postings {
		if accountRoot(po.Account) == root {
			return extract(po.Account)
		}
	}
	return "Other"
}

// ---- 时间口径(单一真相源,前后统计共用) ----

func isoWeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// weekStart 返回 t 所在 ISO 周的周一零点
func weekStart(t time.Time) time.Time {
	offset := (int(t.Weekday()) + 6) % 7 // Monday=0
	d := t.AddDate(0, 0, -offset)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.Local)
}

func monthKey(t time.Time) string {
	return t.Format("2006-01")
}

func sameMonth(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month()
}

// ---- 账户工具 ----

// accountRoot 按 ':' 分段取首段,避免前缀比较把 AssetsFoo 误判为资产
func accountRoot(account string) string {
	return strings.SplitN(account, ":", 2)[0]
}

func getExpenseCategory(account string) string {
	parts := strings.Split(account, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "Other"
}

func getIncomeSource(account string) string {
	parts := strings.Split(account, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "Other"
}

func getAccountShortName(account string) string {
	parts := strings.Split(account, ":")
	return parts[len(parts)-1]
}

func minAmount(a, b Amount) Amount {
	if a < b {
		return a
	}
	return b
}

func maxAmount(a, b Amount) Amount {
	if a > b {
		return a
	}
	return b
}
