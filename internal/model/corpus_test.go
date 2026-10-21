package model

import (
	"fmt"
	"slices"
	"testing"
)

// rossiFamilies mirrors the families of the fixture corpus and the children they list.
// Everyone listed in no family is someone the tree has no parents for, which is the
// ordinary state of the people at its edges.
var rossiFamilies = []struct {
	slug     Slug
	partners []Slug
	children []Child
}{
	{"bianchi-maria--rossi-giovanni", []Slug{"bianchi-maria", "rossi-giovanni"},
		append(ChildrenOf("rossi-anna", "rossi-pietro", "rossi-paolo", "rossi-infante-1866"), Child{ID: Unknown})},
	{"conti-teresa--none", []Slug{"conti-teresa", None},
		[]Child{{Name: PersonName{Text: "Gemma Nocentini"}, Relation: Relationship{Word: Adoptive}}}},
	{"conti-teresa--unknown", []Slug{"conti-teresa", Unknown}, []Child{{ID: "conti-luigi", Relation: Relationship{Word: "natural"}}}},
	{"ferri-stefano--rossi-marco", []Slug{"ferri-stefano", "rossi-marco"},
		[]Child{{ID: "ferri-rossi-sasha", Relation: Relationship{Word: RelationUnknown}}}},
	{"galli-ines--rossi-pietro", []Slug{"galli-ines", "rossi-pietro"}, []Child{{ID: "rossi-giovanni-1902"},
		{ID: "rossi-elena", Relation: Relationship{Partners: []PartnerRelation{{"galli-ines", Step}, {"rossi-pietro", Adoptive}}}}}},
	{"landi-rosa--rossi-francesco", []Slug{"landi-rosa", "rossi-francesco"}, ChildrenOf("rossi-antonio")},
	{"marchi-paolo--rossi-claudia", []Slug{"marchi-paolo", "rossi-claudia"}, ChildrenOf("rossi-marco")},
	{"moretti-giulia--unknown", []Slug{"moretti-giulia", Unknown}, ChildrenOf("rossi-elena")},
	{"neri-caterina--rossi-antonio", []Slug{"neri-caterina", "rossi-antonio"},
		[]Child{{ID: "rossi-domenico"}, {ID: "rossi-lorenzo", Status: Disproven}}},
	{"rossi-annunziata--unknown", []Slug{"rossi-annunziata", Unknown}, ChildrenOf("rossi-lorenzo")},
	{"rossi-antonio--verdi-lucia", []Slug{"rossi-antonio", "verdi-lucia"},
		ChildrenOf("rossi-giovanni", "rossi-giuseppe", "rossi-annunziata")},
	{"rossi-bartolomeo--sarti-angela", []Slug{"rossi-bartolomeo", "sarti-angela"}, nil},
	{"rossi-bartolomeo--sarti-angela-1778", []Slug{"rossi-bartolomeo", "sarti-angela"}, ChildrenOf("rossi-francesco")},
	{"rossi-claudia--sanna-davide", []Slug{"rossi-claudia", "sanna-davide"}, nil},
	{"rossi-giovanni-1902--unknown", []Slug{"rossi-giovanni-1902", Unknown}, ChildrenOf("rossi-claudia")},
}

var rossiOrphans = []Slug{
	"bianchi-maria", "conti-teresa", "ferri-stefano", "galli-ines", "landi-rosa",
	"marchi-paolo", "moretti-giulia", "neri-caterina", "rossi-bartolomeo",
	"sanna-davide", "sarti-angela", "verdi-lucia",
}

func rossi(t *testing.T) *Corpus {
	t.Helper()
	c := NewCorpus()
	for _, slug := range rossiOrphans {
		c.People[slug] = &Person{Slug: slug}
	}
	for _, f := range rossiFamilies {
		c.Families[f.slug] = &Family{Slug: f.slug, Partners: PartnersOf(f.partners...),
			Children: f.children}
		for _, child := range f.children {
			if child.Somebody() {
				c.People[child.ID] = &Person{Slug: child.ID}
			}
		}
	}
	// the families share their lists with the table until cloned
	return c.Clone()
}

func assertSlugs(t *testing.T, what string, got, want []Slug) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestChildren(t *testing.T) {
	c := rossi(t)
	assertSlugs(t, "children of rossi-giovanni", c.Children("rossi-giovanni"),
		[]Slug{"rossi-anna", "rossi-infante-1866", "rossi-paolo", "rossi-pietro"})
	assertSlugs(t, "children of conti-teresa", c.Children("conti-teresa"),
		[]Slug{"conti-luigi"})
	assertSlugs(t, "children of sanna-davide", c.Children("sanna-davide"), nil)
}

