import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import { cpSync, existsSync, readFileSync, rmSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { expect, test, type Page } from '@playwright/test'
import { complaints } from './complaints'
import { binary, handTo } from './origins'

// Editing in place, on a copy of the fixture's history served by an origins of its own:
// the page's requests are handed to it, carrying its origin and its token, so that a
// change made from the page lands in the copy and nowhere else. The copy is made afresh
// beside the corpora and left there, since a git the server started may still hold it
// when the server has gone.

const fixture = resolve('..', 'testgit')
const root = resolve('..', 'corpora', 'edit')
const port = 8216
const served = `http://127.0.0.1:${port}`

test.describe.configure({ mode: 'serial' })

let token = ''
let server: ChildProcess | undefined

test.beforeAll(async () => {
  test.skip(!existsSync(fixture), 'testgit is not built')
  test.setTimeout(5 * 60 * 1000)
  rmSync(root, { recursive: true, force: true, maxRetries: 5 })
  cpSync(fixture, root, { recursive: true })
  // a copy's files are newer than its index says, which git would otherwise put right
  // on the first change, reading every one of them
  execFileSync('git', ['-C', root, 'update-index', '-q', '--refresh'])
  server = spawn(binary, ['browse', '--no-open', '--listen', `127.0.0.1:${port}`, root], { stdio: 'ignore' })
  for (let i = 0; i < 300 && !token; i++) {
    try {
      token = (await (await fetch(served)).text()).match(/name="origins-token" content="([^"]+)"/)?.[1] ?? ''
    } catch {
      await new Promise((r) => setTimeout(r, 100))
    }
  }
  expect(token).not.toBe('')
})

test.afterEach(({ page }) => page.unrouteAll({ behavior: 'ignoreErrors' }))

test.afterAll(async () => {
  if (server && server.exitCode === null) {
    const exited = new Promise((r) => server!.once('exit', r))
    server.kill()
    await exited
  }
})

async function open(page: Page, ref: string) {
  await handTo(page, port)
  await page.goto(`/${ref}`)
  await expect(page.locator('main [data-path]').first()).toBeVisible()
}

const file = (path: string) => readFileSync(join(root, path), 'utf8')

async function seenOf(ref: string): Promise<string> {
  return (await (await fetch(`${served}/api/records/${ref}`)).json()).record.seen
}

test('sets a field from its drawer', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/al-halabi-1889')
  await page.locator('button[data-path="sex"]').click()
  const drawer = page.getByRole('dialog', { name: 'Sex' })
  await drawer.getByRole('combobox', { name: 'Value' }).click()
  await page.getByRole('option', { name: 'M', exact: true }).click()
  await drawer.getByRole('button', { name: 'Save' }).click()
  await expect(drawer).toBeHidden()
  await expect(page.locator('button.value[data-path="sex"]')).toHaveText('M')
  expect(file('people/al-halabi-1889/README.md')).toMatch(/^sex: M$/m)
  await expect(page.getByRole('tabpanel').getByText('Set sex on person/al-halabi-1889')).toBeVisible()
  expect(said).toEqual([])
})

test('offers nothing to remove where an entry is still to be added', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/al-halabi-1889')
  await page.locator('button[data-path="portrait"]').click()
  const drawer = page.getByRole('dialog', { name: 'Portrait' })
  await expect(drawer.getByRole('button', { name: 'Save' })).toBeVisible()
  await expect(drawer.getByRole('button', { name: 'Remove' })).toHaveCount(0)
  expect(said).toEqual([])
})

test('draws a path to the main person as it is now', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/al-halabi-1889')
  await page.getByRole('tab', { name: 'Charts' }).click()
  await page.getByRole('combobox', { name: 'Chart' }).click()
  await page.getByRole('option', { name: 'Relationship', exact: true }).click()
  const box = (slug: string) => page.locator(`.chart [data-person="${slug}"]`)
  await expect(box('verdier-jean-baptiste')).toBeVisible()
  execFileSync(binary, ['set', '-C', root, 'project', 'main-person', 'al-halabi-zaynab'])
  await expect(box('al-halabi-zaynab')).toBeVisible()
  execFileSync(binary, ['set', '-C', root, 'project', 'main-person', 'verdier-jean-baptiste'])
  await expect(box('verdier-jean-baptiste')).toBeVisible()
  expect(said).toEqual([])
})

