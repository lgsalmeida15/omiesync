import api from './client'

export type Recurso = 'dashboard' | 'sync' | 'admin'
export type Acao    = 'ver' | 'editar' | 'forcar_sync'

export interface Permissao {
  id:         string
  usuario_id: string
  empresa_id: string
  recurso:    Recurso
  acao:       Acao
  created_at: string
}

export const permissoesApi = {
  listByUsuario: (usuarioId: string) =>
    api.get(`/admin/permissoes/usuario/${usuarioId}`),

  listByEmpresa: (empresaId: string) =>
    api.get(`/admin/permissoes/empresa/${empresaId}`),

  grant: (payload: {
    usuario_id: string; empresa_id: string; recurso: Recurso; acao: Acao
  }) => api.post('/admin/permissoes/grant', payload),

  revoke: (payload: {
    usuario_id: string; empresa_id: string; recurso: Recurso; acao: Acao
  }) => api.post('/admin/permissoes/revoke', payload)
}
