<template>
  <div class="tab-content">
    <!-- Summary Cards -->
    <section class="summary-section fade-in">
      <div class="grid grid-4">
        <div class="glass-card stat-card net-worth-card">
          <span class="label">净资产</span>
          <span class="value">{{ formatMoney(analytics.summary.netWorth) }}</span>
          <span class="trend positive" v-if="analytics.summary.netWorth > 0">
            资产 {{ formatMoney(analytics.summary.totalAssets) }}
          </span>
        </div>
        <div class="glass-card stat-card">
          <span class="label">本月收入</span>
          <span class="value income">{{ formatMoney(analytics.summary.monthIncome) }}</span>
        </div>
        <div class="glass-card stat-card">
          <span class="label">本月支出</span>
          <span class="value expense">{{ formatMoney(analytics.summary.monthExpense) }}</span>
        </div>
        <div class="glass-card stat-card">
          <span class="label">本月结余</span>
          <span class="value" :class="analytics.summary.monthBalance >= 0 ? 'income' : 'expense'">
            {{ formatMoney(analytics.summary.monthBalance) }}
          </span>
        </div>
      </div>
    </section>

    <!-- Daily Average & Health -->
    <section class="analytics-section fade-in">
      <div class="grid grid-2">
        <div class="glass-card stat-card daily-avg-card">
          <span class="label">日均消费</span>
          <span class="value expense">¥{{ analytics.dailyAverage?.toFixed(2) || '0.00' }}</span>
          <span class="trend">本月已消费 {{ new Date().getDate() }} 天</span>
        </div>
        <div class="glass-card">
          <h3 class="card-title">财务健康</h3>
          <div class="health-stats">
            <div class="health-item">
              <span class="health-label">资产负债比</span>
              <span class="health-value" :class="debtRatio < 0.5 ? 'good' : 'warning'">
                {{ (debtRatio * 100).toFixed(1) }}%
              </span>
            </div>
            <div class="health-item">
              <span class="health-label">月结余率</span>
              <span class="health-value" :class="savingsRate > 0.2 ? 'good' : savingsRate > 0 ? 'ok' : 'warning'">
                {{ (savingsRate * 100).toFixed(1) }}%
              </span>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { formatMoney } from '../../composables/useFormatters';

const props = defineProps({
  analytics: { type: Object, required: true }
});

const debtRatio = computed(() => {
  const liabilities = Math.abs(props.analytics.summary?.totalLiabilities || 0);
  const assets = props.analytics.summary?.totalAssets || 1;
  return liabilities / assets;
});

const savingsRate = computed(() => {
  const income = props.analytics.summary?.monthIncome || 1;
  const balance = props.analytics.summary?.monthBalance || 0;
  return balance / income;
});
</script>
