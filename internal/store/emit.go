package store

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Origins is the sole writer of a corpus, so it guarantees canonical output: fields in
// a fixed order, one fact per line, consistent quoting, no reflowing. Plain git blame
// on a person's file then gives fact-level provenance, and a diff shows only what
// actually changed.
//
// That is three separate guarantees, and every choice in this file answers to them:
// deterministic, so the bytes depend on nothing but the content; idempotent, so
// emitting twice changes nothing the first pass did not; lossless, so parsing what was
// emitted gives back the model it came from.

const (
	indentStep = "  "

	// lineWidth is where prose wraps and where a list stops fitting on one line. It is
	// a fixed width rather than a margin measured from the current indentation, so that
	// the same transcription folds the same way wherever it sits.
	lineWidth = 72
)

// emitter builds frontmatter one line at a time. Nesting is done by running a child
// emitter and indenting what it produced, so no writer needs to know how deep it is.
type emitter struct {
	b   strings.Builder
	err error
}

func (e *emitter) fail(err error) {
	if e.err == nil {
		e.err = err
	}
}

func (e *emitter) line(s string) {
	e.b.WriteString(strings.TrimRight(s, " "))
	e.b.WriteByte('\n')
}

// field writes `key: value`. Both sides are quoted only where YAML would otherwise read
// something else back: the key is usually a field name and needs nothing, but inside an
// edit's value it is data like any other.
func (e *emitter) field(key, value string) { e.line(quote(key) + ": " + quote(value)) }

// optional writes the field only when there is something to write. An absent field and
// an empty one decode to the same model, so the shorter form wins.
func (e *emitter) optional(key, value string) {
	if value != "" {
		e.field(key, value)
	}
}

// lines runs write against a fresh emitter and returns what it produced.
func (e *emitter) lines(write func(*emitter)) []string {
	var child emitter
	write(&child)
	if child.err != nil {
		e.fail(child.err)
	}
	out := child.b.String()
	if out == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(out, "\n"), "\n")
}

func (e *emitter) indent(lines []string) {
	for _, line := range lines {
		e.line(indentStep + line)
	}
}

// mapping opens a nested mapping. An empty one is written as {} rather than skipped,
// because a key with nothing under it is still a key — a citation with no transcription
// has to keep its anchor.
func (e *emitter) mapping(key string, write func(*emitter)) {
	body := e.lines(write)
	if len(body) == 0 {
		e.line(quote(key) + ": {}")
		return
	}
	e.line(quote(key) + ":")
	e.indent(body)
}

// item is one entry of a sequence: text to be quoted for wherever it lands, text
// already in YAML form, or the mapping that qualifies it.
type item struct {
	scalar    string
	verbatim  bool
	qualified func(*emitter)
}

func (i item) yaml(flow bool) string {
	switch {
	case i.verbatim:
		return i.scalar
	case flow:
		return quoteFlow(i.scalar)
	default:
		return quote(i.scalar)
	}
}

// sequence writes a list in flow style while every entry is a bare scalar and the line
// they make is short enough to read, and opens a block otherwise. That keeps
// `also-known-as: [Florence]` on one line, gives a parent with a relation the room to
// say so, and stops a long list of citations from being a line that any addition to it
// rewrites end to end.
//
// Entries are never reordered. A list is ordered by definition, and hypothesis edit
// paths are positional, so sorting events would silently invalidate every hypothesis
// pointing into them.
func (e *emitter) sequence(key string, list []item) {
	if len(list) == 0 {
		return
	}
	if !slices.ContainsFunc(list, func(i item) bool { return i.qualified != nil }) {
		parts := make([]string, len(list))
		for i, one := range list {
			parts[i] = one.yaml(true)
		}
		if flow := quote(key) + ": [" + strings.Join(parts, ", ") + "]"; len(flow) <= lineWidth {
			e.line(flow)
			return
		}
	}
	e.line(quote(key) + ":")
	e.indent(e.lines(func(e *emitter) { e.entries(list) }))
}

// entries writes the dashed lines of a block sequence, with the first field of an entry
// sharing its dash's line and the rest lining up under it.
func (e *emitter) entries(list []item) {
	for _, one := range list {
		if one.qualified == nil {
			e.line("- " + one.yaml(false))
			continue
		}
		body := e.lines(one.qualified)
		if len(body) == 0 {
			e.line("- {}")
			continue
		}
		for i, line := range body {
			if i == 0 {
				e.line("- " + line)
			} else {
				e.line(indentStep + line)
			}
		}
	}
}

