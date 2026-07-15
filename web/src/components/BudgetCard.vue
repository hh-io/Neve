<template>
  <div class="card-static budget-card">
    <div class="budget-header">
      <h3 class="budget-title">本月预算</h3>
      <button class="edit-btn" @click="showEdit = !showEdit">
        {{ showEdit ? '完成' : '设置' }}
      </button>
    </div>

    <!-- Edit Mode -->
    <div v-if="showEdit" class="budget-edit">
      <div v-for="(budget, cat) in budgets" :key="cat" class="budget-edit-row">
        <span class="budget-cat">{{ cat }}</span>
        <input 
          type="number" 
          :value="budget"
          @change="updateBudget(cat, $event.target.value)"
          class="budget-input"
          placeholder="0"
        />
        <button @click="deleteBudget(cat)" class="delete-btn" title="删除">×</button>
      </div>
      <div class="budget-edit-row add-row">
        <input 
          v-model="newCategory" 
          class="budget-select"
          placeholder="输入分类名称..."
          list="category-suggestions"
        />
        <datalist id="category-suggestions">
          <option v-for="cat in availableCategories" :key="cat" :value="cat" />
        </datalist>
        <button @click="addBudget" class="add-btn" :disabled="!newCategory">+</button>
      </div>
    </div>

    <!-- Display Mode -->
    <div v-else class="budget-list">
      <div v-for="(budget, cat) in budgets" :key="cat" class="budget-item">
        <div class="budget-info">
          <span class="budget-category">{{ cat }}</span>
          <span class="budget-amount">
            ¥{{ getSpent(cat).toFixed(0) }} / ¥{{ budget }}
          </span>
        </div>
        <div class="budget-progress-wrap">
          <div 
            class="budget-progress" 
            :style="{ width: Math.min(getProgress(cat), 100) + '%' }"
            :class="{ over: getProgress(cat) > 100, warning: getProgress(cat) > 80 }"
          ></div>
        </div>
        <span class="budget-percent" :class="{ over: getProgress(cat) > 100 }">
          {{ getProgress(cat).toFixed(0) }}%
        </span>
      </div>
      <div v-if="Object.keys(budgets).length === 0" class="empty-state">
        点击"设置"添加预算
      </div>
    </div>

    <!-- Total -->
    <div v-if="Object.keys(budgets).length > 0 && !showEdit" class="budget-total">
      <div class="total-row">
        <span>总预算</span>
        <span>¥{{ totalSpent.toFixed(0) }} / ¥{{ totalBudget }}</span>
      </div>
      <div class="budget-progress-wrap">
        <div 
          class="budget-progress" 
          :style="{ width: Math.min(totalProgress, 100) + '%' }"
          :class="{ over: totalProgress > 100, warning: totalProgress > 80 }"
        ></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue';

const props = defineProps({
  expenseByCategory: { type: Array, default: () => [] },
  allCategories: { type: Array, default: () => [] }
});

const showEdit = ref(false);
const newCategory = ref('');

// Load budgets from server (with localStorage fallback)
const budgets = ref({});

// Fetch budgets from server on mount
onMounted(async () => {
  try {
    const res = await fetch('/api/budgets');
    if (res.ok) {
      budgets.value = await res.json();
    }
  } catch {
    // Fallback to localStorage
    budgets.value = JSON.parse(localStorage.getItem('neve-budgets') || '{}');
  }
});

// Save budgets to server when changed
watch(budgets, async (val) => {
  // Also save to localStorage as backup
  localStorage.setItem('neve-budgets', JSON.stringify(val));
  
  try {
    await fetch('/api/budgets', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(val)
    });
  } catch (e) {
    console.error('Failed to save budgets to server:', e);
  }
}, { deep: true });

const availableCategories = computed(() => {
  return props.allCategories.filter(cat => !budgets.value[cat]);
});

function getSpent(category) {
  const found = props.expenseByCategory.find(e => e.category === category);
  return found?.amount || 0;
}

function getProgress(category) {
  const spent = getSpent(category);
  const budget = budgets.value[category] || 1;
  return (spent / budget) * 100;
}

function updateBudget(category, value) {
  const num = parseFloat(value);
  if (num > 0) {
    budgets.value[category] = num;
  } else {
    delete budgets.value[category];
  }
}

function addBudget() {
  if (newCategory.value && !budgets.value[newCategory.value]) {
    budgets.value[newCategory.value] = 1000;
    newCategory.value = '';
  }
}

function deleteBudget(category) {
  delete budgets.value[category];
}

const totalBudget = computed(() => {
  return Object.values(budgets.value).reduce((sum, b) => sum + b, 0);
});

const totalSpent = computed(() => {
  return Object.keys(budgets.value).reduce((sum, cat) => sum + getSpent(cat), 0);
});

const totalProgress = computed(() => {
  return totalBudget.value > 0 ? (totalSpent.value / totalBudget.value) * 100 : 0;
});
</script>

<style scoped>
.budget-card {
  min-height: 200px;
  padding: var(--space-6);
}

.budget-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
}

.budget-title {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
}

.edit-btn {
  padding: var(--space-2) var(--space-3);
  border: none;
  background: var(--brand-light);
  color: var(--brand-primary);
  font-size: var(--font-size-sm);
  font-weight: 500;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.budget-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.budget-item {
  display: grid;
  grid-template-columns: 1fr 120px 50px;
  align-items: center;
  gap: var(--space-3);
}

.budget-info {
  display: flex;
  flex-direction: column;
}

.budget-category {
  font-weight: 500;
  font-size: var(--font-size-sm);
}

.budget-amount {
  font-size: 12px;
  color: var(--text-tertiary);
}

.budget-progress-wrap {
  height: 6px;
  background: var(--bg-tertiary);
  border-radius: 3px;
  overflow: hidden;
}

.budget-progress {
  height: 100%;
  background: var(--income);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.budget-progress.warning {
  background: var(--warning);
}

.budget-progress.over {
  background: var(--expense);
}

.budget-percent {
  font-size: var(--font-size-sm);
  font-weight: 600;
  text-align: right;
  color: var(--text-secondary);
}

.budget-percent.over {
  color: var(--expense);
}

.budget-total {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px dashed rgba(0, 0, 0, 0.1);
}

.total-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--space-2);
  font-weight: 600;
}

/* Edit Mode */
.budget-edit {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.budget-edit-row {
  display: flex;
  gap: var(--space-3);
  align-items: center;
}

.budget-cat {
  flex: 1;
  font-size: var(--font-size-sm);
}

.budget-input {
  width: 100px;
  padding: var(--space-2);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  text-align: right;
}

.budget-select {
  flex: 1;
  padding: var(--space-2);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
}

.add-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: var(--brand-primary);
  color: white;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 18px;
}

.add-btn:disabled {
  background: var(--text-tertiary);
}

.empty-state {
  text-align: center;
  padding: var(--space-6);
  color: var(--text-tertiary);
}

.delete-btn {
  width: 28px;
  height: 28px;
  border: none;
  background: rgba(255, 59, 48, 0.1);
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
</style>
