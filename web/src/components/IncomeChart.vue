<template>
  <div class="glass-card chart-card">
    <h3 class="card-title">收入来源</h3>
    <div class="chart-container">
      <v-chart :option="chartOption" autoresize />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import VChart from 'vue-echarts';

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const chartOption = computed(() => {
  if (!props.data?.length) {
    return { title: { text: '暂无数据', left: 'center', top: 'center' } };
  }

  const colors = ['#34C759', '#30D158', '#00C7BE', '#5AC8FA', '#007AFF'];

  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: ¥{c} ({d}%)',
    },
    legend: {
      orient: 'vertical',
      right: '5%',
      top: 'center',
      textStyle: { color: '#6E6E73', fontSize: 12 },
    },
    color: colors,
    series: [{
      type: 'pie',
      radius: ['40%', '65%'],
      center: ['35%', '50%'],
      avoidLabelOverlap: true,
      itemStyle: {
        borderRadius: 8,
        borderColor: 'rgba(255,255,255,0.8)',
        borderWidth: 2,
      },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 'bold' },
      },
      data: props.data.map(item => ({
        name: item.source,
        value: Math.round(item.amount * 100) / 100,
      })),
    }],
  };
});
</script>

<style scoped>
.chart-container {
  height: 200px;
}
</style>
