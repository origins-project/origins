package ops

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// A slug is not a fixed address, so the name a hypothesis calls a record by is the name
// it had when the hypothesis was written. Somebody renaming the record since has changed
// nothing about what the hypothesis says of it, and the history says which record went
// where. What pinning does with that is bring the base forward: the records it holds
// are given the names they go by now, with the same rename every other change uses so
// that everything pointing at them moves too, and the edits are made to call them the
// same way before they are carried out. What the patch names is then where the corpus
// keeps it.
//
// Nothing else is brought forward. A record deleted since has nothing an edit could be
// carried onto, and one merged since is part of another record whose facts the edit
// was not written about, so either is drift and the hypothesis says so rather than
// landing somewhere nobody meant. The exception is an edit that is itself what
// happened since — a rename to the name the record has been given, a merge into the
// record it went into, a deletion of a record that has gone — which has nothing left
// to carry, and is how a hypothesis accepted long ago still reads against the corpus
// it went into.

// brought is the edits calling every record by the name it goes by now, with nil
// standing in for an edit the corpus has already had so that the rest keep their
// numbers, or the drift that stops them.
//
// A name an earlier edit gave or took is left as it is written from there on. The edits
// are carried out one after another, so a name one of them has given to a record means
// that record to every edit after it, whatever the name meant at the base.
func brought(h *model.Hypothesis, fates map[model.EntityRef]history.Fate) ([]model.Edit, *Drifted) {
	var (
		bound = map[model.EntityRef]bool{}
		drift = Drifted{Slug: h.Slug, Base: h.Base}
		out   = make([]model.Edit, len(h.Edits))
	)
	fate := func(name model.EntityRef) (history.Fate, bool) {
		if bound[name] {
			return history.Fate{}, false
		}
		f, known := fates[name]
		return f, known
	}
	call := func(name model.EntityRef) model.EntityRef {
		if f, known := fate(name); known && !f.Merged && !f.Gone() {
			return f.Now
		}
		return name
	}
	for i, edit := range h.Edits {
		called := calling(edit, call)
		switch e := called.(type) {
		case model.Rename:
			if e.Entity.Slug == e.To {
				called = nil
			}
		case model.Merge:
			if f, known := fate(edit.(model.Merge).From); known && f.Merged && f.Now == e.Into {
				called = nil
			}
		case model.Delete:
			if f, known := fate(e.Entity); known && f.Gone() {
				called = nil
			}
		}
		if called != nil {
			for _, name := range edit.Subjects() {
				switch f, known := fate(name); {
				case !known:
				case f.Merged:
					drift.Gone = also(drift.Gone, name)
					if drift.Into == nil {
						drift.Into = map[model.EntityRef]model.EntityRef{}
					}
					drift.Into[name] = f.Now
				case f.Gone():
					drift.Gone = also(drift.Gone, name)
				}
			}
		}
		out[i] = called
		for _, given := range gives(edit) {
			bound[given] = true
		}
	}
	if drift.Gone != nil {
		return nil, &drift
	}
	return out, nil
}

// calling is an edit with every record it names handed to a function that may call it
// something else.
func calling(edit model.Edit, call func(model.EntityRef) model.EntityRef) model.Edit {
	switch e := edit.(type) {
	case model.Set:
		e.Entity = call(e.Entity)
		return e
	case model.Unset:
		e.Entity = call(e.Entity)
		return e
	case model.Add:
		e.Entity = call(e.Entity)
		return e
	case model.Remove:
		e.Entity = call(e.Entity)
		return e
	case model.Create:
		e.Entity = call(e.Entity)
		return e
	case model.Delete:
		e.Entity = call(e.Entity)
		return e
	case model.Rename:
		e.Entity = call(e.Entity)
		return e
	case model.Merge:
		e.From, e.Into = call(e.From), call(e.Into)
		return e
	case model.Split:
		into := make([]model.EntityRef, len(e.Into))
		for i, one := range e.Into {
			into[i] = call(one)
		}
		e.Entity, e.Into = call(e.Entity), into
		return e
	}
	return edit
}

// gives is the names an edit gives or takes away.
func gives(edit model.Edit) []model.EntityRef {
	switch e := edit.(type) {
	case model.Create, model.Delete, model.Rename, model.Split:
		return edit.Subjects()
	case model.Merge:
		return []model.EntityRef{e.From}
	}
	return nil
}

