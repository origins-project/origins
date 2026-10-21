package history

// marker is where a clone records the commit origins last wrote.
//
// It sits outside refs/heads and refs/tags on purpose: git's default refspecs carry
// neither, in either direction, so the ref stays on the machine that made it. Each
// clone answers for what its own origins wrote and never inherits a collaborator's
// idea of where history had got to.
const marker = "refs/origins/head"

// LastRecorded is the commit origins last wrote in this clone, or "" if it has written
// none.
//
// Empty is an ordinary answer rather than a failure: a fresh clone, a corpus somebody
// started by hand, a ref that was deleted. What it means is that this clone has no
// record of its own, not that anything is wrong.
//
// A HEAD that is not this sha is history that moved underneath origins — a commit made
// by hand, an amend, a rebase, a reset, a pull. One ref asks all of them at once.
func (r *Repo) LastRecorded() (string, error) {
	// for-each-ref rather than rev-parse, because a ref that is not there is empty
	// output and success. rev-parse exits non-zero for it, which makes a ref nobody
	// has written yet indistinguishable from a repository that cannot be read.
	return r.git("for-each-ref", "--format=%(objectname)", marker)
}

// LastWritten is the newest commit carrying origins' own trailer, with everything
// committed on top of it since, newest first.
//
// It is the other half of the question LastRecorded puts to the marker ref, and the
// half that still answers after a clone, a deleted ref or a rebase: the ref is one
// machine's memory of its own work, and the trailer is written into the history
// itself, where it travels with the commit.
//
// Nothing written is a history origins wrote no part of, which is an ordinary answer.
// Problems come back the way a log's do, beside whatever could still be read, since
// one commit nobody can parse must not take the answer away.
func (r *Repo) LastWritten() (*Entry, []Entry, error) {
	anchor, foreign, _, problems := r.lastWritten()
	return anchor, foreign, problems
}

// lastWritten is LastWritten saying also the commit the marker is on where it is on one
// of the commits read, which are every commit HEAD reaches.
func (r *Repo) lastWritten() (*Entry, []Entry, string, error) {
	entries, marked, problems := r.reading(Query{})
	for i, entry := range entries {
		if !entry.Foreign() {
			return &entries[i], entries[:i], marked, problems
		}
	}
	return nil, entries, marked, problems
}

// Head is the commit the corpus stands on, or "" where nothing has been committed yet.
//
// Nothing committed is an ordinary state and not a failure: a repository somebody has
// just made is one, and so is a corpus about to be recorded for the first time.
func (r *Repo) Head() (string, error) {
	sha, err := r.git("rev-parse", "--verify", "-q", "HEAD")
	if err != nil {
		return "", nil
	}
	return sha, nil
}

// Mark records sha as what origins last wrote.
//
// Every commit origins makes marks itself, so the callers outside this package are the
// ones that put HEAD somewhere without making a commit to get it there.
func (r *Repo) Mark(sha string) error {
	_, err := r.git("update-ref", marker, sha)
	return err
}
