package model

import (
	"cmp"
	"slices"
	"strconv"
)

// Corpus is the whole body of data in memory. Entities reference each other by slug and
// never by pointer, so the graph has no ownership to untangle and no cycles to break.
type Corpus struct {
	Project      Project
	People       map[Slug]*Person
	Families     map[Slug]*Family
	Places       map[Slug]*Place
	Sources      map[Slug]*Source
	Repositories map[Slug]*Repository
	Notes        map[Slug]*Note
	Hypotheses   map[Slug]*Hypothesis
}

// NewCorpus is an empty corpus, every map made.
func NewCorpus() *Corpus {
	return &Corpus{
		People:       map[Slug]*Person{},
		Families:     map[Slug]*Family{},
		Places:       map[Slug]*Place{},
		Sources:      map[Slug]*Source{},
		Repositories: map[Slug]*Repository{},
		Notes:        map[Slug]*Note{},
		Hypotheses:   map[Slug]*Hypothesis{},
	}
}

// Taken reports whether any entity already holds the slug, or it is one of the words a
// partner is written as where there is no record. Slugs share one namespace across
// entity types, so that a reference names exactly one thing.
func (c *Corpus) Taken(s Slug) bool { return c.Entity(s) != nil || s.Reserved() }

// PeopleSlugs returns the people's slugs in slug order.
func (c *Corpus) PeopleSlugs() []Slug { return sortedKeys(c.People) }

// FamilySlugs returns the families' slugs in slug order.
func (c *Corpus) FamilySlugs() []Slug { return sortedKeys(c.Families) }

// PlaceSlugs returns the places' slugs in slug order.
func (c *Corpus) PlaceSlugs() []Slug { return sortedKeys(c.Places) }

// SourceSlugs returns the sources' slugs in slug order.
func (c *Corpus) SourceSlugs() []Slug { return sortedKeys(c.Sources) }

// RepositorySlugs returns the repositories' slugs in slug order.
func (c *Corpus) RepositorySlugs() []Slug { return sortedKeys(c.Repositories) }

// NoteSlugs returns the notes' slugs in slug order.
func (c *Corpus) NoteSlugs() []Slug { return sortedKeys(c.Notes) }

// HypothesisSlugs returns the hypotheses' slugs in slug order.
func (c *Corpus) HypothesisSlugs() []Slug { return sortedKeys(c.Hypotheses) }

