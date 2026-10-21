package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// The road a change takes to the corpus, whoever is asking for it: a command typed, or
// a form sent from a page. There is one of it, so that a change made either way is
// refused for the same reasons and recorded in the same terms, and what differs
// between the two is only how the answer is shown.

// Asked is how a change was asked for: everything the commit recording it says besides
// what the change itself says.
type Asked struct {
	Tool    string              // what is writing, and its version, as Origins-Tool names it
	Command string              // what the lock says is writing, for whoever it refuses
	Subject string              // the subject line in place of the one origins writes
	Sources []model.CitationRef // the citations the change rests on
	DryRun  bool                // work out what the change comes to and write none of it

	// Seen, where it is set, is what the change was asked for about: Seen of the record
	// it addresses as whoever asked was shown it, or for an undo the commit it would
	// take back. A change asked for from a picture of the corpus is refused as Stale
	// once the corpus no longer looks like the picture, since a path is positional and
	// events[1] shown as a death is a burial once somebody adds a baptism before it.
	Seen string
}

// Absent is what Seen says of a record there is none of, which is what a create
// asked for from a page pins: that the name was free.
const Absent = "none"

// Seen is a digest of a record as the corpus holds it, or Absent where it holds none.
// It is taken of what origins would write for the record rather than of the file, so
// that a file somebody has reformatted by hand, saying what it said, is the record
// that was shown.
func Seen(corpus *store.Corpus, ref model.EntityRef) string {
	var written []byte
	var err error
	if ref == model.TheProject {
		written, err = store.EmitProject(corpus.Project)
	} else {
		entity := corpus.Entity(ref.Slug)
		if entity == nil || model.RefOf(entity).Kind != ref.Kind {
			return Absent
		}
		written, err = store.Emit(entity)
	}
	if err != nil {
		return "unwritable"
	}
	sum := sha256.Sum256(written)
	return hex.EncodeToString(sum[:])
}

// Stale is a change refused because what it was asked for about has changed since
// whoever asked was shown it. Nothing is written: what the change meant was worked out
// against something that is no longer there.
type Stale struct{ What string }

func (s *Stale) Error() string {
	return s.What + ", so the change asked for was about something that is no longer there"
}

// holds refuses the change as Stale where the record it addresses is not the one Seen
// says was shown.
func (a Asked) holds(corpus *store.Corpus, ref model.EntityRef) error {
	if a.Seen == "" {
		return nil
	}
	switch now := Seen(corpus, ref); {
	case now == a.Seen:
		return nil
	case a.Seen == Absent:
		return &Stale{What: fmt.Sprintf("there is already a %s", ref)}
	case now == Absent:
		return &Stale{What: fmt.Sprintf("%s has gone since it was shown", ref)}
	}
	return &Stale{What: fmt.Sprintf("%s has changed since it was shown", ref)}
}

// describe puts onto the entry what the asking says about it.
func (a Asked) describe(entry *history.Commit) {
	entry.Tool = a.Tool
	entry.Sources = append(entry.Sources, a.Sources...)
	if a.Subject != "" {
		entry.Subject = a.Subject
	}
}

// Unreadable is a corpus that did not read whole, which nothing writes over. A file
// origins could not parse is not in the model, so writing the model back over the
// directory would record a state that never described it — and the change would look,
// in the history, exactly like one somebody meant.
type Unreadable struct{ Problems error }

func (u *Unreadable) Error() string {
	return fmt.Sprintf("origins will not write a corpus it could not read whole: %v", u.Problems)
}

func (u *Unreadable) Unwrap() error { return u.Problems }

// locked opens the repository a corpus is kept in and takes the lock on it, so that
// nothing else writes the corpus until the release this hands back is called.
//
// Whatever writes reads the corpus after this and never before. A corpus read before
// the lock is one the writer that held it may since have changed, and writing a change
// over it would take that writer's change back without a word.
func locked(dir, command string) (*history.Repo, func() error, error) {
	repo, err := history.Owned(dir)
	if err != nil {
		return nil, nil, err
	}
	release, err := repo.Lock(command)
	if err != nil {
		return nil, nil, err
	}
	return repo, release, nil
}

