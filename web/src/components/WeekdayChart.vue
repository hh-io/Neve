<template>
  <div class="chart-container">
    <v-chart :option="chartOption" autoresize />
  </div>
</template>

<script setup>
import { computed } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { BarChart } from 'echarts/charts';
import { GridComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

use([BarChart, GridComponent, TooltipComponent, CanvasRenderer]);

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const chartOption = computed(() => {
  if (!props.data?.length) {
    return { 
      title: { 
        text: '暂无数据', 
        left: 'center', 
        top: 'center',
        textStyle: { color: 'var(--text-tertiary)', fontSize: 14 }
      } 
    };
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
      },
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
      data: orderedData.map(d => d.name),
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
      type: 'bar',
      data: orderedData.map(d => d.amount),
      itemStyle: {
        color: (params) => {
          // Weekend highlight with warning color
          const name = orderedData[params.dataIndex]?.name;
          return (name === '周六' || name === '周日') ? '#C9A856' : '#5B9A9A';
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
  height: 200px;
}
</style>
