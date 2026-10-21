package gedcom

import (
	"image"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// What a name says about itself comes with it: the kind of name, whatever the version
// spells it as, and what states it, a name nobody divided as much as any other.
func TestANameSaysWhatKindItIsAndWhatStatesIt(t *testing.T) {
	corpus, notes := convert(t, `0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME Maria /Bianchi/
2 TYPE birth
2 SOUR @S1@
3 PAGE 1840-001
1 NAME Maria /Rossi/
2 TYPE MARRIED
1 NAME Suor Maria /Celeste/
2 TYPE OTHER
3 PHRASE religious name
1 NAME Mariuccia
2 TYPE aka
`)
	maria := corpus.People["bianchi-maria"]
	if maria == nil {
		t.Fatalf("people are %v", corpus.PeopleSlugs())
	}
	if maria.Name.Type != "birth" || len(maria.Name.Sources) != 1 ||
		maria.Name.Sources[0].String() != "register#1840-001" {
		t.Errorf("name = %+v", maria.Name)
	}
	var kinds []string
	for _, name := range maria.AlsoKnownAs {
		kinds = append(kinds, name.String()+": "+name.Type)
	}
	if want := []string{"Maria Rossi: married", "Suor Maria Celeste: religious name", "Mariuccia: aka"}; !slices.Equal(kinds, want) {
		t.Errorf("also known as %q, want %q", kinds, want)
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v", notes)
	}
}

// A record is cited as a whole where no one fact on it is, a family too, and a child's
// link to the family is where GEDCOM cites the parentage. A pedigree of OTHER makes
// somebody a parent without saying in what sense, and a sex of X is a record saying
// neither, both of which the model states.
func TestARecordItsParentageAndItsSexAreCarriedWithWhatStatesThem(t *testing.T) {
	corpus, notes := convert(t, `0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME A /One/
1 SEX M
1 SOUR @S1@
2 PAGE p. 1
0 @I2@ INDI
1 NAME B /Two/
1 SEX X
0 @I3@ INDI
1 NAME C /Three/
1 FAMC @F1@
2 PEDI OTHER
2 SOUR @S1@
3 PAGE p. 3
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 SOUR @S1@
2 PAGE p. 2
`)
	if got := corpus.People["one-a"].Sources; len(got) != 1 || got[0].String() != "register#p-1" {
		t.Errorf("the person cites %v", got)
	}
	if got := corpus.People["two-b"].Sex.Value; got != model.Neither {
		t.Errorf("sex = %v", got)
	}
	if got := corpus.Families["one-a--two-b"].Sources; len(got) != 1 || got[0].String() != "register#p-2" {
		t.Errorf("the family cites %v", got)
	}
	children := corpus.Families["one-a--two-b"].Children
	if len(children) != 1 {
		t.Fatalf("children = %+v", children)
	}
	if child := children[0]; child.Relation.Word != model.RelationUnknown || len(child.Sources) != 1 ||
		child.Sources[0].String() != "register#p-3" {
		t.Errorf("child = %+v", child)
	}
	for _, unwanted := range []string{"SOUR under", "pedigree"} {
		if reported(notes, unwanted) {
			t.Errorf("%q was reported in %v", unwanted, notes)
		}
	}
}

// An attribute is a fact held, with what is held on its line: the value of an event in
// the model, over the period its FROM and TO give. FACT is the attribute whose kind its
// TYPE names. Words on an event's line are its value too, and a Y says only that it
// happened.
func TestAnAttributeIsAFactHeldWithAValue(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 OCCU contadino
2 DATE FROM 1858 TO 1871
1 TITL Cavaliere
1 DSCR tall, with red hair
1 FACT 3
2 TYPE Number of horses
1 BIRT a stormy night
1 DEAT Y
0 @I2@ INDI
1 NAME B /Two/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 NCHI 4
`)
	var held []string
	for _, event := range corpus.People["one-a"].Events {
		held = append(held, event.Type+"="+event.Value+" "+event.Date.String()+" "+event.Note)
	}
	want := []string{
		"occupation=contadino FROM 1858 TO 1871 ",
		"title=Cavaliere  ",
		"description=tall, with red hair  ",
		"Number of horses=3  ",
		"birth=a stormy night  ",
		"death=  ",
	}
	if !slices.Equal(held, want) {
		t.Errorf("events are\n%q\nwant\n%q", held, want)
	}
	if got := corpus.Families["one-a--two-b"].Events; len(got) != 1 || got[0].Value != "4" {
		t.Errorf("family events = %+v", got)
	}
}

// The tags Gramps writes for events GEDCOM has none for, and Family Tree Maker's that
// Gramps reads, are events in Gramps' words, each with what its line says as its value; a
// family has its own, and any other extension tag is kept as written.
func TestTheExtensionTagsGrampsKnowsAreEvents(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME A /One/
1 _DEG
2 TYPE Business Management
2 DATE 1988
1 _MILT
2 DATE FROM 1915 TO 1918
1 _HEIG 1.80 m
1 _FUN Held at St Rafael's
2 DATE 3 MAR 1950
2 NOTE Hundreds attended.
1 _XYZ something
0 @I2@ INDI
1 NAME B /Two/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 _SEPR
2 DATE 1940
`)
	var events []string
	for _, event := range corpus.People["one-a"].Events {
		events = append(events, event.Type+"="+event.Value+" "+event.Date.String()+" "+event.Note)
	}
	want := []string{
		"degree= 1988 ", "military-service= FROM 1915 TO 1918 ", "height=1.80 m  ",
		"funeral=Held at St Rafael's 3 MAR 1950 Hundreds attended.",
	}
	if !slices.Equal(events, want) {
		t.Errorf("events are\n%q\nwant\n%q", events, want)
	}
	if got := corpus.Families["one-a--two-b"].Events; len(got) != 1 || got[0].Type != "separation" {
		t.Errorf("family events = %+v", got)
	}
	if !reported(notes, "_XYZ was kept as written") || reported(notes, "_DEG was kept") {
		t.Errorf("notes = %v", notes)
	}
}

