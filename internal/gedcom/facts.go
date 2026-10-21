package gedcom

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/origins-project/origins/internal/model"
)

// individualEvents and familyEvents name what the model calls each GEDCOM tag. The
// model's event types are an open set of ordinary words, because real records produce
// events no fixed list anticipates — so these tables spell the vocabulary out rather
// than abbreviate it, and a tag missing from them is kept as written and not guessed at.
var individualEvents = map[string]string{
	"BIRT": "birth", "CHR": "christening", "CHRA": "adult-christening",
	"BAPM": "baptism", "BLES": "blessing", "CONF": "confirmation",
	"FCOM": "first-communion", "ORDN": "ordination",
	"BARM": "bar-mitzvah", "BASM": "bas-mitzvah",
	"DEAT": "death", "BURI": "burial", "CREM": "cremation", "PROB": "probate",
	"WILL": "will", "ADOP": "adoption", "GRAD": "graduation", "RETI": "retirement",
	"CENS": "census", "EMIG": "emigration", "IMMI": "immigration",
	"NATU": "naturalisation",

	"OCCU": "occupation", "RESI": "residence", "EDUC": "education",
	"NATI": "nationality", "RELI": "religion", "PROP": "property",
	"TITL": "title", "CAST": "caste", "DSCR": "description",
	"IDNO": "identifying-number", "SSN": "social-security-number",
	"NCHI": "number-of-children", "NMR": "number-of-marriages",

	"BAPL": "lds-baptism", "CONL": "lds-confirmation", "INIL": "lds-initiatory",
	"ENDL": "lds-endowment", "SLGC": "lds-sealing-to-parents",

	// what Gramps writes for its own events GEDCOM has no tag for, in the words its XML
	// gives them, and what Family Tree Maker writes that Gramps reads
	"_DEG": "degree", "_ELEC": "elected", "_MDCL": "medical-information",
	"_MILT": "military-service",
	"_FUN":  "funeral", "_CIRC": "circumcision", "_EXCM": "excommunication",
	"_MISN": "lds-mission", "_INIT": "lds-initiatory", "_ORDI": "ordinance",
	"_DEST": "destination", "_EMPLOY": "employment", "_DCAUSE": "cause-of-death",
	"_DNA": "dna", "_HEIG": "height", "_WEIG": "weight", "_EYC": "eye-colour",
	"_MILTID": "military-id", "_ORIG": "origin", "_NAMS": "namesake",
}

var familyEvents = map[string]string{
	"MARR": "marriage", "MARB": "marriage-banns", "MARC": "marriage-contract",
	"MARL": "marriage-licence", "MARS": "marriage-settlement",
	"ENGA": "engagement", "ANUL": "annulment",
	"DIV": "divorce", "DIVF": "divorce-filed",
	"CENS": "census", "RESI": "residence", "NCHI": "number-of-children",

	"SLGS": "lds-sealing-to-spouse",

	"_SEPR": "separation", "_COML": "common-law-marriage",
}

// ordinanceStatuses are the words for the states of a rite of the Latter-day Saints
// that GEDCOM abbreviates; any other is written in lower case with a hyphen for the
// underscore.
var ordinanceStatuses = map[string]string{
	"BIC": "born-in-covenant", "DNS": "do-not-seal", "DNS_CAN": "do-not-seal-canceled",
}

// tied are the events whose FAMC names the family they concern.
var tied = map[string]bool{"BIRT": true, "CHR": true, "ADOP": true, "SLGC": true}

func (c *converter) primaryName(record *Node) model.PersonName {
	for _, node := range record.All("NAME") {
		return readName(node)
	}
	return model.PersonName{}
}

// named keeps a part of a name that is one, and discards a placeholder standing in for
// one the record does not give. Exports are full of these — a lone question mark, a row
// of underscores, an ellipsis — and a part with no letter and no digit anywhere in it is
// not a name anybody had. Saying nothing is what the model does when a record says
// nothing, and it is reported like any other absent surname.
//
// The test is over the whole part rather than word by word, because a word with no
// letter in it is ordinary inside a part that has them: the ampersand of "Bishop of
// Salisbury & Durham" is punctuation in a title, not a stand-in for one nobody recorded.
func named(s string) string {
	s = collapse(s)
	if strings.ContainsFunc(s, func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r)
	}) {
		return s
	}
	return ""
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// displayName is how a person is described when a collision has to be settled by hand.
func displayName(name model.PersonName, xref string) string {
	if !name.IsZero() {
		return name.String()
	}
	return "the individual " + xref
}

