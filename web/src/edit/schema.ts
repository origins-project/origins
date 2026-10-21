// The schema the forms are drawn from: the node at a record's path, and what control a
// node is. A form names no field itself, so a field the schema gains is a control.

import { shallowRef, type ShallowRef } from 'vue'
import type { Client, Kind } from '../api/client'

export type Node = { [key: string]: unknown }

// The served schema, each kind's and common's by name.
export type Schemas = { [name: string]: Node }

// A node with the definitions its $refs passed through, so that a date is known as a
// date wherever it is reached from.
export interface Found {
  node: Node
  refs: string[]
  // the schema a $ref in the node is read against
  from: string
}

let loaded: ShallowRef<Schemas | undefined> | undefined

// The schema, read once for the tab.
export function useSchemas(client: Client): ShallowRef<Schemas | undefined> {
  if (!loaded) {
    const schemas = shallowRef<Schemas>()
    loaded = schemas
    void client.schema().then(
      (s) => (schemas.value = s as Schemas),
      () => (loaded = undefined),
    )
  }
  return loaded
}

// Follows every $ref of a node to what it names, keeping the node's own description.
export function resolve(schemas: Schemas, found: Found): Found {
  let { node, refs, from } = found
  const seen = new Set<string>()
  while (typeof node.$ref === 'string') {
    const [target = '', pointer = ''] = node.$ref.split('#')
    const into = target === '' ? from : target.replace(/\.json$/, '')
    const key = `${into}#${pointer}`
    if (seen.has(key)) break
    seen.add(key)
    let next: unknown = schemas[into]
    for (const step of pointer.split('/').filter(Boolean)) next = (next as Node | undefined)?.[step]
    if (!next || typeof next !== 'object') break
    const description = node.description
    node = description === undefined ? (next as Node) : { ...(next as Node), description }
    refs = [...refs, key]
    from = into
  }
  return { node, refs, from }
}

// The alternatives a node allows, each resolved.
export function branches(schemas: Schemas, found: Found): Found[] {
  const at = resolve(schemas, found)
  const oneOf = at.node.oneOf
  if (!Array.isArray(oneOf)) return [at]
  return oneOf.flatMap((o) => branches(schemas, { node: o as Node, refs: at.refs, from: at.from }))
}

const isObject = (n: Node) => n.type === 'object' || n.properties !== undefined
const isArray = (n: Node) => n.type === 'array'

export const objectOf = (schemas: Schemas, f: Found) =>
  branches(schemas, f).find((b) => isObject(b.node) && b.node.properties !== undefined)
export const mapOf = (schemas: Schemas, f: Found) =>
  branches(schemas, f).find(
    (b) => isObject(b.node) && b.node.properties === undefined && typeof b.node.additionalProperties === 'object',
  )
export const arrayOf = (schemas: Schemas, f: Found) => branches(schemas, f).find((b) => isArray(b.node))
export const scalarOf = (schemas: Schemas, f: Found) =>
  branches(schemas, f).find((b) => !isObject(b.node) && !isArray(b.node))

// A step of a path: a field's name, a key of a mapping, or an index. A path is given
// as its steps where a key holds a character the written form uses to divide them.
export type Step = string | number
export type Path = string | Step[]

// The steps of a path.
export function steps(path: Path): Step[] {
  if (typeof path !== 'string') return path
  const out: Step[] = []
  for (const m of path.matchAll(/([^.[\]]+)|\[(\d+)\]/g)) out.push(m[2] !== undefined ? Number(m[2]) : m[1]!)
  return out
}

// A path written as the server reads one, each key as it is even where the server
// could not read it back, which `addressable` says.
export const written = (path: Path): string =>
  steps(path).reduce<string>((out, s, i) => (typeof s === 'number' ? `${out}[${s}]` : i ? `${out}.${s}` : s), '')

// Whether a key can be written as a step of a path.
export const addressable = (step: Step) => typeof step === 'number' || !/[.[\]]/.test(step)

