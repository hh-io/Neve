<template>
  <div class="card-static panel">
    <div class="panel-head">
      <div class="panel-head-left">
        <div class="panel-icon bg-brand-light">
          <Settings2 :size="20" color="var(--accent)" />
        </div>
        <span class="panel-title">待还配置</span>
      </div>
      <div class="editor-actions">
        <button class="btn btn-ghost" :disabled="saving" @click="$emit('cancel')">取消</button>
        <button class="btn btn-primary" :disabled="saving" @click="submit">
          {{ saving ? '保存中...' : '保存' }}
        </button>
      </div>
    </div>

    <!-- 长期负债:影响概览页净资产口径,不参与账期计算 -->
    <h4 class="editor-section-title">长期负债账户</h4>
    <p class="editor-section-hint">
      房贷这类长期负债对应的资产(如房产)通常不在账本内,单边扣减会让净资产变成几十年不变的巨额负数。
      勾选后,概览与账户页的净资产不再扣减这些账户,完整口径降级为补充信息。
    </p>
    <div v-if="liabilityAccounts.length" class="longterm-list">
      <label v-for="acc in liabilityAccounts" :key="acc.account" class="longterm-item">
        <input v-model="local.longTermAccounts" type="checkbox" :value="acc.account" class="longterm-check" />
        <span class="longterm-name">{{ acc.name }}</span>
        <span class="longterm-account">{{ acc.account }}</span>
      </label>
    </div>
    <p v-else class="editor-section-hint">账本里还没有负债账户。</p>

    <!-- 额度类 -->
    <h4 class="editor-section-title">额度账单(信用卡/白条等)</h4>
    <div v-for="(rc, account) in local.revolving" :key="account" class="editor-row">
      <div class="editor-row-account">{{ account }}</div>
      <div class="editor-row-fields">
        <label class="field">
          <span>显示名</span>
          <input v-model="rc.name" class="editor-input" placeholder="如 招行信用卡" />
        </label>
        <label class="field">
          <span>账单日</span>
          <input v-model.number="rc.billingDay" type="number" min="1" max="31" class="editor-input day-input" />
        </label>
        <label class="field">
          <span>还款日</span>
          <input v-model.number="rc.dueDay" type="number" min="1" max="31" class="editor-input day-input" />
        </label>
        <button class="delete-btn" title="删除" @click="removeRevolving(account)">×</button>
      </div>

      <!-- 内嵌分期:未出账部分从本期应还中扣减。报告只算当期,改删即时生效,无需 append-only -->
      <div v-if="rc.installments.length" class="inst-list">
        <div v-for="(ri, ii) in rc.installments" :key="ii" class="editor-row-fields">
          <label class="field field-grow">
            <span>分期名称</span>
            <input v-model="ri.name" class="editor-input" placeholder="如 妙控键盘 24 期免息" />
          </label>
          <label class="field">
            <span>总金额(元)</span>
            <input v-model.number="ri.totalAmount" type="number" min="0.01" step="0.01" class="editor-input amount-input" />
          </label>
          <label class="field">
            <span>期数</span>
            <input v-model.number="ri.months" type="number" min="1" class="editor-input day-input" />
          </label>
          <label class="field">
            <span>每期(元)</span>
            <input v-model.number="ri.monthlyAmount" type="number" min="0.01" step="0.01" class="editor-input amount-input" />
          </label>
          <label class="field">
            <span>首期账单月</span>
            <input v-model="ri.firstBillMonth" type="month" class="editor-input" />
          </label>
          <button class="delete-btn delete-btn-sm" title="删除该分期" @click="rc.installments.splice(ii, 1)">×</button>
        </div>
      </div>
      <button class="btn btn-ghost inst-add-btn" @click="addRevInstallment(rc)">+ 添加分期</button>
    </div>
    <div class="editor-add-row">
      <input
        v-model="newAccount"
        class="editor-input"
        placeholder="账户名,如 Liabilities:CreditCard:CMB"
        list="debt-account-suggestions"
      />
      <datalist id="debt-account-suggestions">
        <option v-for="acc in accountSuggestions" :key="acc" :value="acc" />
      </datalist>
      <button class="btn btn-secondary" :disabled="!newAccount" @click="addRevolving">添加账户</button>
    </div>

    <!-- 分期类 -->
    <h4 class="editor-section-title">固定分期(房贷/车贷等)</h4>
    <div v-for="(ins, idx) in local.installments" :key="ins.id" class="editor-installment">
      <div class="editor-row-fields">
        <label class="field">
          <span>名称</span>
          <input v-model="ins.name" class="editor-input" placeholder="如 房贷" />
        </label>
        <label class="field field-grow">
          <span>关联账户</span>
          <input
            v-model="ins.account"
            class="editor-input"
            placeholder="Liabilities:Loan:Mortgage"
            list="debt-account-suggestions"
          />
        </label>
        <label class="field">
          <span>还款日</span>
          <input v-model.number="ins.dueDay" type="number" min="1" max="31" class="editor-input day-input" />
        </label>
        <button class="delete-btn" title="删除该分期" @click="removeInstallment(idx)">×</button>
      </div>

      <!-- 月供调整只追加不改历史:改历史会静默改写过去账期的口径 -->
      <div class="schedule-list">
        <div v-for="(ph, pi) in ins.schedule" :key="ph.effectiveFrom + pi" class="schedule-item">
          <span class="schedule-date">{{ ph.effectiveFrom }} 起</span>
          <span class="schedule-amount">{{ formatMoney(ph.amount) }}/月</span>
          <button
            v-if="pi === ins.schedule.length - 1"
            class="delete-btn delete-btn-sm"
            title="撤销最新一条"
            @click="ins.schedule.pop()"
          >×</button>
        </div>
        <div class="schedule-append">
          <input v-model="phaseDrafts[ins.id].from" type="date" class="editor-input" />
          <input
            v-model.number="phaseDrafts[ins.id].amount"
            type="number"
            min="0.01"
            step="0.01"
            class="editor-input amount-input"
            placeholder="月供(元)"
          />
          <button class="btn btn-ghost" :disabled="!canAppend(ins.id)" @click="appendPhase(ins)">追加金额</button>
        </div>
      </div>
    </div>
    <div class="editor-add-row">
      <button class="btn btn-secondary" @click="addInstallment">添加分期</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue';
