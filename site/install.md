# Install

Origins is one program, `origins`, and it runs git for every change and for the
history. You need git on your `PATH`. Origins is tested with git 2.45 and later, and
it hasn't been tried with older versions.

## From a release

Each [release](https://github.com/origins-project/origins/releases) has an archive for
Linux, macOS and Windows, on amd64 and arm64. Each archive holds the `origins` binary,
the licence, the README and the notices of the software that Origins includes.
Download the one for your system, unpack it and copy `origins` to a directory on your
`PATH`. Then check that Origins finds git. `origins version` prints the version, the
commit it was built from, the version of Go that built it, and the git it found. On
Linux, the output looks like this:

```console
$ origins version
origins 0.1.0
commit  0123456789abcdef0123456789abcdef01234567
go      go1.27.3
git     2.45.2, at /usr/bin/git
```

### Checking the download

`SHA256SUMS`, beside the archives, lists the checksum of each archive. GitHub also
holds an attestation that each archive was built from the tagged commit by the release
workflow. You can check both, the second with the [GitHub CLI](https://cli.github.com/):

```bash
sha256sum -c --ignore-missing SHA256SUMS
gh attestation verify origins_0.1.0_linux_amd64.tar.gz --repo origins-project/origins
```

### macOS

On macOS, a file you download with a browser is marked as quarantined. The binary
isn't signed with an Apple Developer ID, so macOS refuses to run it while it has that
mark. You can remove the mark:

```bash
xattr -d com.apple.quarantine origins
```

## With Docker

To serve a corpus from a server, so that you can open it from another computer or a
phone, use the image `ghcr.io/origins-project/origins`. It is built for linux/amd64 and
linux/arm64. [Running on a server](./guides/docker) describes how to run it with
Docker Compose behind a reverse proxy.

## From source

You need Go 1.27.3 or later and git. To build the web application that
`origins browse` serves, you also need Node.js 22.18, or 24.12 or later. If you build
without Node, everything else works, and `origins browse` shows a placeholder page
instead of the application.

```bash
git clone https://github.com/origins-project/origins.git
cd origins
npm --prefix web ci
npm --prefix web run build
go build ./cmd/origins
```

These commands build the `origins` binary in the current directory. Copy it to a
directory on your `PATH`.

`go install github.com/origins-project/origins/cmd/origins@latest` also works, but it
builds without Node, so `origins browse` shows the placeholder page.

## Versions

Version numbers follow [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).
Before version 1.0, a new release can change the format, the commands, their flags and
the `--json` output. The notes of each release on the
[releases page](https://github.com/origins-project/origins/releases) list the new
features and the fixes it brings.

A corpus records the version of its format in `origins.yaml`, as `format-version`. If
a corpus has a later format version than the one your `origins` knows, Origins refuses
to read it and tells you which version it found. Upgrade Origins to read it.
