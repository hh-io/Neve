<template>
  <div class="animate-fade-in-up ov">
    <!-- 4 统计大数卡:小图标 + 标签 / 大数 / chip + 环比 -->
    <div class="ov-stats">
      <div v-for="s in statCards" :key="s.key" class="card ov-stat">
        <div class="ov-stat-head">
          <component :is="s.icon" :size="16" class="ov-stat-ic" />
          <span class="ov-stat-label">{{ s.label }}</span>
        </div>
        <div class="ov-stat-value tabular-nums" :style="{ color: s.valueColor }">{{ s.value }}</div>
        <div class="ov-stat-foot">
          <span class="chip" :class="s.chipCls">
            <component :is="s.trendIcon" :size="12" />
            <span class="tabular-nums">{{ s.delta }}</span>
          </span>
          <span class="ov-stat-hint">{{ s.hint }}</span>
        </div>
      </div>
    </div>

    <!-- 日均支出 + 资产负债率 + 储蓄率 -->
    <div class="ov-row2">
      <div class="card ov-daily">
        <div class="ov-daily-head">
          <span class="ov-mini-label">日均支出</span>
          <span class="ov-mini-sub">本月已过 {{ dayOfMonth }} / {{ daysInMonth }} 天</span>
        </div>
        <div class="ov-daily-value tabular-nums">{{ formatMoney(dailyAverage) }}</div>
        <div class="ov-daily-foot">
          <div class="progress-bar ov-daily-bar">
            <div class="progress-fill" :style="{ width: `${monthProgress}%`, background: 'var(--accent)' }"></div>
          </div>
          <div class="ov-daily-cap">
            <span>本月累计 <span class="tabular-nums ov-cap-strong">{{ formatMoney(monthlyExpense) }}</span></span>
            <span>预计月末 <span class="tabular-nums ov-cap-strong">{{ formatMoney(projectedExpense) }}</span></span>
          </div>
        </div>
      </div>

      <div class="card ov-health">
        <span class="ov-mini-label">资产负债率</span>
        <div class="ov-health-value tabular-nums">{{ debtRatio.toFixed(1) }}%</div>
        <div class="progress-bar ov-health-bar">
          <div class="progress-fill" :style="{ width: `${Math.min(100, debtRatio)}%`, background: debtRatio > 50 ? 'var(--expense)' : 'var(--income)' }"></div>
        </div>
        <span class="ov-health-cap" :style="{ color: debtRatio > 50 ? 'var(--expense)' : 'var(--income)' }">{{ debtRatioCaption }}</span>
      </div>

      <div class="card ov-health">
        <span class="ov-mini-label">月储蓄率</span>
        <div class="ov-health-value tabular-nums">{{ savingsRate }}%</div>
        <div class="progress-bar ov-health-bar">
          <div class="progress-fill" :style="{ width: `${Math.min(100, Math.max(0, savingsRate))}%`, background: savingsRate >= 20 ? 'var(--accent)' : 'var(--warning)' }"></div>
        </div>
        <span class="ov-health-cap">{{ savingsCaption }}</span>
      </div>
    </div>

    <!-- 支出分类环形图 + 最近交易 -->
    <div class="ov-row3">
      <section class="card ov-panel">
        <div class="ov-panel-head">
          <h3>支出分类</h3>
          <span class="ov-panel-sub">本月</span>
        </div>
        <div v-if="expenseByCategory.length > 0" class="ov-donut-body">
          <div class="ov-donut-chart">
            <v-chart :option="expensePieOption" autoresize />
          </div>
          <div class="ov-legend">
            <div v-for="item in legendItems" :key="item.category" class="ov-legend-row">
              <span class="ov-legend-dot" :style="{ background: item.color }"></span>
              <span class="ov-legend-name">{{ item.name }}</span>
              <span class="ov-legend-amt tabular-nums">{{ item.amount }}</span>
              <span class="ov-legend-pct tabular-nums">{{ item.pct }}</span>
            </div>
          </div>
        </div>
        <div v-else class="chart-empty">暂无支出数据</div>
      </section>

      <section class="card ov-panel">
        <div class="ov-panel-head">
          <h3>最近交易</h3>
          <span class="ov-panel-sub">共 {{ transactions.length }} 条</span>
        </div>
        <TransactionList
          :transactions="transactions"
          max-height="360px"
          :show-account="false"
        />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { use } from 'echarts/core';
