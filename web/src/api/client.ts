// The one place the application calls the server. Every request carries what the tab
// is looking at — the hidden view switched on, the hypotheses it reads supposed and
// whether it is supposing, the draft —
// and every write the token the server put into the page, so that no page builds a
// URL or a header of its own.

export type Kind = 'person' | 'family' | 'place' | 'source' | 'repository' | 'note' | 'hypothesis'

export const kindsInOrder: Kind[] = [
  'person',
  'family',
  'place',
  'source',
  'repository',
  'note',
  'hypothesis',
]

// The meta the server puts the token in, and the header a write carries it back in.
export const tokenMeta = 'origins-token'
export const tokenHeader = 'X-Origins-Token'

// What a view leaves out.
export interface Hiding {
  private: boolean
  living: boolean
}

// How a hypothesis served with --with was read.
export interface Supposition {
  hypothesis: string
  base?: string
  why?: string
}

// What an export was made from: when, by the clock of the machine that made it, the
// commit the corpus stood on, empty where it had no history, whether its files differed
// from that commit, the origins that made it, and the notice every page shows.
export interface ExportedFrom {
  at: string
  commit: string
  changed: boolean
  version: string
  notice: string
}

// What the application asks first: what it is serving, whether it may change it,
// whether a server is behind it, telling it that the corpus moved and answering what
// was not written down beforehand, whether it may ask of the history, and what an
// export was made from.
export interface Session {
  project: string
  writes: boolean
  live: boolean
  history: boolean
  view: Hiding | null
  with: Supposition[]
  problems: string[]
  export: ExportedFrom | null
}

// One path into a record and the operations that apply to it.
export interface Editable {
  path: string
  ops: string[]
}

// A record as the API serves it, its fields spelled as the file spells them, with each
// piece of its prose rendered under its path and where each of its events goes in
// order.
export interface Record {
  ref: string
  fields: { [field: string]: unknown }
  seen: string
  file: string
  attached: string[]
  edits: Editable[]
  sorted: { [path: string]: string }
  html: { [path: string]: string }
  // the manifest as origins writes it, for the project alone
  written?: string
}

// A record and every record it refers to or that refers to it, one step out.
export interface Page {
  record: Record
  around: Record[]
}

// The derived graph the lists and charts are drawn from, a list for each kind.
export interface Indexed {
  ref: string
  [field: string]: unknown
}

export interface Index {
  project: { [field: string]: unknown }
  people: Indexed[]
  families: Indexed[]
  places: Indexed[]
  sources: Indexed[]
  repositories: Indexed[]
  notes: Indexed[]
  hypotheses: Indexed[]
  read?: Supposition[]
  problems: string[]
}

export interface Finding {
  rule: string
  severity: string
  entity: string
  path?: string
  message: string
}

// What check finds of the corpus, and research's tasks beside it.
export interface Findings {
  findings: Finding[]
  todo: Finding[]
  errors: number
  warnings: number
  problems: string[]
}

// One change in the history, as `origins log` reads it back.
export interface Logged {
  commit: string
  date: string
  author: string
  subject: string
  op: string
  subjects: string[]
  hypothesis: string
  'proposed-by': string
}

export interface Log {
  entries: Logged[]
  problems: string[]
}

// A date as the server reads it: as the corpus writes it, whether it has a place on
// the calendar, and where a list puts it.
export interface DateRead {
  date: string
  anchored: boolean
  sort?: string
}

// One edit, as a hypothesis writes it: op, entity, path, value and the rest.
export type Edit = { op: string; [field: string]: unknown }

// A change asked for: the edit, the seen the page was shown, the citations it rests
// on, the subject line in place of the one origins writes, and whether only to say
// what it comes to.
export interface Change {
  edit: Edit
  seen: string
  sources?: string[]
  message?: string
  dry?: boolean
}

// Taking a change back, or settling a hypothesis.
export interface Settling {
  seen?: string
  sources?: string[]
  message?: string
  dry?: boolean
}

