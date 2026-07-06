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

export const useAuthStore = defineStore('auth', () => {
  const accessToken  = ref(localStorage.getItem('access_token') || '')
  const refreshToken = ref(localStorage.getItem('refresh_token') || '')
  const user         = ref<User | null>(null)

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdminGlobal   = computed(() => user.value?.role === 'admin_global')
  const isAdminGrupo    = computed(() => user.value?.role === 'admin_grupo')
  const isViewer        = computed(() => user.value?.role === 'viewer')
  const isAdmin         = computed(() => ['admin_global', 'admin_grupo'].includes(user.value?.role ?? ''))

  function setTokens(access: string, refresh: string) {
    accessToken.value  = access
    refreshToken.value = refresh
    localStorage.setItem('access_token',  access)
    localStorage.setItem('refresh_token', refresh)
  }

  function clearTokens() {
    accessToken.value  = ''
    refreshToken.value = ''
    user.value         = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  }

  async function login(email: string, password: string) {
    const { data } = await api.post('/auth/login', { email, password })
    setTokens(data.data.access_token, data.data.refresh_token)
    await fetchMe()
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

  return {
    accessToken, refreshToken, user,
    isAuthenticated, isAdminGlobal, isAdminGrupo, isViewer, isAdmin,
    login, logout, refresh, fetchMe, clearTokens, setTokens
  }
})
