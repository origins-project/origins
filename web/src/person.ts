// What a person's page derives from the record and the index: the name in its parts,
// the people around them by how they are related, the moments of their life in order,
// and the citations the record makes.

import type { Index, Indexed, Record } from './api/client'
import type { NamePart } from './ui'

export interface Dated {
  type?: string
  date?: string
  sort?: string
  place?: string
}

export interface Portrait {
  record: string
  file: string
  region?: number[]
}

export interface IndexedPerson extends Indexed {
  name?: string
  sex?: string
  living?: boolean
  birth?: Dated
  death?: Dated
  portrait?: Portrait
  'partner-in'?: string[]
  'child-in'?: string[]
}

export interface IndexedChild {
  person?: string
  name?: string
  relations?: { partner: string; relation: string }[]
  status?: string
}

export interface IndexedFamily extends Indexed {
  type?: string
  partners?: { person?: string; name?: string }[]
  first?: Dated
  children?: IndexedChild[]
}

// A partner who is no record: nobody known, or nobody at all.
export const nobody = ['unknown', 'none']

const text = (value: unknown) => (typeof value === 'string' ? value : '')

// A name in its parts as a heading shows them: the prefix, the given names with the one
// the person was called by marked, the nickname, and the surnames with their particles.
// A name written as a string, or only as the record wrote it, is one part.
export function nameParts(name: unknown): NamePart[] {
  if (typeof name === 'string') return name ? [{ text: name }] : []
  if (name === null || typeof name !== 'object') return []
  const n = name as { [field: string]: unknown }
  const parts: NamePart[] = []
  const prefix = text(n.prefix)
  if (prefix) parts.push({ text: prefix, soft: true })
  const call = text(n.call)
  const given = text(n.given).split(/\s+/).filter(Boolean)
  if (call && !given.includes(call)) parts.push({ text: call, call: true })
  for (const word of given) parts.push({ text: word, call: word === call })
  const nickname = text(n.nickname)
  if (nickname) parts.push({ text: nickname, nickname: true })
  const surnamePrefix = text(n['surname-prefix'])
  if (surnamePrefix) parts.push({ text: surnamePrefix })
  const surnames = Array.isArray(n.surname) ? n.surname : n.surname === undefined ? [] : [n.surname]
  for (const surname of surnames) {
    if (typeof surname === 'string') {
      parts.push({ text: surname })
      continue
    }
    const s = (surname ?? {}) as { [field: string]: unknown }
    const words = [text(s.prefix), text(s.name), text(s.connector)].filter(Boolean).join(' ')
    if (words) parts.push({ text: words })
  }
  const suffix = text(n.suffix)
  if (suffix) parts.push({ text: suffix, soft: true })
  if (parts.length === 0 && text(n.text)) parts.push({ text: text(n.text) })
  return parts
}

// The year of a sort key, as a reader counts it.
export function yearOf(sort: string | undefined): string {
  const found = /^-?\d+/.exec(sort ?? '')
  return found ? String(Number(found[0])) : ''
}

// The years somebody lived, as a card beside their name gives them.
export function years(person: IndexedPerson | undefined): string {
  const born = yearOf(person?.birth?.sort)
  const died = yearOf(person?.death?.sort)
  return born || died ? `${born} – ${died}` : ''
}

// Initials for whoever has no picture: the first letters of the first and last words
// that begin with a capital, or with a letter of a script that has none, past an
// article joined to the word, as in d'Annunzio, al-Halabi and الحلبي. A name in
// ideographs or Hangul gives its first character, the surname, and a name with no
// capital its first letter.
export function initials(name: string): string {
  const first = [...name.trim()][0] ?? ''
  if (/[\p{Script=Han}\p{Script=Hangul}]/u.test(first)) return first
  const letters = name.split(/\s+/).map(initial).filter((l) => l !== '')
  const picked = letters.length > 1 ? [letters[0]!, letters[letters.length - 1]!] : letters
  return picked.join('') || (name.match(/\p{L}/u)?.[0].toUpperCase() ?? '')
}

const capital = /^[\p{Lu}\p{Lt}\p{Lo}]/u

