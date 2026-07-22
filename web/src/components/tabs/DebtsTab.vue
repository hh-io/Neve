<template>
  <div class="animate-fade-in-up debts">
    <!-- 全局看板 -->
    <div class="debts-summary">
      <div class="card debt-sum">
        <div class="debt-sum-label">本期总应还</div>
        <div class="debt-sum-value tabular-nums">{{ formatMoney(summary?.monthDue ?? 0) }}</div>
      </div>
      <div class="card debt-sum">
        <div class="debt-sum-label">剩余待还</div>
        <div class="debt-sum-value tabular-nums" :class="(summary?.monthRemaining ?? 0) > 0 ? 'text-expense' : 'text-income'">
          {{ formatMoney(summary?.monthRemaining ?? 0) }}
        </div>
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
        <span v-if="isOverdue" class="debt-sum-dot"></span>
      </div>
    </div>

    <!-- 有欠款但未配置的账户 -->
    <div v-if="!editing && unconfigured.length" class="unconfigured-banner">
      <div class="unconfigured-text">
        <AlertTriangle :size="16" />
        <span>{{ unconfigured.length }} 个负债账户有欠款但未配置账单周期</span>
      </div>
      <button
        v-for="item in unconfigured"
        :key="item.account"
        class="unconfigured-btn"
        @click="startEdit(item.account)"
      >
        {{ item.name }} {{ formatMoney(item.balance) }} · 去配置
      </button>
    </div>

    <!-- 编辑态 -->
    <DebtConfigEditor
      v-if="editing"
      :config="config"
      :prefill="prefillAccount"
      :saving="saving"
      @save="onSave"
      @cancel="editing = false"
    />

    <!-- 展示态 -->
    <template v-else>
      <!-- 额度账单 -->
      <section class="debt-section">
        <div class="debt-section-head">
          <h3><CreditCard :size="16" class="debt-section-ic" />额度账单 <span class="debt-section-count tabular-nums">{{ revolvingCards.length }}</span></h3>
          <button class="btn btn-ghost debt-edit-btn" @click="startEdit()"><Pencil :size="14" />编辑配置</button>
        </div>
        <div v-if="!revolvingCards.length" class="empty-state">点击"编辑配置"添加信用卡/白条等账单周期</div>
        <div v-else class="debt-grid">
          <div v-for="d in revolvingCards" :key="d.key" class="debt-card" :class="{ 'debt-card-overdue': d.overdue }">
            <span v-if="d.overdue" class="debt-card-bar"></span>
            <div class="debt-card-top">
              <div class="debt-card-icon"><component :is="d.icon" :size="20" /></div>
              <div class="debt-card-id">
                <div class="debt-card-name">{{ d.name }}</div>
                <div class="debt-card-sub">{{ d.sub }}</div>
              </div>
              <span class="debt-badge" :class="d.status.cls">{{ d.status.text }}</span>
            </div>
            <div class="debt-card-figures">
              <div>
                <div class="debt-fig-label">本期应还</div>
                <div class="debt-fig-due tabular-nums">{{ d.dueAmount }}</div>
              </div>
              <div class="align-right">
                <div class="debt-fig-label">剩余待还</div>
                <div class="debt-fig-remain tabular-nums" :style="{ color: d.remainColor }">{{ d.remainAmount }}</div>
              </div>
            </div>
            <div class="debt-card-progress">
              <div class="progress-bar debt-bar">
                <div class="progress-fill" :style="{ width: d.pct + '%', background: d.barColor }"></div>
              </div>
              <div class="debt-card-progress-cap">
                <span>已还 <span class="tabular-nums">{{ d.paidAmount }}</span></span>
                <span class="tabular-nums">{{ Math.round(d.pct) }}%</span>
              </div>
            </div>
            <div class="debt-card-foot">
              <span class="debt-foot-date"><CalendarClock :size="16" class="debt-foot-ic" />最后还款日 <span class="tabular-nums debt-foot-strong">{{ d.dueDate }}</span></span>
              <span class="tabular-nums debt-countdown" :style="{ color: d.countdown.color }">{{ d.countdown.text }}</span>
            </div>
            <div v-if="d.accountMissing" class="debt-missing">账本无此账户</div>
          </div>
        </div>
      </section>

      <!-- 固定分期 -->
      <section class="debt-section">
        <div class="debt-section-head">
          <h3><CalendarRange :size="16" class="debt-section-ic" />固定分期 <span class="debt-section-count tabular-nums">{{ installmentCards.length }}</span></h3>
          <button class="btn btn-ghost debt-edit-btn" @click="startEdit()"><Pencil :size="14" />编辑配置</button>
        </div>
        <div v-if="!installmentCards.length" class="empty-state">点击"编辑配置"添加房贷/车贷等固定月供</div>
        <div v-else class="debt-grid">
          <div v-for="d in installmentCards" :key="d.key" class="debt-card" :class="{ 'debt-card-overdue': d.overdue }">
            <span v-if="d.overdue" class="debt-card-bar"></span>
            <div class="debt-card-top">
              <div class="debt-card-icon"><component :is="d.icon" :size="20" /></div>
              <div class="debt-card-id">
                <div class="debt-card-name">{{ d.name }}</div>
                <div class="debt-card-sub">{{ d.sub }}</div>
              </div>
              <span class="debt-badge" :class="d.status.cls">{{ d.status.text }}</span>
            </div>
            <div class="debt-card-figures">
              <div>
                <div class="debt-fig-label">本期应还</div>
                <div class="debt-fig-due tabular-nums">{{ d.dueAmount }}</div>
              </div>
              <div class="align-right">
                <div class="debt-fig-label">剩余待还</div>
                <div class="debt-fig-remain tabular-nums" :style="{ color: d.remainColor }">{{ d.remainAmount }}</div>
              </div>
            </div>
            <div class="debt-card-progress">
              <div class="progress-bar debt-bar">
                <div class="progress-fill" :style="{ width: d.pct + '%', background: d.barColor }"></div>
              </div>
              <div class="debt-card-progress-cap">
                <span>已还 <span class="tabular-nums">{{ d.paidAmount }}</span></span>
                <span class="tabular-nums">{{ Math.round(d.pct) }}%</span>
              </div>
            </div>
            <div class="debt-card-foot">
              <span class="debt-foot-date"><CalendarClock :size="16" class="debt-foot-ic" />最后还款日 <span class="tabular-nums debt-foot-strong">{{ d.dueDate }}</span></span>
              <span class="tabular-nums debt-countdown" :style="{ color: d.countdown.color }">{{ d.countdown.text }}</span>
            </div>
            <div v-if="d.accountMissing" class="debt-missing">账本无此账户</div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { CreditCard, Landmark, AlertTriangle, Pencil, CalendarClock, CalendarRange } from '@lucide/vue';
