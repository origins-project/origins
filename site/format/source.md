# Sources and citations

A source record describes a document that exists: a register, a census, a book, a
website, an interview. A citation is an entry in that source, such as one baptism in a
register. The citations are inside the source record, each under a short name called
its anchor, and a fact cites one by the source's slug and the anchor:
`parish-san-lorenzo-baptisms#1779-042`. The transcription of an entry is kept once,
in its citation, and every fact that rests on the entry cites it.

```corpus sources/parish-san-lorenzo-baptisms/README.md
---
title: Basilica di San Lorenzo, Firenze — Baptismal Register 1770-1790
short-title: S. Lorenzo, Battesimi 12
type: parish-register
covers:
  - events: [baptism]
    date: FROM 1770 TO 1790
    place: firenze
repositories:
  - repository: archivio-diocesano-di-firenze
    call-number: ADF/SL/B/12
  - repository: archivio-di-stato-di-firenze
    call-number: Film 1903581
    medium: microfilm
citations:
  1779-042:
    event: baptism
    page: c. 88r, n. 112
    date: 5 APR 1779
    text: >
      Caterina, figlia di Bartolomeo Rossi e Angela Sarti, del popolo di San
      Michele a Pian del Mugnone, nata il 3 aprile 1779 e battezzata il 5
      detto.
    with:
      - person: rossi-caterina
        role: child
      - person: rossi-bartolomeo
        role: father
    images:
      - file: scans/1779-042.jpg
        region: [0.06, 0.38, 0.94, 0.61]
      - scans/1779-042-continued.jpg
    note: >
      *Translation, en:* Caterina, daughter of Bartolomeo Rossi and Angela
      Sarti, of the parish of San Michele at Pian del Mugnone, born on 3 April
      1779 and baptised on the 5th.
---

The hand is consistent through 1781 and then changes.
```

## Fields

| Field | Holds |
|---|---|
| `title` | the source's title |
| `short-title` | the abbreviation it is cited by, which the pages show beside a citation |
| `type` | the kind of source, such as `parish-register`, `census`, `book`, or a word of your own |
| `author`, `publication` | for a book or a website |
| `covers` | the events the source records, over which dates and for which place |
| `repositories` | where the source is kept |
| `text` | the whole text of a short source, apart from your notes in the body |
| `attributes` | anything else said of the source, such as the `agency` that kept it |
| `citations` | the entries, each under its anchor |

A source also takes `media`, `ids`, `links`, `tags`, `same-as` and `private`, as
[most records](./#fields-that-most-records-take) do. A source takes no `sources`, because
its citations are what other records cite.

`covers` lists what a register or an index records. Each entry has the `events` it
records, the `date` as a period, the `place` and a `note`. The register above covers
the baptisms in Firenze from 1770 to 1790, which tells you where to look for a person
born there in those years.

Each entry of `repositories` is a repository's slug, or a mapping with `repository`,
the `call-number` there, the `medium` that the repository keeps the source in, and a
`note`. The `medium` is a word of your own, or one of GEDCOM's media that
[Pictures and files](./media#fields) lists, such as `book`, `microfilm` or
`electronic`. A source with two call numbers in one archive, an old one and a new one,
has two entries for that archive.

## Citations

An anchor is short and has no spaces, such as `1779-042` or `p112`. A citation takes
these fields:

| Field | Holds |
|---|---|
| `event` | the kind of entry, such as `baptism` or `burial`, which needn't be the fact that cites it |
| `page` | where the entry is in the source, as somebody would read it: `c. 88r, n. 112` |
| `date` | when the entry was written, which can differ from when the event happened |
| `text` | the transcription |
| `with` | the people the entry names and their roles, as in an [event](./events#people-named-in-an-event) |
| `images` | the scans of the entry, each a path or a [media entry](./media) whose `region` marks the entry on the page |
| `links` | the entry online, for example on Antenati or FamilySearch |
| `attributes`, `ids`, `tags` | as on an event |
| `note` | what you say about the entry, such as a reading you doubt |

## Citing

Every fact that takes `sources` lists citations: an event, a name, a sex written as a
mapping, a child's link to a family, a place's `within`, a person named in an event,
an attribute, a picture and a whole record. A hypothesis's `supports` and `contradicts`
list citations in the same way.

```yaml
sources:
  - parish-san-lorenzo-baptisms#1779-042
```

## Grades

A citation can carry a grade after it, a space and four characters. The first three
are the questions that the Genealogical Proof Standard asks of a piece of evidence,
and the fourth is how much you trust it for this fact:

| Position | Question | Values |
|---|---|---|
| 1 | Is the source original, derivative or authored? | `O`, `D`, `A` |
| 2 | Is the information primary, secondary or undetermined? | `P`, `S`, `U` |
| 3 | Is the evidence direct, indirect or negative? | `D`, `I`, `N` |
| 4 | How far can it be trusted for this fact? | `0` unreliable, `1` doubtful, `2` reliable, `3` certain |

A `-` stands for a position you haven't assessed. You can leave off the fourth
position, or the whole grade:

```yaml
sources:
  - london-deaths-1901#1103-17 OPD3           # the death, stated by the informant
  - london-deaths-1901#1103-17 OSI2           # the same entry, cited for the birth year
  - parish-san-lorenzo-baptisms#1779-042 O--  # only the source assessed
  - census-1861-england#f214 ---1             # only the weight
  - french-church-london-baptisms#1842-007    # no grade
```

The grade belongs to the fact that cites the entry, and not to the source. A death
record is primary information for the death and secondary information for the birth
year it implies, so the same citation has a different grade for each.
