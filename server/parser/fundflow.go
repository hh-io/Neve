package parser

import (
	"sort"
	"strings"
)

// 资金流向图(收支分析页的桑基图)的三层聚合:收入来源 → 资金账户 → 支出分类。
//
// 放在后端算而不是让前端遍历 transactions:这张图与同页的 ExpenseByCategory /
// IncomeBreakdown 并排展示、共用「本月」口径,前端自行聚合等于把净额口径(退款冲减)、
// 未来日期过滤、时区归属各重写一遍,任何一处漏掉就出现两张卡数字对不上的情况。

// FundFlowNode 是流向图的一个节点。Key 全局唯一(带层前缀),Label 是展示名:
// account 层已消歧到不重名的最短后缀,income/expense 层给原始分类键,由前端映射中文。
type FundFlowNode struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Type  string `json:"type"` // income | account | expense
}

// FundFlowLink 是相邻两层之间的一条流量,金额恒为正。
type FundFlowLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Amount Amount `json:"amount"`
}

type FundFlow struct {
	Nodes []FundFlowNode `json:"nodes"`
	Links []FundFlowLink `json:"links"`
}

// 交易里找不到任何资金腿时的兜底节点(理论上不该出现:一笔平衡的收支总有资金腿)
const fundFlowUnknownAccount = "(未记名账户)"

type fundShare struct {
	key    string
	amount Amount
}

type fundEdge struct {
	source string
	target string
}

// computeFundFlow 汇总本月的资金流向。txs 必须已按「本月 + 非未来 + 非 opening」过滤好,
// 与 ExpenseByCategory 同源同期;allAccounts 是账本里 open 过的全部账户,只用来给节点取名。
func computeFundFlow(txs []Transaction, allAccounts []Account) FundFlow {
	gross := make(map[fundEdge]Amount)
	expenseGross := make(map[string]Amount)  // 分类正向支出合计
	expenseRefund := make(map[string]Amount) // 分类退款合计(Expenses 负腿)
	incomeGross := make(map[string]Amount)
	incomeOffset := make(map[string]Amount) // 收入冲正(Income 正腿)
	usedAccounts := make(map[string]bool)

	for _, tx := range txs {
		// 付款方 / 收款方:同一笔交易里资金腿可能有多条(组合支付、还款含手续费),
		// 取第一条会让归属依赖 posting 的书写顺序,故按金额占比分摊
		var payers, receivers []fundShare
		for _, p := range tx.Postings {
			switch accountRoot(p.Account) {
			case "Assets", "Liabilities":
				if p.Amount < 0 {
					payers = append(payers, fundShare{p.Account, -p.Amount})
				} else if p.Amount > 0 {
					receivers = append(receivers, fundShare{p.Account, p.Amount})
				}
			}
		}
		// 支出的钱从付款方出;没有付款方时(如工资代扣:Income 负腿 + Expenses 正腿 +
		// Assets 正腿)钱其实是从收入直接扣的,按流向图的三层语义仍记在入账账户上
		expenseFrom := payers
		if len(expenseFrom) == 0 {
			expenseFrom = receivers
		}
		incomeTo := receivers
		if len(incomeTo) == 0 {
			incomeTo = payers
		}

		for _, p := range tx.Postings {
			switch accountRoot(p.Account) {
			case "Expenses":
				category := getExpenseCategory(p.Account)
				if p.Amount > 0 {
					expenseGross[category] += p.Amount
					for _, share := range allocateAmount(p.Amount, expenseFrom, fundFlowUnknownAccount) {
						if share.amount <= 0 {
							continue
						}
						gross[fundEdge{accountNodeKey(share.key), expenseNodeKey(category)}] += share.amount
						usedAccounts[share.key] = true
					}
				} else if p.Amount < 0 {
					expenseRefund[category] += -p.Amount
				}
			case "Income":
				source := getIncomeSource(p.Account)
				if p.Amount < 0 {
					incomeGross[source] += -p.Amount
					for _, share := range allocateAmount(-p.Amount, incomeTo, fundFlowUnknownAccount) {
						if share.amount <= 0 {
							continue
						}
						gross[fundEdge{incomeNodeKey(source), accountNodeKey(share.key)}] += share.amount
						usedAccounts[share.key] = true
					}
				} else if p.Amount > 0 {
					incomeOffset[source] += p.Amount
				}
			}
		}
	}

	// 退款/冲正按分类等比缩减该分类的全部链路,总额对齐 ExpenseByCategory / IncomeBreakdown
	// 的净额口径。不能从某条链路上直接扣:退款到账的账户与原消费账户常常不是同一个
	// (退款进支付宝、原单走白条),按链路扣会把那条链路压成负数。
	links := scaleToNet(gross, expenseNodeKey, expenseGross, expenseRefund)
	links = scaleToNet(links, incomeNodeKey, incomeGross, incomeOffset)

	return assembleFundFlow(links, usedAccounts, allAccounts)
}

