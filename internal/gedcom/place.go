package gedcom

import (
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// places turns the place strings scattered through a GEDCOM into place entities.
//
// GEDCOM writes a place as one comma-separated line, most specific first, and repeats
// the whole line at every event that happened there. That is a hierarchy stated
// thousands of times over, and the model wants it stated once with `within:` linking the
// levels — so the work here is to recognise that two lines share a tail.
//
// The key is the tail and not the name: "Firenze, Toscana" and "Firenze, Sicilia" are
// two places that happen to be spelled the same, and collapsing them would invent a
// migration nobody made. Within one tail, a place is all its mentions say of it, and
// one place only where they agree: a mention that types a level otherwise, names it in
// another language, or puts the place at other coordinates or under another identifier
// at the same service is of a place of its own, and so is everything it puts within it.
type places struct {
	corpus *model.Corpus
	namer  *importing.Namer
	notes  *importing.Notes
	form   *Node // the header's FORM, for a place without its own

	// seen is the places of each name within each place, in the order they were made.
	seen  map[string][]model.Slug
	prose map[model.Slug][]string
}

func newPlaces(corpus *model.Corpus, namer *importing.Namer, notes *importing.Notes) *places {
	return &places{corpus: corpus, namer: namer, notes: notes, seen: map[string][]model.Slug{}, prose: map[model.Slug][]string{}}
}

// level is what a mention says of one level of a place.
type level struct {
	name  string
	typ   string
	other []model.PlaceName
}

// specific is what a mention says of the most specific place it names alone.
type specific struct {
	coordinates *model.Coordinates
	ids         model.IDs
	notes       []string
}

// place records the place a PLAC names and returns its slug, or "" where it names
// nothing. The line is divided at its commas, FORM gives each level its type where it
// has as many titles, and an empty level holds the FORM in line and is then dropped,
// being no place. LANG is the language of every level's name; TRAN, FONE and ROMN give
// each level another name, or the most specific place one as written where they are
// not divided as the place is. MAP, EXID and NOTE are the most specific place's.
func (c *converter) place(plac *Node) model.Slug {
	if plac == nil {
		return ""
	}
	written := strings.Split(plac.Value, ",")
	levels := make([]level, len(written))
	for i, name := range written {
		levels[i].name = collapse(name)
	}
	c.places.typed(plac, levels)
	lang := language(plac.First("LANG"))
	otherNames(plac, levels)
	levels = slices.DeleteFunc(levels, func(l level) bool { return l.name == "" })
	if len(levels) == 0 {
		return ""
	}

	own := &specific{coordinates: coordinates(plac.First("MAP")), ids: c.ids(plac), notes: c.noteTexts(plac)}
	if words := c.extension(plac, "_COORDINATES"); own.coordinates == nil && words != "" {
		own.coordinates = &model.Coordinates{Wording: words} // what origins could not write as a MAP
	}
	var slug model.Slug
	for i := len(levels) - 1; i >= 0; i-- {
		var mine *specific
		if i == 0 {
			mine = own
		}
		if slug = c.places.level(levels[i:], lang, slug, mine); slug == "" {
			return ""
		}
	}
	plac.Mark()
	return slug
}

// typed gives each level the type its title in the FORM names, the place's own or else
// the header's, where there are as many titles as levels. A FORM that does not line up
// says what no single level is, and is left unread.
func (p *places) typed(plac *Node, levels []level) {
	form := plac.First("FORM")
	if form == nil {
		form = p.form
	}
	titles := strings.Split(form.Text(), ",")
	if form == nil || len(titles) != len(levels) {
		return
	}
	form.Mark()
	for i, title := range titles {
		levels[i].typ = collapse(title)
	}
}

// otherNames reads a TRAN, the place written in another language, and 5.5.1's FONE and
// ROMN, written phonetically and in Latin letters, into the levels they name again. A
// level spelled the same way again says nothing more of it.
func otherNames(plac *Node, levels []level) {
	first := slices.IndexFunc(levels, func(l level) bool { return l.name != "" })
	for _, written := range plac.Children {
		var name model.PlaceName
		switch written.Tag {
		case "TRAN":
			name.Lang = language(written.First("LANG"))
		case "FONE", "ROMN":
			name.Type = collapse(written.Take("TYPE").Text())
			if name.Type == "" {
				name.Type = map[string]string{"FONE": "phonetic", "ROMN": "romanized"}[written.Tag]
			}
		default:
			continue
		}
		whole := collapse(written.Value)
		if whole == "" || first < 0 {
			continue
		}
		written.Mark()
		parts := strings.Split(written.Value, ",")
		if len(parts) != len(levels) {
			name.Text = whole
			levels[first].other = append(levels[first].other, name)
			continue
		}
		for i, part := range parts {
			if text := collapse(part); text != "" && levels[i].name != "" && text != levels[i].name {
				name.Text = text
				levels[i].other = append(levels[i].other, name)
			}
		}
	}
}

// level finds or makes the place levels[0] names within parent, the rest of levels
// being the names of what contains it. mine is what the mention says of it as the most
// specific place it names, or nil where it names it only as containing another.
func (p *places) level(levels []level, lang model.Language, parent model.Slug, mine *specific) model.Slug {
	this := levels[0]
	key := string(parent) + "\x00" + this.name
	for _, slug := range p.seen[key] {
		if place := p.corpus.Places[slug]; agrees(place, this, lang, mine) {
			p.merge(place, this, lang, mine)
			return slug
		}
	}

	// The containing levels are what tells two places of the same name apart, so they
	// are exactly the right discriminators to offer.
	var within []string
	for _, l := range levels[1:] {
		within = append(within, l.name)
	}
	written := strings.Join(append([]string{this.name}, within...), ", ")
	slug := p.namer.Assign(filedAs(this), within, written)
	if slug == "" {
		return ""
	}
	place := &model.Place{Slug: slug, Name: model.PlaceName{Text: this.name}}
	if parent != "" {
		place.Within = []model.PlaceRef{{Place: parent}}
	}
	p.merge(place, this, lang, mine)
	p.corpus.Places[slug] = place
	if len(p.seen[key]) > 0 {
		p.notes.Add("a mention of %s disagrees with another and is a place of its own, %s", written, slug)
	}
	p.seen[key] = append(p.seen[key], slug)
	return slug
}

// filedAs is the slug a place is filed under: its name's, or where that is in a script
// a slug cannot spell, the first of its other names that can be, and otherwise "place".
func filedAs(this level) model.Slug {
	if base, err := model.Slugify(this.name); err == nil {
		return base
	}
	for _, other := range this.other {
		if base, err := model.Slugify(other.Text); err == nil {
			return base
		}
	}
	return "place"
}

// agrees reports whether a mention says nothing of a place that the place says
// otherwise. Silence is no disagreement.
func agrees(place *model.Place, this level, lang model.Language, mine *specific) bool {
	if this.typ != "" && place.Type != "" && !model.SameWord(this.typ, place.Type) {
		return false
	}
	if lang != "" && place.Name.Lang != "" && lang != place.Name.Lang {
		return false
	}
	if mine == nil {
		return true
	}
	if mine.coordinates != nil && place.Coordinates != nil && *mine.coordinates != *place.Coordinates {
		return false
	}
	for service, ids := range mine.ids {
		if held, ok := place.IDs[service]; ok && held[0].ID != ids[0].ID {
			return false
		}
	}
	return true
}

// merge adds to a place what a mention that agrees with it says.
func (p *places) merge(place *model.Place, this level, lang model.Language, mine *specific) {
	if place.Type == "" {
		place.Type = this.typ
	}
	if place.Name.Lang == "" {
		place.Name.Lang = lang
	}
	for _, name := range this.other {
		if !slices.ContainsFunc(place.AlsoKnownAs, func(known model.PlaceName) bool { return reflect.DeepEqual(known, name) }) {
			place.AlsoKnownAs = append(place.AlsoKnownAs, name)
		}
	}
	if mine == nil {
		return
	}
	if place.Coordinates == nil {
		place.Coordinates = mine.coordinates
	}
	for _, service := range sortedKeys(mine.ids) {
		place.IDs, _ = identify(place.IDs, service, mine.ids[service][0].ID)
	}
	for _, note := range mine.notes {
		if !slices.Contains(p.prose[place.Slug], note) {
			p.prose[place.Slug] = append(p.prose[place.Slug], note)
		}
	}
	place.Body = strings.Join(p.prose[place.Slug], "\n\n")
}

func sortedKeys(ids model.IDs) []string {
	keys := make([]string, 0, len(ids))
	for key := range ids {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// headerForm is the FORM the header gives every place without its own, which is read
// where a place uses it.
func headerForm(records []*Node) *Node {
	for _, record := range records {
		if record.Tag == "HEAD" {
			if plac := record.First("PLAC"); plac != nil {
				return plac.Mark().First("FORM")
			}
		}
	}
	return nil
}

// components splits a PLAC value into levels, most specific first, without its empty
// ones.
func components(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// coordinates reads a MAP subtree. GEDCOM writes the hemisphere as a letter in front of
// the number, which is the only thing standing between it and a float; where that is not
// what LATI and LONG say, what they say is kept as the coordinates' words.
func coordinates(node *Node) *model.Coordinates {
	if node == nil {
		return nil
	}
	lati, long := node.First("LATI").Text(), node.First("LONG").Text()
	latitude, north := degrees(lati, 'N', 'S')
	longitude, east := degrees(long, 'E', 'W')
	var at *model.Coordinates
	switch words := collapse(lati + " " + long); {
	case north && east:
		at = &model.Coordinates{Latitude: latitude, Longitude: longitude}
	case words != "":
		at = &model.Coordinates{Wording: words}
	default:
		return nil
	}
	node.Mark().Take("LATI")
	node.Take("LONG")
	return at
}

func degrees(text string, positive, negative byte) (float64, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, false
	}
	sign := 1.0
	switch text[0] {
	case positive, positive | 0x20:
		text = text[1:]
	case negative, negative | 0x20:
		text, sign = text[1:], -1
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil {
		return 0, false
	}
	return sign * value, true
}
