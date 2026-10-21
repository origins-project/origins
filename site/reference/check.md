# Findings

`origins check` reports what in a corpus contradicts itself, and `origins todo` lists
the research that is still to do. Each finding names a rule, the record, and the field
it is about:

```console
$ origins check
error: person/rossi-annunziata events[0].date: born 25 MAR 1866, before rossi-giovanni was born in 12 MAR 1872 (child-before-parent)
warning: person/rossi-giovanni events[1]: occupation cites nothing (unsourced-event)
1 error(s), 1 warning(s)
```

`check` exits 1 if it finds an error, and 0 if it finds only warnings or nothing. The
pages show the same findings on the Findings page and on the page of each record.

A command, or a change in the pages, that would add an error is refused, and writes
nothing:

```console
$ origins add person/rossi-annunziata events '{type: birth, date: 25 MAR 1866}'
the change would make the corpus contradict itself:
  person/rossi-annunziata events[0].date: born 25 MAR 1866, before rossi-giovanni was born in 12 MAR 1872 (child-before-parent)
```

So errors come from files edited by hand, from an import of another program's tree,
and from a merge with git.

## Errors

An error is something that can't be true, or a reference that leads nowhere.

### References

| Rule | Reported when |
|---|---|
| `dangling-partner`, `dangling-child`, `dangling-named`, `dangling-depicted` | a family, an event or a picture names a person who has no record |
| `dangling-place` | an event, a picture, a place's `within` or a source's `covers` names a place that has no record |
| `dangling-family` | an event's `family` names a family that has no record |
| `dangling-repository` | a source names a repository that has no record |
| `dangling-source` | a citation names a source that has no record |
| `dangling-citation` | a citation names an anchor that the source doesn't have |
| `dangling-same-as` | `same-as` names a record that doesn't exist |
| `dangling-link` | a link in the body of a record leads to a record that doesn't exist |
| `dangling-main-person` | `main-person` in `origins.yaml` names a person who has no record |
| `dangling-bookmark` | a bookmark names a record that doesn't exist |
| `reserved-slug` | a record is called `unknown` or `none`, the words for a partner without a record |
| `same-as-itself` | a record says it is the same as itself |
| `same-as-another-kind` | `same-as` names a record of another kind |

### Families

| Rule | Reported when |
|---|---|
| `family-needs-partners` | a family has fewer than two partners |
| `family-none` | `none` doesn't stand beside exactly one person |
| `family-slug-mismatch` | the family's slug isn't the one its partners give it |
| `family-event-family` | a family's event has a `family`, which only a person's event takes |
| `child-listed-twice` | a family lists a child twice |
| `child-is-partner` | a family lists one of its partners as a child |
| `child-of-nobody` | a child's relation is `none` to every partner |
| `relation-not-a-partner` | a child's relation names somebody who isn't a partner of the family |
| `none-with-birth-child` | a family with `none` as a partner has a birth child, whose other parent is `unknown` rather than nobody |
| `too-many-birth-parents` | a person is a birth child of more than two people across their families |
| `circular-ancestry` | a person is their own ancestor |

### Dates

These rules compare only dates that leave no doubt. `ABT 1850` and `1850` can be the
same year, and an estimated or partial date never makes an error on its own.

| Rule | Reported when |
|---|---|
| `death-before-birth` | a person's `death` comes before their `birth` |
| `child-before-parent` | a person is born before a birth parent |
| `parent-too-young` | a birth parent was under 12 when the child was born |
| `posthumous-child` | a person is born a year or more after a birth parent's death |
| `family-event-order` | an `engagement`, `marriage-banns` or `marriage-licence` comes after the `marriage`, or a `separation`, `divorce-filed`, `divorce` or `annulment` before it, or a `divorce` before the `divorce-filed` |
| `family-ended-after-death` | a `separation` or a `divorce` comes after a partner's death |
| `ruled-out` | an event happened that a `not` entry says didn't, within its date or at all |

### Names

| Rule | Reported when |
|---|---|
| `two-primary-surnames` | two surnames of one name are marked `primary` |
| `surname-prefix-beside-list` | a name has a `surname-prefix` and a list of surnames, where each surname takes its own `prefix` |

### Pictures and files

| Rule | Reported when |
|---|---|
| `missing-portrait` | a person's `portrait` names a file that isn't beside the person |
| `missing-image` | a citation's image names a file that isn't beside the source |
| `media-outside` | a media path leads out of every record's directory |
| `dangling-media` | a media path leads into the directory of a record that doesn't exist |
| `unlisted-media` | a media path leads to a file of another record that has no entry for it |
| `duplicate-media` | a list of media has two entries for the same file |
| `impossible-region` | a region has an edge outside the picture, or its left edge isn't left of its right edge, or its top above its bottom |

### Hypotheses

| Rule | Reported when |
|---|---|
| `hypothesis-changes-nothing` | a hypothesis has no `edits` |

## Warnings

A warning is something that may be wrong, or that you probably want to look at.

| Rule | Reported when |
|---|---|
| `parent-too-old` | a birth mother was over 55, or a birth father over 80, when the child was born. Where the parent's sex isn't `F`, the limit is 80. |
| `possibly-ruled-out` | an event may fall within the date of a `not` entry that says it didn't happen |
| `duplicate-person` | two people have the same surname and first given name, and no birth or death date tells them apart |
| `same-as` | a record has a `same-as` claim that you haven't settled by merging the records or removing the claim |
| `disproven-link` | a family lists a child with the status `disproven` |
| `surname-slug-mismatch` | a person's slug doesn't begin with their surname, so the directory listing files them elsewhere |
| `coordinates-as-words` | a place's coordinates are words in parentheses, which the map can't place |
| `missing-media` | a media entry names a file that isn't there, such as a path from the program an import came from |
| `unevidenced-hypothesis` | a hypothesis has no citation in `supports` or `contradicts` |
| `unevidenced-proposal` | a hypothesis that somebody else proposed has no citation in `supports` or `contradicts` |
| `unsourced-event` | an event cites no source |
| `untranscribed-scan` | a file beside a source isn't the image of any citation |

## Research

`origins todo` lists the research that is still to do. None of it is an error, and
`todo` exits 0:

| Rule | Listed when |
|---|---|
| `unsourced-event` | an event cites no source |
| `untranscribed-scan` | a file beside a source isn't the image of any citation |
| `open-hypothesis` | a hypothesis is still `open` |
| `unrecorded-death` | a person is taken as dead, by `living: false` or by the [living rule](../guides/privacy#living-people), and no death, burial or cremation is recorded |
| `task` | a record's body has a task that isn't ticked, `- [ ] …` |
