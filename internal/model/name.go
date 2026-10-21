package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"
)

// PersonName is a name somebody was known by: the words a record wrote, the parts
// GEDCOM and Gramps divide one into, or both.
//
// Text is the name as a record wrote it — the clerk's "Gio. Rossi" — a spelling to match
// on that nobody has to divide. Beside parts it is the record's own words where the
// parts do not spell them, and nothing keeps the two in step, since the record does not
// change when somebody reads it better. There is no display string beside the parts to
// fall out of step with them: Display is a way of putting them together.
type PersonName struct {
	Text string

	// Private leaves the name out of a hidden view. Only another name a person was
	// known by takes it: the one they are filed under is hidden with the person, and
	// one an event gives with the event.
	Private bool

	Prefix         string
	Given          string
	Call           string // the given name the person went by, usually one of Given
	Nickname       string
	FamilyNickname string
	SurnamePrefix  string // de, van der: apart, so that the surname files on its own
	Surname        Surnames
	Suffix         string
	Group          string // the surname the name is filed among, where its own would file it apart

	Lang Language

	// Type is what kind of name it is — birth, married, aka, religious — an open list
	// as event types are. From and Until are when it was borne, which is what makes a
	// married name a married name and not a spelling.
	Type string

	// Display and Sort are patterns of the name's parts, by which this one name is
	// written out and sorted in place of the usual order.
	Display string
	Sort    string

	From    Date
	Until   Date
	Note    string
	Sources []CitationRef
}

// Surname is one of a name's surnames, with what is said of it where it is one of
// several: its own prefix, the word joining it to the next, where it came from, and
// whether it is the one the name is filed under.
type Surname struct {
	Name      string
	Prefix    string
	Connector string
	Origin    string
	Primary   bool
}

// Qualified reports whether anything is said of the surname beyond the name.
func (s Surname) Qualified() bool {
	return s.Prefix != "" || s.Connector != "" || s.Origin != "" || s.Primary
}

// Surnames is a name's surnames in their order: one, written as a string, or several.
// It is one value, so that a name with one surname reads and writes as a word.
type Surnames []Surname

// SurnameOf is one surname written as a word, or none where the word is empty.
func SurnameOf(name string) Surnames {
	if name == "" {
		return nil
	}
	return Surnames{{Name: name}}
}

// String is the surnames each with its own prefix, joined by their connectors.
func (s Surnames) String() string { return PersonName{Surname: s}.join(s) }

// Listed reports whether the surnames can only be written as a list.
func (s Surnames) Listed() bool { return len(s) > 1 || len(s) == 1 && s[0].Qualified() }

// Qualified reports whether the name says anything beyond the words of a record, which
// a name written as a bare string cannot.
func (n PersonName) Qualified() bool {
	return n.Private || n.HasParts() || n.Lang != "" || n.Type != "" || n.Display != "" || n.Sort != "" ||
		!n.From.IsZero() || !n.Until.IsZero() || n.Note != "" || len(n.Sources) > 0
}

// HasParts reports whether any part of the name is written, as against only its words
// or what qualifies it.
func (n PersonName) HasParts() bool {
	return n.Prefix != "" || n.Given != "" || n.Call != "" || n.Nickname != "" || n.FamilyNickname != "" ||
		n.SurnamePrefix != "" || len(n.Surname) > 0 || n.Suffix != "" || n.Group != ""
}

// IsZero reports whether the name says nothing at all.
func (n PersonName) IsZero() bool { return n.Text == "" && !n.Qualified() }

// PrimarySurname is the surname the name is filed under: the one marked primary, else
// the first.
func (n PersonName) PrimarySurname() Surname {
	for _, s := range n.Surname {
		if s.Primary {
			return s
		}
	}
	if len(n.Surname) == 0 {
		return Surname{}
	}
	return n.Surname[0]
}

// String is the name as a line of output shows it: by its display pattern where it has
// one, and otherwise the parts in order, the nickname in quotation marks and the call
// name standing for the given names where only it is known. A name of words alone is
// its words.
func (n PersonName) String() string {
	if n.Display != "" {
		if written := n.Render(n.Display); written != "" {
			return written
		}
	}
	return n.plain()
}

func (n PersonName) plain() string {
	if !n.HasParts() {
		return n.Text
	}
	given := n.Given
	if given == "" {
		given = n.Call
	}
	nickname := ""
	if n.Nickname != "" {
		nickname = "“" + n.Nickname + "”"
	}
	return joinWords(n.Prefix, given, nickname, n.surnames(), n.Suffix)
}

// Slug is what a person of this name is filed under: the primary surname without its
// prefix, so that de Rossi files among the Rossis, then the given names. A name of
// words alone gives its slug from the whole of them, nobody having said which part is
// the surname.
func (n PersonName) Slug() (Slug, error) {
	words := joinWords(n.PrimarySurname().Name, n.Given)
	if words == "" {
		words = n.plain()
	}
	return Slugify(words)
}

