import { describe, expect, it, vi } from 'vitest'
import { Client, Refused, tokenHeader, type Event } from '../api/client'

interface Asked {
  url: string
  method: string
  headers: { [name: string]: string }
  body?: unknown
}

// server answers every request with what answer gives for it, keeping what was asked.
function server(answer: (asked: Asked) => { status?: number; body?: unknown } = () => ({ body: {} })) {
  const asked: Asked[] = []
  const fetch = vi.fn(async (url: string, init: RequestInit) => {
    const one: Asked = {
      url,
      method: init.method ?? 'GET',
      headers: init.headers as { [name: string]: string },
      body: init.body === undefined ? undefined : JSON.parse(String(init.body)),
    }
    asked.push(one)
    const { status = 200, body } = answer(one)
    return new Response(body === undefined ? '' : JSON.stringify(body), { status })
  })
  return { asked, fetch: fetch as unknown as typeof globalThis.fetch }
}

class FakeSocket {
  static opened: FakeSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((m: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  closed = false
  constructor(readonly url: string) {
    FakeSocket.opened.push(this)
  }
  close() {
    this.closed = true
  }
}

const at = { protocol: 'http:', host: '127.0.0.1:8100' }

function client(fetch: typeof globalThis.fetch) {
  return new Client({
    fetch,
    WebSocket: FakeSocket as unknown as typeof WebSocket,
    location: at,
    token: 'the-token',
  })
}

describe('the client', () => {
  it('reads without the token, and with the view and the draft the tab asks for', async () => {
    const { asked, fetch } = server(() => ({ body: { project: 'Rossi', writes: true } }))
    const c = client(fetch)
    expect((await c.session()).project).toBe('Rossi')
    c.view = { private: true, living: true }
    await c.record('person', 'rossi-giovanni')
    c.view = { private: false, living: false }
    c.drafting = true
    await c.log('person/rossi-giovanni', 5)
    c.drafted = true
    await c.log('person/rossi-giovanni', 5)

    expect(asked.map((a) => a.url)).toEqual([
      '/api/session',
      '/api/records/person/rossi-giovanni?view=hide-private&view=hide-living',
      '/api/log?entity=person%2Frossi-giovanni&n=5',
      '/api/log?entity=person%2Frossi-giovanni&n=5&draft=',
    ])
    for (const a of asked) {
      expect(a.method).toBe('GET')
      expect(a.headers[tokenHeader]).toBeUndefined()
    }
  })

  it('reads with the hypotheses the tab supposes, beside the view', async () => {
    const { asked, fetch } = server(() => ({ body: {} }))
    const c = client(fetch)
    c.view = { private: true, living: false }
    c.with = ['unrecorded-child-1845', 'annunziata-born-1844']
    await c.index()

    expect(asked.map((a) => a.url)).toEqual([
      '/api/index?view=hide-private&with=unrecorded-child-1845&with=annunziata-born-1844',
    ])
  })

  it('writes with the token and the change as JSON', async () => {
    const { asked, fetch } = server(() => ({ body: { commit: 'a41c09e', stands: [] } }))
    const c = client(fetch)
    const edit = { op: 'set', entity: 'person/rossi-giovanni', path: 'sex', value: 'M' }
    const made = await c.change({ edit, seen: 'abc' })
    expect(made).toMatchObject({ commit: 'a41c09e' })
    await c.accept('annunziata-born-1844', { dry: true })
    await c.discardDraft()

    expect(asked.map((a) => [a.method, a.url])).toEqual([
      ['POST', '/api/edit'],
      ['POST', '/api/hypotheses/annunziata-born-1844/accept'],
      ['DELETE', '/api/draft'],
    ])
    expect(asked[0]!.headers[tokenHeader]).toBe('the-token')
    expect(asked[0]!.headers['Content-Type']).toBe('application/json')
    expect(asked[0]!.body).toEqual({ edit, seen: 'abc' })
    expect(asked[2]!.headers[tokenHeader]).toBe('the-token')
  })

  it('sends a change to the draft while the tab is supposing', async () => {
    const { asked, fetch } = server(() => ({ body: { base: 'abc', edits: [] } }))
    const c = client(fetch)
    c.drafting = true
    await c.change({ edit: { op: 'unset', entity: 'person/x', path: 'sex' }, seen: 's' })
    c.drafted = true
    await c.change({ edit: { op: 'set', entity: 'person/x', path: 'sex', value: 'M' }, seen: 's' })
    expect(asked.map((a) => a.url)).toEqual(['/api/draft/edit', '/api/draft/edit?draft='])
  })

  it('throws a refusal with its kind', async () => {
    const { fetch } = server(() => ({
      status: 409,
      body: { kind: 'stale', error: 'the record changed', findings: [{ rule: 'r', severity: 'error', entity: 'e', message: 'm' }] },
    }))
    const refusal = await client(fetch).change({ edit: { op: 'set' }, seen: '' }).catch((e: unknown) => e)
    expect(refusal).toBeInstanceOf(Refused)
    expect(refusal).toMatchObject({ status: 409, kind: 'stale', message: 'the record changed' })
    expect((refusal as Refused).findings).toHaveLength(1)
  })

  it('reads an answer that is not JSON as the words of a refusal', async () => {
    const fetch = (async () => new Response('bad gateway', { status: 502 })) as unknown as typeof globalThis.fetch
    const refusal = (await client(fetch).index().catch((e: unknown) => e)) as Refused
    expect(refusal.kind).toBe('unknown')
    expect(refusal.message).toBe('bad gateway')
  })

  it('hands on the events, and opens the socket again when it closes', () => {
    vi.useFakeTimers()
    FakeSocket.opened = []
    const heard: Event[] = []
    const up: boolean[] = []
    const stop = client(server().fetch).events((e) => heard.push(e), { pause: 100, connected: (u) => up.push(u) })

    const first = FakeSocket.opened[0]!
    expect(first.url).toBe('ws://127.0.0.1:8100/api/events')
    first.onopen!()
    first.onmessage!({ data: JSON.stringify({ type: 'moved', head: 'abc' }) })
    first.onmessage!({ data: 'not an event' })
    expect(heard).toEqual([{ type: 'moved', head: 'abc' }])

    first.onclose!()
    vi.advanceTimersByTime(100)
    const second = FakeSocket.opened[1]!
    second.onclose!()
    vi.advanceTimersByTime(100)
    const third = FakeSocket.opened[2]!
    third.onopen!()
    expect(heard).toEqual([{ type: 'moved', head: 'abc' }, { type: 'moved' }])
    expect(up).toEqual([true, false, true])

    stop()
    expect(third.closed).toBe(true)
    third.onclose!()
    vi.advanceTimersByTime(1000)
    expect(FakeSocket.opened).toHaveLength(3)
    expect(up).toEqual([true, false, true])
    vi.useRealTimers()
  })
})
