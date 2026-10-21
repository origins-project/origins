package model

import (
	"fmt"
	"reflect"
	"slices"
)

// Hiding is which of the two kinds of hiding a view applies. They are independent,
// because a tree shown to relatives may keep its living members and hide what is
// private, or the reverse.
type Hiding struct {
	Private bool // what the researcher has marked private
	Living  bool // whoever is taken as living
}

// Any reports whether the view hides anything at all.
func (h Hiding) Any() bool { return h.Private || h.Living }

// String says what the view leaves out, as the end of a sentence.
func (h Hiding) String() string {
	switch {
	case h.Private && h.Living:
		return "what is private and who is living"
	case h.Private:
		return "what is private"
	case h.Living:
		return "who is living"
	}
	return "nothing"
}

// PlaceholderName is the name every hidden person is shown under.
const PlaceholderName = "Private"

// placeholderOf is the person a hidden view shows in the place of somebody it hides.
func placeholderOf(slug Slug) *Person {
	return &Person{Slug: slug, Private: true, Name: PersonName{Text: PlaceholderName}}
}

// IsPlaceholder reports whether the person is one a hidden view put in the place of
// somebody it hides, saying nothing of them but that they are there.
func (p *Person) IsPlaceholder() bool {
	return reflect.DeepEqual(p, placeholderOf(p.Slug))
}

// Hide returns the corpus as somebody it is not meant for in full would see it, hiding
// what hiding asks for and deciding who is living as of today. It is the one place
// where what is hidden is decided, and what it returns is a corpus like any other, so
// that whatever reads one reads a hidden view without knowing. The corpus handed to it
// is not changed, and asked to hide nothing it is what comes back.
//
// A hidden person is replaced rather than dropped, and so is every reference to them,
// so that the tree keeps its shape and loses what is in it. A family with a hidden
// partner, or private itself, keeps its partners and nothing else. A private event,
// name or picture is removed from its record, and so is any picture of a hidden person,
// with every entry elsewhere referring to a picture removed; a private place, source,
// repository or note is dropped with every reference to it; and so, afterwards, is any
// place, source or repository that only what has gone referred to. A claim that two
// records are one keeps neither in the view, and goes where the view does not show the
// other as it is. Hypotheses are not in a hidden view, the corpus being the standing
// conclusion.
//
// Words are not read: a body or a transcription naming a hidden person is as it was. A
// link is not words but a reference, and one to a place, source, repository or note
// that went goes with it, its text included.
func Hide(c *Corpus, hiding Hiding, today Date) *Corpus {
	if !hiding.Any() {
		return c
	}
	var hiddenPeople []Slug
	lives := c.Lives(today)
	for _, slug := range c.PeopleSlugs() {
		person := c.People[slug]
		if hiding.Private && person.Private || hiding.Living && lives.Living(person) {
			hiddenPeople = append(hiddenPeople, slug)
		}
	}
	h := hider{Hiding: hiding, placeholders: map[Slug]Slug{}, stripped: map[Slug]bool{}}
	n := 0
	for _, slug := range hiddenPeople {
		for {
			n++
			placeholder := Slug(fmt.Sprintf("private-%d", n))
			if slices.Contains(hiddenPeople, placeholder) || !c.Taken(placeholder) {
				h.placeholders[slug] = placeholder
				break
			}
		}
	}
	nobodys := unreferenced(c)

	out := c.Clone()
	out.Hypotheses = map[Slug]*Hypothesis{}
	if placeholder, hidden := h.placeholders[out.Project.MainPerson]; hidden {
		out.Project.MainPerson = placeholder
	}

	people := map[Slug]*Person{}
	for slug, person := range out.People {
		if placeholder, hidden := h.placeholders[slug]; hidden {
			person = placeholderOf(placeholder)
		} else {
			h.person(person)
		}
		people[person.Slug] = person
	}
	out.People = people

	families := map[Slug]*Family{}
	renamed := map[Slug]Slug{}
	for slug, family := range out.Families {
		if hiding.Private && family.Private || slices.ContainsFunc(family.Partners, h.hiddenPartner) {
			h.stripped[slug] = true
			if family = h.family(family); family == nil {
				renamed[slug] = ""
				continue
			}
		} else {
			family.Children = h.children(family.Children, false)
			family.Events, _ = h.events(family.Events, &family.Files)
			family.Media, _ = h.media(family.Media, &family.Files)
		}
		renamed[slug] = family.Slug
		families[family.Slug] = family
	}
	out.Families = families
	// a family named after a hidden partner goes by the name the placeholder makes
	rename := func(events []Event) {
		for i := range events {
			if to, ok := renamed[events[i].Family]; ok {
				events[i].Family = to
			}
		}
	}
	for _, person := range out.People {
		rename(person.Events)
	}
	for _, note := range out.Notes {
		rename(note.Events)
	}

	for _, place := range out.Places {
		place.Media, _ = h.media(place.Media, &place.Files)
	}
	for _, source := range out.Sources {
		h.source(source)
	}
	for _, repository := range out.Repositories {
		repository.Media, _ = h.media(repository.Media, &repository.Files)
	}
	for _, note := range out.Notes {
		note.Events, _ = h.events(note.Events, &note.Files)
		note.Media, _ = h.media(note.Media, &note.Files)
	}

	if hiding.Private {
		dropped := map[Slug]bool{}
		for _, kind := range []Kind{KindPlace, KindSource, KindRepository, KindNote} {
			for _, slug := range out.SlugsOf(kind) {
				if private(out.Entity(slug)) {
					dropped[slug] = true
					out.Delete(slug)
				}
			}
		}
		forget(out, dropped)
	}
	unrefer(out, removedPictures(c, out))
	sweep(out, nobodys)
	unlinkGone(c, out)
	h.unclaim(c, out)
	out.Project.Bookmarks = keep(out.Project.Bookmarks, func(b EntityRef) bool {
		return !h.hidden(b.Slug) && !h.stripped[b.Slug] && out.KindOf(b.Slug) == b.Kind
	})
	return out
}

