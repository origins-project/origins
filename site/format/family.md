# Families

A family record holds the people a child was born to or raised by, the partnership
between them, and the children. The marriage, a divorce and the sources for them are
in the family, and so is the list of whose children are whose. Origins works out every
person's parents, partners and children from the families, and from nowhere else.

```corpus families/ershov-ivan--lang-anna-maria/README.md
---
partners: [ershov-ivan, lang-anna-maria]
type: married
children:
  - ershov-michael
  - id: ershov-lawrence
    status: disproven
    note: An index of births read the household's surname for his parentage.
events:
  - type: marriage
    date: 7 MAY 1853
    place: london
---
```

## Fields

| Field | Holds |
|---|---|
| `partners` | the partners, at least two |
| `type` | the kind of partnership: `married`, `civil-union`, `unmarried` or a word of your own |
| `children` | the children, in the order the record gives them |
| `events` | the couple's events, such as the marriage, as [Events](./events) describes |
| `not` | events that the sources show didn't happen |

A family also takes the fields that [most records take](./#fields-that-most-records-take).

Leave `type` out when nothing says what kind of partnership there was. A couple who
married twice, or a person who married twice, are two families, each with its own
events and children.

## Partners

Each partner is a person's slug or one of two words:

- `unknown` means that there was somebody, and nobody knows who, such as the father of
  a child baptised as illegitimate. When you find them, put their slug in place of
  `unknown`, and Origins renames the family to match. A family of `[unknown, unknown]`
  holds children who are known to be siblings.
- `none` means that there was nobody, such as a woman who adopted a child alone. It
  stands beside exactly one person, and the family has no birth children.

```corpus families/chen-yuying--none/README.md
---
partners: [chen-yuying, none]
children:
  - name: 陳玉珍
    relation: adoptive
---
```

A partner can be a mapping: `person` and a `note` about this family, or `name` in
place of `person` for a partner the record names and who has no record of their own.

## Children

A child is a person's slug when they are the birth child of every partner. Otherwise,
or when there is more to say, the child is a mapping with `id` and these fields:

| Field | Holds |
|---|---|
| `relation` | the child's relation to the partners, one word for all of them or a mapping from each partner to a word |
| `status` | `proven`, `challenged` or `disproven` |
| `note`, `sources` | as on any mapping |

The relations that Origins works kinship out from are `birth`, `adoptive`, `step`,
`foster` and `guardian`. `unknown` means that the record makes somebody a parent
without saying in what sense, and a word of your own counts as `unknown`. `none` says
that the child is listed in the family but isn't a child of that partner, such as a
wife's son from before the marriage.

A disproven link stays in the family with the sources that disproved it, but Origins
doesn't treat it as parentage. `origins check` warns about it, because you usually
remove such a link once you have noted why.

A child can be `unknown`, for a child the record mentions without a name, or a `name`
in place of `id` for a child the record names and who has no record:

```yaml
children:
  - verdier-anna
  - verdier-pauline
  - id: unknown
    note: Jean Baptiste's death record counts four children of the marriage, and three are known.
```

A person can be a child of several families: the one they were born into, and the one
that adopted them. Across all of their families, a person has at most two birth
parents, and `origins check` reports a person who would have more.
