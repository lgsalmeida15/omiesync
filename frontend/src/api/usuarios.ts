import api from './client'

export type Role = 'admin_global' | 'admin_grupo' | 'viewer'

export interface Usuario {
  id:         string
  grupo_id:   string
  nome:       string
  email:      string
  role:       Role
  ativo:      boolean
  created_at: string
  updated_at: string
}

export const usuariosApi = {
  list: (grupoId: string, params?: { page?: number; per_page?: number }) =>
    api.get(`/admin/grupos/${grupoId}/usuarios`, { params }),

  get: (grupoId: string, usuarioId: string) =>
    api.get(`/admin/grupos/${grupoId}/usuarios/${usuarioId}`),

  create: (grupoId: string, payload: {
    nome: string; email: string; password: string; role: Role
  }) => api.post(`/admin/grupos/${grupoId}/usuarios`, payload),

  update: (grupoId: string, usuarioId: string, payload: {
    nome: string; role: Role; ativo: boolean
  }) => api.put(`/admin/grupos/${grupoId}/usuarios/${usuarioId}`, payload),

  updatePassword: (grupoId: string, usuarioId: string, payload: { password: string }) =>
    api.put(`/admin/grupos/${grupoId}/usuarios/${usuarioId}/password`, payload),

  delete: (grupoId: string, usuarioId: string) =>
    api.delete(`/admin/grupos/${grupoId}/usuarios/${usuarioId}`)
}
