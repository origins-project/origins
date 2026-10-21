package model

import (
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"
)

var hideToday = DateOf(time.Date(2026, time.September, 28, 12, 0, 0, 0, time.UTC))

// hideCorpus is three generations nobody alive could be in, with nothing private: a
// couple and their daughter, a birth in a village in a valley in a region, a marriage
// in a city, a register kept in an archive, a picture of the couple, and a place
// nothing names.
func hideCorpus(t *testing.T) *Corpus {
	t.Helper()
	date := func(s string) Date { return mustParse(t, s) }
	c := NewCorpus()
	c.Project = Project{Name: "Rossi", MainPerson: "rossi-anna", FormatVersion: FormatVersion}
	c.People["rossi-giovanni"] = &Person{Slug: "rossi-giovanni",
		Name: PersonName{Given: "Giovanni", Surname: SurnameOf("Rossi")}, Sex: StatedSex{Value: Male},
		Events: []Event{
			{Type: "birth", Date: date("1850"), Place: "borgo",
				Sources: []CitationRef{{Source: "battesimi", Anchor: "1850-3"}}},
			{Type: "death", Date: date("1920")},
		}}
	c.People["verdi-lucia"] = &Person{Slug: "verdi-lucia",
		Name:   PersonName{Given: "Lucia", Surname: SurnameOf("Verdi")},
		Events: []Event{{Type: "death", Date: date("1925")}}}
	c.People["rossi-anna"] = &Person{Slug: "rossi-anna",
		Name:   PersonName{Given: "Anna", Surname: SurnameOf("Rossi")},
		Events: []Event{{Type: "birth", Date: date("1882")}, {Type: "death", Date: date("1960")}}}
	c.Families["rossi-giovanni--verdi-lucia"] = &Family{Slug: "rossi-giovanni--verdi-lucia",
		Partners: PartnersOf("rossi-giovanni", "verdi-lucia"), Type: Married,
		Children: []Child{{ID: "rossi-anna", Sources: []CitationRef{{Source: "battesimi", Anchor: "1850-3"}}}},
		Events:   []Event{{Type: "marriage", Date: date("1880"), Place: "firenze"}},
		Media: []Media{{File: "photos/wedding.jpg", Title: "The wedding",
			Depicts: []Depicted{{Person: "rossi-giovanni"}, {Person: "verdi-lucia"}}}},
		Body: "Married in the Duomo.", Files: []string{"photos/wedding.jpg"}}
	c.Places["borgo"] = &Place{Slug: "borgo", Name: PlaceName{Text: "Borgo"}, Within: []PlaceRef{{Place: "mugello"}}}
	c.Places["mugello"] = &Place{Slug: "mugello", Name: PlaceName{Text: "Mugello"}, Within: []PlaceRef{{Place: "toscana"}}}
	c.Places["toscana"] = &Place{Slug: "toscana", Name: PlaceName{Text: "Toscana"}}
	c.Places["firenze"] = &Place{Slug: "firenze", Name: PlaceName{Text: "Firenze"}}
	c.Places["nowhere"] = &Place{Slug: "nowhere", Name: PlaceName{Text: "Nowhere"}}
	c.Sources["battesimi"] = &Source{Slug: "battesimi", Title: "Battesimi",
		Repositories: []Holding{{Repository: "archivio"}},
		Citations:    map[string]Citation{"1850-3": {Text: "Giovanni Rossi"}}}
	c.Repositories["archivio"] = &Repository{Slug: "archivio", Name: "Archivio"}
	return c
}

func TestHidingNothingIsTheCorpus(t *testing.T) {
	c := hideCorpus(t)
	if Hide(c, Hiding{}, hideToday) != c {
		t.Error("a view asking to hide nothing is not the corpus it was handed")
	}
	for _, hiding := range []Hiding{{Private: true}, {Living: true}, {Private: true, Living: true}} {
		if got := Hide(c, hiding, hideToday); !reflect.DeepEqual(got, c) {
			t.Errorf("%+v: a corpus with nothing to hide changed:\n%+v", hiding, got)
		}
	}
}

