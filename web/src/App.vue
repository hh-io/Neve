<template>
  <div class="app-layout" :class="themeClass">
    <!-- Sidebar -->
    <aside class="sidebar">
      <!-- Logo -->
      <div class="logo-section animate-fade-in-up">
        <div class="logo-icon animate-pulse-soft">
          <span v-html="icons.layers"></span>
        </div>
        <div class="logo-text">
          <h1>Neve</h1>
          <p>智能记账系统</p>
        </div>
      </div>

      <!-- Navigation -->
      <nav class="nav-menu">
        <div class="nav-section">
          <button
            v-for="(item, index) in navItems"
            :key="item.id"
            class="nav-item animate-slide-in-left"
            :class="{ active: activeTab === item.id }"
            :style="{ animationDelay: `${index * 0.1}s` }"
            @click="activeTab = item.id"
          >
            <div class="nav-icon">
              <span v-html="icons[item.icon]"></span>
            </div>
            <span>{{ item.label }}</span>
          </button>
        </div>

        <div class="nav-divider"></div>

        <button class="nav-item" @click="activeTab = 'settings'">
          <div class="nav-icon">
            <span v-html="icons.settings"></span>
          </div>
          <span>设置</span>
        </button>
      </nav>

      <!-- User Section -->
      <div class="user-section">
        <div class="user-card">
          <div class="user-avatar">N</div>
          <div class="user-info">
            <div class="user-name">Neve 用户</div>
            <div class="user-email">neve@example.com</div>
          </div>
        </div>
      </div>
    </aside>

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
            <!-- Theme Switcher -->
            <div class="theme-switcher">
              <div class="theme-slider" :class="themeMode"></div>
              <button 
                class="theme-btn" 
                :class="{ active: themeMode === 'light' }"
                @click="setTheme('light')"
                title="亮色模式"
              >
                <span v-html="icons.sun"></span>
              </button>
              <button 
                class="theme-btn" 
                :class="{ active: themeMode === 'dark' }"
                @click="setTheme('dark')"
                title="暗色模式"
              >
                <span v-html="icons.moon"></span>
              </button>
              <button 
                class="theme-btn" 
                :class="{ active: themeMode === 'system' }"
                @click="setTheme('system')"
                title="跟随系统"
              >
                <span v-html="icons.monitor"></span>
              </button>
            </div>

            <!-- Refresh -->
            <button class="btn btn-secondary" @click="refresh" :disabled="loading">
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
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from "vue";

// Components
import OverviewTab from "./components/tabs/OverviewTab.vue";
import SpendingTab from "./components/tabs/SpendingTab.vue";
import TrendsTab from "./components/tabs/TrendsTab.vue";
import AccountsTab from "./components/tabs/AccountsTab.vue";
import TransactionsTab from "./components/tabs/TransactionsTab.vue";
import BudgetCard from "./components/BudgetCard.vue";

// Composables
import { formatDateTime } from "./composables/useFormatters";
import { icons, navItems } from "./composables/icons";

// State
const analytics = ref(null);
const loading = ref(false);
const error = ref(null);
const activeTab = ref('overview');

// Theme
const themeMode = ref('system'); // 'light' | 'dark' | 'system'

const themeClass = computed(() => {
  if (themeMode.value === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'theme-dark' : 'theme-light';
  }
  return `theme-${themeMode.value}`;
});

// Apply theme to document root
function applyTheme() {
  const html = document.documentElement;
  html.classList.remove('theme-light', 'theme-dark');
  html.classList.add(themeClass.value);
}

watch(themeClass, applyTheme, { immediate: true });

function setTheme(mode) {
  themeMode.value = mode;
  localStorage.setItem('neve-theme', mode);
}

// Initialize theme
onMounted(() => {
  const saved = localStorage.getItem('neve-theme');
  if (saved && ['light', 'dark', 'system'].includes(saved)) {
    themeMode.value = saved;
  }
  applyTheme();
  
  // Listen for system theme changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (themeMode.value === 'system') {
      applyTheme();
    }
  });
});

// Page info
const currentPageTitle = computed(() => {
  const titles = {
    overview: '概览',
    spending: '收支分析',
    trends: '趋势图表',
    accounts: '账户管理',
    budget: '预算管理',
    transactions: '交易明细',
    settings: '设置',
  };
  return titles[activeTab.value] || '概览';
});

const currentPageDesc = computed(() => {
  const descs = {
    overview: '欢迎回来，这是您的财务概况',
    spending: '查看收入与支出的详细分析',
    trends: '了解您的财务变化趋势',
    accounts: '管理您的所有账户',
    budget: '设置并跟踪您的预算目标',
    transactions: '查看所有交易记录',
    settings: '自定义您的偏好设置',
  };
  return descs[activeTab.value] || '';
});

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

<style>
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
