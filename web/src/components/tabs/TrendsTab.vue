<template>
  <div class="animate-fade-in-up tr">
    <!-- 月度收支走势 -->
    <section class="section-card">
      <div class="section-head">
        <h3 class="section-title">收支走势</h3>
        <div class="filter-pills">
          <button
            v-for="period in periods"
            :key="period.value"
            class="filter-pill"
            :class="{ active: selectedPeriod === period.value }"
            @click="selectedPeriod = period.value"
          >
            {{ period.label }}
          </button>
        </div>
      </div>
      <div class="section-body tr-trend-body">
        <v-chart v-if="trendData && trendData.length > 0" class="tr-trend" :option="trendChartOption" autoresize />
        <div v-else class="tr-empty tr-empty-trend">暂无趋势数据</div>
      </div>
    </section>

    <!-- 分类消费环比 + 按星期几分布 -->
    <div class="tr-row">
      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">分类消费环比</h3>
        </div>
        <div class="section-body">
          <CategoryTrend v-if="categoryTrends.length > 0" :data="categoryTrends" />
          <div v-else class="tr-empty tr-empty-sm">暂无分类趋势数据</div>
        </div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">按星期几的消费分布</h3>
        </div>
        <div class="section-body">
          <WeekdayChart v-if="weekdayDistribution.length > 0" :data="weekdayDistribution" />
          <div v-else class="tr-empty tr-empty-sm">暂无周消费数据</div>
        </div>
      </section>
    </div>

    <!-- 年度报告排行 -->
    <div class="tr-row">
      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title"><Trophy :size="16" class="sec-ic" />年度“剁手”商户 Top 5</h3>
        </div>
        <div class="section-body">
          <div v-if="topPayees.length > 0">
            <div v-for="item in topPayees" :key="item.name" class="tr-report-row">
              <div class="tr-report-left">
                <span class="tr-report-rank">{{ item.rank }}</span>
                <span class="tr-report-name">{{ item.name }}</span>
              </div>
              <span class="tr-report-value tabular-nums">¥{{ Number(item.amount).toFixed(2) }}</span>
            </div>
          </div>
          <div v-else class="tr-empty tr-empty-sm">暂无商户排行数据</div>
        </div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title"><Tag :size="16" class="sec-ic" />高频生活标签 Top 5</h3>
        </div>
        <div class="section-body">
          <div v-if="topTags.length > 0">
            <div v-for="item in topTags" :key="item.name" class="tr-report-row">
              <div class="tr-report-left">
                <span class="tr-report-rank">{{ item.rank }}</span>
                <span class="tr-report-name">#{{ item.name }}</span>
              </div>
              <span class="tr-report-value tabular-nums">{{ item.value }}</span>
            </div>
          </div>
          <div v-else class="tr-empty tr-empty-sm">暂无标签排行数据</div>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import WeekdayChart from '../WeekdayChart.vue';
import CategoryTrend from '../CategoryTrendChart.vue';
import type { DailyData, WeeklyData, MonthlyData } from '../../types/api';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { Tag, Trophy } from '@lucide/vue';
import { useAnalytics } from '../../composables/useAnalytics';

use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const { analytics } = useAnalytics();

const dailyTrend = computed(() => analytics.value?.dailyTrend || []);
const weekdayDistribution = computed(() => analytics.value?.weekdayDistribution || []);
const categoryTrends = computed(() => analytics.value?.categoryTrends || []);

type Period = 'day' | 'week' | 'month';
const periods: { value: Period; label: string }[] = [
  { value: 'day', label: '日' },
  { value: 'week', label: '周' },
  { value: 'month', label: '月' },
];
const selectedPeriod = ref<Period>('day');

// Get the appropriate trend data based on selected period
const trendData = computed<(DailyData | WeeklyData | MonthlyData)[]>(() => {
  switch (selectedPeriod.value) {
    case 'day':
      return dailyTrend.value;
    case 'week':
      return analytics.value?.weeklyTrend || [];
    case 'month':
      return analytics.value?.monthlyTrend || [];
    default:
      return analytics.value?.monthlyTrend || [];
  }
});