import { Settings2 } from '@lucide/vue';
import type { DebtsConfig, InstallmentConfig, RevolvingConfig } from '../types/api';
import { formatMoney } from '../composables/useFormatters';
import { useAnalytics } from '../composables/useAnalytics';

const props = defineProps<{
  config: DebtsConfig;
  prefill?: string;
  saving?: boolean;
}>();

const emit = defineEmits<{ save: [next: DebtsConfig]; cancel: [] }>();

const { analytics } = useAnalytics();

// 编辑副本:保存成功前不写回单例,取消无副作用
const local = ref<DebtsConfig>(JSON.parse(JSON.stringify(props.config)));
const newAccount = ref('');
// 每个分期一份"追加月供"草稿(生效日期 + 金额)
const phaseDrafts = reactive<Record<string, { from: string; amount: number | null }>>({});

// 必须在 setup 同步阶段补齐:模板直接读 phaseDrafts[ins.id].from,
// 放进 onMounted 会晚于首次渲染,有固定分期时整个编辑器会渲染报错
local.value.installments.forEach(ensureDraft);
// 老配置(加各功能前保存的)回显可能没有这些字段
Object.values(local.value.revolving).forEach(rc => { rc.installments ??= []; });
local.value.longTermAccounts ??= [];

onMounted(() => {
  if (props.prefill && !local.value.revolving[props.prefill]) {
    newAccount.value = props.prefill;
  }
});

// 长期负债候选取 accountBalances 而非 liabilityBreakdown:后者只含有欠款的账户,
// 已还清的房贷账户仍应能保留勾选,否则重新保存配置会把标记丢掉
const liabilityAccounts = computed(() =>
  (analytics.value?.accountBalances ?? [])
    .filter(acc => acc.type === 'Liabilities')
    .map(acc => ({ account: acc.account, name: acc.account.split(':').pop() ?? acc.account }))
);

