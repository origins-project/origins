import { describe, expect, it } from 'vitest'
import type { Finding, Indexed } from '../api/client'
import { answers, counted, familySlug, grouped, placeTree, slugify, surnames, type Branch } from '../lists'

describe('a filter', () => {
  it('asks every word somewhere in the words, case and accents ignored', () => {
    expect(answers('', ['anything'])).toBe(true)
    expect(answers('rossi giov', ['Giovanni Rossi', '1834'])).toBe(true)
    expect(answers('cambiasu', [['Cambiasù'], undefined])).toBe(true)
    expect(answers('rossi 1900', ['Giovanni Rossi', '1834'])).toBe(false)
  })
})

describe('the surnames', () => {
  it('are counted, the most first, folding case and accents together', () => {
    const people = [
      { ref: 'person/a', surname: 'Rossi' },
      { ref: 'person/b', surname: 'rossi' },
      { ref: 'person/c', surname: 'Bianchi' },
      { ref: 'person/d' },
      { ref: 'person/e', surname: 'Conti' },
    ]
    expect(surnames(people)).toEqual([
      { surname: 'Rossi', count: 2 },
      { surname: 'Bianchi', count: 1 },
      { surname: 'Conti', count: 1 },
    ])
  })
})

describe('the place tree', () => {
  const shape = (branches: Branch[]): unknown[] =>
    branches.map((b) => [b.place.ref, b.from ?? '', b.until ?? '', shape(b.under)])

  it('puts a place under every place it lay within, with when', () => {
    const places: Indexed[] = [
      { ref: 'place/borgo', name: 'Borgo San Lorenzo', within: [{ place: 'granducato', until: '1860' }, { place: 'toscana', from: '1861' }] },
      { ref: 'place/toscana', name: 'Toscana', within: [{ place: 'italia' }] },
      { ref: 'place/italia', name: 'Italia' },
      { ref: 'place/granducato', name: 'Granducato di Toscana' },
      { ref: 'place/elsewhere', name: 'Elsewhere', within: [{ place: 'not-in-the-index' }] },
    ]
    expect(shape(placeTree(places))).toEqual([
      ['place/elsewhere', '', '', []],
      ['place/granducato', '', '', [['place/borgo', '', '1860', []]]],
      ['place/italia', '', '', [['place/toscana', '', '', [['place/borgo', '1861', '', []]]]]],
    ])
  })

  it('draws a ring of places once, from its first by name', () => {
    const places: Indexed[] = [
      { ref: 'place/b', name: 'B', within: [{ place: 'a' }] },
      { ref: 'place/a', name: 'A', within: [{ place: 'b' }] },
      { ref: 'place/self', name: 'Self', within: [{ place: 'self' }] },
    ]
    expect(shape(placeTree(places))).toEqual([
      ['place/a', '', '', [['place/b', '', '', []]]],
      ['place/self', '', '', []],
    ])
  })
})

describe('the findings', () => {
  const finding = (rule: string, severity: string, entity: string, message = rule): Finding => ({ rule, severity, entity, message })
  const found = {
    findings: [
      finding('unsourced-event', 'warning', 'person/a'),
      finding('dangling-reference', 'error', 'person/b'),
      finding('unsourced-event', 'warning', 'person/c'),
    ],
    todo: [finding('unsourced-event', 'research', 'person/a'), finding('task', 'research', 'note/x', 'Look again')],
  }

  it('are grouped by severity and rule, what to do leaving out what check already said', () => {
    const g = grouped({ ...found, todo: found.todo.map((f) => ({ ...f, severity: 'warning' })) })
    expect(g.errors.map((r) => [r.rule, r.findings.length])).toEqual([['dangling-reference', 1]])
    expect(g.warnings.map((r) => [r.rule, r.findings.length])).toEqual([['unsourced-event', 2]])
    expect(g.todo.map((r) => r.rule)).toEqual(['task'])
    expect(counted(g.warnings)).toBe(2)
  })

  it('are filtered by any of their words', () => {
    const g = grouped(found, 'person/c')
    expect(counted(g.errors) + counted(g.warnings) + counted(g.todo)).toBe(1)
  })
})

describe('a slug', () => {
  it('is derived from a name as the model derives it', () => {
    expect(slugify('Rossi Giovanni')).toBe('rossi-giovanni')
    expect(slugify("D'Angelo Cambiasù")).toBe('dangelo-cambiasu')
    expect(slugify('Straße Ærø')).toBe('strasse-aero')
    expect(slugify('  ¿?  ')).toBe('')
    expect(slugify('a'.repeat(40) + ' ' + 'b'.repeat(40))).toBe('a'.repeat(40))
  })

  it('names a family by its people in order, then the words for nobody', () => {
    expect(familySlug(['rossi-giovanni', 'bianchi-maria'])).toBe('bianchi-maria--rossi-giovanni')
    expect(familySlug(['unknown', 'conti-teresa'])).toBe('conti-teresa--unknown')
  })
})
