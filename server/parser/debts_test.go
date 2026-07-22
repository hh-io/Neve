package parser

import (
	"encoding/json"
	"testing"
	"time"
)

func atDate(s string) time.Time {
	d, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		panic(err)
	}
	return d
}

func debtLedger(accounts []string, txs ...Transaction) *Ledger {
	l := &Ledger{BaseCurrency: baseCurrency, Transactions: txs}
	for _, name := range accounts {
		l.Accounts = append(l.Accounts, Account{Name: name, Type: accountRoot(name), Currency: baseCurrency})
	}
	return l
}

const ccAccount = "Liabilities:CreditCard:CMB"

func revolvingCfg(billingDay, dueDay int) *DebtsConfig {
	return &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {Name: "招行信用卡", BillingDay: billingDay, DueDay: dueDay},
		},
	}
}

func TestComputeDebtsRevolvingSnapshot(t *testing.T) {
	ledger := debtLedger(
		[]string{ccAccount, "Assets:Cash:Alipay"},
		// opening 设定初始欠款 500 元,必须计入快照
		mkTx("2026-01-01",
			po(ccAccount, -50000),
			po("Equity:Opening-Balances", 50000)),
		// 账单日当天消费计入快照
		mkTx("2026-07-09",
			po("Expenses:Food:Coffee", 2500),
			po(ccAccount, -2500)),
		// 账单日同日还款,快照按净额
		mkTx("2026-07-09",
			po("Assets:Cash:Alipay", -10000),
			po(ccAccount, 10000)),
		// 账单日次日消费:不计入本期应还,只影响当前欠款
		mkTx("2026-07-10",
			po("Expenses:Shopping:Daily", 8000),
			po(ccAccount, -8000)),
		// 账单日后带手续费还款:只按转入负债账户的本金冲减
		mkTx("2026-07-15",
			po("Assets:Cash:Alipay", -30500),
			po(ccAccount, 30000),
			po("Expenses:Financial:ServiceFee", 500)),
	)

	report := ComputeDebts(ledger, revolvingCfg(9, 20), atDate("2026-07-20"))
	if len(report.Revolving) != 1 {
		t.Fatalf("revolving 条目数 = %d, want 1", len(report.Revolving))
	}
	rv := report.Revolving[0]

	if rv.StatementDate != "2026-07-09" || rv.DueDate != "2026-07-20" {
		t.Errorf("账期 = %s → %s, want 2026-07-09 → 2026-07-20", rv.StatementDate, rv.DueDate)
	}
	// 50000 + 2500 - 10000 = 42500
	if rv.StatementDue != 42500 {
		t.Errorf("StatementDue = %v, want 42500", rv.StatementDue)
	}
	if rv.PaidSince != 30000 {
		t.Errorf("PaidSince = %v, want 30000", rv.PaidSince)
	}
	if rv.Remaining != 12500 {
		t.Errorf("Remaining = %v, want 12500", rv.Remaining)
	}
	// 42500 + 8000 - 30000 = 20500
	if rv.CurrentBalance != 20500 {
		t.Errorf("CurrentBalance = %v, want 20500", rv.CurrentBalance)
	}
	// 今天就是还款日:不逾期,倒计时 0
	if rv.DaysUntilDue != 0 || rv.Overdue {
		t.Errorf("DaysUntilDue = %d, Overdue = %v, want 0/false", rv.DaysUntilDue, rv.Overdue)
	}
	if rv.AccountMissing {
		t.Error("账户存在于账本,不应标 AccountMissing")
	}
	if report.Summary.MonthDue != 42500 || report.Summary.MonthRemaining != 12500 {
		t.Errorf("Summary = %v/%v, want 42500/12500", report.Summary.MonthDue, report.Summary.MonthRemaining)
	}
}

