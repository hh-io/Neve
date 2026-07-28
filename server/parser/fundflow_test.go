package parser

import (
	"testing"
	"time"
)

// 流向图用例要用到 openHeader 之外的账户(多张卡、多个分类才看得出分摊与消歧)
const flowOpenHeader = `option "operating_currency" "CNY"
2020-01-01 open Assets:Cash:WeChat CNY
2020-01-01 open Assets:Cash:Alipay CNY
2020-01-01 open Assets:Bank:CMB CNY
2020-01-01 open Liabilities:CreditCard:CMB CNY
2020-01-01 open Liabilities:JD:BNPL CNY
2020-01-01 open Expenses:Food:Delivery CNY
2020-01-01 open Expenses:Food:Grocery CNY
2020-01-01 open Expenses:Shopping:Digital CNY
2020-01-01 open Expenses:Shopping:Clothing CNY
2020-01-01 open Expenses:Transport:Metro CNY
2020-01-01 open Expenses:Housing:Rent CNY
2020-01-01 open Expenses:Financial:ServiceFee CNY
2020-01-01 open Expenses:Tax:Income CNY
2020-01-01 open Income:Salary CNY
2020-01-01 open Income:SecondHand CNY
`

func parseFlow(t *testing.T, body string) *Ledger {
	t.Helper()
	return parseFixture(t, map[string]string{"main.bean": flowOpenHeader + body})
}

func flowLink(t *testing.T, flow FundFlow, source, target string) Amount {
	t.Helper()
	for _, link := range flow.Links {
		if link.Source == source && link.Target == target {
			return link.Amount
		}
	}
	return 0
}

func flowNode(t *testing.T, flow FundFlow, key string) (FundFlowNode, bool) {
	t.Helper()
	for _, node := range flow.Nodes {
		if node.Key == key {
			return node, true
		}
	}
	return FundFlowNode{}, false
}

// 流向图与同页的 ExpenseByCategory / IncomeBreakdown 并排展示,总额对不上就是可见的 bug
func TestFundFlowMatchesCategoryTotals(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	ledger := parseFlow(t, `
2026-06-25 * "老板" "上月工资"
  Income:Salary         -1000.00 CNY
  Assets:Cash:WeChat     1000.00 CNY

2026-07-05 * "美团" "午餐"
  Expenses:Food:Delivery   32.50 CNY
  Assets:Cash:Alipay      -32.50 CNY

2026-07-06 * "京东" "耳机"
  Expenses:Shopping:Digital  1899.00 CNY
  Liabilities:JD:BNPL       -1899.00 CNY

2026-07-09 * "拼多多" "退款-数据线"
  Assets:Cash:Alipay        19.90 CNY
  Expenses:Shopping:Digital -19.90 CNY

2026-07-14 * "公司" "七月工资"
  Income:Salary        -15000.00 CNY
  Assets:Bank:CMB       15000.00 CNY

2026-07-31 * "未来" "月底预记的房租"
  Expenses:Housing:Rent  3000.00 CNY
  Assets:Bank:CMB       -3000.00 CNY
`)

	a := AnalyzeAt(ledger, now)

	var flowExpense, flowIncome Amount
	for _, link := range a.FundFlow.Links {
		if link.Amount <= 0 {
			t.Errorf("链路金额必须为正: %+v", link)
		}
		switch {
		case len(link.Target) > 8 && link.Target[:8] == "expense:":
			flowExpense += link.Amount
		case len(link.Source) > 7 && link.Source[:7] == "income:":
			flowIncome += link.Amount
		}
	}

	var categoryExpense Amount
	for _, c := range a.ExpenseByCategory {
		categoryExpense += c.Amount
	}
	var breakdownIncome Amount
	for _, s := range a.IncomeBreakdown {
		breakdownIncome += s.Amount
	}

	// 退款冲减后 Shopping 净额 1879.10 + Food 32.50;上月工资与未来房租都不在本月口径里
	if flowExpense != categoryExpense {
		t.Errorf("流向图支出合计 = %s,ExpenseByCategory 合计 = %s,两者必须同期同口径", flowExpense, categoryExpense)
	}
	if flowExpense != 191160 {
		t.Errorf("流向图支出合计 = %s,期望 1911.60(退款已冲减、未来房租未计入)", flowExpense)
	}
	if flowIncome != breakdownIncome || flowIncome != 1500000 {
		t.Errorf("流向图收入合计 = %s,IncomeBreakdown 合计 = %s,期望 15000.00", flowIncome, breakdownIncome)
	}
	if got := flowLink(t, a.FundFlow, "account:Liabilities:JD:BNPL", "expense:Shopping"); got != 187910 {
		t.Errorf("白条→购物 = %s,期望 1879.10(整个分类只有这一条链路,退款全额落在它身上)", got)
	}
	if got := flowLink(t, a.FundFlow, "account:Assets:Bank:CMB", "expense:Housing"); got != 0 {
		t.Errorf("未来日期的房租不该出现在流向图里: %s", got)
	}
}

