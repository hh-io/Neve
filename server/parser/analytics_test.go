package parser

import (
	"testing"
	"time"
)

func mkTx(date string, postings ...Posting) Transaction {
	d, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		panic(err)
	}
	return Transaction{Date: d, Flag: "*", Postings: postings}
}

func po(account string, fen Amount) Posting {
	return Posting{Account: account, Amount: fen, Currency: baseCurrency, hasAmount: true}
}

func TestClassifyTransaction(t *testing.T) {
	cases := []struct {
		name         string
		tx           Transaction
		wantKind     string
		wantDisplay  Amount
		wantTransfer Amount
		wantFee      Amount
		wantCategory string
	}{
		{
			name: "普通支出",
			tx: mkTx("2026-07-10",
				po("Expenses:Food:Coffee", 2500),
				po("Assets:Cash:WeChat", -2500)),
			wantKind: "expense", wantDisplay: 2500, wantCategory: "Food",
		},
		{
			name: "退款(负支出)",
			tx: mkTx("2026-07-10",
				po("Expenses:Food:Coffee", -2500),
				po("Assets:Cash:WeChat", 2500)),
			wantKind: "expense", wantDisplay: -2500, wantCategory: "Food",
		},
		{
			name: "收入",
			tx: mkTx("2026-07-10",
				po("Income:Salary", -800000),
				po("Assets:Cash:WeChat", 800000)),
			wantKind: "income", wantDisplay: 800000, wantCategory: "Salary",
		},
		{
			name: "纯转账(提现)",
			tx: mkTx("2026-07-10",
				po("Assets:Cash:WeChat", -10000),
				po("Assets:Cash:Alipay", 10000)),
			wantKind: "transfer", wantDisplay: 10000, wantTransfer: 10000, wantCategory: "Financial",
		},
		{
			name: "还款+手续费",
			tx: mkTx("2026-07-10",
				po("Assets:Cash:WeChat", -500500),
				po("Liabilities:CreditCard:CMB", 500000),
				po("Expenses:Financial:ServiceFee", 500)),
			wantKind: "transfer", wantDisplay: 500000, wantTransfer: 500000, wantFee: 500,
			wantCategory: "Financial",
		},
		{
			name: "期初余额",
			tx: mkTx("2025-12-01",
				po("Assets:Cash:WeChat", 10000),
				po("Equity:Opening-Balances", -10000)),
			wantKind: "opening",
		},
		{
			name: "收支混合(税前工资)",
			tx: mkTx("2026-07-10",
				po("Income:Salary", -1000000),
				po("Expenses:Financial:ServiceFee", 200000),
				po("Assets:Cash:WeChat", 800000)),
			wantKind: "mixed", wantDisplay: 800000, wantCategory: "Salary",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := tc.tx
			classifyTransaction(&tx)
			if tx.Kind != tc.wantKind {
				t.Errorf("Kind = %s,期望 %s", tx.Kind, tc.wantKind)
			}
			if tx.DisplayAmount != tc.wantDisplay {
				t.Errorf("DisplayAmount = %d,期望 %d", tx.DisplayAmount, tc.wantDisplay)
			}
			if tx.TransferAmount != tc.wantTransfer {
				t.Errorf("TransferAmount = %d,期望 %d", tx.TransferAmount, tc.wantTransfer)
			}
			if tx.FeeAmount != tc.wantFee {
				t.Errorf("FeeAmount = %d,期望 %d", tx.FeeAmount, tc.wantFee)
			}
			if tc.wantCategory != "" && tx.Category != tc.wantCategory {
				t.Errorf("Category = %s,期望 %s", tx.Category, tc.wantCategory)
			}
			if (tx.Kind == "transfer") != tx.IsTransfer {
				t.Errorf("IsTransfer 与 Kind 不一致: %+v", tx)
			}
		})
	}
}

func TestIsoWeekHelpers(t *testing.T) {
	cases := []struct {
		date      string
		wantKey   string
		wantStart string
	}{
		// 跨年周:2025-12-29(周一)属于 ISO 2026-W01
		{"2025-12-29", "2026-W01", "2025-12-29"},
		{"2026-01-01", "2026-W01", "2025-12-29"},
		// 2021-01-01(周五)属于 ISO 2020-W53
		{"2021-01-01", "2020-W53", "2020-12-28"},
		{"2026-07-15", "2026-W29", "2026-07-13"},
	}
	for _, tc := range cases {
		d, _ := time.ParseInLocation("2006-01-02", tc.date, time.Local)
		if got := isoWeekKey(d); got != tc.wantKey {
			t.Errorf("isoWeekKey(%s) = %s,期望 %s", tc.date, got, tc.wantKey)
		}
		if got := weekStart(d).Format("2006-01-02"); got != tc.wantStart {
			t.Errorf("weekStart(%s) = %s,期望 %s", tc.date, got, tc.wantStart)
		}
	}
}

