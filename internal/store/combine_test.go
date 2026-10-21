package store

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// Where one record is silent the other's value stands. Two records of one person are
// almost always one full and one sparse — a census stub, a duplicate carried in by an
// import — and taking what only one of them says is the whole of what makes merging
// them worth doing.
func TestWhatOneRecordIsSilentAboutTheOtherFillsIn(t *testing.T) {
	into := &model.Person{Slug: "rossi-giovanni", Name: model.PersonName{Surname: model.SurnameOf("Rossi")}}
	from := &model.Person{
		Slug: "rossi-giovanni-1902", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
		AlsoKnownAs: []model.PersonName{{Text: "G. Rossi"}},
		Body:        "Named in the 1861 census.",
	}
	if err := Combine(into, from); err != nil {
		t.Fatal(err)
	}
	switch {
	case into.Name.Given != "Giovanni" || into.Name.PrimarySurname().Name != "Rossi":
		t.Errorf("the name became %+v, taking the given name only the other record gave", into.Name)
	case into.Sex.Value != model.Male:
		t.Errorf("the sex the other record gave was not taken: %v", into.Sex)
	case len(into.AlsoKnownAs) != 1 || into.AlsoKnownAs[0].Text != "G. Rossi":
		t.Errorf("the other names read %+v", into.AlsoKnownAs)
	case into.Body != "Named in the 1861 census.":
		t.Errorf("the body reads %q", into.Body)
	case into.Slug != "rossi-giovanni":
		t.Errorf("the slug became %q, and identity is not a field to be combined", into.Slug)
	}
}

