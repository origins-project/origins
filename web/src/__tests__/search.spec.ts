import { describe, expect, it } from 'vitest'
import type { Index } from '../api/client'
import { fold, search, segments, shown } from '../search'

function index(over: Partial<Index> = {}): Index {
  return {
    project: {},
    people: [],
    families: [],
    places: [],
    sources: [],
    repositories: [],
    notes: [],
    hypotheses: [],
    problems: [],
    ...over,
  }
}

const corpus = index({
  people: [
    {
      ref: 'person/rossi-giovanni',
      name: 'Giovanni Rossi',
      birth: { type: 'birth', sort: '1820-03-12' },
      death: { type: 'death', sort: '1891-01-02' },
    },
    { ref: 'person/bianchi-maria', name: 'Maria Bianchi', 'also-known-as': ['Maria Rossi'] },
    { ref: 'person/nicolo', name: 'Niccolò Cerù' },
    { ref: 'person/grossi-anna', name: 'Anna Grossi' },
  ],
  families: [
    {
      ref: 'family/rossi-bianchi',
      type: 'married',
      partners: [{ person: 'rossi-giovanni' }, { person: 'bianchi-maria' }],
    },
  ],
  places: [{ ref: 'place/lucca', name: 'Lucca', type: 'city' }],
  sources: [{ ref: 'source/register', name: 'Registro dei battesimi', 'short-title': 'Battesimi', author: 'Parrocchia' }],
  hypotheses: [{ ref: 'hypothesis/born-1844', name: 'Was Annunziata born in 1844?', status: 'open' }],
})

describe('the search', () => {
  it('folds case and accents, knowing where each letter came from', () => {
    const folded = fold('Cerù É')
    expect(folded.text).toBe('ceru e')
    expect(folded.from[3]).toBe(3)
    expect(folded.to[3]).toBe(4)
  })

  it('cuts words into marked and unmarked pieces', () => {
    expect(segments('Giovanni Rossi', [[9, 13], [0, 3], [11, 14]])).toEqual([
      { text: 'Gio', marked: true },
      { text: 'vanni ' },
      { text: 'Rossi', marked: true },
    ])
    expect(segments('Lucca')).toEqual([{ text: 'Lucca' }])
  })

  it('finds nothing for nothing asked', () => {
    expect(search(corpus, '  ')).toEqual([])
  })

  it('finds every kind by any name, accents ignored, in the order of the lists', () => {
    const found = search(corpus, 'ross')
    expect(found.map((g) => g.kind)).toEqual(['person', 'family'])
    expect(found[0]!.hits.map((h) => h.ref)).toEqual([
      'person/rossi-giovanni',
      'person/bianchi-maria',
      'person/grossi-anna',
    ])
    expect(found[0]!.hits[0]!.text).toEqual([{ text: 'Giovanni ' }, { text: 'Ross', marked: true }, { text: 'i' }])
    expect(found[0]!.hits[0]!.sub).toEqual([{ text: '1820–1891' }])
    expect(found[0]!.hits[1]!.text).toEqual([{ text: 'Maria Bianchi' }])
    expect(found[0]!.hits[1]!.sub).toEqual([{ text: 'Maria ' }, { text: 'Ross', marked: true }, { text: 'i' }])
    expect(found[1]!.hits[0]!.sub).toEqual([{ text: 'married' }])

    expect(search(corpus, 'NICCOLO ceru')[0]!.hits[0]!.text).toEqual([
      { text: 'Niccolò', marked: true },
      { text: ' ' },
      { text: 'Cerù', marked: true },
    ])
  })

  it('needs every word in one name', () => {
    expect(search(corpus, 'maria lucca')).toEqual([])
    expect(search(corpus, 'rossi maria')[0]!.hits.map((h) => h.ref)).toEqual(['person/bianchi-maria'])
  })

  it('puts a name starting with what was asked first, then words starting with it', () => {
    const found = search(corpus, 'ann')
    expect(found.map((g) => g.kind)).toEqual(['person', 'family', 'hypothesis'])
    expect(found[0]!.hits.map((h) => h.ref)).toEqual(['person/grossi-anna', 'person/rossi-giovanni'])
    expect(search(corpus, 'battesimi')[0]!.hits[0]!.sub).toEqual([{ text: 'Battesimi', marked: true }])
    expect(search(corpus, 'registro')[0]!.hits[0]!.sub).toEqual([{ text: 'Parrocchia' }])
  })

  it('shows at most so many of a kind', () => {
    const many = index({
      people: Array.from({ length: shown + 5 }, (_, i) => ({ ref: `person/p${i}`, name: `Person ${i}` })),
    })
    expect(search(many, 'person')[0]!.hits).toHaveLength(shown)
  })
})

