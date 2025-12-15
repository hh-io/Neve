<template>
  <div class="animate-fade-in-up">
    <div class="grid-2-1 section-mb">
      <!-- Expense Pie Chart -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
          <div style="display: flex; align-items: center; gap: var(--space-3);">
            <div class="stat-icon bg-expense-light" style="width: 40px; height: 40px;">
              <span v-html="icons.pieChart" style="stroke: var(--expense); width: 20px; height: 20px;"></span>
            </div>
            <span style="font-weight: 600; color: var(--text-primary);">支出分类</span>
          </div>
          <span style="font-size: var(--font-size-sm); color: var(--text-secondary);">本月</span>
        </div>
        <div v-if="analytics.expenseByCategory" style="height: 280px;">
          <v-chart :option="expensePieOption" autoresize />
        </div>
      </div>

      <!-- Income Chart -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-income-light" style="width: 40px; height: 40px;">
            <span v-html="icons.pieChart" style="stroke: var(--income); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">收入来源</span>
        </div>
        <IncomeChart v-if="analytics.incomeBreakdown" :data="analytics.incomeBreakdown" />
        <div v-else style="height: 200px; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无收入数据
        </div>
      </div>
    </div>

    <!-- Rankings -->
    <div class="grid-1-1 section-mb">
      <!-- Platform Ranking -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.creditCard" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">支付平台排行</span>
        </div>
        <PlatformRanking v-if="analytics.platformRanking" :data="analytics.platformRanking" />
      </div>

      <!-- Merchant Ranking -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-warning-light" style="width: 40px; height: 40px;">
            <span v-html="icons.shopping" style="stroke: var(--warning); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">商户消费排行</span>
        </div>
        <MerchantRanking v-if="analytics.merchantRanking" :data="analytics.merchantRanking" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { PieChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import IncomeChart from '../IncomeChart.vue';
import PlatformRanking from '../PlatformRanking.vue';
import MerchantRanking from '../MerchantRanking.vue';
import { icons } from '../../composables/icons';

use([PieChart, TitleComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const props = defineProps({
  analytics: { type: Object, required: true }
});

const expensePieOption = computed(() => ({
  tooltip: {
    trigger: 'item',
    formatter: '{b}: ¥{c} ({d}%)',
    backgroundColor: 'var(--bg-secondary)',
    borderColor: 'var(--border)',
    textStyle: { color: 'var(--text-primary)' }
  },
  legend: {
    orient: 'vertical',
    right: 10,
    top: 'center',
    textStyle: { color: 'var(--text-secondary)', fontSize: 12 }
  },
  color: ['#5B9A9A', '#6B9B7A', '#C27B7B', '#C9A856', '#7B9BC2', '#9B7BA6'],
  series: [{
    type: 'pie',
    radius: ['45%', '70%'],
    center: ['35%', '50%'],
    avoidLabelOverlap: true,
    itemStyle: { borderRadius: 8, borderColor: 'var(--bg-secondary)', borderWidth: 2 },
    label: { show: false },
    emphasis: {
      label: { show: true, fontSize: 14, fontWeight: 'bold' },
      itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.2)' }
    },
    data: props.analytics.expenseByCategory?.slice(0, 6).map(item => ({
      name: item.category,
      value: Math.abs(item.amount)
    })) || []
  }]
}));
</script>
