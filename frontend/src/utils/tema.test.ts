// @vitest-environment jsdom
// Único arquivo de teste que precisa de DOM: getComputedStyle não tem
// equivalente fora do navegador, e é justamente o que este módulo encapsula.
import { describe, it, expect, afterEach } from 'vitest'
import { token, comAlfa, coresGrafico } from './tema'

function comEstilo(css: string): HTMLElement {
  const style = document.createElement('style')
  style.textContent = css
  document.head.appendChild(style)
  return document.documentElement
}

afterEach(() => {
  document.head.querySelectorAll('style').forEach(s => s.remove())
  document.documentElement.removeAttribute('data-theme')
})

describe('token', () => {
  it('lê o valor declarado na raiz', () => {
    const raiz = comEstilo(':root { --cor-teste: #123456; }')
    expect(token('--cor-teste', '#000', raiz)).toBe('#123456')
  })

  it('cai na alternativa quando o token não existe', () => {
    expect(token('--nao-existe', '#abcdef')).toBe('#abcdef')
  })

  it('cai na alternativa quando o token existe vazio', () => {
    const raiz = comEstilo(':root { --vazio: ; }')
    expect(token('--vazio', '#fallback', raiz)).toBe('#fallback')
  })

  it('remove espaços em volta do valor', () => {
    const raiz = comEstilo(':root { --espaco:   #ff0000  ; }')
    expect(token('--espaco', '#000', raiz)).toBe('#ff0000')
  })
})

describe('comAlfa', () => {
  it('converte hex de 6 dígitos', () => {
    expect(comAlfa('#22c55e', 0.25)).toBe('rgba(34, 197, 94, 0.25)')
  })

  it('expande hex de 3 dígitos', () => {
    expect(comAlfa('#f00', 1)).toBe('rgba(255, 0, 0, 1)')
  })

  it('aceita maiúsculas', () => {
    expect(comAlfa('#EF4444', 0.2)).toBe('rgba(239, 68, 68, 0.2)')
  })

  it('ignora espaços em volta', () => {
    expect(comAlfa('  #22c55e  ', 0.5)).toBe('rgba(34, 197, 94, 0.5)')
  })

  // Um token pode guardar rgba() ou um nome de cor; devolver a entrada mantém
  // uma cor válida no gráfico em vez de produzir "rgba(NaN, ...)".
  it('devolve inalterado o que não for hex', () => {
    expect(comAlfa('rgba(0,0,0,0.5)', 0.2)).toBe('rgba(0,0,0,0.5)')
    expect(comAlfa('transparent', 0.2)).toBe('transparent')
    expect(comAlfa('#12345', 0.2)).toBe('#12345')
  })
})

describe('coresGrafico', () => {
  // O motivo do módulo existir: nada que chega ao canvas pode ser var().
  it('nunca devolve var() em nenhum campo', () => {
    for (const [campo, valor] of Object.entries(coresGrafico())) {
      expect(valor, `campo ${campo}`).not.toContain('var(')
      expect(valor, `campo ${campo}`).not.toBe('')
    }
  })

  it('usa o token quando ele existe', () => {
    const raiz = comEstilo(':root { --success: #00ff00; --danger: #ff0000; }')
    const c = coresGrafico(raiz)
    expect(c.receita).toBe('#00ff00')
    expect(c.despesa).toBe('#ff0000')
  })

  it('preenche todos os campos mesmo sem CSS carregado', () => {
    const c = coresGrafico()
    const esperados = ['receita', 'despesa', 'linha', 'grade', 'tick', 'rotulo',
                       'fundo', 'tooltipFundo', 'tooltipBorda', 'tooltipTexto', 'fonte']
    expect(Object.keys(c).sort()).toEqual(esperados.sort())
  })

  // Receita e despesa lado a lado no mesmo gráfico: se colidirem, as barras
  // deixam de ser distinguíveis.
  it('receita e despesa são cores distintas', () => {
    const c = coresGrafico()
    expect(c.receita).not.toBe(c.despesa)
  })
})