import type { DebtsConfig, RevolvingStatus, InstallmentStatus } from '../../types/api';
import { formatMoney } from '../../composables/useFormatters';
import { useDebts } from '../../composables/useDebts';
import DebtConfigEditor from '../DebtConfigEditor.vue';

const { config, report, loadDebts, saveDebts } = useDebts();

const editing = ref(false);
const saving = ref(false);
const prefillAccount = ref('');

onMounted(loadDebts);

const summary = computed(() => report.value?.summary);
const revolving = computed(() => report.value?.revolving ?? []);
const installments = computed(() => report.value?.installments ?? []);
const unconfigured = computed(() => report.value?.unconfigured ?? []);
const isOverdue = computed(() => (summary.value?.nextDueInDays ?? 0) < 0);

const countdownText = computed(() => {
  const days = summary.value?.nextDueInDays ?? 0;
  if (days < 0) return `已逾期 ${-days} 天`;
  if (days === 0) return '今天到期';
  return `${days} 天后`;
});

function shortDate(date: string): string {
  return date.slice(5).replace('-', '/');
}

// 倒计时文案 + 颜色
function countdownFor(overdue: boolean, settled: boolean, days: number): { text: string; color: string } {
  if (settled) return { text: '已还清', color: 'var(--income)' };
  if (overdue) return { text: `逾期 ${-days} 天`, color: 'var(--expense)' };
  if (days === 0) return { text: '今天到期', color: 'var(--warning)' };
  return { text: `还剩 ${days} 天`, color: days <= 3 ? 'var(--warning)' : 'var(--text-secondary)' };
}

function revolvingBadge(rv: RevolvingStatus): { text: string; cls: string } {
  if (rv.statementDue <= 0) return { text: '本期无账单', cls: 'badge-idle' };
  if (rv.remaining <= 0) return { text: '已结清', cls: 'badge-paid' };
  if (rv.overdue) return { text: `逾期 ${-rv.daysUntilDue} 天`, cls: 'badge-overdue' };
  if (rv.daysUntilDue === 0) return { text: '今天到期', cls: 'badge-pending' };
  if (rv.daysUntilDue <= 5) return { text: '即将到期', cls: 'badge-pending' };
  return { text: '待还', cls: 'badge-idle' };
}

function installmentBadge(ins: InstallmentStatus): { text: string; cls: string } {
  if (!ins.monthlyAmount) return { text: '尚未生效', cls: 'badge-idle' };
  if (ins.paid) return { text: '本月已还', cls: 'badge-paid' };
  if (ins.overdue) return { text: `逾期 ${-ins.daysUntilDue} 天`, cls: 'badge-overdue' };
  if (ins.daysUntilDue <= 5) return { text: '即将到期', cls: 'badge-pending' };
  return { text: '待还', cls: 'badge-idle' };
}

