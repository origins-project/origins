package store

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/model"
)

func set(t *testing.T, entity any, path string, value any) {
	t.Helper()
	field, err := Resolve(entity, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := field.Set(value); err != nil {
		t.Fatal(err)
	}
}

// The path language is the on-disk spelling, so every key the emitter writes has to be
// a key a path can reach. Nothing links the two but this test: the emitter is a hand
// written run of field names and so is the table behind Resolve, and a field added to
// one and not the other leaves a corpus origins can write and cannot edit.
//
// It runs over the fixture because the fixture is where the awkward shapes are — a
// place emitted with a bare within, a name with dates on it, a source's anchors, a
// person with two photographs — and a table of invented entities would only cover the
// shapes whoever wrote the table thought of.
func TestEveryFieldTheEmitterWritesCanBeAddressed(t *testing.T) {
	corpus := fixture(t)
	for _, slug := range corpus.Slugs() {
		entity := corpus.Entity(slug)
		written, err := Emit(entity)
		if err != nil {
			t.Fatal(err)
		}
		frontmatter, _, err := Split(written)
		if err != nil {
			t.Fatal(err)
		}
		var document yaml.Node
		if err := yaml.Unmarshal(frontmatter, &document); err != nil {
			t.Fatal(err)
		}
		if len(document.Content) == 0 {
			continue
		}
		addressed(t, entity, string(slug), "", document.Content[0])
	}

	// the manifest is addressed by the paths it spells under project, and what is
	// under origins is no edit's to change
	written, err := EmitProject(corpus.Project)
	if err != nil {
		t.Fatal(err)
	}
	var manifest yaml.Node
	if err := yaml.Unmarshal(written, &manifest); err != nil {
		t.Fatal(err)
	}
	top := manifest.Content[0].Content
	if top[0].Value != "project" {
		t.Fatalf("the manifest begins with %s and not project", top[0].Value)
	}
	addressed(t, &corpus.Project, ProjectFile, "", top[1])
	if _, err := Resolve(&corpus.Project, "format-version"); err == nil {
		t.Error("format-version is addressable, and only fmt changes it")
	}
}

// A tag's style is a mapping keyed by the tag as written, so a path steps into it by
// the word, and a style left saying nothing goes, a tag needing none to be used.
func TestATagsStyleIsAddressedByTheTag(t *testing.T) {
	project := model.Project{Tags: map[string]model.TagStyle{"To do": {Color: "#d33", Priority: "1"}}}
	color, err := Resolve(&project, "tags.To do.color")
	if err != nil {
		t.Fatal(err)
	}
	if err := color.Set(Typed("#a00")); err != nil {
		t.Fatal(err)
	}
	if got := project.Tags["To do"].Color; got != "#a00" {
		t.Errorf("the color is %q", got)
	}

	fresh, err := Resolve(&project, "tags.Emigrants.priority")
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.Set(Typed("2")); err != nil {
		t.Fatal(err)
	}
	if got := project.Tags["Emigrants"]; got != (model.TagStyle{Priority: "2"}) {
		t.Errorf("a style set for a tag that had none is %+v", got)
	}
	if err := fresh.Unset(); err != nil {
		t.Fatal(err)
	}
	if _, kept := project.Tags["Emigrants"]; kept {
		t.Error("a style saying nothing is kept")
	}

	whole, err := Resolve(&project, "tags")
	if err != nil {
		t.Fatal(err)
	}
	if err := whole.Set(Typed("{a.b: {color: red}}")); err != nil {
		t.Fatal(err)
	}
	if got := project.Tags; len(got) != 1 || got["a.b"].Color != "red" {
		t.Errorf("tags set whole are %+v", got)
	}
	if err := whole.Set(Typed("{' ': {color: red}}")); err == nil {
		t.Error("a tag with no word is styled")
	}
}

// addressed walks what the emitter wrote and asks for every path in it.
func addressed(t *testing.T, entity any, slug, prefix string, node *yaml.Node) {
	t.Helper()
	reach := func(path string, value *yaml.Node) {
		// A hypothesis's edits are the one thing written that no path reaches. They are
		// the language a change is expressed in, and an edit rewriting another edit is
		// not a thing the corpus has any use for.
		if path == "edits" {
			return
		}
		field, err := Resolve(entity, path)
		if err != nil {
			t.Errorf("%s: the emitter writes %s and nothing can address it: %v", slug, path, err)
			return
		}
		// a region is written as the list of its four edges, and a child's relation as
		// a word for each partner, and each is one value, so a path reaches the region
		// and not an edge of it, the relation and not a partner's word
		if value.Kind == yaml.SequenceNode && field.value.Kind() != reflect.Slice ||
			value.Kind == yaml.MappingNode && field.value.Type() == reflect.TypeFor[model.Relationship]() {
			return
		}
		addressed(t, entity, slug, path, value)
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
}

// Prose is how a rename finds the links in a record, how check follows them and how
// todo finds a task, so every note and transcription the emitter writes has to be one
// that model.Prose hands out, at the path that reaches it. The fixture leaves some
// forms without a note, so a record with a note on every one goes through as well.
func TestProseIsWhatTheEmitterWritesAsProse(t *testing.T) {
	corpus := fixture(t)
	var entities []any
	for _, slug := range corpus.Slugs() {
		entities = append(entities, corpus.Entity(slug))
	}
	name := model.PersonName{Given: "Maria", Note: "n"}
	named := model.Named{Name: name, Role: "witness", Note: "n"}
	media := []model.Media{{File: "a.jpg", Note: "n", Depicts: []model.Depicted{{Person: "p", Note: "n"}}}}
	entities = append(entities,
		&model.Person{Slug: "p", Name: name, AlsoKnownAs: []model.PersonName{name}, Body: "b", Media: media,
			Events: []model.Event{{Type: "birth", Note: "n", With: []model.Named{named}}}},
		&model.Family{Slug: "f", Partners: []model.Partner{{Name: name, Note: "n"}, {Person: model.None}},
			Children: []model.Child{{Name: name, Note: "n"}}, Body: "b"},
		&model.Place{Slug: "q", Name: model.PlaceName{Text: "Vicchio"}, AlsoKnownAs: []model.PlaceName{{Text: "V", Note: "n"}},
			Within: []model.PlaceRef{{Place: "toscana", Note: "n"}}, Body: "b"},
		&model.Source{Slug: "s", Citations: map[string]model.Citation{"a": {Text: "t", Note: "n"}}, Body: "b"},
		&model.Repository{Slug: "r", Media: media, Body: "b",
			SameAs: []model.SameAs{{ID: "s", Note: "n"}, {Name: "Archivio", Note: "n"}}},
		&model.Note{Slug: "n", Events: []model.Event{{Type: "marriage", Note: "n", With: []model.Named{named}}},
			Media: media, Body: "b"},
		&model.Hypothesis{Slug: "h", Body: "b"})

	for _, entity := range entities {
		slug := model.RefOf(entity).String()
		handed := map[string]bool{}
		model.Prose(entity, func(path string, text *string) {
			field, err := Resolve(entity, path)
			if err != nil {
				t.Errorf("%s: Prose hands out %s and nothing can address it: %v", slug, path, err)
				return
			}
			if got := field.Get(); got != *text {
				t.Errorf("%s: %s is %q, and Prose handed out %q", slug, path, got, *text)
			}
			if *text != "" {
				handed[path] = true
			}
		})
		written, err := Emit(entity)
		if err != nil {
			t.Fatal(err)
		}
		frontmatter, body, err := Split(written)
		if err != nil {
			t.Fatal(err)
		}
		var document yaml.Node
		if err := yaml.Unmarshal(frontmatter, &document); err != nil {
			t.Fatal(err)
		}
		wrote := map[string]bool{}
		if strings.TrimSpace(string(body)) != "" {
			wrote["body"] = true
		}
		if len(document.Content) > 0 {
			writtenProse(document.Content[0], "", wrote)
		}
		for path := range wrote {
			if !handed[path] {
				t.Errorf("%s: the emitter writes prose at %s and Prose does not hand it out", slug, path)
			}
		}
		for path := range handed {
			if !wrote[path] {
				t.Errorf("%s: Prose hands out %s and the emitter writes nothing there", slug, path)
			}
		}
	}
}

// writtenProse collects the paths of every note the emitter wrote, and of a citation's
// transcription.
func writtenProse(node *yaml.Node, prefix string, out map[string]bool) {
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			if key == "note" || key == "text" && strings.HasPrefix(prefix, "citations.") {
				out[path] = true
				continue
			}
			if key != "edits" {
				writtenProse(node.Content[i+1], path, out)
			}
		}
	case yaml.SequenceNode:
		for i, entry := range node.Content {
			writtenProse(entry, prefix+"["+strconv.Itoa(i)+"]", out)
		}
	}
}

