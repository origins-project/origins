package gedcom

import (
	"maps"
	"math"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// attributeTags are the tags whose line carries what was held, which an event's value is;
// every other event's value is an extension beside it.
var attributeTags = map[string]bool{
	"OCCU": true, "RESI": true, "EDUC": true, "NATI": true, "RELI": true, "PROP": true,
	"TITL": true, "CAST": true, "DSCR": true, "IDNO": true, "SSN": true, "NCHI": true, "NMR": true,
}

// rites are the tags of the rites of the Latter-day Saints, the only events 7.0 gives a
// TEMP and a STAT.
var rites = map[string]bool{
	"BAPL": true, "CONL": true, "INIL": true, "ENDL": true, "SLGC": true, "SLGS": true,
}

// riteStatuses are the states of a rite 7.0 enumerates.
var riteStatuses = map[string]bool{
	"BIC": true, "CANCELED": true, "CHILD": true, "COMPLETED": true, "EXCLUDED": true, "DNS": true,
	"DNS_CAN": true, "INFANT": true, "PRE_1970": true, "STILLBORN": true, "SUBMITTED": true,
	"UNCLEARED": true,
}

// mediums are the words of the 7.0 enumeration a MEDI takes, in the words the corpus
// writes them; microfilm and microfiche are FILM and FICHE.
var mediums = map[string]bool{
	"AUDIO": true, "BOOK": true, "CARD": true, "ELECTRONIC": true, "MAGAZINE": true,
	"MANUSCRIPT": true, "MAP": true, "NEWSPAPER": true, "PHOTO": true, "TOMBSTONE": true, "VIDEO": true,
}

// exids are the services an EXID is written for, by the address 7.0 types it with, from
// which the import names the service again.
var exids = map[string]string{
	"familysearch": "https://www.familysearch.org/tree/person/",
	"findagrave":   "https://www.findagrave.com/memorial/",
	"geonames":     "https://www.geonames.org/",
	"wikidata":     "https://www.wikidata.org/entity/",
	"wikitree":     "https://www.wikitree.com/wiki/",
}

// formats are the media types of the files a picture is, by their extension.
var formats = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif",
	".tif": "image/tiff", ".tiff": "image/tiff", ".bmp": "image/bmp", ".webp": "image/webp",
	".heic": "image/heic", ".svg": "image/svg+xml", ".pdf": "application/pdf",
	".mp3": "audio/mpeg", ".wav": "audio/wav", ".mp4": "video/mp4", ".txt": "text/plain",
	".html": "text/html",
}

// keep makes an OBJE record of every picture a record keeps and every scan a source does,
// in the order of the kinds and the slugs, so that every reference to one, from the
// record keeping it or any other, points at the one record; and notes everybody each
// shows, for their INDI to point at it too.
func (e *exporter) keep() {
	e.files = map[string]string{}
	for _, kind := range model.Kinds {
		for _, slug := range e.corpus.SlugsOf(kind) {
			entity := e.corpus.Entity(slug)
			model.Pictures(entity, func(_ string, m *model.Media) {
				if !m.Refers() {
					e.register(entity, *m)
				}
			})
			if source, ok := entity.(*model.Source); ok {
				source.Scans(func(_ string, m *model.Media) {
					if !m.Refers() {
						e.register(entity, *m)
					}
				})
			}
		}
	}
	// a file beside a record that no entry names is no picture GEDCOM can point at
	for _, kind := range model.Kinds {
		if kind == model.KindHypothesis {
			continue
		}
		for _, slug := range e.corpus.SlugsOf(kind) {
			if files := model.Files(e.corpus.Entity(slug)); files != nil {
				for _, file := range *files {
					if _, ok := e.objects[kind.Dir()+"/"+string(slug)+"/"+file]; !ok {
						e.omit("files")
					}
				}
			}
		}
	}
}

// key is where a picture an entry of the record from names is, as a path from the root of
// the corpus, or as written where it is still to be found.
func key(from model.EntityRef, file string) string {
	at := model.Locate(from, file)
	if at.Abroad || at.Record.Kind == "" {
		return file
	}
	return at.Record.Kind.Dir() + "/" + string(at.Record.Slug) + "/" + at.File
}