const revolvingCards = computed(() => revolving.value.map((rv) => {
  const settled = rv.remaining <= 0 || rv.statementDue <= 0;
  const overdue = rv.overdue && rv.remaining > 0;
  return {
    key: rv.account,
    name: rv.name,
    icon: CreditCard,
    sub: `账单 ${shortDate(rv.statementDate)} → 还款 ${shortDate(rv.dueDate)}`,
    accountMissing: rv.accountMissing,
    overdue,
    dueAmount: formatMoney(rv.statementDue),
    remainAmount: formatMoney(Math.max(rv.remaining, 0)),
    remainColor: rv.remaining > 0 ? (overdue ? 'var(--expense)' : 'var(--text-primary)') : 'var(--income)',
    paidAmount: formatMoney(rv.paidSince),
    pct: rv.statementDue > 0 ? Math.min((rv.paidSince / rv.statementDue) * 100, 100) : 100,
    barColor: overdue ? 'var(--expense)' : settled ? 'var(--income)' : 'var(--accent)',
    dueDate: shortDate(rv.dueDate),
    status: revolvingBadge(rv),
    countdown: countdownFor(overdue, settled, rv.daysUntilDue),
  };
}));

const installmentCards = computed(() => installments.value.map((ins) => {
  const due = ins.monthlyAmount;
  const remain = ins.paid || !due ? 0 : Math.max(due - ins.paidAmount, 0);
  const overdue = ins.overdue && !ins.paid && !!due;
  return {
    key: ins.id,
    name: ins.name,
    icon: Landmark,
    sub: `还款日 ${shortDate(ins.dueDate)} · 剩余本金 ${formatMoney(ins.currentBalance)}`,
    accountMissing: ins.accountMissing,
    overdue,
    dueAmount: due ? formatMoney(due) : '未生效',
    remainAmount: due ? formatMoney(remain) : '—',
    remainColor: !due ? 'var(--text-tertiary)' : remain > 0 ? (overdue ? 'var(--expense)' : 'var(--text-primary)') : 'var(--income)',
    paidAmount: formatMoney(ins.paidAmount),
    pct: due ? Math.min((ins.paidAmount / due) * 100, 100) : 0,
    barColor: overdue ? 'var(--expense)' : ins.paid || !due ? 'var(--income)' : 'var(--accent)',
    dueDate: shortDate(ins.dueDate),
    status: installmentBadge(ins),
    countdown: countdownFor(overdue, ins.paid || !due, ins.daysUntilDue),
  };
}));

function startEdit(account = '') {
  prefillAccount.value = account;
  editing.value = true;
}

async function onSave(next: DebtsConfig) {
  saving.value = true;
  const ok = await saveDebts(next);
  saving.value = false;
  if (ok) editing.value = false;
}
</script>

<style scoped>
.debts {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.align-right { text-align: right; }

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

.debt-sum-alert { border-color: var(--expense); }

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
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
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
  transition: background-color var(--transition-base), color var(--transition-base);
}

.unconfigured-btn:hover {
  background: var(--warning);
  color: #fff;
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

.debt-section-ic { color: var(--text-tertiary); }

.debt-section-count {
  color: var(--text-tertiary);
  font-weight: 500;
}

.debt-edit-btn {
  gap: 6px;
  color: var(--text-secondary);
}

/* ===== 卡片网格 ===== */
.debt-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-4);
}

.debt-card {
  position: relative;
  background: var(--surface-1);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  overflow: hidden;
}

.debt-card-overdue { border-color: var(--expense); }

.debt-card-bar {
  position: absolute;
  inset: 0 0 auto 0;
  height: 3px;
  background: var(--expense);
}

.debt-card-top {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
}

.debt-card-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: var(--surface-3);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex: none;
}

.debt-card-id {
  flex: 1;
  min-width: 0;
}

.debt-card-name {
  font-size: var(--font-size-base);
  font-weight: 620;
}

.debt-card-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.debt-badge {
  font-size: var(--font-size-xs);
  padding: 2px 9px;
  border-radius: var(--radius-full);
  font-weight: 650;
  white-space: nowrap;
  flex: none;
}

.badge-paid { background: var(--income-light); color: var(--income); }
.badge-pending { background: var(--warning-light); color: var(--warning); }
.badge-overdue { background: var(--expense-light); color: var(--expense); }
.badge-idle { background: var(--surface-3); color: var(--text-secondary); }

.debt-card-figures {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-top: var(--space-4);
}

.debt-fig-label {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.debt-fig-due {
  font-size: var(--font-size-xl);
  font-weight: 700;
}

.debt-fig-remain {
  font-size: var(--font-size-lg);
  font-weight: 700;
}

.debt-card-progress {
  margin-top: var(--space-3);
}

.debt-bar { height: 7px; }

.debt-card-progress-cap {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.debt-card-foot {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--hairline);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.debt-foot-date {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.debt-foot-ic { color: var(--text-tertiary); }
.debt-foot-strong { color: var(--text-primary); }

.debt-countdown {
  font-size: var(--font-size-sm);
  font-weight: 700;
}

.debt-missing {
  margin-top: var(--space-3);
  font-size: var(--font-size-xs);
  color: var(--expense);
}

.empty-state {
  text-align: center;
  padding: var(--space-6);
  color: var(--text-tertiary);
  background: var(--surface-1);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
}

@media (max-width: 900px) {
  .debts-summary { grid-template-columns: 1fr; }
  .debt-grid { grid-template-columns: 1fr; }
}
</style>