// model.Tags is held to the emitter as model.Prose is: every list it hands out is one a
// path reaches, and every tags the emitter writes is one of them, so a tag added to a
// form and missed there is a word no rename rewrites.
func TestTagsAreWhatTheEmitterWritesAsTags(t *testing.T) {
	corpus := fixture(t)
	var entities []any
	for _, slug := range corpus.Slugs() {
		entities = append(entities, corpus.Entity(slug))
	}
	tagged := []string{"To do"}
	media := []model.Media{{File: "a.jpg", Tags: tagged}}
	events := []model.Event{{Type: "birth", Media: media, Tags: tagged}}
	entities = append(entities,
		&model.Person{Slug: "p", Name: model.PersonName{Given: "Maria"}, Events: events, Media: media, Tags: tagged},
		&model.Family{Slug: "f", Partners: []model.Partner{{Person: model.Unknown}, {Person: model.None}},
			Events: events, Media: media, Tags: tagged},
		&model.Place{Slug: "q", Name: model.PlaceName{Text: "Vicchio"}, Media: media, Tags: tagged},
		&model.Source{Slug: "s", Title: "Battesimi", Media: media, Tags: tagged,
			Citations: map[string]model.Citation{"a": {Page: "1", Images: media, Tags: tagged}}},
		&model.Repository{Slug: "r", Name: "Archivio", Media: media, Tags: tagged},
		&model.Note{Slug: "n", Events: events, Media: media, Tags: tagged},
		&model.Hypothesis{Slug: "h", Body: "b"})

	for _, entity := range entities {
		slug := model.RefOf(entity).String()
		handed := map[string]bool{}
		model.Tags(entity, func(path string, tags *[]string) {
			field, err := Resolve(entity, path)
			if err != nil {
				t.Errorf("%s: Tags hands out %s and nothing can address it: %v", slug, path, err)
				return
			}
			if got := field.Get(); !reflect.DeepEqual(got, *tags) {
				t.Errorf("%s: %s is %q, and Tags handed out %q", slug, path, got, *tags)
			}
			if len(*tags) > 0 {
				handed[path] = true
			}
		})
		written, err := Emit(entity)
		if err != nil {
			t.Fatal(err)
		}
		frontmatter, _, err := Split(written)
		if err != nil {
			t.Fatal(err)
		}
		var document yaml.Node
		if err := yaml.Unmarshal(frontmatter, &document); err != nil {
			t.Fatal(err)
		}
		wrote := map[string]bool{}
		if len(document.Content) > 0 {
			writtenTags(document.Content[0], "", wrote)
		}
		for path := range wrote {
			if !handed[path] {
				t.Errorf("%s: the emitter writes tags at %s and Tags does not hand them out", slug, path)
			}
		}
		for path := range handed {
			if !wrote[path] {
				t.Errorf("%s: Tags hands out %s and the emitter writes nothing there", slug, path)
			}
		}
	}
}

