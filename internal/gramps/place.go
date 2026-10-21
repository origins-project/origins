package gramps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// placeTypes are Gramps' own types of place. Unknown says nothing.
var placeTypes = vocabulary(map[string]string{"Unknown": ""},
	"Country", "State", "County", "City", "Parish", "Locality", "Street", "Province",
	"Region", "Department", "Neighborhood", "District", "Borough", "Municipality", "Town",
	"Village", "Hamlet", "Farm", "Building", "Number",
)

// placeName is what a place is called: its first name, or where it has none the title
// Gramps falls back on.
func placeName(place *Element) string {
	if name := collapse(place.First("pname").Attr("value")); name != "" {
		return name
	}
	return collapse(place.First("ptitle").Text())
}

// filedAs is the slug a place is filed under: its name's, or where that is in a script
// a slug cannot spell, the first of its other names that can be, as the GEDCOM import
// files one, and then its title.
func filedAs(place *Element, name string) (model.Slug, error) {
	base, err := model.Slugify(name)
	if err == nil {
		return base, nil
	}
	others := place.All("pname")
	if len(others) > 0 {
		others = others[1:]
	}
	for _, other := range others {
		if base, err := model.Slugify(other.Attr("value")); err == nil {
			return base, nil
		}
	}
	if base, err := model.Slugify(place.First("ptitle").Text()); err == nil {
		return base, nil
	}
	return "", err
}

// namePlaces gives every place its slug before any is built, since a place names the
// places it lies within by slug and may come before them in the file.
func (c *converter) namePlaces() {
	for _, place := range c.all("places", "placeobj") {
		name := placeName(place)
		containers := c.containers(place)
		base, err := filedAs(place, name)
		if err != nil {
			if base, err = model.Slugify(place.Attr("id")); err != nil {
				c.notes.Add("a place had no name and no usable identifier, and was not imported")
				continue
			}
			if name == "" {
				c.notes.Add("a place had no name and is filed under its identifier")
			} else {
				c.notes.Add("a place had no name a slug can spell and is filed under its identifier")
			}
		}
		// The places it lies within are what tells two places of the same name apart,
		// so they are the discriminators to offer.
		what := strings.Join(append([]string{name}, containers...), ", ")
		slug := c.namer.Assign(base, containers, what)
		if slug == "" {
			c.notes.Add("no free slug could be found for the place %q", what)
			continue
		}
		c.place[place.Attr("handle")] = slug
	}
}

// containers are the names of the places a place lies within, nearest first, following
// the first of its containers at each step.
func (c *converter) containers(place *Element) []string {
	var names []string
	seen := map[*Element]bool{place: true}
	for {
		within := c.object(place.First("placeref"), "placeobj")
		if within == nil || seen[within] {
			return names
		}
		seen[within] = true
		if name := placeName(within); name != "" {
			names = append(names, name)
		}
		place = within
	}
}

func (c *converter) readPlaces() {
	for _, e := range c.all("places", "placeobj") {
		slug, ok := c.place[e.Attr("handle")]
		if !ok {
			continue
		}
		e.Mark()
		c.home(e, slug, "body")
		place := &model.Place{
			Slug:    slug,
			Private: private(e),
			Name:    model.PlaceName{Text: placeName(e)},
			IDs:     c.grampsID(e),
			Sources: c.citations(e),
		}
		names := e.All("pname")
		if len(names) > 0 && collapse(names[0].Attr("value")) != "" {
			c.home(names[0], slug, "name.note")
			place.Name = c.otherName(names[0])
			c.title(e, place)
		} else {
			e.Take("ptitle")
		}
		for i, name := range names {
			if i == 0 {
				continue
			}
			at := fmt.Sprintf("also-known-as[%d].note", len(place.AlsoKnownAs))
			c.home(name, slug, at)
			place.AlsoKnownAs = append(place.AlsoKnownAs, c.otherName(name))
		}
		if ids, ok := identify(place.IDs, "code", collapse(e.Take("code").Text())); ok {
			place.IDs = ids
		}
		place.Type = typeWord(placeTypes, e.Use("type"))
		for _, ref := range e.All("placeref") {
			within, ok := c.place[ref.Attr("hlink")]
			if !ok {
				c.notes.Add("a place was said to lie within a place that is not in the file")
				continue
			}
			ref.Mark()
			c.home(ref, slug, fmt.Sprintf("within[%d].note", len(place.Within)))
			from, until := bounds(c.date(ref))
			place.Within = append(place.Within, model.PlaceRef{Place: within, From: from, Until: until})
		}
		place.Coordinates = c.coordinates(e.First("coord"))
		for _, location := range e.TakeAll("location") {
			place.Addresses = append(place.Addresses, locationOf(location))
		}
		place.Links = c.links(e, &place.Private)
		place.Tags = c.tags(e)
		place.Body = c.body(e)
		c.corpus.Places[slug] = place
		c.pictures(e, place, &place.Media, -1)
	}
}

