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
              <RefreshCw :size="16" />
              <span>{{ loading ? '刷新中...' : '刷新数据' }}</span>
            </button>
          </div>
        </header>

        <!-- 解析错误/断言失败横幅:软失败的坏数据必须显眼,否则会误信统计数字 -->
        <IssuesBanner :issues="analytics.parseIssues || []" :balanceChecks="analytics.balanceChecks || []" />

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

        <TransactionsTab v-show="activeTab === 'transactions'" :transactions="analytics.transactions || []" />

        <!-- Footer -->
        <footer style="text-align: center; padding: var(--space-8); color: var(--text-tertiary); font-size: var(--font-size-sm); border-top: 1px solid var(--border); margin-top: var(--space-8);">
          <p>最后更新: {{ formatDateTime(analytics.summary?.lastUpdated) }}</p>
        </footer>
      </template>
    </main>

    <!-- Toast(自订阅 useToast 单例) -->
    <AppToast />

    <!-- Mobile Bottom Navigation (visible on mobile only via CSS) -->
    <MobileNav :activeTab="activeTab" @update:activeTab="activeTab = $event" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from "vue";
import { RefreshCw } from "@lucide/vue";

// Layout Components
import AppSidebar from "./components/layout/AppSidebar.vue";
import ThemeSwitcher from "./components/layout/ThemeSwitcher.vue";
import MobileNav from "./components/layout/MobileNav.vue";
import AppToast from "./components/common/AppToast.vue";
import IssuesBanner from "./components/common/IssuesBanner.vue";

// Tab Components
import OverviewTab from "./components/tabs/OverviewTab.vue";
import SpendingTab from "./components/tabs/SpendingTab.vue";
import TrendsTab from "./components/tabs/TrendsTab.vue";
import AccountsTab from "./components/tabs/AccountsTab.vue";
import TransactionsTab from "./components/tabs/TransactionsTab.vue";
import BudgetCard from "./components/BudgetCard.vue";

// Composables(数据/主题为模块级单例)
import { formatDateTime } from "./composables/useFormatters";
import { useAnalytics } from "./composables/useAnalytics";
import { useTheme } from "./composables/useTheme";

const { analytics, loading, error, load, refresh } = useAnalytics();
const { themeMode, themeClass } = useTheme();

const activeTab = ref('overview');

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

// Stats:记账口径由后端统一计算,不再基于交易列表推算
const totalTransactionCount = computed(() => analytics.value?.summary?.transactionCount || 0);
const trackingDays = computed(() => analytics.value?.summary?.trackingDays || 0);

onMounted(load);
</script>

<style>
@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
