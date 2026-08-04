import api from './client'
import { paramsToQuery, type DashboardParams } from './dashboard'

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
  const { data } = await api.get(`/dados/${grupoID}/pivot`, { params: paramsToQuery(params) })
  return data.data as PivotData
}
