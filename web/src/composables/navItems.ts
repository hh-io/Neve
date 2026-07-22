import type { FunctionalComponent } from 'vue'
import { LayoutDashboard, PieChart, TrendingUp, Wallet, CreditCard, ReceiptText } from '@lucide/vue'

export interface NavItem {
  id: string
  label: string
  icon: FunctionalComponent
}

// 侧边栏与移动端底部导航共用的主导航项(图标对齐设计稿)
export const navItems: NavItem[] = [
  { id: 'overview', label: '概览', icon: LayoutDashboard },
  { id: 'spending', label: '收支', icon: PieChart },
  { id: 'trends', label: '趋势', icon: TrendingUp },
  { id: 'accounts', label: '账户', icon: Wallet },
  { id: 'debts', label: '待还', icon: CreditCard },
  { id: 'transactions', label: '交易', icon: ReceiptText },
]