// writtenTags collects the paths of every list of tags the emitter wrote.
func writtenTags(node *yaml.Node, prefix string, out map[string]bool) {
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			if key == "tags" {
				out[path] = true
				continue
			}
			if key != "edits" {
				writtenTags(node.Content[i+1], path, out)
			}
		}
	case yaml.SequenceNode:
		for i, entry := range node.Content {
			writtenTags(entry, prefix+"["+strconv.Itoa(i)+"]", out)
		}
	}
}

// A value an edit carries is read by the code that reads a file, so the grammar is one
// grammar: 1834 is a year rather than an integer, a child is a slug or a mapping, a
// relation is a word or a word for each partner, and none of that is written twice.
func TestAValueIsReadTheWayTheCorpusReadsIt(t *testing.T) {
	person := &model.Person{Slug: "rossi-giovanni"}
	set(t, person, "name", "Giovanni")
	set(t, person, "sex", "M")
	set(t, person, "events", []any{map[string]any{"type": "birth", "date": "1834"}})
	set(t, person, "events[0].place", "firenze")
	family := &model.Family{Slug: "rossi-antonio--verdi-lucia"}
	set(t, family, "children", []any{
		"rossi-giovanni",
		map[string]any{"id": "rossi-elena", "relation": map[string]any{"rossi-antonio": "adoptive"}},
	})

	switch {
	case person.Name.Text != "Giovanni":
		t.Errorf("name = %+v", person.Name)
	case person.Sex.Value != model.Male:
		t.Errorf("sex = %v", person.Sex)
	case len(person.Events) != 1 || person.Events[0].Date.String() != "1834":
		t.Errorf("events = %+v", person.Events)
	case person.Events[0].Place != "firenze":
		t.Errorf("place = %q", person.Events[0].Place)
	case len(family.Children) != 2:
		t.Fatalf("children = %+v", family.Children)
	case !reflect.DeepEqual(family.Children[0], model.Child{ID: "rossi-giovanni"}):
		t.Errorf("the bare child read as %+v", family.Children[0])
	case !reflect.DeepEqual(family.Children[1], model.Child{ID: "rossi-elena", Relation: model.Relationship{
		Partners: []model.PartnerRelation{{Partner: "rossi-antonio", Relation: model.Adoptive}}}}):
		t.Errorf("the qualified child read as %+v", family.Children[1])
	}

	place := &model.Place{Slug: "firenze"}
	set(t, place, "coordinates", "43.7696, 11.2558")
	if place.Coordinates == nil || place.Coordinates.Latitude != 43.7696 {
		t.Errorf("coordinates = %+v", place.Coordinates)
	}
	if err := mustFail(place, "coordinates", "somewhere in Tuscany"); err == nil {
		t.Error("a coordinate that is not a coordinate was accepted")
	}
	set(t, place, "coordinates", "(N43.77.5,  E11.25.3)")
	if place.Coordinates == nil || place.Coordinates.Placed() || place.Coordinates.Wording != "N43.77.5, E11.25.3" {
		t.Errorf("coordinates as words = %+v", place.Coordinates)
	}
	for _, words := range []string{"(N43.77.5, E11.25.3", "()", "NaN, 11.2558", "43.7696, nan", "91, 0", "0, -Inf"} {
		if err := mustFail(place, "coordinates", words); err == nil {
			t.Errorf("coordinates %q were accepted", words)
		}
	}

	set(t, place, "name", "Firenze")
	set(t, place, "name.lang", "it")
	set(t, place, "name.from", "1865")
	if got := place.Name; got.Text != "Firenze" || got.Lang != "it" || got.From.String() != "1865" {
		t.Errorf("the place's own name = %+v", got)
	}
	set(t, place, "name.name", "")
	if _, err := Emit(place); err == nil || !strings.Contains(err.Error(), unnamedPlace) {
		t.Errorf("a name of no words was written: %v", err)
	}
	unset(t, place, "name")

	set(t, place, "also-known-as", Typed("[Florence]"))
	set(t, place, "also-known-as[0].lang", "EN")
	add(t, place, "also-known-as[0].sources", "censimento-1861-firenze#f088")
	set(t, place, "also-known-as[0].type", "exonym")
	if got := place.AlsoKnownAs[0]; got.Lang != "en" || got.Type != "exonym" || len(got.Sources) != 1 {
		t.Errorf("the English name = %+v", got)
	}
	if err := mustFail(place, "also-known-as[0].lang", "English"); err == nil {
		t.Error("a language that is not a language tag was accepted")
	}
	set(t, place, "within", Typed("[toscana]"))
	add(t, place, "within[0].sources", "censimento-1861-firenze#f088")
	add(t, place, "sources", "censimento-1861-firenze#f088")
	if len(place.Within[0].Sources) != 1 || len(place.Sources) != 1 {
		t.Errorf("within = %+v, sources = %+v", place.Within, place.Sources)
	}
}

