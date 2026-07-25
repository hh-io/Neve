<template>
  <!-- 展示态 -->
  <div v-if="!editing && view" class="debt-card" :class="{ 'debt-card-overdue': view.overdue }">
    <span v-if="view.overdue" class="debt-card-bar"></span>
    <div class="debt-card-top">
      <div class="debt-card-icon"><CreditCard :size="20" /></div>
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
    <div v-if="view.inst" class="debt-inst">
      <button class="debt-inst-toggle" @click="expanded = !expanded">
        <span
          >含本期分期 <span class="tabular-nums">{{ view.inst.thisPeriod }}</span> · 未出账已扣
          <span class="tabular-nums">{{ view.inst.unbilled }}</span></span
        >
        <ChevronDown
          :size="14"
          class="debt-inst-chevron"
          :class="{ 'debt-inst-chevron-open': expanded }"
        />
      </button>
      <div v-if="expanded" class="debt-inst-details">
        <div
          v-for="item in view.inst.items"
          :key="item.name + item.firstBillMonth"
          class="debt-inst-item"
          :class="{ 'debt-inst-dimmed': item.dimmed }"
        >
          <div class="debt-inst-item-head">
            <span class="debt-inst-name">{{ item.name }}</span>
            <span v-if="item.dimmed" class="debt-badge badge-idle">已出账完毕</span>
            <span v-else-if="item.notStarted" class="debt-badge badge-idle">下期起</span>
            <span class="tabular-nums debt-inst-periods">{{ item.progressText }}</span>
          </div>
          <div class="progress-bar debt-inst-bar">
            <div class="progress-fill debt-inst-fill" :style="{ width: item.pct + '%' }"></div>
          </div>
          <div class="debt-inst-item-cap">
            <span
              >每期 <span class="tabular-nums">{{ item.monthlyText }}</span></span
            >
            <span
              >未出账 <span class="tabular-nums">{{ item.remainText }}</span></span
            >
          </div>
        </div>
        <div class="debt-inst-balance">
          当前欠款 <span class="tabular-nums">{{ view.inst.balance }}</span> · 其中未出账分期
          <span class="tabular-nums">{{ view.inst.balanceUnbilled }}</span>
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
        <div class="debt-edit-title">{{ isNew ? '新增额度账单' : local.name || account }}</div>
        <div v-if="!isNew" class="debt-edit-account">{{ account }}</div>
      </div>
    </div>

    <div class="form-row">
      <!-- 已有条目的账户是 config.revolving 的 key,改它等于删旧建新,故只读 -->
      <label v-if="isNew" class="field field-grow">
        <span>账户</span>
        <input
          v-model="localAccount"
          class="form-input"
          placeholder="Liabilities:CreditCard:CMB"
          :list="listId"
        />
        <datalist :id="listId">
          <option v-for="acc in accountOptions" :key="acc" :value="acc" />
        </datalist>
      </label>
      <label class="field field-grow">
        <span>显示名</span>
        <input v-model="local.name" class="form-input" placeholder="如 招行信用卡" />
      </label>
      <label class="field">
        <span>账单日</span>
        <input
          v-model.number="local.billingDay"
          type="number"
          min="1"
          max="31"
          class="form-input day-input"
        />
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
    </div>

    <label class="debt-longterm">
      <input v-model="localLongTerm" type="checkbox" />
      <span>计入长期负债</span>
    </label>
    <p class="debt-edit-hint">
      勾选后概览与账户页的净资产默认口径不再扣减该账户(房贷这类对应资产不在账本内的长期负债),完整口径降级为补充信息。
    </p>

    <!-- 内嵌分期:未出账部分从本期应还中扣减。报告只算当期,改删即时生效,无需 append-only -->
    <div class="debt-edit-block">
      <h4 class="debt-edit-block-title">内嵌分期(信用卡免息分期)</h4>
      <div v-for="(ri, ii) in local.installments" :key="ii" class="form-row inst-row">
        <label class="field field-grow">
          <span>分期名称</span>
          <input v-model="ri.name" class="form-input" placeholder="如 妙控键盘 24 期免息" />
        </label>
        <label class="field">
          <span>总金额(元)</span>
          <input
            v-model.number="ri.totalAmount"
            type="number"
            min="0.01"
            step="0.01"
            class="form-input amount-input"
          />
        </label>
        <label class="field">
          <span>期数</span>
          <input v-model.number="ri.months" type="number" min="1" class="form-input day-input" />
        </label>
        <label class="field">
          <span>每期(元)</span>
          <input
            v-model.number="ri.monthlyAmount"
            type="number"
            min="0.01"
            step="0.01"
            class="form-input amount-input"
          />
        </label>
        <label class="field">
          <span>首期账单月</span>
          <input v-model="ri.firstBillMonth" type="month" class="form-input" />
        </label>
        <button
          class="delete-btn delete-btn-sm"
          title="删除该分期"
          @click="local.installments.splice(ii, 1)"
        >
          ×
        </button>
      </div>
      <button class="btn btn-ghost inst-add-btn" @click="addInstallment">+ 添加分期</button>
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
import { ref, computed, useId, watch } from 'vue'
import { CreditCard, Pencil, CalendarClock, ChevronDown } from '@lucide/vue'
import type { RevolvingConfig, RevolvingStatus } from '../../types/api'
import { formatMoney } from '../../composables/useFormatters'
import { shortDate, countdownFor } from '../../composables/useDebtDisplay'
import { validateRevolving } from '../../composables/useDebtValidation'

