package ops

import (
	"fmt"
	"maps"
	"slices"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// fold decides that two records describe one thing, and leaves the corpus holding one.
//
// A merge is a rename whose new name is already taken. Everything a rename does a merge
// does too — every reference to the record that goes is rewritten to the one that
// stays, and a person's families are renamed after them — and the one thing a rename
// never has to face is the record already sitting at the new name. Combining the two is
// that, and it is the only part of a merge where the records can be found disagreeing.
//
// What comes back is the record the merge took into another. It is named because what
// was attached to it has to follow it, and by the time the corpus is written there is
// nothing left in it to say where it went.
func fold(corpus *model.Corpus, e model.Merge) ([]folded, error) {
	switch {
	case e.From == e.Into:
		return nil, fmt.Errorf("%s: that is the record it is being merged into", e.From)
	case e.From.Kind != e.Into.Kind:
		return nil, fmt.Errorf("%s into %s: a %s and a %s are not two records of one thing",
			e.From, e.Into, e.From.Kind, e.Into.Kind)
	}
	from, err := entity(corpus, e.From)
	if err != nil {
		return nil, err
	}
	into, err := entity(corpus, e.Into)
	if err != nil {
		return nil, err
	}
	moves, err := couples(corpus, e)
	if err != nil {
		return nil, err
	}
	if err := store.Combine(into, from); err != nil {
		return nil, fmt.Errorf("%s into %s: %w", e.From, e.Into, err)
	}
	if err := erase(corpus, e.From); err != nil {
		return nil, err
	}
	references(corpus, func(slug *model.Slug) {
		if *slug == e.From.Slug {
			*slug = e.Into.Slug
		} else if to, moving := moves[*slug]; moving {
			*slug = to
		}
	})
	for from, to := range moves {
		rekey(corpus, from, to)
	}
	beside(into, e.Into)
	settled(into, e.Into.Slug)
	corpus.Project.Bookmarks = distinct(corpus.Project.Bookmarks)
	return []folded{{From: e.From.Slug, Into: e.Into.Slug}}, nil
}

// settled takes out the claim either record made that it was the other, which the merge
// has just decided, and which would otherwise have the record that stays saying it is
// the same as itself.
func settled(entity any, slug model.Slug) {
	same := model.SameAsOf(entity)
	if same == nil {
		return
	}
	if *same = slices.DeleteFunc(*same, func(claim model.SameAs) bool { return claim.ID == slug }); len(*same) == 0 {
		*same = nil
	}
}

// distinct keeps the first of every bookmark, the two records merged having each been
// kept to hand.
func distinct(bookmarks []model.EntityRef) []model.EntityRef {
	var out []model.EntityRef
	for _, b := range bookmarks {
		if !slices.Contains(out, b) {
			out = append(out, b)
		}
	}
	return out
}

// beside writes a path the record that went held into the directory of the one that
// stays as the path beside it, which is where it now leads from.
func beside(entity any, ref model.EntityRef) {
	model.Pictures(entity, func(_ string, m *model.Media) {
		if at := model.Locate(ref, m.File); m.Refers() && at.Record == ref {
			m.File = at.File
		}
	})
}

// couples is the families that merging two people renames, read before anything
// changes and while a family still answers to the name it is losing.
//
// A family is named by its partners, so merging one of them renames it, as a rename
// does. Where the name it takes is one another family already has, the two stay two
// and the one renamed is told apart: one couple's marriage recorded twice and a couple
// who married twice look the same from the names, and which it is is for the researcher
// to say, by merging the families.
//
// Two records that are partners in one family are two people. Whatever else that family
// records, it records them as separate enough to have married each other, and a merge
// that went ahead would leave somebody married to themselves. A child and a partner of
// one family are two people for the same reason, or the merge would make somebody their
// own parent.
func couples(corpus *model.Corpus, e model.Merge) (map[model.Slug]model.Slug, error) {
	moves := map[model.Slug]model.Slug{}
	if e.From.Kind != model.KindPerson {
		return moves, nil
	}
	for _, slug := range corpus.FamiliesOf(e.From.Slug) {
		partner := func(p model.Partner) bool { return p.Person == e.Into.Slug }
		if slices.ContainsFunc(corpus.Families[slug].Partners, partner) {
			return nil, fmt.Errorf("%s and %s are partners in family/%s, so they are not one person",
				e.From, e.Into, slug)
		}
	}
	for _, pair := range [][2]model.Slug{{e.From.Slug, e.Into.Slug}, {e.Into.Slug, e.From.Slug}} {
		for _, slug := range corpus.FamiliesOf(pair[0]) {
			child := func(c model.Child) bool { return c.ID == pair[1] }
			if slices.ContainsFunc(corpus.Families[slug].Children, child) {
				return nil, fmt.Errorf("%s and %s are a partner and a child in family/%s, so they are not one person",
					e.From, e.Into, slug)
			}
		}
	}
	named(corpus, e.From.Slug, e.Into.Slug, moves)
	return moves, nil
}

// coupled is the families a merge renames, named so that the commit says what became of
// them, it being the only place that ever will.
//
// It is read before the merge is applied, like the rename it is the other half of, and
// a merge that is about to be refused simply names nothing.
func coupled(corpus *model.Corpus, e model.Merge) []model.EntityRef {
	moves, err := couples(corpus, e)
	if err != nil {
		return nil
	}
	refs := make([]model.EntityRef, 0, 2*len(moves))
	for _, was := range slices.Sorted(maps.Keys(moves)) {
		refs = append(refs,
			model.EntityRef{Kind: model.KindFamily, Slug: was},
			model.EntityRef{Kind: model.KindFamily, Slug: moves[was]})
	}
	return refs
}

// divide separates one record into several, for when it turns out to describe more than
// one person.
//
// Origins brings the records into being and stops there. Which fact belongs to which of
// them is the judgement that made the split worth proposing, and there is nothing
// mechanical to decide it with: the first record named keeps everything the one being
// split had, references to it included, and the rest arrive holding nothing. Copying the
// record into each of them would look more helpful and would be a corpus asserting that
// two people were born on the same day in the same place to the same parents — which
// nobody stated, and which origins would then have to be argued back out of.
func divide(corpus *model.Corpus, e model.Split) error {
	if len(e.Into) < 2 {
		return fmt.Errorf("%s: a split needs at least two records to split into", e.Entity)
	}
	if _, err := entity(corpus, e.Entity); err != nil {
		return err
	}
	// every name is asked for before any of them is taken, so that a split stopping
	// at the third of four records leaves no first and second behind
	seen := map[model.EntityRef]bool{}
	for _, one := range e.Into {
		standing := corpus.KindOf(one.Slug)
		switch {
		case one.Kind != e.Entity.Kind:
			return fmt.Errorf("%s: a %s does not split into a %s", e.Entity, e.Entity.Kind, one.Kind)
		case seen[one]:
			return fmt.Errorf("%s: %s is named twice", e.Entity, one)
		case standing != "" && one != e.Entity:
			return fmt.Errorf("%s: the corpus already has a %s called %q",
				e.Entity, standing, one.Slug)
		}
		seen[one] = true
	}

	if first := e.Into[0]; first != e.Entity {
		if err := rename(corpus, model.Rename{Entity: e.Entity, To: first.Slug}); err != nil {
			return err
		}
	}
	for _, one := range e.Into[1:] {
		if err := create(corpus, model.Create{Entity: one}); err != nil {
			return err
		}
	}
	return nil
}
