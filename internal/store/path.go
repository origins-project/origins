package store

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/model"
)

// A field path addresses one value inside an entity, spelled the way the corpus spells
// it: surname, events[0].date, citations.1103-17.text. It is the language check reports
// a finding in and a hypothesis writes an edit in, so there is one way of saying where
// in a record something is, whoever is saying it.
//
// Paths are positional, and never re-resolved against a later corpus. An event inserted
// in between would otherwise move events[1] from a death to a marriage without saying
// anything, which is why a hypothesis pins the commit its paths were written against.

// Typed is a value as somebody wrote it, before anything has read it.
//
// The characters alone do not say what they are. "Stato civile: nascite" is the title
// of a register and YAML would make a mapping of it; "43.7696, 11.2558" is a pair of
// coordinates and nothing but a string until something knows to look. Only the field a
// value is going into can tell those apart, so the reading waits until the path has
// been resolved: a field that holds text takes the characters as they were written,
// and everywhere else the text is YAML and is read the way a file is read.
type Typed string

// Field is the value a path reached: where it sits, and what it takes to change it
// there rather than in a copy of it.
type Field struct {
	value reflect.Value

	// write pushes the value back through a container that handed out a copy instead
	// of a location — a mapping, whose entries are not addressable in Go. Everything
	// reached through such an entry inherits it, so a change to a citation's text lands
	// in the source and not in the copy the walk was carrying.
	write func() error

	// drop takes the value out of the mapping it came from, where it came from one.
	drop func() error

	// list and index are the list the value is an entry in, where it is an entry in
	// one. They are what lets unset take the entry out rather than blank it.
	list  reflect.Value
	index int
}

