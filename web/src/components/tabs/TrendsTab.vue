<template>
  <div class="animate-fade-in-up">
    <!-- Trend Period Selector -->
    <div class="card-static section-mb period-bar">
      <div class="period-title">
        <div class="panel-icon bg-brand-light period-icon">
          <LineChartIcon :size="18" color="var(--accent)" />
        </div>
        <span class="period-label">趋势周期</span>
      </div>
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

    <!-- Main Trend Chart + Transaction Calendar (Side by Side) -->
    <div class="grid-7-3 section-mb">
      <!-- Main Trend Chart -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-info-light">
              <TrendingUp :size="20" color="var(--info)" />
            </div>
            <span class="panel-title">收支趋势</span>
          </div>
          <div class="series-legend">
            <button
              class="series-toggle"
              :class="{ off: !seriesVisible.income }"
              @click="toggleSeries('income')"
            >
              <span class="series-dot dot-income"></span>
              <span class="series-name">收入</span>
            </button>
            <button
              class="series-toggle"
              :class="{ off: !seriesVisible.expense }"
              @click="toggleSeries('expense')"
            >
              <span class="series-dot dot-expense"></span>
              <span class="series-name">支出</span>
            </button>
          </div>
        </div>
        <div class="chart-xl">
          <v-chart v-if="trendData && trendData.length > 0" :option="trendChartOption" autoresize />
          <div v-else class="chart-empty chart-empty-full">暂无趋势数据</div>
        </div>
      </div>

      <!-- Transaction Calendar -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-brand-light">
              <Calendar :size="20" color="var(--accent)" />
            </div>
            <span class="panel-title">交易日历</span>
          </div>
        </div>
        <TransactionCalendar :dailyData="dailyTrend" />
      </div>
    </div>

    <!-- Expense Heatmap -->
    <div class="card-static section-mb panel">
      <div class="panel-head">
        <div class="panel-head-left">
          <div class="panel-icon bg-warning-light">
            <Calendar :size="20" color="var(--warning)" />
          </div>
          <span class="panel-title">消费日历热力图</span>
        </div>
      </div>
      <div class="chart-heatmap">
        <v-chart v-if="heatmapOption" :option="heatmapOption" autoresize />
        <div v-else class="chart-empty chart-empty-full">暂无足够数据生成热力图</div>
      </div>
    </div>

    <!-- Weekday Distribution -->
    <div class="grid-1-1 section-mb">
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-warning-light">
              <Calendar :size="20" color="var(--warning)" />
            </div>
            <span class="panel-title">周消费分布</span>
          </div>
        </div>
        <WeekdayChart v-if="weekdayDistribution.length > 0" :data="weekdayDistribution" />
        <div v-else class="chart-sm chart-empty">暂无周消费数据</div>
      </div>

      <!-- Category Trend -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-expense-light">
              <Tag :size="20" color="var(--expense)" />
            </div>
            <span class="panel-title">分类消费环比</span>
          </div>
        </div>
        <CategoryTrend v-if="categoryTrends.length > 0" :data="categoryTrends" />
        <div v-else class="chart-sm chart-empty">暂无分类趋势数据</div>
      </div>
    </div>

    <!-- Annual Report Style Rankings -->
    <div class="grid-1-1 section-mb">
      <!-- Top Payees -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-brand-light">
              <Trophy :size="20" color="var(--accent)" />
            </div>
            <span class="panel-title">年度“剁手”商户 Top 5</span>
          </div>
        </div>
        <div v-if="topPayees.length > 0">
          <div v-for="item in topPayees" :key="item.name" class="report-row">
            <div class="report-left">
              <span class="report-rank" :style="{ color: getRankColor(item.rank) }">#{{ item.rank }}</span>
              <span class="report-name">{{ item.name }}</span>
            </div>
            <span class="report-value">¥{{ Number(item.amount).toFixed(2) }}</span>
          </div>
        </div>
        <div v-else class="report-empty">暂无商户排行数据</div>
      </div>

      <!-- Top Tags -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-info-light">
              <Tag :size="20" color="var(--info)" />
            </div>
            <span class="panel-title">高频生活标签 Top 5</span>
          </div>
        </div>
        <div v-if="topTags.length > 0">
          <div v-for="item in topTags" :key="item.name" class="report-row">
            <div class="report-left">
              <span class="report-rank" :style="{ color: getRankColor(item.rank) }">#{{ item.rank }}</span>
              <span class="report-name">#{{ item.name }}</span>
            </div>
            <span class="report-value">{{ item.value }}</span>
          </div>
        </div>
        <div v-else class="report-empty">暂无标签排行数据</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { LineChart, HeatmapChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, LegendComponent, CalendarComponent, VisualMapComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import WeekdayChart from '../WeekdayChart.vue';
import CategoryTrend from '../CategoryTrendChart.vue';
import TransactionCalendar from '../TransactionCalendar.vue';
import type { DailyData, WeeklyData, MonthlyData } from '../../types/api';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { LineChart as LineChartIcon, TrendingUp, Calendar, Tag, Trophy } from '@lucide/vue';
import { useAnalytics } from '../../composables/useAnalytics';

use([LineChart, HeatmapChart, GridComponent, TooltipComponent, LegendComponent, CalendarComponent, VisualMapComponent, CanvasRenderer]);

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

// Series visibility state
const seriesVisible = ref({
  income: true,
  expense: true
});

const toggleSeries = (series: 'income' | 'expense') => {
  seriesVisible.value[series] = !seriesVisible.value[series];
};

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
    grid: { left: 50, right: 20, top: 20, bottom: 30 },
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
        lineStyle: { color: incomeColor, width: 2, opacity: seriesVisible.value.income ? 1 : 0 },
        itemStyle: { color: incomeColor, opacity: seriesVisible.value.income ? 1 : 0 },
        areaStyle: seriesVisible.value.income ? { color: areaGradient(incomeColor) } : { opacity: 0 },
        data: seriesVisible.value.income ? data.map(d => d.income) : data.map(() => null)
      },
      {
        name: '支出',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: selectedPeriod.value === 'day' ? 4 : 6,
        lineStyle: { color: expenseColor, width: 2, opacity: seriesVisible.value.expense ? 1 : 0 },
        itemStyle: { color: expenseColor, opacity: seriesVisible.value.expense ? 1 : 0 },
        areaStyle: seriesVisible.value.expense ? { color: areaGradient(expenseColor) } : { opacity: 0 },
        data: seriesVisible.value.expense ? data.map(d => Math.abs(d.expense)) : data.map(() => null)
      }
    ]
  };
});