// title reads the title Gramps keeps for a place where one was typed or kept from an
// older database, which is another name of the place unless it is only the place's name,
// or that and the names of the places it lies within, which the hierarchy says already.
func (c *converter) title(e *Element, place *model.Place) {
	title := collapse(e.First("ptitle").Text())
	if title == "" {
		return
	}
	e.Take("ptitle")
	if title == place.Name.Text || title == strings.Join(append([]string{place.Name.Text}, c.containers(e)...), ", ") {
		return
	}
	place.AlsoKnownAs = append(place.AlsoKnownAs, model.PlaceName{Text: title})
}

// locationOf reads an alternate location of a place, which Gramps writes as the
// attributes of one element, as an address, its telephone a link.
func locationOf(e *Element) model.Address {
	part := func(name string) string { return collapse(e.Use(name)) }
	address := model.Address{
		Street: part("street"), Locality: part("locality"), City: part("city"),
		Parish: part("parish"), County: part("county"), State: part("state"),
		Postal: part("postal"), Country: part("country"),
	}
	if phone := part("phone"); phone != "" {
		address.Links = []model.Link{{URL: "tel:" + phone}}
	}
	return address
}

// otherName reads another name a place had, with its language and when it bore it. A
// language that is not a language tag is left unread.
func (c *converter) otherName(name *Element) model.PlaceName {
	name.Mark()
	other := model.PlaceName{Text: collapse(name.Use("value"))}
	if lang, err := model.ParseLanguage(name.Attr("lang")); err == nil {
		other.Lang = lang
		name.Use("lang")
	}
	other.From, other.Until = bounds(c.date(name))
	return other
}

// bounds reads the date Gramps puts on something that held for a while — a name, a
// place lying within another — as when it began and when it ended. A period is its
// two ends, after a date is from it and before one until it, and a date at one time is
// both, the one time it is known to have held.
func bounds(date model.Date) (from, until model.Date) {
	switch {
	case date.Period():
		if date.From != nil {
			from = *date.From
		}
		if date.To != nil {
			until = *date.To
		}
	case date.Qualifier == model.After:
		date.Qualifier = model.Exact
		from = date
	case date.Qualifier == model.Before:
		date.Qualifier = model.Exact
		until = date
	default:
		from, until = date, date
	}
	return from, until
}

// coordinates reads a place's latitude and longitude in any of the notations Gramps
// accepts them in: decimal degrees signed or with a hemisphere's letter, and degrees,
// minutes and seconds marked with symbols or colons. Coordinates that are not two
// angles are kept as their words, the latitude's and then the longitude's.
func (c *converter) coordinates(coord *Element) *model.Coordinates {
	latText, longText := coord.Attr("lat"), coord.Attr("long")
	words := collapse(latText + " " + longText)
	if words == "" {
		return nil
	}
	coord.Mark()
	coord.Use("lat")
	coord.Use("long")
	latitude, north := degrees(latText, 'N', 'S', 90)
	longitude, east := degrees(longText, 'E', 'W', 180)
	if !north || !east {
		return &model.Coordinates{Wording: words}
	}
	return &model.Coordinates{Latitude: latitude, Longitude: longitude}
}

// marks are what separates degrees, minutes and seconds in the notations people type.
var marks = strings.NewReplacer("°", " ", "º", " ", "′", " ", "″", " ", "'", " ", `"`, " ", ":", " ")

func degrees(text string, positive, negative byte, limit float64) (float64, bool) {
	text = strings.ToUpper(strings.TrimSpace(text))
	sign := 1.0
	for _, cut := range []func(string, string) (string, bool){strings.CutPrefix, strings.CutSuffix} {
		if rest, found := cut(text, string(positive)); found {
			text = rest
		} else if rest, found := cut(text, string(negative)); found {
			text, sign = rest, -1
		}
	}
	if !strings.Contains(text, ".") {
		text = strings.ReplaceAll(text, ",", ".")
	}
	fields := strings.Fields(marks.Replace(text))
	if len(fields) == 0 || len(fields) > 3 {
		return 0, false
	}
	value := 0.0
	for i, field := range fields {
		n, err := strconv.ParseFloat(field, 64)
		if err != nil || (i > 0 && (n < 0 || n >= 60)) {
			return 0, false
		}
		if i == 0 && n < 0 {
			sign, n = -sign, -n
		}
		value += n / []float64{1, 60, 3600}[i]
	}
	if value > limit {
		return 0, false
	}
	return sign * value, true
}