// The node at a path into a record of a kind, or the manifest's.
export function nodeAt(schemas: Schemas, kind: Kind | 'project', path: Path): Found | undefined {
  let at: Found | undefined = { node: schemas[kind] ?? {}, refs: [], from: kind }
  for (const step of steps(path)) {
    if (!at) return undefined
    if (typeof step === 'number') {
      const array = arrayOf(schemas, at)
      at = array && { node: array.node.items as Node, refs: array.refs, from: array.from }
      continue
    }
    const object = objectOf(schemas, at)
    const property = (object?.node.properties as Node | undefined)?.[step]
    if (object && property) {
      at = { node: property as Node, refs: object.refs, from: object.from }
      continue
    }
    const map = mapOf(schemas, at)
    at = map && { node: map.node.additionalProperties as Node, refs: map.refs, from: map.from }
  }
  return at && resolve(schemas, at)
}

// The kind of record a field holding a slug names, by the field's name; same-as names
// the record's own kind.
const referring: { [field: string]: Kind } = {
  place: 'place',
  within: 'place',
  person: 'person',
  partners: 'person',
  children: 'person',
  depicts: 'person',
  'main-person': 'person',
  family: 'family',
  repository: 'repository',
  repositories: 'repository',
}

// The control a scalar is changed with.
export type Control =
  | { is: 'text'; multiline?: boolean }
  | { is: 'date' }
  | { is: 'flag' }
  | { is: 'choice'; options: string[] }
  | { is: 'record'; kind: Kind }
  | { is: 'citation' }

const prose = new Set(['note', 'body', 'transcription', 'argument'])

// The control for a scalar, by the definition it is and the path it is held at.
export function controlOf(scalar: Found, path: Path, own: Kind | 'project'): Control {
  const names = steps(path).filter((s): s is string => typeof s === 'string')
  let field = names.at(-1) ?? ''
  if (field === 'id') field = names.at(-2) ?? field
  const defined = scalar.refs.at(-1)
  if (defined === 'common#/$defs/date') return { is: 'date' }
  if (defined === 'common#/$defs/citation') return { is: 'citation' }
  if (defined === 'common#/$defs/slug') {
    if (field === 'same-as') return { is: 'record', kind: own === 'project' ? 'person' : own }
    return { is: 'record', kind: referring[field] ?? 'person' }
  }
  const choices = scalar.node.enum
  if (Array.isArray(choices)) {
    if (choices.length === 1 && choices[0] === 'true') return { is: 'flag' }
    return { is: 'choice', options: choices as string[] }
  }
  if (defined === 'common#/$defs/note' || defined === 'common#/$defs/body' || prose.has(field))
    return { is: 'text', multiline: true }
  return { is: 'text' }
}

// The field a mapping is written short as, where the record may give it alone: a
// surname's name, an identifier's id, a link's url, the archive a source is kept in.
const leading = ['value', 'id', 'name', 'url', 'file', 'person', 'place', 'repository', 'lines', 'text', 'type']

export function leadOf(object: Node): string | undefined {
  const properties = Object.keys((object.properties as Node | undefined) ?? {})
  const required = Array.isArray(object.required) ? (object.required as string[]) : []
  return required.find((r) => properties.includes(r)) ?? leading.find((l) => properties.includes(l))
}

// How a field is changed where it is shown: a value in a popover, values of a list each
// in one, an entry of several fields in a drawer, or entries of a list each in one. A
// field that may be written as an entry is drawn as one even where it is written short,
// so that what the entry holds besides is reached from it.
export type Drawn = 'value' | 'values' | 'entry' | 'entries'

export function drawnOf(schemas: Schemas, found: Found, value: unknown): Drawn {
  const list = arrayOf(schemas, found)
  if (list && (Array.isArray(value) || value === undefined)) {
    const items = { node: list.node.items as Found['node'], refs: list.refs, from: list.from }
    return objectOf(schemas, items) || mapOf(schemas, items) || !scalarOf(schemas, items) ? 'entries' : 'values'
  }
  const entry = objectOf(schemas, found) || mapOf(schemas, found)
  if (scalarOf(schemas, found) && !entry && (value === undefined || typeof value !== 'object')) return 'value'
  return 'entry'
}
