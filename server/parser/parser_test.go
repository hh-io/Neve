package parser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testNow 是测试用的固定"当前时刻"
var testNow = time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

// openHeader 是测试账本共用的账户定义
const openHeader = `option "operating_currency" "CNY"
2020-01-01 open Assets:Cash:WeChat CNY
2020-01-01 open Assets:Cash:Alipay CNY
2020-01-01 open Liabilities:CreditCard:CMB CNY
2020-01-01 open Expenses:Food:Coffee CNY
2020-01-01 open Expenses:Financial:ServiceFee CNY
2020-01-01 open Income:Salary CNY
2020-01-01 open Equity:Opening-Balances CNY
`

// parseFixture 把 files 写入临时目录并解析,files 必须包含 main.bean
func parseFixture(t *testing.T, files map[string]string) *Ledger {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := NewParser(dir)
	p.now = testNow
	ledger, err := p.Parse()
	if err != nil {
		t.Fatalf("Parse() 返回硬错误: %v", err)
	}
	return ledger
}

func parseMain(t *testing.T, body string) *Ledger {
	t.Helper()
	return parseFixture(t, map[string]string{"main.bean": openHeader + body})
}

func issueCodes(ledger *Ledger) map[string]int {
	codes := make(map[string]int)
	for _, issue := range ledger.Issues {
		codes[issue.Code]++
	}
	return codes
}

func TestParseAmount(t *testing.T) {
	valid := []struct {
		in   string
		want Amount
	}{
		{"1,234.56", 123456},
		{"-0.05", -5},
		{"100", 10000},
		{"0.5", 50},
		{"12.", 1200},
		{"-25794.63", -2579463},
	}
	for _, tc := range valid {
		got, err := parseAmount(tc.in)
		if err != nil {
			t.Errorf("parseAmount(%q) 报错: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseAmount(%q) = %d,期望 %d", tc.in, got, tc.want)
		}
	}

	invalid := []string{"1.234", "abc", "12a", "", ".5", "1.2.3"}
	for _, in := range invalid {
		if _, err := parseAmount(in); err == nil {
			t.Errorf("parseAmount(%q) 应报错", in)
		}
	}
}

func TestAmountJSON(t *testing.T) {
	cases := []struct {
		in   Amount
		want string
	}{
		{123456, "1234.56"},
		{-5, "-0.05"},
		{0, "0.00"},
		{10000, "100.00"},
	}
	for _, tc := range cases {
		got, err := tc.in.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != tc.want {
			t.Errorf("Amount(%d).MarshalJSON() = %s,期望 %s", tc.in, got, tc.want)
		}
	}
}

func TestBalancedTransaction(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "星巴克" "拿铁" #meituan
  Expenses:Food:Coffee    25.00 CNY
  Assets:Cash:WeChat     -25.00 CNY
`)
	if len(ledger.Issues) != 0 {
		t.Fatalf("不应有 issue,得到: %+v", ledger.Issues)
	}
	if len(ledger.Transactions) != 1 {
		t.Fatalf("期望 1 笔交易,得到 %d", len(ledger.Transactions))
	}
	tx := ledger.Transactions[0]
	if tx.Payee != "星巴克" || tx.Postings[0].Amount != 2500 || tx.Tags[0] != "meituan" {
		t.Errorf("交易解析不符: %+v", tx)
	}
	if tx.Date.Location() != time.Local {
		t.Errorf("日期应按本地时区解析,得到 %v", tx.Date.Location())
	}
}

func TestUnbalancedTransaction(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "星巴克" "拿铁"
  Expenses:Food:Coffee    25.00 CNY
  Assets:Cash:WeChat     -24.99 CNY
`)
	if issueCodes(ledger)["UNBALANCED"] != 1 {
		t.Fatalf("期望 1 条 UNBALANCED,得到: %+v", ledger.Issues)
	}
	if len(ledger.Transactions) != 0 {
		t.Error("不平衡交易应被跳过")
	}
}

