<template>
  <div class="animate-fade-in-up">
    <!-- Trend Period Selector -->
    <div class="card-static section-mb" style="padding: var(--space-4); display: flex; align-items: center; gap: var(--space-4);">
      <div style="display: flex; align-items: center; gap: var(--space-2);">
        <div class="stat-icon bg-brand-light" style="width: 36px; height: 36px;">
          <span v-html="icons.lineChart" style="stroke: var(--brand-primary); width: 18px; height: 18px;"></span>
        </div>
        <span style="font-weight: 500; color: var(--text-primary);">趋势周期</span>
      </div>
      <div style="display: flex; gap: var(--space-2);">
        <button
          v-for="period in periods"
          :key="period.value"
          class="btn"
          :class="selectedPeriod === period.value ? 'btn-primary' : 'btn-secondary'"
          @click="selectedPeriod = period.value"
        >
          {{ period.label }}
        </button>
      </div>
    </div>

    <!-- Main Trend Chart + Transaction Calendar (Side by Side) -->
    <div class="grid-7-3 section-mb">
      <!-- Main Trend Chart -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
          <div style="display: flex; align-items: center; gap: var(--space-3);">
            <div class="stat-icon bg-info-light" style="width: 40px; height: 40px;">
              <span v-html="icons.trends" style="stroke: var(--info); width: 20px; height: 20px;"></span>
            </div>
            <span style="font-weight: 600; color: var(--text-primary);">收支趋势</span>
          </div>
          <div style="display: flex; gap: var(--space-4); font-size: var(--font-size-sm);">
            <button 
              @click="toggleSeries('income')" 
              style="display: flex; align-items: center; gap: var(--space-2); background: none; border: none; cursor: pointer; padding: var(--space-1) var(--space-2); border-radius: var(--radius-md); transition: all 0.2s ease;"
              :style="{ opacity: seriesVisible.income ? 1 : 0.4, background: seriesVisible.income ? 'var(--bg-tertiary)' : 'transparent' }"
            >
              <span style="width: 12px; height: 12px; border-radius: 50%; background: var(--income);"></span>
              <span style="color: var(--text-secondary);">收入</span>
            </button>
            <button 
              @click="toggleSeries('expense')" 
              style="display: flex; align-items: center; gap: var(--space-2); background: none; border: none; cursor: pointer; padding: var(--space-1) var(--space-2); border-radius: var(--radius-md); transition: all 0.2s ease;"
              :style="{ opacity: seriesVisible.expense ? 1 : 0.4, background: seriesVisible.expense ? 'var(--bg-tertiary)' : 'transparent' }"
            >
              <span style="width: 12px; height: 12px; border-radius: 50%; background: var(--expense);"></span>
              <span style="color: var(--text-secondary);">支出</span>
            </button>
          </div>
        </div>
        <div style="height: 380px;">
          <v-chart v-if="trendData && trendData.length > 0" :option="trendChartOption" autoresize />
          <div v-else style="height: 100%; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
            暂无趋势数据
          </div>
        </div>
      </div>

      <!-- Transaction Calendar -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.calendar" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">交易日历</span>
        </div>
        <TransactionCalendar :dailyData="analytics.dailyTrend || []" />
      </div>
    </div>

    <!-- Expense Heatmap -->
    <div class="card-static section-mb" style="padding: var(--space-6);">
      <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
        <div class="stat-icon bg-warning-light" style="width: 40px; height: 40px;">
          <span v-html="icons.calendar" style="stroke: var(--warning); width: 20px; height: 20px;"></span>
        </div>
        <span style="font-weight: 600; color: var(--text-primary);">消费日历热力图</span>
      </div>
      <div style="height: 180px;">
        <v-chart v-if="heatmapOption" :option="heatmapOption" autoresize />
        <div v-else style="height: 100%; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无足够数据生成热力图
        </div>
      </div>
    </div>

    <!-- Weekday Distribution -->
    <div class="grid-1-1 section-mb">
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-warning-light" style="width: 40px; height: 40px;">
            <span v-html="icons.calendar" style="stroke: var(--warning); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">周消费分布</span>
        </div>
        <WeekdayChart v-if="analytics.weekdayDistribution && analytics.weekdayDistribution.length > 0" :data="analytics.weekdayDistribution" />
        <div v-else style="height: 200px; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无周消费数据
        </div>
      </div>

      <!-- Category Trend -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-expense-light" style="width: 40px; height: 40px;">
            <span v-html="icons.tags" style="stroke: var(--expense); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">分类消费环比</span>
        </div>
        <CategoryTrend v-if="analytics.categoryTrends && analytics.categoryTrends.length > 0" :data="analytics.categoryTrends" />
        <div v-else style="height: 200px; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无分类趋势数据
        </div>
      </div>
    </div>

    <!-- Annual Report Style Rankings -->
    <div class="grid-1-1 section-mb">
      <!-- Top Payees -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.trophy" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">年度“剁手”商户 Top 5</span>
        </div>
        <div v-if="topPayees.length > 0">
          <div v-for="item in topPayees" :key="item.name" style="display: flex; align-items: center; justify-content: space-between; padding: var(--space-3) 0; border-bottom: 1px solid var(--border-light);">
            <div style="display: flex; align-items: center; gap: var(--space-3);">
              <span :style="{ color: getRankColor(item.rank), fontWeight: 'bold', minWidth: '20px' }">#{{ item.rank }}</span>
              <span style="color: var(--text-primary); font-weight: 500;">{{ item.name }}</span>
            </div>
            <span style="font-weight: 600; color: var(--text-primary);">¥{{ Number(item.amount).toFixed(2) }}</span>
          </div>
        </div>
        <div v-else style="padding: var(--space-4); text-align: center; color: var(--text-tertiary);">
          暂无商户排行数据
        </div>
      </div>

      <!-- Top Tags -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-info-light" style="width: 40px; height: 40px;">
            <span v-html="icons.tags" style="stroke: var(--info); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">高频生活标签 Top 5</span>
        </div>
        <div v-if="topTags.length > 0">
          <div v-for="item in topTags" :key="item.name" style="display: flex; align-items: center; justify-content: space-between; padding: var(--space-3) 0; border-bottom: 1px solid var(--border-light);">
            <div style="display: flex; align-items: center; gap: var(--space-3);">
              <span :style="{ color: getRankColor(item.rank), fontWeight: 'bold', minWidth: '20px' }">#{{ item.rank }}</span>
              <span style="color: var(--text-primary); font-weight: 500;">#{{ item.name }}</span>
            </div>
            <span style="font-weight: 600; color: var(--text-primary);">{{ item.value }}</span>
          </div>
        </div>
        <div v-else style="padding: var(--space-4); text-align: center; color: var(--text-tertiary);">
          暂无标签排行数据
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { LineChart, HeatmapChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, LegendComponent, CalendarComponent, VisualMapComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import WeekdayChart from '../WeekdayChart.vue';
import CategoryTrend from '../CategoryTrendChart.vue';
import TransactionCalendar from '../TransactionCalendar.vue';
import { icons } from '../../composables/icons';

use([LineChart, HeatmapChart, GridComponent, TooltipComponent, LegendComponent, CalendarComponent, VisualMapComponent, CanvasRenderer]);

const props = defineProps({
  analytics: { type: Object, required: true }
});

const periods = [
  { value: 'day', label: '日' },
  { value: 'week', label: '周' },
  { value: 'month', label: '月' },
];
const selectedPeriod = ref('day');

// Series visibility state
const seriesVisible = ref({
  income: true,
  expense: true
});

// Toggle series visibility
const toggleSeries = (series) => {
  seriesVisible.value[series] = !seriesVisible.value[series];
};

// Get the appropriate trend data based on selected period
const trendData = computed(() => {
  switch (selectedPeriod.value) {
    case 'day':
      return props.analytics.dailyTrend || [];
    case 'week':
      return props.analytics.weeklyTrend || [];
    case 'month':
      return props.analytics.monthlyTrend || [];
    default:
      return props.analytics.monthlyTrend || [];
  }
});

// Format x-axis label based on period
const formatLabel = (item) => {
  if (selectedPeriod.value === 'day') {
    return item.date?.slice(5) || '';
  } else if (selectedPeriod.value === 'week') {
    return item.week?.split('-W')[1] || '';
  } else {
    return item.month?.slice(5) || '';
  }
};

const trendChartOption = computed(() => {
  const data = trendData.value;
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'var(--bg-secondary)',
      borderColor: 'var(--border)',
      textStyle: { color: 'var(--text-primary)' }
    },
    grid: { left: 50, right: 20, top: 20, bottom: 30 },
    xAxis: {
      type: 'category',
      data: data.map(d => formatLabel(d)),
      axisLine: { lineStyle: { color: 'var(--border)' } },
      axisLabel: { 
        color: '#9CA3AF',
        rotate: selectedPeriod.value === 'day' ? 45 : 0,
        fontSize: selectedPeriod.value === 'day' ? 10 : 12
      }
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } },
      axisLabel: { color: '#9CA3AF', formatter: v => v >= 1000 ? `¥${(v/1000).toFixed(0)}k` : `¥${v}` }
    },
    series: [
      {
        name: '收入',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: selectedPeriod.value === 'day' ? 4 : 6,
        lineStyle: { color: '#6B9B7A', width: 2, opacity: seriesVisible.value.income ? 1 : 0 },
        itemStyle: { color: '#6B9B7A', opacity: seriesVisible.value.income ? 1 : 0 },
        areaStyle: seriesVisible.value.income ? { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(107,155,122,0.3)' }, { offset: 1, color: 'rgba(107,155,122,0)' }] } } : { opacity: 0 },
        data: seriesVisible.value.income ? data.map(d => d.income) : data.map(() => null)
      },
      {
        name: '支出',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: selectedPeriod.value === 'day' ? 4 : 6,
        lineStyle: { color: '#C27B7B', width: 2, opacity: seriesVisible.value.expense ? 1 : 0 },
        itemStyle: { color: '#C27B7B', opacity: seriesVisible.value.expense ? 1 : 0 },
        areaStyle: seriesVisible.value.expense ? { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(194,123,123,0.3)' }, { offset: 1, color: 'rgba(194,123,123,0)' }] } } : { opacity: 0 },
        data: seriesVisible.value.expense ? data.map(d => Math.abs(d.expense)) : data.map(() => null)
      }
    ]
  };
});

