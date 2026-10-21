package gramps

import (
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
	"golang.org/x/text/unicode/norm"
)

// nameNotes makes a note record of every note before anything that refers to one is
// read, titled with the first line of its text. Every Gramps note has an identity and
// may be attached to any number of things, so each is a record of its own that what it
// is attached to links to, and a note two people share is one note both lead to.
func (c *converter) nameNotes() {
	for _, e := range c.all("notes", "note") {
		heading, _, _ := strings.Cut(strings.TrimSpace(norm.NFC.String(e.First("text").Text())), "\n")
		if heading = collapse(heading); heading == "" {
			heading = collapse("note " + e.Attr("id"))
		}
		base, err := model.Slugify(heading)
		if err != nil {
			base = "note"
		}
		slug := c.namer.Assign(importing.Shorten(base), nil, heading)
		if slug == "" {
			c.notes.Add("a note could not be named and was not imported")
			continue
		}
		c.corpus.Notes[slug] = &model.Note{Slug: slug, Title: heading}
		c.note[e.Attr("handle")] = slug
	}
}

// readNotes writes each note into its record once everything a link in it may lead to
// has been read: its text as noteBody writes it, its Gramps ID, its tags and its flag.
func (c *converter) readNotes() {
	for _, e := range c.all("notes", "note") {
		slug, ok := c.note[e.Attr("handle")]
		if !ok {
			continue
		}
		e.Mark()
		c.home(e, slug, "body")
		note := c.corpus.Notes[slug]
		note.Private = private(e)
		note.IDs = c.grampsID(e)
		note.Tags = c.tags(e)
		note.Body = norm.NFC.String(c.noteBody(e))
	}
}

// prose is a link to each note an element refers to, its title the link's text, where
// the note's text would have stood.
func (c *converter) prose(e *Element) []string {
	return c.proseOf(e, func(*Element) bool { return true })
}

// proseOf is prose for the notes which says are wanted, the others left for another
// caller to read.
func (c *converter) proseOf(e *Element, which func(note *Element) bool) []string {
	var links []string
	for _, ref := range e.All("noteref") {
		note := c.object(ref, "note")
		slug, ok := c.note[ref.Attr("hlink")]
		switch {
		case note == nil || !ok:
			if !ref.Taken() {
				c.notes.Add("a note was referred to that is not in the file")
				ref.Mark()
			}
			continue
		case !which(note):
			continue
		}
		ref.Mark()
		if at := c.noteAt[ref.Attr("hlink")]; !slices.Contains(at, e.Name) {
			c.noteAt[ref.Attr("hlink")] = append(at, e.Name)
		}
		links = append(links, recordLink(c.corpus.Notes[slug].Title, model.KindNote, slug, ""))
	}
	return links
}

// transcribed reports whether a note is the words of the record a citation cites, which
// a Transcript and a Source text are; any other note on a citation is about it.
func transcribed(note *Element) bool {
	kind := collapse(note.Attr("type"))
	return kind == "Transcript" || kind == "Source text"
}

// body is the links to the notes an element refers to, as the body of its record.
func (c *converter) body(e *Element) string { return strings.Join(c.prose(e), "\n\n") }

// recordLink is a markdown link to a record, or to an anchor in a source, from any other
// record's directory, every one being as deep as every other.
func recordLink(text string, kind model.Kind, slug model.Slug, anchor string) string {
	escaped := strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`).Replace(text)
	return "[" + escaped + "](" + recordPath(kind, slug, anchor) + ")"
}

func recordPath(kind model.Kind, slug model.Slug, anchor string) string {
	dest := "../../" + kind.Dir() + "/" + string(slug) + "/README.md"
	if anchor != "" {
		dest += "#" + anchor
	}
	return dest
}

// unlabelled are where each of Gramps' types of note says nothing the note's place does
// not: the type Gramps gives a note by where it was made, on what it was made on, and a
// citation's transcription and comment on a citation. A General note says nothing
// anywhere.
var unlabelled = map[string][]string{
	"General": nil, "Unknown": nil, "": nil,
	"Person Note": {"person"}, "Family Note": {"family"}, "Event Note": {"event"},
	"Event Reference Note": {"eventref"}, "Place Note": {"placeobj"},
	"Source Note": {"source"}, "Citation": {"citation"}, "Source Reference Note": {"citation"},
	"Transcript": {"citation"}, "Source text": {"citation"},
	"Repository Note": {"repository"}, "Name Note": {"name"},
	"Attribute Note": {"attribute", "srcattribute"}, "Address Note": {"address"},
	"Child Reference Note": {"childref"}, "Association Note": {"personref"},
	"Repository Reference Note": {"reporef"}, "Media Note": {"object"},
	"Media Reference Note": {"objref"}, "LDS Note": {"lds_ord"},
}

// noteBody writes a note as *Prose is markdown* has it. A type saying something of the
// note leads it as a label, in Gramps' words for its own and as typed for a custom one;
// one naming where the note was made is not written where the note is attached there,
// and a General note says nothing. A To Do note is a task, and an Html code note a
// fenced block marked html, its text as written.
func (c *converter) noteBody(note *Element) string {
	kind := collapse(note.Use("type"))
	if kind == "Html code" {
		if format := note.Attr("format"); format == "0" || format == "1" {
			note.Use("format")
		}
		return fencedAs("html", note.Take("text").Text())
	}
	text := strings.TrimSpace(c.markdown(note))
	where, labelled := unlabelled[kind]
	switch {
	case text == "":
		return ""
	case kind == "To Do":
		return task(text)
	case !labelled || len(where) > 0 && !slices.ContainsFunc(where, func(at string) bool {
		return slices.Contains(c.noteAt[note.Attr("handle")], at)
	}):
		if strings.HasPrefix(text, "```") || strings.HasPrefix(text, "<pre>") {
			return "*" + kind + ":*\n\n" + text
		}
		return "*" + kind + ":* " + text
	}
	return text
}

