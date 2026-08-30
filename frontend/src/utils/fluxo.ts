import type { FluxoTransacao } from '@/api/fluxocaixa'

export interface FiltrosListagem {
  /** null = mês inteiro. */
  dia: number | null
  /** '' = todos. */
  tipo: string
  /** '' = todas; 'efetuada' | 'pendente' | 'atrasada'. */
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
    (!f.situacao || combinaSituacao(t, f.situacao)) &&
    (!q || t.descricao.toLowerCase().includes(q) || t.categoria.toLowerCase().includes(q))
  )
}

/**
 * Atrasado é um caso de `pendente`: nada foi pago. Precisa de opcao propria
 * porque, misturado com o que apenas ainda nao venceu, o vencido desaparece —
 * e ele e o que exige acao.
 */
function combinaSituacao(t: FluxoTransacao, situacao: string): boolean {
  if (situacao === 'atrasada') return t.status === 'Atrasado'
  return (situacao === 'efetuada') === t.realizado
}

/**
 * Classe da pill de status: verde recebido, vermelho pago, âmbar pendente e
 * vermelho forte para atrasado.
 *
 * Atrasado e testado ANTES de `realizado`: sem isso ele cairia em pendente e
 * ficaria visualmente indistinguivel de um titulo que ainda nem venceu.
 */
export function classeStatus(t: Pick<FluxoTransacao, 'tipo' | 'realizado' | 'status'>): string {
  if (t.status === 'Atrasado') return 'pill--atraso'
  if (!t.realizado) return 'pill--pend'
  return t.tipo === 'receita' ? 'pill--in' : 'pill--out'
}