// Resolve finds the value a field path addresses within an entity, which is a pointer
// to one.
func Resolve(entity any, path string) (*Field, error) {
	steps, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	root := reflect.ValueOf(entity)
	if root.Kind() != reflect.Pointer || root.IsNil() {
		return nil, fmt.Errorf("%T is not an entity to address a field of", entity)
	}
	at := &Field{value: root.Elem()}
	for _, one := range steps {
		if at, err = at.step(one); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	return at, nil
}

func (f *Field) step(s step) (*Field, error) {
	at, err := f.named(s.name)
	if err != nil {
		return nil, err
	}
	for _, index := range s.indices {
		switch {
		case at.value.Kind() != reflect.Slice:
			return nil, fmt.Errorf("%s is not a list", s.name)
		case index >= at.value.Len():
			return nil, fmt.Errorf("%s has %d entries, so there is no [%d]", s.name, at.value.Len(), index)
		}
		at = &Field{value: at.value.Index(index), write: at.write, list: at.value, index: index}
	}
	return at, nil
}

func (f *Field) named(name string) (*Field, error) {
	if f.value.Kind() == reflect.Map {
		return f.entry(name)
	}
	var fields map[string]any
	if f.value.CanAddr() {
		fields = addressable(f.value.Addr().Interface())
	}
	if fields == nil {
		return nil, fmt.Errorf("%q names a field of something that has none", name)
	}
	target, ok := fields[name]
	if !ok {
		return nil, fmt.Errorf("there is no %q here, only %s",
			name, strings.Join(slices.Sorted(maps.Keys(fields)), ", "))
	}
	return &Field{value: reflect.ValueOf(target).Elem(), write: f.write}, nil
}

// entry addresses one member of a mapping: a source's citations, a record's
// identifiers by service, or the manifest's tag styles.
//
// A key nobody has written yet is not an error. A mapping is a set of named things and
// writing to a new name is how one joins it, which is the only way a transcription gets
// into a source at all. The cost is that a mistyped anchor makes a citation rather than
// failing, and the diff is where that shows, as it is for every other way of writing
// something wrong into a corpus.
func (f *Field) entry(key string) (*Field, error) {
	mapping, at := f.value, reflect.ValueOf(key)
	holder := reflect.New(mapping.Type().Elem()).Elem()
	if existing := mapping.MapIndex(at); existing.IsValid() {
		holder.Set(existing)
	}
	parent := f.write
	inherited := func() error {
		if parent == nil {
			return nil
		}
		return parent()
	}
	return &Field{
		value: holder,
		write: func() error {
			if holder.Kind() == reflect.Slice && holder.Len() == 0 || holder.Interface() == (model.TagStyle{}) {
				return drop(mapping, at, inherited)
			}
			if mapping.IsNil() {
				mapping.Set(reflect.MakeMap(mapping.Type()))
			}
			mapping.SetMapIndex(at, holder)
			return inherited()
		},
		drop: func() error { return drop(mapping, at, inherited) },
	}, nil
}

// drop takes a key out of a mapping. A list left with nothing in it goes the same way,
// a service with no identifiers being no service, and so does a tag's style that gives
// neither a color nor a priority, a tag needing no style to be used. A mapping left
// with nothing in it is nil, empty and absent being the same corpus.
func drop(mapping, key reflect.Value, inherited func() error) error {
	mapping.SetMapIndex(key, reflect.Value{})
	if mapping.Len() == 0 {
		mapping.Set(reflect.Zero(mapping.Type()))
	}
	return inherited()
}

// Get is what the field holds, as the model holds it.
func (f *Field) Get() any { return f.value.Interface() }

// Set puts value where the path pointed, read as the corpus reads a value of that
// field: a bare scalar, or the qualified mapping form where the field has one.
func (f *Field) Set(value any) error {
	read, err := decodeValue(f.value.Type(), value)
	if err != nil {
		return err
	}
	f.value.Set(read)
	return f.flush()
}

// Unset empties the field.
//
// An entry of a list or of a mapping goes altogether rather than being blanked. A list
// with a hole in the middle of it is not what unsetting one of its entries says, and a
// citation with nothing in it is not the same thing as no citation.
func (f *Field) Unset() error {
	switch {
	case f.drop != nil:
		return f.drop()
	case f.list.IsValid():
		f.list.Set(without(f.list, f.index))
	default:
		f.value.Set(reflect.Zero(f.value.Type()))
	}
	return f.flush()
}

// Append puts one more entry on the end of the list the path named. Order is never
// rearranged, here or anywhere else, because paths are positional and an entry that
// moved would take somebody's edit with it.
func (f *Field) Append(value any) error {
	if f.value.Kind() != reflect.Slice {
		return errors.New("only a list can be added to")
	}
	entry, err := decodeValue(f.value.Type().Elem(), value)
	if err != nil {
		return err
	}
	f.value.Set(reflect.Append(f.value, entry))
	return f.flush()
}

// Remove takes the entry equal to value out of the list the path named.
//
// An entry that is not there is an error rather than a silent success: the edit says
// the corpus holds something it does not, and whoever wrote the edit is working from a
// picture of the corpus that is wrong.
func (f *Field) Remove(value any) error {
	if f.value.Kind() != reflect.Slice {
		return errors.New("only a list can be removed from")
	}
	entry, err := decodeValue(f.value.Type().Elem(), value)
	if err != nil {
		return err
	}
	for i := range f.value.Len() {
		if reflect.DeepEqual(f.value.Index(i).Interface(), entry.Interface()) {
			f.value.Set(without(f.value, i))
			return f.flush()
		}
	}
	return fmt.Errorf("%v is not in the list to take out of it", value)
}

func (f *Field) flush() error {
	if f.write == nil {
		return nil
	}
	return f.write()
}

// without returns the list with one entry taken out. A list left with nothing in it
// comes back nil, because empty and absent are the same corpus and the emitter can
// write only one of them.
func without(list reflect.Value, index int) reflect.Value {
	if list.Len() == 1 {
		return reflect.Zero(list.Type())
	}
	kept := reflect.MakeSlice(list.Type(), 0, list.Len()-1)
	for i := range list.Len() {
		if i != index {
			kept = reflect.Append(kept, list.Index(i))
		}
	}
	return kept
}

// Head is the field a path starts at: events, for events[0].date. It is what an edit
// is about in the large, where the rest of the path says which part of it.
func Head(path string) (string, error) {
	steps, err := parsePath(path)
	if err != nil {
		return "", err
	}
	return steps[0].name, nil
}

// step is one component of a path: a name, and the indices applied to it.
type step struct {
	name    string
	indices []int
}

func parsePath(path string) ([]step, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("a field path says where in a record something is, and this one says nothing")
	}
	var steps []step
	for _, part := range strings.Split(path, ".") {
		name, rest, _ := strings.Cut(part, "[")
		if name == "" {
			return nil, fmt.Errorf("%s: %q names no field", path, part)
		}
		one := step{name: name}
		for rest != "" {
			digits, more, closed := strings.Cut(rest, "]")
			index, err := strconv.Atoi(digits)
			switch {
			case !closed:
				return nil, fmt.Errorf("%s: %q is missing a ]", path, part)
			case err != nil || index < 0:
				return nil, fmt.Errorf("%s: %q is not an index", path, digits)
			case more != "" && !strings.HasPrefix(more, "["):
				return nil, fmt.Errorf("%s: %q carries %q after an index", path, part, more)
			}
			one.indices = append(one.indices, index)
			rest = strings.TrimPrefix(more, "[")
		}
		steps = append(steps, one)
	}
	return steps, nil
}

