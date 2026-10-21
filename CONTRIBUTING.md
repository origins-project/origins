# Contributing to Origins

## Contents

- [1. Before you start](#1-before-you-start)
- [2. Building](#2-building)
- [3. Test data](#3-test-data)
- [4. Tests](#4-tests)
- [5. Commits](#5-commits)
- [6. AI coding tools](#6-ai-coding-tools)

## 1. Before you start

Please open an issue before you start on a large change, so that we can agree on the
approach first. A change to the file format needs particular care, because every
corpus already written depends on it.

Report security problems privately, as [SECURITY.md](SECURITY.md) describes, and not
in a public issue.

## 2. Building

You need Go 1.27.3 or later, git, and Node.js 22.18, or 24.12 or later. The web
application in `web/` is written in Vue 3 and TypeScript. Vite builds it into
`internal/browse/app/dist`, and the Go build embeds it:

```bash
npm --prefix web ci
npm --prefix web run build
go build ./cmd/origins
```

If you build without Node, everything else works, and `origins browse` serves a
placeholder page instead of the application.

## 3. Test data

The tests use two fixtures, which are generated and not committed:

- `test/` is a corpus of 46 people across nine generations, chosen to cover unusual
  cases. `tools/mkfixture.sh` writes it. To change the fixture, edit the script and run
  it again, and don't edit the files it writes.
- `testgit/` is the same corpus as a git repository, built one `origins` command at a
  time so that it has a realistic history. `tools/mkgitfixture.sh` builds it. It runs
  `mkfixture.sh` first and takes a few minutes.

```bash
tools/mkgitfixture.sh
```

The tests that need a fixture skip when it isn't there. In CI, the fixtures are always
built, so no test skips.

The browser tests in `web/e2e/` use three larger corpora, which
`tools/mkcorpora.go` writes into `corpora/`. The largest has 5,000 people.

## 4. Tests

Run the Go tests:

```bash
go build ./... && go vet ./... && go test ./...
```

Run the checks of the web application: the type check, ESLint, the generated types and
Vitest:

```bash
npm --prefix web run check
```

Run the browser tests with Playwright. If Playwright's own Chromium isn't installed,
set `ORIGINS_CHROMIUM` to the path of a Chromium that is:

```bash
go run ./tools/mkcorpora.go corpora && npm --prefix web run test:e2e
```

CI runs all of these, and `gofmt -l`, on Linux, Windows and macOS for every push and
pull request. On Linux, it also runs staticcheck and govulncheck and builds the Docker
image.

Files use LF line endings on every platform, and `.gitattributes` enforces them. The
files that Origins writes must stay byte for byte the same when Origins writes them
again, so a test fails if a change to the writer alters a file of the fixture.

## 5. Commits

A commit message starts with a line of the form `type(context): subject`, of at most
72 characters. The type is one of `feat`, `fix`, `docs`, `refactor`, `test`, `build`
or `chore`. The context is the package or the area the change is in, such as `model`,
`store`, `check`, `cli`, `browse` or `web`. After a blank line, list the reasons for
the change as `*` bullets, wrapped at 72 columns. For example:

```
fix(check): accept a burial on the day of the death

* A burial was flagged as before the death when both had the same day,
  because the comparison took the burial's span as ending at midnight.
```

A small change needs only the subject line. The message describes the code and its
behaviour. It doesn't refer to plans, conversations or earlier attempts at the change.

The notes of a release list the subjects of its `feat` and `fix` commits, so write the
subject for somebody reading those notes. When a pull request is merged, GitHub
squashes it into one commit and uses the pull request's title as the commit's subject.
For that reason the title follows the same form, with the context in lowercase. CI
checks the title of every pull request and the subject of every commit pushed to
`master`, and a pull request whose title doesn't pass can't be merged.

## 6. AI coding tools

Origins was written with the help of AI coding tools. The author directed the work and
reviewed every change.

You can use AI coding tools for a contribution. If you do, please say so in the pull
request. You are responsible for the change as if you had written it yourself: read
every line, make sure you understand it, and run the tests before you open the pull
request. A pull request made with an AI tool is reviewed in the same way as any other,
and the same rules apply to it.
