package model

import (
	"slices"
	"testing"
)

// vocabulary is the edit set as DESIGN.md states it, written out by hand. Go cannot
// check that a set of types covers a set of names, so this is the check: adding an
// operation to the specification without implementing it fails here rather than failing
// silently when a hypothesis is applied.
var vocabulary = []string{
	// field
	"set", "unset", "add", "remove",
	// entity
	"create", "delete", "rename",
	// structural
	"merge", "split",
}

func TestEveryOperationInTheVocabularyIsImplemented(t *testing.T) {
	for _, op := range vocabulary {
		edit, err := NewEdit(op)
		if err != nil {
			t.Errorf("%q is in the vocabulary but not implemented: %v", op, err)
			continue
		}
		if got := edit.Op(); got != op {
			t.Errorf("NewEdit(%q) produced an edit that calls itself %q", op, got)
		}
	}
}

func TestNoOperationIsImplementedThatIsNotInTheVocabulary(t *testing.T) {
	want := slices.Sorted(slices.Values(vocabulary))
	if got := EditOps(); !slices.Equal(got, want) {
		t.Errorf("implemented %q, vocabulary is %q", got, want)
	}
}

func TestNewEditRejectsAnUnknownOperation(t *testing.T) {
	if edit, err := NewEdit("adjust"); err == nil {
		t.Errorf("NewEdit(\"adjust\") = %v, want an error", edit)
	}
}

func TestParseEntityRef(t *testing.T) {
	for in, want := range map[string]EntityRef{
		"person/rossi-annunziata":              {KindPerson, "rossi-annunziata"},
		"family/bianchi-maria--rossi-giovanni": {KindFamily, "bianchi-maria--rossi-giovanni"},
		"place/firenze":                        {KindPlace, "firenze"},
		"source/parish-san-lorenzo-baptisms":   {KindSource, "parish-san-lorenzo-baptisms"},
		"hypothesis/annunziata-born-1844":      {KindHypothesis, "annunziata-born-1844"},
		"project":                              TheProject,
		"tag/To do":                            {KindTag, "To do"},
		"tag/Emigrati/Italia":                  {KindTag, "Emigrati/Italia"},
	} {
		got, err := ParseEntityRef(in)
		if err != nil {
			t.Errorf("ParseEntityRef(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseEntityRef(%q) = %+v, want %+v", in, got, want)
		}
		if got.String() != in {
			t.Errorf("ParseEntityRef(%q).String() = %q", in, got.String())
		}
	}

	for _, in := range []string{
		"", "rossi-giovanni", "people/rossi-giovanni", "person/", "/rossi-giovanni",
		"person/Rossi-Giovanni", "project/rossi-giovanni", "project/", "tag/", "tag/  ",
	} {
		if got, err := ParseEntityRef(in); err == nil {
			t.Errorf("ParseEntityRef(%q) = %v, want an error", in, got)
		}
	}
}

// A tag is read as written, in the one normal form every text is read in, so a word
// typed with a combining accent is the word a file carries with a composed one.
func TestATagIsReadInNormalForm(t *testing.T) {
	got, err := ParseEntityRef("tag/Città")
	if err != nil {
		t.Fatal(err)
	}
	if got.Slug != "Città" {
		t.Errorf("read %q", got.Slug)
	}
}