func TestEitherKindHidesAPerson(t *testing.T) {
	date := func(s string) Date { return mustParse(t, s) }
	c := NewCorpus()
	c.People["dead"] = &Person{Slug: "dead", Events: []Event{{Type: "death", Date: date("1900")}}}
	c.People["dead-private"] = &Person{Slug: "dead-private", Private: true,
		Events: []Event{{Type: "death", Date: date("1900")}}}
	c.People["living"] = &Person{Slug: "living", Events: []Event{{Type: "birth", Date: date("1990")}}}
	c.People["living-private"] = &Person{Slug: "living-private", Private: true}
	c.People["overruled"] = &Person{Slug: "overruled", Living: LivingFalse}

	for _, v := range []struct {
		hiding Hiding
		shown  []Slug
	}{
		{Hiding{Private: true}, []Slug{"dead", "living", "overruled", "private-1", "private-2"}},
		{Hiding{Living: true}, []Slug{"dead", "dead-private", "overruled", "private-1", "private-2"}},
		{Hiding{Private: true, Living: true}, []Slug{"dead", "overruled", "private-1", "private-2", "private-3"}},
	} {
		got := Hide(c, v.hiding, hideToday)
		if shown := got.PeopleSlugs(); !slices.Equal(shown, v.shown) {
			t.Errorf("%+v: shows %v, want %v", v.hiding, shown, v.shown)
		}
		for _, slug := range got.PeopleSlugs() {
			person := got.People[slug]
			if person.Name.Text == PlaceholderName && !person.Private {
				t.Errorf("%+v: placeholder %s is not private", v.hiding, slug)
			}
			if placed := c.People[slug] == nil; person.IsPlaceholder() != placed {
				t.Errorf("%+v: %s is taken for a placeholder: %v, want %v", v.hiding, slug, !placed, placed)
			}
		}
	}
}

func TestAHiddenPersonIsAPlaceholder(t *testing.T) {
	c := hideCorpus(t)
	anna := c.People["rossi-anna"]
	anna.Private = true
	anna.Sex = StatedSex{Value: Female, Sources: []CitationRef{{Source: "battesimi", Anchor: "1850-3"}}}
	anna.AlsoKnownAs = []PersonName{{Given: "Annina", Surname: SurnameOf("Rossi")}}
	c.Families["rossi-giovanni--verdi-lucia"].Children[0].Relation.Word = Adoptive
	anna.Body = "Anna kept the shop."
	anna.Sources = []CitationRef{{Source: "battesimi", Anchor: "1850-3"}}
	anna.IDs = IDs{"wikitree": IdentifiersOf("Rossi-1")}
	anna.Media = []Media{{File: "photos/anna.jpg", Depicts: []Depicted{{Person: "rossi-anna"}}}}
	anna.Portrait = Image{File: "photos/anna.jpg"}
	anna.Files = []string{"notes.pdf", "photos/anna.jpg"}
	c.People["verdi-lucia"].Private = true
	// a slug a placeholder would have taken, held by somebody who is shown
	c.People["private-1"] = &Person{Slug: "private-1", Living: LivingFalse}

	got := Hide(c, Hiding{Private: true}, hideToday)

	want := &Person{Slug: "private-2", Private: true, Name: PersonName{Text: PlaceholderName}}
	if !reflect.DeepEqual(got.People["private-2"], want) {
		t.Errorf("rossi-anna is shown as\n%+v\nwant\n%+v", got.People["private-2"], want)
	}
	family := got.Families["private-3--rossi-giovanni"]
	if want := []Child{{ID: "private-2", Relation: Relationship{Word: Adoptive}}}; family == nil || !reflect.DeepEqual(family.Children, want) {
		t.Errorf("the family %v is %+v, listing %+v", got.FamilySlugs(), family, want)
	}
	if shown := got.PeopleSlugs(); !slices.Equal(shown, []Slug{"private-1", "private-2", "private-3", "rossi-giovanni"}) {
		t.Errorf("shows %v, numbered past the one already taken", shown)
	}
	if got.Project.MainPerson != "private-2" {
		t.Errorf("the main person is %s, not her placeholder", got.Project.MainPerson)
	}
	if !reflect.DeepEqual(got.People["private-1"], c.People["private-1"]) {
		t.Error("the person already called private-1 changed")
	}
}

