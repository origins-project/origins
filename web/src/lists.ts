// What the lists, home and the findings derive from the index: whether a record answers
// a filter, the surnames counted, the places as a tree of what lies within what, the
// findings grouped by rule, and the slug a new record is given.

import type { Finding, Index, Indexed } from './api/client'
import { fold } from './search'

const text = (value: unknown) => (typeof value === 'string' ? value : '')

// Whether every word of what was asked is somewhere in the words, case and accents
// ignored.
export function answers(query: string, words: unknown[]): boolean {
  const asked = fold(query.trim()).text.split(/\s+/).filter(Boolean)
  if (asked.length === 0) return true
  const all = fold(words.flat(2).map(text).filter(Boolean).join(' ')).text
  return asked.every((word) => all.includes(word))
}

export interface Surname {
  surname: string
  count: number
}

// Every surname the people are filed under, with how many are, the most first.
export function surnames(people: Indexed[]): Surname[] {
  const counted = new Map<string, Surname>()
  for (const person of people) {
    const surname = text(person.surname).trim()
    if (!surname) continue
    const key = fold(surname).text
    const one = counted.get(key) ?? { surname, count: 0 }
    one.count++
    counted.set(key, one)
  }
  return [...counted.values()].sort((a, b) => b.count - a.count || a.surname.localeCompare(b.surname))
}

export interface Within {
  place: string
  from?: string
  until?: string
}

export interface Branch {
  place: Indexed
  from?: string
  until?: string
  under: Branch[]
}

const named = (place: Indexed) => text(place.name) || place.ref

// The places as a tree: each under every place it lies within, with when it did, and a
// place within nothing the index holds at the top. A place that lies within itself,
// however far round, is not drawn under itself again.
export function placeTree(places: Indexed[]): Branch[] {
  const bySlug = new Map(places.map((p) => [p.ref.slice(p.ref.indexOf('/') + 1), p]))
  const under = new Map<string, { place: Indexed; from?: string; until?: string }[]>()
  const top: Indexed[] = []
  for (const place of places) {
    const within = ((place.within as Within[] | undefined) ?? []).filter((w) => bySlug.has(w.place))
    if (within.length === 0) top.push(place)
    for (const w of within) {
      const list = under.get(w.place) ?? []
      list.push({ place, from: w.from, until: w.until })
      under.set(w.place, list)
    }
  }
  const byName = (a: { place: Indexed }, b: { place: Indexed }) => named(a.place).localeCompare(named(b.place))
  function grow(place: Indexed, above: Set<string>): Branch['under'] {
    const slug = place.ref.slice(place.ref.indexOf('/') + 1)
    const seen = new Set(above).add(slug)
    return (under.get(slug) ?? [])
      .filter((one) => !seen.has(one.place.ref.slice(one.place.ref.indexOf('/') + 1)))
      .sort(byName)
      .map((one) => ({ ...one, under: grow(one.place, seen) }))
  }
  // A ring of places each within another has no top; its first by name stands for it.
  const reached = new Set<string>()
  const mark = (branches: Branch[]) => branches.forEach((b) => (reached.add(b.place.ref), mark(b.under)))
  const tree = top
    .map((place) => ({ place }))
    .sort(byName)
    .map(({ place }) => ({ place, under: grow(place, new Set()) }))
  mark(tree)
  for (const place of [...places].sort((a, b) => named(a).localeCompare(named(b)))) {
    if (reached.has(place.ref)) continue
    const branch = { place, under: grow(place, new Set()) }
    reached.add(place.ref)
    mark(branch.under)
    tree.push(branch)
  }
  return tree
}

export interface Rule {
  rule: string
  findings: Finding[]
}

export interface Grouped {
  errors: Rule[]
  warnings: Rule[]
  todo: Rule[]
}

const keyOf = (f: Finding) => [f.rule, f.entity, f.path ?? '', f.message].join('\u0000')

function byRule(findings: Finding[]): Rule[] {
  const rules = new Map<string, Finding[]>()
  for (const finding of findings) rules.set(finding.rule, [...(rules.get(finding.rule) ?? []), finding])
  return [...rules].map(([rule, findings]) => ({ rule, findings })).sort((a, b) => a.rule.localeCompare(b.rule))
}

// The findings by severity and rule, those matching the filter: what check found, and
// what research leaves to do that check has not already said.
export function grouped(found: { findings: Finding[]; todo: Finding[] }, query = ''): Grouped {
  const said = new Set(found.findings.map(keyOf))
  const todo = new Map<string, Finding>()
  for (const one of found.todo) if (!said.has(keyOf(one))) todo.set(keyOf(one), one)
  const kept = (f: Finding) => answers(query, [f.rule, f.entity, f.path, f.message])
  return {
    errors: byRule(found.findings.filter((f) => f.severity === 'error' && kept(f))),
    warnings: byRule(found.findings.filter((f) => f.severity !== 'error' && kept(f))),
    todo: byRule([...todo.values()].filter(kept)),
  }
}

export const counted = (rules: Rule[]) => rules.reduce((n, r) => n + r.findings.length, 0)

const expansions: [RegExp, string][] = [
  [/['’]/g, ''],
  [/ß/g, 'ss'],
  [/æ/g, 'ae'],
  [/œ/g, 'oe'],
  [/ø/g, 'o'],
  [/đ/g, 'd'],
  [/ħ/g, 'h'],
  [/ł/g, 'l'],
  [/ŋ/g, 'n'],
  [/ŧ/g, 't'],
  [/ð/g, 'd'],
  [/þ/g, 'th'],
  [/ı/g, 'i'],
]

const maxSlug = 64

function shorten(s: string): string {
  if (s.length <= maxSlug) return s
  const i = s.slice(0, maxSlug + 1).lastIndexOf('-')
  return (i > 0 ? s.slice(0, i) : s.slice(0, maxSlug)).replace(/-+$/, '')
}

// The slug a name gives, as model.Slugify derives it, or '' where nothing is left.
export function slugify(name: string): string {
  let s = name.toLowerCase()
  for (const [from, to] of expansions) s = s.replace(from, to)
  s = s.normalize('NFD').replace(/\p{Mn}/gu, '')
  return shorten(s.replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, ''))
}

// A family's slug, as model.FamilySlug names it: its people in order, then the words
// for whoever is unknown or nobody.
export function familySlug(partners: string[]): string {
  const words = ['unknown', 'none']
  const people = partners.filter((p) => p && !words.includes(p)).sort()
  const reserved = partners.filter((p) => words.includes(p)).sort()
  return shorten([...people, ...reserved].join('--'))
}

// Every slug the corpus already uses, across the seven kinds.
export function taken(index: Index | undefined): Set<string> {
  const all = new Set<string>()
  if (!index) return all
  for (const list of [index.people, index.families, index.places, index.sources, index.repositories, index.notes, index.hypotheses])
    for (const one of list) all.add(one.ref.slice(one.ref.indexOf('/') + 1))
  return all
}

// The slug, or the first of it with a number after it that the corpus does not use, as
// a collision is told apart.
export function free(slug: string, used: Set<string>): string {
  if (!used.has(slug)) return slug
  for (let n = 2; ; n++) if (!used.has(`${slug}-${n}`)) return `${slug}-${n}`
}
