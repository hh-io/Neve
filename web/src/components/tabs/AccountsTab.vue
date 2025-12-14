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

    <!-- Account List -->
    <div class="card-static" style="padding: var(--space-6);">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
        <div style="display: flex; align-items: center; gap: var(--space-3);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.accounts" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">账户列表</span>
        </div>
        <span style="font-size: var(--font-size-sm); color: var(--text-tertiary);">{{ accountList.length }} 个账户</span>
      </div>

      <div style="display: flex; flex-direction: column; gap: var(--space-2);">
        <div
          v-for="(account, index) in accountList"
          :key="account.name"
          class="transaction-item"
          :style="{ animationDelay: `${index * 0.05}s` }"
        >
          <div :class="['transaction-icon', account.balance >= 0 ? 'bg-income-light' : 'bg-expense-light']">
            <span v-html="getAccountIcon(account.type)" :style="{ stroke: account.balance >= 0 ? 'var(--income)' : 'var(--expense)' }"></span>
          </div>
          <div class="transaction-info">
            <div class="transaction-title">{{ account.name }}</div>
            <div class="transaction-date">{{ account.type }}</div>
          </div>
          <div class="transaction-amount" :class="account.balance >= 0 ? 'text-income' : 'text-expense'">
            {{ formatMoney(account.balance) }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { formatMoney } from '../../composables/useFormatters';
import { icons } from '../../composables/icons';

const props = defineProps({
  analytics: { type: Object, required: true }
});

const accountList = computed(() => {
  return props.analytics.accountBalances || [];
});

function getAccountIcon(type) {
  const typeMap = {
    '银行卡': icons.bank,
    '信用卡': icons.creditCard,
    '储蓄卡': icons.piggyBank,
    '支付宝': icons.wallet,
    '微信': icons.wallet,
  };
  return typeMap[type] || icons.wallet;
}
</script>
