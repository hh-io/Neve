<template>
  <div class="app">
    <!-- Header -->
    <header class="header">
      <div class="header-content">
        <div class="logo">
          <img src="/neve.svg" alt="Neve" class="logo-icon" />
          <span class="logo-text">Neve</span>
        </div>
        <button class="btn btn-secondary" @click="refresh" :disabled="loading">
          <svg
            class="icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path
              d="M23 4v6h-6M1 20v-6h6M20.49 9A9 9 0 1 0 5.64 5.64L1 10m22 4l-4.64 4.36A9 9 0 0 1 3.51 15"
            />
          </svg>
          <span>{{ loading ? "刷新中..." : "刷新数据" }}</span>
        </button>
      </div>
    </header>

    <!-- Main Content -->
    <main class="container">
      <!-- Error State -->
      <div v-if="error" class="error-card glass-card">
        <p>{{ error }}</p>
        <button class="btn btn-primary" @click="refresh">重试</button>
      </div>

      <!-- Loading State -->
      <div v-else-if="loading && !analytics" class="loading-state">
        <div class="loading-spinner"></div>
        <p>加载中...</p>
      </div>

      <!-- Dashboard -->
      <template v-else-if="analytics">
        <!-- Summary Cards -->
        <section class="summary-section fade-in">
          <div class="grid grid-4">
            <div class="glass-card stat-card net-worth-card">
              <span class="label">净资产</span>
              <span class="value">{{
                formatMoney(analytics.summary.netWorth)
              }}</span>
              <span
                class="trend positive"
                v-if="analytics.summary.netWorth > 0"
              >
                资产 {{ formatMoney(analytics.summary.totalAssets) }}
              </span>
            </div>
            <div class="glass-card stat-card">
              <span class="label">本月收入</span>
              <span class="value income">{{
                formatMoney(analytics.summary.monthIncome)
              }}</span>
            </div>
            <div class="glass-card stat-card">
              <span class="label">本月支出</span>
              <span class="value expense">{{
                formatMoney(analytics.summary.monthExpense)
              }}</span>
            </div>
            <div class="glass-card stat-card">
              <span class="label">本月结余</span>
              <span
                class="value"
                :class="
                  analytics.summary.monthBalance >= 0 ? 'income' : 'expense'
                "
              >
                {{ formatMoney(analytics.summary.monthBalance) }}
              </span>
            </div>
          </div>
        </section>

        <!-- Charts Section -->
        <section class="charts-section fade-in" style="animation-delay: 0.1s">
          <div class="grid grid-2">
            <!-- Expense Pie Chart -->
            <div class="glass-card chart-card">
              <h3 class="card-title">本月支出分类</h3>
              <div class="chart-container">
                <v-chart :option="expenseChartOption" autoresize />
              </div>
            </div>

            <!-- Monthly Trend -->
            <div class="glass-card chart-card">
              <h3 class="card-title">月度收支趋势</h3>
              <div class="chart-container">
                <v-chart :option="trendChartOption" autoresize />
              </div>
            </div>
          </div>
        </section>

        <!-- Account Balances & Recent Transactions -->
        <section class="details-section fade-in" style="animation-delay: 0.2s">
          <div class="grid grid-2">
            <!-- Account Balances -->
            <div class="glass-card">
              <h3 class="card-title">账户余额</h3>
              <div class="account-list">
                <div
                  v-for="account in analytics.accountBalances.slice(0, 10)"
                  :key="account.account"
                  class="account-item"
                >
                  <div class="account-info">
                    <span class="account-name">{{
                      formatAccountName(account.account)
                    }}</span>
                    <span class="account-type">{{ account.type }}</span>
                  </div>
                  <span
                    class="account-balance"
                    :class="account.balance >= 0 ? 'positive' : 'negative'"
                  >
                    {{ formatMoney(account.balance) }}
                  </span>
                </div>
              </div>
            </div>

            <!-- Recent Transactions -->
            <div class="glass-card">
              <h3 class="card-title">最近交易</h3>
              <div
                class="transaction-list"
                v-if="analytics.recentTransactions?.length"
              >
                <div
                  v-for="(tx, index) in analytics.recentTransactions.slice(
                    0,
                    8
                  )"
                  :key="index"
                  class="transaction-item"
                >
                  <div class="tx-left">
                    <span class="tx-narration">{{
                      tx.narration || tx.payee || "未命名"
                    }}</span>
                    <span class="tx-date">{{ formatDate(tx.date) }}</span>
                  </div>
                  <span
                    class="tx-amount"
                    :class="getTransactionAmountClass(tx)"
                  >
                    {{ formatTransactionAmount(tx) }}
                  </span>
                </div>
              </div>
              <div v-else class="empty-state">
                <p>暂无交易记录</p>
                <p class="hint">在 inbox.bean 中添加交易后刷新</p>
              </div>
            </div>
          </div>
        </section>

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
import VChart from "vue-echarts";
import { use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { PieChart, LineChart, BarChart } from "echarts/charts";
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
} from "echarts/components";

