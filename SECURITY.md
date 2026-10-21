# Security policy

## Contents

- [1. Reporting a vulnerability](#1-reporting-a-vulnerability)
- [2. Supported versions](#2-supported-versions)
- [3. Serving a corpus with origins browse](#3-serving-a-corpus-with-origins-browse)
- [4. Private data](#4-private-data)

## 1. Reporting a vulnerability

Please don't report a vulnerability in a public issue. Report it privately through
GitHub: open the repository's Security tab and choose "Report a vulnerability". Include
the version or commit you tested, the steps to reproduce the problem, and what an
attacker could do with it.

## 2. Supported versions

Origins is alpha software. A fix goes to the `master` branch and into the next
release, and earlier releases don't get fixes of their own.

## 3. Serving a corpus with origins browse

`origins browse` is meant for one researcher. It has no user accounts and no
permissions per record: whoever can use the server can read and change the whole
corpus.

By default, `origins browse` listens on `127.0.0.1`, so other computers can't reach
it. Other web pages open in the same browser can still send requests to it, so the
server checks every request:

- The `Host` header must be the address the server listens on. This stops a page from
  reaching the server through DNS rebinding.
- A request that changes something must carry the server's own `Origin` and a random
  token that only the server's own page holds.

With `--listen`, the server listens on another address, and other computers can reach
it. Then the server doesn't start without a passphrase, and it shows nothing until the
browser gives the passphrase. The server accepts one try per second. After a correct
passphrase, the browser gets a session cookie, marked `HttpOnly` and
`SameSite=Strict`. A session lasts until the server stops.

`origins browse` doesn't serve HTTPS itself. Off the loopback, put it behind a reverse
proxy that serves HTTPS, as [docker/README.md](docker/README.md) shows. Without HTTPS,
the passphrase and the session cookie cross the network in plain text.

## 4. Private data

A corpus can mark records and facts as private, and Origins can work out who is
probably still living. Origins hides them only when you ask it to, with
`--hide-private`, `--hide-living` or `--public`. These flags work with
`origins browse` and `origins export`. When you serve a corpus with one of them, the
server is read-only.

Everything else gives the whole corpus, private data included:

- `origins browse` without those flags.
- `origins push`, and any copy of the git repository.
- `origins mcp`, which gives an AI agent the whole corpus. The agent's client may send
  what it reads to the company that runs the model.

The files of a corpus are not encrypted.
