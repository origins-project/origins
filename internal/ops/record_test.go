package ops

import (
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
)

// vocabulary is one edit of every operation there is, which is what lets a test ask
// the same question of all nine. It is written out here rather than built, so that an
// operation added to the model has to be answered for rather than swept along.
func vocabulary() map[string]model.Edit {
	giovanni := ref(model.KindPerson, "rossi-giovanni")
	return map[string]model.Edit{
		"set":    model.Set{Entity: giovanni, Path: "name.surname", Value: "Rossi Tempesti"},
		"unset":  model.Unset{Entity: giovanni, Path: "name.surname"},
		"add":    model.Add{Entity: giovanni, Path: "sources", Value: "firenze-civil-deaths-1901#p-1"},
		"remove": model.Remove{Entity: giovanni, Path: "sources", Value: "firenze-civil-deaths-1901#p-1"},
		"create": model.Create{Entity: ref(model.KindPerson, "rossi-antonio")},
		"delete": model.Delete{Entity: giovanni},
		"rename": model.Rename{Entity: giovanni, To: "rossi-giovanni-1834"},
		"merge":  model.Merge{From: ref(model.KindPerson, "rossi-giovannino"), Into: giovanni},
		"split": model.Split{Entity: giovanni, Into: []model.EntityRef{
			ref(model.KindPerson, "rossi-giovanni-elder"),
			ref(model.KindPerson, "rossi-giovanni-younger"),
		}},
	}
}

// Every operation describes itself. A description git will not take is a change that
// gets as far as the files and then stops, which is the one state a transaction exists
// to prevent.
func TestEveryOperationDescribesItself(t *testing.T) {
	t.Parallel()
	tried := vocabulary()
	for _, op := range model.EditOps() {
		edit, covered := tried[op]
		if !covered {
			t.Errorf("nothing here tries a %q", op)
			continue
		}
		noun, verb, split := strings.Cut(operation(edit), ".")
		switch {
		case !split || verb != op:
			t.Errorf("%q is filed as %q rather than something.%s", op, operation(edit), op)
		case noun == "":
			t.Errorf("%q is filed under nothing at all", op)
		}
		switch line := subject(edit); {
		case strings.TrimSpace(line) == "":
			t.Errorf("%q has no subject line, and a commit has to have one", op)
		case strings.ContainsAny(line, "\n\r"):
			t.Errorf("%q has a subject of more than one line: %q", op, line)
		case len(edit.Subjects()) == 0:
			t.Errorf("%q names no entity, so no commit could record one", op)
		case !strings.Contains(line, string(edit.Subjects()[0].Slug)):
			t.Errorf("%q does not say what it was about: %q", op, line)
		}
	}
}

// The noun a change is filed under is the field it lands in, spelled the way the
// corpus spells it, and the kind of entity where the change is of a whole one. There
// is no second set of names for the nine operations to drift out of step with.
func TestAChangeIsFiledUnderWhatItChanged(t *testing.T) {
	t.Parallel()
	giovanni := ref(model.KindPerson, "rossi-giovanni")
	for _, one := range []struct {
		edit model.Edit
		want string
	}{
		{model.Set{Entity: giovanni, Path: "name.surname"}, "name.set"},
		{model.Set{Entity: giovanni, Path: "events[0].date"}, "events.set"},
		{model.Add{Entity: giovanni, Path: "events"}, "events.add"},
		{model.Unset{Entity: giovanni, Path: "citations.1103-17.text"}, "citations.unset"},
		{model.Create{Entity: giovanni}, "person.create"},
		{model.Delete{Entity: ref(model.KindPlace, "firenze")}, "place.delete"},
	} {
		if got := operation(one.edit); got != one.want {
			t.Errorf("%+v is filed as %q, want %q", one.edit, got, one.want)
		}
	}
}

// An event added is in the corpus only afterwards and an event taken out only before,
// so the trailer is read on the side of the change the event is on. Recording the
// wrong event is worse than recording none: a reader asking history for every death
// would be answered, and answered wrongly.
func TestAnEventIsNamedFromTheSideItIsOn(t *testing.T) {
	t.Parallel()
	giovanni := ref(model.KindPerson, "rossi-giovanni")
	for _, one := range []struct {
		what string
		edit model.Edit
		want string
	}{
		{"adding one", model.Add{Entity: giovanni, Path: "events",
			Value: map[string]any{"type": "death", "date": "14 MAR 1901"}}, "death"},
		{"changing one", model.Set{Entity: giovanni, Path: "events[0].note", Value: "at home"}, "birth"},
		{"taking one out", model.Unset{Entity: giovanni, Path: "events[0]"}, "birth"},
		{"changing something else", model.Set{Entity: giovanni, Path: "name.surname", Value: "Rossi Tempesti"}, ""},
	} {
		corpus := aCorpus(t)
		entry, _, err := performed(corpus, one.edit)
		if err != nil {
			t.Fatalf("%s: %v", one.what, err)
		}
		got := strings.Join(entry.Events, " ")
		if got != one.want {
			t.Errorf("%s named %q, want %q", one.what, got, one.want)
		}
	}
}

// Perform is Apply with the history entry for it, so an edit it refuses leaves the
// corpus exactly as Apply would have left it — untouched.
func TestAnEditPerformCannotCarryOutChangesNothing(t *testing.T) {
	t.Parallel()
	corpus := aCorpus(t)
	if _, _, err := performed(corpus, model.Set{
		Entity: ref(model.KindPerson, "rossi-giovanni"), Path: "birthplace", Value: "firenze",
	}); err == nil {
		t.Fatal("a field nobody knows was set")
	}
	if got := corpus.People["rossi-giovanni"].Name.PrimarySurname().Name; got != "Rossi" {
		t.Errorf("the person reads %+v", corpus.People["rossi-giovanni"])
	}
}

// A rename nobody described is told by a record gone and one of the same kind arrived
// that write the same bytes, and by nothing looser: a record renamed and edited at once
// is a deletion and a creation, and two records either of which could be the one that
// arrived are left unpaired rather than guessed between.
func TestARenameNobodyDescribedIsToldByTheBytes(t *testing.T) {
	t.Parallel()
	people := func(named map[model.Slug]string) *model.Corpus {
		corpus := model.NewCorpus()
		for slug, surname := range named {
			corpus.People[slug] = &model.Person{Slug: slug, Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf(surname)}}
		}
		return corpus
	}
	before := people(map[model.Slug]string{
		"rossi-giovanni": "Rossi",
		"conti-luigi":    "Conti",
		"galli-ines":     "Galli",
		"ferri-a":        "Ferri",
		"ferri-b":        "Ferri",
	})
	after := people(map[model.Slug]string{
		"rossi-giovanni-1834": "Rossi",
		"conti-luigi-1850":    "Conti Rossi",
		"galli-ines":          "Galli",
		"ferri-c":             "Ferri",
		"ferri-d":             "Ferri",
	})
	after.Places["rossi-giovanni"] = &model.Place{Slug: "rossi-giovanni", Name: model.PlaceName{Text: "Villa Rossi"}}

	got := Renamed(before, after)
	want := []history.Renaming{{
		From: ref(model.KindPerson, "rossi-giovanni"),
		To:   ref(model.KindPerson, "rossi-giovanni-1834"),
	}}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
