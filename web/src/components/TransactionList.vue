<template>
  <div v-if="transactions.length === 0" style="text-align: center; padding: var(--space-8); color: var(--text-tertiary);">
    暂无交易记录
  </div>

  <div v-else class="tx-scroll-container" :style="{ maxHeight: maxHeight }">
    <div
      v-for="(tx, index) in displayedTransactions"
      :key="`${tx.date}-${index}`"
      class="tx-row"
    >
      <!-- Icon -->
      <div :class="['tx-icon', tx.isIncome ? 'bg-income-light' : 'bg-expense-light']">
        <span v-html="getCategoryIcon(tx.category)" :style="{ stroke: tx.isIncome ? 'var(--income)' : 'var(--expense)', width: '18px', height: '18px' }"></span>
      </div>
      
      <!-- Main Info -->
      <div class="tx-main">
        <div class="tx-title">{{ tx.payee || tx.narration || '未知交易' }}</div>
        <div v-if="tx.payee && tx.narration" class="tx-narration">{{ tx.narration }}</div>
        <div class="tx-meta">
          <span class="tx-category">{{ getCategoryLabel(tx.category) }}</span>
          <span class="tx-date">{{ formatDate(tx.date) }}</span>
          <span v-for="tag in (tx.tags || [])" :key="tag" class="tx-tag">#{{ tag }}</span>
        </div>
      </div>
      
      <!-- Amount & Account -->
      <div class="tx-right">
        <div :class="['tx-amount', tx.isIncome ? 'text-income' : 'text-expense']">
          {{ tx.isIncome ? '+' : '-' }}¥{{ Math.abs(tx.amount).toFixed(2) }}
        </div>
        <div v-if="showAccount" class="tx-account">{{ tx.accountShort }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { 
  getCategoryLabel, 
  getCategoryIcon, 
  processTransaction, 
  formatTransactionDate 
} from '../composables/useCategories';

const props = defineProps({
  transactions: { type: Array, required: true },
  maxHeight: { type: String, default: '400px' },
  showAccount: { type: Boolean, default: true }
});

// Process transactions using shared utility
const displayedTransactions = computed(() => {
  return props.transactions.map(processTransaction);
});

// Expose formatDate for template
const formatDate = formatTransactionDate;
</script>

<style scoped>
.tx-scroll-container {
  overflow-y: auto;
  padding-right: var(--space-2);
}

.tx-scroll-container::-webkit-scrollbar {
  width: 6px;
}

.tx-scroll-container::-webkit-scrollbar-track {
  background: var(--bg-tertiary);
  border-radius: 3px;
}

.tx-scroll-container::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 3px;
}

.tx-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  transition: background var(--transition-base);
}

.tx-row:hover {
  background: var(--bg-tertiary);
}

.tx-row:not(:last-child) {
  border-bottom: 1px solid var(--border-light, rgba(0,0,0,0.05));
}

.tx-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tx-icon span {
  display: flex;
  align-items: center;
  justify-content: center;
}

.tx-icon span :deep(svg) {
  width: 18px;
  height: 18px;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
  fill: none;
}

.tx-main {
  flex: 1;
  min-width: 0;
}

.tx-title {
  font-weight: 500;
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: 2px;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.tx-category {
  padding: 1px 6px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-sm);
}

.tx-narration {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-tag {
  padding: 1px 5px;
  background: var(--brand-light);
  color: var(--brand-primary);
  border-radius: var(--radius-sm);
  font-size: 10px;
}

.tx-right {
  text-align: right;
  flex-shrink: 0;
}

.tx-amount {
  font-weight: 600;
  font-size: var(--font-size-sm);
  font-feature-settings: 'tnum';
}

.tx-account {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-top: 2px;
}

.text-income { color: var(--income); }
.text-expense { color: var(--expense); }
.bg-income-light { background: var(--income-light, rgba(107, 155, 122, 0.15)); }
.bg-expense-light { background: var(--expense-light, rgba(194, 123, 123, 0.15)); }
</style>