// 候选账户 = 账本里有欠款的负债账户,已配置的不再重复建议
const accountSuggestions = computed(() => {
  const used = new Set(Object.keys(local.value.revolving));
  return (analytics.value?.liabilityBreakdown ?? [])
    .map(l => l.account)
    .filter(acc => !used.has(acc));
});

function ensureDraft(ins: InstallmentConfig) {
  if (!phaseDrafts[ins.id]) {
    phaseDrafts[ins.id] = { from: '', amount: null };
  }
}

function addRevolving() {
  const account = newAccount.value.trim();
  if (!account || local.value.revolving[account]) return;
  local.value.revolving[account] = { name: '', billingDay: 1, dueDay: 10, installments: [] };
  newAccount.value = '';
}

function addRevInstallment(rc: RevolvingConfig) {
  const now = new Date();
  const month = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  rc.installments.push({ name: '', totalAmount: 0, months: 12, monthlyAmount: 0, firstBillMonth: month });
}

function removeRevolving(account: string) {
  delete local.value.revolving[account];
}

function addInstallment() {
  const ins: InstallmentConfig = {
    id: `inst-${Date.now()}`,
    name: '',
    account: '',
    dueDay: 20,
    schedule: []
  };
  local.value.installments.push(ins);
  ensureDraft(ins);
}

function removeInstallment(idx: number) {
  local.value.installments.splice(idx, 1);
}

function canAppend(id: string): boolean {
  const draft = phaseDrafts[id];
  return !!draft?.from && (draft?.amount ?? 0) > 0;
}

function appendPhase(ins: InstallmentConfig) {
  const draft = phaseDrafts[ins.id];
  if (!draft || !canAppend(ins.id)) return;
  ins.schedule.push({ effectiveFrom: draft.from, amount: draft.amount! });
  phaseDrafts[ins.id] = { from: '', amount: null };
}

function submit() {
  emit('save', JSON.parse(JSON.stringify(local.value)));
}
</script>

<style scoped>
.editor-actions {
  display: flex;
  gap: var(--space-2);
}

.editor-section-title {
  margin: var(--space-4) 0 var(--space-2);
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-secondary);
}

.editor-section-hint {
  margin: 0 0 var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  line-height: 1.6;
}

.longterm-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.longterm-item {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-sm);
  background: var(--surface-1);
  cursor: pointer;
  transition: border-color var(--transition-fast);
}

.longterm-item:hover {
  border-color: var(--hairline-strong);
}

.longterm-check {
  accent-color: var(--accent);
  cursor: pointer;
}

.longterm-name {
  font-size: var(--font-size-sm);
  color: var(--text-primary);
}

.longterm-account {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  font-family: var(--font-numeric);
}

.editor-row,
.editor-installment {
  padding: var(--space-3) 0;
  border-top: 1px solid var(--hairline);
}

.editor-row-account {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-bottom: var(--space-2);
  font-family: var(--font-numeric);
}

.editor-row-fields {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.field-grow {
  flex: 1;
  min-width: 200px;
}

.editor-input {
  padding: var(--space-2);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  background: var(--surface-1);
  color: var(--text-primary);
}

.field-grow .editor-input {
  width: 100%;
}

.day-input {
  width: 64px;
  text-align: right;
}

.amount-input {
  width: 120px;
  text-align: right;
}

.editor-add-row {
  display: flex;
  gap: var(--space-3);
  padding: var(--space-3) 0;
}

.editor-add-row .editor-input {
  flex: 1;
}

.delete-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: var(--expense-light);
  color: var(--expense);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 16px;
  font-weight: 600;
  transition: all var(--transition-fast);
}

.delete-btn:hover {
  background: var(--expense);
  color: white;
}

.delete-btn-sm {
  width: 22px;
  height: 22px;
  font-size: 13px;
}

.schedule-list {
  margin-top: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.schedule-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  font-variant-numeric: tabular-nums;
}

.schedule-amount {
  font-weight: 500;
  color: var(--text-primary);
}

.schedule-append {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.inst-list {
  margin-top: var(--space-3);
  padding-left: var(--space-4);
  border-left: 2px solid var(--hairline);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.inst-add-btn {
  margin-top: var(--space-2);
  font-size: var(--font-size-xs);
}
</style>
