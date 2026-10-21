import { readFile } from 'node:fs/promises'
import type { Index } from '../src/api/client'
import type { Laid, Shape } from '../src/chart'
import '../src/wasm/wasm_exec.js'

// The layout the page runs, internal/chart as WebAssembly, run here to say what a chart
// should draw.
let started: Promise<NonNullable<typeof globalThis.originsChart>> | undefined

export async function laid(index: Index, subject: string, shape: Shape, distance: number, to?: string): Promise<Laid> {
  started ??= (async () => {
    const go = new Go()
    const { instance } = await WebAssembly.instantiate(await readFile(new URL('../src/wasm/chart.wasm', import.meta.url)), go.importObject)
    void go.run(instance)
    const charts = globalThis.originsChart!
    charts.load(JSON.stringify(index))
    return charts
  })()
  const charts = await started
  return JSON.parse(charts.draw(JSON.stringify({ subject, shape, distance, gaps: false, to }))!) as Laid
}