// Format x-axis label based on period
const formatLabel = (item: Partial<DailyData & WeeklyData & MonthlyData>): string => {
  if (selectedPeriod.value === 'day') {
    return item.date?.slice(5) || '';
  } else if (selectedPeriod.value === 'week') {
    // 后端输出 ISO 周("2026-W28")+ 周一日期,x 轴用周一日期更直观
    return item.weekStart?.slice(5) || item.week || '';
  } else {
    return item.month?.slice(5) || '';
  }
};

const trendChartOption = computed(() => {
  void themeVersion.value;
  const data = trendData.value;
  const incomeColor = getThemeColor('--chart-income');
  const expenseColor = getThemeColor('--chart-expense');
  const axisColor = getThemeColor('--text-tertiary');
  // 面积渐变:同色低透明度 → 全透明(canvas 需 rgba/hex8,由主题实色 + alpha 拼出)
  const areaGradient = (color: string) => ({
    type: 'linear' as const, x: 0, y: 0, x2: 0, y2: 1,
    colorStops: [{ offset: 0, color: color + '4D' }, { offset: 1, color: color + '00' }]
  });
  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: getThemeColor('--surface-1'),
      borderColor: getThemeColor('--hairline'),
      textStyle: { color: getThemeColor('--text-primary') }
    },
    legend: {
      data: ['收入', '支出'],
      top: 0,
      right: 0,
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      textStyle: { color: axisColor, fontSize: 12 }
    },
    grid: { left: 50, right: 20, top: 30, bottom: 30 },
    xAxis: {
      type: 'category',
      data: data.map(d => formatLabel(d)),
      axisLine: { lineStyle: { color: getThemeColor('--hairline') } },
      axisLabel: {
        color: axisColor,
        rotate: selectedPeriod.value === 'day' ? 45 : 0,
        fontSize: selectedPeriod.value === 'day' ? 10 : 12
      }
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: { lineStyle: { color: getThemeColor('--hairline'), type: 'dashed' } },
      axisLabel: { color: axisColor, formatter: (v: number) => v >= 1000 ? `¥${(v / 1000).toFixed(0)}k` : `¥${v}` }
    },
    series: [
      {
        name: '收入',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: selectedPeriod.value === 'day' ? 4 : 6,
        lineStyle: { color: incomeColor, width: 2 },
        itemStyle: { color: incomeColor },
        areaStyle: { color: areaGradient(incomeColor) },
        data: data.map(d => d.income)
      },
      {
        name: '支出',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: selectedPeriod.value === 'day' ? 4 : 6,
        lineStyle: { color: expenseColor, width: 2 },
        itemStyle: { color: expenseColor },
        areaStyle: { color: areaGradient(expenseColor) },
        data: data.map(d => Math.abs(d.expense))
      }
    ]
  };
});

// Ranking:直接消费后端全量口径的排行,避免与 SpendingTab 的排行数据打架
const topPayees = computed(() =>
  (analytics.value?.merchantRanking || [])
    .slice(0, 5)
    .map((item, index) => ({ rank: index + 1, name: item.payee, amount: item.amount }))
);

const topTags = computed(() =>
  (analytics.value?.platformRanking || [])
    .slice()
    .sort((a, b) => b.count - a.count)
    .slice(0, 5)
    .map((item, index) => ({ rank: index + 1, name: item.tag, value: `${item.count}次` }))
);
</script>

<style scoped>
.tr {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.tr-row {
  display: grid;
  grid-template-columns: 1.4fr 1fr;
  gap: var(--space-4);
}

.tr-trend-body { padding: var(--space-5); }
.tr-trend { height: 300px; }

.tr-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

.tr-empty-trend { height: 300px; }
.tr-empty-sm { height: 190px; }

/* 年度报告排行行 */
.tr-report-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--hairline);
}

.tr-report-row:last-child { border-bottom: none; }

.tr-report-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.tr-report-rank {
  width: 18px;
  text-align: right;
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
}

.tr-report-name {
  color: var(--text-primary);
  font-weight: 550;
}

.tr-report-value {
  font-weight: 600;
  color: var(--text-primary);
}

@media (max-width: 1024px) {
  .tr-row { grid-template-columns: 1fr; }
}
</style>
