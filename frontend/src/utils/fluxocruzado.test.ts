import { describe, it, expect } from 'vitest'
import {
  selecaoVazia, temRecorte, aplicarSelecao, alternarNumero, alternarRotulo,
  type Selecao,
} from './fluxocruzado'
import type { FluxoTransacao } from '@/api/fluxocaixa'

function t(p: Partial<FluxoTransacao>): FluxoTransacao {
  return {
    dia: 1, data: '01/09/2026', descricao: 'ACME LTDA', tipo: 'receita',
    categoria_superior: '1.01 Vendas', categoria: 'Servicos',
    valor: 100, status: 'Recebido', realizado: true, ...p,
  } as FluxoTransacao
}

const amostra: FluxoTransacao[] = [
  t({ dia: 3, categoria: 'Servicos', descricao: 'ACME LTDA', valor: 1000 }),
  t({ dia: 3, categoria: 'Produtos', descricao: 'BETA SA', valor: 2000 }),
  t({ dia: 10, categoria: 'Servicos', descricao: 'BETA SA', valor: 4000 }),
  t({ dia: 20, categoria: 'Aluguel', descricao: 'IMOB LTDA', valor: 8000, tipo: 'despesa' }),
]

const soma = (ts: FluxoTransacao[]) => ts.reduce((s, x) => s + x.valor, 0)

function sel(p: Partial<Selecao>): Selecao {
  return { ...selecaoVazia(), ...p }
}

describe('temRecorte', () => {
  it('seleção vazia não é recorte', () => {
    expect(temRecorte(selecaoVazia())).toBe(false)
  })

  it('qualquer dimensão preenchida conta', () => {
    expect(temRecorte(sel({ dias: new Set([3]) }))).toBe(true)
    expect(temRecorte(sel({ categorias: new Set(['Servicos']) }))).toBe(true)
    expect(temRecorte(sel({ entidades: new Set(['BETA SA']) }))).toBe(true)
  })
})

describe('aplicarSelecao', () => {
  // Conjunto vazio significa "tudo". Se significasse "nada", a tela abriria em
  // branco e o usuário concluiria que não há dados no mês.
  it('sem recorte devolve a lista intacta', () => {
    expect(aplicarSelecao(amostra, selecaoVazia())).toBe(amostra)
  })

  it('recorta por dia', () => {
    const r = aplicarSelecao(amostra, sel({ dias: new Set([3]) }))
    expect(r).toHaveLength(2)
    expect(soma(r)).toBe(3000)
  })

  it('múltiplos dias somam, não intersectam', () => {
    const r = aplicarSelecao(amostra, sel({ dias: new Set([3, 10]) }))
    expect(soma(r)).toBe(7000)
  })

  it('dimensões diferentes intersectam', () => {
    const r = aplicarSelecao(amostra, sel({
      dias: new Set([3, 10]),
      categorias: new Set(['Servicos']),
      entidades: new Set(['BETA SA']),
    }))
    expect(r).toHaveLength(1)
    expect(soma(r)).toBe(4000)
  })

  it('interseção vazia devolve lista vazia', () => {
    const r = aplicarSelecao(amostra, sel({
      dias: new Set([20]), categorias: new Set(['Servicos']),
    }))
    expect(r).toEqual([])
  })

  it('preserva a ordem de entrada', () => {
    const r = aplicarSelecao(amostra, sel({ entidades: new Set(['BETA SA']) }))
    expect(r.map(x => x.dia)).toEqual([3, 10])
  })

  // Esta é a regra da filtragem cruzada: um gráfico não se autofiltra. Sem ela,
  // clicar numa categoria apagaria todas as outras do donut e o gráfico deixaria
  // de servir para o clique seguinte.
  describe('ignorar a própria dimensão', () => {
    const completa = sel({
      dias: new Set([3]),
      categorias: new Set(['Servicos']),
      entidades: new Set(['ACME LTDA']),
    })

    it('ignorando categorias, as outras categorias do dia continuam presentes', () => {
      const r = aplicarSelecao(amostra, completa, 'categorias')
      // dia 3 + ACME LTDA — Produtos/BETA sai por entidade, não por categoria.
      expect(r.map(x => x.categoria)).toEqual(['Servicos'])

      const semEntidade = aplicarSelecao(amostra, sel({
        dias: new Set([3]), categorias: new Set(['Servicos']),
      }), 'categorias')
      expect(semEntidade.map(x => x.categoria).sort()).toEqual(['Produtos', 'Servicos'])
    })

    it('ignorando dias, os outros dias da categoria continuam presentes', () => {
      const r = aplicarSelecao(amostra, sel({
        dias: new Set([3]), categorias: new Set(['Servicos']),
      }), 'dias')
      expect(r.map(x => x.dia)).toEqual([3, 10])
    })

    it('ignorando entidades, as outras entidades continuam presentes', () => {
      const r = aplicarSelecao(amostra, sel({
        dias: new Set([3]), entidades: new Set(['ACME LTDA']),
      }), 'entidades')
      expect(r.map(x => x.descricao).sort()).toEqual(['ACME LTDA', 'BETA SA'])
    })

    it('ignorar a única dimensão ativa devolve a lista intacta', () => {
      const s = sel({ dias: new Set([3]) })
      expect(aplicarSelecao(amostra, s, 'dias')).toBe(amostra)
    })
  })
})

describe('alternarNumero', () => {
  it('clique simples substitui a seleção', () => {
    expect([...alternarNumero(new Set([3, 10]), 20, false)]).toEqual([20])
  })

  // Mantém o toggle que a tela já tinha: reclicar o dia marcado volta ao mês.
  it('reclicar o único dia marcado limpa', () => {
    expect(alternarNumero(new Set([3]), 3, false).size).toBe(0)
  })

  it('reclicar um dia entre vários apenas isola aquele', () => {
    expect([...alternarNumero(new Set([3, 10]), 3, false)]).toEqual([3])
  })

  it('com ctrl adiciona', () => {
    expect([...alternarNumero(new Set([3]), 10, true)].sort((a, b) => a - b)).toEqual([3, 10])
  })

  it('com ctrl remove o já marcado', () => {
    expect([...alternarNumero(new Set([3, 10]), 3, true)]).toEqual([10])
  })

  // Mutar o Set recebido não dispararia a reatividade do Vue, e a tela não
  // reagiria ao clique.
  it('não muta o conjunto recebido', () => {
    const antes = new Set([3])
    alternarNumero(antes, 10, true)
    expect([...antes]).toEqual([3])
  })
})

describe('alternarRotulo', () => {
  it('acumula sem exigir ctrl', () => {
    const a = alternarRotulo(new Set(), 'Servicos')
    const b = alternarRotulo(a, 'Produtos')
    expect([...b].sort()).toEqual(['Produtos', 'Servicos'])
  })

  it('reclicar remove', () => {
    expect(alternarRotulo(new Set(['Servicos']), 'Servicos').size).toBe(0)
  })

  it('não muta o conjunto recebido', () => {
    const antes = new Set(['Servicos'])
    alternarRotulo(antes, 'Produtos')
    expect([...antes]).toEqual(['Servicos'])
  })
})