// What a file may not say, an edit may not say either. The wire types are strict about
// an unknown field because every write re-emits the file and a field origins did not
// read would be deleted from it; an edit reaching the model by another road would be a
// way round that.
func TestAValueTheCorpusCouldNotHoldIsRefused(t *testing.T) {
	person := &model.Person{Slug: "rossi-giovanni"}
	for _, refused := range []struct {
		what  string
		path  string
		value any
	}{
		{"a sex that is not one", "sex", "male"},
		{"a slug with a space in it", "events", []any{map[string]any{"place": "borgo san lorenzo"}}},
		{"a field nobody knows", "events", []any{map[string]any{"type": "birth", "kinship": "aunt"}}},
		{"a field of an event nobody knows", "events", []any{map[string]any{"type": "birth", "parish": "San Lorenzo"}}},
		{"a list set from a scalar", "events", "rossi-antonio"},
		{"a scalar set from a list", "name", []any{"Giovanni"}},
		{"a sex cited that nothing states", "sex", Typed("{value: U, sources: [censimento-1861-firenze#f088]}")},
		{"a service no path could reach", "ids", map[string]any{"Family Search": "XXXX-YYY"}},
		{"an identifier that is nothing", "ids", map[string]any{"familysearch": ""}},
		{"an age that is not one", "events", Typed("[{type: death, age: about seventy}]")},
		{"somebody named twice over", "events", Typed("[{type: baptism, with: [{person: verdi-giacomo, name: Giacomo Verdi, role: godfather}]}]")},
		{"somebody named by nothing", "events", Typed("[{type: baptism, with: [{role: godfather}]}]")},
		{"somebody named in no role", "events", Typed("[{type: baptism, with: [{person: verdi-giacomo}]}]")},
		{"somebody named in a word", "events", Typed("[{type: baptism, with: [verdi-giacomo]}]")},
		{"nobody named", "events", Typed("[{type: baptism, with: [{person: none, role: witness}]}]")},
		{"an age cited that nothing states", "events", Typed("[{type: death, age: {sources: [atti-morte#f12]}}]")},
		{"a cause said something of that nothing states", "events", Typed("[{type: death, cause: {note: illegible}}]")},
		{"an attribute of no type", "events", Typed("[{type: birth, attributes: [{value: stillborn}]}]")},
		{"an attribute written as a word", "events", Typed("[{type: birth, attributes: [stillborn]}]")},
		{"a field of an attribute nobody knows", "events", Typed("[{type: birth, attributes: [{type: classification, place: firenze}]}]")},
		{"what did not happen of no type", "not", Typed("[{date: FROM 1880 TO 1902}]")},
		{"what did not happen with a value", "not", Typed("[{type: marriage, value: twice}]")},
		{"a flag that is not the word true", "private", Typed("yes")},
		{"a flag set false rather than left out", "private", Typed("false")},
		{"living as a word that is not true or false", "living", Typed("no")},
		{"the same as a record and a name at once", "same-as", Typed("[{id: rossi-giovanni-1834, name: Giovanni Rossi}]")},
		{"the same as nothing", "same-as", Typed("[{note: The same baptism.}]")},
		{"the same as nobody known", "same-as", Typed("[unknown]")},
		{"the same as a field nobody knows", "same-as", Typed("[{id: rossi-giovanni-1834, page: 12}]")},
		{"a tag that is a list", "events", Typed("[{type: birth, tags: [[To do]]}]")},
		{"a tag that is nothing", "media", Typed("[{file: a.jpg, tags: ['']}]")},
	} {
		if err := mustFail(person, refused.path, refused.value); err == nil {
			t.Errorf("%s was accepted", refused.what)
		}
	}
	family := &model.Family{}
	for _, refused := range []struct{ what, path, value string }{
		{"a partner written as words", "partners", "[Mr Stockdale]"},
		{"a partner named twice over", "partners", "[{person: rossi-giovanni, name: Giovanni Rossi}]"},
		{"a partner named by nothing", "partners", "[{note: His second marriage.}]"},
		{"a partner known by a private name", "partners", "[{name: {given: Giovanni, private: true}}]"},
	} {
		if err := mustFail(family, refused.path, Typed(refused.value)); err == nil {
			t.Errorf("%s was accepted", refused.what)
		}
	}
	if err := mustFail(family, "type", Typed("handfasting")); err != nil {
		t.Errorf("a family's type is an open word: %v", err)
	}
}

func mustFail(entity any, path string, value any) error {
	field, err := Resolve(entity, path)
	if err != nil {
		return err
	}
	return field.Set(value)
}

// Every record but a hypothesis can be marked private, and a mark is set by the word
// and taken away by leaving the field out.
func TestARecordIsMarkedPrivateAndUnmarked(t *testing.T) {
	for _, kind := range model.Kinds {
		if kind == model.KindHypothesis {
			continue
		}
		record := model.New(kind)
		set(t, record, "private", Typed("true"))
		field, err := Resolve(record, "private")
		if err != nil {
			t.Fatal(err)
		}
		if field.Get() != true {
			t.Errorf("%T is not private once marked", record)
		}
		if err := field.Unset(); err != nil {
			t.Fatal(err)
		}
		if file, _ := Emit(record); strings.Contains(string(file), "private") {
			t.Errorf("%T is still written private once the mark is taken away:\n%s", record, file)
		}
	}
	if _, err := Resolve(&model.Hypothesis{}, "private"); err == nil {
		t.Error("a hypothesis is not shared or kept back, and has no mark to set")
	}
}

// living is set either way and, taken away, leaves the rule to decide. It is written
// after private, so the first lines of a person say whether they leave the machine.
func TestAPersonIsMarkedLivingOrDeceased(t *testing.T) {
	person := &model.Person{Private: true, Name: model.PersonName{Given: "Ottavio", Surname: model.SurnameOf("Landi")}}
	for _, word := range []string{"true", "false"} {
		set(t, person, "living", Typed(word))
		file, err := Emit(person)
		if err != nil {
			t.Fatal(err)
		}
		if want := "---\nprivate: true\nliving: " + word + "\nname:"; !strings.HasPrefix(string(file), want) {
			t.Errorf("living: %s is written\n%s\nwant it to start\n%s", word, file, want)
		}
	}
	field, err := Resolve(person, "living")
	if err != nil {
		t.Fatal(err)
	}
	if err := field.Unset(); err != nil {
		t.Fatal(err)
	}
	if person.Living != model.LivingPresumed {
		t.Errorf("living is %q once taken away", person.Living)
	}
	if _, err := Resolve(&model.Family{}, "living"); err == nil {
		t.Error("a family has no life to presume, and no living to set")
	}
}

