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
        <!-- 高度是数据驱动的运行时值(节点越多越高),没法交给 token -->
        <v-chart
          v-if="hasSankeyData"
          class="sp-sankey"
          :style="{ height: `${sankeyHeight}px` }"
          :option="sankeyOption"
          autoresize
        />
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
import type { FundFlowNode } from '../../types/api';
import { getCategoryLabel } from '../../composables/useCategories';
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
// 三层聚合全部由后端 computeFundFlow 算好(净额口径、本月过滤、账户消歧、稳定排序),
// 前端只做展示映射:自己遍历 transactions 等于把口径重写一遍,和同页的分类占比一定会漂移。
const fundFlow = computed(() => analytics.value?.fundFlow ?? { nodes: [], links: [] });
const hasSankeyData = computed(() => fundFlow.value.links.length > 0);

// 账户名后端已消歧,分类/来源是原始键,展示时才转中文
function nodeLabel(node: FundFlowNode): string {
  return node.type === 'account' ? node.label : getCategoryLabel(node.label);
}

const nodeLabels = computed(() => new Map(fundFlow.value.nodes.map(n => [n.key, nodeLabel(n)])));

// 节点吞吐自己算:sankey 内建的节点 value 在 tooltip 里不稳定,而这个数正是「这个账户
// 本月进/出了多少」,悬停时最该看到的就是它
const nodeThroughput = computed(() => {
  const inflow = new Map<string, number>();
  const outflow = new Map<string, number>();
  for (const link of fundFlow.value.links) {
    outflow.set(link.source, (outflow.get(link.source) || 0) + link.amount);
    inflow.set(link.target, (inflow.get(link.target) || 0) + link.amount);
  }
  const total = new Map<string, number>();
  for (const node of fundFlow.value.nodes) {
    total.set(node.key, Math.max(inflow.get(node.key) || 0, outflow.get(node.key) || 0));
  }
  return total;
});

// 高度跟着最宽的一层走:固定高度在节点多时会把小额节点压成一条线、标签叠在一起
const sankeyHeight = computed(() => {
  const perLayer = new Map<string, number>();
  for (const node of fundFlow.value.nodes) {
    perLayer.set(node.type, (perLayer.get(node.type) || 0) + 1);
  }
  const widest = Math.max(1, ...perLayer.values());
  return Math.min(560, Math.max(280, widest * 44));
});

const layerDepth: Record<FundFlowNode['type'], number> = { income: 0, account: 1, expense: 2 };

// 边的 data 是喂给 series.links 的那个对象(sankey 的字段叫 value,不是后端契约里的 amount)
interface SankeyTooltipParams {
  dataType?: string;
  name?: string;
  data?: { source: string; target: string; value: number };
}

const sankeyOption = computed(() => {
  void themeVersion.value;
  // 节点按层上色:收入=income,账户=accent,支出=expense
  const layerColor: Record<FundFlowNode['type'], string> = {
    income: getThemeColor('--chart-income'),
    account: getThemeColor('--accent'),
    expense: getThemeColor('--chart-expense')
  };
  const labelOf = (key: string) => nodeLabels.value.get(key) ?? key;
  return {
    tooltip: {
      trigger: 'item',
      triggerOn: 'mousemove',
      // 默认 tooltip 直接吐 data.name,会把 "account:Assets:Bank:CMB" 这种内部 key 露给用户
      formatter: (params: SankeyTooltipParams) => {
        if (params.dataType === 'edge' && params.data) {
          const { source, target, value } = params.data;
          return `${labelOf(source)} → ${labelOf(target)}<br/>${formatMoney(value)}`;
        }
        const key = params.name ?? '';
        return `${labelOf(key)}<br/>${formatMoney(nodeThroughput.value.get(key) ?? 0)}`;
      }
    },
    series: [
      {
        type: 'sankey',
        top: 8,
        bottom: 8,
        left: 4,
        right: 90,
        data: fundFlow.value.nodes.map(n => ({
          name: n.key,
          // depth 必须钉死:sankey 默认按图结构推层,本月没有收入流入的账户(信用卡就是常态)
          // 会被排到最左列和收入来源并肩,与图例宣称的「收入 | 账户 | 支出」三层对不上
          depth: layerDepth[n.type],
          itemStyle: { color: layerColor[n.type] },
          label: { formatter: () => nodeLabel(n) }
        })),
        links: fundFlow.value.links.map(l => ({ source: l.source, target: l.target, value: l.amount })),
        emphasis: { focus: 'adjacency' },
        // 0 = 不做力导迭代,按后端给的顺序摆:开着的话每次 refresh 图都会重排,
        // 同一份数据看两眼是两个样子
        layoutIterations: 0,
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
  width: 100%;
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
