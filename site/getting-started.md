# Getting started

This page starts a corpus for an invented family, the Rossi of Borgo San Lorenzo, and
adds three people, a place and a family to it. Each command is shown with what Origins
prints. Your commit hashes and dates will differ from the ones shown here.

You need `origins` and git on your `PATH`, as [Install](./install) describes.

## Starting a corpus

`origins init` creates a directory with `origins.yaml` in it and records that file as
the first commit. If the directory isn't inside a git repository, Origins creates one:

```console
$ origins init --name 'Rossi family' rossi
edf17909cd  Start the Rossi family corpus
  recorded in a new git repository
  .gitattributes
  origins.yaml
$ cd rossi
```

The rest of this page runs in the `rossi` directory. A command that reads a corpus
uses the working directory unless you name another one with `-C`.

## Adding a person

`origins create` makes a record. You name the record as its kind and its slug, and
give the first values as `path=value`:

```console
$ origins create person/rossi-giovanni name.given=Giovanni name.surname=Rossi sex=M
ff8895fea7 Create person/rossi-giovanni
  people/rossi-giovanni/README.md
$ cat people/rossi-giovanni/README.md
---
name:
  given: Giovanni
  surname: Rossi
sex: M
---
```

The slug of a person is the surname and then the given names, in lowercase and joined
with hyphens. Origins uses the slug as the name of the record's directory, and other
records refer to Giovanni by it.

## Adding a place and an event

A birth needs a place to happen in. Create the place, then add the birth to the list
of Giovanni's events. The entry you add is written in YAML:

```console
$ origins create place/borgo-san-lorenzo name='Borgo San Lorenzo' type=comune
03eb4ab332 Create place/borgo-san-lorenzo
  places/borgo-san-lorenzo/README.md
$ origins add person/rossi-giovanni events '{type: birth, date: ABT 1842, place: borgo-san-lorenzo}'
9cc206ae7f Add to events on person/rossi-giovanni
  people/rossi-giovanni/README.md
```

`ABT 1842` means "about 1842". If you later find the baptism register, `origins set`
changes the one value. `events[0].date` is the date of the first event in the list:

```console
$ origins set person/rossi-giovanni events[0].date '12 MAR 1842'
ffac3388ac Set events[0].date on person/rossi-giovanni
  people/rossi-giovanni/README.md
$ cat people/rossi-giovanni/README.md
---
name:
  given: Giovanni
  surname: Rossi
sex: M
events:
  - type: birth
    date: 12 MAR 1842
    place: borgo-san-lorenzo
---
```

## Adding a family

Parents and children are recorded in a family, and nowhere else. Create Giovanni's
wife, the family of the two of them, and their daughter, and then add the daughter to
the family's children:

```console
$ origins create person/bianchi-maria name.given=Maria name.surname=Bianchi sex=F
5671255955 Create person/bianchi-maria
  people/bianchi-maria/README.md
$ origins create family/bianchi-maria--rossi-giovanni partners='[bianchi-maria, rossi-giovanni]' type=married
82f86ecc48 Create family/bianchi-maria--rossi-giovanni
  families/bianchi-maria--rossi-giovanni/README.md
$ origins create person/rossi-annunziata name.given=Annunziata name.surname=Rossi sex=F
4caa81f573 Create person/rossi-annunziata
  people/rossi-annunziata/README.md
$ origins add family/bianchi-maria--rossi-giovanni children rossi-annunziata
3df03179fb Add to children on family/bianchi-maria--rossi-giovanni
  families/bianchi-maria--rossi-giovanni/README.md
```

The slug of a family is the slugs of its partners in alphabetical order, joined with
two hyphens. Origins now works out from the family that Annunziata is Giovanni's
daughter:

```console
$ origins kinship rossi-annunziata rossi-giovanni
Annunziata Rossi is the daughter of Giovanni Rossi

  rossi-annunziata  Annunziata Rossi
  rossi-giovanni    Giovanni Rossi    her father
```

## Checking the corpus

`origins check` lists the errors and warnings in the corpus. An error is a
contradiction between records. A warning points to research that you still have to
do:

```console
$ origins check
warning: person/rossi-giovanni events[0]: birth cites nothing (unsourced-event)
0 error(s), 1 warning(s)
```

Origins also checks every change before it records it. A change that would put a
contradiction into the corpus is refused, and nothing is written:

```console
$ origins add person/rossi-annunziata events '{type: birth, date: 1838}'
the change would make the corpus contradict itself:
  person/rossi-annunziata events[0].date: born 1838, before rossi-giovanni was born in 12 MAR 1842 (child-before-parent)
```

[Findings](./reference/check) lists every error and warning, and what each one means.

## Opening the corpus in a browser

`origins browse` serves the corpus at `http://127.0.0.1:8100` and opens it in your
browser. You can read and edit every record there, and Origins makes a commit for each
change, as the commands do. Press Ctrl+C in the terminal to stop it.

```bash
origins browse
```

[Working in the browser](./guides/browse) describes the pages.

## Editing the files

You can also edit a record in a text editor. Open
`people/rossi-annunziata/README.md` and change it to this:

```corpus people/rossi-annunziata/README.md
---
name:
  given: Annunziata
  surname: Rossi
sex: F
events:
  - type: birth
    date: 25 MAR 1866
    place: borgo-san-lorenzo
---

Her name and her birthday are the feast of the Annunciation, which is how the family
remembered the date.
```

The part between the two `---` lines is the frontmatter, which Origins reads. The text
below it is yours, for notes and for the reasons behind a conclusion.

Origins doesn't record an edit until you tell it to. `origins adopt` reads the files
you changed, checks them in the same way as a command's change, writes them in the
form Origins writes, and records them as a commit:

```console
$ origins adopt
575572374c  Adopt people/rossi-annunziata/README.md, edited by hand
  people/rossi-annunziata/README.md
$ origins log person/rossi-annunziata
575572374c  2026-10-19  Adopt people/rossi-annunziata/README.md, edited by hand
            manual.adopt  person/rossi-annunziata  (Rosa Bianchi)
4caa81f573  2026-10-19  Create person/rossi-annunziata
            person.create  person/rossi-annunziata  (Rosa Bianchi)
```

## Next steps

`origins todo` lists the research that the corpus asks for:

```console
$ origins todo
person/rossi-annunziata: birth 25 MAR 1866 is more than 110 years ago, and no death is recorded (unrecorded-death)
person/rossi-annunziata events[0]: birth cites nothing (unsourced-event)
person/rossi-giovanni: birth 12 MAR 1842 is more than 110 years ago, and no death is recorded (unrecorded-death)
person/rossi-giovanni events[0]: birth cites nothing (unsourced-event)
4 things to do
```

To cite a register for these births, you need a source record with a citation in it,
as [Sources and citations](./format/source) describes. If you already keep your tree
in another program, you can start from its export instead, as
[Importing from GEDCOM](./guides/import-gedcom) and
[Importing from Gramps](./guides/import-gramps) describe.
