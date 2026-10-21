package gedcom

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Node is one GEDCOM line together with everything indented beneath it. A record is a
// Node at level zero, and the whole file is a forest of them.
//
// Xref is the record's own identifier and is set only on a record; a cross-reference
// *to* another record arrives as a Value, because that is where GEDCOM puts it.
//
// A node remembers the lines it was written as and whether the converter read it, so
// that whatever nothing read can be put into the corpus as it was written.
type Node struct {
	Line     int
	Xref     string
	Tag      string
	Value    string
	Children []*Node

	read bool
	file *written
	// at is the node's first line in file, own the line after its continuations, end
	// the line after the last line under it.
	at, own, end int
}

// written is a file's lines as they stood, blank lines aside.
type written struct{ lines []string }

// Mark records that the node was read, and returns it.
func (n *Node) Mark() *Node {
	if n != nil {
		n.read = true
	}
	return n
}

// markAll marks the node and everything under it.
func (n *Node) markAll() {
	if n == nil {
		return
	}
	n.read = true
	for _, child := range n.Children {
		child.markAll()
	}
}

// Take is the first child carrying tag, marked as read.
func (n *Node) Take(tag string) *Node { return n.First(tag).Mark() }

// TakeAll is every child carrying tag, each marked as read.
func (n *Node) TakeAll(tag string) []*Node {
	all := n.All(tag)
	for _, node := range all {
		node.Mark()
	}
	return all
}

func (n *Node) Parts() []*Node {
	if n == nil {
		return nil
	}
	return n.Children
}
func (n *Node) Taken() bool     { return n.read }
func (n *Node) Loose() []string { return nil }
func (n *Node) Label() string   { return n.Tag }
func (n *Node) Opening() string { return n.text(n.at, n.own) }
func (n *Node) Closing() string { return "" }
func (n *Node) Whole() string   { return n.text(n.at, n.end) }

func (n *Node) text(from, to int) string {
	if n.file == nil {
		return n.Tag + " " + n.Value + "\n"
	}
	return strings.Join(n.file.lines[from:to], "\n") + "\n"
}

// First is the first child carrying tag, or nil. Returning a nil *Node rather than a
// second result is what lets a caller write n.First("BIRT").First("DATE").Text(): an
// absent subtree answers questions about itself without a check at every step.
func (n *Node) First(tag string) *Node {
	if n == nil {
		return nil
	}
	for _, child := range n.Children {
		if child.Tag == tag {
			return child
		}
	}
	return nil
}

// All is every child carrying tag, in file order.
func (n *Node) All(tag string) []*Node {
	if n == nil {
		return nil
	}
	var out []*Node
	for _, child := range n.Children {
		if child.Tag == tag {
			out = append(out, child)
		}
	}
	return out
}

// Text is the node's value, or "" for an absent node.
func (n *Node) Text() string {
	if n == nil {
		return ""
	}
	return n.Value
}

// Parse reads a whole GEDCOM file into records.
//
// The text is composed as the corpus writes it, in NFC, whatever form the exporter
// wrote it in.
//
// The file is read in one go rather than streamed, because the encoding is declared
// inside the header and a corpus-sized export is a few megabytes. Streaming would mean
// decoding the first lines twice and buys nothing at this size.
func Parse(r io.Reader) ([]*Node, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	text, err := decode(raw)
	if err != nil {
		return nil, err
	}
	return parse(norm.NFC.String(text))
}

// maxLine is generous rather than principled: GEDCOM says to fold a long value across
// CONC lines, and exporters that do not fold can produce a very long one.
const maxLine = 1 << 20