func TestAReferenceToAHiddenPersonIsToThePlaceholder(t *testing.T) {
	c := hideCorpus(t)
	c.People["verdi-lucia"].Private = true
	giovanni := c.People["rossi-giovanni"]
	giovanni.Events = append(giovanni.Events, Event{Type: "baptism", With: []Named{
		{Person: "verdi-lucia", Role: "godmother", Age: "20y",
			Sources: []CitationRef{{Source: "battesimi", Anchor: "1850-3"}}},
		{Name: PersonName{Given: "Pietro", Surname: SurnameOf("Neri")}, Role: "godfather", Age: "30y"},
	}})
	giovanni.Media = []Media{
		{File: "photos/with-lucia.jpg", Depicts: []Depicted{{Person: "rossi-giovanni"}, {Person: "verdi-lucia"}}},
		{File: "photos/alone.jpg", Depicts: []Depicted{{Person: "rossi-giovanni"}}},
	}
	giovanni.Portrait = Image{File: "photos/with-lucia.jpg", Region: &Region{Right: 0.5, Bottom: 1}}
	giovanni.Files = []string{"photos/alone.jpg", "photos/with-lucia.jpg"}
	register := c.Sources["battesimi"]
	register.Citations["1850-3"] = Citation{Text: "Giovanni Rossi", With: []Named{
		{Person: "rossi-giovanni", Role: "child"},
		{Person: "verdi-lucia", Role: "mother", Age: "20y", Note: "named as Lucia"},
	}, Images: []Media{
		{File: "scans/1850-3.jpg"},
		{File: "scans/1850-3-margin.jpg", Depicts: []Depicted{{Person: "verdi-lucia"}}},
	}}
	register.Files = []string{"scans/1850-3-margin.jpg", "scans/1850-3.jpg"}
	before := c.Clone()

	got := Hide(c, Hiding{Private: true}, hideToday)

	parents := got.ParentsOf("rossi-anna")
	if want := []Parent{{"rossi-giovanni", Birth, "private-1--rossi-giovanni"},
		{"private-1", Birth, "private-1--rossi-giovanni"}}; !reflect.DeepEqual(parents, want) {
		t.Errorf("her daughter's parents are %+v, want %+v", parents, want)
	}
	with := got.People["rossi-giovanni"].Events[2].With
	if want := (Named{Person: "private-1", Role: "godmother"}); !reflect.DeepEqual(with[0], want) {
		t.Errorf("she is named as %+v, want %+v", with[0], want)
	}
	if !reflect.DeepEqual(with[1], before.People["rossi-giovanni"].Events[2].With[1]) {
		t.Errorf("somebody with no record is named as %+v", with[1])
	}

	g := got.People["rossi-giovanni"]
	if len(g.Media) != 1 || g.Media[0].File != "photos/alone.jpg" ||
		!slices.Equal(g.Files, []string{"photos/alone.jpg"}) {
		t.Errorf("the picture with her in it is still there: %+v, files %v", g.Media, g.Files)
	}
	if g.Portrait != (Image{}) {
		t.Errorf("the portrait still names a picture that has gone: %+v", g.Portrait)
	}

	entry := got.Sources["battesimi"].Citations["1850-3"]
	if want := []Named{{Person: "rossi-giovanni", Role: "child"}, {Person: "private-1", Role: "mother"}}; !reflect.DeepEqual(entry.With, want) {
		t.Errorf("the entry names %+v, want %+v", entry.With, want)
	}
	if want := []Media{{File: "scans/1850-3.jpg"}}; !reflect.DeepEqual(entry.Images, want) ||
		!slices.Equal(got.Sources["battesimi"].Files, []string{"scans/1850-3.jpg"}) {
		t.Errorf("the scan showing her is still there: %+v, files %v", entry.Images, got.Sources["battesimi"].Files)
	}

	family, ok := got.Families["private-1--rossi-giovanni"]
	if want := (&Family{Slug: "private-1--rossi-giovanni", Private: true,
		Partners: PartnersOf("rossi-giovanni", "private-1"),
		Children: []Child{{ID: "rossi-anna"}}}); !ok || !reflect.DeepEqual(family, want) {
		t.Errorf("the family is %v %+v, want %+v", got.FamilySlugs(), family, want)
	}

	if !reflect.DeepEqual(c, before) {
		t.Error("hiding changed the corpus it was handed")
	}
}

