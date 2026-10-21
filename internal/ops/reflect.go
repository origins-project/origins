package ops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Reflection is whether the corpus still holds what a hypothesis it accepted
// concluded.
//
// Accepting one makes it the corpus, and nothing after that stops somebody changing
// the fact it settled — a date corrected again, a record it made taken apart. The
// argument then stands in the history for a conclusion the corpus has left behind, and
// why still points at it. Nothing about that is a contradiction check can see, since
// the corpus agrees with itself either way; it is the corpus disagreeing with its own
// reasons, and only the history can say what those were.
type Reflection struct {
	Slug  model.Slug
	Base  string   // the commit its edits were carried out at
	Holds bool     // whether carrying them again changes nothing
	Drift *Drifted // and where it does not hold, where the two part company
	Why   string   // or where there is no state to read it at, the clause saying why
}

// Reflected supposes an accepted hypothesis against the corpus the way --with does,
// and reports whether that changes anything.
//
// It is the road accepting took, taken again: the edits are carried out at the commit
// the hypothesis pins and brought forward by git's three-way merge, onto a copy of the
// corpus as it stands. A conclusion the corpus still holds comes through as nothing at
// all, pinning finding every change it would make already made. One somebody has
// changed since comes back as a conflict naming the file, or merges cleanly and puts
// back what they took out, and either way the copy is no longer the corpus. Where there
// is no state to read the hypothesis at there is no reading it against the corpus
// either, which would be the answer to some other question, and the reason is said
// instead.
func Reflected(corpus *store.Corpus, slug model.Slug) (*Reflection, error) {
	h, err := proposed(corpus, slug)
	if err != nil {
		return nil, err
	}
	if h.Status != model.Accepted {
		return nil, fmt.Errorf("%s is %s, and only a hypothesis the corpus accepted is one it can hold", slug, h.Status)
	}
	trial, err := copied(corpus)
	if err != nil {
		return nil, err
	}
	defer func() { _ = trial.Close() }()

	reflection := &Reflection{Slug: slug, Base: h.Base}
	pinned, err := trial.pin(h)
	var (
		elsewhere *Unpinned
		drift     *Drifted
	)
	switch {
	case errors.As(err, &elsewhere):
		reflection.Why = elsewhere.Why
		return reflection, nil
	case errors.As(err, &drift):
		reflection.Drift = drift
		return reflection, nil
	case err != nil:
		return nil, err
	}

	conflicts, err := trial.at.Carry(pinned.Patch)
	if err != nil {
		return nil, fmt.Errorf("%s could not be carried from %s: %w", slug, pinned.Base, err)
	}
	changed := conflicts
	if len(changed) == 0 {
		for _, relative := range pinned.Files {
			if !alike(filepath.Join(corpus.Root, filepath.FromSlash(relative)),
				filepath.Join(trial.root, filepath.FromSlash(relative))) {
				changed = append(changed, relative)
			}
		}
	}
	if len(changed) > 0 {
		reflection.Drift = &Drifted{Slug: slug, Base: pinned.Base, Files: changed}
		return reflection, nil
	}
	reflection.Holds = true
	return reflection, nil
}

// alike reports whether two paths hold the same thing: the same bytes, a directory
// each, or nothing either.
func alike(one, other string) bool {
	a, errA := os.Stat(one)
	b, errB := os.Stat(other)
	switch {
	case errA != nil || errB != nil:
		return os.IsNotExist(errA) && os.IsNotExist(errB)
	case a.IsDir() || b.IsDir():
		return a.IsDir() && b.IsDir()
	}
	x, errA := os.ReadFile(one)
	y, errB := os.ReadFile(other)
	return errA == nil && errB == nil && bytes.Equal(x, y)
}
