# Exporting

`origins export` writes the tree for somebody who doesn't use Origins: as a GEDCOM
file, as a folder of web pages, or as a chart. Each kind of export takes
`--hide-private`, `--hide-living` and `--public`, which [Privacy](./privacy)
describes, and `--with`, which exports the corpus as if a
[hypothesis](./hypotheses) were true.

## GEDCOM

`origins export gedcom` writes GEDCOM 7.0 in a GEDZIP file, a zip with the pictures in
it. A program that reads only GEDCOM 5.5.1 can't read this file.

```console
$ origins export gedcom --public --out rossi.gdz
wrote rossi.gdz: 1 FAM, 4 INDI, 1 SOUR, 0 pictures
extensions: 2 _ANCHOR, 1 _NAME, 2 _TYPE
```

GEDCOM has no place for some of what a corpus holds. Origins writes most of it as
extension tags, which it declares in the file's header, and lists them in the report.
What GEDCOM can't hold at all, such as hypotheses, Origins leaves out, and the report
lists that too. If you import the GEDZIP into Origins again, the extensions come back
as the fields they were written from.

## Web pages

`origins export html` writes the same pages that `origins browse` shows into a
directory, read-only. You can open `index.html` from the directory without a server,
put the directory on a web server, or send it as a zip. The
[demo](/demo/index.html){target="_self"} on this site is such an export.

```console
$ origins export html --public --out rossi-pages
wrote rossi-pages: 1 family, 4 people, 1 place, 1 source, 0 files
open rossi-pages/index.html in a browser
```

The directory must be empty, or not exist yet.

`--notice` writes a line of text in a banner below the top bar on every page. The demo
uses it to say that all of its data is generated:

```bash
origins export html --notice "Exported for the family reunion of 2026." --out rossi-pages
```

In the exported pages, the menu has *Info* in place of *Settings*. Info shows the
project's fields, the researchers, the bookmarks and the tags, and these details of
the export:

| Field | Meaning |
|---|---|
| Made | the date and time of the export, by the clock of the computer that made it |
| Commit | the commit the corpus was at, marked *Uncommitted changes* if the files differed from it. A corpus without a history has no commit. |
| Version | the version of Origins that made the export |
| View | what the export leaves out, or *Everything* |
| Supposed | the hypotheses that `--with` supposed |
| Notice | the text of `--notice` |

## Charts

`origins export chart` draws one chart of a person as SVG, PDF or PNG, according to
the extension of the file you name:

```console
$ origins export chart --shape fan --out rossi.pdf rossi-annunziata
drew the fan chart of Annunziata Rossi to rossi.pdf: 3 people
```

The shapes are the same as on the Chart tab of `origins browse`:

| Shape | Chart |
|---|---|
| `ancestors` | the ancestors in columns by generation |
| `descendants` | the descendants in columns by generation |
| `both` | the ancestors and the descendants |
| `relatives` | everybody related to the person in any way |
| `ancestor-tree` | the ancestors in rows, the oldest at the top |
| `descendant-tree` | the descendants in rows, downward |
| `fan` | the ancestors in half rings, one generation to a ring |
| `path` | the people between the person and the one that `--to` names |
| `timeline` | each relative's life as a bar on one axis of years |

`--distance` sets how many steps from the person the chart reaches, 5 if you don't
give it. A chart holds at most 400 people.
