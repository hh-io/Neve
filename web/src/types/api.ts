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
  /** 长期负债(房贷等,对应资产不在账本内),清单见 DebtsConfig.longTermAccounts */
  longTermLiabilities: Yuan
  shortTermLiabilities: Yuan
  /** 资产 − 短期负债,概览默认口径 */
  netWorthExLongTerm: Yuan
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
  longTerm: boolean
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
  longTerm: boolean
}

export interface IncomeSource {
  source: string
  amount: Yuan
  percent: number
  count: number
}

// --- debts.go ---

export interface RevolvingConfig {
  name: string
  billingDay: number
  dueDay: number
  installments: RevolvingInstallment[]
}

/** 额度账户内嵌的分期账单(如信用卡 24 期免息),未出账部分从本期应还中扣减。 */
export interface RevolvingInstallment {
  name: string
  totalAmount: Yuan
  months: number
  monthlyAmount: Yuan
  firstBillMonth: string // "2025-11"
}

export interface InstallmentPhase {
  effectiveFrom: string // "2026-01-01"
  amount: Yuan
}

export interface InstallmentConfig {
  id: string
  name: string
  account: string
  dueDay: number
  /** 末期还款月 YYYY-MM;空串 = 无终止期(房贷) */
  endMonth: string
  schedule: InstallmentPhase[]
}

export interface DebtsConfig {
  /** 长期负债账户(房贷等):概览净资产不扣减这些账户 */
  longTermAccounts: string[]
  revolving: Record<string, RevolvingConfig>
  installments: InstallmentConfig[]
}

export interface DebtsSummary {
  /** 滚动窗口内要还的钱(现金流口径,取自 schedule,含账单与分期) */
  due30: Yuan
  due90: Yuan
  /** 当期账单口径:各卡自己的账单周期,不含未出账,与 due30/due90 不可互推 */
  monthDue: Yuan
  /** 本期应还里尚未还的部分;逾期的钱只在这里(schedule 剔除了已过期的) */
  monthRemaining: Yuan
  nextDueDate: string // 空串表示本期已全部结清
  nextDueName: string
  nextDueInDays: number // 负数 = 已逾期天数
  overdueCount: number
}

export interface RevolvingStatus {
  account: string
  name: string
  accountMissing: boolean
  statementDate: string
  dueDate: string
  statementDue: Yuan // 已扣减未出账分期后的口径
  paidSince: Yuan
  remaining: Yuan
  currentBalance: Yuan
  daysUntilDue: number
  overdue: boolean
  installmentUnbilled: Yuan
  installmentThisPeriod: Yuan
  installments: RevolvingInstallmentStatus[]
}

export interface RevolvingInstallmentStatus {
  name: string
  totalAmount: Yuan
  months: number
  monthlyAmount: Yuan
  firstBillMonth: string
  billedPeriods: number
  thisPeriodAmount: Yuan // 0 = 未开始或已出账完毕
  unbilledAmount: Yuan
  finished: boolean
}

export interface InstallmentStatus {
  id: string
  name: string
  account: string
  accountMissing: boolean
  monthlyAmount: Yuan // 0 表示本期无月供(schedule 全在未来,或已结清)
  dueDate: string
  paid: boolean
  paidAmount: Yuan
  daysUntilDue: number
  overdue: boolean
  currentBalance: Yuan
  /** 尚未还的期数(本期已还则不含本期);-1 = 无终止期,区别于已结清的 0 */
  remainingPeriods: number
  settled: boolean // 已过 endMonth,不再有月供
  endMonth: string
}

/** 一笔确定性还款在某个月的出账 */
export interface ScheduleEntry {
  account: string
  accountName: string
  name: string
  /**
   * statement = 已出账未还的账单(整笔覆盖该期内嵌分期,不重复计入);
   * unbilled = 已消费但未出账,下个账单日出账
   */
  source: 'revolving' | 'installment' | 'statement' | 'unbilled'
  amount: Yuan
  dueDate: string
  longTerm: boolean
}

/** 未来某个自然月的还款计划;cumulative 为自首月起的前缀和 */
export interface ScheduleMonth {
  month: string // "2026-08"
  total: Yuan
  cumulative: Yuan
  entries: ScheduleEntry[]
}

export interface DebtsReport {
  summary: DebtsSummary
  revolving: RevolvingStatus[]
  installments: InstallmentStatus[]
  unconfigured: LiabilityStats[]
  /** 未来 12 个月的确定性出账:已配置分期 + 已出账账单 + 已消费未出账;不预测未来新增消费 */
  schedule: ScheduleMonth[]
}

/** GET/POST /api/debts 的响应;账本尚未加载时 report 为 null。 */
export interface DebtsResponse {
  config: DebtsConfig
  report: DebtsReport | null
}

export interface Analytics {
  summary: Summary
  parseIssues: ParseIssue[]
  balanceChecks: BalanceCheck[]
  /** 本月口径(收支分析页整页同期) */
  expenseByCategory: CategoryAmount[]
  accountBalances: AccountBalance[]
  monthlyTrend: MonthlyData[]
  dailyTrend: DailyData[]
  weeklyTrend: WeeklyData[]
  /** 全量交易(不含 opening),含未来日期的交易 */
  transactions: Transaction[]
  dailyAverage: number
  /** 本月口径 */
  platformRanking: TagStats[]
  /** 本月口径 */
  merchantRanking: PayeeStats[]
  /** 全量口径(跨期视图) */
  weekdayDistribution: WeekdayStats[]
  /** 近 6 个月 */
  categoryTrends: CategoryTrend[]
  liabilityBreakdown: LiabilityStats[]
  /** 本月口径 */
  incomeBreakdown: IncomeSource[]
}
