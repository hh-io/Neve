<template>
  <div class="app-layout" :class="themeClass">
    <!-- Sidebar -->
    <AppSidebar 
      :activeTab="activeTab" 
      :showStats="!!analytics"
      :transactionCount="totalTransactionCount"
      :trackingDays="trackingDays"
      @update:activeTab="activeTab = $event"
    />

    <!-- Main Content -->
    <main class="main-content">
      <!-- Error State -->
      <div v-if="error" class="card-static" style="padding: var(--space-8); text-align: center;">
        <h3 style="color: var(--expense); margin-bottom: var(--space-4);">加载失败</h3>
        <p style="color: var(--text-secondary);">{{ error }}</p>
        <button class="btn btn-primary" style="margin-top: var(--space-4);" @click="refresh">重试</button>
      </div>

      <!-- Loading State -->
      <div v-else-if="loading && !analytics" style="display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 50vh; gap: var(--space-4);">
        <div style="width: 48px; height: 48px; border: 3px solid var(--brand-primary); border-top-color: transparent; border-radius: 50%; animation: spin 1s linear infinite;"></div>
        <p style="color: var(--text-secondary);">加载中...</p>
      </div>

      <!-- Dashboard -->
      <template v-else-if="analytics">
        <!-- Page Header -->
        <header class="page-header animate-fade-in-up">
          <div class="page-title">
            <h2>{{ currentPageTitle }}</h2>
            <p>{{ currentPageDesc }}</p>
          </div>
          <div class="header-actions">
            <ThemeSwitcher v-model="themeMode" />
            
            <button class="btn btn-secondary btn-refresh" @click="refresh" :disabled="loading">
              <span v-html="icons.refresh" style="width: 16px; height: 16px;"></span>
              <span>{{ loading ? '刷新中...' : '刷新数据' }}</span>
            </button>
          </div>
        </header>

        <!-- Tab Contents -->
        <OverviewTab v-show="activeTab === 'overview'" :analytics="analytics" />
        <SpendingTab v-show="activeTab === 'spending'" :analytics="analytics" />
        <TrendsTab v-show="activeTab === 'trends'" :analytics="analytics" />
        <AccountsTab v-show="activeTab === 'accounts'" :analytics="analytics" />
        
        <div v-show="activeTab === 'budget'" class="section-mb">
          <BudgetCard 
            :expenseByCategory="analytics.expenseByCategory || []"
            :allCategories="allCategories"
          />
        </div>
        
        <TransactionsTab v-show="activeTab === 'transactions'" :transactions="analytics.recentTransactions || []" />

        <!-- Footer -->
        <footer style="text-align: center; padding: var(--space-8); color: var(--text-tertiary); font-size: var(--font-size-sm); border-top: 1px solid var(--border); margin-top: var(--space-8);">
          <p>最后更新: {{ formatDateTime(analytics.summary?.lastUpdated) }}</p>
        </footer>
      </template>
    </main>

    <!-- FAB -->
    <button class="fab" title="新增交易">
      <span v-html="icons.plus"></span>
    </button>

    <!-- Toast -->
    <AppToast :show="toast.show" :message="toast.message" :type="toast.type" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from "vue";

// Layout Components
import AppSidebar from "./components/layout/AppSidebar.vue";
import ThemeSwitcher from "./components/layout/ThemeSwitcher.vue";
import AppToast from "./components/common/AppToast.vue";

// Tab Components
import OverviewTab from "./components/tabs/OverviewTab.vue";
import SpendingTab from "./components/tabs/SpendingTab.vue";
import TrendsTab from "./components/tabs/TrendsTab.vue";
import AccountsTab from "./components/tabs/AccountsTab.vue";
import TransactionsTab from "./components/tabs/TransactionsTab.vue";
import BudgetCard from "./components/BudgetCard.vue";

// Composables
import { formatDateTime } from "./composables/useFormatters";
import { icons } from "./composables/icons";

// State
const analytics = ref(null);
const loading = ref(false);
const error = ref(null);
const activeTab = ref('overview');

// Toast
const toast = ref({ show: false, message: '', type: 'success' });

function showToast(message, type = 'success', duration = 3000) {
  toast.value = { show: true, message, type };
  setTimeout(() => { toast.value.show = false; }, duration);
}

// Theme
const themeMode = ref('system');

const themeClass = computed(() => {
  if (themeMode.value === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'theme-dark' : 'theme-light';
  }
  return `theme-${themeMode.value}`;
});

function applyTheme() {
  const html = document.documentElement;
  html.classList.remove('theme-light', 'theme-dark', 'theme-geek');
  html.classList.add(themeClass.value);
}

watch(themeClass, applyTheme, { immediate: true });
watch(themeMode, (mode) => localStorage.setItem('neve-theme', mode));

// Page info
const pageMeta = {
  overview: { title: '概览', desc: '欢迎回来，这是您的财务概况' },
  spending: { title: '收支分析', desc: '查看收入与支出的详细分析' },
  trends: { title: '趋势图表', desc: '了解您的财务变化趋势' },
  accounts: { title: '账户管理', desc: '管理您的所有账户' },
  budget: { title: '预算管理', desc: '设置并跟踪您的预算目标' },
  transactions: { title: '交易明细', desc: '查看所有交易记录' }
};

const currentPageTitle = computed(() => pageMeta[activeTab.value]?.title || '概览');
const currentPageDesc = computed(() => pageMeta[activeTab.value]?.desc || '');

// Categories
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
    showToast('数据刷新成功', 'success');
  } catch (e) {
    error.value = e.message;
    showToast('刷新失败: ' + e.message, 'error');
  } finally {
    loading.value = false;
  }
}

// Stats
const totalTransactionCount = computed(() => analytics.value?.recentTransactions?.length || 0);

const trackingDays = computed(() => {
  if (!analytics.value?.recentTransactions?.length) return 0;
  const dates = analytics.value.recentTransactions.map(t => new Date(t.date).getTime());
  if (dates.length === 0) return 0;
  const minDate = Math.min(...dates);
  const diffTime = Math.abs(new Date().getTime() - minDate);
  return Math.ceil(diffTime / (1000 * 60 * 60 * 24)) || 1;
});

// Init
onMounted(async () => {
  const saved = localStorage.getItem('neve-theme');
  if (saved && ['light', 'dark', 'geek', 'system'].includes(saved)) {
    themeMode.value = saved;
  }
  applyTheme();
  
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (themeMode.value === 'system') applyTheme();
  });

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

<style>
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Stats Card in Sidebar */
.stats-card {
  background: var(--bg-secondary);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  border: 1px solid var(--border);
  transition: all var(--transition-base);
}

.stats-card:hover {
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-sm);
}

.stats-icon-wrapper {
  width: 40px;
  height: 40px;
  background: var(--brand-light);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--brand-primary);
  flex-shrink: 0;
}

.stats-icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stats-icon svg {
  width: 100%;
  height: 100%;
  stroke-width: 2;
  stroke: currentColor;
}

.stats-content {
  flex: 1;
  min-width: 0;
}

.stats-label {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  margin-bottom: 2px;
}

.stats-value {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.2;
}

.stats-unit {
  font-size: var(--font-size-xs);
  font-weight: normal;
  color: var(--text-secondary);
}

.stats-subtitle {
  font-size: 10px;
  color: var(--text-secondary);
  margin-top: 2px;
}
</style>
