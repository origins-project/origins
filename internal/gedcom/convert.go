package gedcom

import (
	"fmt"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// Convert maps parsed GEDCOM records onto the model, and writes every line it did not
// read into the corpus as it was written, counting each in the report.
//
// The shape of the two formats lines up better than it has any right to: INDI is a
// person, FAM is a family, and a FAM's CHIL list is how children come to declare their
// parents. Where they differ, they differ in the model's favour — a place stated at
// every event becomes one place entity, a date phrase becomes a real yearless date, and
// a citation becomes an anchor in the source rather than a copy of the transcription.
//
// A picture comes in where find says the file it names is to hand, as an attachment for
// the caller to copy beside the record it belongs to; a nil find finds nothing.
//
// Order is file order throughout. Nothing here iterates a map to produce output, so the
// same file converts to the same corpus every time.
func Convert(records []*Node, choose Chooser, find Finder) (*model.Corpus, []Attachment, []string) {
	if find == nil {
		find = func(string) (string, bool) { return "", false }
	}
	corpus := model.NewCorpus()
	corpus.Project.FormatVersion = model.FormatVersion

	namer := importing.NewNamer(corpus, choose)
	notes := importing.NewNotes()
	c := &converter{
		corpus:           corpus,
		namer:            namer,
		places:           newPlaces(corpus, namer, notes),
		notes:            notes,
		xref:             map[string]*Node{},
		person:           map[string]model.Slug{},
		source:           map[string]model.Slug{},
		inline:           map[string]model.Slug{},
		repository:       map[string]model.Slug{},
		inlineRepository: map[string]model.Slug{},
		find:             find,
		pictures:         map[string]*kept{},
		waiting:          map[string][]waiting{},
		files:            importing.NewAttacher(),
		kept:             importing.NewKeeper[*Node]("gedcom"),
		famc:             map[string][]back{},
		fams:             map[string][]back{},
		family:           map[string]model.Slug{},
		fam:              map[model.Slug]*Node{},
		note:             map[string]model.Slug{},
		referred:         map[string]bool{},
		submitter:        map[string]bool{},
		individual:       map[model.Slug]bool{},
		reading:          map[*Node]bool{},
		ours:             schema(records),
		mine:             writtenByOrigins(records),
	}

	Describe(records, "")
	c.places.form = headerForm(records)
	c.readVersion(records)
	c.index(records)
	c.nameNotes(records)
	c.nameSources(records)
	c.readSubmitters(records)
	c.readHeader(records)
	c.readRepositories(records)
	c.readSources(records)
	c.readNotes(records)
	c.nameEveryone(records)
	c.readPeople(records)
	c.readFamilies(records)
	c.tie()
	c.unwaited()
	for _, read := range c.later {
		read()
	}
	c.unreferred(records)
	dropped(records)
	c.kept.Keep(corpus, namer, c.notes, records, title)
	return corpus, c.files.List(), c.notes.List()
}

type converter struct {
	corpus *model.Corpus
	namer  *importing.Namer
	places *places
	notes  *importing.Notes

	xref       map[string]*Node      // every record that has an identifier
	person     map[string]model.Slug // INDI identifier to slug, assigned before anything is built
	individual map[model.Slug]bool   // the slugs person hands out
	source     map[string]model.Slug // SOUR identifier to slug
	inline     map[string]model.Slug // the title of a source written in place, to its slug

	repository       map[string]model.Slug // REPO identifier to slug
	inlineRepository map[string]model.Slug // the name of a repository written in place, to its slug

	find     Finder
	pictures map[string]*kept     // OBJE identifier to the record keeping it
	waiting  map[string][]waiting // OBJE identifier to the records naming it before its keeper
	files    *importing.Attacher

	kept *importing.Keeper[*Node] // where what is not read under a line is to go

	famc   map[string][]back     // FAM identifier to the FAMC links naming it, in file order
	fams   map[string][]back     // FAM identifier to the FAMS links naming it
	family map[string]model.Slug // FAM identifier to the family made of it
	fam    map[model.Slug]*Node  // a family to the FAM it was made of
	ties   []tie                 // events whose FAMC names a family, read once every family is

	note      map[string]model.Slug // a shared note's identifier to the note record made of it
	referred  map[string]bool       // OBJE identifiers something points at
	submitter map[string]bool       // SUBM identifiers read as researchers
	reading   map[*Node]bool        // the pictures being read, which a cycle would read again

	// citer is the record being read, whose notes' citations are its own.
	citer model.Slug

	ours  map[string]bool // the extension tags that are origins', as the header declares them
	mine  bool            // whether origins wrote the file
	seven bool            // whether the file declares 7.0, whose FILE is a URI reference
	later []func()        // what is read once every record has its slug
}

// back is a link from an individual to a family: the individual's identifier, and the
// FAMC or FAMS line itself.
type back struct {
	xref string
	link *Node
}

func (c *converter) index(records []*Node) {
	for _, record := range records {
		if record.Xref != "" {
			c.xref[record.Xref] = record
		}
		if record.Tag != "INDI" || record.Xref == "" {
			continue
		}
		for _, link := range record.Children {
			switch xref := pointer(link.Value); {
			case xref == "":
			case link.Tag == "FAMC":
				c.famc[xref] = append(c.famc[xref], back{record.Xref, link})
			case link.Tag == "FAMS":
				c.fams[xref] = append(c.fams[xref], back{record.Xref, link})
			}
		}
	}
}

// home names the prose at path on record as where what is not read under node goes,
// where there is such prose.
func (c *converter) home(node *Node, record model.Slug, path string) {
	if path != "" {
		c.kept.Home(node, record, path)
	}
}

// dropped marks what is left out knowingly: when another program last saved a record,
// which stops being true at the first change and which the history of the corpus
// replaces, a restriction on changing a record, which says nothing about who may see
// it, and the line that ends the file.
func dropped(nodes []*Node) {
	for _, node := range nodes {
		switch {
		case node.Tag == "CHAN" || node.Tag == "CREA" || node.Tag == "_UPD" || node.Tag == "TRLR":
			node.markAll()
			continue
		case node.Tag == "RESN" && locked(node):
			node.Mark()
		}
		dropped(node.Children)
	}
}

// title is what a note record keeping a record of the export nothing else read is
// called.
func title(record *Node) string {
	if record.Tag == "HEAD" {
		return "The header of the export"
	}
	if record.Xref == "" {
		return record.Tag
	}
	return record.Tag + " @" + record.Xref + "@"
}

// nameEveryone assigns every person's slug before any entity is built, because a family
// is named after its partners and a parent is referred to by slug. Doing it in one pass
// first means nothing is ever referred to by a name it has not been given yet.
func (c *converter) nameEveryone(records []*Node) {
	for _, record := range records {
		if record.Tag != "INDI" {
			continue
		}
		name := c.primaryName(record)
		base := c.baseSlug(name, record.Xref)
		if base == "" {
			c.notes.Add("an individual had no name and no usable identifier, and was not imported")
			continue
		}
		slug := c.namer.Assign(base, c.discriminators(record), displayName(name, record.Xref))
		if slug == "" {
			c.notes.Add("no free slug could be found for %q", name)
			continue
		}
		c.person[record.Xref] = slug
		c.individual[slug] = true
	}
}

// baseSlug is the slug a person would have if nobody else wanted it. A nameless
// individual is filed under the identifier the export gave them, because something has
// to name the file and an invented name would be worse than an ugly one.
func (c *converter) baseSlug(name model.PersonName, xref string) model.Slug {
	if slug, err := name.Slug(); err == nil {
		return slug
	}
	if slug, err := model.Slugify(xref); err == nil {
		if name.IsZero() {
			c.notes.Add("an individual had no name and is filed under their identifier")
		} else {
			c.notes.Add("an individual had no name a slug can spell and is filed under their identifier")
		}
		return slug
	}
	return ""
}

// discriminators are the facts worth putting in a slug to tell two people of the same
// name apart, best first: when they were born, where, and when they died.
func (c *converter) discriminators(record *Node) []string {
	var hints []string
	birth := record.First("BIRT")
	if date, _ := readDate(birth.First("DATE").Text()); date.Year != nil {
		hints = append(hints, itoa(*date.Year))
	}
	if levels := components(birth.First("PLAC").Text()); len(levels) > 0 {
		hints = append(hints, levels[0])
	}
	if date, _ := readDate(record.First("DEAT").First("DATE").Text()); date.Year != nil {
		hints = append(hints, itoa(*date.Year))
	}
	return hints
}

func (c *converter) readPeople(records []*Node) {
	for _, record := range records {
		slug, ok := c.person[record.Xref]
		if record.Tag != "INDI" || !ok {
			continue
		}
		c.corpus.People[slug] = c.readPerson(record, slug)
	}
}

func (c *converter) readPerson(record *Node, slug model.Slug) *model.Person {
	record.Mark()
	c.home(record, slug, "body")
	c.citer = slug
	person := &model.Person{Slug: slug, Private: c.private(record)}
	for i, node := range record.TakeAll("NAME") {
		c.names(person, node, slug, i == 0)
	}
	c.aliases(person, record, slug)
	if person.Name.IsZero() {
		person.Name = model.PersonName{Text: string(slug)}
	}
	if len(person.Name.Surname) == 0 {
		c.notes.Add("a name gave no surname, so the person is filed under the whole of it")
	}

	sex := record.First("SEX")
	if person.Sex.Value = readSex(sex.Text()); person.Sex.Value != model.SexUnknown ||
		strings.EqualFold(strings.TrimSpace(sex.Text()), "U") {
		sex.Mark()
	}
	person.Events = c.readEvents(record, individualEvents, person)
	person.Events = append(person.Events, c.associations(record, slug, len(person.Events))...)
	person.Not = c.readNot(record, individualEvents, slug)
	person.Links = c.links(record)
	person.Body = strings.Join(append(c.noteTexts(record), c.research(record)...), "\n\n")
	person.Sources = c.citations(record, slug, "person")
	person.IDs = c.ids(record)
	person.Tags = c.tagged(record)
	if living := record.First("_LIVING"); c.ours["_LIVING"] && living != nil {
		if read, err := model.ParseLiving(collapse(living.Text())); err == nil {
			living.Mark()
			person.Living = read
		}
	}
	c.readMedia(record, person)
	c.portrait(record, person)
	return person
}

func (c *converter) readFamilies(records []*Node) {
	for _, record := range records {
		if record.Tag != "FAM" {
			continue
		}
		c.readFamily(record)
	}
}

// readFamily turns one FAM into a family of the model's, with the children it lists.
// A FAM that is not a couple is a family too, `unknown` standing for each partner it
// does not give, so that a single mother's children and the family's own events, notes
// and citations have somewhere to be. Two FAMs of one couple are two families, the
// second told apart as any collision is: a couple who married twice and one marriage
// recorded twice look the same from the names.
func (c *converter) readFamily(record *Node) {
	partners := c.partners(record)
	base, err := model.FamilySlug(partners)
	if err != nil {
		return
	}
	names := make([]string, len(partners))
	for i, partner := range partners {
		switch person := c.corpus.People[partner.Person]; {
		case person != nil:
			names[i] = displayName(person.Name, string(partner.Person))
		case !partner.Name.IsZero():
			names[i] = partner.Name.String()
		default:
			names[i] = string(partner.Who())
		}
	}
	slug := c.namer.Assign(base, c.familyDiscriminators(record), strings.Join(names, " and "))
	if slug == "" {
		return
	}
	c.family[record.Xref] = slug
	c.fam[slug] = record
	c.citer = slug
	record.Mark()
	c.home(record, slug, "body")
	c.joined(record, partners, slug)
	family := &model.Family{Slug: slug, Private: c.private(record), Partners: partners}
	family.Children = c.readChildren(record, partners, slug)
	family.Events = c.readEvents(record, familyEvents, family)
	family.Not = c.readNot(record, familyEvents, slug)
	family.Links = c.links(record)
	family.Body = strings.Join(c.noteTexts(record), "\n\n")
	family.Sources = c.citations(record, slug, "family")
	family.IDs = c.ids(record)
	family.Type = c.typed(record)
	family.Tags = c.tagged(record)
	c.afterwards(func() { family.SameAs = c.sameAs(record) })
	family.Events = append(family.Events, c.associations(record, slug, len(family.Events))...)
	c.corpus.Families[slug] = family
	c.readMedia(record, family)
}

// familyDiscriminators are what tell a FAM apart from another of the same couple, best
// first, as model.FamilyHints has them: the year of its first dated event, where that
// happened, and the surname of its first child.
func (c *converter) familyDiscriminators(record *Node) []string {
	var hints []string
	for _, node := range record.Children {
		if _, ok := familyEvents[node.Tag]; !ok && node.Tag != "EVEN" && node.Tag != "FACT" {
			continue
		}
		if date, _ := readDate(node.First("DATE").Text()); date.Year != nil {
			hints = append(hints, itoa(*date.Year))
			if levels := components(node.First("PLAC").Text()); len(levels) > 0 {
				hints = append(hints, levels[0])
			}
			break
		}
	}
	if child := c.corpus.People[c.person[pointer(record.First("CHIL").Text())]]; child != nil && child.Name.PrimarySurname().Name != "" {
		hints = append(hints, child.Name.PrimarySurname().Name)
	}
	return hints
}

// partners are the people a FAM joins, whichever side of the link names them: the
// family's HUSB and WIFE, and the FAMS of anybody naming the family back, a damaged file
// carrying one without the other. They are sorted, so that the family's name does not
// depend on which of them the exporter wrote first, and somebody 7.0 points at @VOID@
// follows them, by the words of the PHRASE where there is one, and `unknown` stands for
// whoever else a couple is short of.
func (c *converter) partners(record *Node) []model.Partner {
	var people []model.Slug
	add := func(xref string) {
		if slug, ok := c.person[xref]; ok && !slices.Contains(people, slug) {
			people = append(people, slug)
		}
	}
	for _, node := range spouses(record) {
		add(pointer(node.Value))
	}
	for _, link := range c.fams[record.Xref] {
		add(link.xref)
	}
	slices.Sort(people)
	partners := model.PartnersOf(people...)
	for _, node := range spouses(record) {
		if !voided(node.Value) {
			continue
		}
		partner := model.Partner{Person: model.Unknown}
		if words := named(node.First("PHRASE").Text()); words != "" {
			partner = model.Partner{Name: model.PersonName{Text: words}}
		}
		partners = append(partners, partner)
	}
	for len(partners) < 2 {
		partners = append(partners, model.Partner{Person: model.Unknown})
	}
	return partners
}

// spouses are a FAM's HUSB and WIFE lines, in file order.
func spouses(record *Node) []*Node {
	var out []*Node
	for _, node := range record.Children {
		if node.Tag == "HUSB" || node.Tag == "WIFE" {
			out = append(out, node)
		}
	}
	return out
}

// joined reads what the lines saying who a family joins say of each partner's link to
// it: a PHRASE under a HUSB or WIFE pointing at somebody, which is the file's words for
// the link, and the notes on each FAMS naming the family back, each a paragraph of the
// partner's note. A PHRASE under @VOID@ was the partner's name.
func (c *converter) joined(record *Node, partners []model.Partner, slug model.Slug) {
	notes := make([][]string, len(partners))
	link := func(node *Node, i int) {
		node.Mark()
		c.home(node, slug, fmt.Sprintf("partners[%d].note", i))
	}
	void := slices.IndexFunc(partners, func(p model.Partner) bool { return !p.Somebody() })
	for _, node := range spouses(record) {
		if voided(node.Value) {
			node.Take("PHRASE")
			link(node, void)
			void++
			continue
		}
		i := slices.IndexFunc(partners, func(p model.Partner) bool { return p.Person == c.person[pointer(node.Value)] })
		if i < 0 {
			continue
		}
		link(node, i)
		if words := collapse(node.First("PHRASE").Text()); words != "" {
			node.Take("PHRASE")
			notes[i] = append(notes[i], words)
		}
	}
	for _, back := range c.fams[record.Xref] {
		i := slices.IndexFunc(partners, func(p model.Partner) bool { return p.Person == c.person[back.xref] })
		if i < 0 {
			continue
		}
		link(back.link, i)
		if words := collapse(back.link.First("PHRASE").Text()); words != "" {
			back.link.Take("PHRASE")
			notes[i] = append(notes[i], words)
		}
		notes[i] = append(notes[i], c.noteTexts(back.link)...)
	}
	for i := range partners {
		partners[i].Note = strings.Join(notes[i], "\n\n")
	}
}

// readChildren lists a FAM's children in the order the file gives them: those its CHIL
// lines name, then anybody whose FAMC names the family without the family naming them,
// since either side of the link says whose child somebody is. A child named twice is
// listed once, and a CHIL pointing @VOID@ is a child nobody has a record of, known by
// the PHRASE's words where there are any.
func (c *converter) readChildren(record *Node, partners []model.Partner, slug model.Slug) []model.Child {
	var children []model.Child
	var notes [][]string
	listed := func(person model.Slug) int {
		return slices.IndexFunc(children, func(child model.Child) bool { return child.ID == person })
	}
	for _, node := range record.All("CHIL") {
		at := len(children)
		var phrase string
		if words := collapse(node.First("PHRASE").Text()); words != "" {
			phrase = words
		}
		switch person, ok := c.person[pointer(node.Value)]; {
		case voided(node.Value):
			child := model.Child{ID: model.Unknown}
			if phrase != "" {
				child, phrase = model.Child{Name: model.PersonName{Text: phrase}}, ""
			}
			children, notes = append(children, child), append(notes, nil)
		case !ok:
			continue
		case listed(person) >= 0:
			at = listed(person)
		default:
			child, note := c.child(record, node, pointer(node.Value), person, partners, slug, at)
			children, notes = append(children, child), append(notes, note)
		}
		node.Mark().Take("PHRASE")
		c.home(node, slug, fmt.Sprintf("children[%d].note", at))
		if phrase != "" {
			notes[at] = append([]string{phrase}, notes[at]...)
		}
	}
	for _, back := range c.famc[record.Xref] {
		person, ok := c.person[back.xref]
		if !ok || listed(person) >= 0 {
			continue
		}
		child, note := c.child(record, nil, back.xref, person, partners, slug, len(children))
		children, notes = append(children, child), append(notes, note)
	}
	for i := range children {
		children[i].Note = strings.Join(notes[i], "\n\n")
	}
	return children
}

// link is the first FAMC on the individual xref naming the family family.
func (c *converter) link(xref, family string) *Node {
	for _, back := range c.famc[family] {
		if back.xref == xref {
			return back.link
		}
	}
	return nil
}

// child reads one child of record from the child's own link back to the family, which
// is where GEDCOM says how they are its child, how sure that is, and what states it:
// STAT is the status, and the PHRASE under it, the PHRASE under a PEDI that names a
// relation and each NOTE are paragraphs of the entry's note, handed back to be led by
// whatever the CHIL itself says.
func (c *converter) child(record, chil *Node, xref string, person model.Slug, partners []model.Partner, family model.Slug, at int) (model.Child, []string) {
	famc := c.link(xref, record.Xref)
	if famc != nil {
		famc.Mark()
		c.home(famc, family, fmt.Sprintf("children[%d].note", at))
	}
	child := model.Child{ID: person, Sources: c.citations(famc, person, "parents")}
	var note []string
	child.Relation, note = c.relationship(record, chil, famc, c.xref[xref], partners)
	stat := famc.First("STAT")
	if status, err := model.ParseLinkStatus(strings.ToLower(collapse(stat.Text()))); err == nil && status != "" {
		child.Status = status
		if words := collapse(stat.Mark().First("PHRASE").Text()); words != "" {
			stat.Take("PHRASE")
			note = append(note, words)
		}
	}
	return child, append(note, c.noteTexts(famc)...)
}

// relationship is a child's relation to each partner, written as one word where it is
// the same for all of them and left out where that word is birth, with the words of a
// PHRASE refining a relation origins knows, for the child's note.
func (c *converter) relationship(record, chil, famc, person *Node, partners []model.Partner) (model.Relationship, []string) {
	pedi, words := pedigree(famc)
	adopted := adopters(person, record)
	each := make([]model.PartnerRelation, len(partners))
	same, used := true, false
	for i, partner := range partners {
		relation, fromPedi := c.relation(record, chil, partner.Who(), pedi, adopted)
		used = used || fromPedi
		each[i] = model.PartnerRelation{Partner: partner.Who(), Relation: relation}
		same = same && each[i].Relation == each[0].Relation
	}
	var note []string
	if used {
		famc.Take("PEDI").Take("PHRASE")
		if words != "" {
			note = append(note, words)
		}
	}
	switch {
	case !same:
		return model.Relationship{Partners: each}, note
	case each[0].Relation == model.Birth:
		return model.Relationship{}, note
	}
	return model.Relationship{Word: each[0].Relation}, note
}

// pedigree is the relation a FAMC's PEDI gives, or "" where it gives none origins can
// read, and the words of a PHRASE under one it can that are not the relation itself.
// 7.0's OTHER is the relation the PHRASE names, as a name's TYPE OTHER is.
func pedigree(famc *Node) (model.Relation, string) {
	pedi := famc.First("PEDI")
	words := collapse(pedi.First("PHRASE").Text())
	if strings.EqualFold(strings.TrimSpace(pedi.Text()), "OTHER") && words != "" {
		return model.Relation(words), ""
	}
	if relation, ok := namedRelation(pedi.Text()); ok {
		return relation, words
	}
	return "", ""
}

// adopters are the lines saying who adopted the individual person into the family
// record, as the FAMC under an adoption's ADOP names them: HUSB, WIFE, or both.
func adopters(person, record *Node) map[string]*Node {
	for _, adoption := range person.All("ADOP") {
		for _, famc := range adoption.All("FAMC") {
			if pointer(famc.Value) != record.Xref {
				continue
			}
			by := famc.First("ADOP")
			switch strings.ToUpper(strings.TrimSpace(by.Text())) {
			case "HUSB":
				return map[string]*Node{"HUSB": by}
			case "WIFE":
				return map[string]*Node{"WIFE": by}
			case "BOTH":
				return map[string]*Node{"HUSB": by, "WIFE": by}
			}
		}
	}
	return nil
}

// relation reads how a parent stands to a child, reporting whether PEDI said it. GEDCOM
// states it in four places and no exporter uses all of them: an adoption naming who
// adopted, the _FREL and _MREL extensions that Ancestry writes under CHIL for whoever
// the family's HUSB and WIFE name, and PEDI under the child's own FAMC link, which says
// the same thing about both parents at once. The first three are per-parent and so are
// preferred where they exist, and a PEDI with no word origins has is kept as written.
func (c *converter) relation(record, chil *Node, parent model.Slug, pedi model.Relation, adopted map[string]*Node) (model.Relation, bool) {
	for _, slot := range [][2]string{{"HUSB", "_FREL"}, {"WIFE", "_MREL"}} {
		if c.person[pointer(record.First(slot[0]).Text())] != parent {
			continue
		}
		if by := adopted[slot[0]]; by != nil {
			by.Mark()
			return model.Adoptive, false
		}
		if r, ok := namedRelation(chil.First(slot[1]).Text()); ok {
			chil.Take(slot[1])
			return r, false
		}
	}
	if pedi != "" {
		return pedi, true
	}
	return model.Birth, false
}

// namedRelation maps the words GEDCOM uses onto the model's relations. A sealing is the
// Latter-day Saints' rite binding a child to parents, which is what the relation says,
// in the word for it. 7.0's OTHER makes somebody a parent without saying in what sense,
// which is the model's unknown.
func namedRelation(text string) (model.Relation, bool) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "adopted", "adoptive":
		return model.Adoptive, true
	case "foster":
		return model.Foster, true
	case "step":
		return model.Step, true
	case "guardian":
		return model.Guardian, true
	case "birth", "natural":
		return model.Birth, true
	case "sealing":
		return "sealing", true
	case "other", "unknown":
		return model.RelationUnknown, true
	}
	return model.Birth, false
}

