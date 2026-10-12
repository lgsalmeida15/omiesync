import api from './client'

export interface DashboardCards {
  receita_total: number
  despesa_total: number
  resultado: number
  saldo_contas_correntes: number
  /** Soma dos saldos iniciais cadastrados, antes de qualquer movimento. É a
      abertura da cascata. */
  saldo_inicial: number
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
  /** Resultado do mês somando as duas trilhas — o que a cascata usa hoje. */
  resultado_mes: number
  acumulado: number

  /**
   * As duas trilhas do modelo do Omie, encadeadas:
   * `saldo_inicial(N) = acumulado(N-1)` e `acumulado(N) = saldo_inicial(N) + resultado(N)`.
   *
   * REALIZADO é dinheiro que entrou ou saiu da conta. PREVISTO é como o mês deve
   * fechar: o realizado dele MAIS as provisões — não apenas o que falta acontecer.
   */
  saldo_inicial_realizado: number
  resultado_realizado: number
  acumulado_realizado: number
  saldo_inicial_previsto: number
  resultado_previsto: number
  acumulado_previsto: number
}

/**
 * Saldo das contas correntes ao FIM de cada mês: saldo inicial cadastrado mais
 * os movimentos já realizados até ali. Só realizado — provisão do extrato daria
 * saldo previsto, e faria o saldo de dezembro parecer dinheiro em caixa hoje.
 */
export interface SaldoMes {
  mes: number
  mes_nome: string
  saldo: number
}

export interface ContaCorrenteItem {
  codigo: string
  descricao: string
  /**
   * Marca cFluxoCaixa do Omie: 'S' = considerada no fluxo de caixa,
   * 'N' = não considerada. Vazio = conta que o extrato ainda não sincronizou.
   */
  fluxo_caixa: string
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
  /** Os doze saldos de uma vez, para recortar meses sem ir ao servidor. */
  saldos_mensais: SaldoMes[]
  filtros_disponiveis: FiltrosDisponiveis
  /**
   * Se o acumulado parte do saldo das contas ou do zero. Falso quando há filtro
   * de categoria, departamento ou cliente — dimensões do lançamento, que um
   * saldo bancário não sabe responder.
   */
  ancora_aplicavel: boolean
  /** Primeiro mês projetado (1-12). 13 = ano inteiro realizado. */
  mes_corte: number
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

  /**
   * Inclui os títulos vencidos e não pagos no fluxo. Só as abas Contas a Pagar
   * e Contas a Receber enviam: a aba Fluxo de Caixa e o dashboard continuam
   * lendo apenas o realizado e o previsto, sem mudar de número.
   */
  inadimplencia?: boolean
  /**
   * Natureza do lançamento: 'realizado' é o que já entrou ou saiu da conta;
   * 'previsto' é provisão. Vazio ou os dois = sem recorte.
   */
  trilhas?: string[]
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
  // Só serializa quando ligado: ausente, o backend responde como antes de a
  // opção existir.
  if (params.inadimplencia)             q.inadimplencia      = 'true'
  // Vazio e "as duas marcadas" dão no mesmo: nenhum recorte. Não serializar
  // mantém a URL limpa e a resposta idêntica à de antes do filtro existir.
  if (params.trilhas?.length === 1)     q.trilhas            = params.trilhas.join(',')
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
