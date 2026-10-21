package gedcom

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

func convert(t *testing.T, text string) (*model.Corpus, []string) {
	t.Helper()
	corpus, _, notes := Convert(records(t, text), nil, nil)
	return corpus, notes
}

// The mapping the whole package exists for, on the smallest file that shows all of it.
func TestAnIndividualBecomesAPersonAndAFAMBecomesAFamily(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME Giovanni Battista /Rossi/
1 SEX M
1 BIRT
2 DATE 12 MAR 1834
2 PLAC Borgo San Lorenzo, Firenze, Toscana
1 NOTE He appears as "Gio. Rossi" in the 1861 census.
0 @I2@ INDI
1 NAME Maria /Bianchi/
1 SEX F
0 @I3@ INDI
1 NAME Anna /Rossi/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 MARR
2 DATE ABT 1858
`)

	giovanni := corpus.People["rossi-giovanni-battista"]
	if giovanni == nil {
		t.Fatalf("people are %v", corpus.PeopleSlugs())
	}
	if giovanni.Name.Given != "Giovanni Battista" || giovanni.Name.PrimarySurname().Name != "Rossi" {
		t.Errorf("name = %+v", giovanni.Name)
	}
	if giovanni.Sex.Value != model.Male {
		t.Errorf("sex = %v", giovanni.Sex)
	}
	if !strings.Contains(giovanni.Body, "1861 census") {
		t.Errorf("the note did not become the body: %q", giovanni.Body)
	}
	if len(giovanni.Events) != 1 || giovanni.Events[0].Type != "birth" ||
		giovanni.Events[0].Date.String() != "12 MAR 1834" {
		t.Errorf("events = %+v", giovanni.Events)
	}

	family := corpus.Families["bianchi-maria--rossi-giovanni-battista"]
	if family == nil {
		t.Fatalf("families are %v", corpus.FamilySlugs())
	}
	if len(family.Events) != 1 || family.Events[0].Date.String() != "ABT 1858" {
		t.Errorf("family events = %+v", family.Events)
	}

	if !reflect.DeepEqual(family.Children, model.ChildrenOf("rossi-anna")) {
		t.Errorf("children = %+v", family.Children)
	}
	if len(family.Partners) != 2 {
		t.Errorf("partners = %v", family.Partners)
	}
}

// A FAM with one partner is a family all the same, with `unknown` beside the partner it
// gives, and its children in the order the file lists them.
func TestAFamilyWithOneParentHasAnUnknownPartner(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME Giulia /Moretti/
1 SEX F
0 @I2@ INDI
1 NAME Pietro /Moretti/
0 @I3@ INDI
1 NAME Anna /Moretti/
0 @F1@ FAM
1 WIFE @I1@
1 CHIL @I3@
1 CHIL @I2@
1 NOTE Two children of Giulia's.
`)
	family := corpus.Families["moretti-giulia--unknown"]
	if family == nil {
		t.Fatalf("families are %v", corpus.FamilySlugs())
	}
	if !reflect.DeepEqual(family.Partners, model.PartnersOf("moretti-giulia", model.Unknown)) {
		t.Errorf("partners = %+v", family.Partners)
	}
	if !reflect.DeepEqual(family.Children, model.ChildrenOf("moretti-anna", "moretti-pietro")) {
		t.Errorf("children = %+v", family.Children)
	}
	if family.Body != "Two children of Giulia's." {
		t.Errorf("body = %q", family.Body)
	}
	if got := corpus.ParentsOf("moretti-pietro"); len(got) != 2 || got[0].Slug != "moretti-giulia" || got[1].Slug != model.Unknown {
		t.Errorf("parents = %+v", got)
	}
}

