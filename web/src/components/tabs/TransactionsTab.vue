<template>
  <div class="animate-fade-in-up tx-layout">
    <!-- 左列:筛选 + 按日分组 -->
    <div class="tx-main">
      <!-- 筛选行 -->
      <div class="tx-filters">
        <div class="filter-pills">
          <button
            v-for="opt in typeOptions"
            :key="opt.value"
            class="filter-pill"
            :class="{ active: typeFilter === opt.value }"
            @click="typeFilter = opt.value"
          >{{ opt.label }}</button>
        </div>
        <div class="tx-filters-spacer"></div>
        <div class="tx-select-wrap">
          <select v-model="categoryFilter" class="tx-select">
            <option value="">所有分类</option>
            <option v-for="cat in categories" :key="cat" :value="cat">{{ getCategoryLabel(cat) }}</option>
          </select>
          <ChevronDown :size="14" class="tx-select-caret" />
        </div>
        <div class="tx-search">
          <Search :size="15" class="tx-search-icon" />
          <input v-model="searchQuery" type="text" placeholder="搜索交易…" class="tx-search-input" />
        </div>
      </div>

      <!-- 空态 -->
      <div v-if="processedTransactions.length === 0" class="tx-empty">暂无匹配的交易记录</div>

      <!-- 按日分组 -->
      <template v-else>
        <section v-for="group in groupedTransactions" :key="group.dateStr" class="section-card">
          <div class="tx-day-head">
            <span class="tx-day-date tabular-nums">{{ group.dateLabel }}</span>
            <span class="tx-day-sum tabular-nums">支出 ¥{{ group.expense.toFixed(0) }} · 收入 ¥{{ group.income.toFixed(0) }}</span>
          </div>
          <div>
            <div
              v-for="(tx, index) in group.items"
              :key="`${tx.date}-${index}`"
              class="tx-row"
            >
              <div class="tx-row-icon" :class="tx.iconClass">
                <component :is="getCategoryIcon(tx.category)" :size="16" :color="tx.iconColor" />
              </div>
              <div class="tx-row-info">
                <div class="tx-row-name">{{ tx.payee || '未知交易' }}</div>
                <div class="tx-row-meta">
                  <span class="tx-row-cat">{{ tx.isTransfer ? '转账' : getCategoryLabel(tx.category) }}</span>
                  <span v-if="tx.narration" class="tx-row-narr">{{ tx.narration }}</span>
                  <span class="tx-row-acct">{{ tx.accountShort }}</span>
                </div>
              </div>
              <div class="tx-row-amount-col">
                <div class="tx-row-amount tabular-nums" :class="tx.amountClass">{{ tx.amountText }}</div>
                <div v-if="tx.isTransfer && tx.feeAmount > 0" class="tx-row-fee tabular-nums">手续费 ¥{{ tx.feeAmount.toFixed(2) }}</div>
              </div>
            </div>
          </div>
        </section>

        <!-- 分页 -->
        <div v-if="totalPages > 1" class="tx-pagination">
          <button class="btn btn-secondary btn-sm" :disabled="currentPage === 1" @click="currentPage--">‹ 上一页</button>
          <div class="tx-page-info tabular-nums">
            <span class="tx-current-page">{{ currentPage }}</span>
            <span>/</span>
            <span>{{ totalPages }}</span>
          </div>
          <button class="btn btn-secondary btn-sm" :disabled="currentPage === totalPages" @click="currentPage++">下一页 ›</button>
        </div>
      </template>
    </div>

    <!-- 右列:粘性月历 -->
    <section class="section-card tx-cal">
      <div class="section-head">
        <h3 class="section-title"><Calendar :size="16" class="sec-ic" />交易日历</h3>
      </div>
      <div class="section-body">
        <TransactionCalendar :dailyData="dailyTrend" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import { Search, ChevronDown, Calendar } from '@lucide/vue';
import type { ProcessedTransaction } from '../../composables/useCategories';
import {
  getCategoryLabel,
  processTransaction,
  getRelativeDateLabel,
} from '../../composables/useCategories';
import { getCategoryIcon } from '../../composables/useCategoryIcon';
import { useAnalytics } from '../../composables/useAnalytics';
import TransactionCalendar from '../TransactionCalendar.vue';

const { analytics } = useAnalytics();

