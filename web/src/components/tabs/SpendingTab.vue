<template>
  <div class="animate-fade-in-up">
    <div class="grid-2-1 section-mb">
      <!-- Expense Pie Chart -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-expense-light">
              <PieChartIcon :size="20" color="var(--expense)" />
            </div>
            <span class="panel-title">支出分类</span>
          </div>
          <span class="panel-sub">本月</span>
        </div>
        <div v-if="expenseByCategory.length > 0" class="chart-md">
          <v-chart :option="expensePieOption" autoresize />
        </div>
        <div v-else class="chart-md chart-empty">暂无支出数据</div>
      </div>

      <!-- Income Chart -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-income-light">
              <PieChartIcon :size="20" color="var(--income)" />
            </div>
            <span class="panel-title">收入来源</span>
          </div>
          <span class="panel-sub">本月</span>
        </div>
        <div v-if="incomeBreakdown.length > 0" class="chart-md">
          <v-chart :option="incomePieOption" autoresize />
        </div>
        <div v-else class="chart-md chart-empty">暂无收入数据</div>
      </div>
    </div>

    <!-- Funds Flow Sankey (Full Width) -->
    <div class="card-static section-mb panel">
      <div class="panel-head">
        <div class="panel-head-left">
          <div class="panel-icon bg-brand-light">
            <Repeat :size="20" color="var(--accent)" />
          </div>
          <span class="panel-title">资金流向 (Sankey)</span>
        </div>
      </div>
      <div class="chart-lg">
        <v-chart v-if="hasSankeyData" :option="sankeyOption" autoresize />
        <div v-else class="chart-empty chart-empty-full">暂无足够数据生成流向图</div>
      </div>
    </div>

    <!-- Rankings -->
    <div class="grid-1-1 section-mb">
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-brand-light">
              <CreditCard :size="20" color="var(--accent)" />
            </div>
            <span class="panel-title">支付平台排行</span>
          </div>
        </div>
        <PlatformRanking :data="platformRanking" />
      </div>

      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-warning-light">
              <ShoppingBag :size="20" color="var(--warning)" />
            </div>
            <span class="panel-title">商户消费排行</span>
          </div>
        </div>
        <MerchantRanking :data="merchantRanking" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
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
import { PieChart as PieChartIcon, Repeat, CreditCard, ShoppingBag } from '@lucide/vue';
import { useAnalytics } from '../../composables/useAnalytics';

use([PieChart, SankeyChart, TitleComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const { analytics } = useAnalytics();

const expenseByCategory = computed(() => analytics.value?.expenseByCategory || []);
const incomeBreakdown = computed(() => analytics.value?.incomeBreakdown || []);
const platformRanking = computed(() => analytics.value?.platformRanking || []);
const merchantRanking = computed(() => analytics.value?.merchantRanking || []);

// Sankey:收入来源 → 资金账户 → 支出分类。
// 按 posting 级聚合,一笔交易多条支出腿也能完整呈现;转账只有手续费腿会成为流量。
interface SankeyNode {
  name: string;
  label: string;
  type: 'Income' | 'Account' | 'Expense';
}

const sankeyData = computed(() => {
  const transactions = analytics.value?.transactions || [];
  if (transactions.length === 0) return { nodes: [] as SankeyNode[], links: [] as { source: string; target: string; value: number }[] };

  const nodes = new Set<string>();
  const linkMap: Record<string, number> = {}; // "Source|Target" -> Value

  const addLink = (source: string, target: string, value: number) => {
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

  const layoutNodes: SankeyNode[] = Array.from(nodes).map(name => {
    const [type, label] = name.split(':');
    return { name, label, type: type as SankeyNode['type'] };
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
  // 节点按类型上色:收入=income,账户=accent,支出=expense
  const nodeColor: Record<SankeyNode['type'], string> = {
    Income: getThemeColor('--chart-income'),
    Account: getThemeColor('--accent'),
    Expense: getThemeColor('--chart-expense')
  };
  return {
    tooltip: {
      trigger: 'item',
      triggerOn: 'mousemove'
    },
    series: [
      {
        type: 'sankey',
        data: sankeyData.value.nodes.map(n => ({
          name: n.name,
          itemStyle: { color: nodeColor[n.type] },
          label: { formatter: n.label }
        })),
        links: sankeyData.value.links,
        emphasis: { focus: 'adjacency' },
        nodeMargin: 10,
        lineStyle: { color: 'gradient', curveness: 0.5 },
        label: { color: getThemeColor('--text-primary'), fontSize: 12 }
      }
    ]
  };
});

// 支出/收入饼图共用色板与样式
function pieOption(data: { name: string; value: number }[]) {
  const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'].map(getThemeColor);
  return {
    tooltip: {
      trigger: 'item',
      formatter: '{b}: ¥{c} ({d}%)',
      backgroundColor: getThemeColor('--surface-1'),
      borderColor: getThemeColor('--hairline'),
      textStyle: { color: getThemeColor('--text-primary') }
    },
    legend: {
      orient: 'vertical',
      right: 10,
      top: 'center',
      textStyle: { color: getThemeColor('--text-secondary'), fontSize: 12 }
    },
    color: palette,
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      center: ['35%', '50%'],
      avoidLabelOverlap: true,
      itemStyle: { borderRadius: 8, borderColor: getThemeColor('--surface-1'), borderWidth: 2 },
      label: { show: false },
      emphasis: {
        label: { show: true, fontSize: 14, fontWeight: 'bold' },
        itemStyle: { shadowBlur: 10, shadowOffsetX: 0, shadowColor: 'rgba(0, 0, 0, 0.2)' }
      },
      data
    }]
  };
}

const expensePieOption = computed(() => {
  void themeVersion.value;
  return pieOption(
    expenseByCategory.value.slice(0, 6).map(item => ({
      name: getCategoryLabel(item.category),
      value: Math.abs(item.amount)
    }))
  );
});

const incomePieOption = computed(() => {
  void themeVersion.value;
  return pieOption(
    incomeBreakdown.value.slice(0, 6).map(item => ({
      name: getCategoryLabel(item.source),
      value: Math.abs(item.amount)
    }))
  );
});
</script>

<style scoped>
.chart-md {
  height: 280px;
}

.chart-lg {
  height: 400px;
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
</style>
