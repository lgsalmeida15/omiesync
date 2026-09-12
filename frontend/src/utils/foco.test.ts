import { describe, it, expect } from 'vitest'
import { proximoFoco, deveMostrar } from './foco'

describe('proximoFoco', () => {
  it('sem foco, clicar abre o bloco', () => {
    expect(proximoFoco(null, 'categorias')).toBe('categorias')
  })

  it('reclicar o bloco expandido fecha', () => {
    expect(proximoFoco('categorias', 'categorias')).toBeNull()
  })

  // Dois blocos expandidos ao mesmo tempo não significam nada, e sair de um
  // para cair noutro seria confuso.
  it('clicar noutro bloco troca, não empilha', () => {
    expect(proximoFoco('categorias', 'clientes')).toBe('clientes')
  })

  it('alternar duas vezes no mesmo bloco volta ao início', () => {
    const a = proximoFoco(null, 'intradia')
    expect(proximoFoco(a, 'intradia')).toBeNull()
  })
})

describe('deveMostrar', () => {
  // Este é o caso que não pode errar: invertido, a aba abriria em branco.
  it('sem foco, todos os blocos aparecem', () => {
    for (const id of ['categorias', 'clientes', 'calendario', 'intradia', 'transacoes']) {
      expect(deveMostrar(null, id)).toBe(true)
    }
  })

  it('com foco, só o bloco em foco aparece', () => {
    expect(deveMostrar('calendario', 'calendario')).toBe(true)
    expect(deveMostrar('calendario', 'categorias')).toBe(false)
    expect(deveMostrar('calendario', 'intradia')).toBe(false)
  })

  it('id desconhecido não aparece quando há foco', () => {
    expect(deveMostrar('calendario', 'bloco-que-nao-existe')).toBe(false)
  })

  // Comparação estrita: nomes parecidos não podem se confundir.
  it('não confunde ids com prefixo em comum', () => {
    expect(deveMostrar('conta', 'contas')).toBe(false)
    expect(deveMostrar('contas', 'contas')).toBe(true)
  })
})
