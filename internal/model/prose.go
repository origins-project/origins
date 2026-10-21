package model

import (
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Prose hands every piece of prose an entity holds to a function that may change it,
// with the field path it is at: the body, the note of every qualified form, and a
// citation's transcription and note. It is how a rename finds the links to a record in
// somebody's words, how check finds a link to nothing and how todo finds a task, and a
// field added to the model that holds markdown belongs in it.
func Prose(entity any, each func(path string, text *string)) {
	Names(entity, func(at string, n *PersonName) { each(at+".note", &n.Note) })
	links := func(at string, list []Link) {
		for i := range list {
			each(fmt.Sprintf("%slinks[%d].note", at, i), &list[i].Note)
		}
	}
	address := func(at string, a *Address) {
		links(at+".", a.Links)
		each(at+".note", &a.Note)
	}
	addresses := func(list []Address) {
		for i := range list {
			address(fmt.Sprintf("addresses[%d]", i), &list[i])
		}
	}
	identifiers := func(at string, ids IDs) {
		for _, service := range slices.Sorted(maps.Keys(ids)) {
			for i := range ids[service] {
				each(fmt.Sprintf("%sids.%s[%d].note", at, service, i), &ids[service][i].Note)
			}
		}
	}
	attributes := func(at string, list []Attribute) {
		for i := range list {
			each(fmt.Sprintf("%sattributes[%d].note", at, i), &list[i].Note)
		}
	}
	events := func(list []Event) {
		for i := range list {
			at := fmt.Sprintf("events[%d]", i)
			address(at+".address", &list[i].Address)
			each(at+".age.note", &list[i].Age.Note)
			each(at+".cause.note", &list[i].Cause.Note)
			for j := range list[i].With {
				with := fmt.Sprintf("%s.with[%d]", at, j)
				attributes(with+".", list[i].With[j].Attributes)
				each(with+".note", &list[i].With[j].Note)
			}
			attributes(at+".", list[i].Attributes)
			links(at+".", list[i].Links)
			each(at+".note", &list[i].Note)
			identifiers(at+".", list[i].IDs)
		}
	}
	nonEvents := func(list []NonEvent) {
		for i := range list {
			each(fmt.Sprintf("not[%d].note", i), &list[i].Note)
		}
	}
	picture := func(at string, m *Media) {
		for j := range m.Depicts {
			depicted := fmt.Sprintf("%s.depicts[%d]", at, j)
			attributes(depicted+".", m.Depicts[j].Attributes)
			each(depicted+".note", &m.Depicts[j].Note)
		}
		attributes(at+".", m.Attributes)
		each(at+".note", &m.Note)
		identifiers(at+".", m.IDs)
	}
	switch v := entity.(type) {
	case *Person:
		events(v.Events)
		nonEvents(v.Not)
		links("", v.Links)
		identifiers("", v.IDs)
	case *Family:
		for i := range v.Partners {
			each(fmt.Sprintf("partners[%d].note", i), &v.Partners[i].Note)
		}
		for i := range v.Children {
			each(fmt.Sprintf("children[%d].note", i), &v.Children[i].Note)
		}
		events(v.Events)
		nonEvents(v.Not)
		links("", v.Links)
		identifiers("", v.IDs)
	case *Place:
		each("name.note", &v.Name.Note)
		for i := range v.AlsoKnownAs {
			each(fmt.Sprintf("also-known-as[%d].note", i), &v.AlsoKnownAs[i].Note)
		}
		for i := range v.Within {
			each(fmt.Sprintf("within[%d].note", i), &v.Within[i].Note)
		}
		addresses(v.Addresses)
		links("", v.Links)
		identifiers("", v.IDs)
	case *Source:
		for i := range v.Covers {
			each(fmt.Sprintf("covers[%d].note", i), &v.Covers[i].Note)
		}
		for i := range v.Repositories {
			each(fmt.Sprintf("repositories[%d].note", i), &v.Repositories[i].Note)
		}
		each("text", &v.Text)
		attributes("", v.Attributes)
		for _, anchor := range slices.Sorted(maps.Keys(v.Citations)) {
			at := "citations." + anchor
			citation := v.Citations[anchor]
			text, note := citation.Text, citation.Note
			each(at+".text", &text)
			for j := range citation.With {
				with := fmt.Sprintf("%s.with[%d]", at, j)
				attributes(with+".", citation.With[j].Attributes)
				each(with+".note", &citation.With[j].Note)
			}
			links(at+".", citation.Links)
			attributes(at+".", citation.Attributes)
			each(at+".note", &note)
			identifiers(at+".", citation.IDs)
			if text != citation.Text || note != citation.Note {
				citation.Text, citation.Note = text, note
				v.Citations[anchor] = citation
			}
		}
		v.Scans(picture)
		links("", v.Links)
		identifiers("", v.IDs)
	case *Repository:
		addresses(v.Addresses)
		links("", v.Links)
		identifiers("", v.IDs)
	case *Note:
		events(v.Events)
		links("", v.Links)
		identifiers("", v.IDs)
	}
	Pictures(entity, picture)
	if same := SameAsOf(entity); same != nil {
		for i := range *same {
			each(fmt.Sprintf("same-as[%d].note", i), &(*same)[i].Note)
		}
	}
	if body := bodyOf(entity); body != nil {
		each("body", body)
	}
}

// ProseLink is a link or an image in prose whose destination is a path relative to the
// record whose prose it is in, and where that leads.
//
// Start and End are where the destination is written in the text, so that it can be
// written again. Record is the record the link leads into, or zero where it leads out
// of every record's directory; File is the path of a file beside that record, empty for
// the record itself; Anchor is what follows the #.
type ProseLink struct {
	Start, End int
	Record     EntityRef
	File       string
	Anchor     string

	// slug is which of the destination's segments names Record, -1 where none does
	// because the link stays within the directory of the record it is written in.
	slug int
}

// Links finds every relative link in prose written on the record from, in the order
// they are written. A link with a scheme or rooted at / is not relative and is left
// out, and so is anything inside code, which is text about markdown rather than
// markdown.
func Links(from EntityRef, text string) []ProseLink {
	var out []ProseLink
	for _, d := range destinations(text) {
		if link, ok := resolve(from, d.dest); ok {
			link.Start, link.End = d.start, d.end
			out = append(out, link)
		}
	}
	return out
}

// Leads is where one destination written in prose on the record from leads, as Links
// reads it, and false where it is not relative.
func Leads(from EntityRef, dest string) (ProseLink, bool) {
	return resolve(from, dest)
}

var scheme = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// resolve works out where a destination written on from leads.
func resolve(from EntityRef, dest string) (ProseLink, bool) {
	if dest == "" || scheme.MatchString(dest) || strings.HasPrefix(dest, "/") {
		return ProseLink{}, false
	}
	written, anchor, _ := strings.Cut(dest, "#")
	written, _, _ = strings.Cut(written, "?")

	var parts []string
	if written != "" {
		for _, raw := range strings.Split(written, "/") {
			part, err := url.PathUnescape(raw)
			if err != nil {
				part = raw
			}
			parts = append(parts, part)
		}
	}
	stack, escaped := walk(from, parts)
	link := ProseLink{Anchor: anchor, slug: -1}
	if escaped || len(stack) < 2 {
		return link, true
	}
	kind := KindIn(stack[0].name)
	if kind == "" {
		return link, true
	}
	link.Record = EntityRef{Kind: kind, Slug: Slug(stack[1].name)}
	link.slug = stack[1].written
	var file []string
	for _, s := range stack[2:] {
		file = append(file, s.name)
	}
	if f := strings.Join(file, "/"); f != "README.md" {
		link.File = f
	}
	return link, true
}

type segment struct {
	name    string
	written int // which segment of the path wrote it, -1 for the record's own directory
}

// walk follows a relative path from the directory of the record from, one segment at
// a time, to the segments from the corpus root naming where it leads, or reports that
// it leads out of the corpus.
func walk(from EntityRef, parts []string) ([]segment, bool) {
	stack := []segment{{from.Kind.Dir(), -1}, {string(from.Slug), -1}}
	for i, part := range parts {
		switch part {
		case "", ".":
		case "..":
			if len(stack) == 0 {
				return nil, true
			}
			stack = stack[:len(stack)-1]
		default:
			stack = append(stack, segment{part, i})
		}
	}
	return stack, false
}

// Relink hands the record every link in an entity's prose leads into to a function
// that may rename it, and writes the link again where it does. A link within the
// entity's own directory is left as it is, the directory going wherever the entity
// does.
func Relink(entity any, rewrite func(*Slug)) {
	from := RefOf(entity)
	Prose(entity, func(_ string, text *string) {
		links := Links(from, *text)
		for i := len(links) - 1; i >= 0; i-- {
			link := links[i]
			if link.slug < 0 {
				continue
			}
			slug := link.Record.Slug
			if rewrite(&slug); slug == link.Record.Slug {
				continue
			}
			dest := (*text)[link.Start:link.End]
			parts := strings.Split(dest, "/")
			parts[link.slug] = string(slug)
			*text = (*text)[:link.Start] + strings.Join(parts, "/") + (*text)[link.End:]
		}
	})
}

// Task is a task left unticked in prose, by its words.
type Task struct {
	Text string
}

var taskItem = regexp.MustCompile(`^(\s*)(?:[-*+]|\d{1,9}[.)])\s+\[ \]\s+(.*)$`)
var listItem = regexp.MustCompile(`^\s*(?:[-*+]|\d{1,9}[.)])\s`)

// Tasks finds every task left unticked in prose, in the order written. A task's words
// run on over the lines indented under it, as markdown reads them.
func Tasks(text string) []Task {
	var out []Task
	lines := strings.Split(text, "\n")
	code := codeLines(lines)
	for i := 0; i < len(lines); i++ {
		m := taskItem.FindStringSubmatch(lines[i])
		if m == nil || code[i] {
			continue
		}
		words := []string{strings.TrimSpace(m[2])}
		for i+1 < len(lines) && !code[i+1] && indent(lines[i+1]) > len(m[1]) &&
			strings.TrimSpace(lines[i+1]) != "" && !listItem.MatchString(lines[i+1]) {
			i++
			words = append(words, strings.TrimSpace(lines[i]))
		}
		out = append(out, Task{Text: strings.Join(words, " ")})
	}
	return out
}

func indent(line string) int { return len(line) - len(strings.TrimLeft(line, " \t")) }

// codeLines marks the lines of a text that are inside a fenced code block, fences
// included.
func codeLines(lines []string) []bool {
	code := make([]bool, len(lines))
	fence := ""
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if fence != "" {
			code[i] = true
			if strings.HasPrefix(trimmed, fence) && strings.Trim(trimmed, fence[:1]+" \t") == "" {
				fence = ""
			}
			continue
		}
		if indent(line) < 4 {
			for _, mark := range []string{"```", "~~~"} {
				if strings.HasPrefix(trimmed, mark) {
					fence = trimmed[:len(trimmed)-len(strings.TrimLeft(trimmed, mark[:1]))]
					code[i] = true
				}
			}
		}
	}
	return code
}

