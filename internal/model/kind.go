package model

import "slices"

// Kind names the sort of entity a reference points at. Slugs are unique across all
// kinds, so the kind is redundant for lookup and useful for reading.
type Kind string

// The kinds a reference can point at.
const (
	KindPerson     Kind = "person"
	KindFamily     Kind = "family"
	KindPlace      Kind = "place"
	KindSource     Kind = "source"
	KindRepository Kind = "repository"
	KindNote       Kind = "note"
	KindHypothesis Kind = "hypothesis"

	// KindProject is the manifest, which an edit addresses beside the records. It is
	// not one of Kinds: there is one of it, it has no directory, and nothing names it.
	KindProject Kind = "project"

	// KindTag is a tag, which an edit addresses to rename it. It is not one of Kinds
	// either: a tag is a word the records carry and has no file of its own.
	KindTag Kind = "tag"
)

// Kinds is every kind there is, in the order a corpus lists them.
//
// This file is the one place that goes from a kind to the records of it and back.
// Anything else that has to treat a record without knowing its kind asks it here, so
// that another kind is added in this file and in what genuinely differs between kinds,
// and nowhere else.
var Kinds = []Kind{KindPerson, KindFamily, KindPlace, KindSource, KindRepository, KindNote, KindHypothesis}

// New is an empty record of the kind, or nil where kind is not one.
func New(kind Kind) any {
	switch kind {
	case KindPerson:
		return &Person{}
	case KindFamily:
		return &Family{}
	case KindPlace:
		return &Place{}
	case KindSource:
		return &Source{}
	case KindRepository:
		return &Repository{}
	case KindNote:
		return &Note{}
	case KindHypothesis:
		return &Hypothesis{}
	}
	return nil
}

// Dir is the directory the records of the kind are kept in, each in a directory of its
// own named by its slug, or "" where kind is not one.
func (k Kind) Dir() string {
	switch k {
	case KindPerson:
		return "people"
	case KindFamily:
		return "families"
	case KindPlace:
		return "places"
	case KindSource:
		return "sources"
	case KindRepository:
		return "repositories"
	case KindNote:
		return "notes"
	case KindHypothesis:
		return "hypotheses"
	}
	return ""
}

// KindIn is the kind whose records are kept in dir, or "" where none is.
func KindIn(dir string) Kind {
	for _, kind := range Kinds {
		if kind.Dir() == dir {
			return kind
		}
	}
	return ""
}

// RefOf is what entity is and what it is called, or the zero reference where it is not
// an entity.
func RefOf(entity any) EntityRef {
	switch e := entity.(type) {
	case *Person:
		return EntityRef{KindPerson, e.Slug}
	case *Family:
		return EntityRef{KindFamily, e.Slug}
	case *Place:
		return EntityRef{KindPlace, e.Slug}
	case *Source:
		return EntityRef{KindSource, e.Slug}
	case *Repository:
		return EntityRef{KindRepository, e.Slug}
	case *Note:
		return EntityRef{KindNote, e.Slug}
	case *Hypothesis:
		return EntityRef{KindHypothesis, e.Slug}
	}
	return EntityRef{}
}

// Files is where an entity lists the files kept beside it, as somewhere to change
// rather than a copy of the list, or nil where it is not an entity.
//
// The list is never written to the frontmatter: which files are there is a question
// for the directory, and loading answers it. Within a change it is what the entity is
// to have, so that folding one record into another can say that its photographs now
// sit beside the other.
func Files(entity any) *[]string {
	switch e := entity.(type) {
	case *Person:
		return &e.Files
	case *Family:
		return &e.Files
	case *Place:
		return &e.Files
	case *Source:
		return &e.Files
	case *Repository:
		return &e.Files
	case *Note:
		return &e.Files
	case *Hypothesis:
		return &e.Files
	}
	return nil
}

