<template>
  <div class="tab-content">
    <section class="analytics-section fade-in">
      <div class="grid grid-2">
        <!-- Account Balances -->
        <div class="glass-card">
          <h3 class="card-title">账户余额</h3>
          <div class="account-list">
            <div v-for="account in analytics.accountBalances" :key="account.account" class="account-item">
              <div class="account-info">
                <span class="account-name">{{ formatAccountName(account.account) }}</span>
                <span class="account-type">{{ account.type }}</span>
              </div>
              <span class="account-balance" :class="account.balance >= 0 ? 'positive' : 'negative'">
                {{ formatMoney(account.balance) }}
              </span>
            </div>
          </div>
        </div>
        <LiabilityOverview :data="analytics.liabilityBreakdown" />
      </div>
    </section>
  </div>
</template>

<script setup>
import { formatMoney, formatAccountName } from '../../composables/useFormatters';
import LiabilityOverview from '../LiabilityOverview.vue';

defineProps({
  analytics: { type: Object, required: true }
});
</script>
