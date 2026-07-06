import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    // ── Público ──────────────────────────────────────────────
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
      meta: { public: true }
    },

    // ── Autenticado ──────────────────────────────────────────
    {
      path: '/',
      component: () => import('@/components/layout/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        // Dashboard (Fase 2 — placeholder por ora)
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/views/DashboardView.vue')
        },

        // Grupos — admin_global apenas
        {
          path: 'grupos',
          name: 'Grupos',
          component: () => import('@/views/grupos/GruposView.vue'),
          meta: { roles: ['admin_global'] }
        },
        {
          path: 'grupos/:grupoId/empresas',
          name: 'GrupoEmpresas',
          component: () => import('@/views/empresas/EmpresasView.vue'),
          meta: { roles: ['admin_global'] }
        },

        // Empresas — admin_grupo apenas (admin_global acessa via /grupos/:id/empresas)
        {
          path: 'empresas',
          name: 'MinhasEmpresas',
          component: () => import('@/views/empresas/EmpresasView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Usuários
        {
          path: 'usuarios',
          name: 'Usuarios',
          component: () => import('@/views/usuarios/UsuariosView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Permissões
        {
          path: 'permissoes',
          name: 'Permissoes',
          component: () => import('@/views/permissoes/PermissoesView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Sync
        {
          path: 'sync',
          name: 'Sync',
          component: () => import('@/views/sync/SyncView.vue'),
          meta: { roles: ['admin_grupo'] }
        },
        {
          path: 'sync/:empresaId',
          name: 'SyncEmpresa',
          component: () => import('@/views/sync/SyncEmpresaView.vue'),
          meta: { roles: ['admin_grupo'] }
        },

        // Omie Config — admin_global apenas
        {
          path: 'omie-config',
          name: 'OmieConfig',
          component: () => import('@/views/admin/OmieConfigView.vue'),
          meta: { roles: ['admin_global'] }
        },
        {
          path: 'admin/sync-control',
          name: 'SyncControlCenter',
          component: () => import('@/views/admin/SyncControlCenter.vue'),
          meta: { roles: ['admin_global'] }
        },

        // SQL Explorer — admin_global e admin_grupo
        {
          path: 'sql-explorer',
          name: 'SqlExplorer',
          component: () => import('@/views/sql/SqlExplorerView.vue'),
          meta: { roles: ['admin_global', 'admin_grupo'] }
        },

        // Perfil — todos
        {
          path: 'perfil',
          name: 'Perfil',
          component: () => import('@/views/PerfilView.vue')
        },

        // 403
        {
          path: '403',
          name: 'Forbidden',
          component: () => import('@/views/ForbiddenView.vue')
        }
      ]
    },

    // Catch-all
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

// ── Guards ──────────────────────────────────────────────────
router.beforeEach(async to => {
  const auth = useAuthStore()

  // Rota pública
  if (to.meta.public) {
    // Já logado e tenta acessar /login → vai para home
    if (auth.isAuthenticated) return { name: 'Dashboard' }
    return true
  }

  // Requer autenticação
  if (to.meta.requiresAuth !== false) {
    if (!auth.isAuthenticated) return { name: 'Login' }

    // Carrega dados do usuário se ainda não carregou
    if (!auth.user) {
      try {
        await auth.fetchMe()
      } catch {
        auth.clearTokens()
        return { name: 'Login' }
      }
    }

    // Verifica role
    const requiredRoles = to.meta.roles as string[] | undefined
    if (requiredRoles && !requiredRoles.includes(auth.user!.role)) {
      return { name: 'Forbidden' }
    }
  }

  return true
})

export default router
