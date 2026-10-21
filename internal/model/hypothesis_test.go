package model

import (
	"slices"
	"testing"
)

func person(slug Slug) EntityRef { return EntityRef{Kind: KindPerson, Slug: slug} }

// hypotheses is a small register of arguments about the Rossi family: two that answer
// Annunziata's birth date differently, one about her birthplace, one inside her first
// event, one that creates her unrecorded brother, and a closed one about Giovanni.
func hypotheses() *Corpus {
	c := NewCorpus()
	add := func(slug Slug, status Status, edits ...Edit) {
		c.Hypotheses[slug] = &Hypothesis{Slug: slug, Status: status, Edits: edits}
	}
	add("annunziata-born-1844", Rejected,
		Set{Entity: person("rossi-annunziata"), Path: "events[0].date", Value: "1844"})
	add("annunziata-born-1846", Open,
		Set{Entity: person("rossi-annunziata"), Path: "events[0].date", Value: "1846"})
	add("annunziata-born-in-prato", Open,
		Set{Entity: person("rossi-annunziata"), Path: "events[0].place", Value: "prato"})
	add("annunziata-baptism-was-birth", Open,
		Unset{Entity: person("rossi-annunziata"), Path: "events[0]"})
	add("annunziata-born-elsewhere", Open,
		Set{Entity: person("rossi-annunziata"), Path: "events[10].date", Value: "1850"})
	add("unrecorded-child-1845", Open,
		Create{Entity: person("rossi-bambino-1845"), Value: map[string]any{"parents": []any{"rossi-antonio"}}},
		Add{Entity: person("rossi-bambino-1845"), Path: "events", Value: map[string]any{"type": "birth"}})
	add("giovanni-is-two-men", Accepted,
		Split{Entity: person("rossi-giovanni"), Into: []EntityRef{person("rossi-giovanni-1834"), person("rossi-giovanni-1835")}})
	add("giovanni-is-one-man", Rejected,
		Merge{From: person("rossi-giovanni-1835"), Into: person("rossi-giovanni")})
	return c
}

func TestConcernsAreTheSubjectsOnce(t *testing.T) {
	c := hypotheses()
	got := c.Hypotheses["unrecorded-child-1845"].Concerns()
	if want := []EntityRef{person("rossi-bambino-1845")}; !slices.Equal(got, want) {
		t.Errorf("concerns = %v, want %v", got, want)
	}
	got = c.Hypotheses["giovanni-is-two-men"].Concerns()
	if want := []EntityRef{person("rossi-giovanni"), person("rossi-giovanni-1834"), person("rossi-giovanni-1835")}; !slices.Equal(got, want) {
		t.Errorf("concerns = %v, want %v", got, want)
	}
}

func TestOpenHypothesesConcerningAnEntity(t *testing.T) {
	c := hypotheses()
	assertSlugs(t, "open on Annunziata", c.OpenHypotheses("rossi-annunziata"), []Slug{
		"annunziata-baptism-was-birth", "annunziata-born-1846", "annunziata-born-elsewhere", "annunziata-born-in-prato",
	})
	// the parent named inside a created person's fields is not what the hypothesis is about
	assertSlugs(t, "open on Antonio", c.OpenHypotheses("rossi-antonio"), nil)
	// every hypothesis about Giovanni has been settled
	assertSlugs(t, "open on Giovanni", c.OpenHypotheses("rossi-giovanni"), nil)
}

func TestAlternativesEditTheSamePlace(t *testing.T) {
	c := hypotheses()
	// the rejected answer is still an alternative, and so is the edit that removes the
	// whole event the date is in; the birthplace and the eleventh event are not
	assertSlugs(t, "alternatives to 1846", c.Alternatives("annunziata-born-1846"), []Slug{
		"annunziata-baptism-was-birth", "annunziata-born-1844",
	})
	assertSlugs(t, "alternatives to removing the event", c.Alternatives("annunziata-baptism-was-birth"), []Slug{
		"annunziata-born-1844", "annunziata-born-1846", "annunziata-born-in-prato",
	})
	assertSlugs(t, "alternatives to the eleventh event", c.Alternatives("annunziata-born-elsewhere"), nil)
	// a split and a merge are both about the whole of Giovanni
	assertSlugs(t, "alternatives to the split", c.Alternatives("giovanni-is-two-men"), []Slug{"giovanni-is-one-man"})
	assertSlugs(t, "alternatives to the unrecorded child", c.Alternatives("unrecorded-child-1845"), nil)
	assertSlugs(t, "alternatives to nothing", c.Alternatives("no-such-hypothesis"), nil)
}

func TestWithin(t *testing.T) {
	for _, tc := range []struct {
		path, outer string
		want        bool
	}{
		{"events[0].date", "events[0].date", true},
		{"events[0].date", "events[0]", true},
		{"events[0]", "events", true},
		{"events[0][1]", "events[0]", true},
		{"events[10]", "events[1]", false},
		{"names", "name", false},
		{"events[0]", "events[0].date", false},
		{"", "", true},
		{"events", "", false},
		{"", "events", false},
	} {
		if got := within(tc.path, tc.outer); got != tc.want {
			t.Errorf("within(%q, %q) = %v, want %v", tc.path, tc.outer, got, tc.want)
		}
	}
}
