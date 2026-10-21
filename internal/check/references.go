package check

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// danglingReferences walks every slug written anywhere in the corpus and confirms it
// names something. A reference is the only way entities point at each other, so a
// dangling one is the corpus's equivalent of a broken pointer.
func danglingReferences(c *model.Corpus, r *report) {
	if main := c.Project.MainPerson; main != "" {
		if _, ok := c.People[main]; !ok {
			r.fail("dangling-main-person", "", "project.main-person", "no person %q", main)
		}
	}
	for i, b := range c.Project.Bookmarks {
		if kind := c.KindOf(b.Slug); kind != b.Kind {
			r.fail("dangling-bookmark", "", at("project.bookmarks", i), "no %s %q", b.Kind, b.Slug)
		}
	}

	for _, s := range c.PeopleSlugs() {
		person, entity := c.People[s], ref(model.KindPerson, s)
		citationReferences(c, r, entity, "name.sources", person.Name.Sources)
		for i, name := range person.AlsoKnownAs {
			citationReferences(c, r, entity, at("also-known-as", i)+".sources", name.Sources)
		}
		citationReferences(c, r, entity, "sex.sources", person.Sex.Sources)
		eventReferences(c, r, entity, person.Events)
		nonEventReferences(c, r, entity, person.Not)
		citationReferences(c, r, entity, "sources", person.Sources)
		identifierReferences(c, r, entity, "ids", person.IDs)
	}

	for _, s := range c.FamilySlugs() {
		family, entity := c.Families[s], ref(model.KindFamily, s)
		for i, partner := range family.Partners {
			if _, ok := c.People[partner.Person]; partner.Somebody() && !ok {
				r.fail("dangling-partner", entity, at("partners", i), "no person %q", partner.Person)
			}
			citationReferences(c, r, entity, at("partners", i)+".name.sources", partner.Name.Sources)
		}
		for i, child := range family.Children {
			if _, ok := c.People[child.ID]; child.Somebody() && !ok {
				r.fail("dangling-child", entity, at("children", i), "no person %q", child.ID)
			}
			citationReferences(c, r, entity, at("children", i)+".name.sources", child.Name.Sources)
			citationReferences(c, r, entity, at("children", i)+".sources", child.Sources)
		}
		eventReferences(c, r, entity, family.Events)
		for i, event := range family.Events {
			if event.Family != "" {
				r.fail("family-event-family", entity, at("events", i)+".family",
					"a family's own event is the couple's already, and names no family")
			}
		}
		nonEventReferences(c, r, entity, family.Not)
		citationReferences(c, r, entity, "sources", family.Sources)
		identifierReferences(c, r, entity, "ids", family.IDs)
	}

	for _, s := range c.PlaceSlugs() {
		place, entity := c.Places[s], ref(model.KindPlace, s)
		citationReferences(c, r, entity, "name.sources", place.Name.Sources)
		for i, name := range place.AlsoKnownAs {
			citationReferences(c, r, entity, at("also-known-as", i)+".sources", name.Sources)
		}
		for i, containing := range place.Within {
			if _, ok := c.Places[containing.Place]; !ok {
				r.fail("dangling-place", entity, at("within", i), "no place %q", containing.Place)
			}
			citationReferences(c, r, entity, at("within", i)+".sources", containing.Sources)
		}
		addressReferences(c, r, entity, place.Addresses)
		citationReferences(c, r, entity, "sources", place.Sources)
		identifierReferences(c, r, entity, "ids", place.IDs)
	}

	for _, s := range c.SourceSlugs() {
		source, entity := c.Sources[s], ref(model.KindSource, s)
		for i, covered := range source.Covers {
			if _, ok := c.Places[covered.Place]; covered.Place != "" && !ok {
				r.fail("dangling-place", entity, at("covers", i)+".place", "no place %q", covered.Place)
			}
		}
		for i, held := range source.Repositories {
			path := at("repositories", i)
			if held.Qualified() {
				path += ".repository"
			}
			if _, ok := c.Repositories[held.Repository]; held.Repository != "" && !ok {
				r.fail("dangling-repository", entity, path, "no repository %q", held.Repository)
			}
		}
		attributeReferences(c, r, entity, "attributes", source.Attributes)
		for _, anchor := range slices.Sorted(maps.Keys(source.Citations)) {
			citation, path := source.Citations[anchor], "citations."+anchor
			for j, named := range citation.With {
				namedReferences(c, r, entity, path+"."+at("with", j), named)
			}
			for j, image := range citation.Images {
				mediaReferences(c, r, entity, path+"."+at("images", j), image)
			}
			attributeReferences(c, r, entity, path+".attributes", citation.Attributes)
			identifierReferences(c, r, entity, path+".ids", citation.IDs)
		}
		identifierReferences(c, r, entity, "ids", source.IDs)
	}

	for _, s := range c.RepositorySlugs() {
		repository, entity := c.Repositories[s], ref(model.KindRepository, s)
		addressReferences(c, r, entity, repository.Addresses)
		identifierReferences(c, r, entity, "ids", repository.IDs)
	}

	for _, s := range c.NoteSlugs() {
		note, entity := c.Notes[s], ref(model.KindNote, s)
		eventReferences(c, r, entity, note.Events)
		citationReferences(c, r, entity, "sources", note.Sources)
		identifierReferences(c, r, entity, "ids", note.IDs)
	}

	for _, s := range c.HypothesisSlugs() {
		hypothesis, entity := c.Hypotheses[s], ref(model.KindHypothesis, s)
		citationReferences(c, r, entity, "supports", hypothesis.Supports)
		citationReferences(c, r, entity, "contradicts", hypothesis.Contradicts)
	}

	for _, s := range c.AllSlugs() {
		record := c.Entity(s)
		model.Pictures(record, func(path string, media *model.Media) {
			mediaReferences(c, r, model.RefOf(record).String(), path, *media)
		})
	}
}

