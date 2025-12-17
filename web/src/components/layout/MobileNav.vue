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
        <span v-html="icons[item.icon]"></span>
      </div>
      <span class="mobile-nav-label">{{ item.label }}</span>
    </button>
  </nav>
</template>

<script setup>
import { icons, navItems } from '../../composables/icons';

defineProps({
  activeTab: { type: String, required: true }
});

defineEmits(['update:activeTab']);
</script>