func TestAccountRoot(t *testing.T) {
	cases := map[string]string{
		"Assets:Cash:WeChat": "Assets",
		"AssetsFoo:Bar":      "AssetsFoo", // 前缀相似但不是资产
		"Income":             "Income",
		"Expenses:Other":     "Expenses",
	}
	for account, want := range cases {
		if got := accountRoot(account); got != want {
			t.Errorf("accountRoot(%s) = %s,期望 %s", account, got, want)
		}
	}
	if accountRoot("AssetsFoo:Bar") == "Assets" {
		t.Error("AssetsFoo 不应被判为资产")
	}
}

func TestGetExpenseCategory(t *testing.T) {
	cases := map[string]string{
		"Expenses:Food:Coffee": "Food",
		"Expenses:Other":       "Other",
		"Expenses":             "Other",
	}
	for account, want := range cases {
		if got := getExpenseCategory(account); got != want {
			t.Errorf("getExpenseCategory(%s) = %s,期望 %s", account, got, want)
		}
	}
}

// TestAnalyzeAt 用一个小账本核对"黄金数字"
func TestAnalyzeAt(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	ledger := parseMain(t, `
2025-12-01 * "系统初始化" "期初余额"
  Assets:Cash:WeChat      1000.00 CNY
  Liabilities:CreditCard:CMB -500.00 CNY
  Equity:Opening-Balances

2026-06-20 * "老板" "六月工资" #salary
  Income:Salary          -8000.00 CNY
  Assets:Cash:WeChat      8000.00 CNY

2026-07-05 * "星巴克" "拿铁" #meituan
  Expenses:Food:Coffee      25.00 CNY
  Assets:Cash:WeChat       -25.00 CNY

2026-07-10 * "招商银行" "信用卡还款"
  Assets:Cash:WeChat      -500.00 CNY
  Liabilities:CreditCard:CMB 495.00 CNY
  Expenses:Financial:ServiceFee 5.00 CNY

2026-07-11 * "拼多多" "退款-咖啡"
  Assets:Cash:WeChat         7.50 CNY
  Expenses:Food:Coffee      -7.50 CNY

2026-08-01 * "未来" "下月房租"
  Expenses:Food:Coffee     100.00 CNY
  Assets:Cash:WeChat      -100.00 CNY
`)

	a := AnalyzeAt(ledger, now)

	// 本月支出 = 25 咖啡 + 5 还款手续费 - 7.50 退款冲减(本金不算支出);未来交易不计入
	if a.Summary.MonthExpense != 2250 {
		t.Errorf("MonthExpense = %s,期望 22.50", a.Summary.MonthExpense)
	}
	if a.Summary.MonthIncome != 0 {
		t.Errorf("MonthIncome = %s,期望 0(工资在六月)", a.Summary.MonthIncome)
	}

	// 资产 = 1000 + 8000 - 25 - 500 + 7.50 = 8482.50;负债 = 500 - 495 = 5
	if a.Summary.TotalAssets != 848250 {
		t.Errorf("TotalAssets = %s,期望 8482.50", a.Summary.TotalAssets)
	}
	if a.Summary.TotalLiabilities != 500 {
		t.Errorf("TotalLiabilities = %s,期望 5.00", a.Summary.TotalLiabilities)
	}
	if a.Summary.NetWorth != 847750 {
		t.Errorf("NetWorth = %s,期望 8477.50", a.Summary.NetWorth)
	}

	// 交易列表:不含 opening,含未来交易(共 5 笔);统计口径 4 笔
	if len(a.Transactions) != 5 {
		t.Errorf("Transactions 数 = %d,期望 5", len(a.Transactions))
	}
	if a.Summary.TransactionCount != 4 {
		t.Errorf("TransactionCount = %d,期望 4(不含 opening 和未来)", a.Summary.TransactionCount)
	}
	if a.Summary.FirstDate != "2026-06-20" {
		t.Errorf("FirstDate = %s,期望 2026-06-20", a.Summary.FirstDate)
	}
	// 6-20 到 7-15 共 26 天
	if a.Summary.TrackingDays != 26 {
		t.Errorf("TrackingDays = %d,期望 26", a.Summary.TrackingDays)
	}

	// 日均 = 22.50 / 15 天 = 1.5
	if a.DailyAverage != 1.5 {
		t.Errorf("DailyAverage = %f,期望 1.5", a.DailyAverage)
	}

	// 还款交易应被识别为转账
	var repayment *Transaction
	for i := range a.Transactions {
		if a.Transactions[i].Narration == "信用卡还款" {
			repayment = &a.Transactions[i]
		}
	}
	if repayment == nil {
		t.Fatal("找不到还款交易")
	}
	if repayment.Kind != "transfer" || repayment.TransferAmount != 49500 || repayment.FeeAmount != 500 {
		t.Errorf("还款识别错误: kind=%s transfer=%s fee=%s",
			repayment.Kind, repayment.TransferAmount, repayment.FeeAmount)
	}

	// 本月分类:Food 25.00 - 7.50 退款 = 17.50,Financial 5.00
	catAmounts := make(map[string]Amount)
	for _, c := range a.ExpenseByCategory {
		catAmounts[c.Category] = c.Amount
	}
	if catAmounts["Food"] != 1750 || catAmounts["Financial"] != 500 {
		t.Errorf("本月分类支出不符: %+v", a.ExpenseByCategory)
	}

	// 退款交易净支出为负,不应进入商户排行
	for _, ps := range a.MerchantRanking {
		if ps.Payee == "拼多多" {
			t.Error("退款交易不应进入商户排行")
		}
	}

	// 平台排行:#meituan 只挂了咖啡那笔;salary tag 挂在无支出的收入交易上,不应出现
	foundMeituan := false
	for _, ts := range a.PlatformRanking {
		if ts.Tag == "meituan" {
			foundMeituan = true
			if ts.Amount != 2500 {
				t.Errorf("meituan 平台金额 = %s,期望 25.00", ts.Amount)
			}
		}
		if ts.Tag == "salary" {
			t.Error("无支出的交易 tag 不应进入平台排行")
		}
	}
	if !foundMeituan {
		t.Error("平台排行缺少 meituan")
	}

	// 周趋势应为 8 个不重复的 ISO 周
	if len(a.WeeklyTrend) != 8 {
		t.Errorf("WeeklyTrend 长度 = %d,期望 8", len(a.WeeklyTrend))
	}
	seen := make(map[string]bool)
	for _, w := range a.WeeklyTrend {
		if seen[w.Week] {
			t.Errorf("周 key 重复: %s", w.Week)
		}
		seen[w.Week] = true
		if w.WeekStart == "" {
			t.Errorf("周 %s 缺少 weekStart", w.Week)
		}
	}

	// 月趋势 6 个月,最后一个是本月
	if len(a.MonthlyTrend) != 6 || a.MonthlyTrend[5].Month != "2026-07" {
		t.Errorf("MonthlyTrend 不符: %+v", a.MonthlyTrend)
	}
}

