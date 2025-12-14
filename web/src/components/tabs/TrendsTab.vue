<template>
  <div class="tab-content">
    <section class="analytics-section fade-in">
      <div class="grid grid-2">
        <!-- Trend Chart with Period Selector -->
        <div class="glass-card chart-card">
          <div class="card-header">
            <h3 class="card-title">收支趋势</h3>
            <div class="period-selector">
              <button class="period-btn" :class="{ active: trendPeriod === 'day' }" @click="trendPeriod = 'day'">日</button>
              <button class="period-btn" :class="{ active: trendPeriod === 'week' }" @click="trendPeriod = 'week'">周</button>
              <button class="period-btn" :class="{ active: trendPeriod === 'month' }" @click="trendPeriod = 'month'">月</button>
            </div>
          </div>
          <div class="chart-container">
            <v-chart :option="trendChartOption" autoresize />
          </div>
        </div>

        <!-- Monthly Comparison Bar Chart -->
        <div class="glass-card chart-card">
          <div class="card-header">
            <h3 class="card-title">月度对比</h3>
            <div class="period-selector">
              <button class="period-btn" :class="{ active: comparisonMonths === 3 }" @click="comparisonMonths = 3">3个月</button>
              <button class="period-btn" :class="{ active: comparisonMonths === 6 }" @click="comparisonMonths = 6">6个月</button>
              <button class="period-btn" :class="{ active: comparisonMonths === 12 }" @click="comparisonMonths = 12">12个月</button>
            </div>
          </div>
          <div class="chart-container">
            <v-chart :option="comparisonChartOption" autoresize />
          </div>
        </div>
      </div>
    </section>

    <section class="analytics-section fade-in">
      <div class="grid grid-2">
        <WeekdayChart :data="analytics.weekdayDistribution" />
        <CategoryTrendChart :data="analytics.categoryTrends" />
      </div>
    </section>

    <section class="analytics-section fade-in">
      <AnnualReport :monthlyTrend="analytics.monthlyTrend || []" />
    </section>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import VChart from 'vue-echarts';
import WeekdayChart from '../WeekdayChart.vue';
import CategoryTrendChart from '../CategoryTrendChart.vue';
import AnnualReport from '../AnnualReport.vue';

const props = defineProps({
  analytics: { type: Object, required: true }
});

const trendPeriod = ref('day');
const comparisonMonths = ref(3);

// Trend Chart Option
const trendChartOption = computed(() => {
  let dataByPeriod;
  if (trendPeriod.value === 'month') {
    dataByPeriod = props.analytics.monthlyTrend || [];
  } else if (trendPeriod.value === 'week') {
    dataByPeriod = aggregateByWeek(props.analytics.recentTransactions || []);
  } else {
    dataByPeriod = aggregateByDay(props.analytics.recentTransactions || []);
  }
  
  const labels = dataByPeriod.map(d => d.label || d.month || d.date);
  const incomeData = dataByPeriod.map(d => d.income);
  const expenseData = dataByPeriod.map(d => d.expense);

  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['收入', '支出'], bottom: 0, textStyle: { color: '#6E6E73' } },
    grid: { left: '3%', right: '4%', bottom: '15%', top: '5%', containLabel: true },
    xAxis: { type: 'category', data: labels, axisLabel: { color: '#6E6E73', fontSize: 10 } },
    yAxis: { type: 'value', axisLabel: { color: '#6E6E73' }, splitLine: { lineStyle: { color: '#F2F2F7' } } },
    series: [
      { name: '收入', type: 'line', smooth: true, data: incomeData, itemStyle: { color: '#30D158' }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(48, 209, 88, 0.3)' }, { offset: 1, color: 'rgba(48, 209, 88, 0.05)' }] } } },
      { name: '支出', type: 'line', smooth: true, data: expenseData, itemStyle: { color: '#FF453A' }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(255, 69, 58, 0.3)' }, { offset: 1, color: 'rgba(255, 69, 58, 0.05)' }] } } },
    ],
  };
});

