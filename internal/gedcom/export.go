package gedcom

import (
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Options are what a GEDCOM export needs from outside the corpus: the version of origins
// writing it, which the header names, and the size in pixels of a picture at its path
// into the corpus, by which a region the corpus gives as fractions is written as the
// CROP 7.0 gives in pixels. Without Size, or where it cannot say, a region is written as
// an extension.
type Options struct {
	Version string
	Size    func(path string) (width, height int, ok bool)
}

// Report is what an export wrote: the records by tag, the extension tags by tag, as an
// import's counts what it kept, the lines an import kept as written and written again
// as they were, by tag, and what 7.0 and the extensions had nowhere for, by the path of
// the field it was in; and the pictures the FILE of an OBJE leads into the corpus to,
// as the path from its root that FILE escapes, in the order of the records.
type Report struct {
	Records    map[string]int
	Extensions map[string]int
	Kept       map[string]int
	Omitted    map[string]int
	Files      []string
}

// extensions is where the extension tags origins writes are documented, each under the
// tag's own words.
const extensions = "https://origins.invalid/gedcom/"

// borrowed are the extension tags origins writes that are other programs', which it
// does not document and so does not declare.
var borrowed = map[string]bool{"_FREL": true, "_MREL": true}

// Write writes the corpus as GEDCOM 7.0: the import's mapping run backwards, every
// record and structure the import reads written as the import reads it, and whatever
// 7.0 has no structure for written as an extension tag declared in the header's SCHMA.
//
// The corpus is written as it is given; a hidden view is made with model.Hide before.
// Hypotheses are not written, the corpus being the standing conclusion. Nothing the
// output says depends on when it was written, so a corpus writes the same bytes every
// time.
func Write(w io.Writer, corpus *model.Corpus, options Options) (Report, error) {
	e := &exporter{
		corpus:    corpus,
		options:   options,
		report:    Report{Records: map[string]int{}, Extensions: map[string]int{}, Kept: map[string]int{}, Omitted: map[string]int{}},
		xrefs:     map[model.Slug]string{},
		objects:   map[string]*line{},
		mentioned: map[model.Slug]bool{},
		named:     map[model.Slug]bool{},
		depicted:  map[model.Slug][]framed{},
		cited:     map[string]map[int]bool{},
	}
	e.number()
	e.keep()

	var records []*line
	for _, slug := range corpus.PeopleSlugs() {
		records = append(records, e.person(corpus.People[slug]))
	}
	for _, slug := range corpus.FamilySlugs() {
		records = append(records, e.family(corpus.Families[slug]))
	}
	for _, slug := range corpus.SourceSlugs() {
		records = append(records, e.source(corpus.Sources[slug]))
	}
	for _, slug := range corpus.RepositorySlugs() {
		records = append(records, e.repository(corpus.Repositories[slug]))
	}
	for _, slug := range corpus.NoteSlugs() {
		if note := e.note(corpus.Notes[slug]); note != nil {
			records = append(records, note)
		}
	}
	records = append(records, e.objectList...)
	records = append(records, e.submitters()...)
	e.unwritten()
	e.uncited()

	head := e.header()
	for _, record := range records {
		e.report.Records[record.tag]++
	}
	all := e.settle(append([]*line{head}, records...))
	used := map[string]bool{}
	for _, record := range all {
		record.walk(func(l *line) {
			if strings.HasPrefix(l.tag, "_") && !l.kept {
				e.report.Extensions[l.tag]++
				used[l.tag] = !borrowed[l.tag]
			}
		})
	}
	schema := &line{tag: "SCHMA"}
	for _, tag := range slices.Sorted(maps.Keys(used)) {
		if used[tag] {
			schema.add("TAG", tag+" "+extensions+strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(tag, "_"), "_", "-")))
		}
	}
	if len(schema.sub) > 0 {
		head.sub = slices.Insert(head.sub, 1, schema)
	}

	var out strings.Builder
	for _, record := range all {
		record.emit(&out, 0)
	}
	out.WriteString("0 TRLR\n")
	_, err := io.WriteString(w, out.String())
	return e.report, err
}

