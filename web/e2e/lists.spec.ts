import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import { cpSync, existsSync, readFileSync, rmSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { expect, test, type Page } from '@playwright/test'
import { complaints } from './complaints'
import { binary, handTo, served, shared } from './origins'

// The lists, read from the large corpus, and Settings, writing to a copy of the
// fixture's history that is shared through a bare repository beside it. The large corpus
// is served by the origins the run shares and the copy by one of its own, and the page's
// requests are handed to it, a write carrying that server's origin and token.

const large = shared.large
const fixture = resolve('..', 'testgit')
const copy = { root: resolve('..', 'corpora', 'lists'), port: 8220 }
const bare = resolve('..', 'corpora', 'lists-bare.git')
const other = resolve('..', 'corpora', 'lists-other')

const lists = ['people', 'families', 'places', 'sources', 'repositories', 'notes', 'hypotheses'] as const

const run = (args: string[]) => execFileSync(binary, args, { stdio: 'pipe' }).toString()

test.describe.configure({ mode: 'serial' })

const started: ChildProcess[] = []

async function serve(root: string, port: number) {
  started.push(spawn(binary, ['browse', '--no-open', '--listen', `127.0.0.1:${port}`, root], { stdio: 'ignore' }))
  await served(port)
}

test.beforeAll(async () => {
  test.setTimeout(5 * 60 * 1000)
  if (existsSync(large.root)) await served(large.port)
  if (existsSync(fixture)) {
    for (const dir of [copy.root, bare, other]) rmSync(dir, { recursive: true, force: true, maxRetries: 5 })
    cpSync(fixture, copy.root, { recursive: true })
    // a copy's files are newer than its index says, which git would otherwise put right
    // on the first change, reading every one of them
    execFileSync('git', ['-C', copy.root, 'update-index', '-q', '--refresh'])
    execFileSync('git', ['clone', '--bare', '--quiet', copy.root, bare])
    await serve(copy.root, copy.port)
  }
})

test.afterEach(({ page }) => page.unrouteAll({ behavior: 'ignoreErrors' }))

test.afterAll(async () => {
  for (const server of started) {
    if (server.exitCode !== null) continue
    const exited = new Promise((r) => server.once('exit', r))
    server.kill()
    await exited
  }
})

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })
const names = (page: Page) => page.locator('tbody tr.row td:first-child .text.strong').allInnerTexts()

test.describe('the lists of the large corpus', () => {
  for (const list of lists) {
    test(`sorts and filters ${list}`, async ({ page }) => {
      test.skip(!existsSync(large.root), 'large is not built')
      test.setTimeout(2 * 60 * 1000)
      const said = complaints(page)
      const index = await (await fetch(`http://127.0.0.1:${large.port}/api/index`)).json()
      const all: number = index[list].length
      await handTo(page, large.port)
      await page.goto(`/${list}`)
      const shown = page.locator('[data-path="shown"]')
      await expect(shown).toHaveText(`${all} of ${all}`, { timeout: 60 * 1000 })

      const heading = page.locator('th').first()
      await heading.getByRole('button').click()
      await expect(heading).toHaveAttribute('aria-sort', 'ascending')
      const up = await names(page)
      expect(up.length).toBeGreaterThan(0)
      expect([...up].sort(collator.compare)).toEqual(up)
      await heading.getByRole('button').click()
      await expect(heading).toHaveAttribute('aria-sort', 'descending')
      const down = await names(page)
      expect([...down].sort(collator.compare).reverse()).toEqual(down)

      if (all > 1) {
        const last = up.at(-1)!.toLowerCase()
        const word = up[0]!.match(/[\p{L}\d]+/gu)!.findLast((w) => !last.includes(w.toLowerCase())) ?? up[0]!
        await page.getByRole('searchbox', { name: 'Filter' }).fill(word)
        await expect.poll(() => new URL(page.url()).searchParams.get('q')).toBe(word)
        await expect(shown).not.toHaveText(`${all} of ${all}`)
        const filtered = Number((await shown.innerText()).split(' ')[0])
        expect(filtered).toBeGreaterThan(0)
        expect(filtered).toBeLessThan(all)
        await page.getByRole('searchbox', { name: 'Filter' }).fill('')
        await expect(shown).toHaveText(`${all} of ${all}`)
      }

      if (list === 'places') {
        await page.getByRole('button', { name: 'Tree' }).click()
        await expect(page.getByRole('tree').getByRole('treeitem').first()).toBeVisible()
      }
      expect(said).toEqual([])
    })
  }
})

test('sets each rule of the findings in a card as tall as what it holds', async ({ page }) => {
  test.skip(!existsSync(large.root), 'large is not built')
  const said = complaints(page)
  await page.setViewportSize({ width: 1400, height: 900 })
  await handTo(page, large.port)
  await page.goto('/findings')
  const cards = page.locator('[data-rule]')
  await expect(cards.nth(1)).toBeVisible({ timeout: 60 * 1000 })
  const empty = await cards.evaluateAll((all) =>
    all.flatMap((card) => {
      const s = getComputedStyle(card)
      const held = [...card.children].reduce((sum, c) => sum + c.getBoundingClientRect().height, 0)
      const over = card.getBoundingClientRect().height - held - parseFloat(s.borderTopWidth) - parseFloat(s.borderBottomWidth)
      return over > 1 ? [`${card.getAttribute('data-rule')} ${Math.round(over)}px`] : []
    }),
  )
  expect(empty).toEqual([])
  expect(said).toEqual([])
})

