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

            <!-- Trend Chart with Period Selector -->
            <div class="glass-card chart-card">
              <div class="card-header">
                <h3 class="card-title">收支趋势</h3>
                <div class="period-selector">
                  <button 
                    class="period-btn" 
                    :class="{ active: trendPeriod === 'day' }"
                    @click="trendPeriod = 'day'"
                  >日</button>
                  <button 
                    class="period-btn" 
                    :class="{ active: trendPeriod === 'week' }"
                    @click="trendPeriod = 'week'"
                  >周</button>
                  <button 
                    class="period-btn" 
                    :class="{ active: trendPeriod === 'month' }"
                    @click="trendPeriod = 'month'"
                  >月</button>
                </div>
              </div>
              <div class="chart-container">
                <v-chart :option="trendChartOption" autoresize />
              </div>
            </div>
          </div>
        </section>

        <!-- Account Balances & Monthly Comparison -->
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

            <!-- Monthly Comparison Bar Chart -->
            <div class="glass-card chart-card">
              <div class="card-header">
                <h3 class="card-title">月度对比</h3>
                <div class="period-selector">
                  <button 
                    class="period-btn" 
                    :class="{ active: comparisonMonths === 3 }"
                    @click="comparisonMonths = 3"
                  >3个月</button>
                  <button 
                    class="period-btn" 
                    :class="{ active: comparisonMonths === 6 }"
                    @click="comparisonMonths = 6"
                  >6个月</button>
                  <button 
                    class="period-btn" 
                    :class="{ active: comparisonMonths === 12 }"
                    @click="comparisonMonths = 12"
                  >12个月</button>
                </div>
              </div>
              <div class="chart-container">
                <v-chart :option="comparisonChartOption" autoresize />
              </div>
            </div>
          </div>
        </section>

        <!-- New Analytics Row 1: Income Chart & Daily Stats -->
        <section class="analytics-section fade-in" style="animation-delay: 0.25s">
          <div class="grid grid-2">
            <IncomeChart :data="analytics.incomeBreakdown" />
            <div class="glass-card stat-card daily-avg-card">
              <span class="label">日均消费</span>
              <span class="value expense">¥{{ analytics.dailyAverage?.toFixed(2) || '0.00' }}</span>
              <span class="trend">本月已消费 {{ new Date().getDate() }} 天</span>
            </div>
          </div>
        </section>

        <!-- New Analytics Row 2: Platform & Merchant Rankings -->
        <section class="analytics-section fade-in" style="animation-delay: 0.3s">
          <div class="grid grid-2">
            <PlatformRanking :data="analytics.platformRanking" />
            <MerchantRanking :data="analytics.merchantRanking" />
          </div>
        </section>

        <!-- New Analytics Row 3: Weekday & Category Trends -->
        <section class="analytics-section fade-in" style="animation-delay: 0.35s">
          <div class="grid grid-2">
            <WeekdayChart :data="analytics.weekdayDistribution" />
            <CategoryTrendChart :data="analytics.categoryTrends" />
          </div>
        </section>

        <!-- Liability Overview -->
        <section class="analytics-section fade-in" style="animation-delay: 0.4s">
          <div class="grid grid-2">
            <LiabilityOverview :data="analytics.liabilityBreakdown" />
            <div class="glass-card">
              <h3 class="card-title">财务健康</h3>
              <div class="health-stats">
                <div class="health-item">
                  <span class="health-label">资产负债比</span>
                  <span class="health-value" :class="debtRatio < 0.5 ? 'good' : 'warning'">
                    {{ (debtRatio * 100).toFixed(1) }}%
                  </span>
                </div>
                <div class="health-item">
                  <span class="health-label">月结余率</span>
                  <span class="health-value" :class="savingsRate > 0.2 ? 'good' : savingsRate > 0 ? 'ok' : 'warning'">
                    {{ (savingsRate * 100).toFixed(1) }}%
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <!-- Transaction Details (Full Width) -->
        <section class="transactions-section fade-in" style="animation-delay: 0.3s">
          <div class="glass-card">
            <div class="transactions-header">
              <h3 class="card-title">交易明细</h3>
              <form class="transactions-filters" @submit.prevent>
                <div class="search-box">
                  <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="11" cy="11" r="8"></circle>
                    <path d="m21 21-4.35-4.35"></path>
                  </svg>
                  <input 
                    type="text" 
                    v-model.lazy="searchQuery" 
                    placeholder="搜索交易..."
                    class="search-input"
                    @keydown.enter.prevent
                  />
                </div>
                <select v-model="categoryFilter" class="category-select">
                  <option value="">全部分类</option>
                  <option v-for="cat in expenseCategories" :key="cat" :value="cat">
                    {{ cat }}
                  </option>
                </select>
                <select v-model="tagFilter" class="category-select">
                  <option value="">全部标签</option>
                  <option v-for="tag in availableTags" :key="tag" :value="tag">
                    #{{ tag }}
                  </option>
                </select>
                <div class="date-range">
                  <input 
                    type="date" 
                    v-model="dateStart" 
                    class="date-input"
                    :max="dateEnd || undefined"
                  />
                  <span class="date-separator">至</span>
                  <input 
                    type="date" 
                    v-model="dateEnd" 
                    class="date-input"
                    :min="dateStart || undefined"
                  />
                </div>
              </form>
            </div>
            
            <div class="transactions-table" v-if="paginatedTransactions.length">
              <div class="table-header">
                <span class="col-date">日期</span>
                <span class="col-desc">描述</span>
                <span class="col-category">分类</span>
                <span class="col-tags">标签</span>
                <span class="col-amount">金额</span>
              </div>
              <div 
                v-for="tx in paginatedTransactions" 
                :key="tx.date + tx.narration + (tx.postings?.[0]?.amount || 0)"
                class="table-row"
              >
                <span class="col-date">{{ formatDate(tx.date) }}</span>
                <span class="col-desc">
                  <span class="tx-payee" v-if="tx.payee">{{ tx.payee }}</span>
                  <span class="tx-narration">{{ tx.narration || '未命名' }}</span>
                </span>
                <span class="col-category">{{ getTransactionCategory(tx) }}</span>
                <span class="col-tags">
                  <span 
                    v-for="tag in tx.tags" 
                    :key="tag" 
                    class="tx-tag"
                    :class="getTagClass(tag)"
                  >#{{ tag }}</span>
                </span>
                <span class="col-amount" :class="getTransactionAmountClass(tx)">
                  {{ formatTransactionAmount(tx) }}
                </span>
              </div>
            </div>
            <div v-else class="empty-state">
              <p>没有找到匹配的交易记录</p>
            </div>

            <!-- Pagination -->
            <div class="pagination" v-if="totalPages > 1">
              <button 
                class="page-btn" 
                :disabled="currentPage === 1"
                @click="currentPage--"
              >上一页</button>
              <span class="page-info">{{ currentPage }} / {{ totalPages }}</span>
              <button 
                class="page-btn" 
                :disabled="currentPage === totalPages"
                @click="currentPage++"
              >下一页</button>
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
import { ref, computed, onMounted, watch } from "vue";
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

