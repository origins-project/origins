package history

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The three ways back from a history origins cannot account for.
//
// They differ in what they are willing to do to the past. Noting leaves it exactly as
// it is and records that somebody looked; rewinding takes the commits off and leaves
// their content in the working tree; rewriting replaces them with commits that say
// what they were. Which is allowed turns on whether the commits have left this
// machine, because that is the one part of the question that is a fact rather than a
// preference.

// Note records something against a commit without touching it, which is the only way
// to say anything about a commit that has been shared.
//
// The note is appended rather than set, so a note somebody wrote themselves survives
// origins having something to add to it. Notes stay on this machine, as the marker ref
// does: git's default refspecs carry neither, and a review is this clone saying what
// it has looked at.
func (r *Repo) Note(sha, text string) error {
	_, err := r.git("notes", "append", "-m", text, sha)
	return err
}

// Rewind takes commits off the branch and leaves everything they said in the working
// tree, unrecorded.
//
// It is the resolution that destroys no work and keeps no history: what was committed
// is now an edit somebody has made and not yet recorded, which is a state origins
// already knows how to deal with. A corpus in a subdirectory is not a reason to reach
// for anything narrower — a commit is a commit, and taking one off takes all of it.
func (r *Repo) Rewind(sha string) error {
	_, err := r.git("reset", "--mixed", "-q", sha)
	return err
}

// AsOf checks a commit out on its own, somewhere else, and says where the corpus is
// inside it.
//
// It is how origins asks what the corpus used to say. Reading the files is the only
// way to answer that, since a corpus is what its files mean and not what they contain,
// and the model has no way in but the reader — so there has to be a directory for the
// reader to be pointed at.
//
// The checkout is a repository of its own that borrows the corpus's objects through
// git's alternates, rather than a worktree of the corpus's repository: a worktree is
// registered in the corpus's git directory, and a process killed before it discards
// one leaves it there for good, carried into every copy of the corpus anybody makes.
// This one is nothing the corpus's repository knows about, so a checkout nobody took
// away is a directory in the temporary folder and nothing else.
func (r *Repo) AsOf(sha, dir string) (string, error) {
	objects, err := r.git("rev-parse", "--git-path", "objects")
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(objects) {
		if objects, err = filepath.Abs(filepath.Join(r.dir, objects)); err != nil {
			return "", err
		}
	}
	if _, err := r.git("init", "-q", dir); err != nil {
		return "", err
	}
	if err := os.WriteFile(alternates(dir), []byte(filepath.ToSlash(objects)+"\n"), 0o666); err != nil {
		return "", errors.Join(err, r.Discard(dir))
	}
	checkout := &Repo{dir: dir}
	if _, err := checkout.git("checkout", "--detach", "-q", sha); err != nil {
		return "", errors.Join(err, r.Discard(dir))
	}
	within, err := r.Prefix()
	if err != nil {
		return "", errors.Join(err, r.Discard(dir))
	}
	return filepath.Join(dir, filepath.FromSlash(within)), nil
}

// alternates is the file in a checkout AsOf made naming where else its objects are.
func alternates(dir string) string {
	return filepath.Join(dir, ".git", "objects", "info", "alternates")
}

// Discard takes away a checkout AsOf made.
//
// A path with nothing at it is nothing to discard, and saying so is not the same as
// having failed. A directory that is not one of AsOf's checkouts is refused rather
// than deleted, since a path somebody got wrong is not to cost them what is there.
func (r *Repo) Discard(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	if _, err := os.Stat(alternates(dir)); err != nil {
		return fmt.Errorf("%s is no checkout origins made, and is left where it is", dir)
	}
	return os.RemoveAll(dir)
}

