import type { FluxoTransacao, FluxoResumo } from '@/api/fluxocaixa'

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
export function classeStatus(
  t: Pick<FluxoTransacao, 'tipo' | 'realizado' | 'status'> & Partial<Pick<FluxoTransacao, 'categoria'>>,
): string {
  if (t.status === 'Atrasado') return 'pill--atraso'
  if (!t.realizado) return 'pill--pend'
  return `pill--${classeTipo(t)}`
}

/**
 * Rótulo e classe da coluna TIPO da listagem. Transferência não é recebimento
 * nem pagamento, mas a direção (entrada/saída) ainda importa para quem lê a
 * lista — por isso olha `categoria`, que a matvw já sintetiza por direção (ver
 * provisioner.go).
 */
export function rotuloTipo(t: Pick<FluxoTransacao, 'tipo' | 'categoria'>): string {
  if (t.tipo === 'transferencia') {
    return t.categoria === 'Entrada de Transferência' ? 'Transf. entrada' : 'Transf. saída'
  }
  return t.tipo === 'receita' ? 'Recebimento' : 'Pagamento'
}

export function classeTipo(t: Pick<FluxoTransacao, 'tipo'>): string {
  if (t.tipo === 'transferencia') return 'transf'
  return t.tipo === 'receita' ? 'in' : 'out'
}

/**
 * Totaliza transações no mesmo formato do resumo do servidor.
 *
 * Existe porque a tela recalcula o resumo em dois casos — aba de um lado só e
 * dia selecionado no calendário — e a regra estava escrita à mão nos dois. Com
 * a chegada do "Atrasado", os dois teriam de mudar juntos, e esquecer um faria
 * o vencido voltar a se esconder dentro do previsto justamente nas abas de
 * Contas a Pagar e a Receber, que é onde ele importa.
 *
 * A ordem dos ramos é a mesma do backend: atrasado ANTES de realizado, porque
 * ele também tem `realizado: false`.
 *
 * `somarUmLadoSo` cobre a aba de um lado só, onde tudo tem o mesmo sinal e o
 * total é soma, não diferença — subtrair ali não teria contra o quê.
 */
export function resumirTransacoes(
  transacoes: FluxoTransacao[],
  somarUmLadoSo = false,
): FluxoResumo {
  const r: FluxoResumo = {
    recebido: 0, a_receber: 0, pago: 0, a_pagar: 0,
    atrasado_receber: 0, atrasado_pagar: 0,
    transferido_entrada: 0, transferido_saida: 0, resultado: 0,
  }

  for (const t of transacoes) {
    // Transferência ANTES do receita/despesa: sem este ramo ela cairia em
    // pago/a_pagar (nenhuma das duas é 'receita'), voltando a se misturar com
    // despesa — o mesmo problema que esta mudança resolveu no servidor.
    if (t.tipo === 'transferencia') {
      t.categoria === 'Entrada de Transferência'
        ? (r.transferido_entrada += t.valor)
        : (r.transferido_saida += t.valor)
      continue
    }
    const receita = t.tipo === 'receita'
    if (t.status === 'Atrasado') {
      receita ? (r.atrasado_receber += t.valor) : (r.atrasado_pagar += t.valor)
    } else if (t.realizado) {
      receita ? (r.recebido += t.valor) : (r.pago += t.valor)
    } else {
      receita ? (r.a_receber += t.valor) : (r.a_pagar += t.valor)
    }
  }

  const entradas = r.recebido + r.a_receber + r.atrasado_receber + r.transferido_entrada
  const saidas   = r.pago + r.a_pagar + r.atrasado_pagar + r.transferido_saida
  r.resultado = somarUmLadoSo ? entradas + saidas : entradas - saidas
  return r
}
