package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// A corpus has to begin, and beginning is not something the first change can do on the
// way past. The manifest is the one file that makes a directory a corpus, and a change
// records the files it changes and no others, so a corpus whose first commit changed
// anything but the manifest would have it standing unrecorded while every commit
// recorded a corpus that cannot be read back — and nothing gets out of that state,
// since reading the last commit is how a change made by hand is judged and how a
// hypothesis is taken in.
//
// The other half is the repository. Every change records a commit, so the corpus has
// to be in git before any of them will run, and asking a researcher to know that
// before they have a corpus is asking them to know the wrong thing first.

// Begun is what bringing a corpus into being amounts to.
type Begun struct {
	Corpus  string   `json:"corpus"`
	Name    string   `json:"name"`
	Commit  string   `json:"commit"`
	Subject string   `json:"subject"`
	Files   []string `json:"files"`
	History string   `json:"history"`
	Started bool     `json:"started"`
	Waiting []string `json:"waiting"`
}

// Init starts a corpus at root called name, or after the directory where name is
// empty, and records that it started.
//
// It does not go through owned, and for the reason an import does not: the corpus did
// not exist a moment ago, so whatever the repository holds is not about it and there is
// nothing for the test of ownership to be about. The manifest's absence is what says
// so, and is the one thing this refuses to write over.
func Init(root, name string) (report *Begun, err error) {
	manifest := filepath.Join(root, store.ProjectFile)
	if _, err := os.Stat(manifest); err == nil {
		return nil, fmt.Errorf("%s is a corpus already: %s says so", root, manifest)
	}

	project := model.Project{Name: name, FormatVersion: model.FormatVersion}
	if project.Name == "" {
		project.Name = called(root)
	}
	header, err := store.EmitProject(project)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", store.ProjectFile, err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(manifest, header, 0o644); err != nil {
		return nil, err
	}

	report = &Begun{
		Corpus:  root,
		Name:    project.Name,
		Subject: "Start the " + project.Name + " corpus",
		Files:   []string{store.ProjectFile},
	}
	declared, undeclare, err := store.DeclareEndings(root)
	if err != nil {
		return nil, unstarted(manifest, func() error { return nil }, err)
	}
	if declared {
		report.Files = append([]string{store.AttributesFile}, report.Files...)
	}

	repo, made, err := history.Start(root)
	if err != nil {
		return nil, unstarted(manifest, undeclare, err)
	}
	release, err := repo.Lock("origins init")
	if err != nil {
		return nil, unstarted(manifest, undeclare, err)
	}
	defer func() { err = joined(err, release()) }()
	report.Started = made
	if report.History, err = repo.Root(); err != nil {
		return nil, unstarted(manifest, undeclare, err)
	}

	// the manifest and the declaration alone, because they are the whole of the
	// corpus so far. A corpus started inside a repository of notes and photographs
	// would otherwise record the notes and the photographs as well.
	sha, failed := repo.Commit(history.Commit{
		Subject: report.Subject,
		Tool:    Tool,
		Op:      "corpus.init",
	}, report.Files)
	if failed != nil && sha == "" {
		return nil, unstarted(manifest, undeclare, failed)
	}
	report.Commit = sha

	waiting, unlisted := repo.Dirty()
	report.Waiting = waiting
	if report.Waiting == nil {
		report.Waiting = []string{}
	}
	return report, errors.Join(failed, unlisted)
}

// unstarted is a corpus that could not be recorded, having taken the manifest back off
// the disk first and the declaration back as it was.
//
// A manifest with no commit behind it is the state Init exists to prevent, and it is
// one Init itself would then refuse to start again. A repository it may have just made
// is left where it is: an empty one costs nothing and is opened rather than remade next
// time, and telling it apart from one that was already there is not worth being wrong
// about.
func unstarted(manifest string, undeclare func() error, cause error) error {
	if err := undeclare(); err != nil {
		cause = fmt.Errorf("%w\n  and %s could not be put back as it was: %v",
			cause, store.AttributesFile, err)
	}
	if err := os.Remove(manifest); err != nil {
		return fmt.Errorf("%w\n  and %s is still there, with no history behind it: %v",
			cause, manifest, err)
	}
	return fmt.Errorf("%w\n  nothing has been kept, so origins init will start it again", cause)
}

// called is what to call a corpus nobody has named: the directory's own name, resolved
// first, since the corpus is as often as not the directory origins was run in.
func called(root string) string {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return filepath.Base(root)
	}
	return filepath.Base(absolute)
}
