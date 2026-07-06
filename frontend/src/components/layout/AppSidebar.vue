<template>
  <div v-if="mobileOpen" class="sidebar-overlay" @click="$emit('close')" />

  <aside
    :class="['sidebar', { 'sidebar--expanded': isExpanded, 'sidebar--mobile-open': mobileOpen }]"
    @mouseenter="hovering = true"
    @mouseleave="hovering = false"
  >
    <div class="sidebar-top">
      <div class="logo-icon">O</div>
      <div class="logo-text">
        <div class="logo-name">Omie<span>Sync</span></div>
        <div class="logo-sub">ADMIN PANEL</div>
      </div>
    </div>

    <nav class="nav-scroll">
      <template v-for="section in navSections" :key="section.label">
        <div class="nav-label">{{ section.label }}</div>
        <RouterLink
          v-for="item in section.items"
          :key="item.to"
          :to="item.to"
          custom
          v-slot="{ isActive, navigate }"
        >
          <div
            :class="['nav-item', { active: isActive }]"
            @click="navigate(); $emit('close')"
            role="link"
          >
            <div class="nav-icon">
              <component :is="item.icon" />
            </div>
            <span class="nav-text">{{ item.label }}</span>
          </div>
        </RouterLink>
      </template>
    </nav>

    <div class="sidebar-footer">
      <div class="user-card">
        <div class="user-avatar">{{ initials }}</div>
        <div class="user-info">
          <div class="user-name">{{ auth.user?.nome ?? '...' }}</div>
          <div class="user-role">{{ roleLabel }}</div>
        </div>
        <button class="logout-mini" @click="handleLogout" title="Sair">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75" />
          </svg>
        </button>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import {
  IconGrid, IconBuilding, IconFactory,
  IconUsers, IconKey, IconSync, IconUser, IconDatabase
} from '@/components/ui/icons'

defineProps<{ mobileOpen: boolean }>()
defineEmits<{ close: [] }>()

const auth     = useAuthStore()
const ui       = useUiStore()
const router   = useRouter()
const hovering = ref(false)

async function handleLogout() {
  if (confirm('Deseja realmente sair do sistema?')) {
    await auth.logout()
    router.push('/login')
  }
}

const isExpanded = computed(() => ui.sidebarPinned.value || hovering.value)

const initials = computed(() => {
  const name = auth.user?.nome ?? ''
  return name.split(' ').map((w: string) => w[0]).slice(0, 2).join('').toUpperCase() || 'U'
})

const roleLabel = computed(() => {
  const labels: Record<string, string> = {
    admin_global: 'Admin Global',
    admin_grupo:  'Admin Grupo',
    viewer:       'Viewer'
  }
  return labels[auth.user?.role ?? 'viewer'] ?? ''
})

const navSections = computed(() => {
  const role = auth.user?.role
  const sections: Array<{ label: string; items: Array<{ to: string; label: string; icon: unknown }> }> = []

  sections.push({
    label: 'PRINCIPAL',
    items: [{ to: '/', label: 'Dashboard', icon: IconGrid }]
  })

  const adminItems: Array<{ to: string; label: string; icon: unknown }> = []
  if (role === 'admin_global') {
    adminItems.push({ to: '/grupos', label: 'Grupos', icon: IconBuilding })
    adminItems.push({ to: '/admin/sync-control', label: 'Sync Control', icon: IconSync })
  }
  if (role === 'admin_grupo') {
    adminItems.push({ to: '/empresas',   label: 'Empresas',   icon: IconFactory })
    adminItems.push({ to: '/usuarios',   label: 'Usuarios',   icon: IconUsers })
    adminItems.push({ to: '/permissoes', label: 'Permissoes', icon: IconKey })
  }
  if (adminItems.length) {
    sections.push({ label: 'ADMINISTRACAO', items: adminItems })
  }

  const systemItems: Array<{ to: string; label: string; icon: unknown }> = []
  if (role === 'admin_grupo') {
    systemItems.push({ to: '/sync', label: 'Sync', icon: IconSync })
  }
  if (role === 'admin_global') {
    systemItems.push({ to: '/omie-config', label: 'Config Omie', icon: IconKey })
  }
  if (role === 'admin_global' || role === 'admin_grupo') {
    systemItems.push({ to: '/sql-explorer', label: 'SQL Explorer', icon: IconDatabase })
  }
  systemItems.push({ to: '/perfil', label: 'Perfil', icon: IconUser })
  sections.push({ label: 'SISTEMA', items: systemItems })

  return sections
})
</script>

<style scoped>
.sidebar-overlay {
  position: fixed; inset: 0;
  background: var(--overlay);
  z-index: 200;
  backdrop-filter: blur(2px);
}

.sidebar {
  position: fixed; left: 0; top: 0; bottom: 0;
  width: var(--sidebar-w);
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border);
  backdrop-filter: blur(24px);
  z-index: 300;
  display: flex; flex-direction: column;
  transition: width 0.28s cubic-bezier(0.4,0,0.2,1), transform 0.28s cubic-bezier(0.4,0,0.2,1), box-shadow 0.28s;
  overflow: hidden;
}

