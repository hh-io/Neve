<template>
  <div class="animate-fade-in-up sp">
    <!-- 支出分类占比 + 资金流向 -->
    <div class="sp-row">
      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">支出分类占比</h3>
          <span class="sp-total tabular-nums">¥{{ formatMoney(expenseTotal) }}</span>
        </div>
        <div v-if="expenseByCategory.length > 0" class="sp-donut-body">
          <div class="sp-donut-chart">
            <v-chart :option="expensePieOption" autoresize />
          </div>
          <div class="sp-legend">
            <div v-for="item in legendItems" :key="item.category" class="sp-legend-row">
              <span class="sp-legend-dot" :style="{ background: item.color }"></span>
              <span class="sp-legend-name">{{ item.name }}</span>
              <span class="sp-legend-amt tabular-nums">{{ item.amount }}</span>
            </div>
          </div>
        </div>
        <div v-else class="sp-empty">暂无支出数据</div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">资金流向</h3>
          <div class="sp-flow-legend">
            <span class="sp-flow-tag"><span class="sp-flow-dot" style="background: var(--income)"></span>收入</span>
            <span class="sp-flow-tag"><span class="sp-flow-dot" style="background: var(--accent)"></span>账户</span>
            <span class="sp-flow-tag"><span class="sp-flow-dot" style="background: var(--expense)"></span>支出</span>
          </div>
        </div>
        <div class="sp-flow-body">
          <v-chart v-if="hasSankeyData" class="sp-sankey" :option="sankeyOption" autoresize />
          <div v-else class="sp-empty sp-empty-flow">暂无足够数据生成流向图</div>
        </div>
      </section>
    </div>

    <!-- 支付平台排行 + 商户消费排行 -->
    <div class="sp-row">
      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">支付平台排行</h3>
          <span class="section-sub">按本月支出</span>
        </div>
        <div class="section-body">
          <PlatformRanking :data="platformRanking" />
        </div>
      </section>

      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">商户消费排行</h3>
          <span class="section-sub">按本月支出</span>
        </div>
        <div class="section-body">
          <MerchantRanking :data="merchantRanking" />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { PieChart, SankeyChart } from 'echarts/charts';
import { TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import PlatformRanking from '../PlatformRanking.vue';
import MerchantRanking from '../MerchantRanking.vue';
import { getCategoryLabel } from '../../composables/useCategories';
import { formatMoney } from '../../composables/useFormatters';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { useAnalytics } from '../../composables/useAnalytics';

use([PieChart, SankeyChart, TooltipComponent, CanvasRenderer]);

const { analytics } = useAnalytics();

const expenseByCategory = computed(() => analytics.value?.expenseByCategory || []);
const platformRanking = computed(() => analytics.value?.platformRanking || []);
const merchantRanking = computed(() => analytics.value?.merchantRanking || []);
const expenseTotal = computed(() => expenseByCategory.value.reduce((sum, c) => sum + c.amount, 0));

const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'];

// 自定义图例(替代 echarts 内建图例,避免中文截断)
const legendItems = computed(() => {
  void themeVersion.value;
  const colors = palette.map(getThemeColor);
  return [...expenseByCategory.value]
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 7)
    .map((item, index) => ({
      category: item.category,
      name: getCategoryLabel(item.category),
      color: colors[index % colors.length],
      amount: formatMoney(item.amount),
    }));
});

const expensePieOption = computed(() => {
  void themeVersion.value;
  const colors = palette.map(getThemeColor);
  const data = [...expenseByCategory.value]
    .sort((a, b) => b.amount - a.amount)
    .slice(0, 7)
    .map((item, index) => ({
      name: getCategoryLabel(item.category),
      value: item.amount,
      itemStyle: { color: colors[index % colors.length] },
    }));
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
      data,
    }],
  };
});

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
        top: 8,
        bottom: 8,
        left: 4,
        right: 90,
        data: sankeyData.value.nodes.map(n => ({
          name: n.name,
          itemStyle: { color: nodeColor[n.type] },
          label: { formatter: n.label }
        })),
        links: sankeyData.value.links,
        emphasis: { focus: 'adjacency' },
        nodeMargin: 8,
        nodeWidth: 12,
        lineStyle: { color: 'gradient', curveness: 0.5 },
        label: { color: getThemeColor('--text-primary'), fontSize: 11 }
      }
    ]
  };
});
</script>

<style scoped>
.sp {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.sp-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.sp-total {
  font-size: var(--font-size-sm);
  color: var(--expense);
  font-variant-numeric: tabular-nums;
}

/* ===== 支出占比甜甜圈 + 图例 ===== */
.sp-donut-body {
  padding: var(--space-4) var(--space-5) var(--space-5);
  display: flex;
  gap: var(--space-5);
  align-items: center;
}

.sp-donut-chart {
  width: 150px;
  height: 150px;
  flex: none;
}

.sp-legend {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.sp-legend-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
}

.sp-legend-dot {
  width: 9px;
  height: 9px;
  border-radius: 3px;
  flex: none;
}

.sp-legend-name {
  flex: 1;
  min-width: 0;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sp-legend-amt { color: var(--text-primary); }

/* ===== 资金流向 ===== */
.sp-flow-legend {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.sp-flow-tag {
  display: flex;
  align-items: center;
  gap: 4px;
}

.sp-flow-dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;
}

.sp-flow-body {
  padding: var(--space-4) var(--space-3) var(--space-4) var(--space-5);
}

.sp-sankey {
  height: 224px;
}

.sp-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  height: 150px;
}

.sp-empty-flow { height: 224px; }

@media (max-width: 1024px) {
  .sp-row { grid-template-columns: 1fr; }
}
</style>
