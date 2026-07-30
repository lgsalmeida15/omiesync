<template>
  <RouterView v-if="ready" />
  <div v-else class="app-init">
    <span class="init-dot" /><span class="init-dot" /><span class="init-dot" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import { useUiStore }   from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'

const ui   = useUiStore()
const auth = useAuthStore()
const ready = ref(false)

onMounted(async () => {
  document.documentElement.setAttribute('data-theme', ui.theme)
  await auth.init()
  ready.value = true
})

// Quando o usuário faz login (isAuthenticated muda para true), cicla o ready
// para garantir que todos os componentes montem com auth.user já disponível.
watch(() => auth.isAuthenticated, async (isAuth, wasAuth) => {
  if (isAuth && !wasAuth && ready.value) {
    ready.value = false
    await nextTick()
    ready.value = true
  }
})
</script>

<style>
.app-init {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  gap: 8px;
}
.init-dot {
  display: inline-block;
  width: 8px; height: 8px;
  border-radius: 50%;
  background: var(--accent, #00e5ff);
  animation: init-pulse 1.2s ease-in-out infinite;
}
.init-dot:nth-child(2) { animation-delay: 0.2s; }
.init-dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes init-pulse {
  0%, 80%, 100% { opacity: 0.2; transform: scale(0.8); }
  40%           { opacity: 1;   transform: scale(1.1); }
}
</style>
