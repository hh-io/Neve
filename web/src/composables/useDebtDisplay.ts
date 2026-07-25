// 待还卡片共用的展示派生:额度账单与固定分期两种卡的日期/倒计时口径必须一致

/** 后端日期恒为 YYYY-MM-DD,直接截字符串,不经 Date 以免带上时区偏移 */
export function shortDate(date: string): string {
  return date.slice(5).replace('-', '/')
}

/** 倒计时文案 + 颜色 */
export function countdownFor(
  overdue: boolean,
  settled: boolean,
  days: number,
): { text: string; color: string } {
  if (settled) return { text: '已还清', color: 'var(--income)' }
  if (overdue) return { text: `逾期 ${-days} 天`, color: 'var(--expense)' }
  if (days === 0) return { text: '今天到期', color: 'var(--warning)' }
  return { text: `还剩 ${days} 天`, color: days <= 3 ? 'var(--warning)' : 'var(--text-secondary)' }
}