// A person's parents are the partners of every family listing them, each with the
// relation the family states to that partner. A disproven link is no parentage, and
// none says the child is no child of that partner.
func TestParentsOf(t *testing.T) {
	c := rossi(t)
	for _, want := range []struct {
		person  Slug
		parents []Parent
	}{
		{"rossi-elena", []Parent{
			{"galli-ines", Step, "galli-ines--rossi-pietro"},
			{"rossi-pietro", Adoptive, "galli-ines--rossi-pietro"},
			{"moretti-giulia", Birth, "moretti-giulia--unknown"},
			{Unknown, Birth, "moretti-giulia--unknown"},
		}},
		{"rossi-lorenzo", []Parent{
			{"rossi-annunziata", Birth, "rossi-annunziata--unknown"},
			{Unknown, Birth, "rossi-annunziata--unknown"},
		}},
		{"conti-luigi", []Parent{
			{"conti-teresa", "natural", "conti-teresa--unknown"},
			{Unknown, "natural", "conti-teresa--unknown"},
		}},
		{"rossi-bartolomeo", nil},
	} {
		if got := c.ParentsOf(want.person); !slices.Equal(got, want.parents) {
			t.Errorf("parents of %s are %+v, want %+v", want.person, got, want.parents)
		}
	}

	c.Families["galli-ines--rossi-pietro"].Children[1].Relation.Partners[0].Relation = NoRelation
	assertSlugs(t, "children of galli-ines once she is none to Elena", c.Children("galli-ines"),
		[]Slug{"rossi-giovanni-1902"})
	assertSlugs(t, "children of rossi-antonio", c.Children("rossi-antonio"),
		[]Slug{"rossi-annunziata", "rossi-domenico", "rossi-giovanni", "rossi-giuseppe"})
}

func TestSiblings(t *testing.T) {
	c := rossi(t)

	full, half := c.Siblings("rossi-giovanni")
	assertSlugs(t, "full siblings of rossi-giovanni", full,
		[]Slug{"rossi-annunziata", "rossi-giuseppe"})
	assertSlugs(t, "half siblings of rossi-giovanni", half, []Slug{"rossi-domenico"})

	// Domenico shares only his father, so everyone on that side is a half sibling
	full, half = c.Siblings("rossi-domenico")
	assertSlugs(t, "full siblings of rossi-domenico", full, nil)
	assertSlugs(t, "half siblings of rossi-domenico", half,
		[]Slug{"rossi-annunziata", "rossi-giovanni", "rossi-giuseppe"})

	// Elena was adopted by Giovanni-1902's parents, which makes them siblings
	full, half = c.Siblings("rossi-elena")
	assertSlugs(t, "full siblings of rossi-elena", full, []Slug{"rossi-giovanni-1902"})
	assertSlugs(t, "half siblings of rossi-elena", half, nil)

	full, half = c.Siblings("conti-luigi")
	if full != nil || half != nil {
		t.Errorf("an only child has no siblings, got %q and %q", full, half)
	}
	if full, half := c.Siblings("nobody"); full != nil || half != nil {
		t.Errorf("an unknown person has no siblings, got %q and %q", full, half)
	}
}

func TestFamiliesAndPartners(t *testing.T) {
	c := rossi(t)
	assertSlugs(t, "families of rossi-giovanni", c.FamiliesOf("rossi-giovanni"),
		[]Slug{"bianchi-maria--rossi-giovanni"})
	assertSlugs(t, "partners of rossi-giovanni", c.Partners("rossi-giovanni"),
		[]Slug{"bianchi-maria"})

	// Claudia married twice, which is two families and not a special case
	assertSlugs(t, "families of rossi-claudia", c.FamiliesOf("rossi-claudia"),
		[]Slug{"marchi-paolo--rossi-claudia", "rossi-claudia--sanna-davide"})
	assertSlugs(t, "partners of rossi-claudia", c.Partners("rossi-claudia"),
		[]Slug{"marchi-paolo", "sanna-davide"})

	assertSlugs(t, "families of conti-luigi", c.FamiliesOf("conti-luigi"), nil)
	assertSlugs(t, "families conti-luigi is a child of", c.ChildOf("conti-luigi"),
		[]Slug{"conti-teresa--unknown"})
	assertSlugs(t, "families rossi-lorenzo is a child of", c.ChildOf("rossi-lorenzo"),
		[]Slug{"rossi-annunziata--unknown"})
}

