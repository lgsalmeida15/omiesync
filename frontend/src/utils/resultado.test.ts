import { describe, it, expect } from 'vitest'
import { calcularResultado, chaveSuperior, larguraAoArrastar, alturaDisponivel, totalVisivel } from './resultado'
import type { PivotLinha } from '@/api/pivot'

function l(p: Partial<PivotLinha>): PivotLinha {
  return {
    tipo: 'receita', categoria_superior: 'Receitas', categoria_final: 'Vendas',
    cliente: 'ACME', meses: Array(12).fill(0), total: 0, ...p,
  } as PivotLinha
}

/** Valor só em janeiro, para facilitar a conferência mês a mês. */
function emJaneiro(v: number): number[] {
  const m = Array(12).fill(0)
  m[0] = v
  return m
}

const linhas: PivotLinha[] = [
  l({ tipo: 'receita', categoria_superior: 'Receitas',      meses: emJaneiro(10000), total: 10000 }),
  l({ tipo: 'receita', categoria_superior: 'Transferência', categoria_final: 'Transf. Inter Empresas/Contas',
      meses: emJaneiro(3000), total: 3000 }),
  l({ tipo: 'despesa', categoria_superior: 'Despesas',      meses: emJaneiro(4000), total: 4000 }),
  l({ tipo: 'despesa', categoria_superior: 'Transferência', categoria_final: 'Transf. Inter Empresas/Contas',
      meses: emJaneiro(3000), total: 3000 }),
]

describe('calcularResultado', () => {
  // Este é o teste de regressão: sem exclusão nenhuma, o valor tem de ser o mesmo
  // que o backend soma. Receita 13.000 − despesa 7.000 = 6.000.
  it('sem exclusões reproduz a soma do servidor', () => {
    const r = calcularResultado(linhas)
    expect(r.total).toBe(6000)
    expect(r.meses[0]).toBe(6000)
  })

  it('lista vazia devolve zeros nos doze meses', () => {
    const r = calcularResultado([])
    expect(r.total).toBe(0)
    expect(r.meses).toHaveLength(12)
    expect(r.meses.every(v => v === 0)).toBe(true)
  })

  // A mesma superior existe sob receita e sob despesa. Desmarcar a de receita não
  // pode levar a de despesa junto — foi por isso que a chave carrega o tipo.
  it('exclui só o tipo pedido quando a superior existe nos dois', () => {
    const r = calcularResultado(linhas, new Set([chaveSuperior('receita', 'Transferência')]))
    // Sai 3.000 de receita: 10.000 − 7.000 = 3.000
    expect(r.total).toBe(3000)
  })

  it('excluir uma despesa AUMENTA o resultado', () => {
    const semExclusao = calcularResultado(linhas).total
    const r = calcularResultado(linhas, new Set([chaveSuperior('despesa', 'Transferência')]))
    expect(r.total).toBe(9000)
    expect(r.total).toBeGreaterThan(semExclusao)
  })

  it('excluir uma receita DIMINUI o resultado', () => {
    const semExclusao = calcularResultado(linhas).total
    const r = calcularResultado(linhas, new Set([chaveSuperior('receita', 'Receitas')]))
    expect(r.total).toBeLessThan(semExclusao)
  })

  // É o caso de uso que o usuário descreveu: desmarcando Transferência dos dois
  // lados, o número volta a ser o de antes de ela passar a somar.
  it('excluir Transferência dos dois tipos devolve o resultado sem ela', () => {
    const r = calcularResultado(linhas, new Set([
      chaveSuperior('receita', 'Transferência'),
      chaveSuperior('despesa', 'Transferência'),
    ]))
    expect(r.total).toBe(6000)   // 10.000 − 4.000
  })

  it('excluir tudo zera', () => {
    const todas = new Set(linhas.map(x => chaveSuperior(x.tipo, x.categoria_superior)))
    const r = calcularResultado(linhas, todas)
    expect(r.total).toBe(0)
    expect(r.meses.every(v => v === 0)).toBe(true)
  })

  it('exclusão desconhecida não altera nada', () => {
    const r = calcularResultado(linhas, new Set(['receita|Nao Existe']))
    expect(r.total).toBe(6000)
  })

  // O ELSE do ajuste_receita_despesa no servidor manda o não classificado para
  // despesa. Divergir aqui faria o rodapé discordar do resto do dashboard.
  it('tipo não classificado subtrai, como no servidor', () => {
    const r = calcularResultado([
      l({ tipo: 'receita', meses: emJaneiro(1000), total: 1000 }),
      l({ tipo: 'nao classificado', categoria_superior: 'Outros', meses: emJaneiro(400), total: 400 }),
    ])
    expect(r.total).toBe(600)
  })

  it('soma mês a mês, não só o total', () => {
    const dez = Array(12).fill(0); dez[11] = 500
    const r = calcularResultado([l({ tipo: 'receita', meses: dez, total: 500 })])
    expect(r.meses[11]).toBe(500)
    expect(r.meses[0]).toBe(0)
  })

  // Sem cópia, somar sobre o array de entrada corromperia os dados do pivô a cada
  // clique do usuário.
  it('não modifica as linhas recebidas', () => {
    const original = linhas.map(x => [...x.meses])
    calcularResultado(linhas, new Set([chaveSuperior('despesa', 'Despesas')]))
    expect(linhas.map(x => x.meses)).toEqual(original)
  })
})

