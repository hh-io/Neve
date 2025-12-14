<template>
  <div class="animate-fade-in-up">
    <!-- Filters -->
    <div class="card-static section-mb" style="padding: var(--space-4);">
      <div style="display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-4);">
        <!-- Search -->
        <div style="flex: 1; min-width: 200px; position: relative;">
          <span v-html="icons.search" style="position: absolute; left: 12px; top: 50%; transform: translateY(-50%); width: 18px; height: 18px; stroke: var(--text-tertiary);"></span>
          <input
            type="text"
            v-model="searchQuery"
            placeholder="搜索交易..."
            style="width: 100%; padding: var(--space-2) var(--space-3) var(--space-2) 40px; border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--bg-tertiary); color: var(--text-primary); font-size: var(--font-size-sm); outline: none; transition: border-color var(--transition-base);"
          />
        </div>

        <!-- Category Filter -->
        <select
          v-model="categoryFilter"
          style="padding: var(--space-2) var(--space-3); border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--bg-tertiary); color: var(--text-primary); font-size: var(--font-size-sm); min-width: 120px; outline: none;"
        >
          <option value="">所有分类</option>
          <option v-for="cat in categories" :key="cat" :value="cat">{{ cat }}</option>
        </select>

        <!-- Type Filter -->
        <select
          v-model="typeFilter"
          style="padding: var(--space-2) var(--space-3); border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--bg-tertiary); color: var(--text-primary); font-size: var(--font-size-sm); min-width: 100px; outline: none;"
        >
          <option value="">全部类型</option>
          <option value="expense">支出</option>
          <option value="income">收入</option>
          <option value="transfer">转账</option>
        </select>

        <!-- Reset -->
        <button v-if="hasFilters" class="btn btn-ghost" @click="resetFilters">
          清除筛选
        </button>
      </div>
    </div>

    <!-- Transaction List -->
    <div class="card-static" style="padding: var(--space-6);">
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4);">
        <div style="display: flex; align-items: center; gap: var(--space-3);">
          <div class="stat-icon bg-brand-light" style="width: 40px; height: 40px;">
            <span v-html="icons.transactions" style="stroke: var(--brand-primary); width: 20px; height: 20px;"></span>
          </div>
          <span style="font-weight: 600; color: var(--text-primary);">交易记录</span>
        </div>
        <span style="font-size: var(--font-size-sm); color: var(--text-tertiary);">
          共 {{ filteredTransactions.length }} 条
        </span>
      </div>

      <div v-if="paginatedTransactions.length === 0" style="text-align: center; padding: var(--space-8); color: var(--text-tertiary);">
        暂无匹配的交易记录
      </div>

      <div v-else style="display: flex; flex-direction: column; gap: var(--space-2);">
        <div
          v-for="(tx, index) in paginatedTransactions"
          :key="`${tx.date}-${tx.amount}-${index}`"
          class="transaction-item animate-fade-in-up"
          :style="{ animationDelay: `${index * 0.03}s` }"
        >
          <div :class="['transaction-icon', tx.amount >= 0 ? 'bg-income-light' : 'bg-expense-light']">
            <span v-html="getCategoryIcon(tx.category)" :style="{ stroke: tx.amount >= 0 ? 'var(--income)' : 'var(--expense)' }"></span>
          </div>
          <div class="transaction-info">
            <div class="transaction-title">{{ tx.payee || tx.description || tx.category }}</div>
            <div class="transaction-date">
              {{ tx.category }} · {{ formatDate(tx.date) }}
            </div>
          </div>
          <div class="transaction-amount" :class="tx.amount >= 0 ? 'text-income' : 'text-expense'">
            {{ tx.amount >= 0 ? '+' : '' }}{{ formatMoney(tx.amount) }}
          </div>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" style="display: flex; align-items: center; justify-content: center; gap: var(--space-2); margin-top: var(--space-6); padding-top: var(--space-4); border-top: 1px solid var(--border);">
        <button
          class="btn btn-secondary"
          :disabled="currentPage === 1"
          @click="currentPage--"
        >
          上一页
        </button>
        <span style="padding: 0 var(--space-4); font-size: var(--font-size-sm); color: var(--text-secondary);">
          {{ currentPage }} / {{ totalPages }}
        </span>
        <button
          class="btn btn-secondary"
          :disabled="currentPage === totalPages"
          @click="currentPage++"
        >
          下一页
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { formatMoney, formatDate } from '../../composables/useFormatters';
import { icons } from '../../composables/icons';

const props = defineProps({
  transactions: { type: Array, required: true }
});

// Filters
const searchQuery = ref('');
const categoryFilter = ref('');
const typeFilter = ref('');
const currentPage = ref(1);
const pageSize = 20;

const categories = computed(() => {
  const cats = new Set(props.transactions.map(t => t.category));
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
  let result = props.transactions;
  
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase();
    result = result.filter(t => 
      (t.payee && t.payee.toLowerCase().includes(q)) ||
      (t.description && t.description.toLowerCase().includes(q)) ||
      (t.category && t.category.toLowerCase().includes(q))
    );
  }
  
  if (categoryFilter.value) {
    result = result.filter(t => t.category === categoryFilter.value);
  }
  
  if (typeFilter.value) {
    if (typeFilter.value === 'expense') result = result.filter(t => t.amount < 0);
    else if (typeFilter.value === 'income') result = result.filter(t => t.amount > 0);
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
    '餐饮': icons.food,
    '购物': icons.shopping,
    '交通': icons.transfer,
    '工资': icons.wallet,
  };
  return iconMap[category] || icons.creditCard;
}
</script>
