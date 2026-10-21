import { resolve } from 'node:path'
import process from 'node:process'
import { defineConfig, devices } from '@playwright/test'

// The suites drive the development server, which is the only one with /ui, in front of
// a real `origins browse`. The served corpus is the sparse one tools/mkcorpora.go
// writes; a suite needing another starts its own server on it.
//
// A change made from a page is a commit, and git started from a server while several
// browsers are busy beside it takes seconds on a system that scans every file it opens,
// so an expectation waits longer than Playwright's own default, and no more browsers are
// run at once than such a system bears.
//
// ORIGINS_CHROMIUM names a Chromium to drive in place of the one Playwright installs,
// for a machine that has one and may not download another.
const origins = 'http://127.0.0.1:8199'
const app = 'http://127.0.0.1:5199'

// A change made from a page is a commit authored as whoever git is configured as, and a
// runner configures nobody. Every git the run starts, through a server or a suite, reads
// no configuration of the machine's and commits as the one researcher, so the run is the
// same on every machine.
Object.assign(process.env, {
  GIT_CONFIG_GLOBAL: resolve('..', 'corpora', 'no-such-global-config'),
  GIT_CONFIG_NOSYSTEM: '1',
  GIT_AUTHOR_NAME: 'Origins',
  GIT_AUTHOR_EMAIL: 'origins@example.org',
  GIT_COMMITTER_NAME: 'Origins',
  GIT_COMMITTER_EMAIL: 'origins@example.org',
})

export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/origins.ts',
  timeout: 30 * 1000,
  expect: { timeout: 15 * 1000 },
  forbidOnly: !!process.env.CI,
  workers: process.env.CI ? 1 : 6,
  reporter: process.env.CI ? 'list' : 'html',
  // the ten slowest files, after the rest, so that a run says where its time went
  reportSlowTests: { max: 10, threshold: 1 },
  use: {
    baseURL: app,
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: process.env.ORIGINS_CHROMIUM ? { executablePath: process.env.ORIGINS_CHROMIUM } : {},
      },
    },
  ],
  webServer: [
    {
      command: 'go run ../tools/mkcorpora.go ../corpora && go run ../cmd/origins browse --no-open --listen 127.0.0.1:8199 ../corpora/sparse',
      url: `${origins}/api/session`,
      timeout: 10 * 60 * 1000,
      reuseExistingServer: !process.env.CI,
    },
    {
      command: 'npx vite --host 127.0.0.1 --port 5199 --strictPort',
      url: app,
      env: { ORIGINS_URL: origins },
      reuseExistingServer: !process.env.CI,
    },
  ],
})
