<template>
  <div class="animate-fade-in-up">
    <!-- Stats Grid -->
    <div class="stats-grid">
      <!-- Total Balance -->
      <div class="card stat-card">
        <div class="stat-header">
          <div class="stat-icon bg-brand-light">
            <span v-html="icons.wallet" style="stroke: var(--brand-primary);"></span>
          </div>
          <span class="stat-badge bg-income-light text-income">
            {{ balanceChange }}
          </span>
        </div>
        <div class="stat-label">总资产</div>
        <div class="stat-value" style="color: var(--text-primary);">{{ formatMoney(netWorth) }}</div>
      </div>

      <!-- Monthly Income -->
      <div class="card stat-card delay-100">
        <div class="stat-header">
          <div class="stat-icon bg-income-light">
            <span v-html="icons.arrowUp" style="stroke: var(--income);"></span>
          </div>
          <span class="stat-badge bg-income-light text-income">
            +{{ incomeChange }}%
          </span>
        </div>
        <div class="stat-label">本月收入</div>
        <div class="stat-value text-income">{{ formatMoney(monthlyIncome) }}</div>
      </div>

      <!-- Monthly Expense -->
      <div class="card stat-card delay-200">
        <div class="stat-header">
          <div class="stat-icon bg-expense-light">
            <span v-html="icons.arrowDown" style="stroke: var(--expense);"></span>
          </div>
          <span class="stat-badge bg-expense-light text-expense">
            {{ expenseChange }}%
          </span>
        </div>
        <div class="stat-label">本月支出</div>
        <div class="stat-value text-expense">{{ formatMoney(monthlyExpense) }}</div>
      </div>

      <!-- Savings Rate -->
      <div class="card stat-card delay-300">
        <div class="stat-header">
          <div class="stat-icon bg-warning-light">
            <span v-html="icons.target" style="stroke: var(--warning);"></span>
          </div>
          <span class="stat-badge bg-warning-light text-warning">
            {{ savingsRate }}%
          </span>
        </div>
        <div class="stat-label">储蓄率</div>
        <div class="stat-value" style="color: var(--text-primary);">
          {{ formatMoney(monthlySavings) }}
          <span style="font-size: var(--font-size-sm); font-weight: 400; color: var(--text-tertiary);">/ 月</span>
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
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { formatMoney } from '../../composables/useFormatters';
import { icons } from '../../composables/icons';

const props = defineProps({
  analytics: { type: Object, required: true }
});

// Computed values
const netWorth = computed(() => props.analytics.summary?.netWorth || 0);
const monthlyIncome = computed(() => props.analytics.summary?.totalIncome || 0);
const monthlyExpense = computed(() => Math.abs(props.analytics.summary?.totalExpense || 0));
const monthlySavings = computed(() => monthlyIncome.value - monthlyExpense.value);

const balanceChange = computed(() => {
  const change = ((netWorth.value / 100000) * 2).toFixed(1);
  return change > 0 ? `+${change}%` : `${change}%`;
});

const incomeChange = computed(() => Math.floor(Math.random() * 15) + 5);
const expenseChange = computed(() => -1 * (Math.floor(Math.random() * 10) + 1));

const savingsRate = computed(() => {
  if (monthlyIncome.value === 0) return 0;
  return Math.round((monthlySavings.value / monthlyIncome.value) * 100);
});

const debtRatio = computed(() => {
  const liabilities = Math.abs(props.analytics.summary?.totalLiabilities || 0);
  const assets = props.analytics.summary?.totalAssets || 1;
  return (liabilities / assets) * 100;
});

const dailyAverage = computed(() => {
  const days = new Date().getDate();
  return monthlyExpense.value / days;
});

const daysInMonth = computed(() => {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
});

const monthProgress = computed(() => {
  const now = new Date();
  return Math.round((now.getDate() / daysInMonth.value) * 100);
});
</script>
