import type { FunctionalComponent } from 'vue'
import {
  Utensils,
  ShoppingBag,
  Car,
  Clapperboard,
  Gift,
  Landmark,
  Phone,
  Home,
  Smartphone,
  Heart,
  GraduationCap,
  Receipt,
  Zap,
  Banknote,
  Briefcase,
  Award,
  TrendingUp,
  Repeat,
  Users,
  Shapes,
} from '@lucide/vue'

// 分类 → lucide 图标的唯一映射(键对齐 useCategories.ts 的 categoryLabels)
const categoryIcons: Record<string, FunctionalComponent> = {
  Food: Utensils,
  Shopping: ShoppingBag,
  Transport: Car,
  Entertainment: Clapperboard,
  Gift: Gift,
  Financial: Landmark,
  Communication: Phone,
  Lodging: Home,
  Digital: Smartphone,
  Health: Heart,
  Education: GraduationCap,
  Fixed: Receipt,
  Utilities: Zap,
  Housing: Home,
  Unknown: Shapes,
  Other: Shapes,
  Income: Banknote,
  Salary: Briefcase,
  Bonus: Award,
  Membership: Users,
  Dividend: TrendingUp,
  Investment: TrendingUp,
  SecondHand: Repeat,
  Family: Users,
}

export function getCategoryIcon(category: string | undefined): FunctionalComponent {
  return (category && categoryIcons[category]) || Shapes
}
