<template>
  <IconBase :size="size" :color="color">
    <circle cx="12" cy="12" r="10"/>
    <path d="M16 8l-4 4-4-4"/>
    <path d="M12 16V8"/>
  </IconBase>
</template>

<script setup>
import IconBase from './IconBase.vue';
defineProps({
  size: { type: [Number, String], default: 24 },
  color: { type: String, default: 'currentColor' }
});
</script>
