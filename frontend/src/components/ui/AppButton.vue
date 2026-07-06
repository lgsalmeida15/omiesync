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
  font-family: var(--font);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: var(--trans);
  border: 1px solid transparent;
  white-space: nowrap;
  outline: none;
  min-height: 40px;
}

.app-btn--sm { padding: 6px 14px; font-size: 12px; min-height: 32px; border-radius: 8px; }

.app-btn--primary {
  background: var(--accent);
  color: #080c12;
  border-color: var(--accent);
}
.app-btn--primary:hover:not(:disabled) {
  background: rgba(0,229,255,0.85);
}

.app-btn--secondary {
  background: var(--bg3);
  color: var(--text);
  border-color: var(--border2);
}
.app-btn--secondary:hover:not(:disabled) {
  border-color: var(--accent);
  color: var(--accent);
  background: rgba(0,229,255,0.06);
}

.app-btn--danger {
  background: rgba(239,68,68,0.1);
  color: var(--red);
  border-color: rgba(239,68,68,0.3);
}
.app-btn--danger:hover:not(:disabled) {
  background: rgba(239,68,68,0.2);
  border-color: var(--red);
}

.app-btn--ghost {
  background: transparent;
  color: var(--text2);
  border-color: transparent;
}
.app-btn--ghost:hover:not(:disabled) { color: var(--text); background: var(--bg3); }

.app-btn:disabled,
.app-btn--loading {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}
</style>
