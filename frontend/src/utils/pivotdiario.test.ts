import { describe, it, expect } from 'vitest'
import { montarPivotDiario, resultadoDiario, type NoDiario } from './pivotdiario'
import { chaveSuperior } from './resultado'
import { resumirTransacoes } from './fluxo'
import type { FluxoTransacao } from '@/api/fluxocaixa'

function t(p: Partial<FluxoTransacao>): FluxoTransacao {
  return {
    dia: 1, data: '01/09/2026', descricao: 'ACME LTDA', tipo: 'receita',
    categoria_superior: '1.01 Vendas', categoria: 'Servicos',
    valor: 100, status: 'Recebido', realizado: true, ...p,
  } as FluxoTransacao
}

const DIAS = 30

const amostra: FluxoTransacao[] = [
  t({ dia: 3, valor: 1000 }),
  t({ dia: 3, valor: 500, descricao: 'BETA SA' }),
  t({ dia: 10, valor: 2000, categoria: 'Produtos' }),
  t({ dia: 10, valor: 4000, tipo: 'despesa', categoria_superior: '2.01 Custos',
      categoria: 'Fornecedores', descricao: 'FORNEC SA' }),
  t({ dia: 20, valor: 1500, tipo: 'despesa', categoria_superior: '2.02 Ocupacao',
      categoria: 'Aluguel', descricao: 'IMOB LTDA' }),
]

const acha = (nos: NoDiario[], nivel: number, rotulo: string) =>
  nos.find(n => n.nivel === nivel && n.rotulo === rotulo)!

describe('montarPivotDiario', () => {
  const nos = montarPivotDiario(amostra, DIAS)

  it('produz os quatro níveis', () => {
    expect(new Set(nos.map(n => n.nivel))).toEqual(new Set([1, 2, 3, 4]))
  })

  it('receita vem antes de despesa na raiz', () => {
    expect(nos.filter(n => n.nivel === 1).map(n => n.rotulo)).toEqual(['RECEITA', 'DESPESA'])
  })

  it('cada nó aparece depois do seu pai', () => {
    const posicao = new Map(nos.map((n, i) => [n.id, i]))
    for (const n of nos) {
      if (n.paiId) expect(posicao.get(n.paiId)!).toBeLessThan(posicao.get(n.id)!)
    }
  })

  it('nível 1 soma o lado inteiro', () => {
    expect(acha(nos, 1, 'RECEITA').total).toBe(3500)
    expect(acha(nos, 1, 'DESPESA').total).toBe(5500)
  })

  it('agrega por dia no índice dia-1', () => {
    const receita = acha(nos, 1, 'RECEITA')
    expect(receita.dias[2]).toBe(1500) // dia 3: 1000 + 500
    expect(receita.dias[9]).toBe(2000) // dia 10
    expect(receita.dias[0]).toBe(0)
  })

  it('o array de dias tem o comprimento do mês', () => {
    expect(montarPivotDiario(amostra, 28)[0].dias).toHaveLength(28)
    expect(montarPivotDiario(amostra, 31)[0].dias).toHaveLength(31)
  })

  it('total de cada nó bate com a soma dos seus dias', () => {
    for (const n of nos) {
      expect(n.dias.reduce((s, v) => s + v, 0)).toBeCloseTo(n.total, 6)
    }
  })

  it('só o nível 4 é folha', () => {
    for (const n of nos) expect(n.temFilhos).toBe(n.nivel < 4)
  })

  it('o tipo é herdado da raiz', () => {
    expect(acha(nos, 3, 'Aluguel').tipo).toBe('despesa')
    expect(acha(nos, 4, 'BETA SA').tipo).toBe('receita')
  })

  // A mesma categoria pode existir dos dois lados; se a chave não levasse o
  // tipo, receita e despesa cairiam no mesmo nó e se somariam.
  it('categoria homônima nos dois lados não se mistura', () => {
    const nos2 = montarPivotDiario([
      t({ dia: 1, valor: 100, categoria_superior: 'X', categoria: 'Juros' }),
      t({ dia: 1, valor: 700, tipo: 'despesa', categoria_superior: 'X', categoria: 'Juros' }),
    ], DIAS)
    const juros = nos2.filter(n => n.nivel === 3 && n.rotulo === 'Juros')
    expect(juros).toHaveLength(2)
    expect(juros.map(n => n.total).sort((a, b) => a - b)).toEqual([100, 700])
  })

  it('lista vazia produz árvore vazia', () => {
    expect(montarPivotDiario([], DIAS)).toEqual([])
  })

  // Escrever fora do array produziria um NaN silencioso no total.
  it('dia fora do mês não corrompe o total', () => {
    const nos2 = montarPivotDiario([t({ dia: 31, valor: 900 })], 30)
    expect(nos2[0].dias.every(v => v === 0)).toBe(true)
    expect(nos2[0].total).toBe(900)
  })

  it('inadimplência entra como qualquer outra transação', () => {
    const nos2 = montarPivotDiario(
      [...amostra, t({ dia: 8, valor: 700, status: 'Atrasado', realizado: false })], DIAS)
    expect(acha(nos2, 1, 'RECEITA').total).toBe(4200)
  })
})

