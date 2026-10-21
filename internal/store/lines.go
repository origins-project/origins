package store

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Canonical serialisation is what makes git blame a fact-level answer: one fact per
// line, in a fixed order, so a line of an entity's file is a thing somebody asserted
// and the commit behind it is when they asserted it. What is missing is the name of
// the fact, and that is what this file supplies — the field path each line is written
// on, in the same language check reports a finding in and a hypothesis writes an edit
// in.
//
// The file says where its own fields are. Nothing here re-derives the emitter's
// layout: the frontmatter is parsed and the parser reports the line each value was
// found on, so a change to how the emitter writes something cannot put this out of
// step with it.

// Span is one field of an entity and the lines of its file that it is written on.
// From and To are line numbers in the file, counting the fences.
type Span struct {
	Path     string
	From, To int
}

// Spans says where in an entity's file every field is written, outermost first and
// otherwise in the order the file writes them.
//
// A field that contains others covers all of their lines as well, so the spans nest
// and a line belongs to the innermost of those covering it. At is that lookup, and is
// the reason the order is worth stating.
func Spans(content []byte) ([]Span, error) {
	frontmatter, _, err := Split(content)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(append([]byte("\n"), frontmatter...), &doc); err != nil {
		return nil, err
	}

	var found []start
	fields(&doc, "", 0, &found)

	// a field that reaches the end of the frontmatter reaches no further, so the
	// closing fence is where the last of them stops
	fence := 2 + bytes.Count(frontmatter, []byte("\n"))
	spans := ended(found, fence-1)
	if body, ok := prose(content, fence+1); ok {
		spans = append(spans, body)
	}
	return spans, nil
}

// ManifestSpans says the same of origins.yaml, which is YAML with no fence. Its fields
// are the ones under project, by the paths an edit of the project gives them; what
// origins keeps under origins is no field of the project's, and a line of it belongs
// to none.
func ManifestSpans(content []byte) ([]Span, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil
	}
	top := doc.Content[0].Content
	for i := 0; i+1 < len(top); i += 2 {
		if top[i].Value != "project" {
			continue
		}
		last := bytes.Count(content, []byte("\n"))
		if i+2 < len(top) {
			last = top[i+2].Line - 1
		}
		var found []start
		fields(top[i+1], "", 0, &found)
		return ended(found, last), nil
	}
	return nil, nil
}

// ended gives each field found the lines it runs over, up to the next field at its own
// level or above, and the last of them up to last.
func ended(found []start, last int) []Span {
	spans := make([]Span, len(found))
	for i, one := range found {
		spans[i] = Span{Path: one.path, From: one.line, To: last}
		for _, next := range found[i+1:] {
			if next.depth <= one.depth {
				spans[i].To = next.line - 1
				break
			}
		}
	}
	return spans
}

// At is the field written on a line: the innermost, since a line inside an event
// belongs to that event's date rather than to the whole list of events. The fences
// belong to no field, and neither does a line of a file nothing has been written into.
func At(spans []Span, line int) (Span, bool) {
	var found Span
	var ok bool
	for _, span := range spans {
		if span.From <= line && line <= span.To {
			found, ok = span, true
		}
	}
	return found, ok
}

// start is a field where the parser found it, with how deep in the frontmatter it sits.
// The depth is what says where the field ends: a field runs until the next one at its
// own level or above, whatever is nested inside it in between.
type start struct {
	path  string
	depth int
	line  int
}

// fields walks a parsed value and names every field inside it, in the order the file
// writes them.
func fields(node *yaml.Node, path string, depth int, found *[]start) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			fields(child, path, depth, found)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			at := key.Value
			if path != "" {
				at = path + "." + key.Value
			}
			// the key is where the field starts; a block value begins on the line
			// after it, and the two are one fact
			*found = append(*found, start{at, depth, key.Line})
			inside(value, at, depth+1, found)
		}
	case yaml.SequenceNode:
		for i, item := range node.Content {
			at := fmt.Sprintf("%s[%d]", path, i)
			*found = append(*found, start{at, depth, item.Line})
			inside(item, at, depth+1, found)
		}
	}
}

// inside walks into a value, except one written in flow style. Everything in
// [rossi-antonio, verdi-lucia] is on the line the list itself is on, and a field is
// worth naming separately only where the file gives it a line of its own.
func inside(node *yaml.Node, path string, depth int, found *[]start) {
	if node.Style&yaml.FlowStyle == 0 {
		fields(node, path, depth, found)
	}
}

// prose is where the markdown body sits, given the line the closing fence is on. It is
// a field like any other — it carries the argument for why one Giovanni is that
// Giovanni — and blame has nothing to say about a file it can only account for half of.
func prose(content []byte, after int) (Span, bool) {
	lines := strings.Split(string(content), "\n")
	body := Span{Path: "body"}
	for i := after; i <= len(lines); i++ {
		if strings.TrimSpace(lines[i-1]) == "" {
			continue
		}
		if body.From == 0 {
			body.From = i
		}
		body.To = i
	}
	return body, body.From > 0
}
