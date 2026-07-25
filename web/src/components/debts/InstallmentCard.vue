<template>
  <!-- 展示态 -->
  <div v-if="!editing && view" class="debt-card" :class="{ 'debt-card-overdue': view.overdue }">
    <span v-if="view.overdue" class="debt-card-bar"></span>
    <div class="debt-card-top">
      <div class="debt-card-icon"><Landmark :size="20" /></div>
      <div class="debt-card-id">
        <div class="debt-card-name">{{ view.name }}</div>
        <div class="debt-card-sub">{{ view.sub }}</div>
      </div>
      <span v-if="longTerm" class="debt-badge badge-idle">长期</span>
      <span class="debt-badge" :class="view.status.cls">{{ view.status.text }}</span>
      <button
        class="btn btn-ghost debt-card-edit-btn"
        :disabled="editDisabled"
        :title="editDisabled ? '请先完成当前编辑' : '编辑'"
        @click="$emit('edit')"
      >
        <Pencil :size="14" />
      </button>
    </div>
    <div class="debt-card-figures">
      <div>
        <div class="debt-fig-label">本期应还</div>
        <div class="debt-fig-due tabular-nums">{{ view.dueAmount }}</div>
      </div>
      <div class="align-right">
        <div class="debt-fig-label">剩余待还</div>
        <div class="debt-fig-remain tabular-nums" :style="{ color: view.remainColor }">
          {{ view.remainAmount }}
        </div>
      </div>
    </div>
    <div class="debt-card-progress">
      <div class="progress-bar debt-bar">
        <div
          class="progress-fill"
          :style="{ width: view.pct + '%', background: view.barColor }"
        ></div>
      </div>
      <div class="debt-card-progress-cap">
        <span
          >已还 <span class="tabular-nums">{{ view.paidAmount }}</span></span
        >
        <span class="tabular-nums">{{ Math.round(view.pct) }}%</span>
      </div>
    </div>
    <div class="debt-card-foot">
      <span class="debt-foot-date">
        <CalendarClock :size="16" class="debt-foot-ic" />最后还款日
        <span class="tabular-nums debt-foot-strong">{{ view.dueDate }}</span>
      </span>
      <span class="tabular-nums debt-countdown" :style="{ color: view.countdown.color }">{{
        view.countdown.text
      }}</span>
    </div>
    <div v-if="view.accountMissing" class="debt-missing">账本无此账户</div>
  </div>

  <!-- 编辑态 -->
  <div v-else class="debt-card debt-card-editing">
    <div class="debt-edit-head">
      <div>
        <div class="debt-edit-title">{{ isNew ? '新增固定分期' : local.name || '固定分期' }}</div>
        <div v-if="!isNew" class="debt-edit-account">{{ local.account }}</div>
      </div>
    </div>

    <div class="form-row">
      <label class="field">
        <span>名称</span>
        <input v-model="local.name" class="form-input" placeholder="如 房贷" />
      </label>
      <label class="field field-grow">
        <span>关联账户</span>
        <input
          v-model="local.account"
          class="form-input"
          placeholder="Liabilities:Loan:Mortgage"
          :list="listId"
        />
        <datalist :id="listId">
          <option v-for="acc in accountOptions" :key="acc" :value="acc" />
        </datalist>
      </label>
      <label class="field">
        <span>还款日</span>
        <input
          v-model.number="local.dueDay"
          type="number"
          min="1"
          max="31"
          class="form-input day-input"
        />
      </label>
      <label class="field">
        <span>末期月</span>
        <input v-model="local.endMonth" type="month" class="form-input month-input" />
      </label>
    </div>
    <p class="debt-edit-hint">
      末期月留空表示没有终止期(房贷)。车贷/消费贷这类填上末期还款月,计划表才会在还完那月停下,
      卡片也能显示还剩几期——总期数没法从账本反推(负债余额是剩余本金,不含未来利息)。
    </p>

    <label class="debt-longterm">
      <input v-model="localLongTerm" type="checkbox" />
      <span>计入长期负债</span>
    </label>
    <p class="debt-edit-hint">
      勾选后概览与账户页的净资产默认口径不再扣减该账户(房贷这类对应资产不在账本内的长期负债),完整口径降级为补充信息。
    </p>

    <!-- 月供调整只追加不改历史:改历史会静默改写过去账期的口径 -->
    <div class="debt-edit-block">
      <h4 class="debt-edit-block-title">月供</h4>
      <div v-if="!local.schedule.length" class="schedule-empty">
        还没有月供记录,先在下方追加一条才能保存。
      </div>
      <div v-for="(ph, pi) in local.schedule" :key="ph.effectiveFrom + pi" class="schedule-item">
        <span class="schedule-date tabular-nums">{{ ph.effectiveFrom }} 起</span>
        <span class="schedule-amount tabular-nums">{{ formatMoney(ph.amount) }}/月</span>
        <button
          v-if="pi === local.schedule.length - 1"
          class="delete-btn delete-btn-sm"
          title="撤销最新一条"
          @click="local.schedule.pop()"
        >
          ×
        </button>
      </div>
      <div class="schedule-append">
        <input v-model="draft.from" type="date" class="form-input" />
        <input
          v-model.number="draft.amount"
          type="number"
          min="0.01"
          step="0.01"
          class="form-input amount-input"
          placeholder="月供(元)"
        />
        <button class="btn btn-ghost" :disabled="!canAppend" @click="appendPhase">追加金额</button>
      </div>
      <p class="debt-edit-hint">历史月供只可追加,不可改写——改历史会静默改写过去账期的口径。</p>
    </div>

    <ul v-if="errors.length" class="debt-edit-errors">
      <li v-for="err in errors" :key="err">{{ err }}</li>
    </ul>

    <div class="debt-edit-foot">
      <!-- 单卡保存即时落盘,没有整表单的取消兜底,故删除走两段式确认 -->
      <button v-if="!isNew" class="btn btn-danger" :disabled="saving" @click="onDelete">
        {{ confirmingDelete ? '确认删除?' : '删除' }}
      </button>
      <span v-else></span>
      <div class="debt-edit-foot-right">
        <button class="btn btn-ghost" :disabled="saving" @click="$emit('cancel')">取消</button>
        <button class="btn btn-primary" :disabled="saving" @click="submit">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, useId, watch } from 'vue'