// exporter is one export under way.
type exporter struct {
	corpus  *model.Corpus
	options Options
	report  Report

	// xrefs is the xref of each record of a kind GEDCOM has a record for.
	xrefs map[model.Slug]string
	// objects is the OBJE record of each picture kept, by the path to it from the root
	// of the corpus, and objectList the records in the order they were made.
	objects    map[string]*line
	objectList []*line
	// mentioned is every place written as the most specific of a PLAC, which only its
	// first mention says everything of.
	mentioned map[model.Slug]bool
	// named is every place a PLAC names, at any level of it.
	named map[model.Slug]bool
	// depicted is the pictures each person is in, kept by another record.
	depicted map[model.Slug][]framed
	// files is the path into the corpus of the file each OBJE record is, by its xref.
	files map[string]string
	// cited is which of the people a citation names some citing of it wrote, by the
	// source's slug and the citation's anchor.
	cited map[string]map[int]bool
	// grafts is what an import kept as written, to be written again where it was.
	grafts []graft
}

// framed is a pointer to an OBJE to be written, and the part of it meant.
type framed struct {
	object string
	region *model.Region
}

// restricted marks something private where 7.0 has no RESN for it: a source, a repository,
// a shared note, a name and an association, which 7.0 restricts only through the record
// or the event they are in.
func restricted(l *line, private bool) {
	if private {
		l.add("_PRIVATE", "Y")
	}
}

// omit counts something nothing in the output says, by the path of the field it was in.
func (e *exporter) omit(path string) { e.report.Omitted[path]++ }

// number gives every record an xref, a letter for its kind and a number in the order
// of its slug, so that the same corpus is always written with the same xrefs.
func (e *exporter) number() {
	for _, kind := range []struct {
		letter string
		slugs  []model.Slug
	}{
		{"I", e.corpus.PeopleSlugs()}, {"F", e.corpus.FamilySlugs()}, {"S", e.corpus.SourceSlugs()},
		{"R", e.corpus.RepositorySlugs()}, {"N", e.corpus.NoteSlugs()},
	} {
		for i, slug := range kind.slugs {
			if note := e.corpus.Notes[slug]; kind.letter == "N" && onlyKept(note) {
				continue
			}
			e.xrefs[slug] = kind.letter + itoa(i+1)
		}
	}
	if n := len(e.corpus.Hypotheses); n > 0 {
		e.report.Omitted["hypotheses"] += n
	}
}

// header is the HEAD: the version written, the program writing it, the researchers'
// SUBM, the copyright and the language, and what of the manifest 7.0 has no structure
// for as extensions. Its SCHMA is added once every record is written.
func (e *exporter) header() *line {
	head := &line{tag: "HEAD"}
	head.add("GEDC", "").add("VERS", "7.0")
	sour := head.add("SOUR", "ORIGINS")
	sour.text("VERS", e.options.Version)
	sour.add("NAME", "Origins")
	project := e.corpus.Project
	if len(project.Researchers) > 0 {
		head.point("SUBM", "U1")
	}
	head.text("COPR", project.Copyright)
	head.text("LANG", project.Lang)
	head.text("_NAME", project.Name)
	head.text("_DESCRIPTION", project.Description)
	if xref, ok := e.xrefs[project.MainPerson]; ok {
		head.point("_MAIN_PERSON", xref)
	}
	for _, bookmark := range project.Bookmarks {
		switch xref, ok := e.xrefs[bookmark.Slug]; {
		case ok:
			head.point("_BOOKMARK", xref)
		case bookmark.Kind == model.KindPlace && e.corpus.Places[bookmark.Slug] != nil:
			e.placeChain(head.add("_BOOKMARK", ""), bookmark.Slug)
		default:
			e.omit("bookmarks")
		}
	}
	for _, tag := range slices.Sorted(maps.Keys(project.Tags)) {
		style := project.Tags[tag]
		styled := head.add("_TAG_STYLE", tag)
		styled.text("_COLOR", style.Color)
		styled.text("_PRIORITY", style.Priority)
	}
	return head
}

// submitters are the manifest's researchers, each a SUBM.
func (e *exporter) submitters() []*line {
	var out []*line
	for i, researcher := range e.corpus.Project.Researchers {
		subm := &line{xref: "U" + itoa(i+1), tag: "SUBM"}
		subm.text("NAME", researcher.Name)
		for j, at := range researcher.Addresses {
			if j > 0 {
				e.omit("researchers.addresses")
				continue
			}
			e.address(subm, at, "researchers.addresses")
		}
		e.links(subm, researcher.Links, true)
		e.prose(subm, researcher.Note)
		out = append(out, subm)
	}
	return out
}

