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
      <span class="mobile-nav-label">{{ item.short }}</span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import { navItems } from '../../composables/navItems';

defineProps<{ activeTab: string }>();

defineEmits<{ 'update:activeTab': [id: string] }>();
</script>
