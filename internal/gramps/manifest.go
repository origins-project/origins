package gramps

import (
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// readManifest reads what the export says of the tree rather than of anybody in it into
// the manifest: who compiled it, whom it starts from, the records kept to hand and how
// each tag is shown. It reads last, since a bookmark may lead to any record.
func (c *converter) readManifest() {
	project := &c.corpus.Project
	if researcher, ok := c.researcher(c.root.First("header").First("researcher")); ok {
		project.Researchers = append(project.Researchers, researcher)
	}
	people := c.root.First("people")
	if home, ok := c.person[people.Attr("home")]; ok {
		people.Use("home")
		project.MainPerson = home
	}
	if bookmarks := c.root.First("bookmarks"); bookmarks != nil {
		bookmarks.Mark()
		for _, bookmark := range bookmarks.All("bookmark") {
			ref, ok := c.bookmarked(bookmark.Attr("target"), bookmark.Attr("hlink"))
			if !ok {
				continue
			}
			bookmark.Mark()
			bookmark.Use("target")
			if !slices.Contains(project.Bookmarks, ref) {
				project.Bookmarks = append(project.Bookmarks, ref)
			}
		}
	}
	for _, tag := range c.all("tags", "tag") {
		name := collapse(tag.Attr("name"))
		if name == "" {
			continue
		}
		tag.Mark()
		tag.Use("name")
		if project.Tags == nil {
			project.Tags = map[string]model.TagStyle{}
		}
		project.Tags[name] = model.TagStyle{
			Color:    strings.TrimSpace(tag.Use("color")),
			Priority: strings.TrimSpace(tag.Use("priority")),
		}
	}
}

// researcher reads the researcher Gramps' header names, the owner of the database, as
// an entry of the manifest's researchers: their address, and their telephone and e-mail
// as links. One with nothing in it says nothing, and is read as nothing.
func (c *converter) researcher(e *Element) (model.Researcher, bool) {
	name := collapse(e.First("resname").Text())
	if name == "" {
		if e.blank() {
			e.markAll()
		}
		return model.Researcher{}, false
	}
	e.Mark()
	part := func(name string) string { return collapse(e.Take(name).Text()) }
	researcher := model.Researcher{Name: part("resname")}
	address := model.Address{
		Street: part("resaddr"), Locality: part("reslocality"), City: part("rescity"),
		State: part("resstate"), Country: part("rescountry"), Postal: part("respostal"),
	}
	if address.Qualified() {
		researcher.Addresses = []model.Address{address}
	}
	if phone := part("resphone"); phone != "" {
		researcher.Links = append(researcher.Links, model.Link{URL: "tel:" + phone})
	}
	if email := part("resemail"); email != "" {
		if !strings.HasPrefix(strings.ToLower(email), "mailto:") {
			email = "mailto:" + email
		}
		researcher.Links = append(researcher.Links, model.Link{URL: email})
	}
	return researcher, true
}

// bookmarked is the record a bookmark leads to: the record itself, the source of a
// citation, the record holding an event and the record keeping a picture.
func (c *converter) bookmarked(target, handle string) (model.EntityRef, bool) {
	var slug model.Slug
	var ok bool
	switch target {
	case "person":
		slug, ok = c.person[handle]
	case "family":
		slug, ok = c.family[handle]
	case "place":
		slug, ok = c.place[handle]
	case "source":
		slug, ok = c.source[handle]
	case "repository":
		slug, ok = c.repository[handle]
	case "note":
		slug, ok = c.note[handle]
	case "citation":
		var ref model.CitationRef
		ref, ok = c.citation[handle]
		slug = ref.Source
	case "event":
		slug, ok = c.eventAt[handle]
	case "media":
		var picture *kept
		if picture, ok = c.kept[handle]; ok {
			return picture.record, true
		}
	}
	entity := c.corpus.Entity(slug)
	if !ok || entity == nil {
		return model.EntityRef{}, false
	}
	return model.RefOf(entity), true
}
