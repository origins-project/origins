package history

import (
	"errors"
	"strings"
)

// Taking a change back is reverting it and recording the reversal, never rewriting
// what was there.
//
// A corpus's history is the research, and the research includes a conclusion that was
// drawn and then withdrawn. A history that quietly lost its mistakes would still
// answer every question origins puts to it, and would answer some of them wrongly:
// whence would report a conclusion nobody had drawn, and the commit a hypothesis was
// written against would be one that no longer exists.

const reversal = "history.undo"

// Reversing describes the commit that takes another one back: what it took back, and
// whom that was about.
//
// It says what it touched and nothing about what the change it took back rested on.
// Origins-Source and Origins-Hypothesis are what a conclusion was drawn from, and an
// undo draws none: carrying them would have blame report a restored value as resting
// on the citation that argued against it, and why name the very hypothesis whose
// application had just been withdrawn.
//
// The tool is left to the caller, as it is for any other commit — origins is not the
// only thing that could ever drive this, and nothing here knows what version of it is
// running.
func Reversing(e Entry) Commit {
	return Commit{
		Subject:  `Undo "` + e.Subject + `"`,
		Op:       reversal,
		Entities: e.Entities,
		Events:   e.Events,
		Corrects: e.Sha,
	}
}

// Touched lists the files a commit changed, named the way the corpus names them.
//
// The paths are relative to the corpus rather than to the repository, which is what
// every other path origins handles is relative to and what git is run with here. A
// corpus kept in a subdirectory of a larger repository is what needs the distinction,
// and a commit origins wrote never reaches outside the corpus, so nothing of one is
// left out by asking only about what lies under it.
//
// The first commit of a history has to be asked for as one, because git describes a
// commit by comparing it to its parent and that one has none. It is as fair a thing to
// be asked about as any other: an import somebody thought better of is exactly the
// change to want back.
func (r *Repo) Touched(sha string) ([]string, error) {
	out, err := r.git("diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "--relative", sha)
	if err != nil || out == "" {
		return nil, err
	}
	return strings.Split(out, "\n"), nil
}

// Undo reverts a commit and records the reversal as one of origins' own.
//
// The revert is made without a commit of git's, because the commit has to be origins':
// it carries the trailers saying what was taken back and whom it was about, and one
// git wrote would read, to the next command, as somebody else's work entirely.
//
// Nothing is added to the index here, where every other commit adds what it wrote,
// because the revert has already staged everything it changed. A file it deleted is
// then in neither the working tree nor the index, and asking git to add that back
// would do nothing but fail.
//
// A revert git will not make leaves the corpus alone and says why; the usual reason is
// a file somebody is part way through editing by hand. A commit git will not make is
// the worse case, since the reversal is by then sitting in the working tree unrecorded,
// and abandoning the revert is what takes it out again. That puts back what the revert
// touched and nothing else, so a file being edited beside it is still being edited
// afterwards.
func (r *Repo) Undo(sha string, paths []string, c Commit) (string, error) {
	if err := c.valid(); err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", errors.New("that commit changed nothing in this corpus, so there is nothing to take back")
	}
	if _, err := r.git("revert", "--no-commit", "--no-edit", sha); err != nil {
		return "", err
	}
	written, made, err := r.land(c, paths)
	if err != nil && !made {
		return "", errors.Join(err, r.abandon())
	}
	return written, err
}

// abandon takes a revert back out of the working tree. Git keeps the sequence it was
// part way through, so there is still a revert to abandon once the commit it was
// headed for has been refused.
func (r *Repo) abandon() error {
	_, err := r.git("revert", "--abort")
	return err
}
