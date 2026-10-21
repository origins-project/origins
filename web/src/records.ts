// What the pages of records other than people derive from a record and the records
// around it: the events at a place, who cites a source, what refers to a record, the
// places within another, the citations and records a hypothesis names, what a record
// says of a picture it keeps, and what a draft changes of a record.

import type { Index, Indexed, Kind, Record } from './api/client'
import { citations, placed, type Placed } from './person'

// A citation as a record writes it, `slug#anchor GRADE`, in its parts.
export interface Citation {
  source: string
  anchor: string
  grade: string
}

export function citationOf(written: string): Citation {
  const [at = '', grade = ''] = written.split(' ')
  const hash = at.indexOf('#')
  return { source: hash < 0 ? at : at.slice(0, hash), anchor: hash < 0 ? '' : at.slice(hash + 1), grade }
}

// The citations a list holds, those that are not text left out.
export function citationsOf(list: unknown): Citation[] {
  return Array.isArray(list) ? list.filter((c) => typeof c === 'string').map(citationOf) : []
}

// An event another record holds, with the record.
export interface Held extends Placed {
  record: Record
}

// Every event of the records around a place that took place there: those a date puts in
// order first, in that order, and the rest after them.
export function eventsAt(around: Record[], place: string): Held[] {
  const dated: Held[] = []
  const undated: Held[] = []
  for (const record of around) {
    const { dated: d, undated: u } = placed(record)
    for (const one of d) if (one.event.place === place) dated.push({ ...one, record })
    for (const one of u) if (one.event.place === place) undated.push({ ...one, record })
  }
  dated.sort((a, b) => (a.sort! < b.sort! ? -1 : a.sort! > b.sort! ? 1 : 0))
  return [...dated, ...undated]
}

// A record citing a source, where in it and with what grade.
export interface Citing {
  record: Record
  path: string
  grade: string
}

// Who cites a source among the records around it, by the anchor cited, the source as a
// whole under the empty one.
export function citedBy(around: Record[], source: string): Map<string, Citing[]> {
  const by = new Map<string, Citing[]>()
  for (const record of around)
    for (const cited of citations(record.fields)) {
      if (cited.source !== source) continue
      const list = by.get(cited.anchor) ?? []
      for (const use of cited.uses) list.push({ record, path: use.path, grade: use.grade })
      by.set(cited.anchor, list)
    }
  return by
}

// Whether a record refers to another: a field naming its slug, or a citation of it, or a
// link to it in the record's prose.
export function refersTo(record: Record, kind: Kind, slug: string): boolean {
  const names = (value: unknown): boolean => {
    if (typeof value === 'string') return value === slug || value.startsWith(`${slug}#`)
    if (Array.isArray(value)) return value.some(names)
    if (value !== null && typeof value === 'object') return Object.values(value).some(names)
    return false
  }
  const route = `href="/${kind}/${slug}`
  return (
    names(record.fields) ||
    Object.values(record.html).some((html) => html.includes(`${route}"`) || html.includes(`${route}#`))
  )
}

// The places the index says lie within a place, at some time.
export function within(index: Index | undefined, place: string): Indexed[] {
  return (index?.places ?? []).filter(
    (p) => Array.isArray(p.within) && p.within.some((w: { place?: unknown }) => w?.place === place),
  )
}

// An edit of a hypothesis as its page shows it: what it does, to what, where in it.
export interface Shown {
  op: string
  entities: string[]
  path: string
  value?: unknown
}

export function editOf(edit: { [field: string]: unknown }): Shown {
  const refs = [edit.entity, edit.from, edit.into]
    .flatMap((v) => (Array.isArray(v) ? v : [v]))
    .filter((v): v is string => typeof v === 'string' && v !== '')
  if (typeof edit.to === 'string' && refs[0]) refs.push(`${refs[0].slice(0, refs[0].indexOf('/'))}/${edit.to}`)
  return {
    op: typeof edit.op === 'string' ? edit.op : '',
    entities: refs,
    path: typeof edit.path === 'string' ? edit.path : '',
    value: edit.value,
  }
}

// The records a hypothesis's edits are about, each once, in the order they come.
export function concerned(edits: unknown): string[] {
  const all = Array.isArray(edits) ? edits.flatMap((e) => editOf(e ?? {}).entities) : []
  return [...new Set(all)]
}

// An address on one line: its lines, and its parts after them.
export function addressOf(address: unknown): string {
  if (typeof address === 'string') return address.trim().split('\n').join(', ')
  if (address === null || typeof address !== 'object') return ''
  const a = address as { [part: string]: unknown }
  return ['lines', 'street', 'locality', 'city', 'parish', 'county', 'state', 'postal', 'country']
    .map((part) => (typeof a[part] === 'string' ? (a[part] as string).trim().split('\n').join(', ') : ''))
    .filter(Boolean)
    .join(', ')
}

