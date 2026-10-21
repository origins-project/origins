package model

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// Living is what a person's record says of whether they are alive, overruling the rule
// that presumes it: false for somebody dead with no record of it, true for somebody
// alive whom the rule would not believe. Left out, which it nearly always is, the rule
// decides.
type Living int

// What a record can say of whether a person is living, LivingPresumed being nothing.
const (
	LivingPresumed Living = iota
	LivingTrue
	LivingFalse
)

var livingWord = [...]string{LivingPresumed: "", LivingTrue: "true", LivingFalse: "false"}

func (l Living) String() string {
	if int(l) >= len(livingWord) {
		return "?"
	}
	return livingWord[l]
}

func (l Living) MarshalText() ([]byte, error) { return []byte(l.String()), nil }

func (l *Living) UnmarshalText(b []byte) error {
	parsed, err := ParseLiving(string(b))
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

// ParseLiving reads the words a record overrules the rule with. Every scalar in a
// corpus is text, so yes, no, on and off are not read as either.
func ParseLiving(s string) (Living, error) {
	switch strings.TrimSpace(s) {
	case "true":
		return LivingTrue, nil
	case "false":
		return LivingFalse, nil
	case "":
		return LivingPresumed, nil
	}
	return LivingPresumed, fmt.Errorf("living %q: expected true or false, or left out", s)
}

// LongestLife is how many years past an event of somebody's life they can be presumed
// still alive. Lives longer than this are few enough to be news, so the researcher who
// has one in the corpus already knows, and says so with Living.
const LongestLife = 110

// endsALife are the event types that say a life is over, whatever their date.
var endsALife = []string{"death", "burial", "cremation"}

// DeathRecorded reports whether the person's events say their life is over: a death,
// or a burial or cremation, which nobody has without one.
func (p *Person) DeathRecorded() bool {
	_, recorded := p.recordedEnd()
	return recorded
}

func (p *Person) recordedEnd() (string, bool) {
	for _, event := range p.Events {
		if slices.ContainsFunc(endsALife, func(kind string) bool { return SameWord(kind, event.Type) }) {
			return event.Type, true
		}
	}
	return "", false
}

// Presumption is what the rule says of whether a person is living, and why, read
// without the person's own Living.
type Presumption struct {
	Living bool
	Why    string
}

func (p Presumption) String() string {
	if p.Living {
		return "presumed living: " + p.Why
	}
	return "presumed deceased: " + p.Why
}

// Presume applies the rule to a person. Everybody is presumed living unless a death, a
// burial or a cremation is recorded among their events, or an event of theirs or of a
// family they are a partner in is certainly more than LongestLife years before today —
// nobody's record holding an event from before they were born. A date that might be
// under the limit is not certainly over it, and a person with no dates at all is
// presumed living, being the person most likely to be recent and undocumented.
//
// today is a parameter so that what the rule says is not a fact about the day a test
// runs.
func (c *Corpus) Presume(p *Person, today Date) Presumption {
	return c.presume(p, today, c.FamiliesOf(p.Slug))
}

func (c *Corpus) presume(p *Person, today Date, partnerIn []Slug) Presumption {
	if end, recorded := p.recordedEnd(); recorded {
		return Presumption{Why: "a " + end + " is recorded"}
	}

	dated := false
	over := func(events []Event) (Presumption, bool) {
		for _, event := range events {
			least, _, ok := YearsBetween(event.Date, today)
			dated = dated || ok
			if ok && least > LongestLife {
				return Presumption{Why: fmt.Sprintf("%s %s is more than %d years ago",
					event.Type, event.Date, LongestLife)}, true
			}
		}
		return Presumption{}, false
	}
	if found, ok := over(p.Events); ok {
		return found
	}
	for _, family := range partnerIn {
		if found, ok := over(c.Families[family].Events); ok {
			return found
		}
	}
	if !dated {
		return Presumption{Living: true, Why: "no death recorded, and nothing dated"}
	}
	return Presumption{Living: true,
		Why: fmt.Sprintf("no death recorded, and nothing dated more than %d years ago", LongestLife)}
}

// Living reports whether a person is to be taken as alive today: as their record says,
// where it says, and otherwise as the rule presumes.
func (c *Corpus) Living(p *Person, today Date) bool {
	return living(p, func() Presumption { return c.Presume(p, today) })
}

func living(p *Person, presume func() Presumption) bool {
	switch p.Living {
	case LivingTrue:
		return true
	case LivingFalse:
		return false
	}
	return presume().Living
}

// Lives answers Presume and Living as of one day for whatever asks them of everybody,
// walking the families once rather than for each person.
type Lives struct {
	c         *Corpus
	today     Date
	partnerIn map[Slug][]Slug
}

// Lives is who is living as of today.
func (c *Corpus) Lives(today Date) Lives {
	return Lives{c: c, today: today, partnerIn: c.Ties().PartnerIn}
}

// Presume is Corpus.Presume.
func (l Lives) Presume(p *Person) Presumption {
	return l.c.presume(p, l.today, l.partnerIn[p.Slug])
}

// Living is Corpus.Living.
func (l Lives) Living(p *Person) bool {
	return living(p, func() Presumption { return l.Presume(p) })
}

// DateOf is the day a moment falls on, as an exact date.
func DateOf(t time.Time) Date {
	year, month, day := t.Date()
	m := int(month)
	return Date{Year: &year, Month: &m, Day: &day}
}
