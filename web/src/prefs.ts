// What a browser keeps of how the pages are shown: the theme for every tab, chosen once,
// and the view, the hypotheses supposed and supposing for the one tab they were
// switched on in. A browser that keeps nothing shows the pages as they are first shown.

import type { Hiding } from './api/client'

export type ThemeName = 'light' | 'dark'

const themeKey = 'origins-theme'
const viewKey = 'origins-view'
const supposingKey = 'origins-supposing'
const supposedKey = 'origins-supposed'

function read(storage: () => Storage, key: string): string | null {
  try {
    return storage().getItem(key)
  } catch {
    return null
  }
}

function write(storage: () => Storage, key: string, value: string | null) {
  try {
    if (value === null) storage().removeItem(key)
    else storage().setItem(key, value)
  } catch {
    // a browser that keeps nothing forgets
  }
}

const local = () => window.localStorage
const session = () => window.sessionStorage

// The theme chosen, or undefined while the pages follow the system.
export function theme(): ThemeName | undefined {
  const kept = read(local, themeKey)
  return kept === 'light' || kept === 'dark' ? kept : undefined
}

export function keepTheme(chosen: ThemeName) {
  write(local, themeKey, chosen)
}

export function view(): Hiding {
  const kept = read(session, viewKey)
  return { private: kept?.includes('private') ?? false, living: kept?.includes('living') ?? false }
}

export function keepView(hiding: Hiding) {
  const kept = [hiding.private && 'private', hiding.living && 'living'].filter(Boolean).join(' ')
  write(session, viewKey, kept || null)
}

export function supposing(): boolean {
  return read(session, supposingKey) === 'on'
}

export function keepSupposing(on: boolean) {
  write(session, supposingKey, on ? 'on' : null)
}

export function supposed(): string[] {
  return read(session, supposedKey)?.split(' ').filter(Boolean) ?? []
}

export function keepSupposed(slugs: string[]) {
  write(session, supposedKey, slugs.length ? slugs.join(' ') : null)
}
