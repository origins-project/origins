package model

import (
	"slices"
	"strings"
)

// Concerns are the entities a hypothesis's edits are about, each once, in the order the
// edits first name them. An entity named only inside a value, as a parent in the fields
// of a person it creates, is not among them: the hypothesis changes nothing in it.
func (h *Hypothesis) Concerns() []EntityRef {
	var out []EntityRef
	for _, edit := range h.Edits {
		for _, ref := range edit.Subjects() {
			if !slices.Contains(out, ref) {
				out = append(out, ref)
			}
		}
	}
	return out
}

// OpenHypotheses returns the open hypotheses that concern an entity, in slug order.
func (c *Corpus) OpenHypotheses(about Slug) []Slug {
	var out []Slug
	for _, slug := range c.HypothesisSlugs() {
		hypothesis := c.Hypotheses[slug]
		if hypothesis.Status == Open && slices.ContainsFunc(hypothesis.Concerns(), func(ref EntityRef) bool {
			return ref.Slug == about && ref.Kind != KindTag
		}) {
			out = append(out, slug)
		}
	}
	return out
}

// Alternatives returns every other hypothesis, whatever its status, with an edit landing
// where one of this one's edits lands, in slug order. Two answers to one question are
// alternatives whether or not either has been settled, and a reader weighing one wants
// to see what was argued against it.
//
// Paths are compared as written. Two hypotheses pinned to different bases can mean
// different events by one positional path, and the base is what says which.
func (c *Corpus) Alternatives(to Slug) []Slug {
	hypothesis := c.Hypotheses[to]
	if hypothesis == nil {
		return nil
	}
	var out []Slug
	for _, slug := range c.HypothesisSlugs() {
		if slug != to && collide(hypothesis.Edits, c.Hypotheses[slug].Edits) {
			out = append(out, slug)
		}
	}
	return out
}

func collide(a, b []Edit) bool {
	for _, x := range a {
		for _, y := range b {
			if overlap(x, y) {
				return true
			}
		}
	}
	return false
}

// overlap reports whether two edits share a subject and land at one path, or one inside
// the other. An edit about a whole entity overlaps only another about a whole entity.
func overlap(a, b Edit) bool {
	shared := slices.ContainsFunc(a.Subjects(), func(x EntityRef) bool {
		return slices.ContainsFunc(b.Subjects(), func(y EntityRef) bool { return x == y })
	})
	return shared && (within(a.Field(), b.Field()) || within(b.Field(), a.Field()))
}

// within reports whether path lies at or beneath outer: events[0].date and events[0]
// both lie within events[0], and events[10] does not.
func within(path, outer string) bool {
	if path == "" || outer == "" {
		return path == outer
	}
	rest, found := strings.CutPrefix(path, outer)
	return found && (rest == "" || rest[0] == '.' || rest[0] == '[')
}
