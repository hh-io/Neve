<template>
  <div class="bg">
    <!-- 汇总:预算总额 / 已使用 / 剩余可用 -->
    <div class="bg-summary">
      <div class="card bg-sum-card">
        <span class="bg-sum-label">本月预算总额</span>
        <span class="bg-sum-value tabular-nums">¥{{ totalBudget.toFixed(0) }}</span>
      </div>
      <div class="card bg-sum-card">
        <span class="bg-sum-label">已使用</span>
        <span class="bg-sum-value tabular-nums">¥{{ totalSpent.toFixed(0) }}</span>
      </div>
      <div class="card bg-sum-card" :class="{ 'bg-sum-alert': overCount > 0 }">
        <span class="bg-sum-label">剩余可用 · <span class="tabular-nums">{{ overCount }}</span> 项超支</span>
        <span class="bg-sum-value tabular-nums" :style="{ color: budgetLeft < 0 ? 'var(--expense)' : 'var(--text-primary)' }">
          ¥{{ budgetLeft.toFixed(0) }}
        </span>
      </div>
    </div>

    <!-- 分类预算进度 -->
    <section class="section-card">
      <div class="section-head">
        <h3 class="section-title">分类预算进度</h3>
        <button class="bg-edit-btn" @click="showEdit = !showEdit">
          <component :is="showEdit ? Check : Pencil" :size="14" />
          {{ showEdit ? '完成' : '编辑配置' }}
        </button>
      </div>

      <!-- 编辑 -->
      <div v-if="showEdit" class="section-body bg-edit">
        <div class="bg-edit-labels">
          <span>分类</span><span>月度预算</span><span></span>
        </div>
        <div v-for="(budget, cat) in budgets" :key="cat" class="bg-edit-row">
          <span class="bg-edit-cat">
            <component :is="getCategoryIcon(cat)" :size="15" class="bg-edit-cat-ic" />
            {{ getCategoryLabel(cat) }}
            <span class="bg-edit-spent tabular-nums">已花 ¥{{ getSpent(cat).toFixed(0) }}</span>
          </span>
          <div class="bg-input-wrap">
            <span class="bg-input-prefix">¥</span>
            <input
              type="number"
              min="0"
              :value="budget"
              class="bg-input tabular-nums"
              @input="updateBudget(cat, ($event.target as HTMLInputElement).value)"
            />
          </div>
          <button class="bg-del-btn" title="删除分类" @click="deleteBudget(cat)">
            <Trash2 :size="15" />
          </button>
        </div>
        <!-- 新增分类 -->
        <div class="bg-edit-add">
          <div class="bg-select-wrap">
            <select v-model="newCategory" class="bg-select">
              <option value="" disabled>选择分类</option>
              <option v-for="cat in availableCategories" :key="cat" :value="cat">{{ getCategoryLabel(cat) }}</option>
            </select>
            <ChevronDown :size="15" class="bg-select-caret" />
          </div>
          <div class="bg-input-wrap">
            <span class="bg-input-prefix">¥</span>
            <input v-model.number="newAmount" type="number" min="0" placeholder="金额" class="bg-input tabular-nums" />
          </div>
          <button class="bg-add-btn" title="添加" :disabled="!newCategory || !newAmount" @click="addBudget">
            <Plus :size="16" />
          </button>
        </div>
      </div>

      <!-- 视图 -->
      <div v-else-if="budgetRows.length > 0" class="section-body bg-bars">
        <div v-for="row in budgetRows" :key="row.cat" class="bg-bar-item">
          <div class="bg-bar-head">
            <span class="bg-bar-name">
              <component :is="getCategoryIcon(row.cat)" :size="15" class="bg-bar-ic" />
              {{ getCategoryLabel(row.cat) }}
              <span v-if="row.over" class="bg-over-badge">超支</span>
            </span>
            <span class="bg-bar-nums tabular-nums">¥{{ row.spent.toFixed(0) }} / ¥{{ row.limit.toFixed(0) }}</span>
          </div>
          <div class="progress-bar bg-bar-track">
            <div class="progress-fill" :style="{ width: Math.min(row.pct, 100) + '%', background: row.color }"></div>
          </div>
        </div>
      </div>
      <div v-else class="section-body bg-empty">点击“编辑配置”添加预算分类</div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue';
import { Pencil, Check, Plus, Trash2, ChevronDown } from '@lucide/vue';
import { useAnalytics } from '../composables/useAnalytics';
import { useBudgets } from '../composables/useBudgets';
import { getCategoryLabel } from '../composables/useCategories';
import { getCategoryIcon } from '../composables/useCategoryIcon';

const { analytics } = useAnalytics();
const { budgets, loadBudgets, saveBudgets } = useBudgets();

const expenseByCategory = computed(() => analytics.value?.expenseByCategory || []);
const allCategories = computed(() => expenseByCategory.value.map(e => e.category));

const showEdit = ref(false);
const newCategory = ref('');
const newAmount = ref<number | null>(null);

onMounted(loadBudgets);

// 预算变更即写回服务端(内部含 429/错误处理)+ localStorage 备份
watch(budgets, saveBudgets, { deep: true });

