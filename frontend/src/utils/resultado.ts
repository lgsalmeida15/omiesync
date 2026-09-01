import type { PivotLinha } from '@/api/pivot'

/**
 * Recalcula a linha RESULTADO do pivô excluindo categorias superiores.
 *
 * O backend já envia `resultado_mes` e `resultado_total` prontos, e havia um bom
 * motivo para isso: somar no servidor mantém o arredondamento igual ao dos cards.
 * Passou a ser insuficiente porque o usuário desmarca categorias na própria tela,
 * e o servidor não sabe da escolha.
 *
 * Por isso a soma aqui replica a do servidor linha a linha: receita soma, despesa
 * subtrai, ambas chegando como magnitude positiva. Com o conjunto de exclusões
 * vazio o resultado tem de bater com o do backend — é o que o teste garante, e é o
 * que impede este cálculo de derivar do outro com o tempo.
 */

/**
 * Chave de exclusão. O nome da superior sozinho não serve: a mesma categoria pode
 * existir sob receita e sob despesa, e desmarcar uma apagaria a outra junto.
 */
export function chaveSuperior(tipo: string, categoriaSuperior: string): string {
  return `${tipo}|${categoriaSuperior}`
}

export interface ResultadoCalculado {
  meses: number[]
  total: number
}

export function calcularResultado(
  linhas: PivotLinha[],
  excluidos: Set<string> = new Set(),
): ResultadoCalculado {
  const meses = Array(12).fill(0)
  let total = 0

  for (const l of linhas) {
    if (excluidos.has(chaveSuperior(l.tipo, l.categoria_superior))) continue

    // Mesma regra do servidor: o que não é receita subtrai. O ELSE do
    // ajuste_receita_despesa manda o não classificado para despesa, e replicar
    // isso é o que mantém os dois números iguais.
    const sinal = l.tipo === 'receita' ? 1 : -1
    for (let i = 0; i < 12; i++) meses[i] += sinal * l.meses[i]
    total += sinal * l.total
  }

  return { meses, total }
}

/**
 * Largura de coluna durante o arrasto.
 *
 * O mínimo não é cosmético: sem ele, arrastar para a esquerda além da origem
 * produz largura negativa, e a coluna some sem meio de recuperá-la a não ser
 * recarregando.
 */
export function larguraAoArrastar(inicial: number, dx: number, minimo = 60): number {
  return Math.max(minimo, Math.round(inicial + dx))
}
