<template>
  <div class="animate-fade-in-up ac">
    <!-- 汇总:总资产 / 总负债 / 净资产 -->
    <div class="ac-summary">
      <div class="card ac-sum-card">
        <span class="ac-sum-label">总资产</span>
        <span class="ac-sum-value tabular-nums" style="color: var(--income)">{{ formatMoney(summary?.totalAssets || 0) }}</span>
      </div>
      <div class="card ac-sum-card">
        <span class="ac-sum-label">总负债</span>
        <span class="ac-sum-value tabular-nums" style="color: var(--expense)">-{{ formatMoney(Math.abs(summary?.totalLiabilities || 0)) }}</span>
      </div>
      <div class="card ac-sum-card">
        <span class="ac-sum-label">净资产</span>
        <span class="ac-sum-value tabular-nums">{{ formatMoney(summary?.netWorth || 0) }}</span>
      </div>
    </div>

    <!-- 分组账户列表 -->
    <section v-for="group in accountGroups" :key="group.key" class="section-card">
      <div class="section-head">
        <h3 class="section-title">
          <component :is="group.icon" :size="16" class="sec-ic" />{{ group.title }}
        </h3>
        <span class="ac-group-total tabular-nums" :style="{ color: group.totalColor }">{{ group.total }}</span>
      </div>
      <div>
        <div v-for="acc in group.rows" :key="acc.account" class="ac-row">
          <div class="ac-row-icon">
            <component :is="acc.icon" :size="16" />
          </div>
          <div class="ac-row-info">
            <div class="ac-row-header">
              <span class="ac-row-name">{{ acc.name }}</span>
              <span class="ac-row-tag">{{ acc.tag }}</span>
            </div>
            <div class="ac-row-account tabular-nums">{{ acc.account }}</div>
          </div>
          <div class="ac-row-balance tabular-nums" :style="{ color: acc.color }">{{ acc.balance }}</div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { FunctionalComponent } from 'vue';
import type { AccountBalance } from '../../types/api';
import { formatMoney } from '../../composables/useFormatters';
import { useAnalytics } from '../../composables/useAnalytics';
import { Wallet, Landmark, CreditCard, LineChart, ShoppingBag, Utensils, PiggyBank } from '@lucide/vue';

const { analytics } = useAnalytics();

const summary = computed(() => analytics.value?.summary);

interface AccountRow {
  account: string;
  name: string;
  tag: string;
  icon: FunctionalComponent;
  balance: string;
  color: string;
}
interface AccountGroup {
  key: string;
  title: string;
  icon: FunctionalComponent;
  total: string;
  totalColor: string;
  rows: AccountRow[];
}

const accountGroups = computed<AccountGroup[]>(() => {
  const accounts = analytics.value?.accountBalances || [];
  const defs = [
    { key: 'Assets', title: '资产', icon: Wallet, accentColor: 'var(--income)' },
    { key: 'Liabilities', title: '负债', icon: CreditCard, accentColor: 'var(--expense)' },
  ];

  return defs
    .map(def => {
      const rows = accounts
        .filter(acc => acc.account.split(':')[0] === def.key)
        .map<AccountRow>(acc => ({
          account: acc.account,
          name: getAccountName(acc),
          tag: getAccountTag(acc),
          icon: getAccountIcon(acc),
          balance: formatMoney(acc.balance),
          color: acc.balance < 0 ? 'var(--expense)' : 'var(--text-primary)',
        }));
      const totalNum = accounts
        .filter(acc => acc.account.split(':')[0] === def.key)
        .reduce((sum, acc) => sum + acc.balance, 0);
      return {
        key: def.key,
        title: def.title,
        icon: def.icon,
        total: (totalNum < 0 ? '-' : '') + formatMoney(Math.abs(totalNum)),
        totalColor: def.accentColor,
        rows,
      };
    })
    .filter(g => g.rows.length > 0);
});

