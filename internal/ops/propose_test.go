package ops

import (
	"errors"
	"reflect"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// authored edits a copy of the fixture as though a hypothesis were true and asks what
// the edits were.
func authored(t *testing.T, edit func(t *testing.T, c *model.Corpus)) ([]model.Edit, error) {
	t.Helper()
	before := fixture(t).Corpus
	after := before.Clone()
	edit(t, after)
	return Propose(before, after)
}

func stated(t *testing.T, edit func(t *testing.T, c *model.Corpus), want ...model.Edit) {
	t.Helper()
	got, err := authored(t, edit)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("proposed\n%#v\nwant\n%#v", got, want)
	}
}

func unstated(t *testing.T, edit func(t *testing.T, c *model.Corpus), path string) {
	t.Helper()
	_, err := authored(t, edit)
	var refused *Unstated
	if !errors.As(err, &refused) {
		t.Fatalf("want the difference refused, got %v", err)
	}
	if refused.Path != path {
		t.Errorf("refused at %q, want %q: %v", refused.Path, path, err)
	}
}

var louisa = model.EntityRef{Kind: model.KindPerson, Slug: "verdier-louisa"}

func TestAChangedValueIsSetWhereItIs(t *testing.T) {
	t.Parallel()
	stated(t, func(t *testing.T, c *model.Corpus) {
		c.People["verdier-louisa"].Events[0].Date = mustDate(t, "1844")
	}, model.Set{Entity: louisa, Path: "events[0].date", Value: "1844"})
}

func TestEntriesAreTakenOutChangedAndAddedInThatOrder(t *testing.T) {
	t.Parallel()
	stated(t, func(t *testing.T, c *model.Corpus) {
		person := c.People["verdier-louisa"]
		person.Events[0].Sources = person.Events[0].Sources[1:]
		person.Events[0].Place = "firenze"
		person.Events = append(person.Events, model.Event{Type: "death", Date: mustDate(t, "1901")})
	},
		model.Set{Entity: louisa, Path: "events[0].place", Value: "firenze"},
		model.Remove{Entity: louisa, Path: "events[0].sources", Value: "french-church-london-baptisms#1842-007"},
		model.Add{Entity: louisa, Path: "events", Value: map[string]any{"type": "death", "date": "1901"}},
	)
}

var louisasFamily = model.EntityRef{Kind: model.KindFamily, Slug: "lang-anna-maria--verdier-louis"}

func TestAChildGivenARelationIsSetWhole(t *testing.T) {
	t.Parallel()
	stated(t, func(t *testing.T, c *model.Corpus) {
		c.Families["lang-anna-maria--verdier-louis"].Children[1].Relation = model.Relationship{Word: model.Adoptive}
	}, model.Set{Entity: louisasFamily, Path: "children[1]", Value: map[string]any{"id": "verdier-louisa", "relation": "adoptive"}})
}