// person is an INDI.
func (e *exporter) person(p *model.Person) *line {
	indi := &line{xref: e.xrefs[p.Slug], tag: "INDI"}
	if p.Private {
		indi.add("RESN", "PRIVACY")
	}
	// A TRAN is read back as the next name after the one it is under, so only the names
	// before every other NAME are written as one, and only under a name of no type, whose
	// type a TRAN would otherwise take.
	primary := e.name(indi, "NAME", "name", p.Name, p.Slug)
	under := primary != nil && p.Name.Type == ""
	for _, other := range p.AlsoKnownAs {
		if under = under && translation(other); under {
			e.name(primary, "TRAN", "also-known-as", other, p.Slug)
			continue
		}
		e.name(indi, "NAME", "also-known-as", other, p.Slug)
	}
	if p.Sex.Value != model.SexUnknown {
		indi.add("SEX", p.Sex.Value.String())
	}
	if len(p.Sex.Sources) > 0 {
		e.omit("sex.sources")
	}
	indi.text("_LIVING", p.Living.String())

	var associations []model.Event
	for _, event := range p.Events {
		if association(event) {
			associations = append(associations, event)
			continue
		}
		e.event(indi, event, "INDI", p.Slug)
	}
	e.not(indi, p.Not, individualEvents, p.Slug)
	e.childOf(indi, p.Slug)
	for _, slug := range e.corpus.FamilySlugs() {
		for _, partner := range e.corpus.Families[slug].Partners {
			if partner.Person == p.Slug {
				e.prose(indi.point("FAMS", e.xrefs[slug]), partner.Note)
			}
		}
	}
	for _, event := range associations {
		e.associate(indi, event.With[0], p.Slug, event.Private, event.Note, event.Sources)
	}
	for _, same := range p.SameAs {
		switch xref, ok := e.xrefs[same.ID]; {
		case ok:
			indi.point("ALIA", xref).text("PHRASE", same.Note)
		case same.Name != "":
			indi.void("ALIA").add("PHRASE", same.Name)
			if same.Note != "" {
				e.omit("same-as.note")
			}
		default:
			e.omit("same-as")
		}
		if len(same.Sources) > 0 {
			e.omit("same-as.sources")
		}
	}
	e.prose(indi, p.Body)
	e.cite(indi, p.Sources, p.Slug)
	e.pictures(indi, p, p.Media, "media")
	for _, shown := range e.depicted[p.Slug] {
		e.crop(indi.point("OBJE", shown.object), shown)
	}
	if p.Portrait.File != "" {
		if object, ok := e.object(p, p.Portrait.File); ok {
			e.crop(indi.point("_PORTRAIT", object), framed{object: object, region: p.Portrait.Region})
		} else {
			e.omit("portrait")
		}
	}
	e.ids(indi, p.IDs, "ids")
	e.links(indi, p.Links, false)
	e.tags(indi, p.Tags)
	return indi
}

// association reports whether an event is one an ASSO under the record says whole:
// somebody named beside the person at no event in particular, as the import reads one.
func association(event model.Event) bool {
	return event.Type == "association" && len(event.With) == 1 && event.Value == "" &&
		event.Date.IsZero() && event.Sort.IsZero() && event.Place == "" && event.Address.IsZero() &&
		event.Age.Value == "" && event.Cause.Value == "" && event.Family == "" &&
		len(event.Attributes) == 0 && len(event.Media) == 0 && len(event.Links) == 0 &&
		len(event.Tags) == 0 && len(event.IDs) == 0 && len(event.With[0].Sources) == 0
}

// childOf writes a FAMC for every family listing the person as its child, saying how
// they are its child where the relation is one to every partner, how sure the link is,
// and what the family's entry for them notes and cites.
func (e *exporter) childOf(indi *line, person model.Slug) {
	for _, slug := range e.corpus.FamilySlugs() {
		family := e.corpus.Families[slug]
		for _, child := range family.Children {
			if child.ID != person {
				continue
			}
			famc := indi.point("FAMC", e.xrefs[slug])
			if relation, uniform := e.uniform(family, child); uniform && !child.Relation.IsZero() {
				pedigreeOf(famc, relation)
			}
			famc.text("STAT", strings.ToUpper(string(child.Status)))
			e.prose(famc, child.Note)
			e.cite(famc, child.Sources, person)
		}
	}
}