// SameAsOf is where a record lists the records it is said to be the same as, as
// somewhere to change rather than a copy of the list, or nil where it is not a record
// that can say so: a hypothesis is an argument and not a record of anything.
func SameAsOf(entity any) *[]SameAs {
	switch e := entity.(type) {
	case *Person:
		return &e.SameAs
	case *Family:
		return &e.SameAs
	case *Place:
		return &e.SameAs
	case *Source:
		return &e.SameAs
	case *Repository:
		return &e.SameAs
	case *Note:
		return &e.SameAs
	}
	return nil
}

// bodyOf is where an entity keeps its body, or nil where it is not an entity.
func bodyOf(entity any) *string {
	switch e := entity.(type) {
	case *Person:
		return &e.Body
	case *Family:
		return &e.Body
	case *Place:
		return &e.Body
	case *Source:
		return &e.Body
	case *Repository:
		return &e.Body
	case *Note:
		return &e.Body
	case *Hypothesis:
		return &e.Body
	}
	return nil
}

// Entity is the record with this slug, whatever kind it is, or nil where nothing has
// that name.
func (c *Corpus) Entity(slug Slug) any {
	switch {
	case c.People[slug] != nil:
		return c.People[slug]
	case c.Families[slug] != nil:
		return c.Families[slug]
	case c.Places[slug] != nil:
		return c.Places[slug]
	case c.Sources[slug] != nil:
		return c.Sources[slug]
	case c.Repositories[slug] != nil:
		return c.Repositories[slug]
	case c.Notes[slug] != nil:
		return c.Notes[slug]
	case c.Hypotheses[slug] != nil:
		return c.Hypotheses[slug]
	}
	return nil
}

// KindOf is the kind of the record with this slug, or "" where nothing has that name.
func (c *Corpus) KindOf(slug Slug) Kind { return RefOf(c.Entity(slug)).Kind }

// Put files entity under slug, which becomes its name, in whichever kind it is. A
// record already holding the slug in another kind is left where it is: Put is told
// what to file, and whether the name is free is the caller's question.
func (c *Corpus) Put(slug Slug, entity any) {
	switch e := entity.(type) {
	case *Person:
		e.Slug, c.People[slug] = slug, e
	case *Family:
		e.Slug, c.Families[slug] = slug, e
	case *Place:
		e.Slug, c.Places[slug] = slug, e
	case *Source:
		e.Slug, c.Sources[slug] = slug, e
	case *Repository:
		e.Slug, c.Repositories[slug] = slug, e
	case *Note:
		e.Slug, c.Notes[slug] = slug, e
	case *Hypothesis:
		e.Slug, c.Hypotheses[slug] = slug, e
	}
}

// Delete takes the record with this slug out of the corpus, whatever kind it is.
func (c *Corpus) Delete(slug Slug) {
	delete(c.People, slug)
	delete(c.Families, slug)
	delete(c.Places, slug)
	delete(c.Sources, slug)
	delete(c.Repositories, slug)
	delete(c.Notes, slug)
	delete(c.Hypotheses, slug)
}

// SlugsOf returns the slugs of every record of the kind, in slug order.
func (c *Corpus) SlugsOf(kind Kind) []Slug {
	switch kind {
	case KindPerson:
		return c.PeopleSlugs()
	case KindFamily:
		return c.FamilySlugs()
	case KindPlace:
		return c.PlaceSlugs()
	case KindSource:
		return c.SourceSlugs()
	case KindRepository:
		return c.RepositorySlugs()
	case KindNote:
		return c.NoteSlugs()
	case KindHypothesis:
		return c.HypothesisSlugs()
	}
	return nil
}

// AllSlugs returns the slugs of every record of every kind in one sorted run. Slugs
// share a namespace across the kinds, so each can be asked for again through Entity
// without the caller knowing which kind it was.
func (c *Corpus) AllSlugs() []Slug {
	var out []Slug
	for _, kind := range Kinds {
		out = append(out, c.SlugsOf(kind)...)
	}
	slices.Sort(out)
	return out
}