// tie is an individual's event whose FAMC names the family it concerns: the one a
// birth or a christening was into, the one an adoption was into, and the one a sealing
// to parents sealed the individual to.
type tie struct {
	person model.Slug
	event  int
	famc   *Node
	tag    string
}

// tie reads the FAMC under each event that has one, once every family is made. A
// birth's or a christening's names the family the child was born into, which lists them
// with birth wherever the file agrees with itself, and is the event's family only where
// it does not, the file having said it; an adoption's and a sealing's are always the
// event's family.
func (c *converter) tie() {
	for _, tie := range c.ties {
		slug, ok := c.family[pointer(tie.famc.Value)]
		if !ok {
			continue
		}
		tie.famc.Mark()
		event := &c.corpus.People[tie.person].Events[tie.event]
		if tie.tag == "BIRT" || tie.tag == "CHR" {
			family := c.corpus.Families[slug]
			i := slices.IndexFunc(family.Children, func(child model.Child) bool { return child.ID == tie.person })
			if i >= 0 && family.Children[i].Status != model.Disproven && born(family, family.Children[i]) {
				continue
			}
		}
		event.Family = slug
	}
}

// born reports a child listed as born to every partner of the family.
func born(family *model.Family, child model.Child) bool {
	for _, partner := range family.Partners {
		if child.Relation.To(partner.Who()) != model.Birth {
			return false
		}
	}
	return true
}