describe('resultadoDiario', () => {
  const nos = montarPivotDiario(amostra, DIAS)

  it('receita soma e despesa subtrai', () => {
    expect(resultadoDiario(nos, new Set(), DIAS).total).toBe(3500 - 5500)
  })

  // Somar todos os níveis multiplicaria tudo por quatro, já que cada nível
  // repete os mesmos valores.
  it('não conta os níveis duplicados', () => {
    const r = resultadoDiario(nos, new Set(), DIAS)
    expect(r.dias[2]).toBe(1500)          // dia 3: só receita
    expect(r.dias[9]).toBe(2000 - 4000)   // dia 10: os dois lados
    expect(r.dias[19]).toBe(-1500)        // dia 20: só despesa
  })

  it('o total bate com a soma dos dias', () => {
    const r = resultadoDiario(nos, new Set(), DIAS)
    expect(r.dias.reduce((s, v) => s + v, 0)).toBeCloseTo(r.total, 6)
  })

  it('excluir uma superior a retira da conta', () => {
    const fora = new Set([chaveSuperior('despesa', '2.01 Custos')])
    expect(resultadoDiario(nos, fora, DIAS).total).toBe(3500 - 1500)
  })

  it('a exclusão distingue os lados', () => {
    // Excluir a superior de receita não mexe na despesa homônima.
    const nos2 = montarPivotDiario([
      t({ dia: 1, valor: 100, categoria_superior: 'X' }),
      t({ dia: 1, valor: 700, tipo: 'despesa', categoria_superior: 'X' }),
    ], DIAS)
    expect(resultadoDiario(nos2, new Set([chaveSuperior('receita', 'X')]), DIAS).total).toBe(-700)
    expect(resultadoDiario(nos2, new Set([chaveSuperior('despesa', 'X')]), DIAS).total).toBe(100)
  })

  it('excluir tudo zera', () => {
    const todas = new Set(
      nos.filter(n => n.nivel === 2).map(n => chaveSuperior(n.tipo, n.rotulo)))
    const r = resultadoDiario(nos, todas, DIAS)
    expect(r.total).toBe(0)
    expect(r.dias.every(v => v === 0)).toBe(true)
  })

  it('árvore vazia devolve zeros do tamanho do mês', () => {
    const r = resultadoDiario([], new Set(), 28)
    expect(r.dias).toHaveLength(28)
    expect(r.total).toBe(0)
  })
})

// Esta é a garantia que impede o pivô de derivar do resto da tela: sem exclusões,
// o RESULTADO tem de ser exatamente o mesmo número que o resumo do mês mostra.
describe('coerência com o resumo do mês', () => {
  it('o RESULTADO do pivô é igual ao resultado do resumo', () => {
    const nos = montarPivotDiario(amostra, DIAS)
    expect(resultadoDiario(nos, new Set(), DIAS).total)
      .toBeCloseTo(resumirTransacoes(amostra).resultado, 6)
  })

  it('vale também com inadimplência na lista', () => {
    const com = [
      ...amostra,
      t({ dia: 8, valor: 700, status: 'Atrasado', realizado: false }),
      t({ dia: 9, valor: 300, tipo: 'despesa', status: 'Atrasado', realizado: false,
          categoria_superior: '2.01 Custos' }),
    ]
    const nos = montarPivotDiario(com, DIAS)
    expect(resultadoDiario(nos, new Set(), DIAS).total)
      .toBeCloseTo(resumirTransacoes(com).resultado, 6)
  })
})
