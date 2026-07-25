import type { InstallmentConfig, RevolvingConfig } from '../types/api'

// 保存前的本地校验:规则镜像 server/parser/debts.go 的 Validate(),只为在卡内即时反馈,
// 让用户不必靠一次 400 往返才知道哪里填错(后端只回 details[0],一次只能暴露一条)。
// 后端 Validate() 仍是唯一权威——保存照样过后端,这里放行不等于合法,不要据此省掉后端校验。

const MONTH_RE = /^\d{4}-(0[1-9]|1[0-2])$/
const DATE_RE = /^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$/

function isLiability(account: string): boolean {
  return account.startsWith('Liabilities:')
}

function inDayRange(day: number): boolean {
  return Number.isInteger(day) && day >= 1 && day <= 31
}

// 金额在前端是「元」浮点,先折成分再比,与后端定点口径一致,避免浮点噪声误报
function cents(yuan: number): number {
  return Math.round(yuan * 100)
}

export function validateRevolving(
  account: string,
  rc: RevolvingConfig,
  takenAccounts: string[],
): string[] {
  const errs: string[] = []
  if (!account) {
    errs.push('请填写账户名')
  } else if (!isLiability(account)) {
    errs.push('账户必须以 Liabilities: 开头')
  } else if (takenAccounts.includes(account)) {
    errs.push(`账户 ${account} 已配置过`)
  }
  if (!inDayRange(rc.billingDay)) errs.push('账单日需为 1-31 的整数')
  if (!inDayRange(rc.dueDay)) errs.push('还款日需为 1-31 的整数')

  rc.installments.forEach((ri, i) => {
    const label = ri.name || `第 ${i + 1} 条分期`
    if (!ri.name) errs.push(`${label}缺少名称`)
    if (!(ri.totalAmount > 0)) errs.push(`${label}的总金额必须大于 0`)
    if (!(ri.months >= 1)) errs.push(`${label}的期数必须大于 0`)
    if (!(ri.monthlyAmount > 0)) errs.push(`${label}的每期金额必须大于 0`)
    if (!MONTH_RE.test(ri.firstBillMonth)) errs.push(`${label}的首期账单月应为 YYYY-MM`)
    // 尾差不论落首期还是末期都在一期以内,超出说明期数/金额填错了
    if (ri.totalAmount > 0 && ri.months >= 1 && ri.monthlyAmount > 0) {
      const diff = Math.abs(cents(ri.totalAmount) - cents(ri.monthlyAmount) * ri.months)
      if (diff >= cents(ri.monthlyAmount)) {
        errs.push(`${label}的每期金额×期数与总金额不匹配(偏差超过一期)`)
      }
    }
  })
  return errs
}

export function validateInstallment(ic: InstallmentConfig): string[] {
  const errs: string[] = []
  if (!ic.name) errs.push('请填写名称')
  if (!ic.account) {
    errs.push('请填写关联账户')
  } else if (!isLiability(ic.account)) {
    errs.push('关联账户必须以 Liabilities: 开头')
  }
  if (!inDayRange(ic.dueDay)) errs.push('还款日需为 1-31 的整数')
  // 新建分期默认 schedule 为空,后端会直接拒;在这里挡住并说明要先追加一期月供
  if (!ic.schedule.length) errs.push('至少需要一条月供记录,请先在下方追加')

  ic.schedule.forEach((ph) => {
    if (!DATE_RE.test(ph.effectiveFrom)) errs.push(`生效日期 ${ph.effectiveFrom} 应为 YYYY-MM-DD`)
    if (!(ph.amount > 0)) errs.push('月供金额必须大于 0')
  })

  // 末期月留空 = 无终止期(房贷),合法
  if (ic.endMonth) {
    if (!MONTH_RE.test(ic.endMonth)) {
      errs.push('末期月应为 YYYY-MM')
    } else {
      // 月供在结清之后才生效 = 配置自相矛盾,这条月供永远用不上
      const latest = ic.schedule
        .filter((ph) => DATE_RE.test(ph.effectiveFrom))
        .reduce((acc, ph) => (ph.effectiveFrom > acc ? ph.effectiveFrom : acc), '')
      if (latest && latest.slice(0, 7) > ic.endMonth) {
        errs.push(`末期月 ${ic.endMonth} 早于最新月供生效月 ${latest.slice(0, 7)}`)
      }
    }
  }
  return errs
}
