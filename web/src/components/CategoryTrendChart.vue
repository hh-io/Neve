<template>
  <div>
    <div style="display: flex; justify-content: flex-end; margin-bottom: var(--space-3);">
      <select v-model="selectedCategory" class="category-select">
        <option v-for="cat in categories" :key="cat" :value="cat">
          {{ getCategoryLabel(cat) }}
        </option>
      </select>
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
import { GridComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

use([LineChart, GridComponent, TooltipComponent, CanvasRenderer]);

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const categories = computed(() => props.data.map(d => d.category));
const selectedCategory = ref('');

// Category name mapping
function getCategoryLabel(cat) {
  const labelMap = {
    Shopping: '购物',
    Food: '餐饮',
    Transport: '交通',
    Entertainment: '娱乐',
    Gift: '红包/礼物',
    Financial: '金融',
    Communication: '通讯',
    Lodging: '住宿',
    Digital: '数码',
    Unknown: '其他',
  };
  return labelMap[cat] || cat;
}

watch(() => props.data, (newData) => {
  if (newData?.length && !selectedCategory.value) {
    selectedCategory.value = newData[0].category;
  }
}, { immediate: true });

const selectedData = computed(() => {
  return props.data.find(d => d.category === selectedCategory.value)?.data || [];
});

const chartOption = computed(() => {
  if (!selectedData.value?.length) {
    return { 
      title: { 
        text: '暂无数据', 
        left: 'center', 
        top: 'center',
        textStyle: { color: 'var(--text-tertiary)', fontSize: 14 }
      } 
    };
  }

  const labels = selectedData.value.map(d => d.month.slice(5) + '月');
  const amounts = selectedData.value.map(d => d.amount);

  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params) => `${params[0].name}: ¥${params[0].value.toFixed(2)}`,
      backgroundColor: 'var(--bg-secondary)',
      borderColor: 'var(--border)',
      textStyle: { color: 'var(--text-primary)' }
    },
    grid: {
      left: 10,
      right: 10,
      bottom: 10,
      top: 20,
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: labels,
      axisLine: { lineStyle: { color: 'var(--border)' } },
      axisLabel: { color: 'var(--text-secondary)', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: 'var(--border)', type: 'dashed' } },
      axisLabel: {
        color: 'var(--text-secondary)',
        formatter: (val) => val >= 1000 ? (val / 1000).toFixed(0) + 'k' : val,
      },
    },
    series: [{
      type: 'line',
      data: amounts,
      smooth: true,
      symbol: 'circle',
      symbolSize: 6,
      lineStyle: { width: 2, color: '#C27B7B' },
      itemStyle: { color: '#C27B7B' },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(194, 123, 123, 0.3)' },
            { offset: 1, color: 'rgba(194, 123, 123, 0)' }
          ]
        }
      }
    }],
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

.chart-container {
  height: 200px;
}
</style>