func TestTiesAreEachPersonsFamilies(t *testing.T) {
	c := rossi(t)
	// a partner listed twice is still a partner in the family once
	c.Families["rossi-claudia--sanna-davide"].Partners = PartnersOf("rossi-claudia", "sanna-davide", "rossi-claudia")
	ties := c.Ties()
	for _, person := range c.PeopleSlugs() {
		assertSlugs(t, "families of "+string(person), ties.PartnerIn[person], c.FamiliesOf(person))
		assertSlugs(t, "families "+string(person)+" is a child of", ties.ChildIn[person], c.ChildOf(person))
	}
}

// A partner without a record is nobody's partner, whatever the family says of them.
func TestAPartnerWithoutARecordIsNobodysPartner(t *testing.T) {
	c := rossi(t)
	c.Families["rossi-giovanni--unknown"] = &Family{Slug: "rossi-giovanni--unknown", Partners: []Partner{
		{Person: "rossi-giovanni"}, {Name: PersonName{Text: "Mr Stockdale"}}}}
	c.Families["moretti-giulia--none"] = &Family{Slug: "moretti-giulia--none", Partners: PartnersOf("moretti-giulia", None)}
	assertSlugs(t, "partners of rossi-giovanni", c.Partners("rossi-giovanni"), []Slug{"bianchi-maria"})
	assertSlugs(t, "partners of moretti-giulia", c.Partners("moretti-giulia"), nil)
	assertSlugs(t, "families of moretti-giulia", c.FamiliesOf("moretti-giulia"),
		[]Slug{"moretti-giulia--none", "moretti-giulia--unknown"})
	for _, word := range []Slug{Unknown, None} {
		if !c.Taken(word) {
			t.Errorf("%q is free to name a record", word)
		}
	}
}

// What tells a family apart from another of its partners is the year of its first
// dated event, where that was, and the surname of the first child it lists who has a
// record and is not disproven.
func TestFamilyHints(t *testing.T) {
	c := rossi(t)
	c.People["rossi-annunziata"].Name = PersonName{Given: "Annunziata", Surname: SurnameOf("Rossi")}
	c.People["rossi-domenico"].Name = PersonName{Given: "Domenico", Surname: SurnameOf("Neri Rossi")}
	year := 1832
	family := &Family{Partners: PartnersOf("rossi-antonio", "verdi-lucia"), Events: []Event{
		{Type: "residence"}, {Type: "marriage", Date: Date{Year: &year}, Place: "borgo-san-lorenzo"},
	}, Children: []Child{{Name: PersonName{Surname: SurnameOf("Verdi")}}, {ID: "rossi-domenico", Status: Disproven},
		{ID: "rossi-annunziata"}, {ID: "rossi-domenico"}}}
	if got := c.FamilyHints(family); !slices.Equal(got, []string{"1832", "borgo-san-lorenzo", "Rossi"}) {
		t.Errorf("the hints are %q", got)
	}
	nobody := &Family{Partners: PartnersOf(Unknown, Unknown)}
	if got := c.FamilyHints(nobody); got != nil {
		t.Errorf("a family of nobody known is told apart by %q", got)
	}
}

