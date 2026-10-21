import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import { cpSync, existsSync, rmSync } from 'node:fs'
import { resolve } from 'node:path'
import { expect, test, type Page } from '@playwright/test'
import { complaints } from './complaints'
import { binary, handTo, served, shared } from './origins'

// The frame around every page: the bar, its drawer, the search, the theme and the view
// switches. The development server reads the sparse corpus; the other corpora are each
// served by an origins, of their own or the one the run shares, and a page's reads are
// handed to it.


const corpora = [
  { name: 'empty', root: resolve('..', 'corpora', 'empty'), port: 8213 },
  { name: 'sparse' },
  { name: 'large', ...shared.large, shared: true },
  { name: 'testgit', ...shared.fixture, shared: true },
] as const

// The bar measures what fits a frame after the session and the findings are read.
async function settled(page: Page) {
  await page.waitForLoadState('networkidle')
  await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))))
}

async function noSidewaysScroll(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBeLessThanOrEqual(0)
}

const lists = (page: Page) => page.getByRole('navigation').getByRole('link', { name: 'People', exact: true })
const toggle = (page: Page) => page.getByRole('button', { name: 'Lists' })

test.describe('the frame', () => {
  test('puts the lists in a drawer when they stop fitting', async ({ page }) => {
    const said = complaints(page)
    for (const width of [1400, 1000, 375]) {
      await page.setViewportSize({ width, height: 800 })
      await page.goto('/')
      await expect(page.getByRole('link', { name: /Origins/ })).toBeVisible()
      await expect(lists(page).or(toggle(page))).toBeVisible()
      await settled(page)
      const inBar = await lists(page).isVisible()
      await expect(toggle(page), `at ${width}`).toBeVisible({ visible: !inBar })
      if (width === 1400) await expect(lists(page)).toBeVisible()
      if (width === 375) await expect(lists(page)).toBeHidden()
      if (inBar) {
        const links = (await page.getByRole('navigation').boundingBox())!
        const search = (await page.getByRole('button', { name: 'Search' }).boundingBox())!
        expect(links.x + links.width).toBeLessThanOrEqual(search.x)
      }
      await noSidewaysScroll(page)
    }

    await toggle(page).click()
    const drawer = page.getByRole('dialog', { name: 'Lists' })
    await expect(drawer.getByRole('link', { name: /Families/ })).toBeVisible()
    await drawer.getByRole('link', { name: /People/ }).click()
    await expect(page).toHaveURL(/\/people$/)
    await expect(drawer).toBeHidden()
    expect(said).toEqual([])
  })

  test('opens the search by keyboard and goes where it is told', async ({ page, request }) => {
    const said = complaints(page)
    const index = await (await request.get('/api/index')).json()
    const person = index.people[0]
    const name: string = person.name ?? person.ref.split('/')[1]
    await page.goto('/')
    await expect(page.getByRole('button', { name: 'Search' })).toBeVisible()

    await page.keyboard.press('/')
    const palette = page.getByRole('dialog', { name: 'Search' })
    await expect(palette).toBeVisible()
    await page.keyboard.type(name)
    await expect(palette.getByRole('option').first()).toHaveAttribute('data-highlighted')
    await page.keyboard.press('Enter')
    await expect(palette).toBeHidden()
    await expect(page).toHaveURL(new RegExp(`/${person.ref}$`))
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(name)

    await page.keyboard.press('Control+k')
    await expect(palette).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(palette).toBeHidden()
    expect(said).toEqual([])
  })

  test('keeps the theme chosen', async ({ page }) => {
    await page.emulateMedia({ colorScheme: 'light' })
    await page.goto('/')
    const root = page.locator('html')
    await expect(root).not.toHaveAttribute('data-theme')
    await page.getByRole('button', { name: 'Theme' }).click()
    await expect(root).toHaveAttribute('data-theme', 'dark')
    await page.reload()
    await expect(page.getByRole('button', { name: 'Theme' })).toBeVisible()
    await expect(root).toHaveAttribute('data-theme', 'dark')
    expect(await page.evaluate(() => getComputedStyle(document.body).backgroundColor)).toBe('rgb(21, 17, 13)')
  })

  test('reads a view switched on, and refuses an edit under it', async ({ page }) => {
    const said = complaints(page)
    await page.goto('/')
    const hidden = page.waitForRequest((r) => r.url().includes('/api/index?view=hide-private'))
    await page.getByRole('button', { name: 'More' }).click()
    await page.getByRole('menuitemcheckbox', { name: 'Hide private' }).click()
    await hidden
    await expect(page.getByRole('button', { name: 'Suppose' })).toBeDisabled()

    const refused = await page.evaluate(async () => {
      const token = document.querySelector<HTMLMetaElement>('meta[name="origins-token"]')!.content
      const response = await fetch('/api/edit?view=hide-private', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Origins-Token': token },
        body: JSON.stringify({ edit: { op: 'set', entity: 'project', path: 'name', value: 'x' }, seen: '' }),
      })
      return { status: response.status, kind: (await response.json()).kind }
    })
    expect(refused).toEqual({ status: 403, kind: 'read-only' })
    expect(said.splice(0)).toEqual(['Failed to load resource: the server responded with a status of 403 (Forbidden)'])

    await page.reload()
    await expect(page.getByRole('button', { name: 'Suppose' })).toBeDisabled()
    await page.getByRole('button', { name: 'More' }).click()
    const item = page.getByRole('menuitemcheckbox', { name: 'Hide private' })
    await expect(item).toHaveAttribute('aria-checked', 'true')
    await expect(page.getByRole('menuitemcheckbox', { name: 'Hide living' })).toHaveAttribute('aria-checked', 'false')
    await item.click()
    await expect(page.getByRole('button', { name: 'Suppose' })).toBeEnabled()
    expect(said).toEqual([])
  })
})

