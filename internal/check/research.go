package check

import (
	"github.com/origins-project/origins/internal/model"
)

// Research lists what is still to be found out: the events nothing evidences, the scans
// nobody has transcribed, the hypotheses still open, the people who must have died by
// now with no death recorded, and every task somebody wrote down and has not ticked.
// None of it is wrong, and a corpus in the middle of being researched has plenty of
// each, so every one of them is a warning.
//
// The first two are warnings check gives as well. The rest are not, because an open
// question is the ordinary state of a hypothesis, a missing death the ordinary state of
// somebody alive and a task the ordinary state of research; asked for as research they
// are the list of what to do next.
//
// today is what "by now" means, handed in so that the answer does not change under a
// test and nothing here reads a clock.
func Research(c *model.Corpus, today model.Date) []Finding {
	var r report
	for _, apply := range []rule{
		unsourcedEvents,
		untranscribedScans,
		openHypotheses,
		unrecordedDeaths(today),
		tasks,
	} {
		apply(c, &r)
	}
	return r.sorted()
}

func openHypotheses(c *model.Corpus, r *report) {
	for _, s := range c.HypothesisSlugs() {
		if c.Hypotheses[s].Status == model.Open {
			r.warn("open-hypothesis", ref(model.KindHypothesis, s), "", "is still open")
		}
	}
}

// unrecordedDeaths lists everybody taken as dead with no death recorded: those the
// living rule finds certainly past the limit, and those marked deceased. A date that
// may be near enough to today is not past it, and somebody marked living is left
// alone whatever their dates say.
func unrecordedDeaths(today model.Date) rule {
	return func(c *model.Corpus, r *report) {
		lives := c.Lives(today)
		for _, s := range c.PeopleSlugs() {
			person := c.People[s]
			if person.DeathRecorded() || lives.Living(person) {
				continue
			}
			why := "marked deceased"
			if person.Living == model.LivingPresumed {
				why = lives.Presume(person).Why
			}
			r.warn("unrecorded-death", ref(model.KindPerson, s), "", "%s, and no death is recorded", why)
		}
	}
}

// tasks lists every task left unticked in prose, on the record and at the field it is
// written in.
func tasks(c *model.Corpus, r *report) {
	for _, s := range c.AllSlugs() {
		record := c.Entity(s)
		model.Prose(record, func(path string, text *string) {
			for _, task := range model.Tasks(*text) {
				r.warn("task", model.RefOf(record).String(), path, "%s", task.Text)
			}
		})
	}
}
