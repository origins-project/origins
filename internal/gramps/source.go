package gramps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// repositoryTypes are Gramps' own types of repository. Unknown says nothing.
var repositoryTypes = vocabulary(map[string]string{"Unknown": ""},
	"Library", "Cemetery", "Church", "Archive", "Album", "Web site", "Bookstore",
	"Collection", "Safe",
)

func (c *converter) readRepositories() {
	for _, e := range c.all("repositories", "repository") {
		name := collapse(e.Take("rname").Text())
		if name == "" {
			name = "repository " + e.Attr("id")
		}
		slug := c.newSlug(name, "repository "+e.Attr("id"))
		if slug == "" {
			c.notes.Add("a repository could not be named and was not imported")
			continue
		}
		c.repository[e.Attr("handle")] = slug
		e.Mark()
		c.home(e, slug, "body")

		repository := &model.Repository{Slug: slug, Name: name, Private: private(e), IDs: c.grampsID(e)}
		repository.Type = typeWord(repositoryTypes, e.Take("type").Text())
		repository.Links = c.links(e, &repository.Private)
		repository.Tags = c.tags(e)
		repository.Body = c.body(e)
		c.corpus.Repositories[slug] = repository
	}
}

// readAddresses reads the addresses of every repository, once the citations an address
// may cite are read, which come after the repositories their sources are kept in.
func (c *converter) readAddresses() {
	for _, e := range c.all("repositories", "repository") {
		repository := c.corpus.Repositories[c.repository[e.Attr("handle")]]
		if repository == nil {
			continue
		}
		for _, address := range e.All("address") {
			at := fmt.Sprintf("addresses[%d].note", len(repository.Addresses))
			repository.Addresses = append(repository.Addresses, c.whereAt(address, &repository.Private, repository.Slug, at))
		}
	}
}

// whereAt reads an address of a repository's, with its date, its notes and its
// citations, its telephone among its links, as the entry at path at on record. A private
// one makes private what it is the address of, which the flag given is.
func (c *converter) whereAt(e *Element, flag *bool, record model.Slug, at string) model.Address {
	e.Mark()
	c.home(e, record, at)
	address := addressOf(e)
	address.Date = c.date(e)
	address.Note = strings.Join(c.prose(e), "\n\n")
	address.Sources = c.citations(e)
	if private(e) && !*flag {
		*flag = true
		c.notes.Add("a private address made private the record it is the address of")
	}
	return address
}

func (c *converter) readSources() {
	for _, e := range c.all("sources", "source") {
		title := collapse(e.Take("stitle").Text())
		short := collapse(e.Take("sabbrev").Text())
		named := title
		if named == "" {
			named = short
		}
		if named == "" {
			named = "source " + e.Attr("id")
		}
		slug := c.newSlug(named, "source "+e.Attr("id"))
		if slug == "" {
			c.notes.Add("a source could not be named and was not imported")
			continue
		}
		c.source[e.Attr("handle")] = slug
		e.Mark()
		c.home(e, slug, "body")

		source := &model.Source{
			Slug:        slug,
			Private:     private(e),
			Title:       title,
			ShortTitle:  short,
			Author:      collapse(e.Take("sauthor").Text()),
			Publication: collapse(e.Take("spubinfo").Text()),
			Citations:   map[string]model.Citation{},
			IDs:         c.grampsID(e),
		}
		c.corpus.Sources[slug] = source
		for _, ref := range e.All("reporef") {
			c.keptAt(source, ref)
		}
		for _, attribute := range e.All("srcattribute") {
			source.Attributes = append(source.Attributes, c.said(attribute, &source.Private))
		}
		source.Tags = c.tags(e)
		source.Body = c.body(e)
		c.pictures(e, source, &source.Media, -1)
	}
}

// mediums are Gramps' own media a source is kept in, in the corpus's words. A book is
// what Gramps takes every source to be until somebody says otherwise, and says nothing.
var mediums = vocabulary(map[string]string{"Unknown": "", "Book": ""},
	"Audio", "Card", "Electronic", "Fiche", "Film", "Magazine", "Manuscript", "Map",
	"Newspaper", "Photo", "Tombstone", "Video",
)

// keptAt reads one of the places a source is kept: the repository, the call number
// there, the medium and the notes. A private reference makes the source private.
func (c *converter) keptAt(source *model.Source, ref *Element) {
	slug, ok := c.repository[ref.Attr("hlink")]
	if !ok {
		c.notes.Add("a source is kept in a repository that is not in the file")
		return
	}
	ref.Mark()
	c.home(ref, source.Slug, fmt.Sprintf("repositories[%d].note", len(source.Repositories)))
	source.Repositories = append(source.Repositories, model.Holding{
		Repository: slug,
		CallNumber: collapse(ref.Use("callno")),
		Medium:     typeWord(mediums, ref.Use("medium")),
		Note:       strings.Join(c.prose(ref), "\n\n"),
	})
	if private(ref) && !source.Private {
		source.Private = true
		c.notes.Add("a private reference to a repository made private the source kept there")
	}
}