func TestComputeDebtsRevolvingRefundAndClamp(t *testing.T) {
	base := []Transaction{
		mkTx("2026-07-05",
			po("Expenses:Shopping:Daily", 20000),
			po(ccAccount, -20000)),
	}

	// 账单日后退货:kind=expense 的正向负债 posting 也应冲减剩余待还
	refund := append(append([]Transaction{}, base...),
		mkTx("2026-07-12",
			po("Expenses:Shopping:Daily", -6000),
			po(ccAccount, 6000)))
	report := ComputeDebts(debtLedger([]string{ccAccount}, refund...), revolvingCfg(9, 20), atDate("2026-07-15"))
	if got := report.Revolving[0].Remaining; got != 14000 {
		t.Errorf("退款后 Remaining = %v, want 14000", got)
	}

	// 超额还款:Remaining 钳制为 0,不出负数
	overpay := append(append([]Transaction{}, base...),
		mkTx("2026-07-12",
			po("Assets:Cash:Alipay", -50000),
			po(ccAccount, 50000)))
	report = ComputeDebts(debtLedger([]string{ccAccount, "Assets:Cash:Alipay"}, overpay...), revolvingCfg(9, 20), atDate("2026-07-15"))
	if got := report.Revolving[0].Remaining; got != 0 {
		t.Errorf("超额还款后 Remaining = %v, want 0", got)
	}
	if report.Summary.NextDueDate != "" {
		t.Errorf("已结清仍有 NextDueDate = %q", report.Summary.NextDueDate)
	}
}

func TestComputeDebtsRevolvingCycle(t *testing.T) {
	// 上期(6/9 账单)只还一半,本期应还 = 账单日全余额,自动涵盖旧欠
	ledger := debtLedger(
		[]string{ccAccount, "Assets:Cash:Alipay"},
		mkTx("2026-06-01",
			po("Expenses:Shopping:Daily", 30000),
			po(ccAccount, -30000)),
		mkTx("2026-06-20",
			po("Assets:Cash:Alipay", -10000),
			po(ccAccount, 10000)),
		mkTx("2026-07-01",
			po("Expenses:Food:Dinner", 5000),
			po(ccAccount, -5000)),
	)
	report := ComputeDebts(ledger, revolvingCfg(9, 20), atDate("2026-07-10"))
	// 30000 - 10000 + 5000 = 25000
	if got := report.Revolving[0].StatementDue; got != 25000 {
		t.Errorf("滚动欠款 StatementDue = %v, want 25000", got)
	}

	// 账单日 25、还款日次月 10 的跨月周期
	crossCfg := revolvingCfg(25, 10)
	report = ComputeDebts(ledger, crossCfg, atDate("2026-07-26"))
	rv := report.Revolving[0]
	if rv.StatementDate != "2026-07-25" || rv.DueDate != "2026-08-10" {
		t.Errorf("跨月账期 = %s → %s, want 2026-07-25 → 2026-08-10", rv.StatementDate, rv.DueDate)
	}
	if rv.DaysUntilDue != 15 {
		t.Errorf("DaysUntilDue = %d, want 15", rv.DaysUntilDue)
	}
	report = ComputeDebts(ledger, crossCfg, atDate("2026-08-05"))
	rv = report.Revolving[0]
	if rv.StatementDate != "2026-07-25" || rv.DueDate != "2026-08-10" || rv.DaysUntilDue != 5 {
		t.Errorf("跨月 8/5 视角 = %s → %s (%d 天), want 2026-07-25 → 2026-08-10 (5 天)", rv.StatementDate, rv.DueDate, rv.DaysUntilDue)
	}
}

func TestComputeDebtsRevolvingOverdue(t *testing.T) {
	ledger := debtLedger(
		[]string{ccAccount},
		mkTx("2026-07-01",
			po("Expenses:Shopping:Daily", 10000),
			po(ccAccount, -10000)),
	)
	// 还款日已过、下期未出账:账期不翻篇,逾期天数为负
	report := ComputeDebts(ledger, revolvingCfg(9, 20), atDate("2026-07-23"))
	rv := report.Revolving[0]
	if !rv.Overdue || rv.DaysUntilDue != -3 {
		t.Errorf("Overdue = %v, DaysUntilDue = %d, want true/-3", rv.Overdue, rv.DaysUntilDue)
	}
	if rv.StatementDate != "2026-07-09" {
		t.Errorf("StatementDate = %s, 下期未出账不应翻篇", rv.StatementDate)
	}
	if report.Summary.OverdueCount != 1 || report.Summary.NextDueInDays != -3 {
		t.Errorf("Summary Overdue = %d/%d 天, want 1/-3", report.Summary.OverdueCount, report.Summary.NextDueInDays)
	}
}

