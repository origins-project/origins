package store

import (
	"reflect"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// A value is spelled as the file spells it, except that a list is a list however
// briefly the file writes it, and every name in it is one a path can reach.
func TestFieldsAreTheRecordAsAnEditSpellsIt(t *testing.T) {
	t.Parallel()
	place := &model.Place{
		Slug: "borgo-san-lorenzo", Name: model.PlaceName{Text: "Borgo San Lorenzo"}, Type: "comune",
		Within: []model.PlaceRef{{Place: "firenze"}},
		Body:   "In the Mugello.",
	}
	got, err := Fields(place)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"name": "Borgo San Lorenzo", "type": "comune", "within": []any{"firenze"},
		"body": "In the Mugello.",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("fields = %#v, want %#v", got, want)
	}

	for field, value := range got {
		fresh := &model.Place{Slug: place.Slug}
		set(t, fresh, field, value)
	}
}

// A hypothesis's edits are written in its file and reached by no path, so they are
// not among its fields.
func TestFieldsLeaveOutWhatNoPathReaches(t *testing.T) {
	t.Parallel()
	h := &model.Hypothesis{Slug: "annunziata-born-1844", Status: model.Open, Base: "8f78553",
		Edits: []model.Edit{model.Unset{Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-annunziata"}, Path: "surname"}}}
	got, err := Fields(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["edits"]; ok || got["status"] != "open" {
		t.Errorf("fields = %#v", got)
	}
}