// uniform is a child's relation to the family where it is one to every partner.
func (e *exporter) uniform(family *model.Family, child model.Child) (model.Relation, bool) {
	if len(child.Relation.Partners) == 0 {
		return child.Relation.To(""), true
	}
	first := child.Relation.To(family.Partners[0].Who())
	for _, partner := range family.Partners {
		if child.Relation.To(partner.Who()) != first {
			return "", false
		}
	}
	return first, true
}

// pedigreeOf writes a relation as a PEDI, OTHER with its words where 7.0 has none.
func pedigreeOf(famc *line, relation model.Relation) {
	switch relation.Kin() {
	case model.Birth:
		famc.add("PEDI", "BIRTH")
		return
	case model.Adoptive:
		famc.add("PEDI", "ADOPTED")
		return
	case model.Foster:
		famc.add("PEDI", "FOSTER")
		return
	case model.RelationUnknown:
		if model.SameWord(string(relation), "sealing") {
			famc.add("PEDI", "SEALING")
			return
		}
		if model.SameWord(string(relation), string(model.RelationUnknown)) {
			famc.add("PEDI", "OTHER")
			return
		}
	}
	famc.add("PEDI", "OTHER").add("PHRASE", string(relation))
}

// relations are the words of the _FREL and _MREL Ancestry writes under a CHIL, which the
// import reads, for a child whose relation is not the same to both partners.
var relations = map[model.Relation]string{
	model.Birth: "Natural", model.Adoptive: "Adopted", model.Step: "Step", model.Foster: "Foster",
	model.Guardian: "Guardian", model.RelationUnknown: "Unknown",
}

// pair is who a family's HUSB and WIFE name: a man the first, a woman the second, and
// anybody else in a slot left, in the order the family lists them, and whoever is left
// over. A partner who is nobody, known by no name, takes no slot.
func (e *exporter) pair(family *model.Family) ([2]model.Partner, []model.Partner) {
	var slots [2]model.Partner
	var rest, over []model.Partner
	empty := func(at int) bool { return slots[at].Person == "" && slots[at].Name.IsZero() }
	for _, partner := range family.Partners {
		at := -1
		if partner.Somebody() && e.corpus.People[partner.Person] != nil {
			switch e.corpus.People[partner.Person].Sex.Value {
			case model.Male:
				at = 0
			case model.Female:
				at = 1
			}
		}
		if at >= 0 && empty(at) {
			slots[at] = partner
			continue
		}
		rest = append(rest, partner)
	}
	for _, partner := range rest {
		switch {
		case !partner.Somebody() && partner.Name.IsZero():
		case empty(0):
			slots[0] = partner
		case empty(1):
			slots[1] = partner
		default:
			over = append(over, partner)
		}
	}
	return slots, over
}

// slots is pair, counting what no HUSB or WIFE says: a partner stated to be nobody, and
// a third.
func (e *exporter) slots(family *model.Family) [2]model.Partner {
	slots, over := e.pair(family)
	for _, partner := range family.Partners {
		if partner.Person == model.None {
			e.omit("partners.none")
		}
	}
	for range over {
		e.omit("partners")
	}
	return slots
}