// sameRecords reports every claim that a record is the same as another. Each is research
// still to do, a merge to make or a claim to take away; one naming a record of another
// kind, or the record it is on, is one no merge could settle.
func sameRecords(c *model.Corpus, r *report) {
	for _, s := range c.AllSlugs() {
		same := model.SameAsOf(c.Entity(s))
		if same == nil {
			continue
		}
		kind := c.KindOf(s)
		entity := ref(kind, s)
		for i, claim := range *same {
			path := at("same-as", i)
			citationReferences(c, r, entity, path+".sources", claim.Sources)
			if claim.ID == "" {
				r.warn("same-as", entity, path+".name",
					"said to be the same as %q, who has no record here", claim.Name)
				continue
			}
			if claim.Qualified() {
				path += ".id"
			}
			switch other := c.KindOf(claim.ID); {
			case claim.ID == s:
				r.fail("same-as-itself", entity, path, "a record is not said to be the same as itself")
			case other == "":
				r.fail("dangling-same-as", entity, path, "no %s %q", kind, claim.ID)
			case other != kind:
				r.fail("same-as-another-kind", entity, path,
					"%q is a %s, and a %s is the same as a %s or nothing", claim.ID, other, kind, kind)
			default:
				r.warn("same-as", entity, path,
					"said to be the same %s as %q: merge the two, or take the claim away", kind, claim.ID)
			}
		}
	}
}

// proseLinks follows every relative link in prose to the record it leads into, and
// that record to the file or the citation it names. A link is a reference written in
// words, and a rename rewrites it like any other, so one leading nowhere is as broken as
// a slug naming nobody. A link leading anywhere but into a record is no reference to
// one, and is left alone.
func proseLinks(c *model.Corpus, r *report) {
	for _, s := range c.AllSlugs() {
		record := c.Entity(s)
		from := model.RefOf(record)
		model.Prose(record, func(path string, text *string) {
			for _, link := range model.Links(from, *text) {
				if link.Record.Kind == "" {
					continue
				}
				if problem := leadsNowhere(c, link); problem != "" {
					r.fail("dangling-link", from.String(), path, "a link to %s: %s",
						(*text)[link.Start:link.End], problem)
				}
			}
		})
	}
}

// leadsNowhere is what a link into a record names that is not there, or "" where it is
// all there.
func leadsNowhere(c *model.Corpus, link model.ProseLink) string {
	target := c.Entity(link.Record.Slug)
	if model.RefOf(target).Kind != link.Record.Kind {
		return fmt.Sprintf("no %s %q", link.Record.Kind, link.Record.Slug)
	}
	if link.File != "" {
		if !slices.Contains(*model.Files(target), link.File) {
			return fmt.Sprintf("no file %q beside %s", link.File, link.Record)
		}
		return ""
	}
	if source, ok := target.(*model.Source); ok && link.Anchor != "" {
		if _, ok := source.Citations[link.Anchor]; !ok {
			return fmt.Sprintf("source %q has no citation %q", source.Slug, link.Anchor)
		}
	}
	return ""
}

