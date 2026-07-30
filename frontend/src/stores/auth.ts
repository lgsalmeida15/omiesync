import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api/client'

export interface User {
  id:       string
  grupo_id: string
  nome:     string
  email:    string
  role:     'admin_global' | 'admin_grupo' | 'viewer'
}

export interface GrupoInfo {
  id:          string
  nome:        string
  slug:        string
  schema_name: string
}

export const useAuthStore = defineStore('auth', () => {
  const accessToken  = ref(localStorage.getItem('access_token') || '')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')
  const user         = ref<User | null>(null)

  // Estado de seleção pendente de grupo (multi-grupo no login)
  const preAuthToken  = ref(localStorage.getItem('pre_auth_token') || '')
  const pendingGrupos = ref<GrupoInfo[]>(JSON.parse(localStorage.getItem('pending_grupos') || '[]'))
  const meusGrupos    = ref<GrupoInfo[]>(JSON.parse(localStorage.getItem('meus_grupos') || '[]'))

  const isAuthenticated  = computed(() => !!accessToken.value)
  const needsGroupSelect = computed(() => !!preAuthToken.value && !accessToken.value)
  const isAdminGlobal    = computed(() => user.value?.role === 'admin_global')
  const isAdminGrupo     = computed(() => user.value?.role === 'admin_grupo')
  const isViewer         = computed(() => user.value?.role === 'viewer')
  const isAdmin          = computed(() => ['admin_global', 'admin_grupo'].includes(user.value?.role ?? ''))

  function setTokens(access: string, refresh: string) {
    accessToken.value  = access
    refreshToken.value = refresh
    localStorage.setItem('access_token',  access)
    localStorage.setItem('refresh_token', refresh)
    preAuthToken.value  = ''
    pendingGrupos.value = []
    localStorage.removeItem('pre_auth_token')
    localStorage.removeItem('pending_grupos')
  }

  function clearTokens() {
    accessToken.value   = ''
    refreshToken.value  = ''
    preAuthToken.value  = ''
    pendingGrupos.value = []
    meusGrupos.value    = []
    user.value          = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem('pre_auth_token')
    localStorage.removeItem('pending_grupos')
    localStorage.removeItem('meus_grupos')
  }

  async function login(email: string, password: string) {
    const { data } = await api.post('/auth/login', { email, password })
    const resp = data.data

    if (resp.needs_select) {
      // Múltiplos grupos — salva estado de seleção pendente
      preAuthToken.value  = resp.pre_auth_token
      pendingGrupos.value = resp.grupos ?? []
      localStorage.setItem('pre_auth_token',  resp.pre_auth_token)
      localStorage.setItem('pending_grupos', JSON.stringify(resp.grupos ?? []))
      return
    }

    setTokens(resp.access_token, resp.refresh_token)
    await fetchMe()
    await refreshMeusGrupos()
  }

  async function selectGrupo(grupoID: string) {
    const { data } = await api.post('/auth/select-grupo', {
      pre_auth_token: preAuthToken.value,
      grupo_id: grupoID
    })
    setTokens(data.data.access_token, data.data.refresh_token)
    await fetchMe()
    await refreshMeusGrupos()
  }

  async function trocaGrupo(grupoID: string) {
    const { data } = await api.post('/auth/troca-grupo', { grupo_id: grupoID })
    setTokens(data.data.access_token, data.data.refresh_token)
    await fetchMe()
    await refreshMeusGrupos()
  }

  async function fetchGrupos(): Promise<GrupoInfo[]> {
    const { data } = await api.get('/auth/grupos')
    return data.data ?? []
  }

  async function refreshMeusGrupos() {
    try {
      const grupos = await fetchGrupos()
      meusGrupos.value = grupos
      localStorage.setItem('meus_grupos', JSON.stringify(grupos))
    } catch {
      // silencioso — não crítico
    }
  }

  async function refresh() {
    const { data } = await api.post('/auth/refresh', {
      refresh_token: refreshToken.value
    })
    setTokens(data.data.access_token, data.data.refresh_token)
  }

  async function logout() {
    try {
      await api.post('/auth/logout', { refresh_token: refreshToken.value })
    } finally {
      clearTokens()
    }
  }

  async function fetchMe() {
    const { data } = await api.get('/auth/me')
    user.value = data.data
  }

  // Chamado no startup do App para restaurar o estado do usuário a partir do token salvo.
  async function init() {
    if (!accessToken.value) return
    if (user.value) return
    try {
      await fetchMe()
      await refreshMeusGrupos()
    } catch {
      clearTokens()
    }
  }

  return {
    accessToken, refreshToken, user, preAuthToken, pendingGrupos, meusGrupos,
    isAuthenticated, needsGroupSelect, isAdminGlobal, isAdminGrupo, isViewer, isAdmin,
    login, selectGrupo, trocaGrupo, fetchGrupos, refreshMeusGrupos,
    logout, refresh, fetchMe, init, clearTokens, setTokens
  }
})
