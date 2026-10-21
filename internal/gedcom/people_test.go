package gedcom

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// Either side of a link says whom a family joins and whose child somebody is, so a FAM
// naming nobody is still the family of whoever names it back. What a child's own link
// says of it — how sure it is and the notes on it — is the child's entry, and the notes
// on a partner's link are the partner's.
func TestAFamilyIsReadFromEitherSideOfTheLink(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 FAMS @F1@
2 NOTE Married at twenty.
0 @I2@ INDI
1 NAME Anna /Bianchi/
1 FAMS @F1@
0 @I3@ INDI
1 NAME Luca /Rossi/
1 FAMC @F1@
2 STAT CHALLENGED
3 PHRASE The register gives another father.
2 NOTE Baptised a month after the wedding.
0 @F1@ FAM
`)
	family := corpus.Families["bianchi-anna--rossi-giovanni"]
	if family == nil {
		t.Fatalf("families are %v, notes %v", corpus.FamilySlugs(), notes)
	}
	if want := []model.Partner{{Person: "bianchi-anna"}, {Person: "rossi-giovanni", Note: "Married at twenty."}}; !reflect.DeepEqual(family.Partners, want) {
		t.Errorf("partners = %+v", family.Partners)
	}
	want := []model.Child{{ID: "rossi-luca", Status: model.Challenged,
		Note: "The register gives another father.\n\nBaptised a month after the wedding."}}
	if !reflect.DeepEqual(family.Children, want) {
		t.Errorf("children = %+v", family.Children)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// 7.0 points at @VOID@ for somebody it has no record of, and the PHRASE beside it is
// all it knows of them: a partner or a child known by name, or one not known at all. A
// PHRASE beside a partner who has a record is the file's words for the link.
func TestSomebodyAFamilyHasNoRecordOfIsInIt(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Giovanni /Rossi/
0 @F1@ FAM
1 HUSB @I1@
2 PHRASE the father, by his own account
1 WIFE @VOID@
2 PHRASE Maria la Rossa
1 CHIL @VOID@
2 PHRASE Pietro
1 CHIL @VOID@
`)
	family := corpus.Families["rossi-giovanni--unknown"]
	if family == nil {
		t.Fatalf("families are %v, notes %v", corpus.FamilySlugs(), notes)
	}
	partners := []model.Partner{{Person: "rossi-giovanni", Note: "the father, by his own account"},
		{Name: model.PersonName{Text: "Maria la Rossa"}}}
	children := []model.Child{{Name: model.PersonName{Text: "Pietro"}}, {ID: model.Unknown}}
	if !reflect.DeepEqual(family.Partners, partners) || !reflect.DeepEqual(family.Children, children) {
		t.Errorf("partners = %+v, children = %+v", family.Partners, family.Children)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// A family names its associates as a person does, each an event of its own, and a
// PHRASE beside an associate who has a record is the file's words for the link, leading
// the note, as it is at an event.
func TestAFamilyNamesItsAssociates(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BAPM
2 ASSO @I2@
3 PHRASE his uncle
3 ROLE GODP
0 @I2@ INDI
1 NAME Pietro /Rossi/
0 @F1@ FAM
1 HUSB @I1@
1 ASSO @I2@
2 PHRASE the groom's uncle
2 ROLE OTHER
3 PHRASE best man
2 NOTE Stood in for the groom's father.
1 ASSO @VOID@
2 ROLE CLERGY
`)
	family := corpus.Families["rossi-giovanni--unknown"]
	if family == nil {
		t.Fatalf("families are %v, notes %v", corpus.FamilySlugs(), notes)
	}
	want := []model.Event{
		{Type: "association", Note: "the groom's uncle\nStood in for the groom's father.",
			With: []model.Named{{Person: "rossi-pietro", Role: "best man"}}},
		{Type: "association", With: []model.Named{{Person: model.Unknown, Role: "clergy"}}},
	}
	if !reflect.DeepEqual(family.Events, want) {
		t.Errorf("events = %+v", family.Events)
	}
	with := corpus.People["rossi-giovanni"].Events[0].With
	if !reflect.DeepEqual(with, []model.Named{{Person: "rossi-pietro", Role: "godparent", Note: "his uncle"}}) {
		t.Errorf("the baptism names %+v", with)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// A birth's FAMC names the family the child was born into, which says nothing the family
// does not where the two agree. Where the family lists the child otherwise the file has
// said something of the birth, and the event keeps the family. An adoption's FAMC says
// who adopted, and the child is adopted by them alone.
func TestABirthAndAnAdoptionNameTheirFamilies(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 FAMS @F1@
0 @I2@ INDI
1 NAME Anna /Bianchi/
1 FAMS @F1@
0 @I3@ INDI
1 NAME Luca /Rossi/
1 BIRT
2 FAMC @F1@
1 FAMC @F1@
0 @I4@ INDI
1 NAME Paolo /Rossi/
1 BIRT
2 FAMC @F1@
1 ADOP
2 FAMC @F1@
3 ADOP HUSB
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
`)
	family := corpus.Families["bianchi-anna--rossi-giovanni"]
	if family == nil {
		t.Fatalf("families are %v", corpus.FamilySlugs())
	}
	if luca := corpus.People["rossi-luca"]; luca.Events[0].Family != "" {
		t.Errorf("a birth agreeing with its family names it: %+v", luca.Events[0])
	}
	paolo := corpus.People["rossi-paolo"]
	if paolo.Events[0].Family != family.Slug || paolo.Events[1].Family != family.Slug {
		t.Errorf("events = %+v", paolo.Events)
	}
	i := slices.IndexFunc(family.Children, func(c model.Child) bool { return c.ID == "rossi-paolo" })
	if i < 0 || family.Children[i].Relation.To("rossi-giovanni") != model.Adoptive ||
		family.Children[i].Relation.To("bianchi-anna") != model.Birth {
		t.Errorf("children = %+v", family.Children)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// The Latter-day Saints' rites are events, a temple and an ordinance's status, dated, are
// attributes of them, and a sealing to parents names the family it seals to. A PEDI of
// SEALING is the relation the sealing makes.
func TestTheRitesAreEvents(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 FAMS @F1@
0 @I2@ INDI
1 NAME Luca /Rossi/
1 BAPL
2 DATE 3 MAR 1990
2 TEMP SLAKE
2 STAT COMPLETED
3 DATE 4 MAR 1990
1 ENDL
2 STAT DNS_CAN
1 SLGC
2 FAMC @F1@
1 FAMC @F1@
2 PEDI SEALING
0 @F1@ FAM
1 HUSB @I1@
1 SLGS
2 STAT BIC
`)
	luca := corpus.People["rossi-luca"]
	var types []string
	for _, event := range luca.Events {
		types = append(types, event.Type)
	}
	if want := []string{"lds-baptism", "lds-endowment", "lds-sealing-to-parents"}; !slices.Equal(types, want) {
		t.Fatalf("events are %q, notes %v", types, notes)
	}
	baptism := luca.Events[0].Attributes
	if len(baptism) != 2 || baptism[0].Type != "temple" || baptism[0].Value != "SLAKE" ||
		baptism[1].Type != "status" || baptism[1].Value != "completed" || baptism[1].Date.String() != "4 MAR 1990" {
		t.Errorf("baptism attributes = %+v", baptism)
	}
	if got := luca.Events[1].Attributes; len(got) != 1 || got[0].Value != "do-not-seal-canceled" {
		t.Errorf("endowment attributes = %+v", got)
	}
	family := corpus.Families["rossi-giovanni--unknown"]
	if luca.Events[2].Family != family.Slug {
		t.Errorf("the sealing names %q", luca.Events[2].Family)
	}
	if len(family.Children) != 1 || family.Children[0].Relation.To("rossi-giovanni") != "sealing" {
		t.Errorf("children = %+v", family.Children)
	}
	if len(family.Events) != 1 || family.Events[0].Type != "lds-sealing-to-spouse" ||
		family.Events[0].Attributes[0].Value != "born-in-covenant" {
		t.Errorf("family events = %+v", family.Events)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// A name given in pieces is joined, two given names being one part, and two surnames are
// a list, each with the prefix written in its place. Where the payload says what the
// parts do not, it is kept beside them as the name's text.
func TestANameInPiecesIsJoined(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME María José /de la Fuente y García/
2 GIVN María
2 GIVN José
2 SPFX de la
2 SURN Fuente
2 SURN García
2 NICK Pepa
2 NICK Pepita
0 @I2@ INDI
1 NAME Giò /Rossi/
2 GIVN Giovanni
2 SURN Rossi
`)
	maria := corpus.People["fuente-maria-jose"]
	if maria == nil {
		t.Fatalf("people are %v, notes %v", corpus.PeopleSlugs(), notes)
	}
	name := maria.Name
	if name.Given != "María José" || name.Nickname != "Pepa, Pepita" ||
		!reflect.DeepEqual(name.Surname, model.Surnames{{Name: "Fuente", Prefix: "de la"}, {Name: "García"}}) {
		t.Errorf("name = %+v", name)
	}
	if name.Text != "María José /de la Fuente y García/" {
		t.Errorf("a payload saying more lost its words: %q", name.Text)
	}
	if got := corpus.People["rossi-giovanni"].Name.Text; got != "Giò /Rossi/" {
		t.Errorf("text = %q", got)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// A name written again — in another language, phonetically, in Latin letters — is
// another name the person went by, of the kind the name is, in the language the writing
// says: a TRAN's LANG, a method 5.5.1 names, or one of the file's own as a private-use
// tag. A method in more words than one is said in the name's note.
func TestANameWrittenAgainIsAnotherName(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Taro /Yamada/
2 TYPE birth
2 TRAN 山田 /太郎/
3 LANG ja
2 FONE やまだ たろう
3 TYPE kana
2 ROMN Yamada /Taro/
3 TYPE hepburn
2 ROMN Jamada /Taro/
3 TYPE as the clerk heard it
2 ROMN Yamada Tarō
`)
	person := corpus.People["yamada-taro"]
	if person == nil {
		t.Fatalf("people are %v, notes %v", corpus.PeopleSlugs(), notes)
	}
	type said struct{ text, lang, typ, note string }
	var got []said
	for _, name := range person.AlsoKnownAs {
		got = append(got, said{name.String(), string(name.Lang), name.Type, name.Note})
	}
	want := []said{
		{"山田 太郎", "ja", "birth", ""},
		{"やまだ たろう", "ja-Kana", "birth", ""},
		{"Yamada Taro", "und-Latn-x-hepburn", "birth", ""},
		{"Jamada Taro", "", "birth", "romanized, by as the clerk heard it"},
		{"Yamada Tarō", "", "birth", "romanized"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("also known as\n%q\nwant\n%q", got, want)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// The extension tags Gramps reads under a NAME are other names, of the kind each tag
// gives. A married or an adopted name without slashes is the surname taken, beside the
// name's own given names, however many words it is; one with slashes is a name of its
// own. A name somebody was also known by is its words whole, as a NAME is.
func TestANameTakenIsAnotherName(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Lady Maud /Percy/ jr.
2 NPFX Lady
2 _MARNM de Neville
3 SOUR @S1@
2 _MARN Lucas
2 _MAR Maud /Lane/
2 _ADPN Ferrers
2 _AKA The Fair
2 _AKAN Maudie
2 _MARNM ?
0 @S1@ SOUR
1 TITL Register
`)
	person := corpus.People["percy-maud"]
	if person == nil {
		t.Fatalf("people are %v, notes %v", corpus.PeopleSlugs(), notes)
	}
	type said struct{ name, typ string }
	var got []said
	for _, name := range person.AlsoKnownAs {
		got = append(got, said{name.String(), name.Type})
	}
	want := []said{
		{"Lady Maud de Neville jr.", "married"},
		{"Lady Maud Lucas jr.", "married"},
		{"Maud Lane", "married"},
		{"Lady Maud Ferrers jr.", "Adopted"},
		{"The Fair", "aka"},
		{"Maudie", "aka"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("also known as\n%q\nwant\n%q", got, want)
	}
	married := person.AlsoKnownAs[0]
	if !reflect.DeepEqual(married.Surname, model.SurnameOf("de Neville")) || married.Text != "" || len(married.Sources) != 1 {
		t.Errorf("married name = %+v", married)
	}
	if aka := person.AlsoKnownAs[4]; aka.Text != "The Fair" || aka.HasParts() {
		t.Errorf("a name also known by was divided: %+v", aka)
	}
	if !reported(notes, "kept as written") || !strings.Contains(person.Name.Note, "2 _MARNM ?") {
		t.Errorf("a placeholder taken for a name was not kept: %v %q", notes, person.Name.Note)
	}
}

// ALIA is another record said to be the same person, or under @VOID@ somebody the file
// has no record of; a few dialects write a name on it instead. One pointing at nobody
// in the file is kept as written.
func TestAnAliasIsAClaimOfIdentity(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 ALIA @I2@
2 PHRASE the same signature
1 ALIA @VOID@
2 PHRASE the Giovanni of the 1851 census
1 ALIA Nanni /Rossi/
1 ALIA @I9@
0 @I2@ INDI
1 NAME Giovanni Battista /Rossi/
`)
	giovanni := corpus.People["rossi-giovanni"]
	want := []model.SameAs{{ID: "rossi-giovanni-battista", Note: "the same signature"}, {Name: "the Giovanni of the 1851 census"}}
	if !reflect.DeepEqual(giovanni.SameAs, want) {
		t.Errorf("same as %+v", giovanni.SameAs)
	}
	if len(giovanni.AlsoKnownAs) != 1 || giovanni.AlsoKnownAs[0].String() != "Nanni Rossi" {
		t.Errorf("also known as %+v", giovanni.AlsoKnownAs)
	}
	if !reported(notes, "an ALIA names an individual who is not in the file") ||
		!strings.Contains(giovanni.Body, "1 ALIA @I9@") {
		t.Errorf("the dangling ALIA was not kept: %v %q", notes, giovanni.Body)
	}
}

// ANCI and DESI say that a submitter is interested in somebody's ancestors or
// descendants, which is a task in the person's body.
func TestAnInterestInTheFamilyIsATask(t *testing.T) {
	corpus, notes := convert(t, `0 @U1@ SUBM
1 NAME Anna Bianchi
0 @U2@ SUBM
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 NOTE Born in Firenze.
1 ANCI @U1@
1 DESI @U2@
`)
	body := corpus.People["rossi-giovanni"].Body
	want := "Born in Firenze.\n\n- [ ] Research the ancestors (Anna Bianchi)\n- [ ] Research the descendants"
	if !strings.HasPrefix(body, want) {
		t.Errorf("body = %q, notes %v", body, notes)
	}
	if strings.Contains(prose(corpus), "ANCI") || strings.Contains(prose(corpus), "DESI") {
		t.Errorf("an interest read was kept as well:\n%s", prose(corpus))
	}
}

// NO says what did not happen, from when to when where it says, with its notes and what
// states it, and is no event.
func TestWhatDidNotHappenIsSaid(t *testing.T) {
	corpus, notes := convert(t, `0 @S1@ SOUR
1 TITL Naturalisation index
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 NO NATU
2 DATE FROM 1900 TO 1930
2 NOTE Searched the index under both spellings.
2 SOUR @S1@
1 FAMS @F1@
0 @F1@ FAM
1 HUSB @I1@
1 NO MARR
`)
	giovanni := corpus.People["rossi-giovanni"]
	if len(giovanni.Events) != 0 || len(giovanni.Not) != 1 {
		t.Fatalf("events %+v, not %+v, notes %v", giovanni.Events, giovanni.Not, notes)
	}
	not := giovanni.Not[0]
	if not.Type != "naturalisation" || not.Date.String() != "FROM 1900 TO 1930" ||
		not.Note != "Searched the index under both spellings." || len(not.Sources) != 1 {
		t.Errorf("not = %+v", not)
	}
	if family := corpus.Families["rossi-giovanni--unknown"]; len(family.Not) != 1 || family.Not[0].Type != "marriage" {
		t.Errorf("the family's not = %+v", family.Not)
	}
}

// An event carries everything said about it that the model holds: where it goes in
// order, who acted for it, the religion it was in, what kind of birth it was, where in
// the place it happened, the ways to reach whoever held it, and its own identifier.
func TestAnEventCarriesWhatIsSaidAboutIt(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 TYPE stillborn
2 SDATE 1850
2 AGNC Parrocchia di San Lorenzo
2 RELI Catholic
2 ADDR Piazza San Lorenzo 9
3 CITY Firenze
2 WWW https://example.org/sl
2 EMAIL parrocchia@example.org
2 PHON +39 055 111111
2 FAX +39 055 222222
2 UID 2a3c1e2c-1d6f-4b9e-9b0c-7d3e5f6a7b8c
`)
	birth := corpus.People["rossi-giovanni"].Events[0]
	if birth.Sort.String() != "1850" {
		t.Errorf("sort = %q", birth.Sort.String())
	}
	var attributes []string
	for _, a := range birth.Attributes {
		attributes = append(attributes, a.Type+": "+a.Value)
	}
	if want := []string{"classification: stillborn", "agency: Parrocchia di San Lorenzo", "religion: Catholic"}; !slices.Equal(attributes, want) {
		t.Errorf("attributes = %q", attributes)
	}
	if birth.Address.Lines != "Piazza San Lorenzo 9" || birth.Address.City != "Firenze" {
		t.Errorf("address = %+v", birth.Address)
	}
	want := []model.Link{{URL: "https://example.org/sl"}, {URL: "mailto:parrocchia@example.org"},
		{URL: "tel:+39 055 111111"}, {URL: "tel:+39 055 222222", Type: "fax"}}
	if !slices.Equal(birth.Links, want) {
		t.Errorf("links = %+v", birth.Links)
	}
	if got := birth.IDs["uid"]; len(got) != 1 || got[0].ID != "2a3c1e2c-1d6f-4b9e-9b0c-7d3e5f6a7b8c" {
		t.Errorf("ids = %+v", birth.IDs)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// UID, its older spelling _UID, and RIN, the number the exporting program filed a record
// under, are identifiers of the record elsewhere.
func TestTheExportersOwnIdentifiersAreIDs(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 _UID 6F2A9C1E0B7D4E3A8C5F1D2B3A4C5D6E
1 RIN 1042
0 @I2@ INDI
1 NAME Anna /Rossi/
1 UID 1b2c3d4e-0000-4000-8000-000000000001
`)
	giovanni := corpus.People["rossi-giovanni"]
	if giovanni.IDs["uid"][0].ID != "6F2A9C1E0B7D4E3A8C5F1D2B3A4C5D6E" || giovanni.IDs["rin"][0].ID != "1042" {
		t.Errorf("ids = %+v", giovanni.IDs)
	}
	if anna := corpus.People["rossi-anna"]; anna.IDs["uid"][0].ID != "1b2c3d4e-0000-4000-8000-000000000001" {
		t.Errorf("ids = %+v", anna.IDs)
	}
	if reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}
