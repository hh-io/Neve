<template>
  <div class="animate-fade-in-up">
    <!-- Filters -->
    <div class="card-static section-mb" style="padding: var(--space-4);">
      <div style="display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-3);">
        <!-- Search -->
        <div style="flex: 1; min-width: 200px; position: relative;">
          <span v-html="icons.search" style="position: absolute; left: 12px; top: 50%; transform: translateY(-50%); width: 16px; height: 16px; stroke: var(--text-tertiary);"></span>
          <input
            type="text"
            v-model="searchQuery"
            placeholder="搜索商家、备注..."
            class="search-input"
          />
        </div>

        <!-- Category Filter -->
        <select v-model="categoryFilter" class="filter-select">
          <option value="">所有分类</option>
          <option v-for="cat in categories" :key="cat" :value="cat">{{ getCategoryLabel(cat) }}</option>
        </select>

        <!-- Type Filter -->
        <select v-model="typeFilter" class="filter-select">
          <option value="">全部类型</option>
          <option value="expense">支出</option>
          <option value="income">收入</option>
        </select>

        <!-- Reset -->
        <button v-if="hasFilters" class="btn btn-ghost" @click="resetFilters" style="font-size: var(--font-size-sm);">
          清除
        </button>
      </div>
    </div>

    <!-- Transaction List -->
    <div class="card-static" style="padding: var(--space-4);">
      <!-- Header -->
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-3); padding-bottom: var(--space-3); border-bottom: 1px solid var(--border);">
        <div style="display: flex; align-items: center; gap: var(--space-2);">
          <div class="stat-icon bg-brand-light" style="width: 32px; height: 32px;">
            <span v-html="icons.transactions" style="stroke: var(--brand-primary); width: 16px; height: 16px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary); font-size: var(--font-size-base);">交易明细</span>
        </div>
        <span class="badge">共 {{ filteredTransactions.length }} 条</span>
      </div>

      <!-- Empty State -->
      <div v-if="processedTransactions.length === 0" style="text-align: center; padding: var(--space-8); color: var(--text-tertiary);">
        暂无匹配的交易记录
      </div>

      <!-- Scrollable Transaction List -->
      <div v-else class="transaction-scroll-container">
        <div
          v-for="(tx, index) in paginatedTransactions"
          :key="`${tx.date}-${index}`"
          class="transaction-row"
        >
          <!-- Icon -->
          <div :class="['tx-icon', tx.isIncome ? 'bg-income-light' : 'bg-expense-light']">
            <span v-html="getCategoryIcon(tx.category)" :style="{ stroke: tx.isIncome ? 'var(--income)' : 'var(--expense)', width: '16px', height: '16px' }"></span>
          </div>
          
          <!-- Main Info -->
          <div class="tx-main">
            <div class="tx-title">{{ tx.payee || tx.narration || '未知交易' }}</div>
            <div class="tx-meta">
              <span class="tx-category">{{ getCategoryLabel(tx.category) }}</span>
              <span class="tx-date">{{ formatDate(tx.date) }}</span>
            </div>
          </div>
          
          <!-- Amount & Account -->
          <div class="tx-right">
            <div :class="['tx-amount', tx.isIncome ? 'text-income' : 'text-expense']">
              {{ tx.isIncome ? '+' : '-' }}¥{{ Math.abs(tx.amount).toFixed(2) }}
            </div>
            <div class="tx-account">{{ tx.accountShort }}</div>
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="pagination">
        <button class="btn btn-secondary btn-sm" :disabled="currentPage === 1" @click="currentPage--">
          ‹ 上一页
        </button>
        <div class="page-info">
          <span class="current-page">{{ currentPage }}</span>
          <span class="page-separator">/</span>
          <span class="total-pages">{{ totalPages }}</span>
        </div>
        <button class="btn btn-secondary btn-sm" :disabled="currentPage === totalPages" @click="currentPage++">
          下一页 ›
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { icons } from '../../composables/icons';

const props = defineProps({
  transactions: { type: Array, required: true }
});

