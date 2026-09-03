import { describe, it, expect } from 'vitest'
import { filtrarTransacoes, classeStatus, resumirTransacoes, type FiltrosListagem } from './fluxo'
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

// Amostra separada de propósito: os testes acima afirmam contagens exatas sobre
// `amostra`, e ampliá-la quebraria seis deles sem que nada de real regredisse.
//
// Inadimplência tem `realizado: false` como qualquer pendente — é o `status` que
// a distingue, e é por isso que ela precisa de tratamento próprio.
const comAtraso: FluxoTransacao[] = [
  ...amostra,
  t({ dia: 8, status: 'Atrasado', realizado: false, valor: 7000, descricao: 'DEVEDOR SA' }),
  t({ dia: 20, tipo: 'despesa', status: 'Atrasado', realizado: false, valor: 4000 }),
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
    expect(classeStatus({ tipo: 'receita', realizado: false, status: 'Pendente' })).toBe('pill--pend')
    expect(classeStatus({ tipo: 'despesa', realizado: false, status: 'Pendente' })).toBe('pill--pend')
  })

  it('recebido é verde e pago é vermelho', () => {
    expect(classeStatus({ tipo: 'receita', realizado: true, status: 'Recebido' })).toBe('pill--in')
    expect(classeStatus({ tipo: 'despesa', realizado: true, status: 'Pago' })).toBe('pill--out')
  })
})

describe('filtrarTransacoes — inadimplência', () => {
  it('situacao=atrasada devolve só os vencidos', () => {
    const r = filtrarTransacoes(comAtraso, { ...base, situacao: 'atrasada' })
    expect(r).toHaveLength(2)
    expect(r.every(x => x.status === 'Atrasado')).toBe(true)
  })

  // A regra antiga era `(situacao === 'efetuada') === realizado`. Como atrasado
  // tem realizado false, ele cai em pendente — e precisa continuar caindo, senao
  // sumiria de quem filtra por pendente esperando ver tudo que nao foi pago.
  it('atrasado continua aparecendo em situacao=pendente', () => {
    const r = filtrarTransacoes(comAtraso, { ...base, situacao: 'pendente' })
    expect(r).toHaveLength(4)
    expect(r.filter(x => x.status === 'Atrasado')).toHaveLength(2)
  })

  it('atrasado nunca aparece em situacao=efetuada', () => {
    const r = filtrarTransacoes(comAtraso, { ...base, situacao: 'efetuada' })
    expect(r.some(x => x.status === 'Atrasado')).toBe(false)
  })

  it('sem filtro de situacao, os atrasados vêm junto', () => {
    expect(filtrarTransacoes(comAtraso, base)).toHaveLength(6)
  })

  it('combina atrasada com tipo', () => {
    const r = filtrarTransacoes(comAtraso, { ...base, situacao: 'atrasada', tipo: 'despesa' })
    expect(r).toHaveLength(1)
    expect(r[0].valor).toBe(4000)
  })

  it('combina atrasada com dia', () => {
    expect(filtrarTransacoes(comAtraso, { ...base, situacao: 'atrasada', dia: 8 })).toHaveLength(1)
  })

  it('busca alcança os atrasados', () => {
    const r = filtrarTransacoes(comAtraso, { ...base, busca: 'devedor' })
    expect(r).toHaveLength(1)
    expect(r[0].status).toBe('Atrasado')
  })
})

describe('classeStatus — inadimplência', () => {
  // Sem testar status antes de realizado, o atrasado cairia em pill--pend e
  // ficaria igual a um titulo que ainda nem venceu.
  it('atrasado tem classe propria, nao a de pendente', () => {
    expect(classeStatus({ tipo: 'receita', realizado: false, status: 'Atrasado' })).toBe('pill--atraso')
    expect(classeStatus({ tipo: 'despesa', realizado: false, status: 'Atrasado' })).toBe('pill--atraso')
  })

  it('atrasado nao muda a classe dos demais', () => {
    expect(classeStatus({ tipo: 'receita', realizado: false, status: 'Pendente' })).toBe('pill--pend')
    expect(classeStatus({ tipo: 'receita', realizado: true,  status: 'Recebido' })).toBe('pill--in')
    expect(classeStatus({ tipo: 'despesa', realizado: true,  status: 'Pago' })).toBe('pill--out')
  })
})

describe('resumirTransacoes', () => {
  it('separa realizado de previsto', () => {
    const r = resumirTransacoes(amostra)
    expect(r.recebido).toBe(5000)
    expect(r.a_receber).toBe(8000)
    expect(r.pago).toBe(2000)
    expect(r.a_pagar).toBe(3000)
  })

  // O ponto do ajuste: atrasado tem realizado=false e, sem o ramo proprio antes,
  // voltaria a somar em a_receber/a_pagar — somado, mas invisivel.
  it('atrasado sai de a_receber e a_pagar', () => {
    const r = resumirTransacoes(comAtraso)
    expect(r.atrasado_receber).toBe(7000)
    expect(r.atrasado_pagar).toBe(4000)
    // Os previstos continuam com os valores de antes: o atrasado nao entrou neles
    expect(r.a_receber).toBe(8000)
    expect(r.a_pagar).toBe(3000)
  })

  it('sem atraso os campos ficam zerados', () => {
    const r = resumirTransacoes(amostra)
    expect(r.atrasado_receber).toBe(0)
    expect(r.atrasado_pagar).toBe(0)
  })

  // Separar o atrasado redistribui as linhas, mas o total tem de continuar o
  // mesmo — e a regressao que protege quem ja le esse numero.
  it('o resultado nao muda ao separar o atrasado', () => {
    const r = resumirTransacoes(comAtraso)
    const entradas = r.recebido + r.a_receber + r.atrasado_receber
    const saidas   = r.pago + r.a_pagar + r.atrasado_pagar
    expect(r.resultado).toBe(entradas - saidas)
    expect(r.resultado).toBe((5000 + 8000 + 7000) - (2000 + 3000 + 4000))
  })

  // Na aba de um lado so tudo tem o mesmo sinal: subtrair nao teria contra o que.
  it('um lado so soma em vez de subtrair', () => {
    const soReceitas = comAtraso.filter(t => t.tipo === 'receita')
    const r = resumirTransacoes(soReceitas, true)
    expect(r.resultado).toBe(5000 + 8000 + 7000)
  })

  it('lista vazia devolve tudo zerado', () => {
    const r = resumirTransacoes([])
    expect(Object.values(r).every(v => v === 0)).toBe(true)
  })
})
