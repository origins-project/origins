package history

// Restore puts the corpus back to the state last recorded.
//
// It is the way out of a change origins put in the files and then decided not to
// record: a patch that turned out not to apply, a rehearsal, a result the corpus
// would contradict. Carrying a patch is the one change origins makes that a
// transaction cannot take back, the files having been written by git rather than by
// the writer that remembers what was in them, so the answer is to ask git.
//
// Three commands, because each undoes one of the three things a three-way merge
// leaves behind and none of them undoes another: the index holds what the merge
// staged and what a conflict left half-merged, the working tree holds the files it
// wrote, and a file the patch created is in neither once the index has gone back.
//
// It is right only where nothing unrecorded stood to be lost, which is why the one
// command that uses it refuses to begin until the corpus is clean. What an ignore
// rule keeps out of the history is left where it is, git declining to remove it
// unless it is asked twice: a scratch directory somebody keeps beside a corpus is
// not origins' to take away.
func (r *Repo) Restore() error {
	if _, err := r.git("reset", "-q", "--", "."); err != nil {
		return err
	}
	if _, err := r.git("checkout", "-f", "--", "."); err != nil {
		return err
	}
	_, err := r.git("clean", "-fdq", "--", ".")
	return err
}