// An event carries the age the record gives and its cause, and the people it names: in
// 7.0 an ASSO under the event, by their record or by name alone, and under a family's
// event the partners' ages. An age the grammar cannot hold is kept in the note.
func TestAnEventCarriesAgesCausesAndThePeopleItNames(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BAPM
2 ASSO @I3@
3 ROLE GODP
4 PHRASE godfather
3 SOUR @S1@
4 PAGE 12
2 ASSO @VOID@
3 PHRASE Maria Pieri
3 ROLE GODP
1 DEAT
2 AGE 66y
2 CAUS cholera
0 @I2@ INDI
1 NAME Maria /Bianchi/
1 DEAT
2 AGE CHILD
0 @I3@ INDI
1 NAME Antonio /Rossi/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 HUSB
3 AGE 24y
2 WIFE
3 AGE 18
2 ASSO @I3@
3 ROLE WITN
`)
	giovanni := corpus.People["rossi-giovanni"]
	baptism := giovanni.Events[0]
	want := []model.Named{
		{Person: "rossi-antonio", Role: "godfather",
			Sources: []model.CitationRef{{Source: "register", Anchor: "12"}}},
		{Name: model.PersonName{Text: "Maria Pieri"}, Role: "godparent"},
	}
	if !reflect.DeepEqual(baptism.With, want) {
		t.Errorf("the baptism names %+v", baptism.With)
	}
	if death := giovanni.Events[1]; death.Age.Value != "66y" || death.Cause.Value != "cholera" {
		t.Errorf("death = %+v", death)
	}

	if death := corpus.People["bianchi-maria"].Events[0]; death.Age.Value != "" || death.Note != "age: CHILD" {
		t.Errorf("an age the grammar cannot hold became %+v", death)
	}
	if !reported(notes, `an age could not be read and was kept in the note: "CHILD"`) {
		t.Errorf("the age kept in a note went unreported: %v", notes)
	}

	marriage := corpus.Families["bianchi-maria--rossi-giovanni"].Events[0]
	want = []model.Named{
		{Person: "rossi-giovanni", Role: "husband", Age: "24y"},
		{Person: "bianchi-maria", Role: "wife", Age: "18"},
		{Person: "rossi-antonio", Role: "witness"},
	}
	if !reflect.DeepEqual(marriage.With, want) {
		t.Errorf("the marriage names %+v", marriage.With)
	}
}

// 5.5.1 can write an association only under the person, saying at which event nobody
// knows. It is an event of its own, undated, naming them in the role given.
func TestAnAssociationUnderThePersonIsAnEventOfItsOwn(t *testing.T) {
	corpus, notes := convert(t, `0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 ASSO @I2@
2 RELA Godfather
2 NOTE Named in the family bible.
2 SOUR @S1@
3 PAGE 12
1 ASSO @I9@
2 RELA Friend
0 @I2@ INDI
1 NAME Antonio /Rossi/
`)
	events := corpus.People["rossi-giovanni"].Events
	want := []model.Event{{
		Type:    "association",
		With:    []model.Named{{Person: "rossi-antonio", Role: "Godfather"}},
		Note:    "Named in the family bible.",
		Sources: []model.CitationRef{{Source: "register", Anchor: "12"}},
	}}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("events = %+v", events)
	}
	if !reported(notes, "an ASSO names an individual who is not in the file") {
		t.Errorf("an association with nobody went unreported: %v", notes)
	}
}

// A citation is where in the source the entry is, when the entry was written and what
// it says, and the anchor is made from the page a reader looks up. One entry cited twice
// is one anchor, and the same page saying something else is another.
func TestACitationKeepsItsPageAndTheDateOfTheEntry(t *testing.T) {
	corpus, _ := convert(t, `0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME A /One/
1 BIRT
2 SOUR @S1@
3 PAGE c. 88r, n. 112
3 WWW https://example.org/88r
3 DATA
4 DATE 14 MAR 1834
4 TEXT nato il 12 marzo
1 DEAT
2 SOUR @S1@
3 PAGE c. 88r, n. 112
3 WWW https://example.org/88r
3 DATA
4 DATE 14 MAR 1834
4 TEXT nato il 12 marzo
1 BURI
2 SOUR @S1@
3 PAGE c. 88r, n. 112
3 DATA
4 DATE 15 MAR 1834
`)
	citations := corpus.Sources["register"].Citations
	want := model.Citation{Page: "c. 88r, n. 112", Text: "nato il 12 marzo",
		Links: model.LinksOf("https://example.org/88r")}
	want.Date, _ = model.ParseDate("14 MAR 1834")
	if got := citations["c-88r-n-112"]; !reflect.DeepEqual(got, want) {
		t.Errorf("citation = %+v", got)
	}
	if got := citations["c-88r-n-112-2"]; got.Date.String() != "15 MAR 1834" {
		t.Errorf("the other entry on the page = %+v", got)
	}
	var cited []string
	for _, event := range corpus.People["one-a"].Events {
		cited = append(cited, event.Sources[0].Anchor)
	}
	if want := []string{"c-88r-n-112", "c-88r-n-112", "c-88r-n-112-2"}; !slices.Equal(cited, want) {
		t.Errorf("anchors = %v, want %v", cited, want)
	}
}

// What a record is called elsewhere goes into ids under the service that calls it so:
// an EXID under the name its TYPE is the address of, a REFN under its TYPE or as refn,
// _FSFTID as familysearch and AFN as afn. The same identifier given twice is one, and
// two at one service keep the first.
func TestIdentifiersElsewhereAreIDs(t *testing.T) {
	corpus, notes := convert(t, `0 @R1@ REPO
1 NAME Archivio
1 REFN A-7
0 @S1@ SOUR
1 TITL Register
1 EXID Q2044
2 TYPE http://www.wikidata.org/entity/
0 @I1@ INDI
1 NAME A /One/
1 EXID XXXX-YYY
2 TYPE https://www.familysearch.org/tree/person/
1 _FSFTID XXXX-YYY
1 AFN GH52-6B
1 REFN 1234
1 REFN anc-99
2 TYPE Ancestry
1 REFN 5678
0 @I2@ INDI
1 NAME B /Two/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 REFN F-1
`)
	want := model.IDs{
		"familysearch": model.IdentifiersOf("XXXX-YYY"),
		"afn":          model.IdentifiersOf("GH52-6B"),
		"refn":         model.IdentifiersOf("1234"),
		"ancestry":     model.IdentifiersOf("anc-99"),
	}
	if got := corpus.People["one-a"].IDs; !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
	if !reported(notes, "REFN was kept as written") || corpus.People["one-a"].Body != "```gedcom\n1 REFN 5678\n```" {
		t.Errorf("a second identifier at one service was not kept: %v", notes)
	}
	if got := corpus.Families["one-a--two-b"].IDs; !reflect.DeepEqual(got, model.IDs{"refn": model.IdentifiersOf("F-1")}) {
		t.Errorf("family ids = %v", got)
	}
	if got := corpus.Sources["register"].IDs; !reflect.DeepEqual(got, model.IDs{"wikidata": model.IdentifiersOf("Q2044")}) {
		t.Errorf("source ids = %v", got)
	}
	if got := corpus.Repositories["archivio"].IDs; !reflect.DeepEqual(got, model.IDs{"refn": model.IdentifiersOf("A-7")}) {
		t.Errorf("repository ids = %v", got)
	}
}

// A picture comes in where its file is to hand. It is attached to the first record that
// names it, every other person naming it is somebody in it, and a CROP is turned from
// pixels into the fractions of the picture the model keeps. A scan of a source goes
// under scans/, and a picture that is not there is reported.
func TestAPictureComesInWhereItsFileIsToHand(t *testing.T) {
	dir := t.TempDir()
	picture(t, filepath.Join(dir, "photos", "wedding.png"), 200, 100)
	picture(t, filepath.Join(dir, "register.png"), 10, 10)

	corpus, attachments, notes := Convert(records(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 FILE photos/wedding.png
2 FORM image/png
3 MEDI PHOTO
2 TITL The wedding
0 @S1@ SOUR
1 TITL Register
1 OBJE
2 FILE register.png
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 OBJE @O1@
2 CROP
3 HEIGHT 100
3 WIDTH 100
1 OBJE
2 FILE C:\nowhere\lost.jpg
0 @I2@ INDI
1 NAME Maria /Bianchi/
1 OBJE @O1@
2 CROP
3 LEFT 100
`), nil, Beside(dir))

	giovanni := corpus.People["rossi-giovanni"]
	wedding := []model.Media{{
		File:  "photos/wedding.png",
		Type:  "photo",
		Title: "The wedding",
		Depicts: []model.Depicted{
			{Person: "rossi-giovanni", Region: &model.Region{Left: 0, Top: 0, Right: 0.5, Bottom: 1}},
			{Person: "bianchi-maria", Region: &model.Region{Left: 0.5, Top: 0, Right: 1, Bottom: 1}},
		},
	}}
	wedding = append(wedding, model.Media{File: `C:\nowhere\lost.jpg`})
	if !reflect.DeepEqual(giovanni.Media, wedding) {
		t.Errorf("media = %+v", giovanni.Media)
	}
	if !slices.Equal(giovanni.Files, []string{"photos/wedding.png"}) {
		t.Errorf("files = %v", giovanni.Files)
	}
	if maria := corpus.People["bianchi-maria"]; len(maria.Media) != 0 || len(maria.Files) != 0 {
		t.Errorf("a picture somebody else keeps was attached again: %+v", maria)
	}

	register := corpus.Sources["register"]
	if len(register.Media) != 1 || register.Media[0].File != "scans/register.png" {
		t.Errorf("the source's media = %+v", register.Media)
	}

	want := []Attachment{
		{Record: model.EntityRef{Kind: model.KindSource, Slug: "register"},
			File: "scans/register.png", From: filepath.Join(dir, "register.png")},
		{Record: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
			File: "photos/wedding.png", From: filepath.Join(dir, "photos", "wedding.png")},
	}
	if !reflect.DeepEqual(attachments, want) {
		t.Errorf("attachments = %+v", attachments)
	}
	if !reported(notes, "was not found beside it") {
		t.Errorf("a picture nobody has went unreported: %v", notes)
	}
}

