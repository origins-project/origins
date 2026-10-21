import { describe, expect, it } from 'vitest'
import type { Index, Record } from '../api/client'
import { citations, initials, nameParts, placed, quiet, relatives, years, type IndexedFamily, type IndexedPerson } from '../person'

function person(slug: string, born?: string, died?: string, more: Partial<IndexedPerson> = {}): IndexedPerson {
  return {
    ref: `person/${slug}`,
    name: slug,
    ...(born && { birth: { sort: born } }),
    ...(died && { death: { sort: died } }),
    ...more,
  } as IndexedPerson
}

function family(slug: string, more: Partial<IndexedFamily>): IndexedFamily {
  return { ref: `family/${slug}`, ...more } as IndexedFamily
}

// A man with two wives: a son and a daughter by the first, who died young, an adopted
// son by the second, and a son once said to be his and disproven.
const index = {
  people: [
    person('father', '1800', '1870', { 'child-in': ['grandparents'], 'partner-in': ['first', 'second'], sex: 'M' }),
    person('grandfather', '1770', '1820', { 'partner-in': ['grandparents'] }),
    person('aunt', '1798', '1880', { 'child-in': ['grandparents'] }),
    person('first-wife', '1802', '1830'),
    person('second-wife', '1805', '1890'),
    person('son', '1825', '1900', { 'child-in': ['first'], sex: 'M' }),
    person('daughter', '1828', '1829', { 'child-in': ['first'], sex: 'F' }),
    person('adopted', '1835', undefined, { 'child-in': ['second'] }),
    person('not-his', '1840', undefined, { 'child-in': ['second'] }),
  ],
  families: [
    family('grandparents', { partners: [{ person: 'grandfather' }, { person: 'unknown' }], children: [{ person: 'father' }, { person: 'aunt' }] }),
    family('first', {
      partners: [{ person: 'father' }, { person: 'first-wife' }],
      children: [{ person: 'son' }, { person: 'daughter' }, { name: 'Maria' }],
    }),
    family('second', {
      partners: [{ person: 'father' }, { person: 'second-wife' }],
      children: [
        { person: 'adopted', relations: [{ partner: 'father', relation: 'adoptive' }] },
        { person: 'not-his', status: 'disproven' },
      ],
    }),
  ],
} as unknown as Index

describe('a name in its parts', () => {
  it('marks the name somebody was called by among the given names', () => {
    expect(nameParts({ given: 'Giovanni Battista', call: 'Battista', surname: 'Rossi' })).toEqual([
      { text: 'Giovanni', call: false },
      { text: 'Battista', call: true },
      { text: 'Rossi' },
    ])
  })

  it('puts a call name no given name holds first, and a nickname after the given names', () => {
    expect(nameParts({ given: 'Giovanni', call: 'Nanni', nickname: 'il Rosso', surname: 'Rossi' })).toEqual([
      { text: 'Nanni', call: true },
      { text: 'Giovanni', call: false },
      { text: 'il Rosso', nickname: true },
      { text: 'Rossi' },
    ])
  })

  it('writes each surname with its particles, and the prefix and suffix softly', () => {
    expect(
      nameParts({ prefix: 'Dott.', given: 'Anna', surname: [{ prefix: 'de', name: 'Luca' }, 'Bianchi'], suffix: 'jr.' }),
    ).toEqual([{ text: 'Dott.', soft: true }, { text: 'Anna', call: false }, { text: 'de Luca' }, { text: 'Bianchi' }, { text: 'jr.', soft: true }])
  })

  it('takes a name written as a string, or only as the record wrote it, as one part', () => {
    expect(nameParts('Giovanni Rossi')).toEqual([{ text: 'Giovanni Rossi' }])
    expect(nameParts({ text: 'Giovanni di Rossi' })).toEqual([{ text: 'Giovanni di Rossi' }])
    expect(nameParts(undefined)).toEqual([])
  })
})

