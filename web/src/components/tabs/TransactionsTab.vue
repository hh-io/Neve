<template>
  <div class="animate-fade-in-up">
    <!-- Filters -->
    <div class="card-static section-mb" style="padding: var(--space-4);">
      <div style="display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-3);">
        <!-- Search -->
        <div style="flex: 1; min-width: 200px; position: relative;">
          <Search :size="16" color="var(--text-tertiary)" style="position: absolute; left: 12px; top: 50%; transform: translateY(-50%);" />
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
          <option value="transfer">转账</option>
        </select>

        <!-- Reset -->
        <button v-if="hasFilters" class="btn btn-ghost" @click="resetFilters" style="font-size: var(--font-size-sm);">
          清除
        </button>
      </div>
    </div>

    <!-- Transaction List -->
    <div class="card-static" style="padding: var(--space-4); height: calc(100vh - 340px); min-height: 350px; display: flex; flex-direction: column;">
      <!-- Header -->
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-3); padding-bottom: var(--space-3); border-bottom: 1px solid var(--border);">
        <div style="display: flex; align-items: center; gap: var(--space-2);">
          <div class="stat-icon bg-brand-light" style="width: 32px; height: 32px; display: flex; align-items: center; justify-content: center;">
            <ArrowRightLeft :size="16" color="var(--brand-primary)" />
          </div>
          <span style="font-weight: 600; color: var(--text-primary); font-size: var(--font-size-base);">交易明细</span>
        </div>
        <span class="badge">共 {{ filteredTransactions.length }} 条</span>
      </div>

      <!-- Empty State -->
      <div v-if="processedTransactions.length === 0" style="text-align: center; padding: var(--space-8); color: var(--text-tertiary); flex: 1; display: flex; align-items: center; justify-content: center;">
        暂无匹配的交易记录
      </div>

      <!-- Scrollable Transaction List -->
      <div v-else class="transaction-scroll-container">
        <div v-for="group in groupedTransactions" :key="group.dateLabel" class="tx-group">
          <!-- Date Header -->
          <div class="tx-date-header">
            <span class="date-label">{{ group.dateLabel }}</span>
            <span class="date-total">
              <span v-if="group.income > 0" class="date-income-text">+¥{{ group.income.toFixed(2) }}</span>
              <span v-if="group.expense > 0" class="date-expense-text ml-2">-¥{{ group.expense.toFixed(2) }}</span>
            </span>
          </div>

          <!-- Transactions in Group -->
          <div
            v-for="(tx, index) in group.items"
            :key="`${tx.date}-${index}`"
            class="transaction-row"
          >
            <!-- Icon -->
            <div :class="['tx-icon', tx.iconClass]">
              <component
                :is="getCategoryIcon(tx.category)"
                :size="20"
                :color="tx.iconColor"
              />
            </div>
            
            <!-- Main Content -->
            <div class="tx-content">
              <!-- Top Row: Payee & Tags -->
              <div class="tx-top">
                <span class="tx-payee">{{ tx.payee || '未知交易' }}</span>
                <div class="tx-tags">
                  <span 
                    v-for="tag in (tx.tags || [])" 
                    :key="tag" 
                    class="tx-tag"
                    :style="{ backgroundColor: getTagColor(tag), color: 'var(--text-secondary)' }"
                  >#{{ tag }}</span>
                </div>
              </div>
              
              <!-- Bottom Row: Category, Narration -->
              <div class="tx-bottom">
                <span class="tx-category-badge">{{ tx.isTransfer ? '转账' : getCategoryLabel(tx.category) }}</span>
                <span v-if="tx.narration" class="tx-narration">{{ tx.narration }}</span>
              </div>
            </div>
            
            <!-- Amount & Account (Right Side) -->
            <div class="tx-amount-col">
              <div :class="['tx-amount', tx.amountClass]">{{ tx.amountText }}</div>
              <div v-if="tx.isTransfer && tx.feeAmount > 0" class="tx-fee">手续费 ¥{{ tx.feeAmount.toFixed(2) }}</div>
              <div class="tx-account">{{ tx.accountShort }}</div>
            </div>
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
import { Search, ArrowRightLeft } from '@lucide/vue';
import {
  getCategoryLabel,
  processTransaction,
  getRelativeDateLabel,
  getTagColor
} from '../../composables/useCategories';
import { getCategoryIcon } from '../../composables/useCategoryIcon';

