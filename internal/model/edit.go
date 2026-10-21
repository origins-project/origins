package model

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// EntityRef addresses one entity, written kind/slug, the manifest, written project, or
// a tag, written tag/<word>, the word as written standing where a slug would.
type EntityRef struct {
	Kind Kind
	Slug Slug
}

// TheProject addresses the manifest. It is one file that is always there and has no
// name of its own, so it is written as the bare word.
var TheProject = EntityRef{Kind: KindProject}

// ParseEntityRef reads a reference as written: kind/slug, project or tag/<word>.
func ParseEntityRef(s string) (EntityRef, error) {
	if s == string(KindProject) {
		return TheProject, nil
	}
	kind, slug, found := strings.Cut(s, "/")
	if !found {
		return EntityRef{}, fmt.Errorf("entity %q: expected kind/slug", s)
	}
	if Kind(kind) == KindTag {
		word, err := ParseTag(slug)
		if err != nil {
			return EntityRef{}, fmt.Errorf("entity %q: %w", s, err)
		}
		return EntityRef{Kind: KindTag, Slug: Slug(word)}, nil
	}
	if !slices.Contains(Kinds, Kind(kind)) {
		return EntityRef{}, fmt.Errorf("entity %q: %q is not an entity kind", s, kind)
	}
	parsed, err := ParseSlug(slug)
	if err != nil {
		return EntityRef{}, fmt.Errorf("entity %q: %w", s, err)
	}
	return EntityRef{Kind: Kind(kind), Slug: parsed}, nil
}

// ParseTag reads a tag's word as written, in the normal form all text is read in.
func ParseTag(s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", errors.New("a tag cannot be empty")
	}
	return norm.NFC.String(s), nil
}

// ParseName reads the name a rename gives an entity of the kind: a tag's word, or the
// slug anything else is called by.
func ParseName(kind Kind, s string) (Slug, error) {
	if kind == KindTag {
		word, err := ParseTag(s)
		return Slug(word), err
	}
	return ParseSlug(s)
}

func (e EntityRef) String() string {
	if e == TheProject {
		return string(KindProject)
	}
	return string(e.Kind) + "/" + string(e.Slug)
}

func (e EntityRef) MarshalText() ([]byte, error) { return []byte(e.String()), nil }

func (e *EntityRef) UnmarshalText(b []byte) error {
	parsed, err := ParseEntityRef(string(b))
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

// Edit is one operation in a hypothesis. The vocabulary is closed and tagged: a
// hypothesis says what it would change in terms origins can execute, rather than in
// prose that only a reader can act on.
//
// Field paths are positional and are never re-resolved against a later corpus. The
// hypothesis pins the commit they were written against, and applying them means
// reconstructing that state, applying them where the paths are valid by construction,
// and diffing the result.
type Edit interface {
	// Op is the operation's name as a hypothesis writes it.
	Op() string

	// Subjects are the entities the edit affects. A commit records them, and a reader
	// searching history for what was ever done to somebody searches by them. Most
	// edits name one; a rename names the entity that was and the one it became, a
	// merge names the record that goes and the one it goes into, and a split names the
	// record and everything it becomes.
	Subjects() []EntityRef

	// Field is where inside the entity the edit lands, and empty for the operations
	// that are about a whole entity rather than a value in one.
	Field() string
}

// Set replaces the value at a field path.
type Set struct {
	Entity EntityRef
	Path   string
	Value  any
}

// Unset removes the field at a path.
type Unset struct {
	Entity EntityRef
	Path   string
}

// Add appends a value to the list at a field path.
type Add struct {
	Entity EntityRef
	Path   string
	Value  any
}

// Remove takes a value out of the list at a field path.
type Remove struct {
	Entity EntityRef
	Path   string
	Value  any
}

// Create brings a new entity into being. Value holds its fields as written.
type Create struct {
	Entity EntityRef
	Value  map[string]any
}

// Delete removes an entity.
type Delete struct {
	Entity EntityRef
}

// Rename gives an entity a new slug, which moves its file and rewrites every reference
// pointing at it. A tag renamed is given a new word, written where the slug would be,
// and every list carrying it carries that instead.
type Rename struct {
	Entity EntityRef
	To     Slug
}

// Merge folds one entity into another: combine the events, rewrite every inbound
// reference, delete the source. Deciding that two records describe one person is the
// hardest question in genealogy, and this keeps the answer to one line.
type Merge struct {
	From EntityRef
	Into EntityRef
}

// Split separates one entity into several, for when one record turns out to describe
// more than one person.
type Split struct {
	Entity EntityRef
	Into   []EntityRef
}

func (Set) Op() string    { return "set" }
func (Unset) Op() string  { return "unset" }
func (Add) Op() string    { return "add" }
func (Remove) Op() string { return "remove" }
func (Create) Op() string { return "create" }
func (Delete) Op() string { return "delete" }
func (Rename) Op() string { return "rename" }
func (Merge) Op() string  { return "merge" }
func (Split) Op() string  { return "split" }

func (s Set) Subjects() []EntityRef    { return []EntityRef{s.Entity} }
func (u Unset) Subjects() []EntityRef  { return []EntityRef{u.Entity} }
func (a Add) Subjects() []EntityRef    { return []EntityRef{a.Entity} }
func (r Remove) Subjects() []EntityRef { return []EntityRef{r.Entity} }
func (c Create) Subjects() []EntityRef { return []EntityRef{c.Entity} }
func (d Delete) Subjects() []EntityRef { return []EntityRef{d.Entity} }
func (r Rename) Subjects() []EntityRef {
	return []EntityRef{r.Entity, {Kind: r.Entity.Kind, Slug: r.To}}
}
func (m Merge) Subjects() []EntityRef { return []EntityRef{m.From, m.Into} }
func (s Split) Subjects() []EntityRef { return append([]EntityRef{s.Entity}, s.Into...) }

func (s Set) Field() string    { return s.Path }
func (u Unset) Field() string  { return u.Path }
func (a Add) Field() string    { return a.Path }
func (r Remove) Field() string { return r.Path }
func (Create) Field() string   { return "" }
func (Delete) Field() string   { return "" }
func (Rename) Field() string   { return "" }
func (Merge) Field() string    { return "" }
func (Split) Field() string    { return "" }

// editKinds is the whole vocabulary. Go has no exhaustiveness check over a set of
// types, so this map is the single place the set is written down and a test holds it to
// the vocabulary in DESIGN.md.
var editKinds = map[string]func() Edit{
	"set":    func() Edit { return &Set{} },
	"unset":  func() Edit { return &Unset{} },
	"add":    func() Edit { return &Add{} },
	"remove": func() Edit { return &Remove{} },
	"create": func() Edit { return &Create{} },
	"delete": func() Edit { return &Delete{} },
	"rename": func() Edit { return &Rename{} },
	"merge":  func() Edit { return &Merge{} },
	"split":  func() Edit { return &Split{} },
}

// NewEdit returns an empty edit of the named operation, for a decoder to fill in.
func NewEdit(op string) (Edit, error) {
	build, ok := editKinds[op]
	if !ok {
		return nil, fmt.Errorf("edit op %q: not one of %s", op, strings.Join(EditOps(), ", "))
	}
	return build(), nil
}

// EditOps lists the vocabulary in alphabetical order.
func EditOps() []string {
	ops := make([]string, 0, len(editKinds))
	for op := range editKinds {
		ops = append(ops, op)
	}
	slices.Sort(ops)
	return ops
}
