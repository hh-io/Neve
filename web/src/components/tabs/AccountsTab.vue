<template>
  <div class="animate-fade-in-up">
    <!-- Net Worth Summary -->
    <div class="card section-mb networth-card">
      <div class="networth-top">
        <div>
          <div class="networth-eyebrow">净资产</div>
          <div class="networth-value">{{ formatMoney(summary?.netWorth || 0) }}</div>
        </div>
        <div class="networth-icon">
          <Wallet :size="32" color="white" />
        </div>
      </div>
      <div class="networth-detail">
        <div>
          <div class="networth-sub">总资产</div>
          <div class="networth-sub-value">{{ formatMoney(summary?.totalAssets || 0) }}</div>
        </div>
        <div>
          <div class="networth-sub">总负债</div>
          <div class="networth-sub-value">{{ formatMoney(Math.abs(summary?.totalLiabilities || 0)) }}</div>
        </div>
      </div>
    </div>

    <!-- Account Tree -->
    <div class="card-static panel">
      <div class="panel-head">
        <div class="panel-head-left">
          <div class="panel-icon bg-brand-light">
            <Landmark :size="20" color="var(--accent)" />
          </div>
          <span class="panel-title">账户结构</span>
        </div>
        <div class="tree-actions">
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
                <ChevronRight />
              </span>
              <div :class="['tree-icon', getGroupIconClass(groupKey)]">
                <component :is="getGroupIcon(groupKey)" />
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
                      <ChevronRight />
                    </span>
                    <div :class="['tree-icon-sm', getSubTypeIconClass(subKey)]">
                      <component :is="getSubTypeIcon(subKey)" />
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
                          <component :is="getAccountIcon(account)" />
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

<script setup lang="ts">
import { computed, reactive } from 'vue';
import type { FunctionalComponent } from 'vue';
import type { AccountBalance } from '../../types/api';
import { formatMoney } from '../../composables/useFormatters';
import { useAnalytics } from '../../composables/useAnalytics';
import { Wallet, Landmark, ChevronRight, CreditCard, LineChart, ShoppingBag, Utensils, PiggyBank } from '@lucide/vue';

const { analytics } = useAnalytics();

const summary = computed(() => analytics.value?.summary);

// 账户树:Assets/Liabilities → 子类型 → 账户列表
type AccountTree = Record<string, Record<string, AccountBalance[]>>;

// Expanded state for tree nodes (first level expanded by default)
const expandedNodes = reactive<Record<string, boolean>>({
  Assets: true,
  Liabilities: true
});

// Build tree structure from flat account list
const accountTree = computed<AccountTree>(() => {
  const accounts = analytics.value?.accountBalances || [];
  const tree: AccountTree = {};

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
function toggleNode(key: string) {
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
function getGroupLabel(key: string): string {
  const labels: Record<string, string> = { Assets: '资产', Liabilities: '负债' };
  return labels[key] || key;
}

function getGroupIcon(key: string): FunctionalComponent {
  return key === 'Assets' ? Wallet : CreditCard;
}

function getGroupIconClass(key: string): string {
  return key === 'Assets' ? 'icon-positive' : 'icon-negative';
}

function getGroupTotal(key: string): number {
  const group = accountTree.value[key];
  if (!group) return 0;
  return Object.values(group).flat().reduce((sum, acc) => sum + acc.balance, 0);
}

function getSubGroupTotal(accounts: AccountBalance[]): number {
  return accounts.reduce((sum, acc) => sum + acc.balance, 0);
}

// Sub-type helpers
function getSubTypeLabel(key: string): string {
  const labels: Record<string, string> = {
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

function getSubTypeIcon(key: string): FunctionalComponent {
  const iconMap: Record<string, FunctionalComponent> = {
    Bank: Landmark,
    Cash: Wallet,
    CreditCard: CreditCard,
    Investment: LineChart,
    JD: ShoppingBag,
    Meituan: Utensils,
    Alipay: Wallet,
  };
  return iconMap[key] || Wallet;
}

function getSubTypeIconClass(key: string): string {
  const classMap: Record<string, string> = {
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
function getAccountName(account: AccountBalance): string {
  const parts = account.account.split(':');
  const name = parts[parts.length - 1];

  // Map common account names to Chinese
  const nameMap: Record<string, string> = {
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

function getAccountIcon(account: AccountBalance): FunctionalComponent {
  const path = account.account;
  if (path.includes('CreditCard')) return CreditCard;
  if (path.includes('Bank')) return Landmark;
  if (path.includes('WeChat') || path.includes('Alipay')) return Wallet;
  if (path.includes('Investment')) return LineChart;
  if (path.includes('Huabei')) return ShoppingBag;
  if (path.includes('BNPL') || path.includes('CLO')) return ShoppingBag;
  if (path.includes('Meituan')) return Utensils;
  return PiggyBank;
}

function getAccountTypeLabel(account: AccountBalance): string {
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

function getAccountBadgeClass(account: AccountBalance): string {
  const label = getAccountTypeLabel(account);
  const classMap: Record<string, string> = {
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
/* 净资产汇总卡(accent 渐变) */
.networth-card {
  padding: var(--card-pad);
  background: linear-gradient(135deg, var(--accent), var(--accent-hover));
  border: none;
}

.networth-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.networth-eyebrow {
  font-size: var(--font-size-sm);
  color: rgba(255, 255, 255, 0.8);
  margin-bottom: var(--space-1);
}

.networth-value {
  font-size: var(--font-size-2xl);
  font-weight: 700;
  color: #fff;
  font-variant-numeric: tabular-nums;
}

.networth-icon {
  width: 64px;
  height: 64px;
  border-radius: var(--radius-lg);
  background: rgba(255, 255, 255, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.networth-detail {
  display: flex;
  gap: var(--space-8);
  margin-top: var(--space-4);
}

.networth-sub {
  font-size: var(--font-size-xs);
  color: rgba(255, 255, 255, 0.7);
}

.networth-sub-value {
  font-size: var(--font-size-lg);
  color: #fff;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.tree-actions {
  display: flex;
  gap: var(--space-2);
}

.account-tree {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.tree-node {
  border-radius: var(--radius-md);
}

.tree-node.level-1 {
  background: var(--surface-2);
  padding: var(--space-2);
}

.tree-node.level-2 {
  margin-left: var(--space-4);
  background: var(--surface-1);
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
  background: var(--surface-2);
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

.icon-default { background: var(--surface-2); }
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
  background: var(--surface-2);
  border-radius: 10px;
}

.tree-amount {
  font-weight: 600;
  font-size: var(--font-size-sm);
  min-width: 100px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.tree-badge {
  font-size: var(--font-size-xs);
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
}

.badge-credit { background: var(--expense-light); color: var(--expense); }
.badge-bank { background: var(--info-light); color: var(--info); }
.badge-wechat { background: var(--income-light); color: var(--income); }
.badge-alipay { background: var(--info-light); color: var(--info); }
.badge-huabei { background: var(--warning-light); color: var(--warning); }
.badge-bnpl { background: var(--expense-light); color: var(--expense); }
.badge-invest { background: var(--warning-light); color: var(--warning); }
.badge-default { background: var(--surface-2); color: var(--text-secondary); }

.tree-children {
  padding-bottom: var(--space-2);
}
</style>
