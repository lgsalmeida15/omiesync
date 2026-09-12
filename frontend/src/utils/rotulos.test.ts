import { describe, it, expect } from 'vitest'
import { rotulosGraficos, type TipoVisao } from './rotulos'

const TIPOS: TipoVisao[] = ['todos', 'receita', 'despesa']

describe('rotulosGraficos', () => {
  it('os três tipos têm rótulo', () => {
    for (const t of TIPOS) {
      expect(rotulosGraficos(t).donut).toBeTruthy()
      expect(rotulosGraficos(t).barras).toBeTruthy()
    }
  })

  // O erro que este módulo existe para impedir: quem recebe é cliente, quem
  // cobra é fornecedor. Trocado, não quebra nada e não aparece no console.
  it('recebimento fala de cliente, nunca de fornecedor', () => {
    const r = rotulosGraficos('receita')
    expect(r.barras).toContain('CLIENTES')
    expect(r.barras).not.toContain('FORNECEDOR')
    expect(r.donut).toContain('RECEBIMENTOS')
  })

  it('pagamento fala de fornecedor, nunca de cliente', () => {
    const r = rotulosGraficos('despesa')
    expect(r.barras).toContain('FORNECEDORES')
    expect(r.barras).not.toContain('CLIENTES')
    expect(r.donut).toContain('PAGAMENTOS')
  })

  // Na visão dos dois lados não dá para escolher um nome só.
  it('a visão geral nomeia os dois', () => {
    const r = rotulosGraficos('todos')
    expect(r.barras).toContain('CLIENTE')
    expect(r.barras).toContain('FORNECEDOR')
  })

  it('cada tipo tem um par distinto', () => {
    const donuts = TIPOS.map(t => rotulosGraficos(t).donut)
    const barras = TIPOS.map(t => rotulosGraficos(t).barras)
    expect(new Set(donuts).size).toBe(3)
    expect(new Set(barras).size).toBe(3)
  })

  // Mutar o objeto devolvido afetaria as próximas chamadas, porque a tabela é
  // um literal compartilhado.
  it('a tabela não é alterável pelo chamador sem aviso', () => {
    const antes = rotulosGraficos('receita').barras
    const r = rotulosGraficos('receita')
    r.barras = 'ALTERADO'
    expect(rotulosGraficos('receita').barras).toBe(antes)
  })
})
