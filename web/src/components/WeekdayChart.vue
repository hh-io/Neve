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
        const weekdayData = orderedData[d.dataIndex];
        const dates = weekdayData?.dates || [];
        
        // Build title: 周五(12-12, 12-19)
        const datesInTitle = dates.slice(0, 3).join(', ') + (dates.length > 3 ? `...` : '');
        const title = dates.length > 0 ? `${d.name}(${datesInTitle})` : d.name;
        
        // Build category breakdown string
        const categories = weekdayData?.categoryBreakdown || [];
        const categoryLabels = {
          Shopping: '购物', Food: '餐饮', Transport: '交通', Entertainment: '娱乐',
          Gift: '红包/礼物', Financial: '金融', Communication: '通讯', Lodging: '住宿',
          Digital: '数码', Unknown: '其他'
        };
        const catStr = categories.slice(0, 4).map(c => 
          `${categoryLabels[c.category] || c.category}: ${c.count}笔`
        ).join('<br/>');
        
        let result = `<strong>${title}</strong><br/>`;
        result += `消费: ¥${d.value.toFixed(2)}<br/>`;
        result += `共 ${weekdayData?.count || 0} 笔`;
        if (catStr) {
          result += `<br/><span style="color:#888;font-size:11px;">─────</span><br/>${catStr}`;
        }
        return result;
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
