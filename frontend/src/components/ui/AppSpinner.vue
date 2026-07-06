<template>
  <!-- Fullpage overlay -->
  <div v-if="fullpage" class="spinner-overlay">
    <div class="spinner-ring" :style="sizeStyle" />
  </div>

  <!-- Inline -->
  <div v-else class="spinner-ring" :style="sizeStyle" />
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  fullpage?: boolean
  size?: 'sm' | 'md' | 'lg'
}>(), { fullpage: false, size: 'md' })

const sizes = { sm: '16px', md: '24px', lg: '40px' }
const sizeStyle = computed(() => ({ width: sizes[props.size], height: sizes[props.size] }))
</script>

<style scoped>
.spinner-overlay {
  position: fixed; inset: 0; z-index: 9999;
  background: rgba(8,12,18,0.7);
  display: flex; align-items: center; justify-content: center;
  backdrop-filter: blur(4px);
}

.spinner-ring {
  border: 2px solid var(--border2);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
  flex-shrink: 0;
}

@keyframes spin { to { transform: rotate(360deg); } }
</style>
