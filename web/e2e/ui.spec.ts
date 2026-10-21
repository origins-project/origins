import { expect, test, type Locator } from '@playwright/test'
import { complaints } from './complaints'

// /ui draws every component twice, in a light half and a dark one, whichever theme
// the system asks for; the root follows the system. Each half's overlays open in its
// own theme.

const background = (locator: Locator) => locator.evaluate((el) => getComputedStyle(el).backgroundColor)

const themes = { light: { bg: 'rgb(245, 240, 232)', surface: 'rgb(255, 253, 249)' }, dark: { bg: 'rgb(21, 17, 13)', surface: 'rgb(29, 25, 21)' } }

for (const scheme of ['light', 'dark'] as const) {
  test.describe(`the system in ${scheme}`, () => {
    test.use({ colorScheme: scheme })

    test('draws each half in its own theme', async ({ page }) => {
      const said = complaints(page)
      await page.goto('/ui')
      for (const theme of ['light', 'dark'] as const) {
        const half = page.getByTestId(theme)
        await expect(half.getByRole('heading', { level: 1, name: /^Components/ })).toBeVisible()
        expect(await background(half)).toBe(themes[theme].bg)
      }
      expect(await background(page.locator('body'))).toBe(themes[scheme].bg)
      expect(said).toEqual([])
    })

    test('uses the bundled faces and nothing from elsewhere', async ({ page, baseURL }) => {
      const elsewhere: string[] = []
      page.on('request', (request) => {
        const url = new URL(request.url())
        if (url.protocol.startsWith('http') && url.origin !== new URL(baseURL!).origin) elsewhere.push(request.url())
      })
      await page.goto('/ui')
      const faces = await page.evaluate(async () => {
        const loaded = async (font: string) => (await document.fonts.load(font)).map((f) => f.status)
        return {
          sans: await loaded('16px "Inter Variable"'),
          serif: await loaded('16px "Newsreader Variable"'),
          italic: await loaded('italic 16px "Newsreader Variable"'),
        }
      })
      expect(faces.sans).toContain('loaded')
      expect(faces.serif).toContain('loaded')
      expect(faces.italic).toContain('loaded')
      expect(elsewhere).toEqual([])
    })

    test('outlines the entry on a scan as it outlines who is in a picture', async ({ page }) => {
      await page.goto('/ui')
      const half = page.getByTestId('light')
      const outline = async (box: Locator) =>
        box.evaluate((el) => {
          const s = getComputedStyle(el)
          return [s.borderTopStyle, s.borderTopWidth, s.borderTopColor, s.borderRadius, s.boxShadow]
        })
      const scan = half.locator('.scan [data-region]').first()
      await expect(scan).toBeVisible()
      expect(await outline(scan)).toEqual(await outline(half.locator('.thumb [data-region]').first()))
    })

    test('draws only the rows of the table in view', async ({ page }) => {
      await page.goto('/ui')
      const table = page.getByTestId('light').getByTestId('table')
      await expect(table.getByRole('table')).toHaveAttribute('aria-rowcount', '2001')
      const rows = table.locator('tbody tr.row')
      await expect(rows.first()).toBeVisible()
      expect(await rows.count()).toBeLessThan(60)

      await table.evaluate((el) => el.scrollTo(0, el.scrollHeight))
      await expect(table.locator('tbody tr[aria-rowindex="2001"]')).toBeVisible()
      expect(await rows.count()).toBeLessThan(60)

      // sorted by name to start with; pressing the birth column sorts by year, the
      // undated last
      await table.evaluate((el) => el.scrollTo(0, 0))
      await table.getByRole('button', { name: 'Born' }).click()
      await expect(table.getByRole('columnheader', { name: 'Born' })).toHaveAttribute('aria-sort', 'ascending')
      await expect(rows.first().locator('td').nth(1)).toHaveText('ABT 1700')

      await rows.nth(2).click()
      await expect(rows.nth(2)).toHaveAttribute('aria-current', 'true')
    })

    for (const theme of ['light', 'dark'] as const) {
      test(`opens the ${theme} half's overlays in its theme`, async ({ page }) => {
        const said = complaints(page)
        await page.goto('/ui')
        const half = page.getByTestId(theme)
        const surface = themes[theme].surface

        await half.getByRole('button', { name: 'Delete…' }).click()
        const dialog = page.getByRole('dialog', { name: 'Delete Giovanni Rossi?' })
        await expect(dialog).toBeVisible()
        expect(await background(dialog)).toBe(surface)
        await page.keyboard.press('Escape')
        await expect(dialog).toBeHidden()

        await half.getByRole('button', { name: 'Edit event' }).click()
        const drawer = page.getByRole('dialog', { name: 'Birth' })
        await expect(drawer).toBeVisible()
        expect(await background(drawer)).toBe(surface)
        await drawer.getByRole('button', { name: 'Cancel' }).click()
        await expect(drawer).toBeHidden()

        await half.getByRole('button', { name: 'Lists' }).click()
        const nav = page.getByRole('dialog', { name: 'Lists' })
        await expect(nav).toBeVisible()
        await page.keyboard.press('Escape')
        await expect(nav).toBeHidden()

        await half.getByRole('button', { name: 'More' }).last().click()
        const menu = page.getByRole('menu')
        await expect(menu).toBeVisible()
        expect(await background(menu)).toBe(surface)
        await expect(menu.getByRole('menuitemcheckbox', { name: 'Show private' })).toHaveAttribute('aria-checked', 'true')
        await page.keyboard.press('Escape')
        await expect(menu).toBeHidden()

        await half.getByRole('button', { name: 'Edit in place' }).click()
        const popover = page.getByRole('dialog', { name: 'Edit in place' })
        await expect(popover).toBeVisible()
        await popover.getByRole('button', { name: 'Save' }).click()
        await expect(popover).toBeHidden()

        await half.getByRole('combobox', { name: 'Sex' }).first().click()
        const listbox = page.getByRole('listbox')
        await expect(listbox).toBeVisible()
        await listbox.getByRole('option', { name: 'Female' }).click()
        await expect(half.getByRole('combobox', { name: 'Sex' }).first()).toHaveText(/Female/)

        expect(said).toEqual([])
      })
    }
  })
}
