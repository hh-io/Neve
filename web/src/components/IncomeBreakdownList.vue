<template>
  <div>
    <div v-if="data?.length" class="income-list">
      <div v-for="(item, index) in data" :key="item.source" class="income-item">
        <span class="income-rank tabular-nums">{{ index + 1 }}</span>
        <div class="income-body">
          <div class="income-line">
            <span class="income-name">{{ getCategoryLabel(item.source) }}</span>
            <span class="income-amount tabular-nums">{{ formatMoney(item.amount) }}</span>
          </div>
          <div class="income-bar-wrap">
            <div class="income-bar" :style="{ width: barWidth(item.amount) }"></div>
          </div>
          <div class="income-meta">
            <span class="tabular-nums">{{ item.count }} 笔</span>
            <span class="tabular-nums">占比 {{ item.percent.toFixed(0) }}%</span>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="empty-state">本月暂无收入记录</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { IncomeSource } from '../types/api';
import { formatMoney } from '../composables/useFormatters';
import { getCategoryLabel } from '../composables/useCategories';

const props = withDefaults(defineProps<{ data?: IncomeSource[] }>(), { data: () => [] });

// 条宽按最大项归一,而非按占比:来源只有两三项时按占比画条会几乎等长,看不出主次
const maxAmount = computed(() => props.data.reduce((max, item) => Math.max(max, item.amount), 0) || 1);

function barWidth(amount: number): string {
  return `${(amount / maxAmount.value) * 100}%`;
}
</script>

<style scoped>
.income-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.income-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
}

.income-rank {
  width: 18px;
  flex: none;
  text-align: right;
  font-size: var(--font-size-sm);
  color: var(--text-tertiary);
  line-height: 1.4;
}

.income-body {
  flex: 1;
  min-width: 0;
}

.income-line {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: var(--space-3);
  font-size: var(--font-size-sm);
  margin-bottom: 5px;
}

.income-name {
  font-weight: 550;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.income-amount {
  color: var(--income);
  flex: none;
}

.income-bar-wrap {
  height: 6px;
  background: var(--surface-3);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.income-bar {
  height: 100%;
  border-radius: var(--radius-full);
  background: var(--chart-income);
  transition: width 0.5s ease;
}

.income-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 5px;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}
</style>
