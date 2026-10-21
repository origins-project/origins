package gedcom

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// CONFIDENTIAL and PRIVACY make a person private, written in either case, and a person
// without a RESN is not.
func TestARestrictedIndividualIsPrivate(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Irene /Costa/
1 RESN confidential
0 @I2@ INDI
1 NAME Marco /Rossi/
1 RESN PRIVACY
0 @I3@ INDI
1 NAME Anna /Rossi/
`)
	for slug, want := range map[model.Slug]bool{"costa-irene": true, "rossi-marco": true, "rossi-anna": false} {
		if got := corpus.People[slug].Private; got != want {
			t.Errorf("%s private = %v, want %v", slug, got, want)
		}
	}
	if reported(notes, "RESN") {
		t.Errorf("a RESN that was read was reported: %v", notes)
	}
}

// A FAM's RESN makes its family private, and a second FAM of the same couple is a family
// of its own, private or not as it says, as is a FAM of one partner.
func TestARestrictedFAMIsAPrivateFamily(t *testing.T) {
	corpus, notes := convert(t, `0 @I1@ INDI
1 NAME Claudia /Rossi/
0 @I2@ INDI
1 NAME Davide /Sanna/
0 @I3@ INDI
1 NAME Paolo /Marchi/
0 @I4@ INDI
1 NAME Luca /Marchi/
0 @F1@ FAM
1 HUSB @I2@
1 WIFE @I1@
1 RESN PRIVACY
0 @F2@ FAM
1 HUSB @I3@
1 WIFE @I1@
0 @F3@ FAM
1 HUSB @I3@
1 WIFE @I1@
1 RESN CONFIDENTIAL
0 @F4@ FAM
1 HUSB @I3@
1 CHIL @I4@
1 RESN PRIVACY
`)
	for slug, want := range map[model.Slug]bool{
		"rossi-claudia--sanna-davide":   true,
		"marchi-paolo--rossi-claudia":   false,
		"marchi-paolo--rossi-claudia-2": true,
		"marchi-paolo--unknown":         true,
	} {
		family := corpus.Families[slug]
		if family == nil {
			t.Fatalf("families are %v", corpus.FamilySlugs())
		}
		if family.Private != want {
			t.Errorf("%s private = %v, want %v", slug, family.Private, want)
		}
	}
	if reported(notes, "RESN") {
		t.Errorf("a RESN that was read was reported: %v", notes)
	}
}

// An event's RESN makes that event private, under a person or under a family, and
// leaves the rest of the record as it was.
func TestARestrictedEventIsPrivate(t *testing.T) {
	corpus, _ := convert(t, `0 @I1@ INDI
1 NAME Marco /Rossi/
1 BIRT
2 DATE 1960
1 EVEN
2 TYPE Illness
2 DATE 2009
2 RESN CONFIDENTIAL
0 @I2@ INDI
1 NAME Stefano /Ferri/
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 MARR
2 DATE 2016
2 RESN PRIVACY
`)
	marco := corpus.People["rossi-marco"]
	if marco.Private {
		t.Error("an event's restriction made the whole person private")
	}
	if got := private(marco.Events); !slices.Equal(got, []string{"Illness"}) {
		t.Errorf("private events = %v", got)
	}
	family := corpus.Families["ferri-stefano--rossi-marco"]
	if family.Private {
		t.Error("an event's restriction made the whole family private")
	}
	if got := private(family.Events); !slices.Equal(got, []string{"marriage"}) {
		t.Errorf("private family events = %v", got)
	}
}

func private(events []model.Event) []string {
	var out []string
	for _, e := range events {
		if e.Private {
			out = append(out, e.Type)
		}
	}
	return out
}

// 7.0 allows a list of restrictions. LOCKED is about changing a record, not showing it,
// and origins keeps every change in the history rather than refusing one, so it is read
// and dropped alone or beside one that does make it private; a word that is none of the
// three is kept as written.
func TestALockedRecordIsNotPrivate(t *testing.T) {
	corpus, notes := convert(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @I1@ INDI
1 NAME Irene /Costa/
1 RESN CONFIDENTIAL, LOCKED
0 @I2@ INDI
1 NAME Anna /Rossi/
1 RESN LOCKED
0 @I3@ INDI
1 NAME Luca /Rossi/
1 RESN SECRET
`)
	for slug, want := range map[model.Slug]bool{"costa-irene": true, "rossi-anna": false, "rossi-luca": false} {
		if got := corpus.People[slug].Private; got != want {
			t.Errorf("%s private = %v, want %v", slug, got, want)
		}
	}
	if body := corpus.People["rossi-luca"].Body; !strings.Contains(body, "1 RESN SECRET") {
		t.Errorf("the unknown restriction was not kept: %q", body)
	}
	if body := corpus.People["rossi-anna"].Body + corpus.People["costa-irene"].Body; body != "" {
		t.Errorf("a restriction origins reads was kept: %q", body)
	}
	if !slices.Contains(notes, "RESN was kept as written") {
		t.Errorf("the unknown restriction was not counted once: %v", notes)
	}
}