func readSex(text string) model.Sex {
	switch strings.ToUpper(strings.TrimSpace(text)) {
	case "M":
		return model.Male
	case "F":
		return model.Female
	case "X":
		return model.Neither
	}
	return model.SexUnknown
}

// pointer strips the at-signs from a cross-reference used as a value.
// readVersion checks the version a file declares against the two origins reads.
//
// 5.5.1 and 7.0 agree on the line format and on most of what a record means, and differ
// in a handful of places that are handled where they arise: a date phrase, a shared
// note, a calendar, a pointer to nothing. A version that is neither is read as though it
// were, which mostly works and is worth saying out loud.
func (c *converter) readVersion(records []*Node) {
	for _, record := range records {
		if record.Tag != "HEAD" {
			continue
		}
		version := strings.TrimSpace(record.First("GEDC").First("VERS").Text())
		c.seven = strings.HasPrefix(version, "7.")
		if version == "" || strings.HasPrefix(version, "5.5") || strings.HasPrefix(version, "7.") {
			return
		}
		c.notes.Add("this file declares GEDCOM %s, which origins does not know: "+
			"it reads 5.5.1 and 7.0, and read this one as though it were one of them", version)
		return
	}
}

// Describe says where an export came from, as its header gives it, for the commit that
// records the import: the program that wrote it and its version, when, the file it was
// written as, the GEDCOM version and character set, and the extension tags it declares.
// It marks what it reads of the header, and file is the name the export was read under.
func Describe(records []*Node, file string) string {
	var head *Node
	for _, record := range records {
		if record.Tag == "HEAD" {
			head = record
			break
		}
	}
	read := "Read from an export"
	if file != "" {
		read = "Read from " + file
	}
	if head == nil {
		return read + "."
	}
	head.Mark()

	sour := head.Take("SOUR")
	id, program := collapse(sour.Text()), collapse(sour.Take("NAME").Text())
	switch {
	case program == "":
		program = id
	case id != "" && !strings.EqualFold(id, program):
		program += " (" + id + ")"
	}
	if version := collapse(sour.Take("VERS").Text()); version != "" && program != "" {
		program += " " + version
	}
	date := head.Take("DATE")
	when := collapse(date.Text())
	if time := collapse(date.Take("TIME").Text()); time != "" && when != "" {
		when += " " + time
	}
	gedc := head.Take("GEDC")
	version := collapse(gedc.Take("VERS").Text())
	if form := gedc.First("FORM"); strings.EqualFold(collapse(form.Text()), "LINEAGE-LINKED") {
		form.Mark()
	}
	charset := collapse(head.Take("CHAR").Text())
	written := collapse(head.Take("FILE").Text())

	var how []string
	if when != "" {
		how = append(how, "on "+when)
	}
	if version != "" {
		how = append(how, "as GEDCOM "+version)
	}
	if charset != "" {
		how = append(how, "in "+charset)
	}
	var b strings.Builder
	b.WriteString(read)
	switch {
	case program != "":
		b.WriteString(", which " + program + " exported")
	case len(how) > 0:
		b.WriteString(", which was exported")
	}
	for _, part := range how {
		b.WriteString(" " + part)
	}
	b.WriteString(".")
	if written != "" && written != file {
		fmt.Fprintf(&b, " The export calls itself %s.", written)
	}

	var tags []string
	for _, tag := range head.Take("SCHMA").TakeAll("TAG") {
		if text := collapse(tag.Text()); text != "" {
			tags = append(tags, text)
		}
	}
	if len(tags) > 0 {
		b.WriteString("\n\nExtension tags:\n" + strings.Join(tags, "\n"))
	}
	return b.String()
}

// voided reports the pointer version 7.0 writes to say that something is here whose
// record the file does not give — a child nobody named, a source nobody kept. It is an
// absence stated on purpose, so it is neither a reference to follow nor text to read,
// and every caller that would do one of those has to ask.
func voided(value string) bool { return strings.TrimSpace(value) == "@VOID@" }

func pointer(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 2 && strings.HasPrefix(value, "@") && strings.HasSuffix(value, "@") {
		return value[1 : len(value)-1]
	}
	return ""
}