// Import new components
import PlatformRanking from "./components/PlatformRanking.vue";
import MerchantRanking from "./components/MerchantRanking.vue";
import WeekdayChart from "./components/WeekdayChart.vue";
import CategoryTrendChart from "./components/CategoryTrendChart.vue";
import LiabilityOverview from "./components/LiabilityOverview.vue";
import IncomeChart from "./components/IncomeChart.vue";
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

// Trend chart period selector
const trendPeriod = ref('day');

// Monthly comparison period
const comparisonMonths = ref(3);

// Transaction filters
const searchQuery = ref('');
const categoryFilter = ref('');
const tagFilter = ref('');
const dateStart = ref('');
const dateEnd = ref('');
const currentPage = ref(1);
const pageSize = 10;

// Reset page when filters change
watch([searchQuery, categoryFilter, tagFilter, dateStart, dateEnd], () => {
  currentPage.value = 1;
});

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
  return `${date.getMonth() + 1}/${date.getDate()}`;
}

function formatDateTime(dateStr) {
  const date = new Date(dateStr);
  return date.toLocaleString("zh-CN");
}

function formatAccountName(account) {
  const parts = account.split(":");
  return parts.slice(-2).join(" · ");
}

function getTransactionCategory(tx) {
  const expensePosting = tx.postings?.find((p) =>
    p.account?.startsWith("Expenses")
  );
  if (expensePosting) {
    const parts = expensePosting.account.split(":");
    return parts.length >= 2 ? parts[1] : "Other";
  }
  const incomePosting = tx.postings?.find((p) =>
    p.account?.startsWith("Income")
  );
  if (incomePosting) {
    const parts = incomePosting.account.split(":");
    return parts.length >= 2 ? parts[1] : "收入";
  }
  return "-";
}

