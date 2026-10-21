import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { fieldLabel, kinds, pattern } from '../labels'
import { kindsInOrder } from '../api/client'

// the tests run in web/, whatever directory npm was started in
const dir = join(process.cwd(), '..', 'internal', 'api', 'schema')

type Node = { [key: string]: unknown }

const schemas: { [file: string]: Node } = Object.fromEntries(
  readdirSync(dir)
    .filter((f) => f.endsWith('.json'))
    .map((f) => [f, JSON.parse(readFileSync(join(dir, f), 'utf8')) as Node]),
)

// paths is every property the schema of a file names, as a path pattern: events[].date.
function paths(file: string): string[] {
  const out = new Set<string>()
  const walk = (node: unknown, at: string, from: string, through: string[]) => {
    if (node === null || typeof node !== 'object') return
    const n = node as Node
    if (typeof n.$ref === 'string') {
      const [target = '', pointer = ''] = n.$ref.split('#')
      const into = target === '' ? from : target
      if (through.includes(into + pointer)) return
      let found: unknown = schemas[into]
      for (const step of pointer.split('/').filter(Boolean)) found = (found as Node)[step]
      walk(found, at, into, [...through, into + pointer])
    }
    for (const [name, value] of Object.entries((n.properties ?? {}) as Node)) {
      const path = at === '' ? name : `${at}.${name}`
      out.add(path)
      walk(value, path, from, through)
    }
    if (n.items) walk(n.items, `${at}[]`, from, through)
    if (n.additionalProperties && typeof n.additionalProperties === 'object')
      walk(n.additionalProperties, `${at}.*`, from, through)
    for (const key of ['oneOf', 'anyOf', 'allOf'])
      for (const member of (n[key] ?? []) as unknown[]) walk(member, at, from, through)
  }
  walk(schemas[file], '', file, [])
  return [...out]
}

describe('labels', () => {
  const files = Object.keys(schemas).filter((f) => f !== 'common.json')

  it('reads every schema', () => {
    expect(files).toHaveLength(8)
    for (const file of files) expect(paths(file).length).toBeGreaterThan(0)
  })

  for (const file of files) {
    it(`labels every property of ${file}`, () => {
      const unlabelled = paths(file).filter((path) => fieldLabel(path) === undefined)
      expect(unlabelled).toEqual([])
    })
  }

  it('names every kind', () => {
    for (const kind of kindsInOrder) {
      expect(kinds[kind].one).not.toBe('')
      expect(kinds[kind].many).not.toBe('')
    }
  })

  it('reads a path by its pattern, the longest end first', () => {
    expect(pattern('events[2].sources[0]')).toBe('events[].sources[]')
    expect(fieldLabel('events[2].date')).toBe('Date')
    expect(fieldLabel('citations.*.date')).toBe('Entry date')
    expect(fieldLabel('also-known-as[1]')).toBe('Other names')
    expect(fieldLabel('nothing-of-the-kind')).toBeUndefined()
  })
})