// family is a FAM.
func (e *exporter) family(f *model.Family) *line {
	fam := &line{xref: e.xrefs[f.Slug], tag: "FAM"}
	if f.Private {
		fam.add("RESN", "PRIVACY")
	}
	slots := e.slots(f)
	for i, tag := range []string{"HUSB", "WIFE"} {
		switch partner := slots[i]; {
		case partner.Somebody():
			fam.point(tag, e.xrefs[partner.Person])
		case !partner.Name.IsZero():
			fam.void(tag).add("PHRASE", partner.Name.String())
			if partner.Note != "" {
				e.omit("partners.note")
			}
		}
	}
	for _, child := range f.Children {
		var chil *line
		if child.Somebody() {
			chil = fam.point("CHIL", e.xrefs[child.ID])
		} else {
			chil = fam.void("CHIL")
			chil.text("PHRASE", child.Name.String())
			if child.Note != "" || len(child.Sources) > 0 || child.Status != "" || !child.Relation.IsZero() {
				e.omit("children")
			}
			continue
		}
		if _, uniform := e.uniform(f, child); uniform {
			continue
		}
		for i, tag := range []string{"_FREL", "_MREL"} {
			if !slots[i].Somebody() {
				continue
			}
			if word, ok := relations[child.Relation.To(slots[i].Person).Kin()]; ok &&
				(child.Relation.To(slots[i].Person).Kin() != model.RelationUnknown ||
					model.SameWord(string(child.Relation.To(slots[i].Person)), "unknown")) {
				chil.add(tag, word)
			} else {
				e.omit("children.relation")
			}
		}
	}
	for _, event := range f.Events {
		e.event(fam, event, "FAM", f.Slug)
	}
	e.not(fam, f.Not, familyEvents, f.Slug)
	fam.text("_TYPE", f.Type)
	e.prose(fam, f.Body)
	e.cite(fam, f.Sources, f.Slug)
	e.pictures(fam, f, f.Media, "media")
	e.ids(fam, f.IDs, "ids")
	e.links(fam, f.Links, false)
	e.tags(fam, f.Tags)
	e.sameAs(fam, f.SameAs)
	return fam
}

// source is a SOUR record: what it is, what it covers and where it is kept, its text
// and its body. Its citations are written where each is cited.
func (e *exporter) source(s *model.Source) *line {
	sour := &line{xref: e.xrefs[s.Slug], tag: "SOUR"}
	if s.Private {
		sour.add("_PRIVATE", "Y")
	}
	var data *line
	dataOf := func() *line {
		if data == nil {
			data = sour.add("DATA", "")
		}
		return data
	}
	for _, covers := range s.Covers {
		var tags []string
		for _, word := range covers.Events {
			if tag := eventTag(word, individualEvents); tag != "" {
				tags = append(tags, tag)
			} else if tag := eventTag(word, familyEvents); tag != "" {
				tags = append(tags, tag)
			} else {
				tags = append(tags, word)
			}
		}
		even := dataOf().add("EVEN", strings.Join(tags, ", "))
		e.date(even, "DATE", covers.Date)
		e.place(even, covers.Place)
		if covers.Note != "" {
			e.omit("covers.note")
		}
	}
	var attributes []model.Attribute
	agency := false
	for _, attribute := range s.Attributes {
		if attribute.Type == "agency" && !agency && plainAttribute(attribute) {
			dataOf().add("AGNC", attribute.Value)
			agency = true
			continue
		}
		attributes = append(attributes, attribute)
	}
	sour.text("AUTH", s.Author)
	sour.text("TITL", s.Title)
	sour.text("ABBR", s.ShortTitle)
	sour.text("PUBL", s.Publication)
	for _, held := range s.Repositories {
		var repo *line
		if xref, ok := e.xrefs[held.Repository]; ok {
			repo = sour.point("REPO", xref)
		} else {
			repo = sour.void("REPO")
		}
		e.prose(repo, held.Note)
		if held.CallNumber == "" {
			if held.Medium != "" {
				e.omit("repositories.medium")
			}
			continue
		}
		mediumOf(repo.add("CALN", held.CallNumber), held.Medium)
	}
	sour.text("TEXT", s.Text)
	sour.text("_TYPE", s.Type)
	e.prose(sour, s.Body)
	e.pictures(sour, s, s.Media, "media")
	e.ids(sour, s.IDs, "ids")
	e.links(sour, s.Links, false)
	e.tags(sour, s.Tags)
	e.sameAs(sour, s.SameAs)
	e.attributes(sour, attributes)
	return sour
}

// repository is a REPO.
func (e *exporter) repository(r *model.Repository) *line {
	repo := &line{xref: e.xrefs[r.Slug], tag: "REPO"}
	if r.Private {
		repo.add("_PRIVATE", "Y")
	}
	repo.add("NAME", r.Name)
	for i, at := range r.Addresses {
		if i > 0 {
			e.omit("repositories.addresses")
			continue
		}
		e.address(repo, at, "addresses")
	}
	e.links(repo, r.Links, true)
	e.ids(repo, r.IDs, "ids")
	e.prose(repo, r.Body)
	repo.text("_TYPE", r.Type)
	e.tags(repo, r.Tags)
	e.sameAs(repo, r.SameAs)
	if len(r.Media) > 0 {
		e.omit("repositories.media")
	}
	return repo
}

