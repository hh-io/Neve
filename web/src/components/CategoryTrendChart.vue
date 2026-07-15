<template>
  <div>
    <!-- Header with category selector and stats -->
    <div class="ct-header">
      <div class="ct-selectors">
        <select v-model="selectedCategory" class="category-select">
          <option v-for="cat in categories" :key="cat" :value="cat">
            {{ getCategoryLabel(cat) }}
          </option>
        </select>
        <button 
          v-if="!compareMode" 
          class="compare-btn" 
          title="对比模式"
          @click="compareMode = true"
        >
          + 对比
        </button>
        <select v-if="compareMode" v-model="compareCategory" class="category-select">
          <option value="">取消对比</option>
          <option v-for="cat in categories.filter(c => c !== selectedCategory)" :key="cat" :value="cat">
            {{ getCategoryLabel(cat) }}
          </option>
        </select>
      </div>
      
      <!-- Month-over-month change indicator -->
      <div v-if="momChange !== null" class="mom-indicator" :class="momChange >= 0 ? 'up' : 'down'">
        <span class="mom-arrow">{{ momChange >= 0 ? '↑' : '↓' }}</span>
        <span class="mom-value">{{ Math.abs(momChange).toFixed(1) }}%</span>
        <span class="mom-label">环比</span>
      </div>
    </div>
    
    <!-- Stats row -->
    <div class="ct-stats">
      <div class="stat-chip">
        <span class="stat-chip-label">本月</span>
        <span class="stat-chip-value">¥{{ formatNum(currentMonthAmount) }}</span>
      </div>
      <div class="stat-chip">
        <span class="stat-chip-label">6月均值</span>
        <span class="stat-chip-value">¥{{ formatNum(avgAmount) }}</span>
      </div>
      <div v-if="anomalyMonth" class="stat-chip anomaly">
        <span class="stat-chip-label">异常月份</span>
        <span class="stat-chip-value">{{ anomalyMonth }}</span>
      </div>
    </div>
    
    <div class="chart-container">
      <v-chart :option="chartOption" autoresize />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, MarkLineComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import type { CategoryTrend } from '../types/api';
import { getThemeColor, themeVersion } from '../composables/useThemeColor';
import { getCategoryLabel } from '../composables/useCategories';

use([LineChart, GridComponent, TooltipComponent, MarkLineComponent, CanvasRenderer]);

const props = withDefaults(defineProps<{
  data?: CategoryTrend[];
}>(), {
  data: () => []
});

const categories = computed(() => props.data.map(d => d.category));
const selectedCategory = ref('');
const compareCategory = ref('');
const compareMode = ref(false);

// Watch compareCategory to exit compare mode
watch(compareCategory, (val) => {
  if (!val) compareMode.value = false;
});

function formatNum(val: number): string {
  if (val >= 1000) return (val / 1000).toFixed(1) + 'k';
  return val.toFixed(0);
}

watch(() => props.data, (newData) => {
  if (newData?.length && !selectedCategory.value) {
    selectedCategory.value = newData[0].category;
  }
}, { immediate: true });

const selectedData = computed(() => {
  return props.data.find(d => d.category === selectedCategory.value)?.data || [];
});

const compareData = computed(() => {
  if (!compareCategory.value) return [];
  return props.data.find(d => d.category === compareCategory.value)?.data || [];
});

// Calculate month-over-month change
const momChange = computed(() => {
  const data = selectedData.value;
  if (data.length < 2) return null;
  const current = data[data.length - 1]?.amount || 0;
  const previous = data[data.length - 2]?.amount || 0;
  if (previous === 0) return current > 0 ? 100 : 0;
  return ((current - previous) / previous) * 100;
});

// Current month amount
const currentMonthAmount = computed(() => {
  const data = selectedData.value;
  return data.length > 0 ? data[data.length - 1]?.amount || 0 : 0;
});

// Average amount
const avgAmount = computed(() => {
  const data = selectedData.value;
  if (data.length === 0) return 0;
  const total = data.reduce((sum, d) => sum + d.amount, 0);
  return total / data.length;
});

// Anomaly detection (2x average)
const anomalyMonth = computed(() => {
  const data = selectedData.value;
  if (data.length === 0) return null;
  const avg = avgAmount.value;
  for (const d of data) {
    if (d.amount > avg * 2) {
      return d.month.slice(5) + '月';
    }
  }
  return null;
});

