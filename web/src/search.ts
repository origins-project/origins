// What the search finds: every record of every kind whose name, any name the index
// holds for it, holds every word asked for, accents and case ignored, with the words
// found marked where they are.

import { kindsInOrder, type Index, type Indexed, type Kind } from './api/client'
import { frame } from './labels'

export interface Segment {
  text: string
  marked?: boolean
}

export interface Hit {
  ref: string
  kind: Kind
  text: Segment[]
  sub?: Segment[]
}

// The index's list of each kind.
export const listed = {
  person: 'people',
  family: 'families',
  place: 'places',
  source: 'sources',
  repository: 'repositories',
  note: 'notes',
  hypothesis: 'hypotheses',
} as const satisfies Record<Kind, keyof Index>

// How many of a kind the search shows.
export const shown = 20

// Words folded for comparing, each folded character knowing where in the words it came
// from, so that what is found can be marked in the words as they are written. Case goes,
// and the marks on a letter of a script writing them as accents or as vowels a reader
// may leave out; in any other script, as Devanagari's vowel signs and Japanese voicing,
// a mark is part of the word.
interface Folded {
  text: string
  from: number[]
  to: number[]
}

const unmarked = /[\p{sc=Latin}\p{sc=Greek}\p{sc=Cyrillic}\p{sc=Hebrew}\p{sc=Arabic}]/u

export function fold(words: string): Folded {
  let text = ''
  const from: number[] = []
  const to: number[] = []
  let at = 0
  let drops = false
  let last = 0
  for (const character of words) {
    const end = at + character.length
    for (const decomposed of character.normalize('NFD')) {
      if (/\p{M}/u.test(decomposed)) {
        if (drops) {
          to.fill(end, last)
          continue
        }
      } else {
        drops = unmarked.test(decomposed)
        last = to.length
      }
      for (const folded of decomposed.toLowerCase()) {
        text += folded
        for (let i = 0; i < folded.length; i++) {
          from.push(at)
          to.push(end)
        }
      }
    }
    at = end
  }
  return { text, from, to }
}

function startsWord(text: string, at: number): boolean {
  return at === 0 || !/[\p{L}\p{N}]/u.test(text[at - 1]!)
}

// Where a word is found in a name, at the start of a word of it where it can be.
function find(name: string, word: string): number {
  let first = -1
  for (let at = name.indexOf(word); at >= 0; at = name.indexOf(word, at + 1)) {
    if (startsWord(name, at)) return at
    if (first < 0) first = at
  }
  return first
}

// How well a name answers the words, lower is better, or undefined where it does not:
// the name starting with what was asked, every word starting a word of it, or every
// word only somewhere in it.
function rank(name: Folded, query: string, words: string[]): { rank: number; marks: [number, number][] } | undefined {
  const marks: [number, number][] = []
  let starts = true
  for (const word of words) {
    const at = find(name.text, word)
    if (at < 0) return undefined
    if (!startsWord(name.text, at)) starts = false
    marks.push([name.from[at]!, name.to[at + word.length - 1]!])
  }
  return { rank: name.text.startsWith(query) ? 0 : starts ? 1 : 2, marks }
}

// Words cut into the pieces marked and the pieces not.
export function segments(words: string, marks: [number, number][] = []): Segment[] {
  const sorted = [...marks].sort((a, b) => a[0] - b[0])
  const merged: [number, number][] = []
  for (const mark of sorted) {
    const last = merged[merged.length - 1]
    if (last && mark[0] <= last[1]) last[1] = Math.max(last[1], mark[1])
    else merged.push([...mark])
  }
  const out: Segment[] = []
  let at = 0
  for (const [start, end] of merged) {
    if (start > at) out.push({ text: words.slice(at, start) })
    out.push({ text: words.slice(start, end), marked: true })
    at = end
  }
  if (at < words.length) out.push({ text: words.slice(at) })
  return out
}

function text(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function year(dated: unknown): string {
  const sort = text((dated as { sort?: unknown } | undefined)?.sort)
  const found = /^-?\d+/.exec(sort)
  return found ? String(Number(found[0])) : ''
}

// What a record is called, and every other name it goes by.
export function names(kind: Kind, record: Indexed, people: Map<string, Indexed>): { name: string; others: string[] } {
  const others = Array.isArray(record['also-known-as']) ? record['also-known-as'].map(text).filter(Boolean) : []
  let name = text(record.name)
  if (kind === 'family' && name === '' && Array.isArray(record.partners)) {
    name = (record.partners as { person?: string; name?: string }[])
      .map((p) => text(p.name) || text(people.get(`person/${p.person}`)?.name))
      .filter(Boolean)
      .join(frame.partners)
  }
  if (kind === 'source' && text(record['short-title']) !== '') others.unshift(text(record['short-title']))
  if (name === '') name = record.ref.slice(record.ref.indexOf('/') + 1)
  return { name, others }
}

// What is shown beside a record found by its own name.
function beside(kind: Kind, record: Indexed): string {
  switch (kind) {
    case 'person': {
      const born = year(record.birth)
      const died = year(record.death)
      return born || died ? `${born}–${died}` : ''
    }
    case 'source':
      return text(record.author)
    case 'hypothesis':
      return text(record.status)
    default:
      return text(record.type)
  }
}

// Everything the words find, by kind in the order of the lists, best first and at most
// shown of each.
export function search(index: Index, query: string): { kind: Kind; hits: Hit[] }[] {
  const folded = fold(query.trim().replace(/\s+/g, ' ')).text
  const words = folded.split(' ').filter(Boolean)
  if (words.length === 0) return []
  const people = new Map(index.people.map((p) => [p.ref, p]))
  const groups: { kind: Kind; hits: Hit[] }[] = []
  for (const kind of kindsInOrder) {
    const found: { hit: Hit; rank: number; order: string }[] = []
    for (const record of index[listed[kind]]) {
      const { name, others } = names(kind, record, people)
      let best: { rank: number; hit: Hit } | undefined
      for (const [i, one] of [name, ...others].entries()) {
        const answer = rank(fold(one), folded, words)
        if (!answer || (best && best.rank <= answer.rank)) continue
        const hit: Hit =
          i === 0
            ? { ref: record.ref, kind, text: segments(name, answer.marks) }
            : { ref: record.ref, kind, text: segments(name), sub: segments(one, answer.marks) }
        best = { rank: answer.rank, hit }
      }
      if (!best) continue
      if (!best.hit.sub) {
        const words = beside(kind, record)
        if (words) best.hit.sub = segments(words)
      }
      found.push({ hit: best.hit, rank: best.rank, order: fold(name).text })
    }
    if (found.length === 0) continue
    found.sort((a, b) => a.rank - b.rank || a.order.localeCompare(b.order))
    groups.push({ kind, hits: found.slice(0, shown).map((f) => f.hit) })
  }
  return groups
}
