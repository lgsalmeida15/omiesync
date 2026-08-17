import { describe, it, expect } from 'vitest'
import { porCategoria, topClientes, totalMarcados } from './agregacao'
import type { FluxoTransacao } from '@/api/fluxocaixa'

const t = (descricao: string, categoria: string, valor: number): FluxoTransacao =>
  ({ dia: 1, data: '01/08/2026', descricao, tipo: 'receita', categoria,
     valor, status: 'Recebido', realizado: true }) as FluxoTransacao

describe('porCategoria', () => {
  it('soma valores da mesma categoria', () => {
    const r = porCategoria([t('A', 'Servicos', 100), t('B', 'Servicos', 50)])
    expect(r).toEqual([{ rotulo: 'Servicos', valor: 150 }])
  })

  it('ordena do maior para o menor', () => {
    const r = porCategoria([t('A', 'Menor', 10), t('B', 'Maior', 900), t('C', 'Meio', 100)])
    expect(r.map(x => x.rotulo)).toEqual(['Maior', 'Meio', 'Menor'])
  })

  it('desempata por rótulo para a ordem não oscilar', () => {
    const r = porCategoria([t('A', 'Zeta', 100), t('B', 'Alfa', 100)])
    expect(r.map(x => x.rotulo)).toEqual(['Alfa', 'Zeta'])
  })

  it('preserva o rótulo de sem categoria em vez de descartar', () => {
    const r = porCategoria([t('A', 'Sem categoria', 70)])
    expect(r).toEqual([{ rotulo: 'Sem categoria', valor: 70 }])
  })

  it('lista vazia devolve vazio', () => {
    expect(porCategoria([])).toEqual([])
  })
})

describe('topClientes', () => {
  it('soma por cliente e ordena', () => {
    const r = topClientes([t('ACME', 'X', 100), t('BETA', 'X', 300), t('ACME', 'Y', 250)])
    expect(r).toEqual([{ rotulo: 'ACME', valor: 350 }, { rotulo: 'BETA', valor: 300 }])
  })

  it('corta nos 10 maiores', () => {
    const muitos = Array.from({ length: 25 }, (_, i) => t(`CLIENTE ${i}`, 'X', i + 1))
    const r = topClientes(muitos)
    expect(r).toHaveLength(10)
    expect(r[0].valor).toBe(25)
    expect(r[9].valor).toBe(16)
  })

  it('corta por valor somado, não por quantidade de lançamentos', () => {
    // GRANDE tem 1 lançamento alto; PEQUENO tem 5 baixos. GRANDE deve vir antes.
    const movs = [t('GRANDE', 'X', 500), ...Array.from({ length: 5 }, () => t('PEQUENO', 'X', 10))]
    expect(topClientes(movs, 1)).toEqual([{ rotulo: 'GRANDE', valor: 500 }])
  })

  it('respeita limite customizado', () => {
    const r = topClientes([t('A', 'X', 3), t('B', 'X', 2), t('C', 'X', 1)], 2)
    expect(r.map(x => x.rotulo)).toEqual(['A', 'B'])
  })

  it('lista vazia devolve vazio', () => {
    expect(topClientes([])).toEqual([])
  })
})

describe('totalMarcados', () => {
  const itens = [{ rotulo: 'A', valor: 100 }, { rotulo: 'B', valor: 50 }, { rotulo: 'C', valor: 25 }]

  it('soma só os marcados', () => {
    expect(totalMarcados(itens, new Set(['A', 'C']))).toBe(125)
  })

  it('nenhum marcado soma zero', () => {
    expect(totalMarcados(itens, new Set())).toBe(0)
  })

  it('todos marcados soma o total', () => {
    expect(totalMarcados(itens, new Set(['A', 'B', 'C']))).toBe(175)
  })

  it('rótulo inexistente no conjunto é ignorado', () => {
    expect(totalMarcados(itens, new Set(['A', 'INEXISTENTE']))).toBe(100)
  })
})
