import { fileURLToPath, URL } from 'node:url'

import { defineConfig, type Plugin, type ProxyOptions } from 'vite'
import vue from '@vitejs/plugin-vue'

// The server `npm run dev` hands /api and /files to: an `origins browse` already
// running, on the loopback, at its default address unless ORIGINS_URL says otherwise.
const served = process.env.ORIGINS_URL ?? 'http://127.0.0.1:8100'

// The server takes a write only from one of its own pages, with the token it put into
// that page. In development the page is Vite's, so the proxy says the request came from
// the server's own address, and the page is given the token the server's page holds.
const proxy: ProxyOptions = {
  target: served,
  changeOrigin: true,
  ws: true,
  configure: (server) => {
    server.on('proxyReq', (request) => {
      if (request.getHeader('origin')) request.setHeader('origin', served)
    })
    server.on('proxyReqWs', (request) => {
      if (request.getHeader('origin')) request.setHeader('origin', served)
    })
  },
}

function token(): Plugin {
  return {
    name: 'origins-token',
    apply: 'serve',
    async transformIndexHtml() {
      const page = await fetch(served).then((r) => r.text()).catch(() => '')
      const meta = page.match(/<meta name="origins-token"[^>]*>/)
      return meta ? [{ tag: 'meta', attrs: attributes(meta[0]), injectTo: 'head' }] : []
    },
  }
}

// A page opened from a folder runs no module script, Chrome refusing one whose origin
// is a file, and fetches nothing with CORS: so the build is one classic script, run
// once the page is parsed, as a module would have been, and named for the files it
// asks for to be found from where it is.
function classic(): Plugin {
  return {
    name: 'origins-classic',
    apply: 'build',
    transformIndexHtml: {
      order: 'post',
      handler: (html) =>
        html
          .replace(/ crossorigin/g, '')
          .replace(/<script type="module" src="([^"]*)"><\/script>/, '<script defer id="origins" src="$1"></script>'),
    },
  }
}

function attributes(tag: string): Record<string, string> {
  return Object.fromEntries([...tag.matchAll(/(\w[\w-]*)="([^"]*)"/g)].map((m) => [m[1], m[2]]))
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue(), token(), classic()],
  // relative, so that the page finds its files opened from a folder; the server names
  // them from the root
  base: './',
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    // embedded in the binary by internal/browse
    outDir: fileURLToPath(new URL('../internal/browse/app/dist', import.meta.url)),
    emptyOutDir: true,
    rolldownOptions: {
      output: { format: 'iife', codeSplitting: false },
    },
    // one stylesheet beside the script, whose fonts are found from where it is
    cssCodeSplit: false,
    // the licence of every package bundled, for tools/notices; go:embed leaves out a
    // directory whose name starts with a dot, so the binary does not carry it
    license: { fileName: '.vite/license.json' },
  },
  experimental: {
    // a file the script asks for is beside it, in the folder it was opened from or at
    // the root the server serves it from, whatever route the page is at; a classic
    // script run after the page was parsed knows no address of its own, so it is found
    // by its name
    renderBuiltUrl: (file, { hostType }) =>
      hostType === 'js'
        ? { runtime: `new URL(${JSON.stringify('../' + file)}, document.getElementById('origins').src).href` }
        : { relative: true },
  },
  server: {
    proxy: { '/api': proxy, '/files': proxy },
  },
})
