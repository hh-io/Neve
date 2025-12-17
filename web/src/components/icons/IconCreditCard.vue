<template>
  <IconBase :size="size" :color="color">
    <rect x="1" y="4" width="22" height="16" rx="2" ry="2"/>
    <line x1="1" y1="10" x2="23" y2="10"/>
  </IconBase>
</template>

<script setup>
import IconBase from './IconBase.vue';
defineProps({
  size: { type: [Number, String], default: 24 },
  color: { type: String, default: 'currentColor' }
});
</script>