// Which of two spellings is the surname is a judgement about the records, and there is
// nothing mechanical to decide it with. The fold stops and says which field it was, so
// that whoever knows can unset one of them and ask again.
//
// It stops without having taken anything. Sex is reached before surname and the
// surviving record is silent about it, so a fold that took each field as it came to it
// would leave a record that is partly both and no longer either.
func TestTwoRecordsThatDisagreeAreNotCombined(t *testing.T) {
	into := &model.Person{Slug: "rossi-giovanni", Name: model.PersonName{Surname: model.SurnameOf("Rossi")}}
	from := &model.Person{Slug: "rossi-giovanni-1902", Name: model.PersonName{Surname: model.SurnameOf("Rossi Tempesti")}, Sex: model.StatedSex{Value: model.Male}}

	err := Combine(into, from)
	if err == nil {
		t.Fatal("one of two surnames was chosen")
	}
	for _, want := range []string{"name.surname", `"Rossi"`, `"Rossi Tempesti"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %s: %v", want, err)
		}
	}
	switch {
	case into.Name.PrimarySurname().Name != "Rossi":
		t.Errorf("the surname became %+v", into.Name.Surname)
	case into.Sex.Value != model.SexUnknown:
		t.Errorf("the fold stopped having already taken the sex: %v", into.Sex)
	}
}

// A name's words stand beside its parts, so the words of one record fold beside the
// parts of another; two records' words that differ, or two lists of surnames, are two
// readings and not one.
func TestANameAsWrittenIsFoldedBesideItsParts(t *testing.T) {
	want := model.PersonName{Text: "Gio. Rossi", Given: "Giovanni", Surname: model.SurnameOf("Rossi")}
	for _, names := range [][2]model.PersonName{
		{{Text: "Gio. Rossi"}, {Given: "Giovanni", Surname: model.SurnameOf("Rossi")}},
		{{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}, {Text: "Gio. Rossi"}},
	} {
		into := &model.Person{Slug: "rossi-giovanni", Name: names[0]}
		from := &model.Person{Slug: "rossi-giovanni-1902", Name: names[1]}
		if err := Combine(into, from); err != nil || !reflect.DeepEqual(into.Name, want) {
			t.Errorf("%v and %v folded into %+v: %v", names[0], names[1], into.Name, err)
		}
	}

	for _, names := range [][2]model.PersonName{
		{{Text: "Gio. Rossi"}, {Text: "Giovanni Rossi"}},
		{{Surname: model.SurnameOf("Galli")}, {Surname: model.Surnames{{Name: "Galli"}, {Name: "Cerreto", Prefix: "da"}}}},
	} {
		into := &model.Person{Slug: "rossi-giovanni", Name: names[0]}
		from := &model.Person{Slug: "rossi-giovanni-1902", Name: names[1]}
		err := Combine(into, from)
		if err == nil || !strings.Contains(err.Error(), "name.") {
			t.Errorf("%v and %v folded: %v", names[0], names[1], err)
		}
		if !reflect.DeepEqual(into.Name, names[0]) {
			t.Errorf("the name became %+v", into.Name)
		}
	}
}

// A list holds more than one thing, so it takes both — bar what the surviving record
// already says word for word, two records of one person having been recorded from the
// same register as often as not.
func TestListsAreJoinedWithoutRepeatingWhatIsAlreadyThere(t *testing.T) {
	born := mustDate("1834")
	died := mustDate("14 MAR 1901")
	into := &model.Person{Slug: "rossi-giovanni", Events: []model.Event{
		{Type: "birth", Date: born, Place: "firenze"},
	}}
	from := &model.Person{Slug: "rossi-giovanni-1902", Events: []model.Event{
		{Type: "birth", Date: born, Place: "firenze"},
		{Type: "death", Date: died},
	}}
	if err := Combine(into, from); err != nil {
		t.Fatal(err)
	}
	if len(into.Events) != 2 {
		t.Fatalf("the events read %+v", into.Events)
	}
	if into.Events[1].Type != "death" {
		t.Errorf("the event taken over is %+v", into.Events[1])
	}
}

// Two records of one person at one service are what a service's own merge leaves, so the
// identifiers are joined at that service, the surviving record's first, as the current.
func TestIdentifiersAtOneServiceAreJoined(t *testing.T) {
	into := &model.Person{Slug: "rossi-giovanni", IDs: model.IDs{
		"familysearch": model.IdentifiersOf("XXXX-YYY"),
	}}
	from := &model.Person{Slug: "rossi-giovanni-1902", IDs: model.IDs{
		"familysearch": model.IdentifiersOf("XXXX-XXX", "XXXX-YYY"),
		"wikitree":     model.IdentifiersOf("Rossi-1834"),
	}}
	if err := Combine(into, from); err != nil {
		t.Fatal(err)
	}
	want := model.IDs{
		"familysearch": model.IdentifiersOf("XXXX-YYY", "XXXX-XXX"),
		"wikitree":     model.IdentifiersOf("Rossi-1834"),
	}
	if !reflect.DeepEqual(into.IDs, want) {
		t.Errorf("the identifiers read %+v", into.IDs)
	}
}

// A citation both records transcribe is reconciled the way the records themselves are,
// so a transcription and the scan beside it arriving from two records end up as one
// citation rather than as a collision.
func TestTwoTranscriptionsOfOneCitationAreReconciled(t *testing.T) {
	into := &model.Source{Slug: "a", Citations: map[string]model.Citation{
		"1103-17": {Text: "Giovanni Rossi, di anni 67"},
	}}
	from := &model.Source{Slug: "b", Citations: map[string]model.Citation{
		"1103-17": {Images: []model.Media{{File: "scans/1103-17.jpg"}}},
		"1103-18": {Text: "Maria Bianchi"},
	}}
	if err := Combine(into, from); err != nil {
		t.Fatal(err)
	}
	switch got := into.Citations["1103-17"]; {
	case got.Text != "Giovanni Rossi, di anni 67":
		t.Errorf("the transcription reads %q", got.Text)
	case len(got.Images) != 1 || got.Images[0].File != "scans/1103-17.jpg":
		t.Errorf("the scans read %+v", got.Images)
	case into.Citations["1103-18"].Text != "Maria Bianchi":
		t.Errorf("the anchor only one record had is missing: %+v", into.Citations)
	}
}

// A citation the two records transcribe differently is the same judgement as a surname,
// and it is refused under the anchor it is written at rather than under the mapping.
func TestTwoTranscriptionsThatDisagreeNameTheAnchor(t *testing.T) {
	into := &model.Source{Slug: "a", Citations: map[string]model.Citation{
		"1103-17": {Text: "di anni 67"},
	}}
	from := &model.Source{Slug: "b", Citations: map[string]model.Citation{
		"1103-17": {Text: "di anni 76"},
	}}
	err := Combine(into, from)
	if err == nil {
		t.Fatal("one of two transcriptions was chosen")
	}
	if !strings.Contains(err.Error(), "citations.1103-17.text") {
		t.Errorf("the refusal does not say where it was: %v", err)
	}
}

// Prose is the one text where having both is having more rather than having a muddle,
// so two bodies become two paragraphs instead of a disagreement.
func TestTwoBodiesBecomeTwoParagraphs(t *testing.T) {
	into := &model.Person{Slug: "a", Body: "The parish register has him at 67.\n"}
	from := &model.Person{Slug: "b", Body: "The census gives his age as 76."}
	if err := Combine(into, from); err != nil {
		t.Fatal(err)
	}
	want := "The parish register has him at 67.\n\nThe census gives his age as 76."
	if into.Body != want {
		t.Errorf("the body reads %q, want %q", into.Body, want)
	}
}

// A person and a place are not two records of one thing, whatever they are called.
func TestRecordsOfDifferentKindsAreNotCombined(t *testing.T) {
	if err := Combine(&model.Person{Slug: "a"}, &model.Place{Slug: "b"}); err == nil {
		t.Fatal("a place was folded into a person")
	}
}

// A photograph is part of what a record has to say, so folding two records folds what
// is attached to them. It is the list that is folded and not the files: where they end
// up sitting is the business of whatever writes the corpus back.
//
// The portrait comes across with them, and would name a picture that will not load if
// they did not.
func TestWhatIsAttachedToOneRecordEndsUpAttachedToTheOther(t *testing.T) {
	into := &model.Person{Slug: "rossi-giovanni", Files: []string{"photos/1890-ritratto.jpg"}}
	from := &model.Person{
		Slug:     "rossi-giovanni-1902",
		Portrait: model.Image{File: "photos/1902-passaporto.jpg"},
		Files:    []string{"certificato.pdf", "photos/1902-passaporto.jpg"},
	}
	if err := Combine(into, from); err != nil {
		t.Fatal(err)
	}
	want := []string{"certificato.pdf", "photos/1890-ritratto.jpg", "photos/1902-passaporto.jpg"}
	if !slices.Equal(into.Files, want) {
		t.Errorf("it has %v beside it, want %v", into.Files, want)
	}
	if into.Portrait.File != "photos/1902-passaporto.jpg" {
		t.Errorf("the portrait reads %q", into.Portrait.File)
	}
}

// Whether two files called ritratto.jpg are one photograph cannot be told from the
// names, and keeping either of them would be deciding it. It stops like any other
// disagreement, and stops having taken nothing: the sex is reached before the files
// are, and the surviving record is silent about it.
func TestTwoFilesOfOneNameAreNotCombined(t *testing.T) {
	into := &model.Person{Slug: "rossi-giovanni", Files: []string{"ritratto.jpg"}}
	from := &model.Person{
		Slug: "rossi-giovanni-1902", Sex: model.StatedSex{Value: model.Male},
		Files: []string{"certificato.pdf", "ritratto.jpg"},
	}

	err := Combine(into, from)
	if err == nil {
		t.Fatal("one of two photographs was chosen")
	}
	if !strings.Contains(err.Error(), "ritratto.jpg") {
		t.Errorf("the refusal does not say which file: %v", err)
	}
	if !slices.Equal(into.Files, []string{"ritratto.jpg"}) {
		t.Errorf("it has %v beside it", into.Files)
	}
	if into.Sex.Value != model.SexUnknown {
		t.Error("it stopped at the files having already taken the sex")
	}
}