// readEvents reads the events of record into entity, which it makes, each there as soon
// as it is read, so that a picture one event keeps can be found from the next.
func (c *converter) readEvents(record *Node, kinds map[string]string, entity any) []model.Event {
	subject := model.RefOf(entity).Slug
	events := eventsOf(entity)
	for _, node := range record.Children {
		kind, ok := kinds[node.Tag]
		switch {
		case ok:
			if spelled := c.extension(node, "_TYPE"); spelled != "" {
				kind = spelled // the record's own spelling of a kind 7.0 has a tag for
			}
		case node.Tag == "_EVENT" && c.ours[node.Tag]:
			// A note's event, origins' own, its line the kind.
			if kind = collapse(node.Text()); kind == "" {
				kind = "event"
			}
		case node.Tag == "EVEN" || node.Tag == "FACT":
			// EVEN and FACT are GEDCOM's own escape hatch: the kind is whatever the
			// TYPE subtag says, as written, which is exactly the open vocabulary the
			// model has.
			kind = "event"
			if named := collapse(node.Take("TYPE").Text()); named != "" {
				kind = named
			}
		default:
			continue
		}
		i := len(*events)
		at := fmt.Sprintf("events[%d]", i)
		node.Mark()
		c.home(node, subject, at+".note")
		if famc := node.First("FAMC"); record.Tag == "INDI" && tied[node.Tag] && famc != nil {
			c.ties = append(c.ties, tie{person: subject, event: i, famc: famc, tag: node.Tag})
		}
		*events = append(*events, c.readEvent(record, node, kind, subject, at))
		c.readPictures(node, entity, i, &(*events)[i].Media, at+".")
	}
	return *events
}

// readEvent reads node as the event at the path at in subject.
func (c *converter) readEvent(record, node *Node, kind string, subject model.Slug, at string) model.Event {
	event := model.Event{Type: kind, Private: c.private(node)}
	var note []string

	// What a line carries is the event's value: what an attribute held — the occupation,
	// the number of children — or the words an exporter wrote of an event, as Family
	// Tree Maker writes its description there. A Y on an event's line says only that it
	// happened.
	if value := strings.TrimSpace(node.Text()); value != "Y" && node.Tag != "_EVENT" {
		event.Value = value
	}
	if value := c.take(node, "_VALUE"); value != nil {
		event.Value = strings.TrimSpace(value.Text())
	}

	date, read := readDateNode(node.Take("DATE"))
	event.Date = date
	if !read {
		c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
	}

	sort, read := readDateNode(node.Take("SDATE"))
	event.Sort = sort
	if !read {
		c.notes.Add("a date could not be read and was kept as words: %q", sort.Wording)
	}

	event.Place = c.place(node.First("PLAC"))
	if addr := node.First("ADDR"); addr != nil {
		event.Address = c.address(addr)
		c.home(addr, subject, at+".address.note")
	}
	age, unread := c.age(node.First("AGE"))
	event.Age.Value = age
	if unread != "" {
		note = append(note, "age: "+unread)
	}
	event.Cause.Value = collapse(node.Take("CAUS").Text())
	event.Attributes = c.eventAttributes(node)
	event.Attributes = append(event.Attributes, c.attributes(node, subject)...)
	event.Tags = c.tagged(node)

	if record.Tag == "FAM" {
		partners, unread := c.partnersAged(record, node)
		event.With = partners
		note = append(note, unread...)
	}
	for _, asso := range node.All("ASSO") {
		if who, ok := c.associate(asso); ok {
			asso.Mark()
			c.home(asso, subject, fmt.Sprintf("%s.with[%d].note", at, len(event.With)))
			who.Sources = c.citations(asso, subject, kind)
			event.Private = c.private(asso) || event.Private
			event.With = append(event.With, who)
		}
	}

	event.Links = c.links(node)
	event.IDs = c.ids(node)
	event.Sources = c.citations(node, subject, kind)
	if len(event.Sources) == 0 {
		c.notes.Add("an event cites no source")
	}
	note = append(note, c.noteTexts(node)...)
	event.Note = strings.Join(note, "\n\n")
	return event
}