func mediaReferences(c *model.Corpus, r *report, entity, path string, media model.Media) {
	if media.Place != "" {
		if _, ok := c.Places[media.Place]; !ok {
			r.fail("dangling-place", entity, path+".place", "no place %q", media.Place)
		}
	}
	for j, depicted := range media.Depicts {
		if _, ok := c.People[depicted.Person]; !ok {
			r.fail("dangling-depicted", entity, path+"."+at("depicts", j), "no person %q", depicted.Person)
		}
		attributeReferences(c, r, entity, path+"."+at("depicts", j)+".attributes", depicted.Attributes)
	}
	attributeReferences(c, r, entity, path+".attributes", media.Attributes)
	citationReferences(c, r, entity, path+".sources", media.Sources)
	identifierReferences(c, r, entity, path+".ids", media.IDs)
}

func eventReferences(c *model.Corpus, r *report, entity string, events []model.Event) {
	for i, event := range events {
		if event.Place != "" {
			if _, ok := c.Places[event.Place]; !ok {
				r.fail("dangling-place", entity, at("events", i)+".place", "no place %q", event.Place)
			}
		}
		for j, named := range event.With {
			namedReferences(c, r, entity, at("events", i)+"."+at("with", j), named)
		}
		if event.Family != "" {
			if _, ok := c.Families[event.Family]; !ok {
				r.fail("dangling-family", entity, at("events", i)+".family", "no family %q", event.Family)
			}
		}
		citationReferences(c, r, entity, at("events", i)+".address.sources", event.Address.Sources)
		citationReferences(c, r, entity, at("events", i)+".age.sources", event.Age.Sources)
		citationReferences(c, r, entity, at("events", i)+".cause.sources", event.Cause.Sources)
		attributeReferences(c, r, entity, at("events", i)+".attributes", event.Attributes)
		citationReferences(c, r, entity, at("events", i)+".sources", event.Sources)
		identifierReferences(c, r, entity, at("events", i)+".ids", event.IDs)
	}
}

func namedReferences(c *model.Corpus, r *report, entity, path string, named model.Named) {
	if named.Person != "" && named.Person != model.Unknown {
		if _, ok := c.People[named.Person]; !ok {
			r.fail("dangling-named", entity, path+".person", "no person %q", named.Person)
		}
	}
	citationReferences(c, r, entity, path+".name.sources", named.Name.Sources)
	attributeReferences(c, r, entity, path+".attributes", named.Attributes)
	citationReferences(c, r, entity, path+".sources", named.Sources)
}

func attributeReferences(c *model.Corpus, r *report, entity, path string, attributes []model.Attribute) {
	for i, attribute := range attributes {
		citationReferences(c, r, entity, at(path, i)+".sources", attribute.Sources)
	}
}

func nonEventReferences(c *model.Corpus, r *report, entity string, not []model.NonEvent) {
	for i, entry := range not {
		citationReferences(c, r, entity, at("not", i)+".sources", entry.Sources)
	}
}

func addressReferences(c *model.Corpus, r *report, entity string, addresses []model.Address) {
	for i, address := range addresses {
		citationReferences(c, r, entity, at("addresses", i)+".sources", address.Sources)
	}
}

func identifierReferences(c *model.Corpus, r *report, entity, path string, ids model.IDs) {
	for _, service := range slices.Sorted(maps.Keys(ids)) {
		for i, id := range ids[service] {
			citationReferences(c, r, entity, at(path+"."+service, i)+".sources", id.Sources)
		}
	}
}

func citationReferences(c *model.Corpus, r *report, entity, path string, citations []model.CitationRef) {
	for i, citation := range citations {
		citationReference(c, r, entity, at(path, i), citation)
	}
}

func citationReference(c *model.Corpus, r *report, entity, path string, citation model.CitationRef) {
	source, ok := c.Sources[citation.Source]
	if !ok {
		r.fail("dangling-source", entity, path, "no source %q", citation.Source)
		return
	}
	if _, ok := source.Citations[citation.Anchor]; !ok {
		r.fail("dangling-citation", entity, path,
			"source %q has no citation %q", citation.Source, citation.Anchor)
	}
}

