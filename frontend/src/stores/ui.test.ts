// @vitest-environment jsdom
// A store le e grava localStorage ja na criacao (limpeza de chave morta, tema,
// filtros), entao precisa de DOM mesmo para testar estado que nao persiste.
import { describe, it, expect, beforeEach } from 'vitest'
import { nextTick } from 'vue'
import { setActivePinia, createPinia } from 'pinia'
import { useUiStore } from './ui'

/**
 * Testa a store direto, sem montar componente — o projeto não tem
 * @vue/test-utils e o modo foco não precisa dele: a regra é estado, não
 * renderização.
 */
describe('store ui — modo foco', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('começa sem foco', () => {
    const ui = useUiStore()
    expect(ui.foco).toBeNull()
    expect(ui.haFoco).toBe(false)
  })

  it('alternar abre o bloco', () => {
    const ui = useUiStore()
    ui.alternarFoco('categorias')
    expect(ui.foco).toBe('categorias')
    expect(ui.haFoco).toBe(true)
    expect(ui.emFoco('categorias')).toBe(true)
    expect(ui.emFoco('clientes')).toBe(false)
  })

  it('alternar o mesmo bloco fecha', () => {
    const ui = useUiStore()
    ui.alternarFoco('categorias')
    ui.alternarFoco('categorias')
    expect(ui.foco).toBeNull()
    expect(ui.haFoco).toBe(false)
  })

  it('só um bloco por vez', () => {
    const ui = useUiStore()
    ui.alternarFoco('categorias')
    ui.alternarFoco('intradia')
    expect(ui.foco).toBe('intradia')
    expect(ui.emFoco('categorias')).toBe(false)
  })

  it('sairDoFoco fecha de qualquer bloco', () => {
    const ui = useUiStore()
    ui.alternarFoco('transacoes')
    ui.sairDoFoco()
    expect(ui.foco).toBeNull()
  })

  it('sairDoFoco sem foco ativo não quebra', () => {
    const ui = useUiStore()
    ui.sairDoFoco()
    expect(ui.foco).toBeNull()
  })

  /*
   * O foco é modo de trabalho, não preferência. Persistido, o usuário voltaria
   * dias depois a uma tela sem abas sem lembrar como entrou.
   */
  it('não persiste no localStorage', () => {
    const ui = useUiStore()
    ui.alternarFoco('calendario')

    const chaves = Object.keys(localStorage)
    expect(chaves.some(k => /foco/i.test(k))).toBe(false)
    expect(JSON.stringify(localStorage)).not.toContain('calendario')
  })

  it('uma store nova abre sem foco, mesmo depois de outra ter focado', () => {
    const ui = useUiStore()
    ui.alternarFoco('calendario')

    setActivePinia(createPinia())
    expect(useUiStore().foco).toBeNull()
  })
})

// As preferências de verdade continuam persistindo — o teste acima não pode ter
// sido satisfeito quebrando a persistência de quem depende dela.
describe('store ui — o que ainda persiste', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('centavos e filtros gravam no proprio toggle', () => {
    const ui = useUiStore()
    ui.toggleCentavos()
    ui.toggleFiltros()

    expect(localStorage.getItem('mostrar_centavos')).toBe('false')
    expect(localStorage.getItem('filtros_abertos')).toBe('true')
  })

  // O tema grava por WATCHER, nao dentro do toggle — entao a gravacao so
  // acontece no tick seguinte. Sem o await, a assercao le o valor que o
  // `immediate: true` escreveu na criacao da store.
  it('o tema grava no tick seguinte, via watcher', async () => {
    const ui = useUiStore()
    ui.toggleTheme()
    expect(ui.theme).toBe('light')

    await nextTick()
    expect(localStorage.getItem('theme')).toBe('light')
    expect(document.documentElement.getAttribute('data-theme')).toBe('light')
  })
})
