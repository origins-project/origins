package gramps

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// factTypes are Gramps' own attributes, in the GEDCOM import's words for them where
// those are not Gramps' own words as a slug.
var factTypes = vocabulary(map[string]string{
	"Identification Number": "identifying-number",
	"National Origin":       "nationality",
	"Father Age":            "fathers-age",
	"Mother Age":            "mothers-age",
	"Unknown":               "fact",
},
	"Caste", "Description", "Number of Children", "Social Security Number", "Nickname",
	"Cause", "Agency", "Age", "Witness", "Time", "Occupation",
)

// idTypes are the attributes that are a record's identifier elsewhere, which Gramps'
// own GEDCOM import leaves as attributes, and the services they are at: the programs that
// wrote the file before Gramps kept their own, and a merge in Gramps keeps the
// identifier the record merged away had.
var idTypes = map[string]string{
	"REFN": "refn", "_FSFTID": "familysearch", "_UID": "uid", "RIN": "rin",
	"Merged Gramps ID": "gramps",
}

// merged is the attribute naming an identifier a record had before a merge, which goes
// after the one it has now.
const merged = "Merged Gramps ID"

// rites are the rites of the Latter-day Saints, as Gramps writes them, in the words the
// GEDCOM import gives the same.
var rites = map[string]string{
	"baptism": "lds-baptism", "confirmation": "lds-confirmation", "initiatory": "lds-initiatory",
	"endowment": "lds-endowment", "sealed_to_parents": "lds-sealing-to-parents",
	"sealed_to_spouse": "lds-sealing-to-spouse",
}

// riteStatuses are the states of a rite Gramps writes in other words than a slug of
// them, in GEDCOM's words for the same.
var riteStatuses = map[string]string{
	"BIC": "born-in-covenant", "DNS": "do-not-seal", "DNS/CAN": "do-not-seal-canceled",
}

// urlTypes are Gramps' own types of a URL. Unknown says nothing.
var urlTypes = vocabulary(map[string]string{"Unknown": ""}, "E-mail", "Web Home", "Web Search", "FTP")

// nameEveryone gives every person their slug before any record is built, because an
// event names people by slug and may name them before their own record comes.
func (c *converter) nameEveryone() {
	for _, person := range c.all("people", "person") {
		name := c.readName(person.First("name"))
		base, err := name.Slug()
		if err != nil {
			if base, err = model.Slugify(person.Attr("id")); err != nil {
				c.notes.Add("a person had no name and no usable identifier, and was not imported")
				continue
			}
			if name.IsZero() {
				c.notes.Add("a person had no name and is filed under their identifier")
			} else {
				c.notes.Add("a person had no name a slug can spell and is filed under their identifier")
			}
		}
		what := name.String()
		if name.IsZero() {
			what = "the person " + person.Attr("id")
		}
		slug := c.namer.Assign(base, c.discriminators(person), what)
		if slug == "" {
			c.notes.Add("no free slug could be found for %q", what)
			continue
		}
		c.person[person.Attr("handle")] = slug
	}
}

// discriminators are the facts worth putting in a slug to tell two people of the same
// name apart, best first: when they were born, where, and when they died.
func (c *converter) discriminators(person *Element) []string {
	var hints []string
	year := func(event *Element) {
		if date, _ := dateOf(event); date.Year != nil {
			hints = append(hints, itoa(*date.Year))
		}
	}
	birth, death := c.primary(person, "Birth"), c.primary(person, "Death")
	year(birth)
	if place := c.object(birth.First("place"), "placeobj"); place != nil {
		if name := placeName(place); name != "" {
			hints = append(hints, name)
		}
	}
	year(death)
	return hints
}

// primary is the first event of a type a person holds as its Primary, or nil.
func (c *converter) primary(person *Element, kind string) *Element {
	for _, ref := range person.All("eventref") {
		event := c.object(ref, "event")
		if event != nil && role(ref) == "Primary" && collapse(event.First("type").Text()) == kind {
			return event
		}
	}
	return nil
}

func (c *converter) readPeople() {
	for _, e := range c.all("people", "person") {
		slug, ok := c.person[e.Attr("handle")]
		if !ok {
			continue
		}
		e.Mark()
		c.home(e, slug, "body")
		c.corpus.People[slug] = c.readPerson(e, slug)
	}
}

