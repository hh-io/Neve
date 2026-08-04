<template>
  <div v-if="items.length > 0" class="donut">
    <div class="donut-chart">
      <v-chart ref="chartRef" :option="option" autoresize @mouseover="onChartHover" @mouseout="onChartLeave" @globalout="focus(null)" />
      <!-- 中心态取代 tooltip:环内是现成的空白,浮层反而会盖住扇区与相邻文字 -->
      <div class="donut-center">
        <span class="donut-center-label">
          <span v-if="center.color" class="donut-dot" :style="{ background: center.color }"></span>
          <!-- 文本单独成 flex 项才能省略号:匿名 flex item 上 text-overflow 不生效 -->
          <span class="donut-center-name">{{ center.label }}</span>
        </span>
        <span class="donut-center-value tabular-nums">{{ center.value }}</span>
        <span class="donut-center-sub tabular-nums">{{ center.sub }}</span>
      </div>
    </div>
    <div class="donut-legend">
      <div
        v-for="(item, index) in items"
        :key="item.category"
        class="donut-legend-row"
        :class="{ 'is-active': activeIndex === index }"
        @mouseenter="focus(index)"
        @mouseleave="focus(null)"
      >
        <span class="donut-dot" :style="{ background: item.color }"></span>
        <span class="donut-name">{{ item.name }}</span>
        <span class="donut-amt tabular-nums">{{ item.amount }}</span>
        <span class="donut-pct tabular-nums">{{ item.pct }}</span>
      </div>
    </div>
  </div>
  <div v-else class="donut-empty">{{ emptyText }}</div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { use } from 'echarts/core';
import { PieChart } from 'echarts/charts';
import { CanvasRenderer } from 'echarts/renderers';
import VChart from 'vue-echarts';
import type { CategoryAmount } from '../types/api';
import { formatMoney } from '../composables/useFormatters';
import { getCategoryLabel } from '../composables/useCategories';
import { getThemeColor, themeVersion } from '../composables/useThemeColor';

use([PieChart, CanvasRenderer]);

// 支出构成的唯一视图,只在「收支分析」页出现:概览要的是异动信号而非构成比例,
// 那边走 ExpenseCategoryBoard(金额 + 环比 + 上月基数)。两页给同一个答案时,
// 收支分析页的头一屏等于白给。
const props = withDefaults(
  defineProps<{
    data: CategoryAmount[];
    emptyText?: string;
  }>(),
  { emptyText: '暂无支出数据' },
);

const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'];

const total = computed(() => props.data.reduce((sum, c) => sum + c.amount, 0));

// 自定义图例(替代 echarts 内建图例,避免中文截断);只取前 7 类,长尾并入图外
const items = computed(() => {
  void themeVersion.value;
  const colors = palette.map(getThemeColor);
  const base = total.value || 1;
  return [...props.data]
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 7)
    .map((item, index) => ({
      category: item.category,
      name: getCategoryLabel(item.category),
      color: colors[index % colors.length],
      amount: formatMoney(item.amount),
      pct: `${Math.round((item.amount / base) * 100)}%`,
      value: item.amount,
    }));
});

const chartRef = ref<InstanceType<typeof VChart> | null>(null);
const activeIndex = ref<number | null>(null);

// 图例与扇区双向联动:鼠标在哪一侧都让另一侧跟着高亮,中心文案随之切换
function focus(index: number | null) {
  activeIndex.value = index;
  const chart = chartRef.value;
  if (!chart) return;
  chart.dispatchAction({ type: 'downplay', seriesIndex: 0 });
  if (index !== null) chart.dispatchAction({ type: 'highlight', seriesIndex: 0, dataIndex: index });
}

function onChartHover(params: { dataIndex?: number }) {
  if (typeof params.dataIndex === 'number') activeIndex.value = params.dataIndex;
}

function onChartLeave(params: { dataIndex?: number }) {
  if (typeof params.dataIndex === 'number' && activeIndex.value === params.dataIndex) {
    activeIndex.value = null;
  }
}

const center = computed(() => {
  const item = activeIndex.value === null ? null : items.value[activeIndex.value];
  if (!item) {
    return { label: '合计', value: formatMoney(total.value), sub: `共 ${props.data.length} 类`, color: '' };
  }
  return { label: item.name, value: item.amount, sub: `占比 ${item.pct}`, color: item.color };
});

const option = computed(() => {
  void themeVersion.value;
  return {
    series: [{
      type: 'pie',
      radius: ['62%', '86%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: false,
      silent: false,
      itemStyle: { borderRadius: 4, borderColor: getThemeColor('--surface-1'), borderWidth: 2 },
      // 标签一律交给中心 HTML 层:pie 的 emphasis 标签默认落在扇区外侧,
      // 在这个尺寸的容器里必被裁掉半截
      label: { show: false },
      labelLine: { show: false },
      emphasis: { scale: true, scaleSize: 4 },
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
  /* 与并排卡片等高时(概览的最近交易、收支页的收入来源)吃掉剩余高度,
     内容居中而非全部堆在顶部 */
  flex: 1;
  min-height: 0;
}

.donut-chart {
  position: relative;
  width: 190px;
  height: 190px;
  flex: none;
  margin: 0 auto;
}

.donut-center {
  position: absolute;
  inset: 22%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  text-align: center;
  pointer-events: none;
}

.donut-center-label {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
}

.donut-center-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.donut-center-value {
  font-size: var(--font-size-base);
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--text-primary);
}

.donut-center-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.donut-legend {
  flex: 1;
  min-width: 200px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 2px;
}

.donut-legend-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  transition: background-color var(--transition-base);
}

.donut-legend-row.is-active {
  background: var(--surface-3);
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

@media (max-width: 640px) {
  .donut-chart {
    width: 150px;
    height: 150px;
  }
}
</style>
