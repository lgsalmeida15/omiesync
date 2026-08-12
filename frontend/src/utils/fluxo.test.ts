import { describe, it, expect } from 'vitest'
import { filtrarTransacoes, classeStatus, type FiltrosListagem } from './fluxo'
import type { FluxoTransacao } from '@/api/fluxocaixa'

function t(p: Partial<FluxoTransacao>): FluxoTransacao {
  return {
    dia: 1, data: '01/08/2026', descricao: 'ACME LTDA', tipo: 'receita',
    categoria: 'Servicos', valor: 100, status: 'Recebido', realizado: true, ...p,
  } as FluxoTransacao
}

const base: FiltrosListagem = { dia: null, tipo: '', situacao: '', busca: '' }

const amostra: FluxoTransacao[] = [
  t({ dia: 3, valor: 5000 }),
  t({ dia: 5, tipo: 'despesa', status: 'Pago', valor: 2000, descricao: 'FORNEC SA', categoria: 'Fornecedores' }),
  t({ dia: 13, status: 'Pendente', realizado: false, valor: 8000 }),
  t({ dia: 15, tipo: 'despesa', status: 'Pendente', realizado: false, valor: 3000, categoria: 'Fornecedores' }),
]

describe('filtrarTransacoes', () => {
  it('sem filtros devolve tudo, inclusive pendentes', () => {
    expect(filtrarTransacoes(amostra, base)).toHaveLength(4)
  })

  it('preserva a ordem de entrada (servidor já ordena por data)', () => {
    expect(filtrarTransacoes(amostra, base).map(x => x.dia)).toEqual([3, 5, 13, 15])
  })

  it('filtra por dia mantendo efetuadas e pendentes', () => {
    expect(filtrarTransacoes(amostra, { ...base, dia: 13 }).map(x => x.valor)).toEqual([8000])
  })

  it('dia sem lançamento devolve vazio', () => {
    expect(filtrarTransacoes(amostra, { ...base, dia: 28 })).toHaveLength(0)
  })

  it('situacao=efetuada devolve só realizado', () => {
    const r = filtrarTransacoes(amostra, { ...base, situacao: 'efetuada' })
    expect(r).toHaveLength(2)
    expect(r.every(x => x.realizado)).toBe(true)
  })

  it('situacao=pendente devolve só não realizado', () => {
    const r = filtrarTransacoes(amostra, { ...base, situacao: 'pendente' })
    expect(r).toHaveLength(2)
    expect(r.every(x => !x.realizado)).toBe(true)
  })

  it('filtra por tipo', () => {
    expect(filtrarTransacoes(amostra, { ...base, tipo: 'despesa' })).toHaveLength(2)
  })

  it('combina tipo e situacao', () => {
    const r = filtrarTransacoes(amostra, { ...base, tipo: 'despesa', situacao: 'pendente' })
    expect(r.map(x => x.valor)).toEqual([3000])
  })

  it('busca por descrição, ignorando caixa', () => {
    expect(filtrarTransacoes(amostra, { ...base, busca: 'fornec sa' })).toHaveLength(1)
  })

  it('busca por categoria', () => {
    expect(filtrarTransacoes(amostra, { ...base, busca: 'Fornecedores' })).toHaveLength(2)
  })

  it('busca só com espaços é ignorada', () => {
    expect(filtrarTransacoes(amostra, { ...base, busca: '   ' })).toHaveLength(4)
  })

  it('lista vazia não quebra', () => {
    expect(filtrarTransacoes([], base)).toEqual([])
  })
})

describe('classeStatus', () => {
  it('pendente é âmbar independentemente do tipo', () => {
    expect(classeStatus({ tipo: 'receita', realizado: false })).toBe('pill--pend')
    expect(classeStatus({ tipo: 'despesa', realizado: false })).toBe('pill--pend')
  })

  it('recebido é verde e pago é vermelho', () => {
    expect(classeStatus({ tipo: 'receita', realizado: true })).toBe('pill--in')
    expect(classeStatus({ tipo: 'despesa', realizado: true })).toBe('pill--out')
  })
})
