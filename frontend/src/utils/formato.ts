/**
 * Formatação de valores financeiros em notação contábil.
 *
 * Negativo aparece entre parênteses — (1.234) e não -1.234 — que é a convenção de
 * relatório financeiro. O sinal de menos se perde com facilidade na leitura rápida
 * de uma tabela ou de um rótulo de gráfico; o parêntese não.
 *
 * Vive num módulo próprio porque dashboard e aba Resultado precisam do mesmo
 * comportamento: duplicar levaria as duas telas a divergir na apresentação do mesmo
 * número.
 */

/** Moeda sem centavos, para cards e tooltips. Ex: (R$ 1.234) */
export function fmtMoeda(v: number): string {
  const s = new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
    maximumFractionDigits: 0,
  }).format(Math.abs(v))
  return v < 0 ? `(${s})` : s
}

/**
 * Moeda com centavos, para os cards de KPI. Ex: (R$ 490.000,97)
 *
 * Separado de fmtMoeda porque os dois têm públicos diferentes: no gráfico e no
 * tooltip o centavo é ruído, no card de topo ele é o número que o usuário
 * confere contra o extrato.
 *
 * `casas` permite suprimir os centavos quando o usuário desliga a exibição —
 * há empresas em que o centavo é irrelevante e só atrapalha a leitura. O padrão
 * mantém o comportamento anterior, então nenhum chamador precisa mudar.
 *
 * Arredonda, não trunca: Intl já faz isso, e 490.000,97 vira 490.001.
 */
export function fmtMoedaExata(v: number, casas: 0 | 2 = 2): string {
  const s = new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
    minimumFractionDigits: casas,
    maximumFractionDigits: casas,
  }).format(Math.abs(v))
  return v < 0 ? `(${s})` : s
}

/**
 * Número sem símbolo de moeda, para a tabela do Resultado.
 *
 * Mesma regra de `casas` do fmtMoedaExata. Estas duas são as únicas funções que
 * exibem centavos na aplicação — fmtMoeda e fmtCompacto já arredondam sempre —,
 * por isso o botão de decimais atua só nelas.
 */
export function fmtNumero(v: number, casas: 0 | 2 = 2): string {
  const s = Math.abs(v).toLocaleString('pt-BR', {
    minimumFractionDigits: casas,
    maximumFractionDigits: casas,
  })
  return v < 0 ? `(${s})` : s
}

/**
 * Forma compacta para eixos e rótulos de gráfico, onde não cabe o valor inteiro.
 * Zero devolve string vazia: rótulo "R$ 0" em cima de barra sem altura só suja.
 */
export function fmtCompacto(v: number): string {
  const abs = Math.abs(v)
  if (abs === 0) return ''

  let s: string
  if (abs >= 1_000_000) s = `R$${(abs / 1_000_000).toFixed(1)}M`
  else if (abs >= 1_000) s = `R$${(abs / 1_000).toFixed(0)}K`
  else return fmtMoeda(v)

  return v < 0 ? `(${s})` : s
}
