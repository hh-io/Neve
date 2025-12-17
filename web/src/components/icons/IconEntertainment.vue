<template>
  <IconBase :size="size" :color="color">
    <circle cx="12" cy="12" r="10"/>
    <polygon points="10 8 16 12 10 16 10 8"/>
  </IconBase>
</template>

<script setup>
import IconBase from './IconBase.vue';
defineProps({
  size: { type: [Number, String], default: 24 },
  color: { type: String, default: 'currentColor' }
});
</script>
