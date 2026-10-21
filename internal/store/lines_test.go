package store

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// annunziata is the shape that matters: a flow list, a block list of mappings, a list
// of scalars inside one of those, and a body below the fence.
const annunziata = `---
name:
  given: Annunziata
  surname: Rossi
also-known-as: [Nunziata, Annunziata Rossi]
events:
  - type: birth
    date: 1842
    sources:
      - parish-san-lorenzo-baptisms#1842-007
      - censimento-1861-firenze#f214
---

The two sources disagree.

I follow the parish register.
`

func spanning(t *testing.T, content string) []Span {
	t.Helper()
	spans, err := Spans([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return spans
}

func TestEveryFieldIsFoundWhereTheFileWritesIt(t *testing.T) {
	got := spanning(t, annunziata)
	want := []Span{
		{"name", 2, 4},
		{"name.given", 3, 3},
		{"name.surname", 4, 4},
		{"also-known-as", 5, 5},
		{"events", 6, 11},
		{"events[0]", 7, 11},
		{"events[0].type", 7, 7},
		{"events[0].date", 8, 8},
		{"events[0].sources", 9, 11},
		{"events[0].sources[0]", 10, 10},
		{"events[0].sources[1]", 11, 11},
		{"body", 14, 16},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n\t%+v\nwant\n\t%+v", got, want)
	}
}

// A line belongs to one fact, which is the innermost of the fields covering it. The
// list a citation is in is a field too, and answering with that one would make every
// line of an event read as a change to the whole event.
func TestALineBelongsToTheInnermostFieldOnIt(t *testing.T) {
	spans := spanning(t, annunziata)
	for line, want := range map[int]string{
		2:  "name",
		3:  "name.given",
		4:  "name.surname",
		5:  "also-known-as",
		6:  "events",
		7:  "events[0].type",
		8:  "events[0].date",
		9:  "events[0].sources",
		10: "events[0].sources[0]",
		11: "events[0].sources[1]",
		14: "body",
		16: "body",
	} {
		span, ok := At(spans, line)
		if !ok {
			t.Errorf("line %d belongs to no field", line)
			continue
		}
		if span.Path != want {
			t.Errorf("line %d is %s, want %s", line, span.Path, want)
		}
	}
	for _, fence := range []int{1, 12} {
		if span, ok := At(spans, fence); ok {
			t.Errorf("the fence on line %d was read as %s", fence, span.Path)
		}
	}
}

// Folding is the one thing that puts a single fact on several lines, and it is the
// long transcriptions — the values most worth asking about — that it happens to.
func TestAFoldedValueIsOneFactOverSeveralLines(t *testing.T) {
	spans := spanning(t, `---
title: Baptismal Register
citations:
  1834-042:
    text: >
      Giovanni Battista, figlio di Antonio Rossi e Lucia Verdi, nato il 12
      marzo 1834 e battezzato il 14 detto.
    images: [scans/1834-042.jpg]
---
`)
	for line, want := range map[int]string{
		4: "citations.1834-042",
		5: "citations.1834-042.text",
		6: "citations.1834-042.text",
		7: "citations.1834-042.text",
		8: "citations.1834-042.images",
	} {
		span, ok := At(spans, line)
		if !ok || span.Path != want {
			t.Errorf("line %d is %q (%v), want %s", line, span.Path, ok, want)
		}
	}
}

func TestAFileWithNothingInItHasNoFields(t *testing.T) {
	for what, content := range map[string]string{
		"no fields and no body": "---\n---\n",
		"a body and no fields":  "---\n---\n\nAll I know is that he existed.\n",
	} {
		spans := spanning(t, content)
		var paths []string
		for _, span := range spans {
			paths = append(paths, span.Path)
		}
		if strings.Join(paths, " ") != map[string]string{
			"no fields and no body": "",
			"a body and no fields":  "body",
		}[what] {
			t.Errorf("%s: %v", what, spans)
		}
	}
}

func TestAFileThatIsNotAnEntityIsRefused(t *testing.T) {
	if _, err := Spans([]byte("name: Giovanni\n")); err == nil {
		t.Error("a file with no frontmatter fence was read as an entity")
	}
}

// The path a line is found at has to be a path the rest of origins can act on:
// the one check reports a finding at, and the one an edit is written against. Nothing
// but a test connects the two, so here it is over every file in the fixture.
//
// A hypothesis's edits are the exception, and deliberately so: they are the language a
// change is written in, not something another change reaches inside and rewrites. The
// lines are still named, because the file has them and somebody will ask who wrote
// that edit.
func TestEveryFieldFoundInTheFixtureCanBeResolved(t *testing.T) {
	corpus := fixture(t)
	for _, slug := range corpus.Slugs() {
		relative, _ := corpus.Path(slug)
		content, err := os.ReadFile(filepath.Join(corpus.Root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		spans, err := Spans(content)
		if err != nil {
			t.Fatalf("%s: %v", relative, err)
		}
		if len(spans) == 0 {
			t.Errorf("%s: nothing in it was found", relative)
		}
		for _, span := range spans {
			if strings.HasPrefix(span.Path, "edits") {
				continue
			}
			if _, err := Resolve(corpus.Entity(slug), span.Path); err != nil {
				t.Errorf("%s: %v", relative, err)
			}
		}
	}
}

// The manifest has no fence, and its fields are named as an edit of the project names
// them, without project in front; what origins keeps for itself is no field of it.
func TestTheManifestSaysWhereItsFieldsAre(t *testing.T) {
	manifest := `project:
  name: Rossi family
  main-person: rossi-giovanni
  bookmarks:
    - people/rossi-giovanni
  tags:
    To do:
      color: '#d33'
origins:
  format-version: 1
`
	got, err := ManifestSpans([]byte(manifest))
	if err != nil {
		t.Fatal(err)
	}
	want := []Span{
		{"name", 2, 2},
		{"main-person", 3, 3},
		{"bookmarks", 4, 5},
		{"bookmarks[0]", 5, 5},
		{"tags", 6, 8},
		{"tags.To do", 7, 8},
		{"tags.To do.color", 8, 8},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got\n\t%+v\nwant\n\t%+v", got, want)
	}
	if _, ok := At(got, 10); ok {
		t.Error("format-version is a field of the project")
	}
}