import { PieChart } from 'echarts/charts';
import { TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import VChart from 'vue-echarts';
import {
  Landmark, ArrowDownToLine, ArrowUpFromLine, PiggyBank,
  ArrowUpRight, ArrowDownRight,
} from '@lucide/vue';
import { formatMoney } from '../../composables/useFormatters';
import { getCategoryLabel } from '../../composables/useCategories';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { useAnalytics } from '../../composables/useAnalytics';
import TransactionList from '../TransactionList.vue';

use([PieChart, TooltipComponent, CanvasRenderer]);

const { analytics } = useAnalytics();

const summary = computed(() => analytics.value?.summary);
const netWorth = computed(() => summary.value?.netWorth || 0);
const totalLiabilities = computed(() => Math.abs(summary.value?.totalLiabilities || 0));
const monthlyIncome = computed(() => summary.value?.monthIncome || 0);
const monthlyExpense = computed(() => Math.abs(summary.value?.monthExpense || 0));
const monthlySavings = computed(() => monthlyIncome.value - monthlyExpense.value);
const transactions = computed(() => analytics.value?.transactions || []);
const monthlyTrend = computed(() => analytics.value?.monthlyTrend || []);
const expenseByCategory = computed(() => analytics.value?.expenseByCategory || []);
const dailyAverage = computed(() => analytics.value?.dailyAverage || 0);

// 环比变化(基于月度趋势)
const balanceChange = computed(() => {
  const t = monthlyTrend.value;
  if (t.length < 2) return 0;
  const cur = t[t.length - 1]?.balance || 0;
  const prev = t[t.length - 2]?.balance || 0;
  if (prev === 0) return 0;
  return ((cur - prev) / Math.abs(prev)) * 100;
});
const incomeChange = computed(() => {
  const t = monthlyTrend.value;
  if (t.length < 2) return 0;
  const cur = t[t.length - 1]?.income || 0;
  const prev = t[t.length - 2]?.income || 0;
  if (prev === 0) return cur > 0 ? 100 : 0;
  return ((cur - prev) / prev) * 100;
});
const expenseChange = computed(() => {
  const t = monthlyTrend.value;
  if (t.length < 2) return 0;
  const cur = Math.abs(t[t.length - 1]?.expense || 0);
  const prev = Math.abs(t[t.length - 2]?.expense || 0);
  if (prev === 0) return cur > 0 ? 100 : 0;
  return ((cur - prev) / prev) * 100;
});
const savingsRate = computed(() => {
  if (monthlyIncome.value === 0) return 0;
  return Math.round((monthlySavings.value / monthlyIncome.value) * 100);
});
const debtRatio = computed(() => {
  const assets = summary.value?.totalAssets || 1;
  return (totalLiabilities.value / assets) * 100;
});

const daysInMonth = computed(() => {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
});
const dayOfMonth = computed(() => new Date().getDate());
const monthProgress = computed(() => Math.round((dayOfMonth.value / daysInMonth.value) * 100));
const projectedExpense = computed(() => (monthlyExpense.value / dayOfMonth.value) * daysInMonth.value);

const debtRatioCaption = computed(() => {
  const r = debtRatio.value;
  if (r > 50) return '偏高 · 负债压力较大';
  if (r > 20) return '适中 · 负债可控';
  return '健康 · 负债占比很低';
});
const savingsCaption = computed(() => {
  const s = savingsRate.value;
  if (s >= 30) return '高于目标 30% · 状态良好';
  if (s >= 0) return '低于目标 30% · 可再收紧';
  return '入不敷出 · 需注意';
});

// 环比 chip:pos 表示数值方向,good 表示对财务是否有利
function pct(n: number): string {
  return `${n >= 0 ? '+' : ''}${n.toFixed(1)}%`;
}
const statCards = computed(() => {
  const nwUp = balanceChange.value >= 0;
  const incUp = incomeChange.value >= 0;
  const expUp = expenseChange.value >= 0; // 支出上升=不利
  return [
    {
      key: 'net', label: '净资产', icon: Landmark,
      value: formatMoney(netWorth.value),
      valueColor: netWorth.value < 0 ? 'var(--expense)' : 'var(--text-primary)',
      delta: pct(balanceChange.value), trendIcon: nwUp ? ArrowUpRight : ArrowDownRight,
      chipCls: nwUp ? 'chip-income' : 'chip-expense', hint: '环比上月',
    },
    {
      key: 'income', label: '本月收入', icon: ArrowDownToLine,
      value: formatMoney(monthlyIncome.value), valueColor: 'var(--income)',
      delta: pct(incomeChange.value), trendIcon: incUp ? ArrowUpRight : ArrowDownRight,
      chipCls: incUp ? 'chip-income' : 'chip-expense', hint: '环比上月',
    },
    {
      key: 'expense', label: '本月支出', icon: ArrowUpFromLine,
      value: formatMoney(monthlyExpense.value), valueColor: 'var(--expense)',
      delta: pct(expenseChange.value), trendIcon: expUp ? ArrowUpRight : ArrowDownRight,
      chipCls: expUp ? 'chip-expense' : 'chip-income', hint: '环比上月',
    },
    {
      key: 'savings', label: '月结余', icon: PiggyBank,
      value: formatMoney(monthlySavings.value),
      valueColor: monthlySavings.value < 0 ? 'var(--expense)' : 'var(--text-primary)',
      delta: `${savingsRate.value}%`, trendIcon: savingsRate.value >= 0 ? ArrowUpRight : ArrowDownRight,
      chipCls: savingsRate.value >= 20 ? 'chip-income' : savingsRate.value >= 0 ? 'chip-warning' : 'chip-expense',
      hint: '储蓄率',
    },
  ];
});

// 自定义图例(替代 echarts 内建图例,避免中文截断)
const legendItems = computed(() => {
  void themeVersion.value;
  const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'].map(getThemeColor);
  const total = expenseByCategory.value.reduce((sum, c) => sum + c.amount, 0) || 1;
  return [...expenseByCategory.value]
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 7)
    .map((item, index) => ({
      category: item.category,
      name: getCategoryLabel(item.category),
      color: palette[index % palette.length],
      amount: formatMoney(item.amount),
      pct: `${Math.round((item.amount / total) * 100)}%`,
    }));
});

