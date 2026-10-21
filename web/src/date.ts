// A date of the corpus grammar in its parts, for the date dialog to build one from and
// read one back into. It knows the grammar's shape and its words and nothing of how
// long a month is: whether a date stands is the server's to say, through `/api/date`.

import { dates } from './labels'

export const calendars = {
  GREGORIAN: ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'],
  JULIAN: ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'],
  HEBREW: ['TSH', 'CSH', 'KSL', 'TVT', 'SHV', 'ADR', 'ADS', 'NSN', 'IYR', 'SVN', 'TMZ', 'AAV', 'ELL'],
  FRENCH_R: ['VEND', 'BRUM', 'FRIM', 'NIVO', 'PLUV', 'VENT', 'GERM', 'FLOR', 'PRAI', 'MESS', 'THER', 'FRUC', 'COMP'],
  PERSIAN: ['FARVA', 'ORDIB', 'KHORD', 'TIR', 'MORDA', 'SHAHR', 'MEHR', 'ABAN', 'AZAR', 'DEY', 'BAHMA', 'ESFAN'],
  ISLAMIC: ['MUHAR', 'SAFAR', 'RABIA', 'RABIT', 'JUMAA', 'JUMAT', 'RAJAB', 'SHAAB', 'RAMAD', 'SHAWW', 'DHUAQ', 'DHUAH'],
  SWEDISH: ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'],
} as const

export type Calendar = keyof typeof calendars

// The calendars sharing the common era, the only ones with a year before it, a year
// written as two and a year counted from another day.
export const epochal: readonly Calendar[] = ['GREGORIAN', 'JULIAN']

export type Reached = '' | 'EST' | 'CAL'
export type Qualifier = '' | 'ABT' | 'BEF' | 'AFT' | 'BET'
export type Shape = 'one' | 'before' | 'after' | 'between' | 'period' | 'words'

// One date of the grammar. Every part is optional, written empty; a day with no month
// is a day whose month the record did not give.
export interface Value {
  calendar: Calendar
  day: string
  month: string
  year: string
  dual: boolean
  bce: boolean
  time: string
  newYear: { day: string; month: string }
}

// A date that is not a period: how it was reached, its qualifier, and the value, or
// the two a range runs between.
export interface Qualified {
  reached: Reached
  qualifier: Qualifier
  value: Value
  end: Value
}

// Every date: a moment, a period whose either end may be missing, or words alone, each
// with the record's own words beside it.
export interface Built {
  shape: Shape
  moment: Qualified
  from: Qualified | null
  to: Qualified | null
  wording: string
}

export function emptyValue(calendar: Calendar = 'GREGORIAN'): Value {
  return { calendar, day: '', month: '', year: '', dual: false, bce: false, time: '', newYear: { day: '', month: '' } }
}

export function emptyQualified(): Qualified {
  return { reached: '', qualifier: '', value: emptyValue(), end: emptyValue() }
}

export function emptyDate(): Built {
  return { shape: 'one', moment: emptyQualified(), from: null, to: null, wording: '' }
}

// The shape a qualified date has, read from its qualifier.
export function shapeOf(q: Qualified): Shape {
  return ({ '': 'one', ABT: 'one', BEF: 'before', AFT: 'after', BET: 'between' } as const)[q.qualifier]
}

// The qualifier a shape gives a moment, keeping `ABT` where the shape is one date.
export function qualifierFor(shape: Shape, was: Qualifier): Qualifier {
  switch (shape) {
    case 'before':
      return 'BEF'
    case 'after':
      return 'AFT'
    case 'between':
      return 'BET'
    default:
      return was === 'ABT' ? 'ABT' : ''
  }
}

const trimmed = (s: string) => s.trim()

function dualOf(year: number) {
  return `${year}/${String((year + 1) % 100).padStart(2, '0')}`
}

