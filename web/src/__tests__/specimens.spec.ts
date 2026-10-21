import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

// /ui is where a component is seen in every state, so every component the library
// exports is drawn there.

const src = join(process.cwd(), 'src')

describe('TestEveryComponentHasASpecimen', () => {
  const exported = [...readFileSync(join(src, 'ui', 'index.ts'), 'utf8').matchAll(/^export \{ default as (\w+) \}/gm)].map((m) => m[1]!)
  const specimens = readFileSync(join(src, 'pages', 'ui', 'Specimens.vue'), 'utf8')

  it('reads the components the library exports', () => {
    expect(exported.length).toBeGreaterThan(40)
  })

  it.each(exported)('draws %s', (name) => {
    expect(specimens).toMatch(new RegExp(`<${name}[\\s/>]`))
  })
})
