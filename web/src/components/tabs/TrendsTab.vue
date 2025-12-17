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

    <!-- Main Trend Chart -->
    <div class="card-static section-mb" style="padding: var(--space-6);">
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
      <div style="height: 320px;">
        <v-chart v-if="trendData && trendData.length > 0" :option="trendChartOption" autoresize />
        <div v-else style="height: 100%; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无趋势数据
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
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import WeekdayChart from '../WeekdayChart.vue';
import CategoryTrend from '../CategoryTrendChart.vue';
import { icons } from '../../composables/icons';

use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

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
    default:
      return props.analytics.monthlyTrend || [];
  }
});

// Format x-axis label based on period
const formatLabel = (item) => {
  if (selectedPeriod.value === 'day') {
    // Show date as MM-DD
    return item.date?.slice(5) || '';
  } else if (selectedPeriod.value === 'week') {
    // Show week as W prefix
    return item.week?.split('-W')[1] || '';
  } else {
    // Show month as MM
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
        color: 'var(--text-secondary)',
        rotate: selectedPeriod.value === 'day' ? 45 : 0,
        fontSize: selectedPeriod.value === 'day' ? 10 : 12
      }
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } },
      axisLabel: { color: 'var(--text-secondary)', formatter: v => v >= 1000 ? `¥${(v/1000).toFixed(0)}k` : `¥${v}` }
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
</script>

