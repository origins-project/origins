# Importing from Gramps

If you keep your tree in [Gramps](https://gramps-project.org), export it in Gramps' own
format rather than as GEDCOM. Gramps keeps more than GEDCOM can carry, and Origins
reads all of it: an event shared by several people with each person's role in it, a
place within different places at different dates, the part of a photograph that shows
a person, citations with their confidence, tags with their colours, and the Gramps ID
of every record.

Origins reads the files that Gramps 5.1 to 6.1 write.

## Exporting from Gramps

In Gramps, choose **Family Trees → Export…** and one of these formats:

- **Gramps XML (family tree)** writes a `.gramps` file. It holds the tree and only the
  paths of the pictures.
- **Gramps XML Package (family tree and media)** writes a `.gpkg` file. It holds the
  tree and the pictures together.

Choose the package if you have pictures in the tree. On the export's options page,
leave the filters at the whole database, and leave private records and living people
in. Origins can [hide them](./privacy) later, when you share the tree.

## Running the import

`origins import gramps` takes the file and the directory to create the corpus in:

```console
$ origins import gramps --into rossi rossi.gpkg
rossi: 3 people, 1 family, 3 places, 1 source, 0 repositories

87fa7e04dc  Import rossi.gpkg: 3 people, 1 family, 3 places, 1 source, 0 repositories
  recorded in a new git repository

now run: origins check rossi
```

The directory in `--into` must not exist yet, or must be empty. Origins always makes a
new corpus, and doesn't merge an import into a corpus you already have. If the new
directory is inside a git repository, the import is a commit in that repository.
Otherwise, Origins makes a repository for it.

The commit message names the file, the version of Gramps that wrote it and the date of
the export:

```console
$ git -C rossi log -1 --format=%B
Import rossi.gpkg: 3 people, 1 family, 3 places, 1 source, 0 repositories

Read from rossi.gpkg, which Gramps 6.0.8 exported on 2026-10-14 as Gramps XML 1.7.2.
```

The import takes these flags:

| Flag | Does |
|---|---|
| `--into <dir>` | the directory to create the corpus in |
| `--media <dir>` | another directory to look for the pictures in, described below |
| `--name <name>` | the project's name in `origins.yaml` |
| `--interactive` | asks you for the slug whenever two records would get the same one |
| `--json` | prints the report as JSON |

## Pictures

A `.gpkg` package holds the pictures, and Origins copies each one beside the record
that shows it.

A `.gramps` file holds only the path of each picture, relative to the base media path
that you set in Gramps' preferences, and the file records that path. Origins looks for
each picture in that base path, and then in the directory that the `.gramps` file is
in. If you moved the file to another computer, or the pictures are somewhere else,
give their directory with `--media`:

```bash
origins import gramps --into rossi --media ~/genealogy/media rossi.gramps
```

The report counts the pictures that Origins didn't find:

```console
$ origins import gramps --into rossi rossi.gramps
rossi: 3 people, 1 family, 3 places, 1 source, 0 repositories
  a picture the export names was not found where Gramps would look for it, and is kept as the path it gave
```

The record keeps the path, and `origins check` lists it as a missing file. You can copy
the picture beside the record later, at that path.

## What each Gramps object becomes

| Gramps | Origins |
|---|---|
| Person | a [person](../format/person). The first name is `name`, and the others are in `also-known-as` with their types. |
| Family | a [family](../format/family), with its relationship type as `type` and each child's relation to the father and the mother |
| Event | an entry of `events` in the record of the person or family whose event it is, in the role *Primary* or *Family* |
| A person in an event in another role, such as a witness or a godparent | an entry of the event's `with`, with the role |
| Place | a [place](../format/place), with its type, its other names, its coordinates and every place it is within, with the dates |
| Source | a [source](../format/source) |
| Citation | a citation in its source, cited where Gramps cites it, with the confidence as the grade's weight |
| Repository | a [repository](../format/repository), and the call number in the source |
| Note | the body of the record it is on, or the `note` of the event or entry. A note on its own is a [note](../format/note). |
| Media object | an entry of `media` in each record that refers to it, with the region that shows each person in `depicts` |
| A person's or a family's attribute | an entry of `events` with a `value`, as a GEDCOM attribute is, or an entry of `ids` where it is an identifier elsewhere |
| An event's attribute | the event's `cause`, `age` or time where it is one of those, and otherwise an entry of its `attributes` |
| Address | a `residence` event, with the address's parts |
| Tag | a word in the record's `tags`, and its colour and priority in `origins.yaml` |
| Bookmarks and the home person | `bookmarks` and `main-person` in `origins.yaml` |
| Researcher in the database's owner information | a researcher in `origins.yaml` |
| Gramps ID | `ids.gramps`, such as `gramps: I0001` |
| Private | `private: true` |

Origins writes the event types, the place types and the family types in the same words
as a GEDCOM import does, so that a corpus has the same words whichever program it came
from. Gramps' confidence becomes the fourth character of a citation's
[grade](../format/source#grades): *Very Low* is `0`, *Low* `1`, *High* `2` and
*Very High* `3`. *Normal*, which Gramps gives every new citation, gives no grade.

## An example

The tree in `rossi.gpkg` has Giovanni Rossi and Maria Bianchi, married in Firenze in
1864 with Pietro Sarti as a witness. In Gramps, the marriage is one event that the
family holds and that Pietro refers to in the role *Witness*. Origins writes it in the
family, and names Pietro in its `with`. The citation of the marriage register has the
confidence *High*:

```corpus families/bianchi-maria--rossi-giovanni/README.md
---
partners: [bianchi-maria, rossi-giovanni]
type: married
events:
  - type: marriage
    date: 2 MAY 1864
    place: firenze
    with:
      - person: sarti-pietro
        role: witness
    sources: [san-lorenzo-matrimoni-1860-1870#n-31 ---2]
    ids:
      gramps: E0001
ids:
  gramps: F0001
---
```

Giovanni's photograph has a region drawn on his face in Gramps, written as the
fractions of the picture's width and height from its top-left corner:

```corpus people/rossi-giovanni/README.md
---
name:
  given: Giovanni
  surname: Rossi
  type: birth
sex: M
events:
  - type: birth
    date: 12 MAR 1842
    place: firenze
    ids:
      gramps: E0002
media:
  - file: photos/giovanni-1890.jpg
    title: Giovanni about 1890
    depicts:
      - person: rossi-giovanni
        region: [0.2, 0.1, 0.6, 0.7]
    ids:
      gramps: O0001
ids:
  gramps: I0001
---
```

In Gramps, Firenze is enclosed by the Granducato di Toscana before 1860 and by Toscana
after 1861. Origins keeps both, with their dates:

```corpus places/firenze/README.md
---
name: Firenze
type: city
within:
  - place: granducato-di-toscana
    until: 1860
  - place: toscana
    from: 1861
coordinates: 43.7696, 11.2558
ids:
  gramps: P0001
---
```

::: details The other files of the import
```corpus people/bianchi-maria/README.md
---
name:
  given: Maria
  surname: Bianchi
  type: birth
sex: F
ids:
  gramps: I0002
---
```

```corpus people/sarti-pietro/README.md
---
name:
  given: Pietro
  surname: Sarti
  type: birth
sex: M
ids:
  gramps: I0003
---
```

```corpus sources/san-lorenzo-matrimoni-1860-1870/README.md
---
title: San Lorenzo, Matrimoni 1860-1870
citations:
  n-31:
    page: n. 31
    ids:
      gramps: C0001
ids:
  gramps: S0001
---
```
:::

## The report

The first line counts the records that the import wrote. Each line under it counts
something you may want to look at. Here is the report for
[the royal family tree](https://github.com/DavidMStraub/gramps-web-example-tree-queen)
that Gramps Web publishes as an example, with 4,668 people:

```console
$ origins import gramps --into queen queen.gramps
queen: 4668 people, 2855 families, 955 places, 72 sources, 1 repository
  <citationref> was kept as written
  a citation was referred to that is not in the file
  a name gave no surname, so the person is filed under the whole of it (2541 times)
  a picture the export names was not found where Gramps would look for it, and is kept as the path it gave (175 times)
  a place had no name a slug can spell and is filed under its identifier (4 times)
```

| Line | Means |
|---|---|
| a name gave no surname | the person has no surname in Gramps, as kings and saints often don't, so the slug is made from the whole name |
| a citation was referred to that is not in the file | the file refers to a citation that it doesn't hold, which Gramps' own *Check and Repair* tool would report too |
| `<citationref>` was kept as written | an element that Origins couldn't read, here the reference to that missing citation, kept in the record as described below |
| a place had no name a slug can spell | the place is named only in a script that a slug can't spell, such as Cyrillic, so its slug is made from its Gramps ID |
| a picture the export names was not found | described in [Pictures](#pictures) |

## Elements that Origins has no field for

Origins writes any element of the file that it doesn't read in a fenced block marked
`gramps`, in the body of the record or in the note of the event it was in, so that the
import loses nothing. To find them all:

```bash
grep -rl '```gramps' queen
```

You can leave the blocks where they are, or move what they say into the record's
fields and delete the block.

## After the import

Origins makes each record's slug from its name, such as `rossi-giovanni`. When two
records would get the same slug, it adds something to tell them apart, such as a year
of birth. With `--interactive`, it asks you to choose instead. You can rename a record
later with `origins rename`.

Run `origins check` on the new corpus, and open it with `origins browse`:

```console
$ origins check rossi
warning: person/rossi-giovanni events[0]: birth cites nothing (unsourced-event)
0 error(s), 1 warning(s)
```

Keep the file you imported from. If you find a problem with the import later, the file
is what you need to report it, and to import it again with a newer version of Origins.
