package store

import (
	"reflect"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

func TestAnEditIsReadFromJSON(t *testing.T) {
	person := model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"}
	for written, want := range map[string]model.Edit{
		`{"op": "set", "entity": "person/rossi-giovanni", "path": "events[0].date", "value": 1834}`: model.Set{Entity: person, Path: "events[0].date", Value: "1834"},
		`{"op": "unset", "entity": "person/rossi-giovanni", "path": "sex"}`:                         model.Unset{Entity: person, Path: "sex"},
		`{"op": "add", "entity": "person/rossi-giovanni", "path": "tags", "value": "emigrant"}`:     model.Add{Entity: person, Path: "tags", Value: "emigrant"},
		`{"op": "create", "entity": "person/rossi-giovanni", "value": {"name": {"surname": "Rossi"}, "private": true}}`: model.Create{
			Entity: person, Value: map[string]any{"name": map[string]any{"surname": "Rossi"}, "private": "true"}},
		`{"op": "delete", "entity": "person/rossi-giovanni"}`:                                    model.Delete{Entity: person},
		`{"op": "rename", "entity": "person/rossi-giovanni", "to": "rossi-giovanni-1834"}`:       model.Rename{Entity: person, To: "rossi-giovanni-1834"},
		`{"op": "merge", "from": "person/rossi-giovanni", "into": "person/rossi-giovanni-1834"}`: model.Merge{From: person, Into: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni-1834"}},
		`{"op": "split", "entity": "person/rossi-giovanni", "into": ["person/a-one", "person/b-two"]}`: model.Split{Entity: person, Into: []model.EntityRef{
			{Kind: model.KindPerson, Slug: "a-one"}, {Kind: model.KindPerson, Slug: "b-two"}}},
	} {
		got, err := ParseEdit([]byte(written))
		if err != nil {
			t.Errorf("%s: %v", written, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s read as %#v", written, got)
		}
	}
	for _, refused := range []string{
		``,
		`{"op": "set", "entity": "person/rossi-giovanni", "path": "sex"}`,
		`{"op": "delete", "entity": "person/rossi-giovanni", "path": "sex"}`,
		`{"op": "promote", "entity": "person/rossi-giovanni"}`,
		`{"op": "set", "entity": "person/rossi-giovanni", "path": "sex", "value": "M", "colour": "red"}`,
		`[1, 2]`,
	} {
		if edit, err := ParseEdit([]byte(refused)); err == nil {
			t.Errorf("%q read as %#v", refused, edit)
		}
	}
}

// An edit's value is written out where it stands. An alias read as what it names would
// let a few lines of a hypothesis stand for more text than any machine holds.
func TestAnEditsValueIsNoAlias(t *testing.T) {
	written := "{op: create, entity: person/rossi-giovanni, value: {tags: &a [x, x], note: *a}}"
	if _, err := ParseEdit([]byte(written)); err == nil {
		t.Error("a value written as an alias was read")
	}
}
