import { describe, it, expect } from 'vitest'
import {
  cardsDoRecorte, saldoDoRecorte, cardResultado, margemDoRecorte, TODOS_OS_MESES,
} from './visaogeral'
import type { GraficoMensal, SaldoMes } from '@/api/dashboard'

const NOMES = ['Jan','Fev','Mar','Abr','Mai','Jun','Jul','Ago','Set','Out','Nov','Dez']

/** Doze meses; `vals` informa [receita, despesa] dos meses que importam. */
function serie(vals: Record<number, [number, number]>): GraficoMensal[] {
  return Array.from({ length: 12 }, (_, i) => {
    const [receita, despesa] = vals[i + 1] ?? [0, 0]
    return { mes: i + 1, mes_nome: NOMES[i], receita, despesa, resultado_mes: receita - despesa }
  })
}

function saldos(vals: number[]): SaldoMes[] {
  return vals.map((saldo, i) => ({ mes: i + 1, mes_nome: NOMES[i], saldo }))
}

const amostra = serie({
  1: [1000, 400],
  2: [2000, 2500],   // mês negativo
  9: [5000, 1000],
})

describe('cardsDoRecorte', () => {
  it('sem recorte soma os doze meses', () => {
    const r = cardsDoRecorte(amostra)
    expect(r.receita).toBe(8000)
    expect(r.despesa).toBe(3900)
    expect(r.resultado).toBe(4100)
  })

  it('o padrão é o mesmo que passar todos os meses', () => {
    expect(cardsDoRecorte(amostra)).toEqual(cardsDoRecorte(amostra, TODOS_OS_MESES))
  })

  it('recorta para um mês', () => {
    const r = cardsDoRecorte(amostra, new Set([9]))
    expect(r).toEqual({ receita: 5000, despesa: 1000, resultado: 4000 })
  })

  it('recorta para vários meses', () => {
    const r = cardsDoRecorte(amostra, new Set([1, 2]))
    expect(r).toEqual({ receita: 3000, despesa: 2900, resultado: 100 })
  })

  it('mês com despesa maior que receita dá resultado negativo', () => {
    expect(cardsDoRecorte(amostra, new Set([2])).resultado).toBe(-500)
  })

  it('nenhum mês marcado zera, e não devolve o ano inteiro', () => {
    expect(cardsDoRecorte(amostra, new Set())).toEqual({ receita: 0, despesa: 0, resultado: 0 })
  })

  it('mês sem movimento não muda nada', () => {
    expect(cardsDoRecorte(amostra, new Set([9, 5]))).toEqual(cardsDoRecorte(amostra, new Set([9])))
  })

  it('série vazia não estoura', () => {
    expect(cardsDoRecorte([])).toEqual({ receita: 0, despesa: 0, resultado: 0 })
  })
})

describe('saldoDoRecorte', () => {
  const s = saldos([100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1100, 1200])

  // Saldo é posição num instante. Somar os saldos mensais daria um número sem
  // significado — e grande o bastante para parecer certo.
  it('sem recorte é o saldo de dezembro, não a soma dos doze', () => {
    expect(saldoDoRecorte(s)).toBe(1200)
  })

  it('com um mês é o saldo daquele mês', () => {
    expect(saldoDoRecorte(s, new Set([3]))).toBe(300)
  })

  it('com vários meses é o do último marcado', () => {
    expect(saldoDoRecorte(s, new Set([2, 9, 5]))).toBe(900)
  })

  it('a ordem do Set não altera o resultado', () => {
    expect(saldoDoRecorte(s, new Set([9, 2, 5]))).toBe(900)
  })

  it('nenhum mês marcado devolve zero', () => {
    expect(saldoDoRecorte(s, new Set())).toBe(0)
  })

  it('lista vazia devolve zero em vez de estourar', () => {
    expect(saldoDoRecorte([], new Set([3]))).toBe(0)
  })

  it('saldo negativo é preservado', () => {
    expect(saldoDoRecorte(saldos([-500, -300]), new Set([1]))).toBe(-500)
  })
})

describe('cardResultado', () => {
  // Replica a fórmula do servidor de propósito: com todos os meses marcados o
  // card precisa exibir exatamente o número de antes desta mudança.
  it('soma o saldo ao resultado do período, como o backend faz', () => {
    expect(cardResultado({ receita: 35200, despesa: 12500, resultado: 22700 }, 10000)).toBe(32700)
  })

  it('saldo negativo reduz o card', () => {
    expect(cardResultado({ receita: 100, despesa: 0, resultado: 100 }, -40)).toBe(60)
  })
})

describe('margemDoRecorte', () => {
  it('fração do resultado sobre a receita', () => {
    expect(margemDoRecorte(32700, 35200)).toBeCloseTo(0.929, 3)
  })

  it('receita zero devolve null em vez de dividir por zero', () => {
    expect(margemDoRecorte(500, 0)).toBeNull()
  })

  it('resultado negativo dá margem negativa', () => {
    expect(margemDoRecorte(-50, 100)).toBe(-0.5)
  })
})
