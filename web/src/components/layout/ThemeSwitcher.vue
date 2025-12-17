<template>
  <div class="theme-switcher">
    <div class="theme-slider" :class="modelValue"></div>
    <button 
      class="theme-btn" 
      :class="{ active: modelValue === 'light' }"
      @click="$emit('update:modelValue', 'light')"
      title="亮色模式"
    >
      <span v-html="icons.sun"></span>
    </button>
    <button 
      class="theme-btn" 
      :class="{ active: modelValue === 'dark' }"
      @click="$emit('update:modelValue', 'dark')"
      title="暗色模式"
    >
      <span v-html="icons.moon"></span>
    </button>
    <button 
      class="theme-btn" 
      :class="{ active: modelValue === 'geek' }"
      @click="$emit('update:modelValue', 'geek')"
      title="极客模式"
    >
      <span v-html="icons.terminal"></span>
    </button>
    <button 
      class="theme-btn" 
      :class="{ active: modelValue === 'system' }"
      @click="$emit('update:modelValue', 'system')"
      title="跟随系统"
    >
      <span v-html="icons.monitor"></span>
    </button>
  </div>
</template>

<script setup>
import { icons } from '../../composables/icons';

defineProps({
  modelValue: { type: String, required: true }
});

defineEmits(['update:modelValue']);
</script>