// A link's address and what it is called.
export function linkOf(link: unknown): { url: string; title: string } {
  if (typeof link === 'string') return { url: link, title: link }
  const l = (link ?? {}) as { url?: unknown; title?: unknown }
  const url = typeof l.url === 'string' ? l.url : ''
  return { url, title: typeof l.title === 'string' && l.title ? l.title : url }
}

// Coordinates as two angles in degrees, where they are written so.
export function anglesOf(coordinates: unknown): { latitude: number; longitude: number } | undefined {
  if (typeof coordinates !== 'string') return undefined
  const found = /^\s*([-+]?(?:\d+(?:\.\d*)?|\.\d+))\s*,\s*([-+]?(?:\d+(?:\.\d*)?|\.\d+))\s*$/.exec(coordinates)
  if (!found) return undefined
  const latitude = Number(found[1])
  const longitude = Number(found[2])
  return Math.abs(latitude) <= 90 && Math.abs(longitude) <= 180 ? { latitude, longitude } : undefined
}

// The entry a record keeping a file has for it, wherever among its fields: its own
// media, an event's, a citation's scans; and its path there. A file it keeps and says
// nothing of has none.
export function entryOf(fields: unknown, file: string): { path: string; entry: string | { [field: string]: unknown } } | undefined {
  const walk = (value: unknown, path: string, under: string): ReturnType<typeof entryOf> => {
    if (Array.isArray(value)) {
      for (const [i, v] of value.entries()) {
        const at = `${path}[${i}]`
        if (under === 'media' || under === 'images') {
          const named = typeof v === 'string' ? v : (v as { file?: unknown } | null)?.file
          if (named === file) return { path: at, entry: v }
        } else {
          const found = walk(v, at, '')
          if (found) return found
        }
      }
    } else if (value !== null && typeof value === 'object') {
      for (const [key, v] of Object.entries(value)) {
        const found = walk(v, path ? `${path}.${key}` : key, key)
        if (found) return found
      }
    }
    return undefined
  }
  return walk(fields, '', '')
}

// Somebody a picture shows, as its entry says: where they are in its list, the
// person's slug, the part of the picture they are in where that is marked, and what is
// said of them there.
export interface Depicted {
  at: number
  person: string
  region?: number[]
  note: string
}

export function depictedOf(entry: unknown): Depicted[] {
  const list = (entry as { depicts?: unknown } | null)?.depicts
  if (!Array.isArray(list)) return []
  return list.flatMap((d: unknown, at): Depicted[] => {
    if (typeof d === 'string') return [{ at, person: d, note: '' }]
    const { person, region, note } = (d ?? {}) as { person?: unknown; region?: unknown; note?: unknown }
    if (typeof person !== 'string') return []
    const fractions = Array.isArray(region) ? region.map(Number) : []
    return [
      {
        at,
        person,
        region: fractions.length === 4 && fractions.every(Number.isFinite) ? fractions : undefined,
        note: typeof note === 'string' ? note : '',
      },
    ]
  })
}

// Whether a draft's edits change what a page shows at path of entity, the record as the
// draft has it in fields: a value set or taken out there or around it, an entry added
// to a list, counted back from its end, and all of a record the draft creates.
export function drafts(edits: readonly unknown[], entity: string, path: string, fields: unknown): boolean {
  const under = (inner: string, outer: string) =>
    inner === outer || inner.startsWith(`${outer}.`) || inner.startsWith(`${outer}[`)
  const added = new Map<string, number>()
  for (const e of edits) {
    const { op, entity: of, path: at } = (e ?? {}) as { op?: unknown; entity?: unknown; path?: unknown }
    if (of !== entity) continue
    if (op === 'create') return true
    if (typeof at !== 'string' || at === '') continue
    if (op === 'add') added.set(at, (added.get(at) ?? 0) + 1)
    else if ((op === 'set' || op === 'unset') && (under(path, at) || under(at, path))) return true
  }
  for (const [list, n] of added) {
    let held = fields
    for (const step of list.split(/[.[\]]/).filter(Boolean))
      held = held !== null && typeof held === 'object' ? (held as { [step: string]: unknown })[step] : undefined
    const length = Array.isArray(held) ? held.length : 0
    for (let i = Math.max(0, length - n); i < length; i++) if (under(path, `${list}[${i}]`)) return true
  }
  return false
}
