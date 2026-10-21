# Hypotheses

The records in a corpus hold your current conclusions. When you think that something
might be different, but you aren't sure, you write it as a hypothesis. A hypothesis is
a list of changes to the corpus, pinned to the commit it was written against, with the
evidence and your argument for it. Origins can check the corpus as if the hypothesis
were true, without changing anything, and apply the changes when you decide.

This guide continues the corpus of [Getting started](../getting-started).
[Hypotheses](../format/hypothesis) in the format pages describes the file.

## Proposing a hypothesis

You don't write the list of changes by hand. Edit the records as if the hypothesis were
true, and run `origins hypothesis propose`. Here, `people/bianchi-maria/README.md`
gets a birth about 1845 in Borgo San Lorenzo:

```markdown
---
name:
  given: Maria
  surname: Bianchi
sex: F
events:
  - type: birth
    date: ABT 1845
    place: borgo-san-lorenzo
---
```

Origins writes your edits as the hypothesis's changes and puts the record back as it
was:

```console
$ origins hypothesis propose maria-born-1845
48bb9efb5f  Propose maria-born-1845
  pinned to 207094de86
  add person/bianchi-maria events
  the edited files are back as 207094de86 had them
```

Then open `hypotheses/maria-born-1845/README.md`, write the question as its heading and
your argument below it, and record the edit with `origins adopt`:

```markdown
---
status: open
base: 207094de8615e4e5c224a7d2761675373ef51abb
confidence: possible
edits:
  - op: add
    entity: person/bianchi-maria
    path: events
    value:
      date: ABT 1845
      place: borgo-san-lorenzo
      type: birth
---

# Was Maria Bianchi born in Borgo San Lorenzo about 1845?

The family says that Maria came from the same parish as Giovanni, and she was about
twenty when Annunziata was born. The baptism register of 1845 has not been searched.

**What would settle it:** her baptism in the register of 1844 to 1846.
```

## Testing a hypothesis

`origins check --with` checks the corpus as if the hypothesis were true. Origins
applies the changes at the commit the hypothesis is pinned to, and carries them to the
current corpus with git's three-way merge. Nothing is written to the corpus:

```console
$ origins check --with maria-born-1845
maria-born-1845 read at 207094de8615e4e5c224a7d2761675373ef51abb
warning: hypothesis/maria-born-1845 supports: cites no record on either side (unevidenced-hypothesis)
warning: person/bianchi-maria events[0]: birth cites nothing (unsourced-event)
0 error(s), 2 warning(s)
```

Give `--with` more than once to ask whether two hypotheses can be true together. The
questions about relationships, `ancestors`, `descendants` and `kinship`, take `--with`
too, and so does `origins browse`, which then shows the tree as if the hypothesis were
true.

In the browser, the page of a hypothesis has a *Suppose* button that reads the whole
tab with the hypothesis taken as true. A bar under the top bar lists the hypotheses
that the tab supposes, each with a link to its page and a *Stop supposing* button.
While the tab supposes a hypothesis, you can't change the corpus from it. If you
started `origins browse` with `--with`, the bar lists that hypothesis too, without the
button.

In the browser, you can also write a hypothesis without editing any file. Turn a tab
to Suppose, and each change you make there goes into a draft instead of into the
corpus. The pages show the tree with the draft applied. When you're done, you keep the
draft as a hypothesis, with its question and its argument.

## Accepting or rejecting

When you have decided, accept the hypothesis or reject it:

```console
$ origins hypothesis accept maria-born-1845
811af9a740  Accept maria-born-1845
  carried out at 207094de86, where it was written
  hypotheses/maria-born-1845/README.md
  people/bianchi-maria/README.md
```

Accepting applies the changes and marks the hypothesis `accepted`, in one commit.
`origins hypothesis reject` changes no record. It marks the hypothesis `rejected`,
and the file stays in the corpus with your argument, for whoever finds the same
evidence later.

`origins why` follows a fact back to the hypothesis that put it there:

```console
$ origins why person/bianchi-maria events[0].date
person/bianchi-maria events[0].date
  date: ABT 1845

811af9a740  2026-10-19  Accept maria-born-1845
            hypothesis.accept  person/bianchi-maria  hypothesis/maria-born-1845  (Rosa Bianchi)
            touching birth, by maria-born-1845

hypothesis/maria-born-1845  accepted, possible

# Was Maria Bianchi born in Borgo San Lorenzo about 1845?
...
```

`origins hypothesis status` lists every hypothesis, and checks that the corpus still
holds what each accepted one says:

```console
$ origins hypothesis status
maria-born-1845  accepted, and the corpus still holds it
```

## When the corpus has moved on

A hypothesis refers to entries by their position, such as `events[0]`. If somebody
adds an event before the one the hypothesis changes, the positions in the current
corpus no longer match. This is why Origins applies the changes at the pinned commit
and then merges. If a later change touched the same lines, `check --with` and
`accept` report a conflict that names the file, and they change nothing. If a record
was renamed since, Origins follows the rename.
