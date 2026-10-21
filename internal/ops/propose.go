package ops

import (
	"bytes"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Nobody writes a field path by hand. A hypothesis is authored by editing the corpus as
// though it were true, and origins works out which edits that was: the corpus is the
// language a researcher already writes in, and the vocabulary is what lets origins
// carry the result out, test it and take it back.

// Unstated is a difference the vocabulary cannot state in order. Paths are positional,
// and an edit that reproduces the files while meaning something else — every event
// after an insertion changed in place, one more added at the end — is worse than none:
// it reads as a claim about each of those events that nobody made.
type Unstated struct {
	Entity model.EntityRef
	Path   string // the list, or empty for the record as a whole
	Reason string
}

func (u *Unstated) Error() string {
	if u.Path == "" {
		return fmt.Sprintf("%s: %s", u.Entity, u.Reason)
	}
	return fmt.Sprintf("%s %s: %s", u.Entity, u.Path, u.Reason)
}

// Propose states what after differs from before in, as edits that carry before to
// after when they are carried out in order.
//
// A record gone and a record arrived that are the same bytes under two names is a
// rename, which is how adopting tells one too, and it is stated first so that
// everything pointing at the record follows it as a rename makes it. Then records gone
// are deleted, records arrived are created with every field they have, and what is left
// is each record's fields: set and unset for a value, add and remove for an entry of a
// list, the entries of a list being lined up in the fewest edits and not by position.
// The manifest's fields come last, as edits to project.
//
// What the edits come to is checked before they are handed back, by carrying them out
// on a copy of before and emitting it. A difference no path reaches, such as a
// hypothesis's own edits, fails there and not later.
func Propose(before, after *model.Corpus) ([]model.Edit, error) {
	working := before.Clone()
	var edits []model.Edit
	perform := func(edit model.Edit) error {
		if _, err := applied(working, edit); err != nil {
			return err
		}
		edits = append(edits, edit)
		return nil
	}

	for _, pair := range Renamed(before, after) {
		if err := perform(model.Rename{Entity: pair.From, To: pair.To.Slug}); err != nil {
			return nil, err
		}
	}
	for _, slug := range working.AllSlugs() {
		kind := working.KindOf(slug)
		if still := after.KindOf(slug); still != kind {
			if err := perform(model.Delete{Entity: model.EntityRef{Kind: kind, Slug: slug}}); err != nil {
				return nil, err
			}
		}
	}
	for _, slug := range after.AllSlugs() {
		now, kind := after.Entity(slug), after.KindOf(slug)
		if had := working.KindOf(slug); had == kind {
			continue
		}
		fields, err := store.Fields(now)
		if err != nil {
			return nil, err
		}
		if err := perform(model.Create{Entity: model.EntityRef{Kind: kind, Slug: slug}, Value: fields}); err != nil {
			return nil, err
		}
	}
	for _, slug := range after.AllSlugs() {
		now, kind := after.Entity(slug), after.KindOf(slug)
		was := working.Entity(slug)
		stated, err := differences(model.EntityRef{Kind: kind, Slug: slug}, was, now)
		if err != nil {
			return nil, err
		}
		for _, edit := range stated {
			if err := perform(edit); err != nil {
				return nil, err
			}
		}
	}
	stated, err := differences(model.TheProject, &working.Project, &after.Project)
	if err != nil {
		return nil, err
	}
	for _, edit := range stated {
		if err := perform(edit); err != nil {
			return nil, err
		}
	}

	if err := same(working, after); err != nil {
		return nil, err
	}
	return edits, nil
}

// same holds the edits to the files: every record the one corpus has, the other has as
// the same kind and emits as the same bytes, and the manifests are the same bytes too.
func same(got, want *model.Corpus) error {
	a, err := store.EmitProject(got.Project)
	if err != nil {
		return err
	}
	b, err := store.EmitProject(want.Project)
	if err != nil {
		return err
	}
	if !bytes.Equal(a, b) {
		return &Unstated{Entity: model.TheProject, Reason: "the change to it is not one the edit vocabulary can state"}
	}
	for _, slug := range merged(got.AllSlugs(), want.AllSlugs()) {
		have, kind := got.Entity(slug), got.KindOf(slug)
		wanted, wantedKind := want.Entity(slug), want.KindOf(slug)
		ref := model.EntityRef{Kind: wantedKind, Slug: slug}
		if kind != wantedKind {
			return &Unstated{Entity: ref, Reason: "the edits do not bring the record to where the files have it"}
		}
		a, err := store.Emit(have)
		if err != nil {
			return err
		}
		b, err := store.Emit(wanted)
		if err != nil {
			return err
		}
		if !bytes.Equal(a, b) {
			return &Unstated{Entity: ref, Reason: "the change to it is not one the edit vocabulary can state"}
		}
	}
	return nil
}

func merged(a, b []model.Slug) []model.Slug {
	out := slices.Concat(a, b)
	slices.Sort(out)
	return slices.Compact(out)
}

// differences states the change from one version of a record to another as field
// edits, in the order they are to be carried out.
func differences(ref model.EntityRef, was, now any) ([]model.Edit, error) {
	a, err := store.Fields(was)
	if err != nil {
		return nil, err
	}
	b, err := store.Fields(now)
	if err != nil {
		return nil, err
	}
	d := &differ{entity: ref}
	if err := d.mapping("", a, b); err != nil {
		return nil, err
	}
	return d.edits, nil
}

type differ struct {
	entity model.EntityRef
	edits  []model.Edit
}

func (d *differ) set(path string, value any) {
	d.edits = append(d.edits, model.Set{Entity: d.entity, Path: path, Value: value})
}

// value states the change at one path. Two mappings or two lists are gone into, so
// that what the edit names is the date that changed and not the event it is in; a value
// that changed shape — a parent given a relation, which turns a slug into a mapping —
// is set whole.
func (d *differ) value(path string, was, now any) error {
	if reflect.DeepEqual(was, now) {
		return nil
	}
	switch old := was.(type) {
	case map[string]any:
		if fresh, ok := now.(map[string]any); ok {
			return d.mapping(path, old, fresh)
		}
	case []any:
		if fresh, ok := now.([]any); ok {
			return d.list(path, old, fresh)
		}
	}
	d.set(path, now)
	return nil
}

// mapping goes through the names in order. A list the record did not have before is
// entries added to nothing, which is how the vocabulary says an entry arrived. A
// mapping keyed by words somebody wrote, as a tag's style is, may have a key no path
// can spell, and is then set whole.
func (d *differ) mapping(path string, was, now map[string]any) error {
	keys := slices.Sorted(maps.Keys(merge(was, now)))
	if path != "" && slices.ContainsFunc(keys, func(key string) bool { return strings.ContainsAny(key, ".[") }) {
		d.set(path, now)
		return nil
	}
	for _, key := range keys {
		at := key
		if path != "" {
			at = path + "." + key
		}
		old, had := was[key]
		fresh, has := now[key]
		switch {
		case !has:
			d.edits = append(d.edits, model.Unset{Entity: d.entity, Path: at})
		case !had:
			if entries, listed := fresh.([]any); listed {
				if err := d.list(at, nil, entries); err != nil {
					return err
				}
				continue
			}
			d.set(at, fresh)
		default:
			if err := d.value(at, old, fresh); err != nil {
				return err
			}
		}
	}
	return nil
}

func merge(a, b map[string]any) map[string]any {
	out := maps.Clone(a)
	maps.Copy(out, b)
	return out
}

// list lines the two lists up in the fewest edits and states that. An entry is kept
// where it stands, changed there, taken out, or added — and added only at the end,
// because a path is a position: an alignment that puts an entry in anywhere else, or
// takes one out only to add it again, would reproduce the list while saying something
// about its entries that did not happen, and is refused.
//
// Entries are taken out first and by what they hold, then changed where they have come
// to stand, then added, which is the order the paths are written for.
func (d *differ) list(path string, was, now []any) error {
	type change struct {
		at       int
		was, now any
	}
	var (
		gone    []int
		changed []change
		added   []any
	)
	for _, step := range align(was, now) {
		switch step.kind {
		case changing:
			changed = append(changed, change{at: step.i, was: was[step.i], now: now[step.j]})
		case removing:
			gone = append(gone, step.i)
		case adding:
			if step.i < len(was) {
				return &Unstated{Entity: d.entity, Path: path,
					Reason: "an entry is put in before the end, and every path after it would come to mean the one beside it"}
			}
			added = append(added, now[step.j])
		}
	}
	for _, at := range gone {
		if slices.ContainsFunc(added, func(entry any) bool { return reflect.DeepEqual(entry, was[at]) }) {
			return &Unstated{Entity: d.entity, Path: path, Reason: "its entries are put in another order"}
		}
	}

	for _, at := range gone {
		d.edits = append(d.edits, model.Remove{Entity: d.entity, Path: path, Value: was[at]})
	}
	for _, one := range changed {
		shift := 0
		for _, at := range gone {
			if at < one.at {
				shift++
			}
		}
		if err := d.value(fmt.Sprintf("%s[%d]", path, one.at-shift), one.was, one.now); err != nil {
			return err
		}
	}
	for _, entry := range added {
		d.edits = append(d.edits, model.Add{Entity: d.entity, Path: path, Value: entry})
	}
	return nil
}

type stepKind int

const (
	changing stepKind = iota // kept where it stands, changed or not
	removing
	adding
)

// step is one move in lining two lists up: entry i of the one and entry j of the other
// are where it stands.
type step struct {
	kind stepKind
	i, j int
}

// align lines two lists up in the fewest edits. Where two ways cost the same, the one
// keeping more entries paired wins, so that a date corrected in an event is stated as
// that and not as the event taken out and another added; and after that an entry is
// changed rather than taken out, and taken out rather than added.
func align(a, b []any) []step {
	type score struct{ cost, paired int }
	better := func(x, y score) bool {
		return x.cost < y.cost || x.cost == y.cost && x.paired > y.paired
	}
	best := make([][]score, len(a)+1)
	move := make([][]stepKind, len(a)+1)
	for i := range best {
		best[i] = make([]score, len(b)+1)
		move[i] = make([]stepKind, len(b)+1)
	}
	for i := len(a); i >= 0; i-- {
		for j := len(b); j >= 0; j-- {
			if i == len(a) && j == len(b) {
				continue
			}
			var found bool
			consider := func(kind stepKind, s score) {
				if !found || better(s, best[i][j]) {
					best[i][j], move[i][j], found = s, kind, true
				}
			}
			if i < len(a) && j < len(b) {
				next := best[i+1][j+1]
				consider(changing, score{next.cost + distance(a[i], b[j]), next.paired + 1})
			}
			if i < len(a) {
				next := best[i+1][j]
				consider(removing, score{next.cost + 1, next.paired})
			}
			if j < len(b) {
				next := best[i][j+1]
				consider(adding, score{next.cost + 1, next.paired})
			}
		}
	}
	var out []step
	for i, j := 0, 0; i < len(a) || j < len(b); {
		kind := move[i][j]
		out = append(out, step{kind: kind, i: i, j: j})
		switch kind {
		case changing:
			i, j = i+1, j+1
		case removing:
			i++
		case adding:
			j++
		}
	}
	return out
}

// distance is how many edits it takes to state one entry of a list as another where it
// stands: none for the same entry, one for each field of a record set, unset, added to
// or taken out of, and two for an entry that is something else altogether, which is
// what taking it out and adding the other costs.
func distance(was, now any) int {
	if reflect.DeepEqual(was, now) {
		return 0
	}
	a, isMapping := was.(map[string]any)
	b, alsoMapping := now.(map[string]any)
	if !isMapping || !alsoMapping {
		return 2
	}
	return fieldEdits(a, b)
}

// fieldEdits counts the edits between two mappings, going into what they hold as the
// differ does.
func fieldEdits(a, b map[string]any) int {
	n := 0
	for key := range merge(a, b) {
		x, had := a[key]
		y, has := b[key]
		switch {
		case !had || !has:
			n++
		case reflect.DeepEqual(x, y):
		default:
			n += valueEdits(x, y)
		}
	}
	return n
}

func valueEdits(x, y any) int {
	switch old := x.(type) {
	case map[string]any:
		if fresh, ok := y.(map[string]any); ok {
			return fieldEdits(old, fresh)
		}
	case []any:
		if fresh, ok := y.([]any); ok {
			n := 0
			for _, one := range align(old, fresh) {
				if one.kind == changing {
					n += distance(old[one.i], fresh[one.j])
				} else {
					n++
				}
			}
			return n
		}
	}
	return 1
}
