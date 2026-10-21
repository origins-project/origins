package check

import (
	"maps"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// circularAncestry catches a corpus where someone descends from themselves. Derivation
// walks the graph with a visited set and so survives a cycle quietly, which is the
// right behaviour for a query and the wrong one for a validator.
func circularAncestry(c *model.Corpus, r *report) {
	parentage := c.Parentage()
	component := components(c, parentage)
	for _, s := range c.FamilySlugs() {
		family := c.Families[s]
		for i, child := range family.Children {
			for _, parent := range parentage[child.ID] {
				if parent.Family != s || !child.Somebody() || parent.Slug.Reserved() ||
					parent.Slug != child.ID && component[parent.Slug] != component[child.ID] {
					continue
				}
				r.fail("circular-ancestry", ref(model.KindFamily, s), at("children", i),
					"%q is already a descendant of %q", parent.Slug, child.ID)
			}
		}
	}
}

// components numbers the strongly connected components of the parent graph, so that two
// people carry the same number exactly when each is an ancestor of the other — which is
// what a parent reference standing in a cycle looks like.
//
// The alternative is to ask for the ancestry of every parent of every person and look
// for the child in it. That reads more directly and walks the same subgraph thousands of
// times over: on a royal tree of thirty-four thousand people it costs more than every
// other rule together. Numbering the components is one pass over the edges.
//
// Numbers start at one, so a reference to somebody the corpus does not contain keeps the
// zero that no component has.
func components(c *model.Corpus, parentage map[model.Slug][]model.Parent) map[model.Slug]int {
	var (
		index   = map[model.Slug]int{}
		lowest  = map[model.Slug]int{}
		open    = map[model.Slug]bool{}
		stack   []model.Slug
		reached int
		found   = map[model.Slug]int{}
		count   int
		visit   func(model.Slug)
	)

	visit = func(person model.Slug) {
		reached++
		index[person], lowest[person] = reached, reached
		stack = append(stack, person)
		open[person] = true

		for _, parent := range parentage[person] {
			if _, known := c.People[parent.Slug]; !known {
				continue
			}
			switch {
			case index[parent.Slug] == 0:
				visit(parent.Slug)
				lowest[person] = min(lowest[person], lowest[parent.Slug])
			case open[parent.Slug]:
				lowest[person] = min(lowest[person], index[parent.Slug])
			}
		}

		// A person whose lowest reachable index is their own heads a component, and
		// everything stacked above them belongs to it.
		if lowest[person] != index[person] {
			return
		}
		count++
		for {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			open[top] = false
			found[top] = count
			if top == person {
				return
			}
		}
	}

	for _, s := range c.PeopleSlugs() {
		if index[s] == 0 {
			visit(s)
		}
	}
	return found
}

// duplicateCandidates looks for the same person entered twice. Naming a child after a
// living grandparent is a custom, not a mistake, so this speaks only when the dates
// cannot tell the two apart — which is exactly when a human should look.
func duplicateCandidates(c *model.Corpus, r *report) {
	sameName := map[string][]model.Slug{}
	for _, s := range c.PeopleSlugs() {
		person := c.People[s]
		if key := nameKey(person); key != "" {
			sameName[key] = append(sameName[key], s)
		}
	}

	for _, key := range slices.Sorted(maps.Keys(sameName)) {
		group := sameName[key]
		for i, a := range group {
			for _, b := range group[i+1:] {
				if distinguishable(c, a, b) {
					continue
				}
				r.warn("duplicate-person", ref(model.KindPerson, a), "name",
					"%q may be the same person; no date tells them apart", b)
			}
		}
	}
}

// nameKey is a surname and a first given name, which is as much as records reliably
// agree on, or the whole of a name nobody has divided. A person with no name to compare
// is not a candidate for anything, and neither is a hidden view's placeholder, whose
// name is the view's and not theirs.
func nameKey(person *model.Person) string {
	if person.IsPlaceholder() {
		return ""
	}
	name := person.Name
	if !name.HasParts() {
		whole, err := model.Slugify(name.Text)
		if err != nil {
			return ""
		}
		return string(whole)
	}
	surname, err := model.Slugify(name.PrimarySurname().Name)
	if err != nil {
		return ""
	}
	given := strings.Fields(name.Given)
	if len(given) == 0 {
		given = strings.Fields(name.Call)
	}
	if len(given) == 0 {
		return ""
	}
	first, err := model.Slugify(given[0])
	if err != nil {
		return ""
	}
	return string(surname) + " " + string(first)
}

// distinguishable reports whether any recorded date puts the two beyond confusion.
func distinguishable(c *model.Corpus, a, b model.Slug) bool {
	for _, kind := range []string{"birth", "death"} {
		first, _, hasFirst := firstEvent(c.People[a].Events, kind)
		second, _, hasSecond := firstEvent(c.People[b].Events, kind)
		if !hasFirst || !hasSecond {
			continue
		}
		if order, decided := model.Compare(first.Date, second.Date); decided && order != 0 {
			return true
		}
	}
	return false
}

// hypothesisEvidence reports a hypothesis that argues from nothing. An open hypothesis
// is research still to do and never a fault; one with no citation on either side is a
// hunch with no way back to a record.
//
// Nothing here asks whether a hypothesis's base is reachable, whether its edits still
// apply, or whether an accepted one is reflected in the corpus. Each has to open the
// repository and check never opens one, which is what lets git bisect run it; --with
// and accepting are where those questions are put.
func hypothesisEvidence(c *model.Corpus, r *report) {
	for _, s := range c.HypothesisSlugs() {
		hypothesis, entity := c.Hypotheses[s], ref(model.KindHypothesis, s)

		if len(hypothesis.Edits) == 0 {
			r.fail("hypothesis-changes-nothing", entity, "edits",
				"a hypothesis is a proposed departure from the corpus, and this one proposes none")
		}
		if len(hypothesis.Supports) > 0 || len(hypothesis.Contradicts) > 0 {
			continue
		}
		if hypothesis.ProposedBy != "" {
			r.warn("unevidenced-proposal", entity, "supports",
				"proposed by %q and cites no record on either side", hypothesis.ProposedBy)
			continue
		}
		r.warn("unevidenced-hypothesis", entity, "supports", "cites no record on either side")
	}
}
