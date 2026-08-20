import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { fileURLToPath } from 'node:url'

// fileURLToPath e não .pathname: o caminho do projeto tem espaço ("OneDrive - "),
// que vem percent-encoded na URL e faz o readFileSync falhar com ENOENT.
const RAIZ = fileURLToPath(new URL('..', import.meta.url))
const TOKENS = join(RAIZ, 'assets', 'tokens.css')

/**
 * Contrato dos design tokens.
 *
 * O frontend não tem teste de componente nenhum, então a migração visual
 * dependeria só de olhar a tela. Estas asserções cobrem a parte que dá para
 * verificar mecanicamente: todo token usado existe, existe nos DOIS temas, e a
 * quantidade de cor fixa fora de tokens.css só pode cair.
 */

function arquivosFonte(): string[] {
  const out: string[] = []
  const anda = (dir: string) => {
    for (const nome of readdirSync(dir)) {
      const caminho = join(dir, nome)
      if (statSync(caminho).isDirectory()) { anda(caminho); continue }
      if (/\.(vue|ts)$/.test(nome) && !/\.test\.ts$/.test(nome)) out.push(caminho)
    }
  }
  anda(RAIZ)
  return out
}

const css = readFileSync(TOKENS, 'utf8')

/**
 * Extrai os nomes declarados em TODOS os blocos cujo seletor contém `seletor`.
 *
 * Percorrer todas as ocorrências é necessário: `:root` aparece em mais de um
 * bloco (tema claro, globais e ponte de compatibilidade), e ler só o primeiro
 * dava 28 falsos órfãos.
 */
function declaradosEm(seletor: string): Set<string> {
  // Remove @media e @keyframes antes de fatiar em regras: os blocos aninhados
  // desses at-rules confundem o casamento simples de chaves.
  const plano = css.replace(/@(media|keyframes)[^{]*\{(?:[^{}]*\{[^{}]*\})*[^{}]*\}/g, '')
  const nomes = new Set<string>()
  for (const m of plano.matchAll(/([^{}]+)\{([^{}]*)\}/g)) {
    if (!m[1].includes(seletor)) continue
    for (const d of m[2].matchAll(/--([a-z0-9-]+)\s*:/g)) nomes.add(d[1])
  }
  return nomes
}

const noRoot  = declaradosEm(':root')
const noDark  = declaradosEm('[data-theme="dark"]')
const noLight = declaradosEm('[data-theme="light"]')
const declarados = new Set([...noRoot, ...noDark, ...noLight])

const fontes = arquivosFonte()

describe('tokens: declaração', () => {
  it('define os dois temas', () => {
    expect(noDark.size).toBeGreaterThan(0)
    expect(noLight.size).toBeGreaterThan(0)
  })

  it('todo token de cor existe nos dois temas', () => {
    // Tokens em :root são globais (tipografia, espaçamento) e não precisam
    // duplicação. O que está num tema tem de estar no outro, senão a troca
    // deixa a propriedade sem valor e o navegador cai no fallback.
    const soNoDark  = [...noDark].filter(t => !noLight.has(t) && !noRoot.has(t))
    const soNoLight = [...noLight].filter(t => !noDark.has(t) && !noRoot.has(t))
    expect({ soNoDark, soNoLight }).toEqual({ soNoDark: [], soNoLight: [] })
  })
})

describe('tokens: uso', () => {
  const usados = new Map<string, string[]>()
  for (const arq of fontes) {
    const texto = readFileSync(arq, 'utf8')
    for (const m of texto.matchAll(/var\(\s*--([a-z0-9-]+)/g)) {
      const lista = usados.get(m[1]) ?? []
      lista.push(relative(RAIZ, arq))
      usados.set(m[1], lista)
    }
  }

  it('todo var(--x) referenciado está declarado em tokens.css', () => {
    const orfaos = [...usados.entries()]
      .filter(([nome]) => !declarados.has(nome))
      .map(([nome, arqs]) => `${nome} (${[...new Set(arqs)].join(', ')})`)
    expect(orfaos).toEqual([])
  })

  it('há uso real de tokens (o teste não passou por vacuidade)', () => {
    expect(usados.size).toBeGreaterThan(10)
  })
})

describe('tokens: escala tipográfica', () => {
  it('nenhum font-size em px cru — todos passam pela escala', () => {
    const crus: string[] = []
    for (const arq of fontes) {
      for (const m of readFileSync(arq, 'utf8').matchAll(/font-size:\s*(\d+)px/g)) {
        crus.push(`${relative(RAIZ, arq)}: ${m[1]}px`)
      }
    }
    expect(crus).toEqual([])
  })

  it('a escala define os sete degraus e o piso é 12px', () => {
    const escala = [...css.matchAll(/--fs-([a-z0-9]+):\s*([0-9.]+)rem/g)]
      .map(m => ({ nome: m[1], px: parseFloat(m[2]) * 16 }))
    expect(escala).toHaveLength(7)
    expect(Math.min(...escala.map(e => e.px))).toBe(12)
  })
})

describe('tokens: erradicação de cor fixa', () => {
  // Teto que só desce. Serve de catraca: uma cor fixa nova quebra o teste, e
  // cada arquivo migrado permite baixar o número — sem exigir a limpeza toda
  // de uma vez.
  const TETO_HEX  = 22
  const TETO_RGBA = 12

  const conta = (re: RegExp) => fontes.reduce((soma, arq) => {
    const texto = readFileSync(arq, 'utf8')
    return soma + [...texto.matchAll(re)].length
  }, 0)

  it(`no máximo ${TETO_HEX} cores hex fora de tokens.css`, () => {
    expect(conta(/#[0-9a-fA-F]{3,8}\b/g)).toBeLessThanOrEqual(TETO_HEX)
  })

  it(`no máximo ${TETO_RGBA} rgb/rgba fora de tokens.css`, () => {
    expect(conta(/rgba?\(/g)).toBeLessThanOrEqual(TETO_RGBA)
  })

  // Classes utilitárias do Tailwind com cor fixa (bg-white, text-gray-500...).
  // São cor fixa que nenhum grep de hex/rgba encontra e que NÃO seguem o tema:
  // um bg-white continua branco no escuro.
  const TETO_TAILWIND = 0
  it(`no máximo ${TETO_TAILWIND} classes Tailwind de cor fixa`, () => {
    const re = /\b(bg|text|border|divide)-(gray|blue|red|green|indigo|orange|yellow|amber|white)(-\d{2,3})?\b/g
    expect(conta(re)).toBeLessThanOrEqual(TETO_TAILWIND)
  })
})
