# Importing from GEDCOM

Almost every genealogy program and website can export a tree as a GEDCOM file. Origins
reads that file and writes the tree as a new corpus, recorded as one commit.

Origins reads GEDCOM 5.5.1, which most programs write, including the variations that
Ancestry, FamilySearch and other programs add, and GEDCOM 7.0. It reads a `.ged` file,
or a GEDZIP, a `.gdz` or `.zip` file that holds the GEDCOM file and its pictures
together.

## Exporting from your program

Export the whole tree as GEDCOM, including the private records and the living people,
since Origins can [hide them](./privacy) when you share the tree later. If the program
offers GEDCOM 7.0 or a GEDZIP, choose it: a GEDZIP carries the pictures, and a plain
`.ged` only names them.

If you export a plain `.ged` file, put it in a directory together with the pictures it
names, in the same subdirectories, so that Origins finds them. A program that exports
the pictures as a separate folder usually writes the paths so that this works when the
folder sits beside the file.

## Running the import

The examples on this page import `rossi.ged`, a small file as another program might
export it, with one of the pictures it names in `photos/` beside it:

```text
export/
├── rossi.ged
└── photos/
    └── giovanni-1890.jpg
```

::: details rossi.ged
```text
0 HEAD
1 SOUR FamilyTreeProgram
2 VERS 9.1
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 DATE 14 OCT 2026
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 SEX M
1 BIRT
2 DATE 12 MAR 1842
2 PLAC Borgo San Lorenzo, Firenze, Toscana, Italia
2 SOUR @S1@
3 PAGE n. 57
1 OCCU contadino
2 DATE 1871
2 PLAC Borgo San Lorenzo, Firenze, Toscana, Italia
1 DEAT
2 DATE ABT 1910
1 OBJE
2 FILE photos/giovanni-1890.jpg
3 FORM jpg
2 TITL Giovanni about 1890
1 FAMS @F1@
0 @I2@ INDI
1 NAME Maria /Bianchi/
1 SEX F
1 BIRT
2 DATE 1845
2 PLAC Vicchio, Firenze, Toscana, Italia
1 FAMS @F1@
0 @I3@ INDI
1 NAME Annunziata /Rossi/
1 SEX F
1 BIRT
2 DATE 25 MAR 1866
2 PLAC Borgo San Lorenzo, Firenze, Toscana, Italia
2 SOUR @S1@
3 PAGE n. 112
1 OBJE
2 FILE photos/annunziata-wedding.jpg
3 FORM jpg
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 MARR
2 DATE 1864
2 PLAC Borgo San Lorenzo, Firenze, Toscana, Italia
0 @S1@ SOUR
1 TITL Borgo San Lorenzo, Battesimi 1840-1870
1 REPO @R1@
2 CALN Battesimi 14
0 @R1@ REPO
1 NAME Archivio Arcivescovile di Firenze
0 TRLR
```
:::

`origins import gedcom` takes the file and the directory to create the corpus in:

```console
$ origins import gedcom --into rossi export/rossi.ged
rossi: 3 people, 1 family, 5 places, 1 source, 1 repository
  a picture the export names was not found beside it, and is kept as the path it gave
  an event cites no source (4 times)

067a8b440b  Import rossi.ged: 3 people, 1 family, 5 places, 1 source, 1 repository
  recorded in a new git repository

now run: origins check rossi
```

The directory in `--into` must not exist yet, or must be empty. Origins always makes a
new corpus, and doesn't merge an import into a corpus you already have. If the new
directory is inside a git repository, the import is a commit in that repository.
Otherwise, Origins makes a repository for it.

A GEDZIP is imported in the same way:

```bash
origins import gedcom --into rossi rossi.gdz
```

The import takes these flags:

| Flag | Does |
|---|---|
| `--into <dir>` | the directory to create the corpus in |
| `--name <name>` | the project's name in `origins.yaml`, such as `--name "The Rossi of Borgo San Lorenzo"` |
| `--interactive` | asks you for the slug whenever two records would get the same one |
| `--json` | prints the report as JSON |

The commit message names the file and, from the file's header, the program that wrote
it, the date of the export and the version of GEDCOM:

```console
$ git -C rossi log -1 --format=%B
Import rossi.ged: 3 people, 1 family, 5 places, 1 source, 1 repository

Read from rossi.ged, which FamilyTreeProgram 9.1 exported on 14 OCT 2026 as GEDCOM 5.5.1 in UTF-8.
```

## The report

The first line counts the records that the import wrote. Each line under it counts
something you may want to look at:

