import { existsSync } from 'node:fs'
import { expect, test, type Page } from '@playwright/test'
import { chart } from '../src/labels'
import { complaints } from './complaints'
import { handTo, served, shared } from './origins'

// A name written right to left, on a person's page and in a chart, in the fixture served
// by the origins the run shares with the page's reads handed to it: read in its own
// direction, cut at its own end, and keeping the years beside it in theirs.

const { root: fixture, port } = shared.fixture

test.beforeAll(async () => {
  test.setTimeout(5 * 60 * 1000)
  if (existsSync(fixture)) await served(port)
})

test.afterEach(({ page }) => page.unrouteAll({ behavior: 'ignoreErrors' }))

async function choose(page: Page, name: string, option: string) {
  await page.getByRole('combobox', { name }).click()
  await page.getByRole('option', { name: option, exact: true }).click()
}

const mahmud = 'أبو يوسف محمود بن علي بن حسن الحلبي'
const zaynab = 'زينب بنت محمود الحلبي'

test('a name written right to left reads right to left on the page', async ({ page }) => {
  test.skip(!existsSync(fixture), 'the git fixture is not built')
  const said = complaints(page)
  await handTo(page, port)
  await page.goto('/person/al-halabi-mahmud')
  const name = page.getByRole('heading', { level: 1 }).locator('[dir="auto"]')
  await expect(name).toHaveText(mahmud)
  expect(await name.evaluate((el) => getComputedStyle(el).direction)).toBe('rtl')
  expect(said).toEqual([])
})

test('a name written right to left is cut at its end in a box, and keeps its years in theirs', async ({ page }) => {
  test.skip(!existsSync(fixture), 'the git fixture is not built')
  const said = complaints(page)
  await handTo(page, port)
  await page.goto('/person/al-halabi-zaynab')
  await page.getByRole('tab', { name: 'Charts' }).click()
  await choose(page, 'Chart', chart.shapes.ancestors)

  // the father's name is longer than his box: what shows is its first words, at the right
  const name = page.locator('.chart [data-person="al-halabi-mahmud"] .name')
  await expect(name).toHaveText(mahmud)
  const cut = await name.evaluate((el) => {
    const node = el.firstChild!
    const range = document.createRange()
    range.setStart(node, 0)
    range.setEnd(node, 'أبو'.length)
    const first = range.getBoundingClientRect()
    const box = el.getBoundingClientRect()
    return { direction: getComputedStyle(el).direction, overflows: el.scrollWidth > el.clientWidth, first, box }
  })
  expect(cut.direction).toBe('rtl')
  expect(cut.overflows).toBe(true)
  expect(cut.first.right).toBeLessThanOrEqual(cut.box.right + 0.5)
  expect(cut.first.left).toBeGreaterThanOrEqual(cut.box.left - 0.5)

  // a life's bar writes the name and the years in one line: the name first, then the
  // year of birth, then the year of death
  await choose(page, 'Chart', chart.shapes.timeline)
  const life = page.locator('.chart [data-person="al-halabi-zaynab"] text')
  await expect(life).toContainText(zaynab)
  const order = await life.evaluate((text) => {
    const span = text.querySelector('.lifespan')!.firstChild!
    const words = span.textContent!
    const at = (node: Node, from: number, length: number) => {
      const range = document.createRange()
      range.setStart(node, from)
      range.setEnd(node, from + length)
      return range.getBoundingClientRect()
    }
    const named = document.createRange()
    named.selectNodeContents(text.firstChild!)
    return {
      name: named.getBoundingClientRect().right,
      born: at(span, words.indexOf('1887'), 4).left,
      died: at(span, words.indexOf('1960'), 4).left,
    }
  })
  expect(order.born).toBeGreaterThanOrEqual(order.name - 0.5)
  expect(order.died).toBeGreaterThan(order.born)
  expect(said).toEqual([])
})