func TestARecordArrivedIsCreatedAndOneGoneIsDeleted(t *testing.T) {
	t.Parallel()
	got, err := authored(t, func(t *testing.T, c *model.Corpus) {
		c.People["verdier-baby-1845"] = &model.Person{
			Slug: "verdier-baby-1845", Name: model.PersonName{Given: "Baby", Surname: model.SurnameOf("Verdier")}, Sex: model.StatedSex{Value: model.Male},
			Body: "Named in a letter and nowhere else.",
		}
		family := c.Families["lang-anna-maria--verdier-louis"]
		family.Children = append(family.Children, model.Child{ID: "verdier-baby-1845"})
		delete(c.Sources, "claudia-interview-2019")
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []model.Edit{
		model.Delete{Entity: ref(model.KindSource, "claudia-interview-2019")},
		model.Create{Entity: ref(model.KindPerson, "verdier-baby-1845"), Value: map[string]any{
			"name": map[string]any{"given": "Baby", "surname": "Verdier"}, "sex": "M",
			"body": "Named in a letter and nowhere else.",
		}},
		model.Add{Entity: louisasFamily, Path: "children", Value: "verdier-baby-1845"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("proposed\n%#v\nwant\n%#v", got, want)
	}
}

// A record renamed carries its family and every reference to it, so the rename is the
// whole of what is stated.
func TestARecordUnderAnotherNameIsRenamed(t *testing.T) {
	t.Parallel()
	stated(t, func(t *testing.T, c *model.Corpus) {
		if _, err := applied(c, model.Rename{Entity: louisa, To: "verdier-louisa-1842"}); err != nil {
			t.Fatal(err)
		}
	}, model.Rename{Entity: louisa, To: "verdier-louisa-1842"})
}

func TestAnEntryPutInBeforeTheEndIsRefused(t *testing.T) {
	t.Parallel()
	unstated(t, func(t *testing.T, c *model.Corpus) {
		person := c.People["verdier-louisa"]
		person.Events = append([]model.Event{{Type: "baptism", Date: mustDate(t, "1842")}}, person.Events...)
	}, "events")
}

func TestEntriesPutInAnotherOrderAreRefused(t *testing.T) {
	t.Parallel()
	unstated(t, func(t *testing.T, c *model.Corpus) {
		sources := c.People["verdier-louisa"].Events[0].Sources
		sources[0], sources[1] = sources[1], sources[0]
	}, "events[0].sources")
}

var paulinesFamily = model.EntityRef{Kind: model.KindFamily, Slug: "papadopoulou-maria--verdier-jean-baptiste"}

// A child given a relation and the child after her taken out cost the same stated
// either way round, and the way that keeps the child where she stands is the one stated.
func TestAnEntryChangedWhereAnotherIsTakenOutKeepsItsPlace(t *testing.T) {
	t.Parallel()
	stated(t, func(t *testing.T, c *model.Corpus) {
		family := c.Families["papadopoulou-maria--verdier-jean-baptiste"]
		family.Children = []model.Child{family.Children[0],
			{ID: "verdier-pauline", Relation: model.Relationship{Word: model.Adoptive}}, family.Children[3]}
	},
		model.Remove{Entity: paulinesFamily, Path: "children", Value: "verdier-pierre"},
		model.Set{Entity: paulinesFamily, Path: "children[1]", Value: map[string]any{"id": "verdier-pauline", "relation": "adoptive"}},
	)
}

// An entry put in beside one that also changed is still an entry put in, and is not
// to be stated as the one beside it having turned into it.
func TestAnEntryPutInBesideAChangedOneIsRefused(t *testing.T) {
	t.Parallel()
	unstated(t, func(t *testing.T, c *model.Corpus) {
		person := c.People["verdier-louisa"]
		birth := person.Events[0]
		birth.Date = mustDate(t, "1844")
		birth.Sources = birth.Sources[1:]
		person.Events = []model.Event{{Type: "baptism", Date: mustDate(t, "19 MAR 1842")}, birth,
			{Type: "death", Date: mustDate(t, "1901")}}
	}, "events")
}

// A hypothesis's edits are the language a change is written in and no path reaches
// them, so a change to them is one nothing can state.
func TestADifferenceNoPathReachesIsRefused(t *testing.T) {
	t.Parallel()
	unstated(t, func(t *testing.T, c *model.Corpus) {
		h := c.Hypotheses["louisa-born-1844"]
		h.Edits = h.Edits[:0]
	}, "")
}

// The manifest is compared as a record is, its fields stated as edits to project after
// every record's.
func TestAChangedManifestIsStatedAsEditsToTheProject(t *testing.T) {
	t.Parallel()
	stated(t, func(t *testing.T, c *model.Corpus) {
		c.People["verdier-louisa"].Events[0].Date = mustDate(t, "1844")
		c.Project.MainPerson = "verdier-louisa"
		c.Project.Researchers = append(c.Project.Researchers, model.Researcher{Name: "Rosa Bianchi"})
		c.Project.Tags["Emigrants"] = model.TagStyle{Color: "#2a7", Priority: "2"}
	},
		model.Set{Entity: louisa, Path: "events[0].date", Value: "1844"},
		model.Set{Entity: model.TheProject, Path: "main-person", Value: "verdier-louisa"},
		model.Add{Entity: model.TheProject, Path: "researchers", Value: map[string]any{"name": "Rosa Bianchi"}},
		model.Set{Entity: model.TheProject, Path: "tags.Emigrants.priority", Value: "2"},
	)
}

// A tag whose word no path can spell is styled by setting the styles whole.
func TestAStyleNoPathCanReachIsSetWithTheRest(t *testing.T) {
	t.Parallel()
	stated(t, func(t *testing.T, c *model.Corpus) {
		c.Project.Tags["St. Mary's"] = model.TagStyle{Color: "#00f"}
	}, model.Set{Entity: model.TheProject, Path: "tags", Value: map[string]any{
		"Emigrants":  map[string]any{"color": "#2a7"},
		"St. Mary's": map[string]any{"color": "#00f"},
		"To do":      map[string]any{"color": "#d33", "priority": "1"},
	}})
}

func TestAChangedFormatVersionIsRefused(t *testing.T) {
	t.Parallel()
	_, err := authored(t, func(t *testing.T, c *model.Corpus) {
		c.Project.FormatVersion++
	})
	var refused *Unstated
	if !errors.As(err, &refused) || refused.Entity != model.TheProject {
		t.Fatalf("want the manifest refused as unstated, got %v", err)
	}
}
