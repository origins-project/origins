package gedcom

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// A shared note is a note record, titled with its first line — in HTML, where a line
// ends at a paragraph or a break, the whole of this one — its identifiers its ids,
// and every pointer to it a link to it where its text would have been. A note says what
// language it is in and gives its translations, HTML is written as markdown, and what a
// note cites is linked at its end.
func TestANoteSaysWhatItIsInAndAnIdentifiedOneIsARecord(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @N1@ SNOTE "Gordon" is a <i>Scottish</i> surname.
1 CONT It became a given name.
1 MIME text/html
1 LANG en
1 UID 6a9d3c1e
1 EXID N-17
2 TYPE https://www.familysearch.org/notes/
0 @N2@ NOTE Born in Vicchio, per the family letters.
1 REFN 42
0 @S1@ SOUR
1 TITL Letters
0 @I1@ INDI
1 NAME Gordon /Jones/
2 SNOTE @N1@
1 NOTE Nato a Vicchio.
2 LANG it
2 TRAN Born in Vicchio.
3 LANG en
2 SOUR @S1@
3 PAGE 12
1 NOTE @N2@
0 @I2@ INDI
1 NAME Anna /Jones/
1 NOTE @N2@
`)
	gordon := corpus.Notes["gordon-is-a-scottish-surname-it"]
	if gordon == nil {
		t.Fatalf("notes are %v, report %v", corpus.NoteSlugs(), notes)
	}
	if gordon.Title != `"Gordon" is a Scottish surname. It became a given name.` ||
		gordon.Body != "*en:* \"Gordon\" is a *Scottish* surname. It became a given name." ||
		!reflect.DeepEqual(gordon.IDs, model.IDs{"uid": model.IdentifiersOf("6a9d3c1e"), "familysearch": model.IdentifiersOf("N-17")}) {
		t.Errorf("note = %+v", gordon)
	}
	letters := corpus.Notes["born-in-vicchio-per-the-family"]
	if letters == nil || letters.Body != "Born in Vicchio, per the family letters." ||
		!reflect.DeepEqual(letters.IDs, model.IDs{"refn": model.IdentifiersOf("42")}) {
		t.Fatalf("notes are %v: %+v", corpus.NoteSlugs(), letters)
	}
	link := "[Born in Vicchio, per the family letters.](../../notes/born-in-vicchio-per-the-family/README.md)"
	person := corpus.People["jones-gordon"]
	if want := `["Gordon" is a Scottish surname. It became a given name.](../../notes/gordon-is-a-scottish-surname-it/README.md)`; person.Name.Note != want {
		t.Errorf("name note = %q", person.Name.Note)
	}
	if want := "*it:* Nato a Vicchio.\n\n*Translation, en:* Born in Vicchio.\n\n" +
		"[letters#12](../../sources/letters/README.md#12)\n\n" + link; person.Body != want {
		t.Errorf("body = %q", person.Body)
	}
	if corpus.People["jones-anna"].Body != link {
		t.Errorf("the other body = %q", corpus.People["jones-anna"].Body)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
	emitAll(t, corpus)
}

// HTML becomes markdown where markdown has a mark for it, and stays HTML where it has
// none, so that nothing it said goes.
func TestANoteInHTMLIsMarkdown(t *testing.T) {
	for html, want := range map[string]string{
		"<p>One <b>bold</b>  word.</p><p>Two</p>": "One **bold** word.\n\nTwo",
		"a<br>b": "a\\\nb",
		`see <a href="https://example.org">the page</a>`: "see [the page](https://example.org)",
		"H<sub>2</sub>O and <u>this</u>":                 "H<sub>2</sub>O and <u>this</u>",
		`<font color="red">red</font> &amp; <s>gone</s>`: `<font color="red">red</font> &amp; ~~gone~~`,
	} {
		if got := markdown(html); got != want {
			t.Errorf("markdown(%q) = %q, want %q", html, got, want)
		}
	}
}

// A source has its short title and its own text, says what it covers, and is kept in
// every archive under every call number its REPO lines give, each in its medium.
func TestASourceSaysWhatItCoversAndWhereItIsKept(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @R1@ REPO
1 NAME Archivio Diocesano
0 @R2@ REPO
1 NAME FamilySearch
0 @S1@ SOUR
1 TITL Parish of San Lorenzo, baptisms 1820-1850
1 ABBR S. Lorenzo, Battesimi 4
1 TEXT Liber baptizatorum
2 LANG la
1 DATA
2 EVEN BIRT, CHR
3 DATE FROM 1820 TO 1850
3 PLAC Firenze
2 AGNC Parrocchia di San Lorenzo
2 NOTE Two volumes bound as one.
3 LANG en
1 REPO @R1@
2 NOTE Ask a day ahead.
2 CALN ADF/SL/B/4
2 CALN B-4
3 MEDI BOOK
1 REPO @R2@
2 CALN 1903581
3 MEDI FILM
2 CALN 1903582
3 MEDI OTHER
4 PHRASE Digital images
0 @S2@ SOUR
1 ABBR Stato civile
`)
	source := corpus.Sources["parish-of-san-lorenzo-baptisms-1820"]
	if source == nil {
		t.Fatalf("sources are %v", corpus.SourceSlugs())
	}
	period, _ := model.ParseDate("FROM 1820 TO 1850")
	want := []model.Coverage{{Events: []string{"birth", "christening"}, Date: period, Place: "firenze"}}
	if source.ShortTitle != "S. Lorenzo, Battesimi 4" || source.Text != "*la:* Liber baptizatorum" ||
		!reflect.DeepEqual(source.Covers, want) ||
		!reflect.DeepEqual(source.Attributes, []model.Attribute{{Type: "agency", Value: "Parrocchia di San Lorenzo"}}) ||
		source.Body != "*Coverage, en:* Two volumes bound as one." {
		t.Errorf("source = %+v", source)
	}
	held := []model.Holding{
		{Repository: "archivio-diocesano", CallNumber: "ADF/SL/B/4", Note: "Ask a day ahead."},
		{Repository: "archivio-diocesano", CallNumber: "B-4", Medium: "book"},
		{Repository: "familysearch", CallNumber: "1903581", Medium: "microfilm"},
		{Repository: "familysearch", CallNumber: "1903582", Medium: "Digital images"},
	}
	if !slices.Equal(source.Repositories, held) {
		t.Errorf("repositories = %+v", source.Repositories)
	}
	if civil := corpus.Sources["stato-civile"]; civil == nil || civil.Title != "" || civil.ShortTitle != "Stato civile" {
		t.Errorf("sources are %v", corpus.SourceSlugs())
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
	emitAll(t, corpus)
}