func TestAPrivateFamilyKeepsItsPartners(t *testing.T) {
	c := hideCorpus(t)
	c.Families["rossi-giovanni--verdi-lucia"].Private = true

	got := Hide(c, Hiding{Private: true}, hideToday)
	want := &Family{Slug: "rossi-giovanni--verdi-lucia", Private: true,
		Partners: PartnersOf("rossi-giovanni", "verdi-lucia"), Children: []Child{{ID: "rossi-anna"}}}
	if family := got.Families["rossi-giovanni--verdi-lucia"]; !reflect.DeepEqual(family, want) {
		t.Errorf("the family is %+v, want %+v", family, want)
	}
	if !reflect.DeepEqual(Hide(c, Hiding{Living: true}, hideToday), c) {
		t.Error("hiding the living hides a private family of the dead")
	}
}

// A couple who married twice are two families in a hidden view as well, the second
// still told apart by what told it apart when a placeholder takes a partner's place.
func TestTwoFamiliesOfAHiddenCoupleStayTwo(t *testing.T) {
	c := hideCorpus(t)
	c.Families["rossi-giovanni--verdi-lucia-1870"] = &Family{Slug: "rossi-giovanni--verdi-lucia-1870",
		Partners: PartnersOf("verdi-lucia", "rossi-giovanni")}
	c.People["verdi-lucia"].Private = true

	got := Hide(c, Hiding{Private: true}, hideToday)
	for _, slug := range []Slug{"private-1--rossi-giovanni", "private-1--rossi-giovanni-1870"} {
		if got.Families[slug] == nil {
			t.Errorf("%s is not among %v", slug, got.FamilySlugs())
		}
	}
}

func TestAPrivateEntryIsRemoved(t *testing.T) {
	c := hideCorpus(t)
	anna := c.People["rossi-anna"]
	anna.Events = append(anna.Events, Event{Private: true, Type: "illness", Note: "a heart operation"})
	anna.AlsoKnownAs = []PersonName{
		{Private: true, Given: "Nina"},
		{Given: "Annetta"},
	}
	anna.Media = []Media{{Private: true, File: "photos/hospital.jpg"}, {File: "photos/shop.jpg"}}
	anna.Portrait = Image{File: "photos/hospital.jpg"}
	anna.Files = []string{"photos/hospital.jpg", "photos/shop.jpg"}
	c.Families["rossi-giovanni--verdi-lucia"].Media[0].Private = true

	got := Hide(c, Hiding{Private: true}, hideToday).People["rossi-anna"]
	if len(got.Events) != 2 || got.Events[1].Type != "death" {
		t.Errorf("events are %+v, the illness still among them", got.Events)
	}
	if want := []PersonName{{Given: "Annetta"}}; !reflect.DeepEqual(got.AlsoKnownAs, want) {
		t.Errorf("other names are %+v, want %+v", got.AlsoKnownAs, want)
	}
	if len(got.Media) != 1 || got.Media[0].File != "photos/shop.jpg" ||
		!slices.Equal(got.Files, []string{"photos/shop.jpg"}) || got.Portrait != (Image{}) {
		t.Errorf("pictures %+v, files %v, portrait %+v", got.Media, got.Files, got.Portrait)
	}
	family := Hide(c, Hiding{Private: true}, hideToday).Families["rossi-giovanni--verdi-lucia"]
	if family.Media != nil || family.Files != nil {
		t.Errorf("the family's private picture is still there: %+v %v", family.Media, family.Files)
	}

	if !reflect.DeepEqual(Hide(c, Hiding{Living: true}, hideToday), c) {
		t.Error("hiding the living removes what is private")
	}
}

