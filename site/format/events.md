# Events

`events` is a list in a person, a family or a note. An entry is something that
happened, such as a birth, a marriage or an emigration, or a fact that was held, such
as an occupation, a title or a residence. Both kinds go in the same list. A person's
events are about that person, and a family's events are about the couple.

```corpus people/verdier-pauline/README.md
---
name:
  given: Pauline
  surname: Verdier
sex: F
events:
  - type: birth
    date: 4 JUN 1863
    place: london
    attributes:
      - type: fathers-age
        value: 29
  - type: death
    date: 1941
    place: lisboa
---
```

## Fields

| Field | Holds |
|---|---|
| `type` | what the event was, such as `birth`, `baptism`, `occupation`, or a word of your own |
| `value` | the event's own words: what was held, such as `silk weaver`, or what the record says of the event |
| `date` | when, as [Dates](./dates) describes; for a fact held, usually a period such as `FROM 1855 TO 1880` |
| `sort-date` | where to put the event in order when its `date` doesn't say |
| `place` | the slug of a place |
| `address` | where in the place, such as a house or a church |
| `age` | the age the record gives the person, such as `67y`, `3m 12d` or `< 1y` |
| `cause` | the cause, of a death above all |
| `with` | the other people the record names |
| `family` | on a person's event, the family the record ties the event to |
| `attributes` | anything else the record says of the event |
| `media` | the pictures of the event |
| `links`, `ids`, `tags` | as on a record |
| `note` | what you have to say about this one event |
| `sources` | the citations the event rests on |
| `private` | `true` if the event isn't for sharing |

An event without a date isn't placed on the timeline. The record's page shows it among
the details instead, as "Title: Conte di Montalto".

## Types

The list of types is open. Write a type that Origins doesn't know in the words the
record uses, such as `Military Service (WWI)` or `兵役`. Origins compares types without
regard to capitals and punctuation, so `Birth` and `birth` are the same type.

The importers write the types of GEDCOM in these words, and `origins export gedcom`
writes each of them back as its GEDCOM tag:

| Of a person | Types |
|---|---|
| Life | `birth`, `death`, `burial`, `cremation`, `adoption` |
| Religion | `christening`, `adult-christening`, `baptism`, `blessing`, `confirmation`, `first-communion`, `ordination`, `bar-mitzvah`, `bas-mitzvah` |
| Records | `census`, `probate`, `will`, `emigration`, `immigration`, `naturalisation`, `graduation`, `retirement` |
| Facts held, with a `value` | `occupation`, `residence`, `education`, `nationality`, `religion`, `property`, `title`, `caste`, `description`, `identifying-number`, `social-security-number`, `number-of-children`, `number-of-marriages` |
| Rites of the Latter-day Saints | `lds-baptism`, `lds-confirmation`, `lds-initiatory`, `lds-endowment`, `lds-sealing-to-parents` |

| Of a family | Types |
|---|---|
| Partnership | `engagement`, `marriage-banns`, `marriage-licence`, `marriage-contract`, `marriage-settlement`, `marriage`, `divorce-filed`, `divorce`, `annulment` |
| Facts held | `census`, `residence`, `number-of-children` |
| Rites of the Latter-day Saints | `lds-sealing-to-spouse` |

Gramps and Family Tree Maker have types of their own, which the importers write as
`military-service`, `employment`, `degree`, `elected`, `funeral`, `circumcision`,
`excommunication`, `medical-information`, `cause-of-death`, `height`, `weight`,
`eye-colour`, `dna`, `origin`, `namesake`, `military-id`, `destination`, `lds-mission`
and `ordinance` for a person, and `separation` and `common-law-marriage` for a family.

`origins export gedcom` writes any type that GEDCOM has no tag for as an `EVEN`, or as
a `FACT` if the event has a `value`, with the type in its `TYPE`.

A few types mean something to Origins beyond their name:

- The pages show a person's dates from the `birth` and the `death`. Where there is no
  birth, they use the `baptism` or the `christening`, and where there is no death, the
  `burial` or the `cremation`.
