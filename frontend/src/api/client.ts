import axios from 'axios'

// Nao importar router aqui -- causaria dependencia circular:
// client.ts -> router -> stores/auth -> client.ts
// Redirect para /login feito via window.location.

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? 'http://localhost:8080',
  headers: { 'Content-Type': 'application/json' }
})

// Injeta Bearer token em todas as requests
api.interceptors.request.use(config => {
  const token = localStorage.getItem('access_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// Renova token em 401 com fila de requests pendentes
let isRefreshing = false
let failedQueue: Array<{ resolve: (v: string) => void; reject: (e: unknown) => void }> = []

function processQueue(error: unknown, token: string | null) {
  failedQueue.forEach(p => (error ? p.reject(error) : p.resolve(token!)))
  failedQueue = []
}

function redirectToLogin() {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  if (!window.location.pathname.includes('/login')) {
    window.location.href = '/login'
  }
}

api.interceptors.response.use(
  res => res,
  async error => {
    const original = error.config
    // Rotas de autenticação puras (login/refresh/logout) não devem ser reprocessadas.
    // /auth/me, porém, deve tentar o refresh normalmente para renovar a sessão.
    const isAuthMutation = ['/auth/login', '/auth/refresh', '/auth/logout']
      .some(p => original.url?.includes(p))

    if (error.response?.status === 401 && !original._retry && !isAuthMutation) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        }).then(token => {
          original.headers.Authorization = `Bearer ${token}`
          return api(original)
        })
      }
      original._retry = true
      isRefreshing = true
      const refreshToken = localStorage.getItem('refresh_token')
      if (!refreshToken) {
        redirectToLogin()
        return Promise.reject(error)
      }
      try {
        const { data } = await api.post('/auth/refresh', { refresh_token: refreshToken })
        const newToken = data.data.access_token
        localStorage.setItem('access_token', newToken)
        localStorage.setItem('refresh_token', data.data.refresh_token)
        api.defaults.headers.common.Authorization = `Bearer ${newToken}`
        processQueue(null, newToken)
        original.headers.Authorization = `Bearer ${newToken}`
        return api(original)
      } catch (refreshError) {
        processQueue(refreshError, null)
        redirectToLogin()
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }
    return Promise.reject(error)
  }
)

export default api