import { Landmark, Pencil, CalendarClock } from '@lucide/vue'
import type { InstallmentConfig, InstallmentStatus } from '../../types/api'
import { formatMoney } from '../../composables/useFormatters'
import { shortDate, countdownFor } from '../../composables/useDebtDisplay'
import { validateInstallment } from '../../composables/useDebtValidation'

const props = defineProps<{
  config: InstallmentConfig
  /** 新建时为 null:后端还没算过这条分期 */
  status: InstallmentStatus | null
  /** 长期负债清单原样传入:开关跟着关联账户走,而账户在编辑态是可改的 */
  longTermAccounts: string[]
  editing: boolean
  saving: boolean
  /** 别的卡正在编辑,本卡不能同时进编辑态(保存是合成全量 config,并发会互相覆盖) */
  editDisabled: boolean
  isNew: boolean
  accountOptions: string[]
}>()

const emit = defineEmits<{
  edit: []
  cancel: []
  save: [payload: { config: InstallmentConfig; longTerm: boolean }]
  remove: []
}>()

const confirmingDelete = ref(false)
const errors = ref<string[]>([])
const localLongTerm = ref(false)
// 编辑副本:保存成功前不写回单例,取消无副作用
const local = ref<InstallmentConfig>(JSON.parse(JSON.stringify(props.config)))
// 追加月供的草稿(生效日期 + 金额)
const draft = reactive<{ from: string; amount: number | null }>({ from: '', amount: null })

const longTerm = computed(() => props.longTermAccounts.includes(props.config.account))

// 每次进入编辑态都从 props 重建副本,上次取消的改动不会残留
watch(
  () => props.editing,
  (on) => {
    if (!on) return
    local.value = JSON.parse(JSON.stringify(props.config))
    localLongTerm.value = props.longTermAccounts.includes(props.config.account)
    confirmingDelete.value = false
    errors.value = []
    draft.from = ''
    draft.amount = null
  },
  { immediate: true },
)

