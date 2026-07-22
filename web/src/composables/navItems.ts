import type { FunctionalComponent } from 'vue'
import { LayoutDashboard, PieChart, TrendingUp, Wallet, CreditCard, ReceiptText } from '@lucide/vue'

export interface NavItem {
  id: string
  label: string
  // 移动端底部导航用的短名(四字 label 在窄底栏会挤,降到两字)
  short: string
  icon: FunctionalComponent
}

// 侧边栏与移动端底部导航共用的主导航项(图标对齐设计稿)
export const navItems: NavItem[] = [
  { id: 'overview', label: '财务概览', short: '概览', icon: LayoutDashboard },
  { id: 'spending', label: '收支分析', short: '收支', icon: PieChart },
  { id: 'trends', label: '趋势图表', short: '趋势', icon: TrendingUp },
  { id: 'accounts', label: '账户管理', short: '账户', icon: Wallet },
  { id: 'debts', label: '待还管理', short: '待还', icon: CreditCard },
  { id: 'transactions', label: '交易明细', short: '交易', icon: ReceiptText },
]
