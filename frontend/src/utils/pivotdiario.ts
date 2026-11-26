import type { FluxoTransacao } from '@/api/fluxocaixa'
import { chaveSuperior } from '@/utils/resultado'

/**
 * Pivô categoria × dia do mês, derivado das transações que a aba Fluxo de Caixa
 * já carregou.
 *
 * Os quatro níveis são os mesmos da aba Resultado — tipo → categoria superior →
 * categoria final → cliente —, e a linha RESULTADO segue a mesma regra: receita
 * soma, despesa subtrai, ambas chegando como magnitude positiva.
 *
 * Por que não reaproveita a montagem de árvore do ResultadoPivot: lá a entrada é
 * `PivotLinha[]`, já pivotada pelo SQL, com uma linha por combinação dos quatro
 * níveis. Aqui a entrada é uma lista achatada de lançamentos, e a agregação é
 * parte do trabalho. Compartilhar exigiria um adaptador maior que estas ~50
 * linhas, e obrigaria a mexer numa tela que já está estável.
 */

export interface NoDiario {
  id: string
  nivel: 1 | 2 | 3 | 4
  rotulo: string
  /** Índice 0 = dia 1. Comprimento = dias do mês. */
  dias: number[]
  total: number
  temFilhos: boolean
  paiId: string | null
  /** Herdado da raiz: as três chegam positivas, o sinal não distingue. */
  tipo: 'receita' | 'despesa' | 'transferencia'
}

/**
 * Ordena os irmãos alfabeticamente, com receita, despesa e transferência nesta
 * ordem na raiz — explícito, e não por sorte da ordenação alfabética.
 *
 * A lista de transações vem ordenada por DATA, então a ordem de chegada não
 * agrupa nada — diferente do pivô anual, onde o `GROUP BY` do SQL já entrega
 * agrupado. Sem uma ordem explícita aqui, as linhas apareceriam embaralhadas e
 * mudariam de lugar a cada recorte.
 */
function ordenar(nos: NoDiario[]): NoDiario[] {
  const peso = (n: NoDiario) => {
    if (n.nivel !== 1) return 1
    switch (n.tipo) {
      case 'receita': return 0
      case 'despesa': return 1
      default:        return 2
    }
  }
  return [...nos].sort((a, b) => peso(a) - peso(b) || a.rotulo.localeCompare(b.rotulo, 'pt-BR'))
}

export function montarPivotDiario(
  transacoes: FluxoTransacao[],
  diasNoMes: number,
): NoDiario[] {
  const acc = new Map<string, NoDiario>()

  const somar = (
    id: string,
    nivel: NoDiario['nivel'],
    rotulo: string,
    paiId: string | null,
    t: FluxoTransacao,
  ) => {
    let n = acc.get(id)
    if (!n) {
      n = {
        id, nivel, rotulo, paiId,
        dias: Array(diasNoMes).fill(0),
        total: 0,
        temFilhos: nivel < 4,
        tipo: t.tipo,
      }
      acc.set(id, n)
    }
    // Dia fora do mês exibido seria um erro de dado, mas ignorar em silêncio é
    // melhor que escrever fora do array e produzir um NaN no total.
    if (t.dia >= 1 && t.dia <= diasNoMes) n.dias[t.dia - 1] += t.valor
    n.total += t.valor
  }

  for (const t of transacoes) {
    const i1 = `1|${t.tipo}`
    const i2 = `${i1}|${t.categoria_superior}`
    const i3 = `${i2}|${t.categoria}`
    const i4 = `${i3}|${t.descricao}`
    somar(i1, 1, t.tipo.toUpperCase(), null, t)
    somar(i2, 2, t.categoria_superior, i1, t)
    somar(i3, 3, t.categoria, i2, t)
    somar(i4, 4, t.descricao, i3, t)
  }

  // Percurso em profundidade, para que cada nó apareça sob o seu pai.
  const porPai = new Map<string | null, NoDiario[]>()
  for (const n of acc.values()) {
    const lista = porPai.get(n.paiId) ?? []
    lista.push(n)
    porPai.set(n.paiId, lista)
  }

  const saida: NoDiario[] = []
  const descer = (paiId: string | null) => {
    for (const n of ordenar(porPai.get(paiId) ?? [])) {
      saida.push(n)
      descer(n.id)
    }
  }
  descer(null)
  return saida
}

export interface ResultadoDiario {
  dias: number[]
  total: number
}

/**
 * Linha RESULTADO do rodapé, dia a dia.
 *
 * Soma pelos nós de nível 3 (categoria final) — não o 2 (categoria superior):
 * dentro de transferência, entrada e saída compartilham a MESMA categoria
 * superior ('Transferência'), e um nó de nível 2 já chega com as duas somadas
 * juntas, sem como saber de volta quanto era entrada e quanto era saída. No
 * nível 3 elas continuam separadas ('Entrada de Transferência' / 'Saída de
 * Transferência'), que é o que permite entrada somar e saída subtrair — a
 * mesma regra de `calcularResultado`, em utils/resultado.ts.
 *
 * Nível 3 não repete nem multiplica o total: cada transação cai em exatamente
 * um nó de nível 3, igual acontecia com o de nível 2 antes.
 *
 * A exclusão continua por categoria SUPERIOR, igual à aba Resultado — por isso
 * busca o pai (nível 2) de cada nó para montar a chave.
 */
export function resultadoDiario(
  nos: NoDiario[],
  excluidos: Set<string> = new Set(),
  diasNoMes = 31,
): ResultadoDiario {
  const dias = Array(diasNoMes).fill(0)
  let total = 0

  const porId = new Map(nos.map(n => [n.id, n]))

  for (const n of nos) {
    if (n.nivel !== 3) continue
    const pai = n.paiId ? porId.get(n.paiId) : undefined
    if (!pai) continue
    if (excluidos.has(chaveSuperior(n.tipo, pai.rotulo))) continue

    const sinal = (n.tipo === 'despesa' ||
      (n.tipo === 'transferencia' && n.rotulo === 'Saída de Transferência')) ? -1 : 1
    for (let i = 0; i < dias.length; i++) dias[i] += sinal * (n.dias[i] ?? 0)
    total += sinal * n.total
  }

  return { dias, total }
}