// A 7.0 FILE is a URI reference, so a space in the name of a picture is written escaped
// and read as the space it is; 5.5.1's is the path as written.
func TestAFileIsReadAsTheURIReferenceItIs(t *testing.T) {
	dir := t.TempDir()
	picture(t, filepath.Join(dir, "wedding day.png"), 10, 10)
	picture(t, filepath.Join(dir, "a%20b.png"), 10, 10)

	for version, file := range map[string]string{"7.0": "wedding%20day.png", "5.5.1": "a%20b.png"} {
		corpus, attachments, _ := Convert(records(t, `0 HEAD
1 GEDC
2 VERS `+version+`
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 OBJE
2 FILE `+file+`
`), nil, Beside(dir))
		want, _ := url.PathUnescape(file)
		if version != "7.0" {
			want = file
		}
		media := corpus.People["rossi-giovanni"].Media
		if len(media) != 1 || media[0].File != "photos/"+want || len(attachments) != 1 {
			t.Errorf("%s: media = %+v, attachments = %+v", version, media, attachments)
		}
	}
}

func picture(t *testing.T, path string, width, height int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, image.NewGray(image.Rect(0, 0, width, height))); err != nil {
		t.Fatal(err)
	}
}

// An open word is kept as the file wrote it, in whatever script and however long, and
// only the format's own words are put into origins' words.
func TestAnOpenWordIsKeptAsWritten(t *testing.T) {
	long := "Served in the Regia Marina aboard the corazzata Dante Alighieri until the armistice"
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME Anna /Rossi/
2 TYPE MAIDEN
1 NAME Suor Anna /Rossi/
2 TYPE Nome da religiosa
1 EVEN
2 TYPE 兵役
1 EVEN
2 TYPE `+long+`
1 FACT 3
2 TYPE ???
1 EVEN
`)
	anna := corpus.People["rossi-anna"]
	if anna == nil {
		t.Fatalf("people are %v", corpus.PeopleSlugs())
	}
	if anna.Name.Type != "maiden" {
		t.Errorf("a name typed MAIDEN is %q", anna.Name.Type)
	}
	if len(anna.AlsoKnownAs) != 1 || anna.AlsoKnownAs[0].Type != "Nome da religiosa" {
		t.Errorf("also known as %+v", anna.AlsoKnownAs)
	}
	var kinds []string
	for _, event := range anna.Events {
		kinds = append(kinds, event.Type)
	}
	if want := []string{"兵役", long, "???", "event"}; !slices.Equal(kinds, want) {
		t.Errorf("events are %q, want %q", kinds, want)
	}
}

// A picture origins wrote is kept by the record its _KEEPER names, however many records
// name it before that one, each of them where it named it.
func TestAPictureIsKeptByTheRecordOriginsSaidKeepsIt(t *testing.T) {
	dir := t.TempDir()
	picture(t, filepath.Join(dir, "families/rossi-bianchi/photos/wedding.png"), 10, 10)
	picture(t, filepath.Join(dir, "notes/album/documents/page.png"), 10, 10)
	corpus, _, notes := Convert(records(t, `0 HEAD
1 GEDC
2 VERS 7.0
1 SCHMA
2 TAG _KEEPER `+extensions+`keeper
1 SOUR ORIGINS
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 OBJE @O2@
1 OBJE @O1@
0 @I2@ INDI
1 NAME Maria /Bianchi/
1 OBJE @O1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 OBJE @O1@
0 @O1@ OBJE
1 FILE families/rossi-bianchi/photos/wedding.png
2 FORM image/png
1 _KEEPER @F1@
0 @O2@ OBJE
1 FILE notes/album/documents/page.png
2 FORM image/png
3 MEDI MANUSCRIPT
1 _KEEPER @F9@
`), nil, Beside(dir))
	family := corpus.Families["bianchi-maria--rossi-giovanni"]
	if family == nil {
		t.Fatalf("families = %v", corpus.Families)
	}
	if len(family.Media) != 1 || family.Media[0].File != "photos/wedding.png" {
		t.Errorf("the family's media = %+v", family.Media)
	}
	var depicted []model.Slug
	for _, shown := range family.Media[0].Depicts {
		depicted = append(depicted, shown.Person)
	}
	if !slices.Equal(depicted, []model.Slug{"rossi-giovanni", "bianchi-maria"}) {
		t.Errorf("the wedding depicts %v", depicted)
	}
	// a _KEEPER naming a record that does not name the picture is no keeper
	media := corpus.People["rossi-giovanni"].Media
	if len(media) != 1 || media[0].File != "documents/page.png" {
		t.Errorf("rossi-giovanni's media = %+v", media)
	}
	if len(corpus.People["bianchi-maria"].Media) != 0 {
		t.Errorf("bianchi-maria's media = %+v", corpus.People["bianchi-maria"].Media)
	}
	if !slices.ContainsFunc(notes, func(n string) bool { return strings.Contains(n, "F9") }) {
		t.Errorf("notes = %q", notes)
	}
}
