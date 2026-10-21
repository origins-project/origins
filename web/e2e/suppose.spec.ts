import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import { cpSync, existsSync, readFileSync, rmSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { expect, test, type Page } from '@playwright/test'
import { complaints } from './complaints'
import { binary, handTo, served } from './origins'

// Supposing, on a copy of the fixture's history served by an origins of its own, which
// a test may stop and start again: the edits go to the draft and not into the files,
// and the draft kept is a hypothesis that draws the pages as they were drawn with it.

const fixture = resolve('..', 'testgit')
const root = resolve('..', 'corpora', 'suppose')
const port = 8214

test.describe.configure({ mode: 'serial' })

let server: ChildProcess | undefined

async function serve(at: number, ...flags: string[]): Promise<ChildProcess> {
  const started = spawn(binary, ['browse', '--no-open', '--listen', `127.0.0.1:${at}`, ...flags, root], { stdio: 'ignore' })
  await served(at)
  return started
}

async function halt(one: ChildProcess | undefined) {
  if (!one || one.exitCode !== null) return
  const exited = new Promise((r) => one.once('exit', r))
  one.kill()
  await exited
}

test.beforeAll(async () => {
  test.skip(!existsSync(fixture), 'testgit is not built')
  test.setTimeout(5 * 60 * 1000)
  rmSync(root, { recursive: true, force: true, maxRetries: 5 })
  cpSync(fixture, root, { recursive: true })
  execFileSync('git', ['-C', root, 'update-index', '-q', '--refresh'])
  server = await serve(port)
})

test.afterEach(({ page }) => page.unrouteAll({ behavior: 'ignoreErrors' }))

test.afterAll(() => halt(server))

async function visit(page: Page, ref: string) {
  await page.goto(`/${ref}`)
  await expect(page.locator('main [data-path]').first()).toBeVisible()
}

async function suppose(page: Page, ref: string) {
  await handTo(page, port)
  await visit(page, ref)
  await page.getByRole('button', { name: 'Suppose' }).click()
  await expect(page.locator('[data-draft]')).toBeVisible()
}

async function tag(page: Page, word: string) {
  await page.locator('main button[data-path="tags[0]"]').click()
  const popover = page.getByRole('dialog')
  await popover.getByRole('textbox', { name: 'Tags' }).fill(word)
  await popover.getByRole('textbox', { name: 'Tags' }).press('Enter')
  await expect(popover).toBeHidden()
  await expect(page.locator('main button.value[data-path="tags[0]"][data-drafted]')).toHaveText(word)
}

const file = (path: string) => readFileSync(join(root, path), 'utf8')

async function fields(at: number, ref: string, query = ''): Promise<unknown> {
  return (await (await fetch(`http://127.0.0.1:${at}/api/records/${ref}${query}`)).json()).record.fields
}

const three = ['person/al-halabi-1889', 'place/kumasi', 'note/the-boatman-of-wenzhou']

test('keeps edits across three records as one hypothesis', async ({ page }) => {
  const said = complaints(page)
  await suppose(page, three[0])
  await expect(page.getByRole('tab', { name: 'History' })).toBeHidden()
  for (const [i, ref] of three.entries()) {
    if (i) await visit(page, ref)
    await tag(page, `supposed-${i}`)
    await expect(page.locator('[data-draft] [data-edits]')).toHaveText(`${i + 1} ${i ? 'edits' : 'edit'}`)
  }
  await expect(page.locator('[data-draft]')).toContainText('3 records')
  expect(file('people/al-halabi-1889/README.md')).not.toMatch(/^tags:/m)
  const drafted = await Promise.all(three.map((ref) => fields(port, ref, '?draft')))

  await page.locator('[data-draft] button[data-path="draft"]').click()
  const drawer = page.getByRole('dialog', { name: 'Create hypothesis' })
  await expect(drawer.locator('li')).toHaveCount(3)
  await drawer.getByRole('textbox', { name: 'Question' }).fill('Were they tagged so?')
  await expect(drawer.getByRole('textbox', { name: 'Slug' })).toHaveValue('were-they-tagged-so')
  await drawer.locator('button[data-path="supports[0]"]').click()
  const supports = drawer.locator('[data-path="supports[0]"]')
  await supports.getByRole('combobox', { name: 'Supports' }).fill('f214')
  await page.getByRole('option', { name: /f214/ }).click()
  await supports.getByRole('combobox', { name: 'Source' }).click()
  await page.getByRole('option', { name: 'O', exact: true }).click()
  await supports.getByRole('combobox', { name: 'Weight' }).click()
  await page.getByRole('option', { name: '2', exact: true }).click()
  await drawer.locator('[data-confirm]').click()
  await expect(page).toHaveURL(/\/hypothesis\/were-they-tagged-so$/)
  await expect(page.locator('[data-draft]')).toBeHidden()
  expect(file('hypotheses/were-they-tagged-so/README.md')).toMatch(/^# Were they tagged so\?$/m)
  expect(file('hypotheses/were-they-tagged-so/README.md')).toMatch(/^supports: \[census-1861-england#f214 O--2\]$/m)
  expect(file('people/al-halabi-1889/README.md')).not.toMatch(/^tags:/m)

  const supposing = await serve(8215, '--with', 'were-they-tagged-so')
  try {
    expect(await Promise.all(three.map((ref) => fields(8215, ref)))).toEqual(drafted)
  } finally {
    await halt(supposing)
  }
  expect(said).toEqual([])
})

test('takes up a draft with the server started again', async ({ page }) => {
  const said = complaints(page)
  await suppose(page, three[0])
  await tag(page, 'kept-over')
  await halt(server)
  server = await serve(port)
  await visit(page, three[0])
  await expect(page.locator('[data-draft] [data-edits]')).toHaveText('1 edit')
  await expect(page.locator('main button.value[data-path="tags[0]"][data-drafted]')).toHaveText('kept-over')
  await page.locator('[data-draft]').getByRole('button', { name: 'Discard' }).click()
  await expect(page.locator('[data-draft] [data-edits]')).toHaveText('0 edits')
  await expect(page.locator('main button[data-path="tags[0]"]')).not.toHaveText('kept-over')
  expect(said.filter((s) => !/ERR_CONNECTION|WebSocket/.test(s))).toEqual([])
})

test('shows a draft that drifted with what drifted', async ({ page }) => {
  const said = complaints(page)
  await suppose(page, 'person/chen-yuying')
  await page.locator('main button[data-path="sex"]').click()
  const drawer = page.getByRole('dialog', { name: 'Sex' })
  await drawer.getByRole('combobox', { name: 'Value' }).click()
  await page.getByRole('option', { name: 'M', exact: true }).click()
  await drawer.getByRole('button', { name: 'Save' }).click()
  await expect(drawer).toBeHidden()
  await expect(page.locator('main button.value[data-path="sex"][data-drafted]')).toHaveText('M')

  execFileSync(binary, ['set', '-C', root, 'person/chen-yuying', 'sex', 'X'])
  await expect(page.locator('[data-drifted]')).toBeVisible()
  await expect(page.locator('[data-draft] button[data-path="draft"]')).toBeDisabled()
  await expect(page.locator('main button.value[data-path="sex"]')).toHaveText('X')
  await page.locator('[data-draft]').getByRole('button', { name: 'Discard' }).click()
  await expect(page.locator('[data-drifted]')).toBeHidden()
  expect(said).toEqual([])
})

test('says why a draft was not thrown away from Home', async ({ page }) => {
  const said = complaints(page)
  await suppose(page, three[0])
  await tag(page, 'left-over')
  await page.locator('[data-draft]').getByRole('button', { name: 'Stop' }).click()
  await page.goto('/')
  const waiting = page.locator('main [data-path="draft"]')
  await expect(waiting).toBeVisible()
  await page.route('**/api/draft', (route) =>
    route.request().method() === 'DELETE'
      ? route.fulfill({ status: 423, json: { kind: 'lock', error: 'another change is being made' } })
      : route.fallback(),
  )
  await waiting.getByRole('button', { name: 'Discard' }).click()
  await expect(page.locator('.toast')).toHaveText('another change is being made')
  await expect(waiting).toBeVisible()
  await page.unroute('**/api/draft')
  await waiting.getByRole('button', { name: 'Discard' }).click()
  await expect(waiting).toBeHidden()
  expect(said.filter((s) => !/the server responded with a status of 423/.test(s))).toEqual([])
})