// Heatmap Logic
const heatmapOption = computed(() => {
  const data = props.analytics.dailyTrend || [];
  // Map to [date, expense]
  const heatmapData = data.map(d => [d.date, Math.abs(d.expense)]);
  
  if (heatmapData.length === 0) return null;
  
  const maxExpense = Math.max(...heatmapData.map(d => d[1]));

  return {
    tooltip: {
      formatter: (p) => {
        return `${p.data[0]}: ¥${p.data[1].toFixed(2)}`;
      },
      backgroundColor: 'var(--bg-secondary)',
      borderColor: 'var(--border)',
      textStyle: { color: 'var(--text-primary)' }
    },
    visualMap: {
      min: 0,
      max: maxExpense,
      type: 'continuous',
      orient: 'horizontal',
      left: 'center',
      bottom: 0,
      inRange: {
        color: ['#ebedf0', '#c6e48b', '#7bc96f', '#239a3b', '#196127'] // GitHub-like green
        // Or Expense focused:
        // color: ['#ebedf0', '#f2d0d0', '#e3a1a1', '#c27b7b', '#a65959']
      },
      text: ['High', 'Low'],
      calculable: false,
      show: false // hide legend to save space
    },
    calendar: {
      top: 30,
      left: 30,
      right: 30,
      cellSize: ['auto', 16],
      range: new Date().getFullYear(), // Current Year
      itemStyle: {
        color: 'transparent',
        borderColor: 'var(--border)',
        borderWidth: 1
      },
      yearLabel: { show: false },
      dayLabel: { nameMap: ['S', 'M', 'T', 'W', 'T', 'F', 'S'], color: '#9CA3AF' },
      monthLabel: { nameMap: 'en', color: '#9CA3AF' },
      splitLine: { show: false }
    },
    series: {
      type: 'heatmap',
      coordinateSystem: 'calendar',
      data: heatmapData,
      itemStyle: {
        borderRadius: 2,
        borderColor: 'var(--bg-secondary)', // Gap color
        borderWidth: 1
      }
    }
  };
});