// uri is a picture's path as the URI reference 7.0's FILE is, each segment of a path
// into the corpus escaped; a picture still to be found is written as the corpus has it,
// a URL already or a path on somebody's computer.
func uri(file string, abroad bool) string {
	if abroad {
		return file
	}
	segments := strings.Split(file, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}

// register writes the OBJE of a picture a record keeps.
func (e *exporter) register(entity any, m model.Media) {
	from := model.RefOf(entity)
	file := key(from, m.File)
	if _, ok := e.objects[file]; ok {
		return
	}
	xref := "O" + itoa(len(e.objectList)+1)
	obje := &line{xref: xref, tag: "OBJE"}
	e.objects[file] = obje
	e.files[xref] = file
	e.objectList = append(e.objectList, obje)

	if m.Private {
		obje.add("RESN", "PRIVACY")
	}
	abroad := model.Locate(from, m.File).Abroad
	if !abroad {
		e.report.Files = append(e.report.Files, file)
	}
	written := obje.add("FILE", uri(file, abroad))
	format := formats[strings.ToLower(path.Ext(file))]
	if format == "" {
		format = "application/octet-stream"
	}
	mediumOf(written.add("FORM", format), m.Type)
	written.text("TITL", m.Title)
	e.prose(obje, m.Note)
	e.cite(obje, m.Sources, from.Slug)
	e.ids(obje, m.IDs, "media.ids")
	e.date(obje, "_DATE", m.Date)
	if m.Place != "" {
		e.placeChain(obje.add("_PLACE", ""), m.Place)
	}
	e.attributes(obje, m.Attributes)
	e.tags(obje, m.Tags)
	// GEDCOM has no record keeping a picture, the first to name it being taken for one
	if keeper, ok := e.xrefs[from.Slug]; ok {
		obje.point("_KEEPER", keeper)
	}

	var stated, read []model.Slug
	for _, shown := range m.Depicts {
		if shown.Note != "" || len(shown.Attributes) > 0 {
			e.omit("media.depicts")
		}
		if shown.Person == from.Slug && from.Kind == model.KindPerson {
			// the keeper's own pointer says it is theirs
			if shown.Region != nil {
				e.omit("media.depicts.region")
			}
			stated = append(stated, shown.Person)
			continue
		}
		if _, ok := e.xrefs[shown.Person]; !ok {
			e.omit("media.depicts")
			continue
		}
		stated = append(stated, shown.Person)
		read = append(read, shown.Person)
		e.depicted[shown.Person] = append(e.depicted[shown.Person], framed{object: xref, region: shown.Region})
	}
	// The import reads who is in a picture from each person's pointer to it, in the order
	// the people are written, with a person keeping it first; the keeper alone is
	// somebody whose picture it is, and not said to be in it.
	slices.SortFunc(read, func(a, b model.Slug) int { return e.ordinal(a) - e.ordinal(b) })
	if from.Kind == model.KindPerson && len(read) > 0 {
		read = append([]model.Slug{from.Slug}, read...)
	}
	if !slices.Equal(stated, read) {
		e.omit("media.depicts")
	}
}

// ordinal is where a record comes among those of its kind, as its xref numbers it.
func (e *exporter) ordinal(slug model.Slug) int {
	n, _ := strconv.Atoi(e.xrefs[slug][1:])
	return n
}

// object is the OBJE of the picture an entry of a record names.
func (e *exporter) object(entity any, file string) (string, bool) {
	obje, ok := e.objects[key(model.RefOf(entity), file)]
	if !ok {
		return "", false
	}
	return obje.xref, true
}

// pictures writes a pointer to the OBJE of each entry of a list, with the part of the
// picture the entry means. What an entry referring to another record's picture says of
// it besides has nowhere to go.
func (e *exporter) pictures(parent *line, entity any, list []model.Media, at string) {
	for _, m := range list {
		object, ok := e.object(entity, m.File)
		if !ok {
			e.omit(at)
			continue
		}
		e.crop(parent.point("OBJE", object), framed{object: object, region: m.Region})
		if !m.Refers() {
			continue
		}
		for field, says := range map[string]bool{
			"private": m.Private, "note": m.Note != "", "attributes": len(m.Attributes) > 0, "sources": len(m.Sources) > 0,
		} {
			if says {
				e.omit(at + "." + field)
			}
		}
	}
}

// crop writes the part of a picture a pointer means: as 7.0's CROP in pixels where the
// size of the picture is known, and as the four fractions the corpus gives where it is
// not or where the pixels, read back as the import reads them, would not give them
// again, a picture of few pixels having nowhere for a part of one.
func (e *exporter) crop(l *line, p framed) {
	r := p.region
	if l == nil || r == nil {
		return
	}
	if e.options.Size != nil {
		if width, height, ok := e.options.Size(e.files[p.object]); ok && width > 0 && height > 0 {
			pixels := func(f float64, of int) int { return int(math.Round(f * float64(of))) }
			top, left := pixels(r.Top, height), pixels(r.Left, width)
			tall, wide := pixels(r.Bottom, height)-top, pixels(r.Right, width)-left
			crop := l.add("CROP", "")
			crop.add("TOP", itoa(top))
			crop.add("LEFT", itoa(left))
			crop.add("HEIGHT", itoa(tall))
			crop.add("WIDTH", itoa(wide))
			back := model.Region{Left: fractionOf(left, width), Top: fractionOf(top, height),
				Right: fractionOf(left+wide, width), Bottom: fractionOf(top+tall, height)}
			if back == *r {
				return
			}
		}
	}
	l.add("_REGION", strings.Join([]string{fraction(r.Left), fraction(r.Top), fraction(r.Right), fraction(r.Bottom)}, " "))
}

// fractionOf is a number of pixels as the fraction of the picture the import reads it as.
func fractionOf(n, of int) float64 { return math.Round(float64(n)/float64(of)*1e4) / 1e4 }

func fraction(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// joined is the words that are there, a space between each.
func joined(words ...string) string {
	var out []string
	for _, word := range words {
		if word != "" {
			out = append(out, word)
		}
	}
	return strings.Join(out, " ")
}

// name writes a NAME, or a TRAN of one: the payload as 7.0 writes it, the surname between
// slashes, its parts as the substructures 7.0 has, and the rest of what the corpus says of
// it as extensions, counting what it cannot write under field, the name's path. It returns
// the line, nil where there is no name.
func (e *exporter) name(parent *line, tag, field string, n model.PersonName, subject model.Slug) *line {
	if parent == nil || n.IsZero() {
		return nil
	}
	payload := n.Text
	if n.HasParts() {
		surname := joined(n.SurnamePrefix, n.Surname.String())
		if surname != "" {
			surname = "/" + surname + "/"
		}
		payload = joined(n.Prefix, n.Given, surname, n.Suffix)
	}
	l := parent.add(tag, payload)
	restricted(l, n.Private)
	if tag == "TRAN" {
		l.add("LANG", string(n.Lang))
	} else if word := strings.ToLower(n.Type); nameTypes[word] && word == n.Type {
		l.add("TYPE", strings.ToUpper(word))
	} else if n.Type != "" {
		l.add("TYPE", "OTHER").add("PHRASE", n.Type)
	}
	l.text("NPFX", n.Prefix)
	l.text("GIVN", n.Given)
	l.text("NICK", n.Nickname)
	switch len(n.Surname) {
	case 0:
		l.text("SPFX", n.SurnamePrefix)
	case 1:
		l.text("SPFX", joined(n.SurnamePrefix, n.Surname[0].Prefix))
		l.text("SURN", n.Surname[0].Name)
	default:
		l.text("SPFX", n.SurnamePrefix)
		for _, surname := range n.Surname {
			l.text("SURN", surname.Name)
			if surname.Prefix != "" {
				e.omit(field + ".surname.prefix")
			}
		}
	}
	for _, surname := range n.Surname {
		if surname.Connector != "" {
			e.omit(field + ".surname.connector")
		}
		if surname.Origin != "" {
			e.omit(field + ".surname.origin")
		}
		if surname.Primary {
			e.omit(field + ".surname.primary")
		}
	}
	l.text("NSFX", n.Suffix)
	if n.HasParts() {
		l.text("_TEXT", n.Text)
	}
	l.text("_CALL", n.Call)
	l.text("_FAMILY_NICKNAME", n.FamilyNickname)
	l.text("_GROUP", n.Group)
	l.text("_DISPLAY", n.Display)
	l.text("_SORT", n.Sort)
	e.date(l, "_FROM", n.From)
	e.date(l, "_UNTIL", n.Until)
	if tag != "TRAN" {
		l.text("_LANG", string(n.Lang))
	}
	e.prose(l, n.Note)
	e.cite(l, n.Sources, subject)
	return l
}

// translation reports whether another name is only the name written in another
// language, which a TRAN under the name says and can say nothing more of.
func translation(n model.PersonName) bool {
	return n.Lang != "" && n.Type == "" && n.From.IsZero() && n.Until.IsZero() && !n.Private &&
		n.Note == "" && len(n.Sources) == 0
}

// event writes an event of an INDI, a FAM or, as an extension, of an SNOTE: under its
// own tag where 7.0 has one, its value as the line where the tag carries what was held,
// and as a FACT or an EVEN typed in its words where there is none.
func (e *exporter) event(parent *line, ev model.Event, record string, subject model.Slug) {
	kinds := individualEvents
	if record == "FAM" {
		kinds = familyEvents
	}
	var l *line
	tag := ""
	switch {
	case record == "SNOTE":
		l = parent.add("_EVENT", ev.Type)
	case eventTag(ev.Type, kinds) != "":
		tag = eventTag(ev.Type, kinds)
		if attributeTags[tag] {
			l = parent.add(tag, ev.Value)
		} else {
			l = parent.add(tag, "")
		}
	case ev.Value != "":
		l = parent.add("FACT", ev.Value)
		l.add("TYPE", ev.Type)
	default:
		l = parent.add("EVEN", "")
		l.add("TYPE", ev.Type)
	}
	if ev.Private {
		l.add("RESN", "PRIVACY")
	}
	if tag != "" && ev.Type != kinds[tag] {
		l.add("_TYPE", ev.Type)
	}

	// what 7.0 has a structure for, the first of each type, and the rest as extensions
	var rest []model.Attribute
	said := map[string]bool{}
	for _, attribute := range ev.Attributes {
		var as, value string
		switch {
		case attribute.Type == "classification" && tag != "":
			as, value = "TYPE", attribute.Value
		case attribute.Type == "agency" && record != "SNOTE":
			as, value = "AGNC", attribute.Value
		case attribute.Type == "religion" && record != "SNOTE":
			as, value = "RELI", attribute.Value
		case attribute.Type == "temple" && rites[tag]:
			as, value = "TEMP", attribute.Value
		case attribute.Type == "status" && rites[tag]:
			as, value = "STAT", riteStatus(attribute.Value)
		}
		dated := as == "STAT" || attribute.Date.IsZero()
		if as == "" || value == "" || said[as] || attribute.Note != "" || len(attribute.Sources) > 0 || !dated {
			rest = append(rest, attribute)
			continue
		}
		said[as] = true
		e.date(l.add(as, value), "DATE", attribute.Date)
	}
	if tag != "" && !attributeTags[tag] {
		l.text("_VALUE", ev.Value)
	}
	e.date(l, "DATE", ev.Date)
	e.date(l, "SDATE", ev.Sort)
	e.place(l, ev.Place)
	e.address(l, ev.Address, "events.address")
	l.text("CAUS", ev.Cause.Value)
	if ev.Cause.Note != "" || len(ev.Cause.Sources) > 0 {
		e.omit("events.cause")
	}
	switch {
	case ev.Age.Value == "":
	case record == "INDI":
		l.add("AGE", age(ev.Age.Value))
	default:
		e.omit("events.age")
	}
	if ev.Age.Note != "" || len(ev.Age.Sources) > 0 {
		e.omit("events.age")
	}

	with := ev.With
	if record == "FAM" {
		with = nil
		slots, _ := e.pair(e.corpus.Families[subject])
		aged := [2]bool{}
		for _, who := range ev.With {
			at := slices.Index([]string{"husband", "wife"}, who.Role)
			if at >= 0 && !aged[at] && who.Age != "" && who.Person != "" && who.Person == slots[at].Person &&
				who.Note == "" && len(who.Sources) == 0 && len(who.Attributes) == 0 {
				aged[at] = true
				l.add([]string{"HUSB", "WIFE"}[at], "").add("AGE", age(who.Age))
				continue
			}
			with = append(with, who)
		}
	}
	e.tie(l, ev, record, tag, subject)
	for _, who := range with {
		e.associate(l, who, subject, false, who.Note, who.Sources)
	}
	e.attributes(l, rest)
	e.pictures(l, e.corpus.Entity(subject), ev.Media, "events.media")
	e.links(l, ev.Links, record != "SNOTE")
	e.prose(l, ev.Note)
	e.cite(l, ev.Sources, subject)
	for _, service := range slices.Sorted(maps.Keys(ev.IDs)) {
		if service == "uid" && len(ev.IDs[service]) > 0 {
			l.add("UID", ev.IDs[service][0].ID)
			if len(ev.IDs[service]) > 1 || ev.IDs[service][0].Qualified() {
				e.omit("events.ids")
			}
			continue
		}
		e.omit("events.ids")
	}
	e.tags(l, ev.Tags)

	if tag != "" && !attributeTags[tag] && len(l.sub) == 0 {
		l.value = "Y"
	}
}

// tie writes the FAMC of a person's birth, christening, adoption or sealing to parents,
// the family the event ties them to, and of an adoption who adopted them in it.
func (e *exporter) tie(l *line, ev model.Event, record, tag string, subject model.Slug) {
	if ev.Family == "" {
		return
	}
	xref, ok := e.xrefs[ev.Family]
	if record != "INDI" || !tied[tag] || !ok {
		e.omit("events.family")
		return
	}
	famc := l.point("FAMC", xref)
	if tag != "ADOP" {
		return
	}
	family := e.corpus.Families[ev.Family]
	slots, _ := e.pair(family)
	for _, child := range family.Children {
		if child.ID != subject {
			continue
		}
		var by []bool
		for _, slot := range slots {
			by = append(by, slot.Somebody() && child.To(slot.Person).Kin() == model.Adoptive)
		}
		switch {
		case by[0] && by[1]:
			famc.add("ADOP", "BOTH")
		case by[0]:
			famc.add("ADOP", "HUSB")
		case by[1]:
			famc.add("ADOP", "WIFE")
		}
		return
	}
}

// riteStatus is a rite's state as 7.0 enumerates it, or "" where it does not.
func riteStatus(word string) string {
	for tag, spelled := range ordinanceStatuses {
		if spelled == word {
			return tag
		}
	}
	if tag := strings.ToUpper(strings.ReplaceAll(word, "-", "_")); riteStatuses[tag] && strings.ToLower(tag) == strings.ReplaceAll(word, "-", "_") {
		return tag
	}
	return ""
}

// age is an age as 7.0 writes one, which has no bare number of years.
func age(a model.Age) string {
	if _, err := strconv.ParseUint(string(a), 10, 32); err == nil {
		return string(a) + "y"
	}
	return string(a)
}

// not writes what a record says did not happen as 7.0's NO.
func (e *exporter) not(parent *line, list []model.NonEvent, kinds map[string]string, subject model.Slug) {
	for _, no := range list {
		tag := eventTag(no.Type, kinds)
		if tag == "" {
			e.omit("not")
			continue
		}
		l := parent.add("NO", tag)
		e.date(l, "DATE", no.Date)
		e.prose(l, no.Note)
		e.cite(l, no.Sources, subject)
	}
}

// associate writes somebody an event names as an ASSO: by their record, by their name
// in a PHRASE where they have none, or as nobody known, and in the role 7.0 has a word
// for or in their own words.
func (e *exporter) associate(parent *line, who model.Named, subject model.Slug, hidden bool, note string, sources []model.CitationRef) {
	var l *line
	switch xref, ok := e.xrefs[who.Person]; {
	case ok:
		l = parent.point("ASSO", xref)
	case who.Person == "" && !who.Name.IsZero():
		l = parent.void("ASSO")
		l.add("PHRASE", who.Name.String())
	default:
		if who.Person != "" && who.Person != model.Unknown {
			e.omit("with")
		}
		l = parent.void("ASSO")
	}
	restricted(l, hidden)
	if who.Person == "" && who.Name.HasParts() {
		e.omit("with.name")
	}
	if tag, ok := roleTag(who.Role); ok {
		l.add("ROLE", tag)
	} else {
		l.add("ROLE", "OTHER").add("PHRASE", who.Role)
	}
	if who.Age != "" {
		e.omit("with.age")
	}
	if len(who.Attributes) > 0 {
		e.omit("with.attributes")
	}
	e.prose(l, note)
	e.cite(l, sources, subject)
}

// roleTag is 7.0's word for a role, where it has one.
func roleTag(role string) (string, bool) {
	for _, tag := range slices.Sorted(maps.Keys(roles)) {
		if roles[tag] == role {
			return tag, true
		}
	}
	return "", false
}

// cite writes a SOUR for each citation, saying again where it is cited what the entry
// is, as GEDCOM does: where in the source, when it was written and what it says, the kind
// of entry with the role it names whoever cites it in, how much it weighs for this fact
// and its scans.
func (e *exporter) cite(parent *line, refs []model.CitationRef, subject model.Slug) {
	if parent == nil {
		return
	}
	for _, ref := range refs {
		source := e.corpus.Sources[ref.Source]
		xref, ok := e.xrefs[ref.Source]
		if source == nil || !ok {
			e.omit("sources")
			continue
		}
		l := parent.point("SOUR", xref)
		c, ok := source.Citations[ref.Anchor]
		at := string(ref.Source) + "#" + ref.Anchor
		if e.cited[at] == nil {
			e.cited[at] = map[int]bool{}
		}
		if !ok {
			continue
		}
		l.text("PAGE", c.Page)
		if page, err := model.Slugify(c.Page); err != nil || string(page) != ref.Anchor {
			l.add("_ANCHOR", ref.Anchor)
		}
		if !c.Date.IsZero() || c.Text != "" {
			data := l.add("DATA", "")
			e.date(data, "DATE", c.Date)
			data.text("TEXT", c.Text)
		}
		if c.Event != "" {
			kind := eventTag(c.Event, individualEvents)
			if kind == "" {
				kind = eventTag(c.Event, familyEvents)
			}
			if kind == "" {
				kind = c.Event
			}
			even := l.add("EVEN", kind)
			for i, who := range c.With {
				role, citing := e.citing(who, subject)
				if !citing {
					continue
				}
				given := even.add("ROLE", role)
				if role == "OTHER" {
					given.add("PHRASE", who.Role)
					if who.Note != "" {
						continue
					}
				} else {
					given.text("PHRASE", who.Note)
				}
				if len(who.Attributes) == 0 && who.Age == "" {
					e.cited[at][i] = true
				}
				break
			}
		}
		if ref.Grade.Weight != 0 {
			l.add("QUAY", string(ref.Grade.Weight))
		}
		if grade := (model.Grade{Source: ref.Grade.Source, Information: ref.Grade.Information, Evidence: ref.Grade.Evidence}); !grade.IsZero() {
			l.add("_GRADE", grade.String())
		}
		e.pictures(l, source, c.Images, "citations.images")
		e.prose(l, c.Note)
		e.attributes(l, c.Attributes)
		e.tags(l, c.Tags)
		e.links(l, c.Links, false)
	}
}

// citing is the role somebody an entry names is named in by the ROLE of a citation the
// record subject makes of it: the person, or a family's husband or wife.
func (e *exporter) citing(who model.Named, subject model.Slug) (string, bool) {
	if who.Person == "" || who.Person.Reserved() {
		return "", false
	}
	if _, person := e.corpus.People[subject]; person {
		if who.Person != subject {
			return "", false
		}
		if tag, ok := roleTag(who.Role); ok {
			return tag, true
		}
		return "OTHER", true
	}
	family := e.corpus.Families[subject]
	if family == nil {
		return "", false
	}
	slots, _ := e.pair(family)
	for i, role := range []string{"husband", "wife"} {
		if who.Role == role && who.Person == slots[i].Person && slots[i].Somebody() {
			return []string{"HUSB", "WIFE"}[i], true
		}
	}
	return "", false
}

// uncited counts what the citations say that no SOUR carried: an entry nobody cites,
// somebody an entry names whom no citing of it named, and the identifiers of an entry.
func (e *exporter) uncited() {
	for _, slug := range e.corpus.SourceSlugs() {
		source := e.corpus.Sources[slug]
		for _, anchor := range slices.Sorted(maps.Keys(source.Citations)) {
			c := source.Citations[anchor]
			written, ok := e.cited[string(slug)+"#"+anchor]
			if !ok {
				e.omit("citations")
				continue
			}
			for i := range c.With {
				if !written[i] {
					e.omit("citations.with")
				}
			}
			if len(c.IDs) > 0 {
				e.omit("citations.ids")
			}
		}
	}
}

// ids writes a record's identifiers, the one it goes by now at each service: an EXID
// typed with the address of a service 7.0 knows, UID, a REFN, and a REFN typed with the
// name of any other service, which the import names it by again.
func (e *exporter) ids(parent *line, ids model.IDs, at string) {
	if parent == nil {
		return
	}
	for _, service := range slices.Sorted(maps.Keys(ids)) {
		list := ids[service]
		if len(list) == 0 {
			continue
		}
		id := list[0].ID
		switch address, known := exids[service]; {
		case service == "uid":
			parent.add("UID", id)
		case service == "refn":
			parent.add("REFN", id)
		case service == "exid":
			parent.add("EXID", id)
		case known:
			parent.add("EXID", id).add("TYPE", address)
		default:
			parent.add("REFN", id).add("TYPE", service)
		}
		if len(list) > 1 {
			e.omit(at)
		}
		for _, identifier := range list {
			if identifier.Qualified() {
				e.omit(at)
			}
		}
	}
}

// links writes where something is online and how it is reached: as 7.0's WWW, EMAIL,
// PHON and FAX where the structure has them and the link says no more than where it
// leads, and as an extension with what it says of itself otherwise.
func (e *exporter) links(parent *line, links []model.Link, standard bool) {
	for _, link := range links {
		plain := link.Title == "" && link.Note == ""
		lower := strings.ToLower(link.URL)
		if standard && plain {
			number, phone := strings.CutPrefix(link.URL, "tel:")
			switch {
			case link.Type == "" && (strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")):
				parent.add("WWW", link.URL)
				continue
			case link.Type == "" && strings.HasPrefix(lower, "mailto:"):
				parent.add("EMAIL", link.URL[len("mailto:"):])
				continue
			case link.Type == "" && phone:
				parent.add("PHON", number)
				continue
			case link.Type == "fax" && phone:
				parent.add("FAX", number)
				continue
			}
		}
		l := parent.add("_LINK", link.URL)
		l.text("TYPE", link.Type)
		l.text("TITL", link.Title)
		e.prose(l, link.Note)
	}
}

// address writes an ADDR: its lines as the value, its parts as 7.0's CITY, STAE, POST and
// CTRY, the street as ADR1, the parts 7.0 has no structure for as extensions, and its
// links under it.
func (e *exporter) address(parent *line, a model.Address, at string) {
	if parent == nil || a.IsZero() {
		return
	}
	l := parent.add("ADDR", a.Lines)
	l.text("ADR1", a.Street)
	l.text("CITY", a.City)
	l.text("STAE", a.State)
	l.text("POST", a.Postal)
	l.text("CTRY", a.Country)
	l.text("_LOCALITY", a.Locality)
	l.text("_PARISH", a.Parish)
	l.text("_COUNTY", a.County)
	e.links(l, a.Links, false)
	if !a.Date.IsZero() || a.Note != "" || len(a.Sources) > 0 {
		e.omit(at)
	}
}

// date writes a date as 7.0's grammar allows, and what the corpus says that it does not
// as the PHRASE beside it: the record's own wording where the value says the rest, and
// the whole of the date as the corpus writes it where the value had to say less, or say
// it in the Gregorian calendar. A date with no day to restate, a yearless one, is an
// empty value and its PHRASE.
func (e *exporter) date(parent *line, tag string, d model.Date) {
	if parent == nil || d.IsZero() {
		return
	}
	value, clock, phrase := statement(d)
	l := parent.add(tag, value)
	l.text("TIME", clock)
	l.text("PHRASE", phrase)
}

// statement is a date as 7.0 states it: the value, restated in the Gregorian calendar
// and fitted to the grammar where it must be, the time of day apart, and the phrase
// saying what the value cannot, the date as origins writes it where the value says
// other than it, or nothing but the phrase where there is no value to restate it as.
func statement(d model.Date) (value, clock, phrase string) {
	stated := d
	stated.Wording = ""
	target := d
	if !reckoned(d) {
		restated, ok := d.InGregorian()
		if !ok {
			if stated.String() == "" {
				return "", "", d.Wording
			}
			return "", "", d.String()
		}
		target = restated
	}
	fitted := fit(target)
	phrase = d.Wording
	if fitted.String() != stated.String() {
		phrase = d.String()
	}
	if fitted.Time != nil {
		clock = fitted.Time.String()
	}
	fitted.Time = nil
	return fitted.String(), clock, phrase
}

// reckoned reports whether every end of a date is stated as 7.0 can state one, in a
// calendar it has, with a year, a day only with its month, not dual-dated and with no
// new year of its own; a date that is not is restated in the Gregorian calendar.
func reckoned(d model.Date) bool {
	if d.Period() {
		return (d.From == nil || reckoned(*d.From)) && (d.To == nil || reckoned(*d.To))
	}
	switch d.Calendar {
	case model.Gregorian, model.Julian:
	case model.Hebrew, model.FrenchRepublican:
		if d.BCE {
			return false
		}
	default:
		return false
	}
	if d.Year == nil || d.Day != nil && d.Month == nil || d.Dual || d.NewYear != nil {
		return false
	}
	return d.End == nil || reckoned(*d.End)
}

// fit makes a date one 7.0's grammar can state, saying less where it must: an estimate
// or a calculation that is also a range or an approximation is the range or the
// approximation, the ends of a period and of a range are plain dates, and a time is only
// on a day certainly stated.
func fit(d model.Date) model.Date {
	d.Wording = ""
	if d.Period() {
		out := model.Date{}
		if d.From != nil {
			from := plainEnd(*d.From, true)
			out.From = &from
		}
		if d.To != nil {
			to := plainEnd(*d.To, false)
			out.To = &to
		}
		return out
	}
	if d.Qualifier == model.Between && d.End == nil {
		d.Qualifier = model.Exact
	}
	if d.Qualifier != model.Exact {
		d.Reached = model.Stated
		d.Time = nil
	}
	if d.End != nil {
		end := plainEnd(*d.End, false)
		d.End = &end
	}
	return d
}

// plainEnd is one end of a period or a range as a plain date: a range at that end its
// start or its end, and nothing qualifying it.
func plainEnd(d model.Date, start bool) model.Date {
	if d.Qualifier == model.Between && d.End != nil && !start {
		d = *d.End
	}
	d.Qualifier, d.Reached, d.Time, d.End, d.Wording = model.Exact, model.Stated, nil, nil, ""
	return d
}

// place writes a PLAC, and under the first that ends in a place what it says of that
// place: its names in other languages, where it is and how a service knows it, and its
// body.
func (e *exporter) place(parent *line, slug model.Slug) {
	plac := e.placeChain(parent, slug)
	if plac == nil || e.mentioned[slug] {
		return
	}
	e.mentioned[slug] = true
	place := e.corpus.Places[slug]
	rest := ""
	if _, after, found := strings.Cut(plac.value, ", "); found {
		rest = ", " + after
	}
	for _, other := range place.AlsoKnownAs {
		if other.Lang == "" || other.Type != "" || !other.From.IsZero() || !other.Until.IsZero() ||
			other.Note != "" || len(other.Sources) > 0 {
			continue
		}
		plac.add("TRAN", other.Text+rest).add("LANG", string(other.Lang))
	}
	if at := place.Coordinates; at != nil {
		if at.Placed() {
			hemisphere := func(f float64, positive, negative string) string {
				if f < 0 {
					return negative + fraction(-f)
				}
				return positive + fraction(f)
			}
			m := plac.add("MAP", "")
			m.add("LATI", hemisphere(at.Latitude, "N", "S"))
			m.add("LONG", hemisphere(at.Longitude, "E", "W"))
		} else {
			plac.add("_COORDINATES", at.Wording)
		}
	}
	e.ids(plac, place.IDs, "places.ids")
	e.prose(plac, place.Body)
}

// placeChain writes a PLAC naming a place and what it lies within, each first, the most
// specific first, with the FORM of their types where every one is typed and the LANG of
// their names where all are in one.
func (e *exporter) placeChain(parent *line, slug model.Slug) *line {
	if parent == nil || slug == "" {
		return nil
	}
	var names, types []string
	langs := map[model.Language]bool{}
	seen := map[model.Slug]bool{}
	for at := slug; at != "" && !seen[at]; {
		place := e.corpus.Places[at]
		if place == nil {
			break
		}
		seen[at] = true
		names = append(names, place.Name.Text)
		types = append(types, place.Type)
		langs[place.Name.Lang] = true
		at = ""
		if len(place.Within) > 0 {
			at = place.Within[0].Place
		}
	}
	if len(names) == 0 {
		e.omit("places")
		return nil
	}
	maps.Copy(e.named, seen)
	plac := parent.add("PLAC", strings.Join(names, ", "))
	if !slices.Contains(types, "") {
		plac.add("FORM", strings.Join(types, ", "))
	}
	if len(langs) == 1 && !langs[""] {
		for lang := range langs {
			plac.add("LANG", string(lang))
		}
	}
	return plac
}

// plainAttribute reports whether an attribute says only its value, which the structure
// 7.0 has for it can say.
func plainAttribute(a model.Attribute) bool {
	return a.Note == "" && len(a.Sources) == 0 && a.Date.IsZero()
}

// attributes writes what no structure of 7.0 holds as extensions, each its value, its
// type and what is said of it.
func (e *exporter) attributes(parent *line, list []model.Attribute) {
	for _, attribute := range list {
		l := parent.add("_ATTRIBUTE", attribute.Value)
		l.add("TYPE", attribute.Type)
		e.date(l, "DATE", attribute.Date)
		e.prose(l, attribute.Note)
		e.cite(l, attribute.Sources, "")
	}
}

// mediumOf writes a medium as a MEDI: the word 7.0 enumerates where the corpus's is one,
// FILM and FICHE for microfilm and microfiche, and OTHER with its words otherwise.
func mediumOf(l *line, medium string) {
	if l == nil || medium == "" {
		return
	}
	switch word := strings.ToUpper(medium); {
	case medium == "microfilm":
		l.add("MEDI", "FILM")
	case medium == "microfiche":
		l.add("MEDI", "FICHE")
	case mediums[word] && strings.ToLower(word) == medium:
		l.add("MEDI", word)
	default:
		l.add("MEDI", "OTHER").add("PHRASE", medium)
	}
}
