package history

import (
	"bytes"
	"fmt"
	"strings"
)

// Carry puts a patch made at one state of a corpus onto whatever the corpus has got
// to, and says where the two could not be reconciled.
//
// Git's three-way merge is the whole of the mechanism. The patch names the blobs it
// was made against, this repository has them, and what somebody changed underneath in
// the meantime either merges cleanly or comes back as a conflict. Nothing here judges
// which: a conflict is an answer about two changes to one fact and not a failure to
// arrive at one, so it is returned rather than raised, and only a patch git could make
// nothing of at all is an error.
//
// The paths in a patch are relative to the corpus and git resolves them against the
// top of the working tree, so a corpus kept a few directories down needs the prefix
// put back. Without it git matches nothing, says it skipped the patch, and exits
// nought — a silence that would read as a hypothesis with no consequences.
//
// Everything is staged first. A three-way merge takes the side it is merging onto from
// the index and refuses a file the index and the working tree disagree about, and a
// file git has never seen is invisible to it until it is added. Pointed at a checkout
// made to be read once and thrown away, that costs nothing. Pointed at a corpus
// somebody is keeping — which is what accepting a hypothesis does — it is safe
// because there was nothing in that corpus left unrecorded: the command that points
// it there refuses to begin until that is so, which is the same thing that lets
// Restore put the whole of it back afterwards.
//
// A patch with nothing in it carries nothing. Git reads an empty patch as input it
// cannot make sense of at all, and a hypothesis whose edits were already true where
// they were written makes exactly that — an answer to the question rather than a
// failure to put it.
func (r *Repo) Carry(patch []byte) ([]string, error) {
	if len(patch) == 0 {
		return nil, nil
	}
	if _, err := r.git("add", "-A", "--", "."); err != nil {
		return nil, err
	}
	within, err := r.Prefix()
	if err != nil {
		return nil, err
	}
	apply := []string{"apply", "--3way"}
	if within != "" {
		apply = append(apply, "--directory="+within)
	}
	refused := r.fed(patch, append(apply, "-")...)
	conflicts, err := r.git("diff", "--name-only", "--diff-filter=U", "--relative")
	if err != nil {
		return nil, err
	}
	if standing := lines(conflicts); len(standing) > 0 {
		return standing, nil
	}
	return nil, refused
}

// fed is a command whose argument is too long to be one: a patch is bytes and reaches
// git the way bytes do, and so does a commit message naming thousands of entities.
func (r *Repo) fed(in []byte, args ...string) error {
	command := r.command(args...)
	command.Stdin = bytes.NewReader(in)

	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		said := strings.TrimSpace(stderr.String())
		if said == "" {
			said = err.Error()
		}
		return fmt.Errorf("git %s: %s", attempted(args), said)
	}
	return nil
}
