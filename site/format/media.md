# Pictures and files

You keep a picture or a document in the directory of the record it belongs to, and the
record's `media` says what is in it. Each entry names one file by its path from the
record's directory:

```corpus families/papadopoulou-maria--verdier-jean-baptiste/README.md
---
partners: [papadopoulou-maria, verdier-jean-baptiste]
type: married
children: [verdier-anna, verdier-pauline, verdier-pierre]
media:
  - file: photos/1858-wedding-group.jpg
    title: The wedding party
    date: 21 APR 1858
    place: london
    depicts:
      - person: papadopoulou-maria
        region: [0.41, 0.12, 0.58, 0.47]
        note: Named in pencil on the back, as the bride.
      - person: verdier-jean-baptiste
        region: [0.59, 0.1, 0.76, 0.49]
  - file: ../../people/verdier-jean-baptiste/photos/1875-carte-de-visite.jpg
    note: Kept in the album beside the wedding group.
---
```

## Fields

| Field | Holds |
|---|---|
| `file` | the path of the file, from the record's directory |
| `type` | the medium, such as `photo`, `tombstone`, `newspaper`, `map`, `manuscript`, or a word of your own |
| `title`, `date`, `place` | what the picture is, and when and where it was taken or made |
| `region` | the part of the picture that the record is about, such as one house in a street |
| `depicts` | the people in the picture |
| `attributes`, `note`, `sources`, `ids`, `tags` | as on an event |
| `private` | `true` to leave the picture out of a hidden view |

`type` is a word of your own. GEDCOM's media are `audio`, `book`, `card`,
`electronic`, `magazine`, `manuscript`, `map`, `microfilm`, `microfiche`,
`newspaper`, `photo`, `tombstone` and `video`, and `origins export gedcom` writes these
as GEDCOM's.

You don't need an entry for a file you have nothing to say about. An entry that only
names the file, as a citation's scan often does, can be written as its path:
`- scans/1779-042-verso.jpg`.

Each entry of `depicts` is a person's slug, or a mapping with `person`, the `region`
that shows them, `attributes` and a `note`. A picture is shown on the page of every
person it depicts, whichever record keeps it, and Origins can take a person's portrait
from it.

## Regions

A region is four numbers between 0 and 1: the left, top, right and bottom edges, as
fractions of the picture's width and height. `[0.41, 0.12, 0.58, 0.47]` is a box that
starts 41% of the way across and 12% of the way down. Because the numbers are fractions,
a region stays correct when the picture is scaled. The pages draw the region over the
picture, or crop the picture to it.

## A picture that another record keeps

A file belongs to one record. Another record that has the same picture refers to it by
a path into the first record's directory, as the second entry above does. A markdown
viewer follows the same path, and `origins rename` and `origins merge` rewrite it as
they rewrite every other reference.

An entry like that holds only what this record says about the picture: its `region`,
`attributes`, `note`, `sources` and `private`. The title, the date, the place, the type,
the identifiers and the people in the picture are written once, in the entry of the
record that keeps the file.

## Events

An event takes `media` too, for the pictures of that event, such as the photograph of a
wedding or the gravestone of a burial. The file is kept in the directory of the record
that holds the event.

## Files you don't have

An entry whose file isn't there is still an entry. An import writes the path that the
other program had, often a path on somebody else's computer, and `origins check` lists
it as a file to find. When you find the file, put it beside the record and write its
path in `file`.
