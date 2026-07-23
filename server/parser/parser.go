package parser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// baseCurrency 是系统唯一支持的币种,非 CNY 数据一律报错拒绝。
const baseCurrency = "CNY"

// ParseIssue 是解析/校验过程中发现的一个问题。
// 软失败策略:坏数据跳过并记录 issue,不拖垮整个账本(inbox.bean 由 AI 写入,最易产生脏数据)。
type ParseIssue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Severity string `json:"severity"` // "error" | "warning"
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// BalanceCheck 是一条 balance 断言的校验结果
type BalanceCheck struct {
	Date     string `json:"date"`
	Account  string `json:"account"`
	Expected Amount `json:"expected"`
	Actual   Amount `json:"actual"`
	Diff     Amount `json:"diff"`
	OK       bool   `json:"ok"`
}

// Account represents a Beancount account
type Account struct {
	Name     string    `json:"name"`
	Type     string    `json:"type"` // Assets, Liabilities, Income, Expenses, Equity
	Currency string    `json:"currency"`
	OpenDate time.Time `json:"openDate"`
}

// Posting represents a single posting in a transaction
type Posting struct {
	Account  string `json:"account"`
	Amount   Amount `json:"amount"`
	Currency string `json:"currency"`
	// hasAmount 区分"省略金额待自动平衡"与"显式 0 元",不能用 Amount==0 判断
	hasAmount bool
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
	SourceLine int       `json:"sourceLine"`

	// 以下字段由 Analyze 阶段计算,解析时为零值
	Kind           string `json:"kind"` // expense | income | transfer | opening | mixed
	Category       string `json:"category"`
	DisplayAmount  Amount `json:"displayAmount"`
	TransferAmount Amount `json:"transferAmount"`
	FeeAmount      Amount `json:"feeAmount"`
	IsTransfer     bool   `json:"isTransfer"`

	// seq 记录 include 展开后的全局文件行序,同日交易靠它稳定排序
	seq int
	// broken 表示解析中已发现错误(issue 已记录),finalize 时丢弃
	broken bool
}

// Balance represents a balance assertion
type Balance struct {
	Date       time.Time `json:"date"`
	Account    string    `json:"account"`
	Amount     Amount    `json:"amount"`
	Currency   string    `json:"currency"`
	SourceFile string    `json:"sourceFile"`
	SourceLine int       `json:"sourceLine"`
}

// Ledger holds all parsed data
type Ledger struct {
	Accounts      []Account      `json:"accounts"`
	Transactions  []Transaction  `json:"transactions"`
	Balances      []Balance      `json:"balances"`
	BalanceChecks []BalanceCheck `json:"balanceChecks"`
	Issues        []ParseIssue   `json:"issues"`
	BaseCurrency  string         `json:"baseCurrency"`
	// SourceFiles 是解析实际打开的所有文件(main.bean + 逐层 include),相对 dataDir、
	// 按读取顺序去重。备份用它取账本文件清单,避免重复维护 include 展开逻辑。
	SourceFiles []string `json:"-"`
}

func (l *Ledger) addIssue(file string, line int, severity, code, message string) {
	l.Issues = append(l.Issues, ParseIssue{
		File:     file,
		Line:     line,
		Severity: severity,
		Code:     code,
		Message:  message,
	})
}

// Parser is the beancount parser
type Parser struct {
	dataDir string
	now     time.Time
	seq     int
	// activeFiles 记录当前 include 调用栈上的文件,用于检测循环 include。
	// 不检测的话循环引用会无限递归:先耗尽 fd,再把交易重复计入几百次。
	activeFiles map[string]bool
	// sourceSeen 对已记入 SourceFiles 的文件去重(同一文件被多处 include 只记一次)
	sourceSeen map[string]bool
}

// errIncludeCycle 由 parseFile 返回,include 处捕获后记为 INCLUDE_CYCLE 软错误
var errIncludeCycle = errors.New("循环 include")

