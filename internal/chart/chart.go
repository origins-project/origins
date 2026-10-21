// Package chart lays out the charts of a person. Ancestors, descendants, both and all
// relatives are the same breadth-first walk over the families the index holds, under a
// different reach, laid out in columns by generation, oldest on the right, so that
// every chart is read the same way round. The trees are the walks of ancestors and of
// descendants laid out in rows, oldest on top; the fan is the ancestors in half rings;
// the timeline is all relatives as bars over their lives; and the path is the chain of
// people between two, as model's kinship finds it. The pages run it as WebAssembly and
// the files are drawn from it, so that the two cannot come to draw different charts.
package chart

import (
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Shape is which chart is drawn, as the pages and `export chart` name it.
type Shape string

const (
	Ancestors   Shape = "ancestors"
	Descendants Shape = "descendants"
	Both        Shape = "both"
	Relatives   Shape = "relatives"

	AncestorTree   Shape = "ancestor-tree"
	DescendantTree Shape = "descendant-tree"
	Fan            Shape = "fan"
	Path           Shape = "path"
	Timeline       Shape = "timeline"
)

// Shapes is every chart, in the order the pages offer them.
var Shapes = []Shape{Ancestors, Descendants, Both, Relatives, AncestorTree, DescendantTree, Fan, Path, Timeline}

// Reach is which kinships a chart may follow, and whether one walk may follow more than
// one kind of them: ancestors only go up, descendants only down, both go either way but
// each branch keeps its direction, and all relatives go anywhere.
type Reach struct {
	Up, Down, Partners, Turns bool
}

var reaches = map[Shape]Reach{
	Ancestors:   {Up: true},
	Descendants: {Down: true},
	Both:        {Up: true, Down: true},
	Relatives:   {Up: true, Down: true, Partners: true, Turns: true},

	AncestorTree:   {Up: true},
	DescendantTree: {Down: true},
	Fan:            {Up: true},
	Timeline:       {Up: true, Down: true, Partners: true, Turns: true},
}

// ReachOf is the reach of a shape, and false for a shape there is not or one that is
// not a walk, as the path is not.
func ReachOf(shape Shape) (Reach, bool) {
	r, ok := reaches[shape]
	return r, ok
}

const (
	// DefaultDistance is how many steps a chart walks unless asked for another, and
	// MaxDistance the most it may be asked for.
	DefaultDistance = 5
	MaxDistance     = 12

	// MaxPeople is where a walk stops, more than anyone can read; a chart of all
	// relatives over a large corpus reaches most of it within a few steps.
	MaxPeople = 400
)

// Index is the part of the index a chart is drawn from, read from the same JSON.
type Index struct {
	People   []Person `json:"people"`
	Families []Family `json:"families"`
}

// Person is somebody the index lists, with what a chart shows of them.
type Person struct {
	Ref    string `json:"ref"`
	Sex    string `json:"sex"`
	Living bool   `json:"living"`
	Birth  *Dated `json:"birth"`
	Death  *Dated `json:"death"`
}

// Dated is an event by the day it sorts on, as the index writes it.
type Dated struct {
	Sort string `json:"sort"`
}

// Family is a family the index lists, with who is whose parent in it.
type Family struct {
	Ref      string    `json:"ref"`
	Partners []Partner `json:"partners"`
	Children []Child   `json:"children"`
}

// Partner is one of a family's partners, a slug, `unknown` or `none`.
type Partner struct {
	Person string `json:"person"`
}

// Child is a child a family lists, their relation to each partner and their link's
// status.
type Child struct {
	Person    string     `json:"person"`
	Relations []Relation `json:"relations"`
	Status    string     `json:"status"`
}

// Relation is a child's relation to one partner.
type Relation struct {
	Partner  string `json:"partner"`
	Relation string `json:"relation"`
}

// Tie is how two people are drawn joined: a parent by birth or not, a partnership, or
// a gap to the child it is missing from.
type Tie string

const (
	BirthTie   Tie = "birth"
	OtherTie   Tie = "other"
	PartnerTie Tie = "partner"
	GapTie     Tie = "gap"
)

// Kinship is two people a chart joins, and how.
type Kinship struct {
	A   string `json:"a"`
	B   string `json:"b"`
	Tie Tie    `json:"tie"`
}

// Gap is a parent nobody has a record of, where a chart could show one: a partner
// unknown in a family listing the person as a child, by its place in the family's
// partners, or either parent of a person no family lists.
type Gap struct {
	Child   string `json:"child"`
	Family  string `json:"family,omitempty"`
	Partner *int   `json:"partner,omitempty"`
}

// Slot is a gap as a chart draws it, in the column of the child's parents.
type Slot struct {
	Gap
	Key        string `json:"key"`
	Generation int    `json:"generation"`
}

// Parent is somebody's parent, and whether by birth.
type Parent struct {
	Slug  string
	Birth bool
}

// Kin is who is whose parent, child and partner, read off the families as model reads
// them: a link disproven, a partner none and a relation none give no parent at all,
// and nobody without a record is walked to.
type Kin struct {
	people   map[string]bool
	persons  map[string]Person
	parents  map[string][]Parent
	children map[string][]string
	partners map[string][]string
	listed   map[string]bool
	gaps     map[string][]Gap

	// corpus is the people and families as model holds them, for model's kinship to
	// be asked of.
	corpus *model.Corpus
}

// KinOf reads who is whose kin off the index.
func KinOf(index Index) *Kin {
	k := &Kin{
		people:  map[string]bool{},
		persons: map[string]Person{},
		parents: map[string][]Parent{},
		listed:  map[string]bool{},
		gaps:    map[string][]Gap{},
		corpus:  model.NewCorpus(),
	}
	for _, p := range index.People {
		slug := strings.TrimPrefix(p.Ref, "person/")
		k.people[slug] = true
		k.persons[slug] = p
		sex, _ := model.ParseSex(p.Sex)
		k.corpus.People[model.Slug(slug)] = &model.Person{Slug: model.Slug(slug), Sex: model.StatedSex{Value: sex}}
	}
	for _, f := range index.Families {
		k.corpus.Families[model.Slug(strings.TrimPrefix(f.Ref, "family/"))] = familyOf(f)
	}
	families := slices.Clone(index.Families)
	slices.SortStableFunc(families, func(x, y Family) int { return strings.Compare(x.Ref, y.Ref) })
	children := map[string]map[string]bool{}
	partners := map[string]map[string]bool{}
	add := func(sets map[string]map[string]bool, key, value string) {
		if sets[key] == nil {
			sets[key] = map[string]bool{}
		}
		sets[key][value] = true
	}
	for _, family := range families {
		var known []string
		for _, p := range family.Partners {
			if k.people[p.Person] {
				known = append(known, p.Person)
			}
		}
		for _, one := range known {
			for _, other := range known {
				if one != other {
					add(partners, one, other)
				}
			}
		}
		for _, child := range family.Children {
			if child.Person == "" || child.Status == string(model.Disproven) {
				continue
			}
			k.listed[child.Person] = true
			for i, p := range family.Partners {
				if p.Person != string(model.Unknown) || relationTo(child, string(model.Unknown)) == model.NoRelation {
					continue
				}
				k.gaps[child.Person] = append(k.gaps[child.Person], Gap{Child: child.Person, Family: strings.TrimPrefix(family.Ref, "family/"), Partner: &i})
			}
			for _, partner := range known {
				relation := relationTo(child, partner)
				if relation == model.NoRelation {
					continue
				}
				k.parents[child.Person] = append(k.parents[child.Person], Parent{Slug: partner, Birth: relation == model.Birth})
				if k.people[child.Person] {
					add(children, partner, child.Person)
				}
			}
		}
	}
	sorted := func(sets map[string]map[string]bool) map[string][]string {
		out := make(map[string][]string, len(sets))
		for key, set := range sets {
			out[key] = slices.Sorted(func(yield func(string) bool) {
				for v := range set {
					if !yield(v) {
						return
					}
				}
			})
		}
		return out
	}
	k.children = sorted(children)
	k.partners = sorted(partners)
	return k
}

// familyOf is a family of the index as model holds one, with what kinship reads of it:
// the partners, and each child in their relation to each partner and their link's
// status.
func familyOf(f Family) *model.Family {
	family := &model.Family{Slug: model.Slug(strings.TrimPrefix(f.Ref, "family/"))}
	for _, p := range f.Partners {
		family.Partners = append(family.Partners, model.Partner{Person: model.Slug(p.Person)})
	}
	for _, c := range f.Children {
		child := model.Child{ID: model.Slug(c.Person), Status: model.LinkStatus(c.Status)}
		for _, r := range c.Relations {
			child.Relation.Partners = append(child.Relation.Partners, model.PartnerRelation{Partner: model.Slug(r.Partner), Relation: model.Relation(r.Relation)})
		}
		family.Children = append(family.Children, child)
	}
	return family
}

// relationTo is the child's relation to the partner as kinship reads it, the first the
// family gives, birth where it gives none.
func relationTo(child Child, partner string) model.Relation {
	for _, r := range child.Relations {
		if r.Partner == partner {
			return model.Relation(r.Relation).Kin()
		}
	}
	return model.Birth
}

// People is everybody with a record, in order.
func (k *Kin) People() []string {
	out := make([]string, 0, len(k.people))
	for slug := range k.people {
		out = append(out, slug)
	}
	slices.Sort(out)
	return out
}

// Has says whether somebody has a record.
func (k *Kin) Has(slug string) bool { return k.people[slug] }

// Parents is somebody's parents with a record, by the families in order.
func (k *Kin) Parents(slug string) []Parent { return k.parents[slug] }

// Children is somebody's children with a record, in order.
func (k *Kin) Children(slug string) []string { return k.children[slug] }

// Partners is whoever shares a family with somebody, in order.
func (k *Kin) Partners(slug string) []string { return k.partners[slug] }

// Gaps is the parents somebody is missing: each partner `unknown` of a family listing
// them, or two where no family lists them.
func (k *Kin) Gaps(slug string) []Gap {
	if k.listed[slug] {
		return k.gaps[slug]
	}
	return []Gap{{Child: slug}, {Child: slug}}
}

type direction int

const (
	either direction = iota
	up
	down
)

type neighbour struct {
	slug        string
	direction   direction
	generations int
}

func (k *Kin) neighbours(from string, reach Reach, travelling direction) []neighbour {
	if !k.people[from] {
		return nil
	}
	may := func(want direction) bool { return reach.Turns || travelling == either || travelling == want }
	var out []neighbour
	if reach.Up && may(up) {
		for _, p := range k.Parents(from) {
			out = append(out, neighbour{p.Slug, up, -1})
		}
	}
	if reach.Down && may(down) {
		for _, c := range k.Children(from) {
			out = append(out, neighbour{c, down, 1})
		}
	}
	if reach.Partners {
		for _, p := range k.Partners(from) {
			out = append(out, neighbour{p, travelling, 0})
		}
	}
	return out
}

// Walked is everybody a walk reached, and how they are joined.
type Walked struct {
	Subject string
	Reach   Reach

	// Order is the people in the order the walk found them, which seeds the layout.
	Order []string

	// Generation is relative to the subject: parents -1, children +1.
	Generation map[string]int
	Edges      []Kinship

	// Slots are the parents missing from the people the chart looks up from.
	Slots     []Slot
	Truncated bool
}

// Walk is everybody a chart of the subject draws, no further than distance steps away
// and no more than limit people.
func Walk(k *Kin, subject string, reach Reach, distance, limit int) Walked {
	order := []string{subject}
	generation := map[string]int{subject: 0}
	truncated := false
	type step struct {
		slug      string
		direction direction
		depth     int
	}
	queue := []step{{subject, either, 0}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.depth >= distance {
			continue
		}
		for _, next := range k.neighbours(current.slug, reach, current.direction) {
			if _, seen := generation[next.slug]; seen {
				continue
			}
			if len(order) >= limit {
				truncated = true
				queue = nil
				break
			}
			order = append(order, next.slug)
			generation[next.slug] = generation[current.slug] + next.generations
			queue = append(queue, step{next.slug, next.direction, current.depth + 1})
		}
	}
	return Walked{
		Subject:    subject,
		Reach:      reach,
		Order:      order,
		Generation: generation,
		Edges:      k.kinships(order, generation, reach),
		Slots:      k.slots(order, generation, reach, distance),
		Truncated:  truncated,
	}
}

// slots are the parents missing from everybody a chart looking up reaches within the
// distance, all relatives aside, whose parents are a turn the walk takes rather than a
// direction.
func (k *Kin) slots(order []string, generation map[string]int, reach Reach, distance int) []Slot {
	if !reach.Up || reach.Turns {
		return nil
	}
	var out []Slot
	for _, slug := range order {
		gen := generation[slug]
		if gen > 0 || -gen >= distance || !k.people[slug] {
			continue
		}
		for i, gap := range k.Gaps(slug) {
			out = append(out, Slot{Gap: gap, Key: slug + " " + strconv.Itoa(i), Generation: gen - 1})
		}
	}
	return out
}

// kinships are every tie between two people the walk kept, rather than only those it
// travelled, so that a marriage of cousins shows as two lines meeting rather than two
// branches.
func (k *Kin) kinships(order []string, member map[string]int, reach Reach) []Kinship {
	var out []Kinship
	seen := map[[2]string]bool{}
	add := func(a, b string, tie Tie) {
		key := [2]string{a, b}
		if b < a {
			key = [2]string{b, a}
		}
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, Kinship{a, b, tie})
	}
	for _, slug := range order {
		if !k.people[slug] {
			continue
		}
		if reach.Up || reach.Down {
			for _, p := range k.Parents(slug) {
				if _, ok := member[p.Slug]; ok {
					tie := OtherTie
					if p.Birth {
						tie = BirthTie
					}
					add(p.Slug, slug, tie)
				}
			}
		}
		if reach.Partners {
			for _, p := range k.Partners(slug) {
				if _, ok := member[p]; ok {
					add(slug, p, PartnerTie)
				}
			}
		}
	}
	return out
}
