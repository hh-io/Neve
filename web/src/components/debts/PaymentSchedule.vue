<template>
  <section class="section-card">
    <div class="section-head">
      <h3 class="section-title"><CalendarClock :size="16" class="sec-ic" />未来还款计划</h3>
      <span class="section-count">{{ months.length }} 个月</span>
    </div>
    <div class="section-body">
      <p class="section-sub ps-hint">
        仅含已配置分期的确定性出账,不含循环账单当期余额与未来新增消费。
      </p>

      <div v-if="!hasAnyEntry" class="empty-state">
        配置信用卡分期或固定分期后,这里会展开未来每月要还多少
      </div>

      <div v-else class="ps-list">
        <div v-for="m in months" :key="m.month" class="ps-row" :class="{ 'ps-row-empty': !m.total }">
          <button
            class="ps-head"
            :disabled="!m.entries.length"
            :aria-expanded="expanded === m.month"
            @click="toggle(m.month)"
          >
            <ChevronRight
              :size="14"
              class="ps-caret"
              :class="{ 'ps-caret-open': expanded === m.month }"
            />
            <span class="ps-month">{{ monthLabel(m.month) }}</span>
            <span class="ps-bar">
              <!-- 宽度是数据驱动的运行时值,只能内联绑定 -->
              <span class="ps-bar-fill" :style="{ width: barWidth(m.total) }"></span>
            </span>
            <span class="ps-total tabular-nums">{{ formatMoney(m.total) }}</span>
            <span class="ps-cumulative tabular-nums">累计 {{ formatMoney(m.cumulative) }}</span>
          </button>

          <div v-if="expanded === m.month" class="ps-entries">
            <div v-for="(e, i) in m.entries" :key="e.account + e.name + i" class="ps-entry">
              <span class="ps-entry-date tabular-nums">{{ shortDate(e.dueDate) }}</span>
              <span class="ps-entry-name">{{ e.name }}</span>
              <span v-if="e.longTerm" class="debt-badge badge-idle">长期</span>
              <!-- 固定分期的账单名本身就唯一,再补账户短名只是噪音;额度分期要靠卡名区分 -->
              <span v-if="e.source === 'revolving'" class="ps-entry-account">
                {{ e.accountName }}
              </span>
              <span class="ps-entry-amount tabular-nums">{{ formatMoney(e.amount) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { CalendarClock, ChevronRight } from '@lucide/vue'
import type { ScheduleMonth } from '../../types/api'
import { formatMoney } from '../../composables/useFormatters'
import { shortDate } from '../../composables/useDebtDisplay'

const props = defineProps<{ months: ScheduleMonth[] }>()

// 同时只展开一个月:纯局部展示态,不必提到 composable
const expanded = ref<string | null>(null)

const hasAnyEntry = computed(() => props.months.some((m) => m.entries.length > 0))

// 条形按窗口内最大月对齐,让"哪几个月压力大"一眼可辨
const peak = computed(() => Math.max(...props.months.map((m) => m.total), 0))

function barWidth(total: number): string {
  if (peak.value <= 0) return '0%'
  return `${(total / peak.value) * 100}%`
}

// 后端月份恒为 YYYY-MM,直接切字符串,不经 Date 以免带上时区偏移
function monthLabel(month: string): string {
  const [year, mon] = month.split('-')
  return `${year} 年 ${Number(mon)} 月`
}

function toggle(month: string) {
  expanded.value = expanded.value === month ? null : month
}
</script>

<style scoped>
.ps-hint {
  margin: 0 0 var(--space-3);
  line-height: 1.6;
}

.ps-list {
  display: flex;
  flex-direction: column;
}

.ps-row {
  border-top: 1px solid var(--hairline);
}

.ps-row:first-child {
  border-top: none;
}

.ps-row-empty {
  opacity: 0.5;
}

.ps-head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  padding: var(--space-2) 0;
  background: none;
  border: none;
  text-align: left;
  cursor: pointer;
  color: inherit;
}

.ps-head:disabled {
  cursor: default;
}

.ps-caret {
  flex: none;
  color: var(--text-tertiary);
  transition: transform 0.15s ease;
}

.ps-caret-open {
  transform: rotate(90deg);
}

.ps-head:disabled .ps-caret {
  visibility: hidden;
}

.ps-month {
  flex: none;
  width: 6.5em;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
}

.ps-bar {
  flex: 1;
  min-width: 0;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--surface-3);
  overflow: hidden;
}

.ps-bar-fill {
  display: block;
  height: 100%;
  border-radius: var(--radius-full);
  background: var(--accent);
  transition: width 0.4s ease;
}

.ps-total {
  flex: none;
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
}

.ps-cumulative {
  flex: none;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ps-entries {
  display: flex;
  flex-direction: column;
  padding: 0 0 var(--space-3) var(--space-6);
}

.ps-entry {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding: var(--space-1) 0;
}

.ps-entry-date {
  flex: none;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ps-entry-name {
  flex: 1;
  min-width: 0;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ps-entry-account {
  flex: none;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  white-space: nowrap;
}

.ps-entry-amount {
  flex: none;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

@media (max-width: 640px) {
  .ps-cumulative {
    display: none;
  }

  .ps-month {
    width: 5.5em;
  }
}
</style>
