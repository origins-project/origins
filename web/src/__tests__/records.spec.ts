import { describe, expect, it } from 'vitest'
import type { Index, Record } from '../api/client'
import {
  addressOf,
  anglesOf,
  citationOf,
  citedBy,
  concerned,
  depictedOf,
  drafts,
  editOf,
  entryOf,
  eventsAt,
  linkOf,
  refersTo,
  within,
} from '../records'

function record(ref: string, fields: Record['fields'], more: Partial<Record> = {}): Record {
  return { ref, fields, seen: '', file: '', attached: [], edits: [], sorted: {}, html: {}, ...more }
}

const giovanni = record(
  'person/rossi-giovanni',
  {
    events: [
      { type: 'birth', date: '12 MAR 1834', place: 'borgo-san-lorenzo', sources: ['parish#1834-042 OSP'] },
      { type: 'residence', place: 'firenze' },
      { type: 'death', date: '1890', place: 'firenze', sources: ['register'] },
    ],
    sources: ['census#f088'],
  },
  { sorted: { 'events[0]': '1834-03-12', 'events[2]': '1890' } },
)
const family = record(
  'family/bianchi-maria--rossi-giovanni',
  { events: [{ type: 'marriage', date: '1858', place: 'firenze', sources: ['parish#1858-112 OSP', 'census#f088 OSI'] }] },
  { sorted: { 'events[0]': '1858' }, html: { body: '<p>See <a href="/note/a-wedding#banns">the note</a>.</p>' } },
)

describe('a citation', () => {
  it('is read in its parts', () => {
    expect(citationOf('parish#1834-042 OSP')).toEqual({ source: 'parish', anchor: '1834-042', grade: 'OSP' })
    expect(citationOf('register')).toEqual({ source: 'register', anchor: '', grade: '' })
  })

  it('is found wherever the records around a source cite it, by its anchor', () => {
    const by = citedBy([giovanni, family], 'census')
    expect([...by.keys()]).toEqual(['f088'])
    expect(by.get('f088')!.map((c) => [c.record.ref, c.path, c.grade])).toEqual([
      ['person/rossi-giovanni', '', ''],
      ['family/bianchi-maria--rossi-giovanni', 'events[0]', 'OSI'],
    ])
    expect(citedBy([giovanni], 'register').get('')!.map((c) => c.path)).toEqual(['events[2]'])
  })
})

describe('a place', () => {
  it('has the events of the records around it that happened there, dated first', () => {
    const at = eventsAt([family, giovanni], 'firenze')
    expect(at.map((e) => `${e.record.ref} ${e.path}`)).toEqual([
      'family/bianchi-maria--rossi-giovanni events[0]',
      'person/rossi-giovanni events[2]',
      'person/rossi-giovanni events[1]',
    ])
  })

  it('contains the places the index says lie within it', () => {
    const index = {
      places: [
        { ref: 'place/firenze', within: [{ place: 'toscana' }] },
        { ref: 'place/prato', within: [{ place: 'toscana', from: '1861' }] },
        { ref: 'place/toscana' },
      ],
    } as unknown as Index
    expect(within(index, 'toscana').map((p) => p.ref)).toEqual(['place/firenze', 'place/prato'])
  })

  it('is on a map where its coordinates are two angles', () => {
    expect(anglesOf('43.7696, 11.2558')).toEqual({ latitude: 43.7696, longitude: 11.2558 })
    expect(anglesOf('+43.5,-.5')).toEqual({ latitude: 43.5, longitude: -0.5 })
    expect(anglesOf("(43°48'N 11°17'E)")).toBeUndefined()
    expect(anglesOf('95, 11')).toBeUndefined()
  })
})

describe('a record', () => {
  it('refers to another by a field naming it or a link to it in its prose', () => {
    expect(refersTo(family, 'note', 'a-wedding')).toBe(true)
    expect(refersTo(family, 'note', 'a-wedding-2')).toBe(false)
    expect(refersTo(giovanni, 'place', 'firenze')).toBe(true)
    expect(refersTo(giovanni, 'source', 'parish')).toBe(true)
    expect(refersTo(giovanni, 'place', 'prato')).toBe(false)
  })

  it('writes an address and a link on one line', () => {
    expect(addressOf('Viale Giovine Italia 6\n50122 Firenze\n')).toBe('Viale Giovine Italia 6, 50122 Firenze')
    expect(addressOf({ city: 'Fiesole', parish: 'San Michele', note: 'kept' })).toBe('Fiesole, San Michele')
    expect(linkOf('https://example.org')).toEqual({ url: 'https://example.org', title: 'https://example.org' })
    expect(linkOf({ url: 'tel:+39', title: 'Desk' })).toEqual({ url: 'tel:+39', title: 'Desk' })
  })
})