// sortedKeys sorts because map iteration in Go is deliberately random, and everything
// downstream of derivation has to be stable.
func sortedKeys[V any](m map[Slug]V) []Slug {
	out := make([]Slug, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// Parent is somebody a person is a child of through one family: a partner's Who, so a
// slug or Unknown, and the relation as the family writes it.
type Parent struct {
	Slug     Slug
	Relation Relation
	Family   Slug
}

// ParentsOf returns a person's parents from every family listing them, in the order of
// the families' slugs and of the partners within each. A disproven link gives none, nor
// does a partner who is none or whom the child is no child of.
func (c *Corpus) ParentsOf(person Slug) []Parent {
	var listing []Slug
	for slug, family := range c.Families {
		if slices.ContainsFunc(family.Children, func(child Child) bool { return child.ID == person }) {
			listing = append(listing, slug)
		}
	}
	slices.Sort(listing)
	var out []Parent
	for _, slug := range listing {
		for _, child := range c.Families[slug].Children {
			if child.ID == person {
				out = c.parentsThrough(slug, child, out)
			}
		}
	}
	return out
}

// Parentage returns the parents of everyone a family lists, as ParentsOf gives them,
// walking the families once for whatever asks of everybody.
func (c *Corpus) Parentage() map[Slug][]Parent {
	out := map[Slug][]Parent{}
	for _, slug := range c.FamilySlugs() {
		for _, child := range c.Families[slug].Children {
			if child.Somebody() {
				out[child.ID] = c.parentsThrough(slug, child, out[child.ID])
			}
		}
	}
	return out
}

func (c *Corpus) parentsThrough(family Slug, child Child, out []Parent) []Parent {
	if child.Status == Disproven {
		return out
	}
	for _, partner := range c.Families[family].Partners {
		relation := child.To(partner.Who())
		if partner.Person == None || relation.Kin() == NoRelation {
			continue
		}
		out = append(out, Parent{Slug: partner.Who(), Relation: relation, Family: family})
	}
	return out
}

// Children returns everyone a family lists as a child of parent, the link neither
// disproven nor none.
func (c *Corpus) Children(parent Slug) []Slug {
	var out []Slug
	for _, family := range c.Families {
		if !slices.ContainsFunc(family.Partners, func(p Partner) bool { return p.Person == parent }) {
			continue
		}
		for _, child := range family.Children {
			if child.Somebody() && child.Status != Disproven && child.To(parent).Kin() != NoRelation &&
				!slices.Contains(out, child.ID) {
				out = append(out, child.ID)
			}
		}
	}
	slices.Sort(out)
	return out
}

// ChildOf returns the families listing person as a child, the link not disproven.
func (c *Corpus) ChildOf(person Slug) []Slug {
	var out []Slug
	for slug, family := range c.Families {
		if slices.ContainsFunc(family.Children, func(child Child) bool {
			return child.ID == person && child.Status != Disproven
		}) {
			out = append(out, slug)
		}
	}
	slices.Sort(out)
	return out
}

// Ties are the families each person is a partner in and a child of, as FamiliesOf and
// ChildOf give them, walking the families once for whatever asks of everybody.
type Ties struct {
	PartnerIn, ChildIn map[Slug][]Slug
}

// Ties returns every person's ties.
func (c *Corpus) Ties() Ties {
	ties := Ties{PartnerIn: map[Slug][]Slug{}, ChildIn: map[Slug][]Slug{}}
	once := func(in map[Slug][]Slug, person, family Slug) {
		if list := in[person]; len(list) == 0 || list[len(list)-1] != family {
			in[person] = append(list, family)
		}
	}
	for _, slug := range c.FamilySlugs() {
		for _, partner := range c.Families[slug].Partners {
			once(ties.PartnerIn, partner.Person, slug)
		}
		for _, child := range c.Families[slug].Children {
			if child.Status != Disproven {
				once(ties.ChildIn, child.ID, slug)
			}
		}
	}
	return ties
}

// Siblings splits the people who share a parent with person into those sharing two or
// more and those sharing exactly one. person is never their own sibling.
func (c *Corpus) Siblings(person Slug) (full, half []Slug) {
	shared := map[Slug]int{}
	for _, parent := range c.parentSlugs(person) {
		for _, child := range c.Children(parent) {
			if child != person {
				shared[child]++
			}
		}
	}
	for slug, count := range shared {
		if count >= 2 {
			full = append(full, slug)
		} else {
			half = append(half, slug)
		}
	}
	slices.Sort(full)
	slices.Sort(half)
	return full, half
}

// FamiliesOf returns the families a person is a partner in.
func (c *Corpus) FamiliesOf(person Slug) []Slug {
	var out []Slug
	for slug, family := range c.Families {
		if slices.ContainsFunc(family.Partners, func(p Partner) bool { return p.Person == person }) {
			out = append(out, slug)
		}
	}
	slices.Sort(out)
	return out
}

// Partners returns everyone with a record a person shares a family with.
func (c *Corpus) Partners(person Slug) []Slug {
	var out []Slug
	for _, slug := range c.FamiliesOf(person) {
		for _, partner := range c.Families[slug].Partners {
			if partner.Somebody() && partner.Person != person && !slices.Contains(out, partner.Person) {
				out = append(out, partner.Person)
			}
		}
	}
	slices.Sort(out)
	return out
}

// FamilyHints are what tell a family apart from another its partners name, best first:
// the year of its first dated event, where that happened, and the surname of the first
// child it lists with a record.
func (c *Corpus) FamilyHints(family *Family) []string {
	var hints []string
	for _, event := range family.Events {
		if start := event.Date.Start(); start.Year != nil {
			hints = append(hints, strconv.Itoa(*start.Year))
			if event.Place != "" {
				hints = append(hints, string(event.Place))
			}
			break
		}
	}
	for _, child := range family.Children {
		if person, ok := c.People[child.ID]; ok && child.Status != Disproven && person.Name.PrimarySurname().Name != "" {
			hints = append(hints, person.Name.PrimarySurname().Name)
			break
		}
	}
	return hints
}

// Naming is a place in another record's event where a person is named: the event,
// which of its `with` entries names them, and in what role.
type Naming struct {
	Record EntityRef
	Event  int
	Entry  int
	Role   string
}

// NamedIn returns everywhere an event names person in its `with`, in the order of the
// record's slug and then of where in the record it is written.
func (c *Corpus) NamedIn(person Slug) []Naming {
	var out []Naming
	collect := func(record EntityRef, events []Event) {
		for i, event := range events {
			for j, named := range event.With {
				if named.Person == person {
					out = append(out, Naming{Record: record, Event: i, Entry: j, Role: named.Role})
				}
			}
		}
	}
	for slug, p := range c.People {
		collect(EntityRef{Kind: KindPerson, Slug: slug}, p.Events)
	}
	for slug, u := range c.Families {
		collect(EntityRef{Kind: KindFamily, Slug: slug}, u.Events)
	}
	for slug, n := range c.Notes {
		collect(EntityRef{Kind: KindNote, Slug: slug}, n.Events)
	}
	slices.SortFunc(out, func(a, b Naming) int {
		return cmp.Or(cmp.Compare(a.Record.Slug, b.Record.Slug), cmp.Compare(a.Event, b.Event),
			cmp.Compare(a.Entry, b.Entry))
	})
	return out
}

// generations walks outwards from person and returns how many steps away each relative
// is, and which relative one step nearer each was reached through. step yields the next
// ring; a visited set keeps a corpus that accidentally makes someone their own ancestor
// from looping forever.
func (c *Corpus) generations(person Slug, step func(Slug) []Slug) (depth map[Slug]int, via map[Slug]Slug) {
	depth, via = map[Slug]int{person: 0}, map[Slug]Slug{}
	frontier := []Slug{person}
	for len(frontier) > 0 {
		var next []Slug
		for _, current := range frontier {
			for _, relative := range step(current) {
				if _, seen := depth[relative]; seen {
					continue
				}
				depth[relative] = depth[current] + 1
				via[relative] = current
				next = append(next, relative)
			}
		}
		slices.Sort(next)
		frontier = next
	}
	delete(depth, person)
	return depth, via
}

// byGeneration groups relatives by how far away they are, the nearest first. Someone
// reached two ways, as a cousin marriage makes an ancestor, stands in the nearer.
func byGeneration(depth map[Slug]int) [][]Slug {
	var out [][]Slug
	for _, slug := range sortedKeys(depth) {
		for len(out) < depth[slug] {
			out = append(out, nil)
		}
		out[depth[slug]-1] = append(out[depth[slug]-1], slug)
	}
	return out
}

// parentSlugs are the parents of person with a record, each once.
func (c *Corpus) parentSlugs(person Slug) []Slug {
	var out []Slug
	for _, parent := range c.ParentsOf(person) {
		if !parent.Slug.Reserved() && !slices.Contains(out, parent.Slug) {
			out = append(out, parent.Slug)
		}
	}
	return out
}

// Ancestors returns everyone reachable upwards through the families, a generation
// to an entry: the parents first, then the grandparents, each in slug order. Adoptive
// and step relations count: an adoptive family is a real one, and the rules that need
// blood ask Relation.Biological for themselves.
func (c *Corpus) Ancestors(person Slug) [][]Slug {
	depth, _ := c.generations(person, c.parentSlugs)
	return byGeneration(depth)
}

// Descendants returns everyone reachable downwards through the families, the
// children first.
func (c *Corpus) Descendants(person Slug) [][]Slug {
	depth, _ := c.generations(person, c.Children)
	return byGeneration(depth)
}

// Kinship is how two people are related: the ancestor they have in common, how many
// generations each stands below them, and the people between. Up 1, Down 1 is a
// sibling; 2 and 2 first cousins; 0 and 2 a grandparent looking down.
//
// Path runs from the first person up to the common ancestor and down to the second,
// each of them once.
type Kinship struct {
	CommonAncestor Slug
	Up, Down       int
	Path           []Slug
}

// Kinship finds the closest common ancestor of two people. ok is false when they have
// none, which in a tree assembled from records is common and not an error. Ties are
// broken by slug so the answer does not depend on map iteration.
func (c *Corpus) Kinship(a, b Slug) (Kinship, bool) {
	up, upVia := c.generations(a, c.parentSlugs)
	up[a] = 0
	down, downVia := c.generations(b, c.parentSlugs)
	down[b] = 0

	best, found := Kinship{}, false
	for _, ancestor := range sortedKeys(up) {
		stepsDown, shared := down[ancestor]
		if !shared {
			continue
		}
		candidate := Kinship{CommonAncestor: ancestor, Up: up[ancestor], Down: stepsDown}
		if !found || candidate.Up+candidate.Down < best.Up+best.Down {
			best, found = candidate, true
		}
	}
	if !found {
		return best, false
	}

	climbed := chain(best.CommonAncestor, a, upVia)
	slices.Reverse(climbed)
	best.Path = append(climbed, chain(best.CommonAncestor, b, downVia)[1:]...)
	return best, true
}

// chain follows a walk back from where it got to until it reaches where it started.
func chain(from, to Slug, via map[Slug]Slug) []Slug {
	out := []Slug{from}
	for from != to {
		from = via[from]
		out = append(out, from)
	}
	return out
}

// Clone copies the corpus deeply enough that edits applied to one cannot be seen in the
// other. Testing a hypothesis means applying its edits to a throwaway.
func (c *Corpus) Clone() *Corpus {
	out := NewCorpus()
	out.Project = c.Project.Clone()
	for slug, person := range c.People {
		copied := *person
		copied.Name = person.Name.Clone()
		copied.AlsoKnownAs = make([]PersonName, 0, len(person.AlsoKnownAs))
		for _, name := range person.AlsoKnownAs {
			copied.AlsoKnownAs = append(copied.AlsoKnownAs, name.Clone())
		}
		if person.AlsoKnownAs == nil {
			copied.AlsoKnownAs = nil
		}
		copied.Sex.Sources = slices.Clone(person.Sex.Sources)
		copied.Events = cloneEvents(person.Events)
		copied.Not = cloneNonEvents(person.Not)
		copied.Portrait.Region = cloneRegion(person.Portrait.Region)
		copied.Media = cloneMedia(person.Media)
		copied.Links = slices.Clone(person.Links)
		copied.Sources = slices.Clone(person.Sources)
		copied.Tags = slices.Clone(person.Tags)
		copied.SameAs = cloneSameAs(person.SameAs)
		copied.IDs = cloneIDs(person.IDs)
		out.People[slug] = &copied
	}
	for slug, family := range c.Families {
		copied := *family
		copied.Partners = slices.Clone(family.Partners)
		for i := range copied.Partners {
			copied.Partners[i].Name = copied.Partners[i].Name.Clone()
		}
		copied.Children = slices.Clone(family.Children)
		for i := range copied.Children {
			child := &copied.Children[i]
			child.Name = child.Name.Clone()
			child.Relation = child.Relation.Clone()
			child.Sources = slices.Clone(child.Sources)
		}
		copied.Events = cloneEvents(family.Events)
		copied.Not = cloneNonEvents(family.Not)
		copied.Media = cloneMedia(family.Media)
		copied.Links = slices.Clone(family.Links)
		copied.Sources = slices.Clone(family.Sources)
		copied.Tags = slices.Clone(family.Tags)
		copied.SameAs = cloneSameAs(family.SameAs)
		copied.IDs = cloneIDs(family.IDs)
		out.Families[slug] = &copied
	}
	for slug, place := range c.Places {
		copied := *place
		copied.Name.Sources = slices.Clone(place.Name.Sources)
		copied.AlsoKnownAs = slices.Clone(place.AlsoKnownAs)
		for i := range copied.AlsoKnownAs {
			copied.AlsoKnownAs[i].Sources = slices.Clone(place.AlsoKnownAs[i].Sources)
		}
		copied.Within = slices.Clone(place.Within)
		for i := range copied.Within {
			copied.Within[i].Sources = slices.Clone(place.Within[i].Sources)
		}
		copied.Addresses = cloneAddresses(place.Addresses)
		copied.Media = cloneMedia(place.Media)
		copied.Links = slices.Clone(place.Links)
		copied.Sources = slices.Clone(place.Sources)
		if place.Coordinates != nil {
			coordinates := *place.Coordinates
			copied.Coordinates = &coordinates
		}
		copied.Tags = slices.Clone(place.Tags)
		copied.SameAs = cloneSameAs(place.SameAs)
		copied.IDs = cloneIDs(place.IDs)
		out.Places[slug] = &copied
	}
	for slug, source := range c.Sources {
		copied := *source
		copied.Covers = slices.Clone(source.Covers)
		for i := range copied.Covers {
			copied.Covers[i].Events = slices.Clone(source.Covers[i].Events)
		}
		copied.Repositories = slices.Clone(source.Repositories)
		copied.Attributes = cloneAttributes(source.Attributes)
		copied.Citations = make(map[string]Citation, len(source.Citations))
		for anchor, citation := range source.Citations {
			citation.With = cloneNamed(citation.With)
			citation.Images = cloneMedia(citation.Images)
			citation.Links = slices.Clone(citation.Links)
			citation.Attributes = cloneAttributes(citation.Attributes)
			citation.Tags = slices.Clone(citation.Tags)
			citation.IDs = cloneIDs(citation.IDs)
			copied.Citations[anchor] = citation
		}
		copied.Media = cloneMedia(source.Media)
		copied.Links = slices.Clone(source.Links)
		copied.Tags = slices.Clone(source.Tags)
		copied.SameAs = cloneSameAs(source.SameAs)
		copied.IDs = cloneIDs(source.IDs)
		out.Sources[slug] = &copied
	}
	for slug, repository := range c.Repositories {
		copied := *repository
		copied.Addresses = cloneAddresses(repository.Addresses)
		copied.Media = cloneMedia(repository.Media)
		copied.Links = slices.Clone(repository.Links)
		copied.Tags = slices.Clone(repository.Tags)
		copied.SameAs = cloneSameAs(repository.SameAs)
		copied.IDs = cloneIDs(repository.IDs)
		out.Repositories[slug] = &copied
	}
	for slug, note := range c.Notes {
		copied := *note
		copied.Events = cloneEvents(note.Events)
		copied.Media = cloneMedia(note.Media)
		copied.Links = slices.Clone(note.Links)
		copied.Sources = slices.Clone(note.Sources)
		copied.Tags = slices.Clone(note.Tags)
		copied.SameAs = cloneSameAs(note.SameAs)
		copied.IDs = cloneIDs(note.IDs)
		out.Notes[slug] = &copied
	}
	for slug, hypothesis := range c.Hypotheses {
		copied := *hypothesis
		copied.Supports = slices.Clone(hypothesis.Supports)
		copied.Contradicts = slices.Clone(hypothesis.Contradicts)
		copied.Edits = slices.Clone(hypothesis.Edits)
		out.Hypotheses[slug] = &copied
	}
	return out
}

func cloneEvents(events []Event) []Event {
	out := slices.Clone(events)
	for i := range out {
		out[i].With = cloneNamed(events[i].With)
		out[i].Address = cloneAddress(events[i].Address)
		out[i].Age.Sources = slices.Clone(events[i].Age.Sources)
		out[i].Cause.Sources = slices.Clone(events[i].Cause.Sources)
		out[i].Attributes = cloneAttributes(events[i].Attributes)
		out[i].Media = cloneMedia(events[i].Media)
		out[i].Links = slices.Clone(events[i].Links)
		out[i].Sources = slices.Clone(events[i].Sources)
		out[i].Tags = slices.Clone(events[i].Tags)
		out[i].IDs = cloneIDs(events[i].IDs)
	}
	return out
}

func cloneSameAs(list []SameAs) []SameAs {
	out := slices.Clone(list)
	for i := range out {
		out[i].Sources = slices.Clone(list[i].Sources)
	}
	return out
}

func cloneNamed(list []Named) []Named {
	out := slices.Clone(list)
	for i := range out {
		out[i].Name = list[i].Name.Clone()
		out[i].Attributes = cloneAttributes(list[i].Attributes)
		out[i].Sources = slices.Clone(list[i].Sources)
	}
	return out
}

func cloneAttributes(list []Attribute) []Attribute {
	out := slices.Clone(list)
	for i := range out {
		out[i].Sources = slices.Clone(list[i].Sources)
	}
	return out
}

func cloneNonEvents(list []NonEvent) []NonEvent {
	out := slices.Clone(list)
	for i := range out {
		out[i].Sources = slices.Clone(list[i].Sources)
	}
	return out
}

func cloneAddress(address Address) Address {
	address.Links = slices.Clone(address.Links)
	address.Sources = slices.Clone(address.Sources)
	return address
}

func cloneAddresses(addresses []Address) []Address {
	out := slices.Clone(addresses)
	for i := range out {
		out[i] = cloneAddress(out[i])
	}
	return out
}

func cloneIDs(ids IDs) IDs {
	if ids == nil {
		return nil
	}
	out := make(IDs, len(ids))
	for service, list := range ids {
		out[service] = slices.Clone(list)
		for i := range list {
			out[service][i].Sources = slices.Clone(list[i].Sources)
		}
	}
	return out
}
