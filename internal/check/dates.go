package check

import (
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// The bounds a rule has to clear before it will speak. They are deliberately far outside
// what is ordinary: the job here is to catch a transcription error, not to tell a
// genealogist that an unusual family was unusual.
const (
	youngestParent = 12
	oldestMother   = 55
	oldestFather   = 80
)

// firstEvent returns the earliest-listed event of a type, and where it sits, because a
// person may carry two death records from sources that disagree.
func firstEvent(events []model.Event, kind string) (model.Event, int, bool) {
	for i, event := range events {
		if model.SameWord(event.Type, kind) {
			return event, i, true
		}
	}
	return model.Event{}, 0, false
}

// personDates checks a person against themselves. Every rule here speaks only when the
// dates leave no room for doubt: Compare answers with whether it could decide, and an
// undecided answer is a floating or imprecise date doing its job rather than a fault.
func personDates(c *model.Corpus, r *report) {
	for _, s := range c.PeopleSlugs() {
		person := c.People[s]
		birth, _, hasBirth := firstEvent(person.Events, "birth")
		death, deathAt, hasDeath := firstEvent(person.Events, "death")
		if !hasBirth || !hasDeath {
			continue
		}
		// a birth and a death on the same day is an infant death, not a contradiction
		if order, decided := model.Compare(death.Date, birth.Date); decided && order < 0 {
			r.fail("death-before-birth", ref(model.KindPerson, s), at("events", deathAt)+".date",
				"died %s, born %s", death.Date, birth.Date)
		}
	}
}

// parentDates checks a person against the people they descend from. Age rules apply to
// birth parents only: an adoptive mother may be younger than her daughter, and a
// guardian may have been born after the person they took in.
func parentDates(c *model.Corpus, r *report) {
	parentage := c.Parentage()
	for _, s := range c.PeopleSlugs() {
		child := c.People[s]
		birth, birthAt, ok := firstEvent(child.Events, "birth")
		if !ok {
			continue
		}
		entity, path := ref(model.KindPerson, s), at("events", birthAt)+".date"

		for _, link := range parentage[s] {
			parent, ok := c.People[link.Slug]
			if !ok || !link.Relation.Biological() {
				continue
			}
			parentBirth, _, hasBirth := firstEvent(parent.Events, "birth")
			if hasBirth {
				ageGap(r, entity, path, link.Slug, parent.Sex.Value, parentBirth.Date, birth.Date)
			}
			if parentDeath, _, ok := firstEvent(parent.Events, "death"); ok {
				posthumous(r, entity, path, link.Slug, parentDeath.Date, birth.Date)
			}
		}
	}
}

func ageGap(r *report, entity, path string, parent model.Slug, parentSex model.Sex, parentBirth, birth model.Date) {
	if order, decided := model.Compare(birth, parentBirth); decided && order < 0 {
		r.fail("child-before-parent", entity, path,
			"born %s, before %s was born in %s", birth, parent, parentBirth)
		return
	}
	least, most, ok := model.YearsBetween(parentBirth, birth)
	if !ok {
		return
	}
	if most < youngestParent {
		r.fail("parent-too-young", entity, path,
			"%s would have been at most %.0f", parent, most)
	}

	// a sex unrecorded or recorded as neither takes the looser bound, because guessing
	// would invent evidence
	oldest := float64(oldestFather)
	if parentSex == model.Female {
		oldest = oldestMother
	}
	if least > oldest {
		r.warn("parent-too-old", entity, path,
			"%s would have been at least %.0f", parent, least)
	}
}

// posthumous speaks only when a child was certainly born a whole year or more after a
// parent died. A child born months after its father's death is ordinary, and the looser
// bound covers both parents without inventing a rule per sex.
func posthumous(r *report, entity, path string, parent model.Slug, parentDeath, birth model.Date) {
	if least, _, ok := model.YearsBetween(parentDeath, birth); ok && least >= 1 {
		r.fail("posthumous-child", entity, path,
			"born %s, at least %.0f year(s) after %s died in %s", birth, least, parent, parentDeath)
	}
}

// familyOrder is what the dates of a family's events have to agree with: each entry an
// event, the one it is measured against, and whether it comes after that one rather
// than before. A marriage contract or settlement may be made on either side of a
// wedding, so neither is here, and a type the list does not name is never read.
var familyOrder = []struct {
	event, other string
	after        bool
}{
	{"engagement", "marriage", false},
	{"marriage-banns", "marriage", false},
	{"marriage-licence", "marriage", false},
	{"separation", "marriage", true},
	{"divorce-filed", "marriage", true},
	{"divorce", "marriage", true},
	{"annulment", "marriage", true},
	{"divorce", "divorce-filed", true},
}

// familyEnders are the events a partner's death forecloses, the death having ended the
// marriage already. An annulment is not among them: one may be sought of a marriage
// that has ended, to say it never was one.
var familyEnders = []string{"separation", "divorce"}

// familyDates checks a family's events against each other and against its partners'
// deaths.
func familyDates(c *model.Corpus, r *report) {
	for _, s := range c.FamilySlugs() {
		family := c.Families[s]
		entity := ref(model.KindFamily, s)

		for _, rule := range familyOrder {
			event, eventAt, ok := firstEvent(family.Events, rule.event)
			if !ok {
				continue
			}
			other, _, ok := firstEvent(family.Events, rule.other)
			if !ok {
				continue
			}
			order, decided := model.Compare(event.Date, other.Date)
			if !decided || order == 0 || (order > 0) == rule.after {
				continue
			}
			side := "before"
			if order > 0 {
				side = "after"
			}
			r.fail("family-event-order", entity, at("events", eventAt)+".date",
				"%s %s, %s the %s %s", rule.event, event.Date, side, rule.other, other.Date)
		}

		for _, kind := range familyEnders {
			event, eventAt, ok := firstEvent(family.Events, kind)
			if !ok {
				continue
			}
			for _, partner := range family.Partners {
				person, ok := c.People[partner.Person]
				if !ok {
					continue
				}
				death, _, ok := firstEvent(person.Events, "death")
				if !ok {
					continue
				}
				if order, decided := model.Compare(event.Date, death.Date); decided && order > 0 {
					r.fail("family-ended-after-death", entity, at("events", eventAt)+".date",
						"%s %s, after %s died in %s", kind, event.Date, partner.Person, death.Date)
				}
			}
		}
	}
}

// ruledOut holds what did not happen against what did. An event of a type a `not`
// entry names, certainly within its date or at any time where it has none, contradicts
// it, and one that may fall within it is to be looked at. A person's is held against the
// events of the families they are a partner in as well, a marriage being the family's.
func ruledOut(c *model.Corpus, r *report) {
	partnerIn := c.Ties().PartnerIn
	for _, s := range c.PeopleSlugs() {
		person, entity := c.People[s], ref(model.KindPerson, s)
		ruleOut(r, entity, person.Not, entity, person.Events)
		for _, f := range partnerIn[s] {
			ruleOut(r, entity, person.Not, ref(model.KindFamily, f), c.Families[f].Events)
		}
	}
	for _, s := range c.FamilySlugs() {
		family, entity := c.Families[s], ref(model.KindFamily, s)
		ruleOut(r, entity, family.Not, entity, family.Events)
	}
}

func ruleOut(r *report, entity string, not []model.NonEvent, owner string, events []model.Event) {
	for i, entry := range not {
		for j, event := range events {
			if !model.SameWord(event.Type, entry.Type) {
				continue
			}
			what := strings.TrimSpace(event.Type + " " + event.Date.String())
			where := owner + " " + at("events", j)
			if entry.Date.IsZero() {
				r.fail("ruled-out", entity, at("not", i), "%s at %s, which never happened", what, where)
				continue
			}
			switch certainly, possibly := model.Within(event.Date, entry.Date); {
			case certainly:
				r.fail("ruled-out", entity, at("not", i), "%s at %s, within %s when none happened",
					what, where, entry.Date)
			case possibly:
				r.warn("possibly-ruled-out", entity, at("not", i), "%s at %s, perhaps within %s when none happened",
					what, where, entry.Date)
			}
		}
	}
}
