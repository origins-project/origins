package model

import "testing"

func TestANameIsShownInItsParts(t *testing.T) {
	for _, c := range []struct {
		name PersonName
		want string
	}{
		{PersonName{Given: "Giovanni Battista", Call: "Giovanni", Surname: SurnameOf("Rossi")}, "Giovanni Battista Rossi"},
		{PersonName{Prefix: "Don", Given: "Pietro", Nickname: "Pierino", SurnamePrefix: "de'",
			Surname: SurnameOf("Medici"), Suffix: "sen."}, "Don Pietro “Pierino” de' Medici sen."},
		{PersonName{Call: "Nanni", Surname: SurnameOf("Rossi")}, "Nanni Rossi"},
		{PersonName{Surname: SurnameOf("Rossi"), Type: "married", From: Date{}}, "Rossi"},
		{PersonName{Text: "Gio. Rossi"}, "Gio. Rossi"},
	} {
		if got := c.name.String(); got != c.want {
			t.Errorf("%+v is shown as %q, want %q", c.name, got, c.want)
		}
	}
}

func TestABareNameIsUnqualified(t *testing.T) {
	bare := PersonName{Text: "Maria Bianca"}
	if bare.HasParts() || bare.Qualified() || bare.IsZero() {
		t.Errorf("%+v: parts %v, qualified %v, zero %v", bare, bare.HasParts(), bare.Qualified(), bare.IsZero())
	}
	typed := PersonName{Type: "married"}
	if !typed.Qualified() || typed.HasParts() {
		t.Errorf("%+v: qualified %v, parts %v", typed, typed.Qualified(), typed.HasParts())
	}
	if !(PersonName{}).IsZero() {
		t.Error("an empty name is not zero")
	}
}

// The record's words beside the parts are what the record said, and the parts are what
// the name is shown and filed by.
func TestTheWordsBesidePartsAreNotShown(t *testing.T) {
	name := PersonName{Text: "Gio. Batta di Antonio Rossi", Given: "Giovanni Battista", Surname: SurnameOf("Rossi")}
	if got := name.String(); got != "Giovanni Battista Rossi" {
		t.Errorf("the name is shown as %q", got)
	}
	if slug, _ := name.Slug(); slug != "rossi-giovanni-battista" {
		t.Errorf("the name is filed under %q", slug)
	}
}

func TestSurnamesAreJoinedByTheirConnectors(t *testing.T) {
	for _, c := range []struct {
		surnames Surnames
		want     string
	}{
		{Surnames{{Name: "Galli"}, {Name: "Cerreto", Prefix: "da"}}, "Ines Galli da Cerreto"},
		{Surnames{{Name: "García", Connector: "y"}, {Name: "Lorca"}}, "Ines García y Lorca"},
		{Surnames{{Name: "Lloyd", Connector: "-"}, {Name: "George"}}, "Ines Lloyd-George"},
	} {
		if got := (PersonName{Given: "Ines", Surname: c.surnames}).String(); got != c.want {
			t.Errorf("%+v is shown as %q, want %q", c.surnames, got, c.want)
		}
	}
}

// A person is filed under the primary surname without its prefix, whichever place in
// the list it has.
func TestANameIsFiledUnderItsPrimarySurname(t *testing.T) {
	name := PersonName{Given: "Ines", Surname: Surnames{{Name: "García"}, {Name: "Lorca", Prefix: "de", Primary: true}}}
	if slug, _ := name.Slug(); slug != "lorca-ines" {
		t.Errorf("the name is filed under %q", slug)
	}
	if got := name.Render("{primary-surname}, {given} ({other-surnames})"); got != "de Lorca, Ines (García)" {
		t.Errorf("the primary surname is rendered %q", got)
	}
}

func TestANameIsWrittenOutByItsPattern(t *testing.T) {
	name := PersonName{
		Prefix: "Don", Given: "Pietro Paolo", Nickname: "Pierino", SurnamePrefix: "de'",
		Surname: SurnameOf("Medici"), Suffix: "sen.",
	}
	for pattern, want := range map[string]string{
		"{SURNAME} {given}":                   "DE' MEDICI Pietro Paolo",
		"{surname}, {given} {suffix}":         "de' Medici, Pietro Paolo sen.",
		"{initials} {surname}":                "P.P. de' Medici",
		"{common} {surname}":                  "Pierino de' Medici",
		"{surname} [{call}]":                  "de' Medici",
		"({call}) {given}":                    "Pietro Paolo",
		"{surname}, {call}, {given}":          "de' Medici, Pietro Paolo",
		"{given} {patronymic}":                "Pietro Paolo",
		"{surname-prefix} | {surname} {nope}": "de' | de' Medici {nope}",
		"{Given}":                             "{Given}",
	} {
		if got := name.Render(pattern); got != want {
			t.Errorf("%q writes %q, want %q", pattern, got, want)
		}
	}
	if got := (PersonName{Given: "Ivan", Surname: Surnames{{Name: "Petrov"}, {Name: "Ivanovich", Origin: "Patronymic"}}}).
		Render("{given} {patronymic}"); got != "Ivan Ivanovich" {
		t.Errorf("the patronymic is %q", got)
	}
	shown := PersonName{Given: "Luigi", Surname: SurnameOf("Conti"), Display: "{SURNAME} {given}"}
	if got := shown.String(); got != "CONTI Luigi" {
		t.Errorf("a name with a display pattern is shown as %q", got)
	}
	shown.Display = "{call}"
	if got := shown.String(); got != "Luigi Conti" {
		t.Errorf("a display pattern writing nothing is shown as %q", got)
	}
}

func TestACloneOfANameHasItsOwnSources(t *testing.T) {
	name := PersonName{Surname: SurnameOf("Rossi"), Sources: []CitationRef{{Source: "a", Anchor: "1"}}}
	copied := name.Clone()
	copied.Sources[0].Anchor = "2"
	if name.Sources[0].Anchor != "1" {
		t.Error("changing the clone's sources changed the original's")
	}
}