// GEDCOM restates the whole place at every event that happened there. The model states
// it once and links the levels, which is the difference between a hierarchy as data and
// a hierarchy retyped four thousand times.
func TestRepeatedPlacesBecomeOneHierarchy(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 BIRT
2 PLAC Borgo San Lorenzo, Firenze, Toscana
1 DEAT
2 PLAC Firenze, Toscana
3 MAP
4 LATI N43.7696
4 LONG E11.2558
0 @I2@ INDI
1 NAME B /Two/
1 BIRT
2 PLAC Borgo San Lorenzo, Firenze, Toscana
`)
	if got := corpus.PlaceSlugs(); !slices.Equal(got,
		[]model.Slug{"borgo-san-lorenzo", "firenze", "toscana"}) {
		t.Fatalf("places = %v", got)
	}
	if got := corpus.Places["borgo-san-lorenzo"].Within; len(got) != 1 || got[0].Place != "firenze" {
		t.Errorf("Borgo San Lorenzo is within %v", got)
	}
	if got := corpus.Places["firenze"].Within; len(got) != 1 || got[0].Place != "toscana" {
		t.Errorf("Firenze is within %v", got)
	}
	if got := corpus.Places["toscana"].Within; got != nil {
		t.Errorf("Toscana is within %v", got)
	}
	if got := corpus.Places["firenze"].Coordinates; got == nil || got.Latitude != 43.7696 {
		t.Errorf("coordinates = %+v", got)
	}
}

// Two places of the same name in different regions are two places. Collapsing them by
// name alone would invent a migration nobody made.
func TestTwoPlacesOfTheSameNameStayApart(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 BIRT
2 PLAC San Giovanni, Toscana
1 DEAT
2 PLAC San Giovanni, Sicilia
`)
	if len(corpus.Places) != 4 {
		t.Fatalf("places = %v, want two San Giovannis and two regions", corpus.PlaceSlugs())
	}
	first, second := corpus.Places["san-giovanni"], corpus.Places["san-giovanni-sicilia"]
	if first == nil || second == nil {
		t.Fatalf("places = %v", corpus.PlaceSlugs())
	}
	if first.Within[0].Place != "toscana" || second.Within[0].Place != "sicilia" {
		t.Errorf("%v and %v", first.Within, second.Within)
	}
}

// GEDCOM copies the transcription to every event that cites it. The model states it once
// in the source and points at it, so the same page cited twice has to become one anchor —
// and a different page the same slug would be silently merged evidence.
func TestTheSamePageCitedTwiceBecomesOneAnchor(t *testing.T) {
	corpus, _ := convert(t, `0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME A /One/
1 BIRT
2 SOUR @S1@
3 PAGE 1834-042
3 DATA
4 TEXT baptised 14 March 1834
0 @I2@ INDI
1 NAME B /Two/
1 BIRT
2 SOUR @S1@
3 PAGE 1834-042
3 DATA
4 TEXT baptised 14 March 1834
0 @I3@ INDI
1 NAME C /Three/
1 BIRT
2 SOUR @S1@
3 PAGE 1836-011
`)
	source := corpus.Sources["register"]
	if source == nil {
		t.Fatalf("sources = %v", corpus.SourceSlugs())
	}
	if len(source.Citations) != 2 {
		t.Errorf("citations = %v, want one per page", source.Citations)
	}
	if got := source.Citations["1834-042"].Text; got != "baptised 14 March 1834" {
		t.Errorf("transcription = %q", got)
	}
	for _, slug := range []model.Slug{"one-a", "two-b"} {
		person := corpus.People[slug]
		if person == nil {
			t.Fatalf("people = %v", corpus.PeopleSlugs())
		}
		if got := person.Events[0].Sources; len(got) != 1 || got[0].String() != "register#1834-042" {
			t.Errorf("%s cites %v", slug, got)
		}
	}
}

// Slugs are derived from names, and two people are routinely given the same one. What is
// offered has to be worth reading in three years, so a birth year comes before a bare
// number — and the numeric suffix stays available for when nothing else is known.
func TestACollisionTakesADiscriminatorWorthReading(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 DATE 1834
0 @I2@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 DATE 1889
0 @I3@ INDI
1 NAME Giovanni /Rossi/
`)
	want := []model.Slug{"rossi-giovanni", "rossi-giovanni-1889", "rossi-giovanni-2"}
	if got := corpus.PeopleSlugs(); !slices.Equal(got, want) {
		t.Errorf("slugs = %v, want %v", got, want)
	}
}

// DESIGN.md asks origins to propose and the researcher to choose, because the person who
// knows the family names it better than any rule can.
func TestAChooserSettlesTheCollision(t *testing.T) {
	var asked []string
	choose := func(what string, proposals []model.Slug) model.Slug {
		asked = append(asked, what)
		return "rossi-giovanni-il-vecchio"
	}
	corpus, _, _ := Convert(records(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
0 @I2@ INDI
1 NAME Giovanni /Rossi/
`), choose, nil)

	if corpus.People["rossi-giovanni-il-vecchio"] == nil {
		t.Errorf("the chosen slug was not used: %v", corpus.PeopleSlugs())
	}
	if !slices.Equal(asked, []string{"Giovanni Rossi"}) {
		t.Errorf("asked about %v", asked)
	}
}

