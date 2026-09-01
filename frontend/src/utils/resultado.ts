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

/**
 * Altura que sobra na janela para a tabela, a partir de onde ela começa.
 *
 * Substituiu um `calc(100vh - 240px)` fixo. O 240 era uma estimativa do que fica
 * acima, mas isso varia em três eixos independentes — barra de filtros aberta ou
 * fechada, modo foco (que remove a faixa de abas) e controles recolhidos. No modo
 * foco o número fixo desperdiçava quase 80px justamente onde o usuário quer área;
 * com os filtros abertos, estourava a tela.
 *
 * `folga` é o respiro abaixo da tabela — o padding inferior da página. Sem ele a
 * tabela encosta na borda e some a sensação de fim de conteúdo.
 *
 * O mínimo evita que uma janela baixa, ou um topo grande demais, produza altura
 * negativa e faça a tabela desaparecer.
 */
export function alturaDisponivel(
  topo: number,
  alturaJanela: number,
  folga = 32,
  minimo = 320,
): number {
  return Math.max(minimo, Math.round(alturaJanela - topo - folga))
}
