import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

export function usePermission() {
  const auth = useAuthStore()

  const role = computed(() => auth.user?.role)

  /** Verifica se o usuário tem uma das roles indicadas */
  function hasRole(...roles: string[]): boolean {
    return roles.includes(auth.user?.role ?? '')
  }

  /** admin_global pode gerenciar qualquer grupo */
  const canManageGroups = computed(() => auth.isAdminGlobal.value)

  /** admin_global ou admin_grupo podem gerenciar empresas/usuários */
  const canManageResources = computed(() => auth.isAdmin.value)

  /** Só admin_global pode criar outros admin_global */
  function canAssignRole(targetRole: string): boolean {
    if (targetRole === 'admin_global') return auth.isAdminGlobal.value
    return auth.isAdmin.value
  }

  /** Roles disponíveis para criar usuário (baseado na role do logado) */
  const assignableRoles = computed(() => {
    if (auth.isAdminGlobal.value) return [
      { value: 'admin_global', label: 'Admin Global' },
      { value: 'admin_grupo',  label: 'Admin Grupo' },
      { value: 'viewer',       label: 'Viewer' }
    ]
    return [
      { value: 'admin_grupo', label: 'Admin Grupo' },
      { value: 'viewer',      label: 'Viewer' }
    ]
  })

  return { role, hasRole, canManageGroups, canManageResources, canAssignRole, assignableRoles }
}
