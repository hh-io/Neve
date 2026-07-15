<template>
  <div class="theme-switcher">
    <div class="theme-slider" :class="modelValue"></div>
    <button 
      class="theme-btn" 
      :class="{ active: modelValue === 'light' }"
      @click="$emit('update:modelValue', 'light')"
      title="亮色模式"
    >
      <Sun :size="16" />
    </button>
    <button
      class="theme-btn"
      :class="{ active: modelValue === 'dark' }"
      @click="$emit('update:modelValue', 'dark')"
      title="暗色模式"
    >
      <Moon :size="16" />
    </button>
    <button
      class="theme-btn"
      :class="{ active: modelValue === 'geek' }"
      @click="$emit('update:modelValue', 'geek')"
      title="极客模式"
    >
      <Terminal :size="16" />
    </button>
    <button
      class="theme-btn"
      :class="{ active: modelValue === 'system' }"
      @click="$emit('update:modelValue', 'system')"
      title="跟随系统"
    >
      <Monitor :size="16" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Sun, Moon, Terminal, Monitor } from '@lucide/vue';
import type { ThemeMode } from '../../composables/useTheme';

defineProps<{ modelValue: ThemeMode }>();

defineEmits<{ 'update:modelValue': [mode: ThemeMode] }>();
</script>
