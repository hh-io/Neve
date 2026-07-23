<template>
  <!-- 雪山品牌标(Neve = 意大利语/冰川学的"雪、粒雪"):次峰 + 主峰 + 雪顶三层,
       与 public/neve.svg favicon 同一几何,单尺寸抗糊;currentColor 跟随容器 -->
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
    <path d="M2 18L9 6L15 15H10L8 18H2Z" fill-opacity="0.5" />
    <path d="M8 18L15 4L22 18H8Z" fill-opacity="0.9" />
    <path d="M15 4L12 9L15 11L18 9L15 4Z" />
  </svg>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ size?: number }>(), {
  size: 22,
});
</script>