// TestAnalyzeAtMonthAnchor 验证月末不会因 AddDate 归一化跳月
func TestAnalyzeAtMonthAnchor(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.Local)
	ledger := parseMain(t, `
2026-07-01 * "星巴克" "拿铁"
  Expenses:Food:Coffee      25.00 CNY
  Assets:Cash:WeChat       -25.00 CNY
`)
	a := AnalyzeAt(ledger, now)
	want := []string{"2026-02", "2026-03", "2026-04", "2026-05", "2026-06", "2026-07"}
	for i, m := range want {
		if a.MonthlyTrend[i].Month != m {
			t.Errorf("MonthlyTrend[%d] = %s,期望 %s", i, a.MonthlyTrend[i].Month, m)
		}
	}
}

// TestAnalyzeDoesNotMutateLedger 确认分类计算不写回 ledger(可安全并发重算)
func TestAnalyzeDoesNotMutateLedger(t *testing.T) {
	ledger := parseMain(t, `
2026-07-05 * "星巴克" "拿铁"
  Expenses:Food:Coffee      25.00 CNY
  Assets:Cash:WeChat       -25.00 CNY
`)
	AnalyzeAt(ledger, testNow)
	if ledger.Transactions[0].Kind != "" {
		t.Error("AnalyzeAt 不应修改 ledger 内的交易")
	}
}