test('adds to a list and takes from it', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/al-halabi-1889')
  await page.locator('button[data-path="tags[0]"]').click()
  const popover = page.getByRole('dialog')
  await popover.getByRole('textbox', { name: 'Tags' }).fill('stillborn')
  await popover.getByRole('textbox', { name: 'Tags' }).press('Enter')
  await expect(popover).toBeHidden()
  const tag = page.locator('button.value[data-path="tags[0]"]')
  await expect(tag).toHaveText('stillborn')
  expect(file('people/al-halabi-1889/README.md')).toMatch(/^tags: \[stillborn\]$/m)

  await tag.click()
  await page.getByRole('dialog').getByRole('button', { name: 'Remove' }).click()
  await expect(tag).toBeHidden()
  await expect(page.locator('button[data-path="tags[0]"]')).toBeVisible()
  expect(file('people/al-halabi-1889/README.md')).not.toMatch(/^tags:/m)
  expect(said).toEqual([])
})

test('previews a delete and makes it', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'note/a-london-wedding-of-1859')
  await page.getByRole('button', { name: 'Actions' }).click()
  await page.getByRole('menuitem', { name: 'Delete' }).click()
  const dialog = page.getByRole('dialog', { name: 'Delete' })
  await expect(dialog.locator('[data-preview]')).toContainText('notes/a-london-wedding-of-1859/README.md')
  expect(existsSync(join(root, 'notes', 'a-london-wedding-of-1859'))).toBe(true)
  await dialog.locator('[data-confirm]').click()
  await expect(page).toHaveURL(/\/$/)
  expect(existsSync(join(root, 'notes', 'a-london-wedding-of-1859', 'README.md'))).toBe(false)
  expect(said).toEqual([])
})

test('refuses an edit made to a record that changed since it was opened', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/ershov-michael')
  await page.locator('button[data-path="tags[1]"]').click()
  const popover = page.getByRole('dialog')
  await expect(popover.getByRole('textbox', { name: 'Tags' })).toBeVisible()

  const meanwhile = await fetch(`${served}/api/edit`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Origin: served, 'X-Origins-Token': token },
    body: JSON.stringify({
      edit: { op: 'add', entity: 'person/ershov-michael', path: 'tags', value: 'meanwhile' },
      seen: await seenOf('person/ershov-michael'),
    }),
  })
  expect(meanwhile.status).toBe(200)

  await popover.getByRole('textbox', { name: 'Tags' }).fill('late')
  await popover.getByRole('button', { name: 'Add' }).click()
  await expect(popover.locator('[data-refused="stale"]')).toBeVisible()
  expect(file('people/ershov-michael/README.md')).not.toMatch(/^tags: .*\blate\b/m)
  expect(said).toEqual(['Failed to load resource: the server responded with a status of 409 (Conflict)'])
})

test('gives somebody with no parents a family from a gap in the chart', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/rossi-bartolomeo')
  await page.getByRole('tab', { name: 'Charts' }).click()
  await page.locator('.chart [data-gap="rossi-bartolomeo 0"] button').click()
  const dialog = page.getByRole('dialog', { name: 'Add a parent' })
  await dialog.getByRole('combobox', { name: 'Parent' }).fill('John')
  await page.getByRole('option', { name: /John/ }).click()
  await expect(dialog.locator('[data-preview]')).toContainText('families/doe-john--unknown/README.md')
  await dialog.locator('[data-confirm]').click()
  await expect(dialog).toBeHidden()
  await expect(page.locator('.chart [data-person="doe-john"]')).toBeVisible()
  await expect(page.locator('.chart [data-gap="rossi-bartolomeo 0"]')).toBeVisible()
  expect(file('families/doe-john--unknown/README.md')).toMatch(/^partners: \[doe-john, unknown\]\nchildren: \[rossi-bartolomeo\]$/m)
  expect(said).toEqual([])
})

