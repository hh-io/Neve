<template>
  <div v-if="rows.length > 0" class="cb">
    <div v-for="row in rows" :key="row.key" class="cb-row">
      <div class="cb-line">
        <component :is="row.icon" :size="15" class="cb-ic" />
        <span class="cb-name">{{ row.name }}</span>
        <span class="cb-amt tabular-nums">{{ row.amount }}</span>
        <span class="cb-delta tabular-nums" :class="row.deltaCls">
          <component :is="row.deltaIcon" v-if="row.deltaIcon" :size="12" />
          {{ row.delta }}
        </span>
      </div>
      <div class="cb-track">
        <div class="cb-fill" :style="{ width: row.width, background: row.color }"></div>
        <!-- 预算刻度:与占比条共用同一条归一轴,填充越过刻度即超支,不额外占一行高度 -->
        <div v-if="row.budgetMark" class="cb-mark" :style="{ left: row.budgetMark }"></div>
      </div>
      <span class="cb-note" :class="row.noteCls">{{ row.note }}</span>
    </div>
  </div>
  <div v-else class="empty-state">本月暂无支出数据</div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import type { FunctionalComponent } from 'vue';
import { ArrowUpRight, ArrowDownRight, Shapes } from '@lucide/vue';
import type { CategoryAmount } from '../types/api';
import { formatMoney } from '../composables/useFormatters';
import { getCategoryLabel } from '../composables/useCategories';
import { getCategoryIcon } from '../composables/useCategoryIcon';
import { useBudgets } from '../composables/useBudgets';

// 概览页的分类视图刻意不是环形:构成比例月月相似,概览要回答的是「本月哪不对劲」。
// 榜单同一行里塞下金额、环比与预算执行,信息量高于环形的 7 类 + 长尾丢弃;
// 构成本身留给「收支分析」页的 ExpenseDonut,两页不再给同一个答案。
const props = withDefaults(defineProps<{ data?: CategoryAmount[]; maxRows?: number }>(), {
  data: () => [],
  maxRows: 8,
});

// budgets 是模块级单例(loaded 幂等),与预算 Tab 共用一次请求
const { budgets, loadBudgets } = useBudgets();
onMounted(loadBudgets);

const sorted = computed(() => [...props.data].sort((a, b) => b.amount - a.amount));
// 条宽按最大项归一而非按占比:头部一项占比高时,后面几类会几乎等长看不出主次
const maxAmount = computed(() => sorted.value.reduce((max, c) => Math.max(max, c.amount), 0) || 1);
const totalAmount = computed(() => sorted.value.reduce((sum, c) => sum + c.amount, 0));

interface Row {
  key: string;
  name: string;
  icon: FunctionalComponent;
  amount: string;
  delta: string;
  deltaCls: string;
  deltaIcon: FunctionalComponent | null;
  width: string;
  color: string;
  budgetMark: string;
  note: string;
  noteCls: string;
}

// 环比不足 5% 当持平:分类金额本就零散,每行都挂一个 ±2% 的箭头会把真正的异动淹掉
const FLAT_THRESHOLD = 5;

function widthOf(amount: number): string {
  // 净退款让分类金额为负时条归零(条画不出负长度),金额与环比仍如实显示
  return `${Math.max(0, Math.min(100, (amount / maxAmount.value) * 100))}%`;
}

function pctOf(amount: number): number {
  return totalAmount.value > 0 ? (amount / totalAmount.value) * 100 : 0;
}

