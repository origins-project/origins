package api

import (
	"maps"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Index is the corpus as a list or a chart needs it: every record, what it is called and
// what ties it to the others, and none of the prose. A page asks for it once and finds
// its way from there, asking for a record in full only when it shows one.
type Index struct {
	Project      IndexedProject      `json:"project"`
	People       []IndexedPerson     `json:"people"`
	Families     []IndexedFamily     `json:"families"`
	Places       []IndexedPlace      `json:"places"`
	Sources      []IndexedSource     `json:"sources"`
	Repositories []IndexedRepository `json:"repositories"`
	Notes        []IndexedNote       `json:"notes"`
	Hypotheses   []IndexedHypothesis `json:"hypotheses"`
	Read         []Supposition       `json:"read,omitempty"`
	Problems     []string            `json:"problems"`
}

// Counts is how many records of each kind the index lists.
func (i *Index) Counts() Counts {
	return Counts{
		People: len(i.People), Families: len(i.Families), Places: len(i.Places),
		Sources: len(i.Sources), Repositories: len(i.Repositories), Notes: len(i.Notes),
		Hypotheses: len(i.Hypotheses),
	}
}

// IndexedProject is what the manifest says that every page shows.
type IndexedProject struct {
	Name       string              `json:"name,omitempty"`
	MainPerson string              `json:"main-person,omitempty"`
	Bookmarks  []string            `json:"bookmarks,omitempty"`
	Tags       map[string]TagShown `json:"tags,omitempty"`
}

// TagShown is how a tag is shown, as the manifest gives it.
type TagShown struct {
	Color    string `json:"color,omitempty"`
	Priority string `json:"priority,omitempty"`
}

// Indexed is what every record is listed with: how to ask for it, what it is called and
// what else, whether it is private, and the researcher's words for it.
type Indexed struct {
	Ref         string   `json:"ref"`
	Name        string   `json:"name,omitempty"`
	AlsoKnownAs []string `json:"also-known-as,omitempty"`
	Private     bool     `json:"private,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// Dated is an event as a list shows it: what it was, its date as written, the key it
// sorts by and where it happened. Sort is left out for a date with no place on the line.
type Dated struct {
	Type  string `json:"type"`
	Date  string `json:"date,omitempty"`
	Sort  string `json:"sort,omitempty"`
	Place string `json:"place,omitempty"`
}

// IndexedPerson is a person as a list and a chart show them.
type IndexedPerson struct {
	Indexed
	Surname string `json:"surname,omitempty"`
	Given   string `json:"given,omitempty"`
	Sex     string `json:"sex,omitempty"`
	Living  bool   `json:"living"`

	// Birth and Death are the events that say when a life began and ended, or, where
	// there is none, the baptism or christening and the burial or cremation, their
	// type saying which.
	Birth *Dated `json:"birth,omitempty"`
	Death *Dated `json:"death,omitempty"`

	// Portrait is the picture the person is shown by, chosen by model.PortraitOf.
	Portrait *IndexedPortrait `json:"portrait,omitempty"`

	PartnerIn []string `json:"partner-in,omitempty"`
	ChildIn   []string `json:"child-in,omitempty"`
}

// IndexedPortrait is a file beside a record, and the part of it that is the person
// where that is marked, as fractions left, top, right and bottom.
type IndexedPortrait struct {
	Record string    `json:"record"`
	File   string    `json:"file"`
	Region []float64 `json:"region,omitempty"`
}

// IndexedFamily is a family as a list shows it, with its partners and children.
type IndexedFamily struct {
	Indexed
	Type     string           `json:"type,omitempty"`
	Partners []IndexedPartner `json:"partners"`

	// First is the family's first dated event, in the order the record writes them.
	First    *Dated         `json:"first,omitempty"`
	Children []IndexedChild `json:"children,omitempty"`
}

// IndexedPartner is a partner by their record, unknown or none, or by their name alone.
type IndexedPartner struct {
	Person string `json:"person,omitempty"`
	Name   string `json:"name,omitempty"`
}

// IndexedChild is a child the family lists, in their relation to each partner and with
// the link's status, a disproven one included so that the page can show it struck out.
type IndexedChild struct {
	Person    string            `json:"person,omitempty"`
	Name      string            `json:"name,omitempty"`
	Relations []IndexedRelation `json:"relations"`
	Status    string            `json:"status,omitempty"`
}

// IndexedRelation is a child's relation to one partner, who is named as the family's
// partners are counted: their slug, or unknown or none.
type IndexedRelation struct {
	Partner  string `json:"partner"`
	Relation string `json:"relation"`
}

// IndexedPlace is a place as a list and the tree of places show it.
type IndexedPlace struct {
	Indexed
	Type        string          `json:"type,omitempty"`
	Within      []IndexedWithin `json:"within,omitempty"`
	Coordinates *IndexedCoords  `json:"coordinates,omitempty"`

	// References is how many references from other records reach the place.
	References int `json:"references"`
}

// IndexedWithin is a place another lay within, and when.
type IndexedWithin struct {
	Place string `json:"place"`
	From  string `json:"from,omitempty"`
	Until string `json:"until,omitempty"`
}

// IndexedCoords are two angles, or the words a record gave in place of them.
type IndexedCoords struct {
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	Wording   string   `json:"wording,omitempty"`
}

// IndexedSource is a source as a list shows it, with the anchors of its citations.
type IndexedSource struct {
	Indexed
	ShortTitle string        `json:"short-title,omitempty"`
	Type       string        `json:"type,omitempty"`
	Author     string        `json:"author,omitempty"`
	Kept       []IndexedKept `json:"kept,omitempty"`

	// Citations is how many citations of it the corpus makes, and Anchors the entries
	// it transcribes, each cited as the source and the anchor.
	Citations int      `json:"citations"`
	Anchors   []string `json:"anchors,omitempty"`
}

// IndexedKept is where a source is kept: a repository and its call number there.
type IndexedKept struct {
	Repository string `json:"repository"`
	CallNumber string `json:"call-number,omitempty"`
	Medium     string `json:"medium,omitempty"`
}

// IndexedRepository is a repository as a list shows it.
type IndexedRepository struct {
	Indexed
	Type    string   `json:"type,omitempty"`
	Sources []string `json:"sources,omitempty"`
}

// IndexedNote is a note as a list shows it, with the events it holds.
type IndexedNote struct {
	Indexed
	Events []IndexedNoteEvent `json:"events,omitempty"`
}

// IndexedNoteEvent is an event a note holds, and whoever it names.
type IndexedNoteEvent struct {
	Dated
	With []IndexedNamed `json:"with,omitempty"`
}

// IndexedNamed is somebody an event names, by their record or by their name.
type IndexedNamed struct {
	Person string `json:"person,omitempty"`
	Name   string `json:"name,omitempty"`
	Role   string `json:"role,omitempty"`
}

// IndexedHypothesis is a hypothesis as a list shows it.
type IndexedHypothesis struct {
	Indexed
	Status     string `json:"status,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Base       string `json:"base,omitempty"`
	ProposedBy string `json:"proposed-by,omitempty"`
}

// IndexOf lists every record of the corpus, as it reads with each hypothesis in with.
func IndexOf(root string, with []model.Slug) (*Index, error) {
	q, err := asking(root, with)
	if err != nil {
		return nil, err
	}
	defer q.close()
	index := indexOf(q.corpus.Corpus, today())
	index.Read, index.Problems = q.read, q.problems
	return index, q.warning
}

// IndexOfCorpus is the index of a corpus already read, deciding who is living as of
// today, for a server that holds a corpus and indexes whichever view of it is asked for.
func IndexOfCorpus(c *model.Corpus, today model.Date) *Index {
	return indexOf(c, today)
}

// birthOrFallback and deathOrFallback are the events that say when a life began and
// ended, best first: where a person has no birth, the baptism or christening is the
// nearest thing the records hold, as the burial or cremation is to a death.
var (
	birthOrFallback = [][]string{{"birth"}, {"baptism", "christening"}}
	deathOrFallback = [][]string{{"death"}, {"burial", "cremation"}}
)

// indexOf is the index of a corpus as it stands, deciding who is living as of today,
// which is what a page serving a hidden view hands it.
func indexOf(c *model.Corpus, today model.Date) *Index {
	index := &Index{
		Project:      indexedProject(c.Project),
		People:       []IndexedPerson{},
		Families:     []IndexedFamily{},
		Places:       []IndexedPlace{},
		Sources:      []IndexedSource{},
		Repositories: []IndexedRepository{},
		Notes:        []IndexedNote{},
		Hypotheses:   []IndexedHypothesis{},
		Problems:     []string{},
	}

	references, citations := map[model.Slug]int{}, map[model.Slug]int{}
	for _, slug := range c.AllSlugs() {
		entity, times := c.Entity(slug), map[model.Slug]int{}
		model.References(entity, func(s *model.Slug) {
			if *s != slug {
				times[*s]++
			}
		})
		if same := model.SameAsOf(entity); same != nil {
			for _, claim := range *same {
				times[claim.ID]--
			}
		}
		for s, n := range times {
			references[s] += max(n, 0)
		}
		model.Cited(entity, func(ref *model.CitationRef) { citations[ref.Source]++ })
	}
	kept := map[model.Slug][]string{}
	for _, slug := range c.SourceSlugs() {
		for _, held := range c.Sources[slug].Repositories {
			if !slices.Contains(kept[held.Repository], string(slug)) {
				kept[held.Repository] = append(kept[held.Repository], string(slug))
			}
		}
	}

	portraits, ties, lives := c.Portraits(), c.Ties(), c.Lives(today)
	for _, slug := range c.PeopleSlugs() {
		p := c.People[slug]
		one := IndexedPerson{
			Indexed: indexed(p, p.Private, p.Tags),
			Given:   p.Name.Given,
			Living:  lives.Living(p),
			Birth:   lifeEvent(p.Events, birthOrFallback),
			Death:   lifeEvent(p.Events, deathOrFallback),
		}
		one.Name = p.Name.String()
		for _, other := range p.AlsoKnownAs {
			one.AlsoKnownAs = append(one.AlsoKnownAs, other.String())
		}
		one.Surname = p.Name.Group
		if one.Surname == "" {
			one.Surname = p.Name.PrimarySurname().Name
		}
		if p.Sex.Value != model.SexUnknown {
			one.Sex = p.Sex.Value.String()
		}
		one.PartnerIn = slugStrings(ties.PartnerIn[slug])
		one.ChildIn = slugStrings(ties.ChildIn[slug])
		if portrait, ok := portraits[slug]; ok {
			one.Portrait = &IndexedPortrait{Record: portrait.Record.String(), File: portrait.File}
			if r := portrait.Region; r != nil {
				one.Portrait.Region = []float64{r.Left, r.Top, r.Right, r.Bottom}
			}
		}
		index.People = append(index.People, one)
	}

	for _, slug := range c.FamilySlugs() {
		f := c.Families[slug]
		one := IndexedFamily{Indexed: indexed(f, f.Private, f.Tags), Type: f.Type, Partners: []IndexedPartner{}}
		for _, partner := range f.Partners {
			one.Partners = append(one.Partners, IndexedPartner{Person: string(partner.Person), Name: partner.Name.String()})
		}
		for _, event := range f.Events {
			if event.Date.Anchored() {
				one.First = dated(event)
				break
			}
		}
		for _, child := range f.Children {
			listed := IndexedChild{
				Person: string(child.ID), Name: child.Name.String(),
				Relations: []IndexedRelation{}, Status: string(child.Status),
			}
			for _, partner := range f.Partners {
				listed.Relations = append(listed.Relations, IndexedRelation{
					Partner: string(partner.Who()), Relation: string(child.To(partner.Who())),
				})
			}
			one.Children = append(one.Children, listed)
		}
		index.Families = append(index.Families, one)
	}

	for _, slug := range c.PlaceSlugs() {
		p := c.Places[slug]
		one := IndexedPlace{Indexed: indexed(p, p.Private, p.Tags), Type: p.Type, References: references[slug]}
		one.Name = p.Name.Text
		for _, other := range p.AlsoKnownAs {
			one.AlsoKnownAs = append(one.AlsoKnownAs, other.Text)
		}
		for _, within := range p.Within {
			one.Within = append(one.Within, IndexedWithin{
				Place: string(within.Place), From: dateString(within.From), Until: dateString(within.Until),
			})
		}
		if at := p.Coordinates; at != nil {
			if at.Placed() {
				one.Coordinates = &IndexedCoords{Latitude: &at.Latitude, Longitude: &at.Longitude}
			} else {
				one.Coordinates = &IndexedCoords{Wording: at.Wording}
			}
		}
		index.Places = append(index.Places, one)
	}

	for _, slug := range c.SourceSlugs() {
		s := c.Sources[slug]
		one := IndexedSource{
			Indexed:    indexed(s, s.Private, s.Tags),
			ShortTitle: s.ShortTitle, Type: s.Type, Author: s.Author,
			Citations: citations[slug],
		}
		one.Name = s.Title
		one.Anchors = slices.Sorted(maps.Keys(s.Citations))
		for _, held := range s.Repositories {
			one.Kept = append(one.Kept, IndexedKept{
				Repository: string(held.Repository), CallNumber: held.CallNumber, Medium: held.Medium,
			})
		}
		index.Sources = append(index.Sources, one)
	}

	for _, slug := range c.RepositorySlugs() {
		r := c.Repositories[slug]
		one := IndexedRepository{Indexed: indexed(r, r.Private, r.Tags), Type: r.Type, Sources: kept[slug]}
		one.Name = r.Name
		index.Repositories = append(index.Repositories, one)
	}

	for _, slug := range c.NoteSlugs() {
		n := c.Notes[slug]
		one := IndexedNote{Indexed: indexed(n, n.Private, n.Tags)}
		one.Name = n.Title
		for _, event := range n.Events {
			listed := IndexedNoteEvent{Dated: *dated(event)}
			for _, named := range event.With {
				listed.With = append(listed.With, IndexedNamed{
					Person: string(named.Person), Name: named.Name.String(), Role: named.Role,
				})
			}
			one.Events = append(one.Events, listed)
		}
		index.Notes = append(index.Notes, one)
	}

	for _, slug := range c.HypothesisSlugs() {
		h := c.Hypotheses[slug]
		one := IndexedHypothesis{
			Indexed: indexed(h, false, nil),
			Status:  string(h.Status), Confidence: string(h.Confidence), Base: h.Base,
			ProposedBy: h.ProposedBy,
		}
		one.Name = question(h.Body)
		index.Hypotheses = append(index.Hypotheses, one)
	}
	return index
}

func indexedProject(p model.Project) IndexedProject {
	one := IndexedProject{Name: p.Name, MainPerson: string(p.MainPerson)}
	for _, bookmark := range p.Bookmarks {
		one.Bookmarks = append(one.Bookmarks, bookmark.String())
	}
	for word, style := range p.Tags {
		if one.Tags == nil {
			one.Tags = map[string]TagShown{}
		}
		one.Tags[word] = TagShown{Color: style.Color, Priority: style.Priority}
	}
	return one
}

func indexed(entity any, private bool, tags []string) Indexed {
	return Indexed{Ref: model.RefOf(entity).String(), Private: private, Tags: tags}
}

// lifeEvent is the first of a person's events of the first of the kinds that has one.
func lifeEvent(events []model.Event, kinds [][]string) *Dated {
	for _, words := range kinds {
		for _, event := range events {
			if slices.ContainsFunc(words, func(word string) bool { return model.SameWord(word, event.Type) }) {
				return dated(event)
			}
		}
	}
	return nil
}

func dated(event model.Event) *Dated {
	one := &Dated{Type: event.Type, Date: dateString(event.Date), Place: string(event.Place)}
	one.Sort, _ = event.Placing().SortKey()
	return one
}

func dateString(d model.Date) string {
	if d.IsZero() {
		return ""
	}
	return d.String()
}

// question is what a hypothesis asks: the first heading of its body.
func question(body string) string {
	for line := range strings.Lines(body) {
		if heading, ok := strings.CutPrefix(strings.TrimSpace(line), "# "); ok {
			return strings.TrimSpace(heading)
		}
	}
	return ""
}

func slugStrings(slugs []model.Slug) []string {
	var out []string
	for _, slug := range slugs {
		out = append(out, string(slug))
	}
	return out
}
