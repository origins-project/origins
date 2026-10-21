package history

import (
	"fmt"
	"strings"
	"time"
)

// Snapshot is a state of a corpus somebody thought worth a name.
//
// It is a tag and nothing more elaborate, so that git and everything else that reads
// a git repository see it as well: a state shared with a relative has to be one they
// can be pointed at without origins in the way.
type Snapshot struct {
	Name    string
	Sha     string // the commit it names
	When    time.Time
	Message string
}

// Take names the commit at the head of the history.
//
// The tag is annotated because taking a snapshot is an act with a date of its own:
// what was sent to a relative in March is that state of the corpus as of March,
// whenever the last change to it happened to have been made. A lightweight tag would
// have only the commit's date to offer, which is a different fact.
//
// A message is what the state was named for and origins cannot guess it, so an empty
// one is left empty rather than filled with a sentence that says nothing.
func (r *Repo) Take(name, message string) (Snapshot, error) {
	if _, err := r.git("tag", "-a", "-m", message, "--", name); err != nil {
		return Snapshot{}, err
	}
	taken, err := r.tags("refs/tags/" + name)
	if err != nil {
		return Snapshot{}, err
	}
	if len(taken) != 1 {
		return Snapshot{}, fmt.Errorf("git made the tag %s and then reported %d of them", name, len(taken))
	}
	return taken[0], nil
}

// Snapshots are every snapshot of a corpus, newest first.
//
// Every tag is one. Origins does not keep its tags in a namespace of their own,
// because a snapshot is defined as a tag rather than merely kept as one, and a tag
// somebody made with git is the same statement about the same commit.
func (r *Repo) Snapshots() ([]Snapshot, error) { return r.tags("refs/tags") }

// The fields of one tag. An annotated tag is an object of its own and a lightweight
// one is a name for the commit, which is what the type is asked for: the commit is
// then the tag's own object, the date is the commit's, and there is no message
// anywhere, since what %(contents) would give back is the commit's subject.
const snapshotFormat = "%(refname:short)%00%(objecttype)%00%(objectname)%00" +
	"%(*objectname)%00%(creatordate:iso-strict)%00%(contents:subject)"

func (r *Repo) tags(pattern string) ([]Snapshot, error) {
	out, err := r.git("for-each-ref", "--sort=-creatordate", "--format="+snapshotFormat, pattern)
	if err != nil || out == "" {
		return nil, err
	}

	var snapshots []Snapshot
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, fieldEnd)
		if len(fields) < 6 {
			return nil, fmt.Errorf("git described a tag in %d fields, not 6", len(fields))
		}
		taken := Snapshot{Name: fields[0], Sha: fields[2]}
		if annotated := fields[1] == "tag"; annotated {
			taken.Sha, taken.Message = fields[3], fields[5]
		}
		when, err := time.Parse(time.RFC3339, fields[4])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", taken.Name, err)
		}
		taken.When = when
		snapshots = append(snapshots, taken)
	}
	return snapshots, nil
}
