import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type Tema = 'dark' | 'light'

function temaSalvo(): Tema {
  return localStorage.getItem('theme') === 'light' ? 'light' : 'dark'
}

export const useUiStore = defineStore('ui', () => {
  const theme         = ref<Tema>(temaSalvo())
  const sidebarPinned = ref(localStorage.getItem('sidebar_pinned') === 'true')

  /**
   * Barra de filtros recolhida por padrão: fechada o cabeçalho fixo devolve
   * ~55px de altura ao conteúdo. A preferência persiste para quem trabalha
   * com os filtros sempre à vista.
   */
  const filtrosAbertos = ref(localStorage.getItem('filtros_abertos') === 'true')

  function toggleFiltros() {
    filtrosAbertos.value = !filtrosAbertos.value
    localStorage.setItem('filtros_abertos', String(filtrosAbertos.value))
  }

  /**
   * O atributo no <html> e o localStorage acompanham o ref por watcher, não por
   * atribuição dentro do toggle. Assim qualquer caminho que mude `theme` — e não
   * só o botão — mantém DOM e armazenamento em sincronia.
   *
   * `immediate` cobre o caso de o valor salvo divergir do atributo que o script
   * anti-FOUC do index.html aplicou.
   */
  watch(theme, t => {
    document.documentElement.setAttribute('data-theme', t)
    localStorage.setItem('theme', t)
  }, { immediate: true })

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  function toggleSidebarPin() {
    sidebarPinned.value = !sidebarPinned.value
    localStorage.setItem('sidebar_pinned', String(sidebarPinned.value))
  }

  return { theme, sidebarPinned, filtrosAbertos, toggleTheme, toggleSidebarPin, toggleFiltros }
})