const availableCategories = computed(() => {
  return allCategories.value.filter(cat => !budgets.value[cat]);
});

function getSpent(category: string): number {
  const found = expenseByCategory.value.find(e => e.category === category);
  return found?.amount || 0;
}

function getProgress(category: string): number {
  const spent = getSpent(category);
  const budget = budgets.value[category] || 1;
  return (spent / budget) * 100;
}

// 进度条颜色:超支红 / 临界(>80%)黄 / 正常靛蓝
function barColor(pct: number): string {
  if (pct > 100) return 'var(--expense)';
  if (pct > 80) return 'var(--warning)';
  return 'var(--accent)';
}

const budgetRows = computed(() =>
  Object.keys(budgets.value).map(cat => {
    const spent = getSpent(cat);
    const limit = budgets.value[cat];
    const pct = getProgress(cat);
    return { cat, spent, limit, pct, over: pct > 100, color: barColor(pct) };
  })
);

function updateBudget(category: string, value: string) {
  const num = parseFloat(value);
  if (num > 0) {
    budgets.value[category] = num;
  } else {
    delete budgets.value[category];
  }
}

function addBudget() {
  if (newCategory.value && newAmount.value && !budgets.value[newCategory.value]) {
    budgets.value[newCategory.value] = newAmount.value;
    newCategory.value = '';
    newAmount.value = null;
  }
}

function deleteBudget(category: string) {
  delete budgets.value[category];
}

const totalBudget = computed(() => Object.values(budgets.value).reduce((sum, b) => sum + b, 0));
const totalSpent = computed(() => Object.keys(budgets.value).reduce((sum, cat) => sum + getSpent(cat), 0));
const budgetLeft = computed(() => totalBudget.value - totalSpent.value);
const overCount = computed(() => Object.keys(budgets.value).filter(cat => getProgress(cat) > 100).length);
</script>

<style scoped>
.bg {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* ===== 汇总卡 ===== */
.bg-summary {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: var(--space-4);
}

.bg-sum-card {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.bg-sum-alert { border-color: var(--warning); }

.bg-sum-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.bg-sum-value {
  font-size: var(--font-size-2xl);
  font-weight: 700;
}

/* ===== 编辑按钮 ===== */
.bg-edit-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  border: 1px solid var(--hairline);
  background: var(--surface-2);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: border-color var(--transition-base), color var(--transition-base);
}

.bg-edit-btn:hover {
  border-color: var(--hairline-strong);
  color: var(--text-primary);
}

/* ===== 进度条视图 ===== */
.bg-bars {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.bg-bar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.bg-bar-name {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  font-weight: 550;
  color: var(--text-primary);
}

.bg-bar-ic { color: var(--text-tertiary); }

.bg-over-badge {
  padding: 1px 7px;
  border-radius: var(--radius-full);
  background: var(--expense-light);
  color: var(--expense);
  font-size: 11px;
  font-weight: 650;
}

.bg-bar-nums {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.bg-bar-track { height: 8px; }

/* ===== 编辑视图 ===== */
.bg-edit {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.bg-edit-labels,
.bg-edit-row,
.bg-edit-add {
  display: grid;
  grid-template-columns: 1fr 160px 40px;
  gap: var(--space-3);
  align-items: center;
}

.bg-edit-labels {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  padding: 0 var(--space-1);
}

.bg-edit-cat {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-sm);
  font-weight: 550;
  color: var(--text-primary);
}

.bg-edit-cat-ic { color: var(--text-tertiary); }

.bg-edit-spent {
  color: var(--text-tertiary);
  font-weight: 400;
  font-size: var(--font-size-xs);
}

.bg-edit-add {
  margin-top: var(--space-2);
  padding-top: var(--space-4);
  border-top: 1px dashed var(--hairline);
}

.bg-input-wrap,
.bg-select-wrap {
  position: relative;
}

.bg-input-prefix {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
  font-size: var(--font-size-sm);
}

.bg-input {
  width: 100%;
  padding: 9px 12px 9px 26px;
  border-radius: var(--radius-md);
  border: 1px solid var(--hairline);
  background: var(--surface-2);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
}

.bg-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-subtle);
}

.bg-select {
  width: 100%;
  padding: 9px 30px 9px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--hairline);
  background: var(--surface-2);
  color: var(--text-primary);
  font-size: var(--font-size-sm);
  cursor: pointer;
  appearance: none;
}

.bg-select:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-subtle);
}

.bg-select-caret {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
  pointer-events: none;
}

.bg-del-btn,
.bg-add-btn {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all var(--transition-base);
}

.bg-del-btn {
  border: 1px solid var(--hairline);
  background: transparent;
  color: var(--text-tertiary);
}

.bg-del-btn:hover {
  border-color: var(--expense);
  color: var(--expense);
  background: var(--expense-light);
}

.bg-add-btn {
  border: none;
  background: var(--accent);
  color: #fff;
}

.bg-add-btn:hover { background: var(--accent-hover); }
.bg-add-btn:disabled { background: var(--text-tertiary); cursor: not-allowed; }

.bg-empty {
  text-align: center;
  color: var(--text-tertiary);
}

@media (max-width: 768px) {
  .bg-summary { grid-template-columns: 1fr; }
}
</style>
