import { existsSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { expect, test, type Page } from '@playwright/test'
import { complaints } from './complaints'
import { handTo, served, shared } from './origins'

// A record's page: every path the server says can be edited is a control on the page or
// in a drawer it opens, for every record of the fixture, and the page of each kind draws
// on the sparse corpus and on the large one. The fixture and the large corpus are served
// by the origins the run shares, and the page's reads are handed to it.

const fixture = shared.fixture.root
const large = shared.large.root
const ports = { fixture: shared.fixture.port, large: shared.large.port }

test.describe.configure({ mode: 'parallel', retries: 0 })

test.beforeAll(async () => {
  test.setTimeout(5 * 60 * 1000)
  for (const { root, port } of Object.values(shared)) if (existsSync(root)) await served(port)
})

test.afterEach(({ page }) => page.unrouteAll({ behavior: 'ignoreErrors' }))

async function noSidewaysScroll(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth), page.url()).toBeLessThanOrEqual(0)
}

// The paths of those given that something in the scope is a control for: an element
// carrying the path that is, or holds, a control that can be used, or for a list one
// carrying the path of an entry of it. On the page, the scope leaves out what belongs to
// another record, as a family's events do.
async function controlled(page: Page, scope: 'page' | 'drawer', entity: string, paths: string[]) {
  const found = await page.evaluate(
    ({ scope, entity }) => {
      const dialogs = document.querySelectorAll('[role="dialog"]')
      const root = scope === 'page' ? document.querySelector('main') : dialogs[dialogs.length - 1]
      if (!root) return []
      const usable =
        'button:not([disabled]), input:not([disabled]), textarea:not([disabled]), [role="combobox"]:not([data-disabled]), [role="switch"]:not([data-disabled])'
      const found = new Set<string>()
      for (const el of root.querySelectorAll('[data-path]')) {
        const owner = el.closest('[data-entity]')
        if (owner && owner.getAttribute('data-entity') !== entity) continue
        if (el.matches(usable) || el.querySelector(usable)) found.add(el.getAttribute('data-path')!)
      }
      return [...found]
    },
    { scope, entity },
  )
  return paths.filter((p) => found.some((f) => f === p || (f.startsWith(`${p}[`) && /^\[\d+\]$/.test(f.slice(p.length)))))
}

const under = (path: string, parent: string) => path.startsWith(`${parent}.`) || path.startsWith(`${parent}[`)

const triggers = (page: Page, scope: 'page' | 'drawer', entity: string) =>
  page.evaluate(
    ({ scope, entity }) => {
      const dialogs = document.querySelectorAll('[role="dialog"]')
      const root = scope === 'page' ? document.querySelector('main') : dialogs[dialogs.length - 1]
      if (!root) return []
      return [...root.querySelectorAll('button[data-path]:not([disabled])')]
        .filter((el) => {
          const owner = el.closest('[data-entity]')
          return !owner || owner.getAttribute('data-entity') === entity
        })
        .map((el) => el.getAttribute('data-path')!)
    },
    { scope, entity },
  )

// Looks for a control for each path wanted in the scope, the page or the drawer on top:
// it opens out every field a drawer holds in short, then presses each button carrying a
// path above one still wanted, which opens a drawer to look in the same way or draws
// what it adds where it is, and returns what it found.
async function explore(page: Page, entity: string, wanted: string[], depth: number): Promise<Set<string>> {
  const scope = depth === 0 ? 'page' : 'drawer'
  const dialogs = page.getByRole('dialog')
  const root = scope === 'page' ? page.locator('main') : dialogs.last()
  const found = new Set<string>()
  const look = async () => {
    if (scope === 'drawer') {
      const more = root.getByRole('button', { name: 'More fields' })
      for (let i = 0; i < 50 && (await more.count()); i++) await more.first().click()
    }
    for (const p of await controlled(page, scope, entity, wanted)) found.add(p)
  }
  await look()
  const pressed = new Set<string>()
  for (;;) {
    const next = (await triggers(page, scope, entity)).find(
      (t) => !pressed.has(t) && wanted.some((p) => under(p, t) && !found.has(p)),
    )
    if (next === undefined) break
    pressed.add(next)
    const before = await dialogs.count()
    const theirs = `[data-entity]:not([data-entity="${entity}"]) *`
    await root.locator(`button[data-path="${next}"]:not(${theirs})`).filter({ visible: true }).first().click()
    const opened = await page
      .waitForFunction((n) => document.querySelectorAll('[role="dialog"]').length > n, before, { timeout: 1000 })
      .then(() => true, () => false)
    if (!opened) {
      await look()
      continue
    }
    const below = wanted.filter((p) => under(p, next) && !found.has(p))
    for (const p of await explore(page, entity, below, depth + 1)) found.add(p)
    await page.keyboard.press('Escape')
    await expect(dialogs).toHaveCount(before)
  }
  return found
}

