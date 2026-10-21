package gramps

import (
	"maps"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// Convert maps a Gramps export onto the model. Whatever in the export it does not read
// is kept as written, in a fenced block in the note or the body it belongs in, and the
// report counts it.
//
// Gramps keeps every person, event and place as an object of its own and has the others
// point at it by handle, where a corpus keeps an event inside the record it is about and
// a place as a record that others name by slug. So the objects are indexed by handle
// first, everybody and every place is given a slug before anything is built, and an
// event shared by several people is copied into each record it belongs to.
//
// A picture comes in where find says the file it names is to hand, as an attachment for
// the caller to copy beside the record keeping it; a nil find finds nothing.
//
// Order is file order throughout. Nothing here iterates a map to produce output, so the
// same file converts to the same corpus every time.
func Convert(export *Export, choose importing.Chooser, find importing.Finder) (*model.Corpus, []importing.Attachment, []string) {
	if find == nil {
		find = func(string) (string, bool) { return "", false }
	}
	corpus := model.NewCorpus()
	corpus.Project.FormatVersion = model.FormatVersion

	c := &converter{
		corpus:     corpus,
		namer:      importing.NewNamer(corpus, choose),
		notes:      importing.NewNotes(),
		root:       export.Root,
		handles:    map[string]*Element{},
		person:     map[string]model.Slug{},
		place:      map[string]model.Slug{},
		family:     map[string]model.Slug{},
		holders:    map[string][]holder{},
		familyHeld: map[string]bool{},
		events:     map[string]model.Event{},
		citation:   map[string]model.CitationRef{},
		source:     map[string]model.Slug{},
		repository: map[string]model.Slug{},
		note:       map[string]model.Slug{},
		noteAt:     map[string][]string{},
		eventAt:    map[string]model.Slug{},
		find:       find,
		files:      importing.NewAttacher(),
		kept:       map[string]*kept{},
		keeper:     importing.NewKeeper[*Element]("gramps"),
	}
	if export.Newer {
		c.notes.Add("this file is Gramps XML %s, newer than the %s origins knows, "+
			"and was read as though it were that", export.Version, newest)
	}

	// A picture is kept by the first record to name it, so the order records are read
	// in is the order they claim their pictures in: a scan by its source, then a place's
	// pictures, a family's, and a person's last, so that a photograph of a family is
	// the family's and everybody else naming it is somebody in it.
	c.index()
	Describe(export, "")
	c.nameEveryone()
	c.namePlaces()
	c.nameNotes()
	c.readRepositories()
	c.readSources()
	c.readCitations()
	c.readAddresses()
	c.readPlaces()
	c.readEvents()
	c.readFamilies()
	c.readPeople()
	c.unheldPictures()
	c.unreferred()
	c.readNotes()
	c.readManifest()
	c.keeper.Keep(corpus, c.namer, c.notes, c.roots(), title)
	return corpus, c.files.List(), c.notes.List()
}

type converter struct {
	corpus *model.Corpus
	namer  *importing.Namer
	notes  *importing.Notes
	root   *Element

	handles map[string]*Element    // every object, by its handle
	person  map[string]model.Slug  // a person's handle to their slug, assigned before anything is built
	place   map[string]model.Slug  // a place's handle to its slug
	family  map[string]model.Slug  // a family's handle to its slug, once it is read
	holders map[string][]holder    // an event's handle to the people pointing at it, in file order
	events  map[string]model.Event // an event's handle to what every copy of it shares

	// familyHeld are the events a family points at, which are the family's and not
	// copied into its partners' records.
	familyHeld map[string]bool

	citation   map[string]model.CitationRef // a citation's handle to the anchor it became
	source     map[string]model.Slug        // a source's handle to its slug
	repository map[string]model.Slug        // a repository's handle to its slug
	note       map[string]model.Slug        // a note's handle to its slug
	noteAt     map[string][]string          // a note's handle to the elements it is attached to
	eventAt    map[string]model.Slug        // an event's handle to the first record holding it

	find  importing.Finder
	files *importing.Attacher
	kept  map[string]*kept // a media object's handle to the record keeping it

	keeper *importing.Keeper[*Element] // where what is not read under an element is to go
}

// holder is a person pointing at an event, and the reference saying in what role.
type holder struct {
	person string
	ref    *Element
}

// sections are the lists a Gramps database holds its objects in, and the element each
// object is.
var sections = map[string]string{
	"people": "person", "families": "family", "events": "event", "places": "placeobj",
	"citations": "citation", "sources": "source", "objects": "object",
	"repositories": "repository", "notes": "note", "tags": "tag",
}

func (c *converter) index() {
	c.root.Use("xmlns")
	for _, section := range c.root.Children {
		object, ok := sections[section.Name]
		if !ok {
			continue
		}
		section.Mark()
		for _, e := range section.All(object) {
			if handle := e.Attr("handle"); handle != "" {
				c.handles[handle] = e
			}
		}
	}
	for _, person := range c.all("people", "person") {
		for _, ref := range person.All("eventref") {
			hlink := ref.Attr("hlink")
			c.holders[hlink] = append(c.holders[hlink], holder{person.Attr("handle"), ref})
		}
	}
	for _, family := range c.all("families", "family") {
		for _, ref := range family.All("eventref") {
			c.familyHeld[ref.Attr("hlink")] = true
		}
	}
	c.dropped()
}

// dropped marks what is left out knowingly: the handles Gramps names its objects by
// and refers to them with, which mean nothing outside the database, when each object
// was last changed, which stops being true at the first change and which the history
// of the corpus replaces, and the event other programs write for the same.
func (c *converter) dropped() {
	updates := map[string]bool{}
	for _, event := range c.all("events", "event") {
		if collapse(event.First("type").Text()) == updated {
			event.markAll()
			updates[event.Attr("handle")] = true
		}
	}
	c.root.walk(func(e *Element) {
		e.Use("handle")
		e.Use("change")
		e.Use("hlink")
		if e.Name == "eventref" && updates[e.Attr("hlink")] {
			e.markAll()
		}
	})
}

// roots are what the export holds, in the order written, as the keeper walks it: each
// object, the attributes of the list holding it, and everything that is not a list of
// objects as a whole.
func (c *converter) roots() []*Element {
	var roots []*Element
	for _, section := range c.root.Children {
		if _, ok := sections[section.Name]; !ok {
			roots = append(roots, section)
			continue
		}
		roots = append(roots, section.bare())
		roots = append(roots, section.Children...)
	}
	return roots
}

// title is what a note record keeping a part of the export nothing else read is called.
func title(e *Element) string {
	switch {
	case e.Name == "header":
		return "The header of the export"
	case e.Attr("id") != "":
		return e.Name + " " + e.Attr("id")
	case e.Attr("name") != "":
		return e.Name + " " + e.Attr("name")
	}
	return e.Name
}

// all is every object of one kind, in file order.
func (c *converter) all(section, object string) []*Element {
	return c.root.First(section).All(object)
}

// object is the object a reference's hlink points at, if it is one of the kind given.
func (c *converter) object(ref *Element, kind string) *Element {
	if e := c.handles[ref.Attr("hlink")]; e != nil && e.Name == kind {
		return e
	}
	return nil
}

func (c *converter) date(e *Element) model.Date {
	date, notes := dateOf(e)
	for _, note := range notes {
		c.notes.Add("%s", note)
	}
	return date
}

// private reports whether Gramps marks an element private, for a caller that makes what
// it reads it into private where it is.
func private(e *Element) bool { return e.Use("priv") == "1" }

// typeWord is a type or a role Gramps names in words: one of Gramps' own in the words
// the corpus writes it in, and one somebody typed as they typed it.
func typeWord(own map[string]string, words string) string {
	words = collapse(words)
	if word, ok := own[words]; ok {
		return word
	}
	return words
}

// vocabulary is the strings Gramps writes for the types of one list of its own, each in
// the corpus's words: its slug, or the word renamed gives it. Gramps writes a type of its
// own in English whatever language it is run in, and anything else is somebody's.
func vocabulary(renamed map[string]string, words ...string) map[string]string {
	own := map[string]string{}
	maps.Copy(own, renamed)
	for _, word := range words {
		if _, ok := own[word]; !ok {
			slug, _ := model.Slugify(word)
			own[word] = string(slug)
		}
	}
	return own
}

// grampsID is the identifier Gramps shows its user for a record, which is what anybody
// matching the corpus against the tree again would look for.
func (c *converter) grampsID(e *Element) model.IDs {
	ids, _ := identify(nil, "gramps", e.Use("id"))
	return ids
}

// identify adds an identifier, reporting whether it could: a record giving a second at
// the same service keeps the first, and the second is left unread.
func identify(ids model.IDs, service, id string) (model.IDs, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return ids, true
	}
	if ids == nil {
		ids = model.IDs{}
	}
	switch existing, taken := ids[service]; {
	case !taken:
		ids[service] = model.IdentifiersOf(id)
	case existing[0].ID != id:
		return ids, false
	}
	return ids, true
}

// tags are the words of the tags an object is marked with, read through each tag's
// name. What else a tag says is the manifest's.
func (c *converter) tags(e *Element) []string {
	var words []string
	for _, ref := range e.All("tagref") {
		tag := c.object(ref, "tag")
		if tag == nil {
			c.notes.Add("a tag was referred to that is not in the file")
			continue
		}
		if word := collapse(tag.Attr("name")); word != "" && !slices.Contains(words, word) {
			ref.Mark()
			words = append(words, word)
		}
	}
	return words
}

// home names the prose at path on record as where what is not read under e goes.
func (c *converter) home(e *Element, record model.Slug, path string) {
	if e != nil {
		c.keeper.Home(e, record, path)
	}
}
