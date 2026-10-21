package api

import (
	"errors"
	"fmt"
	"slices"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// Settling a hypothesis is one act with two answers: accepting takes it in and
// rejecting leaves the corpus as it was, and both are recorded, since which way a
// question was answered is the part a reader in three years needs and the part no file
// would otherwise carry.

// Decided is what settling a hypothesis amounts to.
type Decided struct {
	Hypothesis string   `json:"hypothesis"`
	Status     string   `json:"status"`
	Base       string   `json:"base,omitempty"`
	Commit     string   `json:"commit"`
	Subject    string   `json:"subject"`
	Files      []string `json:"files"`
	Emitted    []string `json:"emitted"`
	Subjects   []string `json:"subjects"`

	// Empty is a decision that changes nothing in the corpus.
	Empty bool `json:"-"`
}

// Decide accepts or rejects a hypothesis.
//
// It goes through history.Owned, unlike with. Reading a hypothesis at a commit is a
// question about what the files said there and is answerable whoever wrote the
// commit; changing the corpus on the strength of one rests on origins having written
// the past, which is the line the rule draws.
//
// The hypothesis and the corpus disagreeing about one fact comes back as the
// ops.Drifted it is, and a corpus that could not be put back as the ops.Unrestored,
// beside whatever it was being put back from; neither leaves a decision standing.
func Decide(asking Asking, slug model.Slug, status model.Status) (*Decided, error) {
	decision, staged, sha, err := ops.Decide(asking.Root, slug, status, asking.asked())
	err = refusing(err, "settle a hypothesis in a corpus")
	var drifted *ops.Drifted
	unrestored, _ := Without[*ops.Unrestored](err)
	if errors.As(err, &drifted) || len(unrestored) > 0 || !landed(sha, err) {
		return nil, err
	}
	report := &Decided{
		Hypothesis: string(decision.Slug),
		Status:     string(decision.Status),
		Base:       decision.Base,
		Commit:     sha,
		Subject:    decision.Entry.Subject,
		Files:      merged(decision.Files, staged.Written()),
		Emitted:    staged.Written(),
		Subjects:   written(decision.Entry.Entities),
		Empty:      staged.Empty() && len(decision.Files) == 0,
	}
	if report.Files == nil {
		report.Files = []string{}
	}
	return report, err
}

func merged(a, b []string) []string {
	out := slices.Concat(a, b)
	slices.Sort(out)
	return slices.Compact(out)
}

// Holding is where one hypothesis stands.
type Holding struct {
	Hypothesis string            `json:"hypothesis"`
	Status     string            `json:"status"`
	Base       string            `json:"base,omitempty"`
	Holds      *bool             `json:"holds,omitempty"`
	Changed    []string          `json:"changed,omitempty"`
	Gone       []string          `json:"gone,omitempty"`
	Into       map[string]string `json:"into,omitempty"`
	Why        string            `json:"why,omitempty"`

	// Reasons is what the corpus has done since that it no longer holds one.
	Reasons []string `json:"-"`
}

// Status lists every hypothesis, or the one named, with where it stands, and asks of
// each one the corpus accepted whether the corpus still holds it.
//
// Accepting one is the corpus changing its mind, and the corpus can change it again
// without anybody saying so: a date the hypothesis settled corrected afterwards leaves
// the argument standing in the history for a conclusion the corpus no longer draws.
// That is a disagreement check cannot see, the corpus agreeing with itself either way,
// and it is asked here because the answer is in the history. It goes through owned,
// since the answer is worth what origins' account of that history is worth.
func Status(root string, named model.Slug) ([]Holding, error) {
	corpus, problems := store.Load(root)
	if problems != nil {
		return nil, unreadable(problems, "origins will not say what a corpus holds when it could not read it whole")
	}
	asked := corpus.HypothesisSlugs()
	if named != "" {
		if _, ours := corpus.Hypotheses[named]; !ours {
			return nil, fmt.Errorf("this corpus has no hypothesis called %s", named)
		}
		asked = []model.Slug{named}
	}
	if _, err := owned(corpus.Root); err != nil {
		return nil, err
	}

	report := []Holding{}
	for _, slug := range asked {
		h := corpus.Hypotheses[slug]
		one := Holding{Hypothesis: string(slug), Status: string(h.Status), Base: h.Base}
		if h.Status == model.Accepted {
			reflection, err := ops.Reflected(corpus, slug)
			if err != nil {
				return nil, err
			}
			one.held(reflection)
		}
		report = append(report, one)
	}
	return report, nil
}

// held writes down what came of asking whether the corpus holds a hypothesis.
func (one *Holding) held(r *ops.Reflection) {
	if r.Why != "" {
		one.Why = r.Why
		return
	}
	holds := r.Holds
	one.Holds = &holds
	if r.Drift == nil {
		return
	}
	one.Changed = r.Drift.Files
	for _, gone := range r.Drift.Gone {
		one.Gone = append(one.Gone, gone.String())
		if into, merged := r.Drift.Into[gone]; merged {
			if one.Into == nil {
				one.Into = map[string]string{}
			}
			one.Into[gone.String()] = into.String()
		}
	}
	one.Reasons = r.Drift.Reasons()
}