function initial(word: string): string {
  const joined = /^\p{Ll}+['’-](.+)/u.exec(word)
  if (joined && capital.test(joined[1]!)) word = joined[1]!
  else if (/^ال\p{L}{2}/u.test(word)) word = word.slice(2)
  return capital.test(word) ? [...word][0]! : ''
}

// A relative, by how they are related to the person whose page it is.
export interface Relative {
  ref: string
  // the record, where there is one
  person?: IndexedPerson
  name: string
  // how they are related: parent, sibling, half-sibling, partner or child
  kin: Kin
  // the relation of a child to a parent where it is not by birth
  relation?: string
  status?: string
}

export type Kin = 'parent' | 'sibling' | 'halfSibling' | 'partner' | 'child'

// A family the person is in, as a child or as a partner, with who is in it besides.
export interface Union {
  family: IndexedFamily
  // as a child of it, the parents; as a partner of it, the other partners
  partners: Relative[]
  // as a child of it, the siblings; as a partner of it, the children
  children: Relative[]
  // as a child of it, the person's own relation to the parents
  status?: string
}

export interface Relatives {
  parents: Union[]
  partners: Union[]
  halfSiblings: Relative[]
}

function lookup(index: Index) {
  const people = new Map(index.people.map((p) => [p.ref, p as IndexedPerson]))
  const families = new Map(index.families.map((f) => [f.ref, f as IndexedFamily]))
  return { people, families }
}

function named(people: Map<string, IndexedPerson>, slug: string | undefined, name: string | undefined) {
  const person = slug ? people.get(`person/${slug}`) : undefined
  // a ref only where there is a record to lead to, never `unknown` nor a slug nobody has
  return { ref: person ? `person/${slug}` : '', person, name: text(person?.name) || text(name) || (slug ?? '') }
}

// Everybody the families tie the person to: their parents and siblings by each family
// they are a child of, their partners and children by each they are a partner in, and
// the children of their parents' other families, who are their half-siblings.
export function relatives(index: Index, slug: string): Relatives {
  const { people, families } = lookup(index)
  const me = people.get(`person/${slug}`)
  const out: Relatives = { parents: [], partners: [], halfSiblings: [] }
  if (!me) return out
  const full = new Set<string>()
  const parentSlugs: string[] = []
  for (const ref of me['child-in'] ?? []) {
    const family = families.get(`family/${ref}`)
    if (!family) continue
    const mine = family.children?.find((c) => c.person === slug)
    const partners = (family.partners ?? [])
      .filter((p) => !(p.person && nobody.includes(p.person)))
      .map((p) => {
        const relation = mine?.relations?.find((r) => r.partner === p.person)?.relation
        return { ...named(people, p.person, p.name), kin: 'parent' as const, relation: relation === 'birth' ? undefined : relation }
      })
    for (const p of family.partners ?? []) if (p.person && !nobody.includes(p.person)) parentSlugs.push(p.person)
    const children = (family.children ?? [])
      .filter((c) => c.person !== slug)
      .map((c) => ({ ...named(people, c.person, c.name), kin: 'sibling' as const, status: c.status }))
    for (const c of children) if (c.ref) full.add(c.ref)
    out.parents.push({ family, partners, children, status: mine?.status })
  }
  const halves = new Set<string>()
  for (const parent of parentSlugs) {
    for (const ref of people.get(`person/${parent}`)?.['partner-in'] ?? []) {
      if ((me['child-in'] ?? []).includes(ref)) continue
      for (const c of families.get(`family/${ref}`)?.children ?? []) {
        if (c.status === 'disproven' || c.person === slug) continue
        const one = named(people, c.person, c.name)
        const key = one.ref || `${ref}#${one.name}`
        if ((one.ref && full.has(one.ref)) || halves.has(key)) continue
        halves.add(key)
        out.halfSiblings.push({ ...one, kin: 'halfSibling' })
      }
    }
  }
  for (const ref of me['partner-in'] ?? []) {
    const family = families.get(`family/${ref}`)
    if (!family) continue
    const partners = (family.partners ?? [])
      .filter((p) => p.person !== slug && !(p.person && nobody.includes(p.person)))
      .map((p) => ({ ...named(people, p.person, p.name), kin: 'partner' as const }))
    const children = (family.children ?? []).map((c) => {
      const relation = c.relations?.find((r) => r.partner === slug)?.relation
      return {
        ...named(people, c.person, c.name),
        kin: 'child' as const,
        relation: relation === 'birth' ? undefined : relation,
        status: c.status,
      }
    })
    out.partners.push({ family, partners, children })
  }
  return out
}

// A relative's birth or death, among the person's own events in a quieter style.
export interface Quiet {
  sort: string
  event: Dated
  relative: Relative
}

// The births and deaths of the people around somebody that fall in their life: their
// parents' deaths, their siblings' births, their partners' deaths, and their children's
// births and deaths. A link disproven is nobody's family. The life is counted in years,
// from the year of the birth to the year of the death, either end open where it is not
// known.
export function quiet(relatives: Relatives, life?: IndexedPerson): Quiet[] {
  const out: Quiet[] = []
  const from = Number(yearOf(life?.birth?.sort) || -Infinity)
  const until = Number(yearOf(life?.death?.sort) || Infinity)
  const add = (relative: Relative, event: Dated | undefined) => {
    if (!event?.sort || relative.status === 'disproven') return
    const year = Number(yearOf(event.sort))
    if (year >= from && year <= until) out.push({ sort: event.sort, event, relative })
  }
  for (const union of relatives.parents) {
    for (const parent of union.partners) add(parent, parent.person?.death)
    for (const sibling of union.children) add(sibling, sibling.person?.birth)
  }
  for (const half of relatives.halfSiblings) add(half, half.person?.birth)
  for (const union of relatives.partners) {
    for (const partner of union.partners) add(partner, partner.person?.death)
    for (const child of union.children) {
      add(child, child.person?.birth)
      add(child, child.person?.death)
    }
  }
  return out
}

// The word for a relative, by how they are related and their sex.
export function kinWord(words: { [kin in Kin]: { M: string; F: string; other: string } }, relative: Relative): string {
  const sex = relative.person?.sex
  const said = words[relative.kin]
  return sex === 'M' ? said.M : sex === 'F' ? said.F : said.other
}

// One citation a record makes: where, and how it graded the evidence there.
export interface Use {
  path: string
  grade: string
}

export interface Cited {
  source: string
  anchor: string
  uses: Use[]
}

// Every citation in a record's fields, the citation's source and anchor once with each
// place it is cited from, in the order the record first cites them.
export function citations(fields: unknown): Cited[] {
  const found = new Map<string, Cited>()
  const walk = (value: unknown, path: string) => {
    if (Array.isArray(value)) {
      value.forEach((v, i) => walk(v, `${path}[${i}]`))
      return
    }
    if (value === null || typeof value !== 'object') return
    for (const [key, v] of Object.entries(value)) {
      const at = path ? `${path}.${key}` : key
      if (key === 'sources' && Array.isArray(v)) {
        for (const citation of v) {
          if (typeof citation !== 'string') continue
          const [cited = '', grade = ''] = citation.split(' ')
          const hash = cited.indexOf('#')
          const source = hash < 0 ? cited : cited.slice(0, hash)
          const anchor = hash < 0 ? '' : cited.slice(hash + 1)
          const one = found.get(cited) ?? { source, anchor, uses: [] }
          one.uses.push({ path, grade })
          found.set(cited, one)
        }
        continue
      }
      walk(v, at)
    }
  }
  walk(fields, '')
  return [...found.values()]
}

// A record's event, by its path, with where it goes in order.
export interface Placed {
  path: string
  index: number
  event: { [field: string]: unknown }
  sort?: string
}

// A record's events, those a date or a sort date places in order and those nothing
// does, which are facts held without a date.
export function placed(record: Record | undefined): { dated: Placed[]; undated: Placed[] } {
  const events = record?.fields.events
  const out = { dated: [] as Placed[], undated: [] as Placed[] }
  if (!Array.isArray(events)) return out
  events.forEach((event, index) => {
    const path = `events[${index}]`
    const one = { path, index, event: (event ?? {}) as Placed['event'], sort: record?.sorted?.[path] }
    if (one.sort) out.dated.push(one)
    else out.undated.push(one)
  })
  return out
}

// Whether a date is a period, drawn as a span down the timeline.
export function isPeriod(date: unknown): boolean {
  return typeof date === 'string' && /^\s*(FROM|TO)\b/i.test(date)
}