// prose writes free text: a transcription, a note, a project description. Long text
// folds so that the file stays readable and a diff points at the sentence that changed
// rather than at one enormous line, and text that already has line breaks is written
// literally so that they survive.
func (e *emitter) prose(key, value string) {
	if value == "" {
		return
	}
	head, lines := proseText(value)
	e.line(quote(key) + ": " + head)
	e.indent(lines)
}

// proseText is free text as what follows its key on the key's own line and the lines
// under it, which only a block scalar has.
func proseText(value string) (string, []string) {
	// Line breaks at the end are not lines; they are the block's chomping, and holding
	// them apart is what lets text that merely ends in a newline still fold.
	body := strings.TrimRight(value, "\n")
	switch trailing := len(value) - len(body); {
	case !blockable(body):
		return quote(value), nil
	case strings.Contains(body, "\n"):
		return block("|", strings.Split(body, "\n"), trailing)
	case trailing == 0 && len(body) <= lineWidth:
		return quote(body), nil
	default:
		return block(">", fold(body), trailing)
	}
}

// blockable reports whether a block scalar can carry the text as it stands. A line with
// whitespace at the end of it is lost in a block, and a first line that starts with
// whitespace would need an explicit indentation indicator, and a character that has to be
// escaped can be written only between double quotes. Quoting is uglier and always
// exact, so it takes those — which is why this is decided on the text as it arrived,
// before folding has had a chance to disguise the problem.
func blockable(body string) bool {
	if body == "" || strings.HasPrefix(body, " ") || strings.HasPrefix(body, "\t") ||
		strings.ContainsFunc(body, func(r rune) bool { return r != '\n' && r != '\t' && unprintable(r) }) {
		return false
	}
	return !slices.ContainsFunc(strings.Split(body, "\n"),
		func(line string) bool { return strings.TrimRight(line, " \t") != line })
}

// block lays out a literal or folded scalar. Trailing blank lines cannot be written as
// lines — the parser would read them as belonging to whatever comes next — so they are
// carried by the chomping indicator instead.
func block(style string, lines []string, trailing int) (string, []string) {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
		trailing++
	}
	switch trailing {
	case 0:
		style += "-"
	case 1:
	default:
		style += "+"
	}
	for range trailing - 1 {
		lines = append(lines, "")
	}
	return style, lines
}

// fold breaks a line of prose into lines no wider than lineWidth.
func fold(s string) []string {
	var lines []string
	for {
		at := breakPoint(s)
		if at < 0 {
			return append(lines, s)
		}
		lines = append(lines, s[:at])
		s = s[at+1:]
	}
}

// breakPoint is the space to turn into a line break: the last one that leaves the line
// within lineWidth, or the first one past it when none fits, so that a long word runs
// the line over rather than being cut.
//
// A folded scalar joins its lines back with a single space, so only a space with a
// non-space on either side can be broken at: a run of spaces would come back shorter
// than it went in, and a line starting with one would come back with a line break the
// text never had.
func breakPoint(s string) int {
	if len(s) <= lineWidth {
		return -1
	}
	best := -1
	for i := 1; i < len(s)-1; i++ {
		if s[i] != ' ' || s[i-1] == ' ' || s[i+1] == ' ' {
			continue
		}
		if i > lineWidth && best >= 0 {
			break
		}
		best = i
	}
	return best
}

// value writes an edit's payload. Its shape is not ours to fix: an edit sets a field of
// an entity, so the value is whatever that field holds. It is emitted structurally, and
// mappings in sorted key order because Go randomises map iteration. Text with line
// breaks of its own is a body or a transcription whatever field it is going into, and
// is written as prose is, literally.
func (e *emitter) value(key string, v any) {
	switch v := v.(type) {
	case string:
		if strings.Contains(strings.TrimRight(v, "\n"), "\n") {
			e.prose(key, v)
		} else {
			e.line(quote(key) + ": " + quote(v))
		}
	case []any:
		e.sequence(key, items(v, valueItem))
	case map[string]any:
		e.mapping(key, func(e *emitter) { e.fields(v) })
	default:
		one := valueItem(v)
		if one.qualified != nil {
			one.qualified(e) // an unwritable value, which reports itself
			return
		}
		e.line(quote(key) + ": " + one.yaml(false))
	}
}

func (e *emitter) fields(m map[string]any) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		e.value(k, m[k])
	}
}

// valueItem renders one edit payload. The set of shapes is exactly what the reader
// produces, because every value in the format is text, a list of them, or a mapping of
// them: anything else is a caller who built an edit out of something the corpus cannot
// hold, and saying so is better than guessing at a spelling for it.
func valueItem(v any) item {
	switch v := v.(type) {
	case nil:
		return item{verbatim: true} // an empty value reads back as nothing
	case string:
		return item{scalar: v}
	case []any:
		return item{qualified: func(e *emitter) { e.entries(items(v, valueItem)) }}
	case map[string]any:
		return item{qualified: func(e *emitter) { e.fields(v) }}
	default:
		return item{qualified: func(e *emitter) { e.fail(fmt.Errorf("cannot write a %T", v)) }}
	}
}

