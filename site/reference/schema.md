<!-- Written by go test ./cmd/origins -run TestTheSitesGeneratedPages -update from origins schema. -->

# JSON Schema

`origins schema <kind>` prints the JSON Schema of one kind of record, and the
files below are the same schemas. An editor that reads JSON Schema can use them to
complete and check the frontmatter of a record. Each schema refers to
`common.json` for the values that every kind shares, so keep the files together.

Origins doesn't read a corpus through these schemas, and `origins check` doesn't
use them. A file that the schema accepts can still be refused by Origins, for
example when a reference names a record that doesn't exist.

| Kind | Schema | Description |
|---|---|---|
| person | [person.json](/schema/person.json){target="_self"} | A person: the name they are filed under, the other names, their sex and what happened to them. Parents, families and children are derived from the families that list them. |
| family | [family.json](/schema/family.json){target="_self"} | The people a child was born to or raised by, with the partnership between them where there was one, and the children. |
| place | [place.json](/schema/place.json){target="_self"} | A place, with the names it went by, what it lies within and when it was there. |
| source | [source.json](/schema/source.json){target="_self"} | A thing that exists in the world, and the citations into it, each a named anchor. |
| repository | [repository.json](/schema/repository.json){target="_self"} | The archive, library, parish office or website a source is kept in. |
| note | [note.json](/schema/note.json){target="_self"} | What is about no record in particular: a research log, a list of things to do, a letter to an archive. |
| hypothesis | [hypothesis.json](/schema/hypothesis.json){target="_self"} | A proposed departure from the corpus, expressed as edits origins can apply, test and reverse. |
| project (`origins.yaml`) | [project.json](/schema/project.json){target="_self"} | Who keeps the tree and for whom: the mapping origins.yaml writes under project. |
| common | [common.json](/schema/common.json){target="_self"} | The values every kind of record is written with, each spelled as the corpus spells it: every scalar is text. |