function getTransactionAmountClass(tx) {
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

function getTagClass(tag) {
  const tagLower = tag.toLowerCase();
  const tagColors = {
    meituan: 'tag-meituan',
    jd: 'tag-jd',
    taobao: 'tag-taobao',
    eleme: 'tag-eleme',
    pdd: 'tag-pdd',
    didi: 'tag-didi',
    douyin: 'tag-douyin',
    xianyu: 'tag-xianyu',
    offline: 'tag-offline',
    subscription: 'tag-subscription',
    unknown: 'tag-unknown',
  };
  return tagColors[tagLower] || 'tag-default';
}

// Extract expense categories from transactions
const expenseCategories = computed(() => {
  if (!analytics.value?.recentTransactions) return [];
  const cats = new Set();
  analytics.value.recentTransactions.forEach(tx => {
    const cat = getTransactionCategory(tx);
    if (cat && cat !== '-') cats.add(cat);
  });
  return Array.from(cats).sort();
});

// Extract available tags from transactions
const availableTags = computed(() => {
  if (!analytics.value?.recentTransactions) return [];
  const tags = new Set();
  analytics.value.recentTransactions.forEach(tx => {
    tx.tags?.forEach(tag => tags.add(tag));
  });
  return Array.from(tags).sort();
});

// Filtered and paginated transactions
const filteredTransactions = computed(() => {
  if (!analytics.value?.recentTransactions) return [];
  
  return analytics.value.recentTransactions.filter(tx => {
    // Search filter
    if (searchQuery.value) {
      const q = searchQuery.value.toLowerCase();
      const matchNarration = tx.narration?.toLowerCase().includes(q);
      const matchPayee = tx.payee?.toLowerCase().includes(q);
      const matchTags = tx.tags?.some(t => t.toLowerCase().includes(q));
      if (!matchNarration && !matchPayee && !matchTags) return false;
    }
    
    // Category filter
    if (categoryFilter.value) {
      const txCat = getTransactionCategory(tx);
      if (txCat !== categoryFilter.value) return false;
    }
    
    // Tag filter
    if (tagFilter.value) {
      if (!tx.tags?.includes(tagFilter.value)) return false;
    }
    
    // Date range filter
    if (dateStart.value || dateEnd.value) {
      const txDate = new Date(tx.date).toISOString().slice(0, 10);
      if (dateStart.value && txDate < dateStart.value) return false;
      if (dateEnd.value && txDate > dateEnd.value) return false;
    }
    
    return true;
  });
});

const totalPages = computed(() => 
  Math.ceil(filteredTransactions.value.length / pageSize)
);

const paginatedTransactions = computed(() => {
  const start = (currentPage.value - 1) * pageSize;
  return filteredTransactions.value.slice(start, start + pageSize);
});

// Monthly comparison data
const monthlyComparison = computed(() => {
  const trend = analytics.value?.monthlyTrend || [];
  const current = trend[trend.length - 1] || { income: 0, expense: 0, balance: 0 };
  const previous = trend[trend.length - 2] || { income: 0, expense: 0, balance: 0 };
  
  const incomeChange = current.income - previous.income;
  const expenseChange = current.expense - previous.expense;
  const balanceChange = (current.income - current.expense) - (previous.income - previous.expense);
  
  return {
    currentIncome: current.income,
    previousIncome: previous.income,
    incomeChange,
    incomeChangePercent: previous.income ? (incomeChange / previous.income) * 100 : 0,
    
    currentExpense: current.expense,
    previousExpense: previous.expense,
    expenseChange,
    expenseChangePercent: previous.expense ? (expenseChange / previous.expense) * 100 : 0,
    
    currentBalance: current.income - current.expense,
    previousBalance: previous.income - previous.expense,
    balanceChange,
  };
});

// Financial health metrics
const debtRatio = computed(() => {
  const assets = analytics.value?.summary?.totalAssets || 1;
  const liabilities = analytics.value?.summary?.totalLiabilities || 0;
  return liabilities / assets;
});

const savingsRate = computed(() => {
  const income = analytics.value?.summary?.monthIncome || 1;
  const expense = analytics.value?.summary?.monthExpense || 0;
  return (income - expense) / income;
});

// Chart options
const expenseChartOption = computed(() => {
  if (!analytics.value?.expenseByCategory?.length) {
    return { title: { text: "暂无数据", left: "center", top: "center" } };
  }

  const colors = [
    "#007AFF", "#5856D6", "#FF9F0A", "#FF453A", "#30D158",
    "#BF5AF2", "#FF2D55", "#64D2FF", "#FFCC00", "#00C7BE",
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
      textStyle: { color: "#6E6E73", fontSize: 12 },
    },
    color: colors,
    series: [
      {
        type: "pie",
        radius: ["45%", "70%"],
        center: ["35%", "50%"],
        avoidLabelOverlap: true,
        itemStyle: {
          borderRadius: 10,
          borderColor: "rgba(255,255,255,0.8)",
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

// Monthly comparison bar chart - show last N months
const comparisonChartOption = computed(() => {
  const trend = analytics.value?.monthlyTrend || [];
  const monthsToShow = Math.min(comparisonMonths.value, trend.length);
  const recentMonths = trend.slice(-monthsToShow);
  
  if (recentMonths.length === 0) {
    return { title: { text: "暂无数据", left: "center", top: "center" } };
  }
  
  const labels = recentMonths.map(m => m.month.slice(5) + "月");
  const incomeData = recentMonths.map(m => m.income);
  const expenseData = recentMonths.map(m => m.expense);
  const balanceData = recentMonths.map(m => m.income - m.expense);
  
  // Helper to get borderRadius based on value
  const getRadius = (value) => value >= 0 ? [4, 4, 0, 0] : [0, 0, 4, 4];
  
  return {
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "shadow" },
      formatter: (params) => {
        const month = params[0].axisValue;
        let html = `<strong>${month}</strong><br/>`;
        params.forEach(p => {
          // For 结余, use dynamic color based on value
          let marker = p.marker;
          if (p.seriesName === '结余') {
            const color = p.value >= 0 ? '#007AFF' : '#AF52DE';
            marker = `<span style="display:inline-block;margin-right:4px;border-radius:10px;width:10px;height:10px;background-color:${color};"></span>`;
          }
          html += `${marker} ${p.seriesName}: ¥${p.value.toFixed(2)}<br/>`;
        });
        return html;
      }
    },
    legend: {
      data: ["收入", "支出", "结余"],
      top: 0,
      textStyle: { color: "#6E6E73", fontSize: 12 },
    },
    grid: {
      left: "3%",
      right: "4%",
      bottom: "3%",
      top: "15%",
      containLabel: true,
    },
    xAxis: {
      type: "category",
      data: labels,
      axisLine: { lineStyle: { color: "#E5E5EA" } },
      axisLabel: { color: "#6E6E73", fontSize: 11 },
    },
    yAxis: {
      type: "value",
      axisLine: { show: false },
      splitLine: { lineStyle: { color: "#F2F2F7" } },
      axisLabel: {
        color: "#6E6E73",
        formatter: (val) => (Math.abs(val) >= 1000 ? (val / 1000).toFixed(1) + "k" : val),
      },
    },
    series: [
      {
        name: "收入",
        type: "bar",
        data: incomeData.map(v => ({ value: v, itemStyle: { borderRadius: getRadius(v) } })),
        itemStyle: { color: "#30D158" },
        barWidth: "20%",
      },
      {
        name: "支出",
        type: "bar",
        data: expenseData.map(v => ({ value: v, itemStyle: { borderRadius: getRadius(v) } })),
        itemStyle: { color: "#FF453A" },
        barWidth: "20%",
      },
      {
        name: "结余",
        type: "bar",
        data: balanceData.map(v => ({ 
          value: v, 
          itemStyle: { 
            borderRadius: getRadius(v) 
          } 
        })),
        itemStyle: { 
          color: (params) => {
            return balanceData[params.dataIndex] >= 0 ? "#007AFF" : "#AF52DE";
          }
        },
        barWidth: "20%",
      },
    ],
  };
});

// Trend chart with period support
const trendChartOption = computed(() => {
  if (!analytics.value?.monthlyTrend?.length && !analytics.value?.recentTransactions?.length) {
    return { title: { text: "暂无数据", left: "center", top: "center" } };
  }

  let labels = [];
  let incomeData = [];
  let expenseData = [];

  if (trendPeriod.value === 'month') {
    // Monthly data from monthlyTrend
    const trend = analytics.value.monthlyTrend || [];
    labels = trend.map((m) => m.month.slice(5) + "月");
    incomeData = trend.map((m) => m.income);
    expenseData = trend.map((m) => m.expense);
  } else if (trendPeriod.value === 'week') {
    // Aggregate by week from transactions
    const weeklyData = aggregateByWeek(analytics.value.recentTransactions || []);
    labels = weeklyData.map(w => w.label);
    incomeData = weeklyData.map(w => w.income);
    expenseData = weeklyData.map(w => w.expense);
  } else {
    // Daily data from transactions
    const dailyData = aggregateByDay(analytics.value.recentTransactions || []);
    labels = dailyData.map(d => d.label);
    incomeData = dailyData.map(d => d.income);
    expenseData = dailyData.map(d => d.expense);
  }

  return {
    tooltip: {
      trigger: "axis",
      axisPointer: { type: "shadow" },
    },
    legend: {
      data: ["收入", "支出"],
      top: 0,
      textStyle: { color: "#6E6E73", fontSize: 12 },
    },
    grid: {
      left: "3%",
      right: "4%",
      bottom: "3%",
      top: "15%",
      containLabel: true,
    },
    xAxis: {
      type: "category",
      data: labels,
      axisLine: { lineStyle: { color: "#E5E5EA" } },
      axisLabel: { color: "#6E6E73", fontSize: 11 },
    },
    yAxis: {
      type: "value",
      axisLine: { show: false },
      splitLine: { lineStyle: { color: "#F2F2F7" } },
      axisLabel: {
        color: "#6E6E73",
        formatter: (val) => (val >= 1000 ? val / 1000 + "k" : val),
      },
    },
    series: [
      {
        name: "收入",
        type: "line",
        data: incomeData,
        smooth: true,
        symbol: "circle",
        symbolSize: 8,
        lineStyle: {
          width: 3,
          color: "#30D158",
        },
        itemStyle: {
          color: "#30D158",
          borderWidth: 2,
          borderColor: "#fff",
        },
        areaStyle: {
          color: {
            type: "linear",
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: "rgba(48, 209, 88, 0.3)" },
              { offset: 1, color: "rgba(48, 209, 88, 0.05)" }
            ]
          }
        }
      },
      {
        name: "支出",
        type: "line",
        data: expenseData,
        smooth: true,
        symbol: "circle",
        symbolSize: 8,
        lineStyle: {
          width: 3,
          color: "#FF453A",
        },
        itemStyle: {
          color: "#FF453A",
          borderWidth: 2,
          borderColor: "#fff",
        },
        areaStyle: {
          color: {
            type: "linear",
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [
              { offset: 0, color: "rgba(255, 69, 58, 0.3)" },
              { offset: 1, color: "rgba(255, 69, 58, 0.05)" }
            ]
          }
        }
      },
    ],
  };
});

// Aggregate transactions by day (last 14 days)
function aggregateByDay(transactions) {
  const days = {};
  const now = new Date();
  
  // Initialize last 14 days
  for (let i = 13; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i);
    const key = date.toISOString().slice(0, 10);
    days[key] = { income: 0, expense: 0, label: `${date.getMonth() + 1}/${date.getDate()}` };
  }
  
  transactions.forEach(tx => {
    const date = new Date(tx.date);
    const key = date.toISOString().slice(0, 10);
    if (!days[key]) return;
    
    tx.postings?.forEach(p => {
      if (p.account?.startsWith('Expenses') && p.amount > 0) {
        days[key].expense += p.amount;
      } else if (p.account?.startsWith('Income') && p.amount < 0) {
        days[key].income += -p.amount;
      }
    });
  });
  
  return Object.values(days);
}

// Aggregate transactions by week (last 8 weeks)
function aggregateByWeek(transactions) {
  const weeks = {};
  const now = new Date();
  
  // Initialize last 8 weeks
  for (let i = 7; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i * 7);
    const weekNum = getWeekNumber(date);
    const key = `${date.getFullYear()}-W${weekNum}`;
    weeks[key] = { income: 0, expense: 0, label: `第${weekNum}周` };
  }
  
  transactions.forEach(tx => {
    const date = new Date(tx.date);
    const weekNum = getWeekNumber(date);
    const key = `${date.getFullYear()}-W${weekNum}`;
    if (!weeks[key]) return;
    
    tx.postings?.forEach(p => {
      if (p.account?.startsWith('Expenses') && p.amount > 0) {
        weeks[key].expense += p.amount;
      } else if (p.account?.startsWith('Income') && p.amount < 0) {
        weeks[key].income += -p.amount;
      }
    });
  });
  
  return Object.values(weeks);
}