const props = defineProps<{
  account: string
  config: RevolvingConfig
  /** 新建时为 null:后端还没算过这个账户 */
  status: RevolvingStatus | null
  /** 长期负债清单原样传入:开关跟着账户走,新建时账户是边填边变的 */
  longTermAccounts: string[]
  editing: boolean
  saving: boolean
  /** 别的卡正在编辑,本卡不能同时进编辑态(保存是合成全量 config,并发会互相覆盖) */
  editDisabled: boolean
  isNew: boolean
  accountOptions: string[]
  /** 已被占用的账户名,新建时查重 */
  takenAccounts: string[]
}>()

const emit = defineEmits<{
  edit: []
  cancel: []
  save: [payload: { account: string; config: RevolvingConfig; longTerm: boolean }]
  remove: []
}>()

const expanded = ref(false)
const confirmingDelete = ref(false)
const errors = ref<string[]>([])

const localAccount = ref('')
const localLongTerm = ref(false)
// 编辑副本:保存成功前不写回单例,取消无副作用
const local = ref<RevolvingConfig>(cloneConfig())

function cloneConfig(): RevolvingConfig {
  const rc: RevolvingConfig = JSON.parse(JSON.stringify(props.config))
  // 老配置(加内嵌分期前保存的)回显缺该字段
  rc.installments ??= []
  return rc
}

const longTerm = computed(() => props.longTermAccounts.includes(props.account))

// 每次进入编辑态都从 props 重建副本,上次取消的改动不会残留
watch(
  () => props.editing,
  (on) => {
    if (!on) return
    local.value = cloneConfig()
    localAccount.value = props.account
    localLongTerm.value = props.longTermAccounts.includes(props.account)
    confirmingDelete.value = false
    errors.value = []
  },
  { immediate: true },
)

// 开关描述的是"这个账户是不是长期负债",新建时账户边填边变,
// 必须跟着重读该账户的既有标记——否则给一个已标记的账户新建账单会把标记冲掉
watch(localAccount, (account) => {
  localLongTerm.value = props.longTermAccounts.includes(account)
})

// 改动一落键就撤掉上次的报错,避免用户已经改对了红字还挂着
watch(local, () => (errors.value = []), { deep: true })

const listId = useId()