// destination is where a link, an image or a link definition writes where it leads,
// or a use of a definition by its label, which writes nowhere.
type destination struct {
	start, end int
	dest       string

	from, to int    // the whole of what is written, text and all
	label    string // a definition's label or the one a use names, folded
	use      bool
}

var definition = regexp.MustCompile(`^ {0,3}\[([^\]^][^\]]*)\]:[ \t]*(<[^>\n]*>|\S+)`)

// destinations finds where each inline link, image and link definition in a text
// writes its destination, and each use of a definition, leaving out code.
func destinations(text string) []destination {
	var out []destination
	offset := 0
	lines := strings.SplitAfter(text, "\n")
	bare := make([]string, len(lines))
	for i, line := range lines {
		bare[i] = strings.TrimSuffix(line, "\n")
	}
	code := codeLines(bare)
	var prose strings.Builder
	for i, line := range lines {
		if code[i] {
			prose.WriteString(strings.Repeat(" ", len(bare[i])))
			prose.WriteString(line[len(bare[i]):])
		} else {
			if m := definition.FindStringSubmatchIndex(bare[i]); m != nil {
				d := unbracket(offset+m[4], bare[i][m[4]:m[5]])
				d.from, d.to, d.label = offset, offset+len(line), folded(bare[i][m[2]:m[3]])
				out = append(out, d)
			}
			prose.WriteString(line)
		}
		offset += len(line)
	}
	blank := prose.String()
	for i := 0; i < len(blank); i++ {
		switch blank[i] {
		case '\\':
			i++
		case '`':
			run := len(blank[i:]) - len(strings.TrimLeft(blank[i:], "`"))
			if end := strings.Index(blank[i+run:], blank[i:i+run]); end >= 0 {
				i += run + end + run - 1
			} else {
				i += run - 1
			}
		case ']':
			open := opened(blank[:i])
			if open < 0 {
				continue
			}
			from := open
			if from > 0 && blank[from-1] == '!' {
				from--
			}
			next := byte(0)
			if i+1 < len(blank) {
				next = blank[i+1]
			}
			switch next {
			case '(':
				if d, ok := inline(blank, i+2); ok {
					d.from = from
					out = append(out, d)
				}
			case '[':
				end := strings.IndexByte(blank[i+2:], ']')
				if end < 0 {
					continue
				}
				label := blank[i+2 : i+2+end]
				if strings.TrimSpace(label) == "" {
					label = blank[open+1 : i]
				}
				out = append(out, destination{start: from, from: from, to: i + 3 + end, label: folded(label), use: true})
				i += 2 + end
			case ':':
			default:
				out = append(out, destination{start: from, from: from, to: i + 1, label: folded(blank[open+1 : i]), use: true})
			}
		}
	}
	slices.SortFunc(out, func(a, b destination) int { return a.start - b.start })
	return out
}

