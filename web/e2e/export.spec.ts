import { execFileSync } from 'node:child_process'
import { existsSync, mkdtempSync, readFileSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { pathToFileURL } from 'node:url'
import { expect, test, type Page } from '@playwright/test'
import { kindsInOrder, type Index } from '../src/api/client'
import type { Shape } from '../src/chart'
import { chart } from '../src/labels'
import { listed } from '../src/search'
import { complaints } from './complaints'
import { binary, shared } from './origins'

// The export of testgit opened from its folder, with no server behind it: every list,
// every record and every chart, each link on them leading somewhere the export holds and
// no file the page asks for missing.

const { root } = shared.fixture
const shapes = Object.keys(chart.shapes) as Shape[]
const notice = 'An invented family.'

test.describe.configure({ mode: 'serial' })

let out = ''
let index: Index

test.beforeAll(() => {
  if (!existsSync(root)) return
  out = join(mkdtempSync(join(tmpdir(), 'origins-export-')), 'site')
  execFileSync(binary, ['export', 'html', '--notice', notice, '--out', out], { cwd: root })
  const script = readFileSync(join(out, 'data', 'api', 'index.js'), 'utf8')
  index = JSON.parse(script.slice(script.indexOf(',') + 1, script.lastIndexOf(')')))
})

test.afterAll(() => {
  if (out) rmSync(join(out, '..'), { recursive: true, force: true })
})

const opened = (route: string) => `${pathToFileURL(join(out, 'index.html')).href}#${route}`

// What went wrong on the page: what it logged, and every file it asked for and did not get.
function failures(page: Page): string[] {
  const said = complaints(page)
  page.on('requestfailed', (request) => said.push(`${request.url()}: ${request.failure()?.errorText}`))
  return said
}

// Every link on the page leads to a route of a record the export holds, a list, a page
// it has, or a file it wrote; a link off to the web is the record's own.
async function linksResolve(page: Page) {
  const refs = new Set(kindsInOrder.flatMap((kind) => index[listed[kind]].map((one) => one.ref)))
  const lists = new Set(Object.values(listed).map((list) => `/${list}`))
  const hrefs = await page.locator('a[href]').evaluateAll((links) => links.map((a) => a.getAttribute('href')!))
  for (const href of hrefs) {
    if (/^(https?|mailto|tel):/.test(href)) continue
    if (href.startsWith('#/')) {
      const route = decodeURIComponent(href.slice(1).replace(/[?#].*$/, ''))
      expect(
        route === '/' || route === '/findings' || route === '/info' || lists.has(route) || refs.has(route.slice(1)),
        `${href} on ${page.url()}`,
      ).toBe(true)
    } else if (href.startsWith('#')) {
      continue // an anchor on the page itself
    } else {
      expect(existsSync(join(out, decodeURIComponent(href))), `${href} on ${page.url()}`).toBe(true)
    }
  }
}

test('an export opened from its folder shows every list and record, every link resolving', async ({ page }) => {
  test.skip(!existsSync(root), 'testgit is not built')
  test.setTimeout(5 * 60 * 1000)
  const said = failures(page)

  for (const route of ['/', '/findings', '/info', ...Object.values(listed).map((list) => `/${list}`)]) {
    await page.goto(opened(route))
    await expect(page.getByRole('heading', { level: 1 }).first()).toBeVisible()
    await linksResolve(page)
  }
  for (const kind of kindsInOrder) {
    for (const one of index[listed[kind]]) {
      await page.goto(opened(`/${one.ref}`))
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      await expect(page.getByRole('tab', { name: 'History' })).toHaveCount(0)
      await expect(page.locator('[data-path="actions"]')).toHaveCount(0)
      await linksResolve(page)
    }
  }
  expect(said).toEqual([])
})

test('an export draws every chart, and offers nothing a server would have to answer', async ({ page }) => {
  test.skip(!existsSync(root), 'testgit is not built')
  test.setTimeout(3 * 60 * 1000)
  const said = failures(page)
  const ties = (p: Index['people'][number]) => (p['child-in']?.length ?? 0) + (p['partner-in']?.length ?? 0)
  const person = index.people.reduce((a, b) => (ties(b) > ties(a) ? b : a))
  const other = index.people.find((p) => p.ref !== person.ref && p['child-in']?.length)!
  await page.goto(opened(`/${person.ref}`))
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Suppose' })).toHaveCount(0)
  await page.getByRole('tab', { name: 'Charts' }).click()
  for (const shape of shapes) {
    await page.getByRole('combobox', { name: 'Chart' }).click()
    await page.getByRole('option', { name: chart.shapes[shape], exact: true }).click()
    if (shape === 'path') {
      const slug = other.ref.slice('person/'.length)
      await page.getByRole('combobox', { name: 'To' }).fill(slug)
      await page.getByRole('option', { name: new RegExp(slug) }).click()
    }
    await expect(page.locator('.chart [data-person]').first()).toBeVisible({ timeout: 30 * 1000 })
    await expect(page.locator('.chart [data-gap]')).toHaveCount(0)
    await linksResolve(page)
  }
  expect(said).toEqual([])
})

test('an export shows its notice on every page, and says on Info what it was made from', async ({ page }) => {
  test.skip(!existsSync(root), 'testgit is not built')
  const said = failures(page)
  const head = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim()
  for (const route of ['/', `/${index.people[0]!.ref}`]) {
    await page.goto(opened(route))
    await expect(page.locator('[data-notice]')).toHaveText(notice)
  }
  await expect(page.locator('[data-supposed]')).toHaveCount(0)

  await page.getByRole('button', { name: 'More' }).click()
  await expect(page.getByRole('menuitemcheckbox')).toHaveCount(0)
  await page.getByRole('menuitem', { name: 'Info' }).click()
  await expect(page).toHaveURL(/#\/info$/)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Info')
  await expect(page.locator('[data-path="export.commit"]')).toHaveText(head.slice(0, 7))
  await expect(page.locator('[data-path="export.view"]')).toHaveText('Everything')
  await expect(page.locator('[data-path="export.notice"]')).toHaveText(notice)
  await expect(page.locator('[data-path="export.at"]')).toHaveText(/^\d{4}-\d\d-\d\d \d\d:\d\d$/)
  await expect(page.getByRole('heading', { name: 'Researchers' }).first()).toBeVisible()
  await expect(page.getByRole('heading', { name: 'origins.yaml' })).toHaveCount(0)
  await expect(page.getByRole('heading', { name: 'Git' })).toHaveCount(0)
  await linksResolve(page)
  expect(said).toEqual([])
})
