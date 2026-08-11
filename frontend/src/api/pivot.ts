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
  /**
   * Receita MENOS despesa de cada mês — não a soma das linhas. Não inclui o saldo
   * das contas correntes, então difere do card RESULTADO da Visão Geral pelo caixa
   * de abertura.
   */
  resultado_mes: number[]
  resultado_total: number
  /** Primeiro mês projetado (1-12). 13 = ano inteiro realizado. */
  mes_corte: number
}

export async function fetchPivot(grupoID: string, params: DashboardParams = {}): Promise<PivotData> {
  const { data } = await api.get(`/dados/${grupoID}/pivot`, { params: paramsToQuery(params) })
  return data.data as PivotData
}
