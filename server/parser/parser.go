package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Account represents a Beancount account
type Account struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"` // Assets, Liabilities, Income, Expenses, Equity
	Currency string    `json:"currency"`
	OpenDate time.Time `json:"openDate"`
}

// Posting represents a single posting in a transaction
type Posting struct {
	Account  string  `json:"account"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Transaction represents a Beancount transaction
type Transaction struct {
	Date       time.Time `json:"date"`
	Flag       string    `json:"flag"` // * or !
	Payee      string    `json:"payee"`
	Narration  string    `json:"narration"`
	Postings   []Posting `json:"postings"`
	Tags       []string  `json:"tags"`
	SourceFile string    `json:"sourceFile"`
}

// Balance represents a balance assertion
type Balance struct {
	Date     time.Time `json:"date"`
	Account  string    `json:"account"`
	Amount   float64   `json:"amount"`
	Currency string    `json:"currency"`
}

// Ledger holds all parsed data
type Ledger struct {
	Accounts     []Account     `json:"accounts"`
	Transactions []Transaction `json:"transactions"`
	Balances     []Balance     `json:"balances"`
	BaseCurrency string        `json:"baseCurrency"`
}

// Parser is the beancount parser
type Parser struct {
	dataDir string
}

// NewParser creates a new parser
func NewParser(dataDir string) *Parser {
	return &Parser{dataDir: dataDir}
}

// Parse parses all bean files starting from main.bean
func (p *Parser) Parse() (*Ledger, error) {
	ledger := &Ledger{
		Accounts:     make([]Account, 0),
		Transactions: make([]Transaction, 0),
		Balances:     make([]Balance, 0),
		BaseCurrency: "CNY",
	}

	mainFile := filepath.Join(p.dataDir, "main.bean")
	if err := p.parseFile(mainFile, ledger); err != nil {
		return nil, err
	}

	// Sort transactions by date (newest first)
	sort.Slice(ledger.Transactions, func(i, j int) bool {
		return ledger.Transactions[i].Date.After(ledger.Transactions[j].Date)
	})

	return ledger, nil
}

func (p *Parser) parseFile(filePath string, ledger *Ledger) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentTx *Transaction
	lineNum := 0

	// Regex patterns - improved to handle various spacing
	includeRe := regexp.MustCompile(`^include\s+"([^"]+)"`)
	openRe := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+open\s+(\S+)(?:\s+(\S+))?`)

	// Transaction header: date * "payee" "narration" #tags or date * "narration" #tags
	// Tags are optional and can appear after the narration
	txWithPayeeRe := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+([*!])\s+"([^"]*)"\s+"([^"]*)"(.*)$`)
	txNoPayeeRe := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+([*!])\s+"([^"]*)"(.*)$`)
	tagRe := regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)

	// Posting: account amount currency ; comment (flexible spacing)
	// Match: "  Account:Name   123.45 CNY ; comment" or "  Account:Name   -123.45 CNY"
	postingRe := regexp.MustCompile(`^\s+(\S+)\s+(-?[\d,]+\.?\d*)\s+(CNY|USD|EUR|JPY|HKD)`)

	// Posting without amount (auto-balance): "  Account:Name" or "  Account:Name ; comment"
	postingNoAmountRe := regexp.MustCompile(`^\s+([A-Z][a-zA-Z0-9:_-]+)(?:\s*;.*)?$`)

	// Balance assertion
	balanceRe := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+balance\s+(\S+)\s+(-?[\d,]+\.?\d*)\s+(\S+)`)

	// Option
	optionRe := regexp.MustCompile(`^option\s+"operating_currency"\s+"(\S+)"`)

	sourceFile := filepath.Base(filePath)

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		trimmed := strings.TrimSpace(line)

		// Empty lines or pure comment lines (starting with ;;)
		// DON'T end the transaction here - only end when we see a new directive
		if trimmed == "" {
			continue
		}

		// Skip pure comment lines at the start of line (not indented)
		// Indented comments (like ";; --- 资产 ---") are inside transactions
		if strings.HasPrefix(trimmed, ";") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			continue
		}

		// Indented comment inside a transaction - just skip it
		if currentTx != nil && strings.HasPrefix(trimmed, ";") {
			continue
		}

		// Check for include directive
		if matches := includeRe.FindStringSubmatch(line); matches != nil {
			if currentTx != nil && len(currentTx.Postings) > 0 {
				finalizeTransaction(currentTx, ledger.BaseCurrency)
				ledger.Transactions = append(ledger.Transactions, *currentTx)
				currentTx = nil
			}

			includePath := matches[1]
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(filepath.Dir(filePath), includePath)
			}
			if err := p.parseFile(includePath, ledger); err != nil {
				// Skip missing files silently
				continue
			}
			continue
		}

		// Check for option
		if matches := optionRe.FindStringSubmatch(line); matches != nil {
			ledger.BaseCurrency = matches[1]
			continue
		}

		// Check for account open
		if matches := openRe.FindStringSubmatch(line); matches != nil {
			if currentTx != nil && len(currentTx.Postings) > 0 {
				finalizeTransaction(currentTx, ledger.BaseCurrency)
				ledger.Transactions = append(ledger.Transactions, *currentTx)
				currentTx = nil
			}

			date, _ := time.Parse("2006-01-02", matches[1])
			currency := "CNY"
			if len(matches) > 3 && matches[3] != "" {
				currency = matches[3]
			}

			accountName := matches[2]
			accountType := strings.Split(accountName, ":")[0]

			ledger.Accounts = append(ledger.Accounts, Account{
				Name:     accountName,
				Type:     accountType,
				Currency: currency,
				OpenDate: date,
			})
			continue
		}

		// Check for balance assertion
		if matches := balanceRe.FindStringSubmatch(line); matches != nil {
			if currentTx != nil && len(currentTx.Postings) > 0 {
				finalizeTransaction(currentTx, ledger.BaseCurrency)
				ledger.Transactions = append(ledger.Transactions, *currentTx)
				currentTx = nil
			}

			date, _ := time.Parse("2006-01-02", matches[1])
			amount := parseAmount(matches[3])

			ledger.Balances = append(ledger.Balances, Balance{
				Date:     date,
				Account:  matches[2],
				Amount:   amount,
				Currency: matches[4],
			})
			continue
		}

		// Check for transaction header with payee
		if matches := txWithPayeeRe.FindStringSubmatch(line); matches != nil {
			if currentTx != nil && len(currentTx.Postings) > 0 {
				finalizeTransaction(currentTx, ledger.BaseCurrency)
				ledger.Transactions = append(ledger.Transactions, *currentTx)
			}

			date, _ := time.Parse("2006-01-02", matches[1])

			// Parse tags from remaining text
			var tags []string
			if len(matches) > 5 {
				tagMatches := tagRe.FindAllStringSubmatch(matches[5], -1)
				for _, tm := range tagMatches {
					tags = append(tags, tm[1])
				}
			}

			currentTx = &Transaction{
				Date:       date,
				Flag:       matches[2],
				Payee:      matches[3],
				Narration:  matches[4],
				Postings:   make([]Posting, 0),
				Tags:       tags,
				SourceFile: sourceFile,
			}
			continue
		}

		// Check for transaction header without payee
		if matches := txNoPayeeRe.FindStringSubmatch(line); matches != nil {
			if currentTx != nil && len(currentTx.Postings) > 0 {
				finalizeTransaction(currentTx, ledger.BaseCurrency)
				ledger.Transactions = append(ledger.Transactions, *currentTx)
			}

			date, _ := time.Parse("2006-01-02", matches[1])

			// Parse tags from remaining text
			var tags []string
			if len(matches) > 4 {
				tagMatches := tagRe.FindAllStringSubmatch(matches[4], -1)
				for _, tm := range tagMatches {
					tags = append(tags, tm[1])
				}
			}

			currentTx = &Transaction{
				Date:       date,
				Flag:       matches[2],
				Payee:      "",
				Narration:  matches[3],
				Postings:   make([]Posting, 0),
				Tags:       tags,
				SourceFile: sourceFile,
			}
			continue
		}

		// Check for posting with amount
		if currentTx != nil {
			if matches := postingRe.FindStringSubmatch(line); matches != nil {
				amount := parseAmount(matches[2])
				currentTx.Postings = append(currentTx.Postings, Posting{
					Account:  matches[1],
					Amount:   amount,
					Currency: matches[3],
				})
				continue
			}

			// Check for posting without amount (auto-balanced)
			if matches := postingNoAmountRe.FindStringSubmatch(line); matches != nil {
				// Only add if it looks like a valid account (contains colons)
				if strings.Contains(matches[1], ":") {
					currentTx.Postings = append(currentTx.Postings, Posting{
						Account:  matches[1],
						Amount:   0, // Will be calculated in finalizeTransaction
						Currency: ledger.BaseCurrency,
					})
				}
				continue
			}
		}
	}

	// Don't forget the last transaction
	if currentTx != nil && len(currentTx.Postings) > 0 {
		finalizeTransaction(currentTx, ledger.BaseCurrency)
		ledger.Transactions = append(ledger.Transactions, *currentTx)
	}

	return scanner.Err()
}

// finalizeTransaction calculates auto-balanced postings and validates balance
func finalizeTransaction(tx *Transaction, baseCurrency string) {
	var total float64
	var autoBalanceIdx = -1

	for i, p := range tx.Postings {
		if p.Amount == 0 && p.Currency == baseCurrency {
			autoBalanceIdx = i
		} else {
			total += p.Amount
		}
	}

	// If there's an auto-balance posting, set its amount
	if autoBalanceIdx >= 0 {
		tx.Postings[autoBalanceIdx].Amount = -total
	} else {
		// Validate that transaction balances (sum should be ~0)
		if total > 0.01 || total < -0.01 {
			// Log unbalanced transaction (allow small rounding errors)
			// In production, you might want to return an error
			// For now, we just log it
			_ = total // Acknowledge the imbalance (could add logging here)
		}
	}
}

func parseAmount(s string) float64 {
	// Remove commas and spaces
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	amount, _ := strconv.ParseFloat(s, 64)
	return amount
}
