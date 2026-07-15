<template>
  <div v-if="transactions.length === 0" class="tx-empty">
    暂无交易记录
  </div>

  <div v-else class="tx-scroll-container" :style="{ maxHeight }">
    <div
      v-for="(tx, index) in displayedTransactions"
      :key="`${tx.date}-${index}`"
      class="tx-row"
    >
      <!-- Icon (using Vue components) -->
      <div :class="['tx-icon', tx.iconClass]">
        <component
          :is="getCategoryIcon(tx.category)"
          :size="18"
          :color="tx.iconColor"
        />
      </div>

      <!-- Main Info -->
      <div class="tx-main">
        <div class="tx-title">{{ tx.payee || tx.narration || '未知交易' }}</div>
        <div v-if="tx.payee && tx.narration" class="tx-narration">{{ tx.narration }}</div>
        <div class="tx-meta">
          <span class="tx-category">{{ tx.isTransfer ? '转账' : getCategoryLabel(tx.category) }}</span>
          <span class="tx-date">{{ formatDate(tx.date) }}</span>
          <span v-for="tag in (tx.tags || [])" :key="tag" class="tx-tag">#{{ tag }}</span>
        </div>
      </div>

      <!-- Amount & Account -->
      <div class="tx-right">
        <div :class="['tx-amount', tx.amountClass]">{{ tx.amountText }}</div>
        <div v-if="tx.isTransfer && tx.feeAmount > 0" class="tx-fee">手续费 ¥{{ tx.feeAmount.toFixed(2) }}</div>
        <div v-if="showAccount" class="tx-account">{{ tx.accountShort }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { Transaction } from '../types/api';
import {
  getCategoryLabel,
  processTransaction,
  formatTransactionDate
} from '../composables/useCategories';
import { getCategoryIcon } from '../composables/useCategoryIcon';

const props = withDefaults(defineProps<{
  transactions: Transaction[];
  maxHeight?: string;
  showAccount?: boolean;
}>(), {
  maxHeight: '400px',
  showAccount: true
});

// Process transactions using shared utility
const displayedTransactions = computed(() => {
  return props.transactions.map(processTransaction);
});

// Expose formatDate for template
const formatDate = formatTransactionDate;
</script>

<style scoped>
.tx-empty {
  text-align: center;
  padding: var(--space-8);
  color: var(--text-tertiary);
}

.tx-scroll-container {
  overflow-y: auto;
  padding-right: var(--space-2);
}

.tx-scroll-container::-webkit-scrollbar {
  width: 6px;
}

.tx-scroll-container::-webkit-scrollbar-track {
  background: var(--surface-2);
  border-radius: 3px;
}

.tx-scroll-container::-webkit-scrollbar-thumb {
  background: var(--hairline);
  border-radius: 3px;
}

/* changelog-row:扁平行,靠发丝线底边分隔,无卡片嵌套、无斑马纹 */
.tx-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  transition: background var(--transition-base);
}

.tx-row:hover {
  background: var(--surface-2);
}

.tx-row:not(:last-child) {
  border-bottom: 1px solid var(--hairline);
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
  background: var(--surface-2);
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
  background: var(--accent-subtle);
  color: var(--accent);
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
  font-family: var(--font-numeric);
  font-variant-numeric: tabular-nums;
}

.tx-account {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-top: 2px;
}

.text-income { color: var(--income); }
.text-expense { color: var(--expense); }
.text-transfer { color: var(--text-secondary); }
.bg-income-light { background: var(--income-light); }
.bg-expense-light { background: var(--expense-light); }
.bg-brand-light { background: var(--accent-subtle); }

.tx-fee {
  font-size: 10px;
  color: var(--text-tertiary);
  margin-top: 2px;
}
</style>
