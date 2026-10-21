<h1><img src="web/public/lockup.svg" alt="Origins" height="40"></h1>

Origins is genealogy software that stores a family tree as text files in a git
repository. Each person, family, place, source and hypothesis is a Markdown file with
YAML frontmatter. You can read and edit the files with any text editor, and git keeps
the history of every change.

Origins checks the tree for contradictions, records each change as a git commit,
imports GEDCOM and Gramps files, and shows the tree in a browser.

## Contents

- [1. Status](#1-status)
- [2. Overview](#2-overview)
- [3. Installation](#3-installation)
- [4. Usage](#4-usage)
- [5. Contributing](#5-contributing)

## 1. Status

Origins is alpha software. Please read the following before you use it for your
research.

- The format, the commands and the `--json` output can change before version 1.0.
  When you upgrade, you may have to convert your corpus to the new format.
- Origins has been tested on its own test data and on public GEDCOM and Gramps sample
  files, but very little on real research. Keep your corpus in a git repository that
  you push to a remote, and keep the file you imported it from.
- Origins was written with the help of AI coding tools. The author directed the work
  and reviewed the code.

## 2. Overview

### 2.1 The corpus

Origins calls the directory that holds a family tree a corpus. Origins stores each
record in `<kind>/<slug>/README.md`, where the slug is the name of the record, such as
`rossi-caterina`. Files that belong to a record, such as photographs and scanned
documents, go in the same directory as its `README.md`:

```
rossi/
  origins.yaml
  people/
    rossi-caterina/
      README.md
      photos/1830-portrait.jpg
    lang-georg/README.md
  families/
    lang-georg--rossi-caterina/README.md
  places/
    hamburg/README.md
  sources/
    parish-san-lorenzo-baptisms/README.md
  hypotheses/
    louisa-born-1844/README.md
```

`origins.yaml` holds the name and the settings of the corpus. Each record is named
`README.md` so that GitHub and other git hosts display it when you open its directory.

### 2.2 Records

Here is a person record. Origins reads the YAML frontmatter at the top of the file.
Below it, you can write notes in Markdown, for example to explain how you reached a
conclusion:

```markdown
---
name:
  given: Caterina
  surname: Rossi
also-known-as:
  - given: Katharina
    surname: Lang
    type: married
    from: 1802
sex: F
events:
  - type: birth
    date: 3 APR 1779
    place: pian-del-mugnone
    sources: [parish-san-lorenzo-baptisms#1779-042]
  - type: death
    date: 1840
    place: hamburg
---

She married a Hamburg merchant at Florence in 1802 and went north with him, and every
German record after that calls her Katharina Lang.
```

A record refers to another record by its slug. In the example above, `hamburg` is the
slug of a place record.

Parents and children are stored only in family records. Origins reads them from there
when it shows a person's parents or children:

```markdown
---
partners: [lang-georg, rossi-caterina]
type: married
children: [lang-joseph, lang-anna-maria]
events:
  - type: marriage
    date: 1802
    place: firenze
---
```

### 2.3 Dates

Dates use the syntax of GEDCOM 7, with some extensions. For example, you can write
`3 ? 1850` when the record gives no month, or `ABT 1854 (the winter of the cholera)` to
keep the wording of the record. Origins supports seven calendars and keeps each date in
the calendar it was written in.

### 2.4 Hypotheses

If you aren't sure about something, you can write it as a hypothesis. A hypothesis is
a list of changes to the corpus, pinned to a commit, with your evidence and your
argument for it. `origins check --with <hypothesis>` checks the corpus as if the
hypothesis were true. When you have decided, `origins hypothesis accept` applies the
changes to the corpus, or `origins hypothesis reject` records that you rejected it.

## 3. Installation

Origins runs git for every change and for the history, so you need git on your
`PATH`. `origins version` tells you which git it found. Origins is tested with git 2.45
and later, and it hasn't been tried with older versions.

### 3.1 From a release

Each [release](https://github.com/origins-project/origins/releases) has an archive for
Linux, macOS and Windows, on amd64 and arm64. Each archive holds the `origins` binary,
the licence and this README. Download the one for your system, unpack it and copy
`origins` to a directory on your `PATH`.

`SHA256SUMS` lists the checksum of each archive, and GitHub holds an attestation that
the archive was built from the tagged commit by the release workflow. You can check
both:

```bash
sha256sum -c --ignore-missing SHA256SUMS
gh attestation verify origins_0.1.0_linux_amd64.tar.gz --repo origins-project/origins
```

On macOS, a file you download with a browser is marked as quarantined. The binary
isn't signed with an Apple Developer ID, so macOS refuses to run it while it has that
mark. You can remove the mark with `xattr -d com.apple.quarantine origins`.

### 3.2 From source

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

### 3.3 With Docker

To serve a corpus from a server, so that you can open it from another computer or a
phone, use the image `ghcr.io/origins-project/origins`, described in
[docker/README.md](docker/README.md). It is built for linux/amd64 and linux/arm64.

## 4. Usage

Run `origins help` to list the commands. `origins help <command>` and
`origins <command> -h` print the usage of a command, its flags and an example. Most
commands accept `--json` and then print their output as JSON.

### 4.1 Creating a corpus

`origins init` creates a corpus. If the directory isn't in a git repository, Origins
creates one. The first commit holds `origins.yaml`:

```
$ origins init --name 'Rossi family' rossi
ac5cf12a64  Start the Rossi family corpus
  recorded in a new git repository
  .gitattributes
  origins.yaml
$ cd rossi
```

Then you can add people, places and families. Origins makes one commit for each
command:

```
$ origins create person/rossi-giovanni name.given=Giovanni name.surname=Rossi sex=M
c0d3bfed3d Create person/rossi-giovanni
  people/rossi-giovanni/README.md
$ origins create person/bianchi-maria name.given=Maria name.surname=Bianchi sex=F
$ origins create place/borgo-san-lorenzo name='Borgo San Lorenzo' type=comune
$ origins create family/bianchi-maria--rossi-giovanni partners='[bianchi-maria, rossi-giovanni]' type=married
$ origins add person/rossi-giovanni events '{type: birth, date: ABT 1842, place: borgo-san-lorenzo}'
$ origins create person/rossi-annunziata name.given=Annunziata name.surname=Rossi sex=F
$ origins add family/bianchi-maria--rossi-giovanni children rossi-annunziata
```

The other commands that change records are `set`, `unset`, `remove`, `rename`,
`delete`, `merge` and `split`. `origins undo` takes back the last change.

### 4.2 Checking a corpus

Before each change, Origins checks that the corpus stays consistent. If the change
would introduce a contradiction, Origins refuses it and prints the reason:

```
$ origins add person/rossi-annunziata events '{type: birth, date: 1838}'
the change would make the corpus contradict itself:
  person/rossi-annunziata events[0].date: born 1838, before rossi-giovanni was born in ABT 1842 (child-before-parent)
```

`origins check` lists errors and warnings. An error is a contradiction between
records. A warning points to research you still have to do, such as an event without a
source:

```
$ origins check
warning: person/rossi-giovanni events[0]: birth cites nothing (unsourced-event)
0 error(s), 1 warning(s)
```

`origins check` exits with status 1 if it finds an error, so you can run it in a git
hook or in CI.

### 4.3 Editing in a browser or a text editor

`origins browse` opens the tree in your browser. You can edit the records there too,
and Origins makes a commit for each change.

You can also edit the files in a text editor. Afterwards, run `origins adopt`. Origins
checks your edits and records them in a commit.

### 4.4 Importing from GEDCOM and Gramps

To start from a tree you keep in another program, export it as a GEDCOM file or a
Gramps file, and import it into a new corpus:

```
$ origins import gedcom --into rossi rossi.ged
$ origins import gramps --into rossi rossi.gramps
```

Origins prints a report of the data it could not map to its own fields. It keeps that
data in the text of the record, so the import loses no data.

### 4.5 Querying the tree

`ancestors`, `descendants` and `kinship` answer questions about relationships:

```
$ origins kinship rossi-annunziata rossi-giovanni
Annunziata Rossi is the daughter of Giovanni Rossi

  rossi-annunziata  Annunziata Rossi
  rossi-giovanni    Giovanni Rossi    her father
```

`origins todo` lists the research you still have to do:

```
$ origins todo
person/rossi-giovanni: birth ABT 1842 is more than 110 years ago, and no death is recorded (unrecorded-death)
person/rossi-giovanni events[0]: birth cites nothing (unsourced-event)
2 things to do
```

### 4.6 History

`origins log` lists the changes to the corpus or to one record, and `origins blame`
shows when each fact in a record was recorded. `origins whence` lists the changes that
cite a source. `origins why` shows the commit that recorded one fact, and the
hypothesis that the commit accepted, if there is one:

```
$ origins why person/rossi-giovanni events[0].date
person/rossi-giovanni events[0].date
  date: ABT 1842

6fdf128447  2026-10-19  Add to events on person/rossi-giovanni
            events.add  person/rossi-giovanni  (Rosa Bianchi)
            touching birth
```

### 4.7 Sharing

You share a corpus with other people through a git remote, with `origins remote`,
`fetch`, `push`, `pull` and `clone`. `push` and `pull` refuse to send or take in
commits that Origins didn't write:

```
$ origins remote add origin git@example.org:rossi.git
$ origins push
$ origins clone git@example.org:rossi.git rossi
```

### 4.8 Exporting

`origins export` writes the tree as a GEDCOM 7 file, as a folder of HTML pages that you
can open without a server, or as a chart in SVG, PDF or PNG:

```
$ origins export gedcom --out rossi.gdz
$ origins export html --out site
$ origins export chart --shape descendant-tree --out rossi.pdf rossi-giovanni
```

### 4.9 AI agents

`origins mcp` serves the corpus to an AI agent over the Model Context Protocol. The
agent can read the tree and propose hypotheses. It can't accept or reject a
hypothesis, or change records in any other way.

## 5. Contributing

Please report bugs and send requests through the repository's GitHub issues. If a
GEDCOM or Gramps file doesn't import correctly and you can share it, attach the file or
a smaller one that shows the problem. Report security problems privately, as
[SECURITY.md](SECURITY.md) describes.

[CONTRIBUTING.md](CONTRIBUTING.md) describes how to build Origins, run the tests and
submit a change.

Origins is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
