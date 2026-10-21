import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

// the tests run in web/, whatever directory npm was started in
const css = readFileSync(join(process.cwd(), 'src', 'ui', 'tokens.css'), 'utf8')

function block(selector: string): { [token: string]: string } {
  const start = css.indexOf(`${selector} {`)
  expect(start, selector).toBeGreaterThanOrEqual(0)
  const body = css.slice(start, css.indexOf('}', start))
  return Object.fromEntries([...body.matchAll(/--([a-z0-9-]+):\s*(#[0-9a-f]{3,6});/g)].map((m) => [m[1], m[2]]))
}

const themes = {
  light: block(":root,\n[data-theme='light']"),
  dark: block("[data-theme='dark']"),
}

function luminance(hex: string): number {
  const full = hex.length === 4 ? hex.replace(/[0-9a-f]/g, (c) => c + c).slice(1) : hex.slice(1)
  const [r, g, b] = [0, 2, 4].map((i) => {
    const c = parseInt(full.slice(i, i + 2), 16) / 255
    return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r! + 0.7152 * g! + 0.0722 * b!
}

function ratio(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi! + 0.05) / (lo! + 0.05)
}

// every colour text is set in, on every background it is set on
const grounds = ['bg', 'surface', 'surface-2']
const pairs: [string, string][] = [
  ...['ink', 'muted', 'faint', 'accent', 'suppose', 'good', 'warn', 'danger'].flatMap((text) =>
    grounds.map((ground): [string, string] => [text, ground]),
  ),
  ['ink', 'accent-soft'],
  ['ink', 'suppose-soft'],
  ['accent', 'accent-soft'],
  ['suppose', 'suppose-soft'],
  ['good', 'good-soft'],
  ['warn', 'warn-soft'],
  ['danger', 'danger-soft'],
  ['on-accent', 'accent'],
  ['on-accent', 'accent-hover'],
  ['on-suppose', 'suppose'],
  ['on-suppose', 'suppose-hover'],
  ['on-warn', 'warn'],
  ['on-danger', 'danger'],
  ['on-brand', 'brand'],
  ['male-ink', 'male'],
  ['female-ink', 'female'],
  ['unknown-ink', 'unknown'],
]

describe('the tokens', () => {
  it('give the dark theme the same colours whether the system or the page asks for it', () => {
    expect(block(":root:not([data-theme='light'])")).toEqual(themes.dark)
  })

  for (const [theme, tokens] of Object.entries(themes)) {
    it.each(pairs)(`set %s on %s at WCAG AA in the ${theme} theme`, (text, ground) => {
      expect(tokens[text], text).toBeDefined()
      expect(tokens[ground], ground).toBeDefined()
      expect(ratio(tokens[text]!, tokens[ground]!)).toBeGreaterThanOrEqual(4.5)
    })
  }
})