describe('chaveSuperior', () => {
  it('separa tipo de categoria', () => {
    expect(chaveSuperior('receita', 'Transferência')).toBe('receita|Transferência')
    expect(chaveSuperior('receita', 'X')).not.toBe(chaveSuperior('despesa', 'X'))
  })
})

describe('larguraAoArrastar', () => {
  it('soma o deslocamento à largura inicial', () => {
    expect(larguraAoArrastar(280, 40)).toBe(320)
    expect(larguraAoArrastar(280, -40)).toBe(240)
  })

  // Sem o piso, arrastar para a esquerda alem da origem daria largura negativa e
  // a coluna sumiria sem como recuperar.
  it('nunca desce abaixo do mínimo', () => {
    expect(larguraAoArrastar(280, -500)).toBe(60)
    expect(larguraAoArrastar(100, -100, 80)).toBe(80)
  })

  it('arredonda para pixel inteiro', () => {
    expect(larguraAoArrastar(280, 10.6)).toBe(291)
  })

  it('deslocamento zero devolve a largura inicial', () => {
    expect(larguraAoArrastar(280, 0)).toBe(280)
  })
})

describe('alturaDisponivel', () => {
  it('ocupa o que sobra abaixo do topo, descontando a folga', () => {
    expect(alturaDisponivel(240, 900)).toBe(628)   // 900 − 240 − 32
  })

  // É o ganho do modo foco: sem a faixa de abas o topo sobe e a tabela cresce
  // na mesma medida — o que o número fixo não fazia.
  it('topo mais alto na tela rende mais altura', () => {
    const normal = alturaDisponivel(240, 900)
    const foco   = alturaDisponivel(160, 900)
    expect(foco).toBeGreaterThan(normal)
    expect(foco - normal).toBe(80)
  })

  it('janela maior rende mais altura', () => {
    expect(alturaDisponivel(240, 1200) - alturaDisponivel(240, 900)).toBe(300)
  })

  // Sem o piso, janela baixa ou topo grande fariam a tabela sumir.
  it('nunca desce abaixo do mínimo', () => {
    expect(alturaDisponivel(240, 400)).toBe(320)
    expect(alturaDisponivel(900, 500)).toBe(320)
  })

  it('nunca devolve altura negativa', () => {
    expect(alturaDisponivel(2000, 600)).toBeGreaterThan(0)
  })

  it('folga e mínimo são configuráveis', () => {
    expect(alturaDisponivel(240, 900, 0)).toBe(660)
    expect(alturaDisponivel(240, 400, 32, 100)).toBe(128)
  })
})

describe('calcularResultado — meses visíveis', () => {
  // Linhas com valor em janeiro e em fevereiro, para recortar entre eles.
  const doisMeses: PivotLinha[] = [
    l({ tipo: 'receita', meses: [100, 200, ...Array(10).fill(0)], total: 300 }),
    l({ tipo: 'despesa', categoria_superior: 'Despesas',
        meses: [40, 60, ...Array(10).fill(0)], total: 100 }),
  ]

  it('sem recorte soma o ano, como antes', () => {
    expect(calcularResultado(doisMeses).total).toBe(200)   // 300 − 100
  })

  it('recorte soma só os meses escolhidos', () => {
    expect(calcularResultado(doisMeses, new Set(), new Set([1])).total).toBe(60)   // 100 − 40
    expect(calcularResultado(doisMeses, new Set(), new Set([2])).total).toBe(140)  // 200 − 60
  })

  // O ponto do ajuste: o total nao pode vir de l.total, que e sempre o ano
  // fechado — sob um recorte ele mostraria o anual embaixo de colunas parciais.
  it('o total ignora l.total e soma os meses', () => {
    const mentiroso = [l({ tipo: 'receita', meses: [10, ...Array(11).fill(0)], total: 99999 })]
    expect(calcularResultado(mentiroso, new Set(), new Set([1])).total).toBe(10)
  })

  it('meses fora do recorte ficam zerados no vetor', () => {
    const r = calcularResultado(doisMeses, new Set(), new Set([1]))
    expect(r.meses[0]).toBe(60)
    expect(r.meses[1]).toBe(0)
  })

  it('recorte vazio zera tudo', () => {
    const r = calcularResultado(doisMeses, new Set(), new Set())
    expect(r.total).toBe(0)
    expect(r.meses.every(v => v === 0)).toBe(true)
  })

  it('recorte e exclusão de categoria se combinam', () => {
    const r = calcularResultado(doisMeses, new Set([chaveSuperior('despesa', 'Despesas')]), new Set([1]))
    expect(r.total).toBe(100)   // só a receita de janeiro
  })
})

describe('totalVisivel', () => {
  it('soma só os meses escolhidos', () => {
    expect(totalVisivel([10, 20, 30, ...Array(9).fill(0)], new Set([1, 3]))).toBe(40)
  })

  it('sem recorte soma tudo', () => {
    expect(totalVisivel([10, 20, 30, ...Array(9).fill(0)])).toBe(60)
  })

  it('recorte vazio devolve zero', () => {
    expect(totalVisivel([10, 20], new Set())).toBe(0)
  })
})
