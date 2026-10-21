package ops

import (
	"fmt"
	"maps"
	"slices"

	"github.com/origins-project/origins/internal/model"
)

// rename gives an entity a new name and rewrites everything that pointed at the old
// one.
//
// The slug is the identity and the file name at once, so there is no id field
// somewhere to keep in step — there is only every other entity that named this one.
// They all change in the same breath, because a corpus that briefly holds a child
// whose parent does not exist is a corpus somebody could read in that state.
func rename(corpus *model.Corpus, e model.Rename) error {
	if _, err := model.ParseSlug(string(e.To)); err != nil {
		return fmt.Errorf("%s: %w", e.Entity, err)
	}
	if e.To == e.Entity.Slug {
		return fmt.Errorf("%s: that is the name it has", e.Entity)
	}
	if _, err := entity(corpus, e.Entity); err != nil {
		return err
	}
	if e.To.Reserved() {
		return fmt.Errorf("%s: %w", e.Entity, reserved(e.To))
	}
	if kind := corpus.KindOf(e.To); kind != "" {
		return fmt.Errorf("%s: the corpus already has a %s called %q", e.Entity, kind, e.To)
	}

	moves := map[model.Slug]model.Slug{e.Entity.Slug: e.To}
	if e.Entity.Kind == model.KindPerson {
		named(corpus, e.Entity.Slug, e.To, moves)
	}
	references(corpus, func(slug *model.Slug) {
		if to, moving := moves[*slug]; moving {
			*slug = to
		}
	})
	for from, to := range moves {
		rekey(corpus, from, to)
	}
	return nil
}

// reserved is why a record cannot be given one of the two words a partner without a
// record is written as.
func reserved(slug model.Slug) error {
	return fmt.Errorf("%q is the word for a partner without a record, and names no record", slug)
}

// named adds to moves the families that the renaming of a person renames as well.
//
// A family is named by its partners, so a partner renamed and the family left alone
// leaves the corpus in a state check calls an error — and a rename that has to be
// followed by a second one before the corpus is legal again is not one origins can
// offer. A family whose new name another already has is told apart from it rather than
// refused or merged: one couple's marriage recorded twice and a couple who married
// twice look the same from the names, and which it is is the researcher's to say.
func named(corpus *model.Corpus, from, to model.Slug, moves map[model.Slug]model.Slug) {
	taken := func(slug model.Slug) bool {
		return corpus.Taken(slug) || slices.Contains(slices.Collect(maps.Values(moves)), slug)
	}
	for _, slug := range corpus.FamiliesOf(from) {
		family := corpus.Families[slug]
		partners := slices.Clone(family.Partners)
		for i := range partners {
			if partners[i].Person == from {
				partners[i].Person = to
			}
		}
		if renamed := refamily(corpus, family, family.Partners, partners, taken); renamed != slug {
			moves[slug] = renamed
		}
	}
}

// refamily is the name a family goes by once its partners are now rather than was: the
// one they make, keeping whatever told it apart from another family of the same
// partners, and told apart again by what it says itself where that name is taken. A
// family its partners did not name keeps the name it has, which is somebody's to put
// right rather than a name to work out.
func refamily(corpus *model.Corpus, family *model.Family, was, now []model.Partner, taken func(model.Slug) bool) model.Slug {
	before, err := model.FamilySlug(was)
	if err != nil {
		return family.Slug
	}
	after, err := model.FamilySlug(now)
	if err != nil || after == before {
		return family.Slug
	}
	if _, named := family.Slug.Discriminator(before); !named {
		return family.Slug
	}
	if rebased := family.Slug.Rebase(before, after); !taken(rebased) {
		return rebased
	}
	proposals, err := model.Discriminate(after, corpus.FamilyHints(family), taken)
	if err != nil || len(proposals) == 0 {
		return family.Slug
	}
	return proposals[0]
}

// regroup renames a family whose partners an edit has changed after the partners it
// now has, as renaming one of them would: putting somebody in the place of unknown is
// finding who they were, and the family is named by who they were.
func regroup(corpus *model.Corpus, family *model.Family, was []model.Partner) {
	from := family.Slug
	to := refamily(corpus, family, was, family.Partners, corpus.Taken)
	if to == from {
		return
	}
	references(corpus, func(slug *model.Slug) {
		if *slug == from {
			*slug = to
		}
	})
	rekey(corpus, from, to)
}

// cascade is the entities a rename renames besides the one it was asked about: the
// families whose own names are made out of the partner being renamed.
//
// The commit has to name them. A family's file moves in that commit like any other
// renamed entity's, and an entity whose renaming nothing recorded is one whose history
// stops there — a search for it under the new name finds a record that begins the day
// it was renamed, and a search under the old one finds a record that ends then.
//
// It is read before the rename is applied, while the family still answers to the name
// it is losing.
func cascade(corpus *model.Corpus, e model.Rename) []model.EntityRef {
	if e.Entity.Kind != model.KindPerson {
		return nil
	}
	moves := map[model.Slug]model.Slug{}
	named(corpus, e.Entity.Slug, e.To, moves)

	refs := make([]model.EntityRef, 0, 2*len(moves))
	for _, from := range slices.Sorted(maps.Keys(moves)) {
		refs = append(refs,
			model.EntityRef{Kind: model.KindFamily, Slug: from},
			model.EntityRef{Kind: model.KindFamily, Slug: moves[from]})
	}
	return refs
}

// rekey files an entity under a different name, which for an entity is the whole of
// being renamed: the slug is the identity and no second copy of it is kept anywhere.
func rekey(corpus *model.Corpus, from, to model.Slug) {
	entity := corpus.Entity(from)
	corpus.Delete(from)
	corpus.Put(to, entity)
}

// references hands every slug written anywhere in the corpus to a function that may
// change it: the main person and the bookmarks, what each record refers to as model.References walks it,
// which leaves a hypothesis's edits alone, and every record a link in prose leads into,
// as model.Relink finds them.
func references(corpus *model.Corpus, rewrite func(*model.Slug)) {
	rewrite(&corpus.Project.MainPerson)
	for i := range corpus.Project.Bookmarks {
		rewrite(&corpus.Project.Bookmarks[i].Slug)
	}
	for _, slug := range corpus.AllSlugs() {
		model.References(corpus.Entity(slug), rewrite)
		model.Relink(corpus.Entity(slug), rewrite)
	}
}
