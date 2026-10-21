# Privacy

A corpus usually holds things that you don't want to show to everybody: living people,
an address, a photograph taken at home. Origins always reads the whole corpus when you
work on it. When you show the tree to somebody else or export it, you can ask for a
hidden view, which leaves those things out.

## Marking what is private

`private: true` marks a record, or an entry inside one, as not for sharing. A person, a
family, a place, a source, a repository and a note take it, and so do an event, a name
in `also-known-as` and a media entry. Origins writes it first in the mapping, so the
first line of a record tells you whether it is meant to leave your computer:

```corpus people/marchena-marco/README.md
---
name:
  given: Marco
  surname:
    - name: Marchena
      origin: patrilineal
    - name: Chen
      origin: matrilineal
sex: M
events:
  - type: birth
    date: 14 MAR 1975 09:20
    place: madrid
  - type: residence
    date: FROM 2010
    place: kinkerstraat-41
media:
  - private: true
    file: photos/2019-at-home.jpg
    title: At home, with Stefán and Sasha
    date: 2019
    depicts:
      - marchena-marco
      - solveigarson-stefan
      - marchena-solveigarson-sasha
---
```

Here the photograph is private, and so is the place `kinkerstraat-41`, his address,
in its own record. You set and unset the flag like any other field:

```bash
origins set person/marchena-marco events[1].private true
origins unset person/marchena-marco events[1].private
```

## Living people

Origins takes a person as living unless one of these is true:

- A death, a burial or a cremation is among their events.
- An event of theirs, or of a family they are a partner in, is certainly more than 110
  years ago.

A person with no dates at all is taken as living. `living: false` on a person says
that they are dead when nothing in the record shows it, and `living: true` says that
they are alive when the rule would take them as dead.

## The hidden view

`--hide-private` leaves out what is private, `--hide-living` leaves out the people
taken as living, and `--public` does both. `origins browse` and the three kinds of
`origins export` take them:

```bash
origins browse --public
origins export html --public --out rossi-pages
origins export gedcom --hide-living --out rossi.gdz
```

`origins browse` with one of these flags serves the corpus read-only. Its menu shows
*Hide private* and *Hide living* ticked as the flags set them, and you can't switch
them off.

In a hidden view:

- A hidden person becomes a placeholder named Private, with a slug such as
  `private-1`. The placeholder keeps its place in its families and loses everything
  else: the other names, the events, the notes, the sources and the pictures.
- A family with a hidden partner keeps its partners and children and loses its events
  and its notes.
- A private event, name or picture is removed from its record. A picture is also
  removed if it shows a hidden person.
- A private place, source, repository or note is removed, and so is every reference to
  it. A place or a source that only hidden things referred to is removed too.
- Hypotheses are left out.

Origins doesn't read your prose. If the body of a record names a living person, the
hidden view shows it as you wrote it. Mark that record private if its text isn't for
sharing.