// addressable names every field a path can reach, spelled as the emitter writes it.
// Adding a field to the model means adding it here, and path_test.go fails until it is.
//
// Three sorts of field are deliberately absent. An entity's slug is its identity and
// moves only by rename, which has a file to move and references to rewrite rather than
// a value to change. Its files are the directory it sits in rather than anything it
// says. And a hypothesis's edits are the language a change is written in, not something
// another change reaches inside and rewrites.
//
// The body is here, though it is prose below the fence rather than a field above it.
// It carries the argument for why one Giovanni is that Giovanni, which is the substance
// of the corpus and the last thing that should be out of reach.
func addressable(entity any) map[string]any {
	switch v := entity.(type) {
	case *model.Person:
		return map[string]any{
			"private": &v.Private, "living": &v.Living, "name": &v.Name, "also-known-as": &v.AlsoKnownAs,
			"sex": &v.Sex, "portrait": &v.Portrait,
			"events": &v.Events, "not": &v.Not, "media": &v.Media, "links": &v.Links, "sources": &v.Sources,
			"tags": &v.Tags, "same-as": &v.SameAs, "ids": &v.IDs, "body": &v.Body,
		}
	case *model.Family:
		return map[string]any{
			"private": &v.Private, "partners": &v.Partners, "type": &v.Type, "children": &v.Children,
			"events": &v.Events, "not": &v.Not, "media": &v.Media, "links": &v.Links, "sources": &v.Sources,
			"tags": &v.Tags, "same-as": &v.SameAs, "ids": &v.IDs, "body": &v.Body,
		}
	case *model.Place:
		return map[string]any{
			"private": &v.Private, "name": &v.Name, "also-known-as": &v.AlsoKnownAs, "type": &v.Type,
			"within": &v.Within, "coordinates": &v.Coordinates, "addresses": &v.Addresses,
			"from": &v.From, "until": &v.Until, "media": &v.Media, "links": &v.Links, "sources": &v.Sources,
			"tags": &v.Tags, "same-as": &v.SameAs, "ids": &v.IDs, "body": &v.Body,
		}
	case *model.Source:
		return map[string]any{
			"private": &v.Private, "title": &v.Title, "short-title": &v.ShortTitle, "type": &v.Type,
			"author": &v.Author, "publication": &v.Publication, "covers": &v.Covers,
			"repositories": &v.Repositories, "attributes": &v.Attributes, "text": &v.Text,
			"citations": &v.Citations, "media": &v.Media, "links": &v.Links, "tags": &v.Tags,
			"same-as": &v.SameAs, "ids": &v.IDs, "body": &v.Body,
		}
	case *model.Coverage:
		return map[string]any{"events": &v.Events, "date": &v.Date, "place": &v.Place, "note": &v.Note}
	case *model.Holding:
		return map[string]any{
			"repository": &v.Repository, "call-number": &v.CallNumber, "medium": &v.Medium, "note": &v.Note,
		}
	case *model.Repository:
		return map[string]any{
			"private": &v.Private, "name": &v.Name, "type": &v.Type, "addresses": &v.Addresses,
			"media": &v.Media, "links": &v.Links, "tags": &v.Tags, "same-as": &v.SameAs, "ids": &v.IDs,
			"body": &v.Body,
		}
	case *model.Note:
		return map[string]any{
			"private": &v.Private, "title": &v.Title, "events": &v.Events, "media": &v.Media,
			"links": &v.Links, "sources": &v.Sources, "tags": &v.Tags, "same-as": &v.SameAs, "ids": &v.IDs,
			"body": &v.Body,
		}
	case *model.SameAs:
		return map[string]any{"id": &v.ID, "name": &v.Name, "note": &v.Note, "sources": &v.Sources}
	case *model.Hypothesis:
		return map[string]any{
			"status": &v.Status, "base": &v.Base, "confidence": &v.Confidence,
			"proposed-by": &v.ProposedBy, "supports": &v.Supports,
			"contradicts": &v.Contradicts, "body": &v.Body,
		}
	case *model.Event:
		return map[string]any{
			"private": &v.Private, "type": &v.Type, "value": &v.Value, "date": &v.Date, "sort-date": &v.Sort,
			"place": &v.Place, "address": &v.Address, "age": &v.Age, "cause": &v.Cause, "with": &v.With,
			"family": &v.Family, "attributes": &v.Attributes, "media": &v.Media, "links": &v.Links, "note": &v.Note,
			"sources": &v.Sources, "tags": &v.Tags, "ids": &v.IDs,
		}
	case *model.StatedAge:
		return map[string]any{"value": &v.Value, "note": &v.Note, "sources": &v.Sources}
	case *model.StatedCause:
		return map[string]any{"value": &v.Value, "note": &v.Note, "sources": &v.Sources}
	case *model.Attribute:
		return map[string]any{
			"type": &v.Type, "value": &v.Value, "date": &v.Date, "note": &v.Note, "sources": &v.Sources,
		}
	case *model.NonEvent:
		return map[string]any{"type": &v.Type, "date": &v.Date, "note": &v.Note, "sources": &v.Sources}
	case *model.Address:
		return map[string]any{
			"lines": &v.Lines, "street": &v.Street, "locality": &v.Locality, "city": &v.City,
			"parish": &v.Parish, "county": &v.County, "state": &v.State, "postal": &v.Postal,
			"country": &v.Country, "date": &v.Date, "links": &v.Links, "note": &v.Note, "sources": &v.Sources,
		}
	case *model.Link:
		return map[string]any{"url": &v.URL, "type": &v.Type, "title": &v.Title, "note": &v.Note}
	case *model.Identifier:
		return map[string]any{"id": &v.ID, "note": &v.Note, "sources": &v.Sources}
	case *model.Named:
		return map[string]any{
			"person": &v.Person, "name": &v.Name, "role": &v.Role,
			"age": &v.Age, "attributes": &v.Attributes, "note": &v.Note, "sources": &v.Sources,
		}
	case *model.PersonName:
		return map[string]any{
			"private": &v.Private, "text": &v.Text, "prefix": &v.Prefix, "given": &v.Given, "call": &v.Call,
			"nickname": &v.Nickname, "family-nickname": &v.FamilyNickname, "surname-prefix": &v.SurnamePrefix,
			"surname": &v.Surname, "suffix": &v.Suffix, "group": &v.Group, "lang": &v.Lang, "type": &v.Type,
			"display": &v.Display, "sort": &v.Sort, "from": &v.From, "until": &v.Until, "note": &v.Note,
			"sources": &v.Sources,
		}
	case *model.Surname:
		return map[string]any{
			"name": &v.Name, "prefix": &v.Prefix, "connector": &v.Connector, "origin": &v.Origin,
			"primary": &v.Primary,
		}
	case *model.PlaceName:
		return map[string]any{
			"name": &v.Text, "lang": &v.Lang, "type": &v.Type, "from": &v.From, "until": &v.Until,
			"note": &v.Note, "sources": &v.Sources,
		}
	case *model.StatedSex:
		return map[string]any{"value": &v.Value, "sources": &v.Sources}
	case *model.Partner:
		return map[string]any{"person": &v.Person, "name": &v.Name, "note": &v.Note}
	case *model.Child:
		return map[string]any{
			"id": &v.ID, "name": &v.Name, "relation": &v.Relation, "status": &v.Status, "note": &v.Note,
			"sources": &v.Sources,
		}
	case *model.PlaceRef:
		return map[string]any{
			"place": &v.Place, "from": &v.From, "until": &v.Until, "note": &v.Note, "sources": &v.Sources,
		}
	case *model.Citation:
		return map[string]any{
			"event": &v.Event, "page": &v.Page, "date": &v.Date, "text": &v.Text, "with": &v.With,
			"images": &v.Images, "links": &v.Links, "attributes": &v.Attributes, "note": &v.Note,
			"tags": &v.Tags, "ids": &v.IDs,
		}
	case *model.Image:
		return map[string]any{"file": &v.File, "region": &v.Region}
	case *model.Media:
		return map[string]any{
			"private": &v.Private, "file": &v.File, "type": &v.Type, "title": &v.Title, "date": &v.Date,
			"place": &v.Place, "region": &v.Region, "depicts": &v.Depicts, "attributes": &v.Attributes,
			"note": &v.Note, "sources": &v.Sources, "tags": &v.Tags, "ids": &v.IDs,
		}
	case *model.Depicted:
		return map[string]any{"person": &v.Person, "region": &v.Region, "attributes": &v.Attributes, "note": &v.Note}
	case *model.Project:
		return map[string]any{
			"name": &v.Name, "description": &v.Description, "main-person": &v.MainPerson, "lang": &v.Lang,
			"copyright": &v.Copyright, "researchers": &v.Researchers, "bookmarks": &v.Bookmarks, "tags": &v.Tags,
		}
	case *model.Researcher:
		return map[string]any{"name": &v.Name, "addresses": &v.Addresses, "links": &v.Links, "note": &v.Note}
	case *model.TagStyle:
		return map[string]any{"color": &v.Color, "priority": &v.Priority}
	}
	return nil
}

