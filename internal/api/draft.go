package api

import (
	"errors"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// A draft is a hypothesis written a change at a time from the pages, which ops.Draft
// says more about. What a page is told of one is what a hypothesis would say of
// itself — its commit and its edits as a hypothesis spells them — and whether it still
// carries onto the corpus in front of it.

// Drafting is a draft as a page is shown it.
type Drafting struct {
	Base    string   `json:"base"`
	Since   int      `json:"since"`
	Edits   []any    `json:"edits"`
	Stated  []string `json:"stated"`
	Records []string `json:"records"`
	Drifted []string `json:"drifted"`
}

// Drafted is the corpus at root with the draft kept for it carried in, and the draft
// as a page is shown it. A draft that no longer carries is shown with what drifted, and
// comes back with no corpus and the ops.Drifted it is; where there is no draft all
// three are nil. The caller closes what it hands back.
func Drafted(corpus *store.Corpus) (*ops.Drafted, *Drafting, error) {
	draft, err := ops.ReadDraft(corpus.Root)
	if errors.As(err, new(ops.Undrafted)) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	report, err := drafting(draft)
	if err != nil {
		return nil, nil, err
	}
	drafted, err := draft.In(corpus)
	var drifted *ops.Drifted
	switch {
	case errors.As(err, &drifted):
		report.Drifted = drifted.Reasons()
		return nil, report, err
	case err != nil:
		return nil, report, err
	}
	report.Since = drafted.Since
	return drafted, report, nil
}

func drafting(draft *ops.Draft) (*Drafting, error) {
	report := &Drafting{Base: draft.Base, Edits: []any{}, Stated: []string{}, Records: []string{}, Drifted: []string{}}
	edits, err := store.EditFields(draft.Edits)
	if err != nil {
		return nil, err
	}
	report.Edits = edits
	seen := map[model.EntityRef]bool{}
	for _, edit := range draft.Edits {
		report.Stated = append(report.Stated, stating(edit))
		for _, ref := range edit.Subjects() {
			if !seen[ref] {
				seen[ref] = true
				report.Records = append(report.Records, ref.String())
			}
		}
	}
	return report, nil
}

// DraftEdit takes one edit into the draft kept for the corpus, beginning one where
// none has been, pinned by Seen to the record as the draft showed it.
func DraftEdit(asking Asking, edit model.Edit) (*Drafting, error) {
	draft, err := ops.DraftEdit(asking.Root, edit, asking.asked())
	if draft == nil {
		return nil, refusing(err, "draft a change to a corpus")
	}
	return draftReport(draft, err)
}

// DraftPop takes the last edit back out of the draft.
func DraftPop(asking Asking) (*Drafting, error) {
	draft, err := ops.DraftPop(asking.Root, "origins "+asking.Command)
	if draft == nil {
		return nil, err
	}
	return draftReport(draft, err)
}

// DraftDiscard throws the draft away.
func DraftDiscard(asking Asking) error {
	return ops.DraftDiscard(asking.Root, "origins "+asking.Command)
}

func draftReport(draft *ops.Draft, err error) (*Drafting, error) {
	report, stated := drafting(draft)
	if stated != nil {
		return nil, errors.Join(err, stated)
	}
	return report, err
}

// Keeping is a draft asked to be kept as a hypothesis.
type Keeping = ops.Keeping

// Keep writes the draft as an open hypothesis pinned to its commit, records it as one
// commit and takes the draft away. What it rests on is the sources the asking names
// where the keeping names none, as a proposal's are.
func Keep(asking Asking, keeping Keeping) (*Proposal, error) {
	if keeping.Question == "" {
		return nil, errors.New("a hypothesis puts a question, and a draft is kept with the one it puts")
	}
	if keeping.Supports == nil {
		keeping.Supports = asking.Sources
	}
	entry, h, sha, err := ops.Keep(asking.Root, keeping, asking.asked())
	err = refusing(err, "keep a draft of a corpus")
	if h == nil || !landed(sha, err) {
		return nil, err
	}
	report := &Proposal{
		Hypothesis: string(h.Slug),
		Base:       h.Base,
		Commit:     sha,
		Subject:    entry.Subject,
		Edits:      []string{},
		Restored:   []string{},
	}
	for _, edit := range h.Edits {
		report.Edits = append(report.Edits, stating(edit))
	}
	return report, err
}
