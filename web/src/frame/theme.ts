import { onScopeDispose, ref } from 'vue'
import * as prefs from '../prefs'

const dark = () =>
  typeof window.matchMedia === 'function' ? window.matchMedia('(prefers-color-scheme: dark)') : undefined

// Puts the theme the browser keeps on the root, before anything is drawn.
export function applyKeptTheme() {
  const kept = prefs.theme()
  if (kept) document.documentElement.dataset.theme = kept
}

// The theme the pages are shown in, the system's until one is chosen, and switching it,
// which chooses one and keeps it.
export function useThemeSwitch() {
  const query = dark()
  const system = () => (query?.matches ? 'dark' : 'light')
  const theme = ref<prefs.ThemeName>(prefs.theme() ?? system())
  const follow = () => {
    if (!prefs.theme()) theme.value = system()
  }
  query?.addEventListener('change', follow)
  onScopeDispose(() => query?.removeEventListener('change', follow))

  function toggle() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
    prefs.keepTheme(theme.value)
    document.documentElement.dataset.theme = theme.value
  }
  return { theme, toggle }
}
