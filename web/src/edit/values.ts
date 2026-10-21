// A record's fields as values a form changes: the value at a path, a copy to change,
// and what a form hands back with whatever was left empty taken out.

import { steps, type Path, type Step } from './schema'

export type Value = unknown

export function valueAt(fields: Value, path: Path): Value {
  let at = fields
  for (const step of steps(path)) {
    if (at === null || typeof at !== 'object') return undefined
    at = (at as { [key: string]: Value })[step as string]
  }
  return at
}

export function copy<T>(value: T): T {
  return value === undefined ? value : (JSON.parse(JSON.stringify(value)) as T)
}

// A copy of a value with what is at the steps into it replaced, or taken out where the
// replacement is undefined.
export function placed(whole: Value, at: Step[], value: Value): Value {
  const [step, ...rest] = at
  if (step === undefined) return value
  const out = (copy(whole) ?? (typeof step === 'number' ? [] : {})) as { [key: string]: Value }
  const next = placed(out[step as string], rest, value)
  if (next === undefined && !Array.isArray(out)) delete out[step as string]
  else out[step as string] = next
  return out
}

// Whether a value says nothing: no text, no entries, no field that says anything.
export function empty(value: Value): boolean {
  if (value === undefined || value === null || value === '' || value === false) return true
  if (Array.isArray(value)) return value.every(empty)
  if (typeof value === 'object') return Object.values(value).every(empty)
  return false
}

// The value with every part that says nothing taken out, entries of a list kept in
// their places where something after them says something, since a path is positional.
export function pruned(value: Value): Value {
  if (Array.isArray(value)) {
    const kept = value.map(pruned)
    while (kept.length && empty(kept.at(-1))) kept.pop()
    return kept.length ? kept : undefined
  }
  if (value !== null && typeof value === 'object') {
    const kept: { [key: string]: Value } = {}
    for (const [key, v] of Object.entries(value)) {
      const p = pruned(v)
      if (!empty(p)) kept[key] = p
    }
    return Object.keys(kept).length ? kept : undefined
  }
  if (typeof value === 'string') return value.trim() === '' ? undefined : value
  return empty(value) ? undefined : value
}

// A value as one line of text, for a page to show where it has nothing better.
export function shown(value: Value): string {
  if (value === undefined || value === null) return ''
  if (typeof value !== 'object') return String(value)
  if (Array.isArray(value)) return value.map(shown).filter(Boolean).join(', ')
  const o = value as { [key: string]: Value }
  for (const lead of ['text', 'value', 'name', 'type', 'id', 'url', 'file', 'person', 'place', 'lines', 'title']) {
    if (typeof o[lead] === 'string' && o[lead] !== '') {
      const rest = lead === 'type' ? [o.date, o.value].filter((v) => typeof v === 'string' && v) : []
      return [o[lead], ...rest].join(' · ')
    }
  }
  const given = [o.given, o.surname].map(shown).filter(Boolean).join(' ')
  return given || Object.values(o).map(shown).filter(Boolean).join(' · ')
}
