package gedcom

import (
	"fmt"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// nameSources makes a source of every SOUR record before anything is read, since a
// note anywhere in the file, a repository's or another source's among them, can cite one.
func (c *converter) nameSources(records []*Node) {
	for _, record := range records {
		if record.Tag != "SOUR" {
			continue
		}
		named := collapse(record.First("TITL").Text())
		if named == "" {
			named = collapse(record.First("ABBR").Text())
		}
		if named == "" {
			named = "source " + record.Xref
		}
		if slug := c.newSource(named, "source "+record.Xref); slug != "" {
			c.source[record.Xref] = slug
		}
	}
}

func (c *converter) readSources(records []*Node) {
	for _, record := range records {
		slug, ok := c.source[record.Xref]
		if record.Tag != "SOUR" || !ok {
			continue
		}
		title, short := firstText(record, "TITL"), collapse(firstText(record, "ABBR"))
		if title == "" && short == "" {
			title = "source " + record.Xref
		}
		c.citer = slug
		record.Mark()
		c.home(record, slug, "body")

		source := c.corpus.Sources[slug]
		source.Title, source.ShortTitle = title, short
		source.Author = strings.TrimSpace(record.Take("AUTH").Text())
		source.Publication = strings.TrimSpace(record.Take("PUBL").Text())
		source.Private = c.private(record) || c.flagged(record)
		source.Type = c.typed(record)
		source.Tags = c.tagged(record)
		c.afterwards(func() { source.SameAs = c.sameAs(record) })
		var coverage []string
		source.Covers, source.Attributes, coverage = c.coverage(record.First("DATA"), slug)
		source.Attributes = append(source.Attributes, c.attributes(record, slug)...)
		source.Repositories = c.keptAt(record, slug)
		source.Links = c.links(record)
		source.IDs = c.ids(record)
		var text []string
		for _, node := range record.All("TEXT") {
			if words, _ := c.written(node, ""); words != "" {
				node.Mark()
				text = append(text, words)
			}
		}
		source.Text = strings.Join(text, "\n\n")
		source.Body = strings.Join(append(c.noteTexts(record), coverage...), "\n\n")
	}

	// A scan's own record can cite a source, which has to be there to be cited.
	for _, record := range records {
		if slug, ok := c.source[record.Xref]; ok && record.Tag == "SOUR" {
			c.readMedia(record, c.corpus.Sources[slug])
		}
	}
}

// coverage reads a source's DATA, what it records: each EVEN an entry of covers, the
// tags it lists in the words an event's type is written in, over the period its DATE
// gives and for the place its PLAC names; AGNC the agency that kept it; and each NOTE
// a paragraph of the body, led by *Coverage:*, since what it says is about the record
// as a whole.
func (c *converter) coverage(data *Node, slug model.Slug) ([]model.Coverage, []model.Attribute, []string) {
	if data == nil {
		return nil, nil, nil
	}
	data.Mark()
	c.home(data, slug, "body")
	var covers []model.Coverage
	for _, even := range data.All("EVEN") {
		var events []string
		for _, tag := range strings.Split(even.Value, ",") {
			if word := eventWord(tag); word != "" {
				events = append(events, word)
			}
		}
		date, read := readDateNode(even.Take("DATE"))
		if !read {
			c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
		}
		coverage := model.Coverage{Events: events, Date: date, Place: c.place(even.First("PLAC"))}
		if len(events) == 0 && coverage.Date.IsZero() && coverage.Place == "" {
			continue
		}
		even.Mark()
		c.home(even, slug, fmt.Sprintf("covers[%d].note", len(covers)))
		covers = append(covers, coverage)
	}
	var attributes []model.Attribute
	if agency := collapse(data.First("AGNC").Text()); agency != "" {
		data.Take("AGNC")
		attributes = append(attributes, model.Attribute{Type: "agency", Value: agency})
	}
	return covers, attributes, c.labelled(data, "Coverage")
}

// eventWord is what the model calls an event a tag names, and a word that is no tag of
// GEDCOM's as written.
func eventWord(tag string) string {
	tag = collapse(tag)
	if word, ok := individualEvents[strings.ToUpper(tag)]; ok {
		return word
	}
	if word, ok := familyEvents[strings.ToUpper(tag)]; ok {
		return word
	}
	if strings.EqualFold(tag, "EVEN") {
		return "event"
	}
	return tag
}

// readRepositories makes a repository of every REPO record, before the sources that
// are kept in them are read.
func (c *converter) readRepositories(records []*Node) {
	for _, record := range records {
		if record.Tag != "REPO" {
			continue
		}
		name := collapse(record.Take("NAME").Text())
		if name == "" {
			name = "repository " + record.Xref
		}
		slug := c.newRepository(name, "repository "+record.Xref)
		if slug == "" {
			continue
		}
		c.repository[record.Xref] = slug
		c.citer = slug
		record.Mark()
		c.home(record, slug, "body")

		repository := c.corpus.Repositories[slug]
		if at := c.address(record.First("ADDR")); !at.IsZero() {
			repository.Addresses = []model.Address{at}
			c.home(record.First("ADDR"), slug, "addresses[0].note")
		}
		repository.Links = c.links(record)
		repository.IDs = c.ids(record)
		repository.Private = c.flagged(record)
		repository.Type = c.typed(record)
		repository.Tags = c.tagged(record)
		c.afterwards(func() { repository.SameAs = c.sameAs(record) })
		repository.Body = strings.Join(c.noteTexts(record), "\n\n")
	}
}

// newRepository files a repository under its name, or where that is in a script a slug
// cannot spell, under otherwise, the words for its record.
func (c *converter) newRepository(name, otherwise string) model.Slug {
	base, err := filedUnder(name, otherwise)
	if err != nil {
		return ""
	}
	slug := c.namer.Assign(importing.Shorten(base), nil, name)
	if slug == "" {
		return ""
	}
	c.corpus.Repositories[slug] = &model.Repository{Slug: slug, Name: name}
	return slug
}

// links reads every WWW, EMAIL, PHON and FAX under node as a link of what node is, in
// the order the file gives them: an address as written, a mailbox as mailto: and a
// number as tel:, a FAX typed fax.
func (c *converter) links(node *Node) []model.Link {
	var out []model.Link
	for _, line := range node.Children {
		text := strings.TrimSpace(line.Text())
		if text == "" {
			continue
		}
		var link model.Link
		switch line.Tag {
		case "WWW":
			link = model.Link{URL: text}
		case "EMAIL":
			link = model.Link{URL: "mailto:" + text}
		case "PHON":
			link = model.Link{URL: "tel:" + text}
		case "FAX":
			link = model.Link{URL: "tel:" + text, Type: "fax"}
		case "_LINK":
			if !c.ours[line.Tag] {
				continue
			}
			link = model.Link{
				URL:   text,
				Type:  collapse(line.Take("TYPE").Text()),
				Title: collapse(line.Take("TITL").Text()),
				Note:  strings.TrimSpace(line.Take("NOTE").Text()),
			}
		default:
			continue
		}
		line.Mark()
		out = append(out, link)
	}
	return out
}

// address reads an ADDR as an address: its value the lines, CITY, STAE, POST and CTRY
// the parts they name, and ADR1 to ADR3 the lines again, as 7.0 defines them, or the
// street where they say what the lines do not, an exporter having written only them.
// The links under it are the address's own. Only what it uses is marked read.
func (c *converter) address(node *Node) model.Address {
	if node == nil {
		return model.Address{}
	}
	a := model.Address{
		Lines:   strings.TrimSpace(node.Text()),
		City:    collapse(node.Take("CITY").Text()),
		State:   collapse(node.Take("STAE").Text()),
		Postal:  collapse(node.Take("POST").Text()),
		Country: collapse(node.Take("CTRY").Text()),
		Links:   c.links(node),
	}
	var street []string
	for _, tag := range []string{"ADR1", "ADR2", "ADR3"} {
		line := collapse(node.Take(tag).Text())
		if line != "" && !strings.Contains(collapse(a.Lines), line) {
			street = append(street, line)
		}
	}
	a.Street = strings.Join(street, ", ")
	if !a.IsZero() {
		node.Mark()
	}
	return a
}

// sourceOf resolves the SOUR under an event to a source entity.
//
// A pointer names a record that was already read. A source written in place has no
// record at all, and is created from its own text the first time it is seen — so the
// same book cited by forty events becomes one source, not forty notes.
func (c *converter) sourceOf(sour *Node) model.Slug {
	if voided(sour.Value) {
		return ""
	}
	if xref := pointer(sour.Value); xref != "" {
		if slug, ok := c.source[xref]; ok {
			return slug
		}
		c.notes.Add("a citation names source %s, which is not in the file", xref)
		return ""
	}

	title := strings.TrimSpace(sour.Value)
	if title == "" {
		title = strings.TrimSpace(sour.Take("TITL").Text())
	}
	if title == "" {
		return ""
	}
	if slug, ok := c.inline[title]; ok {
		return slug
	}
	slug := c.newSource(title, "")
	if slug == "" {
		return ""
	}
	c.inline[title] = slug
	return slug
}

// newSource files a source as newRepository files a repository.
func (c *converter) newSource(title, otherwise string) model.Slug {
	base, err := filedUnder(title, otherwise)
	if err != nil {
		return ""
	}
	slug := c.namer.Assign(importing.Shorten(base), nil, title)
	if slug == "" {
		return ""
	}
	c.corpus.Sources[slug] = &model.Source{
		Slug:      slug,
		Title:     title,
		Citations: map[string]model.Citation{},
	}
	return slug
}

// keptAt reads every REPO under a source as where it is kept. GEDCOM points at a REPO
// record for the institution and puts the call numbers on the pointer, so each CALN is
// an entry of its own, with its MEDI as the medium, and a REPO with none is one entry;
// the NOTE on the pointer is the first entry's. An archive named in place rather than as
// a record of its own is made one the first time it is seen, as a source written in
// place is. What it reads it marks, and a REPO it cannot read is kept as written.
func (c *converter) keptAt(record *Node, slug model.Slug) []model.Holding {
	var held []model.Holding
	for _, repo := range record.All("REPO") {
		var repository model.Slug
		switch xref, name := pointer(repo.Value), collapse(repo.Value); {
		case voided(repo.Value):
		case xref != "":
			var ok bool
			if repository, ok = c.repository[xref]; !ok {
				c.notes.Add("a source is kept in repository %s, which is not in the file", xref)
				continue
			}
		case name != "":
			var ok bool
			if repository, ok = c.inlineRepository[name]; !ok {
				if repository = c.newRepository(name, ""); repository != "" {
					c.inlineRepository[name] = repository
				}
			}
		}
		first := len(held)
		for _, caln := range repo.All("CALN") {
			number := collapse(caln.Text())
			if number == "" {
				continue
			}
			caln.Mark()
			c.home(caln, slug, fmt.Sprintf("repositories[%d].note", len(held)))
			held = append(held, model.Holding{Repository: repository, CallNumber: number, Medium: medium(caln.First("MEDI"))})
		}
		if len(held) == first {
			if repository == "" {
				continue
			}
			held = append(held, model.Holding{Repository: repository})
		}
		repo.Mark()
		c.home(repo, slug, fmt.Sprintf("repositories[%d].note", first))
		held[first].Note = strings.Join(c.noteTexts(repo), "\n\n")
	}
	return held
}

// firstText is the text of the first of tags under record that has any, marking that
// one read.
func firstText(record *Node, tags ...string) string {
	for _, tag := range tags {
		if text := strings.TrimSpace(record.First(tag).Text()); text != "" {
			record.Take(tag)
			return text
		}
	}
	return ""
}

func filedUnder(name, otherwise string) (model.Slug, error) {
	if base, err := model.Slugify(name); err == nil || otherwise == "" {
		return base, err
	}
	return model.Slugify(otherwise)
}
