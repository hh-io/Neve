<template>
  <div class="animate-fade-in-up ov">
    <!-- 4 统计大数卡:小图标 + 标签 / 大数 / chip + 环比 -->
    <div class="ov-stats">
      <div v-for="s in statCards" :key="s.key" class="card ov-stat">
        <div class="ov-stat-head">
          <component :is="s.icon" :size="16" class="ov-stat-ic" />
          <span class="ov-stat-label">{{ s.label }}</span>
          <span v-if="s.note" class="ov-stat-note">{{ s.note }}</span>
        </div>
        <div class="ov-stat-value tabular-nums" :style="{ color: s.valueColor }">{{ s.value }}</div>
        <div class="ov-stat-foot">
          <span v-if="s.delta" class="chip" :class="s.chipCls">
            <component :is="s.trendIcon" :size="12" />
            <span class="tabular-nums">{{ s.delta }}</span>
          </span>
          <span class="ov-stat-hint">{{ s.hint }}</span>
        </div>
        <!-- 底部补充区(四卡统一:发丝线 + 可选占比条 + 两列 label/value) -->
        <div class="ov-stat-extra">
          <div v-if="s.bar" class="ov-stat-bar">
            <div class="ov-bar-a" :style="{ width: s.bar.a }"></div>
            <div class="ov-bar-b" :style="{ width: s.bar.b }"></div>
          </div>
          <div class="ov-supp">
            <div v-for="it in s.supp" :key="it.label" class="ov-supp-item">
              <span class="ov-supp-label">{{ it.label }}</span>
              <span class="ov-supp-value tabular-nums" :class="{ 'ov-supp-sm': it.small }" :style="{ color: it.color }">{{ it.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 日均支出 + 资产负债率 + 储蓄率 -->
    <div class="ov-row2">
      <div class="card ov-daily">
        <div class="ov-daily-head">
          <span class="ov-mini-label">日均支出</span>
          <span class="ov-mini-sub">本月已过 {{ dayOfMonth }} / {{ daysInMonth }} 天</span>
        </div>
        <div class="ov-daily-value tabular-nums">{{ formatMoney(dailyAverage) }}</div>
        <div class="ov-daily-foot">
          <div class="progress-bar ov-daily-bar">
            <div class="progress-fill" :style="{ width: `${monthProgress}%`, background: 'var(--accent)' }"></div>
          </div>
          <div class="ov-daily-cap">
            <span>本月累计 <span class="tabular-nums ov-cap-strong">{{ formatMoney(monthlyExpense) }}</span></span>
            <span>预计月末 <span class="tabular-nums ov-cap-strong">{{ formatMoney(projectedExpense) }}</span></span>
          </div>
        </div>
      </div>

      <div class="card ov-health">
        <span class="ov-mini-label">
          资产负债率
          <span v-if="hasLongTerm" class="ov-mini-note">不含长期负债</span>
        </span>
        <div class="ov-health-value tabular-nums">{{ debtRatio.toFixed(1) }}%</div>
        <div class="progress-bar ov-health-bar">
          <div class="progress-fill" :style="{ width: `${Math.min(100, debtRatio)}%`, background: debtRatio > 50 ? 'var(--expense)' : 'var(--income)' }"></div>
        </div>
        <span class="ov-health-cap" :style="{ color: debtRatio > 50 ? 'var(--expense)' : 'var(--income)' }">{{ debtRatioCaption }}</span>
      </div>

      <div class="card ov-health">
        <span class="ov-mini-label">月储蓄率</span>
        <div class="ov-health-value tabular-nums">{{ savingsRate }}%</div>
        <div class="progress-bar ov-health-bar">
          <div class="progress-fill" :style="{ width: `${Math.min(100, Math.max(0, savingsRate))}%`, background: savingsRate >= 20 ? 'var(--accent)' : 'var(--warning)' }"></div>
        </div>
        <span class="ov-health-cap">{{ savingsCaption }}</span>
      </div>

      <!-- 待还速览:即期现金压力是每天要看的量,不该只藏在待还管理页。
           口径与待还页看板同源(due30 / nextDue),逾期时才露出 monthRemaining -->
      <div class="card ov-health" :class="{ 'ov-health--alert': debtsOverdue }">
        <span class="ov-mini-label">
          未来 30 天待还
          <span class="ov-mini-note">账单与分期</span>
        </span>
        <div class="ov-health-value tabular-nums">{{ formatMoney(due30) }}</div>
        <span v-if="!debtsReport" class="ov-health-cap">待还数据不可用</span>
        <span v-else-if="debtsOverdue" class="ov-health-cap text-expense">
          逾期未还 {{ formatMoney(debtsSummary?.monthRemaining ?? 0) }}
        </span>
        <span v-else-if="debtsSummary?.nextDueDate" class="ov-health-cap">
          {{ debtsSummary.nextDueName }} · {{ nextDueText }}
        </span>
        <span v-else class="ov-health-cap text-income">本期已结清</span>
      </div>
    </div>

    <!-- 消费日历热力图 -->
    <section class="card ov-panel">
      <div class="ov-panel-head">
        <h3>消费日历热力图</h3>
        <span class="ov-panel-sub">每格 = 当日支出强度 · {{ currentYear }} 年</span>
      </div>
      <div class="ov-heat-body">
        <v-chart v-if="heatmapOption" class="ov-heat-chart" :option="heatmapOption" autoresize />
        <div v-else class="chart-empty">暂无足够数据生成热力图</div>
      </div>
    </section>

    <!-- 支出分类榜 + 最近交易 -->
    <div class="ov-row3">
      <section class="card ov-panel">
        <div class="ov-panel-head">
          <h3>支出分类</h3>
          <span class="ov-panel-sub">本月 · 环比上月</span>
        </div>
        <ExpenseCategoryBoard :data="expenseByCategory" />
      </section>

      <section class="card ov-panel">
        <div class="ov-panel-head">
          <h3>最近交易</h3>
          <span class="ov-panel-sub">最近 {{ recentTransactions.length }} / 共 {{ transactions.length }} 条</span>
        </div>
        <TransactionList
          :transactions="recentTransactions"
          max-height="360px"
          :show-account="false"
        />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { use } from 'echarts/core';
import { HeatmapChart } from 'echarts/charts';
import { TooltipComponent, CalendarComponent, VisualMapComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import VChart from 'vue-echarts';
import {
  Landmark, ArrowDownToLine, ArrowUpFromLine, PiggyBank,
  ArrowUpRight, ArrowDownRight,
} from '@lucide/vue';
import { formatMoney } from '../../composables/useFormatters';
import { getThemeColor, themeVersion } from '../../composables/useThemeColor';
import { useAnalytics } from '../../composables/useAnalytics';
import { useDebts } from '../../composables/useDebts';
import TransactionList from '../TransactionList.vue';
import ExpenseCategoryBoard from '../ExpenseCategoryBoard.vue';

use([HeatmapChart, TooltipComponent, CalendarComponent, VisualMapComponent, CanvasRenderer]);

const { analytics } = useAnalytics();
// loadDebts 是幂等单例(loaded 标志),与待还页各自 onMounted 互不重复请求
const { report: debtsReport, loadDebts } = useDebts();
onMounted(loadDebts);

const currentYear = new Date().getFullYear();

const summary = computed(() => analytics.value?.summary);
// 净资产按「不含长期负债」口径展示:房贷这类负债对应的资产(房产)不在账本内,
// 单边扣减会得到一个几十年不变的巨额负数,掩盖近期真实财务状况。
// 长期负债账户清单在待还管理页配置,分层由后端算好(见 summary.longTermLiabilities)。
const netWorth = computed(() => summary.value?.netWorthExLongTerm || 0);
const totalLiabilities = computed(() => Math.abs(summary.value?.shortTermLiabilities || 0));
const longTermLiabilities = computed(() => Math.abs(summary.value?.longTermLiabilities || 0));
const hasLongTerm = computed(() => longTermLiabilities.value > 0);
const monthlyIncome = computed(() => summary.value?.monthIncome || 0);
const monthlyExpense = computed(() => Math.abs(summary.value?.monthExpense || 0));
const monthlySavings = computed(() => monthlyIncome.value - monthlyExpense.value);
const transactions = computed(() => analytics.value?.transactions || []);
// 面板只有 360px 高,把全量交易塞进 DOM 靠 overflow 裁掉,账本长了会拖慢概览首屏。
// 后端已按日期倒序(同日按文件行序),取头部即最近的那些
const recentTransactions = computed(() => transactions.value.slice(0, 30));
const monthlyTrend = computed(() => analytics.value?.monthlyTrend || []);
const expenseByCategory = computed(() => analytics.value?.expenseByCategory || []);
const dailyAverage = computed(() => analytics.value?.dailyAverage || 0);
const dailyTrend = computed(() => analytics.value?.dailyTrend || []);

// 环比变化(基于月度趋势)
const incomeChange = computed(() => {
  const t = monthlyTrend.value;
  if (t.length < 2) return 0;
  const cur = t[t.length - 1]?.income || 0;
  const prev = t[t.length - 2]?.income || 0;
  if (prev === 0) return cur > 0 ? 100 : 0;
  return ((cur - prev) / prev) * 100;
});
const expenseChange = computed(() => {
  const t = monthlyTrend.value;
  if (t.length < 2) return 0;
  const cur = Math.abs(t[t.length - 1]?.expense || 0);
  const prev = Math.abs(t[t.length - 2]?.expense || 0);
  if (prev === 0) return cur > 0 ? 100 : 0;
  return ((cur - prev) / prev) * 100;
});
const savingsRate = computed(() => {
  if (monthlyIncome.value === 0) return 0;
  return Math.round((monthlySavings.value / monthlyIncome.value) * 100);
});
const debtRatio = computed(() => {
  const assets = summary.value?.totalAssets || 1;
  return (totalLiabilities.value / assets) * 100;
});

const daysInMonth = computed(() => {
  const now = new Date();
  return new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
});
const dayOfMonth = computed(() => new Date().getDate());
const monthProgress = computed(() => Math.round((dayOfMonth.value / daysInMonth.value) * 100));
const projectedExpense = computed(() => (monthlyExpense.value / dayOfMonth.value) * daysInMonth.value);

const debtRatioCaption = computed(() => {
  const r = debtRatio.value;
  if (r > 50) return '偏高 · 负债压力较大';
  if (r > 20) return '适中 · 负债可控';
  return '健康 · 负债占比很低';
});
const savingsCaption = computed(() => {
  const s = savingsRate.value;
  if (s >= 30) return '高于目标 30% · 状态良好';
  if (s >= 0) return '低于目标 30% · 可再收紧';
  return '入不敷出 · 需注意';
});

// 统计卡底部补充信息:资产/负债明细、本月收支笔数与笔均、日均结余
const totalAssets = computed(() => summary.value?.totalAssets || 0);
const monthPrefix = (() => {
  const n = new Date();
  return `${n.getFullYear()}-${String(n.getMonth() + 1).padStart(2, '0')}`;
})();
// 笔数/笔均按本月口径统计,与本月收入/支出金额对齐(交易日期前缀匹配当前年月)
const monthTransactions = computed(() => transactions.value.filter(t => (t.date || '').slice(0, 7) === monthPrefix));
const incomeCount = computed(() => monthTransactions.value.filter(t => t.kind === 'income' || t.kind === 'mixed').length);
const expenseCount = computed(() => monthTransactions.value.filter(t => t.kind === 'expense').length);
const incomeAvg = computed(() => incomeCount.value > 0 ? monthlyIncome.value / incomeCount.value : 0);
const expenseAvg = computed(() => expenseCount.value > 0 ? monthlyExpense.value / expenseCount.value : 0);
const dailySavings = computed(() => dayOfMonth.value > 0 ? monthlySavings.value / dayOfMonth.value : 0);
const projectedSavings = computed(() => dailySavings.value * daysInMonth.value);
// 资产/负债占比条:资产段宽度
const assetPct = computed(() => {
  const gross = totalAssets.value + totalLiabilities.value;
  return gross > 0 ? (totalAssets.value / gross) * 100 : 100;
});
// 月结余环比(结余 = 收入 - 支出,取月度趋势末两月)
const savingsChange = computed(() => {
  const t = monthlyTrend.value;
  if (t.length < 2) return 0;
  const cur = (t[t.length - 1]?.income || 0) - Math.abs(t[t.length - 1]?.expense || 0);
  const prev = (t[t.length - 2]?.income || 0) - Math.abs(t[t.length - 2]?.expense || 0);
  if (prev === 0) return cur > 0 ? 100 : 0;
  return ((cur - prev) / Math.abs(prev)) * 100;
});
// 带 +/- 号的金额(formatMoney 已带负号,正数补 +)
const signedMoney = (n: number) => (n >= 0 ? '+' : '') + formatMoney(n);

// 环比 chip:pos 表示数值方向,good 表示对财务是否有利
function pct(n: number): string {
  return `${n >= 0 ? '+' : ''}${n.toFixed(1)}%`;
}
type SuppItem = { label: string; value: string; color: string; small?: boolean };

const statCards = computed(() => {
  const incUp = incomeChange.value >= 0;
  const expUp = expenseChange.value >= 0; // 支出上升=不利
  const balUp = savingsChange.value >= 0;
  const balColor = monthlySavings.value < 0 ? 'var(--expense)' : 'var(--income)';
  // 长期负债作为补充区第三列出现,四卡补充区仍是单行,顶部一排卡片高度保持对齐;
  // 三列时该列字号收小一档,房贷这类六七位数金额才不会撑破卡片
  const netSupp: SuppItem[] = [
    { label: '资产', value: formatMoney(totalAssets.value), color: 'var(--income)' },
    { label: hasLongTerm.value ? '短期负债' : '负债', value: formatMoney(totalLiabilities.value), color: 'var(--expense)' },
  ];
  if (hasLongTerm.value) {
    netSupp.push({ label: '长期负债', value: formatMoney(longTermLiabilities.value), color: 'var(--text-tertiary)', small: true });
  }
  return [
    {
      key: 'net', label: '净资产', note: hasLongTerm.value ? '不含长期负债' : '', icon: Landmark,
      value: formatMoney(netWorth.value),
      valueColor: netWorth.value < 0 ? 'var(--expense)' : 'var(--text-primary)',
      // 不给环比 chip:这里原先吃的是 balanceChange(= 月度趋势 balance 的环比,也就是「月结余」
      // 的环比),把一个流量的变化率贴在时点存量上是错的口径——净资产卡和月结余卡会显示同一个
      // 百分比。要正确算需要「上月末净资产」快照,后端 Summary 目前没有这个字段;
      // 显示一个来源错误的数字比不显示更坏。
      hint: '时点余额',
      bar: { a: `${assetPct.value}%`, b: `${100 - assetPct.value}%` },
      supp: netSupp,
    },
    {
      key: 'income', label: '本月收入', note: '', icon: ArrowDownToLine,
      value: formatMoney(monthlyIncome.value), valueColor: 'var(--income)',
      delta: pct(incomeChange.value), trendIcon: incUp ? ArrowUpRight : ArrowDownRight,
      chipCls: incUp ? 'chip-income' : 'chip-expense', hint: '环比上月',
      supp: [
        { label: '收入笔数', value: `${incomeCount.value} 笔`, color: 'var(--text-primary)' },
        { label: '笔均', value: formatMoney(incomeAvg.value), color: 'var(--text-primary)' },
      ] as SuppItem[],
    },
    {
      key: 'expense', label: '本月支出', note: '', icon: ArrowUpFromLine,
      value: formatMoney(monthlyExpense.value), valueColor: 'var(--expense)',
      delta: pct(expenseChange.value), trendIcon: expUp ? ArrowUpRight : ArrowDownRight,
      chipCls: expUp ? 'chip-expense' : 'chip-income', hint: '环比上月',
      supp: [
        { label: '消费笔数', value: `${expenseCount.value} 笔`, color: 'var(--text-primary)' },
        { label: '笔均', value: formatMoney(expenseAvg.value), color: 'var(--text-primary)' },
      ] as SuppItem[],
    },
    {
      key: 'savings', label: '月结余', note: '', icon: PiggyBank,
      value: formatMoney(monthlySavings.value),
      valueColor: monthlySavings.value < 0 ? 'var(--expense)' : 'var(--text-primary)',
      delta: pct(savingsChange.value), trendIcon: balUp ? ArrowUpRight : ArrowDownRight,
      chipCls: balUp ? 'chip-income' : 'chip-expense', hint: '环比上月',
      supp: [
        { label: '日均结余', value: signedMoney(dailySavings.value), color: balColor },
        { label: '预计月末', value: signedMoney(projectedSavings.value), color: balColor },
      ] as SuppItem[],
    },
  ];
});

// 待还速览:数字全部取自后端已算好的报告,前端不做任何本地推算
const debtsSummary = computed(() => debtsReport.value?.summary);
const due30 = computed(() => debtsSummary.value?.due30 ?? 0);
const debtsOverdue = computed(() => (debtsSummary.value?.overdueCount ?? 0) > 0);
const nextDueText = computed(() => {
  const days = debtsSummary.value?.nextDueInDays ?? 0;
  if (days === 0) return '今天到期';
  return `${days} 天后到期`;
});

// 消费日历热力图(与趋势页同口径:顺序绿标度,与收入语义绿独立)
const heatmapOption = computed(() => {
  void themeVersion.value;
  // 净支出为负的日子(退款多于消费)当 0:顺序绿标度画不出负值,取绝对值会把
  // 「净退回 40」染成「花了 40」的深色。与资金流向图丢弃负净额分类同一取舍
  const heatmapData = dailyTrend.value.map(d => [d.date, Math.max(0, d.expense)] as [string, number]);
  if (heatmapData.length === 0) return null;
  const maxExpense = Math.max(...heatmapData.map(d => d[1])) || 1;
  return {
    tooltip: {
      formatter: (o: { data: [string, number] }) => `${o.data[0]}<br/>支出 ¥${Number(o.data[1]).toLocaleString('en-US')}`,
      backgroundColor: getThemeColor('--surface-1'),
      borderColor: getThemeColor('--hairline'),
      textStyle: { color: getThemeColor('--text-primary') }
    },
    visualMap: {
      min: 0,
      max: maxExpense,
      type: 'continuous',
      orient: 'horizontal',
      left: 'center',
      bottom: 0,
      itemWidth: 12,
      itemHeight: 120,
      text: ['高', '低'],
      textStyle: { color: getThemeColor('--text-tertiary'), fontSize: 11 },
      inRange: { color: ['--heat-0', '--heat-1', '--heat-2', '--heat-3', '--heat-4'].map(getThemeColor) },
      calculable: false,
      show: true
    },
    calendar: {
      top: 18,
      left: 24,
      right: 12,
      bottom: 44,
      cellSize: ['auto', 14],
      range: currentYear,
      splitLine: { show: false },
      itemStyle: {
        color: getThemeColor('--surface-1'),
        borderColor: getThemeColor('--canvas'),
        borderWidth: 3
      },
      yearLabel: { show: false },
      monthLabel: {
        color: getThemeColor('--text-tertiary'),
        fontSize: 11,
        nameMap: ['1月', '2月', '3月', '4月', '5月', '6月', '7月', '8月', '9月', '10月', '11月', '12月']
      },
      dayLabel: {
        color: getThemeColor('--text-tertiary'),
        fontSize: 10,
        firstDay: 1,
        nameMap: ['日', '一', '二', '三', '四', '五', '六']
      }
    },
    series: {
      type: 'heatmap',
      coordinateSystem: 'calendar',
      data: heatmapData,
      itemStyle: { borderRadius: 2 }
    }
  };
});
</script>

<style scoped>
.ov {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

/* ===== 统计大数卡 ===== */
.ov-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
}

.ov-stat {
  padding: var(--space-5);
}

.ov-stat-head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
}

.ov-stat-ic {
  color: var(--text-tertiary);
}

/* 口径注解:紧跟标签,弱化到不与大数抢视线 */
.ov-stat-note,
.ov-mini-note {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  font-weight: 400;
  white-space: nowrap;
}

.ov-stat-value {
  margin-top: var(--space-3);
  font-size: 1.75rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.ov-stat-foot {
  margin-top: var(--space-3);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.ov-stat-hint {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

/* 统计卡底部补充区(四卡统一:发丝线 + 可选占比条 + 两列 label/value) */
.ov-stat-extra {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--hairline);
}

.ov-stat-bar {
  display: flex;
  height: 4px;
  border-radius: var(--radius-full);
  overflow: hidden;
  background: var(--surface-3);
  margin-bottom: var(--space-3);
}

.ov-bar-a { background: var(--income); }
.ov-bar-b { background: var(--expense); }

.ov-supp {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
}

.ov-supp-item {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.ov-supp-label {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  white-space: nowrap;
}

.ov-supp-value {
  font-size: var(--font-size-sm);
  font-weight: 600;
  letter-spacing: -0.01em;
  white-space: nowrap;
}

.ov-supp-sm {
  font-size: var(--font-size-xs);
  font-weight: 500;
}

/* ===== chip ===== */
.chip {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: 650;
}

.chip-income { background: var(--income-light); color: var(--income); }
.chip-expense { background: var(--expense-light); color: var(--expense); }
.chip-warning { background: var(--warning-light); color: var(--warning); }

/* ===== 日均 + 健康 行 ===== */
.ov-row2 {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 1fr;
  gap: var(--space-4);
}

.ov-mini-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}

.ov-mini-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ov-daily {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
}

.ov-daily-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

.ov-daily-value {
  margin-top: var(--space-2);
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--expense);
}

.ov-daily-foot {
  margin-top: auto;
  padding-top: var(--space-4);
}

.ov-daily-bar { height: 8px; }

.ov-daily-cap {
  display: flex;
  justify-content: space-between;
  margin-top: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ov-cap-strong { color: var(--text-secondary); }

.ov-health {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.ov-health--alert {
  border-color: var(--expense);
}

.ov-health-value {
  font-size: 1.5rem;
  font-weight: 700;
}

.ov-health-bar { height: 6px; }

.ov-health-cap {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

/* ===== 环形图 + 最近交易 ===== */
.ov-row3 {
  display: grid;
  grid-template-columns: 1.1fr 1.25fr;
  gap: var(--space-4);
}

.ov-panel {
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.ov-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--hairline);
}

.ov-panel-head h3 {
  margin: 0;
  font-size: var(--font-size-base);
  font-weight: 620;
}

.ov-panel-sub {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.ov-heat-body {
  padding: var(--space-4) var(--space-5) var(--space-3);
}

.ov-heat-chart {
  width: 100%;
  height: 240px;
}

.chart-empty {
  height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

/* 四张小卡在中等宽度下并排会把「未来 30 天待还」的金额挤断行,先降到两列 */
@media (max-width: 1400px) {
  .ov-row2 { grid-template-columns: repeat(2, 1fr); }
}

@media (max-width: 1024px) {
  .ov-stats { grid-template-columns: repeat(2, 1fr); }
  .ov-row3 { grid-template-columns: 1fr; }
}

@media (max-width: 640px) {
  .ov-stats { grid-template-columns: 1fr; }
  .ov-row2 { grid-template-columns: 1fr; }
}
</style>