// A citation says what kind of entry it is and whom it names: the person citing it in
// the role its ROLE gives, or on a family the partner the role names. One entry cited by
// two people is one anchor naming both. Every TEXT is a paragraph of the transcription,
// its scans are kept by the source, and QUAY is the weight of the reference.
func TestACitationSaysWhatEntryItIsAndWhomItNames(t *testing.T) {
	dir := t.TempDir()
	picture(t, filepath.Join(dir, "scan.png"), 10, 10)
	corpus, _, notes := Convert(records(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @S1@ SOUR
1 TITL Register
0 @I1@ INDI
1 NAME Luca /Rossi/
1 BIRT
2 SOUR @S1@
3 PAGE n. 112
3 DATA
4 TEXT Lucas filius Antonii
5 LANG la
4 TEXT Luca, son of Antonio
5 LANG en
3 EVEN BAPM
4 ROLE CHIL
3 QUAY 2
3 OBJE
4 FILE scan.png
3 NOTE The hand is clear.
0 @I2@ INDI
1 NAME Antonio /Rossi/
1 BIRT
2 SOUR @S1@
3 PAGE n. 112
3 DATA
4 TEXT Lucas filius Antonii
5 LANG la
4 TEXT Luca, son of Antonio
5 LANG en
3 EVEN BAPM
4 ROLE FATH
3 OBJE
4 FILE scan.png
3 NOTE The hand is clear.
0 @I3@ INDI
1 NAME Anna /Bianchi/
0 @F1@ FAM
1 HUSB @I2@
1 WIFE @I3@
1 MARR
2 SOUR @S1@
3 PAGE n. 9
3 EVEN MARR
4 ROLE WIFE
2 SOUR @S1@
3 PAGE n. 9
3 EVEN MARR
4 ROLE OTHER
5 PHRASE best man
`), nil, Beside(dir))
	luca := corpus.People["rossi-luca"]
	if got := luca.Events[0].Sources; len(got) != 1 || got[0].String() != "register#n-112 ---2" {
		t.Fatalf("sources = %v, notes %v", got, notes)
	}
	citations := corpus.Sources["register"].Citations
	want := model.Citation{
		Event:  "baptism",
		Page:   "n. 112",
		Text:   "*la:* Lucas filius Antonii\n\n*en:* Luca, son of Antonio",
		With:   []model.Named{{Person: "rossi-luca", Role: "child"}, {Person: "rossi-antonio", Role: "father"}},
		Images: []model.Media{{File: "scans/scan.png"}},
		Note:   "The hand is clear.",
	}
	if got := citations["n-112"]; !reflect.DeepEqual(got, want) {
		t.Errorf("citation = %+v", got)
	}
	if files := corpus.Sources["register"].Files; !slices.Equal(files, []string{"scans/scan.png"}) {
		t.Errorf("files = %v", files)
	}
	wedding := []model.Named{{Person: "bianchi-anna", Role: "wife"}, {Person: model.Unknown, Role: "best man"}}
	if got := citations["n-9"]; got.Event != "marriage" || !reflect.DeepEqual(got.With, wedding) {
		t.Errorf("the wedding's citation = %+v", got)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
	emitAll(t, corpus)
}

// A picture is an event's where the OBJE is under the event, its MEDI its type and the
// record's identifiers its ids. A crop on a picture not to hand is kept in the file's
// words, a TRAN found stands in for a file that is not, a record naming a picture kept
// by another refers to it by its path, and a picture nothing names is a note keeping
// it, as a record of a type origins does not read is a note keeping it as written.
func TestEveryPictureHasAnEntry(t *testing.T) {
	dir := t.TempDir()
	picture(t, filepath.Join(dir, "grave.png"), 10, 10)
	picture(t, filepath.Join(dir, "letter.png"), 10, 10)
	corpus, attachments, notes := Convert(records(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 FILE grave.png
2 FORM image/png
3 MEDI TOMBSTONE
2 TITL The grave
1 UID 1b2c
0 @O2@ OBJE
1 FILE letter.tif
2 FORM image/tiff
3 MEDI MANUSCRIPT
2 TRAN letter.png
3 FORM image/png
0 @O3@ OBJE
1 FILE album/page3.jpg
2 FORM image/jpeg
2 TITL The album, page 3
0 @I1@ INDI
1 NAME Luca /Rossi/
1 BURI
2 OBJE @O1@
1 OBJE @O2@
1 OBJE
2 FILE lost.jpg
3 FORM image/jpeg
2 CROP
3 TOP 10
3 LEFT 20
0 @F1@ FAM
1 HUSB @I1@
1 OBJE @O2@
2 CROP
3 WIDTH 5
0 @L1@ _LOC
1 NAME Firenze
`), nil, Beside(dir))
	luca := corpus.People["rossi-luca"]
	grave := []model.Media{{File: "documents/grave.png", Type: "tombstone", Title: "The grave", IDs: model.IDs{"uid": model.IdentifiersOf("1b2c")}}}
	if !reflect.DeepEqual(luca.Events[0].Media, grave) {
		t.Errorf("the burial's media = %+v", luca.Events[0].Media)
	}
	own := []model.Media{
		{File: "documents/letter.png", Type: "manuscript"},
		{File: "lost.jpg", Attributes: []model.Attribute{{Type: "crop", Value: "TOP 10, LEFT 20"}}},
	}
	if !reflect.DeepEqual(luca.Media, own) {
		t.Errorf("media = %+v", luca.Media)
	}
	family := corpus.Families["rossi-luca--unknown"]
	if want := []model.Media{{File: "../../people/rossi-luca/documents/letter.png",
		Region: &model.Region{Left: 0, Top: 0, Right: 0.5, Bottom: 1}}}; !reflect.DeepEqual(family.Media, want) {
		t.Errorf("the family's media = %+v", family.Media)
	}
	if len(attachments) != 2 {
		t.Errorf("attachments = %+v", attachments)
	}
	album := corpus.Notes["the-album-page-3"]
	if album == nil || album.Title != "The album, page 3" ||
		!reflect.DeepEqual(album.Media, []model.Media{{File: "album/page3.jpg", Title: "The album, page 3"}}) {
		t.Errorf("notes are %v: %+v", corpus.NoteSlugs(), album)
	}
	location := corpus.Notes["loc-l1"]
	if location == nil || location.Title != "_LOC @L1@" || location.Body != "```gedcom\n0 @L1@ _LOC\n1 NAME Firenze\n```" {
		t.Errorf("notes are %v: %+v", corpus.NoteSlugs(), location)
	}
	if !reported(notes, "_LOC was kept as written") || reported(notes, "OBJE was kept") || reported(notes, "TRAN was kept") {
		t.Errorf("notes = %v", notes)
	}
	emitAll(t, corpus)
}