// familyShape holds a family to its partners: at least two of them, nobody beside
// exactly one person, and a name they make. The slug is built from the partners, so a
// family whose partners have changed without its directory being renamed would
// otherwise keep a name that no longer describes it. A family told apart from another
// of the same partners is named by them all the same.
func familyShape(c *model.Corpus, r *report) {
	for _, s := range c.FamilySlugs() {
		family, entity := c.Families[s], ref(model.KindFamily, s)
		if len(family.Partners) < 2 {
			r.fail("family-needs-partners", entity, "partners",
				"a family has at least two partners, found %d", len(family.Partners))
			continue
		}
		for i, partner := range family.Partners {
			if partner.Who() != model.None {
				continue
			}
			if len(family.Partners) != 2 || !family.Partners[1-i].Somebody() {
				r.fail("family-none", entity, at("partners", i),
					"none stands beside exactly one person, there being nobody else")
			}
		}
		expected, err := model.FamilySlug(family.Partners)
		if err != nil {
			continue
		}
		if _, named := s.Discriminator(expected); !named {
			r.fail("family-slug-mismatch", entity, "partners", "the partners name the family %q", expected)
		}
	}
}

// familyChildren holds a family's children to its partners. A relation is keyed by a
// partner, a child is listed once and is not a partner too, and is a child of somebody
// in the family; a family with `none` in it has no birth child, since anybody born had
// two parents. A disproven link is kept as research done, and warned of, being usually
// one to remove.
func familyChildren(c *model.Corpus, r *report) {
	for _, s := range c.FamilySlugs() {
		family, entity := c.Families[s], ref(model.KindFamily, s)
		alone := slices.ContainsFunc(family.Partners, func(p model.Partner) bool { return p.Who() == model.None })
		for i, child := range family.Children {
			path := at("children", i)
			for j, keyed := range child.Relation.Partners {
				if !slices.ContainsFunc(family.Partners, func(p model.Partner) bool { return p.Who() == keyed.Partner }) {
					r.fail("relation-not-a-partner", entity, path+".relation",
						"%s is not a partner, at %d of the relation", keyed.Partner, j)
				}
			}
			if child.Somebody() {
				if slices.ContainsFunc(family.Children[:i], func(c model.Child) bool { return c.ID == child.ID }) {
					r.fail("child-listed-twice", entity, path, "%s is listed already", child.ID)
				}
				if slices.ContainsFunc(family.Partners, func(p model.Partner) bool { return p.Person == child.ID }) {
					r.fail("child-is-partner", entity, path, "%s is a partner in the family", child.ID)
				}
			}
			var anybody, born bool
			for _, partner := range family.Partners {
				if partner.Who() == model.None {
					continue
				}
				kin := child.To(partner.Who()).Kin()
				anybody = anybody || kin != model.NoRelation
				born = born || kin == model.Birth
			}
			if !anybody {
				r.fail("child-of-nobody", entity, path+".relation", "%s is no child of any partner", child.Who())
			}
			if alone && born && child.Status != model.Disproven {
				r.fail("none-with-birth-child", entity, path,
					"%s was born to the family, so the other parent is unknown rather than none", child.Who())
			}
			if child.Status == model.Disproven {
				r.warn("disproven-link", entity, path+".status",
					"%s is listed as no child of the family", child.Who())
			}
		}
	}
}

// birthParents holds a person to two birth parents across every family listing them, a
// partner nobody knows counting as one apiece.
func birthParents(c *model.Corpus, r *report) {
	parentage := c.Parentage()
	for _, s := range c.PeopleSlugs() {
		var seen []model.Slug
		count := 0
		for _, parent := range parentage[s] {
			if !parent.Relation.Biological() || slices.Contains(seen, parent.Slug) {
				continue
			}
			if parent.Slug != model.Unknown {
				seen = append(seen, parent.Slug)
			}
			count++
		}
		if count > 2 {
			r.fail("too-many-birth-parents", ref(model.KindPerson, s), "",
				"%d birth parents across the families listing them", count)
		}
	}
}

// reservedSlugs keeps the two words a partner is written as where there is no record
// from naming one, since a partner written as either would then be two things at once.
func reservedSlugs(c *model.Corpus, r *report) {
	for _, s := range c.AllSlugs() {
		if s.Reserved() {
			r.fail("reserved-slug", ref(c.KindOf(s), s), "",
				"%q is the word for a partner without a record, and names no record", s)
		}
	}
}

