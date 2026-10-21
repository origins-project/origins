# The manifest

`origins.yaml` at the top of the corpus says whose tree it is and who keeps it. It is
the one file that isn't a record.

```corpus origins.yaml
project:
  name: Marchena Sólveigarson family
  description: Descendants of Per Andersson and of Bartolomeo Rossi
  main-person: verdier-jean-baptiste
  lang: en
  copyright: © 2024 Ingrid de Vries
  researchers:
    - name: Ingrid de Vries
      addresses:
        - |-
          Lindengracht 112
          1015 KM Amsterdam
      links: [mailto:ingrid@example.org]
    - name: Tomasz Nowak
      note: >-
        Compiled the Papadopoulou branch, 2015–2019, from the registers of
        Trieste.
  bookmarks:
    - people/verdier-jean-baptiste
    - sources/parish-san-lorenzo-baptisms
    - notes/a-london-wedding-of-1859
  tags:
    Emigrants:
      color: '#2a7'
    To do:
      color: '#d33'
      priority: 1
origins:
  format-version: 1
```

## Fields

| Field | Holds |
|---|---|
| `name`, `description` | the tree's name and what it is about |
| `main-person` | the person whose tree it is, where the pages start |
| `lang` | the language of the corpus's text, where a record doesn't say |
| `copyright` | the tree's copyright, as you write it |
| `researchers` | the people who compiled it, each with a `name`, `addresses`, `links` and a `note` |
| `bookmarks` | records you keep to hand, each as its directory and its slug |
| `tags` | a colour and a priority for each tag that has one |

A researcher's address is as [Repositories](./repository#addresses) describes, without
`sources`.

`origins check` reports a bookmark to a record that doesn't exist. `origins rename`,
`merge` and `delete` change the bookmarks of the records they change.

## Tags

Every record, event, citation and media entry takes `tags`, a list of your own words,
such as `To do` or `Emigrants`. The pages list what carries each tag. A tag needs no
entry under `tags` here. An entry gives it the `color` and `priority` that the pages
show it with.

Origins compares tags as you wrote them, so `To do` and `to do` are two tags.
`origins rename` renames a tag everywhere it is used:

```bash
origins rename 'tag/To do' Done
```

Renaming a tag to one that already exists makes them one tag. If both tags have an entry
here and the entries differ, Origins refuses and shows you both.

## Changing the manifest

You change the manifest with the same commands as a record, addressing it as `project`:

```bash
origins set project main-person verdier-jean-baptiste
origins add project researchers '{name: Tomasz Nowak}'
origins log project
```

## format-version

`format-version` is the version of the corpus format, so that a version of Origins that
changes the format can tell which format a corpus is in. If a corpus has a later format
version than your `origins` knows, `origins` refuses to read it and tells you which
version it found.