// Rewrite replaces commits origins did not write with commits that say what they were,
// keeping each one as its own.
//
// Each is kept rather than the lot being squashed into one, because a researcher's
// sequence of edits is the shape of their thinking and origins has no business
// flattening it. What changes is only what git records about them: the content is
// whatever they said, re-emitted by the caller so that it is canonical, and the
// message gains the trailers it should have had.
//
// The author and the date of each are carried across. Somebody wrote that commit on
// that day, which remains true of the commit that replaces it; the committer becomes
// whoever is running origins, which is what git means by the field and what a rebase
// does with it.
//
// Nothing moves until all of it has worked. The replay happens on a detached HEAD and
// the branch is pointed at the result only at the end, so a failure part way through
// leaves the branch exactly where it was and costs nothing but the checkout.
//
// The commits are taken newest first, as a survey reports them, and replayed in the
// order they were made.
func (r *Repo) Rewrite(anchor string, foreign []Entry, rewritten func(Entry) (Commit, error)) ([]string, error) {
	branch, err := r.git("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return nil, errors.New("origins will not rewrite history from a detached HEAD, " +
			"because there would be no branch to point at the result")
	}
	dirty, err := r.Dirty()
	if err != nil {
		return nil, err
	}
	if len(dirty) > 0 {
		return nil, fmt.Errorf("the corpus has changes nobody has recorded, and rewriting would "+
			"write over them: %s", strings.Join(dirty, ", "))
	}
	if _, err := r.git("checkout", "--detach", "-q", anchor); err != nil {
		return nil, err
	}

	made, err := r.replay(foreign, rewritten)
	if err != nil {
		return nil, errors.Join(err, r.leave(branch))
	}
	if _, err := r.git("branch", "--force", branch, "HEAD"); err != nil {
		return nil, errors.Join(err, r.leave(branch))
	}
	if err := r.leave(branch); err != nil {
		return nil, err
	}
	if len(made) > 0 {
		return made, r.Mark(made[len(made)-1])
	}
	return made, nil
}

// leave puts HEAD back on the branch it was taken off. Forced, because what is being
// left is a checkout origins made itself and there is nothing in it to protect.
func (r *Repo) leave(branch string) error {
	_, err := r.git("checkout", "--force", "-q", branch)
	return err
}

func (r *Repo) replay(foreign []Entry, rewritten func(Entry) (Commit, error)) ([]string, error) {
	var made []string
	for i := len(foreign) - 1; i >= 0; i-- {
		was := foreign[i]

		// the tree of the commit being replaced, entire. What it said outside the
		// corpus is part of what it said, and a rewrite that dropped it would be
		// deleting somebody's work rather than describing it.
		if _, err := r.git("read-tree", "-u", "--reset", was.Sha); err != nil {
			return made, err
		}
		c, err := rewritten(was)
		if err != nil {
			return made, err
		}
		if err := c.valid(); err != nil {
			return made, err
		}
		if err := r.unignored([]string{"."}); err != nil {
			return made, err
		}
		if _, err := r.git("add", "--all", "--", "."); err != nil {
			return made, err
		}

		who, err := r.authorOf(was.Sha)
		if err != nil {
			return made, err
		}
		if err := r.fed([]byte(c.Message()), "commit", "--allow-empty", "--file=-",
			"--author="+who.who, "--date="+who.when); err != nil {
			return made, err
		}
		sha, err := r.git("rev-parse", "HEAD")
		if err != nil {
			return made, err
		}
		made = append(made, sha)
	}
	return made, nil
}

// author is who wrote a commit and when, in the spellings git's own flags take back.
type author struct{ who, when string }

func (r *Repo) authorOf(sha string) (author, error) {
	out, err := r.git("show", "--no-patch", "--no-show-signature", "--format=%an <%ae>%x00%aI", sha)
	if err != nil {
		return author{}, err
	}
	name, when, found := strings.Cut(out, fieldEnd)
	if !found {
		return author{}, fmt.Errorf("git described the author of %s in one field, not 2", short(sha))
	}
	return author{who: name, when: when}, nil
}