// An event, another name and a picture are each marked by a path into the entry. A name
// as a record wrote it takes no mark, as it takes no type, and neither does a name that
// stands for somebody.
func TestAnEntryIsMarkedPrivateAndUnmarked(t *testing.T) {
	anna := func() *model.Person {
		return &model.Person{
			Name:        model.PersonName{Given: "Anna", Surname: model.SurnameOf("Rossi")},
			AlsoKnownAs: []model.PersonName{{Given: "Nina"}},
			Events: []model.Event{{Type: "baptism", With: []model.Named{
				{Name: model.PersonName{Given: "Pietro"}, Role: "godfather"},
			}}},
			Media: []model.Media{{File: "photos/a.jpg"}},
		}
	}
	person := anna()
	for _, path := range []string{"events[0].private", "also-known-as[0].private", "media[0].private"} {
		set(t, person, path, Typed("true"))
		if file, _ := Emit(person); !strings.Contains(string(file), "- private: true\n") {
			t.Errorf("%s is not written first in its entry:\n%s", path, file)
		}
		field, err := Resolve(person, path)
		if err != nil {
			t.Fatal(err)
		}
		if err := field.Unset(); err != nil {
			t.Fatal(err)
		}
		if file, _ := Emit(person); strings.Contains(string(file), "private") {
			t.Errorf("%s is still written once the mark is taken away:\n%s", path, file)
		}
	}

	for _, path := range []string{"name.private", "events[0].with[0].name.private"} {
		marked := anna()
		set(t, marked, path, Typed("true"))
		if _, err := Emit(marked); err == nil || !strings.Contains(err.Error(), "only a name in also-known-as is private") {
			t.Errorf("%s was written: %v", path, err)
		}
	}
	bare := &model.Person{AlsoKnownAs: []model.PersonName{{Text: "Annetta"}}}
	set(t, bare, "also-known-as[0].private", Typed("true"))
	if got, err := Emit(bare); err != nil || !strings.Contains(string(got), "  - private: true\n    text: Annetta\n") {
		t.Errorf("a name as a record wrote it was marked private as %v\n%s", err, got)
	}
}

// Text nobody has read yet is read as the field it is going into, because nothing else
// can tell the cases apart: "Stato civile: registri" is the title of a register and
// YAML would make a mapping of it, and "43.7696, 11.2558" is a pair of coordinates and
// not a sentence. A field that holds text takes the characters; everywhere else the
// text is YAML and is read the way a file is read, down to refusing what a file could
// not have said.
func TestTextIsReadAsTheFieldItGoesInto(t *testing.T) {
	source := &model.Source{Slug: "firenze-civil-deaths-1901"}
	set(t, source, "title", Typed("Stato civile: registri di morte, 1901"))
	if source.Title != "Stato civile: registri di morte, 1901" {
		t.Errorf("title = %q", source.Title)
	}

	person := &model.Person{Slug: "rossi-giovanni"}
	set(t, person, "body", Typed("---\nwhat looks like a fence\n---"))
	set(t, person, "sex", Typed("M"))
	set(t, person, "events", Typed("[{type: birth, date: 1834}]"))
	family := &model.Family{Slug: "rossi-antonio--verdi-lucia"}
	set(t, family, "children", Typed("[rossi-giovanni, {id: rossi-elena, relation: adoptive}]"))
	switch {
	case person.Body != "---\nwhat looks like a fence\n---":
		t.Errorf("body = %q", person.Body)
	case person.Sex.Value != model.Male:
		t.Errorf("sex = %v", person.Sex)
	case len(person.Events) != 1 || person.Events[0].Date.String() != "1834":
		t.Errorf("events = %+v", person.Events)
	case len(family.Children) != 2 || family.Children[1].Relation.Word != model.Adoptive:
		t.Errorf("children = %+v", family.Children)
	}

	place := &model.Place{Slug: "firenze"}
	set(t, place, "coordinates", Typed("43.7696, 11.2558"))
	if place.Coordinates == nil || place.Coordinates.Latitude != 43.7696 {
		t.Fatalf("coordinates = %+v", place.Coordinates)
	}
	// a pointer is the one field where absent and zero look different, and nothing
	// typed is absent rather than a point in the Gulf of Guinea
	set(t, place, "coordinates", Typed(""))
	if place.Coordinates != nil {
		t.Errorf("typing nothing left %+v behind", place.Coordinates)
	}

	for _, refused := range []struct {
		what  string
		path  string
		value Typed
	}{
		{"a sex that is not one", "sex", "male"},
		{"a field of an event nobody knows", "events", "[{type: birth, parish: San Lorenzo}]"},
		{"a list set from a scalar", "events", "rossi-antonio"},
	} {
		if err := mustFail(person, refused.path, refused.value); err == nil {
			t.Errorf("%s was accepted", refused.what)
		}
	}
}

