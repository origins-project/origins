package ops

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// folded is one record taken into another, and is the single thing about a change that
// the corpus afterwards cannot be asked.
//
// Everything a change does to the files is worked out by emitting the corpus and
// comparing it to what is on disk. That answers for every edit but this one: a record
// folded into another leaves the model looking exactly as it would have if the record
// had simply been deleted, and nothing in it says where the photographs that sat
// beside it are now to go. The two names go out of here to whatever is writing.
type folded struct{ From, Into model.Slug }

// applied carries out one edit against a corpus held in memory, and names the records
// it took into other records.
//
// It is the single place a corpus is changed, whether the change came from a command
// somebody typed or from a hypothesis somebody accepted. The vocabulary is closed and
// both go through it the same way, which is what lets a hypothesis be tried against a
// copy before it is believed — and it means a command is a hypothesis that nobody
// bothered to write down.
//
// Nothing here asks whether the result makes sense. Deleting a person two children
// declare as a parent leaves the corpus contradicting itself, and the transaction is
// where that is caught, once, for every way there is of introducing it.
//
// What it does ask is whether the edit can be carried out at all. A merge of records
// that disagree and a split into a name already taken are refused here, because there
// is no corpus on the far side of either to hand the transaction.
//
// It is reached only through a transaction, which is the one thing that knows where
// the photographs of a record folded into another are to go: a caller holding the
// corpus alone would carry out the fold and lose them without an error.
func applied(corpus *model.Corpus, edit model.Edit) ([]folded, error) {
	if subjects := edit.Subjects(); subjects[0].Kind == model.KindTag {
		if rename, ok := edit.(model.Rename); ok {
			return nil, retag(corpus, rename)
		}
		return nil, fmt.Errorf("%s: a tag is a word the records carry and is only renamed; its style is the manifest's, at project tags", subjects[0])
	}
	if edit.Field() == "" && slices.Contains(edit.Subjects(), model.TheProject) {
		return nil, fmt.Errorf("%s: the manifest is one file that is always there, so it cannot be %s",
			model.TheProject, wholly[edit.Op()])
	}
	switch e := edit.(type) {
	case model.Set:
		return nil, at(corpus, e.Entity, e.Path, func(f *store.Field) error { return f.Set(e.Value) })
	case model.Unset:
		return nil, at(corpus, e.Entity, e.Path, (*store.Field).Unset)
	case model.Add:
		return nil, at(corpus, e.Entity, e.Path, func(f *store.Field) error { return f.Append(e.Value) })
	case model.Remove:
		return nil, at(corpus, e.Entity, e.Path, func(f *store.Field) error { return f.Remove(e.Value) })
	case model.Create:
		return nil, create(corpus, e)
	case model.Delete:
		if err := erase(corpus, e.Entity); err != nil {
			return nil, err
		}
		unbookmark(corpus)
		return nil, nil
	case model.Rename:
		return nil, rename(corpus, e)
	case model.Merge:
		return fold(corpus, e)
	case model.Split:
		return nil, divide(corpus, e)
	}
	return nil, fmt.Errorf("op %q is registered but not carried out", edit.Op())
}

// wholly is what each operation on a whole record would do to the manifest.
var wholly = map[string]string{
	"create": "created", "delete": "deleted", "rename": "renamed", "merge": "merged", "split": "split",
}

// at resolves a field path inside an entity and does something to what it finds. A
// family whose partners it changes is renamed after the partners it has now, as it is
// when one of them is renamed.
func at(corpus *model.Corpus, ref model.EntityRef, path string, do func(*store.Field) error) error {
	found, err := entity(corpus, ref)
	if err != nil {
		return err
	}
	if head, _ := store.Head(path); ref == model.TheProject && head == "format-version" {
		return fmt.Errorf("%s: format-version says which form the files are written in, and origins fmt changes it by rewriting them", ref)
	}
	field, err := store.Resolve(found, path)
	if err != nil {
		return fmt.Errorf("%s: %w", ref, err)
	}
	family, _ := found.(*model.Family)
	var partners []model.Partner
	if family != nil {
		partners = slices.Clone(family.Partners)
	}
	if err := do(field); err != nil {
		return fmt.Errorf("%s %s: %w", ref, path, err)
	}
	if family != nil {
		regroup(corpus, family, partners)
	}
	return nil
}