// GEDCOM states a relation in three places and no exporter uses all of them. An adoptive
// parent matters because the biological age rules in check must not apply to them.
func TestAdoptionIsReadFromWhicheverTagCarriesIt(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 SEX M
0 @I2@ INDI
1 NAME B /Two/
1 SEX F
0 @I3@ INDI
1 NAME C /Three/
1 FAMC @F1@
2 PEDI adopted
0 @I4@ INDI
1 NAME D /Four/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 CHIL @I4@
2 _FREL Step
`)
	family := corpus.Families["one-a--two-b"]
	if family == nil {
		t.Fatalf("families are %v", corpus.FamilySlugs())
	}
	want := []model.Child{
		{ID: "three-c", Relation: model.Relationship{Word: model.Adoptive}},
		// _FREL names the father's relation alone
		{ID: "four-d", Relation: model.Relationship{Partners: []model.PartnerRelation{
			{Partner: "one-a", Relation: model.Step}, {Partner: "two-b", Relation: model.Birth}}}},
	}
	if !reflect.DeepEqual(family.Children, want) {
		t.Errorf("children = %+v", family.Children)
	}
}

// Two FAM records naming the same couple are two families: a couple who married twice
// and one marriage recorded twice look the same from the names, and which it is is the
// researcher's to say. The second is told apart by the year of its first dated event,
// and one with no date by the surname of its first child.
func TestTheSameCoupleTwiceIsTwoFamilies(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME A /One/
0 @I2@ INDI
1 NAME B /Two/
0 @I3@ INDI
1 NAME C /Landi/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 1858
0 @F2@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 DIV
2 DATE 1867
0 @F3@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
`)
	for slug, events := range map[model.Slug]int{"one-a--two-b": 1, "one-a--two-b-1867": 1, "one-a--two-b-landi": 0} {
		family := corpus.Families[slug]
		if family == nil {
			t.Fatalf("families = %v", corpus.FamilySlugs())
		}
		if len(family.Events) != events {
			t.Errorf("%s holds %+v", slug, family.Events)
		}
	}
	if len(corpus.Families) != 3 || reported(notes, "same couple") {
		t.Errorf("families = %v, notes = %v", corpus.FamilySlugs(), notes)
	}
}

// An import that silently discards a third of a file looks exactly like one that worked,
// so what nothing reads comes in as the export wrote it, and is counted.
func TestWhatCouldNotBeCarriedIsKeptAsWritten(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 RFN 1234
1 _XMILITARY Served in the artillery
1 DEAT
2 DATE @#DROMAN@ ante diem III Kalendas Apriles DCCCX
0 @U1@ _CUSTOM
1 NOTE something this exporter invented
`)
	for _, want := range []string{
		"RFN was kept as written",
		"_XMILITARY was kept as written",
		"_CUSTOM was kept as written",
		"ante diem III Kalendas Apriles DCCCX (Roman)",
		"an event cites no source",
	} {
		if !reported(notes, want) {
			t.Errorf("%q was not reported in %v", want, notes)
		}
	}
	want := "```gedcom\n1 RFN 1234\n1 _XMILITARY Served in the artillery\n```"
	if body := corpus.People["one-a"].Body; body != want {
		t.Errorf("the person's body is %q, want %q", body, want)
	}
	var custom *model.Note
	for _, note := range corpus.Notes {
		custom = note
	}
	if len(corpus.Notes) != 1 || custom.Title != "_CUSTOM @U1@" ||
		custom.Body != "```gedcom\n0 @U1@ _CUSTOM\n1 NOTE something this exporter invented\n```" {
		t.Errorf("notes = %+v", custom)
	}
}

// A tag under an event is kept in the event's note, with everything under it.
func TestATagUnderAnEventIsKeptInItsNote(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 BIRT
2 DATE 1834
2 _WEATHER rain
3 _WIND north
`)
	want := "```gedcom\n2 _WEATHER rain\n3 _WIND north\n```"
	if got := corpus.People["one-a"].Events[0].Note; got != want {
		t.Errorf("the birth's note is %q, want %q", got, want)
	}
	if !slices.Contains(notes, "_WEATHER was kept as written") || reported(notes, "_WIND") {
		t.Errorf("notes = %q", notes)
	}
}