const dailyTrend = computed(() => analytics.value?.dailyTrend || []);

interface DateGroup {
  dateStr: string;
  dateLabel: string;
  items: ProcessedTransaction[];
  income: number;
  expense: number;
}

const typeOptions = [
  { value: '', label: '全部' },
  { value: 'income', label: '收入' },
  { value: 'expense', label: '支出' },
  { value: 'transfer', label: '转账' }
] as const;

// Filters
const searchQuery = ref('');
const categoryFilter = ref('');
const typeFilter = ref('');
const currentPage = ref(1);
const pageSize = 20;

// Process transactions using shared utility
const processedTransactions = computed(() => {
  return (analytics.value?.transactions ?? []).map(processTransaction);
});

const categories = computed(() => {
  const cats = new Set(processedTransactions.value.map(t => t.category));
  return Array.from(cats).sort();
});

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
const groupedTransactions = computed<DateGroup[]>(() => {
  const groups: Record<string, DateGroup> = {};

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
    return new Date(b.dateStr).getTime() - new Date(a.dateStr).getTime();
  });
});

watch([searchQuery, categoryFilter, typeFilter], () => {
  currentPage.value = 1;
});
</script>

<style scoped>
.tx-layout {
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: var(--space-4);
  align-items: start;
}

.tx-main {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}

/* ===== 筛选行 ===== */
.tx-filters {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.tx-filters-spacer { flex: 1; }

.tx-select-wrap {
  position: relative;
}

.tx-select {
  appearance: none;
  padding: 6px 28px 6px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--hairline);
  background: var(--surface-1);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  cursor: pointer;
}

.tx-select:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-subtle);
}

.tx-select-caret {
  position: absolute;
  right: 9px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
  pointer-events: none;
}

.tx-search {
  position: relative;
  display: flex;
  align-items: center;
}

.tx-search-icon {
  position: absolute;
  left: 12px;
  color: var(--text-tertiary);
  pointer-events: none;
}

.tx-search-input {
  padding: 6px 12px 6px 34px;
  border-radius: var(--radius-md);
  border: 1px solid var(--hairline);
  background: var(--surface-1);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  width: 180px;
}

.tx-search-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-subtle);
}

/* ===== 日分组 ===== */
.tx-day-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-5);
  border-bottom: 1px solid var(--hairline);
  background: var(--surface-2);
}

.tx-day-date {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
}

.tx-day-sum {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.tx-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-5);
  border-bottom: 1px solid var(--hairline);
  transition: background var(--transition-base);
}

.tx-row:last-child { border-bottom: none; }
.tx-row:hover { background: var(--surface-2); }

.tx-row-icon {
  width: 34px;
  height: 34px;
  flex: none;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}

.tx-row-info {
  flex: 1;
  min-width: 0;
}

.tx-row-name {
  font-size: var(--font-size-sm);
  font-weight: 550;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-row-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-row-cat {
  padding: 1px 6px;
  background: var(--surface-3);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
}

.tx-row-narr {
  overflow: hidden;
  text-overflow: ellipsis;
}

.tx-row-acct { color: var(--text-tertiary); }

.tx-row-amount-col {
  flex: none;
  text-align: right;
}

.tx-row-amount {
  font-size: var(--font-size-sm);
  font-weight: 600;
}

.tx-row-fee {
  font-size: 10px;
  color: var(--text-tertiary);
}

/* 交易类型色(图标底 + 金额字) */
.bg-income-light { background: var(--income-light); }
.bg-expense-light { background: var(--expense-light); }
.bg-brand-light { background: var(--accent-subtle); }
.text-income { color: var(--income); }
.text-expense { color: var(--expense); }
.text-transfer { color: var(--text-secondary); }

.tx-empty {
  padding: var(--space-8);
  text-align: center;
  color: var(--text-tertiary);
}

/* ===== 分页 ===== */
.tx-pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  padding-top: var(--space-2);
}

.tx-page-info {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.tx-current-page {
  font-weight: 600;
  color: var(--accent);
}

.btn-sm {
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-xs);
}

/* ===== 粘性日历 ===== */
.tx-cal {
  position: sticky;
  top: var(--space-5);
}

@media (max-width: 1024px) {
  .tx-layout { grid-template-columns: 1fr; }
  .tx-cal { position: static; }
}
</style>