// 开关描述的是"这个账户是不是长期负债",关联账户改了就得跟着重读该账户的既有标记——
// 否则给一个已标记的账户新建分期会把标记冲掉
watch(
  () => local.value.account,
  (account) => {
    localLongTerm.value = props.longTermAccounts.includes(account)
  },
)

// 改动一落键就撤掉上次的报错,避免用户已经改对了红字还挂着
watch(local, () => (errors.value = []), { deep: true })

const listId = useId()

const view = computed(() => {
  const ins = props.status
  if (!ins) return null
  const due = ins.monthlyAmount
  const remain = ins.paid || !due ? 0 : Math.max(due - ins.paidAmount, 0)
  const overdue = ins.overdue && !ins.paid && !!due
  // remainingPeriods 为 -1 表示无终止期(房贷),不展示期数
  const periods = ins.settled
    ? `已于 ${ins.endMonth} 还清`
    : ins.remainingPeriods > 0
      ? `剩 ${ins.remainingPeriods} 期(至 ${ins.endMonth})`
      : ''
  return {
    name: ins.name,
    sub: [
      `还款日 ${shortDate(ins.dueDate)}`,
      `剩余本金 ${formatMoney(ins.currentBalance)}`,
      periods,
    ]
      .filter(Boolean)
      .join(' · '),
    accountMissing: ins.accountMissing,
    overdue,
    // 结清与未生效都是月供 0,文案不能混
    dueAmount: due ? formatMoney(due) : ins.settled ? '—' : '未生效',
    remainAmount: due ? formatMoney(remain) : '—',
    remainColor: !due
      ? 'var(--text-tertiary)'
      : remain > 0
        ? overdue
          ? 'var(--expense)'
          : 'var(--text-primary)'
        : 'var(--income)',
    paidAmount: formatMoney(ins.paidAmount),
    pct: due ? Math.min((ins.paidAmount / due) * 100, 100) : 0,
    barColor: overdue ? 'var(--expense)' : ins.paid || !due ? 'var(--income)' : 'var(--accent)',
    dueDate: shortDate(ins.dueDate),
    status: badge(ins),
    countdown: countdownFor(overdue, ins.paid || !due, ins.daysUntilDue),
  }
})

function badge(ins: InstallmentStatus): { text: string; cls: string } {
  // 结清与"尚未生效"同为月供 0,但含义相反,得先分流
  if (ins.settled) return { text: '已结清', cls: 'badge-paid' }
  if (!ins.monthlyAmount) return { text: '尚未生效', cls: 'badge-idle' }
  if (ins.paid) return { text: '本月已还', cls: 'badge-paid' }
  if (ins.overdue) return { text: `逾期 ${-ins.daysUntilDue} 天`, cls: 'badge-overdue' }
  if (ins.daysUntilDue <= 5) return { text: '即将到期', cls: 'badge-pending' }
  return { text: '待还', cls: 'badge-idle' }
}

const canAppend = computed(() => !!draft.from && (draft.amount ?? 0) > 0)

function appendPhase() {
  if (!canAppend.value) return
  local.value.schedule.push({ effectiveFrom: draft.from, amount: draft.amount! })
  draft.from = ''
  draft.amount = null
}

function onDelete() {
  if (!confirmingDelete.value) {
    confirmingDelete.value = true
    return
  }
  emit('remove')
}

function submit() {
  // 不就地改 local:深监听会在下一 tick 把刚设好的 errors 清掉
  const next: InstallmentConfig = JSON.parse(JSON.stringify(local.value))
  next.name = next.name.trim()
  next.account = next.account.trim()
  errors.value = validateInstallment(next)
  if (errors.value.length) return
  emit('save', { config: next, longTerm: localLongTerm.value })
}
</script>

<style scoped>
.align-right {
  text-align: right;
}

.schedule-empty {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-bottom: var(--space-2);
}

.schedule-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.schedule-amount {
  font-weight: 500;
  color: var(--text-primary);
}

.schedule-append {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
</style>