// 7.0 puts a RESN on a picture's own record, and the picture comes in private.
func TestARestrictedPictureIsPrivate(t *testing.T) {
	corpus, _, notes := Convert(records(t, `0 HEAD
1 GEDC
2 VERS 7.0
0 @O1@ OBJE
1 RESN PRIVACY
1 FILE photos/at-home.jpg
2 FORM image/jpeg
0 @O2@ OBJE
1 FILE photos/portrait.jpg
2 FORM image/jpeg
0 @O3@ OBJE
1 RESN CONFIDENTIAL, LOCKED
1 FILE photos/lost.jpg
0 @I1@ INDI
1 NAME Marco /Rossi/
1 OBJE @O1@
1 OBJE @O2@
0 @I2@ INDI
1 NAME Anna /Rossi/
1 OBJE @O3@
0 @I3@ INDI
1 NAME Luca /Rossi/
1 OBJE @O3@
`), nil, func(name string) (string, bool) { return name, name != "photos/lost.jpg" })
	marco := corpus.People["rossi-marco"]
	if len(marco.Media) != 2 || !marco.Media[0].Private || marco.Media[1].Private {
		t.Errorf("media = %+v", marco.Media)
	}
	// a picture nobody has is an entry all the same, private as its record says
	lost := []model.Media{{Private: true, File: "photos/lost.jpg",
		Depicts: []model.Depicted{{Person: "rossi-anna"}, {Person: "rossi-luca"}}}}
	if got := corpus.People["rossi-anna"].Media; !reflect.DeepEqual(got, lost) {
		t.Errorf("the picture nobody has is %+v", got)
	}
	if !reported(notes, "was not found beside it") || reported(notes, "kept as written") {
		t.Errorf("notes = %v", notes)
	}
}

// A RESN below the record is read where what it is under can be marked: a source, the
// name a person is filed under, which is hidden with them, another name, and somebody
// named beside a person, which is an event. Where nothing can be — a citation — it is
// kept as written as any tag with no home is, and makes nothing private.
func TestARestrictionBelowTheRecordMarksWhatItIsUnder(t *testing.T) {
	corpus, notes := convert(t, `0 @S1@ SOUR
1 TITL Register
1 RESN CONFIDENTIAL
0 @I1@ INDI
1 NAME Marco /Rossi/
1 NAME Marcuccio /Rossi/
2 RESN PRIVACY
1 ASSO @I2@
2 ROLE GODP
2 RESN PRIVACY
1 BIRT
2 SOUR @S1@
3 RESN PRIVACY
0 @I2@ INDI
1 NAME Anna /Rossi/
2 RESN CONFIDENTIAL
`)
	if !corpus.Sources["register"].Private {
		t.Error("the source is not private")
	}
	if !corpus.People["rossi-anna"].Private {
		t.Error("the person filed under a private name is not")
	}
	marco := corpus.People["rossi-marco"]
	if marco.Private || len(marco.AlsoKnownAs) != 1 || !marco.AlsoKnownAs[0].Private {
		t.Errorf("names = %+v %+v", marco.Name, marco.AlsoKnownAs)
	}
	if len(marco.Events) != 2 || !marco.Events[1].Private || marco.Events[0].Private {
		t.Errorf("events = %+v", marco.Events)
	}
	citation := corpus.Sources["register"].Citations[marco.Events[0].Sources[0].Anchor]
	if citation.Note != "```gedcom\n3 RESN PRIVACY\n```" || !slices.Contains(notes, "RESN was kept as written") {
		t.Errorf("the citation's RESN was not kept: %q %v", citation.Note, notes)
	}
}