test.describe('Settings', () => {
  const file = (path: string) => readFileSync(join(copy.root, path), 'utf8')

  test('renames a tag on every record carrying it', async ({ page }) => {
    test.skip(!existsSync(fixture), 'testgit is not built')
    const said = complaints(page)
    await handTo(page, copy.port)
    await page.goto('/settings')
    await page.getByRole('button', { name: 'Rename Emigrants' }).click()
    const dialog = page.getByRole('dialog', { name: 'Rename' })
    await dialog.getByRole('textbox').fill('Emigranti')
    await expect(dialog.locator('[data-preview]')).toContainText('origins.yaml')
    await dialog.locator('[data-confirm]').click()
    await expect(dialog).toBeHidden()
    await expect(page.locator('[data-tag="Emigranti"]')).toBeVisible()
    expect(file('origins.yaml')).toMatch(/^ {4}Emigranti:$/m)
    expect(file('origins.yaml')).not.toMatch(/Emigrants/)
    expect(file('people/ershov-michael/README.md')).toMatch(/\bEmigranti\b/)
    expect(said).toEqual([])
  })

  test('styles a tag whose word is no step of a path', async ({ page }) => {
    test.skip(!existsSync(fixture), 'testgit is not built')
    const said = complaints(page)
    const project = await (await fetch(`http://127.0.0.1:${copy.port}/api/records/project`)).json()
    const tags = { ...project.record.fields.tags, 'St. Mary [1859]': { color: '#36c' } }
    run(['set', '-C', copy.root, 'project', 'tags', JSON.stringify(tags)])
    await handTo(page, copy.port)
    await page.goto('/settings')
    await page.locator('button.value[data-path="tags.St. Mary [1859].color"]').click()
    const popover = page.getByRole('dialog')
    await popover.getByRole('textbox', { name: 'Colour' }).fill('#c36')
    await popover.getByRole('button', { name: 'Save' }).click()
    await expect(popover).toBeHidden()
    expect(file('origins.yaml')).toMatch(/^ {4}St\. Mary \[1859\]:\n {6}color: '#c36'$/m)
    expect(file('origins.yaml')).toMatch(/^ {4}To do:\n {6}color: '#d33'$/m)
    expect(said).toEqual([])
  })

  test('pushes to a remote and pulls what another clone pushed', async ({ page }) => {
    test.skip(!existsSync(fixture), 'testgit is not built')
    const said = complaints(page)
    await handTo(page, copy.port)
    await page.goto('/settings')
    await page.getByRole('textbox', { name: 'Name' }).fill('origin')
    await page.getByRole('textbox', { name: 'URL' }).fill(bare)
    await page.getByRole('button', { name: 'Add', exact: true }).click()
    await expect(page.locator('[data-remote="origin"]')).toBeVisible()

    const outgoing = page.locator('[data-path="outgoing"]')
    await outgoing.getByRole('button', { name: 'Push' }).click()
    await expect(outgoing).toContainText('origin/main')
    await expect(outgoing).toContainText('0 commits')
    const head = execFileSync('git', ['-C', copy.root, 'rev-parse', 'HEAD']).toString().trim()
    expect(execFileSync('git', ['-C', bare, 'rev-parse', 'main']).toString().trim()).toBe(head)

    run(['clone', bare, other])
    run(['set', '-C', other, 'project', 'description', 'Pushed from another clone'])
    run(['push', '-C', other])

    const incoming = page.locator('[data-path="incoming"]')
    await incoming.getByRole('button', { name: 'Fetch' }).click()
    await expect(incoming).toContainText('1 commit')
    await incoming.getByRole('button', { name: 'Pull' }).click()
    await expect(incoming).toContainText('0 commits')
    expect(file('origins.yaml')).toMatch(/^ {2}description: Pushed from another clone$/m)
    await expect(page.getByText('Pushed from another clone').first()).toBeVisible()
    expect(said).toEqual([])
  })
})

test.describe('a hypothesis an agent proposed', () => {
  const agent = 'research-agent'

  test('says who proposed it wherever it is shown', async ({ page }) => {
    test.skip(!existsSync(large.root), 'large is not built')
    test.setTimeout(2 * 60 * 1000)
    const said = complaints(page)
    const index = await (await fetch(`http://127.0.0.1:${large.port}/api/index`)).json()
    const all: { ref: string; 'proposed-by'?: string }[] = index.hypotheses
    const proposed = all.filter((h) => h['proposed-by'])
    expect(proposed.map((h) => h['proposed-by'])).toEqual([agent])
    await handTo(page, large.port)

    await page.goto('/hypotheses')
    const shown = page.locator('[data-path="shown"]')
    await expect(shown).toHaveText(`${all.length} of ${all.length}`, { timeout: 60 * 1000 })
    await expect(page.getByRole('columnheader', { name: 'Proposed by' })).toBeVisible()
    await page.getByRole('searchbox', { name: 'Filter' }).fill(agent)
    await expect(shown).toHaveText(`1 of ${all.length}`)
    await expect(page.locator('tbody tr.row')).toContainText(agent)

    // the open hypotheses read the index, and the last changes the commit's trailer
    await page.goto('/')
    await expect(page.locator('main [data-proposed-by]')).toHaveText([`Proposed by ${agent}`, `Proposed by ${agent}`], {
      timeout: 60 * 1000,
    })

    // the heading reads the file, and the last changes the commit's trailer
    await page.goto(`/${proposed[0]!.ref}`)
    await expect(page.locator('main [data-proposed-by]')).toHaveText([`Proposed by ${agent}`, `Proposed by ${agent}`], {
      timeout: 60 * 1000,
    })
    expect(said).toEqual([])
  })
})