// A picture removed from the record keeping it goes from every record referring to it,
// and an event's pictures are removed as the record's are.
func TestAReferenceToARemovedPictureGoes(t *testing.T) {
	c := hideCorpus(t)
	anna := c.People["rossi-anna"]
	anna.Events[1].Media = []Media{{Private: true, File: "photos/stone.jpg", Type: "tombstone"}}
	anna.Media = []Media{{File: "photos/shop.jpg"}}
	anna.Files = []string{"photos/shop.jpg", "photos/stone.jpg"}
	family := c.Families["rossi-giovanni--verdi-lucia"]
	family.Events[0].Media = []Media{{File: "../../people/rossi-anna/photos/stone.jpg"}}
	family.Media = append(family.Media,
		Media{File: "../../people/rossi-anna/photos/shop.jpg", Note: "Anna's"},
		Media{File: "../../people/rossi-anna/photos/stone.jpg", Note: "Anna's grave"})

	view := Hide(c, Hiding{Private: true}, hideToday)
	got := view.People["rossi-anna"]
	if got.Events[1].Media != nil || !slices.Equal(got.Files, []string{"photos/shop.jpg"}) {
		t.Errorf("the private gravestone is still there: %+v %v", got.Events[1].Media, got.Files)
	}
	shown := view.Families["rossi-giovanni--verdi-lucia"]
	if shown.Events[0].Media != nil || len(shown.Media) != 2 || shown.Media[1].Note != "Anna's" {
		t.Errorf("the family refers to %+v and %+v", shown.Events[0].Media, shown.Media)
	}
	if len(family.Media) != 3 || family.Events[0].Media == nil {
		t.Error("the view wrote into the corpus it was handed")
	}

	// a picture of somebody hidden goes with them, and so does a reference to one
	anna.Living = LivingTrue
	view = Hide(c, Hiding{Living: true}, hideToday)
	shown = view.Families["rossi-giovanni--verdi-lucia"]
	if len(shown.Media) != 1 || shown.Events[0].Media != nil {
		t.Errorf("the family still refers to a hidden person's pictures: %+v %+v", shown.Media, shown.Events[0].Media)
	}
}

func TestAPrivatePlaceSourceOrRepositoryIsDropped(t *testing.T) {
	c := hideCorpus(t)
	c.Places["mugello"].Private = true
	c.Places["borgo"].Within = append(c.Places["borgo"].Within, PlaceRef{Place: "toscana"})
	c.People["rossi-anna"].Events[0].Place = "mugello"
	c.People["rossi-anna"].Media = []Media{{File: "photos/view.jpg", Place: "mugello"}}
	c.Sources["battesimi"].Private = true
	c.Sources["morti"] = &Source{Slug: "morti", Title: "Morti",
		Covers:       []Coverage{{Events: []string{"death"}, Place: "mugello"}},
		Repositories: []Holding{{Repository: "carte", CallNumber: "M/1"}, {Repository: "archivio", CallNumber: "M/2"}},
		Citations:    map[string]Citation{"1960-1": {}}}
	c.Repositories["carte"] = &Repository{Slug: "carte", Private: true}
	c.People["rossi-anna"].Events[1].Sources = []CitationRef{{Source: "morti", Anchor: "1960-1"}}

	got := Hide(c, Hiding{Private: true}, hideToday)

	if slugs := slices.Sorted(maps.Keys(got.Places)); !slices.Equal(slugs, []Slug{"borgo", "firenze", "nowhere", "toscana"}) {
		t.Errorf("places are %v", slugs)
	}
	if want := []PlaceRef{{Place: "toscana"}}; !reflect.DeepEqual(got.Places["borgo"].Within, want) {
		t.Errorf("the village lies within %+v, want %+v", got.Places["borgo"].Within, want)
	}
	anna := got.People["rossi-anna"]
	if anna.Events[0].Place != "" || anna.Media[0].Place != "" {
		t.Errorf("an event and a picture still name the private place: %q %q",
			anna.Events[0].Place, anna.Media[0].Place)
	}

	if _, kept := got.Sources["battesimi"]; kept {
		t.Error("the private source is still there")
	}
	if got.People["rossi-giovanni"].Events[0].Sources != nil || got.Families["rossi-giovanni--verdi-lucia"].Children[0].Sources != nil {
		t.Error("a citation of the private source is still there")
	}
	if _, kept := got.Repositories["carte"]; kept {
		t.Error("the private repository is still there")
	}
	if want := []Holding{{Repository: "archivio", CallNumber: "M/2"}}; !reflect.DeepEqual(got.Sources["morti"].Repositories, want) {
		t.Errorf("the register is kept in %+v, want %+v", got.Sources["morti"].Repositories, want)
	}
	if want := []Coverage{{Events: []string{"death"}}}; !reflect.DeepEqual(got.Sources["morti"].Covers, want) {
		t.Errorf("the register covers %+v, want %+v", got.Sources["morti"].Covers, want)
	}
}

