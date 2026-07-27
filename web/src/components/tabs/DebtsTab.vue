<template>
  <div class="animate-fade-in-up debts">
    <!-- 全局看板 -->
    <div class="debts-summary">
      <div class="card debt-sum">
        <div class="debt-sum-label">未来 30 天待还</div>
        <div class="debt-sum-value tabular-nums">{{ formatMoney(summary?.due30 ?? 0) }}</div>
        <div class="countdown-sub">账单与分期合计</div>
      </div>
      <div class="card debt-sum">
        <div class="debt-sum-label">未来 90 天待还</div>
        <div class="debt-sum-value tabular-nums">{{ formatMoney(summary?.due90 ?? 0) }}</div>
        <div class="countdown-sub">逐月明细见下方计划表</div>
      </div>
      <div class="card debt-sum" :class="{ 'debt-sum-alert': isOverdue }">
        <div class="debt-sum-label">最近还款日</div>
        <template v-if="!report">
          <div class="debt-sum-value">—</div>
          <div class="countdown-sub">服务端不可达,计算不可用</div>
        </template>
        <template v-else-if="!summary?.nextDueDate">
          <div class="debt-sum-value text-income">本期已结清</div>
        </template>
        <template v-else>
          <div class="debt-sum-value" :class="{ 'text-expense': summary.nextDueInDays < 0 }">
            {{ countdownText }}
          </div>
          <div class="countdown-sub">{{ summary.nextDueName }} · {{ summary.nextDueDate }}</div>
        </template>
        <!-- 逾期的钱已被计划表剔除(due 已过), monthRemaining 是唯一还能反映它的量 -->
        <div v-if="isOverdue" class="countdown-sub text-expense">
          逾期未还 {{ formatMoney(summary?.monthRemaining ?? 0) }}
        </div>
        <span v-if="isOverdue" class="debt-sum-dot"></span>
      </div>
    </div>

    <!-- 有欠款但未配置的账户 -->
    <div v-if="unconfigured.length" class="unconfigured-banner">
      <div class="unconfigured-text">
        <AlertTriangle :size="16" />
        <span>{{ unconfigured.length }} 个负债账户有欠款但未配置账单周期</span>
      </div>
      <button
        v-for="item in unconfigured"
        :key="item.account"
        class="unconfigured-btn"
        :disabled="busy"
        @click="startNewRevolving(item.account)"
      >
        {{ item.name }} {{ formatMoney(item.balance) }} · 去配置
      </button>
    </div>

    <!-- 额度账单 -->
    <section class="debt-section">
      <div class="debt-section-head">
        <h3>
          <CreditCard :size="16" class="debt-section-ic" />额度账单
          <span class="debt-section-count tabular-nums">{{ revolvingList.length }}</span>
        </h3>
        <button class="btn btn-ghost debt-edit-btn" :disabled="busy" @click="startNewRevolving()">
          <Plus :size="14" />添加额度账单
        </button>
      </div>
      <div v-if="!revolvingList.length && !draftRevolving" class="empty-state empty-state-boxed">
        点击"添加额度账单"配置信用卡/白条等账单周期
      </div>
      <div v-else class="debt-grid">
        <RevolvingCard
          v-if="draftRevolving"
          :account="draftRevolving.account"
          :config="draftRevolving.config"
          :status="null"
          :long-term-accounts="config.longTermAccounts"
          :editing="true"
          :saving="saving"
          :edit-disabled="false"
          :is-new="true"
          :account-options="accountSuggestions"
          :taken-accounts="Object.keys(config.revolving)"
          @cancel="cancelEdit"
          @save="saveRevolving"
        />
        <RevolvingCard
          v-for="rv in revolvingList"
          :key="rv.account"
          :account="rv.account"
          :config="revolvingConfigOf(rv.account)"
          :status="rv"
          :long-term-accounts="config.longTermAccounts"
          :editing="editingKey === 'rev:' + rv.account"
          :saving="saving"
          :edit-disabled="busy && editingKey !== 'rev:' + rv.account"
          :is-new="false"
          :account-options="[]"
          :taken-accounts="[]"
          @edit="editingKey = 'rev:' + rv.account"
          @cancel="cancelEdit"
          @save="saveRevolving"
          @remove="removeRevolving(rv.account)"
        />
      </div>
    </section>

    <!-- 固定分期 -->
    <section class="debt-section">
      <div class="debt-section-head">
        <h3>
          <CalendarRange :size="16" class="debt-section-ic" />固定分期
          <span class="debt-section-count tabular-nums">{{ installmentList.length }}</span>
        </h3>
        <button class="btn btn-ghost debt-edit-btn" :disabled="busy" @click="startNewInstallment">
          <Plus :size="14" />添加固定分期
        </button>
      </div>
      <div v-if="!installmentList.length && !draftInstallment" class="empty-state empty-state-boxed">
        点击"添加固定分期"配置房贷/车贷等固定月供
      </div>
      <div v-else class="debt-grid">
        <InstallmentCard
          v-if="draftInstallment"
          :config="draftInstallment"
          :status="null"
          :long-term-accounts="config.longTermAccounts"
          :editing="true"
          :saving="saving"
          :edit-disabled="false"
          :is-new="true"
          :account-options="liabilityAccounts"
          @cancel="cancelEdit"
          @save="saveInstallment"
        />
        <InstallmentCard
          v-for="ins in installmentList"
          :key="ins.id"
          :config="installmentConfigOf(ins.id)"
          :status="ins"
          :long-term-accounts="config.longTermAccounts"
          :editing="editingKey === 'ins:' + ins.id"
          :saving="saving"
          :edit-disabled="busy && editingKey !== 'ins:' + ins.id"
          :is-new="false"
          :account-options="liabilityAccounts"
          @edit="editingKey = 'ins:' + ins.id"
          @cancel="cancelEdit"
          @save="saveInstallment"
          @remove="removeInstallment(ins.id)"
        />
      </div>
    </section>

    <!-- 未来每月要还多少;report 为 null 时后端算不出计划,整段不渲染 -->
    <PaymentSchedule v-if="report" :months="schedule" />

    <!-- 剩下那些没配账期的负债账户,长期负债标记在这勾 -->
    <LongTermOthers />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { CreditCard, AlertTriangle, Plus, CalendarRange } from '@lucide/vue'
