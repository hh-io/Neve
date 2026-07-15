<template>
  <div>
    <!-- Header with category selector and stats -->
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-3); flex-wrap: wrap; gap: var(--space-2);">
      <div style="display: flex; align-items: center; gap: var(--space-2);">
        <select v-model="selectedCategory" class="category-select">
          <option v-for="cat in categories" :key="cat" :value="cat">
            {{ getCategoryLabel(cat) }}
          </option>
        </select>
        <button 
          v-if="!compareMode" 
          @click="compareMode = true" 
          class="compare-btn"
          title="对比模式"
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
    <div style="display: flex; gap: var(--space-4); margin-bottom: var(--space-3); flex-wrap: wrap;">
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

<script setup>
import { ref, computed, watch } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, MarkLineComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { getThemeColor, themeVersion } from '../composables/useThemeColor';
import { getCategoryLabel } from '../composables/useCategories';

use([LineChart, GridComponent, TooltipComponent, MarkLineComponent, CanvasRenderer]);

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const categories = computed(() => props.data.map(d => d.category));
const selectedCategory = ref('');
const compareCategory = ref('');
const compareMode = ref(false);

// Watch compareCategory to exit compare mode
watch(compareCategory, (val) => {
  if (!val) compareMode.value = false;
});

function formatNum(val) {
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
  themeVersion.value;
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

  const series = [{
    name: getCategoryLabel(selectedCategory.value),
    type: 'line',
    data: amounts,
    smooth: true,
    symbol: 'circle',
    symbolSize: (value) => value > avg * 2 ? 10 : 6, // Larger symbol for anomalies
    lineStyle: { width: 2, color: getThemeColor('--expense') || '#F87171' },
    itemStyle: { 
      color: (params) => params.value > avg * 2 ? getThemeColor('--expense') || '#F87171' : getThemeColor('--expense') || '#F87171'
    },
    areaStyle: {
      color: {
        type: 'linear',
        x: 0, y: 0, x2: 0, y2: 1,
        colorStops: [
          { offset: 0, color: getThemeColor('--expense-light') || 'rgba(248, 113, 113, 0.15)' },
          { offset: 1, color: 'rgba(0,0,0,0)' }
        ]
      }
    },
    markLine: {
      silent: true,
      symbol: 'none',
      lineStyle: { color: getThemeColor('--text-tertiary') || '#94A3B8', type: 'dashed', width: 1 },
      data: [{ yAxis: avg, label: { formatter: '均值', position: 'end', fontSize: 10, color: getThemeColor('--text-tertiary') || '#94A3B8' } }]
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
      lineStyle: { width: 2, color: getThemeColor('--income') || '#34D399' },
      itemStyle: { color: getThemeColor('--income') || '#34D399' },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: getThemeColor('--income-light') || 'rgba(52, 211, 153, 0.15)' },
            { offset: 1, color: 'rgba(0,0,0,0)' }
          ]
        }
      }
    });
  }

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: getThemeColor('--bg-secondary'),
      borderColor: getThemeColor('--border'),
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
      axisLine: { lineStyle: { color: getThemeColor('--border') } },
      axisLabel: { color: getThemeColor('--text-secondary'), fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: getThemeColor('--border'), type: 'dashed' } },
      axisLabel: {
        color: getThemeColor('--text-secondary'),
        formatter: (val) => val >= 1000 ? (val / 1000).toFixed(0) + 'k' : val,
      },
    },
    series,
  };
});
</script>

<style scoped>
.category-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  cursor: pointer;
  outline: none;
  transition: border-color var(--transition-base);
}

.category-select:hover {
  border-color: var(--border-hover);
}

.category-select:focus {
  border-color: var(--brand-primary);
}

.compare-btn {
  padding: var(--space-1) var(--space-2);
  border: 1px dashed var(--border);
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-base);
}

.compare-btn:hover {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
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
  background: rgba(255, 107, 107, 0.1);
  color: #FF6B6B;
}

.mom-indicator.down {
  background: rgba(107, 155, 122, 0.1);
  color: #6B9B7A;
}

.mom-arrow {
  font-weight: 600;
}

.mom-label {
  font-size: var(--font-size-xs);
  opacity: 0.8;
}

.stat-chip {
  display: flex;
  flex-direction: column;
  padding: var(--space-2) var(--space-3);
  background: var(--bg-tertiary);
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
}

.stat-chip.anomaly {
  background: rgba(255, 107, 107, 0.1);
  border: 1px solid rgba(255, 107, 107, 0.3);
}

.stat-chip-label {
  color: var(--text-tertiary);
}

.stat-chip-value {
  font-weight: 600;
  color: var(--text-primary);
}

.chart-container {
  height: 200px;
}
</style>
