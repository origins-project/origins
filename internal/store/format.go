package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/origins-project/origins/internal/model"
)

// Later says so when a corpus is written in a later format than the one this origins
// writes, and says nothing otherwise.
//
// Reading one is worth attempting, since the formats may differ in nothing the corpus
// uses and what cannot be read is reported file by file anyway. Writing one is not:
// every write re-emits what it touches, so an older origins would put the corpus back
// into the older format a file at a time, and nothing would say that it had.
func Later(p model.Project) error {
	if p.FormatVersion > model.FormatVersion {
		return fmt.Errorf("%s says format-version %d, which is later than the %d this "+
			"origins reads and writes", ProjectFile, p.FormatVersion, model.FormatVersion)
	}
	return nil
}

// Format re-emits every file of a corpus canonically and reports the ones whose bytes
// change, in sorted order and relative to the corpus root. When write is false nothing
// on disk is touched, which is how a corpus is checked for canonical form without being
// altered.
//
// A file that did not load is not in the corpus and is left exactly as it is. Emitting
// it would mean emitting a model that never held its contents, which is the one way a
// formatter can destroy the thing it was pointed at.
//
// A corpus in a later format is refused whole, asked only to list or not: what is
// canonical is a question about the format, and this origins knows only its own.
func Format(corpus *Corpus, write bool) ([]string, error) {
	if err := Later(corpus.Project); err != nil {
		return nil, fmt.Errorf("%w, so it cannot say what that format's canonical form is", err)
	}
	var (
		root     = corpus.Root
		problems error
		changed  []string
	)
	rewrite := func(relative string, canonical []byte) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		current, err := os.ReadFile(path)
		if err != nil {
			problems = errors.Join(problems, fmt.Errorf("%s: %w", relative, err))
			return
		}
		if bytes.Equal(current, canonical) {
			return
		}
		changed = append(changed, relative)
		if !write {
			return
		}
		if err := os.WriteFile(path, canonical, 0o644); err != nil {
			problems = errors.Join(problems, fmt.Errorf("%s: %w", relative, err))
		}
	}

	// origins.yaml is only re-emitted when it read cleanly. An unreadable one leaves
	// the project zero-valued, and writing that out would delete the header.
	if !corpus.Project.IsZero() {
		if project, err := EmitProject(corpus.Project); err != nil {
			problems = errors.Join(problems, fmt.Errorf("%s: %w", ProjectFile, err))
		} else {
			rewrite(ProjectFile, project)
		}
	}

	for _, slug := range corpus.Slugs() {
		relative, ok := corpus.Path(slug)
		if !ok {
			continue
		}
		canonical, err := Emit(corpus.Entity(slug))
		if err != nil {
			problems = errors.Join(problems, fmt.Errorf("%s: %w", relative, err))
			continue
		}
		rewrite(relative, canonical)
	}
	slices.Sort(changed)
	return changed, problems
}