const view = computed(() => {
  const rv = props.status
  if (!rv) return null
  const settled = rv.remaining <= 0 || rv.statementDue <= 0
  const overdue = rv.overdue && rv.remaining > 0
  return {
    name: rv.name,
    sub: `账单 ${shortDate(rv.statementDate)} → 还款 ${shortDate(rv.dueDate)}`,
    accountMissing: rv.accountMissing,
    overdue,
    dueAmount: formatMoney(rv.statementDue),
    remainAmount: formatMoney(Math.max(rv.remaining, 0)),
    remainColor:
      rv.remaining > 0 ? (overdue ? 'var(--expense)' : 'var(--text-primary)') : 'var(--income)',
    paidAmount: formatMoney(rv.paidSince),
    pct: rv.statementDue > 0 ? Math.min((rv.paidSince / rv.statementDue) * 100, 100) : 100,
    barColor: overdue ? 'var(--expense)' : settled ? 'var(--income)' : 'var(--accent)',
    dueDate: shortDate(rv.dueDate),
    status: badge(rv),
    countdown: countdownFor(overdue, settled, rv.daysUntilDue),
    inst: rv.installments.length
      ? {
          thisPeriod: formatMoney(rv.installmentThisPeriod),
          // 参与本期扣减的未出账(后端已剔除账单日后新购的分期)
          unbilled: formatMoney(rv.installmentUnbilled),
          balance: formatMoney(rv.currentBalance),
          // 今天口径的未出账合计:新购分期本金已计入当前欠款,这里要含它
          balanceUnbilled: formatMoney(
            rv.installments.reduce((sum, it) => sum + it.unbilledAmount, 0),
          ),
          items: rv.installments.map((it) => ({
            name: it.name,
            firstBillMonth: it.firstBillMonth,
            progressText: `${it.billedPeriods}/${it.months} 期`,
            pct: it.months > 0 ? Math.min((it.billedPeriods / it.months) * 100, 100) : 0,
            monthlyText: formatMoney(it.monthlyAmount),
            remainText: formatMoney(it.unbilledAmount),
            dimmed: it.finished && it.thisPeriodAmount <= 0,
            notStarted: it.billedPeriods === 0 && it.thisPeriodAmount <= 0 && !it.finished,
          })),
        }
      : null,
  }
})

function badge(rv: RevolvingStatus): { text: string; cls: string } {
  if (rv.statementDue <= 0) return { text: '本期无账单', cls: 'badge-idle' }
  if (rv.remaining <= 0) return { text: '已结清', cls: 'badge-paid' }
  if (rv.overdue) return { text: `逾期 ${-rv.daysUntilDue} 天`, cls: 'badge-overdue' }
  if (rv.daysUntilDue === 0) return { text: '今天到期', cls: 'badge-pending' }
  if (rv.daysUntilDue <= 5) return { text: '即将到期', cls: 'badge-pending' }
  return { text: '待还', cls: 'badge-idle' }
}

function addInstallment() {
  const now = new Date()
  const month = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  local.value.installments.push({
    name: '',
    totalAmount: 0,
    months: 12,
    monthlyAmount: 0,
    firstBillMonth: month,
  })
}

function onDelete() {
  if (!confirmingDelete.value) {
    confirmingDelete.value = true
    return
  }
  emit('remove')
}

function submit() {
  const account = localAccount.value.trim()
  errors.value = validateRevolving(account, local.value, props.isNew ? props.takenAccounts : [])
  if (errors.value.length) return
  emit('save', {
    account,
    config: JSON.parse(JSON.stringify(local.value)),
    longTerm: localLongTerm.value,
  })
}
</script>

<style scoped>
.align-right {
  text-align: right;
}

/* ===== 内嵌分期展示区 ===== */
.debt-inst {
  margin-top: var(--space-3);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-md);
  background: var(--surface-2);
  overflow: hidden;
}

.debt-inst-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  cursor: pointer;
}

.debt-inst-chevron {
  color: var(--text-tertiary);
  transition: transform var(--transition-fast);
}

.debt-inst-chevron-open {
  transform: rotate(180deg);
}

.debt-inst-details {
  padding: var(--space-2) var(--space-3) var(--space-3);
  border-top: 1px solid var(--hairline);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.debt-inst-item-head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-xs);
  margin-bottom: var(--space-1);
}

.debt-inst-name {
  flex: 1;
  min-width: 0;
  color: var(--text-primary);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.debt-inst-periods {
  color: var(--text-tertiary);
}

.debt-inst-bar {
  height: 4px;
}

.debt-inst-fill {
  background: var(--accent);
}

.debt-inst-dimmed {
  opacity: 0.55;
}

.debt-inst-item-cap {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.debt-inst-balance {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  padding-top: var(--space-2);
  border-top: 1px solid var(--hairline);
}

/* ===== 编辑态内嵌分期 ===== */
.inst-row {
  padding: var(--space-2) 0;
  border-top: 1px solid var(--hairline);
}

.inst-row:first-of-type {
  border-top: none;
}

.inst-add-btn {
  margin-top: var(--space-2);
  font-size: var(--font-size-xs);
}
</style>
