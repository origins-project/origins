package gedcom

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// What an import kept as written, a fenced gedcom block in a note or a body, goes back
// out as the lines it was rather than as note text: under the structure the note
// belongs to, its levels counted from there, a block that starts at level 0 being
// records of its own. The lines above the kept ones that the keeper wrote to say where
// they were, which the export has written again from what was read, are the lines
// already there, so what was kept goes back under them.

// graft is lines kept as written, to go under parent once every record is written, or
// at the top of the file where they are records.
type graft struct {
	parent  *line
	records bool
	lines   []*line
}

// fenceOpening is the first line of a fenced gedcom block, as the keeper writes it.
var fenceOpening = regexp.MustCompile("^(`{3,})gedcom$")

// prose writes text as the NOTE under parent, each gedcom block in it taken out and
// written as its lines.
func (e *exporter) prose(parent *line, text string) *line {
	return parent.text("NOTE", e.unkept(parent, text))
}

// unkept is text without its gedcom blocks, each block's lines to go under parent. A
// block that does not read as GEDCOM lines stays in the text, as words.
func (e *exporter) unkept(parent *line, text string) string {
	if !strings.Contains(text, "```") {
		return text
	}
	lines := strings.Split(text, "\n")
	var out []string
	gap := false // a block was taken out since the last line of text
	for i := 0; i < len(lines); i++ {
		opening := fenceOpening.FindStringSubmatch(lines[i])
		if opening == nil {
			if gap && lines[i] == "" {
				continue
			}
			if gap && len(out) > 0 {
				out = append(out, "") // one blank line where the block stood
			}
			gap = false
			out = append(out, lines[i])
			continue
		}
		end := i + 1
		for end < len(lines) && lines[end] != opening[1] {
			end++
		}
		if end == len(lines) {
			out = append(out, lines[i])
			continue
		}
		read, top, ok := keptLines(lines[i+1 : end])
		if !ok {
			out = append(out, lines[i])
			continue
		}
		// lines for a structure that is not written go with it
		if top == 0 || parent != nil {
			e.grafts = append(e.grafts, graft{parent: parent, records: top == 0, lines: read})
			for _, l := range read {
				e.report.Kept[l.tag]++
			}
		}
		for len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}
		gap, i = true, end
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// keptLines reads the lines of a block as the lines at its least level, each with what
// is under it, and that level; a line that is no GEDCOM line, or one deeper than the
// line above it allows, and the block is not lines at all.
func keptLines(raw []string) ([]*line, int, bool) {
	type read struct {
		level int
		line  *line
	}
	var all []read
	top := -1
	for _, text := range raw {
		level, xref, tag, value, ok := keptLine(text)
		if !ok {
			return nil, 0, false
		}
		all = append(all, read{level, &line{xref: xref, tag: tag, value: value, pointer: true, kept: true}})
		if top < 0 || level < top {
			top = level
		}
	}
	if len(all) == 0 || all[0].level != top {
		return nil, 0, false
	}
	var out []*line
	var stack []read
	for _, r := range all {
		for len(stack) > 0 && stack[len(stack)-1].level >= r.level {
			stack = stack[:len(stack)-1]
		}
		switch {
		case r.level == top:
			out = append(out, r.line)
		case len(stack) == 0 || r.level != stack[len(stack)-1].level+1:
			return nil, 0, false
		default:
			above := stack[len(stack)-1].line
			above.sub = append(above.sub, r.line)
		}
		stack = append(stack, r)
	}
	return out, top, true
}

// keptLine reads a line as written: its level, its xref, and its tag and value spelled
// as the file spelled them, an at-sign still doubled.
func keptLine(text string) (level int, xref, tag, value string, ok bool) {
	level, xref, upper, _, err := split(text)
	if err != nil {
		return 0, "", "", "", false
	}
	rest := strings.TrimLeft(strings.TrimLeft(strings.TrimLeft(text, " \t"), "0123456789"), " ")
	if xref != "" {
		rest = strings.TrimLeft(rest[len(xref)+2:], " ")
	}
	tag, rest = rest[:len(upper)], rest[len(upper):]
	return level, xref, tag, strings.TrimPrefix(rest, " "), true
}

// settle puts every kept line where it goes, once all of records is written: under the
// line it was kept beneath, or at the top, a line the keeper wrote above the kept ones
// being the line written there already. A record kept keeps its xref where nothing
// else written has it, and a pointer to one goes on pointing at it; a pointer to
// anything else is @VOID@, what it pointed at being written under another xref or not
// at all, and is counted.
func (e *exporter) settle(records []*line) []*line {
	root := &line{sub: records}
	for _, g := range e.grafts {
		parent := g.parent
		if g.records {
			parent = root
		}
		for _, l := range g.lines {
			attach(parent, l)
		}
	}

	used := map[string]bool{}
	for _, record := range root.sub {
		if !record.kept {
			used[record.xref] = true
		}
	}
	renamed := map[string]string{}
	fresh := 0
	for _, record := range root.sub {
		if !record.kept || record.xref == "" {
			continue
		}
		xref := record.xref
		for used[xref] {
			fresh++
			xref = "K" + itoa(fresh)
		}
		used[xref] = true
		if _, ok := renamed[record.xref]; !ok {
			renamed[record.xref] = xref
		}
		record.xref = xref
	}
	for _, record := range root.sub {
		record.walk(func(l *line) {
			if !l.kept {
				return
			}
			xref := keptPointer(l.value)
			if xref == "" || xref == "VOID" {
				return
			}
			if to, ok := renamed[xref]; ok {
				l.value = "@" + to + "@"
				return
			}
			l.value = "@VOID@"
			e.omit("kept.pointers")
		})
	}
	return root.sub
}

// attach puts a kept line under parent: where it frames others and parent has the same
// line already, what it frames goes under that one.
func attach(parent *line, l *line) {
	if len(l.sub) > 0 && l.xref == "" {
		for _, written := range parent.sub {
			if !written.kept && strings.EqualFold(written.tag, l.tag) && written.value == unescape(l.value) {
				for _, under := range l.sub {
					attach(written, under)
				}
				return
			}
		}
	}
	parent.sub = append(parent.sub, l)
}

// keptPointer is the xref a value as written points at: an at-sign, an identifier and
// an at-sign, and nothing else.
func keptPointer(value string) string {
	xref := pointer(value)
	if xref == "" || value != "@"+xref+"@" || strings.ContainsAny(xref, "@ #") {
		return ""
	}
	return xref
}

// onlyKept reports whether a note says nothing but its title and records an import kept
// as written, which is what the import makes of a record nothing read.
func onlyKept(n *model.Note) bool {
	if n == nil || !strings.Contains(n.Body, "```") {
		return false
	}
	rest := *n
	rest.Slug, rest.Title, rest.Body = "", "", ""
	if !reflect.ValueOf(rest).IsZero() {
		return false
	}
	scratch := &exporter{report: Report{Kept: map[string]int{}, Omitted: map[string]int{}}}
	if strings.TrimSpace(scratch.unkept(nil, n.Body)) != "" {
		return false
	}
	for _, g := range scratch.grafts {
		if !g.records {
			return false
		}
	}
	return len(scratch.grafts) > 0
}