function getWeekNumber(date) {
  const d = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const dayNum = d.getUTCDay() || 7;
  d.setUTCDate(d.getUTCDate() + 4 - dayNum);
  const yearStart = new Date(Date.UTC(d.getUTCFullYear(), 0, 1));
  return Math.ceil(((d - yearStart) / 86400000 + 1) / 7);
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
  border-bottom: 1px solid rgba(255, 255, 255, 0.2);
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
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-4);
}

.card-title {
  font-size: var(--font-size-lg);
  margin-bottom: var(--space-5);
  color: var(--color-text-primary);
}

.card-header .card-title {
  margin-bottom: 0;
}

.chart-card {
  min-height: 320px;
}

.chart-container {
  height: 280px;
}

/* Period Selector */
.period-selector {
  display: flex;
  gap: var(--space-1);
  background: rgba(0, 0, 0, 0.04);
  padding: 3px;
  border-radius: var(--radius-sm);
}

.period-btn {
  padding: var(--space-2) var(--space-4);
  border: none;
  background: transparent;
  border-radius: calc(var(--radius-sm) - 2px);
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.period-btn:hover {
  color: var(--color-text-primary);
}

.period-btn.active {
  background: white;
  color: var(--color-text-primary);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
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

/* Monthly Comparison */
.comparison-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.comparison-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-4);
  background: rgba(0, 0, 0, 0.02);
  border-radius: var(--radius-md);
}

