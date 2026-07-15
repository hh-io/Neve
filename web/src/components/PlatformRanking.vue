<template>
  <div>
    <div class="ranking-list" v-if="data?.length">
      <div 
        v-for="(item, index) in data" 
        :key="item.tag" 
        class="ranking-item"
      >
        <div class="rank-info">
          <span class="rank-num" :class="getRankClass(index)">{{ index + 1 }}</span>
          <span class="rank-tag">#{{ item.tag }}</span>
        </div>
        <div class="rank-bar-wrap">
          <div 
            class="rank-bar" 
            :style="{ width: (item.amount / maxAmount * 100) + '%', opacity: getBarOpacity(index) }"
          ></div>
        </div>
        <div class="rank-amount">¥{{ formatAmount(item.amount) }}</div>
      </div>
    </div>
    <div v-else class="empty-state">暂无数据</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { TagStats } from '../types/api';

const props = withDefaults(defineProps<{
  data?: TagStats[];
}>(), {
  data: () => []
});

const maxAmount = computed(() => {
  return props.data.length ? props.data[0].amount : 1;
});

function getRankClass(index: number): string {
  if (index === 0) return 'gold';
  if (index === 1) return 'silver';
  if (index === 2) return 'bronze';
  return '';
}

function formatAmount(amount: number): string {
  return amount >= 1000 ? (amount / 1000).toFixed(1) + 'k' : amount.toFixed(0);
}

function getBarOpacity(index: number): number {
  // 第一名 1.0，之后每名递减 0.12,最低 0.4
  return Math.max(0.4, 1 - index * 0.12);
}
</script>

<style scoped>
.ranking-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.ranking-item {
  display: grid;
  grid-template-columns: 100px 1fr 60px;
  align-items: center;
  gap: var(--space-3);
}

.rank-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.rank-num {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  background: var(--surface-2);
  color: var(--text-secondary);
}

.rank-num.gold { background: linear-gradient(135deg, #FFD700, #FFA500); color: white; }
.rank-num.silver { background: linear-gradient(135deg, #C0C0C0, #A0A0A0); color: white; }
.rank-num.bronze { background: linear-gradient(135deg, #CD7F32, #B8860B); color: white; }

.rank-tag {
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-primary);
}

.rank-bar-wrap {
  height: 8px;
  background: var(--surface-2);
  border-radius: 4px;
  overflow: hidden;
}

.rank-bar {
  height: 100%;
  background: var(--info);
  border-radius: 4px;
  transition: width 0.5s ease;
}

.rank-amount {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-secondary);
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.empty-state {
  text-align: center;
  padding: var(--space-8);
  color: var(--text-tertiary);
}
</style>