// An import is recorded as one commit, which says what program wrote the export it read
// and how.
func TestTheHeaderSaysWhatExportWasRead(t *testing.T) {
	head := records(t, `0 HEAD
1 SOUR GRAMPS
2 NAME Gramps
2 VERS 5.2.3
1 DATE 5 OCT 2026
2 TIME 12:00:00
1 GEDC
2 VERS 5.5.1
2 FORM LINEAGE-LINKED
1 CHAR UTF-8
1 FILE tree.ged
1 SCHMA
2 TAG _SKYPEID http://xmlns.com/foaf/0.1/skypeID
0 TRLR
`)
	want := "Read from family.ged, which Gramps 5.2.3 exported on 5 OCT 2026 12:00:00 as GEDCOM 5.5.1 in UTF-8. " +
		"The export calls itself tree.ged.\n\nExtension tags:\n_SKYPEID http://xmlns.com/foaf/0.1/skypeID"
	if got := Describe(head, "family.ged"); got != want {
		t.Errorf("described as %q, want %q", got, want)
	}
	if got := Describe(records(t, "0 TRLR\n"), ""); got != "Read from an export." {
		t.Errorf("an export with no header was described as %q", got)
	}
}

func reported(notes []string, want string) bool {
	return slices.ContainsFunc(notes, func(note string) bool { return strings.Contains(note, want) })
}

// prose is every piece of prose in corpus, each under its record and its path.
func prose(corpus *model.Corpus) string {
	var b strings.Builder
	for _, slugs := range [][]model.Slug{corpus.PeopleSlugs(), corpus.FamilySlugs(), corpus.PlaceSlugs(),
		corpus.SourceSlugs(), corpus.RepositorySlugs(), corpus.NoteSlugs()} {
		for _, slug := range slugs {
			model.Prose(corpus.Entity(slug), func(path string, text *string) {
				if *text != "" {
					b.WriteString(string(slug) + " " + path + ":\n" + *text + "\n")
				}
			})
		}
	}
	return b.String()
}

// Everything downstream of an import is byte-compared, so the import itself has to be a
// pure function of the file. Go randomises map iteration precisely so that a conversion
// that walks one fails here rather than in someone's git history.
func TestConvertingTheSameFileTwiceGivesTheSameCorpus(t *testing.T) {
	const text = `0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 PLAC Borgo San Lorenzo, Firenze, Toscana
2 SOUR @S1@
3 PAGE 1834-042
0 @I2@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 PLAC Firenze, Toscana
0 @I3@ INDI
1 NAME Maria /Bianchi/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I3@
1 CHIL @I2@
`
	first, firstNotes := convert(t, text)
	for pass := 0; pass < 5; pass++ {
		again, againNotes := convert(t, text)
		if !slices.Equal(firstNotes, againNotes) {
			t.Fatalf("notes differ:\n%v\n%v", firstNotes, againNotes)
		}
		if a, b := emitAll(t, first), emitAll(t, again); a != b {
			t.Fatalf("conversion is not deterministic:\n%s\n---\n%s", a, b)
		}
	}
}

