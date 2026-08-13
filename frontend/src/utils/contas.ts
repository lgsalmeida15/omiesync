import type { ContaCorrenteItem } from '@/api/dashboard'

export interface GrupoContas {
  /** Chave estável para :key e para o estado de expandido. */
  id: 'consideradas' | 'nao_consideradas' | 'sem_marca'
  rotulo: string
  contas: ContaCorrenteItem[]
}

/**
 * Agrupa as contas correntes pela marca cFluxoCaixa do Omie.
 *
 * A semântica é invertida em relação ao nome do campo, e isso não é engano de
 * leitura: no Omie 'S' significa que a conta NÃO é considerada no fluxo de
 * caixa, e 'N' que ela é. Confirmado com o usuário.
 *
 * Contas sem marca ficam num terceiro grupo em vez de irem para "não
 * consideradas": a ausência do campo significa que o extrato ainda não
 * sincronizou aquela conta, não que ela tenha sido classificada.
 *
 * Grupos vazios são omitidos, e a ordem de entrada é preservada — o servidor já
 * devolve ordenado por descrição.
 */
export function agruparContas(itens: ContaCorrenteItem[]): GrupoContas[] {
  const grupos: GrupoContas[] = [
    { id: 'consideradas',     rotulo: 'Consideradas',     contas: [] },
    { id: 'nao_consideradas', rotulo: 'Não consideradas', contas: [] },
    { id: 'sem_marca',        rotulo: 'Sem marcação',     contas: [] },
  ]

  for (const c of itens) {
    const marca = (c.fluxo_caixa ?? '').trim().toUpperCase()
    if (marca === 'N')      grupos[0].contas.push(c)
    else if (marca === 'S') grupos[1].contas.push(c)
    else                    grupos[2].contas.push(c)
  }

  return grupos.filter(g => g.contas.length > 0)
}

/**
 * Marca ou desmarca de uma vez todas as contas de um grupo, devolvendo a nova
 * seleção.
 *
 * Segue a convenção já usada pelos outros filtros: lista vazia significa
 * "todas". Por isso a seleção atual é materializada antes de operar, e volta a
 * vazia quando o resultado cobre tudo — assim uma conta nova entra sozinha.
 *
 * A ordem de `todos` é preservada no retorno.
 */
export function aplicarGrupo(
  todos: string[],
  atual: string[],
  doGrupo: string[],
  marcar: boolean,
): string[] {
  const sel = new Set(atual.length === 0 ? todos : atual)
  for (const cod of doGrupo) {
    if (marcar) sel.add(cod)
    else sel.delete(cod)
  }
  const proximo = todos.filter(c => sel.has(c))
  return proximo.length === todos.length ? [] : proximo
}
