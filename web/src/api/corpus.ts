// What every page of the tab shares: the session, the index and the findings, read
// once and again whenever the corpus moves, whether the socket is open, the view, the
// hypotheses supposed and the supposing the tab switched on with the draft it writes,
// and the notices waiting at the corner.

import { computed, inject, onScopeDispose, ref, type InjectionKey } from 'vue'
import { Refused, type Client, type Drafting, type Event, type Findings, type Hiding, type Index, type Session } from './client'
import { notices as said } from '../labels'
import * as prefs from '../prefs'

export interface Toast {
  key: number
  text: string
  // what can be done about it, and doing it
  action?: string
  run?: () => void
}

// How many notices wait at once, the oldest going first.
const waiting = 3

export function createCorpus(client: Client) {
  const session = ref<Session>()
  const index = ref<Index>()
  const findings = ref<Findings>()
  const live = ref(false)
  const signIn = ref(false)
  const failed = ref<string>()
  const view = ref<Hiding>(prefs.view())
  const supposing = ref(prefs.supposing())
  const supposed = ref<string[]>(prefs.supposed())
  // the draft while the tab is supposing, null where there is none yet
  const draft = ref<Drafting | null>()
  // whether the tab's changes go to the draft, which no history or undo is offered under
  const drafting = computed(() => supposing.value && writes())
  // whether a change is offered at all, a server being behind the page to take one even
  // where it refuses it now; an export offers none, and draws no control that would
  const offered = computed(() => session.value?.live !== false)
  // whether the history can be asked of: a server keeps it, and the tab is not drafting
  const history = computed(() => session.value?.history !== false && !drafting.value)
  // whether a page can be drawn, the draft it is drawn with read where there is one
  const ready = computed(() => !drafting.value || draft.value !== undefined)
  // counted up whenever what the tab reads changes, so that a page is drawn again
  const shown = ref(0)
  // whether the tab is asking to take the last change back
  const undoing = ref(false)
  const toasts = ref<Toast[]>([])
  const listeners = new Set<(event: Event) => void>()
  const made = new Set<string>()
  let toasted = 0
  let stop: (() => void) | undefined

  function notify(text: string, action?: { label: string; run: () => void }) {
    const toast: Toast = { key: ++toasted, text, action: action?.label, run: action?.run }
    toasts.value = [...toasts.value, toast].slice(-waiting)
  }

  function act(key: number) {
    const toast = toasts.value.find((t) => t.key === key)
    dismiss(key)
    toast?.run?.()
  }

  function dismiss(key: number) {
    toasts.value = toasts.value.filter((t) => t.key !== key)
  }

  function apply() {
    const locked = session.value?.view != null
    // with no server behind the page nothing can be supposed, nor a draft written
    if (session.value?.live === false) {
      supposed.value = []
      supposing.value = false
    }
    client.view = locked ? { private: false, living: false } : { ...view.value }
    client.with = [...supposed.value]
    client.drafting = drafting.value
    if (client.drafting) document.documentElement.dataset.supposing = ''
    else delete document.documentElement.dataset.supposing
  }

  // Whether the tab reads a hidden view, its own or the one the server is locked to.
  function hidden(): boolean {
    const v = session.value?.view ?? view.value
    return v.private || v.living
  }

  async function read() {
    await readDraft()
    try {
      const [i, f] = await Promise.all([client.index(), client.findings()])
      index.value = i
      findings.value = f
      failed.value = undefined
    } catch (e) {
      failed.value = e instanceof Error ? e.message : String(e)
    }
  }

  // The draft, read while the tab writes to it, and forgotten when it stops. A read
  // overtaken by a later one waits for that one, so that whatever is read after it is
  // read with the draft as it now stands and not with one a change has since kept.
  let asked = 0
  let latest: Promise<void> = Promise.resolve()
  function readDraft(): Promise<void> {
    const reading = readDraftAs(++asked)
    latest = reading
    return reading
  }

  async function readDraftAs(mine: number) {
    if (!client.drafting) {
      draft.value = undefined
      client.drafted = false
      return
    }
    let now: Drafting | null = null
    try {
      now = await client.draft()
    } catch {
      // read as no draft, which the next change begins
    }
    if (mine === asked) drafted(now)
    else await latest
  }

  // The draft as it now stands, which the pages are read with until it no longer carries.
  function drafted(now: Drafting | null) {
    draft.value = now
    client.drafted = !!now && !now.drifted.length
  }

  // A change the tab is making, from the click to wherever its answer took the page.
  const changing = new Set<Promise<unknown>>()
  function change<T>(work: () => Promise<T>): Promise<T> {
    const doing = work()
    changing.add(doing)
    void doing.catch(() => {}).finally(() => changing.delete(doing))
    return doing
  }

  async function quiet() {
    await client.settled()
    while (changing.size) await Promise.allSettled([...changing])
  }

  // An event waits for the tab's own changes to land, so that a page the change took
  // the tab away from does not ask again for a record that went with it; whatever is
  // read again is read after the draft, which the change may have kept or thrown away.
  async function heard(event: Event) {
    if (event.type === 'moved') await read()
    else if (event.type === 'draft') await readDraft()
    else if (event.type === 'findings')
      void readDraft()
        .then(() => client.findings())
        .then((f) => (findings.value = f))
        .catch(() => {})
    await quiet()
    if (event.type === 'moved' && event.head && !made.has(event.head)) notify(said.moved(event.head.slice(0, 7)))
    for (const listen of listeners) listen(event)
  }

  async function start() {
    try {
      session.value = await client.session()
    } catch (e) {
      if (e instanceof Refused && e.kind === 'login') signIn.value = true
      else failed.value = e instanceof Error ? e.message : String(e)
      return
    }
    apply()
    await read()
    stop = client.events(heard, { connected: (up) => (live.value = up) })
  }

  function setView(hiding: Hiding) {
    view.value = { ...hiding }
    prefs.keepView(view.value)
    apply()
    shown.value++
    void read()
  }

  function setSupposing(on: boolean) {
    supposing.value = on
    prefs.keepSupposing(on)
    apply()
    shown.value++
    void read()
  }

  // The hypotheses the tab reads supposed true, which make it read-only as a view does.
  function setSupposed(slugs: string[]) {
    supposed.value = [...slugs]
    prefs.keepSupposed(supposed.value)
    apply()
    shown.value++
    void read()
  }

  // Whether the tab may change the corpus: the server writes, and the tab shows neither a
  // hidden view nor a supposition, each of which is a copy.
  function writes(): boolean {
    return session.value?.writes === true && !hidden() && !supposed.value.length
  }

  // Hands every event to listen for as long as the calling component lives.
  function listen(to: (event: Event) => void) {
    listeners.add(to)
    onScopeDispose(() => listeners.delete(to))
  }

  return {
    session,
    index,
    findings,
    live,
    signIn,
    failed,
    view,
    supposing,
    supposed,
    draft,
    drafting,
    history,
    offered,
    drafted,
    ready,
    shown,
    toasts,
    undoing,
    hidden,
    start,
    stop: () => stop?.(),
    read,
    setView,
    setSupposing,
    setSupposed,
    writes,
    listen,
    change,
    notify,
    dismiss,
    act,
    // a commit the tab made itself, which its move does not announce
    made: (commit: string) => made.add(commit),
  }
}

export type Corpus = ReturnType<typeof createCorpus>

export const corpusKey: InjectionKey<Corpus> = Symbol('corpus')

export function useCorpus(): Corpus {
  const corpus = inject(corpusKey)
  if (corpus === undefined) throw new Error('the application was mounted without a corpus')
  return corpus
}
