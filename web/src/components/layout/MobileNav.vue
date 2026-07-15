<template>
  <nav class="mobile-nav">
    <button
      v-for="item in navItems"
      :key="item.id"
      class="mobile-nav-item"
      :class="{ active: activeTab === item.id }"
      @click="$emit('update:activeTab', item.id)"
    >
      <div class="mobile-nav-icon">
        <component :is="item.icon" />
      </div>
      <span class="mobile-nav-label">{{ item.label }}</span>
    </button>
  </nav>
</template>

<script setup>
import { navItems } from '../../composables/navItems';

defineProps({
  activeTab: { type: String, required: true }
});

defineEmits(['update:activeTab']);
</script>