import type { DebtsConfig, InstallmentConfig, RevolvingConfig } from '../../types/api'
import { formatMoney } from '../../composables/useFormatters'
import { useDebts } from '../../composables/useDebts'
import { useAnalytics } from '../../composables/useAnalytics'
import RevolvingCard from '../debts/RevolvingCard.vue'
import InstallmentCard from '../debts/InstallmentCard.vue'
import LongTermOthers from '../debts/LongTermOthers.vue'
import PaymentSchedule from '../debts/PaymentSchedule.vue'

const { config, report, loadDebts, saveDebts } = useDebts()
const { analytics } = useAnalytics()

// 同一时刻只允许一张卡进编辑态:每张卡保存时都要基于当前 config 合成全量配置再 POST,
// 并发编辑会让后保存的那张覆盖先保存的改动
const editingKey = ref<string | null>(null)
const saving = ref(false)
const draftRevolving = ref<{ account: string; config: RevolvingConfig } | null>(null)
const draftInstallment = ref<InstallmentConfig | null>(null)

const busy = computed(() => editingKey.value !== null)

onMounted(loadDebts)

const summary = computed(() => report.value?.summary)
const revolvingList = computed(() => report.value?.revolving ?? [])
const installmentList = computed(() => report.value?.installments ?? [])
const unconfigured = computed(() => report.value?.unconfigured ?? [])
const schedule = computed(() => report.value?.schedule ?? [])
const isOverdue = computed(() => (summary.value?.nextDueInDays ?? 0) < 0)

const countdownText = computed(() => {
  const days = summary.value?.nextDueInDays ?? 0
  if (days < 0) return `已逾期 ${-days} 天`
  if (days === 0) return '今天到期'
  return `${days} 天后`
})

// 新建额度账单的候选 = 账本里有欠款的负债账户,已配置的不再重复建议
const accountSuggestions = computed(() => {
  const used = new Set(Object.keys(config.value.revolving))
  return (analytics.value?.liabilityBreakdown ?? [])
    .map((l) => l.account)
    .filter((acc) => !used.has(acc))
})

const liabilityAccounts = computed(() =>
  (analytics.value?.accountBalances ?? [])
    .filter((acc) => acc.type === 'Liabilities')
    .map((acc) => acc.account),
)

function revolvingConfigOf(account: string): RevolvingConfig {
  return (
    config.value.revolving[account] ?? { name: '', billingDay: 1, dueDay: 10, installments: [] }
  )
}

function installmentConfigOf(id: string): InstallmentConfig {
  return (
    config.value.installments.find((ins) => ins.id === id) ?? {
      id,
      name: '',
      account: '',
      dueDay: 20,
      endMonth: '',
      schedule: [],
    }
  )
}