func TestALinkToWhatWentGoesWithItsText(t *testing.T) {
	c := hideCorpus(t)
	c.Notes["the-carter"] = &Note{Slug: "the-carter", Private: true, Body: "# He was a carter\n"}
	c.Places["nowhere"].Private = true
	c.People["rossi-anna"].Body = "Her father [He was a carter](../../notes/the-carter/README.md), and she\n" +
		"was born in [Borgo](../../places/borgo/README.md), not [Nowhere](../../places/nowhere/README.md).\n" +
		"Her mother is [Lucia](../../people/verdi-lucia/README.md).\n"
	c.People["verdi-lucia"].Private = true

	got := Hide(c, Hiding{Private: true}, hideToday)

	want := "Her father, and she\nwas born in [Borgo](../../places/borgo/README.md), not.\n" +
		"Her mother is [Lucia](../../people/verdi-lucia/README.md).\n"
	if body := got.People["rossi-anna"].Body; body != want {
		t.Errorf("body =\n%s\nwant\n%s", body, want)
	}
	if c.People["rossi-anna"].Body == want {
		t.Error("hiding changed the corpus it was handed")
	}
}

// A private note goes as a private place does, and a claim that two records are one
// stays only where the view shows both as they are.
func TestAClaimStaysOnlyWhereTheViewShowsBoth(t *testing.T) {
	c := hideCorpus(t)
	c.People["verdi-lucia"].Private = true
	c.Places["nowhere"].SameAs = []SameAs{{ID: "firenze"}, {ID: "prato"}, {Name: "Firenze vecchia"}}
	c.Places["prato"] = &Place{Slug: "prato", Name: PlaceName{Text: "Prato"}}
	c.People["rossi-anna"].SameAs = []SameAs{{ID: "verdi-lucia", Note: "n"}, {ID: "rossi-giovanni"}}
	c.Families["verdi-lucia--rossi-giovanni"] = &Family{Slug: "verdi-lucia--rossi-giovanni",
		Partners: PartnersOf("rossi-giovanni", "verdi-lucia"),
		SameAs:   []SameAs{{ID: "rossi-giovanni--verdi-lucia"}}}
	c.Notes["borgo-1850"] = &Note{Slug: "borgo-1850", Private: true, SameAs: []SameAs{{ID: "nowhere"}},
		Events: []Event{{Type: "baptism", Place: "prato", With: []Named{{Person: "verdi-lucia", Role: "godmother"}}}}}
	c.Notes["firenze-1880"] = &Note{Slug: "firenze-1880", SameAs: []SameAs{{ID: "borgo-1850"}},
		Events: []Event{{Type: "marriage", Place: "firenze", With: []Named{{Person: "verdi-lucia", Role: "witness"}}}}}

	got := Hide(c, Hiding{Private: true}, hideToday)

	if slugs := got.NoteSlugs(); !slices.Equal(slugs, []Slug{"firenze-1880"}) {
		t.Errorf("notes are %v", slugs)
	}
	if want := []Named{{Person: "private-1", Role: "witness"}}; !reflect.DeepEqual(got.Notes["firenze-1880"].Events[0].With, want) {
		t.Errorf("the note names %+v, want %+v", got.Notes["firenze-1880"].Events[0].With, want)
	}
	if same := got.Notes["firenze-1880"].SameAs; same != nil {
		t.Errorf("the note is still the same as %+v, which went", same)
	}
	if want := []SameAs{{ID: "firenze"}, {Name: "Firenze vecchia"}}; !reflect.DeepEqual(got.Places["nowhere"].SameAs, want) {
		t.Errorf("the place is the same as %+v, want %+v", got.Places["nowhere"].SameAs, want)
	}
	if want := []SameAs{{ID: "rossi-giovanni"}}; !reflect.DeepEqual(got.People["rossi-anna"].SameAs, want) {
		t.Errorf("the daughter is the same as %+v, want %+v", got.People["rossi-anna"].SameAs, want)
	}
	for slug, family := range got.Families {
		if family.SameAs != nil {
			t.Errorf("family/%s is the same as %+v, a family shown as its partners alone", slug, family.SameAs)
		}
	}
	// only the private note used the town, a claim being no use of a record
	if _, kept := got.Places["prato"]; kept {
		t.Error("a place only a private note and a claim named is still there")
	}
	if len(c.Places["nowhere"].SameAs) != 3 || len(c.People["rossi-anna"].SameAs) != 2 {
		t.Error("hiding changed the corpus it was handed")
	}
}

