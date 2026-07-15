import type { FunctionalComponent } from 'vue'
import { LayoutGrid, ArrowUpDown, TrendingUp, Wallet, ArrowRightLeft } from '@lucide/vue'

export interface NavItem {
  id: string
  label: string
  icon: FunctionalComponent
}

// 侧边栏与移动端底部导航共用的主导航项
export const navItems: NavItem[] = [
  { id: 'overview', label: '概览', icon: LayoutGrid },
  { id: 'spending', label: '收支', icon: ArrowUpDown },
  { id: 'trends', label: '趋势', icon: TrendingUp },
  { id: 'accounts', label: '账户', icon: Wallet },
  { id: 'transactions', label: '交易', icon: ArrowRightLeft },
]
