<template>
  <div class="animate-fade-in-up">
    <!-- Net Worth Summary -->
    <div class="card section-mb" style="padding: var(--space-6); background: linear-gradient(135deg, var(--brand-primary), var(--brand-secondary)); border: none;">
      <div style="display: flex; align-items: center; justify-content: space-between;">
        <div>
          <div style="font-size: var(--font-size-sm); color: rgba(255,255,255,0.8); margin-bottom: var(--space-1);">净资产</div>
          <div style="font-size: var(--font-size-2xl); font-weight: 700; color: white;">{{ formatMoney(analytics.summary?.netWorth || 0) }}</div>
        </div>
        <div style="width: 64px; height: 64px; border-radius: var(--radius-lg); background: rgba(255,255,255,0.2); display: flex; align-items: center; justify-content: center;">
          <span v-html="icons.wallet" style="stroke: white; width: 32px; height: 32px;"></span>
        </div>
      </div>
      <div style="display: flex; gap: var(--space-8); margin-top: var(--space-4);">
        <div>
          <div style="font-size: var(--font-size-xs); color: rgba(255,255,255,0.7);">总资产</div>
          <div style="font-size: var(--font-size-lg); color: white; font-weight: 500;">{{ formatMoney(analytics.summary?.totalAssets || 0) }}</div>
        </div>
        <div>
          <div style="font-size: var(--font-size-xs); color: rgba(255,255,255,0.7);">总负债</div>
          <div style="font-size: var(--font-size-lg); color: white; font-weight: 500;">{{ formatMoney(Math.abs(analytics.summary?.totalLiabilities || 0)) }}</div>
        </div>
      </div>
    </div>

    <!-- Account Tree -->
    <div class="card-static" style="padding: var(--space-6);">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
        <div style="display: flex; align-items: center; gap: var(--space-3);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.accounts" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">账户结构</span>
        </div>
        <div style="display: flex; gap: var(--space-2);">
          <button class="btn btn-ghost" @click="expandAll">展开全部</button>
          <button class="btn btn-ghost" @click="collapseAll">收起全部</button>
        </div>
      </div>

      <!-- Tree Structure -->
      <div class="account-tree">
        <template v-for="(group, groupKey) in accountTree" :key="groupKey">
          <!-- Level 1: Assets / Liabilities -->
          <div class="tree-node level-1">
            <div class="tree-header" @click="toggleNode(groupKey)">
              <span class="tree-toggle" :class="{ expanded: expandedNodes[groupKey] }">
                <span v-html="icons.chevronRight"></span>
              </span>
              <div :class="['tree-icon', getGroupIconClass(groupKey)]">
                <span v-html="getGroupIcon(groupKey)"></span>
              </div>
              <span class="tree-label">{{ getGroupLabel(groupKey) }}</span>
              <span class="tree-count">{{ Object.keys(group).length }} 类</span>
              <span class="tree-amount" :class="groupKey === 'Liabilities' ? 'text-expense' : 'text-income'">
                {{ formatMoney(getGroupTotal(groupKey)) }}
              </span>
            </div>

            <!-- Level 2: Bank / Cash / CreditCard etc -->
            <div v-show="expandedNodes[groupKey]" class="tree-children">
              <template v-for="(accounts, subKey) in group" :key="`${groupKey}:${subKey}`">
                <div class="tree-node level-2">
                  <div class="tree-header" @click="toggleNode(`${groupKey}:${subKey}`)">
                    <span class="tree-toggle" :class="{ expanded: expandedNodes[`${groupKey}:${subKey}`] }">
                      <span v-html="icons.chevronRight"></span>
                    </span>
                    <div :class="['tree-icon-sm', getSubTypeIconClass(subKey)]">
                      <span v-html="getSubTypeIcon(subKey)"></span>
                    </div>
                    <span class="tree-label">{{ getSubTypeLabel(subKey) }}</span>
                    <span class="tree-count">{{ accounts.length }} 个</span>
                    <span class="tree-amount" :class="getSubGroupTotal(accounts) < 0 ? 'text-expense' : 'text-income'">
                      {{ formatMoney(getSubGroupTotal(accounts)) }}
                    </span>
                  </div>

                  <!-- Level 3: Individual accounts -->
                  <div v-show="expandedNodes[`${groupKey}:${subKey}`]" class="tree-children">
                    <div
                      v-for="account in accounts"
                      :key="account.account"
                      class="tree-node level-3 tree-leaf"
                    >
                      <div class="tree-header">
                        <span class="tree-indent"></span>
                        <div :class="['tree-icon-sm', account.balance >= 0 ? 'icon-positive' : 'icon-negative']">
                          <span v-html="getAccountIcon(account)"></span>
                        </div>
                        <span class="tree-label">{{ getAccountName(account) }}</span>
                        <span class="tree-badge" :class="getAccountBadgeClass(account)">
                          {{ getAccountTypeLabel(account) }}
                        </span>
                        <span class="tree-amount" :class="account.balance >= 0 ? 'text-income' : 'text-expense'">
                          {{ formatMoney(account.balance) }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </template>
            </div>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, reactive } from 'vue';
