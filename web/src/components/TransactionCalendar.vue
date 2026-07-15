<template>
  <div class="calendar-container">
    <!-- 月份导航 -->
    <div class="calendar-header">
      <button class="nav-btn" @click="prevMonth">
        <ChevronLeft />
      </button>
      <span class="month-title">{{ currentYear }}年{{ currentMonth }}月</span>
      <button class="nav-btn" @click="nextMonth">
        <ChevronRight />
      </button>
    </div>

    <!-- 星期表头 -->
    <div class="weekday-header">
      <span v-for="day in weekdays" :key="day" class="weekday-cell">{{ day }}</span>
    </div>

    <!-- 日期网格 -->
    <div class="calendar-grid">
      <div 
        v-for="(cell, index) in calendarCells" 
        :key="index"
        class="date-cell"
        :class="getCellClass(cell)"
      >
        <template v-if="cell.date">
          <span class="date-num">{{ cell.day }}</span>
          <span v-if="cell.expense" class="expense-amount">-{{ formatAmount(cell.expense) }}</span>
          <span v-if="cell.income" class="income-amount">+{{ formatAmount(cell.income) }}</span>
        </template>
        <template v-else>
          <span class="date-num empty">{{ cell.day }}</span>
        </template>
      </div>
    </div>

    <!-- 图例 -->
    <div class="legend">
      <div class="legend-item">
        <span class="legend-dot expense"></span>
        <span>支出</span>
      </div>
      <div class="legend-item">
        <span class="legend-dot income"></span>
        <span>收入</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { ChevronLeft, ChevronRight } from '@lucide/vue';

const props = defineProps({
  dailyData: { type: Array, default: () => [] }
});

const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

// 当前显示的年月
const now = new Date();
const currentYear = ref(now.getFullYear());
const currentMonth = ref(now.getMonth() + 1);

// 月份导航
function prevMonth() {
  if (currentMonth.value === 1) {
    currentMonth.value = 12;
    currentYear.value--;
  } else {
    currentMonth.value--;
  }
}

function nextMonth() {
  if (currentMonth.value === 12) {
    currentMonth.value = 1;
    currentYear.value++;
  } else {
    currentMonth.value++;
  }
}

// 将数据转换为日期索引的 map
const dailyMap = computed(() => {
  const map = {};
  props.dailyData.forEach(d => {
    if (d.date) {
      map[d.date] = {
        expense: Math.abs(d.expense || 0),
        income: d.income || 0
      };
    }
  });
  return map;
});

// 生成日历单元格
const calendarCells = computed(() => {
  const year = currentYear.value;
  const month = currentMonth.value;
  
  // 该月第一天是星期几
  const firstDay = new Date(year, month - 1, 1).getDay();
  // 该月有多少天
  const daysInMonth = new Date(year, month, 0).getDate();
  // 上个月有多少天
  const daysInPrevMonth = new Date(year, month - 1, 0).getDate();
  
  const cells = [];
  
  // 上个月的尾部日期（填充）
  for (let i = firstDay - 1; i >= 0; i--) {
    cells.push({
      day: daysInPrevMonth - i,
      date: null,
      isOtherMonth: true
    });
  }
  
  // 当月日期
  for (let d = 1; d <= daysInMonth; d++) {
    const dateStr = `${year}-${String(month).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
    const data = dailyMap.value[dateStr] || {};
    cells.push({
      day: d,
      date: dateStr,
      expense: data.expense || 0,
      income: data.income || 0,
      isToday: isToday(year, month, d),
      isOtherMonth: false
    });
  }
  
  // 下个月的开头日期（填充到 6 行，共 42 格）
  const remaining = 42 - cells.length;
  for (let i = 1; i <= remaining; i++) {
    cells.push({
      day: i,
      date: null,
      isOtherMonth: true
    });
  }
  
  return cells;
});

function isToday(year, month, day) {
  const today = new Date();
  return today.getFullYear() === year && 
         today.getMonth() + 1 === month && 
         today.getDate() === day;
}

function getCellClass(cell) {
  const classes = [];
  if (cell.isOtherMonth) classes.push('other-month');
  if (cell.isToday) classes.push('today');
  if (cell.expense > 0 && cell.income > 0) classes.push('mixed');
  else if (cell.expense > 0) classes.push('has-expense');
  else if (cell.income > 0) classes.push('has-income');
  return classes;
}

function formatAmount(amount) {
  if (amount >= 1000) {
    return (amount / 1000).toFixed(1) + 'k';
  }
  return Math.round(amount);
}
</script>

<style scoped>
.calendar-container {
  width: 100%;
}

.calendar-header {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  margin-bottom: var(--space-3);
}

.nav-btn {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  background: var(--brand-primary);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.nav-btn:hover {
  transform: scale(1.1);
  box-shadow: var(--shadow-sm);
}

.nav-btn :deep(svg) {
  width: 14px;
  height: 14px;
  stroke: currentColor;
}

.month-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
  min-width: 100px;
  text-align: center;
}

.weekday-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  text-align: center;
  padding: var(--space-1) 0;
  border-bottom: 1px solid var(--border);
  margin-bottom: var(--space-1);
}

.weekday-cell {
  font-size: var(--font-size-xs);
  font-weight: 500;
  color: var(--text-tertiary);
}

.calendar-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
}

.date-cell {
  height: 52px;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  gap: 1px;
  background: var(--bg-tertiary);
  transition: all 0.15s ease;
  overflow: hidden;
}

.date-cell:hover:not(.other-month) {
  box-shadow: var(--shadow-sm);
}

.date-cell.other-month {
  opacity: 0.25;
  background: transparent;
}

.date-cell.today {
  background: var(--brand-light);
  border: 1.5px solid var(--brand-primary);
}

.date-cell.has-expense {
  background: rgba(194, 123, 123, 0.12);
}

.date-cell.has-income {
  background: rgba(107, 155, 122, 0.12);
}

.date-cell.mixed {
  background: linear-gradient(135deg, rgba(107, 155, 122, 0.12) 0%, rgba(194, 123, 123, 0.12) 100%);
}

.date-num {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1;
}

.date-num.empty {
  font-weight: 400;
  color: var(--text-tertiary);
}

.expense-amount {
  font-size: 9px;
  font-weight: 500;
  color: var(--expense);
  line-height: 1;
}

.income-amount {
  font-size: 9px;
  font-weight: 500;
  color: var(--income);
  line-height: 1;
}

.legend {
  display: flex;
  justify-content: center;
  gap: var(--space-4);
  margin-top: var(--space-3);
  padding-top: var(--space-2);
  border-top: 1px solid var(--border);
}

.legend-item {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.legend-dot {
  width: 10px;
  height: 10px;
  border-radius: 2px;
}

.legend-dot.expense {
  background: rgba(194, 123, 123, 0.5);
}

.legend-dot.income {
  background: rgba(107, 155, 122, 0.5);
}

/* 响应式 */
@media (max-width: 640px) {
  .date-cell {
    height: 44px;
    padding: 2px;
  }
  
  .date-num {
    font-size: 10px;
  }
  
  .expense-amount,
  .income-amount {
    font-size: 8px;
  }
}
</style>
