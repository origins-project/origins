# History

Origins records every change to a corpus as one git commit. The commit message says
what changed in words, and its trailers say it again for Origins to read: the
operation, the records it touched, the event, and the citations the change rests on.
The commands on this page read that history back. They continue the corpus of
[Getting started](../getting-started), after a source has been added and cited.

## The changes to a record

`origins log` lists the changes, the newest first. Name a record to see only the
changes to it, and use `-n` to see only the newest:

```console
$ origins log -n 2 person/rossi-giovanni
207094de86  2026-10-19  Add to events on person/rossi-giovanni
            events.add  person/rossi-giovanni  (Rosa Bianchi)
            touching baptism, from battesimi-borgo-san-lorenzo#1842-057
6bb9250c02  2026-10-19  Add to events[0].sources on person/rossi-giovanni
            events.add  person/rossi-giovanni  (Rosa Bianchi)
            touching birth
```

`origins blame` lists the facts of a record, each with the commit that recorded it:

```console
$ origins blame person/rossi-giovanni
ff8895fea7  2026-10-19  name
ff8895fea7  2026-10-19  name.given
ff8895fea7  2026-10-19  name.surname
ff8895fea7  2026-10-19  sex
9cc206ae7f  2026-10-19  events
9cc206ae7f  2026-10-19  events[0].type
ffac3388ac  2026-10-19  events[0].date
9cc206ae7f  2026-10-19  events[0].place
6bb9250c02  2026-10-19  events[0].sources
...
```

`origins why` shows one fact, the commit that recorded it and, if that commit accepted
a hypothesis, the hypothesis with its argument:

```console
$ origins why person/rossi-giovanni events[0].date
person/rossi-giovanni events[0].date
  date: 12 MAR 1842

ffac3388ac  2026-10-19  Set events[0].date on person/rossi-giovanni
            events.set  person/rossi-giovanni  (Rosa Bianchi)
            touching birth
```

## The changes that rest on a source

Each command that changes the corpus takes `--source`, which names the citation the
change rests on. Origins records it in the commit:

```console
$ origins add --source battesimi-borgo-san-lorenzo#1842-057 person/rossi-giovanni events '{type: baptism, date: 13 MAR 1842, place: borgo-san-lorenzo, sources: [battesimi-borgo-san-lorenzo#1842-057]}'
207094de86 Add to events on person/rossi-giovanni
  people/rossi-giovanni/README.md
```

If you later find that a source is unreliable, `origins whence` lists every change
that rests on it, and every hypothesis that cites it:

```console
$ origins whence battesimi-borgo-san-lorenzo
207094de86  2026-10-19  Add to events on person/rossi-giovanni
            events.add  person/rossi-giovanni  (Rosa Bianchi)
            touching baptism, from battesimi-borgo-san-lorenzo#1842-057
```

## Taking a change back

`origins undo` takes back the last change in a new commit, so the history keeps both.
With `--dry-run` it says what it would take back:

```console
$ origins undo --dry-run
207094de86  2026-10-19  Add to events on person/rossi-giovanni
            events.add  person/rossi-giovanni  (Rosa Bianchi)
            touching baptism, from battesimi-borgo-san-lorenzo#1842-057
would be taken back
  people/rossi-giovanni/README.md
```

## Naming a state

`origins snapshot` gives the current state a name, as a git tag. Use it before you send
the tree to somebody, so that you can see later what they received:

```console
$ origins snapshot -m 'the tree as sent to zia Maria' sent-to-zia-maria
207094de86  2026-10-19  sent-to-zia-maria  the tree as sent to zia Maria
```

## Commits that Origins didn't write

The corpus is an ordinary git repository, and you can commit to it with git. Origins
can't tell what such a commit changed, so every command that reads or writes the
history stops when it finds one, until you run `origins recover`. Without a flag,
`recover` says what it found. `--adopt` keeps the commits and records that Origins has
read them, `--rewrite` replaces them with commits that describe their changes, and
`--reset` removes them and leaves their changes in the working tree.

To record an edit that you made in a text editor, run `origins adopt` instead of
`git commit`.