// create brings a new entity into the corpus carrying the fields the edit gives it.
//
// They are set through the same resolution every other edit goes through, so there is
// nothing a create can write that a set could not. A slug already in use is refused:
// one slug names one thing, and it names it across all seven kinds at once. So is either
// word a partner without a record is written as, which names nothing. The refusal
// says which names are free instead, since a second Giovanni Rossi is an ordinary
// thing to have found, and the names are the ones model.Discriminate makes out of what
// the create itself says, a bare number coming last.
func create(corpus *model.Corpus, e model.Create) error {
	if corpus.Taken(e.Entity.Slug) {
		refusal := fmt.Sprintf("%s: the corpus already has a %s of that name", e.Entity, corpus.KindOf(e.Entity.Slug))
		if e.Entity.Slug.Reserved() {
			refusal = fmt.Sprintf("%s: %v", e.Entity, reserved(e.Entity.Slug))
		}
		free, err := model.Discriminate(e.Entity.Slug, hints(corpus, e), corpus.Taken)
		if err != nil || len(free) == 0 {
			return errors.New(refusal)
		}
		names := make([]string, len(free))
		for i, slug := range free {
			names[i] = string(slug)
		}
		return fmt.Errorf("%s; names still free: %s", refusal, strings.Join(names, ", "))
	}
	fresh := model.New(e.Entity.Kind)
	if fresh == nil {
		return fmt.Errorf("%s: %q is not an entity kind", e.Entity, e.Entity.Kind)
	}
	if err := fill(fresh, e); err != nil {
		return err
	}
	corpus.Put(e.Entity.Slug, fresh)
	return nil
}

// hints are what a create says that would tell the new entity apart from the one
// already holding its name, best first, as import offers them: when and where somebody
// was born and when they died, what a family says that model.FamilyHints reads, what a
// place lies within. The fields are filled into an entity nobody will keep, and one that
// will not fill leaves the fields before it, a refusal being on its way anyway.
func hints(corpus *model.Corpus, e model.Create) []string {
	var hints []string
	year := func(d model.Date) {
		if start := d.Start(); start.Year != nil {
			hints = append(hints, strconv.Itoa(*start.Year))
		}
	}
	event := func(events []model.Event, kind string, place bool) {
		for _, event := range events {
			if kind == "" || model.SameWord(event.Type, kind) {
				year(event.Date)
				if place && event.Place != "" {
					hints = append(hints, string(event.Place))
				}
				return
			}
		}
	}
	fresh := model.New(e.Entity.Kind)
	fill(fresh, e)
	switch v := fresh.(type) {
	case *model.Person:
		event(v.Events, "birth", true)
		event(v.Events, "death", false)
	case *model.Family:
		hints = corpus.FamilyHints(v)
	case *model.Place:
		for _, within := range v.Within {
			hints = append(hints, string(within.Place))
		}
	}
	return hints
}

// fill sets the fields a create gives on a fresh entity. The fields are taken in name
// order so that an entity with two bad ones reports the same one every time.
func fill(fresh any, e model.Create) error {
	for _, field := range slices.Sorted(maps.Keys(e.Value)) {
		found, err := store.Resolve(fresh, field)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Entity, err)
		}
		if err := found.Set(e.Value[field]); err != nil {
			return fmt.Errorf("%s %s: %w", e.Entity, field, err)
		}
	}
	return nil
}

// erase takes an entity out of the corpus. What is left pointing at it is not this
// function's business: a dangling reference is an error the validator names precisely,
// and the transaction refuses the change on the strength of it.
func erase(corpus *model.Corpus, ref model.EntityRef) error {
	if _, err := entity(corpus, ref); err != nil {
		return err
	}
	corpus.Delete(ref.Slug)
	return nil
}

// unbookmark takes out every bookmark to a record that is not there. A bookmark keeps a
// record to hand and says nothing about it, so a record deleted takes its bookmark with
// it rather than being refused for leaving one behind.
func unbookmark(corpus *model.Corpus) {
	bookmarks := slices.DeleteFunc(corpus.Project.Bookmarks, func(b model.EntityRef) bool {
		return corpus.KindOf(b.Slug) != b.Kind
	})
	if len(bookmarks) == 0 {
		bookmarks = nil
	}
	corpus.Project.Bookmarks = bookmarks
}

// entity is what a reference points at.
//
// The kind is checked rather than taken on trust. Slugs are unique across the seven
// kinds, so the slug alone would find it — and an edit naming person/firenze would
// then quietly rewrite a place. The kind is the writer saying what they believe they
// are editing, and a corpus where that belief is wrong is worth stopping over.
//
// The manifest is addressed as project, and its format-version is the one field of it
// no edit reaches: the number says which canonical form the files are written in, and
// changing it is a migration of every file, which fmt makes.
func entity(corpus *model.Corpus, ref model.EntityRef) (any, error) {
	if ref == model.TheProject {
		return &corpus.Project, nil
	}
	found, kind := corpus.Entity(ref.Slug), corpus.KindOf(ref.Slug)
	switch {
	case kind == "":
		return nil, fmt.Errorf("%s: there is nothing of that name in the corpus", ref)
	case kind != ref.Kind:
		return nil, fmt.Errorf("%s: %s is a %s", ref, ref.Slug, kind)
	}
	return found, nil
}