func TestClampedDate(t *testing.T) {
	cases := []struct {
		year  int
		month time.Month
		day   int
		want  string
	}{
		{2026, time.February, 31, "2026-02-28"}, // 平年顺延月末,绝不进位到 3/2
		{2024, time.February, 30, "2024-02-29"}, // 闰年
		{2026, time.July, 31, "2026-07-31"},
		{2026, time.April, 31, "2026-04-30"},
	}
	for _, c := range cases {
		got := clampedDate(c.year, c.month, c.day, time.Local).Format("2006-01-02")
		if got != c.want {
			t.Errorf("clampedDate(%d, %v, %d) = %s, want %s", c.year, c.month, c.day, got, c.want)
		}
	}
}

func TestComputeDebtsInstallment(t *testing.T) {
	const mortgage = "Liabilities:Loan:Mortgage"
	cfg := &DebtsConfig{
		Installments: []InstallmentConfig{{
			ID: "mortgage", Name: "房贷", Account: mortgage, DueDay: 20,
			Schedule: []InstallmentPhase{
				{EffectiveFrom: "2023-06-01", Amount: 543210},
				{EffectiveFrom: "2026-08-01", Amount: 521000},
			},
		}},
	}
	ledger := debtLedger(
		[]string{mortgage, "Assets:Bank:CMB"},
		mkTx("2023-06-01",
			po(mortgage, -100000000),
			po("Equity:Opening-Balances", 100000000)),
		// 本期窗口 (6/20, 7/20] 内的月供:本金转入负债,利息走 Expenses
		mkTx("2026-07-18",
			po("Assets:Bank:CMB", -550000),
			po(mortgage, 543210),
			po("Expenses:Financial:Interest", 6790)),
	)

	report := ComputeDebts(ledger, cfg, atDate("2026-07-20"))
	ins := report.Installments[0]
	if ins.MonthlyAmount != 543210 {
		t.Errorf("切换前 MonthlyAmount = %v, want 543210", ins.MonthlyAmount)
	}
	if !ins.Paid || ins.PaidAmount != 543210 {
		t.Errorf("Paid = %v, PaidAmount = %v, want true/543210", ins.Paid, ins.PaidAmount)
	}
	if ins.CurrentBalance != 100000000-543210 {
		t.Errorf("CurrentBalance = %v, want %v", ins.CurrentBalance, 100000000-543210)
	}
	// 已还的分期不进剩余待还
	if report.Summary.MonthDue != 543210 || report.Summary.MonthRemaining != 0 {
		t.Errorf("Summary = %v/%v, want 543210/0", report.Summary.MonthDue, report.Summary.MonthRemaining)
	}

	// 8 月视角:schedule 切换到新月供;上期(7/18)的还款不算本期
	report = ComputeDebts(ledger, cfg, atDate("2026-08-19"))
	ins = report.Installments[0]
	if ins.MonthlyAmount != 521000 {
		t.Errorf("切换后 MonthlyAmount = %v, want 521000", ins.MonthlyAmount)
	}
	if ins.Paid {
		t.Error("上期还款不应算入本期 Paid")
	}
	if ins.DueDate != "2026-08-20" || ins.DaysUntilDue != 1 {
		t.Errorf("DueDate = %s (%d 天), want 2026-08-20 (1 天)", ins.DueDate, ins.DaysUntilDue)
	}

	// 还款日已过未还:逾期
	report = ComputeDebts(ledger, cfg, atDate("2026-08-25"))
	ins = report.Installments[0]
	if !ins.Overdue || ins.DaysUntilDue != -5 {
		t.Errorf("Overdue = %v (%d 天), want true/-5", ins.Overdue, ins.DaysUntilDue)
	}
}