.comparison-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
  font-weight: 500;
}

.comparison-values {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
}

.comparison-values .current {
  font-size: var(--font-size-xl);
  font-weight: 700;
}

.comparison-values .change {
  font-size: var(--font-size-sm);
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 6px;
}

.comparison-values .change.positive {
  background: rgba(48, 209, 88, 0.15);
  color: var(--color-green);
}

.comparison-values .change.negative {
  background: rgba(255, 69, 58, 0.15);
  color: var(--color-red);
}

.comparison-item .previous {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

/* Transactions Section */
.transactions-section .glass-card {
  padding: var(--space-8);
}

.transactions-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
  gap: var(--space-4);
}

.transactions-header .card-title {
  margin-bottom: 0;
}

.transactions-filters {
  display: flex;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  width: 16px;
  height: 16px;
  color: var(--color-text-tertiary);
}

.search-input {
  padding: var(--space-3) var(--space-4) var(--space-3) 40px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  background: rgba(255, 255, 255, 0.8);
  min-width: 200px;
  transition: all var(--transition-fast);
}

.search-input:focus {
  outline: none;
  border-color: var(--color-blue);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

.category-select {
  padding: var(--space-3) var(--space-4);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  background: rgba(255, 255, 255, 0.8);
  min-width: 120px;
  cursor: pointer;
}

.category-select:focus {
  outline: none;
  border-color: var(--color-blue);
}

/* Date Range */
.date-range {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.date-input {
  padding: var(--space-3) var(--space-3);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  background: rgba(255, 255, 255, 0.8);
  font-family: var(--font-family);
  color: var(--color-text-primary);
  cursor: pointer;
}

.date-input:focus {
  outline: none;
  border-color: var(--color-blue);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
}

.date-separator {
  color: var(--color-text-tertiary);
  font-size: var(--font-size-sm);
}

/* Transactions Table */
.transactions-table {
  border-radius: var(--radius-md);
  overflow: hidden;
}

.table-header, .table-row {
  display: grid;
  grid-template-columns: 80px 1fr 100px 120px 100px;
  gap: var(--space-3);
  padding: var(--space-4);
  align-items: center;
}

.table-header {
  background: rgba(0, 0, 0, 0.03);
  font-size: var(--font-size-xs);
  color: var(--color-text-secondary);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.table-row {
  border-bottom: 1px solid rgba(0, 0, 0, 0.04);
  font-size: var(--font-size-sm);
  transition: background var(--transition-fast);
}

.table-row:last-child {
  border-bottom: none;
}

.table-row:hover {
  background: rgba(0, 0, 0, 0.02);
}

.col-date {
  color: var(--color-text-secondary);
}

.col-desc {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.tx-payee {
  font-size: var(--font-size-xs);
  color: var(--color-text-tertiary);
}

.tx-narration {
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-category {
  color: var(--color-text-secondary);
}

.col-tags {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.col-amount {
  font-weight: 600;
  text-align: right;
}

.col-amount.expense {
  color: var(--color-red);
}

.col-amount.income {
  color: var(--color-green);
}

/* Tags */
.tx-tag {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  font-weight: 500;
  background: rgba(0, 0, 0, 0.05);
  color: var(--color-text-secondary);
}

.tx-tag.tag-meituan {
  background: rgba(255, 190, 0, 0.15);
  color: #D4A300;
}

.tx-tag.tag-jd {
  background: rgba(230, 0, 18, 0.12);
  color: #C80010;
}

.tx-tag.tag-taobao {
  background: rgba(255, 85, 0, 0.12);
  color: #E04A00;
}

.tx-tag.tag-eleme {
  background: rgba(0, 150, 230, 0.12);
  color: #0086CC;
}

.tx-tag.tag-pdd {
  background: rgba(230, 0, 35, 0.12);
  color: #D60020;
}

.tx-tag.tag-didi {
  background: rgba(255, 140, 0, 0.12);
  color: #E07800;
}

.tx-tag.tag-douyin {
  background: rgba(0, 0, 0, 0.08);
  color: #1C1C1C;
}

.tx-tag.tag-xianyu {
  background: rgba(255, 200, 50, 0.15);
  color: #CC9900;
}

.tx-tag.tag-offline {
  background: rgba(100, 100, 100, 0.1);
  color: #666666;
}

.tx-tag.tag-subscription {
  background: rgba(90, 50, 200, 0.12);
  color: #5A32C8;
}

.tx-tag.tag-unknown {
  background: rgba(150, 150, 150, 0.15);
  color: #888888;
}

.tx-tag.tag-default {
  background: rgba(0, 122, 255, 0.1);
  color: #007AFF;
}

/* Pagination */
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: var(--space-4);
  margin-top: var(--space-6);
  padding-top: var(--space-6);
  border-top: 1px solid rgba(0, 0, 0, 0.04);
}

.page-btn {
  padding: var(--space-2) var(--space-4);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: var(--radius-sm);
  background: white;
  font-size: var(--font-size-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.page-btn:hover:not(:disabled) {
  background: var(--color-bg);
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-info {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
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

.net-worth-card:hover {
  background: var(--gradient-blue);
}

.net-worth-card::before {
  background: linear-gradient(135deg, rgba(255,255,255,0.3) 0%, rgba(255,255,255,0) 50%);
}

.net-worth-card .label,
.net-worth-card .trend {
  color: rgba(255, 255, 255, 0.8);
}

.net-worth-card .value {
  color: white;
}

/* Daily Average Card */
.daily-avg-card {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-height: 200px;
}

/* Health Stats */
.health-stats {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.health-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-4);
  background: rgba(0, 0, 0, 0.02);
  border-radius: var(--radius-md);
}

.health-label {
  font-size: var(--font-size-sm);
  color: var(--color-text-secondary);
}

.health-value {
  font-size: var(--font-size-xl);
  font-weight: 700;
}

.health-value.good {
  color: var(--color-green);
}

.health-value.ok {
  color: var(--color-orange);
}

.health-value.warning {
  color: var(--color-red);
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
@media (max-width: 1024px) {
  .table-header, .table-row {
    grid-template-columns: 60px 1fr 80px 100px;
  }
  .col-tags {
    display: none;
  }
}

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

  .transactions-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .transactions-filters {
    width: 100%;
  }

  .search-input {
    flex: 1;
  }

  .table-header, .table-row {
    grid-template-columns: 50px 1fr 80px;
  }

  .col-category {
    display: none;
  }
}
</style>
