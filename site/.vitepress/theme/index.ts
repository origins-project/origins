import { h, watchEffect } from 'vue'
import { useData } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import type { Theme } from 'vitepress'
import Brand from '../../../web/src/ui/Brand.vue'
import './style.css'

export default {
  extends: DefaultTheme,
  Layout() {
    // the tokens follow data-theme, which follows the switch VitePress keeps
    const { isDark } = useData()
    watchEffect(() => {
      if (typeof document !== 'undefined') document.documentElement.dataset.theme = isDark.value ? 'dark' : 'light'
    })
    return h(DefaultTheme.Layout, null, {
      'nav-bar-title-before': () => h(Brand, { title: 'Origins' }),
    })
  },
} satisfies Theme