// note is an SNOTE: the note's title and body as its text, and the events it holds as
// extensions, naming whoever they name. A note that is only records an import kept, as
// the import makes one of a record nothing read, is those records and no SNOTE.
func (e *exporter) note(n *model.Note) *line {
	if _, ok := e.xrefs[n.Slug]; !ok {
		e.unkept(nil, n.Body)
		return nil
	}
	snote := &line{xref: e.xrefs[n.Slug], tag: "SNOTE"}
	snote.value = e.unkept(snote, n.Body)
	if n.Title != "" {
		snote.value = strings.TrimSpace(n.Title + "\n\n" + snote.value)
	}
	if n.Private {
		snote.add("_PRIVATE", "Y")
	}
	e.cite(snote, n.Sources, n.Slug)
	e.ids(snote, n.IDs, "ids")
	for _, event := range n.Events {
		e.event(snote, event, "SNOTE", n.Slug)
	}
	e.links(snote, n.Links, false)
	e.tags(snote, n.Tags)
	e.sameAs(snote, n.SameAs)
	if len(n.Media) > 0 {
		e.omit("notes.media")
	}
	return snote
}

// sameAs writes the records of a kind other than a person's said to be the same as
// this one, which 7.0 has ALIA for only between individuals.
func (e *exporter) sameAs(record *line, list []model.SameAs) {
	for _, same := range list {
		var l *line
		switch xref, ok := e.xrefs[same.ID]; {
		case ok:
			l = record.point("_SAME_AS", xref)
		case same.Name != "":
			l = record.add("_SAME_AS", same.Name)
		case same.ID != "":
			l = record.add("_SAME_AS", string(same.ID))
		default:
			continue
		}
		e.prose(l, same.Note)
		if len(same.Sources) > 0 {
			e.omit("same-as.sources")
		}
	}
}

// tags writes the researcher's words.
func (e *exporter) tags(record *line, tags []string) {
	for _, tag := range tags {
		record.add("_TAG", tag)
	}
}

// unwritten counts what the places say that no PLAC carried: a place is written only
// as the words of the chain it ends, and what a mention says of the most specific place
// in it once, and a place no chain names is not written at all.
func (e *exporter) unwritten() {
	for _, slug := range e.corpus.PlaceSlugs() {
		place := e.corpus.Places[slug]
		if !e.named[slug] {
			e.omit("places")
			continue
		}
		count := func(field string, says bool) {
			if says {
				e.omit("places." + field)
			}
		}
		if !e.mentioned[slug] {
			count("also-known-as", len(place.AlsoKnownAs) > 0)
			count("coordinates", place.Coordinates != nil)
			count("ids", len(place.IDs) > 0)
			count("body", place.Body != "")
		} else {
			for _, other := range place.AlsoKnownAs {
				count("also-known-as", other.Lang == "" || other.Type != "" || !other.From.IsZero() ||
					!other.Until.IsZero() || other.Note != "" || len(other.Sources) > 0)
			}
		}
		name := place.Name
		count("name", name.Type != "" || !name.From.IsZero() || !name.Until.IsZero() || name.Note != "" || len(name.Sources) > 0)
		count("within", len(place.Within) > 1)
		for _, within := range place.Within {
			count("within", !within.From.IsZero() || !within.Until.IsZero() || within.Note != "" || len(within.Sources) > 0)
		}
		count("private", place.Private)
		count("addresses", len(place.Addresses) > 0)
		count("from", !place.From.IsZero())
		count("until", !place.Until.IsZero())
		count("media", len(place.Media) > 0)
		count("links", len(place.Links) > 0)
		count("sources", len(place.Sources) > 0)
		count("tags", len(place.Tags) > 0)
		count("same-as", len(place.SameAs) > 0)
	}
}

// eventTag is GEDCOM's tag for an event of type in kinds, or "" where it has none of its
// own: the extension tags in the tables are other programs', which origins reads and
// does not write.
func eventTag(kind string, kinds map[string]string) string {
	for _, tag := range slices.Sorted(maps.Keys(kinds)) {
		if !strings.HasPrefix(tag, "_") && model.SameWord(kinds[tag], kind) {
			return tag
		}
	}
	return ""
}