// folded is a label as markdown matches it, whatever its case and spacing.
func folded(label string) string {
	return strings.ToLower(strings.Join(strings.Fields(label), " "))
}

// opened is where the [ a ] closes is in the text before it, or -1 where it is not.
func opened(before string) int {
	depth := 0
	for i := len(before) - 1; i >= 0; i-- {
		if i > 0 && before[i-1] == '\\' {
			continue
		}
		switch before[i] {
		case ']':
			depth++
		case '[':
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

// inline reads the destination of an inline link starting at i, just inside its (,
// and where the link ends.
func inline(text string, i int) (destination, bool) {
	for i < len(text) && (text[i] == ' ' || text[i] == '\t') {
		i++
	}
	if i < len(text) && text[i] == '<' {
		end := strings.IndexAny(text[i:], ">\n")
		if end < 0 || text[i+end] != '>' {
			return destination{}, false
		}
		d := unbracket(i, text[i:i+end+1])
		d.to = closing(text, i+end+1)
		return d, true
	}
	start, depth := i, 0
	for ; i < len(text); i++ {
		c := text[i]
		if c == '\\' {
			i++
			continue
		}
		if c == '(' {
			depth++
		} else if c == ')' {
			if depth == 0 {
				break
			}
			depth--
		} else if c == ' ' || c == '\t' || c == '\n' {
			break
		}
	}
	if i >= len(text) || start == i {
		return destination{}, false
	}
	return destination{start: start, end: i, dest: text[start:i], to: closing(text, i)}, true
}

// closing is where an inline link whose destination ends at i ends, past its title and
// its ), or i where it has no ).
func closing(text string, i int) int {
	j := skipSpace(text, i)
	if j < len(text) && strings.IndexByte(`"'(`, text[j]) >= 0 {
		end := text[j]
		if end == '(' {
			end = ')'
		}
		for j++; j < len(text) && text[j] != end; j++ {
			if text[j] == '\\' {
				j++
			}
		}
		j = skipSpace(text, j+1)
	}
	if j < len(text) && text[j] == ')' {
		return j + 1
	}
	return i
}

func skipSpace(text string, i int) int {
	for i < len(text) && strings.IndexByte(" \t\n", text[i]) >= 0 {
		i++
	}
	return i
}

// unbracket is a destination written at start, without the angle brackets it may be
// written in.
func unbracket(start int, written string) destination {
	if strings.HasPrefix(written, "<") && strings.HasSuffix(written, ">") {
		return destination{start: start + 1, end: start + len(written) - 1, dest: written[1 : len(written)-1]}
	}
	return destination{start: start, end: start + len(written), dest: written}
}

// unlink takes out of prose written on the record from every link and image leading
// into a record that is gone, with its text, and every definition leading into one with
// each use of it.
func unlink(from EntityRef, text string, gone func(EntityRef) bool) string {
	all := destinations(text)
	defined, dead := map[string]bool{}, map[string]bool{}
	var cuts []destination
	for _, d := range all {
		if d.use {
			continue
		}
		link, ok := resolve(from, d.dest)
		leads := ok && link.Record != (EntityRef{}) && gone(link.Record)
		if d.label != "" && !defined[d.label] {
			defined[d.label], dead[d.label] = true, leads
		}
		if leads {
			cuts = append(cuts, d)
		}
	}
	for _, d := range all {
		if d.use && dead[d.label] {
			cuts = append(cuts, d)
		}
	}
	if cuts == nil {
		return text
	}
	slices.SortFunc(cuts, func(a, b destination) int { return a.from - b.from })
	var out strings.Builder
	at := 0
	for _, cut := range cuts {
		if cut.to <= at {
			continue
		}
		from, to := max(cut.from, at), cut.to
		// one space is left between the words either side, and none before punctuation
		if from > at && text[from-1] == ' ' {
			if to < len(text) && text[to] == ' ' {
				to++
			} else if to == len(text) || strings.IndexByte("\n,.;:!?)", text[to]) >= 0 {
				from--
			}
		}
		out.WriteString(text[at:from])
		at = to
	}
	out.WriteString(text[at:])
	return out.String()
}