describe('a hypothesis', () => {
  const edits = [
    { op: 'create', entity: 'person/rossi-unknown-1845', value: { name: { surname: 'Rossi' } } },
    { op: 'add', entity: 'family/rossi-antonio--verdi-lucia', path: 'children', value: 'rossi-unknown-1845' },
    { op: 'merge', from: 'person/a', into: 'person/b' },
    { op: 'rename', entity: 'person/a', to: 'c' },
    { op: 'split', entity: 'person/b', into: ['person/b', 'person/d'] },
  ]

  it('shows each edit by what it does and to which records', () => {
    expect(editOf(edits[1]!)).toEqual({
      op: 'add',
      entities: ['family/rossi-antonio--verdi-lucia'],
      path: 'children',
      value: 'rossi-unknown-1845',
    })
    expect(editOf(edits[3]!).entities).toEqual(['person/a', 'person/c'])
  })

  it('changes every record its edits are about, each once', () => {
    expect(concerned(edits)).toEqual([
      'person/rossi-unknown-1845',
      'family/rossi-antonio--verdi-lucia',
      'person/a',
      'person/b',
      'person/c',
      'person/d',
    ])
  })
})

describe('a picture', () => {
  const fields = {
    media: ['photos/loose.jpg', { file: 'photos/wedding.jpg', depicts: ['marchi-luca'] }],
    events: [{ type: 'burial', media: [{ file: 'photos/gravestone.jpg' }] }],
    sources: ['census#f088'],
  }

  it('is found in whichever list of the record keeping it names it', () => {
    expect(entryOf(fields, 'photos/loose.jpg')).toEqual({ path: 'media[0]', entry: 'photos/loose.jpg' })
    expect(entryOf(fields, 'photos/wedding.jpg')?.path).toBe('media[1]')
    expect(entryOf(fields, 'photos/gravestone.jpg')?.path).toBe('events[0].media[0]')
    expect(entryOf(fields, 'photos/elsewhere.jpg')).toBeUndefined()
  })

  it('says who is in it, by slug or with the part of it they are in', () => {
    const entry = {
      file: 'photos/wedding.jpg',
      depicts: [
        'marchi-luca',
        { person: 'costa-irene', region: ['0.1', '0.2', '0.4', '0.9'], note: 'On the left.' },
        { person: 'rossi-elena', region: ['0.1', 'x'] },
        { region: ['0', '0', '1', '1'] },
      ],
    }
    expect(depictedOf(entry)).toEqual([
      { at: 0, person: 'marchi-luca', note: '' },
      { at: 1, person: 'costa-irene', region: [0.1, 0.2, 0.4, 0.9], note: 'On the left.' },
      { at: 2, person: 'rossi-elena', region: undefined, note: '' },
    ])
    expect(depictedOf('photos/loose.jpg')).toEqual([])
  })
})

describe('a draft', () => {
  const fields = { sex: 'M', tags: ['a', 'b', 'c'], events: [{ type: 'birth', date: '1834' }] }
  const edits = [
    { op: 'set', entity: 'person/x', path: 'events[0].date', value: '1835' },
    { op: 'add', entity: 'person/x', path: 'tags', value: 'c' },
    { op: 'set', entity: 'person/y', path: 'sex', value: 'F' },
    { op: 'create', entity: 'person/z', value: {} },
  ]

  it('marks a value it sets, and what holds it', () => {
    expect(drafts(edits, 'person/x', 'events[0].date', fields)).toBe(true)
    expect(drafts(edits, 'person/x', 'events[0]', fields)).toBe(true)
    expect(drafts(edits, 'person/x', 'events[0].type', fields)).toBe(false)
    expect(drafts(edits, 'person/x', 'sex', fields)).toBe(false)
  })

  it('marks the entries it adds, counted from the end', () => {
    expect(drafts(edits, 'person/x', 'tags[2]', fields)).toBe(true)
    expect(drafts(edits, 'person/x', 'tags[1]', fields)).toBe(false)
  })

  it('marks everything of a record it creates, and nothing of another', () => {
    expect(drafts(edits, 'person/z', 'name', {})).toBe(true)
    expect(drafts(edits, 'person/w', 'sex', fields)).toBe(false)
  })
})
