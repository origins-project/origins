# Repositories

A repository record describes an archive, a library, a parish office or a website that
keeps sources. Each source names the repositories it is kept in, so an archive is
described once, however many sources it keeps. Write what only a visitor needs, such as
opening hours and how to order copies, in the body.

```corpus repositories/archivio-diocesano-di-firenze/README.md
---
name: Archivio Diocesano di Firenze
type: archive
addresses:
  - lines: |
      Piazza San Giovanni 3
      50129 Firenze
    street: Piazza San Giovanni 3
    city: Firenze
    postal: 50129
    country: Italia
    links:
      - tel:055 213456
      - url: tel:055 213457
        type: fax
  - lines: |
      Via dei Servi 15
      50122 Firenze
    date: TO 1998
links:
  - https://www.example.org/archivio-diocesano
  - mailto:archivio@example.org
---

Open Tuesday and Thursday mornings. Registers are ordered a day ahead, and
photographs without flash are allowed.
```

## Fields

| Field | Holds |
|---|---|
| `name` | the repository's name |
| `type` | the kind of repository, such as `archive`, `library`, or a word of your own |
| `addresses` | where it is, or was |

A repository also takes `media`, `ids`, `links`, `tags`, `same-as` and `private`, as
[most records](./#fields-that-most-records-take) do. It takes no `sources`.

A telephone number is a link `tel:` followed by the number, an email address is
`mailto:` followed by the address, and a fax number is a `tel:` link with the type
`fax`.

## Addresses

An address is used by a repository, a place, a researcher in the manifest, and an
event's `address`. It is the address as lines of text, its parts, or both:

| Field | Holds |
|---|---|
| `lines` | the address as you would write it on an envelope |
| `street`, `locality`, `city`, `parish`, `county`, `state`, `postal`, `country` | the parts |
| `links` | the telephone numbers and email addresses that go with the address |
| `date` | when it was the address |
| `note`, `sources` | as on any mapping |

Write both the lines and the parts when you know both. The lines don't say which one is
the city, and the parts don't say how the lines were set out.
