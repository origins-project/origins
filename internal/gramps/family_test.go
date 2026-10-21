package gramps

import (
	"reflect"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// A Gramps family is a family, holding the events it points at and listing its children
// in the relation each child's reference gives to the father and to the mother; one of
// one partner has `unknown` beside them. A second Gramps family of the same couple is a
// second family, told apart by the year of its first event.
func TestAFamilyListsItsChildrenInTheRelationsTheirReferencesGive(t *testing.T) {
	corpus, notes := convert(t, `
  <events>
    <event handle="_marriage" change="1" id="E0001">
      <type>Marriage</type>
      <dateval val="1870-05-01"/>
    </event>
    <event handle="_second" change="1" id="E0002">
      <type>Divorce Filing</type>
      <dateval val="1880"/>
    </event>
    <event handle="_census" change="1" id="E0003">
      <type>Census</type>
      <dateval val="1881"/>
    </event>
  </events>
  <people>
    <person handle="_carlo" change="1" id="I0001">
      <gender>M</gender>
      <name><first>Carlo</first><surname>Rossi</surname></name>
      <eventref hlink="_marriage" role="Family"/>
      <parentin hlink="_f1"/>
    </person>
    <person handle="_anna" change="1" id="I0002">
      <gender>F</gender>
      <name><first>Anna</first><surname>Bianchi</surname></name>
      <eventref hlink="_marriage" role="Bride"/>
      <parentin hlink="_f1"/>
    </person>
    <person handle="_pietro" change="1" id="I0003">
      <gender>M</gender>
      <name><first>Pietro</first><surname>Neri</surname></name>
      <eventref hlink="_marriage" role="Witness"/>
    </person>
    <person handle="_luigi" change="1" id="I0004">
      <gender>M</gender>
      <name><first>Luigi</first><surname>Rossi</surname></name>
      <childof hlink="_f1"/>
    </person>
    <person handle="_maria" change="1" id="I0005">
      <gender>F</gender>
      <name><first>Maria</first><surname>Rossi</surname></name>
    </person>
    <person handle="_teresa" change="1" id="I0006">
      <gender>F</gender>
      <name><first>Teresa</first><surname>Rossi</surname></name>
    </person>
    <person handle="_giulia" change="1" id="I0007">
      <gender>F</gender>
      <name><first>Giulia</first><surname>Verdi</surname></name>
    </person>
    <person handle="_elena" change="1" id="I0008">
      <gender>F</gender>
      <name><first>Elena</first><surname>Verdi</surname></name>
    </person>
  </people>
  <families>
    <family handle="_f1" change="1" id="F0001">
      <rel type="Civil Union"/>
      <father hlink="_carlo"/>
      <mother hlink="_anna"/>
      <eventref hlink="_marriage" role="Family"/>
      <childref hlink="_luigi" frel="Adopted">
        <citationref hlink="_c1"/>
        <noteref hlink="_n1"/>
      </childref>
      <childref hlink="_maria" mrel="Sponsored" frel="None"/>
      <childref hlink="_teresa" mrel="Custom relationship to mother" frel="Stepchild" priv="1"/>
      <attribute type="Number of Children" value="3"/>
      <attribute type="_UID" value="0123456789ABCDEF"/>
      <attribute type="_FSFTID" value="ABCD-123"/>
      <noteref hlink="_n2"/>
      <citationref hlink="_c1"/>
      <lds_ord type="sealed_to_spouse"><temple val="FLORE"/></lds_ord>
    </family>
    <family handle="_f2" change="1" id="F0002">
      <rel type="Married"/>
      <father hlink="_carlo"/>
      <mother hlink="_anna"/>
      <eventref hlink="_second" role="Family"/>
    </family>
    <family handle="_f3" change="1" id="F0003" priv="1">
      <rel type="Unknown"/>
      <mother hlink="_giulia"/>
      <eventref hlink="_census" role="Family"/>
      <childref hlink="_elena"/>
    </family>
  </families>
  <citations>
    <citation handle="_c1" change="1" id="C0001">
      <page>n. 12</page>
      <confidence>2</confidence>
      <sourceref hlink="_s1"/>
    </citation>
  </citations>
  <sources>
    <source handle="_s1" change="1" id="S0001">
      <stitle>Stato civile di Firenze</stitle>
    </source>
  </sources>
  <notes>
    <note handle="_n1" change="1" id="N0001" type="General"><text>Taken in at two.</text></note>
    <note handle="_n2" change="1" id="N0002" type="Family Note"><text>They met in Prato.</text></note>
  </notes>`)

	family := corpus.Families["bianchi-anna--rossi-carlo"]
	if family == nil {
		t.Fatalf("no family among %v", corpus.FamilySlugs())
	}
	if !reflect.DeepEqual(family.Partners, model.PartnersOf("bianchi-anna", "rossi-carlo")) || family.Type != model.CivilUnion {
		t.Errorf("family = %+v", family)
	}
	if idAt(family.IDs, "gramps") != "F0001" || idAt(family.IDs, "familysearch") != "ABCD-123" {
		t.Errorf("ids = %v", family.IDs)
	}
	if family.Body != "[They met in Prato.](../../notes/they-met-in-prato/README.md)" || len(family.Sources) != 1 || family.Sources[0].String() != "stato-civile-di-firenze#n-12" {
		t.Errorf("body = %q, sources = %v", family.Body, family.Sources)
	}
	var types []string
	for _, e := range family.Events {
		types = append(types, e.Type)
	}
	if !slices.Equal(types, []string{"marriage", "lds-sealing-to-spouse", "number-of-children"}) {
		t.Fatalf("the family's events are %v", types)
	}
	second := corpus.Families["bianchi-anna--rossi-carlo-1880"]
	switch {
	case second == nil:
		t.Fatalf("the couple's second family is not among %v", corpus.FamilySlugs())
	case second.Type != model.Married || idAt(second.IDs, "gramps") != "F0002":
		t.Errorf("the second family is %+v", second)
	case len(second.Events) != 1 || second.Events[0].Type != "divorce-filed":
		t.Errorf("the second family's events are %+v", second.Events)
	}
	// the partners are whom the marriage is about, and only the witness is named in it
	if got := names(family.Events[0].With); !slices.Equal(got, []string{"neri-pietro witness"}) {
		t.Errorf("the marriage names %v", got)
	}
	if rite := family.Events[1]; !reflect.DeepEqual(rite.Attributes, []model.Attribute{{Type: "temple", Value: "FLORE"}}) {
		t.Errorf("the sealing = %+v", rite)
	}
	if family.Events[2].Value != "3" {
		t.Errorf("the number of children = %+v", family.Events[2])
	}
	for _, p := range []model.Slug{"rossi-carlo", "bianchi-anna"} {
		if events := person(t, corpus, p).Events; len(events) != 0 {
			t.Errorf("%s was given the family's events: %+v", p, events)
		}
	}

	relations := func(father, mother model.Relation) model.Relationship {
		return model.Relationship{Partners: []model.PartnerRelation{
			{Partner: "rossi-carlo", Relation: father}, {Partner: "bianchi-anna", Relation: mother}}}
	}
	want := []model.Relationship{
		relations(model.Adoptive, model.Birth),
		relations(model.NoRelation, model.Guardian),
		relations(model.Step, "Custom relationship to mother"),
	}
	if len(family.Children) != len(want) {
		t.Fatalf("children = %+v", family.Children)
	}
	for i, child := range family.Children {
		if !reflect.DeepEqual(child.Relation, want[i]) {
			t.Errorf("%s is %+v, want %+v", child.ID, child.Relation, want[i])
		}
	}
	if got := family.Children[0]; got.ID != "rossi-luigi" || len(got.Sources) != 1 || got.Sources[0].Anchor != "n-12" {
		t.Errorf("the first child is %+v", got)
	}

	alone := corpus.Families["verdi-giulia--unknown"]
	switch {
	case alone == nil:
		t.Fatalf("the family of one partner is not among %v", corpus.FamilySlugs())
	case !alone.Private || len(alone.Events) != 1 || alone.Events[0].Type != "census":
		t.Errorf("the family of one partner is %+v", alone)
	case !reflect.DeepEqual(alone.Children, model.ChildrenOf("verdi-elena")):
		t.Errorf("its children are %+v", alone.Children)
	}

	if idAt(family.IDs, "uid") != "0123456789ABCDEF" {
		t.Errorf("the family's ids are %v", family.IDs)
	}
	if !family.Private {
		t.Error("a private child's entry did not make the family listing the child private")
	}
	if got := family.Children[0].Note; got != "[Taken in at two.](../../notes/taken-in-at-two/README.md)" {
		t.Errorf("the note on Luigi's entry is %q", got)
	}
	for slug, want := range map[model.Slug]string{"taken-in-at-two": "Taken in at two.", "they-met-in-prato": "They met in Prato."} {
		if note := corpus.Notes[slug]; note == nil || note.Body != want {
			t.Errorf("the note %s is %+v among %v", slug, note, corpus.NoteSlugs())
		}
	}
	if !reported(notes, "a private child's entry made private the family listing the child") {
		t.Errorf("did not report the private child's entry in %q", notes)
	}
	if reported(notes, "kept as written") {
		t.Errorf("kept what is read: %q", notes)
	}
	if reported(notes, "nobody holds") || slices.Contains(notes, "<event> was kept as written") ||
		reported(notes, "same couple") || reported(notes, "two identifiers") {
		t.Errorf("reported %q", notes)
	}
}
