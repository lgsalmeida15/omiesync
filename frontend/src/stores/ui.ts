import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
  const theme          = ref<'dark' | 'light'>((localStorage.getItem('theme') as 'dark' | 'light') || 'dark')
  const sidebarPinned  = ref(localStorage.getItem('sidebar_pinned') === 'true')

  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
    document.documentElement.setAttribute('data-theme', theme.value)
    localStorage.setItem('theme', theme.value)
  }

  function toggleSidebarPin() {
    sidebarPinned.value = !sidebarPinned.value
    localStorage.setItem('sidebar_pinned', String(sidebarPinned.value))
  }

  return { theme, sidebarPinned, toggleTheme, toggleSidebarPin }
})