// opened is locked, and then the corpus read whole.
func opened(dir, command string) (*history.Repo, *store.Corpus, func() error, error) {
	repo, release, err := locked(dir, command)
	if err != nil {
		return nil, nil, nil, err
	}
	corpus, problems := store.Load(dir)
	if problems != nil {
		return nil, nil, nil, released(&Unreadable{Problems: problems}, release)
	}
	return repo, corpus, release, nil
}

// released gives up the lock, and says so beside whatever else went wrong where it
// could not be: every origins after this one refuses until somebody removes it.
func released(err error, release func() error) error {
	if kept := release(); kept != nil {
		return errors.Join(err, kept)
	}
	return err
}

// Carry makes one edit to the corpus at dir and records it as a commit, or for a dry
// run works out what it would write and writes nothing.
//
// What comes back is what Tx.Commit hands back, with the entry the commit says: a
// Contradiction for a change the corpus would disagree with itself over, and an error
// with a sha beside it for a change recorded with something after it gone wrong, which
// is not to be undone. An empty sha and no error is a change that amounted to nothing.
func Carry(dir string, edit model.Edit, asked Asked) (entry history.Commit, change *Change, sha string, err error) {
	repo, corpus, release, err := opened(dir, asked.Command)
	if err != nil {
		return history.Commit{}, nil, "", err
	}
	defer func() { err = released(err, release) }()

	if err = asked.holds(corpus, edit.Subjects()[0]); err != nil {
		return history.Commit{}, nil, "", err
	}
	tx := Begin(corpus)
	if entry, err = tx.Perform(edit); err != nil {
		return history.Commit{}, nil, "", err
	}
	asked.describe(&entry)
	if asked.DryRun {
		change, err = tx.Stage()
		return entry, change, "", err
	}
	change, sha, err = tx.Commit(repo, entry)
	return entry, change, sha, err
}

// Reversal is the last change origins made, taken back: the commit that made it, the
// files that puts back, and the commit that records it.
type Reversal struct {
	Undid    history.Entry
	Entry    history.Commit
	Files    []string
	Problems error // commits whose trailers would not read, found on the way to it
	Sha      string
}

// Undo takes back the last change origins made in the corpus at dir, or for a dry run
// says which change that would be.
//
// It is the last commit as well, and not by coincidence: a commit origins cannot
// account for is refused before anything is asked, so there is nothing between it and
// the change it is taking back. The corpus is never read, what is put back being what
// a commit said, and nothing is validated afterwards: what an undo restores is a state
// origins itself wrote, and refusing it for contradicting itself would refuse at
// exactly the moment getting out of a bad change matters most.
//
// Asked.Sources are not recorded. They are what a conclusion is drawn from, and an
// undo draws none.
func Undo(dir string, asked Asked) (taken *Reversal, err error) {
	repo, release, err := locked(dir, asked.Command)
	if err != nil {
		return nil, err
	}
	defer func() { err = released(err, release) }()

	// nothing written at all is a history with no commits in it, which is where a
	// corpus starts
	wrote, _, problems := repo.LastWritten()
	switch {
	case wrote == nil && problems != nil:
		return nil, problems
	case wrote == nil:
		return nil, errors.New("origins has recorded nothing here, so there is nothing of its own to take back")
	}
	if asked.Seen != "" && asked.Seen != wrote.Sha {
		return nil, &Stale{What: "origins has recorded another change since the last one was shown"}
	}
	if wrote.Op == history.Pulled {
		// reverting a merge takes one side's work out of the branch for good, git
		// counting it merged ever after, so another clone's changes are taken back
		// one at a time with the edit commands, which say what they take back
		return nil, errors.New("the last change origins recorded is a pull, which joined another " +
			"clone's work to this one and is not taken back by undo")
	}
	files, err := repo.Touched(wrote.Sha)
	if err != nil {
		return nil, err
	}
	taken = &Reversal{Undid: *wrote, Entry: history.Reversing(*wrote), Files: files, Problems: problems}
	taken.Entry.Tool = asked.Tool
	if asked.Subject != "" {
		taken.Entry.Subject = asked.Subject
	}
	if asked.DryRun {
		return taken, nil
	}
	// a sha beside an error is a reversal in history with something after it gone
	// wrong, and taking back the one part that did work is not the answer
	taken.Sha, err = repo.Undo(wrote.Sha, files, taken.Entry)
	if err != nil && taken.Sha == "" {
		return nil, err
	}
	return taken, err
}