// emitAll writes the whole corpus through the canonical emitter, which is the comparison
// that matters: two corpora are the same when they write the same bytes.
func emitAll(t *testing.T, corpus *model.Corpus) string {
	t.Helper()
	var b strings.Builder
	write := func(slug model.Slug, entity any) {
		content, err := store.Emit(entity)
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		b.WriteString("# " + string(slug) + "\n")
		b.Write(content)
	}
	for _, slug := range corpus.PeopleSlugs() {
		write(slug, corpus.People[slug])
	}
	for _, slug := range corpus.FamilySlugs() {
		write(slug, corpus.Families[slug])
	}
	for _, slug := range corpus.PlaceSlugs() {
		write(slug, corpus.Places[slug])
	}
	for _, slug := range corpus.SourceSlugs() {
		write(slug, corpus.Sources[slug])
	}
	for _, slug := range corpus.RepositorySlugs() {
		write(slug, corpus.Repositories[slug])
	}
	for _, slug := range corpus.NoteSlugs() {
		write(slug, corpus.Notes[slug])
	}
	return b.String()
}

// A GEDCOM names a photograph without containing it, and one whose file is not to hand
// is an entry all the same, at the path the export gave, and reported. A QUAY under a
// citation is how much it weighs, and one anywhere else is kept as written.
func TestWhatTheModelHasNoPlaceForIsStillReported(t *testing.T) {
	corpus, notes := convert(t, `0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME A /One/
1 BIRT
2 SOUR @S1@
3 QUAY 3
1 DEAT
2 QUAY 2
1 OBJE
2 FILE C:\photos\one.jpg
`)
	for _, want := range []string{
		"a picture the export names was not found beside it",
		"QUAY was kept as written",
	} {
		if !reported(notes, want) {
			t.Errorf("%q was not reported in %v", want, notes)
		}
	}
	person := corpus.People["one-a"]
	if want := []model.Media{{File: `C:\photos\one.jpg`}}; !reflect.DeepEqual(person.Media, want) || len(person.Files) != 0 {
		t.Errorf("a picture nobody has came in as %+v %v", person.Media, person.Files)
	}
	if got := person.Events[0].Sources; len(got) != 1 || got[0].String() != "register#one-a-birth ---3" ||
		person.Events[1].Note != "```gedcom\n2 QUAY 2\n```" {
		t.Errorf("events = %+v", person.Events)
	}
}

// A source written where it is cited rather than as a record of its own is common in
// older files. Forty events citing the same book become one source, because evidence
// that is buried in a note is evidence nothing can follow.
func TestASourceWrittenInPlaceBecomesASource(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 BIRT
2 SOUR Libro dei battezzati
3 PAGE 12
0 @I2@ INDI
1 NAME B /Two/
1 BIRT
2 SOUR Libro dei battezzati
3 PAGE 40
`)
	if got := corpus.SourceSlugs(); len(got) != 1 || got[0] != "libro-dei-battezzati" {
		t.Fatalf("sources = %v", got)
	}
	source := corpus.Sources["libro-dei-battezzati"]
	if source.Title != "Libro dei battezzati" || len(source.Citations) != 2 {
		t.Errorf("source = %+v", source)
	}
	if got := corpus.People["one-a"].Events[0].Sources; len(got) != 1 ||
		got[0].String() != "libro-dei-battezzati#12" {
		t.Errorf("citation = %v", got)
	}
}

// A person goes under one name and the others are still theirs. ALIA is the tag old
// exports use for the rest, except where a dialect spells it as a pointer, which claims
// two records are one person and is kept as written. A nickname is a part of the name it
// is given under.
func TestOtherNamesForTheSamePersonAreKept(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Kamehameha /Kalani/
2 NICK Paiea
2 ALIA William/Pitt/
1 NAME Kamehameha the Great
1 ALIA -/'Iolani/
1 ALIA Kamehameha the Great
0 @I2@ INDI
1 NAME Liholiho /Kalani/
1 ALIA @I1@
`)

	person := corpus.People["kalani-kamehameha"]
	if person == nil {
		t.Fatalf("people are %v", corpus.PeopleSlugs())
	}
	if person.Name.Nickname != "Paiea" {
		t.Errorf("the nickname is %q", person.Name.Nickname)
	}
	var known []string
	for _, name := range person.AlsoKnownAs {
		known = append(known, name.String())
	}
	want := []string{"William Pitt", "Kamehameha the Great", "'Iolani"}
	if !slices.Equal(known, want) {
		t.Errorf("also known as %q, want %q", known, want)
	}

	// An ALIA pointing at another record says the two are one person.
	if got := corpus.People["kalani-liholiho"].SameAs; len(notes) != 0 ||
		!reflect.DeepEqual(got, []model.SameAs{{ID: "kalani-kamehameha"}}) {
		t.Errorf("the pointer was not read: %v %+v", notes, got)
	}
}

