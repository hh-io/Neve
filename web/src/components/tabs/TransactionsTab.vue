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
    <div class="card-static" style="padding: var(--space-4); height: calc(100vh - 220px); min-height: 500px; display: flex; flex-direction: column;">
      <!-- Header -->
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-3); padding-bottom: var(--space-3); border-bottom: 1px solid var(--border);">
        <div style="display: flex; align-items: center; gap: var(--space-2);">
          <div class="stat-icon bg-brand-light" style="width: 32px; height: 32px; display: flex; align-items: center; justify-content: center;">
            <span v-html="icons.transactions" style="stroke: var(--brand-primary); width: 16px; height: 16px; display: flex; align-items: center; justify-content: center;"></span>
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
            <div :class="['tx-icon', tx.isIncome ? 'bg-income-light' : 'bg-expense-light']">
              <span v-html="getCategoryIcon(tx.category)" :style="{ stroke: tx.isIncome ? 'var(--income)' : 'var(--expense)', width: '20px', height: '20px' }"></span>
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
                <span class="tx-category-badge">{{ getCategoryLabel(tx.category) }}</span>
                <span v-if="tx.narration" class="tx-narration">{{ tx.narration }}</span>
              </div>
            </div>
            
            <!-- Amount & Account (Right Side) -->
            <div class="tx-amount-col">
              <div :class="['tx-amount', tx.isIncome ? 'text-income' : 'text-expense']">
                {{ tx.isIncome ? '+' : '-' }}¥{{ Math.abs(tx.amount).toFixed(2) }}
              </div>
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
import { icons } from '../../composables/icons';

const props = defineProps({
  transactions: { type: Array, required: true }
});

// Filters
const searchQuery = ref('');
const categoryFilter = ref('');
const typeFilter = ref('');
const currentPage = ref(1);
const pageSize = 20; // Increased slightly for better density

// Category label mapping
const categoryLabels = {
  Food: '餐饮', Shopping: '购物', Transport: '交通', Entertainment: '娱乐',
  Gift: '红包/礼物', Financial: '金融', Communication: '通讯', Lodging: '住宿',
  Digital: '数码', Health: '健康', Education: '教育', Income: '收入', Other: '其他'
};

function getCategoryLabel(cat) {
  return categoryLabels[cat] || cat || '其他';
}

function getRelativeDateLabel(dateStr) {
  const date = new Date(dateStr);
  const today = new Date();
  const yesterday = new Date();
  yesterday.setDate(today.getDate() - 1);

  // Reset hours to compare just dates
  date.setHours(0,0,0,0);
  today.setHours(0,0,0,0);
  yesterday.setHours(0,0,0,0);

  if (date.getTime() === today.getTime()) {
    return '今天';
  } else if (date.getTime() === yesterday.getTime()) {
    return '昨天';
  } else {
    // Format: MM月DD日 Weekday
    const month = date.getMonth() + 1;
    const day = date.getDate();
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    const weekday = weekdays[date.getDay()];
    return `${month}月${day}日 ${weekday}`;
  }
}

// Low saturation pastel colors for tags
function getTagColor(tag) {
  let hash = 0;
  for (let i = 0; i < tag.length; i++) {
    hash = tag.charCodeAt(i) + ((hash << 5) - hash);
  }
  
  // High lightness, low saturation for pastel background
  const h = hash % 360;
  return `hsl(${h}, 30%, 90%)`;
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
    // Calculate total for header (only for visible items on this page)
    if (tx.isIncome) {
      groups[dateStr].income += Math.abs(tx.amount);
    } else {
      groups[dateStr].expense += Math.abs(tx.amount);
    }
  });

  // Sort groups by date descending
  return Object.values(groups).sort((a, b) => {
    return new Date(b.dateStr) - new Date(a.dateStr);
  });
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

.bg-income-light {
  background: var(--income-light, rgba(107, 155, 122, 0.15));
}

.bg-expense-light {
  background: var(--expense-light, rgba(194, 123, 123, 0.15));
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
