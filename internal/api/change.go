package api

import (
	"errors"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
)

// The changes are the edit vocabulary and nothing besides. There is no add-person
// taking a field at a time: the same nine operations serve a person, a place and a
// source, they are the operations a hypothesis is written in, and a tenth cannot
// appear without a tenth word to go with it.
//
// Everything goes through the one applier and lands as one commit, so a change typed
// at a command, sent from a page and accepted from a hypothesis reach the corpus by
// the same road and are recorded in the same terms.

// Asking is what every change takes besides the change itself: the corpus, what the
// lock is to say is writing, and what the commit says beside the change.
type Asking struct {
	Root    string
	Command string // the command as typed, "set" or "hypothesis accept"
	Message string // the subject line, in place of the one origins writes
	Sources []model.CitationRef
	DryRun  bool
	Seen    string // the record as whoever asked was shown it, where it was
}

// asked is how the change was asked for, as the change records it.
func (a Asking) asked() ops.Asked {
	return ops.Asked{
		Tool:    Tool,
		Command: "origins " + a.Command,
		Subject: a.Message,
		Sources: a.Sources,
		DryRun:  a.DryRun,
		Seen:    a.Seen,
	}
}

// Recorded is what a change amounts to.
type Recorded struct {
	Op      string    `json:"op"`
	Subject string    `json:"subject"`
	Commit  string    `json:"commit"`
	Written []string  `json:"written"`
	Moved   []Carried `json:"moved"`
	Removed []string  `json:"removed"`

	// Empty is a change to nothing, the corpus already saying it.
	Empty bool `json:"-"`
}

// Carried is one thing the change called something else.
type Carried struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Carry makes one edit to a corpus and records it as a commit, or for a dry run says
// what it would write.
func Carry(asking Asking, edit model.Edit) (*Recorded, error) {
	entry, staged, sha, err := ops.Carry(asking.Root, edit, asking.asked())
	err = refusing(err, "change a corpus")
	if !landed(sha, err) {
		return nil, err
	}
	report := &Recorded{
		Op:      entry.Op,
		Subject: entry.Subject,
		Commit:  sha,
		Written: staged.Written(),
		Moved:   []Carried{},
		Removed: staged.Removed(),
		Empty:   staged.Empty(),
	}
	for _, moved := range staged.Moved() {
		report.Moved = append(report.Moved, Carried{From: moved.From, To: moved.To})
	}
	if report.Removed == nil {
		report.Removed = []string{}
	}
	return report, err
}

// landed is whether a change stands, in the three answers every change that records
// one gives: refused, done, or done with something going wrong after the commit was
// made, which taking back would not answer. A lock not given up is beside the point,
// being about the next origins.
//
// A corpus that contradicts itself is a finding about the records rather than a
// mistake in how origins was asked, and it is the one thing recording will not do:
// what is wrong has to be put right where it is wrong.
func landed(sha string, err error) bool {
	_, rest := Without[*history.Unreleased](err)
	var contradiction *ops.Contradiction
	return rest == nil || sha != "" && !errors.As(rest, &contradiction)
}

// Reversal is an undo: the commit origins made, and the commit it took back.
type Reversal struct {
	Commit   string   `json:"commit"`
	Subject  string   `json:"subject"`
	Undid    Entry    `json:"undid"`
	Files    []string `json:"files"`
	Problems []string `json:"problems"`
}

// Undo takes back the last change origins made, which ops.Undo says more about.
// Nothing is validated afterwards, and check is one question away.
func Undo(asking Asking) (*Reversal, error) {
	taken, err := ops.Undo(asking.Root, asking.asked())
	if taken == nil {
		return nil, err
	}
	// the reversal is in history; what failed came after it, and taking back the
	// one part that did work is not the answer
	report := &Reversal{
		Commit:   taken.Sha,
		Subject:  taken.Entry.Subject,
		Undid:    described(taken.Undid),
		Files:    taken.Files,
		Problems: Flatten(taken.Problems),
	}
	if report.Files == nil {
		report.Files = []string{}
	}
	return report, err
}
