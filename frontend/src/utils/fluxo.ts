import type { FluxoTransacao } from '@/api/fluxocaixa'

export interface FiltrosListagem {
  /** null = mês inteiro. */
  dia: number | null
  /** '' = todos. */
  tipo: string
  /** '' = todas; 'efetuada' | 'pendente'. */
  situacao: string
  busca: string
}

/**
 * Filtra a listagem de transações. Extraído do componente para ser testável:
 * a combinação dia × tipo × situação × busca é a regra mais fácil de quebrar
 * sem que a tela acuse nada.
 *
 * A ordem de entrada é preservada — o servidor já devolve ordenado por data.
 */
export function filtrarTransacoes(
  transacoes: FluxoTransacao[],
  f: FiltrosListagem,
): FluxoTransacao[] {
  const q = f.busca.trim().toLowerCase()
  return transacoes.filter(t =>
    (f.dia === null || t.dia === f.dia) &&
    (!f.tipo || t.tipo === f.tipo) &&
    (!f.situacao || (f.situacao === 'efetuada') === t.realizado) &&
    (!q || t.descricao.toLowerCase().includes(q) || t.categoria.toLowerCase().includes(q))
  )
}

/** Classe da pill de status: verde recebido, vermelho pago, âmbar pendente. */
export function classeStatus(t: Pick<FluxoTransacao, 'tipo' | 'realizado'>): string {
  if (!t.realizado) return 'pill--pend'
  return t.tipo === 'receita' ? 'pill--in' : 'pill--out'
}