// A copy of the fixture of their own, served twice: once as origins is started
// ordinarily, and once locked to a view and a hypothesis.
test.describe('the frame of the hypotheses supposed', () => {
  test.describe.configure({ mode: 'serial' })
  const root = resolve('..', 'corpora', 'frame')
  const port = 8221
  const locked = 8222
  const started: ChildProcess[] = []

  test.beforeAll(async () => {
    if (!existsSync(shared.fixture.root)) return
    test.setTimeout(5 * 60 * 1000)
    rmSync(root, { recursive: true, force: true, maxRetries: 5 })
    cpSync(shared.fixture.root, root, { recursive: true })
    execFileSync('git', ['-C', root, 'update-index', '-q', '--refresh'])
    started.push(spawn(binary, ['browse', '--no-open', '--listen', `127.0.0.1:${port}`, root], { stdio: 'ignore' }))
    started.push(
      spawn(binary, ['browse', '--no-open', '--listen', `127.0.0.1:${locked}`, '--hide-living', '--with', 'louisa-born-1844', root], {
        stdio: 'ignore',
      }),
    )
    await Promise.all([served(port), served(locked)])
  })

  test.afterEach(({ page }) => page.unrouteAll({ behavior: 'ignoreErrors' }))

  test.afterAll(() => {
    for (const server of started) server.kill()
  })

  test('shows the hypotheses the tab supposes, each of which it can stop supposing', async ({ page }) => {
    test.skip(!started.length, 'testgit is not built')
    const said = complaints(page)
    await handTo(page, port)
    await page.goto('/hypothesis/louisa-born-1844')
    const question = page.getByRole('heading', { level: 1 })
    await expect(question).toBeVisible()
    const bar = page.locator('[data-supposed]')
    await expect(bar).toHaveCount(0)
    const supposed = page.waitForRequest((r) => r.url().includes('/api/index?with=louisa-born-1844'))
    await page.locator('button[data-path="suppose"]').click()
    await supposed
    const one = bar.locator('[data-hypothesis="louisa-born-1844"]')
    await expect(one.getByRole('link')).toHaveText((await question.textContent())!.trim())

    await page.goto('/')
    await expect(one).toBeVisible()
    await one.getByRole('link').click()
    await expect(page).toHaveURL(/\/hypothesis\/louisa-born-1844$/)
    await one.getByRole('button', { name: 'Stop supposing' }).click()
    await expect(bar).toHaveCount(0)
    await expect(page.locator('button[data-path="suppose"]')).toHaveAttribute('aria-pressed', 'false')
    expect(said).toEqual([])
  })

  test('shows the view checked and the hypothesis supposed, and switches neither', async ({ page }) => {
    test.skip(!started.length, 'testgit is not built')
    const said = complaints(page)
    await handTo(page, locked)
    await page.goto('/')
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    const one = page.locator('[data-supposed] [data-hypothesis="louisa-born-1844"]')
    await expect(one).toBeVisible()
    await expect(one).toHaveText('louisa-born-1844')
    await expect(one.getByRole('button')).toHaveCount(0)

    await page.getByRole('button', { name: 'More' }).click()
    const living = page.getByRole('menuitemcheckbox', { name: 'Hide living' })
    const priv = page.getByRole('menuitemcheckbox', { name: 'Hide private' })
    await expect(living).toHaveAttribute('aria-checked', 'true')
    await expect(living).toHaveAttribute('data-disabled')
    await expect(priv).toHaveAttribute('aria-checked', 'false')
    await expect(priv).toHaveAttribute('data-disabled')
    expect(said).toEqual([])
  })
})