const kinds = {
  people: 'person',
  families: 'family',
  places: 'place',
  sources: 'source',
  repositories: 'repository',
  notes: 'note',
  hypotheses: 'hypothesis',
} as const
const records = Object.entries(kinds).flatMap(([dir, kind]) =>
  existsSync(resolve(fixture, dir)) ? readdirSync(resolve(fixture, dir)).map((slug) => `${kind}/${slug}`) : [],
)

test.describe('every record of the fixture', () => {
  test.skip(!existsSync(fixture), 'testgit is not built')

  for (const entity of records) {
    test(`can edit everything of ${entity}`, async ({ page }) => {
      test.setTimeout(2 * 60 * 1000)
      const said = complaints(page)
      const kind = entity.slice(0, entity.indexOf('/'))
      const { record } = await (await fetch(`http://127.0.0.1:${ports.fixture}/api/records/${entity}`)).json()
      await handTo(page, ports.fixture)
      await page.goto(`/${entity}`)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      await page.waitForLoadState('networkidle')
      // A path the store reaches and the reader refuses, as a mark on the name a person
      // is filed under, is no field a form can offer.
      const paths: string[] = await page.evaluate(
        async ({ kind, edits }) => {
          const { nodeAt } = await import('/src/edit/schema.ts')
          const schemas = await (await fetch('/api/schema')).json()
          return edits.filter((path) => nodeAt(schemas, kind, path) !== undefined)
        },
        { kind, edits: record.edits.map((e: { path: string }) => e.path) as string[] },
      )

      const found = await explore(page, entity, paths, 0)
      expect(paths.filter((p) => !found.has(p))).toEqual([])
      expect(said).toEqual([])
    })
  }
})

test.describe('the record pages', () => {
  for (const dir of Object.keys(kinds)) {
    test(`draw the first of ${dir} on the sparse corpus`, async ({ page, request }) => {
      const said = complaints(page)
      const one = (await (await request.get('/api/index')).json())[dir]?.[0]
      expect(one, `sparse has no ${dir}`).toBeTruthy()
      for (const width of [1400, 375]) {
        await page.setViewportSize({ width, height: 800 })
        await page.goto(`/${one.ref}`)
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
        await expect(page.getByRole('tab', { name: 'Overview' })).toBeVisible()
        await page.waitForLoadState('networkidle')
        await noSidewaysScroll(page)
      }
      expect(said).toEqual([])
    })
  }

  test('draw somebody with the most family on the large corpus', async ({ page }) => {
    test.skip(!existsSync(large), 'large is not built')
    // The index of the large corpus takes some seconds to pass through the route.
    test.setTimeout(2 * 60 * 1000)
    const said = complaints(page)
    const index = await (await fetch(`http://127.0.0.1:${ports.large}/api/index`)).json()
    const ties = (p: { 'partner-in'?: string[]; 'child-in'?: string[] }) =>
      (p['partner-in']?.length ?? 0) + (p['child-in']?.length ?? 0)
    const person = index.people.reduce((a: typeof index.people[0], b: typeof index.people[0]) => (ties(b) > ties(a) ? b : a))
    await handTo(page, ports.large)
    for (const width of [1400, 375]) {
      await page.setViewportSize({ width, height: 800 })
      await page.goto(`/${person.ref}`)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 30 * 1000 })
      await expect(page.locator('[data-moment]').first()).toBeVisible()
      await page.waitForLoadState('networkidle')
      await noSidewaysScroll(page)
    }
    await page.getByRole('tab', { name: 'History' }).click()
    await expect(page.locator('[data-commit]').first()).toBeVisible()
    expect(said).toEqual([])
  })

  test('draw the first of every other kind on the large corpus', async ({ page }) => {
    test.skip(!existsSync(large), 'large is not built')
    test.setTimeout(3 * 60 * 1000)
    const said = complaints(page)
    const index = await (await fetch(`http://127.0.0.1:${ports.large}/api/index`)).json()
    await handTo(page, ports.large)
    for (const dir of Object.keys(kinds).filter((d) => d !== 'people')) {
      const one = index[dir]?.[0]
      if (!one) continue
      await page.goto(`/${one.ref}`)
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible({ timeout: 30 * 1000 })
      await page.waitForLoadState('networkidle')
      await noSidewaysScroll(page)
    }
    expect(said).toEqual([])
  })
})
