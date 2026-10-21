package store

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/model"
)

// Fields is an entity spelled the way an edit's value is: every field a path can reach
// and the file writes, under the name the file writes it by, each as text, a list or a
// mapping, with the body beside them under its own name.
//
// It is read back off the emitted file rather than built from the model, so it says
// whatever the file says, spelled as the file spells it. A list is always a list,
// though: the emitter writes a place within one other place in a word, and a value is
// shaped by the field it stands for and not by how briefly that field can be written.
//
// The manifest is the mapping origins.yaml writes under project:, which is where its
// paths start, and has no body.
func Fields(entity any) (map[string]any, error) {
	var frontmatter, body []byte
	if project, ok := entity.(*model.Project); ok {
		emitted, err := EmitProject(*project)
		if err != nil {
			return nil, err
		}
		frontmatter = emitted
	} else {
		emitted, err := Emit(entity)
		if err != nil {
			return nil, err
		}
		if frontmatter, body, err = Split(emitted); err != nil {
			return nil, err
		}
	}
	var document yaml.Node
	if err := yaml.Unmarshal(frontmatter, &document); err != nil {
		return nil, err
	}
	out := map[string]any{}
	if len(document.Content) > 0 {
		read, err := literal(document.Content[0])
		if err != nil {
			return nil, err
		}
		written, ok := read.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%T emits frontmatter that is not a mapping", entity)
		}
		out = written
	}
	if _, ok := entity.(*model.Project); ok {
		out, _ = out["project"].(map[string]any)
		if out == nil {
			out = map[string]any{}
		}
	}
	reachable := addressable(entity)
	for name, value := range out {
		target, ok := reachable[name]
		if !ok {
			delete(out, name)
			continue
		}
		if _, listed := value.([]any); !listed && reflect.TypeOf(target).Elem().Kind() == reflect.Slice {
			out[name] = []any{value}
		}
	}
	if len(body) > 0 {
		out["body"] = string(body)
	}
	return out, nil
}

// Written is what a path reaches in an entity's Fields: text, a list or a mapping, or
// nil where the file says nothing there.
func Written(fields map[string]any, path string) any {
	steps, err := parsePath(path)
	if err != nil {
		return nil
	}
	var at any = fields
	for _, s := range steps {
		mapping, ok := at.(map[string]any)
		if !ok {
			return nil
		}
		at = mapping[s.name]
		for _, index := range s.indices {
			list, ok := at.([]any)
			if !ok || index >= len(list) {
				return nil
			}
			at = list[index]
		}
	}
	return at
}

// Addressable names every field of an entity a path can start at, whether or not the
// entity says anything there.
func Addressable(entity any) []string {
	return slices.Sorted(maps.Keys(addressable(entity)))
}

// Place is one path into an entity, whether or not the entity says anything there,
// and what can be done at it.
type Place struct {
	Path string

	// List is a path at a list, which takes an entry on its end and gives one up as
	// well as being set and unset whole. Every other path is set and unset.
	List bool
}

// Places is every path into an entity that Resolve reaches, the fields of each value in
// the order of the alphabet: each field, the empty ones included, and inside each
// entry of a list and each member of a mapping it holds. A mapping's keys are whatever names
// somebody has written, so a member nobody has written is not among them, nor an
// entry a list does not hold.
func Places(entity any) []Place {
	var out []Place
	walkPlaces(reflect.ValueOf(entity), "", &out)
	return out
}

func walkPlaces(at reflect.Value, prefix string, out *[]Place) {
	fields := addressable(at.Interface())
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		placed(reflect.ValueOf(fields[name]).Elem(), path, out)
	}
}

// placed puts down the path to one value and everything inside it.
func placed(value reflect.Value, path string, out *[]Place) {
	switch value.Kind() {
	case reflect.Slice:
		*out = append(*out, Place{Path: path, List: true})
		for i := range value.Len() {
			placed(value.Index(i), path+"["+strconv.Itoa(i)+"]", out)
		}
	case reflect.Map:
		*out = append(*out, Place{Path: path})
		keys := value.MapKeys()
		slices.SortFunc(keys, func(a, b reflect.Value) int { return strings.Compare(a.String(), b.String()) })
		for _, key := range keys {
			held := reflect.New(value.Type().Elem()).Elem()
			held.Set(value.MapIndex(key))
			placed(held, path+"."+key.String(), out)
		}
	default:
		*out = append(*out, Place{Path: path})
		if value.CanAddr() && addressable(value.Addr().Interface()) != nil {
			walkPlaces(value.Addr(), path, out)
		}
	}
}

// Shape is every path into an entity of its type, the empty ones included, with [] for
// any entry of a list and * for any member of a mapping, so that what a corpus can say
// is read off the table Resolve reads rather than off one corpus that says it.
func Shape(entity any) []string {
	var out []string
	walkShape(reflect.TypeOf(entity).Elem(), "", &out)
	return out
}

func walkShape(t reflect.Type, prefix string, out *[]string) {
	fields := addressable(reflect.New(t).Interface())
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		shaped(reflect.TypeOf(fields[name]).Elem(), path, out)
	}
}

// shaped puts down the path to a value of one type and every path inside it.
func shaped(t reflect.Type, path string, out *[]string) {
	*out = append(*out, path)
	switch t.Kind() {
	case reflect.Slice:
		shaped(t.Elem(), path+"[]", out)
	case reflect.Map:
		shaped(t.Elem(), path+".*", out)
	default:
		if addressable(reflect.New(t).Interface()) != nil {
			walkShape(t, path, out)
		}
	}
}
