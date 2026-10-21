import { existsSync } from 'node:fs'
import { expect, test, type Page } from '@playwright/test'
import type { Index } from '../src/api/client'
import type { Shape } from '../src/chart'
import { chart } from '../src/labels'
import { laid } from './chart'
import { complaints } from './complaints'
import { handTo, served, shared } from './origins'

// Every chart of somebody with family on every side, in the large corpus, served by the
// origins the run shares with the page's reads handed to it.

const { root: large, port } = shared.large
const shapes = Object.keys(chart.shapes) as Shape[]

test.describe.configure({ mode: 'serial' })

let index: Index

test.beforeAll(async () => {
  test.setTimeout(5 * 60 * 1000)
  if (!existsSync(large)) return
  await served(port)
  index = await (await fetch(`http://127.0.0.1:${port}/api/index`)).json()
})

test.afterEach(({ page }) => page.unrouteAll({ behavior: 'ignoreErrors' }))

// Somebody in the middle of the tree: a child of one family and a partner in another,
// with the most such ties.
function middle(): string {
  const ties = (p: { 'partner-in'?: string[]; 'child-in'?: string[] }) =>
    p['child-in']?.length && p['partner-in']?.length ? p['child-in'].length + p['partner-in'].length : 0
  const person = index.people.reduce((a, b) => (ties(b) > ties(a) ? b : a))
  return person.ref.slice('person/'.length)
}

// A chart is drawn from the index, which of the large corpus takes some seconds to pass
// through the route while other suites read it too.
const boxes = (page: Page) => page.locator('.chart [data-person]')

async function choose(page: Page, name: string, option: string) {
  await page.getByRole('combobox', { name }).click()
  await page.getByRole('option', { name: option, exact: true }).click()
}

test('every chart of somebody in the large corpus draws whoever the walk reaches', async ({ page }) => {
  test.skip(!existsSync(large), 'large is not built')
  test.setTimeout(3 * 60 * 1000)
  const said = complaints(page)
  const slug = middle()
  await handTo(page, port)
  await page.goto(`/person/${slug}`)
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 60 * 1000 })
  await page.getByRole('tab', { name: 'Charts' }).click()
  const farthest = (await laid(index, slug, 'relatives', 5)).people.at(-1)!.slug
  for (const shape of shapes) {
    await choose(page, 'Chart', chart.shapes[shape])
    if (shape === 'path') {
      await page.getByRole('combobox', { name: 'To' }).fill(farthest)
      await page.getByRole('option', { name: new RegExp(farthest) }).click()
    }
    const reached = (await laid(index, slug, shape, 5, farthest)).people
    expect(reached.length).toBeGreaterThan(1)
    await expect(boxes(page)).toHaveCount(reached.length, { timeout: 30 * 1000 })
    await expect(page.locator(`.chart [data-person="${slug}"] .subject`)).toBeVisible()
  }
  expect(said).toEqual([])
})

test('a chart reaches as far as it is asked, zooms, and says where it stopped', async ({ page }) => {
  test.skip(!existsSync(large), 'large is not built')
  test.setTimeout(3 * 60 * 1000)
  const said = complaints(page)
  const slug = middle()
  await handTo(page, port)
  await page.goto(`/person/${slug}`)
  await page.getByRole('tab', { name: 'Charts' }).click({ timeout: 60 * 1000 })
  await choose(page, 'Chart', chart.shapes.relatives)

  await choose(page, 'Steps', '1')
  await expect(boxes(page)).toHaveCount((await laid(index, slug, 'relatives', 1)).people.length, { timeout: 30 * 1000 })

  const svg = page.locator('.chart svg[data-scale]')
  const before = Number(await svg.getAttribute('data-scale'))
  await page.getByRole('button', { name: 'Zoom in' }).click()
  await expect.poll(async () => Number(await svg.getAttribute('data-scale'))).toBeGreaterThan(before)

  await choose(page, 'Steps', '12')
  const far = await laid(index, slug, 'relatives', 12)
  await expect(boxes(page)).toHaveCount(far.people.length, { timeout: 30 * 1000 })
  if (far.truncated) await expect(page.getByText(`First ${far.limit} people`)).toBeVisible()

  await page.setViewportSize({ width: 375, height: 800 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  expect(said).toEqual([])
})