test('puts somebody new in the place of a parent unknown', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/galvao-helena')
  await page.getByRole('tab', { name: 'Charts' }).click()
  await page.locator('.chart [data-gap="galvao-helena 0"] button').click()
  const dialog = page.getByRole('dialog', { name: 'Add a parent' })
  await dialog.getByRole('button', { name: 'Somebody new' }).click()
  await dialog.getByRole('textbox', { name: 'Given names' }).fill('Pieter')
  await dialog.getByRole('textbox', { name: 'Surname' }).fill('Bakker')
  await expect(dialog.locator('[data-preview]')).toContainText('people/bakker-pieter/README.md')
  await dialog.locator('[data-confirm]').click()
  await expect(dialog).toBeHidden()
  await expect(page.locator('.chart [data-person="bakker-pieter"]')).toBeVisible()
  await expect(page.locator('.chart [data-gap="galvao-helena 0"]')).toHaveCount(0)
  expect(file('families/bakker-pieter--wit-geertruida/README.md')).toMatch(/^partners: \[wit-geertruida, bakker-pieter\]$/m)
  expect(said).toEqual([])
})

test('writes notes beside what they will look like', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/al-halabi-1889')
  await page.locator('button[data-path="body"]').click()
  const drawer = page.getByRole('dialog', { name: 'Notes' })
  await drawer.getByText('Side by side', { exact: true }).click()
  await drawer.getByRole('textbox', { name: 'Notes' }).fill('**Stillborn**, as the register says.')
  await expect(drawer.locator('[data-preview] strong')).toHaveText('Stillborn')
  await drawer.getByRole('button', { name: 'Save' }).click()
  await expect(drawer).toBeHidden()
  await expect(page.locator('main strong', { hasText: 'Stillborn' })).toBeVisible()
  expect(file('people/al-halabi-1889/README.md')).toMatch(/^\*\*Stillborn\*\*, as the register says\.$/m)
  expect(said).toEqual([])
})

test('says who is in a picture by drawing round them', async ({ page }) => {
  const said = complaints(page)
  await open(page, 'person/marchena-lucas')
  await page.locator('button[data-path="media[0]"]').first().click()
  const lightbox = page.getByRole('dialog', { name: "Lucas's christening, in his mother's arms" })
  const picture = lightbox.locator('[data-picture]')
  await expect(picture.locator('img')).toBeVisible()
  const box = (await picture.boundingBox())!
  await page.mouse.move(box.x + box.width * 0.1, box.y + box.height * 0.2)
  await page.mouse.down()
  await page.mouse.move(box.x + box.width * 0.3, box.y + box.height * 0.5, { steps: 4 })
  await page.mouse.move(box.x + box.width * 0.4, box.y + box.height * 0.9, { steps: 4 })
  await page.mouse.up()
  const tagging = lightbox.locator('[data-tagging]')
  await tagging.getByRole('combobox', { name: 'Who' }).fill('Pablo')
  await page.getByRole('option', { name: /Pablo/ }).first().click()
  await tagging.getByRole('button', { name: 'Tag' }).click()
  await expect(tagging).toBeHidden()
  await expect(lightbox.locator('[data-region]')).toHaveCount(1)
  expect(file('people/marchena-lucas/README.md')).toMatch(/^ {6}- person: marchena-pablo[^\n]*\n {8}region: \[0\.1\d*, 0\.\d+, 0\.\d+, 0\.\d+\]$/m)
  await page.keyboard.press('Escape')
  await expect(lightbox).toBeHidden()
  await expect(page.locator('button[data-path="media[0]"] [data-region]').first()).toBeAttached()
  expect(said).toEqual([])
})
