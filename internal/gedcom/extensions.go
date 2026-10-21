package gedcom

import (
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// schema is the extension tags a header's SCHMA maps to where origins documents its
// own, which are origins' and mean what origins wrote them to mean. 7.0 gives an
// extension tag its meaning only through the URI SCHMA maps it to, so another program's
// _TYPE is not origins' and is kept as written. It reads the header without marking it,
// Describe being what marks it.
func schema(records []*Node) map[string]bool {
	ours := map[string]bool{}
	for _, record := range records {
		if record.Tag != "HEAD" {
			continue
		}
		for _, tag := range record.First("SCHMA").All("TAG") {
			name, uri, _ := strings.Cut(collapse(tag.Text()), " ")
			if strings.HasPrefix(name, "_") && strings.HasPrefix(uri, extensions) {
				ours[strings.ToUpper(name)] = true
			}
		}
		break
	}
	return ours
}

// writtenByOrigins reports a file origins wrote, as its header names the program, whose text is
// what origins wrote down to its whitespace and whose shared notes begin with a title.
func writtenByOrigins(records []*Node) bool {
	for _, record := range records {
		if record.Tag == "HEAD" {
			return strings.TrimSpace(record.First("SOUR").Text()) == "ORIGINS"
		}
	}
	return false
}

// take is the first line tag under node, marked read, where tag is one of origins'.
func (c *converter) take(node *Node, tag string) *Node {
	if !c.ours[tag] {
		return nil
	}
	return node.Take(tag)
}

// takeAll is every line tag under node, marked read, where tag is one of origins'.
func (c *converter) takeAll(node *Node, tag string) []*Node {
	if !c.ours[tag] {
		return nil
	}
	return node.TakeAll(tag)
}

// extension is the words of origins' tag under node, marked read.
func (c *converter) extension(node *Node, tag string) string {
	return collapse(c.take(node, tag).Text())
}

// flagged reads the _PRIVATE origins writes where 7.0 has no RESN, on a source, a
// repository, a shared note, a name and an association.
func (c *converter) flagged(node *Node) bool {
	return strings.EqualFold(c.extension(node, "_PRIVATE"), "Y")
}

// tagged reads the researcher's words a record, an event, a citation or a picture is
// tagged with.
func (c *converter) tagged(node *Node) []string {
	var out []string
	for _, tag := range c.takeAll(node, "_TAG") {
		if word := collapse(tag.Text()); word != "" {
			out = append(out, word)
		}
	}
	return out
}

// typed reads the _TYPE origins writes for the kind of a family, a source and a
// repository.
func (c *converter) typed(node *Node) string { return c.extension(node, "_TYPE") }

// attributes reads what no field holds, each _ATTRIBUTE its value, its TYPE, its DATE,
// its notes and its citations, which subject makes.
func (c *converter) attributes(node *Node, subject model.Slug) []model.Attribute {
	var out []model.Attribute
	for _, line := range c.takeAll(node, "_ATTRIBUTE") {
		date, read := readDateNode(line.Take("DATE"))
		if !read {
			c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
		}
		out = append(out, model.Attribute{
			Type:    collapse(line.Take("TYPE").Text()),
			Value:   strings.TrimSpace(line.Text()),
			Date:    date,
			Note:    strings.Join(c.noteTexts(line), "\n\n"),
			Sources: c.citations(line, subject, "attribute"),
		})
	}
	return out
}

// sameAs reads the _SAME_AS origins writes for the records of a kind other than a
// person's said to be the same as this one: a pointer to the record, or the words naming
// one the file does not have, with its note. It is read once every record has its slug.
func (c *converter) sameAs(node *Node) []model.SameAs {
	var out []model.SameAs
	for _, line := range c.takeAll(node, "_SAME_AS") {
		claim := model.SameAs{Note: strings.Join(c.noteTexts(line), "\n\n")}
		if xref := pointer(line.Value); xref != "" {
			slug, ok := c.slugOf(xref)
			if !ok {
				c.notes.Add("a record is said to be the same as %s, which is not in the file", xref)
				continue
			}
			claim.ID = slug
		} else if claim.Name = collapse(line.Value); claim.Name == "" {
			continue
		}
		out = append(out, claim)
	}
	return out
}

// slugOf is the record a cross-reference of the file was made into.
func (c *converter) slugOf(xref string) (model.Slug, bool) {
	for _, made := range []map[string]model.Slug{c.person, c.family, c.source, c.repository, c.note} {
		if slug, ok := made[xref]; ok {
			return slug, true
		}
	}
	return "", false
}

// refOf is the record an xref names, of whichever kind it was made.
func (c *converter) refOf(xref string) (model.EntityRef, bool) {
	for kind, made := range map[model.Kind]map[string]model.Slug{
		model.KindPerson: c.person, model.KindFamily: c.family, model.KindSource: c.source,
		model.KindRepository: c.repository, model.KindNote: c.note,
	} {
		if slug, ok := made[xref]; ok {
			return model.EntityRef{Kind: kind, Slug: slug}, true
		}
	}
	return model.EntityRef{}, false
}

// project reads what origins writes of the manifest into the header: its name and
// description, the main person, the bookmarks, a place's as its PLAC, and each tag's
// style. The pointers are followed once every record has its slug.
func (c *converter) project(head *Node) {
	project := &c.corpus.Project
	project.Name = c.extension(head, "_NAME")
	if words := c.take(head, "_DESCRIPTION"); words != nil {
		project.Description = strings.TrimSpace(words.Text())
	}
	for _, styled := range c.takeAll(head, "_TAG_STYLE") {
		if project.Tags == nil {
			project.Tags = map[string]model.TagStyle{}
		}
		project.Tags[styled.Text()] = model.TagStyle{
			Color:    collapse(c.take(styled, "_COLOR").Text()),
			Priority: collapse(c.take(styled, "_PRIORITY").Text()),
		}
	}
	c.afterwards(func() {
		if main := head.First("_MAIN_PERSON"); main != nil && c.ours[main.Tag] {
			if slug, ok := c.person[pointer(main.Value)]; ok {
				main.Mark()
				project.MainPerson = slug
			}
		}
		for _, bookmark := range head.All("_BOOKMARK") {
			if !c.ours[bookmark.Tag] {
				break
			}
			ref, ok := c.refOf(pointer(bookmark.Value))
			if place := c.place(bookmark.First("PLAC")); !ok && place != "" {
				ref, ok = model.EntityRef{Kind: model.KindPlace, Slug: place}, true
			}
			if ok {
				bookmark.Mark()
				project.Bookmarks = append(project.Bookmarks, ref)
			}
		}
	})
}

// afterwards defers what is read once every record has its slug, in the order it was asked.
func (c *converter) afterwards(read func()) { c.later = append(c.later, read) }

// fractions reads the _REGION origins writes where a picture's size was not known, the
// left, top, right and bottom of the part meant as fractions of the picture.
func (c *converter) fractions(node *Node) *model.Region {
	if !c.ours["_REGION"] {
		return nil
	}
	line := node.First("_REGION")
	parts := strings.Fields(line.Text())
	if len(parts) != 4 {
		return nil
	}
	var f [4]float64
	for i, part := range parts {
		value, err := strconv.ParseFloat(part, 64)
		if err != nil || value < 0 || value > 1 {
			return nil
		}
		f[i] = value
	}
	line.Mark()
	return &model.Region{Left: f[0], Top: f[1], Right: f[2], Bottom: f[3]}
}

// portrait reads the _PORTRAIT origins writes for the picture a person is shown by: the
// picture the OBJE it points at was made into, from the person's directory, and the part
// of it its CROP or _REGION marks. It is read once every record has its pictures, since
// the record keeping the picture may be read after the person.
func (c *converter) portrait(record *Node, person *model.Person) {
	line := c.take(record, "_PORTRAIT")
	if line == nil {
		return
	}
	c.afterwards(func() {
		shared, ok := c.pictures[pointer(line.Value)]
		if !ok || len(shared.entries) == 0 {
			c.notes.Add("a portrait names a picture that is not in the file")
			return
		}
		file := (*shared.list())[shared.entries[0]].File
		if ref := model.RefOf(person); shared.record != ref && model.Locate(shared.record, file).Record == shared.record {
			file = "../../" + shared.record.Kind.Dir() + "/" + string(shared.record.Slug) + "/" + file
		}
		person.Portrait = model.Image{File: file}
		var attributes []model.Attribute
		c.cropped(&person.Portrait.Region, &attributes, line, c.files.Found(shared.record, (*shared.list())[shared.entries[0]].File))
	})
}
