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
            <span v-html="icons[item.icon]"></span>
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
          <span v-html="icons.budget"></span>
        </div>
        <span>预算</span>
      </button>
    </nav>

    <!-- Stats Section -->
    <div v-if="showStats" class="user-section animate-fade-in-up" style="animation-delay: 0.2s;">
      <div class="stats-card">
        <div class="stats-icon-wrapper">
          <span v-html="icons.trophy" class="stats-icon"></span>
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
import { icons, navItems } from '../../composables/icons';

defineProps({
  activeTab: { type: String, required: true },
  showStats: { type: Boolean, default: false },
  transactionCount: { type: Number, default: 0 },
  trackingDays: { type: Number, default: 0 }
});

defineEmits(['update:activeTab']);
</script>