import { formatMoney } from '../../composables/useFormatters';
import { icons } from '../../composables/icons';

const props = defineProps({
  analytics: { type: Object, required: true }
});

// Expanded state for tree nodes (first level expanded by default)
const expandedNodes = reactive({
  Assets: true,
  Liabilities: true
});

// Build tree structure from flat account list
const accountTree = computed(() => {
  const accounts = props.analytics.accountBalances || [];
  const tree = {};

  accounts.forEach(acc => {
    const parts = acc.account.split(':');
    if (parts.length < 2) return;
    
    const level1 = parts[0]; // Assets, Liabilities
    const level2 = parts[1]; // Bank, Cash, CreditCard, etc.
    
    // Skip Equity accounts
    if (level1 === 'Equity') return;
    
    // Only show Assets and Liabilities (skip Expenses, Income)
    if (level1 !== 'Assets' && level1 !== 'Liabilities') return;

    if (!tree[level1]) tree[level1] = {};
    if (!tree[level1][level2]) tree[level1][level2] = [];
    tree[level1][level2].push(acc);
  });

  return tree;
});

// Toggle node expansion
function toggleNode(key) {
  expandedNodes[key] = !expandedNodes[key];
}

function expandAll() {
  Object.keys(accountTree.value).forEach(l1 => {
    expandedNodes[l1] = true;
    Object.keys(accountTree.value[l1]).forEach(l2 => {
      expandedNodes[`${l1}:${l2}`] = true;
    });
  });
}

function collapseAll() {
  Object.keys(expandedNodes).forEach(key => {
    expandedNodes[key] = false;
  });
}

// Group helpers
function getGroupLabel(key) {
  const labels = { Assets: '资产', Liabilities: '负债' };
  return labels[key] || key;
}

function getGroupIcon(key) {
  return key === 'Assets' ? icons.wallet : icons.creditCard;
}

function getGroupIconClass(key) {
  return key === 'Assets' ? 'icon-positive' : 'icon-negative';
}

function getGroupTotal(key) {
  const group = accountTree.value[key];
  if (!group) return 0;
  return Object.values(group).flat().reduce((sum, acc) => sum + acc.balance, 0);
}

function getSubGroupTotal(accounts) {
  return accounts.reduce((sum, acc) => sum + acc.balance, 0);
}

// Sub-type helpers
function getSubTypeLabel(key) {
  const labels = {
    Bank: '银行卡',
    Cash: '现金',
    CreditCard: '信用卡',
    Investment: '投资',
    JD: '京东',
    Meituan: '美团',
    Alipay: '支付宝',
  };
  return labels[key] || key;
}

function getSubTypeIcon(key) {
  const iconMap = {
    Bank: icons.bank,
    Cash: icons.wallet,
    CreditCard: icons.creditCard,
    Investment: icons.lineChart,
    JD: icons.shopping,
    Meituan: icons.food,
    Alipay: icons.wallet,
  };
  return iconMap[key] || icons.wallet;
}

function getSubTypeIconClass(key) {
  const classMap = {
    Bank: 'icon-bank',
    Cash: 'icon-cash',
    CreditCard: 'icon-credit',
    Investment: 'icon-invest',
    JD: 'icon-bnpl',
    Meituan: 'icon-bnpl',
    Alipay: 'icon-cash',
  };
  return classMap[key] || 'icon-default';
}

// Account helpers
function getAccountName(account) {
  const parts = account.account.split(':');
  const name = parts[parts.length - 1];
  
  // Map common account names to Chinese
  const nameMap = {
    CMBC: '招商银行',
    ICBC: '工商银行',
    PSBC: '邮储银行',
    WeChat: '微信零钱',
    Alipay: '支付宝余额',
    Huabei: '花呗',
    BNPL: '白条',
    CLO: '金条',
    MP: '月付',
    Stock: '股票',
    Fund: '基金',
    Crypto: '加密货币',
  };
  
  return nameMap[name] || name;
}

