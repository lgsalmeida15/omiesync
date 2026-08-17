/**
 * Paleta categórica dos gráficos de composição.
 *
 * Começa no ciano de destaque da aplicação e segue por matizes bem separados,
 * para que fatias vizinhas do donut não se confundam. Doze cores porque acima
 * disso a distinção visual deixa de ser confiável; passando disso a paleta se
 * repete, e a legenda lateral é que desfaz a ambiguidade.
 */
export const PALETA = [
  '#00e5ff', '#22c55e', '#f59e0b', '#a855f7', '#ef4444', '#3b82f6',
  '#14b8a6', '#ec4899', '#84cc16', '#f97316', '#6366f1', '#eab308',
] as const

export function corDe(indice: number): string {
  return PALETA[indice % PALETA.length]
}

/** Mesma cor com alfa, para preenchimentos. */
export function corDeAlpha(indice: number, alpha: number): string {
  const hex = corDe(indice)
  const n = parseInt(hex.slice(1), 16)
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`
}
