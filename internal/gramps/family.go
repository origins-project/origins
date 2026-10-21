package gramps

import (
	"fmt"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// familyTypes are Gramps' own relationships, in the model's words. Unknown says nothing.
var familyTypes = map[string]string{
	"Married":     model.Married,
	"Unmarried":   model.Unmarried,
	"Civil Union": model.CivilUnion,
	"Unknown":     "",
}

// readFamilies makes a family of the model's of every Gramps family, one that is not a
// couple among them, `unknown` standing for the father or the mother it does not give.
func (c *converter) readFamilies() {
	for _, e := range c.all("families", "family") {
		c.readFamily(e, c.partners(e))
	}
}

// partners are the people a family joins, sorted so that the family's name does not
// depend on which of them Gramps has as the father.
func (c *converter) partners(e *Element) []model.Slug {
	var partners []model.Slug
	for _, role := range []string{"father", "mother"} {
		slug, ok := c.person[e.First(role).Attr("hlink")]
		if !ok {
			continue
		}
		e.Take(role)
		if !slices.Contains(partners, slug) {
			partners = append(partners, slug)
		}
	}
	slices.Sort(partners)
	for len(partners) < 2 {
		partners = append(partners, model.Unknown)
	}
	return partners
}

// readFamily reads a Gramps family into a family of its own. Two Gramps families of one
// couple are two families, the second told apart as any collision is: a couple who
// married twice and one marriage recorded twice look the same from the names.
func (c *converter) readFamily(e *Element, partners []model.Slug) {
	base, err := model.FamilySlug(model.PartnersOf(partners...))
	if err != nil {
		c.notes.Add("a couple's family could not be named and was not imported")
		return
	}
	names := make([]string, len(partners))
	for i, partner := range partners {
		names[i] = string(partner)
	}
	slug := c.namer.Assign(base, c.familyDiscriminators(e), strings.Join(names, " and "))
	if slug == "" {
		c.notes.Add("a couple's family could not be named and was not imported")
		return
	}
	e.Mark()
	c.home(e, slug, "body")
	c.family[e.Attr("handle")] = slug
	family := &model.Family{
		Slug:     slug,
		Partners: model.PartnersOf(partners...),
		Private:  private(e),
		IDs:      c.grampsID(e),
		Type:     typeWord(familyTypes, e.Take("rel").Use("type")),
	}
	family.Children = c.children(e, family)
	c.corpus.Families[slug] = family

	for _, ref := range e.All("eventref") {
		handle := ref.Attr("hlink")
		if _, read := c.events[handle]; !read {
			if c.object(ref, "event") == nil {
				c.notes.Add("an event was referred to that is not in the file")
			}
			continue
		}
		// the partners are whom the family's events are about, not named in them again
		event := c.copyOf(handle, ref, slug, fmt.Sprintf("events[%d].note", len(family.Events)))
		event.With = slices.DeleteFunc(event.With, func(n model.Named) bool {
			return slices.Contains(partners, n.Person)
		})
		family.Events = append(family.Events, event)
		last := len(family.Events) - 1
		c.eventPictures(handle, family, &family.Events[last], last)
	}
	for _, rite := range e.All("lds_ord") {
		at := fmt.Sprintf("events[%d].note", len(family.Events))
		family.Events = append(family.Events, c.rite(rite, slug, at))
	}
	for _, attribute := range e.All("attribute") {
		if c.identifier(&family.IDs, &family.Private, slug, attribute) {
			continue
		}
		at := fmt.Sprintf("events[%d].note", len(family.Events))
		family.Events = append(family.Events, c.fact(attribute, slug, at))
	}

	family.Tags = c.tags(e)
	family.Body = c.body(e)
	family.Sources = c.citations(e)
	c.pictures(e, family, &family.Media, -1)
}

// familyDiscriminators are what tell a Gramps family apart from another of the same
// couple, best first, as model.FamilyHints has them: the year of its first dated event,
// where that happened, and the surname of its first child.
func (c *converter) familyDiscriminators(e *Element) []string {
	var hints []string
	for _, ref := range e.All("eventref") {
		event := c.object(ref, "event")
		if date, _ := dateOf(event); event != nil && date.Year != nil {
			hints = append(hints, itoa(*date.Year))
			if place := c.object(event.First("place"), "placeobj"); place != nil {
				if name := placeName(place); name != "" {
					hints = append(hints, name)
				}
			}
			break
		}
	}
	if child := c.object(e.First("childref"), "person"); child != nil {
		if surname := collapse(child.First("name").First("surname").Text()); surname != "" {
			hints = append(hints, surname)
		}
	}
	return hints
}

// children are a family's children in the order of its childrefs, each in the relation
// its reference gives to the father and to the mother.
//
// A private reference makes the family private, a child's entry having no flag of its
// own.
func (c *converter) children(e *Element, family *model.Family) []model.Child {
	var children []model.Child
	for _, ref := range e.All("childref") {
		slug, ok := c.person[ref.Attr("hlink")]
		if !ok {
			c.notes.Add("a child was referred to that is not in the file")
			continue
		}
		if slices.ContainsFunc(children, func(listed model.Child) bool { return listed.ID == slug }) {
			continue
		}
		ref.Mark()
		c.home(ref, family.Slug, fmt.Sprintf("children[%d].note", len(children)))
		children = append(children, model.Child{
			ID:       slug,
			Relation: c.relationship(e, ref),
			Note:     strings.Join(c.prose(ref), "\n\n"),
			Sources:  c.citations(ref),
		})
		if private(ref) && !family.Private {
			family.Private = true
			c.notes.Add("a private child's entry made private the family listing the child")
		}
	}
	return children
}

// relationship is a child's relation to the father and to the mother, one word where
// the two are the same and left out where that word is birth. A partner the family does
// not give is `unknown`, which a relation is keyed by as any partner is.
func (c *converter) relationship(family, ref *Element) model.Relationship {
	var each []model.PartnerRelation
	for _, role := range []struct{ element, relation string }{{"father", "frel"}, {"mother", "mrel"}} {
		partner, ok := c.person[family.First(role.element).Attr("hlink")]
		if !ok {
			partner = model.Unknown
		}
		each = append(each, model.PartnerRelation{Partner: partner, Relation: relation(ref.Attr(role.relation))})
	}
	ref.Use("frel")
	switch {
	case each[0].Relation != each[1].Relation:
		if each[0].Partner == each[1].Partner {
			// the mother's is left unread
			return model.Relationship{Word: each[0].Relation}
		}
		ref.Use("mrel")
		return model.Relationship{Partners: each}
	}
	ref.Use("mrel")
	if each[0].Relation == model.Birth {
		return model.Relationship{}
	}
	return model.Relationship{Word: each[0].Relation}
}

// relation reads how a partner stands to a child, which Gramps leaves out where it is
// by birth. A sponsor takes on what a guardian does, None says the child is no child of
// that partner, and a custom relation is kept in its own words.
func relation(text string) model.Relation {
	switch text = collapse(text); text {
	case "", "Birth":
		return model.Birth
	case "Adopted":
		return model.Adoptive
	case "Stepchild":
		return model.Step
	case "Foster":
		return model.Foster
	case "Sponsored":
		return model.Guardian
	case "None":
		return model.NoRelation
	case "Unknown":
		return model.RelationUnknown
	}
	return model.Relation(text)
}
