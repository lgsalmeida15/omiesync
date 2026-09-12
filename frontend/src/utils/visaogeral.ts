import type { GraficoMensal, SaldoMes } from '@/api/dashboard'

/**
 * Recorte de meses da aba Visão Geral.
 *
 * Os cards vinham prontos do servidor, somados sobre os doze meses. Com o
 * recorte de meses eles passam a ser recomputados aqui: o servidor já devolve a
 * série mensal inteira, e refazer a consulta a cada clique numa barra tornaria a
 * interação lenta demais para servir de análise.
 *
 * Puro e testado porque é aritmética que não dá para conferir olhando a tela —
 * um número errado continua parecendo um número.
 */

export interface CardsRecorte {
  receita: number
  despesa: number
  /** Receita menos despesa do recorte. NÃO inclui o saldo — ver cardResultado. */
  resultado: number
}

/** Conjunto com os doze meses. Padrão de `mesesVisiveis` para quem não recorta. */
export const TODOS_OS_MESES: Set<number> = new Set([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12])

export function cardsDoRecorte(
  mensal: GraficoMensal[],
  mesesVisiveis: Set<number> = TODOS_OS_MESES,
): CardsRecorte {
  let receita = 0
  let despesa = 0
  for (const m of mensal) {
    if (!mesesVisiveis.has(m.mes)) continue
    receita += m.receita
    despesa += m.despesa
  }
  return { receita, despesa, resultado: receita - despesa }
}

/**
 * Saldo do recorte: o do ÚLTIMO mês marcado.
 *
 * Saldo é posição num instante, não soma de um período — um conjunto de meses
 * não define um saldo, o fim do último define. Somar os saldos mensais daria um
 * número sem significado nenhum, e grande o bastante para parecer certo.
 *
 * Sem recorte, é o saldo de dezembro: saldo inicial mais todo o realizado do ano.
 * Lista vazia devolve 0 em vez de estourar — a tela pode renderizar antes de os
 * dados chegarem.
 */
export function saldoDoRecorte(
  saldos: SaldoMes[],
  mesesVisiveis: Set<number> = TODOS_OS_MESES,
): number {
  let escolhido: SaldoMes | undefined
  for (const s of saldos) {
    if (mesesVisiveis.has(s.mes)) escolhido = s
  }
  return escolhido?.saldo ?? 0
}

/**
 * Valor do card Resultado.
 *
 * Mantém a fórmula do servidor — receita − despesa + saldo (dashboard.go) —, que
 * mistura fluxo do período com posição de caixa e infla a margem exibida abaixo
 * do card. Está sinalizado ao usuário e aguarda decisão dele; replicar aqui, e
 * não corrigir por conta própria, é o que mantém o número idêntico ao de antes
 * quando todos os meses estão marcados.
 */
export function cardResultado(cards: CardsRecorte, saldo: number): number {
  return cards.resultado + saldo
}

/** Margem do resultado sobre a receita, em fração. Receita zero devolve null. */
export function margemDoRecorte(resultado: number, receita: number): number | null {
  return receita === 0 ? null : resultado / receita
}
