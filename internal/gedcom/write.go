package gedcom

import (
	"strconv"
	"strings"
)

// line is a GEDCOM line to be written, with the lines under it: the tree Parse reads,
// built the other way. A record carries its xref, and a value that points at one is
// written as it is, where any other value has a leading at-sign doubled.
//
// Every method is safe on a nil line, adding nothing, so that a structure that was not
// written takes no substructure either. A line kept is one an import kept as written,
// written again as it was.
type line struct {
	xref, tag, value string
	pointer, kept    bool
	sub              []*line
}

// add writes a line under l, whatever its value.
func (l *line) add(tag, value string) *line {
	if l == nil {
		return nil
	}
	child := &line{tag: tag, value: value}
	l.sub = append(l.sub, child)
	return child
}

// text writes a line under l only where it says something.
func (l *line) text(tag, value string) *line {
	if value == "" {
		return nil
	}
	return l.add(tag, value)
}

// point writes a line under l pointing at the record xref.
func (l *line) point(tag, xref string) *line {
	child := l.add(tag, "@"+xref+"@")
	if child != nil {
		child.pointer = true
	}
	return child
}

// void writes a line under l pointing at nothing, on purpose.
func (l *line) void(tag string) *line { return l.point(tag, "VOID") }

// walk hands every line of the tree to a function, l first.
func (l *line) walk(each func(*line)) {
	each(l)
	for _, child := range l.sub {
		child.walk(each)
	}
}

// emit writes the line at level and everything under it: a value of several lines as
// its first and a CONT for each after it, a carriage return dropped, since 7.0 ends a
// line at one and the corpus writes none, and the newline a block of YAML ends in
// dropped too, being no line of what the value says.
func (l *line) emit(out *strings.Builder, level int) {
	out.WriteString(strconv.Itoa(level))
	if l.xref != "" {
		out.WriteString(" @" + l.xref + "@")
	}
	out.WriteString(" " + l.tag)
	segments := strings.Split(strings.TrimRight(strings.ReplaceAll(l.value, "\r", ""), "\n"), "\n")
	first := segments[0]
	if !l.pointer {
		first = escaped(first)
	}
	if first != "" {
		out.WriteString(" " + first)
	}
	out.WriteByte('\n')
	for _, segment := range segments[1:] {
		out.WriteString(strconv.Itoa(level+1) + " CONT")
		if segment != "" {
			out.WriteString(" " + escaped(segment))
		}
		out.WriteByte('\n')
	}
	for _, child := range l.sub {
		child.emit(out, level+1)
	}
}

// escaped doubles the at-sign a value starts with, which would otherwise read as a
// pointer.
func escaped(value string) string {
	if strings.HasPrefix(value, "@") {
		return "@" + value
	}
	return value
}
