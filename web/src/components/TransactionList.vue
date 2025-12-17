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
        <span v-html="getCategoryIcon(tx.category)" :style="{ stroke: tx.isIncome ? 'var(--income)' : 'var(--expense)', width: '16px', height: '16px' }"></span>
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
import { icons } from '../composables/icons';

const props = defineProps({
  transactions: { type: Array, required: true },
  maxHeight: { type: String, default: '400px' },
  showAccount: { type: Boolean, default: true }
});

// Category label mapping
const categoryLabels = {
  Food: '餐饮', Shopping: '购物', Transport: '交通', Entertainment: '娱乐',
  Gift: '红包/礼物', Financial: '金融', Communication: '通讯', Lodging: '住宿',
  Digital: '数码', Health: '健康', Education: '教育', Income: '收入', Other: '其他'
};

function getCategoryLabel(cat) {
  return categoryLabels[cat] || cat || '其他';
}

function formatDate(dateStr) {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${month}-${day}`;
}

// Process transactions to extract amount and category from postings
const displayedTransactions = computed(() => {
  return props.transactions.map(tx => {
    // If already processed, return as-is
    if (tx.isIncome !== undefined && tx.amount !== undefined) {
      return tx;
    }
    
    let amount = 0;
    let category = 'Other';
    let isIncome = false;
    let accountShort = '';
    
    if (tx.postings && tx.postings.length > 0) {
      for (const posting of tx.postings) {
        const account = posting.account || '';
        
        if (account.startsWith('Expenses:')) {
          amount = posting.amount;
          isIncome = false;
          const parts = account.split(':');
          category = parts.length > 1 ? parts[1] : 'Other';
        } else if (account.startsWith('Income:')) {
          amount = Math.abs(posting.amount);
          isIncome = true;
          const parts = account.split(':');
          category = parts.length > 1 ? parts[1] : 'Income';
        }
        
        if (account.startsWith('Assets:') || account.startsWith('Liabilities:')) {
          const parts = account.split(':');
          accountShort = parts.length > 2 ? parts[2] : (parts.length > 1 ? parts[1] : account);
        }
      }
    }
    
    return {
      ...tx,
      amount,
      category,
      isIncome,
      accountShort
    };
  });
});

function getCategoryIcon(category) {
  const iconMap = {
    Food: icons.food,
    Shopping: icons.shopping,
    Transport: icons.transfer,
    Gift: icons.gift,
    Entertainment: icons.entertainment,
    Financial: icons.wallet,
    Income: icons.arrowDown,
    Health: icons.heart,
  };
  return iconMap[category] || icons.creditCard;
}
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
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
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