const props = defineProps({
  transactions: { type: Array, required: true }
});

// Filters
const searchQuery = ref('');
const categoryFilter = ref('');
const typeFilter = ref('');
const currentPage = ref(1);
const pageSize = 20;

// Process transactions using shared utility
const processedTransactions = computed(() => {
  return props.transactions.map(processTransaction);
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
    if (typeFilter.value === 'expense') result = result.filter(t => !t.isIncome && !t.isTransfer);
    else if (typeFilter.value === 'income') result = result.filter(t => t.isIncome);
    else if (typeFilter.value === 'transfer') result = result.filter(t => t.isTransfer);
  }
  
  return result;
});

const totalPages = computed(() => Math.ceil(filteredTransactions.value.length / pageSize));

const paginatedTransactions = computed(() => {
  const start = (currentPage.value - 1) * pageSize;
  return filteredTransactions.value.slice(start, start + pageSize);
});

// Group paginated transactions by date
const groupedTransactions = computed(() => {
  const groups = {};
  
  paginatedTransactions.value.forEach(tx => {
    const dateStr = tx.date;
    if (!groups[dateStr]) {
      groups[dateStr] = {
        dateStr,
        dateLabel: getRelativeDateLabel(dateStr),
        items: [],
        income: 0,
        expense: 0
      };
    }
    groups[dateStr].items.push(tx);
    // 日合计:转账只计手续费,退款(负支出)冲减当日支出
    if (tx.isTransfer) {
      groups[dateStr].expense += tx.feeAmount || 0;
    } else if (tx.isIncome) {
      groups[dateStr].income += Math.abs(tx.amount);
    } else {
      groups[dateStr].expense += tx.amount;
    }
  });

  return Object.values(groups).sort((a, b) => {
    return new Date(b.dateStr) - new Date(a.dateStr);
  });
});

watch([searchQuery, categoryFilter, typeFilter], () => {
  currentPage.value = 1;
});
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
  overflow-y: auto;
  padding-right: var(--space-2);
  flex: 1;
  /* Ensure a minimum height if flex content is small, but flex:1 usually handles it */
  min-height: 0; 
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

/* Group & Headers */
.tx-group {
  margin-bottom: var(--space-4);
}

.tx-date-header {
  position: sticky;
  top: 0;
  background-color: var(--bg-secondary);
  z-index: 10;
  padding: var(--space-2) var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  font-weight: 500;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.date-total {
  font-size: 11px;
}

.date-income-text {
  color: rgba(107, 155, 122, 0.8); /* Low saturation income */
}

.date-expense-text {
  color: rgba(194, 123, 123, 0.8); /* Low saturation expense */
}

.ml-2 { margin-left: 8px; }

/* Transaction Row */
.transaction-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  transition: background var(--transition-base);
  margin-bottom: 2px;
}

.transaction-row:hover {
  background: var(--bg-tertiary);
}

.tx-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tx-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tx-top {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.tx-payee {
  font-weight: 500;
  color: var(--text-primary);
  font-size: var(--font-size-base);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-tags {
  display: flex;
  gap: 4px;
}

.tx-tag {
  padding: 0px 4px;
  border-radius: var(--radius-sm);
  font-size: 10px;
  white-space: nowrap;
}

.tx-bottom {
  display: flex;
  align-items: center;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-category-badge {
  padding: 1px 6px;
  background: var(--bg-tertiary); /* User requested background */
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  margin-right: 6px;
  font-size: 11px;
}

.tx-narration {
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-dot {
  margin: 0 4px;
  color: var(--border);
}

.tx-account {
  font-size: 11px;
  color: var(--text-tertiary);
  margin-top: 2px;
}

.tx-amount-col {
  text-align: right;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-end; /* Align amount and account to right */
}

.tx-amount {
  font-weight: 600;
  font-size: var(--font-size-base);
  font-feature-settings: 'tnum';
}

.text-income {
  color: var(--income);
}

.text-expense {
  color: var(--expense);
}

.text-transfer {
  color: var(--text-secondary);
}

.bg-income-light {
  background: var(--income-light, rgba(107, 155, 122, 0.15));
}

.bg-expense-light {
  background: var(--expense-light, rgba(194, 123, 123, 0.15));
}

.bg-brand-light {
  background: var(--brand-light);
}

.tx-fee {
  font-size: 10px;
  color: var(--text-tertiary);
}

/* Pagination */
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