// Exports are full of stand-ins for a name nobody recorded. A part with no letter and no
// digit in it is one of those, and the model already knows how to say a record gave no
// surname.
func TestAPlaceholderIsNotAName(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Elizabeth /_____/
0 @I2@ INDI
1 NAME ? /?/
0 @I3@ INDI
1 NAME ... /Stewart/
0 @I4@ INDI
1 NAME Robert /Neville/
2 NSFX Bishop of Salisbury & Durham
`)

	elizabeth := corpus.People["elizabeth"]
	if elizabeth == nil {
		t.Fatalf("people are %v", corpus.PeopleSlugs())
	}
	if elizabeth.Name.Given != "Elizabeth" || elizabeth.Name.PrimarySurname().Name != "" {
		t.Errorf("name = %+v", elizabeth.Name)
	}

	stewart := corpus.People["stewart"]
	if stewart == nil || stewart.Name.Given != "" || stewart.Name.PrimarySurname().Name != "Stewart" {
		t.Errorf("stewart = %+v", stewart)
	}

	// The test is over a whole part, not word by word: a word with no letter in it is
	// ordinary inside a part that has them.
	robert := corpus.People["neville-robert"]
	if robert == nil || robert.Name.Suffix != "Bishop of Salisbury & Durham" {
		t.Errorf("neville = %+v", robert)
	}

	// A record that gives nothing at all still has to be filed somewhere, and the
	// identifier it came under is the only thing left to file it by.
	if !reported(notes, "no name") {
		t.Errorf("a nameless individual went unreported: %v", notes)
	}
}

// The payload of a NAME is the whole name and the subtags name its parts, so a prefix or
// a suffix is taken off the name that already contains it rather than added to it. SPFX
// is kept apart from the surname, so that the person files among the others of it.
func TestANamePrefixIsPartOfTheNameAndNotAnAdditionToIt(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME Lt. Cmndr. Joseph /de Allen/ jr.
2 NPFX Lt. Cmndr.
2 GIVN Joseph
2 SPFX de
2 SURN Allen
2 NSFX jr.
0 @I2@ INDI
1 NAME Maria
2 SPFX van der
2 SURN Berg
0 @I3@ INDI
1 NAME Joseph /Ball/
2 NSFX Captain
`)

	joseph := corpus.People["allen-joseph"]
	if joseph == nil {
		t.Fatalf("people are %v", corpus.PeopleSlugs())
	}
	want := model.PersonName{Prefix: "Lt. Cmndr.", Given: "Joseph", SurnamePrefix: "de", Surname: model.SurnameOf("Allen"), Suffix: "jr."}
	if !reflect.DeepEqual(joseph.Name, want) || joseph.Name.String() != "Lt. Cmndr. Joseph de Allen jr." {
		t.Errorf("name = %+v", joseph.Name)
	}

	// With no slashes there is nothing but the subtags.
	maria := corpus.People["berg"]
	if maria == nil || maria.Name.SurnamePrefix != "van der" || maria.Name.PrimarySurname().Name != "Berg" {
		t.Errorf("maria = %+v", maria)
	}

	// A suffix the payload does not spell out is still part of the name. Most exports
	// write it in both places and this one writes it in only the second.
	joseph2 := corpus.People["ball-joseph"]
	if joseph2 == nil || joseph2.Name.Suffix != "Captain" || joseph2.Name.String() != "Joseph Ball Captain" {
		t.Errorf("ball = %+v", joseph2)
	}
}