export function writeValue(v: Value): string {
  const parts: string[] = []
  if (v.calendar !== 'GREGORIAN') parts.push(v.calendar)
  const day = trimmed(v.day)
  const year = trimmed(v.year)
  if (day) parts.push(String(Number(day) || day))
  if (v.month) parts.push(v.month)
  else if (day) parts.push('?')
  if (year) parts.push(v.dual && /^\d+$/.test(year) ? dualOf(Number(year)) : year)
  if (v.bce) parts.push('BCE')
  if (trimmed(v.time)) parts.push(trimmed(v.time))
  if (trimmed(v.newYear.day) && v.newYear.month) parts.push('NY', trimmed(v.newYear.day), v.newYear.month)
  return parts.join(' ')
}

export function writeQualified(q: Qualified): string {
  const parts = [q.reached, q.qualifier, writeValue(q.value)]
  if (q.qualifier === 'BET') parts.push('AND', writeValue(q.end))
  return parts.filter(Boolean).join(' ')
}

// The date as the corpus writes it, or empty where it says nothing.
export function writeDate(d: Built): string {
  let stated = ''
  if (d.shape === 'period') {
    const from = d.from && writeQualified(d.from)
    const to = d.to && writeQualified(d.to)
    stated = [from && `FROM ${from}`, to && `TO ${to}`].filter(Boolean).join(' ')
  } else if (d.shape !== 'words') {
    stated = writeQualified(d.moment)
  }
  const wording = d.wording.split(/\s+/).filter(Boolean).join(' ')
  if (!wording) return stated
  return stated ? `${stated} (${wording})` : `(${wording})`
}

const same = (a: string, b: string) => a.toUpperCase() === b.toUpperCase()

function calendarNamed(word: string): Calendar | undefined {
  return (Object.keys(calendars) as Calendar[]).find((c) => same(c, word))
}

function monthIn(calendar: Calendar, word: string): string | undefined {
  return calendars[calendar].find((m) => same(m, word))
}

const isDay = (s: string) => /^\d{1,2}$/.test(s) && Number(s) >= 1 && Number(s) <= 31
const isYear = (s: string) => /^\d+(\/\d\d)?$/.test(s)

function parseValue(fields: string[]): Value | undefined {
  const v = emptyValue()
  const named = fields.length > 0 ? calendarNamed(fields[0]!) : undefined
  if (named) {
    v.calendar = named
    fields = fields.slice(1)
  }
  const ny = fields.findIndex((f) => same(f, 'NY'))
  if (ny >= 0) {
    const [day, month, ...more] = fields.slice(ny + 1)
    const m = month && monthIn(v.calendar, month)
    if (!day || !isDay(day) || !m || more.length) return undefined
    v.newYear = { day: String(Number(day)), month: m }
    fields = fields.slice(0, ny)
  }
  const time = fields[fields.length - 1]
  if (time?.includes(':')) {
    if (!/^\d\d:\d\d(:\d\d)?$/.test(time)) return undefined
    v.time = time
    fields = fields.slice(0, -1)
  }
  const era = fields[fields.length - 1]
  if (era && same(era, 'BCE')) {
    v.bce = true
    fields = fields.slice(0, -1)
  }
  const year = (s: string) => {
    if (!isYear(s)) return false
    const [y, dual] = s.split('/')
    if (dual !== undefined && dual !== String((Number(y) + 1) % 100).padStart(2, '0')) return false
    v.year = String(Number(y))
    v.dual = dual !== undefined
    return true
  }
  const day = (s: string) => {
    if (!isDay(s)) return false
    v.day = String(Number(s))
    return true
  }
  const month = (s: string) => {
    const m = monthIn(v.calendar, s)
    if (m) v.month = m
    return m !== undefined
  }
  const [a, b, c] = fields
  switch (fields.length) {
    case 1:
      return month(a!) || year(a!) ? v : undefined
    case 2:
      if (b === '?') return day(a!) ? v : undefined
      if (monthIn(v.calendar, b!)) return day(a!) && month(b!) ? v : undefined
      return month(a!) && year(b!) ? v : undefined
    case 3:
      return day(a!) && (b === '?' || month(b!)) && year(c!) ? v : undefined
  }
  return undefined
}

