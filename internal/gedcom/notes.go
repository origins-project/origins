package gedcom

import (
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// nameNotes makes a note record of every shared note, 5.5.1's NOTE record and 7.0's
// SNOTE, before anything that points at one is read, titled with its first line. A
// note an export gave an identity is a record of its own however many things it was
// attached to, so that a note two people share is one note both lead to, as it was in
// the file. Its xref names it only inside the file, and is not kept.
func (c *converter) nameNotes(records []*Node) {
	for _, record := range records {
		if record.Xref == "" || record.Tag != "NOTE" && record.Tag != "SNOTE" {
			continue
		}
		heading := record.Value
		if html(record) {
			heading = plain(heading)
		}
		heading, _, _ = strings.Cut(strings.TrimSpace(heading), "\n")
		if heading = collapse(heading); heading == "" {
			heading = title(record)
		}
		base, err := model.Slugify(heading)
		if err != nil {
			base = "note"
		}
		slug := c.namer.Assign(importing.Shorten(base), nil, heading)
		if slug == "" {
			continue
		}
		c.corpus.Notes[slug] = &model.Note{Slug: slug, Title: heading}
		c.note[record.Xref] = slug
	}
}

// readNotes writes what each shared note says into its record, once the sources its
// citations are anchors in are there: its text as any note's is, and its UID, REFN and
// EXID as its ids.
func (c *converter) readNotes(records []*Node) {
	for _, record := range records {
		slug, ok := c.note[record.Xref]
		if !ok || record.Tag != "NOTE" && record.Tag != "SNOTE" {
			continue
		}
		record.Mark()
		c.home(record, slug, "body")
		note := c.corpus.Notes[slug]
		note.Body = c.noteProse(record, slug, "")
		if c.mine {
			// Origins writes a titled note's title as its first paragraph.
			if body, ok := strings.CutPrefix(note.Body, note.Title); ok && (body == "" || strings.HasPrefix(body, "\n\n")) {
				note.Body = strings.TrimPrefix(body, "\n\n")
			}
		}
		note.IDs = c.ids(record)
		note.Links = c.links(record)
		note.Private = c.flagged(record)
		note.Tags = c.tagged(record)
		// Whoever its events name has no slug yet.
		c.afterwards(func() {
			c.readEvents(record, nil, note)
			note.SameAs = c.sameAs(record)
		})
	}
}

// noteTexts is the prose of every NOTE and SNOTE under node, a paragraph or more each,
// marking what it read: a note written in place as noteProse writes it, and a pointer to
// a shared note a link to its record, where the note's text would have stood.
func (c *converter) noteTexts(node *Node) []string { return c.labelled(node, "") }

// labelled is noteTexts with every note led by kind, as a label the language of a note
// written in place joins.
func (c *converter) labelled(node *Node, kind string) []string {
	var out []string
	for _, line := range node.Parts() {
		if line.Tag != "NOTE" && line.Tag != "SNOTE" || voided(line.Value) {
			continue
		}
		if xref := pointer(line.Value); xref != "" {
			if slug, ok := c.note[xref]; ok {
				line.Mark()
				link := recordLink(c.corpus.Notes[slug].Title, model.KindNote, slug, "")
				if kind != "" {
					link = "*" + kind + ":* " + link
				}
				out = append(out, link)
			}
			continue
		}
		switch text := c.noteProse(line, c.citer, kind); {
		case text != "":
			out = append(out, text)
			line.Mark()
		case len(line.Children) == 0:
			line.Mark() // a note with nothing in it says nothing
		}
	}
	return out
}

// noteProse writes a note as markdown, with what it says of itself, as *Prose is
// markdown* has it: its kind and its language leading it as a label, each TRAN after it
// led by its own, a text/html one converted, and what it cites linked at its end, the
// citation an anchor in its source as any other is. subject is the record the note is
// on.
func (c *converter) noteProse(note *Node, subject model.Slug, kind string) string {
	text, lang := c.written(note, kind)
	if text == "" {
		return ""
	}
	paragraphs := []string{text}
	for _, tran := range note.All("TRAN") {
		words := strings.TrimSpace(tran.Value)
		if words == "" {
			continue
		}
		in := lang
		if own := language(tran.First("LANG")); own != "" {
			in = own
		}
		if mime, ok := readMIME(tran.First("MIME")); ok && mime == "text/html" || !ok && html(note) {
			words = markdown(words)
		}
		label := "*Translation:*"
		if in != "" {
			label = "*Translation, " + string(in) + ":*"
		}
		paragraphs = append(paragraphs, label+" "+words)
		tran.Mark()
	}
	var cited []string
	for _, ref := range c.citations(note, subject, "note") {
		cited = append(cited, recordLink(ref.String(), model.KindSource, ref.Source, ref.Anchor))
	}
	if len(cited) > 0 {
		paragraphs = append(paragraphs, strings.Join(cited, " "))
	}
	return strings.Join(paragraphs, "\n\n")
}

// written is the text of a NOTE or a TEXT as markdown, led by kind and its LANG as a
// label, and the language it is in: converted where its MIME says it is HTML, and as
// written otherwise. A MIME that is neither HTML nor plain text is left unread, and so
// kept as written beside it.
func (c *converter) written(node *Node, kind string) (string, model.Language) {
	text := strings.TrimSpace(node.Text())
	if text == "" {
		return "", ""
	}
	if html(node) {
		text = markdown(text)
	}
	readMIME(node.First("MIME"))
	lang := language(node.First("LANG"))
	label := kind
	switch {
	case label != "" && lang != "":
		label += ", " + string(lang)
	case lang != "":
		label = string(lang)
	}
	if label != "" {
		text = "*" + label + ":* " + text
	}
	return text, lang
}

// html reports whether a note's MIME says it is written in HTML.
func html(node *Node) bool {
	mime, _ := mimeOf(node.First("MIME"))
	return mime == "text/html"
}

// readMIME reads a MIME origins knows, text/plain or text/html, marking it read, and
// reports whether there was one.
func readMIME(node *Node) (string, bool) {
	mime, known := mimeOf(node)
	if known {
		node.Mark()
	}
	return mime, known
}

func mimeOf(node *Node) (string, bool) {
	mime, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(node.Text())), ";")
	mime = strings.TrimSpace(mime)
	return mime, mime == "text/plain" || mime == "text/html"
}

// recordLink is a markdown link to a record, or to an anchor in a source, from any other
// record's directory, every one being as deep as every other.
func recordLink(text string, kind model.Kind, slug model.Slug, anchor string) string {
	dest := "../../" + kind.Dir() + "/" + string(slug) + "/README.md"
	if anchor != "" {
		dest += "#" + anchor
	}
	escaped := strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`).Replace(text)
	return "[" + escaped + "](" + dest + ")"
}
