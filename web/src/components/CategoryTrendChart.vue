<template>
  <div class="glass-card chart-card">
    <div class="card-header">
      <h3 class="card-title">分类趋势</h3>
      <select v-model="selectedCategory" class="category-select">
        <option v-for="cat in categories" :key="cat" :value="cat">
          {{ cat }}
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

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const categories = computed(() => props.data.map(d => d.category));
const selectedCategory = ref('');

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
    return { title: { text: '暂无数据', left: 'center', top: 'center' } };
  }

  const labels = selectedData.value.map(d => d.month.slice(5) + '月');
  const amounts = selectedData.value.map(d => d.amount);

  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params) => `${params[0].name}: ¥${params[0].value.toFixed(2)}`,
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '10%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: labels,
      axisLine: { lineStyle: { color: '#E5E5EA' } },
      axisLabel: { color: '#6E6E73', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: '#F2F2F7' } },
      axisLabel: {
        color: '#6E6E73',
        formatter: (val) => val >= 1000 ? (val / 1000).toFixed(0) + 'k' : val,
      },
    },
    series: [{
      type: 'line',
      data: amounts,
      smooth: true,
      symbol: 'circle',
      symbolSize: 8,
      lineStyle: { width: 3, color: '#5856D6' },
      itemStyle: { color: '#5856D6', borderWidth: 2, borderColor: '#fff' },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(88, 86, 214, 0.3)' },
            { offset: 1, color: 'rgba(88, 86, 214, 0.05)' }
          ]
        }
      }
    }],
  };
});
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
}

.card-header .card-title {
  margin-bottom: 0;
}

.category-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  background: white;
  cursor: pointer;
}

.chart-container {
  height: 200px;
}
</style>
