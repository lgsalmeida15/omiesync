import api from './client'

export interface Empresa {
  id:            string
  grupo_id:      string
  nome:          string
  cnpj:          string
  app_key:       string
  app_secret:    string   // sempre mascarado pela API (ex: "key1****")
  status:        'ativa' | 'inativa' | 'deletando'
  status_sync:   'ativo' | 'pausado' | 'erro' | 'deletando'
  ultimo_sync_at: string | null
  created_at:    string
  updated_at:    string
}

export const empresasApi = {
  list: (grupoId: string, params?: { page?: number; per_page?: number }) =>
    api.get(`/admin/grupos/${grupoId}/empresas`, { params }),

  get: (grupoId: string, empresaId: string) =>
    api.get(`/admin/grupos/${grupoId}/empresas/${empresaId}`),

  create: (grupoId: string, payload: {
    nome: string; cnpj?: string; app_key: string; app_secret: string
  }) => api.post(`/admin/grupos/${grupoId}/empresas`, payload),

  update: (grupoId: string, empresaId: string, payload: {
    nome: string; cnpj?: string; app_key: string; app_secret: string
  }) => api.put(`/admin/grupos/${grupoId}/empresas/${empresaId}`, payload),

  delete: (grupoId: string, empresaId: string) =>
    api.delete(`/admin/grupos/${grupoId}/empresas/${empresaId}`),

  reativar: (grupoId: string, empresaId: string) =>
    api.post(`/admin/grupos/${grupoId}/empresas/${empresaId}/reativar`)
}
