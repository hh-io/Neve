<template>
  <aside class="sidebar">
    <!-- Logo -->
    <div class="logo-section animate-fade-in-up">
      <div class="logo-icon-new">
        <MountainSnow :size="20" />
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
          @click="$emit('update:activeTab', item.id)"
        >
          <div class="nav-icon">
            <component :is="item.icon" />
          </div>
          <span>{{ item.label }}</span>
        </button>
      </div>

      <div class="nav-divider"></div>

      <button
        class="nav-item"
        :class="{ active: activeTab === 'budget' }"
        @click="$emit('update:activeTab', 'budget')"
      >
        <div class="nav-icon">
          <PiggyBank />
        </div>
        <span>预算</span>
      </button>
    </nav>

    <!-- Stats Section -->
    <div v-if="showStats" class="user-section animate-fade-in-up delay-200">
      <div class="stats-card">
        <div class="stats-icon-wrapper">
          <Trophy :size="20" />
        </div>
        <div class="stats-content">
          <div class="stats-label">已记录交易</div>
          <div class="stats-value">{{ transactionCount }} <span class="stats-unit">笔</span></div>
          <div class="stats-subtitle">坚持记账 {{ trackingDays }} 天</div>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { MountainSnow, PiggyBank, Trophy } from '@lucide/vue';
import { navItems } from '../../composables/navItems';

defineProps<{
  activeTab: string;
  showStats?: boolean;
  transactionCount?: number;
  trackingDays?: number;
}>();

defineEmits<{ 'update:activeTab': [id: string] }>();
</script>

<style scoped>
/* Stats Card in Sidebar(自 App.vue 迁入,归位到侧边栏组件) */
.stats-card {
  background: var(--surface-1);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  align-items: center;
  gap: var(--space-3);
  border: 1px solid var(--hairline);
  transition: border-color var(--transition-base);
}

.stats-card:hover {
  border-color: var(--accent);
}

.stats-icon-wrapper {
  width: 40px;
  height: 40px;
  background: var(--accent-subtle);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--accent);
  flex-shrink: 0;
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
  font-family: var(--font-numeric);
  font-variant-numeric: tabular-nums;
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
