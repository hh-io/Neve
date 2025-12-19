package parser

import (
	"sort"
	"time"
)

// Summary holds the financial summary
type Summary struct {
	NetWorth         float64   `json:"netWorth"`
	TotalAssets      float64   `json:"totalAssets"`
	TotalLiabilities float64   `json:"totalLiabilities"`
	MonthIncome      float64   `json:"monthIncome"`
	MonthExpense     float64   `json:"monthExpense"`
	MonthBalance     float64   `json:"monthBalance"`
	LastUpdated      time.Time `json:"lastUpdated"`
}

// CategoryAmount represents expense by category
type CategoryAmount struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Percent  float64 `json:"percent"`
	Count    int     `json:"count"`
}

// AccountBalance represents an account's balance
type AccountBalance struct {
	Account  string  `json:"account"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
}

// MonthlyData represents monthly income/expense
type MonthlyData struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Balance float64 `json:"balance"`
}

// DailyData represents daily income/expense
type DailyData struct {
	Date    string  `json:"date"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Balance float64 `json:"balance"`
}

// WeeklyData represents weekly income/expense
type WeeklyData struct {
	Week    string  `json:"week"` // format: 2006-W01
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Balance float64 `json:"balance"`
}

// TagStats represents spending by tag (platform)
type TagStats struct {
	Tag     string  `json:"tag"`
	Amount  float64 `json:"amount"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

// PayeeStats represents spending by payee (merchant)
type PayeeStats struct {
	Payee  string  `json:"payee"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// WeekdayCategoryCount represents count per category for a weekday
type WeekdayCategoryCount struct {
	Category string  `json:"category"`
	Count    int     `json:"count"`
	Amount   float64 `json:"amount"`
}

// WeekdayStats represents spending by day of week
type WeekdayStats struct {
	Weekday           int                    `json:"weekday"` // 0=Sunday, 1=Monday, ...
	Name              string                 `json:"name"`
	Amount            float64                `json:"amount"`
	Count             int                    `json:"count"`
	Dates             []string               `json:"dates"`             // List of dates (MM-DD format) with expenses
	CategoryBreakdown []WeekdayCategoryCount `json:"categoryBreakdown"` // Spending breakdown by category
}

// MonthlyAmount for category trends
type MonthlyAmount struct {
	Month  string  `json:"month"`
	Amount float64 `json:"amount"`
}

// CategoryTrend represents monthly trend for a category
type CategoryTrend struct {
	Category string          `json:"category"`
	Data     []MonthlyAmount `json:"data"`
}

// LiabilityStats represents liability account details
type LiabilityStats struct {
	Account  string  `json:"account"`
	Name     string  `json:"name"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

// IncomeSource represents income breakdown by source
type IncomeSource struct {
	Source  string  `json:"source"`
	Amount  float64 `json:"amount"`
	Percent float64 `json:"percent"`
	Count   int     `json:"count"`
}

// Analytics holds all analytical data
type Analytics struct {
	Summary            Summary          `json:"summary"`
	ExpenseByCategory  []CategoryAmount `json:"expenseByCategory"`
	AccountBalances    []AccountBalance `json:"accountBalances"`
	MonthlyTrend       []MonthlyData    `json:"monthlyTrend"`
	DailyTrend         []DailyData      `json:"dailyTrend"`
	WeeklyTrend        []WeeklyData     `json:"weeklyTrend"`
	RecentTransactions []Transaction    `json:"recentTransactions"`
	// New analytics
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
	now := time.Now()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

	analytics := &Analytics{
		Summary: Summary{
			LastUpdated: now,
		},
		ExpenseByCategory:  make([]CategoryAmount, 0),
		AccountBalances:    make([]AccountBalance, 0),
		MonthlyTrend:       make([]MonthlyData, 0),
		RecentTransactions: make([]Transaction, 0),
	}

	// Calculate account balances from transactions
	accountBalances := make(map[string]float64)
	monthlyData := make(map[string]*MonthlyData)
	dailyData := make(map[string]*DailyData)
	weeklyData := make(map[string]*WeeklyData)
	categoryExpense := make(map[string]struct {
		amount float64
		count  int
	})

	for _, tx := range ledger.Transactions {
		txMonth := tx.Date.Format("2006-01")
		txDate := tx.Date.Format("2006-01-02")
		txYear, txWeek := tx.Date.ISOWeek()
		txWeekStr := time.Date(txYear, 1, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, (txWeek-1)*7).Format("01-02")

		// Initialize monthly data if needed
		if _, ok := monthlyData[txMonth]; !ok {
			monthlyData[txMonth] = &MonthlyData{Month: txMonth}
		}

		// Initialize daily data if needed
		if _, ok := dailyData[txDate]; !ok {
			dailyData[txDate] = &DailyData{Date: txDate}
		}

		// Initialize weekly data if needed (use year-week as key)
		weekKey := tx.Date.Format("2006") + "-W" + txWeekStr
		if _, ok := weeklyData[weekKey]; !ok {
			weeklyData[weekKey] = &WeeklyData{Week: weekKey}
		}

		for _, posting := range tx.Postings {
			// Update account balance
			accountBalances[posting.Account] += posting.Amount

			// Calculate monthly income/expense
			if tx.Date.Year() == now.Year() && tx.Date.Month() == now.Month() {
				if isExpenseAccount(posting.Account) && posting.Amount > 0 {
					analytics.Summary.MonthExpense += posting.Amount
				} else if isIncomeAccount(posting.Account) && posting.Amount < 0 {
					analytics.Summary.MonthIncome += -posting.Amount // Income is negative in postings
				}
			}

			// Aggregate expense by category (current month)
			if tx.Date.After(currentMonth) || tx.Date.Equal(currentMonth) {
				if isExpenseAccount(posting.Account) && posting.Amount > 0 {
					category := getExpenseCategory(posting.Account)
					ce := categoryExpense[category]
					ce.amount += posting.Amount
					ce.count++
					categoryExpense[category] = ce
				}
			}

			// Monthly trend
			if isExpenseAccount(posting.Account) && posting.Amount > 0 {
				monthlyData[txMonth].Expense += posting.Amount
				dailyData[txDate].Expense += posting.Amount
				weeklyData[weekKey].Expense += posting.Amount
			} else if isIncomeAccount(posting.Account) && posting.Amount < 0 {
				monthlyData[txMonth].Income += -posting.Amount
				dailyData[txDate].Income += -posting.Amount
				weeklyData[weekKey].Income += -posting.Amount
			}
		}
	}

	// Calculate net worth and totals
	for account, balance := range accountBalances {
		if isAssetAccount(account) {
			analytics.Summary.TotalAssets += balance
		} else if isLiabilityAccount(account) {
			analytics.Summary.TotalLiabilities += -balance // Liabilities are negative
		}
	}
	analytics.Summary.NetWorth = analytics.Summary.TotalAssets - analytics.Summary.TotalLiabilities
	analytics.Summary.MonthBalance = analytics.Summary.MonthIncome - analytics.Summary.MonthExpense

	// Build expense by category
	var totalExpense float64
	for _, ce := range categoryExpense {
		totalExpense += ce.amount
	}
	for category, ce := range categoryExpense {
		percent := 0.0
		if totalExpense > 0 {
			percent = (ce.amount / totalExpense) * 100
		}
		analytics.ExpenseByCategory = append(analytics.ExpenseByCategory, CategoryAmount{
			Category: category,
			Amount:   ce.amount,
			Percent:  percent,
			Count:    ce.count,
		})
	}
	// Sort by amount descending
	sort.Slice(analytics.ExpenseByCategory, func(i, j int) bool {
		return analytics.ExpenseByCategory[i].Amount > analytics.ExpenseByCategory[j].Amount
	})

	// Build account balances (only Assets and Liabilities)
	for account, balance := range accountBalances {
		if isAssetAccount(account) || isLiabilityAccount(account) {
			analytics.AccountBalances = append(analytics.AccountBalances, AccountBalance{
				Account:  account,
				Balance:  balance,
				Currency: ledger.BaseCurrency,
				Type:     getAccountType(account),
			})
		}
	}
	// Sort by balance descending
	sort.Slice(analytics.AccountBalances, func(i, j int) bool {
		return analytics.AccountBalances[i].Balance > analytics.AccountBalances[j].Balance
	})

	// Build monthly trend (last 6 months, always include even if empty)
	var last6Months []string
	for i := 5; i >= 0; i-- {
		m := now.AddDate(0, -i, 0)
		last6Months = append(last6Months, m.Format("2006-01"))
	}
	for _, month := range last6Months {
		data := monthlyData[month]
		if data == nil {
			data = &MonthlyData{Month: month}
		}
		data.Balance = data.Income - data.Expense
		analytics.MonthlyTrend = append(analytics.MonthlyTrend, *data)
	}

	// Build daily trend (last 30 days)
	var last30Days []string
	for i := 29; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		last30Days = append(last30Days, d.Format("2006-01-02"))
	}
	for _, day := range last30Days {
		data := dailyData[day]
		if data == nil {
			data = &DailyData{Date: day}
		}
		data.Balance = data.Income - data.Expense
		analytics.DailyTrend = append(analytics.DailyTrend, *data)
	}

	// Build weekly trend (last 8 weeks)
	var last8Weeks []string
	for i := 7; i >= 0; i-- {
		d := now.AddDate(0, 0, -i*7)
		year, week := d.ISOWeek()
		weekStart := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, (week-1)*7)
		weekKey := d.Format("2006") + "-W" + weekStart.Format("01-02")
		last8Weeks = append(last8Weeks, weekKey)
	}
	// Remove duplicates and keep order
	seen := make(map[string]bool)
	var uniqueWeeks []string
	for _, w := range last8Weeks {
		if !seen[w] {
			seen[w] = true
			uniqueWeeks = append(uniqueWeeks, w)
		}
	}
	for _, week := range uniqueWeeks {
		data := weeklyData[week]
		if data == nil {
			data = &WeeklyData{Week: week}
		}
		data.Balance = data.Income - data.Expense
		analytics.WeeklyTrend = append(analytics.WeeklyTrend, *data)
	}

	// Recent transactions (last 100 for filtering, excluding opening balance)
	var recentTxs []Transaction
	for _, tx := range ledger.Transactions {
		if isOpeningBalanceTransaction(tx) {
			continue
		}
		recentTxs = append(recentTxs, tx)
		if len(recentTxs) >= 100 {
			break
		}
	}
	analytics.RecentTransactions = recentTxs

	// ========== NEW ANALYTICS ==========

	// Platform ranking (by tag)
	tagSpending := make(map[string]struct {
		amount float64
		count  int
	})
	// Merchant ranking (by payee)
	payeeSpending := make(map[string]struct {
		amount float64
		count  int
	})
	// Weekday distribution
	weekdaySpending := make(map[int]struct {
		amount     float64
		count      int
		dates      []string
		categories map[string]struct {
			count  int
			amount float64
		}
	})
	// Category monthly trends
	categoryMonthly := make(map[string]map[string]float64)
	// Income by source
	incomeBySource := make(map[string]struct {
		amount float64
		count  int
	})
	// Track days with expenses for daily average
	expenseDays := make(map[string]bool)

	for _, tx := range ledger.Transactions {
		if isOpeningBalanceTransaction(tx) {
			continue
		}

		var txExpenseAmount float64
		var txCategories = make(map[string]float64) // Track categories for this transaction

		for _, posting := range tx.Postings {
			if isExpenseAccount(posting.Account) && posting.Amount > 0 {
				txExpenseAmount += posting.Amount

				// Category monthly trend
				category := getExpenseCategory(posting.Account)
				txMonth := tx.Date.Format("2006-01")
				if categoryMonthly[category] == nil {
					categoryMonthly[category] = make(map[string]float64)
				}
				categoryMonthly[category][txMonth] += posting.Amount

				// Track category for weekday breakdown
				txCategories[category] += posting.Amount
			}

			// Income breakdown
			if isIncomeAccount(posting.Account) && posting.Amount < 0 {
				source := getIncomeSource(posting.Account)
				is := incomeBySource[source]
				is.amount += -posting.Amount
				is.count++
				incomeBySource[source] = is
			}
		}

		if txExpenseAmount > 0 {
			// Track expense day
			expenseDays[tx.Date.Format("2006-01-02")] = true

			// Weekday
			weekday := int(tx.Date.Weekday())
			ws := weekdaySpending[weekday]
			ws.amount += txExpenseAmount
			ws.count++

			// Initialize categories map if nil
			if ws.categories == nil {
				ws.categories = make(map[string]struct {
					count  int
					amount float64
				})
			}

			// Add category counts for this transaction
			for cat, amt := range txCategories {
				catData := ws.categories[cat]
				catData.count++
				catData.amount += amt
				ws.categories[cat] = catData
			}

			// Add date if not already in the list (using MM-DD format)
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

			// Tags (platform)
			for _, tag := range tx.Tags {
				ts := tagSpending[tag]
				ts.amount += txExpenseAmount
				ts.count++
				tagSpending[tag] = ts
			}

			// Payee (merchant)
			if tx.Payee != "" {
				ps := payeeSpending[tx.Payee]
				ps.amount += txExpenseAmount
				ps.count++
				payeeSpending[tx.Payee] = ps
			}
		}
	}

	// Daily average (current month)
	if daysInMonth := now.Day(); daysInMonth > 0 {
		analytics.DailyAverage = analytics.Summary.MonthExpense / float64(daysInMonth)
	}

	// Platform ranking
	var totalTagSpending float64
	for _, ts := range tagSpending {
		totalTagSpending += ts.amount
	}
	for tag, ts := range tagSpending {
		percent := 0.0
		if totalTagSpending > 0 {
			percent = (ts.amount / totalTagSpending) * 100
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

		// Build category breakdown
		var catBreakdown []WeekdayCategoryCount
		for cat, data := range ws.categories {
			catBreakdown = append(catBreakdown, WeekdayCategoryCount{
				Category: cat,
				Count:    data.count,
				Amount:   data.amount,
			})
		}
		// Sort by count descending
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

	// Get last 6 months for category trends
	var catLast6Months []string
	for i := 5; i >= 0; i-- {
		m := now.AddDate(0, -i, 0)
		catLast6Months = append(catLast6Months, m.Format("2006-01"))
	}

	for _, category := range topCategories {
		trend := CategoryTrend{Category: category}
		monthData := categoryMonthly[category]
		for _, month := range catLast6Months {
			trend.Data = append(trend.Data, MonthlyAmount{
				Month:  month,
				Amount: monthData[month],
			})
		}
		analytics.CategoryTrends = append(analytics.CategoryTrends, trend)
	}

	// Liability breakdown
	for account, balance := range accountBalances {
		if isLiabilityAccount(account) && balance < 0 {
			analytics.LiabilityBreakdown = append(analytics.LiabilityBreakdown, LiabilityStats{
				Account:  account,
				Name:     getAccountShortName(account),
				Balance:  -balance, // Show as positive number
				Currency: ledger.BaseCurrency,
			})
		}
	}
	sort.Slice(analytics.LiabilityBreakdown, func(i, j int) bool {
		return analytics.LiabilityBreakdown[i].Balance > analytics.LiabilityBreakdown[j].Balance
	})

	// Income breakdown
	var totalIncome float64
	for _, is := range incomeBySource {
		totalIncome += is.amount
	}
	for source, is := range incomeBySource {
		percent := 0.0
		if totalIncome > 0 {
			percent = (is.amount / totalIncome) * 100
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

func isAssetAccount(account string) bool {
	return len(account) > 6 && account[:6] == "Assets"
}

func isLiabilityAccount(account string) bool {
	return len(account) > 11 && account[:11] == "Liabilities"
}

func isExpenseAccount(account string) bool {
	return len(account) > 8 && account[:8] == "Expenses"
}

func isIncomeAccount(account string) bool {
	return len(account) > 6 && account[:6] == "Income"
}

// isOpeningBalanceTransaction checks if a transaction is an opening balance entry
func isOpeningBalanceTransaction(tx Transaction) bool {
	// Check if any posting involves Equity:Opening-Balances
	for _, p := range tx.Postings {
		if len(p.Account) > 6 && p.Account[:6] == "Equity" {
			return true
		}
	}
	// Also check for common patterns in payee/narration
	if tx.Payee == "系统初始化" || tx.Narration == "录入现有资产与负债" {
		return true
	}
	return false
}

func getAccountType(account string) string {
	parts := splitAccount(account)
	if len(parts) > 0 {
		return parts[0]
	}
	return "Unknown"
}

func getExpenseCategory(account string) string {
	parts := splitAccount(account)
	if len(parts) >= 2 {
		return parts[1]
	}
	return "Other"
}

func getIncomeSource(account string) string {
	parts := splitAccount(account)
	if len(parts) >= 2 {
		return parts[1]
	}
	return "Other"
}

func getAccountShortName(account string) string {
	parts := splitAccount(account)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return account
}

func splitAccount(account string) []string {
	return splitString(account, ":")
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