// followed is the old name of each record renamed since the base and the name it goes by
// now, for the references an edit writes by value. A name somebody else has taken since
// is left out, because a reference to it can no longer be told apart from one to them.
func followed(base *model.Corpus, fates map[model.EntityRef]history.Fate) map[model.Slug]model.Slug {
	moves := map[model.Slug]model.Slug{}
	for name, fate := range fates {
		if fate.Merged || fate.Gone() || fate.Now == name {
			continue
		}
		if held := base.KindOf(name.Slug); held == "" {
			moves[name.Slug] = fate.Now.Slug
		}
	}
	return moves
}

// forward brings the base to the names its records go by now, and takes away the
// records that have gone since. Those go first, so that a name one of them held and
// somebody else has been given since is free to be given again.
//
// The renames are made in whatever order leaves each name free for the record taking
// it. Two records that swapped names since the base can never be renamed one after the
// other, so where nothing is free one of them is set aside under a name nobody has, and
// the rest follow. A person's families are renamed along with them, as they were when
// it happened, and are then only brought on from wherever that left them.
func forward(base *store.Corpus, fates map[model.EntityRef]history.Fate) (*store.Corpus, error) {
	removed := false
	for _, name := range slices.SortedFunc(maps.Keys(fates), byName) {
		fate := fates[name]
		if !fate.Merged && !fate.Gone() {
			continue
		}
		kept, ok := base.Path(name.Slug)
		if !ok {
			continue
		}
		if err := os.RemoveAll(filepath.Join(base.Root, filepath.FromSlash(store.Location(kept)))); err != nil {
			return nil, err
		}
		removed = true
	}
	if removed {
		var problems error
		if base, problems = store.Load(base.Root); problems != nil {
			return nil, problems
		}
	}

	current := map[model.EntityRef]model.Slug{}
	var pending []model.EntityRef
	for name, fate := range fates {
		if !fate.Merged && !fate.Gone() && fate.Now != name {
			current[name] = name.Slug
			pending = append(pending, name)
		}
	}
	if len(pending) == 0 {
		return base, nil
	}
	// a person first, since renaming one renames the families named after them
	slices.SortFunc(pending, func(a, b model.EntityRef) int {
		if a.Kind != b.Kind && (a.Kind == model.KindPerson || b.Kind == model.KindPerson) {
			if a.Kind == model.KindPerson {
				return -1
			}
			return 1
		}
		return byName(a, b)
	})

	// Nothing here is judged, the base being what it was, so the transaction is opened
	// without the validation pass that would be for.
	tx := &Tx{corpus: base, taken: map[model.Slug]model.Slug{}}
	rename := func(kind model.Kind, from, to model.Slug) error {
		e := model.Rename{Entity: model.EntityRef{Kind: kind, Slug: from}, To: to}
		carried := cascade(tx.Corpus(), e)
		if _, err := tx.Perform(e); err != nil {
			return err
		}
		for i := 0; i+1 < len(carried); i += 2 {
			for name, now := range current {
				if name.Kind == model.KindFamily && now == carried[i].Slug {
					current[name] = carried[i+1].Slug
				}
			}
		}
		for name, now := range current {
			if name.Kind == kind && now == from {
				current[name] = to
			}
		}
		return nil
	}
	for rounds := 0; len(pending) > 0; rounds++ {
		if rounds > 2*len(current) {
			return nil, fmt.Errorf("%s could not be given the name it goes by now, %s",
				pending[0], fates[pending[0]].Now)
		}
		var left []model.EntityRef
		for _, name := range pending {
			now, to := current[name], fates[name].Now.Slug
			if now == to {
				continue
			}
			if held := tx.Corpus().KindOf(to); held != "" {
				left = append(left, name)
				continue
			}
			if err := rename(name.Kind, now, to); err != nil {
				return nil, fmt.Errorf("%s, renamed %s since: %w", name, fates[name].Now, err)
			}
		}
		if len(left) == len(pending) {
			aside, err := model.Discriminate(current[left[0]], nil, tx.Corpus().Taken)
			if err != nil || len(aside) == 0 {
				return nil, fmt.Errorf("%s could not be set aside to be renamed %s", left[0], fates[left[0]].Now)
			}
			if err := rename(left[0].Kind, current[left[0]], aside[0]); err != nil {
				return nil, err
			}
		}
		pending = left
	}

	change, err := tx.staged()
	if err != nil {
		return nil, err
	}
	if err := change.Apply(); err != nil {
		return nil, err
	}
	forwarded, problems := store.Load(base.Root)
	if problems != nil {
		return nil, problems
	}
	return forwarded, nil
}

func byName(a, b model.EntityRef) int { return strings.Compare(a.String(), b.String()) }
