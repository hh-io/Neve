<template>
  <div class="animate-fade-in-up">
    <!-- Stats Grid -->
    <div class="stats-grid">
      <!-- Net Worth Card -->
      <div class="card stat-card">
        <div class="stat-header">
          <div class="stat-icon bg-brand-light">
            <Wallet color="var(--accent)" />
          </div>
          <span :class="['stat-badge', netWorth >= 0 ? 'bg-income-light text-income' : 'bg-expense-light text-expense']">
            {{ balanceChange }}
          </span>
        </div>
        <div class="eyebrow">净资产</div>
        <div class="stat-value stat-value-hero" :class="netWorth >= 0 ? 'text-income' : 'text-expense'">{{ formatMoney(netWorth) }}</div>
        <div class="stat-detail">
          <div class="stat-detail-col">
            <span class="stat-detail-label">资产</span>
            <span class="stat-detail-val text-income">{{ formatMoney(totalAssets) }}</span>
          </div>
          <div class="stat-detail-col align-right">
            <span class="stat-detail-label">负债</span>
            <span class="stat-detail-val text-expense">{{ formatMoney(totalLiabilities) }}</span>
          </div>
        </div>
      </div>

      <!-- Monthly Income -->
      <div class="card stat-card delay-100">
        <div class="stat-header">
          <div class="stat-icon bg-income-light">
            <ArrowUp color="var(--income)" />
          </div>
          <span :class="['stat-badge', incomeChange >= 0 ? 'bg-income-light text-income' : 'bg-expense-light text-expense']">
            {{ incomeChange >= 0 ? '+' : '' }}{{ incomeChange }}%
          </span>
        </div>
        <div class="eyebrow">本月收入</div>
        <div class="stat-value stat-value-hero text-income">{{ formatMoney(monthlyIncome) }}</div>
        <div class="stat-detail">
          <div class="stat-detail-col">
            <span class="stat-detail-label">收入笔数</span>
            <span class="stat-detail-val">{{ incomeCount }} 笔</span>
          </div>
          <div v-if="incomeBreakdown.length > 0" class="stat-detail-col align-right">
            <span class="stat-detail-label">主要来源</span>
            <span class="stat-detail-val text-income">{{ incomeBreakdown[0]?.source || '-' }}</span>
          </div>
        </div>
      </div>

      <!-- Monthly Expense -->
      <div class="card stat-card delay-200">
        <div class="stat-header">
          <div class="stat-icon bg-expense-light">
            <ArrowDown color="var(--expense)" />
          </div>
          <span :class="['stat-badge', expenseChange <= 0 ? 'bg-income-light text-income' : 'bg-expense-light text-expense']">
            {{ expenseChange >= 0 ? '+' : '' }}{{ expenseChange }}%
          </span>
        </div>
        <div class="eyebrow">本月支出</div>
        <div class="stat-value stat-value-hero text-expense">{{ formatMoney(monthlyExpense) }}</div>
        <div class="stat-detail">
          <div class="stat-detail-col">
            <span class="stat-detail-label">消费笔数</span>
            <span class="stat-detail-val">{{ expenseCount }} 笔</span>
          </div>
          <div v-if="topCategory" class="stat-detail-col align-right">
            <span class="stat-detail-label">主要类目</span>
            <span class="stat-detail-val text-expense">{{ topCategory.name }} {{ topCategory.percent }}%</span>
          </div>
        </div>
      </div>

      <!-- Savings Rate -->
      <div class="card stat-card delay-300">
        <div class="stat-header">
          <div class="stat-icon" :class="savingsRate >= 0 ? 'bg-income-light' : 'bg-expense-light'">
            <Target :color="savingsRate >= 0 ? 'var(--income)' : 'var(--expense)'" />
          </div>
          <span :class="['stat-badge', savingsRate >= 20 ? 'bg-income-light text-income' : savingsRate >= 0 ? 'bg-warning-light text-warning' : 'bg-expense-light text-expense']">
            {{ savingsRate }}%
          </span>
        </div>
        <div class="eyebrow">月结余</div>
        <div class="stat-value stat-value-hero" :class="monthlySavings >= 0 ? 'text-income' : 'text-expense'">
          {{ formatMoney(monthlySavings) }}
        </div>
        <div class="stat-detail">
          <div class="stat-detail-col">
            <span class="stat-detail-label">月末预测</span>
            <span class="stat-detail-val" :class="predictedSavings >= 0 ? 'text-income' : 'text-expense'">{{ formatMoney(predictedSavings) }}</span>
          </div>
          <div class="stat-detail-col align-right">
            <span class="stat-detail-label">日均结余</span>
            <span class="stat-detail-val" :class="dailySavings >= 0 ? 'text-income' : 'text-expense'">{{ formatMoney(dailySavings) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Stats Row -->
    <div class="grid-1-1 section-mb">
      <!-- Daily Average -->
      <div class="card-static panel">
        <div class="quick-head">
          <div class="quick-head-left">
            <div class="panel-icon bg-info-light">
              <Calendar :size="20" color="var(--info)" />
            </div>
            <div>
              <div class="quick-label">日均支出</div>
              <div class="quick-value">{{ formatMoney(dailyAverage) }}</div>
            </div>
          </div>
          <div class="align-right">
            <div class="quick-sub">本月天数</div>
            <div class="quick-value">{{ daysInMonth }} 天</div>
          </div>
        </div>
        <div class="progress-bar">
          <div class="progress-fill progress-info" :style="{ width: `${monthProgress}%` }"></div>
        </div>
        <div class="quick-footer">
          <span>本月进度</span>
          <span>{{ monthProgress }}%</span>
        </div>
      </div>

      <!-- Financial Health -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-brand-light">
              <PieChartIcon :size="20" color="var(--accent)" />
            </div>
            <div class="panel-title">财务健康指标</div>
          </div>
        </div>
        <div class="metric-grid">
          <div class="metric-box">
            <div class="metric-label">资产负债率</div>
            <div class="metric-value" :class="debtRatio > 50 ? 'text-expense' : 'text-income'">
              {{ debtRatio.toFixed(1) }}%
            </div>
          </div>
          <div class="metric-box">
            <div class="metric-label">月储蓄率</div>
            <div class="metric-value" :class="savingsRate > 20 ? 'text-income' : 'text-warning'">
              {{ savingsRate }}%
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Expense Category & Recent Transactions Row -->
    <div class="grid-2-1 section-mb">
      <!-- Expense Category Pie Chart -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-expense-light">
              <PieChartIcon :size="20" color="var(--expense)" />
            </div>
            <span class="panel-title">支出分类</span>
          </div>
          <span class="panel-sub">本月</span>
        </div>
        <div v-if="expenseByCategory.length > 0" class="pie-wrap">
          <v-chart :option="expensePieOption" autoresize />
        </div>
        <div v-else class="chart-empty">
          暂无支出数据
        </div>
      </div>

      <!-- Recent Transactions -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-brand-light">
              <ArrowRightLeft :size="20" color="var(--accent)" />
            </div>
            <span class="panel-title">最近交易</span>
          </div>
          <span class="panel-sub">共 {{ transactions.length }} 条</span>
        </div>
        <TransactionList
          :transactions="transactions"
          max-height="350px"
          :show-account="false"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { use } from 'echarts/core';
import { PieChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import VChart from 'vue-echarts';
import { formatMoney } from '../../composables/useFormatters';
import { getCategoryLabel } from '../../composables/useCategories';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { Wallet, ArrowUp, ArrowDown, Target, Calendar, ArrowRightLeft, PieChart as PieChartIcon } from '@lucide/vue';
import { useAnalytics } from '../../composables/useAnalytics';
import TransactionList from '../TransactionList.vue';

use([PieChart, TitleComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const { analytics } = useAnalytics();

// Computed values
const summary = computed(() => analytics.value?.summary);
const netWorth = computed(() => summary.value?.netWorth || 0);
const totalAssets = computed(() => summary.value?.totalAssets || 0);
const totalLiabilities = computed(() => Math.abs(summary.value?.totalLiabilities || 0));
const monthlyIncome = computed(() => summary.value?.monthIncome || 0);
const monthlyExpense = computed(() => Math.abs(summary.value?.monthExpense || 0));
const monthlySavings = computed(() => monthlyIncome.value - monthlyExpense.value);

const transactions = computed(() => analytics.value?.transactions || []);

// Get monthly trend data for calculating changes
const monthlyTrend = computed(() => analytics.value?.monthlyTrend || []);

// Calculate real month-over-month changes
const balanceChange = computed(() => {
  const trend = monthlyTrend.value;
  if (trend.length < 2) return '0%';
  const currentBalance = trend[trend.length - 1]?.balance || 0;
  const previousBalance = trend[trend.length - 2]?.balance || 0;
  if (previousBalance === 0) return currentBalance >= 0 ? '+0%' : '0%';
  const change = ((currentBalance - previousBalance) / Math.abs(previousBalance)) * 100;
  return change >= 0 ? `+${change.toFixed(1)}%` : `${change.toFixed(1)}%`;
});

const incomeChange = computed(() => {
  const trend = monthlyTrend.value;
  if (trend.length < 2) return 0;
  const currentIncome = trend[trend.length - 1]?.income || 0;
  const previousIncome = trend[trend.length - 2]?.income || 0;
  if (previousIncome === 0) return currentIncome > 0 ? 100 : 0;
  return Math.round(((currentIncome - previousIncome) / previousIncome) * 100);
});

const expenseChange = computed(() => {
  const trend = monthlyTrend.value;
  if (trend.length < 2) return 0;
  const currentExpense = Math.abs(trend[trend.length - 1]?.expense || 0);
  const previousExpense = Math.abs(trend[trend.length - 2]?.expense || 0);
  if (previousExpense === 0) return currentExpense > 0 ? 100 : 0;
  return Math.round(((currentExpense - previousExpense) / previousExpense) * 100);
});

const savingsRate = computed(() => {
  if (monthlyIncome.value === 0) return 0;
  return Math.round((monthlySavings.value / monthlyIncome.value) * 100);
});

const debtRatio = computed(() => {
  const liabilities = Math.abs(summary.value?.totalLiabilities || 0);
  const assets = summary.value?.totalAssets || 1;
  return (liabilities / assets) * 100;
});

// 日均支出由后端按统一口径计算
const dailyAverage = computed(() => analytics.value?.dailyAverage || 0);

const daysInMonth = computed(() => {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
});

const monthProgress = computed(() => {
  const now = new Date();
  return Math.round((now.getDate() / daysInMonth.value) * 100);
});

// Income breakdown data
const incomeBreakdown = computed(() => analytics.value?.incomeBreakdown || []);
const incomeCount = computed(() => {
  return incomeBreakdown.value.reduce((sum, item) => sum + (item.count || 0), 0) ||
    (monthlyIncome.value > 0 ? 1 : 0);
});

// Expense count and top category
const expenseByCategory = computed(() => analytics.value?.expenseByCategory || []);
const expenseCount = computed(() => {
  return expenseByCategory.value.reduce((sum, item) => sum + (item.count || 0), 0) ||
    (monthlyExpense.value > 0 ? 1 : 0);
});

const topCategory = computed(() => {
  if (expenseByCategory.value.length === 0) return null;
  const sorted = [...expenseByCategory.value].sort((a, b) => b.amount - a.amount);
  const top = sorted[0];
  const percent = monthlyExpense.value > 0 ? Math.round((top.amount / monthlyExpense.value) * 100) : 0;
  return { name: getCategoryLabel(top.category), percent };
});

// Predicted savings (projected to end of month based on current pace)
const predictedSavings = computed(() => {
  const daysElapsed = new Date().getDate();
  const dailyIncome = monthlyIncome.value / daysElapsed;
  const dailyExpense = monthlyExpense.value / daysElapsed;
  const projectedIncome = dailyIncome * daysInMonth.value;
  const projectedExpense = dailyExpense * daysInMonth.value;
  return projectedIncome - projectedExpense;
});

// Daily savings
const dailySavings = computed(() => {
  const daysElapsed = new Date().getDate();
  return monthlySavings.value / daysElapsed;
});

// Expense Pie Chart Option
const expensePieOption = computed(() => {
  // canvas 不解析 CSS 变量,颜色必须取实际值;依赖 themeVersion 以便主题切换时重算
  void themeVersion.value;
  const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'].map(getThemeColor);
  const data = expenseByCategory.value.map((item, index) => ({
    name: getCategoryLabel(item.category),
    value: item.amount,
    itemStyle: { color: palette[index % palette.length] }
  }));

  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: ¥{c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      textStyle: { fontSize: 11, color: getThemeColor('--text-secondary') }
    },
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['35%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 6, borderColor: getThemeColor('--surface-1'), borderWidth: 2 },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 'bold' }
      },
      data
    }]
  };
});
</script>

<style scoped>
/* 概览主数值:Revolut 式紧排大数字 */
.stat-value-hero {
  font-size: 1.875rem;
  font-weight: 500;
  line-height: 1.1;
  letter-spacing: -0.02em;
  margin-top: var(--space-1);
}

/* 卡片底部明细:发丝线分隔的两列 */
.stat-detail {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--hairline);
  font-size: var(--font-size-xs);
}

.stat-detail-col {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.align-right {
  text-align: right;
}

.stat-detail-label {
  color: var(--text-tertiary);
}

.stat-detail-val {
  color: var(--text-primary);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

/* Quick stats(日均支出) */
.quick-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}

.quick-head-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.quick-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.quick-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.quick-value {
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.progress-info {
  background-color: var(--info);
}

.quick-footer {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
}

/* 财务健康指标 */
.metric-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.metric-box {
  padding: var(--space-3);
  background: var(--surface-2);
  border-radius: var(--radius-md);
}

.metric-label {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.metric-value {
  font-size: var(--font-size-lg);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

/* 图表容器 */
.pie-wrap {
  height: 220px;
}

.chart-empty {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}
</style>
