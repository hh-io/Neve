<template>
  <div class="glass-card chart-card">
    <h3 class="card-title">负债概览</h3>
    <div class="liability-list" v-if="data?.length">
      <div 
        v-for="item in data" 
        :key="item.account" 
        class="liability-item"
      >
        <div class="liability-info">
          <span class="liability-name">{{ item.name }}</span>
        </div>
        <div class="liability-bar-wrap">
          <div 
            class="liability-bar" 
            :style="{ width: (item.balance / maxBalance * 100) + '%' }"
          ></div>
        </div>
        <span class="liability-amount">¥{{ formatAmount(item.balance) }}</span>
      </div>
      <div class="liability-total">
        <span>总负债</span>
        <span class="total-amount">¥{{ formatAmount(totalLiability) }}</span>
      </div>
    </div>
    <div v-else class="empty-state">暂无负债</div>
  </div>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  data: { type: Array, default: () => [] }
});

const maxBalance = computed(() => {
  return props.data.length ? Math.max(...props.data.map(d => d.balance)) : 1;
});

const totalLiability = computed(() => {
  return props.data.reduce((sum, d) => sum + d.balance, 0);
});

function formatAmount(amount) {
  return amount >= 1000 
    ? (amount / 1000).toFixed(2) + 'k' 
    : amount.toFixed(2);
}
</script>

<style scoped>
.liability-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.liability-item {
  display: grid;
  grid-template-columns: 80px 1fr 70px;
  align-items: center;
  gap: var(--space-3);
}

.liability-name {
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--color-text-primary);
}

.liability-bar-wrap {
  height: 8px;
  background: var(--color-bg);
  border-radius: 4px;
  overflow: hidden;
}

.liability-bar {
  height: 100%;
  background: linear-gradient(90deg, #FF3B30, #FF9500);
  border-radius: 4px;
  transition: width 0.5s ease;
}

.liability-amount {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--color-red);
  text-align: right;
}

.liability-total {
  display: flex;
  justify-content: space-between;
  padding-top: var(--space-3);
  margin-top: var(--space-2);
  border-top: 1px dashed rgba(0, 0, 0, 0.1);
  font-weight: 600;
}

.total-amount {
  color: var(--color-red);
}

.empty-state {
  text-align: center;
  padding: var(--space-6);
  color: var(--color-text-tertiary);
}
</style>
