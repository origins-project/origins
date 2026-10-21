# People

A person record holds the person's names, their sex as the sources give it, their
events and their pictures. It doesn't name the person's parents, partners or children:
Origins reads those from the [family records](./family) that list the person.

```corpus people/verdier-jean-baptiste/README.md
---
name:
  text: Jean Baptiste Verdier, fils de Louis
  given: Jean Baptiste
  call: Jean
  surname: Verdier
  sources: [french-church-london-baptisms#1834-042 OPD]
also-known-as:
  - J. B. Verdier
  - given: John
    surname: Verdia
    type: aka
    sources: [london-marriages-1858#p112 ---1]
sex: M
portrait: photos/1890-portrait.jpg
events:
  - type: birth
    date: 12 MAR 1834
    place: london
    sources: [french-church-london-baptisms#1834-042 OPD3]
  - type: death
    date: 3 NOV 1901
    place: london
media:
  - file: photos/1890-portrait.jpg
    title: Jean Baptiste in his fifties
    date: ABT 1890
    place: london
    depicts: [verdier-jean-baptiste]
  - file: photos/1875-carte-de-visite.jpg
    type: photo
    title: A carte de visite
    date: ABT 1875
ids:
  familysearch: XXXX-YYY
---

He appears as "J. B. Verdier" in the 1861 census and as "John Verdia" on his marriage
certificate.
```

## Fields

| Field | Holds |
|---|---|
| `name` | the name the person is filed under, usually the name at birth |
| `also-known-as` | every other name, each in the same form as `name` |
| `sex` | `M`, `F` or `X`, as the sources record it |
| `portrait` | the picture to show for the person |
| `events` | what happened to the person and the facts held about them, as [Events](./events) describes |
| `not` | events that the sources show didn't happen, as [Events](./events#events-that-didn-t-happen) describes |
| `living` | `true` or `false`, to overrule the rule that decides who is living, as [Private records](./private) describes |

A person also takes the fields that [most records take](./#fields-that-most-records-take).

## Names

A name is made of parts, and each part is optional:

| Part | Holds | Example |
|---|---|---|
| `prefix` | a title before the name | Dott., Don |
| `given` | the given names | Jean Baptiste |
| `call` | the given name the person went by | Jean |
| `nickname` | a familiar name | Gianni |
| `surname-prefix` | the part of the surname that isn't sorted on | de, van der |
| `surname` | the surname | Verdier |
| `suffix` | a part after the name | jr., III |
| `family-nickname` | the name a branch of the family went by | del Fabbro |
| `group` | the surname to file the name with, when its spelling would file it apart | Rossi, for Rosi |

A name with several surnames gives `surname` as a list. Each entry is a surname, or a
mapping with `name`, `prefix`, the `connector` to the next surname, such as `y`, the
`origin`, such as `patrilineal` or `matrilineal`, and `primary: true` on the one to
file the name under when that isn't the first:

```yaml
name:
  given: Gabriel José
  surname:
    - name: García
      origin: patrilineal
    - name: Márquez
      origin: matrilineal
```

`text` is the name as the record wrote it, without dividing it into parts. A name with
only `text` can be written as a plain string, as `J. B. Verdier` is above. Beside the
parts, `text` keeps the record's own words, as `Jean Baptiste Verdier, fils de Louis`
does.

`lang` is the language or script of the name, as a BCP 47 tag: `ru` for a name written
again in Cyrillic, `ja-Latn-hepburn` for a Japanese name in Latin letters.

`display` and `sort` say how to write out and sort a name, when the usual order
doesn't fit. Each is a pattern with the parts in braces, such as `{surname} {given}`
for a name written surname first. A part written in capitals, `{SURNAME}`, is written
in capitals. A pattern can use every part in the table above, and these, which Origins
makes from the parts:

| Code | Writes |
|---|---|
| `{initials}` | the initials of the given names, `J.B.` |
| `{common}` | the nickname, or else the call name, or else the first given name |
| `{primary-surname}` | the surname that the name is filed under |
| `{other-surnames}` | the other surnames |
| `{patronymic}` | the surname whose `origin` is `patronymic` or `matronymic` |

If the name has no value for a part, Origins leaves out the part and the text between
it and the part before it, so `{surname}, {given}` is `Verdier` for a name without
given names.

## Other names

Each name, in `also-known-as` or in `name`, can take a `type`. GEDCOM's types are
`birth`, `maiden`, `married`, `immigrant`, `professional`, `aka` for any other name the
person was known by, and `other`, and `origins export gedcom` writes these as GEDCOM's.
You can write a word of your own, such as `religious` or `adopted`. Each name can also
take `from` and `until` for when the person bore it.
A name with dates is one the person had from then on. A name without dates is a
spelling to match records against:

```yaml
also-known-as:
  - Maria Papadopulo          # a clerk's spelling, never borne
  - given: Maria              # the name she had after her marriage
    surname: Verdier
    type: married
    from: 1858
```

## Sex

`sex` records what the sources say. `X` is for a person whom the records set down as
neither male nor female. Leave `sex` out when the sources don't say. Origins also reads
`U`, as GEDCOM writes an unknown sex, as if `sex` were left out. To cite the
sources for it, write it as a mapping: `{value: F, sources: [...]}`.

## Portrait

`portrait` is the path of a picture in the person's directory, or a mapping with
`file` and the `region` of the picture that shows this person. If you leave it out,
Origins uses the first picture under `photos/`, and otherwise the first picture
anywhere in the corpus that shows the person.
