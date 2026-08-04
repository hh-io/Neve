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
      </div>
      <span class="cb-note">{{ row.note }}</span>
    </div>
  </div>
  <div v-else class="empty-state">本月暂无支出数据</div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { FunctionalComponent } from 'vue';
import { ArrowUpRight, ArrowDownRight, Shapes } from '@lucide/vue';
import type { CategoryAmount } from '../types/api';
import { formatMoney } from '../composables/useFormatters';
import { getCategoryLabel } from '../composables/useCategories';
import { getCategoryIcon } from '../composables/useCategoryIcon';

// 概览页的分类视图刻意不是环形:构成比例月月相似,概览要回答的是「本月哪不对劲」。
// 每行给金额 + 环比 + 上月基数,信息量高于环形的 7 类 + 长尾丢弃;
// 构成本身留给「收支分析」页的 ExpenseDonut,两页不再给同一个答案。
// 第二行刻意不写「占比 X%」:那正是环形图例给的信息,写了等于又绕回重复。
const props = withDefaults(defineProps<{ data?: CategoryAmount[]; maxRows?: number }>(), {
  data: () => [],
  maxRows: 8,
});

const sorted = computed(() => [...props.data].sort((a, b) => b.amount - a.amount));
// 条宽按最大项归一而非按占比:头部一项占比高时,后面几类会几乎等长看不出主次
const maxAmount = computed(() => sorted.value.reduce((max, c) => Math.max(max, c.amount), 0) || 1);

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
  note: string;
}

// 环比不足 5% 当持平:分类金额本就零散,每行都挂一个 ±2% 的箭头会把真正的异动淹掉
const FLAT_THRESHOLD = 5;

function widthOf(amount: number): string {
  // 净退款让分类金额为负时条归零(条画不出负长度),金额与环比仍如实显示
  return `${Math.max(0, Math.min(100, (amount / maxAmount.value) * 100))}%`;
}

// 第二行给环比的基数:箭头只说了变了多少,「上月 ¥X」才让人判断这个百分比值不值得紧张
function noteOf(prev: number, count: number): string {
  const base = prev > 0 ? `上月 ${formatMoney(prev)}` : '上月无支出';
  return `${base} · ${count} 笔`;
}

const rows = computed<Row[]>(() => {
  const list = sorted.value;
  const head = list.slice(0, props.maxRows);
  const rest = list.slice(props.maxRows);

  const result: Row[] = head.map(item => ({
    key: item.category,
    name: getCategoryLabel(item.category),
    icon: getCategoryIcon(item.category),
    amount: formatMoney(item.amount),
    ...deltaOf(item.amount, item.prevAmount),
    width: widthOf(item.amount),
    color: 'var(--accent)',
    note: noteOf(item.prevAmount, item.count),
  }));

  if (rest.length > 0) {
    const restAmount = rest.reduce((sum, c) => sum + c.amount, 0);
    const restPrev = rest.reduce((sum, c) => sum + c.prevAmount, 0);
    const restCount = rest.reduce((sum, c) => sum + c.count, 0);
    result.push({
      key: '__rest',
      name: `其他 ${rest.length} 类`,
      icon: Shapes,
      amount: formatMoney(restAmount),
      // 长尾是「本月排名靠后的那批分类」,它们上月未必也在长尾里,合计的涨跌
      // 不对应任何一个可追问的对象,故只给基数不给箭头
      delta: '',
      deltaCls: '',
      deltaIcon: null,
      width: widthOf(restAmount),
      color: 'var(--text-tertiary)',
      note: noteOf(restPrev, restCount),
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

.cb-note {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}
</style>
