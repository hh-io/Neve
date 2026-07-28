<template>
  <div class="animate-fade-in-up sp">
    <!-- 支出分类占比 + 收入来源(整页均为本月口径,后端已按当月聚合) -->
    <div class="sp-row">
      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">支出分类占比</h3>
          <span class="sp-head-right">
            <span class="section-sub">本月</span>
            <span class="sp-total tabular-nums">{{ formatMoney(expenseTotal) }}</span>
          </span>
        </div>
        <ExpenseDonut :data="expenseByCategory" />
      </section>

      <section class="section-card">
        <div class="section-head">
          <h3 class="section-title">收入来源</h3>
          <span class="sp-head-right">
            <span class="section-sub">本月</span>
            <span class="sp-total sp-total--income tabular-nums">{{ formatMoney(incomeTotal) }}</span>
          </span>
        </div>
        <div class="section-body">
          <IncomeBreakdownList :data="incomeBreakdown" />
        </div>
      </section>
    </div>

    <!-- 资金流向(全宽:收入→账户→支出三层节点在半宽里标签会互相压) -->
    <section class="section-card">
      <div class="section-head">
        <h3 class="section-title">资金流向</h3>
        <div class="sp-flow-legend">
          <span class="section-sub">本月</span>
          <span class="sp-flow-tag"><span class="sp-flow-dot sp-flow-dot--income"></span>收入</span>
          <span class="sp-flow-tag"><span class="sp-flow-dot sp-flow-dot--account"></span>账户</span>
          <span class="sp-flow-tag"><span class="sp-flow-dot sp-flow-dot--expense"></span>支出</span>
        </div>
      </div>
      <div class="sp-flow-body">
        <v-chart v-if="hasSankeyData" class="sp-sankey" :option="sankeyOption" autoresize />
        <div v-else class="sp-empty sp-empty-flow">本月暂无足够数据生成流向图</div>
      </div>
    </section>

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
import { SankeyChart } from 'echarts/charts';
import { TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import PlatformRanking from '../PlatformRanking.vue';
import MerchantRanking from '../MerchantRanking.vue';
import ExpenseDonut from '../ExpenseDonut.vue';
import IncomeBreakdownList from '../IncomeBreakdownList.vue';
import { getCategoryLabel, isCurrentMonth } from '../../composables/useCategories';
import { formatMoney } from '../../composables/useFormatters';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { useAnalytics } from '../../composables/useAnalytics';

use([SankeyChart, TooltipComponent, CanvasRenderer]);

const { analytics } = useAnalytics();

const expenseByCategory = computed(() => analytics.value?.expenseByCategory || []);
const incomeBreakdown = computed(() => analytics.value?.incomeBreakdown || []);
const platformRanking = computed(() => analytics.value?.platformRanking || []);
const merchantRanking = computed(() => analytics.value?.merchantRanking || []);
const expenseTotal = computed(() => expenseByCategory.value.reduce((sum, c) => sum + c.amount, 0));
const incomeTotal = computed(() => incomeBreakdown.value.reduce((sum, s) => sum + s.amount, 0));

// Sankey:收入来源 → 资金账户 → 支出分类。
// 按 posting 级聚合,一笔交易多条支出腿也能完整呈现;转账只有手续费腿会成为流量。
// 本月过滤在前端做:transactions 是全量下发的,而本页其余卡片都是后端的本月口径,
// 并排放着必须同期。
interface SankeyNode {
  name: string;
  label: string;
  type: 'Income' | 'Account' | 'Expense';
}

const sankeyData = computed(() => {
  const transactions = (analytics.value?.transactions || []).filter(tx => isCurrentMonth(tx.date));
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
        // sankey 的节点间距是 nodeGap(没有 nodeMargin 这个配置项);
        // 留够间距,否则金额小的节点高度趋近 0,标签会互相压住
        nodeGap: 14,
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

.sp-head-right {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
}

.sp-total {
  font-size: var(--font-size-sm);
  color: var(--expense);
  font-variant-numeric: tabular-nums;
}

.sp-total--income { color: var(--income); }

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

.sp-flow-dot--income {
  background: var(--income);
}

.sp-flow-dot--account {
  background: var(--accent);
}

.sp-flow-dot--expense {
  background: var(--expense);
}

.sp-flow-body {
  padding: var(--space-4) var(--space-3) var(--space-4) var(--space-5);
}

.sp-sankey {
  height: 300px;
}

.sp-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
  height: 150px;
}

.sp-empty-flow { height: 300px; }

@media (max-width: 1024px) {
  .sp-row { grid-template-columns: 1fr; }
}
</style>
