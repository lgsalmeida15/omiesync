import api from './client'
import { paramsToQuery, type DashboardParams } from './dashboard'

export interface FluxoTransacao {
  dia: number
  data: string          // DD/MM/YYYY
  descricao: string     // cliente_final
  tipo: 'receita' | 'despesa'
  categoria: string
  valor: number
  // 'Atrasado' vem dos titulos vencidos de contas_pagar/contas_receber, e nao da
  // view materializada — que so conhece realizado e previsto.
  status: 'Recebido' | 'Pago' | 'Pendente' | 'Atrasado'
  /** true = movimento realizado; false = provisão do extrato. */
  realizado: boolean
}

export interface FluxoResumo {
  recebido: number
  a_receber: number
  pago: number
  a_pagar: number
  resultado: number
}

export interface FluxoCaixaData {
  ano: number
  mes: number
  transacoes: FluxoTransacao[]
  resumo: FluxoResumo
  /** A partir de hoje, independente do mês selecionado. */
  proximos_vencimentos: FluxoTransacao[]
}

export async function fetchFluxoCaixa(
  grupoID: string,
  params: DashboardParams & { mes?: number } = {},
): Promise<FluxoCaixaData> {
  const query = paramsToQuery(params)
  if (params.mes) query.mes = String(params.mes)
  const { data } = await api.get(`/dados/${grupoID}/fluxo-caixa`, { params: query })
  return data.data as FluxoCaixaData
}
