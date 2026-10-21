package api

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/api/schema"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// The schema is held to the store by a validator of its own, here and nowhere else,
// so that origins takes on no dependency to describe what it already decides. It knows
// the keywords the schema uses and refuses a schema using any other, so that a keyword
// added to the schema and not to it cannot pass by being ignored.

// schemas is every schema read, by the name of its file.
type schemas map[string]map[string]any

func readSchemas(t *testing.T) schemas {
	t.Helper()
	out := schemas{}
	for _, name := range schema.Names() {
		read, err := schema.Of(name)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := json.Unmarshal(read, &doc); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out[name+".json"] = doc
	}
	return out
}

// known is every keyword the validator reads or may pass over.
var known = map[string]bool{
	"$schema": true, "$id": true, "$defs": true, "$ref": true, "title": true, "description": true,
	"type": true, "properties": true, "items": true, "required": true, "enum": true, "pattern": true,
	"oneOf": true, "additionalProperties": true, "minItems": true, "maxItems": true,
}

// resolve follows a reference from the document it is written in, to the document and
// the schema it names.
func (s schemas) resolve(doc, ref string) (string, map[string]any, error) {
	file, pointer, _ := strings.Cut(ref, "#")
	if file != "" {
		doc = file
	}
	schema, ok := s[doc]
	if !ok {
		return "", nil, fmt.Errorf("%s: no such schema", ref)
	}
	var at any = schema
	for _, step := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		if step == "" {
			continue
		}
		mapping, ok := at.(map[string]any)
		if !ok {
			return "", nil, fmt.Errorf("%s: %s is not there", ref, step)
		}
		at = mapping[step]
	}
	found, ok := at.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("%s leads to no schema", ref)
	}
	return doc, found, nil
}

// validate is what is wrong with an instance against a schema, nothing where nothing
// is. The instance is as YAML reads a corpus with every scalar text: a string, a list
// or a mapping.
func (s schemas) validate(doc string, sch map[string]any, instance any, at string) []string {
	var wrong []string
	for keyword := range sch {
		if !known[keyword] {
			wrong = append(wrong, fmt.Sprintf("%s: the validator does not know %s", at, keyword))
		}
	}
	if ref, ok := sch["$ref"].(string); ok {
		refDoc, target, err := s.resolve(doc, ref)
		if err != nil {
			return append(wrong, fmt.Sprintf("%s: %v", at, err))
		}
		wrong = append(wrong, s.validate(refDoc, target, instance, at)...)
	}
	if want, ok := sch["type"].(string); ok {
		var is bool
		switch want {
		case "string":
			_, is = instance.(string)
		case "array":
			_, is = instance.([]any)
		case "object":
			_, is = instance.(map[string]any)
		default:
			return append(wrong, fmt.Sprintf("%s: the validator does not know the type %s", at, want))
		}
		if !is {
			return append(wrong, fmt.Sprintf("%s: %v is not %s", at, instance, want))
		}
	}
	if enum, ok := sch["enum"].([]any); ok && !slices.Contains(enum, instance) {
		wrong = append(wrong, fmt.Sprintf("%s: %v is none of %v", at, instance, enum))
	}
	if pattern, ok := sch["pattern"].(string); ok {
		if text, ok := instance.(string); ok && !regexp.MustCompile(pattern).MatchString(text) {
			wrong = append(wrong, fmt.Sprintf("%s: %q does not match %s", at, text, pattern))
		}
	}
	if list, ok := instance.([]any); ok {
		if least, ok := sch["minItems"].(float64); ok && len(list) < int(least) {
			wrong = append(wrong, fmt.Sprintf("%s: fewer than %v entries", at, least))
		}
		if most, ok := sch["maxItems"].(float64); ok && len(list) > int(most) {
			wrong = append(wrong, fmt.Sprintf("%s: more than %v entries", at, most))
		}
		if items, ok := sch["items"].(map[string]any); ok {
			for i, entry := range list {
				wrong = append(wrong, s.validate(doc, items, entry, fmt.Sprintf("%s[%d]", at, i))...)
			}
		}
	}
	if mapping, ok := instance.(map[string]any); ok {
		properties, _ := sch["properties"].(map[string]any)
		for _, name := range slices.Sorted(maps.Keys(mapping)) {
			if property, ok := properties[name].(map[string]any); ok {
				wrong = append(wrong, s.validate(doc, property, mapping[name], at+"."+name)...)
				continue
			}
			switch additional := sch["additionalProperties"].(type) {
			case bool:
				if !additional {
					wrong = append(wrong, fmt.Sprintf("%s: %s is not a field", at, name))
				}
			case map[string]any:
				wrong = append(wrong, s.validate(doc, additional, mapping[name], at+"."+name)...)
			}
		}
		if required, ok := sch["required"].([]any); ok {
			for _, name := range required {
				if _, ok := mapping[name.(string)]; !ok {
					wrong = append(wrong, fmt.Sprintf("%s: %s is missing", at, name))
				}
			}
		}
	}
	if branches, ok := sch["oneOf"].([]any); ok {
		matched := 0
		for _, branch := range branches {
			if len(s.validate(doc, branch.(map[string]any), instance, at)) == 0 {
				matched++
			}
		}
		if matched != 1 {
			wrong = append(wrong, fmt.Sprintf("%s: %v matches %d of the %d shapes it may take", at, instance, matched, len(branches)))
		}
	}
	return wrong
}