// Heatmap Logic
const heatmapOption = computed(() => {
  void themeVersion.value;
  const data = dailyTrend.value;
  // Map to [date, expense]
  const heatmapData = data.map(d => [d.date, Math.abs(d.expense)] as [string, number]);

  if (heatmapData.length === 0) return null;

  const maxExpense = Math.max(...heatmapData.map(d => d[1]));

  return {
    tooltip: {
      formatter: (p: { data: [string, number] }) => {
        return `${p.data[0]}: ¥${p.data[1].toFixed(2)}`;
      },
      backgroundColor: getThemeColor('--surface-1'),
      borderColor: getThemeColor('--hairline'),
      textStyle: { color: getThemeColor('--text-primary') }
    },
    visualMap: {
      min: 0,
      max: maxExpense,
      type: 'continuous',
      orient: 'horizontal',
      left: 'center',
      bottom: 0,
      // 顺序绿渐变走热力专用色阶 token(热力图为顺序标度,不取分类色板;与收入语义绿独立)
      inRange: {
        color: ['--heat-0', '--heat-1', '--heat-2', '--heat-3', '--heat-4'].map(getThemeColor)
      },
      text: ['High', 'Low'],
      calculable: false,
      show: false // hide legend to save space
    },
    calendar: {
      top: 30,
      left: 30,
      right: 30,
      cellSize: ['auto', 16],
      range: new Date().getFullYear(), // Current Year
      itemStyle: {
        color: 'transparent',
        borderColor: getThemeColor('--hairline'),
        borderWidth: 1
      },
      yearLabel: { show: false },
      dayLabel: { nameMap: ['S', 'M', 'T', 'W', 'T', 'F', 'S'], color: getThemeColor('--text-tertiary') },
      monthLabel: { nameMap: 'en', color: getThemeColor('--text-tertiary') },
      splitLine: { show: false }
    },
    series: {
      type: 'heatmap',
      coordinateSystem: 'calendar',
      data: heatmapData,
      itemStyle: {
        borderRadius: 2,
        borderColor: getThemeColor('--surface-1'), // Gap color
        borderWidth: 1
      }
    }
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

function getRankColor(rank: number): string {
  if (rank === 1) return '#FFD700'; // Gold
  if (rank === 2) return '#C0C0C0'; // Silver
  if (rank === 3) return '#CD7F32'; // Bronze
  return 'var(--text-tertiary)';
}
</script>

<style scoped>
/* 周期选择条 */
.period-bar {
  padding: var(--space-4);
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.period-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.period-icon {
  width: 36px;
  height: 36px;
}

.period-label {
  font-weight: 500;
  color: var(--text-primary);
}

/* 收支趋势图例(可切换显隐) */
.series-legend {
  display: flex;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.series-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: var(--surface-2);
  border: none;
  cursor: pointer;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  transition: opacity var(--transition-base), background var(--transition-base);
}

.series-toggle.off {
  opacity: 0.4;
  background: transparent;
}

.series-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.dot-income {
  background: var(--income);
}

.dot-expense {
  background: var(--expense);
}

/* 图表容器高度 */
.chart-xl {
  height: 380px;
}

.chart-heatmap {
  height: 180px;
}

.chart-sm {
  height: 200px;
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

.chart-empty-full {
  height: 100%;
}

/* 年度报告排行行 */
.report-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) 0;
  border-bottom: 1px solid var(--hairline);
}

.report-left {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.report-rank {
  font-weight: bold;
  min-width: 20px;
}

.report-name {
  color: var(--text-primary);
  font-weight: 500;
}

.report-value {
  font-weight: 600;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
}

.report-empty {
  padding: var(--space-4);
  text-align: center;
  color: var(--text-tertiary);
}
</style>
