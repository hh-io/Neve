<template>
  <div v-if="hasIssues" class="issues-banner" :class="hasErrors ? 'level-error' : 'level-warning'">
    <button class="issues-summary" @click="expanded = !expanded">
      <span class="issues-title">
        {{ hasErrors ? '⚠ 账本数据存在问题,以下记录未计入统计' : '账本数据有提醒' }}
      </span>
      <span class="issues-counts">
        <span v-if="errorCount > 0" class="count-error">{{ errorCount }} 个错误</span>
        <span v-if="warningCount > 0" class="count-warning">{{ warningCount }} 个提醒</span>
        <span class="expand-hint">{{ expanded ? '收起 ▲' : '详情 ▼' }}</span>
      </span>
    </button>

    <div v-if="expanded" class="issues-list">
      <div v-for="(issue, i) in allIssues" :key="i" class="issue-row">
        <span class="issue-severity" :class="issue.severity === 'error' ? 'count-error' : 'count-warning'">
          {{ issue.severity === 'error' ? '错误' : '提醒' }}
        </span>
        <span class="issue-location">{{ issue.file }}:{{ issue.line }}</span>
        <span class="issue-message">{{ issue.message }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { ParseIssue, BalanceCheck } from '../../types/api';

const props = withDefaults(defineProps<{
  issues?: ParseIssue[];
  balanceChecks?: BalanceCheck[];
}>(), {
  issues: () => [],
  balanceChecks: () => []
});

const expanded = ref(false);

// balance 断言失败已由后端同时生成 BALANCE_FAILED issue,这里只展示 issues 即可,
// balanceChecks 用于判断是否有失败断言(将来可扩展为对账视图)
const allIssues = computed(() => {
  return [...props.issues].sort((a, b) => {
    if (a.severity !== b.severity) return a.severity === 'error' ? -1 : 1;
    return a.file === b.file ? a.line - b.line : a.file.localeCompare(b.file);
  });
});

const errorCount = computed(() => props.issues.filter(i => i.severity === 'error').length);
const warningCount = computed(() => props.issues.filter(i => i.severity === 'warning').length);
const hasErrors = computed(() =>
  errorCount.value > 0 || props.balanceChecks.some(c => !c.ok)
);
const hasIssues = computed(() => props.issues.length > 0 || hasErrors.value);
</script>

<style scoped>
.issues-banner {
  border-radius: var(--radius-lg);
  border: 1px solid;
  margin-bottom: var(--space-6);
  overflow: hidden;
}

.level-error {
  border-color: var(--expense);
  background: var(--expense-light);
}

.level-warning {
  border-color: var(--warning);
  background: var(--warning-light);
}

.issues-summary {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: none;
  border: none;
  cursor: pointer;
  font-size: var(--font-size-sm);
}

.issues-title {
  font-weight: 600;
  color: var(--text-primary);
}

.issues-counts {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-size: var(--font-size-xs);
}

.count-error {
  color: var(--expense);
  font-weight: 600;
}

.count-warning {
  color: var(--warning);
  font-weight: 600;
}

.expand-hint {
  color: var(--text-tertiary);
}

.issues-list {
  padding: var(--space-2) var(--space-4) var(--space-3);
  border-top: 1px solid var(--hairline);
  max-height: 240px;
  overflow-y: auto;
}

.issue-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
  padding: var(--space-1) 0;
  font-size: var(--font-size-xs);
}

.issue-severity {
  flex-shrink: 0;
}

.issue-location {
  flex-shrink: 0;
  font-family: ui-monospace, monospace;
  color: var(--text-secondary);
}

.issue-message {
  color: var(--text-primary);
}
</style>