// missingImages ties a source's citations to the scans beside them: an image a citation
// points at must be there, save one still to be found, whose path is the one it was last
// known by and which is research, as any picture's is.
func missingImages(c *model.Corpus, r *report) {
	for _, s := range c.SourceSlugs() {
		source := c.Sources[s]
		for _, anchor := range slices.Sorted(maps.Keys(source.Citations)) {
			for i, image := range source.Citations[anchor].Images {
				if model.Locate(model.RefOf(source), image.File).Abroad {
					r.warn("missing-media", ref(model.KindSource, s), at("citations."+anchor+".images", i),
						"a picture still to be found, last known as %q", image.File)
				} else if !slices.Contains(source.Files, image.File) {
					r.fail("missing-image", ref(model.KindSource, s), at("citations."+anchor+".images", i),
						"no file %q beside the source", image.File)
				}
			}
		}
	}
}

// impossibleRegions holds every region of a picture to the picture: four fractions of
// it, the left edge left of the right and the top above the bottom.
func impossibleRegions(c *model.Corpus, r *report) {
	for _, s := range c.SourceSlugs() {
		source := c.Sources[s]
		for _, anchor := range slices.Sorted(maps.Keys(source.Citations)) {
			for i, image := range source.Citations[anchor].Images {
				mediaRegions(r, ref(model.KindSource, s), at("citations."+anchor+".images", i), image)
			}
		}
	}
	for _, s := range c.PeopleSlugs() {
		if portrait := c.People[s].Portrait; portrait.Region != nil {
			region(r, ref(model.KindPerson, s), "portrait.region", *portrait.Region)
		}
	}
	for _, s := range c.AllSlugs() {
		record := c.Entity(s)
		model.Pictures(record, func(path string, media *model.Media) {
			mediaRegions(r, model.RefOf(record).String(), path, *media)
		})
	}
}

func mediaRegions(r *report, entity, path string, media model.Media) {
	if media.Region != nil {
		region(r, entity, path+".region", *media.Region)
	}
	for j, depicted := range media.Depicts {
		if depicted.Region != nil {
			region(r, entity, path+"."+at("depicts", j)+".region", *depicted.Region)
		}
	}
}

func region(r *report, entity, path string, region model.Region) {
	for _, edge := range []float64{region.Left, region.Top, region.Right, region.Bottom} {
		if edge < 0 || edge > 1 {
			r.fail("impossible-region", entity, path,
				"%v lies outside the picture, a region being four fractions of it between 0 and 1", edge)
			return
		}
	}
	switch {
	case region.Left >= region.Right:
		r.fail("impossible-region", entity, path,
			"is turned inside out: its left edge %v is not left of its right edge %v", region.Left, region.Right)
	case region.Top >= region.Bottom:
		r.fail("impossible-region", entity, path,
			"is turned inside out: its top edge %v is not above its bottom edge %v", region.Top, region.Bottom)
	}
}

// untranscribedScans is the other direction: a scan no citation points at is a
// transcription waiting to be made.
func untranscribedScans(c *model.Corpus, r *report) {
	for _, s := range c.SourceSlugs() {
		source := c.Sources[s]
		used := map[string]bool{}
		for _, citation := range source.Citations {
			for _, image := range citation.Images {
				used[image.File] = true
			}
		}
		for _, file := range source.Files {
			if !used[file] {
				r.warn("untranscribed-scan", ref(model.KindSource, s), "",
					"%q is not the image of any citation", file)
			}
		}
	}
}

// attachments ties what a record says about its files to the files beside it. A
// picture whose file is not there is an entry all the same, one still to be found or a
// photograph renamed on disk and not in the frontmatter, and is warned of. A path
// leading into the corpus leads into a record's directory, and one into another
// record's to a picture that record has an entry for, the reference saying nothing of
// what the picture is. Two entries of one list naming one file are two accounts of one
// picture, and which of them is right is not something to choose by position.
func attachments(c *model.Corpus, r *report) {
	for _, s := range c.PeopleSlugs() {
		person := c.People[s]
		if person.Portrait.File != "" && !slices.Contains(person.Files, person.Portrait.File) {
			r.fail("missing-portrait", ref(model.KindPerson, s), "portrait",
				"no file %q beside the person", person.Portrait.File)
		}
	}
	for _, s := range c.AllSlugs() {
		record := c.Entity(s)
		from := model.RefOf(record)
		entity, files := from.String(), *model.Files(record)
		model.MediaLists(record, func(list string, media []model.Media) {
			named := map[string]int{}
			for i, m := range media {
				path := at(list, i) + ".file"
				where := model.Locate(from, m.File)
				picture := where.Record.String() + "/" + where.File
				switch {
				case where.Abroad:
					picture = m.File
					r.warn("missing-media", entity, path, "a picture still to be found, last known as %q", m.File)
				case where.Record.Kind == "":
					picture = m.File
					r.fail("media-outside", entity, path, "%q leads into no record's directory", m.File)
				case m.Refers():
					if kept := c.Entity(where.Record.Slug); model.RefOf(kept) != where.Record {
						r.fail("dangling-media", entity, path, "no %s %q", where.Record.Kind, where.Record.Slug)
					} else if !model.Keeps(kept, where.File) {
						r.fail("unlisted-media", entity, path, "%s has no entry for %q", where.Record, where.File)
					}
				default:
					if !slices.Contains(files, where.File) {
						r.warn("missing-media", entity, path, "no file %q beside the %s", where.File, from.Kind)
					}
				}
				if picture == "" {
					continue
				}
				if first, twice := named[picture]; twice {
					r.fail("duplicate-media", entity, path, "%s already says what %q is", at(list, first), m.File)
					continue
				}
				named[picture] = i
			}
		})
	}
}

