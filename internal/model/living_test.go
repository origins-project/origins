package model

import (
	"strings"
	"testing"
	"time"
)

// Every branch of the rule, each on a person built for it, on a day fixed so that what
// the rule says does not move with the day the test runs.
func TestTheRulePresumesWhoIsLiving(t *testing.T) {
	today := DateOf(time.Date(2026, time.September, 28, 12, 0, 0, 0, time.UTC))
	event := func(kind, date string) Event { return Event{Type: kind, Date: mustParse(t, date)} }

	for _, c := range []struct {
		what    string
		events  []Event
		family  []Event // the events of a family the person is a partner in
		living  bool
		because string
	}{
		{"a death recorded", []Event{event("death", "1990")}, nil,
			false, "a death is recorded"},
		{"a burial recorded and no death", []Event{event("burial", "1990")}, nil,
			false, "a burial is recorded"},
		{"a cremation recorded and no death", []Event{event("cremation", "1990")}, nil,
			false, "a cremation is recorded"},
		{"a burial written in capitals", []Event{event("BURIAL", "1990")}, nil,
			false, "a BURIAL is recorded"},
		{"a death with no date", []Event{event("birth", "1990"), {Type: "death"}}, nil,
			false, "a death is recorded"},
		{"a birth certainly over the limit", []Event{event("birth", "1900")}, nil,
			false, "birth 1900 is more than 110 years ago"},
		{"a birth that might be under the limit", []Event{event("birth", "1916")}, nil,
			true, "no death recorded, and nothing dated more than 110 years ago"},
		{"a birth that may be recent", []Event{event("birth", "AFT 1900")}, nil,
			true, "nothing dated more than 110 years ago"},
		{"a birth before a year itself over the limit", []Event{event("birth", "BEF 1900")}, nil,
			false, "birth BEF 1900 is more than 110 years ago"},
		{"a birth before a year under the limit", []Event{event("birth", "BEF 1950")}, nil,
			true, "nothing dated more than 110 years ago"},
		{"a birth in a decade reaching under the limit", []Event{event("birth", "BET 1910 AND 1919")}, nil,
			true, "nothing dated more than 110 years ago"},
		{"an event other than a birth over the limit", []Event{event("occupation", "FROM 1880 TO 1900")}, nil,
			false, "occupation FROM 1880 TO 1900 is more than 110 years ago"},
		{"a period still open", []Event{event("residence", "FROM 1880")}, nil,
			true, "nothing dated more than 110 years ago"},
		{"a family's event over the limit", nil, []Event{event("marriage", "1890")},
			false, "marriage 1890 is more than 110 years ago"},
		{"a family's event under it", nil, []Event{event("marriage", "1950")},
			true, "nothing dated more than 110 years ago"},
		{"a birth with no year", []Event{event("birth", "3 MAR")}, nil,
			true, "no death recorded, and nothing dated"},
		{"no events at all", nil, nil,
			true, "no death recorded, and nothing dated"},
	} {
		corpus := NewCorpus()
		person := &Person{Slug: "rossi-giovanni", Events: c.events}
		corpus.People[person.Slug] = person
		if c.family != nil {
			corpus.People["verdi-lucia"] = &Person{Slug: "verdi-lucia"}
			corpus.Families["rossi-giovanni--verdi-lucia"] = &Family{Slug: "rossi-giovanni--verdi-lucia",
				Partners: PartnersOf("rossi-giovanni", "verdi-lucia"), Events: c.family}
		}

		got := corpus.Presume(person, today)
		if got.Living != c.living || !strings.HasSuffix(got.Why, c.because) {
			t.Errorf("%s: %s, want living %v because %q", c.what, got, c.living, c.because)
		}
		if corpus.Living(person, today) != c.living {
			t.Errorf("%s: the rule is not what decides with nothing said against it", c.what)
		}
		if lives := corpus.Lives(today); lives.Presume(person) != got || lives.Living(person) != c.living {
			t.Errorf("%s: asked of everybody, the rule says %s", c.what, lives.Presume(person))
		}

		// Each override wins over whatever the rule says, and leaves what the rule
		// says as it was.
		for _, override := range []Living{LivingTrue, LivingFalse} {
			person.Living = override
			if want := override == LivingTrue; corpus.Living(person, today) != want {
				t.Errorf("%s: living %s is not taken as %v", c.what, override, want)
			}
			if corpus.Presume(person, today) != got {
				t.Errorf("%s: living %s changes what the rule says", c.what, override)
			}
		}
	}
}

func TestPresumptionSaysWhatTheRuleSaysAndWhy(t *testing.T) {
	for _, c := range []struct {
		p    Presumption
		want string
	}{
		{Presumption{Living: true, Why: "no death recorded, and nothing dated"},
			"presumed living: no death recorded, and nothing dated"},
		{Presumption{Why: "a burial is recorded"}, "presumed deceased: a burial is recorded"},
	} {
		if got := c.p.String(); got != c.want {
			t.Errorf("%q, want %q", got, c.want)
		}
	}
}

func TestLivingIsTrueOrFalseOrLeftOut(t *testing.T) {
	for _, word := range []string{"", "true", "false"} {
		living, err := ParseLiving(word)
		if err != nil || living.String() != word {
			t.Errorf("%q reads as %q, %v", word, living, err)
		}
	}
	for _, word := range []string{"yes", "no", "on", "1", "True"} {
		if _, err := ParseLiving(word); err == nil {
			t.Errorf("%q read as a word living takes", word)
		}
	}
}
