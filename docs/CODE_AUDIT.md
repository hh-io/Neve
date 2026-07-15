# Neve 项目深度代码审查与健康度检查报告

本报告由系统审计自动生成，归纳了当前项目在正确性、性能、安全性与可维护性等维度存在的中高风险缺陷，并附带了具体的重构优化方案。

---

## 1. 综合诊断报告

| 检查维度 | 风险等级 | 核心问题概述 | 对应修复方案 |
| :--- | :--- | :--- | :--- |
| **正确性与逻辑** | 高 | 解析器中的 `include` 存在递归调用，未做循环引用检测，在配置错误时会导致**栈溢出崩溃**；`Amount.String()` 未处理 `math.MinInt64` 溢出。 | 引入 include 链路追踪，过滤循环引用；做溢出安全检查。 |
| **性能与效率** | 中 | `checkBalances` 中核对 balance 断言采用嵌套循环，算法复杂度达 $O(B \times T \times P)$。若账本较大，存在极高的时间与 CPU 开销。 | 将断言与交易列表分别按时间升序排序，采用双指针时间线滑动累加，实现 $O(T \log T + B \log B + T \times P)$ 复杂度。 |
| **安全与健壮性** | 高 | `/api/refresh` 接口限流校验与刷新逻辑之间无原子锁，高并发请求下存在**竞态条件导致并发解析与重算**；前端 LocalStorage 损坏可导致 `JSON.parse` 报错致使页面阻断。 | 引入 `refreshMu` 并发锁，并实施“双重检查（Double-Checked Locking）”；前端 `JSON.parse` 增加容错防御。 |
| **可维护性与规范** | 低 | 静态文件使用自定义路由逐一 ReadFile 后下发，未提供断点续传（Range）、强缓存与自动 MIME 识别支持。 | 建议未来使用标准 `http.FileServer` 或 Gin 的原生静态分发进行替换。 |

---

## 2. 具体缺陷分析与修改方案

### 2.1 递归 Include 未防范循环引用
* **缺陷位置**：`server/parser/parser.go` 的 `parseFile` 函数
* **缺陷描述**：当 Beancount 文件因为人工编辑或外部脚本生成错误，出现循环 Include（例如 `A.bean` include `B.bean` 且 `B.bean` include `A.bean`）时，`parseFile` 递归调用由于没有追踪当前正在处理的文件，会导致无限递归，直至栈溢出（Stack Overflow）引发进程崩溃退出。
* **重构方案**：在 `Parser` 结构体中增加一个 `activeFiles` 的哈希表用来跟踪当前正在解析的 Include 链路（相当于调用栈）。在进入 `parseFile` 时将绝对路径计入，退出时移除。若发现待解析路径已被计入，则中断递归并记录 `INCLUDE_CYCLE` 错误。
* **优化后代码**：
```go
// server/parser/parser.go

type Parser struct {
	dataDir     string
	now         time.Time
	seq         int
	activeFiles map[string]bool // 追踪当前解析调用栈中的绝对路径，防范循环引用
}

func NewParser(dataDir string) *Parser {
	return &Parser{
		dataDir:     dataDir, 
		now:         time.Now(),
		activeFiles: make(map[string]bool),
	}
}

func (p *Parser) parseFile(filePath string, ledger *Ledger) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	
	// 检测循环 include，记录为软错误并跳过
	if p.activeFiles[absPath] {
		sourceFile := p.relPath(filePath)
		ledger.addIssue(sourceFile, 0, "error", "INCLUDE_CYCLE",
			fmt.Sprintf("检测到循环 include 引用: %s", sourceFile))
		return nil
	}
	
	// 标记开始解析当前文件
	p.activeFiles[absPath] = true
	defer delete(p.activeFiles, absPath) // 解析完后安全出栈

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
    
	// ... 后续行读取与匹配逻辑保持不变
}
```

---

### 2.2 `checkBalances` 算法时间复杂度过高 ($O(N^2)$ 级)
* **缺陷位置**：`server/parser/parser.go` 的 `checkBalances` 函数
* **缺陷描述**：当前的 `checkBalances` 实现对每一个 Balance 余额断言都要重新扫描并累加全量的交易和 postings，算法复杂度为 $O(B \times T \times P)$。若记账数据规模上升，频繁调用接口或执行 `Refresh` 会严重损耗服务器性能。
* **重构方案**：使用双指针与时间线合并策略。首先将断言和交易各自在内存中按时间升序排列，然后使用单次双指针扫描完成累加。通过维护累计余额哈希表来做到 $O(1)$ 的余额查询，将算法时间复杂度优化至接近线性水平 $O(T \log T + B \log B + T \times P)$。
* **优化后代码**：
```go
// server/parser/parser.go

func (p *Parser) checkBalances(ledger *Ledger, openAccounts map[string]bool) {
	if len(ledger.Balances) == 0 {
		return
	}

	// 1. 结构封装，记录 balance 原始索引，以保证校验输出后的切片顺序一致
	type indexedBalance struct {
		index int
		b     Balance
	}
	balances := make([]indexedBalance, len(ledger.Balances))
	for i, b := range ledger.Balances {
		balances[i] = indexedBalance{index: i, b: b}
	}

	// 按照断言的日期升序排序
	sort.SliceStable(balances, func(i, j int) bool {
		return balances[i].b.Date.Before(balances[j].b.Date)
	})

	// 2. 对交易列表进行升序副本排序（原列表为降序供前端显示，不能直接修改）
	txs := make([]Transaction, len(ledger.Transactions))
	copy(txs, ledger.Transactions)
	sort.SliceStable(txs, func(i, j int) bool {
		ti, tj := txs[i], txs[j]
		if !ti.Date.Equal(tj.Date) {
			return ti.Date.Before(tj.Date)
		}
		return ti.seq < tj.seq
	})

	// 3. 时间线滑动扫描
	currentBalances := make(map[string]Amount)
	txIdx := 0
	n := len(txs)

	checks := make([]BalanceCheck, len(ledger.Balances))
	filled := make([]bool, len(ledger.Balances))

	for _, ib := range balances {
		b := ib.b
		idx := ib.index

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

		// 双指针移动：累加断言日期前 (tx.Date < b.Date) 的所有 postings
		for txIdx < n && txs[txIdx].Date.Before(b.Date) {
			for _, po := range txs[txIdx].Postings {
				currentBalances[po.Account] += po.Amount
			}
			txIdx++
		}

		actual := currentBalances[b.Account]
		diff := actual - b.Amount
		check := BalanceCheck{
			Date:     b.Date.Format("2006-01-02"),
			Account:  b.Account,
			Expected: b.Amount,
			Actual:   actual,
			Diff:     diff,
			OK:       diff == 0,
		}
		
		checks[idx] = check
		filled[idx] = true

		if !check.OK {
			ledger.addIssue(b.SourceFile, b.SourceLine, "error", "BALANCE_FAILED",
				fmt.Sprintf("balance 断言失败: %s 在 %s 应为 %s,实际 %s,差额 %s",
					b.Account, check.Date, b.Amount, actual, diff))
		}
	}

	// 4. 将完成核对的结果按原账本内物理顺序写回
	for i, f := range filled {
		if f {
			ledger.BalanceChecks = append(ledger.BalanceChecks, checks[i])
		}
	}
}
```

