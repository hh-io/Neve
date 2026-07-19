<template>
  <div class="animate-fade-in-up">
    <!-- 全局看板 -->
    <div class="debts-summary section-mb">
      <div class="card-static stat-card">
        <div class="stat-label">本期总应还</div>
        <div class="stat-value">{{ formatMoney(summary?.monthDue ?? 0) }}</div>
      </div>
      <div class="card-static stat-card">
        <div class="stat-label">剩余待还</div>
        <div class="stat-value" :class="(summary?.monthRemaining ?? 0) > 0 ? 'text-expense' : 'text-income'">
          {{ formatMoney(summary?.monthRemaining ?? 0) }}
        </div>
      </div>
      <div class="card-static stat-card">
        <div class="stat-label">最近还款日</div>
        <template v-if="!report">
          <div class="stat-value">—</div>
          <div class="countdown-sub">服务端不可达,计算不可用</div>
        </template>
        <template v-else-if="!summary?.nextDueDate">
          <div class="stat-value text-income">本期已结清</div>
        </template>
        <template v-else>
          <div class="stat-value" :class="{ 'text-expense': summary.nextDueInDays < 0 }">
            {{ countdownText }}
          </div>
          <div class="countdown-sub">{{ summary.nextDueName }} · {{ summary.nextDueDate }}</div>
        </template>
      </div>
    </div>

    <!-- 有欠款但未配置的账户 -->
    <div v-if="!editing && unconfigured.length" class="card-static unconfigured-banner section-mb">
      <div class="unconfigured-text">
        <AlertTriangle :size="16" />
        <span>{{ unconfigured.length }} 个负债账户有欠款但未配置账单周期:</span>
      </div>
      <button
        v-for="item in unconfigured"
        :key="item.account"
        class="filter-pill"
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
      <!-- 额度类 -->
      <div class="card-static panel section-mb">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-brand-light">
              <CreditCard :size="20" color="var(--accent)" />
            </div>
            <span class="panel-title">额度账单</span>
          </div>
          <button class="btn btn-ghost" @click="startEdit()">编辑配置</button>
        </div>

        <div v-if="!revolving.length" class="empty-state">点击"编辑配置"添加信用卡/白条等账单周期</div>
        <div v-for="rv in revolving" :key="rv.account" class="debt-row">
          <div class="debt-row-main">
            <div class="debt-row-title">
              <span class="debt-name">{{ rv.name }}</span>
              <span class="debt-badge" :class="revolvingBadge(rv).cls">{{ revolvingBadge(rv).text }}</span>
              <span v-if="rv.accountMissing" class="debt-badge badge-missing">账本无此账户</span>
            </div>
            <span class="debt-remaining" :class="rv.remaining > 0 ? 'text-expense' : 'text-income'">
              {{ formatMoney(rv.remaining) }}
            </span>
          </div>
          <div class="debt-row-sub">
            <span>账单 {{ shortDate(rv.statementDate) }} → 还款 {{ shortDate(rv.dueDate) }}</span>
            <span>本期应还 {{ formatMoney(rv.statementDue) }}</span>
            <span>已还 {{ formatMoney(rv.paidSince) }}</span>
            <span>当前欠款 {{ formatMoney(rv.currentBalance) }}</span>
          </div>
          <div class="debt-progress-wrap">
            <div class="debt-progress" :class="{ over: rv.overdue }" :style="{ width: paidPercent(rv) + '%' }"></div>
          </div>
        </div>
      </div>

      <!-- 分期类 -->
      <div class="card-static panel">
        <div class="panel-head">
          <div class="panel-head-left">
            <div class="panel-icon bg-brand-light">
              <Landmark :size="20" color="var(--accent)" />
            </div>
            <span class="panel-title">固定分期</span>
          </div>
          <button class="btn btn-ghost" @click="startEdit()">编辑配置</button>
        </div>

        <div v-if="!installments.length" class="empty-state">点击"编辑配置"添加房贷/车贷等固定月供</div>
        <div v-for="ins in installments" :key="ins.id" class="debt-row">
          <div class="debt-row-main">
            <div class="debt-row-title">
              <span class="debt-name">{{ ins.name }}</span>
              <span class="debt-badge" :class="installmentBadge(ins).cls">{{ installmentBadge(ins).text }}</span>
              <span v-if="ins.accountMissing" class="debt-badge badge-missing">账本无此账户</span>
            </div>
            <span class="debt-remaining" :class="ins.paid || !ins.monthlyAmount ? 'text-income' : 'text-expense'">
              {{ ins.monthlyAmount ? formatMoney(ins.monthlyAmount) : '未生效' }}
            </span>
          </div>
          <div class="debt-row-sub">
            <span>还款日 {{ shortDate(ins.dueDate) }}</span>
            <span v-if="ins.paid">本月已还 {{ formatMoney(ins.paidAmount) }}</span>
            <span>剩余本金 {{ formatMoney(ins.currentBalance) }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue';
import { CreditCard, Landmark, AlertTriangle } from '@lucide/vue';
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

const countdownText = computed(() => {
  const days = summary.value?.nextDueInDays ?? 0;
  if (days < 0) return `已逾期 ${-days} 天`;
  if (days === 0) return '今天到期';
  return `${days} 天后`;
});

function shortDate(date: string): string {
  return date.slice(5).replace('-', '/');
}

function paidPercent(rv: RevolvingStatus): number {
  if (rv.statementDue <= 0) return 100;
  return Math.min((rv.paidSince / rv.statementDue) * 100, 100);
}

function revolvingBadge(rv: RevolvingStatus): { text: string; cls: string } {
  if (rv.statementDue <= 0) return { text: '本期无账单', cls: 'badge-idle' };
  if (rv.remaining <= 0) return { text: '已结清', cls: 'badge-paid' };
  if (rv.overdue) return { text: `逾期 ${-rv.daysUntilDue} 天`, cls: 'badge-overdue' };
  if (rv.daysUntilDue === 0) return { text: '今天到期', cls: 'badge-pending' };
  return { text: `${rv.daysUntilDue} 天内还款`, cls: 'badge-pending' };
}

function installmentBadge(ins: InstallmentStatus): { text: string; cls: string } {
  if (!ins.monthlyAmount) return { text: '尚未生效', cls: 'badge-idle' };
  if (ins.paid) return { text: '本月已还', cls: 'badge-paid' };
  if (ins.overdue) return { text: `逾期 ${-ins.daysUntilDue} 天`, cls: 'badge-overdue' };
  if (ins.daysUntilDue === 0) return { text: '今天到期', cls: 'badge-pending' };
  return { text: `${ins.daysUntilDue} 天内还款`, cls: 'badge-pending' };
}

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
.debts-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-6);
}

