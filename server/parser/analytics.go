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

// WeekdayStats represents spending by day of week
type WeekdayStats struct {
	Weekday int     `json:"weekday"` // 0=Sunday, 1=Monday, ...
	Name    string  `json:"name"`
	Amount  float64 `json:"amount"`
	Count   int     `json:"count"`
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
}

// Analytics holds all analytical data
type Analytics struct {
	Summary            Summary          `json:"summary"`
	ExpenseByCategory  []CategoryAmount `json:"expenseByCategory"`
	AccountBalances    []AccountBalance `json:"accountBalances"`
	MonthlyTrend       []MonthlyData    `json:"monthlyTrend"`
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
	categoryExpense := make(map[string]float64)

	for _, tx := range ledger.Transactions {
		txMonth := tx.Date.Format("2006-01")

		// Initialize monthly data if needed
		if _, ok := monthlyData[txMonth]; !ok {
			monthlyData[txMonth] = &MonthlyData{Month: txMonth}
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
					categoryExpense[category] += posting.Amount
				}
			}

			// Monthly trend
			if isExpenseAccount(posting.Account) && posting.Amount > 0 {
				monthlyData[txMonth].Expense += posting.Amount
			} else if isIncomeAccount(posting.Account) && posting.Amount < 0 {
				monthlyData[txMonth].Income += -posting.Amount
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
	for _, amount := range categoryExpense {
		totalExpense += amount
	}
	for category, amount := range categoryExpense {
		percent := 0.0
		if totalExpense > 0 {
			percent = (amount / totalExpense) * 100
		}
		analytics.ExpenseByCategory = append(analytics.ExpenseByCategory, CategoryAmount{
			Category: category,
			Amount:   amount,
			Percent:  percent,
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

	// Build monthly trend (last 12 months)
	var months []string
	for month := range monthlyData {
		months = append(months, month)
	}
	sort.Strings(months)
	if len(months) > 12 {
		months = months[len(months)-12:]
	}
	for _, month := range months {
		data := monthlyData[month]
		data.Balance = data.Income - data.Expense
		analytics.MonthlyTrend = append(analytics.MonthlyTrend, *data)
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
		amount float64
		count  int
	})
	// Category monthly trends
	categoryMonthly := make(map[string]map[string]float64)
	// Income by source
	incomeBySource := make(map[string]float64)
	// Track days with expenses for daily average
	expenseDays := make(map[string]bool)

	for _, tx := range ledger.Transactions {
		if isOpeningBalanceTransaction(tx) {
			continue
		}

		var txExpenseAmount float64
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
			}

			// Income breakdown
			if isIncomeAccount(posting.Account) && posting.Amount < 0 {
				source := getIncomeSource(posting.Account)
				incomeBySource[source] += -posting.Amount
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
		analytics.WeekdayDistribution = append(analytics.WeekdayDistribution, WeekdayStats{
			Weekday: i,
			Name:    weekdayNames[i],
			Amount:  ws.amount,
			Count:   ws.count,
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

	// Get last 6 months
	var last6Months []string
	for i := 5; i >= 0; i-- {
		m := now.AddDate(0, -i, 0)
		last6Months = append(last6Months, m.Format("2006-01"))
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
	for _, amount := range incomeBySource {
		totalIncome += amount
	}
	for source, amount := range incomeBySource {
		percent := 0.0
		if totalIncome > 0 {
			percent = (amount / totalIncome) * 100
		}
		analytics.IncomeBreakdown = append(analytics.IncomeBreakdown, IncomeSource{
			Source:  source,
			Amount:  amount,
			Percent: percent,
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