const rows = computed<Row[]>(() => {
  const list = sorted.value;
  const head = list.slice(0, props.maxRows);
  const rest = list.slice(props.maxRows);

  const result: Row[] = head.map(item => {
    const limit = budgets.value[item.category];
    const used = limit > 0 ? (item.amount / limit) * 100 : 0;
    // 条色编码预算状态而非分类身份:分类名就在同一行,颜色再编码一遍分类是浪费。
    // 未配预算的分类保持中性 accent
    let color = 'var(--accent)';
    let note = `占比 ${pctOf(item.amount).toFixed(0)}% · ${item.count} 笔`;
    let noteCls = '';
    if (limit > 0) {
      if (used > 100) {
        color = 'var(--expense)';
        noteCls = 'cb-note--over';
        note = `超预算 ${formatMoney(item.amount - limit)} · 预算 ${formatMoney(limit)}`;
      } else {
        if (used >= 80) color = 'var(--warning)';
        note = `预算已用 ${used.toFixed(0)}% · 余 ${formatMoney(limit - item.amount)}`;
      }
    }

    return {
      key: item.category,
      name: getCategoryLabel(item.category),
      icon: getCategoryIcon(item.category),
      amount: formatMoney(item.amount),
      ...deltaOf(item.amount, item.prevAmount),
      width: widthOf(item.amount),
      color,
      // 预算落在轴外(预算远高于本月最大分类)时不画刻度:贴在最右端会被误读成「快超了」
      budgetMark: limit > 0 && limit < maxAmount.value ? `${(limit / maxAmount.value) * 100}%` : '',
      note,
      noteCls,
    };
  });

  if (rest.length > 0) {
    const restAmount = rest.reduce((sum, c) => sum + c.amount, 0);
    const restCount = rest.reduce((sum, c) => sum + c.count, 0);
    result.push({
      key: '__rest',
      name: `其他 ${rest.length} 类`,
      icon: Shapes,
      amount: formatMoney(restAmount),
      // 长尾是一批分类的合计,环比对不上任何单一分类,不给箭头
      delta: '',
      deltaCls: '',
      deltaIcon: null,
      width: widthOf(restAmount),
      color: 'var(--text-tertiary)',
      budgetMark: '',
      note: `占比 ${pctOf(restAmount).toFixed(0)}% · ${restCount} 笔`,
      noteCls: '',
    });
  }

  return result;
});

// 支出上升 = 不利,故涨用 expense 色、跌用 income 色(与概览统计卡的 chip 语义一致)
function deltaOf(amount: number, prev: number): Pick<Row, 'delta' | 'deltaCls' | 'deltaIcon'> {
  // 上月为 0(本月新增的分类)算不出百分比;上月净额为负(退款多于消费)同样没有可比基数
  if (prev <= 0) {
    // 「新增」走中性色:上月没有这一类算不出涨跌,染成 expense 红会把物业费这类
    // 首次出现的固定支出报成异常
    return {
      delta: amount > 0 ? '新增' : '—',
      deltaCls: 'cb-delta--flat',
      deltaIcon: null,
    };
  }
  const change = ((amount - prev) / prev) * 100;
  if (Math.abs(change) < FLAT_THRESHOLD) {
    return { delta: '持平', deltaCls: 'cb-delta--flat', deltaIcon: null };
  }
  const up = change > 0;
  return {
    delta: `${up ? '+' : ''}${change.toFixed(0)}%`,
    deltaCls: up ? 'cb-delta--up' : 'cb-delta--down',
    deltaIcon: up ? ArrowUpRight : ArrowDownRight,
  };
}
</script>

<style scoped>
.cb {
  padding: var(--space-4) var(--space-5) var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  /* 与并排的「最近交易」等高时吃掉剩余高度,行间距均分而非全堆在顶部 */
  flex: 1;
  min-height: 0;
  justify-content: center;
}

.cb-row {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.cb-line {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.cb-ic {
  color: var(--text-tertiary);
  flex: none;
}

.cb-name {
  flex: 1;
  min-width: 0;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cb-amt {
  color: var(--text-primary);
  font-weight: 600;
}

.cb-delta {
  display: inline-flex;
  align-items: center;
  gap: 1px;
  width: 58px;
  justify-content: flex-end;
  font-size: var(--font-size-xs);
  font-weight: 600;
  flex: none;
}

.cb-delta--up { color: var(--expense); }
.cb-delta--down { color: var(--income); }
.cb-delta--flat { color: var(--text-tertiary); }

.cb-track {
  position: relative;
  height: 5px;
  border-radius: var(--radius-full);
  background: var(--surface-3);
  overflow: hidden;
}

.cb-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width var(--transition-slow);
}

/* 超支时刻度落在填充段内,单一灰线会被条色吃掉:补一圈卡片底色描边,
   在填充上是缺口、在空轨上是实线,两种背景下都看得见 */
.cb-mark {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--text-tertiary);
  box-shadow: 0 0 0 1px var(--surface-1);
}

.cb-note {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.cb-note--over { color: var(--expense); }
</style>
