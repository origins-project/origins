// Writes src/wasm/chart.wasm, internal/chart built as WebAssembly by cmd/chartwasm, and
// beside it the wasm_exec.js of the Go that built it, which runs it in the page.

import { execFileSync } from 'node:child_process'
import { readFileSync, rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('../..', import.meta.url))
const target = fileURLToPath(new URL('../src/wasm/', import.meta.url))

execFileSync('go', ['build', '-trimpath', '-ldflags=-s -w', '-o', join(target, 'chart.wasm'), './cmd/chartwasm'], {
  cwd: root,
  env: { ...process.env, GOOS: 'js', GOARCH: 'wasm' },
  stdio: 'inherit',
})
const goroot = execFileSync('go', ['env', 'GOROOT'], { cwd: root, encoding: 'utf8' }).trim()
// Written rather than copied, which would carry over the read-only mode a toolchain
// in the module cache has and refuse the next run.
const exec = join(target, 'wasm_exec.js')
rmSync(exec, { force: true })
writeFileSync(exec, readFileSync(join(goroot, 'lib', 'wasm', 'wasm_exec.js')))
