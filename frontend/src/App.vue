<template>
  <RouterView />
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

onMounted(() => {
  // Restaura tema salvo
  document.documentElement.setAttribute('data-theme', ui.theme)
})
</script>