func parse(text string) ([]*Node, error) {
	var (
		records []*Node
		// stack[i] is the node currently open at level i, so a line at level n attaches
		// to stack[n-1] and becomes stack[n].
		stack []*Node
		file  = &written{}
	)

	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 0, 64*1024), maxLine)
	for number := 1; scanner.Scan(); number++ {
		line := strings.TrimRight(scanner.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		file.lines = append(file.lines, line)
		at := len(file.lines) - 1
		level, xref, tag, value, err := split(line)
		if err != nil {
			// A file whose very first line is not a GEDCOM line is usually not a GEDCOM
			// file at all — a download that fetched a web page, most often — and saying
			// which line is wrong invites a hunt for a typo that is not there.
			if len(records) == 0 && len(stack) == 0 {
				return nil, fmt.Errorf("this does not look like a GEDCOM file: line %d: %w", number, err)
			}
			return nil, fmt.Errorf("line %d: %w", number, err)
		}

		// A level that skips a step means an exporter miscounted. Attaching the line
		// one below the deepest open node keeps the rest of the record rather than
		// abandoning it, and no real file nests deeply enough for the guess to matter.
		if level > len(stack) {
			level = len(stack)
		}

		// CONC continues the value of the line above with nothing between, CONT with a
		// newline. Both belong to that line rather than being children of it.
		if (tag == "CONC" || tag == "CONT") && level > 0 && level <= len(stack) {
			parent := stack[level-1]
			if tag == "CONT" {
				parent.Value += "\n"
			}
			parent.Value += value
			if parent.own == at {
				parent.own++
			}
			continue
		}

		node := &Node{Line: number, Xref: xref, Tag: tag, Value: value, file: file, at: at, own: at + 1}
		stack = append(stack[:level], node)
		if level == 0 {
			records = append(records, node)
			continue
		}
		parent := stack[level-1]
		parent.Children = append(parent.Children, node)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	for _, record := range records {
		record.close()
	}
	return records, nil
}

// close works out where each node's lines end, which is where the last line under it
// does.
func (n *Node) close() {
	n.end = n.own
	for _, child := range n.Children {
		child.close()
		n.end = max(n.end, child.end)
	}
}

// split reads one GEDCOM line: a level, an optional cross-reference identifier, a tag,
// and whatever is left.
func split(line string) (level int, xref, tag, value string, err error) {
	// Leading whitespace is not legal and several exporters indent anyway.
	rest := strings.TrimLeft(line, " \t")

	digits, rest := span(rest, func(r byte) bool { return r >= '0' && r <= '9' })
	if digits == "" {
		return 0, "", "", "", fmt.Errorf("%q does not begin with a level", line)
	}
	level, err = strconv.Atoi(digits)
	if err != nil {
		return 0, "", "", "", fmt.Errorf("level %q: %w", digits, err)
	}

	rest = strings.TrimLeft(rest, " ")
	if strings.HasPrefix(rest, "@") {
		end := strings.Index(rest[1:], "@")
		if end < 0 {
			return 0, "", "", "", fmt.Errorf("%q has an unterminated identifier", line)
		}
		xref, rest = rest[1:end+1], strings.TrimLeft(rest[end+2:], " ")
	}

	tag, rest = span(rest, func(r byte) bool { return r != ' ' })
	if tag == "" {
		return 0, "", "", "", fmt.Errorf("%q has no tag", line)
	}
	// Some exporters write the tag in mixed case; the vocabulary is upper case.
	tag = strings.ToUpper(tag)

	// Exactly one space separates the tag from the value, and the rest is data: a
	// transcription may legitimately begin with spaces.
	value = strings.TrimPrefix(rest, " ")
	return level, xref, tag, unescape(value), nil
}

func span(s string, keep func(byte) bool) (taken, rest string) {
	i := 0
	for i < len(s) && keep(s[i]) {
		i++
	}
	return s[:i], s[i:]
}

// unescape undoes the one escape GEDCOM requires in a value. A pointer and a calendar
// escape are single at-signs and survive untouched, as does the undoubled at-sign in
// the email address that half the exports in the world contain.
func unescape(value string) string {
	if !strings.Contains(value, "@@") {
		return value
	}
	return strings.ReplaceAll(value, "@@", "@")
}
