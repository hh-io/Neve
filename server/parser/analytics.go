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

// Analytics holds all analytical data
type Analytics struct {
	Summary            Summary          `json:"summary"`
	ExpenseByCategory  []CategoryAmount `json:"expenseByCategory"`
	AccountBalances    []AccountBalance `json:"accountBalances"`
	MonthlyTrend       []MonthlyData    `json:"monthlyTrend"`
	RecentTransactions []Transaction    `json:"recentTransactions"`
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

	// Recent transactions (last 20, excluding opening balance transactions)
	var recentTxs []Transaction
	for _, tx := range ledger.Transactions {
		// Skip opening balance transactions
		if isOpeningBalanceTransaction(tx) {
			continue
		}
		recentTxs = append(recentTxs, tx)
		if len(recentTxs) >= 20 {
			break
		}
	}
	analytics.RecentTransactions = recentTxs

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