// eventAttributes reads what GEDCOM says of an event that no field of the model's holds:
// a standard event's TYPE as its classification, the TYPE of an EVEN or a FACT being its
// kind; AGNC as its agency and RELI as its religion; and a rite's TEMP as its temple and
// STAT as its status, dated as the STAT is, in origins' words.
func (c *converter) eventAttributes(node *Node) []model.Attribute {
	var out []model.Attribute
	for _, line := range node.Children {
		var attribute model.Attribute
		switch line.Tag {
		case "TYPE":
			if node.Tag == "EVEN" || node.Tag == "FACT" {
				continue
			}
			attribute.Type = "classification"
		case "AGNC":
			attribute.Type = "agency"
		case "RELI":
			attribute.Type = "religion"
		case "TEMP":
			attribute.Type = "temple"
		case "STAT":
			attribute.Type = "status"
			date, read := readDateNode(line.Take("DATE"))
			if !read {
				c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
			}
			attribute.Date = date
		default:
			continue
		}
		attribute.Value = collapse(line.Text())
		if line.Tag == "STAT" {
			attribute.Value = ordinanceStatus(attribute.Value)
		}
		if attribute.Value == "" && attribute.Date.IsZero() {
			continue
		}
		line.Mark()
		out = append(out, attribute)
	}
	return out
}

// ordinanceStatus is the word for the state of a rite.
func ordinanceStatus(word string) string {
	if spelled, ok := ordinanceStatuses[strings.ToUpper(word)]; ok {
		return spelled
	}
	return strings.ReplaceAll(strings.ToLower(word), "_", "-")
}

// readNot reads 7.0's NO, what a record says did not happen, its tag in the words of
// the event it denies, its DATE the period over which it did not, and its notes and
// citations its own. A NO of a tag kinds has no word for is kept as written.
func (c *converter) readNot(record *Node, kinds map[string]string, subject model.Slug) []model.NonEvent {
	var out []model.NonEvent
	for _, no := range record.All("NO") {
		kind, ok := kinds[strings.ToUpper(strings.TrimSpace(no.Value))]
		if !ok {
			continue
		}
		no.Mark()
		c.home(no, subject, fmt.Sprintf("not[%d].note", len(out)))
		date, read := readDateNode(no.Take("DATE"))
		if !read {
			c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
		}
		out = append(out, model.NonEvent{
			Type:    kind,
			Date:    date,
			Note:    strings.Join(c.noteTexts(no), "\n"),
			Sources: c.citations(no, subject, "not-"+kind),
		})
	}
	return out
}

// age reads the age a record gives under an event, and hands back the words of one the
// grammar cannot hold — 5.5.1's CHILD, INFANT and STILLBORN, or a 7.0 PHRASE — for the
// note.
func (c *converter) age(node *Node) (model.Age, string) {
	text := collapse(node.Mark().Text())
	words := collapse(node.Take("PHRASE").Text())
	age, err := model.ParseAge(text)
	if err != nil {
		c.notes.Add("an age could not be read and was kept in the note: %q", text)
		return "", strings.TrimSpace(text + " " + words)
	}
	return age, words
}

// partnerRoles are the partners a family event can give the age of, and the role each
// is named in.
var partnerRoles = []struct{ tag, role string }{{"HUSB", "husband"}, {"WIFE", "wife"}}

// partnersAged names the partners whose ages a family event gives, which GEDCOM writes
// under HUSB and WIFE beneath the event, as people the event names in that role.
func (c *converter) partnersAged(family, event *Node) (named []model.Named, unread []string) {
	for _, partner := range partnerRoles {
		held := event.First(partner.tag)
		age, words := c.age(held.First("AGE"))
		if words != "" {
			unread = append(unread, partner.role+"'s age: "+words)
			held.Mark()
		}
		if age == "" {
			continue
		}
		slug, ok := c.person[pointer(family.First(partner.tag).Text())]
		if !ok {
			continue
		}
		held.Mark()
		named = append(named, model.Named{Person: slug, Role: partner.role, Age: age})
	}
	return named, unread
}

// associations are the people a record names beside whom it is about without saying at
// which event, which is the only way 5.5.1 can write one. Each is an event of its own,
// undated and naming them in the role given, so that the association is kept without a
// guess at the baptism a godfather stood at.
//
// first is how many events the record has before them.
func (c *converter) associations(record *Node, subject model.Slug, first int) []model.Event {
	var events []model.Event
	for _, asso := range record.All("ASSO") {
		who, ok := c.associate(asso)
		if !ok {
			continue
		}
		asso.Mark()
		c.home(asso, subject, fmt.Sprintf("events[%d].note", first+len(events)))
		event := model.Event{
			Type:    "association",
			Private: c.private(asso) || c.flagged(asso),
			Note:    who.Note,
			Sources: c.citations(asso, subject, "association"),
		}
		who.Note = ""
		event.With = []model.Named{who}
		if len(event.Sources) == 0 {
			c.notes.Add("an event cites no source")
		}
		events = append(events, event)
	}
	return events
}

