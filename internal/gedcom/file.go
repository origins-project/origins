package gedcom

import (
	"fmt"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// readHeader reads what the header says of the tree rather than of the export: its COPR
// as the project's copyright and its LANG as the language the corpus is in. Its NOTE is
// a note record, titled as a shared note is, and what else of the header nothing reads,
// and 5.5.1's SUBN, a submission to a program that no longer takes them, are kept as
// written in it, or in a note record of the header's own where it has none.
func (c *converter) readHeader(records []*Node) {
	var head *Node
	var subn []*Node
	for _, record := range records {
		switch record.Tag {
		case "HEAD":
			if head == nil {
				head = record
			}
		case "SUBN":
			subn = append(subn, record)
		}
	}
	if head == nil {
		return
	}
	c.corpus.Project.Copyright = strings.TrimSpace(head.Take("COPR").Text())
	c.corpus.Project.Lang = string(language(head.First("LANG")))
	if xref := pointer(head.First("SUBM").Text()); c.submitter[xref] {
		head.Take("SUBM")
	}
	c.project(head)

	note := head.First("NOTE")
	if note == nil && len(subn) == 0 {
		return
	}
	heading := title(head)
	if note != nil {
		text := note.Value
		if html(note) {
			text = plain(text)
		}
		text, _, _ = strings.Cut(strings.TrimSpace(text), "\n")
		if text = collapse(text); text != "" {
			heading = text
		}
	}
	base, err := model.Slugify(heading)
	if err != nil {
		base = "note"
	}
	slug := c.namer.Assign(importing.Shorten(base), nil, heading)
	if slug == "" {
		return
	}
	record := &model.Note{Slug: slug, Title: heading}
	c.corpus.Notes[slug] = record
	head.Mark()
	c.citer = slug
	record.Body = strings.Join(c.noteTexts(head), "\n\n")
	c.home(head, slug, "body")
	for _, submission := range subn {
		c.home(submission, slug, "body")
	}
}

// readSubmitters makes each SUBM record an entry of the project's researchers, with its
// address and links and its notes as the entry's note, where what else a submitter
// says — the languages they prefer, a picture of them — is kept as written.
func (c *converter) readSubmitters(records []*Node) {
	for _, record := range records {
		if record.Tag != "SUBM" {
			continue
		}
		name := collapse(record.First("NAME").Text())
		if name == "" {
			continue
		}
		record.Mark().Take("NAME")
		researcher := model.Researcher{Name: name, Links: c.links(record)}
		if at := c.address(record.First("ADDR")); !at.IsZero() {
			researcher.Addresses = []model.Address{at}
		}
		researcher.Note = strings.Join(c.noteTexts(record), "\n\n")
		c.home(record, "", fmt.Sprintf("researchers[%d].note", len(c.corpus.Project.Researchers)))
		c.corpus.Project.Researchers = append(c.corpus.Project.Researchers, researcher)
		c.submitter[record.Xref] = true
	}
}
