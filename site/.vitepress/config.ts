import { execFileSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { copyFile } from 'node:fs/promises'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitepress'
import type MarkdownIt from 'markdown-it'

const repository = 'https://github.com/origins-project/origins'

// The icons and the social preview are the application's, kept once in web/public.
const icons = ['favicon.svg', 'favicon.ico', 'apple-touch-icon.png', 'social.png']
const webPublic = fileURLToPath(new URL('../../web/public/', import.meta.url))
const repo = fileURLToPath(new URL('../../', import.meta.url))
const notice =
  'This is a read-only demo of Origins. All of its data is generated. Any resemblance to real people, living or dead, is coincidental and not intended.'

// A block marked `corpus` is a file of a corpus, its path after the word, which the
// Go tests read; it is shown under its path and highlighted as the file it is.
function corpusFiles(md: MarkdownIt) {
  const fence = md.renderer.rules.fence!
  md.renderer.rules.fence = (tokens, idx, options, env, self) => {
    const token = tokens[idx]
    const [lang, path] = token.info.trim().split(/\s+/)
    if (lang !== 'corpus' || path === undefined) return fence(tokens, idx, options, env, self)
    token.info = path.endsWith('.yaml') ? 'yaml' : 'md'
    const shown = md.utils.escapeHtml(path)
    return `<div class="corpus-file"><div class="corpus-path">${shown}</div>${fence(tokens, idx, options, env, self)}</div>`
  }
}

export default defineConfig({
  title: 'Origins',
  description: 'Genealogy software that keeps a family tree as text files in a git repository.',
  lang: 'en',
  base: '/origins/',
  cleanUrls: true,
  // the demo is the pages origins export html writes, added when the site is built
  ignoreDeadLinks: [/^\/demo\//],
  head: [
    ['link', { rel: 'icon', href: '/origins/favicon.svg', type: 'image/svg+xml' }],
    ['link', { rel: 'icon', href: '/origins/favicon.ico', sizes: '32x32' }],
    ['link', { rel: 'apple-touch-icon', href: '/origins/apple-touch-icon.png' }],
    ['meta', { property: 'og:image', content: 'https://origins-project.github.io/origins/social.png' }],
    // The tokens choose their theme by data-theme, and VitePress by a class: this sets
    // the one from the other before the page is drawn, and the theme keeps them together.
    [
      'script',
      {},
      `(() => {
        const chosen = localStorage.getItem('vitepress-theme-appearance') || 'auto'
        const dark = chosen === 'dark' || (chosen === 'auto' && matchMedia('(prefers-color-scheme: dark)').matches)
        document.documentElement.dataset.theme = dark ? 'dark' : 'light'
      })()`,
    ],
  ],
  markdown: {
    config: corpusFiles,
  },
  async buildEnd(site) {
    for (const icon of icons) await copyFile(join(webPublic, icon), join(site.outDir, icon))
    // The demo is the git fixture whole, its people invented, written here because
    // VitePress empties its output directory before it builds.
    if (!existsSync(join(repo, 'testgit'))) throw new Error('the demo needs testgit/: run tools/mkgitfixture.sh')
    const out = join(site.outDir, 'demo')
    execFileSync('go', ['run', './cmd/origins', 'export', 'html', '--notice', notice, '-C', 'testgit', '--out', out], {
      cwd: repo,
      stdio: 'inherit',
    })
  },
  vite: {
    // the theme reads the application's tokens and its lockup from web/, whose own
    // imports are found in web/node_modules
    server: { fs: { allow: ['..'] } },
    resolve: { dedupe: ['vue'] },
  },
  themeConfig: {
    siteTitle: false,
    nav: [
      { text: 'Install', link: '/install' },
      { text: 'Getting started', link: '/getting-started' },
      { text: 'Guides', link: '/guides/browse' },
      { text: 'Format', link: '/format/' },
      { text: 'Reference', link: '/reference/commands' },
      { text: 'Demo', link: '/demo/index.html', target: '_self' },
    ],
    sidebar: [
      {
        text: 'Introduction',
        items: [
          { text: 'Install', link: '/install' },
          { text: 'Getting started', link: '/getting-started' },
        ],
      },
      {
        text: 'Guides',
        items: [
          { text: 'Working in the browser', link: '/guides/browse' },
          { text: 'Importing from GEDCOM', link: '/guides/import-gedcom' },
          { text: 'Importing from Gramps', link: '/guides/import-gramps' },
          { text: 'Hypotheses', link: '/guides/hypotheses' },
          { text: 'History', link: '/guides/history' },
          { text: 'Sharing a corpus', link: '/guides/sharing' },
          { text: 'Privacy', link: '/guides/privacy' },
          { text: 'Exporting', link: '/guides/exporting' },
          { text: 'Running on a server', link: '/guides/docker' },
          { text: 'AI agents', link: '/guides/agents' },
        ],
      },
      {
        text: 'Format',
        items: [
          { text: 'The corpus', link: '/format/' },
          { text: 'People', link: '/format/person' },
          { text: 'Events', link: '/format/events' },
          { text: 'Families', link: '/format/family' },
          { text: 'Places', link: '/format/place' },
          { text: 'Sources and citations', link: '/format/source' },
          { text: 'Repositories', link: '/format/repository' },
          { text: 'Notes', link: '/format/note' },
          { text: 'Pictures and files', link: '/format/media' },
          { text: 'Dates', link: '/format/dates' },
          { text: 'Private records', link: '/format/private' },
          { text: 'The manifest', link: '/format/project' },
          { text: 'Hypotheses', link: '/format/hypothesis' },
        ],
      },
      {
        text: 'Reference',
        items: [
          { text: 'Commands', link: '/reference/commands' },
          { text: 'Findings', link: '/reference/check' },
          { text: 'JSON Schema', link: '/reference/schema' },
        ],
      },
    ],
    socialLinks: [{ icon: 'github', link: repository }],
    search: { provider: 'local' },
    outline: 'deep',
    footer: {
      message: 'Origins is licensed under the Apache License 2.0.',
    },
  },
})