// Render writes the name out by a pattern of its parts, each named in braces:
// {surname}, {given} for a name written surname first. Beside the parts a pattern can
// name what is made from them — {initials}, {primary-surname}, {patronymic},
// {other-surnames} and {common} — and a code in capitals writes its part in capitals.
// A part the name does not have is left out with what separates it from the part
// before, and anything in braces that names no part is written as it stands.
func (n PersonName) Render(pattern string) string {
	var (
		out     strings.Builder
		pending string // what was written since the last part, waiting on the next
		named   bool   // whether a part has been named
		written bool   // whether a part has been written
		last    = true // whether the last part named was written
	)
	for pattern != "" {
		open := strings.IndexByte(pattern, '{')
		end := strings.IndexByte(pattern[max(open, 0):], '}')
		if open < 0 || end < 0 {
			pending += pattern
			break
		}
		end += open
		value, known := n.code(pattern[open+1 : end])
		if !known {
			pending += pattern[:end+1]
			pattern = pattern[end+1:]
			continue
		}
		pending += pattern[:open]
		pattern = pattern[end+1:]
		first := !named
		named, last = true, value != ""
		if !last {
			pending = ""
			continue
		}
		if written || first {
			out.WriteString(pending)
		}
		out.WriteString(value)
		pending, written = "", true
	}
	if last && (written || !named) {
		out.WriteString(pending)
	}
	return strings.TrimSpace(out.String())
}

// code is what one code of a pattern writes, and whether it names anything.
func (n PersonName) code(code string) (string, bool) {
	lower := strings.ToLower(code)
	if code != lower && code != strings.ToUpper(code) {
		return "", false
	}
	var value string
	switch lower {
	case "prefix":
		value = n.Prefix
	case "given":
		value = n.Given
	case "call":
		value = n.Call
	case "nickname":
		value = n.Nickname
	case "family-nickname":
		value = n.FamilyNickname
	case "surname-prefix":
		prefixes := []string{n.SurnamePrefix}
		for _, s := range n.Surname {
			prefixes = append(prefixes, s.Prefix)
		}
		value = joinWords(prefixes...)
	case "surname":
		value = n.surnames()
	case "suffix":
		value = n.Suffix
	case "group":
		value = n.Group
	case "initials":
		var initials strings.Builder
		for _, word := range strings.Fields(n.Given) {
			first, _ := utf8.DecodeRuneInString(word)
			initials.WriteRune(first)
			initials.WriteByte('.')
		}
		value = initials.String()
	case "primary-surname":
		value = n.withPrefix(n.PrimarySurname())
	case "patronymic":
		for _, s := range n.Surname {
			if SameWord(s.Origin, "patronymic") || SameWord(s.Origin, "matronymic") {
				value = n.withPrefix(s)
				break
			}
		}
	case "other-surnames":
		primary := n.PrimarySurname()
		var others Surnames
		for _, s := range n.Surname {
			if s != primary {
				others = append(others, s)
			}
		}
		value = n.join(others)
	case "common":
		value = n.Nickname
		if value == "" {
			value = n.Call
		}
		if value == "" {
			value, _, _ = strings.Cut(strings.TrimSpace(n.Given), " ")
		}
	default:
		return "", false
	}
	if code != lower {
		value = strings.ToUpper(value)
	}
	return value, true
}

// surnames writes every surname with its prefix, joined by their connectors.
func (n PersonName) surnames() string { return n.join(n.Surname) }

func (n PersonName) join(list Surnames) string {
	var out strings.Builder
	for i, s := range list {
		if i > 0 {
			switch connector := list[i-1].Connector; connector {
			case "":
				out.WriteByte(' ')
			case "-":
				out.WriteString(connector)
			default:
				out.WriteString(" " + connector + " ")
			}
		}
		out.WriteString(n.withPrefix(s))
	}
	return out.String()
}

// withPrefix is one surname with its prefix, a surname written as a string taking the
// name's surname-prefix as its own.
func (n PersonName) withPrefix(s Surname) string {
	prefix := s.Prefix
	if prefix == "" && !n.Surname.Listed() {
		prefix = n.SurnamePrefix
	}
	return joinWords(prefix, s.Name)
}

// Names hands every name of a person an entity holds to a function that may change it,
// with the field path it is at: a person's own and the others they were known by, a
// partner's and a child's a family gives, and the name of somebody an event names.
func Names(entity any, each func(path string, name *PersonName)) {
	events := func(list []Event) {
		for i := range list {
			for j := range list[i].With {
				each(fmt.Sprintf("events[%d].with[%d].name", i, j), &list[i].With[j].Name)
			}
		}
	}
	switch v := entity.(type) {
	case *Person:
		each("name", &v.Name)
		for i := range v.AlsoKnownAs {
			each(fmt.Sprintf("also-known-as[%d]", i), &v.AlsoKnownAs[i])
		}
		events(v.Events)
	case *Family:
		for i := range v.Partners {
			each(fmt.Sprintf("partners[%d].name", i), &v.Partners[i].Name)
		}
		for i := range v.Children {
			each(fmt.Sprintf("children[%d].name", i), &v.Children[i].Name)
		}
		events(v.Events)
	case *Note:
		events(v.Events)
	case *Source:
		for _, anchor := range slices.Sorted(maps.Keys(v.Citations)) {
			// held in the map by value, and who it names in a list the copy shares
			with := v.Citations[anchor].With
			for j := range with {
				each(fmt.Sprintf("citations.%s.with[%d].name", anchor, j), &with[j].Name)
			}
		}
	}
}

// Clone copies the name so that a change to its surnames or sources is not seen in the
// original.
func (n PersonName) Clone() PersonName {
	n.Surname = slices.Clone(n.Surname)
	n.Sources = slices.Clone(n.Sources)
	return n
}

func joinWords(words ...string) string {
	return strings.Join(strings.Fields(strings.Join(words, " ")), " ")
}
