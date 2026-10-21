import type { Page } from '@playwright/test'

// The page's own warnings count as failures: a Vue warning is a component misused.
export function complaints(page: Page): string[] {
  const out: string[] = []
  page.on('pageerror', (error) => out.push(error.message))
  page.on('console', (message) => {
    if (message.type() === 'error' || message.type() === 'warning') out.push(message.text())
  })
  return out
}
