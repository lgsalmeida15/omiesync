import { describe, it, expect } from 'vitest'
import { montarCascata, type PassoCascata } from './cascata'
import type { GraficoAcumulado } from '@/api/dashboard'

const NOMES = ['Jan','Fev','Mar','Abr','Mai','Jun','Jul','Ago','Set','Out','Nov','Dez']

/** Doze meses; `vals` informa o resultado dos meses que importam. */
function serie(vals: Record<number, number>): GraficoAcumulado[] {
  let acc = 0
  return Array.from({ length: 12 }, (_, i) => {
    const resultado_mes = vals[i + 1] ?? 0
    acc += resultado_mes
    return { mes: i + 1, mes_nome: NOMES[i], resultado_mes, acumulado: acc }
  })
}

// jan +5000, fev −2000, mar +1000; o resto do ano sem movimento.
const meses = serie({ 1: 5000, 2: -2000, 3: 1000 })
const SALDO = 10000

const porRotulo = (p: PassoCascata[], r: string) => p.find(x => x.rotulo === r)!

describe('montarCascata', () => {
  const passos = montarCascata(SALDO, meses)

  it('abre com o saldo, do zero até ele', () => {
    expect(passos[0]).toEqual({
      rotulo: 'Saldo', de: 0, ate: 10000, valor: 10000, tipo: 'abertura',
    })
  })

  it('fecha com o total, ancorado no zero', () => {
    const t = passos[passos.length - 1]
    expect(t.rotulo).toBe('Total')
    expect(t.tipo).toBe('total')
    expect(t.de).toBe(0)
    expect(t.ate).toBe(14000) // 10000 + 5000 − 2000 + 1000
  })

  it('tem abertura + doze meses + total', () => {
    expect(passos).toHaveLength(14)
  })

  // A propriedade que define uma cascata. Se um degrau não começar onde o
  // anterior terminou, o gráfico continua parecendo certo e mente.
  it('cada mês parte de onde o anterior terminou', () => {
    const escada = passos.filter(p => p.tipo !== 'total')
    for (let i = 1; i < escada.length; i++) {
      expect(escada[i].de).toBe(escada[i - 1].ate)
    }
  })

  it('o topo de cada mês é a base mais o valor', () => {
    for (const p of passos.filter(x => x.tipo === 'aumento' || x.tipo === 'reducao')) {
      expect(p.ate).toBeCloseTo(p.de + p.valor, 6)
    }
  })

  it('mês positivo sobe e é aumento', () => {
    const jan = porRotulo(passos, 'Jan')
    expect(jan).toMatchObject({ de: 10000, ate: 15000, valor: 5000, tipo: 'aumento' })
  })

  it('mês negativo desce e é redução', () => {
    const fev = porRotulo(passos, 'Fev')
    expect(fev).toMatchObject({ de: 15000, ate: 13000, valor: -2000, tipo: 'reducao' })
  })

  // Degrau de altura zero é informação — o mês existiu e não mexeu no saldo.
  it('mês sem movimento vira degrau de altura zero, no nível atual', () => {
    const abr = porRotulo(passos, 'Abr')
    expect(abr).toMatchObject({ de: 14000, ate: 14000, valor: 0, tipo: 'aumento' })
  })

  it('o total é igual ao topo do último mês exibido', () => {
    const ultimoMes = passos[passos.length - 2]
    expect(passos[passos.length - 1].ate).toBe(ultimoMes.ate)
  })
})

describe('montarCascata com recorte de meses', () => {
  it('mostra só os meses marcados', () => {
    const p = montarCascata(SALDO, meses, new Set([1, 2]))
    expect(p.map(x => x.rotulo)).toEqual(['Saldo', 'Jan', 'Fev', 'Total'])
  })

  it('o total segue o recorte', () => {
    const p = montarCascata(SALDO, meses, new Set([1, 2]))
    expect(p[p.length - 1].ate).toBe(13000) // 10000 + 5000 − 2000
  })

  // Meses fora do recorte são ignorados por completo: a escada não "pula" o
  // valor deles, ela nem os enxerga.
  it('recorte não-contíguo encadeia os marcados entre si', () => {
    const p = montarCascata(SALDO, meses, new Set([1, 3]))
    expect(p.map(x => x.rotulo)).toEqual(['Saldo', 'Jan', 'Mar', 'Total'])
    expect(porRotulo(p, 'Mar')).toMatchObject({ de: 15000, ate: 16000 })
    expect(p[p.length - 1].ate).toBe(16000)
  })

  it('sem meses marcados sobram abertura e total, iguais ao saldo', () => {
    const p = montarCascata(SALDO, meses, new Set())
    expect(p.map(x => x.rotulo)).toEqual(['Saldo', 'Total'])
    expect(p[1].ate).toBe(SALDO)
  })

  it('a propriedade da escada vale sob recorte', () => {
    const escada = montarCascata(SALDO, meses, new Set([2, 3])).filter(p => p.tipo !== 'total')
    for (let i = 1; i < escada.length; i++) {
      expect(escada[i].de).toBe(escada[i - 1].ate)
    }
  })
})

describe('montarCascata em casos de borda', () => {
  it('saldo inicial negativo abre abaixo do zero', () => {
    const p = montarCascata(-500, serie({ 1: 200 }))
    expect(p[0]).toMatchObject({ de: 0, ate: -500, tipo: 'abertura' })
    expect(porRotulo(p, 'Jan')).toMatchObject({ de: -500, ate: -300 })
  })

  it('total pode ficar negativo', () => {
    const p = montarCascata(1000, serie({ 1: -3000 }))
    expect(p[p.length - 1].ate).toBe(-2000)
  })

  it('saldo zero e sem movimento devolve tudo em zero', () => {
    const p = montarCascata(0, serie({}))
    expect(p.every(x => x.de === 0 || x.ate === 0)).toBe(true)
    expect(p[p.length - 1].ate).toBe(0)
  })

  it('série vazia devolve só abertura e total', () => {
    const p = montarCascata(700, [])
    expect(p.map(x => x.rotulo)).toEqual(['Saldo', 'Total'])
    expect(p[1].ate).toBe(700)
  })

  it('rótulos de abertura e total são configuráveis', () => {
    const p = montarCascata(1, [], new Set(), 'Inicial', 'Final')
    expect(p.map(x => x.rotulo)).toEqual(['Inicial', 'Final'])
  })
})
