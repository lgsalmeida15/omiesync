import api from './client'

export interface DashboardCards {
  receita_total: number
  despesa_total: number
  resultado: number
  saldo_contas_correntes: number
}

export interface GraficoMensal {
  mes: number
  mes_nome: string
  receita: number
  despesa: number
  resultado_mes: number
}

export interface GraficoAcumulado {
  mes: number
  mes_nome: string
  resultado_mes: number
  acumulado: number
}

export interface ContaCorrenteItem {
  codigo: string
  descricao: string
}

export interface EmpresaItem {
  id: string
  nome: string
}

export interface FiltrosDisponiveis {
  contas_correntes: ContaCorrenteItem[]
  departamentos: string[]
  categorias: string[]
  clientes: string[]
  empresas: EmpresaItem[]
}

export interface DashboardData {
  cards: DashboardCards
  grafico_mensal: GraficoMensal[]
  grafico_resultado_acumulado: GraficoAcumulado[]
  filtros_disponiveis: FiltrosDisponiveis
}

export interface DashboardParams {
  ano?: number
  empresas?: string[]
  contas_correntes?: string[]
  departamentos?: string[]
  categorias?: string[]
  cliente?: string
  /**
   * Lista de EXCLUSÃO, não de inclusão. Com inclusão, uma categoria nova no Omie
   * ficaria fora dos números em silêncio até alguém marcá-la.
   */
  categorias_excluir?: string[]
}

/** Serializa os filtros para query string. Compartilhado por dashboard, pivot e filtros. */
export function paramsToQuery(params: DashboardParams): Record<string, string> {
  const q: Record<string, string> = {}
  if (params.ano)                       q.ano                = String(params.ano)
  if (params.empresas?.length)          q.empresas           = params.empresas.join(',')
  if (params.contas_correntes?.length)  q.contas_correntes   = params.contas_correntes.join(',')
  if (params.departamentos?.length)     q.departamentos      = params.departamentos.join(',')
  if (params.categorias?.length)        q.categorias         = params.categorias.join(',')
  if (params.cliente)                   q.cliente            = params.cliente
  if (params.categorias_excluir?.length) q.categorias_excluir = params.categorias_excluir.join(',')
  return q
}

/** Só as opções de filtro, em cascata. Bem mais leve que o dashboard completo. */
export async function fetchFiltros(grupoID: string, params: DashboardParams = {}): Promise<FiltrosDisponiveis> {
  const { data } = await api.get(`/dados/${grupoID}/filtros`, { params: paramsToQuery(params) })
  return data.data as FiltrosDisponiveis
}

export async function fetchDashboard(grupoID: string, params: DashboardParams = {}): Promise<DashboardData> {
  const { data } = await api.get(`/dados/${grupoID}/dashboard`, { params: paramsToQuery(params) })
  return data.data as DashboardData
}