test.describe('the frame on every corpus', () => {
  test.describe.configure({ mode: 'serial' })
  const started: ChildProcess[] = []

  test.beforeAll(async () => {
    test.setTimeout(5 * 60 * 1000)
    for (const corpus of corpora) {
      if (!('root' in corpus) || !existsSync(corpus.root)) continue
      if (!('shared' in corpus)) started.push(spawn(binary, ['browse', '--no-open', '--listen', `127.0.0.1:${corpus.port}`, corpus.root], { stdio: 'ignore' }))
      await served(corpus.port)
    }
  })

  test.afterAll(() => {
    for (const server of started) server.kill()
  })

  for (const corpus of corpora) {
    test(`draws the frame on ${corpus.name}`, async ({ page }) => {
      test.skip('root' in corpus && !existsSync(corpus.root), `${corpus.name} is not built`)
      const said = complaints(page)
      if ('port' in corpus) await handTo(page, corpus.port)
      for (const width of [1400, 375]) {
        await page.setViewportSize({ width, height: 800 })
        await page.goto('/')
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 30 * 1000 })
        await expect(page.getByRole('link', { name: 'Findings' })).toBeVisible()
        await settled(page)
        await noSidewaysScroll(page)
      }

      await page.keyboard.press('Control+k')
      const palette = page.getByRole('dialog', { name: 'Search' })
      await expect(palette).toBeVisible()
      await page.keyboard.type('a')
      if (corpus.name === 'large') {
        await expect(palette.getByRole('option').first()).toBeVisible()
        await page.keyboard.press('ArrowDown', { delay: 10 })
        for (let i = 0; i < 14; i++) await page.keyboard.press('ArrowDown')
        const highlighted = palette.locator('[data-highlighted]')
        await expect(highlighted).toBeInViewport()
        const list = (await palette.getByRole('listbox').boundingBox())!
        const box = (await highlighted.boundingBox())!
        expect(box.y).toBeGreaterThanOrEqual(list.y)
        expect(box.y + box.height).toBeLessThanOrEqual(list.y + list.height + 1)
      }
      await page.keyboard.press('Escape')
      expect(said).toEqual([])
    })
  }
})