describe('a lifespan', () => {
  it('gives the years of the birth and the death, either missing', () => {
    expect(years(person('a', '1800-03-12', '1870'))).toBe('1800 – 1870')
    expect(years(person('a', '1800'))).toBe('1800 – ')
    expect(years(person('a'))).toBe('')
  })

  it('makes initials of the first and last capitalised words', () => {
    expect(initials('Giovanni Battista de Rossi')).toBe('GR')
    expect(initials('Maria')).toBe('M')
  })

  it('makes initials in a script that has no capitals', () => {
    expect(initials('زينب الحلبي')).toBe('زح')
    expect(initials('מרים כהן')).toBe('מכ')
    expect(initials('रमेश विठ्ठल')).toBe('रव')
    expect(initials('陳家蘭')).toBe('陳')
    expect(initials('박서연')).toBe('박')
    expect(initials('山田 太郎')).toBe('山')
  })

  it('makes initials past an article joined to a word', () => {
    expect(initials("Gabriele d'Annunzio")).toBe('GA')
    expect(initials('Zaynab al-Halabi')).toBe('ZH')
    expect(initials("Maria dell'Acqua")).toBe('MA')
    expect(initials("Seán O'Brien")).toBe('SO')
  })

  it('makes an initial of a name with no capital', () => {
    expect(initials('maria rossi')).toBe('M')
    expect(initials('')).toBe('')
  })
})

describe('the people around somebody', () => {
  const around = relatives(index, 'father')

  it('gives the parents and siblings by the family they are a child of, and nobody for a partner unknown', () => {
    expect(around.parents).toHaveLength(1)
    expect(around.parents[0]!.partners.map((p) => p.ref)).toEqual(['person/grandfather'])
    expect(around.parents[0]!.children.map((c) => c.ref)).toEqual(['person/aunt'])
  })

  it('gives the partners and children by each family they are a partner in', () => {
    expect(around.partners.map((u) => u.partners.map((p) => p.ref))).toEqual([['person/first-wife'], ['person/second-wife']])
    expect(around.partners[0]!.children.map((c) => c.name)).toEqual(['son', 'daughter', 'Maria'])
    expect(around.partners[1]!.children.map((c) => [c.ref, c.relation, c.status])).toEqual([
      ['person/adopted', 'adoptive', undefined],
      ['person/not-his', undefined, 'disproven'],
    ])
  })

  it('gives the children of a parent by another partner as half-siblings, and none disproven', () => {
    const son = relatives(index, 'son')
    expect(son.parents[0]!.children.map((c) => c.name)).toEqual(['daughter', 'Maria'])
    expect(son.halfSiblings.map((h) => h.ref)).toEqual(['person/adopted'])
  })

  it('puts the births and deaths around somebody that fall in their life beside their own', () => {
    const father = index.people[0] as IndexedPerson
    const said = quiet(around, father).map((q) => `${q.relative.name} ${q.sort}`)
    expect(said).toEqual(['grandfather 1820', 'first-wife 1830', 'son 1825', 'daughter 1828', 'daughter 1829', 'adopted 1835'])
  })

  it('counts a life open at an end it does not know', () => {
    const said = quiet(relatives(index, 'adopted'), index.people[7] as IndexedPerson).map((q) => `${q.relative.name} ${q.sort}`)
    expect(said).toEqual(['father 1870', 'second-wife 1890'])
  })
})

describe('the citations a record makes', () => {
  it('gives each citation once, with every place it is cited from and its grade there', () => {
    expect(
      citations({
        sources: ['census#entry-3 3332'],
        events: [{ sources: ['census#entry-3 3331', 'register'] }, { name: { sources: ['register'] } }],
      }),
    ).toEqual([
      { source: 'census', anchor: 'entry-3', uses: [{ path: '', grade: '3332' }, { path: 'events[0]', grade: '3331' }] },
      { source: 'register', anchor: '', uses: [{ path: 'events[0]', grade: '' }, { path: 'events[1].name', grade: '' }] },
    ])
  })
})

describe('the events of a record', () => {
  it('parts those a sort key places in order from those nothing does', () => {
    const record = {
      fields: { events: [{ type: 'birth' }, { type: 'occupation' }, { type: 'death' }] },
      sorted: { 'events[0]': '1800', 'events[2]': '1870' },
    } as unknown as Record
    const { dated, undated } = placed(record)
    expect(dated.map((p) => [p.path, p.sort])).toEqual([
      ['events[0]', '1800'],
      ['events[2]', '1870'],
    ])
    expect(undated.map((p) => p.path)).toEqual(['events[1]'])
  })
})
