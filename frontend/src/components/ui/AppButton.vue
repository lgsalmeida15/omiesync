<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="['app-btn', `app-btn--${variant}`, { 'app-btn--loading': loading, 'app-btn--sm': size === 'sm' }]"
    v-bind="$attrs"
  >
    <AppSpinner v-if="loading" size="sm" />
    <slot v-else />
  </button>
</template>

<script setup lang="ts">
import AppSpinner from './AppSpinner.vue'

withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
  type?:    'button' | 'submit' | 'reset'
  loading?: boolean
  disabled?: boolean
  size?:    'sm' | 'md'
}>(), { variant: 'primary', type: 'button', loading: false, disabled: false, size: 'md' })
</script>

<style scoped>
.app-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  padding: 10px 18px;
  border-radius: 10px;
  font-family: var(--font-body);
  font-size: var(--fs-sm);
  font-weight: 600;
  cursor: pointer;
  transition: var(--transition);
  border: 1px solid transparent;
  white-space: nowrap;
  outline: none;
  min-height: 40px;
}

.app-btn--sm { padding: 6px 14px; font-size: var(--fs-xs); min-height: 32px; border-radius: 8px; }

.app-btn--primary {
  background: var(--primary);
  color: var(--text-oncolor);
  border-color: var(--primary);
}
.app-btn--primary:hover:not(:disabled) {
  background: var(--primary-hover);
}

.app-btn--secondary {
  background: var(--surface-2);
  color: var(--text);
  border-color: var(--border-strong);
}
.app-btn--secondary:hover:not(:disabled) {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-weak);
}

.app-btn--danger {
  background: var(--danger-weak);
  color: var(--danger);
  border-color: var(--danger);
}
.app-btn--danger:hover:not(:disabled) {
  background: var(--danger-weak);
  border-color: var(--danger);
}

.app-btn--ghost {
  background: transparent;
  color: var(--text-muted);
  border-color: transparent;
}
.app-btn--ghost:hover:not(:disabled) { color: var(--text); background: var(--surface-2); }

.app-btn:disabled,
.app-btn--loading {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}
</style>