const chartOption = computed(() => {
  void themeVersion.value;
  if (!selectedData.value?.length) {
    return {
      title: {
        text: '暂无数据',
        left: 'center',
        top: 'center',
        textStyle: { color: getThemeColor('--text-tertiary'), fontSize: 14 }
      }
    };
  }

  const labels = selectedData.value.map(d => d.month.slice(5) + '月');
  const amounts = selectedData.value.map(d => d.amount);
  const avg = avgAmount.value;

  const series: Record<string, unknown>[] = [{
    name: getCategoryLabel(selectedCategory.value),
    type: 'line',
    data: amounts,
    smooth: true,
    symbol: 'circle',
    symbolSize: (value: number) => value > avg * 2 ? 10 : 6, // Larger symbol for anomalies
    lineStyle: { width: 2, color: getThemeColor('--expense') },
    itemStyle: {
      color: getThemeColor('--expense')
    },
    areaStyle: {
      color: {
        type: 'linear',
        x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: getThemeColor('--expense-light') },
          { offset: 1, color: 'rgba(0,0,0,0)' }
        ]
      }
    },
    markLine: {
      silent: true,
      symbol: 'none',
      lineStyle: { color: getThemeColor('--text-tertiary'), type: 'dashed', width: 1 },
      data: [{ yAxis: avg, label: { formatter: '均值', position: 'end', fontSize: 10, color: getThemeColor('--text-tertiary') } }]
    }
  }];

  // Add compare series if in compare mode
  if (compareMode.value && compareData.value.length > 0) {
    series.push({
      name: getCategoryLabel(compareCategory.value),
      type: 'line',
      data: compareData.value.map(d => d.amount),
      smooth: true,
      symbol: 'circle',
      symbolSize: 6,
      lineStyle: { width: 2, color: getThemeColor('--income') },
      itemStyle: { color: getThemeColor('--income') },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: getThemeColor('--income-light') },
            { offset: 1, color: 'rgba(0,0,0,0)' }
          ]
        }
      }
    });
  }

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: getThemeColor('--surface-1'),
      borderColor: getThemeColor('--hairline'),
      textStyle: { color: getThemeColor('--text-primary') }
    },
    legend: compareMode.value && compareCategory.value ? {
      data: [getCategoryLabel(selectedCategory.value), getCategoryLabel(compareCategory.value)],
      bottom: 0,
      textStyle: { color: getThemeColor('--text-secondary'), fontSize: 11 }
    } : undefined,
    grid: {
      left: 10,
      right: 10,
      bottom: compareMode.value && compareCategory.value ? 30 : 10,
      top: 20,
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: labels,
      axisLine: { lineStyle: { color: getThemeColor('--hairline') } },
      axisLabel: { color: getThemeColor('--text-secondary'), fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: getThemeColor('--hairline'), type: 'dashed' } },
      axisLabel: {
        color: getThemeColor('--text-secondary'),
        formatter: (val: number) => val >= 1000 ? (val / 1000).toFixed(0) + 'k' : val,
      },
    },
    series,
  };
});
</script>

<style scoped>
.ct-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-3);
  flex-wrap: wrap;
  gap: var(--space-2);
}

.ct-selectors {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.ct-stats {
  display: flex;
  gap: var(--space-4);
  margin-bottom: var(--space-3);
  flex-wrap: wrap;
}

.category-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  background: var(--surface-2);
  color: var(--text-primary);
  cursor: pointer;
  outline: none;
  transition: border-color var(--transition-base);
}

.category-select:hover {
  border-color: var(--hairline-strong);
}

.category-select:focus {
  border-color: var(--accent);
}

.compare-btn {
  padding: var(--space-1) var(--space-2);
  border: 1px dashed var(--hairline);
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-base);
}

.compare-btn:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.mom-indicator {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 500;
}

.mom-indicator.up {
  background: var(--expense-light);
  color: var(--expense);
}

.mom-indicator.down {
  background: var(--income-light);
  color: var(--income);
}

.mom-arrow {
  font-weight: 600;
}

.mom-value {
  font-variant-numeric: tabular-nums;
}

.mom-label {
  font-size: var(--font-size-xs);
  opacity: 0.8;
}

.stat-chip {
  display: flex;
  flex-direction: column;
  padding: var(--space-2) var(--space-3);
  background: var(--surface-2);
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
}

.stat-chip.anomaly {
  background: var(--expense-light);
  border: 1px solid var(--expense);
}

.stat-chip-label {
  color: var(--text-tertiary);
}

.stat-chip-value {
  font-weight: 600;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.chart-container {
  height: 200px;
}
</style>
