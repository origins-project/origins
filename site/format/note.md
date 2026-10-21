# Notes

A note record holds text that isn't about one record in particular, such as a research
log, a list of things to do or a letter to an archive. Text about one record goes in
that record's body, or in the `note` of the entry it is about.

The slug of a note is made from its title, and the body is the note itself:

```corpus notes/a-london-wedding-of-1859/README.md
---
title: A London wedding of 1859
events:
  - type: marriage
    date: 3 OCT 1859
    place: london
    with:
      - person: lang-anna-maria
        role: witness
      - name:
          given: Jan
          surname: Kowalski
        role: groom
      - name:
          given: Bridget
          surname: Murphy
        role: bride
---

*Research:* Anna Maria Ershov witnessed this wedding, and nobody else in it is anybody
the tree has yet. The bride was one of the girls who sewed for her, which is the only
reason to keep it.

- [ ] Ask St Leonard's, Shoreditch, for the banns of September 1859.
```

## Fields

| Field | Holds |
|---|---|
| `title` | the note's title |
| `events` | events about nobody the corpus has a record of, as [Events](./events) describes |

A note also takes the fields that [most records take](./#fields-that-most-records-take).

An event in a note names the people in it in `with`, as any event does. The page of
Anna Maria Ershov shows the wedding above, with her role in it. When one of the other
people gets a record of their own, you can move the event to that person.

## Notes from an import

GEDCOM and Gramps keep notes that several records share. `origins import` writes each
of them as a note record, and writes a link to the note where its text was, so that two
people who shared a note share the one record:

```markdown
[Born in Vicchio, per the family letters.](../../notes/born-in-vicchio-per-the-family-letters/README.md)
```

The link's text is the first line of the note.
