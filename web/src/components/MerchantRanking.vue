<template>
  <div>
    <div v-if="data?.length" class="ranking-list">
      <div v-for="(item, index) in data" :key="item.payee" class="ranking-item">
        <span class="rank-num">{{ index + 1 }}</span>
        <div class="rank-icon">
          <component :is="iconFor(item.payee)" :size="15" />
        </div>
        <div class="rank-body">
          <div class="rank-line">
            <span class="rank-name">{{ item.payee }}</span>
            <span class="rank-amount">¥{{ formatAmount(item.amount) }}</span>
          </div>
          <div class="rank-bar-wrap">
            <div
              class="rank-bar"
              :style="{ width: (item.amount / maxAmount * 100) + '%', background: colorFor(index) }"
            ></div>
          </div>
        </div>
      </div>
    </div>
    <div v-else class="empty-state">暂无数据</div>
  </div>
</template>

<script setup lang="ts">
import { computed, type FunctionalComponent } from 'vue';
import { Store, Utensils, Coffee, ShoppingCart, Car, Fuel, Pill, Shirt } from '@lucide/vue';
import type { PayeeStats } from '../types/api';
import { getThemeColor } from '../composables/useThemeColor';

const props = withDefaults(defineProps<{
  data?: PayeeStats[];
}>(), {
  data: () => []
});

const maxAmount = computed(() => {
  return props.data.length ? props.data[0].amount : 1;
});

// 商户名关键词 → lucide 图标(命中不了走通用店铺)
function iconFor(payee: string): FunctionalComponent {
  const p = payee.toLowerCase();
  if (/餐|饭|食|烤|面|火锅|外卖|美团|饿了么/.test(p)) return Utensils;
  if (/咖啡|星巴克|瑞幸|茶|奶茶|coffee/.test(p)) return Coffee;
  if (/超市|便利|沃尔玛|盒马|华润|永辉|mart/.test(p)) return ShoppingCart;
  if (/加油|石油|石化|中石/.test(p)) return Fuel;
  if (/滴滴|出租|打车|地铁|高铁|交通/.test(p)) return Car;
  if (/药|医|诊所|医院/.test(p)) return Pill;
  if (/优衣库|zara|服|鞋|uniqlo/.test(p)) return Shirt;
  return Store;
}

const palette = ['--chart-1', '--chart-2', '--chart-3', '--chart-4', '--chart-5', '--chart-6', '--chart-7', '--chart-8'];
function colorFor(index: number): string {
  return getThemeColor(palette[index % palette.length]);
}

function formatAmount(amount: number): string {
  return amount >= 1000 ? (amount / 1000).toFixed(1) + 'k' : amount.toFixed(0);
}
</script>

<style scoped>
.ranking-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.ranking-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.rank-num {
  width: 18px;
  flex: none;
  text-align: right;
  font-size: var(--font-size-sm);
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
}

.rank-icon {
  width: 30px;
  height: 30px;
  flex: none;
  border-radius: var(--radius-md);
  background: var(--surface-3);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.rank-body {
  flex: 1;
  min-width: 0;
}

.rank-line {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: var(--space-3);
  font-size: var(--font-size-sm);
  margin-bottom: 5px;
}

.rank-name {
  font-weight: 550;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.rank-amount {
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
  flex: none;
}

.rank-bar-wrap {
  height: 6px;
  background: var(--surface-3);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.rank-bar {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.5s ease;
}

.empty-state {
  text-align: center;
  padding: var(--space-8);
  color: var(--text-tertiary);
}
</style>
