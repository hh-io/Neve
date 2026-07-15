<template>
  <div class="theme-switcher">
    <div class="theme-slider" :class="modelValue"></div>
    <button 
      class="theme-btn" 
      :class="{ active: modelValue === 'light' }"
      title="亮色模式"
      @click="$emit('update:modelValue', 'light')"
    >
      <Sun :size="16" />
    </button>
    <button
      class="theme-btn"
      :class="{ active: modelValue === 'dark' }"
      title="暗色模式"
      @click="$emit('update:modelValue', 'dark')"
    >
      <Moon :size="16" />
    </button>
    <button
      class="theme-btn"
      :class="{ active: modelValue === 'system' }"
      title="跟随系统"
      @click="$emit('update:modelValue', 'system')"
    >
      <Monitor :size="16" />
    </button>
    <button
      class="theme-btn"
      :class="{ active: modelValue === 'stripe' }"
      title="Stripe 主题"
      @click="$emit('update:modelValue', 'stripe')"
    >
      <Sparkles :size="16" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Sun, Moon, Monitor, Sparkles } from '@lucide/vue';
import type { ThemeMode } from '../../composables/useTheme';

defineProps<{ modelValue: ThemeMode }>();

defineEmits<{ 'update:modelValue': [mode: ThemeMode] }>();
</script>