// Comparison Chart Option
const comparisonChartOption = computed(() => {
  const trend = props.analytics.monthlyTrend || [];
  const data = trend.slice(-comparisonMonths.value);
  const labels = data.map(m => m.month.slice(5) + '月');
  const incomeData = data.map(m => m.income);
  const expenseData = data.map(m => m.expense);
  const balanceData = data.map(m => m.income - m.expense);

  const getRadius = (v) => v >= 0 ? [6, 6, 0, 0] : [0, 0, 6, 6];

  return {
    tooltip: { trigger: 'axis', formatter: (params) => {
      let html = `<strong>${params[0].axisValue}</strong><br/>`;
      params.forEach(p => {
        let marker = p.marker;
        if (p.seriesName === '结余') {
          const color = p.value >= 0 ? '#007AFF' : '#AF52DE';
          marker = `<span style="display:inline-block;margin-right:4px;border-radius:10px;width:10px;height:10px;background-color:${color};"></span>`;
        }
        html += `${marker} ${p.seriesName}: ¥${p.value.toFixed(2)}<br/>`;
      });
      return html;
    }},
    legend: { data: ['收入', '支出', '结余'], bottom: 0, textStyle: { color: '#6E6E73' } },
    grid: { left: '3%', right: '4%', bottom: '15%', top: '5%', containLabel: true },
    xAxis: { type: 'category', data: labels, axisLabel: { color: '#6E6E73' } },
    yAxis: { type: 'value', axisLabel: { color: '#6E6E73' }, splitLine: { lineStyle: { color: '#F2F2F7' } } },
    series: [
      { name: '收入', type: 'bar', data: incomeData.map(v => ({ value: v, itemStyle: { borderRadius: getRadius(v) } })), itemStyle: { color: '#30D158' }, barWidth: '20%' },
      { name: '支出', type: 'bar', data: expenseData.map(v => ({ value: v, itemStyle: { borderRadius: getRadius(v) } })), itemStyle: { color: '#FF453A' }, barWidth: '20%' },
      { name: '结余', type: 'bar', data: balanceData.map(v => ({ value: v, itemStyle: { borderRadius: getRadius(v) } })), itemStyle: { color: (params) => balanceData[params.dataIndex] >= 0 ? '#007AFF' : '#AF52DE' }, barWidth: '20%' },
    ],
  };
});

function aggregateByDay(transactions) {
  const days = {};
  for (let i = 29; i >= 0; i--) {
    const date = new Date();
    date.setDate(date.getDate() - i);
    const key = date.toISOString().slice(0, 10);
    days[key] = { income: 0, expense: 0, label: `${date.getMonth() + 1}/${date.getDate()}` };
  }
  transactions.forEach(tx => {
    const key = tx.date?.slice(0, 10);
    if (!days[key]) return;
    tx.postings?.forEach(p => {
      if (p.account?.startsWith('Expenses') && p.amount > 0) days[key].expense += p.amount;
      else if (p.account?.startsWith('Income') && p.amount < 0) days[key].income += -p.amount;
    });
  });
  return Object.values(days);
}

function aggregateByWeek(transactions) {
  const weeks = {};
  const now = new Date();
  for (let i = 7; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i * 7);
    const weekNum = getWeekNumber(date);
    const key = `${date.getFullYear()}-W${weekNum}`;
    weeks[key] = { income: 0, expense: 0, label: `第${weekNum}周` };
  }
  transactions.forEach(tx => {
    const date = new Date(tx.date);
    const weekNum = getWeekNumber(date);
    const key = `${date.getFullYear()}-W${weekNum}`;
    if (!weeks[key]) return;
    tx.postings?.forEach(p => {
      if (p.account?.startsWith('Expenses') && p.amount > 0) weeks[key].expense += p.amount;
      else if (p.account?.startsWith('Income') && p.amount < 0) weeks[key].income += -p.amount;
    });
  });
  return Object.values(weeks);
}

function getWeekNumber(d) {
  d = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()));
  const dayNum = d.getUTCDay() || 7;
  d.setUTCDate(d.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
  return Math.ceil(((d - yearStart) / 86400000 + 1) / 7);
}
</script>