const expensePieOption = computed(() => {
  void themeVersion.value;
  const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'].map(getThemeColor);
  const data = [...expenseByCategory.value]
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 7)
    .map((item, index) => ({
      name: getCategoryLabel(item.category),
      value: item.amount,
      itemStyle: { color: palette[index % palette.length] },
    }));

  return {
    tooltip: { trigger: 'item', formatter: '{b}: ¥{c} ({d}%)' },
    series: [{
      type: 'pie',
      radius: ['58%', '82%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 4, borderColor: getThemeColor('--surface-1'), borderWidth: 2 },
      label: { show: false },
      labelLine: { show: false },
      emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold', color: getThemeColor('--text-primary') } },
      data,
    }],
  };
});
</script>

<style scoped>
.ov {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

/* ===== 统计大数卡 ===== */
.ov-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
}

.ov-stat {
  padding: var(--space-5);
}

.ov-stat-head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
}

.ov-stat-ic {
  color: var(--text-tertiary);
}

.ov-stat-value {
  margin-top: var(--space-3);
  font-size: 1.75rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.ov-stat-foot {
  margin-top: var(--space-3);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.ov-stat-hint {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

/* ===== chip ===== */
.chip {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: 650;
}

.chip-income { background: var(--income-light); color: var(--income); }
.chip-expense { background: var(--expense-light); color: var(--expense); }
.chip-warning { background: var(--warning-light); color: var(--warning); }

/* ===== 日均 + 健康 行 ===== */
.ov-row2 {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr;
  gap: var(--space-4);
}

.ov-mini-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.ov-mini-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ov-daily {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
}

.ov-daily-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.ov-daily-value {
  margin-top: var(--space-2);
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--expense);
}

.ov-daily-foot {
  margin-top: auto;
  padding-top: var(--space-4);
}

.ov-daily-bar { height: 8px; }

.ov-daily-cap {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ov-cap-strong { color: var(--text-secondary); }

.ov-health {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.ov-health-value {
  font-size: 1.5rem;
  font-weight: 700;
}

.ov-health-bar { height: 6px; }

.ov-health-cap {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

/* ===== 环形图 + 最近交易 ===== */
.ov-row3 {
  display: grid;
  grid-template-columns: 1.1fr 1.25fr;
  gap: var(--space-4);
}

.ov-panel {
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.ov-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--hairline);
}

.ov-panel-head h3 {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: 620;
}

.ov-panel-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ov-donut-body {
  padding: var(--space-4) var(--space-5) var(--space-5);
  display: flex;
  gap: var(--space-5);
  align-items: center;
  flex-wrap: wrap;
}

.ov-donut-chart {
  width: 150px;
  height: 150px;
  flex: none;
  margin: 0 auto;
}

.ov-legend {
  flex: 1;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.ov-legend-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.ov-legend-dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
  flex: none;
}

.ov-legend-name {
  flex: 1;
  min-width: 0;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ov-legend-amt { color: var(--text-primary); }

.ov-legend-pct {
  width: 42px;
  text-align: right;
  color: var(--text-tertiary);
}

.chart-empty {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

@media (max-width: 1024px) {
  .ov-stats { grid-template-columns: repeat(2, 1fr); }
  .ov-row2 { grid-template-columns: 1fr; }
  .ov-row3 { grid-template-columns: 1fr; }
}

@media (max-width: 640px) {
  .ov-stats { grid-template-columns: 1fr; }
}
</style>