func accountNodeKey(account string) string  { return "account:" + account }
func expenseNodeKey(category string) string { return "expense:" + category }
func incomeNodeKey(source string) string    { return "income:" + source }

// scaleToNet 把归属同一个分类节点的链路按净额等比缩放。keyOf 决定分类节点在链路的哪一端:
// 支出分类是 target,收入来源是 source。净额 ≤ 0 的分类(整月只有退款)链路全部丢弃——
// 流向图画不出负流量,这与 ExpenseByCategory 允许出现负值是有意的差异。
func scaleToNet(links map[fundEdge]Amount, keyOf func(string) string, gross, offset map[string]Amount) map[fundEdge]Amount {
	if len(offset) == 0 {
		return links
	}
	out := make(map[fundEdge]Amount, len(links))
	for edge, amount := range links {
		out[edge] = amount
	}
	for category, off := range offset {
		if off <= 0 {
			continue
		}
		node := keyOf(category)
		var parts []fundShare
		for edge, amount := range out {
			if edge.source == node || edge.target == node {
				parts = append(parts, fundShare{edgeKey(edge), amount})
			}
		}
		// map 遍历无序,分摊尾差落在最后一份上,不排序会让同一份账本每次输出不同
		sort.Slice(parts, func(i, j int) bool { return parts[i].key < parts[j].key })

		net := gross[category] - off
		if net <= 0 {
			for _, part := range parts {
				delete(out, decodeEdgeKey(part.key))
			}
			continue
		}
		for _, share := range allocateAmount(net, parts, "") {
			edge := decodeEdgeKey(share.key)
			if share.amount <= 0 {
				delete(out, edge)
				continue
			}
			out[edge] = share.amount
		}
	}
	return out
}

// 链路在分摊时要当成 fundShare 的 key 传递,用换行分隔(账户名不含换行)
const edgeKeySep = "\n"

func edgeKey(e fundEdge) string { return e.source + edgeKeySep + e.target }

func decodeEdgeKey(key string) fundEdge {
	source, target, _ := strings.Cut(key, edgeKeySep)
	return fundEdge{source, target}
}

// allocateAmount 把 total 按各份权重分摊,尾差落最后一份,保证结果之和恒等于 total
// (定点分整除会丢余数,不兜底就会出现流量对不上分类合计的分级误差)。
// parts 为空时全额归 fallback;权重全为 0 时全额归第一份。
// 乘法在分单位下需单笔超过 9e10 元才溢出 int64,个人账本不可达。
func allocateAmount(total Amount, parts []fundShare, fallback string) []fundShare {
	if len(parts) == 0 {
		if fallback == "" {
			return nil
		}
		return []fundShare{{fallback, total}}
	}
	if len(parts) == 1 {
		return []fundShare{{parts[0].key, total}}
	}
	var weight Amount
	for _, p := range parts {
		weight += p.amount
	}
	out := make([]fundShare, 0, len(parts))
	assigned := Amount(0)
	for i, p := range parts {
		var share Amount
		switch {
		case weight <= 0:
			if i == 0 {
				share = total
			}
		case i == len(parts)-1:
			share = total - assigned
		default:
			share = Amount(int64(total) * int64(p.amount) / int64(weight))
		}
		assigned += share
		out = append(out, fundShare{p.key, share})
	}
	return out
}

