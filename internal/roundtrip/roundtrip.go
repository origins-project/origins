// Package roundtrip holds a corpus read back from an export against the corpus
// exported, record by record, for the tests of every export to say the same of it: that
// it differs in nothing the export's report counts as left out.
package roundtrip

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// A difference is where a record read back from an export says something other than
// the record written, as a path of the file's own field names with the kind in front.
type Difference struct {
	Record      model.Slug
	Path        []string
	Want, Found any
}

func (d Difference) String() string {
	return fmt.Sprintf("%s %s: wrote %v, read %v", d.Record, strings.Join(d.Path, "."), brief(d.Want), brief(d.Found))
}

func brief(v any) string {
	if v == nil {
		return "nothing"
	}
	written, _ := json.Marshal(v)
	if len(written) > 160 {
		return string(written[:160]) + "…"
	}
	return string(written)
}

// Compare holds a corpus read back against the one written, record by record,
// matched by what each says rather than by its slug, and gives every difference, every
// record of either with no match in the other being one. The slugs of what was read are
// rewritten to the slugs written before anything is compared, wherever they stand.
func Compare(written, read *model.Corpus) ([]Difference, error) {
	slugs := matching(written, read)
	var found []Difference
	matched := map[model.Slug]bool{}
	for _, kind := range model.Kinds {
		if kind == model.KindHypothesis {
			continue
		}
		for _, slug := range read.SlugsOf(kind) {
			as, ok := slugs[slug]
			if !ok {
				found = append(found, Difference{Path: []string{kind.Dir()}, Found: string(slug)})
				continue
			}
			matched[as] = true
			want, err := store.Fields(written.Entity(as))
			if err != nil {
				return nil, err
			}
			got, err := store.Fields(read.Entity(slug))
			if err != nil {
				return nil, err
			}
			got = renamed(got, slugs)
			if kind == model.KindFamily {
				reordered(want)
				reordered(got)
			}
			for _, d := range diff([]string{kind.Dir()}, want, got) {
				d.Record = as
				found = append(found, d)
			}
		}
		for _, slug := range written.SlugsOf(kind) {
			if !matched[slug] {
				found = append(found, Difference{Path: []string{kind.Dir()}, Want: string(slug)})
			}
		}
	}
	project, kept := written.Project, read.Project
	want, err := store.Fields(&project)
	if err != nil {
		return nil, err
	}
	got, err := store.Fields(&kept)
	if err != nil {
		return nil, err
	}
	return append(found, diff([]string{"project"}, want, renamed(got, slugs))...), nil
}

// Matching pairs each record read with the record written that says the same of
// itself: a person by name, sex and the date of the first event, a family by the people
// its partners and its children were paired with and its first event, a place by its name and its type, and
// a source, a repository and a note by what each is called. Records that say the same
// of themselves, as everybody a hidden view puts a placeholder for, are paired in the
// order they were written and read in: the order of the slugs written, and of the slugs
// read as an import numbers them, private, private-2 and on to private-10.
func matching(written, read *model.Corpus) map[model.Slug]model.Slug {
	slugs := map[model.Slug]model.Slug{}
	for _, kind := range model.Kinds {
		if kind == model.KindHypothesis {
			continue
		}
		byKey := map[string][]model.Slug{}
		for _, slug := range written.SlugsOf(kind) {
			key := identity(written, slug, func(s model.Slug) model.Slug { return s })
			byKey[key] = append(byKey[key], slug)
		}
		for _, slug := range slices.SortedStableFunc(slices.Values(read.SlugsOf(kind)), numbered) {
			key := identity(read, slug, func(s model.Slug) model.Slug { return slugs[s] })
			if as := byKey[key]; len(as) > 0 {
				slugs[slug], byKey[key] = as[0], as[1:]
			}
		}
	}
	return slugs
}

// numbered orders slugs by what is before a number they end in and then by the number,
// a slug ending in none coming first.
func numbered(a, b model.Slug) int {
	split := func(s model.Slug) (string, int) {
		if i := strings.LastIndex(string(s), "-"); i >= 0 {
			if number, err := strconv.Atoi(string(s)[i+1:]); err == nil {
				return string(s)[:i], number
			}
		}
		return string(s), 0
	}
	baseA, numberA := split(a)
	baseB, numberB := split(b)
	if c := strings.Compare(baseA, baseB); c != 0 {
		return c
	}
	return numberA - numberB
}

// identity is what a record says of itself, a family naming its partners and children by
// as, the slugs of the people already matched.
func identity(c *model.Corpus, slug model.Slug, as func(model.Slug) model.Slug) string {
	firstDate := func(events []model.Event) string {
		for _, e := range events {
			if !e.Date.IsZero() {
				return e.Date.String()
			}
		}
		return ""
	}
	switch e := c.Entity(slug).(type) {
	case *model.Person:
		var surnames []string
		for _, surname := range e.Name.Surname {
			surnames = append(surnames, surname.Name)
		}
		return fmt.Sprintf("%s|%s|%s|%s|%s", e.Name.Text, e.Name.Given, surnames, e.Sex.Value, firstDate(e.Events))
	case *model.Family:
		var partners []string
		for _, p := range e.Partners {
			if p.Person == model.None || p.Person == model.Unknown {
				continue
			}
			if _, ok := c.People[p.Person]; ok {
				partners = append(partners, string(as(p.Person)))
			} else {
				partners = append(partners, string(p.Person)+p.Name.Text)
			}
		}
		sort.Strings(partners)
		var children []string
		for _, child := range e.Children {
			if _, ok := c.People[child.ID]; ok {
				children = append(children, string(as(child.ID)))
			}
		}
		return strings.Join(partners, "+") + "|" + firstDate(e.Events) + "|" + strings.Join(children, "+")
	case *model.Place:
		return e.Name.Text + "|" + e.Type
	case *model.Source:
		return e.Title
	case *model.Repository:
		return e.Name
	case *model.Note:
		return e.Title
	}
	return ""
}