---

### 2.3 并发刷新导致竞态解析与资源浪费
* **缺陷位置**：`server/api/handler.go` 的 `handleRefresh` 函数与 `Server` 结构体
* **缺陷描述**：`/api/refresh` 中，虽然使用了限流判定 `sinceLastRefresh < 5*time.Second`，但由于校验和 `Refresh` 并没有作为一个原子锁结构。如果有两个甚至更多请求同时到达，它们都会检查到 `sinceLastRefresh` 超过了限制，从而同时进入 `s.Refresh()` 执行磁盘 I/O 读写和 CPU 密集的语法分析重算，最后产生严重资源耗尽。
* **重构方案**：在 `Server` 结构体中新增排他刷新锁 `refreshMu`，同时在获取锁之后使用“双重检查（Double-Checked Locking）”机制，防止等待排队的重复刷新请求重复读取计算。
* **优化后代码**：
```go
// server/api/handler.go

type Server struct {
	dataDir     string
	mu          sync.RWMutex
	analytics   *parser.Analytics
	lastRefresh time.Time
	budgetMu    sync.Mutex
	refreshMu   sync.Mutex // 保证全局仅有一个 refresh 任务在解析与重算的互斥锁
}

func (s *Server) handleRefresh(c *gin.Context) {
	// 限流的快速初步检查
	s.mu.RLock()
	sinceLastRefresh := time.Since(s.lastRefresh)
	s.mu.RUnlock()

	if sinceLastRefresh < 5*time.Second {
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error":      ErrRateLimited,
			"retryAfter": (5*time.Second - sinceLastRefresh).Seconds(),
		})
		return
	}

	// 加锁，确保在此期间其余并行请求等待或被消化
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()

	// 双重校验：避免在排队等待锁的请求进来后重复解析
	s.mu.RLock()
	sinceLastRefresh = time.Since(s.lastRefresh)
	s.mu.RUnlock()
	if sinceLastRefresh < 5*time.Second {
		c.JSON(http.StatusOK, gin.H{
			"message":     "data refreshed (cached)",
			"summary":     s.analytics.Summary,
			"issueCount":  len(s.analytics.ParseIssues),
			"parseIssues": s.analytics.ParseIssues,
		})
		return
	}

	// 执行实际的解析和分析工作
	if err := s.Refresh(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": NewAPIError("REFRESH_FAILED", err.Error()),
		})
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{
		"message":     "data refreshed",
		"summary":     s.analytics.Summary,
		"issueCount":  len(s.analytics.ParseIssues),
		"parseIssues": s.analytics.ParseIssues,
	})
}
```

---

### 2.4 前端本地配置解析崩溃容错
* **缺陷位置**：`web/src/composables/useBudgets.ts` 的 `loadBudgets` 方法
* **缺陷描述**：当服务端连接失败，前端会自动退避去加载 LocalStorage 内备份的 `neve-budgets` 数据。但是，`JSON.parse(...)` 对损坏的格式是不具备免疫力的。若用户存储脏数据（如空白、非 JSON 数据等），会直接抛出 `SyntaxError` 并且程序没有做捕获，进而打断页面启动。
* **重构方案**：对本地的 `JSON.parse` 逻辑做完备的 `try-catch` 容错隔离，一旦异常则优雅回退至空对象 `{}`，确保不引发崩溃。
* **优化后代码**：
```typescript
// web/src/composables/useBudgets.ts

// 首屏加载:服务端不可达时回退 localStorage 备份
async function loadBudgets(): Promise<void> {
  if (loaded) return
  loaded = true
  try {
    const res = await fetch('/api/budgets')
    if (res.ok) {
      budgets.value = await res.json()
      return
    }
  } catch {
    // 忽略网络级错误，交由下方本地缓存备份还原
  }

  // 严格防范 LocalStorage 数据损坏引发的崩页问题
  try {
    const backup = localStorage.getItem('neve-budgets')
    budgets.value = backup ? JSON.parse(backup) : {}
  } catch (e) {
    budgets.value = {}
  }
}
```