func TestAutoBalance(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "初始化" "余额"
  Assets:Cash:WeChat      100.00 CNY
  Liabilities:CreditCard:CMB -30.00 CNY
  Equity:Opening-Balances
`)
	if len(ledger.Issues) != 0 {
		t.Fatalf("不应有 issue,得到: %+v", ledger.Issues)
	}
	tx := ledger.Transactions[0]
	if tx.Postings[2].Amount != -7000 {
		t.Errorf("自动平衡腿应为 -70.00,得到 %s", tx.Postings[2].Amount)
	}
}

func TestMultiAutoPosting(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "坏数据" "两条省略金额"
  Expenses:Food:Coffee    25.00 CNY
  Assets:Cash:WeChat
  Assets:Cash:Alipay
`)
	if issueCodes(ledger)["MULTI_AUTO_POSTING"] != 1 {
		t.Fatalf("期望 MULTI_AUTO_POSTING,得到: %+v", ledger.Issues)
	}
	if len(ledger.Transactions) != 0 {
		t.Error("多自动平衡腿交易应被跳过")
	}
}

func TestNonCNYRejected(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "Starbucks" "Latte"
  Expenses:Food:Coffee     5.75 USD
  Assets:Cash:WeChat      -5.75 USD
`)
	if issueCodes(ledger)["NON_CNY"] != 1 {
		t.Fatalf("期望 NON_CNY,得到: %+v", ledger.Issues)
	}
	if len(ledger.Transactions) != 0 {
		t.Error("非 CNY 交易应被跳过")
	}
}

func TestUnknownAccount(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "超市" "买菜"
  Expenses:Food:Groceries  45.20 CNY
  Assets:Cash:WeChat      -45.20 CNY
`)
	if issueCodes(ledger)["UNKNOWN_ACCOUNT"] != 1 {
		t.Fatalf("期望 UNKNOWN_ACCOUNT,得到: %+v", ledger.Issues)
	}
	if len(ledger.Transactions) != 0 {
		t.Error("引用未 open 账户的交易应被跳过")
	}
}

func TestBadAmountAndBadDate(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "坏金额" "三位小数"
  Expenses:Food:Coffee    25.123 CNY
  Assets:Cash:WeChat     -25.123 CNY

2026-13-45 * "坏日期" "十三月"
  Expenses:Food:Coffee    25.00 CNY
  Assets:Cash:WeChat     -25.00 CNY
`)
	codes := issueCodes(ledger)
	if codes["BAD_AMOUNT"] == 0 {
		t.Errorf("期望 BAD_AMOUNT,得到: %+v", ledger.Issues)
	}
	if codes["BAD_DATE"] != 1 {
		t.Errorf("期望 BAD_DATE,得到: %+v", ledger.Issues)
	}
	if len(ledger.Transactions) != 0 {
		t.Error("坏金额/坏日期交易应被跳过")
	}
}

func TestIncludeMissing(t *testing.T) {
	ledger := parseMain(t, `
include "not-exist.bean"
`)
	if issueCodes(ledger)["INCLUDE_MISSING"] != 1 {
		t.Fatalf("期望 INCLUDE_MISSING,得到: %+v", ledger.Issues)
	}
}

func TestIncludeCycle(t *testing.T) {
	ledger := parseFixture(t, map[string]string{
		"main.bean": openHeader + `include "a.bean"` + "\n",
		"a.bean": `include "b.bean"
2026-07-10 * "星巴克" "拿铁"
  Expenses:Food:Coffee    25.00 CNY
  Assets:Cash:WeChat     -25.00 CNY
`,
		"b.bean": `include "a.bean"` + "\n",
	})
	if issueCodes(ledger)["INCLUDE_CYCLE"] != 1 {
		t.Fatalf("期望 INCLUDE_CYCLE,得到: %+v", ledger.Issues)
	}
	// 循环被切断后其余内容照常解析,且不因重复展开而重复计入
	if len(ledger.Transactions) != 1 {
		t.Fatalf("期望 1 笔交易,得到 %d 笔", len(ledger.Transactions))
	}
}

func TestIncludeSelfCycle(t *testing.T) {
	ledger := parseFixture(t, map[string]string{
		"main.bean": openHeader + `include "main.bean"` + "\n",
	})
	if issueCodes(ledger)["INCLUDE_CYCLE"] != 1 {
		t.Fatalf("期望 INCLUDE_CYCLE,得到: %+v", ledger.Issues)
	}
}

func TestIncludeParsesRelativeFile(t *testing.T) {
	ledger := parseFixture(t, map[string]string{
		"main.bean": openHeader + `include "2026.bean"` + "\n",
		"2026.bean": `2026-07-10 * "星巴克" "拿铁"
  Expenses:Food:Coffee    25.00 CNY
  Assets:Cash:WeChat     -25.00 CNY
`,
	})
	if len(ledger.Transactions) != 1 {
		t.Fatalf("include 文件的交易未被解析: %+v", ledger.Issues)
	}
	if ledger.Transactions[0].SourceFile != "2026.bean" {
		t.Errorf("SourceFile 应为 2026.bean,得到 %s", ledger.Transactions[0].SourceFile)
	}
}

func TestBalanceAssertion(t *testing.T) {
	// 断言核对的是断言日期当天开始前的余额:7-12 的交易不计入 7-12 的断言
	ledger := parseMain(t, `
2026-07-10 * "充值" "微信零钱"
  Assets:Cash:WeChat      100.00 CNY
  Assets:Cash:Alipay     -100.00 CNY

2026-07-12 * "买咖啡" "拿铁"
  Expenses:Food:Coffee     25.00 CNY
  Assets:Cash:WeChat      -25.00 CNY

2026-07-12 balance Assets:Cash:WeChat 100.00 CNY
2026-07-13 balance Assets:Cash:WeChat 75.00 CNY
2026-07-13 balance Assets:Cash:Alipay 0.00 CNY
`)
	if len(ledger.BalanceChecks) != 3 {
		t.Fatalf("期望 3 条断言结果,得到 %d", len(ledger.BalanceChecks))
	}
	for i, check := range ledger.BalanceChecks[:2] {
		if !check.OK {
			t.Errorf("断言 %d 应通过: %+v", i, check)
		}
	}
	failed := ledger.BalanceChecks[2]
	if failed.OK || failed.Actual != -10000 || failed.Diff != -10000 {
		t.Errorf("Alipay 断言应失败且差额 -100.00,得到: %+v", failed)
	}
	if issueCodes(ledger)["BALANCE_FAILED"] != 1 {
		t.Errorf("断言失败应生成 BALANCE_FAILED issue")
	}
}

func TestStableOrderSameDay(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "第一笔" "早"
  Expenses:Food:Coffee    1.00 CNY
  Assets:Cash:WeChat     -1.00 CNY

2026-07-10 * "第二笔" "中"
  Expenses:Food:Coffee    2.00 CNY
  Assets:Cash:WeChat     -2.00 CNY

2026-07-10 * "第三笔" "晚"
  Expenses:Food:Coffee    3.00 CNY
  Assets:Cash:WeChat     -3.00 CNY
`)
	if len(ledger.Transactions) != 3 {
		t.Fatalf("期望 3 笔交易,得到 %d", len(ledger.Transactions))
	}
	// 日期降序 + 同日文件靠后的在前
	want := []string{"第三笔", "第二笔", "第一笔"}
	for i, w := range want {
		if ledger.Transactions[i].Payee != w {
			t.Errorf("位置 %d 期望 %s,得到 %s", i, w, ledger.Transactions[i].Payee)
		}
	}
}

