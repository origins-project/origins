package check

import (
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

var today = on("28 SEP 2026")

func researched(c *model.Corpus) []string {
	var out []string
	for _, f := range Research(c, today) {
		out = append(out, f.Rule+" "+f.Entity+" "+f.Path)
	}
	return out
}

// Each of the five kinds of thing to do, and nothing that is not one of them: a
// contradiction is check's to report, and the list of research is not where to find it.
func TestResearchListsWhatIsStillToBeFoundOut(t *testing.T) {
	c := build(
		&model.Person{Slug: "rossi-giovanni", Events: []model.Event{born("1834")}},
		&model.Person{Slug: "rossi-anna", Events: []model.Event{born("1860"), died("1850")}},
		childOf("rossi-giovanni", "rossi-anna"),
		&model.Source{Slug: "parish", Files: []string{"page-1.jpg", "page-2.jpg"},
			Citations: map[string]model.Citation{"1": {Text: "…", Images: []model.Media{{File: "page-1.jpg"}},
				Note: "- [x] Read the margin\n- [ ] Ask about the stamp\n"}}},
		&model.Hypothesis{Slug: "open-one", Status: model.Open},
		&model.Hypothesis{Slug: "settled-one", Status: model.Rejected},
	)
	want := []string{
		"open-hypothesis hypothesis/open-one ",
		"unsourced-event person/rossi-anna events[0]",
		"unsourced-event person/rossi-anna events[1]",
		"unrecorded-death person/rossi-giovanni ",
		"unsourced-event person/rossi-giovanni events[0]",
		"task source/parish citations.1.note",
		"untranscribed-scan source/parish ",
	}
	if got := researched(c); !slices.Equal(got, want) {
		t.Errorf("research is\n%q\nwant\n%q", got, want)
	}
}

// A missing death is research only when somebody is taken as dead: by the living rule,
// which needs a date certainly long enough ago, or by their record saying so. A date
// that might be recent enough, one with no year, a life ended by a burial, and anybody
// marked living are all left alone.
func TestAnUnrecordedDeathIsListedOnlyWhenItCertainlyHappened(t *testing.T) {
	event := func(kind, date string) model.Event { return model.Event{Type: kind, Date: on(date)} }
	for _, c := range []struct {
		what   string
		events []model.Event
		family []model.Event
		living model.Living
		listed string
	}{
		{"born a hundred and twenty-six years ago", []model.Event{born("1900")}, nil, model.LivingPresumed,
			"birth 1900 is more than 110 years ago, and no death is recorded"},
		{"a year that may be under a hundred and ten years ago", []model.Event{born("1916")}, nil, model.LivingPresumed, ""},
		{"born within living memory", []model.Event{born("1978")}, nil, model.LivingPresumed, ""},
		{"a birth with no year", []model.Event{born("3 MAR")}, nil, model.LivingPresumed, ""},
		{"a birth that may be recent", []model.Event{born("AFT 1900")}, nil, model.LivingPresumed, ""},
		{"a death recorded", []model.Event{born("1900"), died("1970")}, nil, model.LivingPresumed, ""},
		{"a burial recorded", []model.Event{born("1900"), event("burial", "1970")}, nil, model.LivingPresumed, ""},
		{"no birth to count from", nil, nil, model.LivingPresumed, ""},
		{"no birth, but a census long ago", []model.Event{event("census", "1901")}, nil, model.LivingPresumed,
			"census 1901 is more than 110 years ago, and no death is recorded"},
		{"no birth, but a marriage long ago", nil, []model.Event{event("marriage", "1895")}, model.LivingPresumed,
			"marriage 1895 is more than 110 years ago, and no death is recorded"},
		{"born long ago and marked living", []model.Event{born("1900")}, nil, model.LivingTrue, ""},
		{"marked deceased with no dates", nil, nil, model.LivingFalse,
			"marked deceased, and no death is recorded"},
		{"marked deceased with a death recorded", []model.Event{died("1970")}, nil, model.LivingFalse, ""},
	} {
		corpus := build(&model.Person{Slug: "rossi-giovanni", Events: c.events, Living: c.living},
			&model.Person{Slug: "verdi-lucia", Events: []model.Event{died("1970")}})
		if c.family != nil {
			corpus.Families["rossi-giovanni--verdi-lucia"] = &model.Family{Slug: "rossi-giovanni--verdi-lucia",
				Partners: model.PartnersOf("rossi-giovanni", "verdi-lucia"), Events: c.family}
		}
		listed := ""
		for _, f := range Research(corpus, today) {
			if f.Rule == "unrecorded-death" {
				listed = f.Entity + ": " + f.Message
			}
		}
		if want := "person/rossi-giovanni: " + c.listed; c.listed == "" && listed != "" || c.listed != "" && listed != want {
			t.Errorf("%s: listed %q, want %q", c.what, listed, c.listed)
		}
	}
}

// An open hypothesis and a death nobody has found are research to do and never an
// error, and check says nothing of either.
func TestCheckIsSilentAboutWhatOnlyResearchLists(t *testing.T) {
	c := build(
		&model.Person{Slug: "rossi-giovanni", Events: []model.Event{born("1834")}},
		&model.Hypothesis{Slug: "open-one", Status: model.Open},
	)
	silent(t, "open-hypothesis", c)
	silent(t, "unrecorded-death", c)
}
