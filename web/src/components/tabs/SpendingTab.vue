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
        <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
          <div style="display: flex; align-items: center; gap: var(--space-3);">
            <div class="stat-icon bg-income-light" style="width: 40px; height: 40px;">
              <span v-html="icons.pieChart" style="stroke: var(--income); width: 20px; height: 20px;"></span>
            </div>
            <span style="font-weight: 600; color: var(--text-primary);">收入来源</span>
          </div>
          <span style="font-size: var(--font-size-sm); color: var(--text-secondary);">本月</span>
        </div>
        <div v-if="analytics.incomeBreakdown?.length > 0" style="height: 280px;">
          <v-chart :option="incomePieOption" autoresize />
        </div>
        <div v-else style="height: 280px; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无收入数据
        </div>
      </div>

    </div>

    <!-- Funds Flow Sankey (Full Width) -->
    <div class="card-static section-mb" style="padding: var(--space-6);">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
        <div style="display: flex; align-items: center; gap: var(--space-3);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.transfer" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">资金流向 (Sankey)</span>
        </div>
      </div>
      <div style="height: 400px;">
        <v-chart v-if="hasSankeyData" :option="sankeyOption" autoresize />
        <div v-else style="height: 100%; display: flex; align-items: center; justify-content: center; color: var(--text-tertiary);">
          暂无足够数据生成流向图
        </div>
      </div>
    </div>

    <!-- Rankings -->
    <div class="grid-1-1 section-mb">
      <!-- ... existing rankings ... -->
      <div class="card-static" style="padding: var(--space-6);">
        <div style="display: flex; align-items: center; gap: var(--space-3); margin-bottom: var(--space-4);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.creditCard" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">支付平台排行</span>
        </div>
        <PlatformRanking v-if="analytics.platformRanking" :data="analytics.platformRanking" />
      </div>

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
import { PieChart, SankeyChart } from 'echarts/charts';
import { TitleComponent, TooltipComponent, LegendComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import PlatformRanking from '../PlatformRanking.vue';
import MerchantRanking from '../MerchantRanking.vue';
import { getCategoryLabel } from '../../composables/useCategories';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { icons } from '../../composables/icons';

use([PieChart, SankeyChart, TitleComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const props = defineProps({
  analytics: { type: Object, required: true }
});

// Sankey:收入来源 → 资金账户 → 支出分类。
// 按 posting 级聚合,一笔交易多条支出腿也能完整呈现;转账只有手续费腿会成为流量。
const sankeyData = computed(() => {
  const transactions = props.analytics.transactions || [];
  if (transactions.length === 0) return { nodes: [], links: [] };

  const nodes = new Set();
  const linkMap = {}; // "Source|Target" -> Value

  const addLink = (source, target, value) => {
    if (value <= 0) return;
    const key = `${source}|${target}`;
    linkMap[key] = (linkMap[key] || 0) + value;
    nodes.add(source);
    nodes.add(target);
  };

  transactions.forEach(tx => {
    if (tx.kind === 'opening') return;

    // 资金账户取第一条 Assets/Liabilities posting 的末段
    let account = 'Unknown';
    for (const p of tx.postings || []) {
      const parts = (p.account || '').split(':');
      if (parts[0] === 'Assets' || parts[0] === 'Liabilities') {
        account = parts[parts.length - 1];
        break;
      }
    }

    for (const p of tx.postings || []) {
      const parts = (p.account || '').split(':');
      if (parts[0] === 'Income' && p.amount < 0) {
        addLink(`Income:${getCategoryLabel(parts[1] || 'Income')}`, `Account:${account}`, -p.amount);
      } else if (parts[0] === 'Expenses' && p.amount > 0) {
        addLink(`Account:${account}`, `Expense:${getCategoryLabel(parts[1] || 'Other')}`, p.amount);
      }
    }
  });

  const layoutNodes = Array.from(nodes).map(name => {
    const [type, label] = name.split(':');
    let color = '#ccc';
    if (type === 'Income') color = '#6B9B7A'; // Green
    if (type === 'Account') color = '#5B9A9A'; // Teal
    if (type === 'Expense') color = '#C27B7B'; // Red
    return { name, value: label, itemStyle: { color } };
  });

  const layoutLinks = Object.keys(linkMap).map(key => {
    const [source, target] = key.split('|');
    return { source, target, value: Number(linkMap[key].toFixed(2)) };
  });

  return { nodes: layoutNodes, links: layoutLinks };
});

const hasSankeyData = computed(() => sankeyData.value.nodes.length > 0 && sankeyData.value.links.length > 0);

const sankeyOption = computed(() => {
  void themeVersion.value;
  return {
    tooltip: {
      trigger: 'item',
      triggerOn: 'mousemove'
    },
    series: [
      {
        type: 'sankey',
        data: sankeyData.value.nodes.map(n => ({ name: n.name, itemStyle: n.itemStyle, label: { formatter: n.value } })),
        links: sankeyData.value.links,
        emphasis: { focus: 'adjacency' },
        nodeMargin: 10,
        lineStyle: { color: 'gradient', curveness: 0.5 },
        label: { color: getThemeColor('--text-primary'), fontSize: 12 }
      }
    ]
  };
});

const expensePieOption = computed(() => {
  void themeVersion.value;
  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: ¥{c} ({d}%)',
      backgroundColor: getThemeColor('--bg-secondary'),
      borderColor: getThemeColor('--border'),
      textStyle: { color: getThemeColor('--text-primary') }
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      textStyle: { color: getThemeColor('--text-secondary'), fontSize: 12 }
    },
    color: ['#5B9A9A', '#6B9B7A', '#C27B7B', '#C9A856', '#7B9BC2', '#9B7BA6'],
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['35%', '50%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 8, borderColor: getThemeColor('--bg-secondary'), borderWidth: 2 },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 'bold' },
        itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.2)' }
      },
      data: props.analytics.expenseByCategory?.slice(0, 6).map(item => ({
        name: getCategoryLabel(item.category),
        value: Math.abs(item.amount)
      })) || []
    }]
  };
});

// 收入来源饼图 - 与支出分类保持一致的样式
const incomePieOption = computed(() => {
  void themeVersion.value;
  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: ¥{c} ({d}%)',
      backgroundColor: getThemeColor('--bg-secondary'),
      borderColor: getThemeColor('--border'),
      textStyle: { color: getThemeColor('--text-primary') }
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      textStyle: { color: getThemeColor('--text-secondary'), fontSize: 12 }
    },
    color: ['#6B9B7A', '#5B9A9A', '#7BC27B', '#9BC27B', '#7B9BC2', '#A6C27B'],
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['35%', '50%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 8, borderColor: getThemeColor('--bg-secondary'), borderWidth: 2 },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 'bold' },
        itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.2)' }
      },
      data: props.analytics.incomeBreakdown?.slice(0, 6).map(item => ({
        name: getCategoryLabel(item.source),
        value: Math.abs(item.amount)
      })) || []
    }]
  };
});
</script>
