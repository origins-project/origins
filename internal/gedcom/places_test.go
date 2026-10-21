package gedcom

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// A FORM gives each level its type, the place's own or else the header's, where it has
// as many titles as the place has levels, and an empty level holds its title in line and
// is no place. LANG is the language of every level's name. A place is keyed by what
// contains it, so a Toscana within nothing is not the one within Italia.
func TestAPlaceIsTypedByItsForm(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 PLAC
2 FORM City, County, State, Country
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 PLAC Firenze, , Toscana, Italia
3 LANG Italian
1 DEAT
2 PLAC Fiesole, Firenze, Toscana
3 FORM Comune, Provincia, Regione
`)
	type said struct{ slug, typ, lang, within string }
	var got []said
	for _, slug := range corpus.PlaceSlugs() {
		place := corpus.Places[slug]
		var within string
		if len(place.Within) > 0 {
			within = string(place.Within[0].Place)
		}
		got = append(got, said{string(slug), place.Type, string(place.Name.Lang), within})
	}
	want := []said{
		{"fiesole", "Comune", "", "firenze-toscana"},
		{"firenze", "City", "it", "toscana"},
		{"firenze-toscana", "Provincia", "", "toscana-2"},
		{"italia", "Country", "it", ""},
		{"toscana", "State", "it", "italia"},
		{"toscana-2", "Regione", "", ""},
	}
	if !slices.Equal(got, want) {
		t.Errorf("places are\n%v\nwant\n%v", got, want)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// A FORM that does not line up with the place says what no single level is, and is kept
// as written.
func TestAFormThatDoesNotLineUpIsKept(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 PLAC Firenze, Toscana
3 FORM City, County, State, Country
`)
	if got := corpus.Places["firenze"]; got == nil || got.Type != "" {
		t.Errorf("places are %v", corpus.PlaceSlugs())
	}
	if !reported(notes, "FORM was kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// TRAN, FONE and ROMN write the place again, each level another name of the level it
// writes again where they are divided as the place is, and the most specific place's as
// written where they are not. The coordinates, the identifiers elsewhere and the notes
// are the most specific place's, and coordinates that are not two angles are kept as
// their words.
func TestWhatAMentionSaysOfAPlaceIsThePlaces(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 PLAC Firenze, Toscana
3 TRAN Florence, Tuscany
4 LANG en
3 TRAN Fiorenza
4 LANG it-x-old
3 MAP
4 LATI N43.7696
4 LONG E11.2558
3 EXID Q2044
4 TYPE http://www.wikidata.org/entity/
3 NOTE Cathedral city.
1 DEAT
2 PLAC Montebuono, Toscana
3 ROMN Montebuono, Toscana
4 TYPE as the map spells it
3 MAP
4 LATI three miles north of the church
4 LONG on the old road
3 NOTE Cathedral city.
1 BURI
2 PLAC Firenze, Toscana
3 NOTE Cathedral city.
3 NOTE Its baptistery.
`)
	firenze := corpus.Places["firenze"]
	if firenze == nil {
		t.Fatalf("places are %v, notes %v", corpus.PlaceSlugs(), notes)
	}
	want := []model.PlaceName{{Text: "Florence", Lang: "en"}, {Text: "Fiorenza", Lang: "it-x-old"}}
	if !reflect.DeepEqual(firenze.AlsoKnownAs, want) {
		t.Errorf("Firenze is also known as %+v", firenze.AlsoKnownAs)
	}
	if got := corpus.Places["toscana"].AlsoKnownAs; !reflect.DeepEqual(got, []model.PlaceName{{Text: "Tuscany", Lang: "en"}}) {
		t.Errorf("Toscana is also known as %+v", got)
	}
	if !reflect.DeepEqual(firenze.Coordinates, &model.Coordinates{Latitude: 43.7696, Longitude: 11.2558}) ||
		firenze.IDs["wikidata"][0].ID != "Q2044" || firenze.Body != "Cathedral city.\n\nIts baptistery." {
		t.Errorf("Firenze = %+v", firenze)
	}
	montebuono := corpus.Places["montebuono"]
	if !reflect.DeepEqual(montebuono.Coordinates, &model.Coordinates{Wording: "three miles north of the church on the old road"}) ||
		len(montebuono.AlsoKnownAs) != 0 || montebuono.Body != "Cathedral city." {
		t.Errorf("Montebuono = %+v", montebuono)
	}
	if corpus.Places["toscana"].Body != "" {
		t.Errorf("what is said of a place went to what contains it: %q", corpus.Places["toscana"].Body)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// Mentions of a place within the same place are one place where they agree, and silence
// is no disagreement; one that puts it at other coordinates, or under another identifier
// at the same service, is of a place of its own, which the report names for the
// researcher to merge where it is not.
func TestMentionsThatDisagreeAreTwoPlaces(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 PLAC San Giovanni, Toscana
3 MAP
4 LATI N43.5
4 LONG E11.5
1 CHR
2 PLAC San Giovanni, Toscana
1 DEAT
2 PLAC San Giovanni, Toscana
3 MAP
4 LATI N43.9
4 LONG E11.1
1 BURI
2 PLAC San Giovanni, Toscana
3 EXID 1234
4 TYPE https://www.geonames.org/
1 RESI
2 PLAC San Giovanni, Toscana
3 EXID 5678
4 TYPE https://www.geonames.org/
`)
	var at []model.Slug
	for _, event := range corpus.People["rossi-giovanni"].Events {
		at = append(at, event.Place)
	}
	first, second := at[0], at[2]
	if first == second || !slices.Equal(at, []model.Slug{first, first, second, first, second}) {
		t.Errorf("the events are at %v", at)
	}
	if got := corpus.Places[second].Within; len(got) != 1 || got[0].Place != "toscana" {
		t.Errorf("%s is within %+v", second, got)
	}
	if !reported(notes, "a mention of San Giovanni, Toscana disagrees with another and is a place of its own, "+string(second)) {
		t.Errorf("notes = %v", notes)
	}
}

// A place named only in a script a slug cannot spell is filed under a romanization of
// it where there is one.
func TestAPlaceInAnotherScriptIsFiledUnderItsRomanization(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giorgos /Papadopoulos/
1 BIRT
2 PLAC Θεσσαλονίκη
3 ROMN Thessaloniki
1 DEAT
2 PLAC Αθήνα
`)
	if got := corpus.PlaceSlugs(); !slices.Equal(got, []model.Slug{"place", "thessaloniki"}) {
		t.Fatalf("places are %v, notes %v", got, notes)
	}
	if got := corpus.Places["thessaloniki"]; got.Name.Text != "Θεσσαλονίκη" ||
		!reflect.DeepEqual(got.AlsoKnownAs, []model.PlaceName{{Text: "Thessaloniki", Type: "romanized"}}) {
		t.Errorf("place = %+v", got)
	}
	if strings.Contains(prose(corpus), "PLAC") {
		t.Errorf("a place was kept as written:\n%s", prose(corpus))
	}
}