func TestParseCitationRef(t *testing.T) {
	got, err := ParseCitationRef("parish-san-lorenzo-baptisms#1834-042")
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "parish-san-lorenzo-baptisms" || got.Anchor != "1834-042" {
		t.Errorf("got %+v", got)
	}
	if got.String() != "parish-san-lorenzo-baptisms#1834-042" {
		t.Errorf("got %q", got.String())
	}

	for in, want := range map[string]Grade{
		"firenze-civil-deaths-1901#1103-17 OPD":  {Source: 'O', Information: 'P', Evidence: 'D'},
		"firenze-civil-deaths-1901#1103-17 OSI":  {Source: 'O', Information: 'S', Evidence: 'I'},
		"firenze-civil-deaths-1901#1103-17 O--":  {Source: 'O'},
		"firenze-civil-deaths-1901#1103-17 -UN":  {Information: 'U', Evidence: 'N'},
		"firenze-civil-deaths-1901#1103-17 OPD3": {'O', 'P', 'D', '3'},
		"firenze-civil-deaths-1901#1103-17 OSI2": {'O', 'S', 'I', '2'},
		"firenze-civil-deaths-1901#1103-17 ---1": {Weight: '1'},
		"firenze-civil-deaths-1901#1103-17 O--0": {Source: 'O', Weight: '0'},
	} {
		got, err := ParseCitationRef(in)
		if err != nil {
			t.Errorf("ParseCitationRef(%q): %v", in, err)
			continue
		}
		if got.Grade != want || got.String() != in {
			t.Errorf("ParseCitationRef(%q) = %+v, written %q", in, got, got.String())
		}
		if got.Ungraded().String() != "firenze-civil-deaths-1901#1103-17" {
			t.Errorf("%q ungraded is %q", in, got.Ungraded())
		}
	}
	// a grade that assesses nothing says nothing, and is not written, and a weight not
	// assessed is left off
	for in, want := range map[string]string{
		"parish#a ---": "parish#a", "parish#a ----": "parish#a", "parish#a OPD-": "parish#a OPD",
	} {
		if got, err := ParseCitationRef(in); err != nil || got.String() != want {
			t.Errorf("ParseCitationRef(%q) = %q, %v, want %q", in, got, err, want)
		}
	}

	for _, in := range []string{
		"", "parish-san-lorenzo-baptisms", "#1834-042", "parish#", "Parish#1",
		"parish#a b", "parish#a#b", "parish#a ", "parish#a  OPD", "parish#a OPDI",
		"parish#a PPD", "parish#a opd", "parish#a OD", "parish#a DPO", "parish OPD",
		"parish#a OPD4", "parish#a OPDD", "parish#a OPD12", "parish#a 3",
	} {
		if got, err := ParseCitationRef(in); err == nil {
			t.Errorf("ParseCitationRef(%q) = %v, want an error", in, got)
		}
	}
}

func TestParseEnums(t *testing.T) {
	if got, err := ParseSex(""); err != nil || got != SexUnknown {
		t.Errorf("an absent sex should read as unrecorded, got %v, %v", got, err)
	}
	if got, err := ParseSex("X"); err != nil || got != Neither || got.String() != "X" {
		t.Errorf("X is a sex the records state, got %v, %v", got, err)
	}
	if _, err := ParseSex("male"); err == nil {
		t.Error("ParseSex should reject anything but M, F, X and U")
	}

	for word, kin := range map[Relation]Relation{
		"": Birth, Adoptive: Adoptive, RelationUnknown: RelationUnknown, NoRelation: NoRelation,
		"natural": RelationUnknown,
	} {
		if got := word.Kin(); got != kin {
			t.Errorf("%q should count as %q, got %q", word, kin, got)
		}
	}
	if Relation("natural").Biological() || Adoptive.Biological() || !Relation("").Biological() {
		t.Error("only a birth relation, stated or unqualified, is biological")
	}

	if got, err := ParseLinkStatus(""); err != nil || got != "" {
		t.Errorf("an unstated link is proven by nothing, got %q, %v", got, err)
	}
	if got, err := ParseLinkStatus("disproven"); err != nil || got != Disproven {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := ParseLinkStatus("likely"); err == nil {
		t.Error("ParseLinkStatus should reject a status that is not one")
	}

	if _, err := ParseStatus(""); err == nil {
		t.Error("a hypothesis must say where it stands")
	}
	if got, err := ParseStatus("rejected"); err != nil || got != Rejected {
		t.Errorf("got %q, %v", got, err)
	}
	if got, err := ParseConfidence("possible"); err != nil || got != Possible {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := ParseConfidence("likely"); err == nil {
		t.Error("ParseConfidence should reject an unknown confidence")
	}
}
