<template>
  <header class="topbar">
    <!-- Hamburger -->
    <button class="hamburger" @click="$emit('toggle-sidebar')" aria-label="Menu">
      <span /><span /><span />
    </button>

    <!-- Título -->
    <div class="topbar-title">
      {{ pageTitles[route.name as string] ?? 'Omie' }}
      <span v-if="pageSubtitle">{{ pageSubtitle }}</span>
    </div>

    <!-- Direita -->
    <div class="topbar-right">
      <!-- Live indicator -->
      <div class="live-wrap">
        <span class="live-dot" />
        <span class="live-label">AO VIVO</span>
      </div>

      <!-- Theme toggle -->
      <button class="theme-btn" @click="ui.toggleTheme()" :title="ui.theme === 'dark' ? 'Modo claro' : 'Modo escuro'">
        <Sun  v-if="ui.theme === 'dark'" :size="16" />
        <Moon v-else                     :size="16" />
      </button>

      <!-- Logout -->
      <button class="logout-btn" @click="handleLogout" title="Sair do sistema">
        <LogOut :size="16" />
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { Sun, Moon, LogOut } from '@lucide/vue'

defineEmits<{ 'toggle-sidebar': [] }>()

const route = useRoute()
const router = useRouter()
const ui    = useUiStore()
const auth  = useAuthStore()

async function handleLogout() {
  if (confirm('Deseja realmente sair do sistema?')) {
    await auth.logout()
    router.push('/login')
  }
}

const pageTitles: Record<string, string> = {
  Dashboard:    'Dashboard',
  Grupos:       'Grupos',
  GrupoEmpresas:'Empresas',
  MinhasEmpresas:'Empresas',
  Usuarios:     'Usuários',
  Permissoes:   'Permissões',
  Sync:         'Sincronização',
  Perfil:       'Perfil',
  Forbidden:    'Acesso Negado',
}

const pageSubtitle = computed(() => {
  if (route.name === 'Dashboard') return 'Visão Geral'
  if (route.name === 'Sync') return 'Motor ETL'
  return null
})
</script>

<style scoped>
.topbar {
  position: sticky; top: 0; z-index: 50;
  background: var(--topbar-bg);
  backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--border);
  padding: 0 24px;
  height: var(--topbar-h);
  display: flex; align-items: center; gap: 14px;
  transition: background 0.3s;
}

.hamburger {
  width: 38px; height: 38px; border-radius: 10px;
  border: 1px solid var(--border2); background: var(--bg3);
  cursor: pointer; display: none; align-items: center;
  justify-content: center; flex-direction: column; gap: 4px; flex-shrink: 0;
  transition: var(--trans);
}
.hamburger:hover { border-color: var(--accent); }
.hamburger span { display: block; width: 16px; height: 1.5px; background: var(--text2); border-radius: 2px; }

@media (max-width: 768px) { .hamburger { display: flex; } }

.topbar-title {
  font-size: 16px; font-weight: 700; flex: 1; min-width: 0;
  color: var(--text);
}
.topbar-title span { color: var(--accent); margin-left: 6px; }

.topbar-right { display: flex; align-items: center; gap: 10px; }

.live-wrap { display: flex; align-items: center; gap: 6px; }
.live-dot {
  width: 6px; height: 6px; border-radius: 50%;
  background: #22c55e; animation: pulse 2s infinite;
}
@keyframes pulse { 0%,100%{opacity:1} 50%{opacity:0.4} }
.live-label { font-family: var(--mono); font-size: 9px; color: var(--text3); }

@media (max-width: 480px) { .live-wrap { display: none; } }

.theme-btn {
  width: 38px; height: 38px; border-radius: 10px;
  border: 1px solid var(--border2); background: var(--bg3);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  color: var(--text2); transition: var(--trans); flex-shrink: 0;
}
.theme-btn:hover { border-color: var(--accent); color: var(--accent); }
.theme-btn svg { width: 16px; height: 16px; }

.logout-btn {
  width: 38px; height: 38px; border-radius: 10px;
  border: 1px solid var(--border2); background: var(--bg3);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  color: var(--text3); transition: var(--trans); flex-shrink: 0;
}
.logout-btn:hover { border-color: var(--red); color: var(--red); background: rgba(239,68,68,0.08); }
.logout-btn svg { width: 16px; height: 16px; }
</style>