// What came of a change, and the records that now stand where it was about.
export interface Made {
  op: string
  subject: string
  commit: string
  written: string[]
  moved: { from: string; to: string }[]
  removed: string[]
  empty: boolean
  stands: string[]
  problems: string[]
}

// A draft kept as a hypothesis: its slug, the commit it is pinned to and the one that
// recorded it, and its edits in words.
export interface Proposal {
  hypothesis: string
  base: string
  commit: string
  subject: string
  edits: string[]
  restored: string[]
}

// The draft as a page is shown it.
export interface Drafting {
  base: string
  since: number
  edits: Edit[]
  stated: string[]
  records: string[]
  drifted: string[]
}

// Keeping the draft as a hypothesis.
export interface Kept {
  slug: string
  question: string
  argument?: string
  confidence?: string
  supports?: string[]
  contradicts?: string[]
  message?: string
  dry?: boolean
}

export interface Remote {
  name: string
  url: string
}

// The remotes and where the branch stands against the one it tracks, as last fetched.
export interface Remotes {
  remotes: Remote[]
  branch: string
  upstream: string
  ahead: number
  behind: number
}

// What a fetch brought: the remotes it reached and the branch standing after it.
export interface Standing {
  fetched: string[]
  branch: string
  upstream: string
  ahead: number
  behind: number
}

// What a pull took in, from where and how: nothing, fast-forward or merge.
export interface Taken {
  remote: string
  upstream: string
  took: string
  commits: number
  commit: string
  subjects: string[]
}

// What a push sent where.
export interface Sent {
  remote: string
  branch: string
  commit: string
  upstream: boolean
}

// Where origins stands with the history, the way it advises and what a way taken made.
export interface Resolution {
  settled: boolean
  ground: string
  anchor: string
  foreign: Logged[] | null
  dirty: string[] | null
  shared: boolean
  advises: string
  commit: string
  problems: string[] | null
}

// What the server sends a page: the corpus moved to head, the findings changed, or
// the draft did. A page asks again for what it shows.
export interface Event {
  type: 'moved' | 'findings' | 'draft'
  head?: string
}

// A refusal, by its kind, so that a page decides what to show by the kind and never by
// the words: stale, contradiction, lock, drifted, unreadable, refused, unreached,
// forbidden, login and the rest the server names.
export class Refused extends Error {
  readonly status: number
  readonly kind: string
  readonly problems: string[]
  readonly findings: Finding[]

  constructor(status: number, body: { kind?: string; error?: string; problems?: string[]; findings?: Finding[] }) {
    super(body.error ?? `the server answered ${status}`)
    this.name = 'Refused'
    this.status = status
    this.kind = body.kind ?? 'unknown'
    this.problems = body.problems ?? []
    this.findings = body.findings ?? []
  }
}

// The meta an export's page holds, whose answers are scripts written beside it, each
// handing its answer to the function named here.
export const exportMeta = 'origins-export'
export const answerFunction = 'originsAnswer'

// Whether the page is an export's, opened from a folder or wherever it was put, with no
// server behind it.
export const exported: boolean =
  typeof document !== 'undefined' && document.querySelector(`meta[name="${exportMeta}"]`) !== null

// Where a page goes for a path the server would answer at: the path itself, or in an
// export the file written for it beside the page and a route after the hash, the page
// staying where it was opened.
export function address(path: string): string {
  if (!exported || !path.startsWith('/')) return path
  return path.startsWith('/files/') ? path.slice(1) : `#${path}`
}

// The answers an export wrote, each loaded once as the script holding it.
const answers = new Map<string, Promise<unknown>>()
const waiting = new Map<string, (answer: unknown) => void>()

if (exported) {
  ;(globalThis as unknown as { [answerFunction]: (path: string, answer: unknown) => void })[answerFunction] = (
    path,
    answer,
  ) => {
    waiting.get(path)?.(answer)
    waiting.delete(path)
  }
}