func TestComputeDebtsInstallmentEdgeCases(t *testing.T) {
	const mortgage = "Liabilities:Loan:Mortgage"

	// schedule 全在未来:本期不产生应还
	futureCfg := &DebtsConfig{
		Installments: []InstallmentConfig{{
			ID: "car", Name: "车贷", Account: mortgage, DueDay: 20,
			Schedule: []InstallmentPhase{{EffectiveFrom: "2026-09-01", Amount: 300000}},
		}},
	}
	ledger := debtLedger([]string{mortgage})
	report := ComputeDebts(ledger, futureCfg, atDate("2026-07-20"))
	if got := report.Installments[0].MonthlyAmount; got != 0 {
		t.Errorf("未来 schedule 的 MonthlyAmount = %v, want 0", got)
	}
	if report.Summary.MonthDue != 0 || report.Summary.NextDueDate != "" {
		t.Errorf("未来 schedule 不应进汇总: %v / %q", report.Summary.MonthDue, report.Summary.NextDueDate)
	}

	// 窗口内非 transfer 的正向 posting(记账修正)不置 Paid
	cfg := &DebtsConfig{
		Installments: []InstallmentConfig{{
			ID: "mortgage", Name: "房贷", Account: mortgage, DueDay: 20,
			Schedule: []InstallmentPhase{{EffectiveFrom: "2023-06-01", Amount: 543210}},
		}},
	}
	ledger = debtLedger(
		[]string{mortgage},
		mkTx("2026-07-10",
			po(mortgage, 10000),
			po("Income:Other:Adjust", -10000)),
	)
	report = ComputeDebts(ledger, cfg, atDate("2026-07-15"))
	if report.Installments[0].Paid {
		t.Error("非 transfer 的正向 posting 不应判定为月供已还")
	}
}

func TestComputeDebtsSummaryPriority(t *testing.T) {
	const huabei = "Liabilities:Alipay:Huabei"
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			ccAccount: {BillingDay: 9, DueDay: 20}, // due 7/20,已逾期
			huabei:    {BillingDay: 1, DueDay: 28}, // due 7/28,未来
		},
	}
	ledger := debtLedger(
		[]string{ccAccount, huabei},
		mkTx("2026-07-05",
			po("Expenses:Shopping:Daily", 10000),
			po(ccAccount, -10000)),
		mkTx("2026-06-20",
			po("Expenses:Food:Dinner", 5000),
			po(huabei, -5000)),
	)
	report := ComputeDebts(ledger, cfg, atDate("2026-07-23"))
	// 逾期的排最前:CMB due 7/20 早于花呗 7/28
	if report.Summary.NextDueDate != "2026-07-20" || report.Summary.NextDueName != "CMB" {
		t.Errorf("NextDue = %s %q, want 2026-07-20 CMB", report.Summary.NextDueDate, report.Summary.NextDueName)
	}
	if report.Summary.OverdueCount != 1 {
		t.Errorf("OverdueCount = %d, want 1", report.Summary.OverdueCount)
	}
	if report.Summary.MonthDue != 15000 || report.Summary.MonthRemaining != 15000 {
		t.Errorf("Summary = %v/%v, want 15000/15000", report.Summary.MonthDue, report.Summary.MonthRemaining)
	}
	// 列表按还款日排序稳定
	if report.Revolving[0].Account != ccAccount {
		t.Errorf("revolving 排序首位 = %s, want %s", report.Revolving[0].Account, ccAccount)
	}
}

func TestComputeDebtsDegraded(t *testing.T) {
	const huabei = "Liabilities:Alipay:Huabei"
	ledger := debtLedger(
		[]string{huabei},
		mkTx("2026-07-05",
			po("Expenses:Food:Dinner", 5000),
			po(huabei, -5000)),
	)

	// 配置了账本中不存在的账户:标记 AccountMissing,不 panic
	cfg := &DebtsConfig{
		Revolving: map[string]RevolvingConfig{
			"Liabilities:CreditCard:Ghost": {BillingDay: 9, DueDay: 20},
		},
	}
	report := ComputeDebts(ledger, cfg, atDate("2026-07-20"))
	if !report.Revolving[0].AccountMissing {
		t.Error("账本中不存在的账户应标 AccountMissing")
	}
	if report.Revolving[0].StatementDue != 0 {
		t.Errorf("缺失账户 StatementDue = %v, want 0", report.Revolving[0].StatementDue)
	}

	// 空配置:有欠款的账户全部列入 Unconfigured
	report = ComputeDebts(ledger, &DebtsConfig{}, atDate("2026-07-20"))
	if len(report.Unconfigured) != 1 || report.Unconfigured[0].Account != huabei {
		t.Fatalf("Unconfigured = %+v, want 仅 %s", report.Unconfigured, huabei)
	}
	if report.Unconfigured[0].Balance != 5000 {
		t.Errorf("Unconfigured Balance = %v, want 5000", report.Unconfigured[0].Balance)
	}
	// 已配置的账户不再列入 Unconfigured
	report = ComputeDebts(ledger, &DebtsConfig{
		Revolving: map[string]RevolvingConfig{huabei: {BillingDay: 1, DueDay: 10}},
	}, atDate("2026-07-20"))
	if len(report.Unconfigured) != 0 {
		t.Errorf("已配置账户仍出现在 Unconfigured: %+v", report.Unconfigured)
	}
}