// An index key is not a spelling. Exporters write SURN in capitals, and the payload says
// how the name was written; and a name with no slashes and nothing under it is kept as
// the record wrote it, since which of its words is the surname is exactly what the
// record did not say.
func TestAPayloadIsDividedAtItsSlashesOrKeptWhole(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
2 GIVN GIOVANNI
2 SURN ROSSI
0 @I2@ INDI
1 NAME Gio. Rossi
`)
	if got := corpus.People["rossi-giovanni"]; got == nil ||
		!reflect.DeepEqual(got.Name, model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}) {
		t.Errorf("rossi-giovanni = %+v", got)
	}
	if got := corpus.People["gio-rossi"]; got == nil || got.Name.Text != "Gio. Rossi" {
		t.Errorf("gio-rossi = %+v", got)
	}
	if !reported(notes, "no surname") {
		t.Errorf("a name filed under the whole of it went unreported: %v", notes)
	}
}

// Version 7.0 agrees with 5.5.1 on the line format and differs in a handful of places,
// each handled where it arises. What it moved is still found, and what it renamed is
// still read.
func TestTheSeventhVersionIsRead(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 DATE 27 MAR 2022
3 TIME 16:02
3 PHRASE Afternoon
1 DEAT
2 DATE
3 PHRASE 3 March
1 BURI
2 DATE 12 MAR 1834
3 PHRASE the twelfth of March
`)

	if reported(notes, "declares GEDCOM") {
		t.Errorf("a version origins reads was reported as one it does not: %v", notes)
	}

	giovanni := corpus.People["rossi-giovanni"]
	if giovanni == nil || len(giovanni.Events) != 3 {
		t.Fatalf("events are %+v", giovanni)
	}
	// A TIME belongs to the date above it, and the model carries one; a time and a
	// phrase on the same date are both kept, the phrase as its wording.
	if got := giovanni.Events[0].Date.String(); got != "27 MAR 2022 16:02 (Afternoon)" {
		t.Errorf("birth = %q", got)
	}
	// 7.0 moved the date phrase onto its own line; it is still where a yearless date
	// gets recovered from.
	if got := giovanni.Events[1].Date.String(); got != "3 MAR" {
		t.Errorf("death = %q", got)
	}
	// And where the words are not a date, they are the date's wording.
	if got := giovanni.Events[2].Date.String(); got != "12 MAR 1834 (the twelfth of March)" {
		t.Errorf("burial = %q", got)
	}
	if reported(notes, "could not be read") {
		t.Errorf("a phrase carried as wording was reported as lost: %v", notes)
	}
}

// The four places version 7.0 says something 5.5.1 said differently, and the one place
// it says something 5.5.1 could not.
func TestWhatTheSeventhVersionRenamed(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @N1@ SNOTE The parish register was rebound in 1908.
0 @I1@ INDI
1 NAME Giovanni /Rossi/
2 TRAN Ioannes /Rubeus/
3 LANG la
1 SNOTE @N1@
1 BAPM
2 DATE JULIAN 14 FEB 1799
1 DEAT
2 DATE 44 BCE
0 @F1@ FAM
1 HUSB @I1@
1 CHIL @VOID@
`)

	giovanni := corpus.People["rossi-giovanni"]
	if giovanni == nil {
		t.Fatalf("people are %v", corpus.PeopleSlugs())
	}

	// A shared note is pointed at by SNOTE in 7.0 and by NOTE in 5.5.1.
	if !strings.Contains(giovanni.Body, "rebound in 1908") {
		t.Errorf("body = %q", giovanni.Body)
	}
	// A TRAN under a NAME is the same name in another language.
	if len(giovanni.AlsoKnownAs) != 1 || giovanni.AlsoKnownAs[0].String() != "Ioannes Rubeus" {
		t.Errorf("also known as %+v", giovanni.AlsoKnownAs)
	}
	// 7.0 writes the calendar as the first word where 5.5.1 wrapped it in an escape,
	// and the corpus writes the word. Either way the date is read, not parked.
	if got := giovanni.Events[0].Date.String(); got != "JULIAN 14 FEB 1799" {
		t.Errorf("baptism date = %q", got)
	}
	if got := giovanni.Events[1].Date.String(); got != "44 BCE" {
		t.Errorf("death date = %q", got)
	}
	// @VOID@ points at no record on purpose: a child the family had whom nobody could
	// name.
	if family := corpus.Families["rossi-giovanni--unknown"]; family == nil ||
		!reflect.DeepEqual(family.Children, []model.Child{{ID: model.Unknown}}) {
		t.Errorf("families are %v, notes %v", corpus.FamilySlugs(), notes)
	}
}

// A version that is neither of the two is read as though it were, which mostly works and
// is worth saying out loud.
func TestAVersionOriginsDoesNotKnowIsSaidToBeOne(t *testing.T) {
	_, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 8.1
0 @I1@ INDI
1 NAME Giovanni /Rossi/
`)
	if !reported(notes, "declares GEDCOM 8.1") {
		t.Errorf("an unknown version went unreported: %v", notes)
	}
}

