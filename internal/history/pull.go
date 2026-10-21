package history

import (
	"errors"
	"fmt"
	"strings"
)

// Taking in what another clone recorded. The commits that arrive were each written by
// an origins transaction somewhere else, which is what lets a fast-forward be taken on
// trust; what two clones did apart is joined by git's merge, judged by the caller as an
// adoption is, and recorded as one commit of origins' own.

// Pulled is the operation a pull's merge commit records.
const Pulled = "remote.pull"

// Outside lists the files outside the corpus, in a corpus kept inside a larger
// repository, that are staged or differ from what was last committed, named from the
// top of the repository. A merge writes over the whole working tree and its commit
// takes whatever is staged, so a pull has to know of these as well as of the corpus's
// own.
func (r *Repo) Outside() ([]string, error) {
	prefix, err := r.Prefix()
	if err != nil || prefix == "" {
		return nil, err
	}
	head, err := r.Head()
	if err != nil || head == "" {
		return nil, err
	}
	_, _ = r.git("update-index", "-q", "--refresh")
	out, err := r.git("diff-index", "--name-only", "HEAD", "--", ":/")
	if err != nil {
		return nil, err
	}
	var outside []string
	for _, file := range lines(out) {
		if !strings.HasPrefix(file, prefix) {
			outside = append(outside, file)
		}
	}
	return outside, nil
}

// Merging reports a merge git is part way through, which a pull will not start on top
// of.
func (r *Repo) Merging() (bool, error) {
	return r.asks("rev-parse", "--quiet", "--verify", "MERGE_HEAD")
}

// Incoming is the remote-tracking branch the current branch takes in from a remote,
// with the commit it stands on: the branch's upstream where that is on this remote, and
// the remote's branch of the same name otherwise.
func (r *Repo) Incoming(remote string) (ref, sha string, err error) {
	branch, err := r.git("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return "", "", Detached{}
	}
	name := branch
	if follows, _ := r.git("config", "--get", "branch."+branch+".remote"); follows == remote {
		if merge, err := r.git("config", "--get", "branch."+branch+".merge"); err == nil {
			name = strings.TrimPrefix(merge, "refs/heads/")
		}
	}
	ref = remote + "/" + name
	sha, err = r.git("rev-parse", "--quiet", "--verify", "refs/remotes/"+ref+"^{commit}")
	if err != nil {
		return "", "", fmt.Errorf("%s has no branch %s to take in", remote, name)
	}
	return ref, sha, nil
}

// Above is every commit in a revision's history on top of the newest one origins
// wrote there, newest first, which is what Survey asks of HEAD asked of somewhere else.
func (r *Repo) Above(revision string) ([]Entry, error) {
	entries, problems := r.read(Query{}, revision)
	for i, entry := range entries {
		if !entry.Foreign() {
			return entries[:i], problems
		}
	}
	return entries, problems
}

// Descends reports whether the commit ancestor is in the history of the commit
// descendant, a commit being in its own.
func (r *Repo) Descends(descendant, ancestor string) (bool, error) {
	return r.asks("merge-base", "--is-ancestor", ancestor, descendant)
}

// Count is how many commits revision has that HEAD does not.
func (r *Repo) Count(revision string) (int, error) {
	return r.count(revision, "--not", "HEAD")
}

// Beyond is how many commits HEAD has that revision does not: how far the history has
// gone on since it.
func (r *Repo) Beyond(revision string) (int, error) {
	return r.count("HEAD", "--not", revision)
}

func (r *Repo) count(args ...string) (int, error) {
	out, err := r.git(append([]string{"rev-list", "--count"}, args...)...)
	if err != nil {
		return 0, err
	}
	var n int
	_, err = fmt.Sscanf(out, "%d", &n)
	return n, err
}

// FastForward moves the branch on to a commit descended from where it stands, and
// marks it as what origins last wrote: every commit it passes was written by origins
// somewhere, so there is nothing in them to account for.
func (r *Repo) FastForward(revision string) (string, error) {
	if _, err := r.git("merge", "--ff-only", "--quiet", revision); err != nil {
		return "", err
	}
	sha, err := r.git("rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return sha, r.Mark(sha)
}

// Merge joins a revision into the working tree and the index without committing, so
// that the result can be read and judged before anything is recorded. The files with
// lines changed on both sides are handed back, named from the corpus, or from the top
// of the repository where they are all outside it, with the merge abandoned and the
// repository as it was.
func (r *Repo) Merge(revision string) (conflicts []string, err error) {
	_, err = r.git("merge", "--no-commit", "--no-ff", "--quiet", revision)
	if err == nil {
		return nil, nil
	}
	if merging, _ := r.Merging(); !merging {
		return nil, err
	}
	out, listed := r.git("diff", "--name-only", "--relative", "--diff-filter=U")
	if listed == nil && out == "" {
		// lines changed on both sides outside the corpus, named from the top instead
		out, listed = r.git("diff", "--name-only", "--diff-filter=U")
	}
	if listed != nil || out == "" {
		return nil, errors.Join(err, r.AbortMerge())
	}
	return lines(out), r.AbortMerge()
}

// AbortMerge abandons a merge, putting the working tree and the index back as they
// were before it, which is exactly the state a pull begins from: nothing staged and
// nothing modified.
func (r *Repo) AbortMerge() error {
	_, err := r.git("merge", "--abort")
	return err
}

// Conclude records a merge as one of origins' own commits, with both parents.
//
// It is the one commit that takes the whole index rather than the paths it names, git
// refusing any other while a merge is part way through. What the index holds is the
// merge and what the caller re-emitted in the corpus, which is added here, and nothing
// else, because a pull does not begin while anything in the repository is staged.
func (r *Repo) Conclude(c Commit) (string, error) {
	if err := c.valid(); err != nil {
		return "", err
	}
	whole := []string{"."}
	if err := r.unignored(whole); err != nil {
		return "", err
	}
	if _, err := r.git("add", "--all", "--", "."); err != nil {
		return "", err
	}
	sha, _, err := r.land(c, nil)
	return sha, err
}
