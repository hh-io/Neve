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
import { getThemeColor, themeVersion } from '../composables/useThemeColor';
import { getCategoryLabel } from '../composables/useCategories';

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const chartOption = computed(() => {
  void themeVersion.value;
  if (!props.data?.length) {
    return {
      title: {
        text: '暂无数据',
        left: 'center',
        top: 'center',
        textStyle: { color: getThemeColor('--text-tertiary'), fontSize: 14 }
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
        const catStr = categories.slice(0, 4).map(c =>
          `${getCategoryLabel(c.category)}: ${c.count}笔`
        ).join('<br/>');
        
        let result = `<strong>${title}</strong><br/>`;
        result += `消费: ¥${d.value.toFixed(2)}<br/>`;
        result += `共 ${weekdayData?.count || 0} 笔`;
        if (catStr) {
          result += `<br/><span style="color:#888;font-size:11px;">─────</span><br/>${catStr}`;
        }
        return result;
      },
      backgroundColor: getThemeColor('--bg-secondary'),
      borderColor: getThemeColor('--border'),
      textStyle: { color: getThemeColor('--text-primary') }
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
    series: [{
      type: 'bar',
      data: orderedData.map(d => d.amount),
      itemStyle: {
        color: (params) => {
          const name = orderedData[params.dataIndex]?.name;
          return (name === '周六' || name === '周日') ? (getThemeColor('--warning') || '#FBBF24') : (getThemeColor('--brand-primary') || '#6366F1');
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