func assertGenerations(t *testing.T, what string, got, want [][]Slug) {
	t.Helper()
	if !slices.EqualFunc(got, want, slices.Equal[[]Slug]) {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func TestAncestors(t *testing.T) {
	c := rossi(t)
	assertGenerations(t, "ancestors of rossi-giovanni-1902", c.Ancestors("rossi-giovanni-1902"),
		[][]Slug{
			{"galli-ines", "rossi-pietro"},
			{"bianchi-maria", "rossi-giovanni"},
			{"rossi-antonio", "verdi-lucia"},
			{"landi-rosa", "rossi-francesco"},
			{"rossi-bartolomeo", "sarti-angela"},
		})
	assertGenerations(t, "ancestors of rossi-bartolomeo", c.Ancestors("rossi-bartolomeo"), nil)
}

func TestDescendants(t *testing.T) {
	c := rossi(t)
	assertGenerations(t, "descendants of rossi-bartolomeo", c.Descendants("rossi-bartolomeo"),
		[][]Slug{
			{"rossi-francesco"},
			{"rossi-antonio"},
			{"rossi-annunziata", "rossi-domenico", "rossi-giovanni", "rossi-giuseppe"},
			{"rossi-anna", "rossi-infante-1866", "rossi-lorenzo", "rossi-paolo", "rossi-pietro"},
			{"rossi-elena", "rossi-giovanni-1902"},
			{"rossi-claudia"},
			{"rossi-marco"},
			{"ferri-rossi-sasha"},
		})
	assertGenerations(t, "descendants of rossi-marco", c.Descendants("rossi-marco"),
		[][]Slug{{"ferri-rossi-sasha"}})
	assertGenerations(t, "descendants of conti-teresa", c.Descendants("conti-teresa"),
		[][]Slug{{"conti-luigi"}})
	assertGenerations(t, "descendants of neri-caterina", c.Descendants("neri-caterina"),
		[][]Slug{{"rossi-domenico"}})
}

// A pedigree that loops back on itself reaches everybody in it once, at the nearer
// distance, which is how a cousin marriage is drawn as well.
func TestAncestorsStandOnceInTheNearerGeneration(t *testing.T) {
	c := NewCorpus()
	for _, slug := range []Slug{"child", "mother", "father", "uncle", "grandfather"} {
		c.People[slug] = &Person{Slug: slug}
	}
	c.Families["father--mother"] = &Family{Slug: "father--mother",
		Partners: PartnersOf("father", "mother"), Children: ChildrenOf("child")}
	c.Families["grandfather--unknown"] = &Family{Slug: "grandfather--unknown",
		Partners: PartnersOf("grandfather", Unknown), Children: ChildrenOf("mother", "uncle")}
	c.Families["uncle--unknown"] = &Family{Slug: "uncle--unknown",
		Partners: PartnersOf("uncle", Unknown), Children: ChildrenOf("father")}
	assertGenerations(t, "ancestors of a cousin marriage's child", c.Ancestors("child"),
		[][]Slug{{"father", "mother"}, {"grandfather", "uncle"}})
}

func TestKinship(t *testing.T) {
	c := rossi(t)
	for _, want := range []struct {
		what string
		a, b Slug
		Kinship
	}{
		// Giovanni-1902 is his grandson, not his great-nephew
		{"grandfather", "rossi-giovanni", "rossi-giovanni-1902",
			Kinship{"rossi-giovanni", 0, 2,
				[]Slug{"rossi-giovanni", "rossi-pietro", "rossi-giovanni-1902"}}},
		{"siblings", "rossi-giovanni", "rossi-giuseppe",
			Kinship{"rossi-antonio", 1, 1,
				[]Slug{"rossi-giovanni", "rossi-antonio", "rossi-giuseppe"}}},
		// Claudia and Elena share both of Pietro's household; the closest ancestors tie
		// at three steps and the tie breaks on slug order
		{"niece and aunt", "rossi-claudia", "rossi-elena",
			Kinship{"galli-ines", 2, 1,
				[]Slug{"rossi-claudia", "rossi-giovanni-1902", "galli-ines", "rossi-elena"}}},
		{"first cousins", "rossi-lorenzo", "rossi-pietro",
			Kinship{"rossi-antonio", 2, 2, []Slug{
				"rossi-lorenzo", "rossi-annunziata", "rossi-antonio", "rossi-giovanni", "rossi-pietro",
			}}},
		{"a person is their own closest relative", "rossi-giovanni", "rossi-giovanni",
			Kinship{"rossi-giovanni", 0, 0, []Slug{"rossi-giovanni"}}},
	} {
		got, ok := c.Kinship(want.a, want.b)
		if !ok || got.CommonAncestor != want.CommonAncestor || got.Up != want.Up ||
			got.Down != want.Down || !slices.Equal(got.Path, want.Path) {
			t.Errorf("%s: got %+v, %v, want %+v", want.what, got, ok, want.Kinship)
		}
	}

	if k, ok := c.Kinship("rossi-giovanni", "conti-luigi"); ok {
		t.Errorf("unrelated people have no common ancestor, got %+v", k)
	}
}

func TestKinshipSurvivesACycle(t *testing.T) {
	c := NewCorpus()
	c.People["a"] = &Person{Slug: "a"}
	c.People["b"] = &Person{Slug: "b"}
	c.Families["a--unknown"] = &Family{Slug: "a--unknown", Partners: PartnersOf("a", Unknown), Children: ChildrenOf("b")}
	c.Families["b--unknown"] = &Family{Slug: "b--unknown", Partners: PartnersOf("b", Unknown), Children: ChildrenOf("a")}
	assertGenerations(t, "ancestors of a", c.Ancestors("a"), [][]Slug{{"b"}})
	if k, ok := c.Kinship("a", "b"); !ok || !slices.Equal(k.Path, []Slug{"a", "b"}) {
		t.Errorf("got %+v, %v", k, ok)
	}
}

// What the first of two people is to the second. The sex that matters is the first
// one's, and an unrecorded sex is said without guessing.
func TestKinshipName(t *testing.T) {
	for _, c := range []struct {
		up, down int
		sex      Sex
		want     string
	}{
		{0, 0, Male, "the same person"},
		{0, 1, Female, "mother"},
		{0, 1, SexUnknown, "parent"},
		{0, 2, Male, "grandfather"},
		{0, 4, Female, "great-great-grandmother"},
		{1, 0, Male, "son"},
		{3, 0, SexUnknown, "great-grandchild"},
		{1, 1, Female, "sister"},
		{1, 1, SexUnknown, "sibling"},
		{1, 2, Female, "aunt"},
		{1, 3, Male, "great-uncle"},
		{1, 2, SexUnknown, "aunt or uncle"},
		{1, 3, SexUnknown, "great-aunt or great-uncle"},
		{2, 1, Male, "nephew"},
		{4, 1, Female, "great-great-niece"},
		{2, 2, Male, "first cousin"},
		{2, 3, Female, "first cousin once removed"},
		{3, 2, Female, "first cousin once removed"},
		{3, 3, SexUnknown, "second cousin"},
		{2, 5, Male, "first cousin three times removed"},
		{13, 12, Male, "11th cousin once removed"},
		{23, 23, Male, "22nd cousin"},
	} {
		if got := (Kinship{Up: c.up, Down: c.down}).Name(c.sex); got != c.want {
			t.Errorf("up %d, down %d, %v: %q, want %q", c.up, c.down, c.sex, got, c.want)
		}
	}
}

func TestGeneration(t *testing.T) {
	for _, c := range []struct {
		n      int
		upward bool
		want   string
	}{
		{1, true, "parents"},
		{2, true, "grandparents"},
		{4, true, "great-great-grandparents"},
		{1, false, "children"},
		{3, false, "great-grandchildren"},
	} {
		if got := Generation(c.n, c.upward); got != c.want {
			t.Errorf("Generation(%d, %v) = %q, want %q", c.n, c.upward, got, c.want)
		}
	}
}

func TestTaken(t *testing.T) {
	c := rossi(t)
	if !c.Taken("rossi-giovanni") || !c.Taken("bianchi-maria--rossi-giovanni") {
		t.Error("existing entities should read as taken")
	}
	if c.Taken("rossi-giovanni-1834") {
		t.Error("a free slug should not read as taken")
	}
}

func TestCloneIsolatesEdits(t *testing.T) {
	c := rossi(t)
	c.People["rossi-giovanni"].Events = []Event{{Type: "birth", Place: "borgo-san-lorenzo"}}
	c.Sources["parish-san-lorenzo-baptisms"] = &Source{
		Slug:         "parish-san-lorenzo-baptisms",
		Covers:       []Coverage{{Events: []string{"baptism"}, Place: "firenze"}},
		Repositories: []Holding{{Repository: "archivio-diocesano", CallNumber: "ADF/SL/B/4"}},
		Citations: map[string]Citation{"1834-042": {Text: "Giovanni Battista", Images: []Media{
			{File: "scans/1834-042.jpg", Region: &Region{Left: 0.1, Top: 0.2, Right: 0.3, Bottom: 0.4},
				Depicts: []Depicted{{Person: "rossi-giovanni"}}},
		}, With: []Named{{Person: "rossi-giovanni", Role: "child", Name: PersonName{Surname: SurnameOf("Rossi")}}},
			Links: LinksOf("https://www.example.org/1834-042")}},
	}
	c.Repositories["archivio-diocesano"] = &Repository{
		Slug:      "archivio-diocesano",
		Addresses: []Address{{City: "Firenze", Links: LinksOf("tel:055 213456")}},
		Links:     LinksOf("https://www.example.org/archivio"),
		IDs:       IDs{"wikidata": IdentifiersOf("Q1")},
	}
	c.People["rossi-giovanni"].Portrait = Image{File: "photos/group.jpg", Region: &Region{Left: 0.1, Top: 0.2, Right: 0.3, Bottom: 0.4}}
	c.People["rossi-giovanni"].SameAs = []SameAs{{ID: "rossi-giovanni-1834",
		Sources: []CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}}}
	c.People["rossi-giovanni"].Tags = []string{"To do"}
	c.Notes["vicchio-1861"] = &Note{Slug: "vicchio-1861", Tags: []string{"Research log"},
		Events: []Event{{Type: "marriage", With: []Named{{Person: "rossi-giovanni", Role: "witness"}}}}}
	c.Families["bianchi-maria--rossi-giovanni"].Media = []Media{{File: "photos/wedding.jpg", Depicts: []Depicted{
		{Person: "rossi-giovanni", Region: &Region{Left: 0.1, Top: 0.2, Right: 0.3, Bottom: 0.4}},
	}}}

	clone := c.Clone()
	clone.People["rossi-giovanni"].Portrait.Region.Left = 0.5
	wedding := clone.Families["bianchi-maria--rossi-giovanni"].Media
	wedding[0].Depicts[0].Region.Left = 0.5
	wedding[0].Depicts[0].Person = "changed"
	wedding[0].File = "changed"
	clone.People["rossi-giovanni"].Events[0].Place = "firenze"
	clone.Families["rossi-antonio--verdi-lucia"].Children[0].Relation.Word = Adoptive
	register := clone.Sources["parish-san-lorenzo-baptisms"]
	image := register.Citations["1834-042"].Images[0]
	image.Region.Left = 0.5
	image.Depicts[0].Person = "changed"
	register.Citations["1834-042"].Images[0].File = "changed"
	register.Citations["1834-042"].Links[0].URL = "changed"
	register.Citations["1834-042"].With[0].Person = "changed"
	register.Citations["1834-042"].With[0].Name.Surname[0].Name = "changed"
	register.Covers[0].Events[0] = "changed"
	register.Repositories[0].CallNumber = "changed"
	clone.Sources["parish-san-lorenzo-baptisms"].Citations["1834-042"] = Citation{Text: "changed"}
	archive := clone.Repositories["archivio-diocesano"]
	archive.IDs["wikidata"][0].ID = "changed"
	archive.Addresses[0].Links[0].URL = "changed"
	archive.Addresses[0].City = "changed"
	archive.Links[0].URL = "changed"
	delete(clone.People, "rossi-anna")
	clone.People["rossi-giovanni"].SameAs[0].ID = "changed"
	clone.People["rossi-giovanni"].SameAs[0].Sources[0].Anchor = "changed"
	clone.People["rossi-giovanni"].Tags[0] = "changed"
	note := clone.Notes["vicchio-1861"]
	note.Events[0].With[0].Person = "changed"
	note.Tags[0] = "changed"

	if got := c.People["rossi-giovanni"].Events[0].Place; got != "borgo-san-lorenzo" {
		t.Errorf("an edit to the clone reached the original event: %q", got)
	}
	if got := c.Families["rossi-antonio--verdi-lucia"].Children[0].Relation.Word; got != "" {
		t.Errorf("an edit to the clone reached the original children: %v", got)
	}
	if got := c.Sources["parish-san-lorenzo-baptisms"].Citations["1834-042"].Text; got != "Giovanni Battista" {
		t.Errorf("an edit to the clone reached the original citations: %q", got)
	}
	if got := c.Sources["parish-san-lorenzo-baptisms"].Citations["1834-042"].Images[0]; got.File != "scans/1834-042.jpg" ||
		got.Region.Left != 0.1 || got.Depicts[0].Person != "rossi-giovanni" {
		t.Errorf("an edit to the clone reached the original image: %+v %+v", got, *got.Region)
	}
	if got := c.Sources["parish-san-lorenzo-baptisms"].Citations["1834-042"].With[0]; got.Person != "rossi-giovanni" ||
		got.Name.Surname[0].Name != "Rossi" {
		t.Errorf("an edit to the clone reached who the original entry names: %+v", got)
	}
	if got := c.Sources["parish-san-lorenzo-baptisms"]; got.Covers[0].Events[0] != "baptism" ||
		got.Repositories[0].CallNumber != "ADF/SL/B/4" {
		t.Errorf("an edit to the clone reached what the original covers or where it is kept: %+v %+v",
			got.Covers, got.Repositories)
	}
	if got := c.Sources["parish-san-lorenzo-baptisms"].Citations["1834-042"].Links[0].URL; got != "https://www.example.org/1834-042" {
		t.Errorf("an edit to the clone reached the original citation's links: %q", got)
	}
	if got := c.Repositories["archivio-diocesano"]; got.IDs["wikidata"][0].ID != "Q1" ||
		got.Addresses[0].City != "Firenze" || got.Addresses[0].Links[0].URL != "tel:055 213456" ||
		got.Links[0].URL != "https://www.example.org/archivio" {
		t.Errorf("an edit to the clone reached the original repository: %+v", got)
	}
	if _, ok := c.People["rossi-anna"]; !ok {
		t.Error("a deletion in the clone reached the original")
	}
	if got := c.People["rossi-giovanni"]; got.SameAs[0].ID != "rossi-giovanni-1834" ||
		got.SameAs[0].Sources[0].Anchor != "1834-042" || got.Tags[0] != "To do" {
		t.Errorf("an edit to the clone reached the original's claim or tags: %+v %v", got.SameAs, got.Tags)
	}
	if got := c.Notes["vicchio-1861"]; got.Events[0].With[0].Person != "rossi-giovanni" || got.Tags[0] != "Research log" {
		t.Errorf("an edit to the clone reached the original note: %+v", got)
	}
	if got := c.People["rossi-giovanni"].Portrait.Region.Left; got != 0.1 {
		t.Errorf("an edit to the clone reached the original portrait: %v", got)
	}
	if got := c.Families["bianchi-maria--rossi-giovanni"].Media[0]; got.File != "photos/wedding.jpg" ||
		got.Depicts[0].Person != "rossi-giovanni" || got.Depicts[0].Region.Left != 0.1 {
		t.Errorf("an edit to the clone reached the original media: %+v %+v", got, *got.Depicts[0].Region)
	}
}

// Every citation a record makes is its own in a clone, wherever in the record it is
// written, and so is every identifier.
func TestCloneIsolatesWhatIsCited(t *testing.T) {
	c := rossi(t)
	cited := func() []CitationRef {
		return []CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}
	}
	giovanni := c.People["rossi-giovanni"]
	giovanni.Name.Sources = cited()
	giovanni.AlsoKnownAs = []PersonName{{Text: "Gio. Rossi"}, {Surname: SurnameOf("Rosi"), Sources: cited()}}
	giovanni.Sex = StatedSex{Value: Male, Sources: cited()}
	giovanni.Events = []Event{{
		Type:    "baptism",
		Address: Address{Lines: "Pieve di San Lorenzo", Sources: cited()},
		With:    []Named{{Name: PersonName{Given: "Maria", Surname: SurnameOf("Neri"), Sources: cited()}, Role: "godmother", Sources: cited()}},
		Sources: cited(),
	}}
	giovanni.Sources = cited()
	giovanni.IDs = IDs{"familysearch": {{ID: "XXXX-YYY"}, {ID: "XXXX-XXX", Sources: cited()}}}
	giovanni.Media = []Media{{File: "photos/1890.jpg", Sources: cited()}}
	family := c.Families[c.FamilySlugs()[0]]
	family.Sources = cited()
	family.IDs = IDs{"wikidata": IdentifiersOf("Q1")}
	family.Media = []Media{{File: "photos/wedding.jpg", Sources: cited()}}
	family.Children[0].Sources = cited()
	family.Children = append(family.Children, Child{Name: PersonName{Given: "Teresa", Sources: cited()}})
	register := &Source{Slug: "parish-san-lorenzo-baptisms", Citations: map[string]Citation{"1834-042": {
		With: []Named{{Name: PersonName{Given: "Antonio", Sources: cited()}, Role: "father",
			Attributes: []Attribute{{Type: "occupation", Sources: cited()}}}},
		Images: []Media{{File: "scans/1834-042.jpg", Sources: cited(),
			Attributes: []Attribute{{Type: "hand", Sources: cited()}}}},
	}}}
	c.Sources[register.Slug] = register

	clone := c.Clone()
	for _, entity := range []any{clone.People["rossi-giovanni"], clone.Families[family.Slug], clone.Sources[register.Slug]} {
		Cited(entity, func(ref *CitationRef) { ref.Anchor = "changed" })
	}
	clone.People["rossi-giovanni"].IDs["familysearch"][0].ID = "changed"
	clone.Families[family.Slug].IDs["wikidata"][0].ID = "changed"

	count := 0
	for _, entity := range []any{giovanni, family, register} {
		Cited(entity, func(ref *CitationRef) {
			count++
			if ref.Anchor != "1834-042" {
				t.Errorf("an edit to a citation in the clone reached the original: %v", ref)
			}
		})
	}
	if count != 18 {
		t.Errorf("walked %d citations of the original, want 18", count)
	}
	if giovanni.IDs["familysearch"][0].ID != "XXXX-YYY" || family.IDs["wikidata"][0].ID != "Q1" {
		t.Error("an edit to an identifier in the clone reached the original")
	}
}