// Register ECharts components
use([
  CanvasRenderer,
  PieChart,
  LineChart,
  BarChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent,
]);

const analytics = ref(null);
const loading = ref(false);
const error = ref(null);

// Fetch analytics data
async function fetchAnalytics() {
  try {
    const res = await fetch("/api/analytics");
    if (!res.ok) throw new Error("Failed to fetch data");
    return await res.json();
  } catch (e) {
    throw new Error("无法连接到服务器");
  }
}

// Refresh data
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

// Format helpers
function formatMoney(amount) {
  if (amount == null) return "¥0.00";
  const sign = amount < 0 ? "-" : "";
  const absAmount = Math.abs(amount);
  return (
    sign +
    "¥" +
    absAmount.toLocaleString("zh-CN", {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })
  );
}

function formatDate(dateStr) {
  const date = new Date(dateStr);
  return `${date.getMonth() + 1}月${date.getDate()}日`;
}

function formatDateTime(dateStr) {
  const date = new Date(dateStr);
  return date.toLocaleString("zh-CN");
}

function formatAccountName(account) {
  const parts = account.split(":");
  return parts.slice(-2).join(" · ");
}

function getTransactionAmountClass(tx) {
  // Find expense posting
  const expensePosting = tx.postings?.find((p) =>
    p.account?.startsWith("Expenses")
  );
  if (expensePosting) return "expense";
  const incomePosting = tx.postings?.find((p) =>
    p.account?.startsWith("Income")
  );
  if (incomePosting) return "income";
  return "";
}

function formatTransactionAmount(tx) {
  // Get the main amount (expense or income)
  const expensePosting = tx.postings?.find((p) =>
    p.account?.startsWith("Expenses")
  );
  if (expensePosting) return "-" + formatMoney(expensePosting.amount).slice(1);

  const incomePosting = tx.postings?.find((p) =>
    p.account?.startsWith("Income")
  );
  if (incomePosting) return "+" + formatMoney(-incomePosting.amount).slice(1);

  return formatMoney(tx.postings?.[0]?.amount || 0);
}

// Chart options
const expenseChartOption = computed(() => {
  if (!analytics.value?.expenseByCategory?.length) {
    return { title: { text: "暂无数据", left: "center", top: "center" } };
  }

  const colors = [
    "#007AFF",
    "#5856D6",
    "#FF9500",
    "#FF3B30",
    "#34C759",
    "#AF52DE",
    "#FF2D55",
    "#5AC8FA",
    "#FFCC00",
    "#00C7BE",
  ];

  return {
    tooltip: {
      trigger: "item",
      formatter: "{b}: ¥{c} ({d}%)",
    },
    legend: {
      orient: "vertical",
      right: "5%",
      top: "center",
      textStyle: { color: "#86868B", fontSize: 12 },
    },
    color: colors,
    series: [
      {
        type: "pie",
        radius: ["45%", "70%"],
        center: ["35%", "50%"],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 8,
          borderColor: "#fff",
          borderWidth: 2,
        },
        label: { show: false },
        emphasis: {
          label: { show: true, fontSize: 14, fontWeight: "bold" },
        },
        data: analytics.value.expenseByCategory.slice(0, 8).map((item) => ({
          name: item.category,
          value: Math.round(item.amount * 100) / 100,
        })),
      },
    ],
  };
});

