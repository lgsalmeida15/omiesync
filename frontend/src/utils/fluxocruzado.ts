import type { FluxoTransacao } from '@/api/fluxocaixa'

/**
 * Recorte cruzado da aba Fluxo de Caixa.
 *
 * A tela tem quatro consumidores do mesmo conjunto de transações — calendário,
 * gráfico de categorias, gráfico de clientes e o trio pivô/resumo/listagem — e
 * cada um precisa de um subconjunto DIFERENTE dos mesmos filtros. Escrito à mão
 * em cada lugar, isso divergiria na primeira manutenção e a tela passaria a
 * mostrar somas que não fecham entre si sem acusar erro nenhum.
 */

export type Dimensao = 'dias' | 'categorias' | 'entidades'

/** Conjunto vazio significa "tudo" — e não "nada". */
export interface Selecao {
  dias: Set<number>
  categorias: Set<string>
  entidades: Set<string>
}

export function selecaoVazia(): Selecao {
  return { dias: new Set(), categorias: new Set(), entidades: new Set() }
}

/** Há algum recorte ativo? Decide se o resumo vem do servidor ou é recalculado. */
export function temRecorte(sel: Selecao): boolean {
  return sel.dias.size > 0 || sel.categorias.size > 0 || sel.entidades.size > 0
}

/**
 * Aplica o recorte, opcionalmente ignorando uma das dimensões.
 *
 * `ignorar` é o coração da filtragem cruzada: um gráfico nunca é filtrado pela
 * própria dimensão. Sem isso, clicar numa categoria faria todas as outras
 * desaparecerem do donut, e o gráfico deixaria de servir como ponto de partida
 * para o clique seguinte — o usuário ficaria sem como voltar ou comparar.
 *
 * A ordem de entrada é preservada: o servidor já devolve ordenado por data.
 */
export function aplicarSelecao(
  transacoes: FluxoTransacao[],
  sel: Selecao,
  ignorar?: Dimensao,
): FluxoTransacao[] {
  const porDia = ignorar !== 'dias' && sel.dias.size > 0
  const porCat = ignorar !== 'categorias' && sel.categorias.size > 0
  const porEnt = ignorar !== 'entidades' && sel.entidades.size > 0
  if (!porDia && !porCat && !porEnt) return transacoes

  return transacoes.filter(
    t =>
      (!porDia || sel.dias.has(t.dia)) &&
      (!porCat || sel.categorias.has(t.categoria)) &&
      (!porEnt || sel.entidades.has(t.descricao)),
  )
}

/**
 * Clique num período numerado: um dia no calendário do Fluxo de Caixa, um mês na
 * barra do gráfico da Visão Geral. Chamava-se `alternarDia` enquanto só existia
 * o calendário; o nome genérico veio quando o segundo caso apareceu, para não
 * ter duas cópias da mesma regra divergindo.
 *
 * Sem Ctrl o clique SUBSTITUI a seleção, e reclicar o único marcado limpa — é o
 * comportamento de toggle que o calendário já tinha, e mantê-lo evita que quem
 * navega item a item precise aprender uma mecânica nova.
 *
 * Com Ctrl (ou ⌘) adiciona e remove, permitindo comparar dois períodos distantes
 * sem passar por todos os que estão entre eles.
 *
 * Devolve um Set novo: mutar o recebido não dispararia a reatividade do Vue.
 */
export function alternarNumero(atual: Set<number>, n: number, comCtrl: boolean): Set<number> {
  if (comCtrl) {
    const proximo = new Set(atual)
    proximo.has(n) ? proximo.delete(n) : proximo.add(n)
    return proximo
  }
  const soEle = atual.size === 1 && atual.has(n)
  return soEle ? new Set() : new Set([n])
}

/**
 * Clique num rótulo de gráfico (categoria ou cliente). Sempre acumula: aqui não
 * existe o "navegar item a item" do calendário, e exigir Ctrl para somar uma
 * segunda categoria só criaria um atrito sem ganho.
 */
export function alternarRotulo(atual: Set<string>, rotulo: string): Set<string> {
  const proximo = new Set(atual)
  proximo.has(rotulo) ? proximo.delete(rotulo) : proximo.add(rotulo)
  return proximo
}
