<template>
  <div class="animate-fade-in-up">
    <!-- Stats Grid -->
    <div class="stats-grid">
      <!-- Net Worth Card -->
      <div class="card stat-card">
        <div class="stat-header">
          <div class="stat-icon bg-brand-light">
            <span v-html="icons.wallet" style="stroke: var(--brand-primary);"></span>
          </div>
          <span :class="['stat-badge', netWorth >= 0 ? 'bg-income-light text-income' : 'bg-expense-light text-expense']">
            {{ balanceChange }}
          </span>
        </div>
        <div class="stat-label">净资产</div>
        <div class="stat-value" :style="{ color: netWorth >= 0 ? 'var(--income)' : 'var(--expense)' }">{{ formatMoney(netWorth) }}</div>
        <div style="display: flex; justify-content: space-between; margin-top: var(--space-3); padding-top: var(--space-3); border-top: 1px solid var(--border); font-size: var(--font-size-xs);">
          <div style="display: flex; flex-direction: column; gap: 2px;">
            <span style="color: var(--text-tertiary);">资产</span>
            <span style="color: var(--income); font-weight: 500;">{{ formatMoney(totalAssets) }}</span>
          </div>
          <div style="display: flex; flex-direction: column; gap: 2px; text-align: right;">
            <span style="color: var(--text-tertiary);">负债</span>
            <span style="color: var(--expense); font-weight: 500;">{{ formatMoney(totalLiabilities) }}</span>
          </div>
        </div>
      </div>

      <!-- Monthly Income -->
      <div class="card stat-card delay-100">
        <div class="stat-header">
          <div class="stat-icon bg-income-light">
            <span v-html="icons.arrowUp" style="stroke: var(--income);"></span>
          </div>
          <span :class="['stat-badge', incomeChange >= 0 ? 'bg-income-light text-income' : 'bg-expense-light text-expense']">
            {{ incomeChange >= 0 ? '+' : '' }}{{ incomeChange }}%
          </span>
        </div>
        <div class="stat-label">本月收入</div>
        <div class="stat-value text-income">{{ formatMoney(monthlyIncome) }}</div>
        <div style="display: flex; justify-content: space-between; margin-top: var(--space-3); padding-top: var(--space-3); border-top: 1px solid var(--border); font-size: var(--font-size-xs);">
          <div style="display: flex; flex-direction: column; gap: 2px;">
            <span style="color: var(--text-tertiary);">收入笔数</span>
            <span style="color: var(--text-primary); font-weight: 500;">{{ incomeCount }} 笔</span>
          </div>
          <div v-if="incomeBreakdown.length > 0" style="display: flex; flex-direction: column; gap: 2px; text-align: right;">
            <span style="color: var(--text-tertiary);">主要来源</span>
            <span style="color: var(--income); font-weight: 500;">{{ incomeBreakdown[0]?.source || '-' }}</span>
          </div>
        </div>
      </div>

      <!-- Monthly Expense -->
      <div class="card stat-card delay-200">
        <div class="stat-header">
          <div class="stat-icon bg-expense-light">
            <span v-html="icons.arrowDown" style="stroke: var(--expense);"></span>
          </div>
          <span :class="['stat-badge', expenseChange <= 0 ? 'bg-income-light text-income' : 'bg-expense-light text-expense']">
            {{ expenseChange >= 0 ? '+' : '' }}{{ expenseChange }}%
          </span>
        </div>
        <div class="stat-label">本月支出</div>
        <div class="stat-value text-expense">{{ formatMoney(monthlyExpense) }}</div>
        <div style="display: flex; justify-content: space-between; margin-top: var(--space-3); padding-top: var(--space-3); border-top: 1px solid var(--border); font-size: var(--font-size-xs);">
          <div style="display: flex; flex-direction: column; gap: 2px;">
            <span style="color: var(--text-tertiary);">消费笔数</span>
            <span style="color: var(--text-primary); font-weight: 500;">{{ expenseCount }} 笔</span>
          </div>
          <div v-if="topCategory" style="display: flex; flex-direction: column; gap: 2px; text-align: right;">
            <span style="color: var(--text-tertiary);">主要类目</span>
            <span style="color: var(--expense); font-weight: 500;">{{ topCategory.name }} {{ topCategory.percent }}%</span>
          </div>
        </div>
      </div>

      <!-- Savings Rate -->
      <div class="card stat-card delay-300">
        <div class="stat-header">
          <div class="stat-icon" :class="savingsRate >= 0 ? 'bg-income-light' : 'bg-expense-light'">
            <span v-html="icons.target" :style="{ stroke: savingsRate >= 0 ? 'var(--income)' : 'var(--expense)' }"></span>
          </div>
          <span :class="['stat-badge', savingsRate >= 20 ? 'bg-income-light text-income' : savingsRate >= 0 ? 'bg-warning-light text-warning' : 'bg-expense-light text-expense']">
            {{ savingsRate }}%
          </span>
        </div>
        <div class="stat-label">月结余</div>
        <div class="stat-value" :style="{ color: monthlySavings >= 0 ? 'var(--income)' : 'var(--expense)' }">
          {{ formatMoney(monthlySavings) }}
        </div>
        <div style="display: flex; justify-content: space-between; margin-top: var(--space-3); padding-top: var(--space-3); border-top: 1px solid var(--border); font-size: var(--font-size-xs);">
          <div style="display: flex; flex-direction: column; gap: 2px;">
            <span style="color: var(--text-tertiary);">月末预测</span>
            <span :style="{ color: predictedSavings >= 0 ? 'var(--income)' : 'var(--expense)', fontWeight: 500 }">{{ formatMoney(predictedSavings) }}</span>
          </div>
          <div style="display: flex; flex-direction: column; gap: 2px; text-align: right;">
            <span style="color: var(--text-tertiary);">日均结余</span>
            <span :style="{ color: dailySavings >= 0 ? 'var(--income)' : 'var(--expense)', fontWeight: 500 }">{{ formatMoney(dailySavings) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Quick Stats Row -->
    <div class="grid-1-1 section-mb">
      <!-- Daily Average -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
          <div style="display: flex; align-items: center; gap: var(--space-3);">
            <div class="stat-icon bg-info-light" style="width: 40px; height: 40px;">
              <span v-html="icons.calendar" style="stroke: var(--info); width: 20px; height: 20px;"></span>
            </div>
            <div>
              <div style="font-size: var(--font-size-sm); color: var(--text-secondary);">日均支出</div>
              <div style="font-size: var(--font-size-lg); font-weight: 600; color: var(--text-primary);">{{ formatMoney(dailyAverage) }}</div>
            </div>
          </div>
          <div style="text-align: right;">
            <div style="font-size: var(--font-size-xs); color: var(--text-tertiary);">本月天数</div>
            <div style="font-size: var(--font-size-lg); font-weight: 600; color: var(--text-primary);">{{ daysInMonth }} 天</div>
          </div>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: `${monthProgress}%`, backgroundColor: 'var(--info)' }"></div>
        </div>
        <div style="display: flex; justify-content: space-between; margin-top: var(--space-2); font-size: var(--font-size-xs); color: var(--text-tertiary);">
          <span>本月进度</span>
          <span>{{ monthProgress }}%</span>
        </div>
      </div>

      <!-- Financial Health -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.pieChart" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <div style="font-size: var(--font-size-sm); font-weight: 500; color: var(--text-primary);">财务健康指标</div>
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-4);">
          <div style="padding: var(--space-3); background: var(--bg-tertiary); border-radius: var(--radius-md);">
            <div style="font-size: var(--font-size-xs); color: var(--text-tertiary);">资产负债率</div>
            <div style="font-size: var(--font-size-lg); font-weight: 600;" :style="{ color: debtRatio > 50 ? 'var(--expense)' : 'var(--income)' }">
              {{ debtRatio.toFixed(1) }}%
            </div>
          </div>
          <div style="padding: var(--space-3); background: var(--bg-tertiary); border-radius: var(--radius-md);">
            <div style="font-size: var(--font-size-xs); color: var(--text-tertiary);">月储蓄率</div>
            <div style="font-size: var(--font-size-lg); font-weight: 600;" :style="{ color: savingsRate > 20 ? 'var(--income)' : 'var(--warning)' }">
              {{ savingsRate }}%
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Expense Category & Recent Transactions Row -->
    <div class="grid-2-1 section-mb">
      <!-- Expense Category Pie Chart -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
          <div style="display: flex; align-items: center; gap: var(--space-3);">
            <div class="stat-icon bg-expense-light" style="width: 40px; height: 40px;">
              <span v-html="icons.pieChart" style="stroke: var(--expense); width: 20px; height: 20px;"></span>
            </div>
            <span style="font-weight: 600; color: var(--text-primary);">支出分类</span>
          </div>
          <span style="font-size: var(--font-size-sm); color: var(--text-secondary);">本月</span>
        </div>
        <div v-if="expenseByCategory.length > 0" style="height: 220px;">
          <v-chart :option="expensePieOption" autoresize />
        </div>
        <div v-else style="height: 200px; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无支出数据
        </div>
      </div>

      <!-- Recent Transactions -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
          <div style="display: flex; align-items: center; gap: var(--space-3);">
            <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
              <span v-html="icons.transactions" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
            </div>
            <span style="font-weight: 600; color: var(--text-primary);">最近交易</span>
          </div>
          <span style="font-size: var(--font-size-xs); color: var(--text-tertiary);">
            共 {{ (analytics.transactions || []).length }} 条
          </span>
        </div>
        <TransactionList
          :transactions="analytics.transactions || []"
          max-height="350px"
          :show-account="false"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { use } from 'echarts/core';
import { PieChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import VChart from 'vue-echarts';
import { formatMoney } from '../../composables/useFormatters';
import { getCategoryLabel } from '../../composables/useCategories';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { icons } from '../../composables/icons';
import TransactionList from '../TransactionList.vue';

use([PieChart, TitleComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const props = defineProps({
  analytics: { type: Object, required: true }
});

// Computed values
const netWorth = computed(() => props.analytics.summary?.netWorth || 0);
const totalAssets = computed(() => props.analytics.summary?.totalAssets || 0);
const totalLiabilities = computed(() => Math.abs(props.analytics.summary?.totalLiabilities || 0));
const monthlyIncome = computed(() => props.analytics.summary?.monthIncome || 0);
const monthlyExpense = computed(() => Math.abs(props.analytics.summary?.monthExpense || 0));
const monthlySavings = computed(() => monthlyIncome.value - monthlyExpense.value);

// Get monthly trend data for calculating changes
const monthlyTrend = computed(() => props.analytics.monthlyTrend || []);

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
  const liabilities = Math.abs(props.analytics.summary?.totalLiabilities || 0);
  const assets = props.analytics.summary?.totalAssets || 1;
  return (liabilities / assets) * 100;
});

// 日均支出由后端按统一口径计算
const dailyAverage = computed(() => props.analytics.dailyAverage || 0);

const daysInMonth = computed(() => {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
});

const monthProgress = computed(() => {
  const now = new Date();
  return Math.round((now.getDate() / daysInMonth.value) * 100);
});

// Income breakdown data
const incomeBreakdown = computed(() => props.analytics.incomeBreakdown || []);
const incomeCount = computed(() => {
  return incomeBreakdown.value.reduce((sum, item) => sum + (item.count || 0), 0) || 
    (monthlyIncome.value > 0 ? 1 : 0);
});

// Expense count and top category
const expenseByCategory = computed(() => props.analytics.expenseByCategory || []);
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
const pieColors = ['#C27B7B', '#7B9BC2', '#6B9B7A', '#C9A856', '#9B7BC2', '#7BC2B5', '#C2997B', '#7B8BC2'];

const expensePieOption = computed(() => {
  // canvas 不解析 CSS 变量,颜色必须取实际值;依赖 themeVersion 以便主题切换时重算
  void themeVersion.value;
  const data = expenseByCategory.value.map((item, index) => ({
    name: getCategoryLabel(item.category),
    value: item.amount,
    itemStyle: { color: pieColors[index % pieColors.length] }
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
      itemStyle: { borderRadius: 6, borderColor: getThemeColor('--bg-secondary'), borderWidth: 2 },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 'bold' }
      },
      data
    }]
  };
});
</script>