// hider is what a view has decided about who is hidden, and what it does to the
// entries of a record that stays.
type hider struct {
	Hiding
	placeholders map[Slug]Slug // the slug each hidden person is shown under
	stripped     map[Slug]bool // the families shown as their partners alone
}

func (h hider) hidden(s Slug) bool {
	_, ok := h.placeholders[s]
	return ok
}

func (h hider) hiddenPartner(p Partner) bool { return h.hidden(p.Person) }

// person takes out of a visible person's record what the view leaves out, and puts a
// placeholder wherever it names somebody hidden.
func (h hider) person(p *Person) {
	var removed, more []string
	p.Events, removed = h.events(p.Events, &p.Files)
	p.AlsoKnownAs = keep(p.AlsoKnownAs, func(n PersonName) bool { return !(h.Private && n.Private) })
	p.Media, more = h.media(p.Media, &p.Files)
	if slices.Contains(removed, p.Portrait.File) || slices.Contains(more, p.Portrait.File) {
		p.Portrait = Image{}
	}
}

// children puts a placeholder for each hidden child, keeping the relation and the status
// and losing what is said of the link. A hidden family's children all lose it, it being
// a family about which nothing is shown, and are named as its partners are.
func (h hider) children(list []Child, all bool) []Child {
	for i, child := range list {
		placeholder, hidden := h.placeholders[child.ID]
		if !hidden && !all {
			continue
		}
		bare := Child{ID: child.Who(), Relation: child.Relation.Clone(), Status: child.Status}
		if hidden {
			bare.ID = placeholder
		}
		for j, r := range bare.Relation.Partners {
			if placeholder, hidden := h.placeholders[r.Partner]; hidden {
				bare.Relation.Partners[j].Partner = placeholder
			}
		}
		list[i] = bare
	}
	return list
}

// events removes the private ones, where they are hidden, and puts a placeholder for
// each hidden person an event names, keeping the role. The pictures of an event that
// stays are removed as a record's are, from the files of the record holding it, and
// the files they took are returned.
func (h hider) events(list []Event, files *[]string) ([]Event, []string) {
	list = keep(list, func(e Event) bool { return !(h.Private && e.Private) })
	var removed []string
	for i := range list {
		h.named(list[i].With)
		var took []string
		list[i].Media, took = h.media(list[i].Media, files)
		removed = append(removed, took...)
	}
	return list, removed
}

func (h hider) named(list []Named) {
	for j, named := range list {
		if placeholder, hidden := h.placeholders[named.Person]; hidden {
			list[j] = Named{Person: placeholder, Role: named.Role}
		}
	}
}

