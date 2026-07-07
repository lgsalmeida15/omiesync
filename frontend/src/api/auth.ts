import api from './client'

export const authApi = {
  login:   (email: string, password: string) =>
    api.post('/auth/login', { email, password }),

  logout:  (refreshToken: string) =>
    api.post('/auth/logout', { refresh_token: refreshToken }),

  refresh: (refreshToken: string) =>
    api.post('/auth/refresh', { refresh_token: refreshToken }),

  me: () =>
    api.get('/auth/me')
}
