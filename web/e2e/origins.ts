import { execFileSync, spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { resolve } from 'node:path'
import type { Page } from '@playwright/test'

// The origins a suite serving a corpus of its own starts, built once before any suite
// runs, since a build over a binary another suite's server is running fails on Windows.
export const binary = resolve('..', 'corpora', 'bin', process.platform === 'win32' ? 'origins.exe' : 'origins')

// The corpora every suite only reads, each served once for the whole run: the large one
// takes seconds to read, longer than a timeout allows when every suite reads it at
// once. A suite that writes serves a copy of its own.
export const shared = {
  fixture: { root: resolve('..', 'testgit'), port: 8217 },
  large: { root: resolve('..', 'corpora', 'large'), port: 8218 },
}

// Waits for the origins on port to answer.
export async function served(port: number) {
  for (let i = 0; i < 1200; i++) {
    try {
      if ((await fetch(`http://127.0.0.1:${port}/api/session`)).ok) return
    } catch {
      // not listening yet
    }
    await new Promise((r) => setTimeout(r, 100))
  }
  throw new Error(`nothing answered on ${port}`)
}

// The token the origins on port puts into its page, which a request other than a read
// carries, as a preview asked of the server is.
export async function tokenOf(port: number): Promise<string> {
  const page = await (await fetch(`http://127.0.0.1:${port}/`)).text()
  return page.match(/name="origins-token" content="([^"]+)"/)?.[1] ?? ''
}

// Hands every read of the page, every file it shows and the socket it listens on to
// the origins on port, a request other than a read carrying the origin and the token
// that server's own page would send. The token is asked for each time, since a suite
// may start its server again.
export async function handTo(page: Page, port: number) {
  const host = `127.0.0.1:${port}`
  await page.route((url) => /^\/(api|files)\//.test(url.pathname), async (route) => {
    const url = new URL(route.request().url())
    url.host = host
    const headers = { ...route.request().headers(), host }
    if (route.request().method() !== 'GET') Object.assign(headers, { origin: `http://${host}`, 'x-origins-token': await tokenOf(port) })
    await route.fulfill({ response: await route.fetch({ url: url.toString(), headers }) })
  })
  await page.routeWebSocket(/\/api\/events$/, (socket) => {
    const upstream = new WebSocket(`ws://${host}/api/events`)
    upstream.onmessage = (message) => socket.send(String(message.data))
    socket.onClose(() => upstream.close())
  })
}

export default function setup() {
  execFileSync('go', ['build', '-o', binary, '../cmd/origins'], { stdio: 'inherit' })
  const started = Object.values(shared)
    .filter(({ root }) => existsSync(root))
    .map(({ root, port }) => spawn(binary, ['browse', '--no-open', '--listen', `127.0.0.1:${port}`, root], { stdio: 'ignore' }))
  return async () => {
    await Promise.all(
      started.map((server) => {
        if (server.exitCode !== null) return
        const exited = new Promise((r) => server.once('exit', r))
        server.kill()
        return exited
      }),
    )
  }
}