@media (max-width: 768px) {
  .debts-summary {
    grid-template-columns: 1fr;
    gap: var(--space-3);
  }
}

.countdown-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-top: var(--space-1);
}

.unconfigured-banner {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
}

.unconfigured-text {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  color: var(--warning);
}

.debt-row {
  padding: var(--space-3) 0;
  border-top: 1px solid var(--hairline);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.debt-row-main {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.debt-row-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.debt-name {
  font-weight: 600;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
}

.debt-remaining {
  font-weight: 600;
  font-size: var(--font-size-base);
  font-variant-numeric: tabular-nums;
}

.debt-row-sub {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2) var(--space-4);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
}

.debt-badge {
  font-size: var(--font-size-xs);
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.badge-paid { background: var(--income-light); color: var(--income); }
.badge-pending { background: var(--warning-light); color: var(--warning); }
.badge-overdue { background: var(--expense-light); color: var(--expense); }
.badge-idle { background: var(--surface-2); color: var(--text-secondary); }
.badge-missing { background: var(--expense-light); color: var(--expense); }

.debt-progress-wrap {
  height: 6px;
  background: var(--surface-2);
  border-radius: 3px;
  overflow: hidden;
}

.debt-progress {
  height: 100%;
  background: var(--income);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.debt-progress.over {
  background: var(--expense);
}

.empty-state {
  text-align: center;
  padding: var(--space-6);
  color: var(--text-tertiary);
}
</style>
