import type { FluxoTransacao } from '@/api/fluxocaixa'

export interface Agregado {
  rotulo: string
  valor: number
}

/**
 * Soma os valores por rótulo e devolve ordenado do maior para o menor.
 *
 * Empates são desempatados pelo rótulo, senão a ordem oscilaria entre
 * recarregamentos e o gráfico pareceria instável sem motivo.
 */
function agrupar(
  transacoes: FluxoTransacao[],
  chave: (t: FluxoTransacao) => string,
): Agregado[] {
  const soma = new Map<string, number>()
  for (const t of transacoes) {
    const k = chave(t)
    soma.set(k, (soma.get(k) ?? 0) + t.valor)
  }
  return [...soma.entries()]
    .map(([rotulo, valor]) => ({ rotulo, valor }))
    .sort((a, b) => b.valor - a.valor || a.rotulo.localeCompare(b.rotulo))
}

/** Valores somados por categoria final. */
export function porCategoria(transacoes: FluxoTransacao[]): Agregado[] {
  return agrupar(transacoes, t => t.categoria)
}

/**
 * Valores somados por entidade (cliente nos recebimentos, fornecedor nos
 * pagamentos), limitados aos `limite` maiores.
 *
 * O corte é aplicado depois da soma: quem tem muitos lançamentos pequenos
 * compete pelo total, não pela quantidade.
 */
export function topEntidades(transacoes: FluxoTransacao[], limite = 10): Agregado[] {
  return agrupar(transacoes, t => t.descricao).slice(0, limite)
}

/** Total dos itens cujo rótulo está marcado. */
export function totalMarcados(itens: Agregado[], marcados: Set<string>): number {
  return itens.reduce((s, i) => (marcados.has(i.rotulo) ? s + i.valor : s), 0)
}
