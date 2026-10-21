import { globalIgnores } from 'eslint/config'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'
import pluginVitest from '@vitest/eslint-plugin'
import pluginPlaywright from 'eslint-plugin-playwright'

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{vue,ts,mts,tsx}'],
  },

  globalIgnores(['**/dist/**', '**/dist-ssr/**', '**/coverage/**', 'src/wasm/wasm_exec.js']),

  ...pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,

  {
    // The components of src/ui are the page's vocabulary, Stack, Text, Table, and are
    // always imported by name, so none is ever resolved as an element of the same name.
    rules: { 'vue/multi-word-component-names': 'off' },
  },

  {
    ...pluginVitest.configs.recommended,
    files: ['src/**/__tests__/*'],
    rules: {
      ...pluginVitest.configs.recommended.rules,
      // Vitest's expect takes a message as its second argument.
      'vitest/valid-expect': ['error', { maxArgs: 2 }],
    },
  },

  {
    ...pluginPlaywright.configs['flat/recommended'],
    files: ['e2e/**/*.{test,spec}.{js,ts,jsx,tsx}'],
    rules: {
      ...pluginPlaywright.configs['flat/recommended'].rules,
      // A spec skips a corpus that is not built.
      'playwright/no-skipped-test': ['warn', { allowConditional: true }],
      // A spec walks every width, corpus or chart in one test, and what it expects of
      // each differs.
      'playwright/no-conditional-in-test': 'off',
      'playwright/no-conditional-expect': 'off',
      // A page is measured once every request it makes has been answered, which no
      // element on it says.
      'playwright/no-networkidle': 'off',
    },
  },
)