// The page of somebody named in an event derives the link back, as a parent's page
// derives the children: from every record, in an order that does not depend on how the
// maps were walked.
func TestNamedInFindsEveryEventNamingAPerson(t *testing.T) {
	c := rossi(t)
	c.People["rossi-giovanni"].Events = []Event{
		{Type: "birth"},
		{Type: "baptism", With: []Named{
			{Person: "verdi-lucia", Role: "witness"},
			{Name: PersonName{Text: "Maria Neri"}, Role: "godmother"},
			{Person: "rossi-francesco", Role: "godfather"},
		}},
	}
	c.People["rossi-anna"].Events = []Event{{Type: "baptism", With: []Named{{Person: "rossi-francesco", Role: "godfather"}}}}
	family := c.Families["bianchi-maria--rossi-giovanni"]
	family.Events = []Event{{Type: "marriage", With: []Named{{Person: "rossi-francesco", Role: "witness"}}}}
	c.Notes["vicchio-1861"] = &Note{Slug: "vicchio-1861", Events: []Event{
		{Type: "marriage", With: []Named{{Name: PersonName{Text: "Pietro Landi"}, Role: "witness"}, {Person: "rossi-francesco", Role: "witness"}}},
	}}

	want := []Naming{
		{Record: EntityRef{Kind: KindFamily, Slug: family.Slug}, Event: 0, Entry: 0, Role: "witness"},
		{Record: EntityRef{Kind: KindPerson, Slug: "rossi-anna"}, Event: 0, Entry: 0, Role: "godfather"},
		{Record: EntityRef{Kind: KindPerson, Slug: "rossi-giovanni"}, Event: 1, Entry: 2, Role: "godfather"},
		{Record: EntityRef{Kind: KindNote, Slug: "vicchio-1861"}, Event: 0, Entry: 1, Role: "witness"},
	}
	for range 20 {
		if got := c.NamedIn("rossi-francesco"); !slices.Equal(got, want) {
			t.Fatalf("NamedIn = %+v, want %+v", got, want)
		}
	}
	if got := c.NamedIn("rossi-anna"); got != nil {
		t.Errorf("somebody nobody names is named in %+v", got)
	}
}

