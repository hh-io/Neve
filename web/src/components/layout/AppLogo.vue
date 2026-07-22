<template>
  <!-- 「N」monogram:上扬右笔 + 峰值节点(增长/平衡的克制隐喻);currentColor 跟随容器 -->
  <svg :width="size" :height="size" viewBox="0 0 32 32" fill="none" aria-hidden="true">
    <path d="M9 25V7" stroke="currentColor" :stroke-width="stroke" stroke-linecap="round" />
    <path d="M9 8L23 24" stroke="currentColor" :stroke-width="stroke" stroke-linecap="round" stroke-linejoin="round" />
    <path d="M23 25V10" stroke="currentColor" :stroke-width="stroke" stroke-linecap="round" />
    <circle v-if="node" cx="23" cy="7.2" r="2.5" fill="currentColor" />
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ size?: number; stroke?: number; node?: boolean }>(), {
  size: 20,
  stroke: 4.2,
  node: true,
});
</script>