const trendChartOption = computed(() => {
  if (!analytics.value?.monthlyTrend?.length) {
    return { title: { text: "暂无数据", left: "center", top: "center" } };
  }

  const months = analytics.value.monthlyTrend.map(
    (m) => m.month.slice(5) + "月"
  );
  const income = analytics.value.monthlyTrend.map((m) => m.income);
  const expense = analytics.value.monthlyTrend.map((m) => m.expense);

  return {
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "shadow" },
    },
    legend: {
      data: ["收入", "支出"],
      top: 0,
      textStyle: { color: "#86868B", fontSize: 12 },
    },
    grid: {
      left: "3%",
      right: "4%",
      bottom: "3%",
      containLabel: true,
    },
    xAxis: {
      type: "category",
      data: months,
      axisLine: { lineStyle: { color: "#E5E5EA" } },
      axisLabel: { color: "#86868B" },
    },
    yAxis: {
      type: "value",
      axisLine: { show: false },
      splitLine: { lineStyle: { color: "#F2F2F7" } },
      axisLabel: {
        color: "#86868B",
        formatter: (val) => (val >= 1000 ? val / 1000 + "k" : val),
      },
    },
    series: [
      {
        name: "收入",
        type: "bar",
        data: income,
        itemStyle: {
          color: "#34C759",
          borderRadius: [4, 4, 0, 0],
        },
        barWidth: "30%",
      },
      {
        name: "支出",
        type: "bar",
        data: expense,
        itemStyle: {
          color: "#FF3B30",
          borderRadius: [4, 4, 0, 0],
        },
        barWidth: "30%",
      },
    ],
  };
});

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
.app {
  min-height: 100vh;
}

/* Header */
.header {
  position: sticky;
  top: 0;
  z-index: 100;
  background: var(--glass-bg);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
}

.header-content {
  max-width: 1400px;
  margin: 0 auto;
  padding: var(--space-4) var(--space-6);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logo {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.logo-icon {
  width: 36px;
  height: 36px;
}

.logo-text {
  font-size: var(--font-size-xl);
  font-weight: 700;
  letter-spacing: -0.02em;
}

.icon {
  width: 16px;
  height: 16px;
}

/* Sections */
section {
  margin-bottom: var(--space-8);
}

/* Cards */
.card-title {
  font-size: var(--font-size-lg);
  margin-bottom: var(--space-5);
  color: var(--color-text-primary);
}

.chart-card {
  min-height: 320px;
}

.chart-container {
  height: 280px;
}

/* Account List */
.account-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.account-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3) 0;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.account-item:last-child {
  border-bottom: none;
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.account-name {
  font-size: var(--font-size-sm);
  font-weight: 500;
}

.account-type {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.account-balance {
  font-weight: 600;
  font-size: var(--font-size-sm);
}

.account-balance.positive {
  color: var(--color-text-primary);
}

.account-balance.negative {
  color: var(--color-red);
}

/* Transaction List */
.transaction-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.transaction-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3) 0;
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
}

.transaction-item:last-child {
  border-bottom: none;
}

.tx-left {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.tx-narration {
  font-size: var(--font-size-sm);
  font-weight: 500;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tx-date {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.tx-amount {
  font-weight: 600;
  font-size: var(--font-size-sm);
}

.tx-amount.expense {
  color: var(--color-red);
}

.tx-amount.income {
  color: var(--color-green);
}

/* Stat colors */
.stat-card .value.income {
  color: var(--color-green);
}

.stat-card .value.expense {
  color: var(--color-red);
}

.net-worth-card {
  background: var(--gradient-blue);
  color: white;
}

.net-worth-card .label,
.net-worth-card .trend {
  color: rgba(255, 255, 255, 0.8);
}

.net-worth-card .value {
  color: white;
}

/* Loading */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  gap: var(--space-4);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(0, 122, 255, 0.2);
  border-top-color: var(--color-blue);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: var(--space-8) var(--space-4);
  color: var(--color-text-secondary);
}

.empty-state p {
  margin: 0;
}

.empty-state .hint {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
  margin-top: var(--space-2);
}

/* Error */
.error-card {
  text-align: center;
  padding: var(--space-10);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
}

/* Footer */
.footer {
  text-align: center;
  padding: var(--space-6) 0;
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

/* Responsive */
@media (max-width: 768px) {
  .header-content {
    padding: var(--space-3) var(--space-4);
  }

  .logo-icon {
    width: 28px;
    height: 28px;
  }

  .logo-text {
    font-size: var(--font-size-lg);
  }

  .chart-container {
    height: 240px;
  }
}
</style>