// task is text as a task left to do, its later lines indented under the first.
func task(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines[1:] {
		if line != "" {
			lines[i+1] = "  " + line
		}
	}
	return "- [ ] " + strings.Join(lines, "\n")
}

// layers are the styles Gramps puts on the characters of a note, outermost first, in the
// order they are opened around the same characters and closed again.
var layers = []string{
	"link", "fontface", "fontsize", "fontcolor", "highlight", "underline", "superscript",
	"subscript", "strikethrough", "bold", "italic",
}

// mark is what styles one character of a note: the value of each layer on it, "" where
// it is not.
type mark [11]string

// markdown writes a note as markdown. Gramps keeps a note as its text and the ranges of
// it each style covers, counted in characters. Bold, italic, struck through and a link
// become markdown around the same characters, and underlining, superscript and subscript,
// a highlight, a colour, a typeface and a size become HTML in it; a link to a record in
// the tree, which Gramps names by its handle, is a relative link to the record's file. A
// preformatted note is kept as it was written, fenced, or as a <pre> block with its
// styles in HTML where it has any. A style whose value cannot be written, and a link to
// something not in the tree, are left unread.
func (c *converter) markdown(note *Element) string {
	text := []rune(note.Take("text").Text())
	format := note.Attr("format")
	if format == "0" || format == "1" {
		note.Use("format")
	}
	preformatted := format == "1"
	marks := make([]mark, len(text))
	anyStyle := false
	for _, style := range note.All("style") {
		name := style.Attr("name")
		layer := slices.Index(layers, name)
		value, ok := c.styleValue(name, strings.TrimSpace(style.Attr("value")))
		if layer < 0 || !ok {
			continue
		}
		style.Mark()
		style.Use("name")
		style.Use("value")
		for _, r := range style.All("range") {
			start, err1 := strconv.Atoi(r.Attr("start"))
			end, err2 := strconv.Atoi(r.Attr("end"))
			if err1 != nil || err2 != nil {
				continue
			}
			r.Mark()
			r.Use("start")
			r.Use("end")
			for i := max(start, 0); i < min(end, len(text)); i++ {
				marks[i][layer] = value
				anyStyle = true
			}
		}
	}
	switch {
	case preformatted && anyStyle:
		return "<pre>" + styled(text, marks, true) + "</pre>"
	case preformatted:
		return fenced(string(text))
	}
	return styled(text, marks, false)
}

// styleValue is what a style carries as the value of its layer: where a link leads, the
// colour, typeface or size, and "1" for a style that is on or off.
func (c *converter) styleValue(name, value string) (string, bool) {
	switch name {
	case "link":
		if strings.HasPrefix(value, "gramps://") {
			return c.treeLink(value)
		}
		return value, value != ""
	case "fontsize":
		if n, err := strconv.Atoi(value); err != nil || n <= 0 {
			return "", false
		}
		return value, true
	case "fontface", "fontcolor", "highlight":
		return value, value != ""
	}
	return "1", true
}