| Line | Means |
|---|---|
| a picture the export names was not found beside it | the file names a picture that wasn't at the path it gives, or beside the file. The record keeps the path, and `origins check` lists it as a missing file. |
| an event cites no source | an event has no citation. This is common, and `origins check` lists each one. |
| a date could not be read and was kept as words | the date isn't in a form that Origins reads, and is kept as the [record's own words](../format/dates#the-record-s-own-words) |
| a name gave no surname | the name has no part marked as the surname, so the person's slug is made from the whole name |
| `<tag>` was kept as written | the file has a tag that Origins has no field for, kept in the record as described [below](#lines-that-origins-has-no-field-for) |
| a mention of a place disagrees with another and is a place of its own | two events name the same place but give it different coordinates, types or identifiers. Origins writes two places, and you can compare them and merge them. |

Keep the file you imported from. If you find a problem with the import later, the file
is what you need to report it, and to import it again with a newer version of Origins.

## What each GEDCOM record becomes

| GEDCOM | Origins |
|---|---|
| `INDI` | a [person](../format/person), with every `NAME` after the first in `also-known-as` |
| `FAM` | a [family](../format/family): `HUSB` and `WIFE` are its partners, `CHIL` its children, and each child's `PEDI` its relation to the parents |
| `BIRT`, `DEAT`, `MARR` and the other events | entries of `events`, with the type in words, such as `birth`, `death` and `marriage` |
| `OCCU`, `RESI`, `TITL` and the other attributes | entries of `events` with a `value`, such as `occupation: contadino` |
| `PLAC` | a [place](../format/place) for each level of the name, each one `within` the next |
| `SOUR` | a [source](../format/source) |
| `SOUR` under an event, with its `PAGE` | a citation in the source, which the event cites |
| `REPO` | a [repository](../format/repository), and the `CALN` its call number in the source |
| `NOTE` record | a [note](../format/note) |
| `NOTE` under a record or an event | the record's body or the event's `note` |
| `OBJE` | an entry of `media`, and the picture copied into the record's directory |
| `RESN PRIVACY` or `CONFIDENTIAL` | `private: true` |
| `REFN`, `UID`, `EXID`, `RIN`, `AFN` | entries of `ids` |
| `SUBM` | a researcher in `origins.yaml` |

The import of `rossi.ged` wrote these files:

```text
rossi/
├── origins.yaml
├── people/
│   ├── bianchi-maria/README.md
│   ├── rossi-annunziata/README.md
│   └── rossi-giovanni/
│       ├── README.md
│       └── photos/giovanni-1890.jpg
├── families/bianchi-maria--rossi-giovanni/README.md
├── places/
│   ├── borgo-san-lorenzo/README.md
│   ├── firenze/README.md
│   ├── italia/README.md
│   ├── toscana/README.md
│   └── vicchio/README.md
├── sources/borgo-san-lorenzo-battesimi-1840-1870/README.md
└── repositories/archivio-arcivescovile-di-firenze/README.md
```

Here is Giovanni. The birth's `SOUR` with its `PAGE` became a citation that the birth
cites, the `OCCU` an event with a `value`, and the picture was copied beside his file:

```corpus people/rossi-giovanni/README.md
---
name:
  given: Giovanni
  surname: Rossi
sex: M
events:
  - type: birth
    date: 12 MAR 1842
    place: borgo-san-lorenzo
    sources: [borgo-san-lorenzo-battesimi-1840-1870#n-57]
  - type: occupation
    value: contadino
    date: 1871
    place: borgo-san-lorenzo
  - type: death
    date: ABT 1910
media:
  - file: photos/giovanni-1890.jpg
    title: Giovanni about 1890
---
```

The source holds both pages that the file cites, and the repository and call number
from its `REPO`:

```corpus sources/borgo-san-lorenzo-battesimi-1840-1870/README.md
---
title: Borgo San Lorenzo, Battesimi 1840-1870
repositories:
  - repository: archivio-arcivescovile-di-firenze
    call-number: Battesimi 14
citations:
  n-112:
    page: n. 112
  n-57:
    page: n. 57
---
```

```corpus repositories/archivio-arcivescovile-di-firenze/README.md
---
name: Archivio Arcivescovile di Firenze
---
```

## Places

GEDCOM writes a place's full name at every event that happened there, such as
`Borgo San Lorenzo, Firenze, Toscana, Italia`. Origins makes one place record for each
level, `borgo-san-lorenzo` within `firenze` within `toscana` within `italia`, and every
event refers to the most specific one. Two places with the same name in different
regions stay two places, because Origins compares the whole name.

```corpus places/borgo-san-lorenzo/README.md
---
name: Borgo San Lorenzo
within: firenze
---
```

If the file has a `FORM` for its places, such as `City, County, State, Country`, each
level gets that `type`. Coordinates in `MAP` go on the most specific place.

## Lines that Origins has no field for

Programs add tags of their own, whose names begin with an underscore, and some
standard tags have no field in Origins. The import doesn't drop them. It writes each
such line, with the lines under it, in a fenced block marked `gedcom` in the body of the
record or in the note of the event it was under, and the report counts them. To find
them all:

```bash
grep -rl '```gedcom' rossi
```

You can leave the blocks where they are, or move what they say into the record's
fields and delete the block.

## Slugs

Origins makes each record's slug from its name, such as `rossi-giovanni`. When two
records would get the same slug, Origins adds something to tell them apart, such as a
year of birth: `rossi-giovanni` and `rossi-giovanni-1871`. With `--interactive`, it
asks you to choose instead. You can rename a record later with `origins rename`.

## After the import

Run `origins check` on the new corpus. The other program may have allowed things that
contradict each other, and the check is the first place where you see them:

```console
$ origins check rossi
warning: family/bianchi-maria--rossi-giovanni events[0]: marriage cites nothing (unsourced-event)
warning: person/bianchi-maria events[0]: birth cites nothing (unsourced-event)
warning: person/rossi-annunziata media[0].file: no file "photos/annunziata-wedding.jpg" beside the person (missing-media)
warning: person/rossi-giovanni events[1]: occupation cites nothing (unsourced-event)
warning: person/rossi-giovanni events[2]: death cites nothing (unsourced-event)
0 error(s), 5 warning(s)
```

Then open the corpus in the browser with `origins browse rossi`.
