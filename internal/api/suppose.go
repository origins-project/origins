package api

import (
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// --with is how a hypothesis is put to the corpus without being believed, and it is
// the whole of what holding competing readings as data rather than as branches was
// for. The corpus stays the standing conclusion; the question is asked somewhere
// else and answered about the corpus in front of the researcher.

// Supposition is how one hypothesis got into the corpus, for whatever reads the output.
type Supposition struct {
	Hypothesis string `json:"hypothesis"`
	Base       string `json:"base,omitempty"`
	Why        string `json:"why,omitempty"`

	said string
}

// String is how the hypothesis was read, in a sentence.
func (s Supposition) String() string { return s.said }

func suppositions(read []ops.Reading) []Supposition {
	var out []Supposition
	for _, r := range read {
		out = append(out, Supposition{Hypothesis: string(r.Slug), Base: r.Base, Why: r.Why, said: r.String()})
	}
	return out
}

// supposing builds the corpus a --with asks about, and hands back nothing at all
// where none was asked for. The caller closes what it hands back.
//
// It does not come through owned. What this reads at a commit is what the files said
// there, which is true of a commit whoever wrote it, so a history origins cannot
// account for is no reason to decline the question; and a corpus in no repository has
// to be answerable too, that being one of the ways a hypothesis can have no earlier
// state to be read at.
//
// A hypothesis that no longer applies comes back as the ops.Drifted it is, an answer
// about the corpus and the hypothesis together rather than a mistake in the asking.
func supposing(corpus *store.Corpus, with []model.Slug) (*ops.Trial, error) {
	if len(with) == 0 {
		return nil, nil
	}
	return ops.Suppose(corpus, with)
}

// Supposed builds the corpus a server serves with the hypotheses named in with in it,
// and says how each was read. The caller closes the trial.
func Supposed(corpus *store.Corpus, with []model.Slug) (*ops.Trial, []Supposition, error) {
	trial, err := ops.Suppose(corpus, with)
	if err != nil {
		return nil, nil, err
	}
	return trial, suppositions(trial.Read), nil
}