// associate reads an ASSO as somebody a record names: by their record, by the words of
// its PHRASE where 7.0 points at no record, or as unknown where it says nothing of them,
// and in the role it gives them. A PHRASE beside a record is the file's words for the
// link, as it is under a CHIL, and leads the note the ASSO's notes make.
func (c *converter) associate(asso *Node) (model.Named, bool) {
	who := model.Named{Role: role(asso)}
	switch slug, ok := c.person[pointer(asso.Value)]; {
	case voided(asso.Value):
		if words := named(asso.First("PHRASE").Text()); words != "" {
			asso.Take("PHRASE")
			who.Name = model.PersonName{Text: words}
		} else {
			who.Person = model.Unknown
		}
	case ok:
		who.Person = slug
	default:
		c.notes.Add("an ASSO names an individual who is not in the file")
		return who, false
	}
	var note []string
	if words := collapse(asso.First("PHRASE").Text()); words != "" && !voided(asso.Value) {
		asso.Take("PHRASE")
		note = append(note, words)
	}
	who.Note = strings.Join(append(note, c.noteTexts(asso)...), "\n")
	return who, true
}

// roles are the words for the roles 7.0 enumerates.
var roles = map[string]string{
	"CHIL": "child", "CLERGY": "clergy", "FATH": "father", "FRIEND": "friend",
	"GODP": "godparent", "HUSB": "husband", "MOTH": "mother",
	"MULTIPLE": "sibling of the same birth", "NGHBR": "neighbour",
	"OFFICIATOR": "officiator", "PARENT": "parent", "SPOU": "spouse", "WIFE": "wife",
	"WITN": "witness", "OTHER": "other",
}

// role is the role an ASSO gives somebody. A 7.0 PHRASE is the record's own word and
// says more than the enumeration it refines — a godmother where the value says a
// godparent — and 5.5.1's RELA is free text kept as written. An ASSO that says nothing
// names an associate, which is what the tag means.
func role(asso *Node) string {
	given := asso.First("ROLE")
	if phrase := collapse(given.First("PHRASE").Text()); phrase != "" {
		given.Mark().Take("PHRASE")
		return phrase
	}
	if word, ok := roles[strings.ToUpper(strings.TrimSpace(given.Text()))]; ok {
		given.Mark()
		return word
	}
	if written := collapse(given.Text()); written != "" {
		given.Mark()
		return written
	}
	if written := collapse(asso.First("RELA").Text()); written != "" {
		asso.Take("RELA")
		return written
	}
	return "associate"
}

// citations turn each SOUR under a node into a reference to an anchor in a source.
// GEDCOM repeats the transcription at every event that cites it; the model states it
// once in the source and points at it, so the work here is to recognise an entry
// already transcribed. The entry is where in the source it is, when it was written,
// every TEXT of it a paragraph of its transcription, the kind of entry its EVEN says,
// with the ROLE under it naming the record citing it, its scans and its notes; QUAY is
// how much it weighs for what cites it, and so is the reference's.
func (c *converter) citations(node *Node, subject model.Slug, kind string) []model.CitationRef {
	var refs []model.CitationRef
	for _, sour := range node.All("SOUR") {
		source := c.sourceOf(sour)
		if source == "" {
			continue
		}
		sour.Mark()
		data := sour.Take("DATA")
		var text []string
		for _, node := range append(data.All("TEXT"), sour.All("TEXT")...) {
			if words, _ := c.written(node, ""); words != "" {
				node.Mark()
				if c.mine {
					words = node.Text() // a transcription origins wrote ends as it wrote it
				}
				text = append(text, words)
			}
		}
		date, read := readDateNode(data.Take("DATE"))
		if !read {
			c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
		}
		citation := model.Citation{
			Page:  collapse(sour.Take("PAGE").Text()),
			Date:  date,
			Text:  strings.Join(text, "\n\n"),
			Links: c.links(sour),
		}
		var note []string
		citation.Event, citation.With, note = c.cited(sour.First("EVEN"), subject)
		citation.Note = strings.Join(append(note, c.noteTexts(sour)...), "\n\n")
		citation.Images = c.scans(sour, source)
		citation.Attributes = c.attributes(sour, source)
		citation.Tags = c.tagged(sour)
		named := c.extension(sour, "_ANCHOR")
		ref := model.CitationRef{Source: source, Anchor: c.anchor(source, citation, subject, kind, named)}
		if grade := sour.First("_GRADE"); c.ours["_GRADE"] && grade != nil {
			if read, err := model.ParseGrade(collapse(grade.Text())); err == nil && read.Weight == 0 {
				grade.Mark()
				ref.Grade = read
			}
		}
		if quay := strings.TrimSpace(sour.First("QUAY").Text()); len(quay) == 1 && quay >= "0" && quay <= "3" {
			sour.Take("QUAY")
			ref.Grade.Weight = quay[0]
		}
		c.home(sour, source, "citations."+ref.Anchor+".note")
		refs = append(refs, ref)
	}
	return refs
}