// surnameSlugs keeps directory listings grouping families. The slug may carry a
// discriminator after the surname and given name, so this checks the beginning of it
// rather than the whole.
func surnameSlugs(c *model.Corpus, r *report) {
	for _, s := range c.PeopleSlugs() {
		primary := c.People[s].Name.PrimarySurname().Name
		if primary == "" {
			continue
		}
		surname, err := model.Slugify(primary)
		if err != nil {
			continue
		}
		if s != surname && !strings.HasPrefix(string(s), string(surname)+"-") {
			r.warn("surname-slug-mismatch", ref(model.KindPerson, s), "name.surname",
				"surname %q would put this person under %q", primary, surname)
		}
	}
}

// surnameLists holds a name's surnames to saying one thing. A list gives each surname
// its own prefix, so a surname-prefix beside it says nothing about which one it belongs
// to, and a name is filed under one surname, not two.
func surnameLists(c *model.Corpus, r *report) {
	check := func(entity string) func(string, *model.PersonName) {
		return func(path string, name *model.PersonName) {
			if name.SurnamePrefix != "" && name.Surname.Listed() {
				r.fail("surname-prefix-beside-list", entity, path+".surname-prefix",
					"%q belongs to no one of the surnames listed, each of which takes its own prefix", name.SurnamePrefix)
			}
			primary := -1
			for i, surname := range name.Surname {
				if !surname.Primary {
					continue
				}
				if primary >= 0 {
					r.fail("two-primary-surnames", entity, fmt.Sprintf("%s.surname[%d].primary", path, i),
						"%q and %q are both marked primary", name.Surname[primary].Name, surname.Name)
				}
				primary = i
			}
		}
	}
	for _, s := range c.PeopleSlugs() {
		model.Names(c.People[s], check(ref(model.KindPerson, s)))
	}
	for _, s := range c.FamilySlugs() {
		model.Names(c.Families[s], check(ref(model.KindFamily, s)))
	}
}

// unplacedCoordinates reports coordinates kept as a record's words. They are what the
// record gave, so nothing is wrong, but nothing can put the place on a map until
// somebody reads them as two angles.
func unplacedCoordinates(c *model.Corpus, r *report) {
	for _, s := range c.PlaceSlugs() {
		if at := c.Places[s].Coordinates; at != nil && !at.Placed() {
			r.warn("coordinates-as-words", ref(model.KindPlace, s), "coordinates",
				"%q cannot be read as a latitude and a longitude", at.Wording)
		}
	}
}

// unsourcedEvents reports what still needs evidence. A tree assembled from memory
// before the records are found is an ordinary state to be in and never an error, so
// these are warnings and there will usually be many.
func unsourcedEvents(c *model.Corpus, r *report) {
	for _, s := range c.PeopleSlugs() {
		unsourced(r, ref(model.KindPerson, s), c.People[s].Events)
	}
	for _, s := range c.FamilySlugs() {
		unsourced(r, ref(model.KindFamily, s), c.Families[s].Events)
	}
	for _, s := range c.NoteSlugs() {
		unsourced(r, ref(model.KindNote, s), c.Notes[s].Events)
	}
}

func unsourced(r *report, entity string, events []model.Event) {
	for i, event := range events {
		if len(event.Sources) == 0 {
			r.warn("unsourced-event", entity, at("events", i), "%s cites nothing", event.Type)
		}
	}
}