// The answer an export wrote for a path, read from the script beside the page; one it
// did not write is refused as missing, which is what a server would have said of it.
export function written(path: string): Promise<unknown> {
  let answer = answers.get(path)
  if (!answer) {
    answer = new Promise((resolve, reject) => {
      const missing = () => {
        waiting.delete(path)
        answers.delete(path)
        reject(new Refused(404, { kind: 'missing', error: `${path} is not in this export` }))
      }
      waiting.set(path, resolve)
      const script = document.createElement('script')
      script.src = `data${path}.js`
      script.onerror = missing
      script.onload = () => waiting.has(path) && missing()
      document.head.append(script)
    })
    answers.set(path, answer)
  }
  return answer.then((one) => structuredClone(one))
}

// What the client needs of the browser, given so that a test can give something else.
export interface Surroundings {
  fetch: typeof globalThis.fetch
  WebSocket: typeof globalThis.WebSocket
  location: { protocol: string; host: string }
  token: string
  // the answers an export wrote, where the page is one
  written?: (path: string) => Promise<unknown>
}

function browser(): Surroundings {
  return {
    fetch: globalThis.fetch.bind(globalThis),
    WebSocket: globalThis.WebSocket,
    location: globalThis.location,
    token: document.querySelector<HTMLMetaElement>(`meta[name="${tokenMeta}"]`)?.content ?? '',
    written: exported ? written : undefined,
  }
}

type Query = { [name: string]: string | number | undefined }

export class Client {
  // the hidden view the tab has switched on
  view: Hiding = { private: false, living: false }
  // the hypotheses the tab reads supposed true
  with: string[] = []
  // whether the tab is supposing, reading and writing the draft
  drafting = false
  // whether there is a draft that carries to read with, the corpus read as it is where not
  drafted = false

  private readonly around: Surroundings
  private readonly writing = new Set<Promise<unknown>>()

  constructor(around: Partial<Surroundings> = {}) {
    this.around = { ...browser(), ...around }
  }

  // The URL of a request, with the view and the draft the tab asks for.
  url(path: string, query: Query = {}): string {
    const params = new URLSearchParams()
    for (const [name, value] of Object.entries(query)) {
      if (value !== undefined && value !== '') params.append(name, String(value))
    }
    if (this.view.private) params.append('view', 'hide-private')
    if (this.view.living) params.append('view', 'hide-living')
    for (const slug of this.with) params.append('with', slug)
    if (this.drafting && this.drafted) params.append('draft', '')
    const search = params.toString()
    return search === '' ? path : `${path}?${search}`
  }

  private request<T>(method: string, path: string, body?: unknown, query?: Query): Promise<T> {
    const asked = this.ask<T>(method, path, body, query)
    if (method === 'GET') return asked
    this.writing.add(asked)
    const done = () => void this.writing.delete(asked)
    asked.then(done, done)
    return asked
  }

  // Resolves once every write the tab has asked for is answered, so that a move the
  // socket tells of can be told apart from one the tab made before its answer arrived.
  async settled(): Promise<void> {
    while (this.writing.size) await Promise.allSettled([...this.writing])
  }

