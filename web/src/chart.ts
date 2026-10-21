// The charts of a person, laid out by internal/chart run as WebAssembly, so that the
// pages and the files draw the same chart. The index is handed over once each time it
// changes, and a chart comes back with every box at its corner and every line drawn.

import { exported, written, type Index } from './api/client'
import './wasm/wasm_exec.js'
import url from './wasm/chart.wasm?url'

export type Shape =
  | 'ancestors'
  | 'descendants'
  | 'both'
  | 'relatives'
  | 'ancestor-tree'
  | 'descendant-tree'
  | 'fan'
  | 'path'
  | 'timeline'

export const shapes: Shape[] = [
  'ancestors',
  'descendants',
  'both',
  'relatives',
  'ancestor-tree',
  'descendant-tree',
  'fan',
  'path',
  'timeline',
]

export const defaultDistance = 5
export const maxDistance = 12

export type Tie = 'birth' | 'other' | 'partner'

// A parent nobody has a record of, where a chart could show one: a partner unknown in a
// family listing the person as a child, by its place in the family's partners, or either
// parent of a person no family lists.
export interface Gap {
  child: string
  family?: string
  partner?: number
}

export interface Point {
  x: number
  y: number
}

export interface Geometry {
  width: number
  height: number
  'column-gap': number
  'row-gap': number
  margin: number
}

// A person of a fan: a piece of a half ring, the outline to draw, and where its words go
// and how far they are turned, in degrees.
export interface Sector {
  ring: number
  start: number
  end: number
  path: string
  label: Point
  turn: number
}

// A life on a timeline, in years, and where it is drawn; an end with no date is open.
export interface Bar {
  from: number
  to: number
  born: boolean
  ended: boolean
  x: number
  width: number
}

// What a relationship path says: who it runs to, and what the subject is to them.
export interface Related {
  to: string
  related: boolean
  relation?: string
  'common-ancestor'?: string
}

// A chart asked for; a path runs to another person, and a timeline runs a life still
// going to this year unless told another.
export interface Ask {
  subject: string
  shape: Shape
  distance: number
  gaps: boolean
  to?: string
  year?: number
}

// A chart as internal/chart lays it out, each box by its top left corner, a fan's
// people as sectors and a timeline's as bars.
export interface Laid {
  subject: string
  shape: Shape
  distance: number
  box: Geometry
  width: number
  height: number
  truncated: boolean
  limit: number
  people: (Point & { slug: string; generation: number; sector?: Sector; bar?: Bar })[]
  slots: (Gap & Point & { key: string; generation: number })[]
  edges: { a: string; b: string; tie: Tie }[]
  lines: { path: string; tie: Tie | 'gap' }[]
  ticks?: { year: number; x: number }[]
  undated?: number
  path?: Related
}

interface Engine {
  load(index: string): string | null
  draw(ask: string): string | null
}

declare global {
  var originsChart: Engine | undefined
}

let started: Promise<Engine> | undefined

function engine(): Promise<Engine> {
  started ??= (async () => {
    const go = new Go()
    const bytes = exported ? await unpacked() : await fetch(url).then((r) => r.arrayBuffer())
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject)
    void go.run(instance)
    return globalThis.originsChart!
  })()
  return started
}

let loaded: object | undefined

// A chart of the index around the subject, with a slot for each parent missing where
// gaps asks for them.
export async function draw(index: Index | object, ask: Ask): Promise<Laid> {
  const charts = await engine()
  if (loaded !== index) {
    const refused = charts.load(JSON.stringify(index))
    if (refused) throw new Error(refused)
    loaded = index
  }
  return JSON.parse(charts.draw(JSON.stringify(ask))!) as Laid
}

// The WebAssembly an export wrote as a data script in base64, a page opened from a
// folder being unable to fetch the file.
async function unpacked(): Promise<ArrayBuffer> {
  const text = atob((await written('/chart.wasm')) as string)
  const bytes = new Uint8Array(text.length)
  for (let i = 0; i < text.length; i++) bytes[i] = text.charCodeAt(i)
  return bytes.buffer
}