- A `death`, a `burial` or a `cremation` makes the person dead for the rule that
  decides who is living, whatever its date. [Private records](./private#living)
  describes the rule.
- `origins check` compares the `birth` and the `death` of each person, and of their
  birth parents. In a family, it checks that the `engagement`, the `marriage-banns` and
  the `marriage-licence` come before the `marriage`, and that a `separation`, a
  `divorce-filed`, a `divorce` or an `annulment` comes after it, and that no
  `separation` or `divorce` comes after a partner's death. Each of these is an error
  when the dates leave no doubt.

## Age and cause

`age` is the age that the record gives the person the event is about. It is a number of
years, or years, months, weeks and days, each a number followed by `y`, `m`, `w` or
`d`, in that order. A `<` or a `>` before it says that the person was younger or older
than that:

```
67          67 years
67y         the same
3m 12d      3 months and 12 days
2w          2 weeks
< 1y        under a year old
> 80y       over 80
```

Origins keeps the age as the record gives it, and doesn't compare it with the person's
birth. `cause` is the cause that the record gives, as text. Either can be a mapping
with `value`, a `note` and `sources`:

```yaml
age:
  value: 66
  note: Given by Michael, who registered the death.
```

## Address

`address` is where in the place the event happened, such as the house of a residence
or the church of a wedding. It is the address as text, or a mapping with the parts that
[Repositories](./repository#addresses) describes:

```yaml
place: london
address: 14 Fournier Street, Spitalfields
```

## Sort date

`sort-date` is a date that places the event in order among the others, when its `date`
doesn't say or there is none. It says nothing about when the event happened. Use it,
for example, for a residence that you know came before a dated move, or a military
service you can only place between two other events:

```yaml
  - type: residence
    place: stade
    sort-date: 1700
```

## Family

`family` on a person's event is the family that the record ties the event to, such as
the family that a `lds-sealing-to-parents` seals the person to, or the family that an
`adoption` took the person into. It doesn't make anybody a parent. Origins reads parents only from the
families' `children`.

## People named in an event

`with` lists the people that the record names besides the person the event is about:
godparents, witnesses, an informant. Each entry has a `role` and one of these:

- `person`, the slug of somebody with a record
- `name`, for somebody without a record, as a string or in the parts that
  [People](./person#names) describes
- `person: unknown`, for somebody the record says was there but doesn't name

The `role` is a word of your own, as the record puts it. The importers write
GEDCOM's and Gramps' roles as `bride`, `groom`, `husband`, `wife`, `spouse`, `father`,
`mother`, `parent`, `child`, `godparent`, `witness`, `informant`, `clergy`,
`officiator`, `celebrant`, `aide`, `friend` and `other`. A word like
`godmother` says more than `godparent`, so use it when the record does.

An entry can also give the `age` the record gives them, `attributes`, a `note` and
`sources`. The page of each person named shows their part in the event, such as
"godfather at the baptism of Jean Baptiste Verdier, 1834".

```yaml
events:
  - type: marriage
    date: 21 APR 1858
    place: london
    with:
      - person: verdier-jean-baptiste
        role: groom
        age: 24
      - person: papadopoulou-maria
        role: bride
        age: 18
      - name: Mary Ann Collins
        role: witness
```

## Attributes

`attributes` keeps what a record says of an event that no field holds: the agency that
registered it, a father's age at a birth, a column of a census form. Each entry has a
`type`, a `value` as the record gave it, and optionally a `date`, a `note` and
`sources`. Origins doesn't work anything out from an attribute. Pauline's
`fathers-age` above is what the register says, and it doesn't make anybody her father.

People named in an event, sources, citations and pictures take `attributes` too.

## Rites of the Latter-day Saints

The ordinances of the Church of Jesus Christ of Latter-day Saints are events of the
person, such as `lds-baptism`, `lds-endowment` and `lds-sealing-to-parents`, or of the
family, `lds-sealing-to-spouse`. The temple is a `temple` attribute and the status a
`status` attribute. A sealing to parents names the family in `family`:

```yaml
events:
  - type: lds-sealing-to-parents
    date: 14 MAR 1996
    family: papadopoulou-maria--verdier-jean-baptiste
    attributes:
      - type: temple
        value: SLAKE
      - type: status
        value: completed
        date: 20 MAR 1996
```

## Events that didn't happen

A finding that something didn't happen, such as a woman who never married, is evidence
too. A person and a family take `not`, a list of entries each with the event's `type`,
the `date` over which it didn't happen, or none for never, and a `note` and `sources`:

```corpus people/chen-yuying/README.md
---
name:
  given: 玉英
  surname: 陳
  lang: zh-Hant
  display: '{surname}{given}'
sex: F
events:
  - type: birth
    date: 1865
    place: wenzhou
  - type: death
    date: 1932
    place: wenzhou
not:
  - type: marriage
    note: Unmarried on her death record, at sixty-seven.
---
```

Origins keeps `not` apart from the events, so it never takes one for something that
happened. `origins check` reports an error if an event of that type falls within the
date of a `not` entry, and a warning if it might.