// decodeValue reads an edit's value as a value of the field it is going into, through
// the same wire types a file goes through. There is one grammar for what a corpus can
// say and this is it: a date, a slug, a parent in either the bare or the qualified form
// all read here exactly as they read on disk, down to refusing a field nobody knows.
func decodeValue(target reflect.Type, value any) (reflect.Value, error) {
	switch target.Kind() {
	case reflect.Slice:
		if _, whole := decoders[target]; whole {
			break
		}
		// a list somebody typed is YAML whatever it holds, so it is read as one and
		// its entries go on from there like any other
		if typed, unread := value.(Typed); unread {
			var read any
			if err := unmarshal([]byte(typed), &read); err != nil {
				return reflect.Value{}, err
			}
			value = read
		}
		entries, ok := value.([]any)
		if !ok {
			return reflect.Value{}, errors.New("a list is set from a list, and one more entry is added rather than set")
		}
		out := reflect.MakeSlice(target, 0, len(entries))
		for _, entry := range entries {
			one, err := decodeValue(target.Elem(), entry)
			if err != nil {
				return reflect.Value{}, err
			}
			out = reflect.Append(out, one)
		}
		if out.Len() == 0 {
			return reflect.Zero(target), nil
		}
		return out, nil
	case reflect.Pointer:
		// a pointer is the one field where absent and zero look different, and
		// nothing typed is absent rather than a coordinate in the Gulf of Guinea
		if value == nil || value == Typed("") {
			return reflect.Zero(target), nil
		}
		one, err := decodeValue(target.Elem(), value)
		if err != nil {
			return reflect.Value{}, err
		}
		holder := reflect.New(target.Elem())
		holder.Elem().Set(one)
		return holder, nil
	}

	typed, unread := value.(Typed)
	if unread && target == reflect.TypeFor[string]() {
		// the field holds text, so the text is the value — which is the only way a
		// title with a colon in it or a body with a fence in it survives being typed
		return reflect.ValueOf(string(typed)), nil
	}

	read, known := decoders[target]
	if !known {
		return reflect.Value{}, fmt.Errorf("origins cannot read a %s out of an edit", target)
	}
	// Text nobody has read yet is already YAML and goes straight to the wire type.
	// Anything else goes out to YAML and back in, so that the value an edit carries
	// and the value a file carries are read by one piece of code rather than by two
	// that have to agree.
	written := []byte(typed)
	if !unread {
		out, err := yaml.Marshal(value)
		if err != nil {
			return reflect.Value{}, err
		}
		written = out
	}
	one, err := read(written)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(one), nil
}

