package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/model"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// indexToday is the day the index decides who is living as of, so that the golden file
// is not a fact about the day the test runs.
var indexToday = model.DateOf(time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC))

// The fixture's index is a golden file, so that what the pages are given is a diff a
// reader can judge whenever it changes.
func TestTheFixturesIndex(t *testing.T) {
	t.Parallel()
	got, err := json.MarshalIndent(indexOf(fixture(t).Corpus, indexToday), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	path := filepath.Join("testdata", "index.json")
	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v; run go test ./internal/api -update", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("the index is not %s; run go test ./internal/api -update and read the diff", path)
	}
}

// Every slug the index names is a record of the kind the field says, or one of the two
// words where a partner or a child may be one, so a page can follow every one of them.
func TestEverySlugInTheIndexResolves(t *testing.T) {
	t.Parallel()
	corpus := fixture(t).Corpus
	index := indexOf(corpus, indexToday)
	resolves := func(kind model.Kind, slug string, reserved bool, where string) {
		t.Helper()
		if reserved && model.Slug(slug).Reserved() {
			return
		}
		if corpus.KindOf(model.Slug(slug)) != kind {
			t.Errorf("%s names %q, which is no %s", where, slug, kind)
		}
	}
	dated := func(d *Dated, where string) {
		if d != nil && d.Place != "" {
			resolves(model.KindPlace, d.Place, false, where)
		}
	}

	resolves(model.KindPerson, index.Project.MainPerson, false, "the main person")
	for _, bookmark := range index.Project.Bookmarks {
		ref, err := model.ParseEntityRef(bookmark)
		if err != nil {
			t.Fatal(err)
		}
		resolves(ref.Kind, string(ref.Slug), false, "a bookmark")
	}
	for _, p := range index.People {
		dated(p.Birth, p.Ref)
		dated(p.Death, p.Ref)
		for _, family := range slices.Concat(p.PartnerIn, p.ChildIn) {
			resolves(model.KindFamily, family, false, p.Ref)
		}
	}
	for _, f := range index.Families {
		dated(f.First, f.Ref)
		for _, partner := range f.Partners {
			if partner.Person != "" {
				resolves(model.KindPerson, partner.Person, true, f.Ref)
			}
		}
		for _, child := range f.Children {
			if child.Person != "" {
				resolves(model.KindPerson, child.Person, true, f.Ref)
			}
			for _, relation := range child.Relations {
				resolves(model.KindPerson, relation.Partner, true, f.Ref)
			}
		}
	}
	for _, p := range index.Places {
		for _, within := range p.Within {
			resolves(model.KindPlace, within.Place, false, p.Ref)
		}
	}
	for _, s := range index.Sources {
		for _, kept := range s.Kept {
			resolves(model.KindRepository, kept.Repository, false, s.Ref)
		}
	}
	for _, r := range index.Repositories {
		for _, source := range r.Sources {
			resolves(model.KindSource, source, false, r.Ref)
		}
	}
	for _, n := range index.Notes {
		for _, event := range n.Events {
			dated(&event.Dated, n.Ref)
			for _, named := range event.With {
				if named.Person != "" {
					resolves(model.KindPerson, named.Person, true, n.Ref)
				}
			}
		}
	}
	if counts := index.Counts(); counts != counted(corpus) {
		t.Errorf("the index counts %+v, and the corpus holds %+v", counts, counted(corpus))
	}
}

// The index of a hidden view lists the placeholders standing for whoever it hides, and
// names nobody and nothing it hides: whatever page shows it shows only what the view
// does.
func TestAHiddenViewsIndexListsPlaceholders(t *testing.T) {
	t.Parallel()
	corpus := fixture(t).Corpus
	view := model.Hide(corpus, model.Hiding{Private: true, Living: true}, indexToday)
	encoded, err := json.Marshal(indexOf(view, indexToday))
	if err != nil {
		t.Fatal(err)
	}
	var strs []string
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case string:
			strs = append(strs, v)
		case []any:
			for _, e := range v {
				walk(e)
			}
		case map[string]any:
			for k, e := range v {
				strs = append(strs, k)
				walk(e)
			}
		}
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	walk(decoded)

	shown := map[string]bool{}
	for _, slug := range view.PeopleSlugs() {
		shown[view.People[slug].Name.String()] = true
	}
	placeholders := 0
	for _, s := range strs {
		if strings.HasPrefix(s, "person/private-") {
			placeholders++
		}
	}
	if placeholders == 0 {
		t.Error("the hidden view's index lists no placeholder")
	}
	for _, slug := range corpus.AllSlugs() {
		if view.KindOf(slug) != "" {
			continue
		}
		ref := model.RefOf(corpus.Entity(slug)).String()
		var name string
		if person := corpus.People[slug]; person != nil && !shown[person.Name.String()] {
			name = person.Name.String()
		}
		for _, s := range strs {
			if s == string(slug) || s == ref || name != "" && s == name {
				t.Errorf("the hidden view's index names %s, which it hides, as %q", ref, s)
			}
		}
	}
}