function getAccountIcon(account) {
  const path = account.account;
  if (path.includes('CreditCard')) return icons.creditCard;
  if (path.includes('Bank')) return icons.bank;
  if (path.includes('WeChat') || path.includes('Alipay')) return icons.wallet;
  if (path.includes('Investment')) return icons.lineChart;
  if (path.includes('Huabei')) return icons.shopping;
  if (path.includes('BNPL') || path.includes('CLO')) return icons.shopping;
  if (path.includes('Meituan')) return icons.food;
  return icons.piggyBank;
}

function getAccountTypeLabel(account) {
  const path = account.account;
  if (path.includes('CreditCard')) return '信用卡';
  if (path.includes('Bank')) return '储蓄卡';
  if (path.includes('WeChat')) return '微信';
  if (path.includes('Alipay') && !path.includes('Huabei')) return '支付宝';
  if (path.includes('Huabei')) return '花呗';
  if (path.includes('BNPL')) return '白条';
  if (path.includes('CLO')) return '金条';
  if (path.includes('MP')) return '月付';
  if (path.includes('Investment')) return '投资';
  return account.type === 'Liabilities' ? '负债' : '资产';
}

function getAccountBadgeClass(account) {
  const label = getAccountTypeLabel(account);
  const classMap = {
    '信用卡': 'badge-credit',
    '储蓄卡': 'badge-bank',
    '微信': 'badge-wechat',
    '支付宝': 'badge-alipay',
    '花呗': 'badge-huabei',
    '白条': 'badge-bnpl',
    '金条': 'badge-bnpl',
    '月付': 'badge-bnpl',
    '投资': 'badge-invest',
  };
  return classMap[label] || 'badge-default';
}
</script>

<style scoped>
.account-tree {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.tree-node {
  border-radius: var(--radius-md);
}

.tree-node.level-1 {
  background: var(--bg-tertiary);
  padding: var(--space-2);
}

.tree-node.level-2 {
  margin-left: var(--space-4);
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
  margin-top: var(--space-1);
}

.tree-node.level-3 {
  margin-left: var(--space-6);
}

.tree-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: background var(--transition-base);
}

.tree-header:hover {
  background: var(--bg-tertiary);
}

.tree-leaf .tree-header {
  cursor: default;
}

.tree-toggle {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform var(--transition-base);
}

.tree-toggle.expanded {
  transform: rotate(90deg);
}

.tree-toggle :deep(svg) {
  width: 14px;
  height: 14px;
  stroke: var(--text-tertiary);
}

.tree-indent {
  width: 20px;
}

.tree-icon {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
}

.tree-icon-sm {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
}

.tree-icon :deep(svg), .tree-icon-sm :deep(svg) {
  width: 16px;
  height: 16px;
}

.icon-positive { background: var(--income-light); }
.icon-positive :deep(svg) { stroke: var(--income); }

.icon-negative { background: var(--expense-light); }
.icon-negative :deep(svg) { stroke: var(--expense); }

.icon-bank { background: var(--info-light); }
.icon-bank :deep(svg) { stroke: var(--info); }

.icon-cash { background: var(--income-light); }
.icon-cash :deep(svg) { stroke: var(--income); }

.icon-credit { background: var(--expense-light); }
.icon-credit :deep(svg) { stroke: var(--expense); }

.icon-invest { background: var(--warning-light); }
.icon-invest :deep(svg) { stroke: var(--warning); }

.icon-bnpl { background: var(--expense-light); }
.icon-bnpl :deep(svg) { stroke: var(--expense); }

.icon-default { background: var(--bg-tertiary); }
.icon-default :deep(svg) { stroke: var(--text-secondary); }

.tree-label {
  flex: 1;
  font-weight: 500;
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}

.tree-count {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  padding: 2px 8px;
  background: var(--bg-tertiary);
  border-radius: 10px;
}

.tree-amount {
  font-weight: 600;
  font-size: var(--font-size-sm);
  min-width: 100px;
  text-align: right;
}

.tree-badge {
  font-size: var(--font-size-xs);
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
}

.badge-credit { background: var(--expense-light); color: var(--expense); }
.badge-bank { background: var(--info-light); color: var(--info); }
.badge-wechat { background: #e8f5e9; color: #4caf50; }
.badge-alipay { background: #e3f2fd; color: #1976d2; }
.badge-huabei { background: var(--warning-light); color: var(--warning); }
.badge-bnpl { background: var(--expense-light); color: var(--expense); }
.badge-invest { background: var(--warning-light); color: var(--warning); }
.badge-default { background: var(--bg-tertiary); color: var(--text-secondary); }

.tree-children {
  padding-bottom: var(--space-2);
}
</style>
