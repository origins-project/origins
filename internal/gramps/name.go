package gramps

import (
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// nameTypes are Gramps' own types of name, in the words the corpus writes.
var nameTypes = vocabulary(map[string]string{
	"Birth Name":    "birth",
	"Married Name":  "married",
	"Also Known As": "aka",
	"Unknown":       "",
})

// origins are Gramps' own origins of a surname, as slugs. Gramps writes the origin it
// calls unknown with a space after it, and one nobody gave not at all.
var origins = vocabulary(map[string]string{"Unknown": ""},
	"Inherited", "Given", "Taken", "Patronymic", "Matronymic", "Feudal", "Pseudonym",
	"Patrilineal", "Matrilineal", "Occupation", "Location",
)

// formats are the patterns Gramps' own name formats stand for, by the number a name
// gives as its display or sort. Nought is whatever the user's preferences say, which is
// no pattern of the name's own.
var formats = map[string]string{
	"1": "%l, %f %s",
	"2": "%f %l %s",
	"3": "%y, %s %f",
	"4": "%f",
	"5": "%1m %2m %o, %f %1y %s %0m",
}

// codes are what each of Gramps' codes in a name format names, in the corpus's words.
var codes = map[byte]string{
	't': "prefix", 'f': "given", 'l': "surname", 'c': "call", 'x': "common",
	'i': "initials", 'm': "primary-surname", 'y': "patronymic", 'r': "other-surnames",
	'p': "surname-prefix", 's': "suffix", 'n': "nickname", 'g': "family-nickname",
}

// name reads a name with its dates, its notes and its citations, the notes going to
// path on record.
func (c *converter) name(e *Element, record model.Slug, path string) model.PersonName {
	c.home(e, record, path)
	name := c.readName(e)
	if !name.HasParts() {
		return name
	}
	name.From, name.Until = bounds(c.date(e))
	name.Sources = c.citations(e)
	name.Note = strings.Join(c.prose(e), "\n\n")
	return name
}

// readName reads the parts of a Gramps name, its type, its privacy, the group it files
// among and the patterns it is shown and sorted by.
func (c *converter) readName(e *Element) model.PersonName {
	if e == nil {
		return model.PersonName{}
	}
	e.Use("alt")
	name := model.PersonName{
		Private:        private(e),
		Prefix:         collapse(e.Take("title").Text()),
		Given:          collapse(e.Take("first").Text()),
		Call:           collapse(e.Take("call").Text()),
		Nickname:       collapse(e.Take("nick").Text()),
		FamilyNickname: collapse(e.Take("familynick").Text()),
		Suffix:         collapse(e.Take("suffix").Text()),
		Group:          collapse(e.Take("group").Text()),
	}
	name.SurnamePrefix, name.Surname = surnames(e.TakeAll("surname"))
	if name.Group == "" {
		name.Group = c.groupOf(name.PrimarySurname().Name)
	}
	kind := e.Use("type")
	if kind == "" {
		kind = "Birth Name"
	}
	name.Type = typeWord(nameTypes, kind)
	if !name.HasParts() {
		return model.PersonName{}
	}
	name.Display = c.pattern(e, "display")
	name.Sort = c.pattern(e, "sort")
	return name
}

// surnames reads Gramps' list of surnames. One with nothing said of it but a prefix is
// a surname and the name's surname prefix, as a name with one surname is written;
// otherwise each keeps its own prefix, its connector and its origin, and the primary is
// marked where it is not the first. A surname with nothing in it at all, which Gramps
// writes for a name without one, is none.
func surnames(elements []*Element) (string, model.Surnames) {
	var list model.Surnames
	primary := -1
	for _, e := range elements {
		s := model.Surname{
			Name:      collapse(e.Text()),
			Prefix:    collapse(e.Use("prefix")),
			Connector: collapse(e.Use("connector")),
		}
		if origin, ok := origins[collapse(e.Attr("derivation"))]; ok {
			s.Origin = origin
		} else {
			s.Origin = collapse(e.Attr("derivation"))
		}
		e.Use("derivation")
		named := e.Use("prim") != "0"
		if s == (model.Surname{}) {
			continue
		}
		if named && primary < 0 {
			primary = len(list)
		}
		list = append(list, s)
	}
	if primary > 0 {
		list[primary].Primary = true
	}
	if len(list) == 1 && list[0].Connector == "" && list[0].Origin == "" {
		return list[0].Prefix, model.SurnameOf(list[0].Name)
	}
	return "", list
}

// groupOf is the group the file's name maps give a surname, marking the map read where
// there is one.
func (c *converter) groupOf(surname string) string {
	if surname == "" {
		return ""
	}
	maps := c.root.First("namemaps")
	for _, m := range maps.All("map") {
		if m.Attr("type") == "group_as" && m.Attr("key") == surname {
			maps.Mark()
			m.Mark()
			m.Use("type")
			m.Use("key")
			return collapse(m.Use("value"))
		}
	}
	return ""
}

// pattern is the name format a name's attribute names, as a pattern of the model's:
// one of Gramps' own, or one of the file's name formats, marked read where it is used.
// The default format is no pattern, and a number naming no format is left unread.
func (c *converter) pattern(e *Element, attribute string) string {
	number := e.Attr(attribute)
	if number == "" || number == "0" {
		e.Use(attribute)
		return ""
	}
	format, ok := formats[number]
	if !ok {
		defined := c.root.First("name-formats")
		for _, f := range defined.All("format") {
			if f.Attr("number") == number {
				defined.Mark()
				f.markAll()
				format, ok = f.Attr("fmt_str"), true
				break
			}
		}
	}
	if !ok {
		return ""
	}
	e.Use(attribute)
	return convertFormat(format)
}

// convertFormat writes a Gramps name format as a pattern: each code as the part it
// names, in capitals where the code is, and a code naming a part of a surname, or
// anything the model has no part for, as written.
func convertFormat(format string) string {
	var out strings.Builder
	for i := 0; i < len(format); i++ {
		if format[i] != '%' || i+1 == len(format) {
			out.WriteByte(format[i])
			continue
		}
		letter := format[i+1]
		lower := letter | 0x20
		part, ok := codes[lower]
		if !ok || letter < 'A' || lower > 'z' {
			out.WriteByte('%')
			continue
		}
		if letter != lower {
			part = strings.ToUpper(part)
		}
		out.WriteString("{" + part + "}")
		i++
	}
	return out.String()
}
