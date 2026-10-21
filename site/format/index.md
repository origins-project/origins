# The corpus

A corpus is a directory with `origins.yaml` at its root and one directory for each
kind of record. The pages in this section describe each kind and each field, for when
you edit the files yourself. The [JSON Schema](../reference/schema) of each kind lists
the same fields for an editor or a program.

## Directories

Each record is a file `<kind>/<slug>/README.md`:

| Kind | Directory | Example |
|---|---|---|
| [person](./person) | `people/` | `people/verdier-jean-baptiste/README.md` |
| [family](./family) | `families/` | `families/papadopoulou-maria--verdier-jean-baptiste/README.md` |
| [place](./place) | `places/` | `places/pian-del-mugnone/README.md` |
| [source](./source) | `sources/` | `sources/parish-san-lorenzo-baptisms/README.md` |
| [repository](./repository) | `repositories/` | `repositories/archivio-diocesano-di-firenze/README.md` |
| [note](./note) | `notes/` | `notes/a-london-wedding-of-1859/README.md` |
| [hypothesis](./hypothesis) | `hypotheses/` | `hypotheses/louisa-born-1844/README.md` |

The other files in a record's directory, at any depth, belong to the record: its
photographs, scans and documents. You choose their names and subdirectories, such as
`photos/` and `scans/`, and the record refers to them by their path from its directory.
[Pictures and files](./media) describes how.

Origins refuses to read a corpus that has any other file in the seven kind
directories, and names each file. You can keep other files, such as your own notes or a
scan you haven't filed yet, anywhere outside them. Origins doesn't read files and
directories whose names begin with a dot.

## Slugs

The slug is the name of the record's directory, and other records refer to the record
by it. It is in lowercase ASCII letters and digits, with hyphens between the words, and
at most 64 characters long. Origins makes it from the record's name when it creates or
imports a record:

| Kind | Slug | Example |
|---|---|---|
| person | the surname, then the given names | `verdier-jean-baptiste` |
| family | the partners' slugs, joined by two hyphens | `papadopoulou-maria--verdier-jean-baptiste` |
| place, repository | the name | `pian-del-mugnone` |
| source, note | the title | `parish-san-lorenzo-baptisms` |

A person's slug leaves out the prefix of the surname, so `van der Berg` is filed with
the other Bergs. A family's slug puts its partners in alphabetical order, with
`unknown` and `none` last: `verdier-louisa--unknown`.

When a slug is taken, Origins offers others with something added to tell the two
apart, such as a year of birth or a place: `ershov-ivan-1885`. `origins rename` gives
a record a new slug and rewrites every reference to it.

## Frontmatter and body

A record's file starts with YAML frontmatter between two `---` lines, which Origins
reads. The rest of the file is Markdown, for your notes and the reasons behind your
conclusions. Origins shows the body on the record's page and doesn't read anything
from it except links, tasks and the blocks an import kept.

- A link to another record is a relative link to its file, such as
  `[Georg](../../people/lang-georg/README.md)`. Any Markdown viewer follows it, and
  `origins rename` rewrites it.
- A task, `- [ ] Look for the banns in Vicchio`, is listed by `origins todo` until you
  tick it.
- A label in italics at the start of a paragraph, such as `*Research:*` or
  `*Translation, en:*`, says what kind of note the paragraph is and, after a comma,
  its language.

## Plain and qualified values

Wherever a field holds a reference or a name, you can write it plainly or as a mapping
that says more about it. Here the first child is a plain slug, and the second is a
mapping that gives the relation to each parent:

```corpus families/galvao-henrique--verdier-pauline/README.md
---
partners: [galvao-henrique, verdier-pauline]
type: married
children:
  - galvao-joao
  - id: galvao-helena
    relation: {galvao-henrique: step, verdier-pauline: adoptive}
events:
  - type: marriage
    date: 11 JUN 1900
    place: lisboa
---
```

Use the mapping only when you have something to add. It is also where a value takes
its own `sources`, and every mapping takes a `note` about that one entry.

## Fields that most records take

People, families, places, sources, repositories and notes take these fields.
Hypotheses take none of them, and sources and repositories take no `sources`.

| Field | Holds |
|---|---|
| `sources` | the citations the record as a whole rests on, as [Sources and citations](./source) describes |
| `media` | the pictures and files of the record, as [Pictures and files](./media) describes |
| `ids` | the record's identifiers in other services, such as `familysearch: XXXX-YYY` |
| `links` | web addresses for the record, each a URL or a mapping with `url`, `type`, `title` and `note` |
| `tags` | your own words for grouping records, such as `To do` |
| `same-as` | other records of the same kind that may be the same as this one |
| `private` | `true` if the record isn't for sharing, as [Private records](./private) describes |

### ids

`ids` maps the name of a service to the record's identifier there. The list of
services is open: `familysearch`, `wikitree`, `wikidata` and `geonames` are common,
and the importers write `gramps`, `uid`, `rin`, `refn` and `afn` for the identifiers
that the other program kept. A service can list several identifiers, the current one
first, such as the identifiers of records that the service merged into this one. Each
identifier can be a mapping with `id`, a `note` and `sources`:

```yaml
ids:
  wikidata: Q2044
  familysearch:
    - XXXX-YYY
    - id: XXXX-XXX
      note: A duplicate that FamilySearch merged into this one in 2016.
```

`origins merge` keeps the identifiers of both records, those of the record merged away
after the others.

### links

Each link is a URL, or a mapping with `url`, a `type`, a `title` and a `note`. A
telephone number is a `tel:` link and an email address a `mailto:` link:

```yaml
links:
  - https://www.example.org/archivio-diocesano
  - url: https://www.familysearch.org/ark:/61903/1:1:XYZ
    title: The baptism on FamilySearch
  - mailto:archivio@example.org
```

### same-as

`same-as` is a claim that you haven't settled yet: `origins check` warns about each
one, until you merge the two records with `origins merge` or remove the claim.

```yaml
same-as:
  - id: ershov-ivan
    note: A tree on FamilySearch has the two as one man.
```

## Canonical form

Origins writes every file in one canonical form: the fields in a fixed order, the same
quoting and the same line breaks. A file that Origins wrote and you didn't change is
written again with exactly the same bytes, so a diff shows only what changed. Origins
never reorders a list. `origins fmt` rewrites the files you edited in that form, and
`origins adopt` does the same before it records them.

Every value in the frontmatter is text. Where a field takes a word, such as `true` for
`private`, Origins refuses the other words that YAML would read as the same value,
such as `yes`.
