<template>
  <aside class="sidebar">
    <!-- Logo -->
    <div class="logo-section animate-fade-in-up">
      <div class="logo-icon-new">
        <span>N</span>
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
    <div v-if="showStats" class="user-section animate-fade-in-up" style="animation-delay: 0.2s;">
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

<script setup>
import { PiggyBank, Trophy } from '@lucide/vue';
import { navItems } from '../../composables/navItems';

defineProps({
  activeTab: { type: String, required: true },
  showStats: { type: Boolean, default: false },
  transactionCount: { type: Number, default: 0 },
  trackingDays: { type: Number, default: 0 }
});

defineEmits(['update:activeTab']);
</script>

<style scoped>
/* Stats Card in Sidebar(自 App.vue 迁入,归位到侧边栏组件) */
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
