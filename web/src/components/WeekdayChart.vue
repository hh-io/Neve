<template>
  <div class="glass-card chart-card">
    <h3 class="card-title">消费时间分布</h3>
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

  // Reorder: Monday first
  const orderedData = [
    ...props.data.slice(1), // Mon-Sat
    props.data[0]           // Sunday at end
  ];

  return {
    tooltip: {
      trigger: 'axis',
      formatter: (params) => {
        const d = params[0];
        return `${d.name}<br/>消费: ¥${d.value.toFixed(2)}<br/>笔数: ${orderedData[d.dataIndex]?.count || 0}`;
      }
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
      data: orderedData.map(d => d.name),
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
      type: 'bar',
      data: orderedData.map(d => d.amount),
      itemStyle: {
        color: (params) => {
          // Weekend highlight
          const name = orderedData[params.dataIndex]?.name;
          return (name === '周六' || name === '周日') ? '#FF9500' : '#5856D6';
        },
        borderRadius: [4, 4, 0, 0],
      },
      barWidth: '50%',
    }],
  };
});
</script>

<style scoped>
.chart-container {
  height: 220px;
}
</style>
