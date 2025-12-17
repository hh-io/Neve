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
import IncomeChart from '../IncomeChart.vue';
import PlatformRanking from '../PlatformRanking.vue';
import MerchantRanking from '../MerchantRanking.vue';
import { icons } from '../../composables/icons';

use([PieChart, SankeyChart, TitleComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

const props = defineProps({
  analytics: { type: Object, required: true }
});

// Sankey Data Logic
const sankeyData = computed(() => {
  const transactions = props.analytics.recentTransactions || [];
  if (transactions.length === 0) return { nodes: [], links: [] };

  const nodes = new Set();
  const links = [];
  const incomeMap = {}; // Income Category -> Amount
  const expenseMap = {}; // Expense Category -> Amount (via which account?)
  
  // Simplified flow: Income -> [Pool] -> Expense
  // Or better: Income Source -> Account -> Expense Category
  // We need asset information for the middle layer. 
  // Since 'analytics' might not have direct asset mapping, we'll try to reconstruct from postings in recentTransactions if available.
  
  // 1. Aggregate Income Sources
  // 2. Aggregate Expenses by Category
  // 3. Middle layer: "Assets" (Virtual Wallet for now if account info is missing on income side)
  
  // Let's assume a "Wallet" node as the central hub if we can't trace exact flows
  // But wait, transactions have 'postings'. Let's see if we can use them.
  
  // Building Nodes & Links
  const linkMap = {}; // "Source|Target" -> Value
  
  transactions.forEach(tx => {
    // This is a simplified Sankey generator based on what we have.
    // Ideally, we trace Account -> Expense
    
    // Check if it's income or expense
    let isIncome = false;
    let category = 'Other';
    let account = 'Unknown Account';
    let amount = 0;

    if (tx.postings) {
      tx.postings.forEach(p => {
        if (p.account.startsWith('Income:')) {
          isIncome = true;
          category = p.account.split(':')[1] || 'Income';
          amount = Math.abs(p.amount);
        } else if (p.account.startsWith('Expenses:')) {
          isIncome = false;
          category = p.account.split(':')[1] || 'Other';
          amount = p.amount;
        } else if (p.account.startsWith('Assets:') || p.account.startsWith('Liabilities:')) {
           // This is the asset account
           const parts = p.account.split(':');
           account = parts.length > 2 ? parts[2] : (parts.length > 1 ? parts[1] : 'Assets');
        }
      });
    }

    if (amount > 0) {
      if (isIncome) {
        // Income -> Account
        const source = `Income:${category}`; // Prefix to avoid name collision
        const target = `Account:${account}`;
        const key = `${source}|${target}`;
        linkMap[key] = (linkMap[key] || 0) + amount;
        nodes.add(source);
        nodes.add(target);
      } else {
        // Account -> Expense
        const source = `Account:${account}`;
        const target = `Expense:${category}`;
        const key = `${source}|${target}`;
        linkMap[key] = (linkMap[key] || 0) + amount;
        nodes.add(source);
        nodes.add(target);
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
    return { source, target, value: linkMap[key] };
  });

  return { nodes: layoutNodes, links: layoutLinks };
});

const hasSankeyData = computed(() => sankeyData.value.nodes.length > 0 && sankeyData.value.links.length > 0);

const sankeyOption = computed(() => ({
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
      label: { color: 'var(--text-primary)', fontSize: 12 }
    }
  ]
}));

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
