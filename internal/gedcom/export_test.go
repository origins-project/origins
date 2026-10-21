package gedcom

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// export writes a corpus and reads what it wrote back as lines.
func export(t *testing.T, corpus *model.Corpus, options Options) (string, []*Node, Report) {
	t.Helper()
	var out bytes.Buffer
	report, err := Write(&out, corpus, options)
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("the export does not parse: %v\n%s", err, out.String())
	}
	return out.String(), nodes, report
}

// record is the record of the export with the xref.
func record(t *testing.T, nodes []*Node, xref string) *Node {
	t.Helper()
	for _, n := range nodes {
		if n.Xref == xref {
			return n
		}
	}
	t.Fatalf("no record @%s@", xref)
	return nil
}

// under is the first line down a path of tags.
func under(n *Node, tags ...string) *Node {
	for _, tag := range tags {
		n = n.First(tag)
	}
	return n
}

func date(t *testing.T, s string) model.Date {
	t.Helper()
	d, err := model.ParseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func person(slug model.Slug, given, surname string, sex model.Sex) *model.Person {
	return &model.Person{Slug: slug, Name: model.PersonName{Given: given, Surname: model.SurnameOf(surname)},
		Sex: model.StatedSex{Value: sex}}
}

// 7.0 has words for four calendars, and a date in another is written as the Gregorian
// days it covers with the corpus's own form beside it; a date 7.0's grammar cannot
// state for another reason says less, and the PHRASE says the rest. A date with no day
// to restate is an empty DATE and its PHRASE.
func TestADateIsWrittenAsTheGrammarAllowsAndPhrasedWhereItSaysLess(t *testing.T) {
	cases := []struct{ corpus, value, phrase string }{
		{"12 MAR 1834", "12 MAR 1834", ""},
		{"ABT 1850", "ABT 1850", ""},
		{"JULIAN 20 AUG 1917", "JULIAN 20 AUG 1917", ""},
		{"HEBREW 17 IYR 5661", "HEBREW 17 IYR 5661", ""},
		{"ISLAMIC 3 SAFAR 1319", "22 MAY 1901", "ISLAMIC 3 SAFAR 1319"},
		{"SWEDISH 30 FEB 1712", "11 MAR 1712", "SWEDISH 30 FEB 1712"},
		{"12 FEB 1750/51", "12 FEB 1751", "12 FEB 1750/51"},
		{"EST BET 1670 AND 1685", "BET 1670 AND 1685", "EST BET 1670 AND 1685"},
		{"3 MAR", "", "3 MAR"},
	}
	corpus := model.NewCorpus()
	p := person("rossi-anna", "Anna", "Rossi", model.Female)
	for _, c := range cases {
		p.Events = append(p.Events, model.Event{Type: "residence", Date: date(t, c.corpus)})
	}
	corpus.People[p.Slug] = p
	_, nodes, _ := export(t, corpus, Options{})

	residences := record(t, nodes, "I1").All("RESI")
	if len(residences) != len(cases) {
		t.Fatalf("residences = %d", len(residences))
	}
	for i, c := range cases {
		d := residences[i].First("DATE")
		if d.Text() != c.value || d.First("PHRASE").Text() != c.phrase {
			t.Errorf("%s is written DATE %q PHRASE %q, want %q and %q",
				c.corpus, d.Text(), d.First("PHRASE").Text(), c.value, c.phrase)
		}
	}
}

// The header names origins and its version, says who compiled the tree as a SUBM, gives
// the copyright and the language, and declares in SCHMA every extension tag of origins'
// own that the file uses and only those.
func TestTheHeaderSaysWhoWroteItAndDeclaresTheExtensions(t *testing.T) {
	corpus := model.NewCorpus()
	corpus.Project = model.Project{Name: "Rossi", Copyright: "© 2024 Anna Rossi", Lang: "it",
		Researchers: []model.Researcher{{Name: "Anna Rossi"}}}
	p := person("rossi-anna", "Anna", "Rossi", model.Female)
	p.Tags = []string{"To do"}
	corpus.People[p.Slug] = p
	text, nodes, report := export(t, corpus, Options{Version: "1.2.3"})

	head := nodes[0]
	if head.Tag != "HEAD" || under(head, "GEDC", "VERS").Text() != "7.0" {
		t.Fatalf("head = %+v", head)
	}
	if sour := head.First("SOUR"); sour.Value != "ORIGINS" || sour.First("VERS").Text() != "1.2.3" {
		t.Errorf("SOUR = %q %q", sour.Value, sour.First("VERS").Text())
	}
	if head.First("COPR").Text() != "© 2024 Anna Rossi" || head.First("LANG").Text() != "it" {
		t.Errorf("COPR %q LANG %q", head.First("COPR").Text(), head.First("LANG").Text())
	}
	if subm := head.First("SUBM").Text(); subm != "@U1@" || record(t, nodes, "U1").First("NAME").Text() != "Anna Rossi" {
		t.Errorf("SUBM = %q", subm)
	}
	var declared []string
	for _, tag := range under(head, "SCHMA").All("TAG") {
		declared = append(declared, tag.Value)
	}
	want := []string{"_NAME https://origins.invalid/gedcom/name", "_TAG https://origins.invalid/gedcom/tag"}
	if strings.Join(declared, "|") != strings.Join(want, "|") {
		t.Errorf("SCHMA = %q", declared)
	}
	if report.Extensions["_TAG"] != 1 || report.Records["INDI"] != 1 || report.Records["SUBM"] != 1 {
		t.Errorf("report = %+v", report)
	}
	if !strings.HasSuffix(text, "0 TRLR\n") || strings.Contains(text, "\r") {
		t.Errorf("the file ends %q", text[max(0, len(text)-20):])
	}
}

// A child in one relation to every partner is that relation as the PEDI of the child's
// FAMC; a child in a relation of its own to each is written per partner under the CHIL,
// as Ancestry's tags have it, which are not origins' to declare. Somebody the family has
// no record of is @VOID@ with their name as its PHRASE, and a partner stated to be nobody
// is not written and is counted.
func TestHowSomebodyIsAChildIsWrittenWhereTheyAreOne(t *testing.T) {
	corpus := model.NewCorpus()
	for _, p := range []*model.Person{
		person("rossi-giovanni", "Giovanni", "Rossi", model.Male),
		person("bianchi-maria", "Maria", "Bianchi", model.Female),
		person("rossi-luca", "Luca", "Rossi", model.Male),
		person("rossi-anna", "Anna", "Rossi", model.Female),
		person("rossi-pia", "Pia", "Rossi", model.Female),
	} {
		corpus.People[p.Slug] = p
	}
	corpus.Families["bianchi-maria--rossi-giovanni"] = &model.Family{
		Slug:     "bianchi-maria--rossi-giovanni",
		Partners: model.PartnersOf("rossi-giovanni", "bianchi-maria"),
		Children: []model.Child{
			{ID: "rossi-luca"},
			{ID: "rossi-anna", Relation: model.Relationship{Word: model.Adoptive}},
			{ID: "rossi-pia", Relation: model.Relationship{Partners: []model.PartnerRelation{
				{Partner: "rossi-giovanni", Relation: model.Step}, {Partner: "bianchi-maria", Relation: model.Birth}}}},
			{Name: model.PersonName{Text: "Gemma"}},
		},
	}
	corpus.Families["none--rossi-pia"] = &model.Family{
		Slug: "none--rossi-pia", Partners: []model.Partner{{Person: "rossi-pia"}, {Person: model.None}},
	}
	text, nodes, report := export(t, corpus, Options{})

	xref := map[string]string{}
	for _, n := range nodes {
		if n.Tag == "INDI" {
			xref[n.First("NAME").Text()] = n.Xref
		}
	}
	if pedi := under(record(t, nodes, xref["Luca /Rossi/"]), "FAMC", "PEDI"); pedi != nil {
		t.Errorf("a birth is written as PEDI %q", pedi.Value)
	}
	if pedi := under(record(t, nodes, xref["Anna /Rossi/"]), "FAMC", "PEDI").Text(); pedi != "ADOPTED" {
		t.Errorf("an adoption is PEDI %q", pedi)
	}
	family := record(t, nodes, "F1")
	if husb, wife := family.First("HUSB").Text(), family.First("WIFE").Text(); husb != "@"+xref["Giovanni /Rossi/"]+"@" ||
		wife != "@"+xref["Maria /Bianchi/"]+"@" {
		t.Errorf("HUSB %s WIFE %s", husb, wife)
	}
	var pia, gemma *Node
	for _, chil := range family.All("CHIL") {
		switch chil.Value {
		case "@" + xref["Pia /Rossi/"] + "@":
			pia = chil
		case "@VOID@":
			gemma = chil
		}
	}
	if pia == nil || pia.First("_FREL").Text() != "Step" || pia.First("_MREL").Text() != "Natural" {
		t.Errorf("the per-partner relation is %+v", pia)
	}
	if gemma == nil || gemma.First("PHRASE").Text() != "Gemma" {
		t.Errorf("a child known by name is %+v", gemma)
	}
	if strings.Contains(text, "_FREL http") || report.Omitted["partners.none"] != 1 {
		t.Errorf("SCHMA declares Ancestry's tags, or nobody was not counted: %+v", report.Omitted)
	}
	if lone := record(t, nodes, "F2"); lone.First("WIFE") == nil || lone.First("HUSB") != nil {
		t.Errorf("a woman alone is not the WIFE")
	}
}

// RESN is written where 7.0 has it — a record of an individual, a family or a picture,
// and an event — and an extension says the rest is private.
func TestWhatIsPrivateIsRestrictedWhere70HasARestriction(t *testing.T) {
	corpus := model.NewCorpus()
	p := person("rossi-anna", "Anna", "Rossi", model.Female)
	p.Private = true
	p.Name.Private = true
	p.Events = []model.Event{{Type: "birth", Private: true}}
	corpus.People[p.Slug] = p
	corpus.Sources["register"] = &model.Source{Slug: "register", Title: "Register", Private: true}
	_, nodes, _ := export(t, corpus, Options{})

	indi := record(t, nodes, "I1")
	if indi.First("RESN").Text() != "PRIVACY" || under(indi, "BIRT", "RESN").Text() != "PRIVACY" {
		t.Errorf("INDI %+v", indi)
	}
	if under(indi, "NAME", "RESN") != nil || under(indi, "NAME", "_PRIVATE") == nil {
		t.Errorf("a name is restricted by a RESN 7.0 does not have")
	}
	if sour := record(t, nodes, "S1"); sour.First("RESN") != nil || sour.First("_PRIVATE") == nil {
		t.Errorf("SOUR %+v", sour)
	}
}

// A picture is one OBJE record, its file a path from the root of the corpus, and the
// part of it a pointer means is a CROP in pixels where its size is known and the
// corpus's fractions otherwise.
func TestAPictureIsOneObjectAndItsRegionACrop(t *testing.T) {
	corpus := model.NewCorpus()
	p := person("rossi-anna", "Anna", "Rossi", model.Female)
	p.Media = []model.Media{{File: "photos/wedding day.jpg", Title: "The wedding",
		Depicts: []model.Depicted{{Person: "rossi-luca", Region: &model.Region{Left: 0.5, Top: 0.25, Right: 1, Bottom: 0.75}}}}}
	corpus.People[p.Slug] = p
	luca := person("rossi-luca", "Luca", "Rossi", model.Male)
	corpus.People[luca.Slug] = luca

	_, nodes, _ := export(t, corpus, Options{Size: func(path string) (int, int, bool) {
		return 800, 600, path == "people/rossi-anna/photos/wedding day.jpg"
	}})
	obje := record(t, nodes, "O1")
	if file := obje.First("FILE"); file.Value != "people/rossi-anna/photos/wedding%20day.jpg" ||
		file.First("FORM").Text() != "image/jpeg" || file.First("TITL").Text() != "The wedding" {
		t.Errorf("FILE %+v", file)
	}
	if under(record(t, nodes, "I1"), "OBJE").Text() != "@O1@" {
		t.Errorf("the keeper does not point at the picture")
	}
	crop := under(record(t, nodes, "I2"), "OBJE", "CROP")
	if crop.First("TOP").Text() != "150" || crop.First("LEFT").Text() != "400" ||
		crop.First("HEIGHT").Text() != "300" || crop.First("WIDTH").Text() != "400" {
		t.Errorf("CROP %+v", crop)
	}
	if under(record(t, nodes, "I2"), "OBJE", "_REGION") != nil {
		t.Error("a CROP that says the region was written with its fractions")
	}

	// seven pixels have no middle, so the fractions go beside the CROP and are read over it
	_, nodes, _ = export(t, corpus, Options{Size: func(string) (int, int, bool) { return 7, 7, true }})
	if obje := under(record(t, nodes, "I2"), "OBJE"); obje.First("CROP") == nil ||
		obje.First("_REGION").Text() != "0.5 0.25 1 0.75" {
		t.Errorf("OBJE %+v", obje)
	}

	_, nodes, _ = export(t, corpus, Options{})
	if region := under(record(t, nodes, "I2"), "OBJE", "_REGION").Text(); region != "0.5 0.25 1 0.75" {
		t.Errorf("_REGION %q", region)
	}
}

// The fixture is written as the import reads it: every structure the export writes, of
// 7.0's or one of origins' own extensions, is read back, and nothing is kept as written.
// The same corpus writes the same bytes.
func TestTheFixtureIsWrittenAsTheImportReadsIt(t *testing.T) {
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(filepath.Join(root, store.ProjectFile)); err != nil {
		t.Skip("the fixture is not built: run tools/mkfixture.sh")
	}
	corpus, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	text, nodes, report := export(t, corpus.Corpus, Options{Version: "test"})
	again, _, _ := export(t, corpus.Corpus, Options{Version: "test"})
	if again != text {
		t.Error("the same corpus wrote different bytes")
	}

	read, _, notes := Convert(nodes, nil, nil)
	for _, note := range notes {
		if strings.Contains(note, "was kept as written") {
			t.Errorf("the import did not read: %s", note)
		}
	}
	for kind, n := range map[string]int{
		"INDI": len(corpus.People), "FAM": len(corpus.Families), "SOUR": len(corpus.Sources),
		"REPO": len(corpus.Repositories), "SNOTE": len(corpus.Notes),
	} {
		if report.Records[kind] != n {
			t.Errorf("%d %s records for %d", report.Records[kind], kind, n)
		}
	}
	if len(read.People) != len(corpus.People) || len(read.Families) != len(corpus.Families) ||
		len(read.Repositories) != len(corpus.Repositories) {
		t.Errorf("read back %d people, %d families, %d repositories",
			len(read.People), len(read.Families), len(read.Repositories))
	}
	if report.Omitted["hypotheses"] != len(corpus.Hypotheses) {
		t.Errorf("omitted = %+v", report.Omitted)
	}
}

// What an import kept as written is written again as the lines it was, where it was, and
// an import of the export keeps it as the first import did.
func TestWhatAnImportKeptGoesBackOutAsItWas(t *testing.T) {
	first, _ := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
1 _EXPORTED_BY somebody
0 @I1@ INDI
1 NAME A /One/
1 RFN 1234
1 _XMILITARY Served in the artillery
2 _RANK gunner
1 BIRT
2 DATE 1834
2 PLAC Firenze
3 _QUARTER Santa Croce
2 _WEATHER rain
3 _WIND north
0 @U1@ _CUSTOM
1 NOTE something this exporter invented
1 _SELF @U1@
1 _WHO @I1@
0 TRLR
`)
	text, nodes, report := export(t, first, Options{Version: "test"})
	for _, want := range []string{
		"0 HEAD\n", "1 _EXPORTED_BY somebody\n",
		"1 RFN 1234\n1 _XMILITARY Served in the artillery\n2 _RANK gunner\n",
		"2 PLAC Firenze\n3 _QUARTER Santa Croce\n",
		"2 _WEATHER rain\n3 _WIND north\n",
		"0 @U1@ _CUSTOM\n1 NOTE something this exporter invented\n1 _SELF @U1@\n1 _WHO @VOID@\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the export has no %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "```") || strings.Contains(text, "SNOTE") || strings.Count(text, "0 HEAD") != 1 ||
		strings.Contains(text, "_RANK https") || strings.Count(text, "2 PLAC Firenze") != 1 {
		t.Errorf("what was kept is written as notes, twice or declared as origins':\n%s", text)
	}
	if report.Kept["RFN"] != 1 || report.Kept["_CUSTOM"] != 1 || report.Omitted["kept.pointers"] != 1 {
		t.Errorf("the report is %+v", report)
	}
	again, _, _ := Convert(nodes, nil, nil)
	want := strings.Replace(prose(first), "1 _WHO @I1@", "1 _WHO @VOID@", 1)
	if got := prose(again); got != want {
		t.Errorf("imported again, the prose is\n%s\nwant\n%s", got, want)
	}
}