// decimal writes a float as the shortest text that reads back as the same value.
func decimal(f float64) string {
	switch {
	case math.IsNaN(f):
		return ".nan"
	case math.IsInf(f, 1):
		return ".inf"
	case math.IsInf(f, -1):
		return "-.inf"
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func dateText(d model.Date) string {
	if d.IsZero() {
		return ""
	}
	return d.String()
}

// Emit renders an entity as the bytes of its file: canonical frontmatter between
// fences, then the markdown body, and exactly one terminating newline.
func Emit(entity any) ([]byte, error) {
	var (
		e    emitter
		body string
	)
	switch v := entity.(type) {
	case *model.Person:
		emitPerson(&e, v)
		body = v.Body
	case *model.Family:
		emitFamily(&e, v)
		body = v.Body
	case *model.Place:
		emitPlace(&e, v)
		body = v.Body
	case *model.Source:
		emitSource(&e, v)
		body = v.Body
	case *model.Repository:
		emitRepository(&e, v)
		body = v.Body
	case *model.Note:
		emitNote(&e, v)
		body = v.Body
	case *model.Hypothesis:
		emitHypothesis(&e, v)
		body = v.Body
	default:
		return nil, fmt.Errorf("cannot emit a %T", entity)
	}
	if e.err != nil {
		return nil, e.err
	}

	var out strings.Builder
	out.WriteString("---\n")
	out.WriteString(e.b.String())
	out.WriteString("---\n")
	if body = strings.Trim(body, "\n"); body != "" {
		out.WriteString("\n")
		out.WriteString(body)
		out.WriteString("\n")
	}
	return []byte(out.String()), nil
}

// EmitProject renders origins.yaml, the one file in a corpus that is plain YAML rather
// than an entity.
func EmitProject(p model.Project) ([]byte, error) {
	var e emitter
	e.mapping("project", func(e *emitter) {
		e.optional("name", p.Name)
		e.prose("description", p.Description)
		e.optional("main-person", string(p.MainPerson))
		e.optional("lang", p.Lang)
		e.optional("copyright", p.Copyright)
		e.sequence("researchers", items(p.Researchers, researcherItem))
		e.sequence("bookmarks", items(p.Bookmarks, bookmarkItem))
		if len(p.Tags) > 0 {
			e.mapping("tags", func(e *emitter) {
				for _, name := range slices.Sorted(maps.Keys(p.Tags)) {
					if strings.TrimSpace(name) == "" {
						e.fail(errors.New(emptyTagName))
						return
					}
					e.mapping(name, func(e *emitter) {
						e.optional("color", p.Tags[name].Color)
						e.optional("priority", p.Tags[name].Priority)
					})
				}
			})
		}
	})
	e.mapping("origins", func(e *emitter) {
		e.line("format-version: " + strconv.Itoa(p.FormatVersion))
	})
	if e.err != nil {
		return nil, e.err
	}
	return []byte(e.b.String()), nil
}

func researcherItem(r model.Researcher) item {
	return item{qualified: func(e *emitter) {
		if err := whoResearches(r); err != nil {
			e.fail(err)
			return
		}
		e.field("name", r.Name)
		e.sequence("addresses", items(r.Addresses, addressItem))
		e.sequence("links", linkItems(r.Links))
		e.prose("note", r.Note)
	}}
}

func bookmarkItem(b model.EntityRef) item {
	text, err := bookmarkText(b)
	if err != nil {
		return item{qualified: func(e *emitter) { e.fail(err) }}
	}
	return item{scalar: text}
}

func emitPerson(e *emitter, p *model.Person) {
	e.flag("private", p.Private)
	e.optional("living", p.Living.String())
	e.name(p.Name)
	e.sequence("also-known-as", items(p.AlsoKnownAs, personNameItem))
	emitSex(e, p.Sex)
	if p.Portrait != (model.Image{}) {
		e.one("portrait", imageItem(p.Portrait))
	}
	e.sequence("events", eventItems(p.Events))
	e.sequence("not", items(p.Not, nonEventItem))
	e.sequence("media", mediaItems(p.Media))
	e.sequence("links", linkItems(p.Links))
	e.sequence("sources", citationItems(p.Sources))
	e.sequence("tags", tagItems(p.Tags))
	e.sequence("same-as", sameAsItems(p.SameAs))
	e.ids(p.IDs)
}

// name writes a name standing for somebody, which takes no flag.
func (e *emitter) name(n model.PersonName) {
	if n.Private {
		e.fail(errFlaggedName)
		return
	}
	if name := personNameItem(n); name.qualified != nil {
		e.mapping("name", name.qualified)
	} else {
		e.optional("name", name.scalar)
	}
}

// flag writes a field that is true or left out.
func (e *emitter) flag(key string, set bool) {
	if set {
		e.field(key, "true")
	}
}

// one writes a field holding a single value in the scalar-or-qualified pattern.
func (e *emitter) one(key string, value item) {
	if value.qualified != nil {
		e.mapping(key, value.qualified)
	} else {
		e.field(key, value.scalar)
	}
}

// emitSex writes the letter, or the letter with what states it. A sex the sources do
// not state is left out, which is what U was always read as.
func emitSex(e *emitter, s model.StatedSex) {
	switch {
	case !s.Qualified():
		if s.Value != model.SexUnknown {
			e.field("sex", s.Value.String())
		}
	case s.Value == model.SexUnknown:
		e.fail(errors.New("sex: " + uncitedSex))
	default:
		e.mapping("sex", func(e *emitter) {
			e.field("value", s.Value.String())
			e.sequence("sources", citationItems(s.Sources))
		})
	}
}

// stated writes what a record gives as its words, or those words with a note and what
// states them, which have something to be about only when there are words.
func (e *emitter) stated(key, value, note string, sources []model.CitationRef, unstated string) {
	switch {
	case note == "" && len(sources) == 0:
		e.optional(key, value)
	case value == "":
		e.fail(errors.New(key + ": " + unstated))
	default:
		e.mapping(key, func(e *emitter) {
			e.field("value", value)
			e.prose("note", note)
			e.sequence("sources", citationItems(sources))
		})
	}
}

func attributeItem(a model.Attribute) item {
	return item{qualified: func(e *emitter) {
		if strings.TrimSpace(a.Type) == "" {
			e.fail(errors.New(untypedAttribute))
			return
		}
		e.field("type", a.Type)
		e.optional("value", a.Value)
		e.optional("date", dateText(a.Date))
		e.prose("note", a.Note)
		e.sequence("sources", citationItems(a.Sources))
	}}
}

func nonEventItem(n model.NonEvent) item {
	return item{qualified: func(e *emitter) {
		if strings.TrimSpace(n.Type) == "" {
			e.fail(errors.New(untypedNonEvent))
			return
		}
		e.field("type", n.Type)
		e.optional("date", dateText(n.Date))
		e.prose("note", n.Note)
		e.sequence("sources", citationItems(n.Sources))
	}}
}

func emitFamily(e *emitter, u *model.Family) {
	e.flag("private", u.Private)
	e.sequence("partners", items(u.Partners, partnerItem))
	e.optional("type", u.Type)
	e.sequence("children", items(u.Children, childItem))
	e.sequence("events", eventItems(u.Events))
	e.sequence("not", items(u.Not, nonEventItem))
	e.sequence("media", mediaItems(u.Media))
	e.sequence("links", linkItems(u.Links))
	e.sequence("sources", citationItems(u.Sources))
	e.sequence("tags", tagItems(u.Tags))
	e.sequence("same-as", sameAsItems(u.SameAs))
	e.ids(u.IDs)
}

func tagItems(list []string) []item {
	return items(list, func(t string) item {
		if strings.TrimSpace(t) == "" {
			return item{qualified: func(e *emitter) { e.fail(errors.New(emptyTag)) }}
		}
		return item{scalar: t}
	})
}

// sameAsItems writes each record a record is said to be as its slug, and as a mapping
// where more is said of it.
func sameAsItems(list []model.SameAs) []item {
	return items(list, func(s model.SameAs) item {
		if err := whoIsSame(s); err != nil {
			return item{qualified: func(e *emitter) { e.fail(err) }}
		}
		if !s.Qualified() {
			return item{scalar: string(s.ID)}
		}
		return item{qualified: func(e *emitter) {
			e.optional("id", string(s.ID))
			e.optional("name", s.Name)
			e.prose("note", s.Note)
			e.sequence("sources", citationItems(s.Sources))
		}}
	})
}

// ids writes the identifiers a record has elsewhere, in the order of the services'
// names, holding each to the rule the reader holds it to. A service with one identifier
// and nothing said of it says so in one word.
func (e *emitter) ids(ids model.IDs) {
	services := slices.DeleteFunc(slices.Sorted(maps.Keys(ids)),
		func(service string) bool { return len(ids[service]) == 0 })
	if len(services) == 0 {
		return
	}
	e.mapping("ids", func(e *emitter) {
		for _, service := range services {
			if err := serviceName(service); err != nil {
				e.fail(err)
				return
			}
			if list := identifierItems(ids[service]); len(list) == 1 && list[0].qualified == nil {
				e.field(service, list[0].scalar)
			} else {
				e.sequence(service, list)
			}
		}
	})
}

func identifierItems(list model.Identifiers) []item {
	return items(list, func(i model.Identifier) item {
		if strings.TrimSpace(i.ID) == "" {
			return item{qualified: func(e *emitter) { e.fail(errors.New(emptyIdentifier)) }}
		}
		if !i.Qualified() {
			return item{scalar: i.ID}
		}
		return item{qualified: func(e *emitter) {
			e.field("id", i.ID)
			e.prose("note", i.Note)
			e.sequence("sources", citationItems(i.Sources))
		}}
	})
}

func linkItems(list []model.Link) []item {
	return items(list, func(l model.Link) item {
		if strings.TrimSpace(l.URL) == "" {
			return item{qualified: func(e *emitter) { e.fail(errors.New(emptyLink)) }}
		}
		if !l.Qualified() {
			return item{scalar: l.URL}
		}
		return item{qualified: func(e *emitter) {
			e.field("url", l.URL)
			e.optional("type", l.Type)
			e.optional("title", l.Title)
			e.prose("note", l.Note)
		}}
	})
}

// addressItem writes an address as its lines where nothing else is said of it, those
// being a block of their own where there is more than one, and as a mapping otherwise.
func addressItem(a model.Address) item {
	switch {
	case a.IsZero():
		return item{qualified: func(e *emitter) { e.fail(errors.New(emptyAddress)) }}
	case a.Qualified():
		return item{qualified: func(e *emitter) { e.address(a) }}
	}
	head, lines := proseText(a.Lines)
	if len(lines) == 0 {
		return item{scalar: a.Lines}
	}
	return item{qualified: func(e *emitter) {
		e.line(head)
		for _, line := range lines {
			e.line(line)
		}
	}}
}

func (e *emitter) address(a model.Address) {
	e.prose("lines", a.Lines)
	e.optional("street", a.Street)
	e.optional("locality", a.Locality)
	e.optional("city", a.City)
	e.optional("parish", a.Parish)
	e.optional("county", a.County)
	e.optional("state", a.State)
	e.optional("postal", a.Postal)
	e.optional("country", a.Country)
	e.optional("date", dateText(a.Date))
	e.sequence("links", linkItems(a.Links))
	e.prose("note", a.Note)
	e.sequence("sources", citationItems(a.Sources))
}

func emitPlace(e *emitter, p *model.Place) {
	e.flag("private", p.Private)
	switch {
	case p.Name.Text != "":
		e.one("name", placeNameItem(p.Name))
	case p.Name.Qualified():
		e.fail(errors.New("name: " + unnamedPlace))
	}
	e.sequence("also-known-as", items(p.AlsoKnownAs, placeNameItem))
	e.optional("type", p.Type)
	// A place within one undated place is the ordinary case, and says so in one word.
	if within := placeItems(p.Within); len(within) == 1 && within[0].qualified == nil {
		e.field("within", within[0].scalar)
	} else {
		e.sequence("within", within)
	}
	if p.Coordinates != nil {
		e.field("coordinates", coordinatesText(*p.Coordinates))
	}
	e.sequence("addresses", items(p.Addresses, addressItem))
	e.optional("from", dateText(p.From))
	e.optional("until", dateText(p.Until))
	e.sequence("media", mediaItems(p.Media))
	e.sequence("links", linkItems(p.Links))
	e.sequence("sources", citationItems(p.Sources))
	e.sequence("tags", tagItems(p.Tags))
	e.sequence("same-as", sameAsItems(p.SameAs))
	e.ids(p.IDs)
}

func emitSource(e *emitter, s *model.Source) {
	e.flag("private", s.Private)
	e.optional("title", s.Title)
	e.optional("short-title", s.ShortTitle)
	e.optional("type", s.Type)
	e.optional("author", s.Author)
	e.optional("publication", s.Publication)
	e.sequence("covers", items(s.Covers, coverageItem))
	e.sequence("repositories", items(s.Repositories, holdingItem))
	e.sequence("attributes", items(s.Attributes, attributeItem))
	e.prose("text", s.Text)
	if len(s.Citations) > 0 {
		e.mapping("citations", func(e *emitter) {
			for _, anchor := range slices.Sorted(maps.Keys(s.Citations)) {
				citation := s.Citations[anchor]
				e.mapping(anchor, func(e *emitter) {
					e.optional("event", citation.Event)
					e.optional("page", citation.Page)
					e.optional("date", dateText(citation.Date))
					e.prose("text", citation.Text)
					e.sequence("with", items(citation.With, namedInItem))
					e.sequence("images", mediaItems(citation.Images))
					e.sequence("links", linkItems(citation.Links))
					e.sequence("attributes", items(citation.Attributes, attributeItem))
					e.prose("note", citation.Note)
					e.sequence("tags", tagItems(citation.Tags))
					e.ids(citation.IDs)
				})
			}
		})
	}
	e.sequence("media", mediaItems(s.Media))
	e.sequence("links", linkItems(s.Links))
	e.sequence("tags", tagItems(s.Tags))
	e.sequence("same-as", sameAsItems(s.SameAs))
	e.ids(s.IDs)
}

func coverageItem(c model.Coverage) item {
	return item{qualified: func(e *emitter) {
		if err := coversSomething(c); err != nil {
			e.fail(err)
			return
		}
		e.sequence("events", items(c.Events, func(word string) item { return item{scalar: word} }))
		e.optional("date", dateText(c.Date))
		e.optional("place", string(c.Place))
		e.prose("note", c.Note)
	}}
}

// holdingItem writes where a source is kept as the repository's slug, and as a mapping
// where more is said of it.
func holdingItem(h model.Holding) item {
	if h.Repository != "" && !h.Qualified() {
		return item{scalar: string(h.Repository)}
	}
	return item{qualified: func(e *emitter) {
		if h.Repository == "" && h.CallNumber == "" {
			e.fail(errors.New(nowhereKept))
			return
		}
		e.optional("repository", string(h.Repository))
		e.optional("call-number", h.CallNumber)
		e.optional("medium", h.Medium)
		e.prose("note", h.Note)
	}}
}

func emitRepository(e *emitter, r *model.Repository) {
	e.flag("private", r.Private)
	e.optional("name", r.Name)
	e.optional("type", r.Type)
	e.sequence("addresses", items(r.Addresses, addressItem))
	e.sequence("media", mediaItems(r.Media))
	e.sequence("links", linkItems(r.Links))
	e.sequence("tags", tagItems(r.Tags))
	e.sequence("same-as", sameAsItems(r.SameAs))
	e.ids(r.IDs)
}

func emitNote(e *emitter, n *model.Note) {
	e.flag("private", n.Private)
	e.optional("title", n.Title)
	e.sequence("events", eventItems(n.Events))
	e.sequence("media", mediaItems(n.Media))
	e.sequence("links", linkItems(n.Links))
	e.sequence("sources", citationItems(n.Sources))
	e.sequence("tags", tagItems(n.Tags))
	e.sequence("same-as", sameAsItems(n.SameAs))
	e.ids(n.IDs)
}

// imageItem writes a picture as its path, and with the region of it that matters where
// that is marked.
func imageItem(i model.Image) item {
	if strings.TrimSpace(i.File) == "" {
		return item{qualified: func(e *emitter) { e.fail(errors.New("an image names the file it is")) }}
	}
	if !i.Qualified() {
		return item{scalar: i.File}
	}
	return item{qualified: func(e *emitter) {
		e.field("file", i.File)
		e.region(*i.Region)
	}}
}

// region writes the four fractions of a picture in the order they are read.
func (e *emitter) region(r model.Region) {
	e.sequence("region", []item{
		{scalar: decimal(r.Left), verbatim: true}, {scalar: decimal(r.Top), verbatim: true},
		{scalar: decimal(r.Right), verbatim: true}, {scalar: decimal(r.Bottom), verbatim: true},
	})
}

// mediaItems writes each of a record's files as its path, and as a mapping where more is
// said of it.
func mediaItems(list []model.Media) []item {
	return items(list, func(m model.Media) item {
		if strings.TrimSpace(m.File) != "" && !m.Qualified() {
			return item{scalar: m.File}
		}
		return item{qualified: func(e *emitter) {
			if strings.TrimSpace(m.File) == "" {
				e.fail(errors.New(unnamedMedia))
				return
			}
			if m.Refers() && m.Says() {
				e.fail(errors.New(referringMedia))
				return
			}
			e.flag("private", m.Private)
			e.field("file", m.File)
			e.optional("type", m.Type)
			e.optional("title", m.Title)
			e.optional("date", dateText(m.Date))
			e.optional("place", string(m.Place))
			if m.Region != nil {
				e.region(*m.Region)
			}
			e.sequence("depicts", items(m.Depicts, depictedItem))
			e.sequence("attributes", items(m.Attributes, attributeItem))
			e.prose("note", m.Note)
			e.sequence("sources", citationItems(m.Sources))
			e.sequence("tags", tagItems(m.Tags))
			e.ids(m.IDs)
		}}
	})
}

// depictedItem writes somebody a picture shows as their slug, and with the part of the
// picture that is them where that is marked.
func depictedItem(d model.Depicted) item {
	if !d.Qualified() {
		return item{scalar: string(d.Person)}
	}
	return item{qualified: func(e *emitter) {
		e.field("person", string(d.Person))
		if d.Region != nil {
			e.region(*d.Region)
		}
		e.sequence("attributes", items(d.Attributes, attributeItem))
		e.prose("note", d.Note)
	}}
}

func emitHypothesis(e *emitter, h *model.Hypothesis) {
	e.optional("status", string(h.Status))
	e.optional("base", h.Base)
	e.optional("confidence", string(h.Confidence))
	e.optional("proposed-by", h.ProposedBy)
	e.sequence("supports", citationItems(h.Supports))
	e.sequence("contradicts", citationItems(h.Contradicts))
	e.sequence("edits", editItems(h.Edits))
}

// personNameItem writes a name of words alone as the words, and a name with anything
// more as a mapping. A name that is only a type or a date is no name.
func personNameItem(n model.PersonName) item {
	if !n.Qualified() {
		return item{scalar: n.Text}
	}
	return item{qualified: func(e *emitter) {
		if n.Text == "" && !n.HasParts() {
			e.fail(fmt.Errorf("a name has its text or at least one of %s", strings.Join(personNameParts, ", ")))
			return
		}
		e.flag("private", n.Private)
		e.optional("text", n.Text)
		e.optional("prefix", n.Prefix)
		e.optional("given", n.Given)
		e.optional("call", n.Call)
		e.optional("nickname", n.Nickname)
		e.optional("family-nickname", n.FamilyNickname)
		e.optional("surname-prefix", n.SurnamePrefix)
		// One surname with nothing said of it is the ordinary case, and is one word.
		if n.Surname.Listed() {
			e.sequence("surname", surnameItems(n.Surname))
		} else if len(n.Surname) == 1 {
			e.field("surname", n.Surname[0].Name)
		}
		e.optional("suffix", n.Suffix)
		e.optional("group", n.Group)
		e.optional("lang", string(n.Lang))
		e.optional("type", n.Type)
		e.optional("display", n.Display)
		e.optional("sort", n.Sort)
		e.optional("from", dateText(n.From))
		e.optional("until", dateText(n.Until))
		e.prose("note", n.Note)
		e.sequence("sources", citationItems(n.Sources))
	}}
}

func surnameItems(list model.Surnames) []item {
	return items(list, func(s model.Surname) item {
		if !s.Qualified() {
			return item{scalar: s.Name}
		}
		return item{qualified: func(e *emitter) {
			e.field("name", s.Name)
			e.optional("prefix", s.Prefix)
			e.optional("connector", s.Connector)
			e.optional("origin", s.Origin)
			e.flag("primary", s.Primary)
		}}
	})
}

func placeNameItem(n model.PlaceName) item {
	if !n.Qualified() {
		return item{scalar: n.Text}
	}
	return item{qualified: func(e *emitter) {
		e.field("name", n.Text)
		e.optional("lang", string(n.Lang))
		e.optional("type", n.Type)
		e.optional("from", dateText(n.From))
		e.optional("until", dateText(n.Until))
		e.prose("note", n.Note)
		e.sequence("sources", citationItems(n.Sources))
	}}
}

// coordinatesText writes two angles shortest-first, or a record's words in parentheses.
func coordinatesText(c model.Coordinates) string {
	if !c.Placed() {
		return "(" + c.Wording + ")"
	}
	return decimal(c.Latitude) + ", " + decimal(c.Longitude)
}

func placeItems(within []model.PlaceRef) []item {
	return items(within, func(p model.PlaceRef) item {
		if !p.Qualified() {
			return item{scalar: string(p.Place)}
		}
		return item{qualified: func(e *emitter) {
			e.field("place", string(p.Place))
			e.optional("from", dateText(p.From))
			e.optional("until", dateText(p.Until))
			e.prose("note", p.Note)
			e.sequence("sources", citationItems(p.Sources))
		}}
	})
}

func eventItems(events []model.Event) []item {
	return items(events, func(ev model.Event) item {
		return item{qualified: func(e *emitter) {
			e.flag("private", ev.Private)
			e.optional("type", ev.Type)
			e.optional("value", ev.Value)
			e.optional("date", dateText(ev.Date))
			e.optional("sort-date", dateText(ev.Sort))
			e.optional("place", string(ev.Place))
			if ev.Address.Qualified() {
				e.mapping("address", func(e *emitter) { e.address(ev.Address) })
			} else {
				e.prose("address", ev.Address.Lines)
			}
			e.stated("age", ev.Age.Value.String(), ev.Age.Note, ev.Age.Sources, unstatedAge)
			e.stated("cause", ev.Cause.Value, ev.Cause.Note, ev.Cause.Sources, unstatedCause)
			e.sequence("with", items(ev.With, namedItem))
			e.optional("family", string(ev.Family))
			e.sequence("attributes", items(ev.Attributes, attributeItem))
			e.sequence("media", mediaItems(ev.Media))
			e.sequence("links", linkItems(ev.Links))
			e.prose("note", ev.Note)
			e.sequence("sources", citationItems(ev.Sources))
			e.sequence("tags", tagItems(ev.Tags))
			e.ids(ev.IDs)
		}}
	})
}

func namedItem(n model.Named) item {
	return item{qualified: func(e *emitter) {
		if err := whoIsNamed(n); err != nil {
			e.fail(err)
			return
		}
		e.optional("person", string(n.Person))
		e.name(n.Name)
		e.field("role", n.Role)
		e.optional("age", n.Age.String())
		e.sequence("attributes", items(n.Attributes, attributeItem))
		e.prose("note", n.Note)
		e.sequence("sources", citationItems(n.Sources))
	}}
}

// namedInItem writes somebody a citation's entry names, who take no sources, the entry
// being what says so.
func namedInItem(n model.Named) item {
	if len(n.Sources) > 0 {
		return item{qualified: func(e *emitter) {
			e.fail(errors.New("somebody a citation names is cited by the citation, and takes no sources"))
		}}
	}
	return namedItem(n)
}

func partnerItem(p model.Partner) item {
	if !p.Qualified() {
		return item{scalar: string(p.Person)}
	}
	return item{qualified: func(e *emitter) {
		if err := whoPartners(p); err != nil {
			e.fail(err)
			return
		}
		e.optional("person", string(p.Person))
		e.name(p.Name)
		e.prose("note", p.Note)
	}}
}

func childItem(c model.Child) item {
	if !c.Qualified() {
		return item{scalar: string(c.ID)}
	}
	return item{qualified: func(e *emitter) {
		if err := whoChild(c); err != nil {
			e.fail(err)
			return
		}
		e.optional("id", string(c.ID))
		e.name(c.Name)
		if c.Relation.Partners != nil {
			// one value, as a region is, and so on one line
			pairs := make([]string, len(c.Relation.Partners))
			for i, r := range c.Relation.Partners {
				pairs[i] = quoteFlow(string(r.Partner)) + ": " + quoteFlow(string(r.Relation))
			}
			e.line("relation: {" + strings.Join(pairs, ", ") + "}")
		} else {
			e.optional("relation", string(c.Relation.Word))
		}
		e.optional("status", string(c.Status))
		e.prose("note", c.Note)
		e.sequence("sources", citationItems(c.Sources))
	}}
}

func citationItems(refs []model.CitationRef) []item {
	return items(refs, func(r model.CitationRef) item { return item{scalar: r.String()} })
}

func refItems(refs []model.EntityRef) []item {
	return items(refs, func(r model.EntityRef) item { return item{scalar: r.String()} })
}

func items[T any](in []T, one func(T) item) []item {
	out := make([]item, 0, len(in))
	for _, v := range in {
		out = append(out, one(v))
	}
	return out
}

// editItems writes the edit vocabulary. The set is closed, and emit_test.go holds this
// switch to model.EditOps, so registering an operation without giving it a shape here
// fails rather than quietly writing an edit with its fields missing.
func editItems(edits []model.Edit) []item {
	return items(edits, func(edit model.Edit) item {
		return item{qualified: func(e *emitter) {
			e.field("op", edit.Op())
			switch op := edit.(type) {
			case model.Set:
				e.field("entity", op.Entity.String())
				e.field("path", op.Path)
				e.value("value", op.Value)
			case model.Unset:
				e.field("entity", op.Entity.String())
				e.field("path", op.Path)
			case model.Add:
				e.field("entity", op.Entity.String())
				e.field("path", op.Path)
				e.value("value", op.Value)
			case model.Remove:
				e.field("entity", op.Entity.String())
				e.field("path", op.Path)
				e.value("value", op.Value)
			case model.Create:
				e.field("entity", op.Entity.String())
				e.mapping("value", func(e *emitter) { e.fields(op.Value) })
			case model.Delete:
				e.field("entity", op.Entity.String())
			case model.Rename:
				e.field("entity", op.Entity.String())
				e.field("to", string(op.To))
			case model.Merge:
				e.field("from", op.From.String())
				e.field("into", op.Into.String())
			case model.Split:
				e.field("entity", op.Entity.String())
				e.sequence("into", refItems(op.Into))
			default:
				e.fail(fmt.Errorf("edit op %q has no shape to write", edit.Op()))
			}
		}}
	})
}