// newSlug gives a source or a repository its slug from its title or name, shortened as
// the GEDCOM import shortens one, or where that is in a script a slug cannot spell, from
// otherwise, the words for its object.
func (c *converter) newSlug(title, otherwise string) model.Slug {
	base, err := model.Slugify(title)
	if err != nil {
		if base, err = model.Slugify(otherwise); err != nil {
			return ""
		}
	}
	return c.namer.Assign(importing.Shorten(base), nil, title)
}

// weights are the weight a citation's confidence gives every reference to it, as
// Gramps' own GEDCOM export writes them as QUAY. Normal, which every new citation has,
// gives none.
var weights = map[string]byte{"0": '0', "1": '1', "3": '2', "4": '3'}

// readCitations makes each citation an anchor in its source. A Gramps citation is
// already one entry cited from many places, which is what an anchor is, so nothing has
// to be recognised as an entry already read. A note typed as the record's words is its
// transcription and any other its note, and a private citation makes its source
// private, the anchor having no flag of its own.
func (c *converter) readCitations() {
	for _, e := range c.all("citations", "citation") {
		source, ok := c.source[e.First("sourceref").Attr("hlink")]
		if !ok {
			c.notes.Add("a citation of a source that is not in the file was not imported")
			continue
		}
		e.Mark()
		e.Take("sourceref")
		citation := model.Citation{Page: collapse(e.Take("page").Text()), Date: c.date(e), IDs: c.grampsID(e)}
		ref := model.CitationRef{Source: source}
		confidence := collapse(e.First("confidence").Text())
		if weight, graded := weights[confidence]; graded || confidence == "" || confidence == "2" {
			ref.Grade.Weight = weight
			e.Take("confidence")
		}

		held := c.corpus.Sources[source]
		ref.Anchor = c.anchor(source, citation.Page, e.Attr("id"))
		anchor := ref.Anchor
		c.home(e, source, "citations."+anchor+".note")
		citation.Text = strings.Join(c.proseOf(e, transcribed), "\n\n")
		citation.Note = strings.Join(c.proseOf(e, func(note *Element) bool { return !transcribed(note) }), "\n\n")
		for _, attribute := range e.All("srcattribute") {
			citation.Attributes = append(citation.Attributes, c.said(attribute, &held.Private))
		}
		citation.Tags = c.tags(e)
		if private(e) && !held.Private {
			held.Private = true
			c.notes.Add("a private citation made private the source it cites")
		}

		c.citation[e.Attr("handle")] = ref
		for _, objref := range e.All("objref") {
			at := fmt.Sprintf("citations.%s.images[%d].note", anchor, len(citation.Images))
			if image, ok := c.scan(objref, held, at); ok {
				citation.Images = append(citation.Images, image)
			}
		}
		held.Citations[anchor] = citation
	}
}

// anchor names a citation within its source after its page, which is what a reader
// would look up, or its Gramps ID where it gives none; two of one source taking the
// same name are told apart by a number after it.
func (c *converter) anchor(source model.Slug, page, id string) string {
	base, err := model.Slugify(page)
	if err != nil {
		if base, err = model.Slugify(id); err != nil {
			base = "citation"
		}
	}
	within := c.corpus.Sources[source].Citations
	for candidate := 1; ; candidate++ {
		name := string(base)
		if candidate > 1 {
			name += "-" + strconv.Itoa(candidate)
		}
		if _, used := within[name]; !used {
			// held until the citation is read whole, so the next one does not take it
			within[name] = model.Citation{}
			return name
		}
	}
}

// citations are the citations an element refers to, as references to their anchors.
func (c *converter) citations(e *Element) []model.CitationRef {
	var refs []model.CitationRef
	for _, ref := range e.All("citationref") {
		cited, ok := c.citation[ref.Attr("hlink")]
		if !ok {
			if c.object(ref, "citation") == nil {
				c.notes.Add("a citation was referred to that is not in the file")
			}
			continue
		}
		ref.Mark()
		refs = append(refs, cited)
	}
	return refs
}

// addressOf reads the parts of a Gramps address or location, and its telephone as a
// link.
func addressOf(e *Element) model.Address {
	part := func(name string) string { return collapse(e.Take(name).Text()) }
	address := model.Address{
		Street: part("street"), Locality: part("locality"), City: part("city"),
		Parish: part("parish"), County: part("county"), State: part("state"),
		Postal: part("postal"), Country: part("country"),
	}
	if phone := part("phone"); phone != "" {
		address.Links = []model.Link{{URL: "tel:" + phone}}
	}
	return address
}