// The submitters are the researchers who compiled the tree, and the header says whose
// copyright it is and what language it is in, what else a submitter says kept as written
// in their note. The header's NOTE is a note record, and 5.5.1's SUBN, with what else of
// the header nothing reads, is kept as written in it.
func TestTheFileSaysWhoCompiledIt(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 5.5.1
1 SUBM @U1@
1 SUBN @N9@
1 COPR © 2024 Anna Rossi
1 LANG Italian
1 NOTE Compiled from the parish books.
2 CONT Second line.
0 @U1@ SUBM
1 NAME Anna Rossi
1 ADDR Via dei Servi 15
2 CITY Firenze
1 EMAIL anna@example.org
1 NOTE Compiled the Rossi branch.
0 @U2@ SUBM
1 NAME Ottavia Bianchi
1 LANG Italian
0 @N9@ SUBN
1 TEMP SLAKE
`)
	project := corpus.Project
	researchers := []model.Researcher{
		{Name: "Anna Rossi", Addresses: []model.Address{{Lines: "Via dei Servi 15", City: "Firenze"}},
			Links: model.LinksOf("mailto:anna@example.org"), Note: "Compiled the Rossi branch."},
		{Name: "Ottavia Bianchi", Note: "```gedcom\n1 LANG Italian\n```"},
	}
	if project.Copyright != "© 2024 Anna Rossi" || project.Lang != "it" || !reflect.DeepEqual(project.Researchers, researchers) {
		t.Errorf("project = %+v", project)
	}
	header := corpus.Notes["compiled-from-the-parish-books"]
	if header == nil {
		t.Fatalf("notes are %v", corpus.NoteSlugs())
	}
	if want := "Compiled from the parish books.\nSecond line.\n\n```gedcom\n1 SUBN @N9@\n```\n\n```gedcom\n0 @N9@ SUBN\n1 TEMP SLAKE\n```"; header.Body != want {
		t.Errorf("header note = %q", header.Body)
	}
	if slugs := corpus.NoteSlugs(); len(slugs) != 1 {
		t.Errorf("notes are %v", slugs)
	}
	if !reported(notes, "SUBN was kept as written") {
		t.Errorf("notes = %v", notes)
	}
	emitAll(t, corpus)
}

// A picture citing a source whose citation shows the same picture is a cycle 7.0 forbids
// and says to read safely, and is read once.
func TestAPictureCitingItselfIsReadOnce(t *testing.T) {
	corpus, _ := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 FILE page.jpg
2 FORM image/jpeg
1 SOUR @S1@
2 PAGE 3
2 OBJE @O1@
0 @S1@ SOUR
1 TITL Album
0 @I1@ INDI
1 NAME Luca /Rossi/
1 OBJE @O1@
`)
	media := corpus.People["rossi-luca"].Media
	if len(media) != 1 || media[0].File != "page.jpg" || len(media[0].Sources) != 1 {
		t.Errorf("media = %+v", media)
	}
}

// A source and an archive named in a script no slug spells are filed under their
// identifiers, the name kept as written.
func TestASourceNamedInAnotherScriptIsFiledUnderItsIdentifier(t *testing.T) {
	corpus, _ := convert(t, `0 @R1@ REPO
1 NAME دار الوثائق القومية
0 @S1@ SOUR
1 TITL سجل عقود الزواج، حلب
1 REPO @R1@
`)
	source := corpus.Sources["source-s1"]
	if source == nil || source.Title != "سجل عقود الزواج، حلب" {
		t.Fatalf("the source is %+v among %v", source, corpus.SlugsOf(model.KindSource))
	}
	if archive := corpus.Repositories["repository-r1"]; archive == nil || archive.Name != "دار الوثائق القومية" {
		t.Errorf("the archive is %+v among %v", archive, corpus.SlugsOf(model.KindRepository))
	}
	if len(source.Repositories) != 1 || source.Repositories[0].Repository != "repository-r1" {
		t.Errorf("the source is kept in %+v", source.Repositories)
	}
}