func TestABookmarkStaysOnlyWhereTheViewShowsItsRecord(t *testing.T) {
	c := hideCorpus(t)
	c.People["verdi-lucia"].Private = true
	c.Project.Bookmarks = []EntityRef{
		{Kind: KindPerson, Slug: "rossi-anna"},
		{Kind: KindPerson, Slug: "verdi-lucia"},
		{Kind: KindFamily, Slug: "rossi-giovanni--verdi-lucia"},
		{Kind: KindPlace, Slug: "nowhere"},
	}

	got := Hide(c, Hiding{Private: true}, hideToday)

	want := []EntityRef{{Kind: KindPerson, Slug: "rossi-anna"}, {Kind: KindPlace, Slug: "nowhere"}}
	if !reflect.DeepEqual(got.Project.Bookmarks, want) {
		t.Errorf("the bookmarks are %v, want %v", got.Project.Bookmarks, want)
	}
	if len(c.Project.Bookmarks) != 4 {
		t.Error("hiding changed the corpus it was handed")
	}
}

func TestWhatOnlyHiddenThingsReferredToIsDropped(t *testing.T) {
	c := hideCorpus(t)
	c.People["rossi-giovanni"].Private = true
	c.Families["rossi-giovanni--verdi-lucia"].Children[0].Sources = nil
	c.Sources["censimento"] = &Source{Slug: "censimento", Title: "Censimento",
		Citations: map[string]Citation{"p1": {}}}
	// a hypothesis's references keep nothing and drop nothing
	c.Hypotheses["born-1851"] = &Hypothesis{Slug: "born-1851", Status: Open,
		Supports: []CitationRef{{Source: "censimento", Anchor: "p1"}, {Source: "battesimi", Anchor: "1850-3"}}}

	got := Hide(c, Hiding{Private: true}, hideToday)

	// his birth named the village, which the valley contains, which the region contains;
	// the marriage named the city, and went with the family's events
	if slugs := got.PlaceSlugs(); !slices.Equal(slugs, []Slug{"nowhere"}) {
		t.Errorf("places are %v, want only the one nobody ever named", slugs)
	}
	// his birth cited the register, which the archive keeps
	if slugs := got.SourceSlugs(); !slices.Equal(slugs, []Slug{"censimento"}) {
		t.Errorf("sources are %v, want only the one a hypothesis alone cites", slugs)
	}
	if slugs := got.RepositorySlugs(); len(slugs) != 0 {
		t.Errorf("repositories are %v", slugs)
	}
	if got.Hypotheses == nil || len(got.Hypotheses) != 0 {
		t.Errorf("hypotheses are %v, want none", got.HypothesisSlugs())
	}
	if len(c.Places) != 5 || len(c.Hypotheses) != 1 {
		t.Error("hiding changed the corpus it was handed")
	}
}
