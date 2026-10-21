package chart

import "github.com/origins-project/origins/internal/model"

// Related is what a path says besides its people: who it runs to, and, where the two
// have an ancestor in common, what the first is to the second in the words `origins
// kinship` uses and through whom.
type Related struct {
	To             string `json:"to"`
	Related        bool   `json:"related"`
	Relation       string `json:"relation,omitempty"`
	CommonAncestor string `json:"common-ancestor,omitempty"`
}

// path is the walk a relationship path draws: the people from the subject up to the
// ancestor the two share and down to the other, a generation a step, tied as their
// families tie them. Two people with no ancestor in common are each drawn alone.
func path(k *Kin, subject, to string) (Walked, *Related) {
	walked := Walked{Subject: subject, Order: []string{subject}, Generation: map[string]int{subject: 0}, Edges: []Kinship{}}
	if to == "" || !k.Has(to) || !k.Has(subject) {
		return walked, nil
	}
	related := &Related{To: to}
	found, ok := k.corpus.Kinship(model.Slug(subject), model.Slug(to))
	if !ok {
		if to != subject {
			walked.Order = append(walked.Order, to)
			walked.Generation[to] = 0
		}
		return walked, related
	}
	related.Related = true
	related.Relation = found.Name(k.corpus.People[model.Slug(subject)].Sex.Value)
	related.CommonAncestor = string(found.CommonAncestor)
	for i, slug := range found.Path[1:] {
		before := string(found.Path[i])
		after := string(slug)
		upward := i < found.Up
		gen := walked.Generation[before] + 1
		child, parent := before, after
		if upward {
			gen = walked.Generation[before] - 1
		} else {
			child, parent = after, before
		}
		walked.Order = append(walked.Order, after)
		walked.Generation[after] = gen
		tie := OtherTie
		for _, p := range k.Parents(child) {
			if p.Slug == parent && p.Birth {
				tie = BirthTie
			}
		}
		walked.Edges = append(walked.Edges, Kinship{parent, child, tie})
	}
	return walked, related
}
