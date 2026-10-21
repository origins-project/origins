---
layout: home
hero:
  name: Origins
  actions:
    - theme: brand
      text: Install
      link: /install
    - theme: alt
      text: Getting started
      link: /getting-started
    - theme: alt
      text: Demo
      link: /demo/index.html
      target: _self
---

<div class="vp-doc home-doc">

Origins is genealogy software that stores a family tree as text files in a git
repository. Each person, family, place, source and hypothesis is a Markdown file with
YAML frontmatter. You can read and edit the files with any text editor, and git keeps
the history of every change.

Origins checks the tree for contradictions, records each change as a git commit,
imports GEDCOM and Gramps files, and shows the tree in a browser. The
[demo](/demo/index.html){target="_self"} is the tree of an invented family, exported
with `origins export html`, and you can open it without installing anything.

::: warning Alpha software
The format, the commands and the `--json` output can change before version 1.0. When
you upgrade, you may have to convert your corpus to the new format. Origins has been
tested on its own test data and on public GEDCOM and Gramps sample files, but very
little on real research. Keep your corpus in a git repository that you push to a
remote, and keep the file you imported it from.

Origins was written with the help of AI coding tools. The author directed the work and
reviewed the code.
:::

## The corpus

Origins calls the directory that holds a family tree a corpus. Origins stores each
record in `<kind>/<slug>/README.md`, where the slug is the name of the record, such as
`rossi-caterina`. Files that belong to a record, such as photographs and scanned
documents, go in the same directory as its `README.md`:

```text
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
    parish-san-lorenzo-baptisms/
      README.md
      scans/1779-042.jpg
  hypotheses/
    louisa-born-1844/README.md
```

Each record is named `README.md`, so GitHub and other git hosts display it when you
open its directory.

## A record

Here is a person record. Origins reads the YAML frontmatter at the top of the file.
Below it, you can write notes in Markdown, for example to explain how you reached a
conclusion. A record refers to another record by its slug: `hamburg` is the slug of a
place, and `parish-san-lorenzo-baptisms#1779-042` is an entry in a source.

```corpus people/rossi-caterina/README.md
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

The record doesn't name Caterina's parents or children. Origins reads them from the
family records, as [Families](/format/family) describes.

</div>