// NewParser creates a new parser
func NewParser(dataDir string) *Parser {
	return &Parser{dataDir: dataDir, now: time.Now(), activeFiles: make(map[string]bool), sourceSeen: make(map[string]bool)}
}

// Parse parses all bean files starting from main.bean.
// 仅 main.bean 无法打开时返回硬错误;其余问题记入 Ledger.Issues。
func (p *Parser) Parse() (*Ledger, error) {
	ledger := &Ledger{
		Accounts:      make([]Account, 0),
		Transactions:  make([]Transaction, 0),
		Balances:      make([]Balance, 0),
		BalanceChecks: make([]BalanceCheck, 0),
		Issues:        make([]ParseIssue, 0),
		BaseCurrency:  baseCurrency,
		SourceFiles:   make([]string, 0),
	}

	mainFile := filepath.Join(p.dataDir, "main.bean")
	if err := p.parseFile(mainFile, ledger); err != nil {
		return nil, err
	}

	p.finalize(ledger)
	return ledger, nil
}

// relPath 返回相对 dataDir 的路径,用于 issue 定位
func (p *Parser) relPath(filePath string) string {
	if rel, err := filepath.Rel(p.dataDir, filePath); err == nil {
		return rel
	}
	return filepath.Base(filePath)
}

var (
	includeRe = regexp.MustCompile(`^include\s+"([^"]+)"`)
	openRe    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+open\s+(\S+)(?:\s+([A-Z][A-Z0-9'._-]*))?`)

	// Transaction header: date * "payee" "narration" #tags or date * "narration" #tags
	txWithPayeeRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+([*!])\s+"([^"]*)"\s+"([^"]*)"(.*)$`)
	txNoPayeeRe   = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+([*!])\s+"([^"]*)"(.*)$`)
	tagRe         = regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)

	// Posting: "  Account:Name   123.45 CNY ; comment",币种为通用 token,合法性在 finalize 校验
	postingRe = regexp.MustCompile(`^\s+(\S+)\s+(-?[\d,]+\.?\d*)\s+([A-Z][A-Z0-9'._-]*)`)

	// Posting without amount (auto-balance): "  Account:Name" or "  Account:Name ; comment"
	postingNoAmountRe = regexp.MustCompile(`^\s+([A-Z][a-zA-Z0-9:_-]+)(?:\s*;.*)?$`)

	balanceRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\s+balance\s+(\S+)\s+(-?[\d,]+\.?\d*)\s+([A-Z][A-Z0-9'._-]*)`)
	optionRe  = regexp.MustCompile(`^option\s+"operating_currency"\s+"(\S+)"`)
)

func (p *Parser) parseFile(filePath string, ledger *Ledger) error {
	// 所有路径都由 dataDir/include 处 Join 而来,Clean 后可作为循环检测的稳定键
	cleanPath := filepath.Clean(filePath)
	if p.activeFiles[cleanPath] {
		return errIncludeCycle
	}
	p.activeFiles[cleanPath] = true
	defer delete(p.activeFiles, cleanPath)

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var currentTx *Transaction
	lineNum := 0
	sourceFile := p.relPath(filePath)

	// 记录已成功打开的文件,供备份取账本文件清单(按读取顺序去重)
	if !p.sourceSeen[cleanPath] {
		p.sourceSeen[cleanPath] = true
		ledger.SourceFiles = append(ledger.SourceFiles, sourceFile)
	}

	flush := func() {
		p.flushTransaction(ledger, currentTx)
		currentTx = nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// 注释不结束交易(交易内部允许缩进注释行)
		if strings.HasPrefix(trimmed, ";") {
			continue
		}

		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")

		// 任何非缩进行都意味着上一笔交易结束
		if currentTx != nil && !indented {
			flush()
		}

		if !indented {
			switch {
			case includeRe.MatchString(line):
				matches := includeRe.FindStringSubmatch(line)
				includePath := matches[1]
				if !filepath.IsAbs(includePath) {
					includePath = filepath.Join(filepath.Dir(filePath), includePath)
				}
				if err := p.parseFile(includePath, ledger); err != nil {
					if errors.Is(err, errIncludeCycle) {
						ledger.addIssue(sourceFile, lineNum, "error", "INCLUDE_CYCLE",
							fmt.Sprintf("检测到循环 include %q,已跳过", matches[1]))
					} else {
						ledger.addIssue(sourceFile, lineNum, "error", "INCLUDE_MISSING",
							fmt.Sprintf("无法读取 include 文件 %q: %v", matches[1], err))
					}
				}

			case optionRe.MatchString(line):
				matches := optionRe.FindStringSubmatch(line)
				if matches[1] != baseCurrency {
					ledger.addIssue(sourceFile, lineNum, "warning", "NON_CNY",
						fmt.Sprintf("仅支持 %s,operating_currency = %q 已忽略", baseCurrency, matches[1]))
				}

			case openRe.MatchString(line):
				matches := openRe.FindStringSubmatch(line)
				date, err := time.ParseInLocation("2006-01-02", matches[1], time.Local)
				if err != nil {
					ledger.addIssue(sourceFile, lineNum, "error", "BAD_DATE",
						fmt.Sprintf("open 指令日期非法: %q", matches[1]))
					continue
				}
				currency := baseCurrency
				if matches[3] != "" {
					currency = matches[3]
				}
				if currency != baseCurrency {
					ledger.addIssue(sourceFile, lineNum, "warning", "NON_CNY",
						fmt.Sprintf("账户 %s 币种为 %s,仅支持 %s,该账户下的非 CNY 交易将被拒绝", matches[2], currency, baseCurrency))
				}
				accountName := matches[2]
				ledger.Accounts = append(ledger.Accounts, Account{
					Name:     accountName,
					Type:     strings.SplitN(accountName, ":", 2)[0],
					Currency: currency,
					OpenDate: date,
				})

			case balanceRe.MatchString(line):
				matches := balanceRe.FindStringSubmatch(line)
				date, err := time.ParseInLocation("2006-01-02", matches[1], time.Local)
				if err != nil {
					ledger.addIssue(sourceFile, lineNum, "error", "BAD_DATE",
						fmt.Sprintf("balance 断言日期非法: %q", matches[1]))
					continue
				}
				amount, err := parseAmount(matches[3])
				if err != nil {
					ledger.addIssue(sourceFile, lineNum, "error", "BAD_AMOUNT",
						fmt.Sprintf("balance 断言金额非法: %v", err))
					continue
				}
				ledger.Balances = append(ledger.Balances, Balance{
					Date:       date,
					Account:    matches[2],
					Amount:     amount,
					Currency:   matches[4],
					SourceFile: sourceFile,
					SourceLine: lineNum,
				})

			case txWithPayeeRe.MatchString(line):
				matches := txWithPayeeRe.FindStringSubmatch(line)
				currentTx = p.newTransaction(ledger, sourceFile, lineNum,
					matches[1], matches[2], matches[3], matches[4], matches[5])

			case txNoPayeeRe.MatchString(line):
				matches := txNoPayeeRe.FindStringSubmatch(line)
				currentTx = p.newTransaction(ledger, sourceFile, lineNum,
					matches[1], matches[2], "", matches[3], matches[4])
			}
			// 其余顶层行(如 option "title")按不支持的指令忽略
			continue
		}

		// 缩进行:交易内的 posting
		if currentTx == nil {
			continue
		}
		if matches := postingRe.FindStringSubmatch(line); matches != nil {
			amount, err := parseAmount(matches[2])
			if err != nil {
				ledger.addIssue(sourceFile, lineNum, "error", "BAD_AMOUNT",
					fmt.Sprintf("posting 金额非法: %v,整笔交易已跳过", err))
				currentTx.broken = true
				continue
			}
			currentTx.Postings = append(currentTx.Postings, Posting{
				Account:   matches[1],
				Amount:    amount,
				Currency:  matches[3],
				hasAmount: true,
			})
			continue
		}
		if matches := postingNoAmountRe.FindStringSubmatch(line); matches != nil {
			if strings.Contains(matches[1], ":") {
				currentTx.Postings = append(currentTx.Postings, Posting{
					Account:  matches[1],
					Currency: baseCurrency,
				})
				continue
			}
		}
		// 交易内出现无法解析的行:内容不可信,整笔丢弃
		ledger.addIssue(sourceFile, lineNum, "error", "UNPARSED_LINE",
			fmt.Sprintf("无法解析的行: %q,整笔交易已跳过", trimmed))
		currentTx.broken = true
	}

	if currentTx != nil {
		flush()
	}

	return scanner.Err()
}

func (p *Parser) newTransaction(ledger *Ledger, sourceFile string, lineNum int,
	dateStr, flag, payee, narration, tagText string) *Transaction {
	date, err := time.ParseInLocation("2006-01-02", dateStr, time.Local)
	if err != nil {
		ledger.addIssue(sourceFile, lineNum, "error", "BAD_DATE",
			fmt.Sprintf("交易日期非法: %q,整笔交易已跳过", dateStr))
		return &Transaction{broken: true, SourceFile: sourceFile, SourceLine: lineNum}
	}

	// 始终非 nil,JSON 输出 [] 而非 null,前端可直接遍历
	tags := []string{}
	for _, tm := range tagRe.FindAllStringSubmatch(tagText, -1) {
		tags = append(tags, tm[1])
	}

	return &Transaction{
		Date:       date,
		Flag:       flag,
		Payee:      payee,
		Narration:  narration,
		Postings:   make([]Posting, 0),
		Tags:       tags,
		SourceFile: sourceFile,
		SourceLine: lineNum,
	}
}

func (p *Parser) flushTransaction(ledger *Ledger, tx *Transaction) {
	if tx == nil || tx.broken {
		return // broken 交易的 issue 已在解析时记录
	}
	if len(tx.Postings) == 0 {
		ledger.addIssue(tx.SourceFile, tx.SourceLine, "warning", "EMPTY_TRANSACTION",
			"交易没有任何 posting,已忽略")
		return
	}
	tx.seq = p.seq
	p.seq++
	ledger.Transactions = append(ledger.Transactions, *tx)
}

// finalize 在全部文件解析完成后执行账本级校验:
// 账户 open、单币种、借贷平衡、自动平衡腿、未来日期,再稳定排序并核对 balance 断言。
func (p *Parser) finalize(ledger *Ledger) {
	openAccounts := make(map[string]bool, len(ledger.Accounts))
	for _, acc := range ledger.Accounts {
		if openAccounts[acc.Name] {
			ledger.addIssue("main.bean", 0, "warning", "DUPLICATE_OPEN",
				fmt.Sprintf("账户 %s 重复 open", acc.Name))
			continue
		}
		openAccounts[acc.Name] = true
	}

	valid := ledger.Transactions[:0]
	for i := range ledger.Transactions {
		tx := &ledger.Transactions[i]
		if !p.validateTransaction(ledger, tx, openAccounts) {
			continue
		}
		if tx.Date.After(p.now) {
			ledger.addIssue(tx.SourceFile, tx.SourceLine, "warning", "FUTURE_DATE",
				fmt.Sprintf("交易日期 %s 在未来,不计入统计", tx.Date.Format("2006-01-02")))
		}
		valid = append(valid, *tx)
	}
	ledger.Transactions = valid

	// 主键日期降序,次键文件行序降序(同日内文件靠后的视为更新)
	sort.SliceStable(ledger.Transactions, func(i, j int) bool {
		ti, tj := ledger.Transactions[i], ledger.Transactions[j]
		if !ti.Date.Equal(tj.Date) {
			return ti.Date.After(tj.Date)
		}
		return ti.seq > tj.seq
	})

	p.checkBalances(ledger, openAccounts)
}

// validateTransaction 校验单笔交易并补全自动平衡腿,失败时记录 issue 并返回 false。
func (p *Parser) validateTransaction(ledger *Ledger, tx *Transaction, openAccounts map[string]bool) bool {
	for _, po := range tx.Postings {
		if !openAccounts[po.Account] {
			ledger.addIssue(tx.SourceFile, tx.SourceLine, "error", "UNKNOWN_ACCOUNT",
				fmt.Sprintf("账户 %s 未在 main.bean 中 open,整笔交易已跳过", po.Account))
			return false
		}
		if po.Currency != baseCurrency {
			ledger.addIssue(tx.SourceFile, tx.SourceLine, "error", "NON_CNY",
				fmt.Sprintf("仅支持 %s,posting 币种为 %s,整笔交易已跳过", baseCurrency, po.Currency))
			return false
		}
	}

	var total Amount
	autoIdx := -1
	for i := range tx.Postings {
		po := &tx.Postings[i]
		if !po.hasAmount {
			if autoIdx >= 0 {
				ledger.addIssue(tx.SourceFile, tx.SourceLine, "error", "MULTI_AUTO_POSTING",
					"一笔交易只允许一条省略金额的 posting,整笔交易已跳过")
				return false
			}
			autoIdx = i
		} else {
			total += po.Amount
		}
	}

	if autoIdx >= 0 {
		tx.Postings[autoIdx].Amount = -total
		tx.Postings[autoIdx].hasAmount = true
	} else if total != 0 {
		// 整数分求和,容差为 0:两位小数下不存在合法的舍入误差
		ledger.addIssue(tx.SourceFile, tx.SourceLine, "error", "UNBALANCED",
			fmt.Sprintf("借贷不平衡,差额 %s %s,整笔交易已跳过", total, baseCurrency))
		return false
	}
	return true
}

// checkBalances 核对全部 balance 断言。
// 与官方 beancount 语义一致:断言核对的是断言日期当天开始前(date < 断言日)的累计余额。
func (p *Parser) checkBalances(ledger *Ledger, openAccounts map[string]bool) {
	for _, b := range ledger.Balances {
		if !openAccounts[b.Account] {
			ledger.addIssue(b.SourceFile, b.SourceLine, "error", "UNKNOWN_ACCOUNT",
				fmt.Sprintf("balance 断言引用了未 open 的账户 %s", b.Account))
			continue
		}
		if b.Currency != baseCurrency {
			ledger.addIssue(b.SourceFile, b.SourceLine, "error", "NON_CNY",
				fmt.Sprintf("仅支持 %s,balance 断言币种为 %s,已跳过", baseCurrency, b.Currency))
			continue
		}

		var actual Amount
		for _, tx := range ledger.Transactions {
			if !tx.Date.Before(b.Date) {
				continue
			}
			for _, po := range tx.Postings {
				if po.Account == b.Account {
					actual += po.Amount
				}
			}
		}

		diff := actual - b.Amount
		check := BalanceCheck{
			Date:     b.Date.Format("2006-01-02"),
			Account:  b.Account,
			Expected: b.Amount,
			Actual:   actual,
			Diff:     diff,
			OK:       diff == 0,
		}
		ledger.BalanceChecks = append(ledger.BalanceChecks, check)
		if !check.OK {
			ledger.addIssue(b.SourceFile, b.SourceLine, "error", "BALANCE_FAILED",
				fmt.Sprintf("balance 断言失败: %s 在 %s 应为 %s,实际 %s,差额 %s",
					b.Account, check.Date, b.Amount, actual, diff))
		}
	}
}
