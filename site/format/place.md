# Places

A place record holds a place's names, what it lies within, and where it is on a map.
Every place has its own record in `places/`, and the hierarchy is in the records'
`within` fields.

```corpus places/firenze/README.md
---
name: Firenze
also-known-as:
  - name: Florence
    lang: en
  - name: Florenz
    lang: de
    note: As the German guidebooks of the period have it.
type: city
within:
  - place: granducato-di-toscana
    until: 1860
    note: The register heads each volume Granducato di Toscana.
    sources: [parish-san-lorenzo-baptisms#1779-042]
  - place: toscana
    from: 1861
coordinates: 43.7696, 11.2558
ids:
  geonames: 3176959
  wikidata: Q2044
---
```

## Fields

| Field | Holds |
|---|---|
| `name` | the name the place goes by where it is |
| `also-known-as` | its other names |
| `type` | the kind of place, such as `city`, `comune`, `parish`, or a word of your own |
| `within` | the place it lies within, or a list with dates when that changed |
| `coordinates` | latitude and longitude in decimal degrees, north and east positive |
| `addresses` | addresses for the place, as [Repositories](./repository#addresses) describes |
| `from`, `until` | when the place existed |

A place also takes the fields that [most records take](./#fields-that-most-records-take).

`type` is a word of your own, in the language of the place if you like, such as
`comune`, `frazione` or `Landkreis`. The Gramps importer writes Gramps' types as
`country`, `state`, `region`, `province`, `county`, `department`, `district`,
`municipality`, `borough`, `city`, `town`, `village`, `hamlet`, `parish`, `locality`,
`neighborhood`, `street`, `farm`, `building` and `number`. A GEDCOM import takes the
types from the file's `FORM`, in the file's words.

## Names

`name` and each entry of `also-known-as` is a string, or a mapping with `name` and
these fields:

| Field | Holds |
|---|---|
| `lang` | the language of the name, as a tag such as `en`, `de` or `la` |
| `type` | a word for a spelling that the language doesn't explain, such as `pinyin` |
| `from`, `until` | when the place was called that |
| `note`, `sources` | as on any mapping |

A town that was renamed keeps each name with its dates:

```yaml
name:
  name: Ivano-Frankivsk
  from: 1962
also-known-as:
  - name: Stanislav
    from: 1939
    until: 1962
  - name: Stanisławów
    lang: pl
    until: 1939
```

## Within

`within` is the slug of one place, or a list of entries each with `place`, `from`,
`until`, `note` and `sources`, as Firenze's is above. Write a list when the place
belonged to different jurisdictions at different times.

`from` and `until` at the top of the record say when the place existed at all, for a
hamlet that was absorbed into its neighbour or a state that was annexed. An event from
before then still refers to the place as it was.

## Coordinates

Write the coordinates as two numbers separated by a comma, `43.7696, 11.2558`. If a
record gives coordinates that aren't in that form, write its words in parentheses,
such as `(N43.77.5, E11.25.3)`. The page shows them, and the map leaves the place off.