// Ranking Logic
// 1. Top Payees
const topPayees = computed(() => {
  const txs = props.analytics.recentTransactions || [];
  const payeeMap = {};
  
  txs.forEach(tx => {
    let isExpense = false;
    let amount = 0;

    // Try to get data from pre-calculated fields if they exist
    if (typeof tx.isIncome === 'boolean' && tx.amount !== undefined) {
      if (!tx.isIncome) {
        isExpense = true;
        amount = Math.abs(tx.amount);
      }
    } 
    // Fallback to postings logic (checking for Expenses account)
    else if (tx.postings && tx.postings.length) {
      for (const p of tx.postings) {
        if (p.account && p.account.startsWith('Expenses:')) {
          isExpense = true;
          amount += Math.abs(p.amount);
        }
      }
    }

    if (isExpense && tx.payee && amount > 0) {
      payeeMap[tx.payee] = (payeeMap[tx.payee] || 0) + amount;
    }
  });
  
  return Object.entries(payeeMap)
    .sort((a, b) => b[1] - a[1]) // Sort by amount descending
    .slice(0, 5) // Top 5
    .map(([name, amount], index) => ({ rank: index + 1, name, amount }));
});

// 2. Top Tags
const topTags = computed(() => {
  const txs = props.analytics.recentTransactions || [];
  const tagMap = {};
  txs.forEach(tx => {
    if (tx.tags && tx.tags.length) {
      tx.tags.forEach(tag => {
        tagMap[tag] = (tagMap[tag] || 0) + 1; // Count frequency
      });
    }
  });

  return Object.entries(tagMap)
    .sort((a, b) => b[1] - a[1]) // Sort by frequency descending
    .slice(0, 5) // Top 5
    .map(([name, count], index) => ({ rank: index + 1, name, value: `${count}次` }));
});

function getRankColor(rank) {
  if (rank === 1) return '#FFD700'; // Gold
  if (rank === 2) return '#C0C0C0'; // Silver
  if (rank === 3) return '#CD7F32'; // Bronze
  return 'var(--text-tertiary)';
}
</script>

