package gramps

import (
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// convert reads the objects of a Gramps database written in body, in the version
// origins reads.
func convert(t *testing.T, body string) (*model.Corpus, []string) {
	t.Helper()
	corpus, _, notes := convertFinding(t, body, nil)
	return corpus, notes
}

// convertFinding converts as convert does, finding the files the database names with
// find.
func convertFinding(t *testing.T, body string, find importing.Finder) (*model.Corpus, []importing.Attachment, []string) {
	t.Helper()
	root := element(t, `<?xml version="1.0" encoding="UTF-8"?>
<database xmlns="http://gramps-project.org/xml/1.7.2/">
  <header><created date="2026-10-05" version="6.0.0"/></header>
`+body+`
</database>`)
	return Convert(&Export{Root: root, Version: "1.7.2"}, nil, find)
}

func reported(notes []string, part string) bool {
	return slices.ContainsFunc(notes, func(n string) bool { return strings.Contains(n, part) })
}

// apart is a piece of prose the import wrote, as what it says and what of the export it
// kept in it as written.
func apart(text string) (prose, kept string) {
	var said, blocks []string
	for {
		start := strings.Index(text, "```gramps\n")
		if start < 0 {
			break
		}
		said = append(said, text[:start])
		rest := text[start+len("```gramps\n"):]
		end := strings.Index(rest, "\n```")
		if end < 0 {
			end = len(rest)
		}
		blocks = append(blocks, rest[:end])
		text = strings.TrimPrefix(rest[end:], "\n```")
	}
	said = append(said, text)
	return strings.TrimSpace(strings.Join(said, "")), strings.Join(blocks, "\n")
}

// keeps reports whether a piece of prose keeps each of written as written.
func keeps(t *testing.T, what, text string, written ...string) {
	t.Helper()
	_, kept := apart(text)
	for _, w := range written {
		if !strings.Contains(kept, w) {
			t.Errorf("%s does not keep %q: %q", what, w, text)
		}
	}
}

func person(t *testing.T, corpus *model.Corpus, slug model.Slug) *model.Person {
	t.Helper()
	p := corpus.People[slug]
	if p == nil {
		t.Fatalf("no %s among %v", slug, corpus.PeopleSlugs())
	}
	return p
}

// names writes the people an event names as person, role and age, for comparing.
func names(with []model.Named) []string {
	var out []string
	for _, n := range with {
		out = append(out, strings.TrimSpace(string(n.Person)+" "+n.Role+" "+string(n.Age)))
	}
	return out
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

func note(t *testing.T, corpus *model.Corpus, slug model.Slug) *model.Note {
	t.Helper()
	n := corpus.Notes[slug]
	if n == nil {
		t.Fatalf("no note %s among %v", slug, slices.Sorted(maps.Keys(corpus.Notes)))
	}
	return n
}

func place(t *testing.T, corpus *model.Corpus, slug model.Slug) *model.Place {
	t.Helper()
	p := corpus.Places[slug]
	if p == nil {
		t.Fatalf("no %s among %v", slug, corpus.PlaceSlugs())
	}
	return p
}

// The mapping the package exists for, on the smallest export that shows it: a person
// with a name, a sex, an identifier and an event at a place inside another.
func TestAPersonComesInWithTheirEventsAndPlaces(t *testing.T) {
	corpus, notes := convert(t, `
  <events>
    <event handle="_e1" change="1" id="E0001">
      <type>Birth</type>
      <dateval val="1834-03-12"/>
      <place hlink="_p1"/>
      <description>Birth of Rossi, Giovanni</description>
    </event>
    <event handle="_e2" change="1" id="E0002">
      <type>Occupation</type>
      <datespan start="1860" stop="1880"/>
      <description>Mezzadro</description>
    </event>
  </events>
  <people>
    <person handle="_i1" change="1" id="I0001">
      <gender>M</gender>
      <name type="Birth Name">
        <first>Giovanni</first>
        <surname>Rossi</surname>
      </name>
      <eventref hlink="_e1" role="Primary"/>
      <eventref hlink="_e2"/>
    </person>
  </people>
  <places>
    <placeobj handle="_p1" change="1" id="P0001" type="Town">
      <ptitle>Borgo San Lorenzo, Toscana</ptitle>
      <pname value="Borgo San Lorenzo"/>
      <coord long="11.3897" lat="43.9551"/>
      <placeref hlink="_p2"/>
    </placeobj>
    <placeobj handle="_p2" change="1" id="P0002" type="Unknown">
      <pname value="Toscana"/>
    </placeobj>
  </places>`)
	// a title that only repeats the hierarchy says nothing, and nothing else is left
	if len(notes) != 0 {
		t.Errorf("reported %q", notes)
	}

	giovanni := person(t, corpus, "rossi-giovanni")
	if giovanni.Name.Given != "Giovanni" || giovanni.Name.PrimarySurname().Name != "Rossi" || giovanni.Name.Type != "birth" {
		t.Errorf("name = %+v", giovanni.Name)
	}
	if giovanni.Sex.Value != model.Male || idAt(giovanni.IDs, "gramps") != "I0001" {
		t.Errorf("sex = %v, ids = %v", giovanni.Sex.Value, giovanni.IDs)
	}
	if len(giovanni.Events) != 2 {
		t.Fatalf("events = %+v", giovanni.Events)
	}
	birth, occupation := giovanni.Events[0], giovanni.Events[1]
	if birth.Type != "birth" || birth.Date.String() != "12 MAR 1834" || birth.Place != "borgo-san-lorenzo" {
		t.Errorf("birth = %+v", birth)
	}
	// a description is the value of an event as of a fact held, Gramps having one field
	if note, _ := apart(birth.Note); note != "" || birth.Value != "Birth of Rossi, Giovanni" {
		t.Errorf("birth's description became %q, %q", birth.Note, birth.Value)
	}
	if occupation.Type != "occupation" || occupation.Value != "Mezzadro" || occupation.Date.String() != "FROM 1860 TO 1880" {
		t.Errorf("occupation = %+v", occupation)
	}

	borgo := place(t, corpus, "borgo-san-lorenzo")
	if borgo.Name.Text != "Borgo San Lorenzo" || borgo.Type != "town" || idAt(borgo.IDs, "gramps") != "P0001" {
		t.Errorf("place = %+v", borgo)
	}
	if len(borgo.Within) != 1 || borgo.Within[0].Place != "toscana" {
		t.Errorf("within = %+v", borgo.Within)
	}
	if borgo.Coordinates == nil || *borgo.Coordinates != (model.Coordinates{Latitude: 43.9551, Longitude: 11.3897}) {
		t.Errorf("coordinates = %v", borgo.Coordinates)
	}
	if toscana := place(t, corpus, "toscana"); toscana.Type != "" {
		t.Errorf("an Unknown type became %q", toscana.Type)
	}

	out := t.TempDir()
	if err := store.Write(out, corpus); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(out); err != nil {
		t.Errorf("what the import wrote does not read back: %v", err)
	}
}

// One event in Gramps is shared by everybody in it, each in a role. Each person it is
// about has a copy, with what their own reference adds, and everybody else is named in
// every copy.
func TestASharedEventIsCopiedToWhoeverItIsAboutAndNamesTheRest(t *testing.T) {
	corpus, notes := convert(t, `
  <events>
    <event handle="_census" change="1" id="E0001">
      <type>Census</type>
      <dateval val="1881-04-03"/>
    </event>
    <event handle="_baptism" change="1" id="E0002">
      <type>Baptism</type>
      <dateval val="1850-06-02"/>
      <attribute type="Time" value="10:00"/>
    </event>
    <event handle="_wedding" change="1" id="E0003">
      <type>Marriage</type>
      <dateval val="1872"/>
      <attribute type="Witness" value="Giulio Bianchi"/>
      <attribute type="Time" value="11:00"/>
    </event>
    <event handle="_family" change="1" id="E0004">
      <type>Marriage</type>
      <dateval val="1875"/>
    </event>
    <event handle="_death" change="1" id="E0005">
      <type>Death</type>
      <attribute type="Cause" value="consumption"/>
      <attribute type="_UID" value="0123456789ABCDEF"/>
      <attribute type="Father Age" value="61"/>
    </event>
    <event handle="_orphan" change="1" id="E0006">
      <type>Burial</type>
    </event>
    <event handle="_stamp" change="1" id="E0007">
      <type>_UPD</type>
      <description>14 JAN 2001 12:00:00 GMT -0500</description>
    </event>
  </events>
  <people>
    <person handle="_anna" change="1" id="I0001">
      <gender>F</gender>
      <name><first>Anna</first><surname>Rossi</surname></name>
      <eventref hlink="_census" role="Primary">
        <attribute type="Age" value="30"/>
      </eventref>
      <eventref hlink="_baptism" role="Primary"/>
      <eventref hlink="_wedding" role="Bride"/>
      <eventref hlink="_death" role="Primary" priv="1"/>
      <eventref hlink="_stamp" role="Primary"/>
      <eventref hlink="_family" role="Family"/>
    </person>
    <person handle="_carlo" change="1" id="I0002">
      <gender>M</gender>
      <name><first>Carlo</first><surname>Rossi</surname></name>
      <eventref hlink="_census" role="Primary">
        <attribute type="Age" value="32"/>
        <attribute type="Relationship to head" value="Head"/>
      </eventref>
      <eventref hlink="_wedding" role="Groom"/>
    </person>
    <person handle="_pietro" change="1" id="I0003">
      <gender>M</gender>
      <name><first>Pietro</first><surname>Neri</surname></name>
      <eventref hlink="_baptism" role="Godparent"/>
      <eventref hlink="_wedding" role="Witness" priv="1">
        <attribute type="Age" value="40"/>
      </eventref>
      <eventref hlink="_family" role="Witness"/>
      <eventref hlink="_orphan" role="Informant">
        <attribute type="Age" value="about forty"/>
      </eventref>
    </person>
  </people>
  <families>
    <family handle="_f1" change="1" id="F0001">
      <rel type="Married"/>
      <father hlink="_carlo"/>
      <mother hlink="_anna"/>
      <eventref hlink="_family" role="Family"/>
    </family>
  </families>`)

	anna, carlo := person(t, corpus, "rossi-anna"), person(t, corpus, "rossi-carlo")
	types := func(p *model.Person) []string {
		var out []string
		for _, e := range p.Events {
			out = append(out, e.Type)
		}
		return out
	}
	// the marriage is the family's, the update stamp nobody's
	if got := types(anna); !slices.Equal(got, []string{"census", "baptism", "marriage", "death"}) {
		t.Errorf("Anna's events are %v", got)
	}
	if got := types(carlo); !slices.Equal(got, []string{"census", "marriage"}) {
		t.Errorf("Carlo's events are %v", got)
	}
	if len(person(t, corpus, "neri-pietro").Events) != 0 {
		t.Errorf("a witness was given a copy: %+v", corpus.People["neri-pietro"].Events)
	}

	// each copy carries its holder's own age
	if anna.Events[0].Age.Value != "30" || carlo.Events[0].Age.Value != "32" || len(anna.Events[0].With) != 0 {
		t.Errorf("census copies = %+v and %+v", anna.Events[0], carlo.Events[0])
	}
	baptism := anna.Events[1]
	if got := names(baptism.With); !slices.Equal(got, []string{"neri-pietro godparent"}) {
		t.Errorf("the baptism names %+v", baptism.With)
	}
	// the bride and the groom of a marriage no family holds each have it
	wedding := anna.Events[2]
	if got := names(wedding.With); !slices.Equal(got, []string{"witness", "neri-pietro witness 40"}) {
		t.Errorf("the wedding names %+v", wedding.With)
	}
	if carlo.Events[1].Date.String() != "1872" || len(carlo.Events[1].With) != 2 {
		t.Errorf("Carlo's copy of the wedding is %+v", carlo.Events[1])
	}
	// a private reference makes that copy private, and nobody else's
	death := anna.Events[3]
	if !death.Private || death.Cause.Value != "consumption" {
		t.Errorf("death = %+v", death)
	}
	if carlo.Events[0].Private || anna.Events[0].Private {
		t.Error("a copy was made private by another's reference")
	}

	// every copy is the one event, by its identifier
	if idAt(anna.Events[0].IDs, "gramps") != "E0001" || idAt(carlo.Events[0].IDs, "gramps") != "E0001" ||
		idAt(death.IDs, "gramps") != "E0005" || idAt(death.IDs, "uid") != "0123456789ABCDEF" {
		t.Errorf("the copies are identified as %v, %v and %v", anna.Events[0].IDs, carlo.Events[0].IDs, death.IDs)
	}

	// an attribute is a field where the field can hold it, and an attribute otherwise
	if baptism.Date.String() != "2 JUN 1850 10:00" {
		t.Errorf("the baptism is dated %q", baptism.Date)
	}
	if wedding.Date.String() != "1872" || len(wedding.Attributes) != 1 || wedding.Attributes[0].Value != "11:00" {
		t.Errorf("a time on a date without a day became %q and %+v", wedding.Date, wedding.Attributes)
	}
	if len(wedding.With) != 2 || wedding.With[0].Name.Text != "Giulio Bianchi" || wedding.With[0].Role != "witness" {
		t.Errorf("the witness by name is %+v", wedding.With)
	}
	if want := []model.Attribute{{Type: "fathers-age", Value: "61"}}; !slices.EqualFunc(death.Attributes, want, sameAttribute) {
		t.Errorf("the death's attributes are %+v", death.Attributes)
	}
	if want := []model.Attribute{{Type: "Relationship to head", Value: "Head"}}; !slices.EqualFunc(carlo.Events[0].Attributes, want, sameAttribute) {
		t.Errorf("Carlo's census attributes are %+v", carlo.Events[0].Attributes)
	}

	// an event nobody holds is a note's, still naming whoever points at it; the stamp
	// of an update is dropped
	burial := note(t, corpus, "burial")
	if burial.Title != "Burial" || len(burial.Events) != 1 || burial.Events[0].Type != "burial" ||
		idAt(burial.Events[0].IDs, "gramps") != "E0006" {
		t.Fatalf("the event nobody holds is %+v", burial)
	}
	if with := burial.Events[0].With; len(with) != 1 || with[0].Person != "neri-pietro" ||
		with[0].Role != "informant" || len(with[0].Attributes) != 1 || with[0].Attributes[0].Value != "about forty" {
		t.Errorf("the event nobody holds names %+v", with)
	}
	if _, ok := corpus.Notes["event-e0007"]; ok {
		t.Error("the stamp of an update was kept")
	}
	if !reported(notes, "priv on <eventref> was kept as written") {
		t.Errorf("a private reference naming somebody was not kept: %q", notes)
	}
	if reported(notes, "<attribute>") || reported(notes, "<event>") {
		t.Errorf("an attribute or an event was kept: %q", notes)
	}
}

func sameAttribute(a, b model.Attribute) bool {
	return a.Type == b.Type && a.Value == b.Value && a.Date.String() == b.Date.String() &&
		a.Note == b.Note && slices.Equal(a.Sources, b.Sources)
}

// A name is in parts, several surnames joined into the one the model has, and the first
// name is the one the person is filed under.
func TestNamesComeInWithTheirPartsTypesAndDates(t *testing.T) {
	corpus, notes := convert(t, `
  <people>
    <person handle="_i1" change="1" id="I0001">
      <gender>F</gender>
      <name type="Birth Name" display="2" sort="-1">
        <first>María Dolores</first>
        <call>Lola</call>
        <surname prefix="de" connector="y">la Cruz</surname>
        <surname prim="0" prefix="del" derivation="Matrilineal">Monte</surname>
        <suffix>Jr.</suffix>
        <title>Doña</title>
        <nick>Loli</nick>
        <familynick>Cruces</familynick>
      </name>
      <name alt="1" type="Married Name" display="-7">
        <first>María Dolores</first>
        <surname prefix="de">García</surname>
        <datespan start="1890" stop="1920"/>
      </name>
      <name alt="1" type="Also Known As" priv="1">
        <first>Lola</first>
        <surname>Montes</surname>
        <dateval val="1900" type="after"/>
      </name>
      <name alt="1" type="Religious">
        <first>Maria</first>
        <dateval val="1925"/>
      </name>
      <name alt="1" type="Unknown">
        <first>Dolores</first>
        <surname/>
        <surname derivation="Unknown ">Cruces</surname>
      </name>
    </person>
    <person handle="_i2" change="1" id="I0002">
      <gender>U</gender>
      <name priv="1"><first>Ignota</first></name>
    </person>
    <person handle="_i3" change="1" id="I0003">
      <gender>X</gender>
    </person>
    <person handle="_i4" change="1" id="I0004">
      <name>
        <first>Ivan</first>
        <surname derivation="Patronymic" prim="0">Petrovich</surname>
        <surname derivation="Familial">Sidorov</surname>
        <group>Sidorovs</group>
      </name>
    </person>
    <person handle="_i5" change="1" id="I0005">
      <name><first>Anna</first><surname>Smyth</surname></name>
      <name alt="1"><first>Anna</first><surname>Smithe</surname><group>Smithe</group></name>
    </person>
  </people>
  <name-formats>
    <format number="-1" name="Shouted" fmt_str="%L, %f (%x) %0m" active="1"/>
    <format number="-2" name="Unused" fmt_str="%f" active="0"/>
  </name-formats>
  <namemaps>
    <map type="group_as" key="Smyth" value="Smith"/>
    <map type="group_as" key="Brown" value="Braun"/>
  </namemaps>`)

	maria := person(t, corpus, "la-cruz-maria-dolores")
	want := model.PersonName{
		Prefix: "Doña", Given: "María Dolores", Call: "Lola", Nickname: "Loli",
		Surname: model.Surnames{
			{Name: "la Cruz", Prefix: "de", Connector: "y"},
			{Name: "Monte", Prefix: "del", Origin: "matrilineal"},
		},
		Suffix: "Jr.", Type: "birth",
	}
	if !sameName(maria.Name, want) {
		t.Errorf("name = %+v", maria.Name)
	}
	if maria.Name.FamilyNickname != "Cruces" {
		t.Errorf("family nickname = %q", maria.Name.FamilyNickname)
	}
	if maria.Name.Display != "{given} {surname} {suffix}" || maria.Name.Sort != "{SURNAME}, {given} ({common}) %0m" {
		t.Errorf("display %q and sort %q", maria.Name.Display, maria.Name.Sort)
	}
	if got := maria.Name.String(); got != "María Dolores de la Cruz y del Monte Jr." {
		t.Errorf("shown as %q", got)
	}
	if maria.Private {
		t.Error("a private other name made the person private")
	}
	if len(maria.AlsoKnownAs) != 4 {
		t.Fatalf("other names = %+v", maria.AlsoKnownAs)
	}
	married, aka, religious, unknown := maria.AlsoKnownAs[0], maria.AlsoKnownAs[1], maria.AlsoKnownAs[2], maria.AlsoKnownAs[3]
	if married.Type != "married" || married.From.String() != "1890" || married.Until.String() != "1920" {
		t.Errorf("married name = %+v", married)
	}
	if aka.Type != "aka" || !aka.Private || aka.From.String() != "1900" || !aka.Until.IsZero() {
		t.Errorf("aka = %+v", aka)
	}
	if religious.Type != "Religious" || religious.From.String() != "1925" || religious.Until.String() != "1925" {
		t.Errorf("religious name = %+v", religious)
	}
	if unknown.Type != "" || !slices.Equal(unknown.Surname, model.SurnameOf("Cruces")) {
		t.Errorf("an Unknown type and origin became %q and %+v", unknown.Type, unknown.Surname)
	}
	if married.Display != "" || married.SurnamePrefix != "de" || married.Surname.String() != "García" {
		t.Errorf("married name = %+v", married)
	}

	// the primary surname is marked where it is not the first, and a name's own group
	// stands over the name maps'
	ivan := person(t, corpus, "sidorov-ivan")
	if want := (model.Surnames{
		{Name: "Petrovich", Origin: "patronymic"},
		{Name: "Sidorov", Origin: "Familial", Primary: true},
	}); !slices.Equal(ivan.Name.Surname, want) || ivan.Name.Group != "Sidorovs" {
		t.Errorf("Ivan's name = %+v", ivan.Name)
	}
	anna := person(t, corpus, "smyth-anna")
	if anna.Name.Group != "Smith" || anna.AlsoKnownAs[0].Group != "Smithe" {
		t.Errorf("Anna's groups are %q and %q", anna.Name.Group, anna.AlsoKnownAs[0].Group)
	}

	// the name a person is filed under cannot be hidden alone
	ignota := person(t, corpus, "ignota")
	if !ignota.Private || ignota.Name.Private || ignota.Sex.Value != model.SexUnknown {
		t.Errorf("Ignota = %+v", ignota)
	}
	nameless := person(t, corpus, "i0003")
	if nameless.Sex.Value != model.Neither || nameless.Name.Text != "i0003" {
		t.Errorf("the nameless person = %+v", nameless)
	}

	// a format nothing defines is kept on the name, and what nobody used of the file's
	// formats and maps in a note record of its own
	keeps(t, "the married name", married.Note, `display="-7"`)
	formats := note(t, corpus, "name-formats")
	keeps(t, "the unused formats", formats.Body, `name="Unused"`)
	if strings.Contains(formats.Body, "Shouted") {
		t.Errorf("a format used was kept as well: %s", formats.Body)
	}
	maps := note(t, corpus, "namemaps")
	keeps(t, "the unused maps", maps.Body, `key="Brown"`)
	if strings.Contains(maps.Body, "Smyth") {
		t.Errorf("a map used was kept as well: %s", maps.Body)
	}
	for _, want := range []string{
		"a person had no name and is filed under their identifier",
		"a name gave no surname",
	} {
		if !reported(notes, want) {
			t.Errorf("did not report %q in %q", want, notes)
		}
	}
	for _, unwanted := range []string{"prim on", "derivation on", "<familynick>", "<group>", "sort on"} {
		if reported(notes, unwanted) {
			t.Errorf("reported %q: %q", unwanted, notes)
		}
	}
}

func sameName(a, b model.PersonName) bool {
	return a.Prefix == b.Prefix && a.Given == b.Given && a.Call == b.Call &&
		a.Nickname == b.Nickname && a.SurnamePrefix == b.SurnamePrefix && slices.Equal(a.Surname, b.Surname) &&
		a.Suffix == b.Suffix && a.Type == b.Type && a.Private == b.Private
}

// A person's attributes are facts held, an identifier or a nickname; an address is a
// residence; a personref is an association.
func TestAttributesAddressesAndAssociationsComeIn(t *testing.T) {
	corpus, notes := convert(t, `
  <people>
    <person handle="_i1" change="1" id="I0001">
      <gender>M</gender>
      <name><first>Carlo</first><surname>Rossi</surname></name>
      <lds_ord type="sealed_to_parents" priv="1">
        <dateval val="1990-05-02"/>
        <temple val="SLAKE"/>
        <status val="DNS/CAN"/>
        <sealed_to hlink="_f1"/>
      </lds_ord>
      <lds_ord type="baptism">
        <status val="Pre-1970"/>
      </lds_ord>
      <address>
        <dateval val="1881"/>
        <street>Via Roma 3</street>
        <city>Firenze</city>
        <country>Italia</country>
        <phone>055 123</phone>
      </address>
      <attribute type="Occupation" value="Mezzadro"/>
      <attribute type="National Origin" value="Italian"/>
      <attribute type="Identification Number" value="RSSCRL"/>
      <attribute type="Nickname" value="Carlino"/>
      <attribute type="Shoe size" value="43" priv="1"/>
      <attribute type="REFN" value="17"/>
      <attribute type="REFN" value="18"/>
      <attribute type="_FSFTID" value="LZ12-3AB"/>
      <attribute type="_UID" value="0123456789ABCDEF"/>
      <attribute type="RIN" value="42"/>
      <attribute type="Merged Gramps ID" value="I0099"/>
      <personref hlink="_i2" rel="Godfather"/>
      <personref hlink="_i2" rel="" priv="1"/>
      <url href="https://example.org/carlo" type="Web Home" description="His page"/>
      <url href="carlo@example.org" type="E-mail"/>
      <url href="mailto:rossi@example.org" type="E-mail"/>
      <url href="ftp://example.org/" type="Unknown"/>
      <url href="https://example.org/blog" type="Blog"/>
      <tagref hlink="_t1"/>
    </person>
    <person handle="_i2" change="1" id="I0002">
      <gender>M</gender>
      <name><first>Pietro</first><surname>Neri</surname><nick>Pierino</nick></name>
      <attribute type="Nickname" value="Piero"/>
      <url href="https://example.org/pietro" priv="1"/>
      <shoesize>43</shoesize>
    </person>
    <person handle="_i3" change="1" id="I0003">
      <gender>F</gender>
      <name><first>Lucia</first><surname>Neri</surname></name>
      <attribute type="Nickname" value="Lucy" priv="1"/>
    </person>
  </people>
  <families>
    <family handle="_f1" change="1" id="F0001">
      <father hlink="_i2"/>
    </family>
  </families>
  <tags>
    <tag handle="_t1" change="1" name="emigrant" color="#ff0000000000" priority="0"/>
  </tags>`)

	carlo := person(t, corpus, "rossi-carlo")
	if carlo.Name.Nickname != "Carlino" {
		t.Errorf("a Nickname did not become the name's nickname: %+v", carlo.Name)
	}
	if idAt(carlo.IDs, "gramps") != "I0001" || idAt(carlo.IDs, "refn") != "17" || idAt(carlo.IDs, "familysearch") != "LZ12-3AB" ||
		idAt(carlo.IDs, "uid") != "0123456789ABCDEF" || idAt(carlo.IDs, "rin") != "42" {
		t.Errorf("ids = %v", carlo.IDs)
	}
	// an identifier a merge left is one the record had, after its own
	if ids := carlo.IDs["gramps"]; len(ids) != 2 || ids[1].ID != "I0099" {
		t.Errorf("Gramps ids = %v", ids)
	}
	want := []model.Event{
		{Type: "lds-sealing-to-parents", Private: true},
		{Type: "lds-baptism"},
		{Type: "occupation", Value: "Mezzadro"},
		{Type: "nationality", Value: "Italian"},
		{Type: "identifying-number", Value: "RSSCRL"},
		{Type: "Shoe size", Value: "43", Private: true},
		{Type: "residence"},
		{Type: "association", With: []model.Named{{Person: "neri-pietro", Role: "Godfather"}}},
		{Type: "association", Private: true, With: []model.Named{{Person: "neri-pietro", Role: "associate"}}},
	}
	if len(carlo.Events) != len(want) {
		t.Fatalf("events = %+v", carlo.Events)
	}
	for i, got := range carlo.Events {
		if got.Type != want[i].Type || got.Value != want[i].Value || got.Private != want[i].Private ||
			!slices.Equal(names(got.With), names(want[i].With)) {
			t.Errorf("event %d = %+v, want %+v", i, got, want[i])
		}
	}

	// a rite's temple and status are attributes, and the family it seals to its family
	sealing := carlo.Events[0]
	if sealing.Date.String() != "2 MAY 1990" || sealing.Family != "neri-pietro--unknown" ||
		!slices.EqualFunc(sealing.Attributes, []model.Attribute{
			{Type: "temple", Value: "SLAKE"}, {Type: "status", Value: "do-not-seal-canceled"},
		}, sameAttribute) {
		t.Errorf("the sealing = %+v", sealing)
	}
	if attrs := carlo.Events[1].Attributes; len(attrs) != 1 || attrs[0].Value != "pre-1970" {
		t.Errorf("the baptism's attributes are %+v", attrs)
	}

	// an address is where the residence was, its telephone a link
	residence := carlo.Events[6]
	if residence.Date.String() != "1881" || residence.Address.Street != "Via Roma 3" ||
		residence.Address.City != "Firenze" || residence.Address.Country != "Italia" ||
		len(residence.Address.Links) != 1 || residence.Address.Links[0].URL != "tel:055 123" {
		t.Errorf("the residence = %+v", residence)
	}

	// a URL is a link, an e-mail written as one, a type Gramps names a slug
	if want := []model.Link{
		{URL: "https://example.org/carlo", Type: "web-home", Title: "His page"},
		{URL: "mailto:carlo@example.org", Type: "e-mail"},
		{URL: "mailto:rossi@example.org", Type: "e-mail"},
		{URL: "ftp://example.org/"},
		{URL: "https://example.org/blog", Type: "Blog"},
	}; !slices.Equal(carlo.Links, want) {
		t.Errorf("links = %+v", carlo.Links)
	}
	if !slices.Equal(carlo.Tags, []string{"emigrant"}) {
		t.Errorf("tags = %v", carlo.Tags)
	}

	// a nickname where the name has one already, or that says something of itself, is a
	// fact like any other; and a private link makes private what it is a link of
	pietro := person(t, corpus, "neri-pietro")
	if pietro.Name.Nickname != "Pierino" || len(pietro.Events) != 1 || pietro.Events[0].Value != "Piero" {
		t.Errorf("Pietro = %+v", pietro)
	}
	if !pietro.Private || len(pietro.Links) != 1 {
		t.Errorf("Pietro's private link made him %v, with links %v", pietro.Private, pietro.Links)
	}
	lucia := person(t, corpus, "neri-lucia")
	if lucia.Name.Nickname != "" || len(lucia.Events) != 1 || !lucia.Events[0].Private {
		t.Errorf("Lucia = %+v", lucia)
	}

	// a second identifier at one service, and an element no code reads, come back as
	// written
	keeps(t, "Carlo", carlo.Body, `<attribute type="REFN" value="18"/>`)
	keeps(t, "Pietro", pietro.Body, "<shoesize>43</shoesize>")
	for _, want := range []string{
		"<attribute> was kept as written", "<shoesize> was kept as written",
		"a private link made private the record it is a link of",
	} {
		if !reported(notes, want) {
			t.Errorf("did not report %q in %q", want, notes)
		}
	}
	for _, unwanted := range []string{"<lds_ord>", "<phone>", "<tagref>", "<url>", "<temple>", "<status>", "<sealed_to>"} {
		if reported(notes, unwanted) {
			t.Errorf("reported %s: %q", unwanted, notes)
		}
	}
}

// A place is a hierarchy with dates on it, other names in other languages, a type and
// coordinates in whatever notation they were typed.
func TestPlacesComeInAsTheHierarchyTheyAre(t *testing.T) {
	corpus, notes := convert(t, `
  <places>
    <placeobj handle="_p1" change="1" id="P0001" type="City">
      <ptitle>Firenze</ptitle>
      <code>50100</code>
      <pname value="Firenze">
        <dateval val="1200" type="after"/>
      </pname>
      <pname lang="en" value="Florence"/>
      <pname lang="DE" value="Florenz">
        <datespan start="1800" stop="1900"/>
      </pname>
      <pname lang="abbr" value="FI"/>
      <coord long="11°15′E" lat="43° 46.2′ N"/>
      <placeref hlink="_toscana">
        <datespan start="1569" stop="1859"/>
      </placeref>
      <placeref hlink="_italia">
        <dateval val="1861" type="after"/>
      </placeref>
      <location city="Florence" parish="San Lorenzo" phone="055 1"/>
      <url href="https://it.wikipedia.org/wiki/Firenze"/>
      <tagref hlink="_t1"/>
    </placeobj>
    <placeobj handle="_toscana" change="1" id="P0002" type="Grand Duchy" priv="1">
      <pname value="Toscana"/>
      <coord long="11.0" lat="91.5"/>
    </placeobj>
    <placeobj handle="_italia" change="1" id="P0003" type="Country">
      <pname value="Italia"/>
      <coord long="12.5W" lat="-41.9"/>
    </placeobj>
    <placeobj handle="_sg1" change="1" id="P0004" type="Village">
      <ptitle>San Giovanni in Val d'Arno, Toscana</ptitle>
      <pname value="San Giovanni"/>
      <placeref hlink="_toscana"/>
    </placeobj>
    <placeobj handle="_sg2" change="1" id="P0005" type="Village">
      <pname value="San Giovanni"/>
      <placeref hlink="_italia"/>
    </placeobj>
    <placeobj handle="_untitled" change="1" id="P0006" type="Farm">
      <ptitle>Podere Le Lame</ptitle>
    </placeobj>
  </places>
  <tags>
    <tag handle="_t1" change="1" name="visited" color="#000000000000" priority="0"/>
  </tags>`)

	firenze := place(t, corpus, "firenze")
	if firenze.Type != "city" || len(firenze.AlsoKnownAs) != 3 {
		t.Fatalf("Firenze = %+v", firenze)
	}
	// the first name's dates are the name's, and a title that is only the name says nothing
	if firenze.Name.Text != "Firenze" || firenze.Name.From.String() != "1200" || !firenze.Name.Until.IsZero() {
		t.Errorf("Firenze's name = %+v", firenze.Name)
	}
	if idAt(firenze.IDs, "code") != "50100" || !slices.Equal(firenze.Tags, []string{"visited"}) ||
		!slices.Equal(firenze.Links, model.LinksOf("https://it.wikipedia.org/wiki/Firenze")) {
		t.Errorf("Firenze's code, tags and links are %v, %v and %v", firenze.IDs, firenze.Tags, firenze.Links)
	}
	if a := firenze.Addresses; len(a) != 1 || a[0].City != "Florence" || a[0].Parish != "San Lorenzo" ||
		!slices.Equal(a[0].Links, model.LinksOf("tel:055 1")) {
		t.Errorf("Firenze's addresses are %+v", a)
	}
	florence, florenz, abbreviated := firenze.AlsoKnownAs[0], firenze.AlsoKnownAs[1], firenze.AlsoKnownAs[2]
	if florence.Text != "Florence" || florence.Lang != "en" || !florence.From.IsZero() {
		t.Errorf("Florence = %+v", florence)
	}
	if florenz.Lang != "de" || florenz.From.String() != "1800" || florenz.Until.String() != "1900" {
		t.Errorf("Florenz = %+v", florenz)
	}
	if abbreviated.Text != "FI" || abbreviated.Lang != "" {
		t.Errorf("FI = %+v", abbreviated)
	}
	wantWithin := []string{"toscana FROM:1569 UNTIL:1859", "italia FROM:1861 UNTIL:"}
	var within []string
	for _, ref := range firenze.Within {
		within = append(within, string(ref.Place)+" FROM:"+ref.From.String()+" UNTIL:"+ref.Until.String())
	}
	if !slices.Equal(within, wantWithin) {
		t.Errorf("within = %q", within)
	}
	if c := firenze.Coordinates; c == nil || !near(c.Latitude, 43.77) || !near(c.Longitude, 11.25) {
		t.Errorf("coordinates = %+v", c)
	}

	toscana := place(t, corpus, "toscana")
	if !toscana.Private || toscana.Type != "Grand Duchy" ||
		!reflect.DeepEqual(toscana.Coordinates, &model.Coordinates{Wording: "91.5 11.0"}) {
		t.Errorf("Toscana = %+v", toscana)
	}
	if c := place(t, corpus, "italia").Coordinates; c == nil || c.Latitude != -41.9 || c.Longitude != -12.5 {
		t.Errorf("Italia's coordinates = %+v", c)
	}
	// two places of one name are told apart by what they lie within
	if sg := place(t, corpus, "san-giovanni"); len(sg.AlsoKnownAs) != 1 || sg.AlsoKnownAs[0].Text != "San Giovanni in Val d'Arno, Toscana" {
		t.Errorf("a title that says more than the hierarchy is not another name: %+v", sg.AlsoKnownAs)
	}
	if place(t, corpus, "san-giovanni").Within[0].Place != "toscana" ||
		place(t, corpus, "san-giovanni-italia").Within[0].Place != "italia" {
		t.Errorf("places are %v", corpus.PlaceSlugs())
	}
	if place(t, corpus, "podere-le-lame").Name.Text != "Podere Le Lame" {
		t.Error("a place with no name did not fall back on its title")
	}

	keeps(t, "FI", abbreviated.Note, `lang="abbr"`)
	if want := []string{"lang on <pname> was kept as written"}; !slices.Equal(notes, want) {
		t.Errorf("reported %q", notes)
	}
}

func TestCoordinatesAreReadInEveryNotationGrampsAccepts(t *testing.T) {
	for _, tc := range []struct {
		text      string
		longitude bool
		want      float64
		valid     bool
	}{
		{"43.9551", false, 43.9551, true},
		{"-81.2636738", true, -81.2636738, true},
		{"52.3384N", false, 52.3384, true},
		{"0.9514W", true, -0.9514, true},
		{"78.38 W", true, -78.38, true},
		{"S 33.5", false, -33.5, true},
		{"42°19′30″N", false, 42.325, true},
		{"76° 31.5′ W", true, -76.525, true},
		{"42:30:00", false, 42.5, true},
		{"43,5", false, 43.5, true},
		{"95", false, 0, false},
		{"181", true, 0, false},
		{"42°75′N", false, 0, false},
		{"north", false, 0, false},
		{"", false, 0, false},
	} {
		got, ok := degrees(tc.text, 'N', 'S', 90)
		if tc.longitude {
			got, ok = degrees(tc.text, 'E', 'W', 180)
		}
		if ok != tc.valid || (ok && !near(got, tc.want)) {
			t.Errorf("degrees(%q) = %v, %v; want %v, %v", tc.text, got, ok, tc.want, tc.valid)
		}
	}
}

// A newer Gramps is read on the same terms as the one origins knows, and the report
// says that it was.
func TestANewerVersionIsNamedInTheReport(t *testing.T) {
	root := element(t, `<database xmlns="http://gramps-project.org/xml/1.8.0/"/>`)
	_, _, notes := Convert(&Export{Root: root, Version: "1.8.0", Newer: true}, nil, nil)
	if !reported(notes, "Gramps XML 1.8.0, newer than the 1.7.2 origins knows") {
		t.Errorf("notes = %q", notes)
	}
}

// A type or a role of Gramps' own is written in origins' words, and one somebody typed
// as they typed it, whatever its script or length.
func TestATypeSomebodyTypedIsKeptAsWritten(t *testing.T) {
	corpus, _ := convert(t, `
  <events>
    <event handle="_service" change="1" id="E0001"><type>兵役</type></event>
    <event handle="_war" change="1" id="E0002"><type>Military  Service (WWI)</type></event>
    <event handle="_title" change="1" id="E0003"><type>Nobility Title</type></event>
    <event handle="_baptism" change="1" id="E0004"><type>Baptism</type></event>
  </events>
  <people>
    <person handle="_anna" change="1" id="I0001">
      <name type="Nome da religiosa"><first>Anna</first><surname>Rossi</surname></name>
      <eventref hlink="_service" role="Primary"/>
      <eventref hlink="_war" role="Primary"/>
      <eventref hlink="_title" role="Primary"/>
      <eventref hlink="_baptism" role="Primary"/>
      <attribute type="Father Age" value="40"/>
      <attribute type="Statura" value="160"/>
    </person>
    <person handle="_pietro" change="1" id="I0002">
      <name><first>Pietro</first><surname>Neri</surname></name>
      <eventref hlink="_baptism" role="Padrino"/>
    </person>
    <person handle="_carlo" change="1" id="I0003">
      <name><first>Carlo</first><surname>Neri</surname></name>
      <eventref hlink="_baptism" role="Godparent"/>
    </person>
  </people>
  <families>
    <family handle="_f1" change="1" id="F0001">
      <rel type="Handfasting"/>
      <father hlink="_pietro"/>
      <mother hlink="_anna"/>
    </family>
  </families>
  <places>
    <placeobj handle="_p1" change="1" id="P0001" type="Comune">
      <pname value="Borgo San Lorenzo"/>
    </placeobj>
    <placeobj handle="_p2" change="1" id="P0002" type="Village">
      <pname value="Luco"/>
    </placeobj>
  </places>
  <repositories>
    <repository handle="_r1" change="1" id="R0001">
      <rname>Archivio di Stato di Firenze</rname>
      <type>Archivio di Stato</type>
    </repository>
  </repositories>`)

	anna := person(t, corpus, "rossi-anna")
	var types []string
	for _, e := range anna.Events {
		types = append(types, e.Type)
	}
	want := []string{"兵役", "Military Service (WWI)", "title", "baptism", "fathers-age", "Statura"}
	if !slices.Equal(types, want) {
		t.Errorf("types = %q, want %q", types, want)
	}
	if anna.Name.Type != "Nome da religiosa" {
		t.Errorf("name type = %q", anna.Name.Type)
	}
	if got := names(anna.Events[3].With); !slices.Equal(got, []string{"neri-pietro Padrino", "neri-carlo godparent"}) {
		t.Errorf("named at the baptism = %q", got)
	}
	if len(corpus.Families) != 1 || len(corpus.Repositories) != 1 {
		t.Fatalf("%d families and %d repositories", len(corpus.Families), len(corpus.Repositories))
	}
	for _, family := range corpus.Families {
		if family.Type != "Handfasting" {
			t.Errorf("family type = %q", family.Type)
		}
	}
	if got := place(t, corpus, "borgo-san-lorenzo").Type; got != "Comune" {
		t.Errorf("place type = %q", got)
	}
	if got := place(t, corpus, "luco").Type; got != "village" {
		t.Errorf("place type = %q", got)
	}
	for _, repository := range corpus.Repositories {
		if repository.Type != "Archivio di Stato" {
			t.Errorf("repository type = %q", repository.Type)
		}
	}
}

// idAt is the first identifier a record has at a service.
func idAt(ids model.IDs, service string) string {
	if len(ids[service]) == 0 {
		return ""
	}
	return ids[service][0].ID
}

// An element no code reads comes back in the corpus as written, with everything inside
// it, in the note of the nearest thing above it that takes one, or the body of its
// record, and survives the corpus being written and read back.
func TestWhatNoCodeReadsIsKeptAsWritten(t *testing.T) {
	corpus, notes := convert(t, `
  <events>
    <event handle="_e1" change="1" id="E0001">
      <type>Birth</type>
      <dateval val="1834"/>
      <weather kind="rain">
        <wind>north</wind>
      </weather>
    </event>
  </events>
  <people>
    <person handle="_i1" change="1" id="I0001" mood="cheerful">
      <name><first>Anna</first><surname>Rossi</surname></name>
      <eventref hlink="_e1" role="Primary"/>
      <shoesize unit="EU">37</shoesize>
    </person>
  </people>`)

	out := t.TempDir()
	if err := store.Write(out, corpus); err != nil {
		t.Fatal(err)
	}
	read, err := store.Load(out)
	if err != nil {
		t.Fatal(err)
	}
	anna := person(t, read.Corpus, "rossi-anna")
	keeps(t, "Anna", anna.Body, `<person handle="_i1" change="1" id="I0001" mood="cheerful">`,
		`  <shoesize unit="EU">37</shoesize>`, "</person>")
	keeps(t, "her birth", anna.Events[0].Note, `<weather kind="rain">`+"\n  <wind>north</wind>\n</weather>")
	for _, want := range []string{"<shoesize> was kept as written", "<weather> was kept as written", "mood on <person> was kept as written"} {
		if !slices.Contains(notes, want) {
			t.Errorf("did not report %q in %q", want, notes)
		}
	}
}

// An exporter may write an accent as a letter and a mark after it, and the corpus writes
// it as the one character, or the files it writes would not be canonical: in a value, in
// an attribute, in a note whose styles count the characters as written, and in what is
// kept as written.
func TestTextIsComposed(t *testing.T) {
	corpus, _ := convert(t, "<people>\n"+
		"  <person handle=\"_i1\" change=\"1\" id=\"I0001\">\n"+
		"    <name><first>René</first><surname>Pérez</surname></name>\n"+
		"    <mystery value=\"Président\"/>\n"+
		"    <noteref hlink=\"_n1\"/>\n"+
		"  </person>\n"+
		"</people>\n"+
		"<places>\n"+
		"  <placeobj handle=\"_p1\" change=\"1\" id=\"P0001\" type=\"City\">\n"+
		"    <pname value=\"Montréal\"/>\n"+
		"  </placeobj>\n"+
		"</places>\n"+
		"<notes>\n"+
		"  <note handle=\"_n1\" change=\"1\" id=\"N0001\" type=\"General\">\n"+
		"    <text>États-Unis, né à Paris</text>\n"+
		"    <style name=\"bold\"><range start=\"17\" end=\"26\"/></style>\n"+
		"  </note>\n"+
		"</notes>")
	rene := person(t, corpus, "perez-rene")
	if rene.Name.Given != "René" || rene.Name.PrimarySurname().Name != "Pérez" {
		t.Errorf("name = %+v", rene.Name)
	}
	keeps(t, "the body", rene.Body, `<mystery value="Président"/>`)
	for _, place := range corpus.Places {
		if place.Name.Text != "Montréal" {
			t.Errorf("place = %q", place.Name.Text)
		}
	}
	for _, note := range corpus.Notes {
		if note.Title != "États-Unis, né à Paris" || note.Body != "États-Unis, né **à Paris**" {
			t.Errorf("note = %q, %q", note.Title, note.Body)
		}
	}
}

// A name in a script a slug cannot spell is a name all the same. A place is filed under
// the first of its other names a slug can spell, then its title, as the GEDCOM import
// files one, and whatever is filed under its identifier is reported as having a name
// that could not be spelled, not as having none.
func TestARecordIsFiledUnderANameASlugCanSpell(t *testing.T) {
	t.Parallel()
	corpus, notes := convert(t, `
  <people>
    <person handle="_p" change="1" id="I0001">
      <name type="Birth Name"><first>Мария</first><surname>Попова</surname></name>
    </person>
  </people>
  <places>
    <placeobj handle="_drama" change="1" id="P0001" type="City">
      <pname value="Δράμα" lang="el"/>
      <pname value="Drama" lang="en"/>
    </placeobj>
    <placeobj handle="_salonica" change="1" id="P0002" type="City">
      <ptitle>Thessaloniki</ptitle>
      <pname value="Θεσσαλονίκη" lang="el"/>
    </placeobj>
    <placeobj handle="_athens" change="1" id="P0003" type="City">
      <pname value="Αθήνα" lang="el"/>
    </placeobj>
  </places>`)

	for _, slug := range []model.Slug{"drama", "thessaloniki", "p0003"} {
		if corpus.Places[slug] == nil {
			t.Errorf("no place %s among %v", slug, slices.Sorted(maps.Keys(corpus.Places)))
		}
	}
	if drama := corpus.Places["drama"]; drama != nil && drama.Name.Text != "Δράμα" {
		t.Errorf("Drama's name = %+v", drama.Name)
	}
	if maria := person(t, corpus, "i0001"); maria.Name.Given != "Мария" {
		t.Errorf("Maria's name = %+v", maria.Name)
	}
	for _, want := range []string{
		"a person had no name a slug can spell and is filed under their identifier",
		"a place had no name a slug can spell and is filed under its identifier",
	} {
		if !reported(notes, want) {
			t.Errorf("did not report %q in %q", want, notes)
		}
	}
	if reported(notes, "had no name and") {
		t.Errorf("a name was reported as none: %q", notes)
	}
}