function startNewRevolving(account = '') {
  if (busy.value) return
  draftRevolving.value = {
    account,
    config: { name: '', billingDay: 1, dueDay: 10, installments: [] },
  }
  editingKey.value = 'new-rev'
}

function startNewInstallment() {
  if (busy.value) return
  draftInstallment.value = {
    id: `inst-${Date.now()}`,
    name: '',
    account: '',
    dueDay: 20,
    endMonth: '',
    schedule: [],
  }
  editingKey.value = 'new-ins'
}

function cancelEdit() {
  editingKey.value = null
  draftRevolving.value = null
  draftInstallment.value = null
}

function cloneConfig(): DebtsConfig {
  return JSON.parse(JSON.stringify(config.value))
}

// 长期负债标记只跟账户走:这里只增删当前编辑的那个账户,不去清理别的账户的标记。
// 账户即使不再有账期配置,只要还在账本里就会出现在「其他负债账户」组,标记仍有编辑入口。
function applyLongTerm(accounts: string[], account: string, on: boolean): string[] {
  return on ? [...new Set([...accounts, account])] : accounts.filter((a) => a !== account)
}

async function commit(next: DebtsConfig): Promise<void> {
  saving.value = true
  const ok = await saveDebts(next)
  saving.value = false
  if (ok) cancelEdit()
}

async function saveRevolving(payload: {
  account: string
  config: RevolvingConfig
  longTerm: boolean
}) {
  const next = cloneConfig()
  next.revolving[payload.account] = payload.config
  next.longTermAccounts = applyLongTerm(next.longTermAccounts, payload.account, payload.longTerm)
  await commit(next)
}

async function removeRevolving(account: string) {
  const next = cloneConfig()
  delete next.revolving[account]
  await commit(next)
}

async function saveInstallment(payload: { config: InstallmentConfig; longTerm: boolean }) {
  const next = cloneConfig()
  const idx = next.installments.findIndex((ins) => ins.id === payload.config.id)
  if (idx >= 0) next.installments[idx] = payload.config
  else next.installments.push(payload.config)
  next.longTermAccounts = applyLongTerm(
    next.longTermAccounts,
    payload.config.account,
    payload.longTerm,
  )
  await commit(next)
}

async function removeInstallment(id: string) {
  const next = cloneConfig()
  next.installments = next.installments.filter((ins) => ins.id !== id)
  await commit(next)
}
</script>

<style scoped>
.debts {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

/* ===== 看板 ===== */
.debts-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-4);
}

.debt-sum {
  padding: var(--space-5);
  position: relative;
}

.debt-sum-alert {
  border-color: var(--expense);
}

.debt-sum-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.debt-sum-value {
  margin-top: var(--space-2);
  font-size: var(--font-size-2xl);
  font-weight: 700;
}

.countdown-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-top: var(--space-1);
}

.debt-sum-dot {
  position: absolute;
  top: var(--space-4);
  right: var(--space-4);
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  background: var(--expense);
  animation: debtPulse 1.6s ease-in-out infinite;
}

@keyframes debtPulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.4;
  }
}

/* ===== 未配置提示 ===== */
.unconfigured-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  background: var(--warning-light);
  border: 1px solid var(--warning);
  border-radius: var(--radius-lg);
}

.unconfigured-text {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--warning);
  font-weight: 600;
}

.unconfigured-btn {
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-full);
  border: 1px solid var(--warning);
  background: transparent;
  color: var(--warning);
  font-size: var(--font-size-xs);
  font-weight: 600;
  cursor: pointer;
  transition:
    background-color var(--transition-base),
    color var(--transition-base);
}

.unconfigured-btn:hover:not(:disabled) {
  background: var(--warning);
  color: #fff;
}

.unconfigured-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ===== 分区头 ===== */
.debt-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}

.debt-section-head h3 {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: 620;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.debt-section-ic {
  color: var(--text-tertiary);
}

.debt-section-count {
  color: var(--text-tertiary);
  font-weight: 500;
}

.debt-edit-btn {
  gap: 6px;
  color: var(--text-secondary);
}

.debt-edit-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* ===== 卡片网格 ===== */
.debt-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-4);
}

@media (max-width: 900px) {
  .debts-summary {
    grid-template-columns: 1fr;
  }
  .debt-grid {
    grid-template-columns: 1fr;
  }
}
</style>
