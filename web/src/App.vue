<template>
  <div class="app">
    <!-- Header -->
    <header class="header">
      <div class="header-content">
        <div class="logo">
          <img src="/neve.svg" alt="Neve" class="logo-icon" />
          <span class="logo-text">Neve</span>
        </div>
        
        <!-- Time Period Selector -->
        <div class="time-selector" v-if="analytics">
          <button class="time-btn" :class="{ active: timePeriod === 'month' }" @click="setTimePeriod('month')">本月</button>
          <button class="time-btn" :class="{ active: timePeriod === 'lastMonth' }" @click="setTimePeriod('lastMonth')">上月</button>
          <button class="time-btn" :class="{ active: timePeriod === 'year' }" @click="setTimePeriod('year')">本年</button>
          <button class="time-btn" :class="{ active: timePeriod === 'all' }" @click="setTimePeriod('all')">全部</button>
        </div>
        
        <button class="btn btn-secondary" @click="refresh" :disabled="loading">
          <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M23 4v6h-6M1 20v-6h6M20.49 9A9 9 0 1 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15" />
          </svg>
          <span>{{ loading ? "刷新中..." : "刷新数据" }}</span>
        </button>
      </div>
    </header>

    <!-- Main Content -->
    <main class="container">
      <!-- Error State -->
      <div v-if="error" class="error-card glass-card">
        <h3>加载失败</h3>
        <p>{{ error }}</p>
        <button class="btn" @click="refresh">重试</button>
      </div>

      <!-- Loading State -->
      <div v-else-if="loading && !analytics" class="loading-state">
        <div class="spinner"></div>
        <p>加载中...</p>
      </div>

      <!-- Dashboard -->
      <template v-else-if="analytics">
        <!-- Tab Navigation -->
        <TabNavigation v-model="activeTab" :tabs="tabs" />

        <!-- Tab Contents -->
        <OverviewTab v-show="activeTab === 'overview'" :analytics="analytics" />
        <SpendingTab v-show="activeTab === 'spending'" :analytics="analytics" />
        <TrendsTab v-show="activeTab === 'trends'" :analytics="analytics" />
        <AccountsTab v-show="activeTab === 'accounts'" :analytics="analytics" />
        
        <div v-show="activeTab === 'budget'" class="tab-content">
          <section class="analytics-section fade-in">
            <BudgetCard 
              :expenseByCategory="analytics.expenseByCategory || []"
              :allCategories="allCategories"
            />
          </section>
        </div>
        
        <TransactionsTab v-show="activeTab === 'transactions'" :transactions="analytics.recentTransactions || []" />

        <!-- Footer -->
        <footer class="footer">
          <p>最后更新: {{ formatDateTime(analytics.summary.lastUpdated) }}</p>
        </footer>
      </template>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from "vue";

// Components
import TabNavigation from "./components/TabNavigation.vue";
import OverviewTab from "./components/tabs/OverviewTab.vue";
import SpendingTab from "./components/tabs/SpendingTab.vue";
import TrendsTab from "./components/tabs/TrendsTab.vue";
import AccountsTab from "./components/tabs/AccountsTab.vue";
import TransactionsTab from "./components/tabs/TransactionsTab.vue";
import BudgetCard from "./components/BudgetCard.vue";

// Composables
import { formatDateTime } from "./composables/useFormatters";

// State
const analytics = ref(null);
const loading = ref(false);
const error = ref(null);

// Time period
const timePeriod = ref('month');

function setTimePeriod(period) {
  timePeriod.value = period;
}

// Tab navigation
const activeTab = ref('overview');
const tabs = [
  { id: 'overview', label: '概览', icon: '📊' },
  { id: 'spending', label: '收支', icon: '💰' },
  { id: 'trends', label: '趋势', icon: '📈' },
  { id: 'accounts', label: '账户', icon: '💳' },
  { id: 'budget', label: '预算', icon: '🎯' },
  { id: 'transactions', label: '交易', icon: '📝' },
];

// Categories for budget
const allCategories = computed(() => {
  if (!analytics.value?.expenseByCategory) return [];
  return analytics.value.expenseByCategory.map(e => e.category);
});

// API
async function fetchAnalytics() {
  const response = await fetch("/api/analytics");
  if (!response.ok) throw new Error("Failed to fetch analytics");
  return response.json();
}

async function refresh() {
  loading.value = true;
  error.value = null;
  try {
    await fetch("/api/refresh", { method: "POST" });
    analytics.value = await fetchAnalytics();
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}

// Initial load
onMounted(async () => {
  loading.value = true;
  try {
    analytics.value = await fetchAnalytics();
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
});
</script>

<style scoped>
/* Import all the original styles - keeping CSS in App.vue for now */
.app {
  min-height: 100vh;
}

/* Tab Content */
.tab-content {
  padding-top: var(--space-6);
}

/* Header */
.header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
}

.header-content {
  max-width: 1400px;
  margin: 0 auto;
  padding: var(--space-4) var(--space-6);
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-4);
}

.time-selector {
  display: flex;
  gap: var(--space-1);
  background: rgba(0, 0, 0, 0.04);
  padding: var(--space-1);
  border-radius: var(--radius-lg);
}

.time-btn {
  padding: var(--space-2) var(--space-4);
  border: none;
  background: transparent;
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
}

.time-btn:hover { color: var(--color-text-primary); }
.time-btn.active { background: white; color: var(--color-blue); box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1); }

.logo {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.logo-icon { width: 36px; height: 36px; }
.logo-text { font-size: var(--font-size-xl); font-weight: 700; color: var(--color-text-primary); }

.container { max-width: 1400px; margin: 0 auto; padding: var(--space-6); }

/* Error & Loading */
.error-card { text-align: center; padding: var(--space-8); }
.error-card h3 { color: var(--color-red); margin-bottom: var(--space-4); }

.loading-state { display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 50vh; gap: var(--space-4); }

.spinner {
  width: 48px; height: 48px;
  border: 3px solid var(--color-blue);
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* Buttons */
.btn {
  display: inline-flex; align-items: center; gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border: none; border-radius: var(--radius-md);
  font-size: var(--font-size-sm); font-weight: 600;
  cursor: pointer; transition: all var(--transition-fast);
}
.btn:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-secondary { background: var(--glass-bg); backdrop-filter: var(--glass-blur); color: var(--color-text-primary); border: 1px solid rgba(0, 0, 0, 0.1); }
.btn-secondary:hover:not(:disabled) { background: rgba(0, 0, 0, 0.05); }

.icon { width: 16px; height: 16px; }

/* Footer */
.footer { text-align: center; padding: var(--space-8); color: var(--color-text-tertiary); font-size: var(--font-size-sm); border-top: 1px solid rgba(0, 0, 0, 0.05); margin-top: var(--space-8); }

/* Dark Mode */
@media (prefers-color-scheme: dark) {
  .app { background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f0f23 100%); }
  .header { background: rgba(20, 20, 35, 0.9); border-bottom-color: rgba(255, 255, 255, 0.1); }
  .time-selector { background: rgba(255, 255, 255, 0.08); }
  .time-btn { color: rgba(255, 255, 255, 0.6); }
  .time-btn.active { background: rgba(90, 200, 250, 0.2); color: #5AC8FA; box-shadow: none; }
  .footer { border-top-color: rgba(255, 255, 255, 0.1); color: rgba(255, 255, 255, 0.5); }
}
</style>
