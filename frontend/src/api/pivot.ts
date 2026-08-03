import api from './client'
import type { DashboardParams } from './dashboard'

export interface PivotLinha {
  tipo: string               // "receita" | "despesa" | "nao classificado"
  categoria_superior: string
  categoria_final: string
  cliente: string
  meses: number[]            // 12 posições, Jan..Dez
  total: number
}

export interface PivotData {
  ano: number
  linhas: PivotLinha[]
  totais_mes: number[]
  total_geral: number
  /** Primeiro mês projetado (1-12). 13 = ano inteiro realizado. */
  mes_corte: number
}

export async function fetchPivot(grupoID: string, params: DashboardParams = {}): Promise<PivotData> {
  const query: Record<string, string> = {}
  if (params.ano)                      query.ano              = String(params.ano)
  if (params.empresas?.length)         query.empresas         = params.empresas.join(',')
  if (params.contas_correntes?.length) query.contas_correntes = params.contas_correntes.join(',')
  if (params.departamentos?.length)    query.departamentos    = params.departamentos.join(',')
  if (params.categorias?.length)       query.categorias       = params.categorias.join(',')
  if (params.cliente)                  query.cliente          = params.cliente

  const { data } = await api.get(`/dados/${grupoID}/pivot`, { params: query })
  return data.data as PivotData
}