// cited reads a citation's EVEN: the kind of entry it is, in the words an event's type
// is written in, and the ROLE under it as somebody the entry names, in the words an
// ASSO's role is written in — the person whose record cites it, or on a family the
// partner it names as its HUSB or WIFE for those two roles and unknown for any other.
// OTHER is in its PHRASE's words; any other PHRASE is the note of the entry, or of the
// citation, handed back.
func (c *converter) cited(even *Node, subject model.Slug) (string, []model.Named, []string) {
	kind := eventWord(even.Text())
	if kind == "" {
		return "", nil, nil
	}
	even.Mark()
	var note []string
	if phrase := collapse(even.First("PHRASE").Text()); phrase != "" {
		even.Take("PHRASE")
		note = append(note, phrase)
	}
	given := even.First("ROLE")
	word := strings.Trim(collapse(given.Text()), "()")
	if word == "" {
		return kind, nil, note
	}
	given.Mark()
	who := model.Named{Person: c.citing(subject, strings.ToUpper(word)), Role: word}
	if spelled, ok := roles[strings.ToUpper(word)]; ok {
		who.Role = spelled
	}
	if phrase := collapse(given.First("PHRASE").Text()); phrase != "" {
		given.Take("PHRASE")
		if strings.EqualFold(word, "OTHER") {
			who.Role = phrase
		} else {
			who.Note = phrase
		}
	}
	return kind, []model.Named{who}, note
}

// citing is who a citation's role names on the record subject: the person, or the
// partner a family names in that role.
func (c *converter) citing(subject model.Slug, role string) model.Slug {
	if c.individual[subject] {
		return subject
	}
	if fam := c.fam[subject]; fam != nil && (role == "HUSB" || role == "WIFE") {
		if slug, ok := c.person[pointer(fam.First(role).Text())]; ok {
			return slug
		}
	}
	return model.Unknown
}

// anchor names a citation within a source, reusing the existing one when the same entry
// says the same thing, and naming in it whoever else this citing of it names. The name is
// the page where there is one, because that is what a reader would look up, and the
// event otherwise, unless the file names it, as origins does where the page would not.
func (c *converter) anchor(source model.Slug, citation model.Citation, subject model.Slug, kind, named string) string {
	base, err := model.Slugify(named)
	if err != nil {
		base, err = model.Slugify(citation.Page)
	}
	if err != nil {
		base, err = model.Slugify(string(subject) + "-" + kind)
		if err != nil {
			base = "citation"
		}
	}

	within := c.corpus.Sources[source].Citations
	for candidate := 1; ; candidate++ {
		name := string(base)
		if candidate > 1 {
			name += "-" + strconv.Itoa(candidate)
		}
		existing, used := within[name]
		if !used {
			within[name] = citation
			return name
		}
		if sameEntry(existing, citation) {
			for _, who := range citation.With {
				if !slices.ContainsFunc(existing.With, func(w model.Named) bool { return reflect.DeepEqual(w, who) }) {
					existing.With = append(existing.With, who)
				}
			}
			within[name] = existing
			return name
		}
	}
}

// sameEntry reports whether two citings are of one entry, saying the same of it
// whoever they name in it.
func sameEntry(a, b model.Citation) bool {
	a.With, b.With = nil, nil
	if a.Date.String() != b.Date.String() {
		return false
	}
	a.Date, b.Date = model.Date{}, model.Date{}
	return reflect.DeepEqual(a, b)
}

func itoa(n int) string { return strconv.Itoa(n) }