// source puts a placeholder for each hidden person an entry names, and removes the
// scans and pictures the view leaves out.
func (h hider) source(s *Source) {
	s.Media, _ = h.media(s.Media, &s.Files)
	for anchor, citation := range s.Citations {
		h.named(citation.With)
		citation.Images, _ = h.media(citation.Images, &s.Files)
		s.Citations[anchor] = citation
	}
}

// media removes the private pictures, where they are hidden, and every picture of a
// hidden person, the picture being the person. A picture removed takes its file with
// it, and the files it took are returned.
func (h hider) media(list []Media, files *[]string) ([]Media, []string) {
	var removed []string
	list = keep(list, func(m Media) bool {
		if h.Private && m.Private ||
			slices.ContainsFunc(m.Depicts, func(d Depicted) bool { return h.hidden(d.Person) }) {
			removed = append(removed, m.File)
			return false
		}
		return true
	})
	if removed != nil {
		*files = keep(*files, func(file string) bool { return !slices.Contains(removed, file) })
	}
	return list, removed
}

// family is what is shown of a family with a hidden partner, or private itself: its
// partners and children as everything worked out from it counts them, as placeholders
// where they are hidden, under the name the partners make. A family told apart from
// another of the same partners stays told apart by the same words.
func (h hider) family(u *Family) *Family {
	partners := make([]Partner, len(u.Partners))
	for i, partner := range u.Partners {
		partners[i] = Partner{Person: partner.Who()}
		if placeholder, hidden := h.placeholders[partner.Person]; hidden {
			partners[i] = Partner{Person: placeholder}
		}
	}
	now, err := FamilySlug(partners)
	if err != nil {
		return nil
	}
	slug := now
	if was, err := FamilySlug(u.Partners); err == nil {
		slug = u.Slug.Rebase(was, now)
	}
	return &Family{Slug: slug, Private: true, Partners: partners, Children: h.children(u.Children, true)}
}

// unclaim takes out every claim that a record is the same as one the view does not show
// as it is in c: one that went, a hidden person, a family shown as its partners alone.
// A claim naming nobody with a record says nothing of anybody hidden, and stays.
func (h hider) unclaim(c, view *Corpus) {
	for _, slug := range view.AllSlugs() {
		same := SameAsOf(view.Entity(slug))
		if same == nil {
			continue
		}
		*same = keep(*same, func(claim SameAs) bool {
			if claim.ID == "" {
				return claim.Name != ""
			}
			return !h.hidden(claim.ID) && !h.stripped[claim.ID] && view.Entity(claim.ID) != nil &&
				view.KindOf(claim.ID) == c.KindOf(claim.ID)
		})
	}
}

func private(entity any) bool {
	switch e := entity.(type) {
	case *Place:
		return e.Private
	case *Source:
		return e.Private
	case *Repository:
		return e.Private
	case *Note:
		return e.Private
	}
	return false
}

// forget takes out every reference to a record that is no longer there: an event's, a
// picture's or a source's coverage's place is cleared, and what contains a place, a
// citation and an entry of where a source is kept are removed.
func forget(c *Corpus, gone map[Slug]bool) {
	for _, slug := range c.AllSlugs() {
		entity := c.Entity(slug)
		if source, ok := entity.(*Source); ok {
			source.Repositories = keep(source.Repositories, func(held Holding) bool { return !gone[held.Repository] })
		}
		References(entity, func(s *Slug) {
			if gone[*s] {
				*s = ""
			}
		})
		citationLists(entity, func(refs *[]CitationRef) {
			*refs = keep(*refs, func(ref CitationRef) bool { return ref.Source != "" })
		})
		if place, ok := entity.(*Place); ok {
			place.Within = keep(place.Within, func(within PlaceRef) bool { return within.Place != "" })
		}
	}
}

// unlinkGone takes out of the view's prose every link to a place, source, repository or
// note it dropped, text and all: the text of a link to a note is the note's first line,
// which would show of a private note what the view leaves out. A link to somebody hidden
// stays, being words about them.
func unlinkGone(c, view *Corpus) {
	gone := func(r EntityRef) bool {
		switch r.Kind {
		case KindPlace, KindSource, KindRepository, KindNote:
			return c.KindOf(r.Slug) == r.Kind && view.KindOf(r.Slug) != r.Kind
		}
		return false
	}
	for _, slug := range view.AllSlugs() {
		entity := view.Entity(slug)
		from := RefOf(entity)
		Prose(entity, func(_ string, text *string) { *text = unlink(from, *text, gone) })
	}
}

