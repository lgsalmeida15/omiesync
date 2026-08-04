import { describe, it, expect } from 'vitest'
import { fmtMoeda, fmtNumero, fmtCompacto } from './formato'

// Intl usa espaço não separável (U+00A0) depois de "R$". Normaliza para comparar.
const n = (s: string) => s.replace(/ /g, ' ')

describe('fmtMoeda', () => {
  it('positivo sai sem parênteses', () => {
    expect(n(fmtMoeda(1234))).toBe('R$ 1.234')
  })

  it('negativo sai entre parênteses, sem sinal de menos', () => {
    const r = n(fmtMoeda(-1234))
    expect(r).toBe('(R$ 1.234)')
    expect(r).not.toContain('-')
  })

  it('zero não é tratado como negativo', () => {
    expect(n(fmtMoeda(0))).toBe('R$ 0')
  })

  it('valor grande mantém separador de milhar', () => {
    expect(n(fmtMoeda(-1234567))).toBe('(R$ 1.234.567)')
  })
})

describe('fmtNumero', () => {
  it('positivo com dois decimais', () => {
    expect(fmtNumero(1234.5)).toBe('1.234,50')
  })

  it('negativo entre parênteses', () => {
    expect(fmtNumero(-1234.5)).toBe('(1.234,50)')
  })

  it('zero', () => {
    expect(fmtNumero(0)).toBe('0,00')
  })

  // Reproduz o caso real das baixas parciais que somam 13.179,35.
  it('preserva centavos de valor real', () => {
    expect(fmtNumero(13179.35)).toBe('13.179,35')
  })
})

describe('fmtCompacto', () => {
  it('milhões', () => {
    expect(fmtCompacto(1_500_000)).toBe('R$1.5M')
  })

  it('milhões negativos entre parênteses', () => {
    expect(fmtCompacto(-1_500_000)).toBe('(R$1.5M)')
  })

  it('milhares', () => {
    expect(fmtCompacto(45_000)).toBe('R$45K')
  })

  it('milhares negativos entre parênteses', () => {
    expect(fmtCompacto(-45_000)).toBe('(R$45K)')
  })

  // Rótulo em cima de barra sem altura só sujaria o gráfico.
  it('zero devolve vazio', () => {
    expect(fmtCompacto(0)).toBe('')
  })

  it('abaixo de mil cai na formatação de moeda', () => {
    expect(n(fmtCompacto(-500))).toBe('(R$ 500)')
  })
})