// 末段同名的两个账户(招商储蓄卡 / 招行信用卡)必须是两个节点、两个展示名
func TestFundFlowDistinguishesSameLeafAccounts(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	ledger := parseFlow(t, `
2026-07-01 * "公司" "工资"
  Income:Salary        -10000.00 CNY
  Assets:Bank:CMB       10000.00 CNY

2026-07-05 * "超市" "刷信用卡"
  Expenses:Food:Grocery       200.00 CNY
  Liabilities:CreditCard:CMB -200.00 CNY

2026-07-06 * "地铁" "储蓄卡扣款"
  Expenses:Transport:Metro   6.00 CNY
  Assets:Bank:CMB           -6.00 CNY
`)

	flow := AnalyzeAt(ledger, now).FundFlow

	savings, ok := flowNode(t, flow, "account:Assets:Bank:CMB")
	if !ok {
		t.Fatalf("缺少储蓄卡节点: %+v", flow.Nodes)
	}
	credit, ok := flowNode(t, flow, "account:Liabilities:CreditCard:CMB")
	if !ok {
		t.Fatalf("缺少信用卡节点: %+v", flow.Nodes)
	}
	if savings.Label == credit.Label {
		t.Errorf("末段同名的两个账户展示名不能相同: %q / %q", savings.Label, credit.Label)
	}
	if savings.Label != "Bank:CMB" || credit.Label != "CreditCard:CMB" {
		t.Errorf("冲突时应向前多带一段: %q / %q", savings.Label, credit.Label)
	}
	if got := flowLink(t, flow, "account:Liabilities:CreditCard:CMB", "expense:Food"); got != 20000 {
		t.Errorf("信用卡→餐饮 = %s,期望 200.00", got)
	}
	if got := flowLink(t, flow, "account:Assets:Bank:CMB", "expense:Transport"); got != 600 {
		t.Errorf("储蓄卡→交通 = %s,期望 6.00", got)
	}
}

// 消歧范围是账本全部账户:只按当月用到的账户算,同一张卡会月月换名
func TestFundFlowLabelsStableAcrossMonths(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	// 本月只有信用卡有流水,储蓄卡 Assets:Bank:CMB 仍 open 着
	ledger := parseFlow(t, `
2026-07-05 * "超市" "刷信用卡"
  Expenses:Food:Grocery       200.00 CNY
  Liabilities:CreditCard:CMB -200.00 CNY
`)

	flow := AnalyzeAt(ledger, now).FundFlow

	credit, ok := flowNode(t, flow, "account:Liabilities:CreditCard:CMB")
	if !ok {
		t.Fatalf("缺少信用卡节点: %+v", flow.Nodes)
	}
	if credit.Label != "CreditCard:CMB" {
		t.Errorf("展示名 = %q,期望 CreditCard:CMB(储蓄卡本月没流水也要参与消歧)", credit.Label)
	}
}

// 资金腿不止一条时按金额占比分摊,不能取第一条——那样归属就取决于 posting 的书写顺序
func TestFundFlowSplitsAcrossFundingLegs(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	ledger := parseFlow(t, `
2026-07-02 * "商场" "组合支付"
  Expenses:Shopping:Clothing  300.00 CNY
  Assets:Cash:WeChat         -100.00 CNY
  Assets:Cash:Alipay         -200.00 CNY

2026-07-03 * "招商银行" "信用卡还款(含手续费)"
  Assets:Cash:Alipay         -2005.00 CNY
  Liabilities:CreditCard:CMB  2000.00 CNY
  Expenses:Financial:ServiceFee   5.00 CNY
`)

	flow := AnalyzeAt(ledger, now).FundFlow

	if got := flowLink(t, flow, "account:Assets:Cash:WeChat", "expense:Shopping"); got != 10000 {
		t.Errorf("微信→购物 = %s,期望 100.00(按 1:2 分摊)", got)
	}
	if got := flowLink(t, flow, "account:Assets:Cash:Alipay", "expense:Shopping"); got != 20000 {
		t.Errorf("支付宝→购物 = %s,期望 200.00(按 1:2 分摊)", got)
	}
	// 还款那笔只有支付宝是付款方,信用卡是收款方:手续费不能算到信用卡头上
	if got := flowLink(t, flow, "account:Assets:Cash:Alipay", "expense:Financial"); got != 500 {
		t.Errorf("支付宝→金融 = %s,期望 5.00(手续费归付款方)", got)
	}
	if got := flowLink(t, flow, "account:Liabilities:CreditCard:CMB", "expense:Financial"); got != 0 {
		t.Errorf("信用卡是这笔还款的收款方,不该背手续费: %s", got)
	}
	// 转账本金不产生流量:两条腿都是资金账户,没有收入/支出腿
	for _, link := range flow.Links {
		if link.Amount == 200000 {
			t.Errorf("还款本金不该成为流量: %+v", link)
		}
	}
}

