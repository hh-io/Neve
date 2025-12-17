<template>
  <Transition name="toast">
    <div v-if="show" class="toast" :class="type">
      <span v-html="type === 'success' ? icons.check : icons.alert" style="width: 18px; height: 18px;"></span>
      <span>{{ message }}</span>
    </div>
  </Transition>
</template>

<script setup>
import { icons } from '../../composables/icons';

defineProps({
  show: { type: Boolean, default: false },
  message: { type: String, default: '' },
  type: { type: String, default: 'success' } // 'success' | 'error'
});
</script>

<style scoped>
.toast {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  padding: 12px 20px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  font-weight: 500;
  z-index: 9999;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.toast.success {
  background: linear-gradient(135deg, #6B9B7A 0%, #5a8a69 100%);
  color: white;
}

.toast.error {
  background: linear-gradient(135deg, #C27B7B 0%, #b06a6a 100%);
  color: white;
}

.toast-enter-active {
  animation: toast-in 0.3s ease;
}

.toast-leave-active {
  animation: toast-out 0.3s ease;
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateX(-50%) translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateX(-50%) translateY(0);
  }
}

@keyframes toast-out {
  from {
    opacity: 1;
    transform: translateX(-50%) translateY(0);
  }
  to {
    opacity: 0;
    transform: translateX(-50%) translateY(20px);
  }
}
</style>
