import { readFileSync, readdirSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A page only composes components: whatever it looks like, layout included, is the
// library's, so that every page looks as the others do and a theme is the tokens'
// two blocks. The frame around the pages and the forms that change a record are held
// to it as a page is. These are the ways a page could draw something itself.

// the tests run in web/, whatever directory npm was started in
const src = join(process.cwd(), 'src')

function files(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) =>
    entry.isDirectory() ? files(join(dir, entry.name)) : [join(dir, entry.name)],
  )
}

const length = String.raw`-?\d*\.?\d+(?:px|rem|em|ex|ch|vh|vw|vmin|vmax|[sdl]vh|[sdl]vw|cq[whib]|pt|pc|cm|mm|in|%)`
const hex = /#(?:[0-9a-f]{8}|[0-9a-f]{6}|[0-9a-f]{3,4})\b/i
const colourFunction = /\b(?:rgba?|hsla?|hwb|lab|lch|oklab|oklch|color-mix|color)\s*\(/i
const named =
  /\b(?:color|colour|background|fill|stroke|border|outline)[\w-]*\s*[:=]\s*["'`]?\s*(?:red|green|blue|black|white|gr[ae]y|orange|yellow|purple|pink|brown|navy|teal|maroon|olive|silver|gold|transparent|currentcolor)\b/i

// faults is what a page's file holds that draws something, by what the rule calls it.
export function faults(text: string): string[] {
  // A slot's shorthand, <template #actions>, is not a colour however it is spelled.
  const plain = text.replace(/(?<=<[^<>]*)\s#[\w-]+(?=\s|=|\/?>)/g, '')
  const out: string[] = []
  if (/<style[\s>]/i.test(plain)) out.push('a <style> block')
  if (/[\s:]style\s*=|v-bind:style\s*=/i.test(plain)) out.push('a style attribute')
  if (hex.test(plain) || colourFunction.test(plain) || named.test(plain)) out.push('a colour')
  if (/var\(\s*--/.test(plain)) out.push('a token')
  for (const binding of plain.matchAll(/(?:\s:|\sv-bind:|\s)class\s*=\s*("[^"]*"|'[^']*')/g)) {
    if (new RegExp(length, 'i').test(binding[1]!)) out.push('a length in a class binding')
  }
  return out
}

describe('TestPagesAreUnstyled', () => {
  const pages = [...files(join(src, 'pages')), ...files(join(src, 'frame')), ...files(join(src, 'edit')), join(src, 'App.vue')]

  it('finds the pages', () => {
    expect(pages.length).toBeGreaterThan(0)
  })

  for (const file of pages) {
    it(`${relative(src, file)}`, () => {
      expect(faults(readFileSync(file, 'utf8'))).toEqual([])
    })
  }

  it('catches each way a page could draw', () => {
    const drawn: [string, string][] = [
      ['<template><p>x</p></template>\n<style scoped>\np { margin: 0 }\n</style>', 'a <style> block'],
      ['<style>p{}</style>', 'a <style> block'],
      ['<div style="margin: 0"></div>', 'a style attribute'],
      ['<div :style="{ margin: 0 }"></div>', 'a style attribute'],
      ['<div v-bind:style="s"></div>', 'a style attribute'],
      ['<Tag color="#b5562f" />', 'a colour'],
      ["const tint = '#abc'", 'a colour'],
      ['<Tag :color="`rgb(1 2 3)`" />', 'a colour'],
      ["const c = 'oklch(0.7 0.1 30)'", 'a colour'],
      ['<Badge color="red" />', 'a colour'],
      ["const s = { background: 'white' }", 'a colour'],
      ['<Card :class="{ x: true }" data-c="var(--accent)" />', 'a token'],
      ['<div :class="`w-[12px]`"></div>', 'a length in a class binding'],
      ['<div :class="wide ? \'mt-[2rem]\' : \'\'"></div>', 'a length in a class binding'],
      ['<div class="h-[50vh]"></div>', 'a length in a class binding'],
      ['<div v-bind:class="[`gap-[0.5em]`]"></div>', 'a length in a class binding'],
    ]
    for (const [text, fault] of drawn) expect(faults(text), text).toContain(fault)
  })

  it('lets a page compose', () => {
    const composed = `<script setup lang="ts">
import { Card, Stack } from '../ui'
const open = ref(false)
</script>

<template>
  <Stack gap="4" row>
    <Card :title="title" :class="{ wide }">
      <template #add><a href="#">x</a></template>
      <template #actions="{ close }">{{ close }}</template>
      <template #cell-name="{ row }">{{ row.name }}</template>
    </Card>
    <Grid :cols="3" fit="card" />
  </Stack>
</template>
`
    expect(faults(composed)).toEqual([])
  })
})

// The library reads its colours from the tokens, so a theme is the two blocks of
// tokens.css and nothing else.
describe('colours are written in the tokens', () => {
  const library = files(join(src, 'ui')).filter(
    (f) => /\.(vue|css)$/.test(f) && !f.endsWith('tokens.css'),
  )
  for (const file of library) {
    it(`${relative(src, file)}`, () => {
      const text = readFileSync(file, 'utf8').replace(/(?<=<[^<>]*)\s#[\w-]+(?=\s|=|\/?>)/g, '')
      expect(text).not.toMatch(hex)
      expect(text).not.toMatch(/\b(?:rgba?|hsla?|hwb|lab|lch|oklab|oklch)\s*\(/i)
    })
  }
})