const world = index({
  people: [
    { ref: 'person/gunnarsdottir-solveig', name: 'Sólveig Gunnarsdóttir' },
    { ref: 'person/ershov-ivan', name: 'Иван Петрович Ершов', 'also-known-as': ['Ivan Petrovich Ershov'] },
    { ref: 'person/papadopoulou-maria', name: 'Μαρία Παπαδοπούλου', 'also-known-as': ['Maria Papadopoulou'] },
    { ref: 'person/al-halabi-zaynab', name: 'زينب بنت محمود الحلبي' },
    { ref: 'person/rahimi-parvin', name: 'پروین رحیمی' },
    { ref: 'person/modiano-rachel', name: 'Rachel Modiano', 'also-known-as': ['רחל מודיאנו'] },
    { ref: 'person/chen-yuying', name: '陳玉英', 'also-known-as': ['Chén Yùyīng'] },
    { ref: 'person/chen-jiaming', name: '陳家明' },
    { ref: 'person/sato-yumiko', name: '佐藤由美子', 'also-known-as': ['さとう ゆみこ'] },
    { ref: 'person/kato', name: 'かとう' },
    { ref: 'person/bak-seoyeon', name: '박서연' },
    { ref: 'person/vitthal-ramesh', name: 'रमेश विठ्ठल' },
    { ref: 'person/ram', name: 'राम' },
  ],
})

function found(query: string): { refs: string[]; marked: string[] } {
  const hits = search(world, query)[0]?.hits ?? []
  const first = hits[0]
  const marks = [...(first?.text ?? []), ...(first?.sub ?? [])].filter((s) => s.marked).map((s) => s.text)
  return { refs: hits.map((h) => h.ref), marked: marks }
}

describe('the search in every script', () => {
  it.each([
    ['Latin', 'solveig', ['person/gunnarsdottir-solveig'], ['Sólveig']],
    ['Cyrillic', 'ЕРШОВ', ['person/ershov-ivan'], ['Ершов']],
    ['Greek', 'παπαδοπουλου', ['person/papadopoulou-maria'], ['Παπαδοπούλου']],
    ['Arabic', 'مَحْمُود', ['person/al-halabi-zaynab'], ['محمود']],
    ['Arabic, as Persian writes it', 'رحیمی', ['person/rahimi-parvin'], ['رحیمی']],
    ['Hebrew', 'מוֹדִיאנוֹ', ['person/modiano-rachel'], ['מודיאנו']],
    ['Chinese', '玉英', ['person/chen-yuying'], ['玉英']],
    ['Chinese, a surname shared', '陳', ['person/chen-jiaming', 'person/chen-yuying'], ['陳']],
    ['Japanese', '由美子', ['person/sato-yumiko'], ['由美子']],
    ['Japanese kana', 'ゆみこ', ['person/sato-yumiko'], ['ゆみこ']],
    ['Korean', '서연', ['person/bak-seoyeon'], ['서연']],
    ['Devanagari', 'विठ्ठल', ['person/vitthal-ramesh'], ['विठ्ठल']],
  ])('%s: %s', (_, query, refs, marked) => {
    expect(found(query)).toEqual({ refs, marked })
  })

  it('keeps a mark that is part of a word, as Devanagari vowels and Japanese voicing', () => {
    expect(found('रम').refs).toEqual(['person/vitthal-ramesh'])
    expect(found('राम').refs).toEqual(['person/ram'])
    expect(found('がとう').refs).toEqual([])
    expect(found('かとう').refs).toEqual(['person/kato'])
  })

  it('marks a letter with the marks written after it', () => {
    const decomposed = fold('Ceru' + String.fromCodePoint(0x300))
    expect(decomposed.text).toBe('ceru')
    expect(decomposed.to[3]).toBe(5)
  })
})