.sidebar--expanded { width: var(--sidebar-w-expanded); box-shadow: 4px 0 40px rgba(0,0,0,0.25); }

@media (max-width: 768px) {
  .sidebar { transform: translateX(-100%); width: var(--sidebar-w-expanded) !important; }
  .sidebar--mobile-open { transform: translateX(0); box-shadow: 4px 0 40px rgba(0,0,0,0.35); }
}

.sidebar-top {
  padding: 16px 0; border-bottom: 1px solid var(--border);
  display: flex; align-items: center;
  min-height: 64px; flex-shrink: 0;
  padding-left: 15px;
}

.logo-icon {
  width: 34px; height: 34px; border-radius: 10px;
  background: linear-gradient(135deg, var(--accent), var(--accent3));
  display: flex; align-items: center; justify-content: center;
  font-size: 16px; font-weight: 800; color: #080c12; flex-shrink: 0;
  box-shadow: 0 0 18px rgba(0,229,255,0.3);
}

.logo-text {
  opacity: 0; transform: translateX(-8px);
  transition: opacity 0.2s, transform 0.2s;
  white-space: nowrap; margin-left: 12px; overflow: hidden;
}
.sidebar--expanded .logo-text,
.sidebar--mobile-open .logo-text { opacity: 1; transform: translateX(0); }

.logo-name { font-size: 16px; font-weight: 800; color: var(--text); }
.logo-name span { color: var(--accent); }
.logo-sub { font-family: var(--mono); font-size: 9px; color: var(--text3); letter-spacing: 2px; margin-top: 1px; }

.nav-scroll { flex: 1; overflow-y: auto; overflow-x: hidden; padding: 10px 8px; }
.nav-scroll::-webkit-scrollbar { width: 3px; }
.nav-scroll::-webkit-scrollbar-thumb { background: var(--border2); border-radius: 3px; }

.nav-label {
  font-family: var(--mono); font-size: 9px; letter-spacing: 2px; color: var(--text3);
  padding: 8px 8px 4px; white-space: nowrap; overflow: hidden;
  opacity: 0; max-height: 0; transition: opacity 0.2s, max-height 0.2s;
}
.sidebar--expanded .nav-label,
.sidebar--mobile-open .nav-label { opacity: 1; max-height: 30px; }

.nav-item {
  display: flex; align-items: center; padding: 0 8px; height: 42px;
  border-radius: 8px; cursor: pointer; color: var(--text2);
  transition: var(--trans); margin-bottom: 1px;
  white-space: nowrap; overflow: hidden; border: 1px solid transparent;
  text-decoration: none;
}
.nav-item:hover { background: var(--bg3); color: var(--text); }
.nav-item.active { background: rgba(0,229,255,0.09); color: var(--accent); border-color: rgba(0,229,255,0.18); }

.nav-icon { width: 32px; height: 32px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; border-radius: 8px; }
.nav-icon :deep(svg) { width: 20px; height: 20px; }

.nav-text { font-size: 13.5px; font-weight: 600; margin-left: 6px; opacity: 0; width: 0; transition: opacity 0.18s, width 0.18s; }
.sidebar--expanded .nav-text,
.sidebar--mobile-open .nav-text { opacity: 1; width: auto; }

.sidebar-footer { padding: 16px 10px; border-top: 1px solid var(--border); flex-shrink: 0; }

.user-card { display: flex; align-items: center; gap: 10px; padding: 8px; border-radius: 10px; background: var(--bg3); overflow: hidden; }

.user-avatar {
  width: 30px; height: 30px; flex-shrink: 0; border-radius: 8px;
  background: linear-gradient(135deg, var(--accent), var(--accent3));
  display: flex; align-items: center; justify-content: center;
  font-size: 11px; font-weight: 800; color: #080c12;
}

.user-info { opacity: 0; width: 0; overflow: hidden; transition: opacity 0.18s, width 0.18s; }
.sidebar--expanded .user-info,
.sidebar--mobile-open .user-info { opacity: 1; width: auto; }

.user-name { font-size: 12px; font-weight: 700; white-space: nowrap; }
.user-role { font-family: var(--mono); font-size: 9px; color: var(--text3); }

.logout-mini {
  background: transparent; border: none; cursor: pointer;
  color: var(--text3); padding: 4px; border-radius: 6px;
  display: flex; align-items: center; justify-content: center;
  transition: var(--trans); opacity: 0; width: 0; overflow: hidden;
}
.sidebar--expanded .logout-mini,
.sidebar--mobile-open .logout-mini { opacity: 1; width: 28px; margin-left: auto; }
.logout-mini:hover { color: var(--red); background: rgba(239,68,68,0.1); }
.logout-mini svg { width: 14px; height: 14px; }
</style>
