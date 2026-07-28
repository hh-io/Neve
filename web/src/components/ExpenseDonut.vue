<template>
  <div v-if="items.length > 0" class="donut">
    <div class="donut-chart">
      <v-chart :option="option" autoresize />
    </div>
    <div class="donut-legend">
      <div v-for="item in items" :key="item.category" class="donut-legend-row">
        <span class="donut-dot" :style="{ background: item.color }"></span>
        <span class="donut-name">{{ item.name }}</span>
        <span class="donut-amt tabular-nums">{{ item.amount }}</span>
        <span v-if="showPercent" class="donut-pct tabular-nums">{{ item.pct }}</span>
      </div>
    </div>
  </div>
  <div v-else class="donut-empty">{{ emptyText }}</div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { use } from 'echarts/core';
import { PieChart } from 'echarts/charts';
import { TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import VChart from 'vue-echarts';
import type { CategoryAmount } from '../types/api';
import { formatMoney } from '../composables/useFormatters';
import { getCategoryLabel } from '../composables/useCategories';
import { getThemeColor, themeVersion } from '../composables/useThemeColor';

use([PieChart, TooltipComponent, CanvasRenderer]);

// 概览页与收支分析页共用同一份支出占比图:两处口径本就相同(本月),
// 各写一遍只会让图例截断规则、配色顺序在改动中悄悄分叉
const props = withDefaults(
  defineProps<{
    data: CategoryAmount[];
    /** 图例是否带百分比列 */
    showPercent?: boolean;
    emptyText?: string;
  }>(),
  { showPercent: false, emptyText: '暂无支出数据' },
);

const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'];

// 自定义图例(替代 echarts 内建图例,避免中文截断);只取前 7 类,长尾并入图外
const items = computed(() => {
  void themeVersion.value;
  const colors = palette.map(getThemeColor);
  const total = props.data.reduce((sum, c) => sum + c.amount, 0) || 1;
  return [...props.data]
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 7)
    .map((item, index) => ({
      category: item.category,
      name: getCategoryLabel(item.category),
      color: colors[index % colors.length],
      amount: formatMoney(item.amount),
      pct: `${Math.round((item.amount / total) * 100)}%`,
      value: item.amount,
    }));
});

const option = computed(() => {
  void themeVersion.value;
  return {
    tooltip: { trigger: 'item', formatter: '{b}: ¥{c} ({d}%)' },
    series: [{
      type: 'pie',
      radius: ['58%', '82%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 4, borderColor: getThemeColor('--surface-1'), borderWidth: 2 },
      label: { show: false },
      labelLine: { show: false },
      emphasis: { label: { show: true, fontSize: 14, fontWeight: 'bold', color: getThemeColor('--text-primary') } },
      data: items.value.map(item => ({
        name: item.name,
        value: item.value,
        itemStyle: { color: item.color },
      })),
    }],
  };
});
</script>

<style scoped>
.donut {
  padding: var(--space-4) var(--space-5) var(--space-5);
  display: flex;
  gap: var(--space-5);
  align-items: center;
  flex-wrap: wrap;
}

.donut-chart {
  width: 150px;
  height: 150px;
  flex: none;
  margin: 0 auto;
}

.donut-legend {
  flex: 1;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.donut-legend-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.donut-dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
  flex: none;
}

.donut-name {
  flex: 1;
  min-width: 0;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.donut-amt { color: var(--text-primary); }

.donut-pct {
  width: 42px;
  text-align: right;
  color: var(--text-tertiary);
}

.donut-empty {
  height: 150px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}
</style>
