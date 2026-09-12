import type { GraficoAcumulado } from '@/api/dashboard'
import { TODOS_OS_MESES } from '@/utils/visaogeral'

/**
 * Cascata (waterfall) do resultado acumulado.
 *
 * Substitui as barras ancoradas no zero mais a linha de acumulado: ali o
 * acumulado era uma segunda série sobreposta, e a relação entre um mês e o
 * seguinte ficava por conta do leitor. Na cascata, cada mês PARTE de onde o
 * anterior parou — a escada é o acumulado.
 *
 * Abre com o saldo em contas e fecha num Total, que é o saldo projetado ao fim
 * do período exibido.
 *
 * Módulo próprio e puro porque a aritmética de base e topo de cada barra é fácil
 * de errar e impossível de conferir olhando o canvas: um degrau fora do lugar
 * ainda parece um gráfico correto. Em teste, cada passo é uma asserção.
 */

export type TipoPasso = 'abertura' | 'aumento' | 'reducao' | 'total'

export interface PassoCascata {
  rotulo: string
  /** Base e topo da barra flutuante. No Chart.js, `data: [[de, ate]]`. */
  de: number
  ate: number
  /** Valor do passo: o saldo, no caso da abertura e do total; a variação nos meses. */
  valor: number
  tipo: TipoPasso
}

export function montarCascata(
  saldoInicial: number,
  meses: GraficoAcumulado[],
  mesesVisiveis: Set<number> = TODOS_OS_MESES,
  rotuloAbertura = 'Saldo',
  rotuloTotal = 'Total',
): PassoCascata[] {
  const passos: PassoCascata[] = [{
    rotulo: rotuloAbertura,
    de: 0, ate: saldoInicial,
    valor: saldoInicial,
    tipo: 'abertura',
  }]

  let acc = saldoInicial
  for (const m of meses) {
    if (!mesesVisiveis.has(m.mes)) continue
    // Mês sem movimento vira um degrau de altura zero — um traço no nível atual,
    // que é informação (o mês existiu e não mexeu no saldo), não ruído.
    const de = acc
    acc += m.resultado_mes
    passos.push({
      rotulo: m.mes_nome,
      de, ate: acc,
      valor: m.resultado_mes,
      tipo: m.resultado_mes < 0 ? 'reducao' : 'aumento',
    })
  }

  // O total é ancorado no zero, e não flutuante: é uma posição absoluta, não uma
  // variação. É o que a barra destacada representa num waterfall.
  passos.push({
    rotulo: rotuloTotal,
    de: 0, ate: acc,
    valor: acc,
    tipo: 'total',
  })

  return passos
}

/**
 * O número que o passo exibe: a POSIÇÃO na abertura e no total, a VARIAÇÃO nos
 * meses. O rótulo e o lado em que ele é desenhado saem os dois daqui — senão um
 * diria uma coisa e o outro se posicionaria por outra.
 */
export function valorDoPasso(p: PassoCascata): number {
  return p.tipo === 'abertura' || p.tipo === 'total' ? p.ate : p.valor
}

/**
 * De que lado da barra o rótulo fica: acima quando o valor é positivo, abaixo
 * quando é negativo. Nunca dentro.
 *
 * A regra era pelo TIPO do passo — 'reducao' embaixo, o resto em cima — e
 * errava nos dois casos em que tipo e sinal discordam: saldo de abertura
 * negativo e total negativo. Nesses, a barra desce a partir do zero, a âncora
 * do rótulo fica na ponta de baixo, e mandá-lo "para cima" o jogava para dentro
 * da própria barra, na mesma cor dela. Pelo SINAL, a conta fecha sozinha.
 */
export function alinhamentoDoPasso(p: PassoCascata): 'top' | 'bottom' {
  return valorDoPasso(p) < 0 ? 'bottom' : 'top'
}