  private async ask<T>(method: string, path: string, body?: unknown, query?: Query): Promise<T> {
    if (this.around.written) {
      if (method !== 'GET') throw new Refused(405, { kind: 'refused', error: 'an export takes no change' })
      return (await this.around.written(path)) as T
    }
    const headers: { [name: string]: string } = { Accept: 'application/json' }
    if (method !== 'GET') headers[tokenHeader] = this.around.token
    if (body !== undefined) headers['Content-Type'] = 'application/json'
    const response = await this.around.fetch(this.url(path, query), {
      method,
      headers,
      credentials: 'same-origin',
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    const text = await response.text()
    let answer: unknown
    try {
      answer = text === '' ? undefined : JSON.parse(text)
    } catch {
      answer = { error: text }
    }
    if (!response.ok) throw new Refused(response.status, (answer ?? {}) as object)
    return answer as T
  }

  get<T>(path: string, query?: Query): Promise<T> {
    return this.request<T>('GET', path, undefined, query)
  }

  post<T>(path: string, body: unknown = {}): Promise<T> {
    return this.request<T>('POST', path, body)
  }

  delete<T>(path: string): Promise<T> {
    return this.request<T>('DELETE', path)
  }

  // the reads

  session = () => this.get<Session>('/api/session')
  index = () => this.get<Index>('/api/index')
  record = (kind: Kind, slug: string) =>
    this.get<Page>(`/api/records/${kind}/${encodeURIComponent(slug)}`)
  project = () => this.get<Page>('/api/records/project')
  findings = () => this.get<Findings>('/api/findings')
  log = (entity?: string, n?: number) => this.get<Log>('/api/log', { entity, n })
  schema = (name?: string) => this.get<unknown>(name ? `/api/schema/${name}` : '/api/schema')
  date = (text: string) => this.get<DateRead>('/api/date', { text })

  // the writes: a change goes to the draft while the tab is supposing

  change = (change: Change) =>
    this.post<Made | Drafting>(this.drafting ? '/api/draft/edit' : '/api/edit', change)
  undo = (settling: Settling = {}) => this.post<Made>('/api/undo', settling)
  accept = (slug: string, settling: Settling = {}) =>
    this.post<Made>(`/api/hypotheses/${encodeURIComponent(slug)}/accept`, settling)
  reject = (slug: string, settling: Settling = {}) =>
    this.post<Made>(`/api/hypotheses/${encodeURIComponent(slug)}/reject`, settling)
  render = (entity: string, prose: string) =>
    this.post<{ html: string }>('/api/render', { entity, prose })

  // the draft

  draft = () => this.get<Drafting | null>('/api/draft')
  popDraft = () => this.post<Drafting>('/api/draft/pop')
  keepDraft = (kept: Kept) => this.post<Proposal>('/api/draft/keep', kept)
  discardDraft = () => this.delete<unknown>('/api/draft')

  // git

  remotes = () => this.get<Remotes>('/api/remotes')
  addRemote = (remote: Remote) => this.post<unknown>('/api/remotes', remote)
  removeRemote = (name: string) => this.delete<unknown>(`/api/remotes/${encodeURIComponent(name)}`)
  fetchRemote = (remote = '') => this.post<Standing>('/api/fetch', { remote })
  pull = (remote = '') => this.post<Taken>('/api/pull', { remote })
  push = (remote = '') => this.post<Sent>('/api/push', { remote })
  recover = (way = '') => this.post<Resolution>('/api/recover', { way })

  login = (passphrase: string) => this.post<unknown>('/api/login', { passphrase })

  // The events, handed to listen until the returned function is called, and to
  // connected whether the socket is open each time that changes. A socket that closes
  // is opened again after a pause, the page asking again for what it shows as it would
  // after a move, since whatever happened meanwhile went untold.
  events(
    listen: (event: Event) => void,
    { pause = 2000, connected = () => {} }: { pause?: number; connected?: (up: boolean) => void } = {},
  ): () => void {
    if (this.around.written) return () => {}
    const scheme = this.around.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const address = `${scheme}//${this.around.location.host}/api/events`
    let socket: WebSocket | undefined
    let stopped = false
    let timer: ReturnType<typeof setTimeout> | undefined
    let opened = false
    let up: boolean | undefined
    const open = () => {
      socket = new this.around.WebSocket(address)
      socket.onopen = () => {
        up = true
        connected(true)
        if (opened) listen({ type: 'moved' })
        opened = true
      }
      socket.onmessage = (message) => {
        let event: Event
        try {
          event = JSON.parse(String(message.data)) as Event
        } catch {
          return
        }
        listen(event)
      }
      socket.onclose = () => {
        if (stopped) return
        if (up !== false) connected(false)
        up = false
        timer = setTimeout(open, pause)
      }
    }
    open()
    return () => {
      stopped = true
      clearTimeout(timer)
      socket?.close()
    }
  }
}
