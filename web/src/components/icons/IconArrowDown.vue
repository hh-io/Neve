<template>
  <IconBase :size="size" :color="color">
    <line x1="12" y1="5" x2="12" y2="19"/>
    <polyline points="19 12 12 19 5 12"/>
  </IconBase>
</template>

<script setup>
import IconBase from './IconBase.vue';
defineProps({
  size: { type: [Number, String], default: 24 },
  color: { type: String, default: 'currentColor' }
});
</script>
