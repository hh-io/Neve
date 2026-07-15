<template>
  <Transition name="toast">
    <div v-if="toast.show" class="toast" :class="toast.type">
      <component :is="toast.type === 'success' ? CheckCircle2 : AlertCircle" :size="18" />
      <span>{{ toast.message }}</span>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { CheckCircle2, AlertCircle } from '@lucide/vue';
import { useToast } from '../../composables/useToast';

// 自订阅模块级单例,无需父组件传 props
const { toast } = useToast();
</script>

<style scoped>
.toast {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  padding: 12px 20px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: var(--font-size-sm);
  font-weight: 500;
  z-index: 9999;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.toast.success {
  background: var(--income);
  color: #fff;
}

.toast.error {
  background: var(--expense);
  color: #fff;
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
