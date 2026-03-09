import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeMode = 'light' | 'dark'

export const useThemeStore = defineStore('theme', () => {
  const getInitialTheme = (): ThemeMode => {
    const stored = localStorage.getItem('theme')
    if (stored === 'light' || stored === 'dark') return stored
    // Default to light (Jobs aesthetic)
    return 'light'
  }

  const theme = ref<ThemeMode>(getInitialTheme())

  const isDark = () => theme.value === 'dark'

  function setTheme(mode: ThemeMode) {
    theme.value = mode
    localStorage.setItem('theme', mode)
    applyTheme(mode)
  }

  function toggleTheme() {
    setTheme(theme.value === 'light' ? 'dark' : 'light')
  }

  function applyTheme(mode: ThemeMode) {
    document.documentElement.setAttribute('data-theme', mode)
  }

  // Apply on initialization
  applyTheme(theme.value)

  // Watch for changes
  watch(theme, (newTheme) => {
    applyTheme(newTheme)
  })

  return {
    theme,
    isDark,
    setTheme,
    toggleTheme
  }
})
