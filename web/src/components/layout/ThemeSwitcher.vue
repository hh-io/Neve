<template>
  <div ref="rootRef" class="theme-switcher">
    <button
      class="btn btn-icon theme-trigger"
      :class="{ active: open }"
      title="切换主题"
      @click="open = !open"
    >
      <component :is="currentOption.icon" :size="16" />
    </button>

    <Transition name="theme-menu-fade">
      <div v-if="open" class="theme-menu" role="menu">
        <button
          v-for="option in options"
          :key="option.value"
          class="theme-menu-item"
          :class="{ active: modelValue === option.value }"
          role="menuitemradio"
          :aria-checked="modelValue === option.value"
          @click="select(option.value)"
        >
          <component :is="option.icon" :size="16" class="theme-menu-icon" />
          <span class="theme-menu-label">{{ option.label }}</span>
          <Check v-if="modelValue === option.value" :size="14" class="theme-menu-check" />
        </button>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { Check, Monitor, Moon, Sparkles, Sun } from '@lucide/vue';
import type { ThemeMode } from '../../composables/useTheme';

const props = defineProps<{ modelValue: ThemeMode }>();
const emit = defineEmits<{ 'update:modelValue': [mode: ThemeMode] }>();

const options: { value: ThemeMode; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: '亮色模式', icon: Sun },
  { value: 'dark', label: '暗色模式', icon: Moon },
  { value: 'system', label: '跟随系统', icon: Monitor },
  { value: 'stripe', label: 'Stripe 主题', icon: Sparkles },
];

const open = ref(false);
const rootRef = ref<HTMLElement | null>(null);

const currentOption = computed(
  () => options.find((option) => option.value === props.modelValue) ?? options[0],
);

function select(mode: ThemeMode): void {
  emit('update:modelValue', mode);
  open.value = false;
}

function handleClickOutside(event: MouseEvent): void {
  if (rootRef.value && !rootRef.value.contains(event.target as Node)) {
    open.value = false;
  }
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape') {
    open.value = false;
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside);
  document.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
  document.removeEventListener('keydown', handleKeydown);
});
</script>

<style scoped>
.theme-switcher {
  position: relative;
}

.theme-trigger.active {
  color: var(--accent);
  border-color: var(--accent);
}

.theme-menu {
  position: absolute;
  top: calc(100% + var(--space-2));
  right: 0;
  z-index: 50;
  min-width: 160px;
  padding: var(--space-2);
  display: flex;
  flex-direction: column;
  gap: 2px;
  background-color: var(--surface-1);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
}

.theme-menu-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: none;
  background: none;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: var(--font-size-sm);
  text-align: left;
  cursor: pointer;
  transition: background-color var(--transition-base), color var(--transition-base);
}

.theme-menu-item:hover {
  background-color: var(--bg-tertiary);
  color: var(--text-primary);
}

.theme-menu-item.active {
  color: var(--accent);
  font-weight: 600;
}

.theme-menu-icon {
  flex-shrink: 0;
}

.theme-menu-label {
  flex: 1;
}

.theme-menu-check {
  flex-shrink: 0;
  color: var(--accent);
}

.theme-menu-fade-enter-active,
.theme-menu-fade-leave-active {
  transition: opacity var(--transition-base), transform var(--transition-base);
}

.theme-menu-fade-enter-from,
.theme-menu-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