// Filters
const searchQuery = ref('');
const categoryFilter = ref('');
const typeFilter = ref('');
const currentPage = ref(1);
const pageSize = 15;

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
const processedTransactions = computed(() => {
  return props.transactions.map(tx => {
    let amount = 0;
    let category = 'Other';
    let isIncome = false;
    let accountShort = '';
    
    // Find the expense or income posting
    if (tx.postings && tx.postings.length > 0) {
      for (const posting of tx.postings) {
        const account = posting.account || '';
        
        if (account.startsWith('Expenses:')) {
          amount = posting.amount;
          isIncome = false;
          // Extract category from account (e.g., Expenses:Food:Delivery -> Food)
          const parts = account.split(':');
          category = parts.length > 1 ? parts[1] : 'Other';
        } else if (account.startsWith('Income:')) {
          amount = Math.abs(posting.amount);
          isIncome = true;
          const parts = account.split(':');
          category = parts.length > 1 ? parts[1] : 'Income';
        }
        
        // Get payment account (Assets or Liabilities)
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
      accountShort,
      payee: tx.payee || '',
      narration: tx.narration || ''
    };
  });
});

const categories = computed(() => {
  const cats = new Set(processedTransactions.value.map(t => t.category));
  return Array.from(cats).sort();
});

const hasFilters = computed(() => searchQuery.value || categoryFilter.value || typeFilter.value);

function resetFilters() {
  searchQuery.value = '';
  categoryFilter.value = '';
  typeFilter.value = '';
  currentPage.value = 1;
}

const filteredTransactions = computed(() => {
  let result = processedTransactions.value;
  
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase();
    result = result.filter(t => 
      (t.payee && t.payee.toLowerCase().includes(q)) ||
      (t.narration && t.narration.toLowerCase().includes(q)) ||
      (t.category && t.category.toLowerCase().includes(q))
    );
  }
  
  if (categoryFilter.value) {
    result = result.filter(t => t.category === categoryFilter.value);
  }
  
  if (typeFilter.value) {
    if (typeFilter.value === 'expense') result = result.filter(t => !t.isIncome);
    else if (typeFilter.value === 'income') result = result.filter(t => t.isIncome);
  }
  
  return result;
});

const totalPages = computed(() => Math.ceil(filteredTransactions.value.length / pageSize));

const paginatedTransactions = computed(() => {
  const start = (currentPage.value - 1) * pageSize;
  return filteredTransactions.value.slice(start, start + pageSize);
});

watch([searchQuery, categoryFilter, typeFilter], () => {
  currentPage.value = 1;
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
.search-input {
  width: 100%;
  padding: var(--space-2) var(--space-3) var(--space-2) 36px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  outline: none;
  transition: border-color var(--transition-base);
}

.search-input:focus {
  border-color: var(--brand-primary);
}

.filter-select {
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-tertiary);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  min-width: 100px;
  outline: none;
  cursor: pointer;
}

.badge {
  padding: var(--space-1) var(--space-2);
  background: var(--bg-tertiary);
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

.transaction-scroll-container {
  max-height: 500px;
  overflow-y: auto;
  padding-right: var(--space-2);
}

.transaction-scroll-container::-webkit-scrollbar {
  width: 6px;
}

.transaction-scroll-container::-webkit-scrollbar-track {
  background: var(--bg-tertiary);
  border-radius: 3px;
}

.transaction-scroll-container::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 3px;
}

.transaction-scroll-container::-webkit-scrollbar-thumb:hover {
  background: var(--text-tertiary);
}

.transaction-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  transition: background var(--transition-base);
}

.transaction-row:hover {
  background: var(--bg-tertiary);
}

.transaction-row:not(:last-child) {
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

.text-income {
  color: var(--income);
}

.text-expense {
  color: var(--expense);
}

.bg-income-light {
  background: var(--income-light, rgba(107, 155, 122, 0.15));
}

.bg-expense-light {
  background: var(--expense-light, rgba(194, 123, 123, 0.15));
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
}

.page-info {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.current-page {
  font-weight: 600;
  color: var(--brand-primary);
}

.btn-sm {
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-xs);
}
</style>