function getAccountName(account: AccountBalance): string {
  const fullMap: Record<string, string> = {
    'Assets:Bank:CMB': '招商银行',
    'Liabilities:CreditCard:CMB': '招行信用卡',
    'Assets:Bank:ICBC': '工商银行',
    'Assets:Bank:PSBC': '邮储银行',
    'Assets:Cash:WeChat': '微信零钱',
    'Assets:Cash:Alipay': '支付宝余额',
    'Assets:Cash:JDECard': '京东E卡',
    'Assets:Cash:JDBalance': '京东余额',
    'Assets:Investment:Stock': '股票账户',
    'Assets:Investment:Fund': '基金/理财',
    'Assets:Investment:Crypto': '加密货币',
    'Liabilities:Alipay:Huabei': '花呗',
    'Liabilities:JD:BNPL': '京东白条',
    'Liabilities:JD:CLO': '京东金条',
    'Liabilities:Meituan:MP': '美团月付',
    'Liabilities:Loan:ECMB': 'E招贷',
    'Liabilities:Loan:Mortgage': '房贷',
  };
  if (fullMap[account.account]) {
    return fullMap[account.account];
  }

  const parts = account.account.split(':');
  const lastPart = parts[parts.length - 1];
  const shortMap: Record<string, string> = {
    CMB: '招商银行',
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
    JDECard: '京东E卡',
    JDBalance: '京东余额',
    Mortgage: '房贷',
    ECMB: 'E招贷',
  };
  return shortMap[lastPart] || lastPart;
}

function getAccountTag(account: AccountBalance): string {
  const path = account.account;
  if (path.includes('CreditCard')) return '信用卡';
  if (path.includes('Bank')) return '储蓄卡';
  if (path.includes('Cash')) return '现金余额';
  if (path.includes('Investment')) return '投资理财';
  if (path.includes('Huabei') || path.includes('BNPL') || path.includes('CLO') || path.includes('MP')) return '消费信贷';
  if (path.includes('Loan') || path.includes('Mortgage')) return '贷款';

  const parts = path.split(':');
  return parts.length > 1 ? parts[1] : '其他';
}

function getAccountIcon(account: AccountBalance): FunctionalComponent {
  const path = account.account;
  if (path.includes('CreditCard')) return CreditCard;
  if (path.includes('Bank')) return Landmark;
  if (path.includes('WeChat') || path.includes('Alipay')) return Wallet;
  if (path.includes('Investment')) return LineChart;
  if (path.includes('Huabei') || path.includes('BNPL') || path.includes('CLO') || path.includes('JDECard') || path.includes('JDBalance')) return ShoppingBag;
  if (path.includes('Meituan')) return Utensils;
  if (path.includes('Loan') || path.includes('Mortgage')) return Landmark;
  return PiggyBank;
}
</script>

<style scoped>
.ac {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* ===== 汇总卡 ===== */
.ac-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-4);
}

.ac-sum-card {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.ac-sum-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.ac-sum-value {
  font-size: var(--font-size-2xl);
  font-weight: 700;
}

/* ===== 分组账户 ===== */
.ac-group-total {
  font-size: var(--font-size-sm);
  font-weight: 650;
}

.ac-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-5);
  border-bottom: 1px solid var(--hairline);
  transition: background var(--transition-base);
}

.ac-row:last-child { border-bottom: none; }
.ac-row:hover { background: var(--surface-2); }

.ac-row-icon {
  width: 34px;
  height: 34px;
  flex: none;
  border-radius: var(--radius-md);
  background: var(--surface-3);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.ac-row-info {
  flex: 1;
  min-width: 0;
}

.ac-row-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.ac-row-name {
  font-size: var(--font-size-sm);
  font-weight: 550;
  color: var(--text-primary);
}

.ac-row-tag {
  font-size: var(--font-size-xs);
  padding: 1px var(--space-2);
  border-radius: var(--radius-sm);
  background: var(--surface-3);
  color: var(--text-secondary);
  font-weight: 400;
}

.ac-row-account {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  font-family: var(--font-mono);
}

.ac-row-balance {
  font-size: var(--font-size-base);
  font-weight: 650;
  flex: none;
}

@media (max-width: 768px) {
  .ac-summary { grid-template-columns: 1fr; }
}
</style>
