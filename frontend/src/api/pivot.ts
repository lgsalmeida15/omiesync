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

  /**
   * Fechamento de cada mês: a âncora mais o resultado de todos os meses até ali.
   * É a mesma escada que a Visão Geral desenha, na tabela.
   */
  acumulado: number[]
  /** Abertura de janeiro — o saldo das contas no começo do período. */
  saldo_inicial: number
  /**
   * Falso quando há filtro que o saldo das contas não sabe responder (categoria,
   * departamento, cliente). Aí a escada parte do zero e passa a ser o acumulado
   * DO RECORTE — e a tela precisa dizer isso, em vez de apresentar o número como
   * posição de caixa.
   */
  ancora_aplicavel: boolean
}

export async function fetchPivot(grupoID: string, params: DashboardParams = {}): Promise<PivotData> {
  const { data } = await api.get(`/dados/${grupoID}/pivot`, { params: paramsToQuery(params) })
  return data.data as PivotData
}
