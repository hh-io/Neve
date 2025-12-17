<template>
  <IconBase :size="size" :color="color">
    <rect x="2" y="7" width="20" height="14" rx="2"/>
    <path d="M16 3h-8l-2 4h12l-2-4z"/>
  </IconBase>
</template>

<script setup>
import IconBase from './IconBase.vue';
defineProps({
  size: { type: [Number, String], default: 24 },
  color: { type: String, default: 'currentColor' }
});
</script>