// decoders reads each type a field can hold, every one of them through the wire type
// the loader uses. An edit cannot put a value into a corpus that the corpus could not
// have been written with in the first place.
var decoders = map[reflect.Type]func([]byte) (any, error){
	reflect.TypeFor[string]():             decodeAs(func(s string) string { return s }),
	reflect.TypeFor[bool]():               decodeAs(func(f flagged) bool { return bool(f) }),
	reflect.TypeFor[model.Slug]():         decodeAs(func(s slug) model.Slug { return model.Slug(s) }),
	reflect.TypeFor[model.Date]():         decodeAs(func(d date) model.Date { return model.Date(d) }),
	reflect.TypeFor[model.Age]():          decodeAs(func(a age) model.Age { return model.Age(a) }),
	reflect.TypeFor[model.Sex]():          decodeAs(func(s sex) model.Sex { return model.Sex(s) }),
	reflect.TypeFor[model.Living]():       decodeAs(func(l living) model.Living { return model.Living(l) }),
	reflect.TypeFor[model.StatedSex]():    decodeAs(func(s statedSex) model.StatedSex { return model.StatedSex(s) }),
	reflect.TypeFor[model.StatedAge]():    decodeAs(func(a statedAge) model.StatedAge { return model.StatedAge(a) }),
	reflect.TypeFor[model.StatedCause]():  decodeAs(func(c statedCause) model.StatedCause { return model.StatedCause(c) }),
	reflect.TypeFor[model.Attribute]():    decodeAs(func(a attribute) model.Attribute { return model.Attribute(a) }),
	reflect.TypeFor[model.NonEvent]():     decodeAs(func(n nonEvent) model.NonEvent { return model.NonEvent(n) }),
	reflect.TypeFor[model.Relationship](): decodeAs(func(r relationship) model.Relationship { return model.Relationship(r) }),
	reflect.TypeFor[model.LinkStatus]():   decodeAs(func(s linkStatus) model.LinkStatus { return model.LinkStatus(s) }),
	reflect.TypeFor[model.Language]():     decodeAs(func(l language) model.Language { return model.Language(l) }),
	reflect.TypeFor[model.IDs]():          decodeAs(func(i ids) model.IDs { return model.IDs(i) }),
	reflect.TypeFor[model.Status]():       decodeAs(func(s status) model.Status { return model.Status(s) }),
	reflect.TypeFor[model.Confidence]():   decodeAs(func(c confidence) model.Confidence { return model.Confidence(c) }),
	reflect.TypeFor[model.CitationRef]():  decodeAs(func(c citationRef) model.CitationRef { return model.CitationRef(c) }),
	reflect.TypeFor[model.Coordinates]():  decodeAs(func(c coordinates) model.Coordinates { return model.Coordinates(c) }),
	reflect.TypeFor[model.PersonName]():   decodeAs(func(n personName) model.PersonName { return model.PersonName(n) }),
	reflect.TypeFor[model.Surnames](): decodeAs(func(s surnames) model.Surnames {
		return model.Surnames(convert(s, func(s surname) model.Surname { return model.Surname(s) }))
	}),
	reflect.TypeFor[model.Surname]():     decodeAs(func(s surname) model.Surname { return model.Surname(s) }),
	reflect.TypeFor[model.PlaceName]():   decodeAs(func(n placeName) model.PlaceName { return model.PlaceName(n) }),
	reflect.TypeFor[model.Partner]():     decodeAs(func(p partner) model.Partner { return model.Partner(p) }),
	reflect.TypeFor[model.Child]():       decodeAs(func(c child) model.Child { return model.Child(c) }),
	reflect.TypeFor[model.PlaceRef]():    decodeAs(func(p placeRef) model.PlaceRef { return model.PlaceRef(p) }),
	reflect.TypeFor[model.Event]():       decodeAs(func(e eventDoc) model.Event { return events([]eventDoc{e})[0] }),
	reflect.TypeFor[model.Named]():       decodeAs(func(n named) model.Named { return model.Named(n) }),
	reflect.TypeFor[model.Citation]():    decodeAs(citationDoc.citation),
	reflect.TypeFor[model.Coverage]():    decodeAs(func(c coverage) model.Coverage { return model.Coverage(c) }),
	reflect.TypeFor[model.Holding]():     decodeAs(func(h holding) model.Holding { return model.Holding(h) }),
	reflect.TypeFor[model.Image]():       decodeAs(func(i image) model.Image { return model.Image(i) }),
	reflect.TypeFor[model.Region]():      decodeAs(func(r region) model.Region { return model.Region(r) }),
	reflect.TypeFor[model.Media]():       decodeAs(func(m media) model.Media { return model.Media(m) }),
	reflect.TypeFor[model.Depicted]():    decodeAs(func(d depicted) model.Depicted { return model.Depicted(d) }),
	reflect.TypeFor[model.Link]():        decodeAs(func(l link) model.Link { return model.Link(l) }),
	reflect.TypeFor[model.Address]():     decodeAs(func(a address) model.Address { return model.Address(a) }),
	reflect.TypeFor[model.Identifier]():  decodeAs(func(i identifier) model.Identifier { return model.Identifier(i) }),
	reflect.TypeFor[model.Identifiers](): decodeAs(func(l identifiers) model.Identifiers { return model.Identifiers(l) }),
	reflect.TypeFor[model.SameAs]():      decodeAs(func(s sameAs) model.SameAs { return model.SameAs(s) }),
	reflect.TypeFor[model.Researcher]():  decodeAs(func(r researcher) model.Researcher { return model.Researcher(r) }),
	reflect.TypeFor[model.EntityRef]():   decodeAs(func(b bookmark) model.EntityRef { return model.EntityRef(b) }),
	reflect.TypeFor[model.TagStyle]():    decodeAs(func(t tagStyle) model.TagStyle { return model.TagStyle(t) }),
	reflect.TypeFor[map[string]model.TagStyle](): func(written []byte) (any, error) {
		var wire map[string]tagStyle
		if err := unmarshal(written, &wire); err != nil {
			return nil, err
		}
		return tagStyles(wire)
	},
}

func decodeAs[Wire, Out any](one func(Wire) Out) func([]byte) (any, error) {
	return func(written []byte) (any, error) {
		var wire Wire
		if err := unmarshal(written, &wire); err != nil {
			return nil, err
		}
		return one(wire), nil
	}
}