// A picture is on the page of everybody it depicts, as an event is on the page of
// everybody it names: derived from every record that keeps one, in an order that does
// not depend on how the maps were walked.
func TestDepictingFindsEveryPictureOfAPerson(t *testing.T) {
	c := rossi(t)
	region := &Region{Left: 0.4, Top: 0.1, Right: 0.6, Bottom: 0.5}
	c.Families["bianchi-maria--rossi-giovanni"].Media = []Media{{File: "photos/wedding.jpg", Depicts: []Depicted{
		{Person: "rossi-giovanni"}, {Person: "bianchi-maria", Region: region},
	}}}
	c.People["rossi-giovanni"].Media = []Media{
		{File: "photos/1890.jpg", Depicts: []Depicted{{Person: "rossi-giovanni"}}},
		{File: "photos/with-maria.jpg", Depicts: []Depicted{{Person: "bianchi-maria"}}},
	}
	c.Sources["album"] = &Source{Slug: "album", Media: []Media{{File: "p3.jpg", Depicts: []Depicted{{Person: "bianchi-maria"}}}}}
	family := c.Families["bianchi-maria--rossi-giovanni"]
	at := fmt.Sprintf("events[%d].media[0]", len(family.Events))
	family.Events = append(family.Events, Event{Type: "marriage", Media: []Media{
		{File: "photos/church.jpg", Depicts: []Depicted{{Person: "bianchi-maria"}}},
	}})

	want := []Depiction{
		{Record: EntityRef{Kind: KindSource, Slug: "album"}, Media: "media[0]", File: "p3.jpg"},
		{Record: EntityRef{Kind: KindFamily, Slug: "bianchi-maria--rossi-giovanni"}, Media: at, File: "photos/church.jpg"},
		{Record: EntityRef{Kind: KindFamily, Slug: "bianchi-maria--rossi-giovanni"}, Media: "media[0]", Depicted: 1,
			File: "photos/wedding.jpg", Region: region},
		{Record: EntityRef{Kind: KindPerson, Slug: "rossi-giovanni"}, Media: "media[1]", File: "photos/with-maria.jpg"},
	}
	for range 20 {
		if got := c.Depicting("bianchi-maria"); !slices.Equal(got, want) {
			t.Fatalf("Depicting = %+v, want %+v", got, want)
		}
	}
	if got := c.Depicting("rossi-anna"); got != nil {
		t.Errorf("somebody in no picture is depicted in %+v", got)
	}
}

func TestSortedKeysAreStable(t *testing.T) {
	c := rossi(t)
	first := c.PeopleSlugs()
	for range 20 {
		if got := c.PeopleSlugs(); !slices.Equal(got, first) {
			t.Fatalf("map iteration leaked into the result: %q then %q", first, got)
		}
	}
	if !slices.IsSorted(first) {
		t.Errorf("not sorted: %q", first)
	}
	if len(first) != 17+len(rossiOrphans) {
		t.Errorf("got %d people", len(first))
	}
}
