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
}

export async function fetchDashboard(grupoID: string, params: DashboardParams = {}): Promise<DashboardData> {
  const query: Record<string, string> = {}
  if (params.ano)              query.ano              = String(params.ano)
  if (params.empresas?.length)        query.empresas        = params.empresas.join(',')
  if (params.contas_correntes?.length) query.contas_correntes = params.contas_correntes.join(',')
  if (params.departamentos?.length)   query.departamentos   = params.departamentos.join(',')
  if (params.categorias?.length)      query.categorias      = params.categorias.join(',')
  if (params.cliente)          query.cliente          = params.cliente

  const { data } = await api.get(`/dados/${grupoID}/dashboard`, { params: query })
  return data.data as DashboardData
}
