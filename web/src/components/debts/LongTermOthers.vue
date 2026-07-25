<template>
  <section v-if="others.length" class="section-card">
    <div class="section-head">
      <h3 class="section-title"><Landmark :size="16" class="sec-ic" />其他负债账户</h3>
      <span class="section-count">{{ others.length }}</span>
    </div>
    <div class="section-body">
      <p class="section-sub lt-hint">
        这些账户没有配账单周期,若属于房贷这类长期负债(对应资产不在账本内),在此勾选即可让概览与账户页的净资产默认口径不再扣减它们。
      </p>
      <div class="lt-list">
        <label v-for="acc in others" :key="acc.account" class="lt-item">
          <input
            type="checkbox"
            class="lt-check"
            :checked="acc.longTerm"
            :disabled="saving"
            @change="toggle(acc.account, ($event.target as HTMLInputElement).checked)"
          />
          <span class="lt-name">{{ acc.name }}</span>
          <span class="lt-account">{{ acc.account }}</span>
          <span class="lt-balance tabular-nums">{{ formatMoney(acc.balance) }}</span>
        </label>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { Landmark } from '@lucide/vue'
import type { DebtsConfig } from '../../types/api'
import { formatMoney } from '../../composables/useFormatters'
import { useAnalytics } from '../../composables/useAnalytics'
import { useDebts } from '../../composables/useDebts'

const { analytics } = useAnalytics()
const { config, saveDebts } = useDebts()

const saving = ref(false)

// 候选取 accountBalances 而非 liabilityBreakdown:后者只含有欠款的账户,
// 已还清的房贷账户仍应能保留勾选,否则重新保存配置会把标记丢掉。
// 已配了账单周期的账户不列在这——它们的开关在各自的卡里。
const others = computed(() => {
  const configured = new Set<string>([
    ...Object.keys(config.value.revolving),
    ...config.value.installments.map((ins) => ins.account),
  ])
  return (analytics.value?.accountBalances ?? [])
    .filter((acc) => acc.type === 'Liabilities' && !configured.has(acc.account))
    .map((acc) => ({
      account: acc.account,
      name: acc.account.split(':').pop() ?? acc.account,
      balance: acc.balance,
      longTerm: config.value.longTermAccounts.includes(acc.account),
    }))
})

// 单一布尔量,没有"取消"语义,勾选即存;保存期间整组禁用,避免连点撞后端限流
async function toggle(account: string, checked: boolean) {
  if (saving.value) return
  saving.value = true
  const next: DebtsConfig = JSON.parse(JSON.stringify(config.value))
  next.longTermAccounts = checked
    ? [...new Set([...next.longTermAccounts, account])]
    : next.longTermAccounts.filter((a) => a !== account)
  await saveDebts(next)
  saving.value = false
}
</script>

<style scoped>
.lt-hint {
  margin: 0 0 var(--space-3);
  line-height: 1.6;
}

.lt-list {
  display: flex;
  flex-direction: column;
}

.lt-item {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding: var(--space-2) 0;
  border-top: 1px solid var(--hairline);
  cursor: pointer;
}

.lt-item:first-child {
  border-top: none;
}

.lt-check {
  accent-color: var(--accent);
  cursor: pointer;
}

.lt-name {
  font-size: var(--font-size-sm);
  color: var(--text-primary);
}

.lt-account {
  flex: 1;
  min-width: 0;
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  font-family: var(--font-numeric);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lt-balance {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}
</style>
