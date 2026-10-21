package gramps

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// Element is one XML element together with everything inside it. The whole export is
// one, <database>, and everything Gramps keeps is a descendant of it.
//
// Value is the text inside an element that has no children. Gramps' DTD has no mixed
// content, so the whitespace between the children of any other element is layout and is
// not kept.
//
// The converter marks an element and each of its attributes as it reads them, so that
// whatever it did not read can be kept as the file wrote it. First, All, Attr and Text
// only look; Take, TakeAll, Use and Mark read.
type Element struct {
	Name     string
	Attrs    map[string]string
	Value    string
	Children []*Element

	order []string        // the attributes, as written
	used  map[string]bool // the attributes read
	read  bool

	file        []byte
	open, close [2]int // where the start tag and the end tag are in file
}

// First is the first child called name, or nil. Returning a nil *Element rather than a
// second result is what lets a caller write e.First("name").First("first").Text(): an
// absent subtree answers questions about itself without a check at every step.
func (e *Element) First(name string) *Element {
	if e == nil {
		return nil
	}
	for _, child := range e.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

// All is every child called name, in file order.
func (e *Element) All(name string) []*Element {
	if e == nil {
		return nil
	}
	var out []*Element
	for _, child := range e.Children {
		if child.Name == name {
			out = append(out, child)
		}
	}
	return out
}

// Attr is the value of the attribute called name, or "" where there is none.
func (e *Element) Attr(name string) string {
	if e == nil {
		return ""
	}
	return e.Attrs[name]
}

// Text is the element's text as written, or "" for an absent element.
func (e *Element) Text() string {
	if e == nil {
		return ""
	}
	return e.Value
}

// Mark records that the element was read, and hands it back.
func (e *Element) Mark() *Element {
	if e != nil {
		e.read = true
	}
	return e
}

// markAll marks the element and everything inside it read, for what is dropped
// knowingly.
func (e *Element) markAll() {
	e.walk(func(e *Element) {
		e.read = true
		for _, name := range e.order {
			e.used[name] = true
		}
	})
}

// blank reports whether the element says nothing: no attribute and no text, in it or in
// anything inside it.
func (e *Element) blank() bool {
	blank := true
	e.walk(func(e *Element) {
		if len(e.order) > 0 || strings.TrimSpace(e.Value) != "" {
			blank = false
		}
	})
	return blank
}

// Take is the first child called name, marked read.
func (e *Element) Take(name string) *Element { return e.First(name).Mark() }

// TakeAll is every child called name, each marked read.
func (e *Element) TakeAll(name string) []*Element {
	all := e.All(name)
	for _, child := range all {
		child.Mark()
	}
	return all
}

// Use is the value of the attribute called name, marked read.
func (e *Element) Use(name string) string {
	if e == nil {
		return ""
	}
	if _, ok := e.Attrs[name]; ok {
		e.used[name] = true
	}
	return e.Attrs[name]
}

// Parts is what is inside the element, in file order.
func (e *Element) Parts() []*Element { return e.Children }

// Taken reports whether the element was read.
func (e *Element) Taken() bool { return e.read }

// Loose labels each attribute of the element that was not read.
func (e *Element) Loose() []string {
	var out []string
	for _, name := range e.order {
		if !e.used[name] {
			out = append(out, name+" on <"+e.Name+">")
		}
	}
	return out
}

// Label is the element as the report counts it.
func (e *Element) Label() string { return "<" + e.Name + ">" }

// Opening is the element's start tag on a line of its own, or the whole of an element
// with nothing inside it but text, which is what has to be shown of it for its
// attributes to be read in their place.
func (e *Element) Opening() string {
	if len(e.Children) == 0 {
		return e.Whole()
	}
	return e.line(e.open[0], e.open[1])
}

// Closing is the element's end tag on a line of its own, or nothing where Opening is
// the whole of it.
func (e *Element) Closing() string {
	if len(e.Children) == 0 || e.close[0] == e.close[1] {
		return ""
	}
	return e.line(e.close[0], e.close[1])
}

// Whole is the element and everything inside it, as written.
func (e *Element) Whole() string { return e.line(e.open[0], max(e.open[1], e.close[1])) }

// line is the file from start to end, with the indentation before start where nothing
// else comes before it on its line, and a newline after, composed as every value is.
func (e *Element) line(start, end int) string {
	from := start
	for from > 0 && (e.file[from-1] == ' ' || e.file[from-1] == '\t') {
		from--
	}
	if from > 0 && e.file[from-1] != '\n' && e.file[from-1] != '\r' {
		from = start
	}
	text := string(bytes.ReplaceAll(e.file[from:end], []byte("\r\n"), []byte("\n")))
	return norm.NFC.String(text) + "\n"
}

// bare is the element without what is inside it, sharing its marks, for what its own
// start tag says apart from the elements in it.
func (e *Element) bare() *Element {
	return &Element{
		Name: e.Name, Attrs: e.Attrs, order: e.order, used: e.used, read: e.read,
		file: e.file, open: e.open, close: [2]int{e.open[1], e.open[1]},
	}
}

// walk calls visit on e and on everything inside it, in file order.
func (e *Element) walk(visit func(*Element)) {
	if e == nil {
		return
	}
	visit(e)
	for _, child := range e.Children {
		child.walk(visit)
	}
}

// parse reads a whole XML document into its root element. Names are kept without their
// namespace, which Gramps writes once, as the root's xmlns attribute.
//
// Every value is composed as the corpus writes it, in NFC, except a note's text, whose
// styles are ranges over the characters as written; the note is composed once they have
// been applied.
func parse(r io.Reader) (*Element, error) {
	file, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	decoder := xml.NewDecoder(bytes.NewReader(file))
	var root *Element
	var open []*Element
	for {
		start := int(decoder.InputOffset())
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		end := int(decoder.InputOffset())
		switch t := token.(type) {
		case xml.StartElement:
			element := &Element{
				Name:  t.Name.Local,
				Attrs: make(map[string]string, len(t.Attr)),
				used:  map[string]bool{},
				file:  file,
				open:  [2]int{start, end},
			}
			for _, attr := range t.Attr {
				if attr.Name.Space == "" {
					element.Attrs[attr.Name.Local] = norm.NFC.String(attr.Value)
					element.order = append(element.order, attr.Name.Local)
				}
			}
			switch {
			case root == nil:
				root = element
			case len(open) == 0:
				return nil, errors.New("there is an element after the root")
			default:
				parent := open[len(open)-1]
				parent.Children = append(parent.Children, element)
			}
			open = append(open, element)
		case xml.EndElement:
			closed := open[len(open)-1]
			switch {
			case len(closed.Children) > 0:
				closed.Value = ""
			case closed.Name != "text" || len(open) < 2 || open[len(open)-2].Name != "note":
				closed.Value = norm.NFC.String(closed.Value)
			}
			closed.close = [2]int{start, end}
			open = open[:len(open)-1]
		case xml.CharData:
			if len(open) > 0 {
				open[len(open)-1].Value += string(t)
			}
		}
	}
	if root == nil {
		return nil, io.ErrUnexpectedEOF
	}
	return root, nil
}
