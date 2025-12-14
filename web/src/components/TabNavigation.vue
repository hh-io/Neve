<template>
  <nav class="tab-nav">
    <div class="tab-container">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        class="tab-btn"
        :class="{ active: modelValue === tab.id }"
        @click="$emit('update:modelValue', tab.id)"
      >
        <span class="tab-icon">{{ tab.icon }}</span>
        <span class="tab-label">{{ tab.label }}</span>
      </button>
    </div>
  </nav>
</template>

<script setup>
defineProps({
  modelValue: { type: String, required: true },
  tabs: { type: Array, required: true }
});

defineEmits(['update:modelValue']);
</script>

<style scoped>
.tab-nav {
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  position: sticky;
  top: 60px;
  z-index: 90;
}

.tab-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 var(--space-6);
  display: flex;
  gap: var(--space-1);
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}

.tab-container::-webkit-scrollbar {
  display: none;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-5);
  border: none;
  background: transparent;
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  white-space: nowrap;
  position: relative;
  transition: all var(--transition-fast);
}

.tab-btn:hover {
  color: var(--color-text-primary);
  background: rgba(0, 0, 0, 0.03);
}

.tab-btn.active {
  color: var(--color-blue);
}

.tab-btn.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: var(--space-4);
  right: var(--space-4);
  height: 2px;
  background: var(--color-blue);
  border-radius: 2px 2px 0 0;
}

.tab-icon {
  font-size: 16px;
}

.tab-label {
  font-family: var(--font-family);
}

/* Mobile optimization */
@media (max-width: 640px) {
  .tab-container {
    padding: 0 var(--space-3);
  }
  
  .tab-btn {
    padding: var(--space-3) var(--space-4);
    flex-direction: column;
    gap: var(--space-1);
  }
  
  .tab-icon {
    font-size: 18px;
  }
  
  .tab-label {
    font-size: 11px;
  }
}
</style>
