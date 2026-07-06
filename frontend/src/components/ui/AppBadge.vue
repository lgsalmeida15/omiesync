<template>
  <span :class="['badge', `badge-${variant}`]">
    <slot />
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'green' | 'red' | 'blue' | 'orange' | 'gray' | 'purple' | 'yellow'
}>(), { variant: 'gray' })
</script>

<style scoped>
.badge {
  font-family: var(--mono);
  font-size: 10px;
  padding: 3px 9px;
  border-radius: 20px;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.badge-green  { background: rgba(34,197,94,0.12);  color: #22c55e; border: 1px solid rgba(34,197,94,0.22); }
.badge-red    { background: rgba(239,68,68,0.12);  color: #ef4444; border: 1px solid rgba(239,68,68,0.22); }
.badge-blue   { background: rgba(0,229,255,0.10);  color: #00e5ff; border: 1px solid rgba(0,229,255,0.20); }
.badge-orange { background: rgba(255,107,53,0.12); color: #ff6b35; border: 1px solid rgba(255,107,53,0.20); }
.badge-gray   { background: rgba(255,255,255,0.06);color: var(--text3); border: 1px solid var(--border); }
.badge-purple { background: rgba(124,58,237,0.12); color: #7c3aed; border: 1px solid rgba(124,58,237,0.22); }
.badge-yellow { background: rgba(245,158,11,0.12); color: #f59e0b; border: 1px solid rgba(245,158,11,0.22); }
</style>
