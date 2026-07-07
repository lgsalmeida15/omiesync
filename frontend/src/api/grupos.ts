import api from './client'

export interface Grupo {
  id:          string
  nome:        string
  slug:        string
  schema_name: string
  status:      'ativo' | 'inativo' | 'deletando'
  created_at:  string
  updated_at:  string
}

export const gruposApi = {
  list: (params?: { page?: number; per_page?: number }) =>
    api.get('/admin/grupos', { params }),

  get: (id: string) =>
    api.get(`/admin/grupos/${id}`),

  create: (payload: { nome: string; slug: string }) =>
    api.post('/admin/grupos', payload),

  update: (id: string, payload: { nome: string }) =>
    api.put(`/admin/grupos/${id}`, payload),

  delete: (id: string) =>
    api.delete(`/admin/grupos/${id}`)
}