// A path that reaches nothing says which part of it did not work out, because the
// reader has a record in front of them and needs to know where their idea of it and
// the corpus part company.
func TestAPathThatReachesNothingSaysSo(t *testing.T) {
	person := &model.Person{
		Slug:   "rossi-giovanni",
		Events: []model.Event{{Type: "birth"}},
	}
	for _, refused := range []struct {
		why  string
		path string
	}{
		{"nothing at all", ""},
		{"a field that does not exist", "birthplace"},
		{"a field of a field that does not exist", "events[0].parish"},
		{"an index past the end", "events[1].date"},
		{"an index on something that is not a list", "name[0]"},
		{"a bracket that never closes", "events[0.date"},
		{"an index that is not a number", "events[first]"},
		{"an empty name", "events[0]."},
	} {
		if _, err := Resolve(person, refused.path); err == nil {
			t.Errorf("%s was resolved", refused.why)
		}
	}
}

// Unsetting an entry of a list takes it out rather than blanking it, and a list that
// empties comes back absent rather than empty: the emitter can write only one of those
// two, so a corpus holding the other would fail to round-trip.
func TestUnsettingAnEntryTakesItOutOfTheList(t *testing.T) {
	person := &model.Person{
		Slug:   "rossi-giovanni",
		Events: []model.Event{{Type: "birth"}, {Type: "marriage"}, {Type: "death"}},
	}
	unset(t, person, "events[1]")
	if len(person.Events) != 2 || person.Events[1].Type != "death" {
		t.Fatalf("events = %+v", person.Events)
	}

	unset(t, person, "events[0].type")
	if person.Events[0].Type != "" {
		t.Errorf("a field of an entry was taken out rather than emptied: %+v", person.Events[0])
	}

	unset(t, person, "events[1]")
	unset(t, person, "events[0]")
	if person.Events != nil {
		t.Errorf("the emptied list came back as %#v rather than absent", person.Events)
	}
}