func TestFutureDateWarning(t *testing.T) {
	ledger := parseMain(t, `
2027-01-01 * "未来" "预记一笔"
  Expenses:Food:Coffee    25.00 CNY
  Assets:Cash:WeChat     -25.00 CNY
`)
	if issueCodes(ledger)["FUTURE_DATE"] != 1 {
		t.Fatalf("期望 FUTURE_DATE warning,得到: %+v", ledger.Issues)
	}
	// 未来交易保留(是否计入统计由 Analyze 决定)
	if len(ledger.Transactions) != 1 {
		t.Error("未来日期交易应保留在账本中")
	}
}

func TestUnparsedLineBreaksTransaction(t *testing.T) {
	ledger := parseMain(t, `
2026-07-10 * "AI 脏输出" "格式错误"
  Expenses:Food:Coffee    25.00 CNY
  这不是一条合法的 posting
  Assets:Cash:WeChat     -25.00 CNY
`)
	if issueCodes(ledger)["UNPARSED_LINE"] != 1 {
		t.Fatalf("期望 UNPARSED_LINE,得到: %+v", ledger.Issues)
	}
	if len(ledger.Transactions) != 0 {
		t.Error("含无法解析行的交易应整笔跳过")
	}
}

func TestDuplicateOpenWarning(t *testing.T) {
	ledger := parseMain(t, `
2021-01-01 open Assets:Cash:WeChat CNY
`)
	if issueCodes(ledger)["DUPLICATE_OPEN"] != 1 {
		t.Fatalf("期望 DUPLICATE_OPEN,得到: %+v", ledger.Issues)
	}
}