function parseQualified(fields: string[]): Qualified | undefined {
  const q = emptyQualified()
  if (fields[0] && (same(fields[0], 'EST') || same(fields[0], 'CAL'))) {
    q.reached = fields[0].toUpperCase() as Reached
    fields = fields.slice(1)
  }
  const qualifier = (['ABT', 'BEF', 'AFT', 'BET'] as const).find((k) => fields[0] && same(fields[0], k))
  if (qualifier) {
    q.qualifier = qualifier
    fields = fields.slice(1)
  }
  if (qualifier === 'BET') {
    const and = fields.findIndex((f) => same(f, 'AND'))
    if (and < 0) return undefined
    const start = parseValue(fields.slice(0, and))
    const end = parseValue(fields.slice(and + 1))
    if (!start || !end) return undefined
    q.value = start
    q.end = end
    return q
  }
  const value = parseValue(fields)
  if (!value) return undefined
  q.value = value
  return q
}

// Reads a date into its parts, or undefined where the grammar has no reading of it. A
// reading says nothing of whether the date stands: 31 FEB is read, and refused by the
// server.
export function parseDate(text: string): Built | undefined {
  const d = emptyDate()
  let stated = text
  const open = text.indexOf('(')
  if (open >= 0) {
    const rest = text.slice(open).trim()
    if (!rest.endsWith(')')) return undefined
    d.wording = rest.slice(1, -1).split(/\s+/).filter(Boolean).join(' ')
    if (!d.wording) return undefined
    stated = text.slice(0, open)
  }
  const fields = stated.split(/\s+/).filter(Boolean)
  if (fields.length === 0) {
    if (!d.wording) return undefined
    d.shape = 'words'
    return d
  }
  if (same(fields[0]!, 'FROM') || same(fields[0]!, 'TO')) {
    d.shape = 'period'
    let end = fields
    if (same(fields[0]!, 'FROM')) {
      const to = fields.findIndex((f, i) => i > 0 && same(f, 'TO'))
      const start = fields.slice(1, to < 0 ? undefined : to)
      end = to < 0 ? [] : fields.slice(to)
      const from = parseQualified(start)
      if (!from) return undefined
      d.from = from
    }
    if (end.length) {
      const to = parseQualified(end.slice(1))
      if (!to) return undefined
      d.to = to
    }
    return d
  }
  const moment = parseQualified(fields)
  if (!moment) return undefined
  d.moment = moment
  d.shape = shapeOf(moment)
  return d
}

// A date as a reader reads it: its qualifier and its months in words, a calendar other
// than the Gregorian named after it, and the record's own wording in quotation marks.
// Text the grammar has no reading of is given back as it was written.
export function spoken(text: string): string {
  const built = parseDate(text)
  if (!built) return text
  const said = dates.spoken
  const value = (v: Value) => {
    const month = v.month ? said.months[v.calendar][calendars[v.calendar].indexOf(v.month as never)] : v.day ? '?' : ''
    const year = v.year && (v.dual ? dualOf(Number(v.year)) : v.year)
    const words = [v.day, month, year, v.bce && said.BCE, v.time]
    if (v.newYear.day) words.push(said.NY, v.newYear.day, said.months[v.calendar][calendars[v.calendar].indexOf(v.newYear.month as never)])
    if (v.calendar !== 'GREGORIAN') words.push(said.calendars[v.calendar])
    return words.filter(Boolean).join(' ')
  }
  const qualified = (q: Qualified) => {
    const words = [q.reached && said[q.reached], q.qualifier && said[q.qualifier], value(q.value)]
    if (q.qualifier === 'BET') words.push(said.AND, value(q.end))
    return words.filter(Boolean).join(' ')
  }
  let stated = ''
  if (built.shape === 'period') {
    stated = [built.from && `${said.FROM} ${qualified(built.from)}`, built.to && `${said.TO} ${qualified(built.to)}`]
      .filter(Boolean)
      .join(' ')
  } else if (built.shape !== 'words') {
    stated = qualified(built.moment)
  }
  const wording = built.wording && `“${built.wording}”`
  return [stated, wording].filter(Boolean).join(' ')
}
