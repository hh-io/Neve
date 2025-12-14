<template>
  <div class="glass-card annual-report">
    <div class="report-header">
      <h3 class="card-title">{{ year }}年度报告</h3>
      <div class="year-nav">
        <button @click="year--" class="nav-btn">◀</button>
        <span class="year-display">{{ year }}</span>
        <button @click="year++" class="nav-btn" :disabled="year >= currentYear">▶</button>
      </div>
    </div>

    <div class="report-grid">
      <!-- Total Income/Expense -->
      <div class="report-stat">
        <span class="stat-label">年度收入</span>
        <span class="stat-value income">¥{{ formatNum(yearData.totalIncome) }}</span>
      </div>
      <div class="report-stat">
        <span class="stat-label">年度支出</span>
        <span class="stat-value expense">¥{{ formatNum(yearData.totalExpense) }}</span>
      </div>
      <div class="report-stat">
        <span class="stat-label">年度结余</span>
        <span class="stat-value" :class="yearData.totalSavings >= 0 ? 'income' : 'expense'">
          ¥{{ formatNum(yearData.totalSavings) }}
        </span>
      </div>
      <div class="report-stat">
        <span class="stat-label">储蓄率</span>
        <span class="stat-value" :class="yearData.savingsRate > 20 ? 'income' : 'warning'">
          {{ yearData.savingsRate.toFixed(1) }}%
        </span>
      </div>
    </div>

    <!-- Monthly Chart -->
    <div class="chart-container">
      <v-chart :option="chartOption" autoresize />
    </div>

    <!-- Year Comparison -->
    <div v-if="lastYearData.totalIncome > 0" class="comparison">
      <h4 class="comparison-title">同比变化 (vs {{ year - 1 }})</h4>
      <div class="comparison-grid">
        <div class="comparison-item">
          <span>收入</span>
          <span :class="incomeChange >= 0 ? 'positive' : 'negative'">
            {{ incomeChange >= 0 ? '+' : '' }}{{ incomeChange.toFixed(1) }}%
          </span>
        </div>
        <div class="comparison-item">
          <span>支出</span>
          <span :class="expenseChange <= 0 ? 'positive' : 'negative'">
            {{ expenseChange >= 0 ? '+' : '' }}{{ expenseChange.toFixed(1) }}%
          </span>
        </div>
        <div class="comparison-item">
          <span>结余</span>
          <span :class="savingsChange >= 0 ? 'positive' : 'negative'">
            {{ savingsChange >= 0 ? '+' : '' }}{{ savingsChange.toFixed(1) }}%
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import VChart from 'vue-echarts';

const props = defineProps({
  monthlyTrend: { type: Array, default: () => [] }
});

const currentYear = new Date().getFullYear();
const year = ref(currentYear);

const yearData = computed(() => {
  const yearStr = String(year.value);
  const months = props.monthlyTrend.filter(m => m.month.startsWith(yearStr));
  
  const totalIncome = months.reduce((sum, m) => sum + m.income, 0);
  const totalExpense = months.reduce((sum, m) => sum + m.expense, 0);
  const totalSavings = totalIncome - totalExpense;
  const savingsRate = totalIncome > 0 ? (totalSavings / totalIncome) * 100 : 0;
  
  return { totalIncome, totalExpense, totalSavings, savingsRate, months };
});

const lastYearData = computed(() => {
  const yearStr = String(year.value - 1);
  const months = props.monthlyTrend.filter(m => m.month.startsWith(yearStr));
  
  const totalIncome = months.reduce((sum, m) => sum + m.income, 0);
  const totalExpense = months.reduce((sum, m) => sum + m.expense, 0);
  const totalSavings = totalIncome - totalExpense;
  
  return { totalIncome, totalExpense, totalSavings };
});

const incomeChange = computed(() => {
  if (lastYearData.value.totalIncome === 0) return 0;
  return ((yearData.value.totalIncome - lastYearData.value.totalIncome) / lastYearData.value.totalIncome) * 100;
});

const expenseChange = computed(() => {
  if (lastYearData.value.totalExpense === 0) return 0;
  return ((yearData.value.totalExpense - lastYearData.value.totalExpense) / lastYearData.value.totalExpense) * 100;
});

const savingsChange = computed(() => {
  if (lastYearData.value.totalSavings === 0) return 0;
  return ((yearData.value.totalSavings - lastYearData.value.totalSavings) / Math.abs(lastYearData.value.totalSavings)) * 100;
});

const chartOption = computed(() => {
  const months = yearData.value.months;
  const labels = months.map(m => m.month.slice(5) + '月');
  
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '10%', containLabel: true },
    xAxis: {
      type: 'category',
      data: labels,
      axisLabel: { color: '#6E6E73', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#6E6E73', formatter: v => v >= 1000 ? (v / 1000) + 'k' : v },
      splitLine: { lineStyle: { color: '#F2F2F7' } },
    },
    series: [
      {
        name: '收入',
        type: 'bar',
        data: months.map(m => m.income),
        itemStyle: { color: '#30D158', borderRadius: [4, 4, 0, 0] },
      },
      {
        name: '支出',
        type: 'bar',
        data: months.map(m => m.expense),
        itemStyle: { color: '#FF453A', borderRadius: [4, 4, 0, 0] },
      },
    ]
  };
});

function formatNum(num) {
  return num >= 10000 ? (num / 10000).toFixed(2) + '万' : num.toFixed(0);
}
</script>

<style scoped>
.annual-report {
  padding: var(--space-6);
}

.report-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
}

.report-header .card-title {
  margin-bottom: 0;
}

.year-nav {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.nav-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: rgba(0, 0, 0, 0.05);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 12px;
}

.nav-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.year-display {
  font-weight: 600;
  min-width: 50px;
  text-align: center;
}

.report-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

.report-stat {
  text-align: center;
  padding: var(--space-3);
  background: rgba(0, 0, 0, 0.02);
  border-radius: var(--radius-md);
}

.stat-label {
  display: block;
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  margin-bottom: var(--space-1);
}

.stat-value {
  font-size: var(--font-size-lg);
  font-weight: 700;
}

.stat-value.income { color: var(--color-green); }
.stat-value.expense { color: var(--color-red); }
.stat-value.warning { color: var(--color-orange); }

.chart-container {
  height: 200px;
  margin-bottom: var(--space-4);
}

.comparison {
  padding-top: var(--space-4);
  border-top: 1px dashed rgba(0, 0, 0, 0.1);
}

.comparison-title {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  margin-bottom: var(--space-3);
}

.comparison-grid {
  display: flex;
  gap: var(--space-6);
}

.comparison-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.comparison-item span:first-child {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.comparison-item .positive { color: var(--color-green); font-weight: 600; }
.comparison-item .negative { color: var(--color-red); font-weight: 600; }
</style>