// reordered puts a family's partners who have records in the order of their slugs, as an
// import puts them, so that the order a format does not keep is not held against it.
func reordered(family map[string]any) {
	partners, _ := family["partners"].([]any)
	who := func(partner any) string {
		if fields, ok := partner.(map[string]any); ok {
			partner = fields["person"]
		}
		slug, _ := partner.(string)
		if model.Slug(slug).Reserved() {
			return ""
		}
		return slug
	}
	people := 0
	for people < len(partners) && who(partners[people]) != "" {
		people++
	}
	slices.SortStableFunc(partners[:people], func(a, b any) int { return strings.Compare(who(a), who(b)) })
}

// dated are the fields holding a date.
var dated = map[string]bool{"date": true, "sort-date": true, "from": true, "until": true}

// Renamed rewrites every slug of the corpus read as the slug written, wherever a value
// holds one: a reference, and a path into a record's directory, in a field or in a link
// in prose. A slug merely mentioned in prose is words, and stays as written, and a date
// is never a slug, though a year may be spelled as one.
func renamed(fields map[string]any, slugs map[model.Slug]model.Slug) map[string]any {
	token := regexp.MustCompile(`(^|/)[a-z0-9-]+`)
	rename := func(m string) string {
		if to, ok := slugs[model.Slug(m)]; ok {
			return string(to)
		}
		return m
	}
	var walk func(any) any
	walk = func(v any) any {
		switch v := v.(type) {
		case string:
			return token.ReplaceAllStringFunc(v, func(m string) string {
				if slash, ok := strings.CutPrefix(m, "/"); ok {
					return "/" + rename(slash)
				}
				return rename(m)
			})
		case []any:
			out := make([]any, len(v))
			for i, item := range v {
				out[i] = walk(item)
			}
			return out
		case map[string]any:
			out := map[string]any{}
			for k, item := range v {
				if dated[k] {
					out[k] = item
				} else {
					out[k] = walk(item)
				}
			}
			return out
		}
		return v
	}
	return walk(fields).(map[string]any)
}

func diff(path []string, want, found any) []Difference {
	switch w := want.(type) {
	case map[string]any:
		f, ok := found.(map[string]any)
		if !ok {
			break
		}
		var out []Difference
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range f {
			keys[k] = true
		}
		for _, k := range slices.Sorted(func(yield func(string) bool) {
			for k := range keys {
				if !yield(k) {
					return
				}
			}
		}) {
			out = append(out, diff(append(slices.Clone(path), k), w[k], f[k])...)
		}
		return out
	case []any:
		f, ok := found.([]any)
		if !ok {
			break
		}
		var out []Difference
		for i := range max(len(w), len(f)) {
			var a, b any
			if i < len(w) {
				a = w[i]
			}
			if i < len(f) {
				b = f[i]
			}
			out = append(out, diff(append(slices.Clone(path), strconv.Itoa(i)), a, b)...)
		}
		return out
	}
	// a field written as one value where nothing more is said of it, as a sex or a
	// surname, is held against the same field written whole
	if w, ok := want.(map[string]any); ok && found != nil {
		if _, isMap := found.(map[string]any); !isMap {
			for _, key := range []string{"value", "name", "file"} {
				if _, has := w[key]; has {
					return diff(path, w, map[string]any{key: found})
				}
			}
		}
	}
	// and a list of one, as a surname is, against the one value it holds
	if w, ok := want.([]any); ok && len(w) == 1 && found != nil {
		if _, isList := found.([]any); !isList {
			return diff(path, w, []any{found})
		}
	}
	if w, ok := want.(string); ok {
		if f, ok := found.(string); ok {
			// the newline a block of YAML ends in is no part of what it says
			if strings.TrimRight(w, "\n") == strings.TrimRight(f, "\n") {
				return nil
			}
			// an age that is a bare number is years, which a format may have to say
			if at := slices.Index(path, "age"); at >= 0 && at >= len(path)-2 && f == w+"y" {
				if _, err := strconv.ParseUint(w, 10, 32); err == nil {
					return nil
				}
			}
		}
	}
	if reflect.DeepEqual(want, found) {
		return nil
	}
	return []Difference{{Path: path, Want: want, Found: found}}
}

// Counted says whether the report counts a difference as something the export left
// out: some path it names runs, as a sequence of field names, through the difference's
// path with the positions in lists set aside.
func Counted(d Difference, omitted map[string]int) bool {
	var names []string
	for i, p := range d.Path {
		if _, err := strconv.Atoi(p); err == nil || i > 0 && d.Path[i-1] == "citations" {
			continue // a position in a list, or the anchor a citation is kept under
		}
		names = append(names, p)
	}
	if value, ok := d.Want.(string); ok {
		names = append(names, value) // a value stated in words of its own, as partners.none
	}
	for path := range omitted {
		want := strings.Split(path, ".")
		for i := 0; i+len(want) <= len(names); i++ {
			if slices.Equal(names[i:i+len(want)], want) {
				return true
			}
		}
	}
	return false
}
