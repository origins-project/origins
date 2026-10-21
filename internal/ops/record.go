package ops

import (
	"fmt"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// performed carries out an edit and says what it amounts to in the history, naming the
// records the edit took into other records as well.
//
// Applying and describing are one call because the description is not all on one side
// of the change. An event added is in the corpus only afterwards and an event taken out
// only before, so a caller that described the change at the wrong moment would record
// the wrong event, or none, and nothing anywhere would say so.
//
// What comes back is the commit bar the parts nobody can derive from the edit: the tool
// that wrote it, the citations the fact rests on, and a subject line better than the
// one a machine can write. Those are for the caller, who knows why.
//
// The records folded away are for the transaction, which has files to carry after
// them and is the only caller.
func performed(corpus *model.Corpus, edit model.Edit) (history.Commit, []folded, error) {
	entry := history.Commit{
		Subject:  subject(edit),
		Op:       operation(edit),
		Entities: edit.Subjects(),
	}
	switch e := edit.(type) {
	case model.Rename:
		if e.Entity.Kind == model.KindTag {
			entry.Entities = append(entry.Entities, retagging(corpus, e)...)
		} else {
			entry.Entities = append(entry.Entities, cascade(corpus, e)...)
		}
	case model.Merge:
		entry.Entities = append(entry.Entities, coupled(corpus, e)...)
	}

	var taking bool
	switch edit.(type) {
	case model.Unset, model.Remove:
		taking = true
	}
	if taking {
		entry.Events = eventsIn(corpus, edit)
	}
	// a family whose partners the edit changes is renamed with them, and the commit is
	// the only place that will say so
	on := edit.Subjects()[0]
	var family *model.Family
	if on.Kind == model.KindFamily && edit.Field() != "" {
		family = corpus.Families[on.Slug]
	}
	taken, err := applied(corpus, edit)
	if err != nil {
		return history.Commit{}, nil, err
	}
	if !taking {
		entry.Events = eventsIn(corpus, edit)
	}
	if family != nil && family.Slug != on.Slug {
		renamed := model.EntityRef{Kind: model.KindFamily, Slug: family.Slug}
		entry.Entities = append(entry.Entities, renamed)
		entry.Renamed = append(entry.Renamed, history.Renaming{From: on, To: renamed})
	}
	return entry, taken, nil
}

// operation is the <noun>.<verb> a commit is filed under: the field the edit lands in
// and what it did there, or the kind of entity where the edit is of a whole one, and
// tags where a tag is renamed, that being the field it is rewritten in.
//
// The noun is spelled the way the corpus spells it, so the trailer and the path in the
// subject line say the same word, and the verb is the vocabulary's own. There is no
// second set of names for the same nine operations to drift out of step with.
func operation(edit model.Edit) string {
	noun := string(edit.Subjects()[0].Kind)
	if edit.Subjects()[0].Kind == model.KindTag {
		noun = "tags"
	}
	if head, err := store.Head(edit.Field()); err == nil {
		noun = head
	}
	return noun + "." + edit.Op()
}

// subject is the line the log shows. It is written out of the edit rather than left to
// the caller because a commit has to have one, and whoever made the change can always
// say it better — every command that changes a corpus takes a message of its own.
func subject(edit model.Edit) string {
	where, on := edit.Field(), edit.Subjects()[0]
	switch e := edit.(type) {
	case model.Set:
		return fmt.Sprintf("Set %s on %s", where, on)
	case model.Unset:
		return fmt.Sprintf("Unset %s on %s", where, on)
	case model.Add:
		return fmt.Sprintf("Add to %s on %s", where, on)
	case model.Remove:
		return fmt.Sprintf("Remove from %s on %s", where, on)
	case model.Create:
		return fmt.Sprintf("Create %s", on)
	case model.Delete:
		return fmt.Sprintf("Delete %s", on)
	case model.Rename:
		return fmt.Sprintf("Rename %s to %s", on, e.To)
	case model.Merge:
		return fmt.Sprintf("Merge %s into %s", e.From, e.Into)
	case model.Split:
		return fmt.Sprintf("Split %s into %s", on, listed(e.Into))
	}
	op := edit.Op()
	return fmt.Sprintf("%s %s", strings.ToUpper(op[:1])+op[1:], on)
}

// listed writes out a run of entity references for a subject line to carry.
func listed(refs []model.EntityRef) string {
	names := make([]string, 0, len(refs))
	for _, one := range refs {
		names = append(names, one.String())
	}
	return strings.Join(names, ", ")
}

// eventsIn names the kind of event an edit reaches, which is what Origins-Event
// records: the trailer that lets a reader ask history for every death origins has ever
// been told about.
//
// A path says which event it is about — events[0].date is about an event as much as it
// is about a date — and an add to events is about the entry on the end, since that is
// where an add puts one.
//
// Nothing here fails. A path that reaches no event has no event to name, and a path
// that reaches nothing at all is for applied to refuse.
func eventsIn(corpus *model.Corpus, edit model.Edit) []string {
	where := edit.Field()
	if head, err := store.Head(where); err != nil || head != "events" {
		return nil
	}
	found, err := entity(corpus, edit.Subjects()[0])
	if err != nil {
		return nil
	}

	at, _, _ := strings.Cut(where, ".")
	if at == "events" {
		if _, appending := edit.(model.Add); !appending {
			return nil
		}
		list, err := store.Resolve(found, "events")
		if err != nil {
			return nil
		}
		events, ok := list.Get().([]model.Event)
		if !ok || len(events) == 0 {
			return nil
		}
		at = fmt.Sprintf("events[%d]", len(events)-1)
	}

	kind, err := store.Resolve(found, at+".type")
	if err != nil {
		return nil
	}
	// a type is kept as written, and a trailer is one line
	if text, named := kind.Get().(string); named && strings.TrimSpace(text) != "" {
		return []string{strings.Join(strings.Fields(text), " ")}
	}
	return nil
}

// Renamed works out which records a change nobody described gave a new name: a record
// gone and a record of the same kind arrived that emit the same bytes.
//
// That is all there is to tell a rename by, and it is enough, because a record's name
// is its file name and nothing inside it. A record renamed and edited in one go is a
// deletion beside a creation, there being nothing left to join the two, and so are two
// records that could each be the other, since picking one would be a guess recorded as
// a fact.
func Renamed(before, after *model.Corpus) []history.Renaming {
	type key struct {
		kind  model.Kind
		bytes string
	}
	sides := func(from, to *model.Corpus) map[key][]model.EntityRef {
		by := map[key][]model.EntityRef{}
		for _, slug := range from.AllSlugs() {
			found, kind := from.Entity(slug), from.KindOf(slug)
			if still := to.KindOf(slug); still == kind {
				continue
			}
			emitted, err := store.Emit(found)
			if err != nil {
				continue
			}
			k := key{kind, string(emitted)}
			by[k] = append(by[k], model.EntityRef{Kind: kind, Slug: slug})
		}
		return by
	}
	gone, arrived := sides(before, after), sides(after, before)

	var pairs []history.Renaming
	for k, was := range gone {
		if now := arrived[k]; len(was) == 1 && len(now) == 1 {
			pairs = append(pairs, history.Renaming{From: was[0], To: now[0]})
		}
	}
	slices.SortFunc(pairs, func(a, b history.Renaming) int {
		return strings.Compare(a.From.String(), b.From.String())
	})
	return pairs
}

// handed is the names an edit gave up, in the pairs its commit names them in. A rename
// and a merge name nothing else in pairs; a split names the record, then the name it
// keeps going by, then those it makes; and any other edit names in Renamed the family
// it renamed by changing its partners. A tag renamed gives up no record's name.
func handed(edit model.Edit, entry history.Commit) []history.Renaming {
	names := entry.Entities
	switch e := edit.(type) {
	case model.Rename:
		if e.Entity.Kind == model.KindTag {
			return nil
		}
	case model.Merge:
	case model.Split:
		if len(names) < 2 || names[0] == names[1] {
			return nil
		}
		names = names[:2]
	default:
		return entry.Renamed
	}
	var pairs []history.Renaming
	for i := 0; i+1 < len(names); i += 2 {
		pairs = append(pairs, history.Renaming{From: names[i], To: names[i+1]})
	}
	return pairs
}