// picture is a file beside a record.
type picture struct {
	record EntityRef
	file   string
}

// removedPictures is every picture the view removed: one a record kept, as a file
// beside it or an entry of its own, that the record in the view keeps neither way, or
// whose record the view does not have.
func removedPictures(c, view *Corpus) map[picture]bool {
	removed := map[picture]bool{}
	for _, slug := range c.AllSlugs() {
		record := c.Entity(slug)
		from, files := RefOf(record), *Files(record)
		kept := slices.Clone(files)
		Pictures(record, func(_ string, m *Media) {
			if at := Locate(from, m.File); !m.Refers() && at.Record == from {
				kept = append(kept, at.File)
			}
		})
		shown := view.Entity(slug)
		for _, file := range kept {
			if shown == nil || view.KindOf(slug) != from.Kind ||
				!slices.Contains(*Files(shown), file) && (slices.Contains(files, file) || !Keeps(shown, file)) {
				removed[picture{from, file}] = true
			}
		}
	}
	return removed
}

// unrefer removes every media entry naming a picture that was removed, a reference to
// it from another record or an entry of its keeper's own.
func unrefer(c *Corpus, removed map[picture]bool) {
	if len(removed) == 0 {
		return
	}
	for _, slug := range c.AllSlugs() {
		record := c.Entity(slug)
		from := RefOf(record)
		kept := func(m Media) bool {
			at := Locate(from, m.File)
			return !removed[picture{at.Record, at.File}]
		}
		events := func(list []Event) {
			for i := range list {
				list[i].Media = keep(list[i].Media, kept)
			}
		}
		switch e := record.(type) {
		case *Person:
			events(e.Events)
			e.Media = keep(e.Media, kept)
		case *Family:
			events(e.Events)
			e.Media = keep(e.Media, kept)
		case *Place:
			e.Media = keep(e.Media, kept)
		case *Source:
			e.Media = keep(e.Media, kept)
		case *Repository:
			e.Media = keep(e.Media, kept)
		case *Note:
			events(e.Events)
			e.Media = keep(e.Media, kept)
		}
	}
}

// unreferenced is the places, sources and repositories nothing else in the corpus
// refers to.
func unreferenced(c *Corpus) map[Slug]bool {
	referred := referredTo(c)
	out := map[Slug]bool{}
	for _, kind := range []Kind{KindPlace, KindSource, KindRepository} {
		for _, slug := range c.SlugsOf(kind) {
			if !referred[slug] {
				out[slug] = true
			}
		}
	}
	return out
}

// referredTo is every slug a record refers to other than itself, a hypothesis's
// references counting for nothing, and nor does a claim that two records are one, which
// is about the record and not a use of it.
func referredTo(c *Corpus) map[Slug]bool {
	referred := map[Slug]bool{}
	for _, slug := range c.AllSlugs() {
		if c.KindOf(slug) == KindHypothesis {
			continue
		}
		entity, times := c.Entity(slug), map[Slug]int{}
		References(entity, func(s *Slug) {
			if *s != slug {
				times[*s]++
			}
		})
		if same := SameAsOf(entity); same != nil {
			for _, claim := range *same {
				if claim.ID != "" && claim.ID != slug {
					times[claim.ID]--
				}
			}
		}
		for s, n := range times {
			if n > 0 {
				referred[s] = true
			}
		}
	}
	return referred
}

// sweep drops every place, source and repository nothing refers to any longer, and
// again until nothing more goes: a village that only a hidden person's events named
// says where they live, and once it has gone the region containing it is named by
// nothing. One that nothing referred to in the first place, which nobodys lists, is
// nobody's and stays.
func sweep(c *Corpus, nobodys map[Slug]bool) {
	for {
		referred, gone := referredTo(c), false
		for _, kind := range []Kind{KindPlace, KindSource, KindRepository} {
			for _, slug := range c.SlugsOf(kind) {
				if !referred[slug] && !nobodys[slug] {
					c.Delete(slug)
					gone = true
				}
			}
		}
		if !gone {
			return
		}
	}
}

// keep is the entries of list that ok accepts, the list itself where it accepts them
// all and nil where it accepts none. It never writes into list, which may be shared.
func keep[T any](list []T, ok func(T) bool) []T {
	if !slices.ContainsFunc(list, func(v T) bool { return !ok(v) }) {
		return list
	}
	var out []T
	for _, v := range list {
		if ok(v) {
			out = append(out, v)
		}
	}
	return out
}
