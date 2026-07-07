import api from './client'

export interface SyncJob {
  id:            string
  empresa_id:    string
  tipo:          string
  status:        'pendente' | 'rodando' | 'concluido' | 'erro'
  erro:          string
  iniciado_at:   string | null
  concluido_at:  string | null
  created_at:    string
}

export interface SyncControl {
  id:              string
  empresa_id:      string
  ativo:           boolean
  intervalo_min:   number
  ultimo_sync_at:  string | null
  proximo_sync_at: string | null
}

export interface SyncStatus {
  empresa_id: string
  controle:   SyncControl | null
  ultimo_job: SyncJob | null
}

export const syncApi = {
  status: (empresaId: string) =>
    api.get(`/sync/${empresaId}/status`),

  jobs: (empresaId: string, params?: { page?: number; per_page?: number }) =>
    api.get(`/sync/${empresaId}/jobs`, { params }),

  forcar: (empresaId: string, tipo: 'manual' | 'full') =>
    api.post(`/sync/${empresaId}/forcar`, { tipo }),

  configurar: (empresaId: string, payload: { ativo: boolean; intervalo_min: number }) =>
    api.put(`/sync/${empresaId}/configurar`, payload)
}