// assembleFundFlow 把链路集合整理成排序稳定的节点/链路列表:
// 前端用 layoutIterations: 0 保持给定顺序,顺序一抖动图就会在每次 refresh 后重排。
func assembleFundFlow(links map[fundEdge]Amount, usedAccounts map[string]bool, allAccounts []Account) FundFlow {
	flow := FundFlow{Nodes: make([]FundFlowNode, 0), Links: make([]FundFlowLink, 0, len(links))}

	throughput := make(map[string]Amount) // 节点吞吐 = max(流入, 流出),决定同层排序
	inflow := make(map[string]Amount)
	outflow := make(map[string]Amount)
	for edge, amount := range links {
		if amount <= 0 {
			continue
		}
		flow.Links = append(flow.Links, FundFlowLink{Source: edge.source, Target: edge.target, Amount: amount})
		outflow[edge.source] += amount
		inflow[edge.target] += amount
	}
	if len(flow.Links) == 0 {
		return flow
	}
	for node, amount := range outflow {
		throughput[node] = amount
	}
	for node, amount := range inflow {
		if amount > throughput[node] {
			throughput[node] = amount
		}
	}

	// 末段重名的账户(Assets:Bank:CMB 与 Liabilities:CreditCard:CMB)必须展示成不同的名字,
	// 否则它们在图上塌成同一个节点的视觉印象——那是两个账户。
	// 消歧的比较范围是账本里全部资金账户而非本月用到的那几个:按当月范围算的话,
	// 同一张卡会在储蓄卡有流水的月份叫 CreditCard:CMB、没流水的月份叫 CMB,月月换名。
	universe := make([]string, 0, len(allAccounts))
	for _, account := range allAccounts {
		switch account.Type {
		case "Assets", "Liabilities":
			universe = append(universe, account.Name)
		}
	}
	for account := range usedAccounts {
		if !containsString(universe, account) {
			universe = append(universe, account) // 账户未 open 时账本会记 issue,这里仍要有名字
		}
	}
	sort.Strings(universe)
	labels := shortenAccounts(universe)

	typeOf := func(key string) string {
		prefix, _, _ := strings.Cut(key, ":")
		return prefix
	}
	typeRank := map[string]int{"income": 0, "account": 1, "expense": 2}

	for node := range throughput {
		kind := typeOf(node)
		label := strings.TrimPrefix(node, kind+":")
		if kind == "account" {
			if short, ok := labels[label]; ok {
				label = short
			}
		}
		flow.Nodes = append(flow.Nodes, FundFlowNode{Key: node, Label: label, Type: kind})
	}
	sort.Slice(flow.Nodes, func(i, j int) bool {
		a, b := flow.Nodes[i], flow.Nodes[j]
		if a.Type != b.Type {
			return typeRank[a.Type] < typeRank[b.Type]
		}
		if throughput[a.Key] != throughput[b.Key] {
			return throughput[a.Key] > throughput[b.Key]
		}
		return a.Key < b.Key
	})
	sort.Slice(flow.Links, func(i, j int) bool {
		a, b := flow.Links[i], flow.Links[j]
		if a.Amount != b.Amount {
			return a.Amount > b.Amount
		}
		if a.Source != b.Source {
			return a.Source < b.Source
		}
		return a.Target < b.Target
	})
	return flow
}

func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// shortenAccounts 给账户全名取「不与同批其他账户重名的最短后缀」:
// 末段唯一就用末段(CMB),冲突则向前多带一段(Bank:CMB / CreditCard:CMB)。
func shortenAccounts(accounts []string) map[string]string {
	segments := make(map[string][]string, len(accounts))
	for _, account := range accounts {
		segments[account] = strings.Split(account, ":")
	}
	out := make(map[string]string, len(accounts))
	for _, account := range accounts {
		own := segments[account]
		label := account
		for depth := 1; depth <= len(own); depth++ {
			candidate := strings.Join(own[len(own)-depth:], ":")
			unique := true
			for _, other := range accounts {
				if other == account {
					continue
				}
				peer := segments[other]
				if depth <= len(peer) && strings.Join(peer[len(peer)-depth:], ":") == candidate {
					unique = false
					break
				}
			}
			if unique {
				label = candidate
				break
			}
		}
		out[account] = label
	}
	return out
}