// treeLink is where a link Gramps makes to an object of the tree, gramps://Person/handle/
// and the handle, leads in the corpus: the record's file, a citation's anchor in its
// source, the file of the record holding an event or keeping a picture, and a note's own.
func (c *converter) treeLink(value string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(value, "gramps://"), "/")
	if len(parts) != 3 || parts[1] != "handle" {
		return "", false
	}
	// a file writes a handle after an underscore, and a link to it without one
	handle := parts[2]
	if c.handles["_"+handle] != nil {
		handle = "_" + handle
	}
	var slug model.Slug
	var kind model.Kind
	var ok bool
	switch parts[0] {
	case "Person":
		slug, ok = c.person[handle]
		kind = model.KindPerson
	case "Family":
		slug, ok = c.family[handle]
		kind = model.KindFamily
	case "Place":
		slug, ok = c.place[handle]
		kind = model.KindPlace
	case "Source":
		slug, ok = c.source[handle]
		kind = model.KindSource
	case "Repository":
		slug, ok = c.repository[handle]
		kind = model.KindRepository
	case "Note":
		slug, ok = c.note[handle]
		kind = model.KindNote
	case "Citation":
		ref, cited := c.citation[handle]
		if !cited {
			return "", false
		}
		return recordPath(model.KindSource, ref.Source, ref.Anchor), true
	case "Event":
		if slug, ok = c.eventAt[handle]; ok {
			kind = model.RefOf(c.corpus.Entity(slug)).Kind
		}
	case "Media":
		var picture *kept
		if picture, ok = c.kept[handle]; ok {
			slug, kind = picture.record.Slug, picture.record.Kind
		}
	}
	if !ok {
		return "", false
	}
	return recordPath(kind, slug, ""), true
}

// opening and closing are what a layer is opened and closed with, as markdown or, inside
// a <pre> block, as HTML.
func opening(layer int, value string, html bool) (string, string) {
	quoted := strings.NewReplacer("&", "&amp;", `"`, "&quot;", "<", "&lt;").Replace(value)
	span := func(style string) (string, string) { return `<span style="` + style + `">`, "</span>" }
	switch layers[layer] {
	case "link":
		if html {
			return `<a href="` + quoted + `">`, "</a>"
		}
		return "[", "](" + destination(value) + ")"
	case "bold":
		if html {
			return "<b>", "</b>"
		}
		return "**", "**"
	case "italic":
		if html {
			return "<i>", "</i>"
		}
		return "*", "*"
	case "strikethrough":
		if html {
			return "<s>", "</s>"
		}
		return "~~", "~~"
	case "underline":
		return "<u>", "</u>"
	case "superscript":
		return "<sup>", "</sup>"
	case "subscript":
		return "<sub>", "</sub>"
	case "highlight":
		return span("background-color: " + quoted)
	case "fontcolor":
		return span("color: " + quoted)
	case "fontface":
		return span("font-family: " + quoted)
	case "fontsize":
		return span("font-size: " + quoted + "pt")
	}
	return "", ""
}

// styled writes text with the styles marks says, each layer closed where it ends and
// opened again where it is wanted, inner layers closed and opened again around a change
// to an outer one. In markdown, whitespace is written outside the marks around it, since
// markdown does not take emphasis that opens or closes on a space, and every line closes
// what is open, since it does not take emphasis that crosses a line either; inside a
// <pre> block the text is HTML and its whitespace is the text's.
func styled(text []rune, marks []mark, html bool) string {
	var out strings.Builder
	var open mark
	pending := ""
	change := func(want mark) {
		first := 0
		for first < len(layers) && open[first] == want[first] {
			first++
		}
		for layer := len(layers) - 1; layer >= first; layer-- {
			if open[layer] != "" {
				_, closing := opening(layer, open[layer], html)
				out.WriteString(closing)
				open[layer] = ""
			}
		}
		out.WriteString(pending)
		pending = ""
		for layer := first; layer < len(layers); layer++ {
			if want[layer] != "" {
				opened, _ := opening(layer, want[layer], html)
				out.WriteString(opened)
				open[layer] = want[layer]
			}
		}
	}
	for i, r := range text {
		want := marks[i]
		if !html {
			if r == '\n' {
				want = mark{}
			} else if unicode.IsSpace(r) {
				pending += string(r)
				continue
			}
		}
		if want != open {
			change(want)
		}
		out.WriteString(pending)
		pending = ""
		if html {
			out.WriteString(strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(string(r)))
		} else {
			out.WriteRune(r)
		}
	}
	change(mark{})
	out.WriteString(pending)
	return out.String()
}

// destination is a link's address as markdown takes it, in angle brackets where it has
// what would otherwise end it.
func destination(link string) string {
	if strings.ContainsAny(link, " ()<>") {
		return "<" + strings.NewReplacer("<", "%3C", ">", "%3E").Replace(link) + ">"
	}
	return link
}

// fenced is text in a block markdown shows as it is written, with a fence longer than
// any run of backticks inside it.
func fenced(text string) string { return fencedAs("", text) }

// fencedAs is text fenced, the fence marked with the language it is in.
func fencedAs(language, text string) string {
	longest, run := 0, 0
	for _, r := range text {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	fence := strings.Repeat("`", max(3, longest+1))
	return fence + language + "\n" + strings.TrimRight(text, "\n") + "\n" + fence
}