// texts is a YAML node as the instance the validator reads: every scalar its text.
func texts(node *yaml.Node) any {
	switch node.Kind {
	case yaml.DocumentNode:
		return texts(node.Content[0])
	case yaml.AliasNode:
		return texts(node.Alias)
	case yaml.MappingNode:
		out := map[string]any{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			out[node.Content[i].Value] = texts(node.Content[i+1])
		}
		return out
	case yaml.SequenceNode:
		out := []any{}
		for _, entry := range node.Content {
			out = append(out, texts(entry))
		}
		return out
	}
	return node.Value
}

func frontmatter(t *testing.T, yamlText []byte) any {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal(yamlText, &node); err != nil {
		t.Fatal(err)
	}
	if len(node.Content) == 0 {
		return map[string]any{}
	}
	return texts(&node)
}

// generic is a value as JSON reads it back, the form the validator is given.
func generic(t *testing.T, value any) any {
	t.Helper()
	written, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(written, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// kinds is an entity of each kind with a schema, by the schema's name: the seven
// kinds of record and the manifest.
func kinds() map[string]any {
	out := map[string]any{"project.json": &model.Project{}}
	for _, kind := range model.Kinds {
		out[string(kind)+".json"] = model.New(kind)
	}
	return out
}

// paths is every path a schema names, [] for an entry of a list and * for a member of a
// mapping, following every shape a value may take.
func (s schemas) paths(doc string, sch map[string]any, at string, out map[string]bool, depth int) {
	if depth > 32 {
		panic("the schema refers to itself: " + at)
	}
	if ref, ok := sch["$ref"].(string); ok {
		refDoc, target, err := s.resolve(doc, ref)
		if err != nil {
			panic(err)
		}
		s.paths(refDoc, target, at, out, depth+1)
	}
	if properties, ok := sch["properties"].(map[string]any); ok {
		for name, property := range properties {
			path := name
			if at != "" {
				path = at + "." + name
			}
			out[path] = true
			s.paths(doc, property.(map[string]any), path, out, depth+1)
		}
	}
	if items, ok := sch["items"].(map[string]any); ok {
		s.paths(doc, items, at+"[]", out, depth+1)
	}
	if additional, ok := sch["additionalProperties"].(map[string]any); ok {
		s.paths(doc, additional, at+".*", out, depth+1)
	}
	if branches, ok := sch["oneOf"].([]any); ok {
		for _, branch := range branches {
			s.paths(doc, branch.(map[string]any), at, out, depth+1)
		}
	}
}

// plain is a path with nothing in it of how a value is written, list or mapping, since
// the schema says a value written in a word or as a list is one value where the model
// holds a list, and the other way about.
func plain(path string) string {
	return strings.TrimSuffix(strings.ReplaceAll(path, "[]", ""), ".*")
}

// unaddressed is the one thing a schema names that is no path, with everything in it: a
// hypothesis's edits are what it says and not something it is, and are changed by
// writing another.
func unaddressed(name, path string) bool {
	return name == "hypothesis.json" && (path == "edits" || strings.HasPrefix(path, "edits[]"))
}

// The schema describes the corpus the store reads and writes: everything the emitter
// writes of the fixture is valid, the way Fields spells it and the way the file does,
// every object in the schema says what else it takes, so that a field the emitter writes
// and the schema does not name is refused rather than passed, and every path the schema
// names is one Resolve reaches.
func TestTheSchemaIsTheStore(t *testing.T) {
	t.Parallel()
	s := readSchemas(t)

	for doc, sch := range s {
		var openEnded func(node any, at string)
		openEnded = func(node any, at string) {
			switch n := node.(type) {
			case map[string]any:
				if _, ok := n["properties"]; ok {
					if _, ok := n["additionalProperties"]; !ok {
						t.Errorf("%s %s takes fields it does not name", doc, at)
					}
				}
				if n["type"] == "object" {
					if _, ok := n["additionalProperties"]; !ok {
						t.Errorf("%s %s takes fields it does not name", doc, at)
					}
				}
				for _, key := range slices.Sorted(maps.Keys(n)) {
					openEnded(n[key], at+"/"+key)
				}
			case []any:
				for i, entry := range n {
					openEnded(entry, fmt.Sprintf("%s/%d", at, i))
				}
			}
		}
		openEnded(sch, "#")
	}

	corpus := fixture(t)
	root := filepath.Join("..", "..", "test")
	refs := []model.EntityRef{model.TheProject}
	for _, slug := range corpus.Slugs() {
		refs = append(refs, model.RefOf(corpus.Entity(slug)))
	}
	for _, ref := range refs {
		name := string(ref.Kind) + ".json"
		entity := any(&corpus.Project)
		if ref != model.TheProject {
			entity = corpus.Entity(ref.Slug)
		}
		fields, err := store.Fields(entity)
		if err != nil {
			t.Fatalf("%s: %v", ref, err)
		}
		for _, wrong := range s.validate(name, s[name], generic(t, fields), ref.String()) {
			t.Errorf("as Fields spells it: %s", wrong)
		}

		var written any
		if ref == model.TheProject {
			read, err := os.ReadFile(filepath.Join(root, store.ProjectFile))
			if err != nil {
				t.Fatal(err)
			}
			written = frontmatter(t, read).(map[string]any)["project"]
		} else {
			at, err := store.PathOf(entity)
			if err != nil {
				t.Fatal(err)
			}
			read, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(at)))
			if err != nil {
				t.Fatal(err)
			}
			head, _, err := store.Split(read)
			if err != nil {
				t.Fatal(err)
			}
			written = frontmatter(t, head)
		}
		for _, wrong := range s.validate(name, s[name], written, ref.String()) {
			t.Errorf("as the file writes it: %s", wrong)
		}
	}

	for name, entity := range kinds() {
		reached := map[string]bool{}
		for _, path := range store.Shape(entity) {
			reached[plain(path)] = true
		}
		named := map[string]bool{}
		s.paths(name, s[name], "", named, 0)
		for _, path := range slices.Sorted(maps.Keys(named)) {
			if plain(path) == "" || reached[plain(path)] || unaddressed(name, path) {
				continue
			}
			t.Errorf("%s names %s, which no path reaches", name, path)
		}
	}
}