func (c *converter) readPerson(e *Element, slug model.Slug) *model.Person {
	person := &model.Person{Slug: slug, Private: private(e), IDs: c.grampsID(e), Sources: c.citations(e)}
	for i, element := range e.TakeAll("name") {
		if i == 0 {
			name := c.name(element, slug, "name.note")
			// The name a person is filed under cannot be hidden on its own, so a
			// private one hides the person.
			person.Private = person.Private || name.Private
			name.Private = false
			person.Name = name
			continue
		}
		at := fmt.Sprintf("also-known-as[%d].note", len(person.AlsoKnownAs))
		person.AlsoKnownAs = append(person.AlsoKnownAs, c.name(element, slug, at))
	}
	if person.Name.IsZero() {
		person.Name = model.PersonName{Text: string(slug)}
	}
	if len(person.Name.Surname) == 0 {
		c.notes.Add("a name gave no surname, so the person is filed under the whole of it")
	}

	gender := e.First("gender")
	switch collapse(gender.Text()) {
	case "M":
		person.Sex.Value = model.Male
	case "F":
		person.Sex.Value = model.Female
	case "X":
		person.Sex.Value = model.Neither
	case "U":
	default:
		gender = nil
	}
	gender.Mark()
	// whose child somebody is, and whose partner, the families say
	e.TakeAll("childof")
	e.TakeAll("parentin")

	person.Body = c.body(e)
	c.pictures(e, person, &person.Media, -1)

	for _, ref := range e.All("eventref") {
		event := ref.Attr("hlink")
		if _, read := c.events[event]; !read {
			if c.object(ref, "event") == nil {
				c.notes.Add("an event was referred to that is not in the file")
			}
			continue
		}
		if c.copies(role(ref), event) {
			at := fmt.Sprintf("events[%d].note", len(person.Events))
			person.Events = append(person.Events, c.copyOf(event, ref, slug, at))
			last := len(person.Events) - 1
			c.eventPictures(event, person, &person.Events[last], last)
		}
	}
	for _, rite := range e.All("lds_ord") {
		at := fmt.Sprintf("events[%d].note", len(person.Events))
		person.Events = append(person.Events, c.rite(rite, slug, at))
	}
	for _, attribute := range e.All("attribute") {
		c.attribute(person, attribute)
	}
	for _, address := range e.TakeAll("address") {
		at := fmt.Sprintf("events[%d].note", len(person.Events))
		person.Events = append(person.Events, c.address(address, slug, at))
	}
	for _, ref := range e.All("personref") {
		at := fmt.Sprintf("events[%d].note", len(person.Events))
		if association, ok := c.association(ref, slug, at); ok {
			person.Events = append(person.Events, association)
		}
	}
	person.Links = c.links(e, &person.Private)
	person.Tags = c.tags(e)
	return person
}

// attribute reads an attribute of a person. Most are facts held, as GEDCOM's are; an
// identifier is one of the person's identifiers, and a nickname the name's nickname
// where the name has none and nothing is said of it.
func (c *converter) attribute(person *model.Person, e *Element) {
	if c.identifier(&person.IDs, &person.Private, person.Slug, e) {
		return
	}
	if e.Attr("type") == "Nickname" && person.Name.Nickname == "" && person.Name.HasParts() &&
		e.First("noteref") == nil && e.First("citationref") == nil && e.Attr("priv") != "1" {
		person.Name.Nickname = collapse(e.Attr("value"))
		c.used(e)
		c.home(e, person.Slug, "name.note")
		return
	}
	at := fmt.Sprintf("events[%d].note", len(person.Events))
	person.Events = append(person.Events, c.fact(e, person.Slug, at))
}

// identifier reads an attribute that is a record's identifier elsewhere into its ids,
// with its notes and its citations, reporting whether it was one. An identifier a merge
// left goes after those the record has at Gramps; a second identifier at any other
// service, and one already there, are left unread. A private one makes the record
// private, which flag is.
func (c *converter) identifier(ids *model.IDs, flag *bool, record model.Slug, e *Element) bool {
	service := idTypes[e.Attr("type")]
	if service == "" {
		return false
	}
	id := collapse(e.Attr("value"))
	list := (*ids)[service]
	if id == "" || len(list) > 0 && e.Attr("type") != merged ||
		slices.ContainsFunc(list, func(i model.Identifier) bool { return i.ID == id }) {
		return true
	}
	at := fmt.Sprintf("ids.%s[%d].note", service, len(list))
	if record != "" {
		c.home(e, record, at)
	}
	read := c.said(e, flag)
	if *ids == nil {
		*ids = model.IDs{}
	}
	(*ids)[service] = append(list, model.Identifier{ID: id, Note: read.Note, Sources: read.Sources})
	return true
}