func TestComputeDebtsIgnoresFutureTransactions(t *testing.T) {
	ledger := debtLedger(
		[]string{ccAccount},
		mkTx("2026-07-05",
			po("Expenses:Shopping:Daily", 10000),
			po(ccAccount, -10000)),
		// 未来日期的还款不参与任何统计
		mkTx("2026-08-01",
			po("Assets:Cash:Alipay", -10000),
			po(ccAccount, 10000)),
	)
	report := ComputeDebts(ledger, revolvingCfg(9, 20), atDate("2026-07-15"))
	if got := report.Revolving[0].Remaining; got != 10000 {
		t.Errorf("未来交易被计入,Remaining = %v, want 10000", got)
	}
}

func TestDebtsConfigValidate(t *testing.T) {
	valid := InstallmentConfig{
		ID: "m", Name: "房贷", Account: "Liabilities:Loan:Mortgage", DueDay: 20,
		Schedule: []InstallmentPhase{{EffectiveFrom: "2023-06-01", Amount: 100}},
	}
	cases := []struct {
		name    string
		cfg     DebtsConfig
		wantErr bool
	}{
		{"合法配置", DebtsConfig{
			Revolving:    map[string]RevolvingConfig{ccAccount: {BillingDay: 9, DueDay: 20}},
			Installments: []InstallmentConfig{valid},
		}, false},
		{"账单日越界", DebtsConfig{
			Revolving: map[string]RevolvingConfig{ccAccount: {BillingDay: 32, DueDay: 20}},
		}, true},
		{"还款日越界", DebtsConfig{
			Revolving: map[string]RevolvingConfig{ccAccount: {BillingDay: 9, DueDay: 0}},
		}, true},
		{"非 Liabilities 账户", DebtsConfig{
			Revolving: map[string]RevolvingConfig{"Assets:Bank:CMB": {BillingDay: 9, DueDay: 20}},
		}, true},
		{"schedule 为空", DebtsConfig{
			Installments: []InstallmentConfig{{ID: "m", Account: "Liabilities:Loan:M", DueDay: 20}},
		}, true},
		{"重复 id", DebtsConfig{
			Installments: []InstallmentConfig{valid, valid},
		}, true},
		{"缺少 id", DebtsConfig{
			Installments: []InstallmentConfig{{Account: "Liabilities:Loan:M", DueDay: 20,
				Schedule: []InstallmentPhase{{EffectiveFrom: "2023-06-01", Amount: 100}}}},
		}, true},
		{"非法生效日期", DebtsConfig{
			Installments: []InstallmentConfig{{ID: "m", Account: "Liabilities:Loan:M", DueDay: 20,
				Schedule: []InstallmentPhase{{EffectiveFrom: "2023/06/01", Amount: 100}}}},
		}, true},
		{"月供非正数", DebtsConfig{
			Installments: []InstallmentConfig{{ID: "m", Account: "Liabilities:Loan:M", DueDay: 20,
				Schedule: []InstallmentPhase{{EffectiveFrom: "2023-06-01", Amount: 0}}}},
		}, true},
	}
	for _, c := range cases {
		errs := c.cfg.Validate()
		if (len(errs) > 0) != c.wantErr {
			t.Errorf("%s: Validate() = %v, wantErr = %v", c.name, errs, c.wantErr)
		}
	}
}

func TestAmountUnmarshalJSON(t *testing.T) {
	var cfg DebtsConfig
	data := []byte(`{"installments":[{"id":"m","account":"Liabilities:Loan:M","dueDay":20,
		"schedule":[{"effectiveFrom":"2023-06-01","amount":5432.10}]}]}`)
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got := cfg.Installments[0].Schedule[0].Amount; got != 543210 {
		t.Errorf("amount = %v(分), want 543210", got)
	}

	// 超两位小数拒绝,不静默截断
	var a Amount
	if err := json.Unmarshal([]byte("1.234"), &a); err == nil {
		t.Error("三位小数应报错")
	}
}