func unset(t *testing.T, entity any, path string) {
	t.Helper()
	field, err := Resolve(entity, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := field.Unset(); err != nil {
		t.Fatal(err)
	}
}

func add(t *testing.T, entity any, path string, value any) {
	t.Helper()
	field, err := Resolve(entity, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := field.Append(value); err != nil {
		t.Fatal(err)
	}
}

// An entry is added to the end and taken out by value. Order is never rearranged,
// because a hypothesis's paths are positional and an entry that moved would take
// somebody's edit with it.
func TestAnEntryIsAddedToTheEndAndTakenOutByValue(t *testing.T) {
	family := &model.Family{Slug: "bianchi-maria--rossi-giovanni"}
	field, err := Resolve(family, "partners")
	if err != nil {
		t.Fatal(err)
	}
	for _, partner := range []string{"rossi-giovanni", "bianchi-maria"} {
		if err := field.Append(partner); err != nil {
			t.Fatal(err)
		}
	}
	if want := model.PartnersOf("rossi-giovanni", "bianchi-maria"); !reflect.DeepEqual(family.Partners, want) {
		t.Fatalf("partners = %v, want %v", family.Partners, want)
	}

	if err := field.Remove("bianchi-maria"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(family.Partners, model.PartnersOf("rossi-giovanni")) {
		t.Errorf("partners = %v", family.Partners)
	}
	if err := field.Remove("bianchi-maria"); err == nil {
		t.Error("taking out an entry that is not there reported success")
	}
}

// A citation is reached through its anchor, and what is written there lands in the
// source rather than in the copy the walk carried: a mapping's entries are not
// addressable in Go, so this is the one place a resolved field is written back through
// the container it came out of.
// What a picture is, who is in it and where, and the part of a picture that is somebody's
// portrait are each reached by a path, as a typed value or the YAML of one.
func TestAPictureIsReachedThroughTheList(t *testing.T) {
	family := &model.Family{Slug: "bianchi-maria--rossi-giovanni"}
	add(t, family, "media", map[string]any{"file": "photos/1858.jpg", "date": "21 APR 1858"})
	set(t, family, "media[0].title", Typed("The wedding party: outside San Lorenzo"))
	add(t, family, "media[0].depicts", Typed("rossi-giovanni"))
	add(t, family, "media[0].depicts", Typed("{person: bianchi-maria, region: [0.41, 0.12, 0.58, 0.47]}"))
	set(t, family, "media[0].depicts[0].region", Typed("[0.59, 0.1, 0.76, 0.49]"))
	want := []model.Media{{
		File: "photos/1858.jpg", Title: "The wedding party: outside San Lorenzo", Date: mustDate("21 APR 1858"),
		Depicts: []model.Depicted{
			{Person: "rossi-giovanni", Region: &model.Region{Left: 0.59, Top: 0.1, Right: 0.76, Bottom: 0.49}},
			{Person: "bianchi-maria", Region: &model.Region{Left: 0.41, Top: 0.12, Right: 0.58, Bottom: 0.47}},
		},
	}}
	if !reflect.DeepEqual(family.Media, want) {
		t.Errorf("media = %+v", family.Media)
	}
	if field, err := Resolve(family, "media[0].file"); err != nil || field.Set(Typed("")) != nil {
		t.Fatal(err)
	}
	if _, err := Emit(family); err == nil || !strings.Contains(err.Error(), unnamedMedia) {
		t.Errorf("a picture of nothing was written: %v", err)
	}
	set(t, family, "media[0].file", Typed("../../people/rossi-giovanni/photos/1858.jpg"))
	if _, err := Emit(family); err == nil || !strings.Contains(err.Error(), referringMedia) {
		t.Errorf("a reference saying what the picture is was written: %v", err)
	}

	person := &model.Person{Slug: "bianchi-maria"}
	set(t, person, "portrait", Typed("photos/group.jpg"))
	set(t, person, "portrait.region", Typed("[0.1, 0.2, 0.3, 0.4]"))
	if want := (model.Region{Left: 0.1, Top: 0.2, Right: 0.3, Bottom: 0.4}); person.Portrait.File != "photos/group.jpg" ||
		*person.Portrait.Region != want {
		t.Errorf("portrait = %+v", person.Portrait)
	}
	set(t, person, "portrait.file", Typed(""))
	if _, err := Emit(person); err == nil {
		t.Error("a portrait of the part of no picture was written")
	}
}

func TestACitationIsReachedThroughItsAnchor(t *testing.T) {
	source := &model.Source{Slug: "firenze-civil-deaths-1901"}
	set(t, source, "citations.1103-17.text", "Morì Giovanni Rossi, di anni 67.")
	set(t, source, "citations.1103-17.images", []any{"scans/1103.jpg"})
	set(t, source, "citations.1103-17.images[0].region", []any{"0.1", "0.2", "0.9", "0.5"})
	got := source.Citations["1103-17"]
	if got.Text == "" || len(got.Images) != 1 || got.Images[0].File != "scans/1103.jpg" ||
		*got.Images[0].Region != (model.Region{Left: 0.1, Top: 0.2, Right: 0.9, Bottom: 0.5}) {
		t.Fatalf("the citation came out as %+v", got)
	}

	set(t, source, "citations.1104-02.text", "Another entry on the same page.")
	if len(source.Citations) != 2 {
		t.Fatalf("citations = %+v", source.Citations)
	}

	unset(t, source, "citations.1103-17")
	if _, still := source.Citations["1103-17"]; still {
		t.Error("unsetting an anchor left it behind")
	}
	unset(t, source, "citations.1104-02")
	if source.Citations != nil {
		t.Errorf("the emptied mapping came back as %#v rather than absent", source.Citations)
	}
}

// The error names the fields there are, because a reader who has mistyped one is two
// characters from the right answer and should not have to go and look it up.
func TestAnUnknownFieldNamesTheOnesThereAre(t *testing.T) {
	_, err := Resolve(&model.Person{Slug: "rossi-giovanni"}, "name.surnam")
	if err == nil {
		t.Fatal("name.surnam resolved")
	}
	if !strings.Contains(err.Error(), "surname") {
		t.Errorf("the error does not mention surname: %v", err)
	}
}

// The body is the argument for why one Giovanni is that Giovanni, and it goes out to
// YAML and back in like every other value. Anything the round trip reflows, re-indents
// or trims is prose somebody wrote coming back as prose they did not.
func TestSettingTheBodyChangesNothingAboutIt(t *testing.T) {
	for _, body := range []string{
		"# Why this Giovanni\n\nThe parish register has him in 1834.\n\n- one\n- two\n",
		"trailing spaces   \nand a tab\there",
		"---\nwhat looks like a fence\n---",
		"a line ending in a colon:\n  and one indented under it",
		"",
	} {
		person := &model.Person{Slug: "rossi-giovanni"}
		set(t, person, "body", body)
		if person.Body != body {
			t.Errorf("a body of %q came back as %q", body, person.Body)
		}
	}
}

// Every field a path reaches can be set, which takes a decoder for the type it holds.
// The decoders are a table beside addressable's, and a field added to one with a type
// the other does not know is a field an edit reaches and cannot write.
func TestEveryFieldAPathReachesCanBeRead(t *testing.T) {
	var reach func(reflect.Type, string)
	reach = func(target reflect.Type, path string) {
		if fields := addressable(reflect.New(target).Interface()); fields != nil {
			for name, field := range fields {
				reach(reflect.TypeOf(field).Elem(), path+"."+name)
			}
			return
		}
		switch target.Kind() {
		case reflect.Slice, reflect.Map, reflect.Pointer:
			reach(target.Elem(), path)
			return
		}
		if _, known := decoders[target]; !known && target != reflect.TypeFor[string]() {
			t.Errorf("%s holds a %s, which no edit can be read as", path, target)
		}
	}
	for _, kind := range model.Kinds {
		entity := model.New(kind)
		for name, field := range addressable(entity) {
			reach(reflect.TypeOf(field).Elem(), fmt.Sprintf("%T.%s", entity, name))
		}
	}
}

// A child's relation is one value, a word or a mapping, and setting it replaces the
// other form.
func TestARelationIsSetOnABareChild(t *testing.T) {
	family := &model.Family{Children: model.ChildrenOf("rossi-elena")}
	set(t, family, "children[0].relation", Typed("{rossi-pietro: adoptive, galli-ines: step}"))
	want := model.Relationship{Partners: []model.PartnerRelation{
		{Partner: "rossi-pietro", Relation: model.Adoptive}, {Partner: "galli-ines", Relation: model.Step}}}
	if !reflect.DeepEqual(family.Children[0].Relation, want) {
		t.Errorf("the child is %+v", family.Children[0])
	}
	set(t, family, "children[0].relation", Typed("adoptive"))
	if want := (model.Relationship{Word: model.Adoptive}); !reflect.DeepEqual(family.Children[0].Relation, want) {
		t.Errorf("the child is %+v", family.Children[0])
	}
}

// A sex is a letter until something is cited for it, and the letter changes on its own
// without taking the citation with it. A citation of a sex nobody stated is evidence of
// nothing, and the file has no way to write it.
func TestASexIsCitedAndKeepsItsCitation(t *testing.T) {
	person := &model.Person{Slug: "rossi-giovanni"}
	set(t, person, "sex", Typed("M"))
	add(t, person, "sex.sources", Typed("parish-san-lorenzo-baptisms#1834-042 OPD"))
	set(t, person, "sex.value", Typed("X"))
	if person.Sex.Value != model.Neither || len(person.Sex.Sources) != 1 {
		t.Fatalf("the sex is %+v", person.Sex)
	}
	file, err := Emit(person)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(file), "sex:\n  value: X\n  sources: [parish-san-lorenzo-baptisms#1834-042 OPD]\n") {
		t.Errorf("the cited sex was written\n%s", file)
	}

	set(t, person, "sex.value", Typed("U"))
	if _, err := Emit(person); err == nil {
		t.Error("a citation of a sex nobody stated was written")
	}
	unset(t, person, "sex")
	if file, err := Emit(person); err != nil || strings.Contains(string(file), "sex") {
		t.Errorf("a sex nobody stated was written: %v\n%s", err, file)
	}
}

// An identifier is added under the name of its service, which becomes part of a path,
// so a name a path could not reach is refused where the file is written.
func TestAnIdentifierIsReachedThroughItsService(t *testing.T) {
	place := &model.Place{Slug: "firenze", Name: model.PlaceName{Text: "Firenze"}}
	set(t, place, "ids.geonames", Typed("3176959"))
	if !reflect.DeepEqual(place.IDs, model.IDs{"geonames": model.IdentifiersOf("3176959")}) {
		t.Fatalf("ids = %+v", place.IDs)
	}
	add(t, place, "ids.geonames", Typed("{id: '6542283', note: The comune rather than the city.}"))
	set(t, place, "ids.geonames[1].sources", Typed("[villani-cronica#i-38]"))
	file, err := Emit(place)
	if err != nil || !strings.Contains(string(file), "ids:\n  geonames:\n    - 3176959\n    - id: 6542283\n") {
		t.Errorf("two identifiers at one service were not written as a list: %v\n%s", err, file)
	}
	unset(t, place, "ids.geonames[0]")
	unset(t, place, "ids.geonames[0]")
	if place.IDs != nil {
		t.Errorf("the last identifier taken out left %+v", place.IDs)
	}
	set(t, place, "ids.Find a Grave", Typed("12345"))
	if _, err := Emit(place); err == nil {
		t.Error("a service no path could reach was written")
	}
}

// A name's words stand beside its parts, the record's spelling kept where the parts do
// not spell it, and its surnames are one word or a list reached entry by entry. A name
// that is only a type or a date is refused where the file is written, being no name.
func TestANameIsWordsBesideParts(t *testing.T) {
	person := &model.Person{Slug: "rossi-giovanni"}
	set(t, person, "name", Typed("Gio. Rossi"))
	set(t, person, "name.given", "Giovanni")
	set(t, person, "name.surname", "Rossi")
	if got, err := Emit(person); err != nil || !strings.Contains(string(got), "name:\n  text: Gio. Rossi\n  given: Giovanni\n  surname: Rossi\n") {
		t.Errorf("the words beside the parts were written as %v\n%s", err, got)
	}

	add(t, person, "name.surname", Typed("{name: Cerreto, prefix: da, origin: feudal}"))
	set(t, person, "name.surname[0].origin", "patrilineal")
	want := model.Surnames{{Name: "Rossi", Origin: "patrilineal"}, {Name: "Cerreto", Prefix: "da", Origin: "feudal"}}
	if !reflect.DeepEqual(person.Name.Surname, want) {
		t.Errorf("the surnames read %+v, want %+v", person.Name.Surname, want)
	}
	set(t, person, "name.surname", "Rossi")
	if want := model.SurnameOf("Rossi"); !reflect.DeepEqual(person.Name.Surname, want) {
		t.Errorf("a surname set as a word reads %+v", person.Name.Surname)
	}

	set(t, person, "name.text", "")
	set(t, person, "name.given", "")
	unset(t, person, "name.surname")
	set(t, person, "name.type", "married")
	if _, err := Emit(person); err == nil || !strings.Contains(err.Error(), "at least one of") {
		t.Errorf("a name with no words and no parts was emitted: %v", err)
	}

	for _, value := range []string{
		"{type: married}", "{}", "''", "{given: Maria, name: Maria Rossi}",
		"{surname: [{prefix: da}]}", "{surname: {name: Rossi, primary: false}}", "{given: Maria, lang: '!!'}",
	} {
		field, err := Resolve(person, "name")
		if err != nil {
			t.Fatal(err)
		}
		if err := field.Set(Typed(value)); err == nil {
			t.Errorf("the name %s was read", value)
		}
	}
}

// Somebody an event names is reached by their place in the event, and every part of
// them by its name, so that a godmother's age can be set where the register gives it
// and her record put in place of her name once she has one.
func TestSomebodyAnEventNamesIsReachedInPlace(t *testing.T) {
	person := &model.Person{Slug: "rossi-giovanni"}
	set(t, person, "events", Typed("[{type: baptism, with: [{name: Maria Neri, role: godmother}]}]"))
	set(t, person, "events[0].with[0].age", "29")
	set(t, person, "events[0].value", Typed("in the Pieve"))
	add(t, person, "events[0].with", Typed("{person: verdi-giacomo, role: godfather}"))

	want := []model.Named{
		{Name: model.PersonName{Text: "Maria Neri"}, Role: "godmother", Age: "29"},
		{Person: "verdi-giacomo", Role: "godfather"},
	}
	if got := person.Events[0].With; !reflect.DeepEqual(got, want) {
		t.Errorf("with = %+v, want %+v", got, want)
	}

	set(t, person, "events[0].with[0].person", "neri-maria")
	if _, err := Emit(person); err == nil || !strings.Contains(err.Error(), "no name beside it") {
		t.Errorf("somebody named by both a record and a name was emitted: %v", err)
	}
	field, err := Resolve(person, "events[0].with[0].name")
	if err != nil {
		t.Fatal(err)
	}
	if err := field.Unset(); err != nil {
		t.Fatal(err)
	}
	if _, err := Emit(person); err != nil {
		t.Errorf("the godmother given a record did not emit: %v", err)
	}
}
