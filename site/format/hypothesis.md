# Hypotheses

A hypothesis record proposes a change to the corpus as a list of edits, with the
argument for it in the body. [Hypotheses](../guides/hypotheses) in the guides shows how
you propose, test, accept and reject one.

```corpus hypotheses/louisa-born-1844/README.md
---
status: rejected
base: 8f78553
confidence: possible
supports: [census-1861-england#f214 OSI]
contradicts: [french-church-london-baptisms#1842-007]
edits:
  - op: set
    entity: person/verdier-louisa
    path: events[0].date
    value: 1844
---

# Was Louisa born in 1844 rather than 1842?

The French church has her baptism on 19 March 1842. The 1861 census gives her age
as 17, which would put her birth in 1843 or 1844. Both cannot be right.

I rejected this. The register was written at the baptism by the minister who
performed it, and the census ages in this return are wrong by a year or two
throughout.

**What would reopen it:** a death record giving her age, or a marriage record naming
her age. Either would be independent of both existing sources.
```

## Fields

| Field | Holds |
|---|---|
| `status` | `open`, `accepted` or `rejected` |
| `base` | the commit that the edits apply to |
| `confidence` | `probable`, `possible` or `doubtful` |
| `proposed-by` | who proposed it, when somebody else did, such as an AI agent |
| `supports`, `contradicts` | the citations for and against it |
| `edits` | the changes it proposes |

Write the question as the heading of the body and the argument under it.

The corpus always holds the conclusion you have reached. A hypothesis proposes
something different, and the conclusion you already hold has no file. When you reject
a hypothesis, you keep the file so that the argument stays in the corpus.

`origins whence` finds a hypothesis through its `supports` and `contradicts`, as it
finds a fact through its `sources`. When you doubt a source, it shows you the arguments
that rest on it too.

## Edits

Each edit has an `op` and the fields that the operation needs. `entity` is a kind and
a slug, such as `person/verdier-louisa`, or `project` for the manifest. `path` is a
field in the record, such as `events[0].date` or `also-known-as[1].surname`, with the
entries of a list counted from 0.

| op | Fields | Does |
|---|---|---|
| `set` | `entity`, `path`, `value` | sets the field to the value |
| `unset` | `entity`, `path` | removes the field |
| `add` | `entity`, `path`, `value` | adds the value to the end of the list |
| `remove` | `entity`, `path`, `value` | takes the value out of the list |
| `create` | `entity`, `value` | makes a record with the fields in `value` |
| `delete` | `entity` | deletes the record |
| `rename` | `entity`, `to` | gives the record a new slug, and rewrites every reference to it |
| `merge` | `from`, `into` | makes two records one, and rewrites every reference to `from` |
| `split` | `entity`, `into` | makes new records beside one that describes several people |

These are the commands of the same names, and
[the commands' reference](../reference/commands) describes what each one does.

```yaml
edits:
  - op: merge
    from: person/ershov-ivan-1885
    into: person/ershov-ivan
```

`rename` also takes a tag as `tag/<word>`, with the new word in `to`.

## The base

Paths count positions, and the corpus changes after you write a hypothesis: an event
added before the death makes `events[1]` a different event. So Origins applies the
edits to the corpus as it was at `base`, and then merges the result into the corpus as
it is now, as git merges a branch. If the same lines changed in between, Origins
reports a conflict and names the file.
