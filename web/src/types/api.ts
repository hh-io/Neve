// API 契约类型 —— 逐字段对照 server/parser/analytics.go 与 server/parser/parser.go 的 struct JSON tag。
//
// 序列化约定(见 CLAUDE.md 与 amount.go):
// - 后端 Amount 类型 MarshalJSON 输出「元」为单位的 JSON 数字,前端一律按 number 消费。
// - Go time.Time 字段(date / lastUpdated)序列化为 RFC3339 字符串。
// - 改后端 struct 时必须同步本文件,否则前后端契约脱节。

/** 金额:后端定点分(int64)序列化为「元」数字。 */
export type Yuan = number

/** 交易口径,由后端 classifyTransaction 唯一计算,前端禁止从 postings 推断。 */
export type TransactionKind = 'expense' | 'income' | 'transfer' | 'opening' | 'mixed'

// --- parser.go ---

export interface ParseIssue {
  file: string
  line: number
  severity: 'error' | 'warning'
  code: string
  message: string
}

export interface BalanceCheck {
  date: string
  account: string
  expected: Yuan
  actual: Yuan
  diff: Yuan
  ok: boolean
}

export interface Posting {
  account: string
  amount: Yuan
  currency: string
}

export interface Transaction {
  date: string
  flag: string // "*" 或 "!"
  payee: string
  narration: string
  postings: Posting[]
  tags: string[]
  sourceFile: string
  sourceLine: number

  // 以下字段由 Analyze 阶段计算
  kind: TransactionKind
  category: string
  displayAmount: Yuan
  transferAmount: Yuan
  feeAmount: Yuan
  isTransfer: boolean
}

// --- analytics.go ---

export interface Summary {
  netWorth: Yuan
  totalAssets: Yuan
  totalLiabilities: Yuan
  monthIncome: Yuan
  monthExpense: Yuan
  monthBalance: Yuan
  transactionCount: number
  trackingDays: number
  firstDate: string
  lastUpdated: string
}

export interface CategoryAmount {
  category: string
  amount: Yuan
  percent: number
  count: number
}

export interface AccountBalance {
  account: string
  balance: Yuan
  currency: string
  type: string
}

export interface MonthlyData {
  month: string
  income: Yuan
  expense: Yuan
  balance: Yuan
}

export interface DailyData {
  date: string
  income: Yuan
  expense: Yuan
  balance: Yuan
}

export interface WeeklyData {
  week: string // ISO 周,如 "2026-W03"
  weekStart: string // 该周周一,如 "2026-01-12"
  income: Yuan
  expense: Yuan
  balance: Yuan
}

export interface TagStats {
  tag: string
  amount: Yuan
  count: number
  percent: number
}

export interface PayeeStats {
  payee: string
  amount: Yuan
  count: number
}

export interface WeekdayCategoryCount {
  category: string
  count: number
  amount: Yuan
}

export interface WeekdayStats {
  weekday: number // 0=Sunday, 1=Monday, ...
  name: string
  amount: Yuan
  count: number
  dates: string[]
  categoryBreakdown: WeekdayCategoryCount[]
}

export interface MonthlyAmount {
  month: string
  amount: Yuan
}

export interface CategoryTrend {
  category: string
  data: MonthlyAmount[]
}

export interface LiabilityStats {
  account: string
  name: string
  balance: Yuan
  currency: string
}

export interface IncomeSource {
  source: string
  amount: Yuan
  percent: number
  count: number
}

export interface Analytics {
  summary: Summary
  parseIssues: ParseIssue[]
  balanceChecks: BalanceCheck[]
  expenseByCategory: CategoryAmount[]
  accountBalances: AccountBalance[]
  monthlyTrend: MonthlyData[]
  dailyTrend: DailyData[]
  weeklyTrend: WeeklyData[]
  transactions: Transaction[]
  dailyAverage: number
  platformRanking: TagStats[]
  merchantRanking: PayeeStats[]
  weekdayDistribution: WeekdayStats[]
  categoryTrends: CategoryTrend[]
  liabilityBreakdown: LiabilityStats[]
  incomeBreakdown: IncomeSource[]
}
