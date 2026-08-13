import { describe, it, expect } from 'vitest'
import { agruparContas, aplicarGrupo } from './contas'
import type { ContaCorrenteItem } from '@/api/dashboard'

const c = (codigo: string, descricao: string, fluxo_caixa: string): ContaCorrenteItem =>
  ({ codigo, descricao, fluxo_caixa })

describe('agruparContas', () => {
  it("'S' vai para Consideradas", () => {
    const g = agruparContas([c('1', 'Banco A', 'S')])
    expect(g).toHaveLength(1)
    expect(g[0].id).toBe('consideradas')
    expect(g[0].rotulo).toBe('Consideradas')
  })

  it("'N' vai para Não consideradas", () => {
    const g = agruparContas([c('1', 'Banco A', 'N')])
    expect(g[0].id).toBe('nao_consideradas')
    expect(g[0].rotulo).toBe('Não consideradas')
  })

  it('vazio e nulo vão para Sem marcação, não para Não consideradas', () => {
    const g = agruparContas([
      c('1', 'Banco A', ''),
      { codigo: '2', descricao: 'Banco B' } as ContaCorrenteItem,
    ])
    expect(g).toHaveLength(1)
    expect(g[0].id).toBe('sem_marca')
    expect(g[0].contas).toHaveLength(2)
  })

  it('normaliza caixa e espaços da marca', () => {
    const g = agruparContas([c('1', 'A', ' s '), c('2', 'B', 'n')])
    expect(g.map(x => x.id)).toEqual(['consideradas', 'nao_consideradas'])
  })

  it('marca desconhecida cai em Sem marcação em vez de sumir', () => {
    const g = agruparContas([c('1', 'A', 'X')])
    expect(g[0].id).toBe('sem_marca')
  })

  it('separa os três grupos na ordem fixa', () => {
    const g = agruparContas([c('1', 'A', 'N'), c('2', 'B', ''), c('3', 'C', 'S')])
    expect(g.map(x => x.id)).toEqual(['consideradas', 'nao_consideradas', 'sem_marca'])
  })

  it('omite grupos vazios', () => {
    const g = agruparContas([c('1', 'A', 'S'), c('2', 'B', 'S')])
    expect(g).toHaveLength(1)
  })

  it('preserva a ordem de entrada dentro do grupo', () => {
    const g = agruparContas([c('1', 'Zeta', 'S'), c('2', 'Alfa', 'S')])
    expect(g[0].contas.map(x => x.descricao)).toEqual(['Zeta', 'Alfa'])
  })

  it('lista vazia devolve nenhum grupo', () => {
    expect(agruparContas([])).toEqual([])
  })

  it('nenhuma conta é perdida no agrupamento', () => {
    const itens = [c('1', 'A', 'S'), c('2', 'B', 'N'), c('3', 'C', ''), c('4', 'D', 'S')]
    const total = agruparContas(itens).reduce((s, g) => s + g.contas.length, 0)
    expect(total).toBe(itens.length)
  })
})

describe('aplicarGrupo', () => {
  const todos = ['1', '2', '3', '4']

  it('desmarcar um grupo partindo de "todas" materializa o restante', () => {
    expect(aplicarGrupo(todos, [], ['1', '2'], false)).toEqual(['3', '4'])
  })

  it('marcar um grupo partindo de "todas" continua sendo "todas"', () => {
    expect(aplicarGrupo(todos, [], ['1', '2'], true)).toEqual([])
  })

  it('marcar um grupo soma à seleção existente', () => {
    expect(aplicarGrupo(todos, ['3'], ['1'], true)).toEqual(['1', '3'])
  })

  it('marcar tudo volta ao estado "todas" para conta nova entrar sozinha', () => {
    expect(aplicarGrupo(todos, ['3', '4'], ['1', '2'], true)).toEqual([])
  })

  it('desmarcar remove só as contas do grupo', () => {
    expect(aplicarGrupo(todos, ['1', '2', '3'], ['2'], false)).toEqual(['1', '3'])
  })

  it('devolve na ordem de todos, não na ordem da seleção', () => {
    expect(aplicarGrupo(todos, ['4', '1'], ['2'], true)).toEqual(['1', '2', '4'])
  })

  it('desmarcar grupo já ausente não altera a seleção', () => {
    expect(aplicarGrupo(todos, ['1'], ['2', '3'], false)).toEqual(['1'])
  })

  it('não duplica ao marcar conta já selecionada', () => {
    expect(aplicarGrupo(todos, ['1', '2'], ['1'], true)).toEqual(['1', '2'])
  })

  it('grupo vazio é no-op', () => {
    expect(aplicarGrupo(todos, ['1'], [], true)).toEqual(['1'])
  })
})
