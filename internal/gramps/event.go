package gramps

import (
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// eventTypes are Gramps' own event types, a few in other words than the GEDCOM import's
// so that a marriage licence is one type in a corpus whichever file it came from, and
// the rest as a slug, which is already the GEDCOM import's word wherever the two have one.
var eventTypes = vocabulary(map[string]string{
	"Adopted":          "adoption",
	"Naturalization":   "naturalisation",
	"Nobility Title":   "title",
	"Marriage License": "marriage-licence",
	"Divorce Filing":   "divorce-filed",
	"Unknown":          "event",
},
	"Birth", "Death", "Adult Christening", "Baptism", "Bar Mitzvah", "Bas Mitzvah",
	"Blessing", "Burial", "Cause Of Death", "Census", "Christening", "Confirmation",
	"Cremation", "Degree", "Education", "Elected", "Emigration", "First Communion",
	"Immigration", "Graduation", "Medical Information", "Military Service",
	"Number of Marriages", "Occupation", "Ordination", "Probate", "Property", "Religion",
	"Residence", "Retirement", "Will", "Marriage", "Marriage Settlement",
	"Marriage Contract", "Marriage Banns", "Engagement", "Divorce", "Annulment",
	"Alternate Marriage", "Stillbirth",
)

// roles are Gramps' own roles at an event.
var roles = vocabulary(nil,
	"Unknown", "Primary", "Clergy", "Celebrant", "Aide", "Bride", "Groom", "Witness",
	"Family", "Informant", "Godparent", "Father", "Mother", "Parent", "Child",
	"Multiple birth", "Friend", "Neighbor", "Officiator",
)

// updated is the event FTM and its kin write for when they last changed a record, which
// says nothing about the person, as GEDCOM's CHAN does not.
const updated = "_UPD"

func eventType(words string) string {
	if word := typeWord(eventTypes, words); word != "" {
		return word
	}
	return "event"
}

// role is the role a reference to an event gives the person it is from, which Gramps
// leaves out where it is the default.
func role(ref *Element) string {
	if r := collapse(ref.Attr("role")); r != "" {
		return r
	}
	return "Primary"
}

// copies reports whether a person pointing at an event in the role given has a copy of
// it in their own record. An event a family holds is its own and nobody else's;
// otherwise it is copied to whoever it is about, which is the Primary, and the bride
// and the groom of a marriage nobody recorded as a family.
func (c *converter) copies(role, event string) bool {
	if c.familyHeld[event] {
		return false
	}
	return role == "Primary" || role == "Bride" || role == "Groom"
}

// readEvents reads every event once, into what all its copies share, and names in it
// everybody pointing at it who will not hold a copy: the witness, the priest, the
// informant, the godparent. An event nobody holds is held by a note record of its own.
func (c *converter) readEvents() {
	for _, e := range c.all("events", "event") {
		if collapse(e.First("type").Text()) == updated {
			continue
		}
		handle := e.Attr("handle")
		held := c.familyHeld[handle]
		for _, h := range c.holders[handle] {
			held = held || c.copies(role(h.ref), handle)
		}
		e.Mark()
		kind := collapse(e.Take("type").Text())
		event := model.Event{Type: eventType(kind), Private: private(e), IDs: c.grampsID(e)}

		// a description is the event's value, whether it says what was held or what
		// happened, Gramps having the one field for both
		event.Value = collapse(e.Take("description").Text())
		var note []string
		event.Date = c.date(e)
		event.Place = c.placeOf(e.First("place"))
		event.Cause.Value = collapse(e.Take("cause").Text())
		for _, attribute := range e.All("attribute") {
			c.eventAttribute(&event, attribute, "", "")
		}
		note = append(note, c.prose(e)...)
		event.Note = strings.Join(note, "\n\n")
		event.Sources = c.citations(e)
		event.Tags = c.tags(e)

		for _, h := range c.holders[handle] {
			if c.copies(role(h.ref), handle) {
				continue
			}
			if named, ok := c.named(&event, h); ok {
				event.With = append(event.With, named)
			}
		}
		c.events[handle] = event
		if !held {
			c.unheld(e, kind, event)
		}
	}
}

// unheld makes a note record hold an event nobody holds, titled with its type and its
// date, so that it is still there to be placed.
func (c *converter) unheld(e *Element, kind string, event model.Event) {
	title := strings.TrimSpace(kind + " " + event.Date.String())
	base, err := model.Slugify(title)
	if err != nil {
		base = "event"
	}
	slug := c.namer.Assign(importing.Shorten(base), nil, title)
	if slug == "" {
		slug = c.namer.Assign("event", nil, title)
	}
	c.corpus.Notes[slug] = &model.Note{Slug: slug, Title: title, Events: []model.Event{event}}
	c.eventAt[e.Attr("handle")] = slug
	c.homeEvent(e, slug, "events[0].note")
	for _, h := range c.holders[e.Attr("handle")] {
		c.home(h.ref, slug, "events[0].note")
	}
}

// unheldPictures reads the pictures of each event a note holds once every record has
// claimed the pictures it names, since a note holds an event nobody else does.
func (c *converter) unheldPictures() {
	for _, e := range c.all("events", "event") {
		slug, ok := c.eventAt[e.Attr("handle")]
		if note := c.corpus.Notes[slug]; ok && note != nil && len(note.Events) == 1 {
			c.pictures(e, note, &note.Events[0].Media, 0)
		}
	}
}

// homeEvent sends what is unread in an event to path on record, the first record
// holding it being where a link to the event leads.
func (c *converter) homeEvent(e *Element, record model.Slug, path string) {
	c.home(e, record, path)
	if _, held := c.eventAt[e.Attr("handle")]; !held {
		c.eventAt[e.Attr("handle")] = record
	}
}

// named is somebody an event names, in the role their reference to it gives, with what
// the reference says of them. Their reference being private is not something the event
// can say, so it is left unread.
func (c *converter) named(event *model.Event, h holder) (model.Named, bool) {
	who, ok := c.person[h.person]
	if !ok {
		return model.Named{}, false
	}
	h.ref.Mark()
	h.ref.Use("role")
	named := model.Named{Person: who, Role: typeWord(roles, role(h.ref)), Sources: c.citations(h.ref)}
	if named.Role == "" {
		named.Role = "associate"
	}
	for _, attribute := range h.ref.All("attribute") {
		c.namedAttribute(event, &named, attribute)
	}
	named.Note = strings.Join(c.prose(h.ref), "\n\n")
	return named, true
}

// copyOf is a person's or a family's copy of an event, with what their own reference to
// it adds: an age, a cause, and the flag that makes their copy private. It is the event
// at path on record, where what is unread in the event and in the reference goes.
func (c *converter) copyOf(event string, ref *Element, record model.Slug, at string) model.Event {
	ref.Mark()
	ref.Use("role")
	c.homeEvent(c.handles[event], record, at)
	c.home(ref, record, at)

	copied := c.events[event]
	copied.With = slices.Clone(copied.With)
	copied.Attributes = slices.Clone(copied.Attributes)
	copied.Sources = slices.Clone(copied.Sources)
	copied.Tags = slices.Clone(copied.Tags)
	copied.IDs = cloneIDs(copied.IDs)
	copied.Private = copied.Private || private(ref)
	var note []string
	if copied.Note != "" {
		note = append(note, copied.Note)
	}
	for _, attribute := range ref.All("attribute") {
		c.eventAttribute(&copied, attribute, record, at)
	}
	note = append(note, c.prose(ref)...)
	copied.Note = strings.Join(note, "\n\n")
	copied.Sources = append(copied.Sources, c.citations(ref)...)
	return copied
}

// cloneIDs copies identifiers, so that one copy of an event changing its own does not
// change another's.
func cloneIDs(ids model.IDs) model.IDs {
	if ids == nil {
		return nil
	}
	out := model.IDs{}
	for service, list := range ids {
		out[service] = slices.Clone(list)
	}
	return out
}

// said is an attribute as an entry of attributes, with its notes and its citations,
// marked read. A private one makes private what holds it, which the flag given is, and
// the report counts it.
func (c *converter) said(e *Element, flag *bool) model.Attribute {
	c.used(e)
	texts := c.prose(e)
	if e.Use("priv") == "1" && !*flag {
		*flag = true
		c.notes.Add("a private attribute made private what it is on")
	}
	return model.Attribute{
		Type:    typeWord(factTypes, e.Attr("type")),
		Value:   collapse(e.Attr("value")),
		Note:    strings.Join(texts, "\n\n"),
		Sources: c.citations(e),
	}
}

// eventAttribute reads an attribute on an event or on a reference to one into the
// event. A cause is its cause and an age its age, written qualified where the attribute
// has notes or citations; a time is the time of its date where the date has a day and
// nothing is said of the time; a witness is somebody it names by their name. Anything
// else, and any of those whose value the field cannot hold, is an entry of its
// attributes.
func (c *converter) eventAttribute(event *model.Event, e *Element, record model.Slug, at string) {
	if c.identifier(&event.IDs, &event.Private, record, e) {
		return
	}
	read := c.said(e, &event.Private)
	switch e.Attr("type") {
	case "Cause":
		if read.Value != "" && event.Cause.Value == "" {
			event.Cause = model.StatedCause{Value: read.Value, Note: read.Note, Sources: read.Sources}
			return
		}
	case "Age":
		if age, ok := c.age(read.Value); ok && event.Age.Value == "" {
			event.Age = model.StatedAge{Value: age, Note: read.Note, Sources: read.Sources}
			return
		}
	case "Time":
		if read.Note == "" && len(read.Sources) == 0 {
			if date, ok := withTime(event.Date, read.Value); ok {
				event.Date = date
				return
			}
		}
	case "Witness":
		if read.Value != "" {
			event.With = append(event.With, model.Named{
				Name: model.PersonName{Text: read.Value}, Role: "witness", Note: read.Note, Sources: read.Sources,
			})
			return
		}
	}
	event.Attributes = append(event.Attributes, read)
}

// namedAttribute reads an attribute on the reference of somebody an event names: an age
// with nothing said of it is their age, and anything else an entry of their attributes.
func (c *converter) namedAttribute(event *model.Event, named *model.Named, e *Element) {
	read := c.said(e, &event.Private)
	if e.Attr("type") == "Age" && read.Note == "" && len(read.Sources) == 0 && named.Age == "" {
		if age, ok := c.age(read.Value); ok {
			named.Age = age
			return
		}
	}
	named.Attributes = append(named.Attributes, read)
}

// withTime is a date with the time of day Gramps keeps as an attribute, where the date
// gives a day and no time already, and the time is one the grammar reads.
func withTime(date model.Date, clock string) (model.Date, bool) {
	if date.Day == nil || date.Time != nil || date.End != nil || date.Period() || clock == "" {
		return date, false
	}
	timed, err := model.ParseDate(date.String() + " " + clock)
	if err != nil || timed.Time == nil {
		return date, false
	}
	return timed, true
}

// age reads an age as a record gives it, reporting whether the grammar could.
func (c *converter) age(text string) (model.Age, bool) {
	age, err := model.ParseAge(collapse(text))
	return age, err == nil && age != ""
}

// placeOf is the place a reference names.
func (c *converter) placeOf(ref *Element) model.Slug {
	if ref == nil {
		return ""
	}
	if slug, ok := c.place[ref.Attr("hlink")]; ok {
		ref.Mark()
		return slug
	}
	c.notes.Add("a place was referred to that is not in the file")
	return ""
}