// 分摊的尾差必须收敛,否则各链路之和会与分类合计差几分
func TestFundFlowAllocationHasNoRemainder(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	ledger := parseFlow(t, `
2026-07-02 * "三方分摊" "除不尽的分摊"
  Expenses:Food:Delivery     10.00 CNY
  Assets:Cash:WeChat         -3.33 CNY
  Assets:Cash:Alipay         -3.33 CNY
  Assets:Bank:CMB            -3.34 CNY

2026-07-03 * "商家" "部分退款"
  Assets:Cash:WeChat          3.33 CNY
  Expenses:Food:Delivery     -3.33 CNY
`)

	a := AnalyzeAt(ledger, now)

	var flowTotal Amount
	for _, link := range a.FundFlow.Links {
		flowTotal += link.Amount
	}
	var categoryTotal Amount
	for _, c := range a.ExpenseByCategory {
		categoryTotal += c.Amount
	}
	if flowTotal != categoryTotal || flowTotal != 667 {
		t.Errorf("流向图合计 = %s,分类合计 = %s,期望 6.67(尾差必须落在某一条链路上)", flowTotal, categoryTotal)
	}
}

// 整月只有退款时净额为负,流向图画不出负流量,该分类整体消失
func TestFundFlowDropsNegativeNetCategory(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	ledger := parseFlow(t, `
2026-07-02 * "商家" "上月订单的退款"
  Assets:Cash:WeChat         50.00 CNY
  Expenses:Shopping:Digital -50.00 CNY
`)

	flow := AnalyzeAt(ledger, now).FundFlow

	if len(flow.Links) != 0 || len(flow.Nodes) != 0 {
		t.Errorf("净额为负的分类不该出现在流向图里: %+v", flow)
	}
}

// 工资代扣:没有付款方资金腿时,支出仍要挂在入账账户上而不是丢失
func TestFundFlowWithholdingFallsBackToReceiver(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	ledger := parseFlow(t, `
2026-07-10 * "公司" "工资(代扣个税)"
  Income:Salary       -10000.00 CNY
  Expenses:Tax:Income   1000.00 CNY
  Assets:Bank:CMB       9000.00 CNY
`)

	flow := AnalyzeAt(ledger, now).FundFlow

	if got := flowLink(t, flow, "income:Salary", "account:Assets:Bank:CMB"); got != 1000000 {
		t.Errorf("工资→银行 = %s,期望 10000.00(税前全额流入)", got)
	}
	if got := flowLink(t, flow, "account:Assets:Bank:CMB", "expense:Tax"); got != 100000 {
		t.Errorf("银行→税 = %s,期望 1000.00", got)
	}
}

// JSON 序列化前的顺序必须稳定:map 遍历随机,前端用 layoutIterations: 0 直接吃这个顺序
func TestFundFlowOrderIsStable(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.Local)

	source := `
2026-07-01 * "公司" "工资"
  Income:Salary       -10000.00 CNY
  Assets:Bank:CMB      10000.00 CNY

2026-07-02 * "闲鱼" "卖旧手机"
  Income:SecondHand     -500.00 CNY
  Assets:Cash:Alipay     500.00 CNY

2026-07-03 * "超市" "买菜"
  Expenses:Food:Grocery  200.00 CNY
  Assets:Bank:CMB       -200.00 CNY

2026-07-04 * "地铁" "通勤"
  Expenses:Transport:Metro   6.00 CNY
  Assets:Cash:Alipay        -6.00 CNY
`

	first := AnalyzeAt(parseFlow(t, source), now).FundFlow
	for i := 0; i < 8; i++ {
		again := AnalyzeAt(parseFlow(t, source), now).FundFlow
		if len(again.Nodes) != len(first.Nodes) || len(again.Links) != len(first.Links) {
			t.Fatalf("节点/链路数量不稳定: %+v vs %+v", first, again)
		}
		for j := range first.Nodes {
			if again.Nodes[j] != first.Nodes[j] {
				t.Fatalf("节点顺序不稳定: %+v vs %+v", first.Nodes, again.Nodes)
			}
		}
		for j := range first.Links {
			if again.Links[j] != first.Links[j] {
				t.Fatalf("链路顺序不稳定: %+v vs %+v", first.Links, again.Links)
			}
		}
	}

	// 节点按层排序:收入在前、账户居中、支出在后,层内按吞吐降序
	wantOrder := []string{
		"income:Salary", "income:SecondHand",
		"account:Assets:Bank:CMB", "account:Assets:Cash:Alipay",
		"expense:Food", "expense:Transport",
	}
	if len(first.Nodes) != len(wantOrder) {
		t.Fatalf("节点数 = %d,期望 %d: %+v", len(first.Nodes), len(wantOrder), first.Nodes)
	}
	for i, want := range wantOrder {
		if first.Nodes[i].Key != want {
			t.Errorf("节点[%d] = %s,期望 %s", i, first.Nodes[i].Key, want)
		}
	}
}