// used marks an attribute read, with its type and its value.
func (c *converter) used(e *Element) {
	e.Mark()
	e.Use("type")
	e.Use("value")
}

// fact reads an attribute as a fact held, with its notes and its citations, which is
// the event at path on record.
func (c *converter) fact(e *Element, record model.Slug, at string) model.Event {
	c.used(e)
	c.home(e, record, at)
	word := typeWord(factTypes, e.Attr("type"))
	if word == "" {
		word = "fact"
	}
	event := model.Event{Type: word, Value: collapse(e.Attr("value")), Private: private(e)}
	event.Note = strings.Join(c.prose(e), "\n")
	event.Sources = c.citations(e)
	return event
}

// address reads an address as a residence, its parts and its telephone the event's
// address and its date, notes, citations and privacy the event's.
func (c *converter) address(e *Element, record model.Slug, at string) model.Event {
	c.home(e, record, at)
	event := model.Event{
		Type:    "residence",
		Address: addressOf(e),
		Date:    c.date(e),
		Private: private(e),
		Sources: c.citations(e),
	}
	event.Note = strings.Join(c.prose(e), "\n")
	return event
}

// rite reads an ordinance of the Latter-day Saints as an event, its temple and its
// status attributes of it, and the family it seals somebody to its family.
func (c *converter) rite(e *Element, record model.Slug, at string) model.Event {
	e.Mark()
	c.home(e, record, at)
	event := model.Event{Type: "event", Private: private(e)}
	if kind, ok := rites[e.Attr("type")]; ok {
		e.Use("type")
		event.Type = kind
	}
	event.Date = c.date(e)
	event.Place = c.placeOf(e.First("place"))
	if temple := collapse(e.Take("temple").Use("val")); temple != "" {
		event.Attributes = append(event.Attributes, model.Attribute{Type: "temple", Value: temple})
	}
	if status := collapse(e.Take("status").Use("val")); status != "" {
		word, ok := riteStatuses[status]
		if !ok {
			word = strings.ToLower(status)
		}
		event.Attributes = append(event.Attributes, model.Attribute{Type: "status", Value: word})
	}
	if family, ok := c.family[e.First("sealed_to").Attr("hlink")]; ok {
		e.Take("sealed_to")
		event.Family = family
	}
	event.Note = strings.Join(c.prose(e), "\n")
	event.Sources = c.citations(e)
	return event
}

// links reads the URLs of a record as its links, a private one making private the
// record, which flag is.
func (c *converter) links(e *Element, flag *bool) []model.Link {
	var out []model.Link
	for _, u := range e.All("url") {
		href := strings.TrimSpace(u.Attr("href"))
		if href == "" {
			continue
		}
		u.Mark()
		u.Use("href")
		kind := u.Use("type")
		link := model.Link{URL: href, Title: collapse(u.Use("description")), Type: typeWord(urlTypes, kind)}
		if kind == "E-mail" && !strings.HasPrefix(strings.ToLower(href), "mailto:") {
			link.URL = "mailto:" + href
		}
		if private(u) && !*flag {
			*flag = true
			c.notes.Add("a private link made private the record it is a link of")
		}
		out = append(out, link)
	}
	return out
}

// association reads a personref as GEDCOM 5.5.1's ASSO is read: an undated event naming
// the other person in the role it gives, since it says at no event which.
func (c *converter) association(ref *Element, record model.Slug, at string) (model.Event, bool) {
	who, ok := c.person[ref.Attr("hlink")]
	if !ok {
		c.notes.Add("an association names a person who is not in the file")
		return model.Event{}, false
	}
	ref.Mark()
	c.home(ref, record, at)
	role := collapse(ref.Use("rel"))
	if role == "" {
		role = "associate"
	}
	event := model.Event{
		Type:    "association",
		Private: private(ref),
		With:    []model.Named{{Person: who, Role: role}},
		Sources: c.citations(ref),
	}
	event.Note = strings.Join(c.prose(ref), "\n")
	return event, true
}

func itoa(n int) string { return strconv.Itoa(n) }
