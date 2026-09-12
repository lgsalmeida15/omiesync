/**
 * Títulos dos gráficos por tipo de visão.
 *
 * A entidade muda de nome conforme o lado: quem recebe é cliente, quem cobra é
 * fornecedor. Na visão dos dois juntos não dá para escolher, então o rótulo
 * nomeia os dois.
 *
 * Módulo próprio e testado porque trocar "clientes" por "fornecedores" na aba
 * errada não quebra nada, não aparece no console e passa despercebido em
 * revisão — só quem conhece o negócio nota, olhando a tela.
 */

export type TipoVisao = 'todos' | 'receita' | 'despesa'

export interface RotulosGraficos {
  donut: string
  barras: string
}

const ROTULOS: Record<TipoVisao, RotulosGraficos> = {
  todos: {
    donut:  'TOTAL GERAL POR CATEGORIA',
    barras: 'TOTAL GERAL POR CLIENTE/FORNECEDOR',
  },
  receita: {
    donut:  'RECEBIMENTOS POR CATEGORIA',
    barras: 'TOP 10 CLIENTES',
  },
  despesa: {
    donut:  'PAGAMENTOS POR CATEGORIA',
    barras: 'TOP 10 FORNECEDORES',
  },
}

/**
 * Devolve uma CÓPIA, não a entrada da tabela: entregando a referência, um
 * chamador que alterasse o objeto corromperia o rótulo para todas as chamadas
 * seguintes, e o defeito apareceria longe da causa.
 */
export function rotulosGraficos(tipo: TipoVisao): RotulosGraficos {
  return { ...ROTULOS[tipo] }
}
