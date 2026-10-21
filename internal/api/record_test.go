package api

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

func fixture(t *testing.T) *store.Corpus {
	t.Helper()
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	corpus, err := store.Load(root)
	if err != nil {
		t.Fatalf("the fixture should load without a single problem:\n%v", err)
	}
	return corpus
}

// Every path the emitter writes into a record of the fixture is one the record is
// served with, with the operations that apply to it, and every path it is served with
// is one a change can address. A field the model gains is then a field the pages can
// change, or a test that fails.
func TestEveryPathIsServed(t *testing.T) {
	t.Parallel()
	corpus := fixture(t)
	refs := []model.EntityRef{model.TheProject}
	for _, slug := range corpus.Slugs() {
		refs = append(refs, model.RefOf(corpus.Entity(slug)))
	}
	for _, ref := range refs {
		record, err := RecordOf(corpus, ref)
		if err != nil {
			t.Fatal(err)
		}
		entity := any(&corpus.Project)
		written, err := store.EmitProject(corpus.Project)
		if ref != model.TheProject {
			entity = corpus.Entity(ref.Slug)
			var emitted []byte
			if emitted, err = store.Emit(entity); err == nil {
				written, _, err = store.Split(emitted)
			}
		}
		if err != nil {
			t.Fatal(err)
		}

		served := map[string][]string{}
		for _, one := range record.Edits {
			if _, err := store.Resolve(entity, one.Path); err != nil {
				t.Errorf("%s is served with %s, which nothing can address: %v", ref, one.Path, err)
			}
			served[one.Path] = one.Ops
		}
		var document yaml.Node
		if err := yaml.Unmarshal(written, &document); err != nil {
			t.Fatal(err)
		}
		if len(document.Content) == 0 {
			continue
		}
		top := document.Content[0]
		if ref == model.TheProject {
			top = top.Content[1] // under project:
		}
		for path, list := range writtenPaths(t, entity, "", top) {
			ops, ok := served[path]
			switch {
			case !ok:
				t.Errorf("%s writes %s and is not served with it", ref, path)
			case list != slices.Contains(ops, "add"):
				t.Errorf("%s is served with %s taking %v", ref, path, ops)
			}
		}
		if _, ok := served["body"]; !ok && ref != model.TheProject {
			t.Errorf("%s is served without its body", ref)
		}
	}
}

// writtenPaths walks what the emitter wrote as store's own test does, and says of each
// path whether the model holds a list there.
func writtenPaths(t *testing.T, entity any, prefix string, node *yaml.Node) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	reach := func(path string, value *yaml.Node) {
		// a hypothesis's edits are the language a change is written in, which no path
		// reaches
		if path == "edits" {
			return
		}
		field, err := store.Resolve(entity, path)
		if err != nil {
			t.Errorf("the emitter writes %s and nothing can address it: %v", path, err)
			return
		}
		held := reflect.TypeOf(field.Get())
		out[path] = held.Kind() == reflect.Slice
		// a region is one value written as its four edges, and a child's relation one
		// value written as a word for each partner
		if value.Kind == yaml.SequenceNode && held.Kind() != reflect.Slice ||
			value.Kind == yaml.MappingNode && held == reflect.TypeFor[model.Relationship]() {
			return
		}
		for inside, list := range writtenPaths(t, entity, path, value) {
			out[inside] = list
		}
	}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			if prefix != "" {
				key = prefix + "." + key
			}
			reach(key, node.Content[i+1])
		}
	case yaml.SequenceNode:
		for i, entry := range node.Content {
			reach(prefix+"["+strconv.Itoa(i)+"]", entry)
		}
	}
	return out
}

// A record is served as the file spells it, pinned to what was shown.
func TestARecordIsWhatTheFileSays(t *testing.T) {
	t.Parallel()
	corpus := fixture(t)
	record, err := Read(corpus.Root, "verdier-jean-baptiste")
	if err != nil {
		t.Fatal(err)
	}
	ref := model.EntityRef{Kind: model.KindPerson, Slug: "verdier-jean-baptiste"}
	fields, err := store.Fields(corpus.Entity(ref.Slug))
	if err != nil {
		t.Fatal(err)
	}
	switch {
	case record.Ref != ref.String():
		t.Errorf("the record is %s", record.Ref)
	case !reflect.DeepEqual(record.Fields, fields):
		t.Errorf("the fields are\n%v\nnot\n%v", record.Fields, fields)
	case record.Seen != ops.Seen(corpus, ref):
		t.Errorf("the record is pinned to %s", record.Seen)
	case record.File != "people/verdier-jean-baptiste/README.md":
		t.Errorf("the record is kept in %s", record.File)
	case len(record.Attached) == 0:
		t.Error("the photographs beside him are not served")
	case record.Sorted["events[0]"] != "1834-03-12" || record.Sorted["events[2]"] != "1861-01-01":
		t.Errorf("his events are put in order at %v", record.Sorted)
	}

	manifest, err := Read(corpus.Root, "project")
	if err != nil {
		t.Fatal(err)
	}
	if manifest.File != store.ProjectFile || manifest.Fields["name"] == nil {
		t.Errorf("the manifest is served as %+v", manifest)
	}
	if _, err := Read(corpus.Root, "place/verdier-jean-baptiste"); err == nil {
		t.Error("a person is served as a place")
	}
}

// A corpus that did not read whole is refused in the words of what was asked, and a
// lock left behind stays beside the refusal for whoever says it last.
func TestARefusalKeepsWhatIsBesideIt(t *testing.T) {
	t.Parallel()
	lock := &history.Unreleased{Path: "origins.lock", Err: errors.New("busy")}
	err := refusing(errors.Join(&ops.Unreadable{Problems: errors.New("a: not a record")}, lock), "change a corpus")
	refused, rest := Without[*Unreadable](err)
	switch {
	case len(refused) != 1:
		t.Fatalf("refused as %v", err)
	case refused[0].Refusal != "origins will not change a corpus it could not read whole":
		t.Errorf("refused as %q", refused[0].Refusal)
	case !slices.Equal(refused[0].Problems, []string{"a: not a record"}):
		t.Errorf("the problems are %q", refused[0].Problems)
	case !errors.Is(rest, lock):
		t.Errorf("the lock left behind is gone: %v", rest)
	}
}
