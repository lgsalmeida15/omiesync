<template>
  <RouterView v-if="ready" />
  <div v-else class="app-init">
    <span class="init-dot" /><span class="init-dot" /><span class="init-dot" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useUiStore }   from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'

// O store aplica o tema no <html> por watcher com immediate, e o script inline do
// index.html já cobriu a primeira pintura. Nada de setAttribute aqui.
useUiStore()

const auth = useAuthStore()
const ready = ref(false)

onMounted(async () => {
  await auth.init()
  ready.value = true
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
  background: var(--accent, var(--info));
  animation: init-pulse 1.2s ease-in-out infinite;
}
.init-dot:nth-child(2) { animation-delay: 0.2s; }
.init-dot:nth-child(3) { animation-delay: 0.4s; }
@keyframes init-pulse {
  0%, 80%, 100% { opacity: 0.2; transform: scale(0.8); }
  40%           { opacity: 1;   transform: scale(1.1); }
}
</style>