// What the reader refuses for its shape the schema refuses too. Each file is read first,
// to show that the reader does refuse it, so that a case here cannot be one the reader
// came to accept. What the schema cannot say is not here: a reserved word where a slug
// goes, a value with none of the things it may hold, a key a service or a tag may not
// be spelled with, a language that is no tag and coordinates that are no angles.
func TestTheSchemaRefusesWhatTheReaderRefuses(t *testing.T) {
	t.Parallel()
	s := readSchemas(t)
	for _, c := range []struct{ path, frontmatter string }{
		{"people/a/README.md", "name: Giovanni\nnickname: Gio\n"},
		{"people/a/README.md", "living: yes\n"},
		{"people/a/README.md", "living: True\n"},
		{"people/a/README.md", "sex: male\n"},
		{"people/a/README.md", "name:\n  private: true\n  given: Anna\n"},
		{"people/a/README.md", "events:\n  - type: baptism\n    with:\n      - name:\n          private: true\n          given: Pietro\n        role: godfather\n"},
		{"people/a/README.md", "events:\n  - type: baptism\n    with:\n      - person: b\n        name: Pietro\n        role: godfather\n"},
		{"people/a/README.md", "events:\n  - type: birth\n    sources: [register]\n"},
		{"people/a/README.md", "events:\n  - type: death\n    age: very old\n"},
		{"people/a/README.md", "same-as: [{id: b, name: Pietro}]\n"},
		{"people/a/README.md", "same-as: [Pietro Rossi]\n"},
		{"people/a/README.md", "portrait: {file: a.jpg, region: [0.1, 0.2, 0.3]}\n"},
		{"people/a/README.md", "tags: {a: b}\n"},
		{"places/firenze/README.md", "private: yes\nname: Firenze\n"},
		{"places/firenze/README.md", "private: false\nname: Firenze\n"},
		{"families/a--b/README.md", "partners: [a, b]\nchildren: [{id: c, name: Elena Rossi}]\n"},
		{"families/a--b/README.md", "partners: [a, b]\nchildren: [{relation: adoptive}]\n"},
		{"families/a--b/README.md", "partners: [a, b]\nchildren: [{id: c, status: likely}]\n"},
		{"families/a--b/README.md", "partners: [a, b]\nchildren: [{id: c, kind: birth}]\n"},
		{"sources/register/README.md", "citations:\n  a:\n    images:\n      - file: scans/a.jpg\n        region: [0.1, 0.2, 0.3, top]\n"},
		{"sources/register/README.md", "citations:\n  a:\n    images:\n      - file: scans/a.jpg\n        region: 0.1, 0.2, 0.3, 0.4\n"},
		{"hypotheses/h/README.md", "status: maybe\nedits: []\n"},
		{"hypotheses/h/README.md", "status: open\nedits:\n  - op: guess\n    entity: people/a\n"},
	} {
		t.Run(c.path+" "+strings.ReplaceAll(c.frontmatter, "\n", " "), func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			write := func(path, content string) {
				full := filepath.Join(root, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			write(store.ProjectFile, "project:\n  name: x\norigins:\n  format-version: 1\n")
			write(c.path, "---\n"+c.frontmatter+"---\n")
			if _, err := store.Load(root); err == nil {
				t.Fatal("the reader reads it")
			}
			kind, _, _ := strings.Cut(c.path, "/")
			name := ""
			for _, k := range model.Kinds {
				if k.Dir() == kind {
					name = string(k) + ".json"
				}
			}
			if len(s.validate(name, s[name], frontmatter(t, []byte(c.frontmatter)), c.path)) == 0 {
				t.Error("the schema takes what the reader refuses")
			}
		})
	}
	for _, bad := range []string{
		"bookmarks: [person/rossi-giovanni]",
		"bookmarks: [rossi-giovanni]",
		"bookmarks: [people/Rossi Giovanni]",
		"researchers: [Anna Rossi]",
		"researchers: [{note: Compiled it}]",
		"researchers: [{name: Anna Rossi, addresses: [{city: Firenze, sources: [census-1871]}]}]",
		"researchers: [{name: Anna Rossi, email: anna@example.org}]",
		"tags: {To do: {colour: red}}",
	} {
		t.Run("origins.yaml "+bad, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			manifest := "project:\n  name: x\n  " + bad + "\norigins:\n  format-version: 1\n"
			if err := os.WriteFile(filepath.Join(root, store.ProjectFile), []byte(manifest), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(root); err == nil {
				t.Fatal("the reader reads it")
			}
			project := frontmatter(t, []byte(manifest)).(map[string]any)["project"]
			if len(s.validate("project.json", s["project.json"], project, "project")) == 0 {
				t.Error("the schema takes what the reader refuses")
			}
		})
	}
}
