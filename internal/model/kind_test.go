package model

import (
	"slices"
	"testing"
)

// Go cannot check that a switch covers every kind, so this is the check: a kind added
// to Kinds and missed by any lookup fails here.
func TestEveryKindIsReachedByEveryLookup(t *testing.T) {
	for _, kind := range Kinds {
		c := NewCorpus()
		entity := New(kind)
		if entity == nil {
			t.Fatalf("New(%s) = nil", kind)
		}
		c.Put("one", entity)

		if got := RefOf(entity); got != (EntityRef{kind, "one"}) {
			t.Errorf("RefOf = %v, want %s/one", got, kind)
		}
		if c.Entity("one") != entity {
			t.Errorf("%s: Entity does not find what Put filed", kind)
		}
		if got := c.KindOf("one"); got != kind {
			t.Errorf("%s: KindOf = %q", kind, got)
		}
		if got := c.SlugsOf(kind); !slices.Equal(got, []Slug{"one"}) {
			t.Errorf("%s: SlugsOf = %v", kind, got)
		}
		if got := c.AllSlugs(); !slices.Equal(got, []Slug{"one"}) {
			t.Errorf("%s: AllSlugs = %v", kind, got)
		}
		if kind.Dir() == "" || KindIn(kind.Dir()) != kind {
			t.Errorf("%s: kept in %q, which KindIn reads as %q", kind, kind.Dir(), KindIn(kind.Dir()))
		}
		if bodyOf(entity) == nil {
			t.Errorf("%s: bodyOf = nil", kind)
		}
		if !c.Taken("one") {
			t.Errorf("%s: Taken is false for a filed record", kind)
		}
		if files := Files(entity); files == nil {
			t.Errorf("%s: Files = nil", kind)
		} else if *files = []string{"a.jpg"}; !slices.Equal(*Files(c.Entity("one")), []string{"a.jpg"}) {
			t.Errorf("%s: Files is not the record's own list", kind)
		}

		if same := SameAsOf(entity); (same == nil) != (kind == KindHypothesis) {
			t.Errorf("%s: SameAsOf = %v", kind, same)
		}

		c.Delete("one")
		if c.Entity("one") != nil || c.Taken("one") {
			t.Errorf("%s: Delete left the record in place", kind)
		}
	}
}

func TestWhatIsNotAnEntityHasNoKind(t *testing.T) {
	if New("parish") != nil {
		t.Error("New made a record of a kind that does not exist")
	}
	if RefOf(&Event{}) != (EntityRef{}) || RefOf(nil) != (EntityRef{}) {
		t.Error("RefOf gave a kind to something that is not an entity")
	}
	if Files(nil) != nil {
		t.Error("Files gave a list to nothing")
	}
	if Kind("parish").Dir() != "" || KindIn("parishes") != "" {
		t.Error("a kind that does not exist has a directory")
	}
	if NewCorpus().KindOf("nobody") != "" {
		t.Error("KindOf found a record in an empty corpus")
	}
}
