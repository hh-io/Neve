<template>
  <div class="tab-content">
    <section class="transactions-section fade-in">
      <div class="glass-card">
        <div class="transactions-header">
          <h3 class="card-title">交易明细</h3>
          <form class="transactions-filters" @submit.prevent>
            <div class="search-box">
              <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"></circle>
                <path d="m21 21-4.35-4.35"></path>
              </svg>
              <input type="text" v-model.lazy="searchQuery" placeholder="搜索交易..." class="search-input" @keydown.enter.prevent />
            </div>
            <select v-model="categoryFilter" class="category-select">
              <option value="">全部分类</option>
              <option v-for="cat in expenseCategories" :key="cat" :value="cat">{{ cat }}</option>
            </select>
            <select v-model="tagFilter" class="category-select">
              <option value="">全部标签</option>
              <option v-for="tag in availableTags" :key="tag" :value="tag">#{{ tag }}</option>
            </select>
            <div class="date-range">
              <input type="date" v-model="dateStart" class="date-input" :max="dateEnd || undefined" />
              <span class="date-separator">至</span>
              <input type="date" v-model="dateEnd" class="date-input" :min="dateStart || undefined" />
            </div>
          </form>
        </div>
        
        <div class="transactions-table" v-if="paginatedTransactions.length">
          <div class="table-header">
            <span class="col-date">日期</span>
            <span class="col-desc">描述</span>
            <span class="col-category">分类</span>
            <span class="col-tags">标签</span>
            <span class="col-amount">金额</span>
          </div>
          <div v-for="tx in paginatedTransactions" :key="tx.date + tx.narration + (tx.postings?.[0]?.amount || 0)" class="table-row">
            <span class="col-date">{{ formatDate(tx.date) }}</span>
            <span class="col-desc">
              <span class="tx-payee" v-if="tx.payee">{{ tx.payee }}</span>
              <span class="tx-narration">{{ tx.narration || '未命名' }}</span>
            </span>
            <span class="col-category">{{ getTransactionCategory(tx) }}</span>
            <span class="col-tags">
              <span v-for="tag in tx.tags" :key="tag" class="tx-tag" :class="getTagClass(tag)">#{{ tag }}</span>
            </span>
            <span class="col-amount" :class="getTransactionAmountClass(tx)">{{ formatTransactionAmount(tx) }}</span>
          </div>
        </div>
        <div v-else class="empty-state">
          <p>没有找到匹配的交易记录</p>
        </div>

        <!-- Pagination -->
        <div class="pagination" v-if="totalPages > 1">
          <button class="page-btn" :disabled="currentPage === 1" @click="currentPage--">上一页</button>
          <span class="page-info">{{ currentPage }} / {{ totalPages }}</span>
          <button class="page-btn" :disabled="currentPage === totalPages" @click="currentPage++">下一页</button>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { formatDate, getTransactionCategory, getTransactionAmountClass, formatTransactionAmount, getTagClass } from '../../composables/useFormatters';

const props = defineProps({
  transactions: { type: Array, default: () => [] }
});

const searchQuery = ref('');
const categoryFilter = ref('');
const tagFilter = ref('');
const dateStart = ref('');
const dateEnd = ref('');
const currentPage = ref(1);
const pageSize = 10;

// Reset page when filters change
watch([searchQuery, categoryFilter, tagFilter, dateStart, dateEnd], () => {
  currentPage.value = 1;
});

const expenseCategories = computed(() => {
  const cats = new Set();
  props.transactions.forEach(tx => {
    const cat = getTransactionCategory(tx);
    if (cat && cat !== '-') cats.add(cat);
  });
  return Array.from(cats).sort();
});

const availableTags = computed(() => {
  const tags = new Set();
  props.transactions.forEach(tx => {
    tx.tags?.forEach(tag => tags.add(tag));
  });
  return Array.from(tags).sort();
});

const filteredTransactions = computed(() => {
  let result = props.transactions || [];
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase();
    result = result.filter(tx => 
      (tx.narration?.toLowerCase().includes(q)) ||
      (tx.payee?.toLowerCase().includes(q)) ||
      tx.tags?.some(t => t.toLowerCase().includes(q))
    );
  }
  if (categoryFilter.value) {
    result = result.filter(tx => getTransactionCategory(tx) === categoryFilter.value);
  }
  if (tagFilter.value) {
    result = result.filter(tx => tx.tags?.includes(tagFilter.value));
  }
  if (dateStart.value) {
    result = result.filter(tx => tx.date >= dateStart.value);
  }
  if (dateEnd.value) {
    result = result.filter(tx => tx.date <= dateEnd.value);
  }
  return result;
});

const totalPages = computed(() => Math.ceil(filteredTransactions.value.length / pageSize));

const paginatedTransactions = computed(() => {
  const start = (currentPage.value - 1) * pageSize;
  return filteredTransactions.value.slice(start, start + pageSize);
});
</script>