// The archive a source is kept in is a record of its own, written once however many
// sources it keeps, and the call number is on the source, since one source can sit in
// two archives under two numbers. An archive named in place is made one record the
// first time it is seen, as a source written in place is.
func TestWhereASourceIsKeptIsKept(t *testing.T) {
	corpus, notes := convert(t, `0 @R1@ REPO
1 NAME Archivio Diocesano di Firenze
1 ADDR Piazza San Giovanni 3
2 CONT 50129 Firenze
1 WWW https://www.archiviodiocesano.fi.it
1 PHON +39 055 000000
1 NOTE Open Tuesdays.
0 @R2@ REPO
1 NAME Archivio di Stato
2 CITY Firenze
0 @S1@ SOUR
1 TITL San Lorenzo baptisms
1 AUTH Parrocchia di San Lorenzo
1 PUBL manuscript
1 WWW https://example.org/sl
1 REPO @R1@
2 CALN ADF/SL/B/4
1 REPO @R2@
0 @S2@ SOUR
1 TITL San Lorenzo burials
1 REPO Biblioteca Laurenziana
0 @S3@ SOUR
1 TITL San Lorenzo marriages
1 REPO Biblioteca Laurenziana
`)
	if got := corpus.RepositorySlugs(); !slices.Equal(got,
		[]model.Slug{"archivio-di-stato", "archivio-diocesano-di-firenze", "biblioteca-laurenziana"}) {
		t.Fatalf("repositories are %v", got)
	}
	archive := corpus.Repositories["archivio-diocesano-di-firenze"]
	if archive.Name != "Archivio Diocesano di Firenze" ||
		!reflect.DeepEqual(archive.Addresses, []model.Address{{Lines: "Piazza San Giovanni 3\n50129 Firenze"}}) ||
		!slices.Equal(archive.Links, model.LinksOf("https://www.archiviodiocesano.fi.it", "tel:+39 055 000000")) ||
		archive.Body != "Open Tuesdays." {
		t.Errorf("repository = %+v", archive)
	}

	source := corpus.Sources["san-lorenzo-baptisms"]
	if source == nil {
		t.Fatalf("sources are %v", corpus.SourceSlugs())
	}
	if want := []model.Holding{{Repository: "archivio-diocesano-di-firenze", CallNumber: "ADF/SL/B/4"},
		{Repository: "archivio-di-stato"}}; !slices.Equal(source.Repositories, want) {
		t.Errorf("repositories = %+v", source.Repositories)
	}
	if source.Author != "Parrocchia di San Lorenzo" || source.Publication != "manuscript" ||
		!slices.Equal(source.Links, model.LinksOf("https://example.org/sl")) || source.Body != "" {
		t.Errorf("source = %+v", source)
	}
	for _, slug := range []model.Slug{"san-lorenzo-burials", "san-lorenzo-marriages"} {
		if got := corpus.Sources[slug].Repositories; len(got) != 1 || got[0].Repository != "biblioteca-laurenziana" {
			t.Errorf("%s is kept at %+v", slug, got)
		}
	}
	if reported(notes, "REPO was kept as written") {
		t.Errorf("a REPO was kept as written: %v", notes)
	}
}

// A name in a script a slug cannot spell is a name all the same, and an individual
// filed under their identifier for it is not reported as having none.
func TestAnIndividualWhoseNameNoSlugSpellsIsNotNameless(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Мария /Попова/
0 @I2@ INDI
1 SEX F
0 TRLR
`)
	if maria := corpus.People["i1"]; maria == nil || maria.Name.Given != "Мария" {
		t.Fatalf("Maria = %+v among %v", maria, corpus.PeopleSlugs())
	}
	for _, want := range []string{
		"an individual had no name a slug can spell and is filed under their identifier",
		"an individual had no name and is filed under their identifier",
	} {
		if !slices.Contains(notes, want) {
			t.Errorf("did not report %q in %q", want, notes)
		}
	}
}
