package history

import (
	"fmt"
	"slices"
	"strings"
)

// Origins owns the history of a corpus, and owning it means being able to say what
// every commit in it was for. This file is where that claim is tested.
//
// What the claim buys is every other answer origins gives about the past: blame reads
// a fact back to the commit that wrote it, why reads that commit back to the argument
// behind it, whence collects the conclusions drawn from a source, and a hypothesis is
// pinned to the commit it was written against. A commit origins cannot account for
// puts all four in doubt at once, and none of them would report the doubt — they would
// simply answer, and answer wrongly.

// Ground is what became of the commit origins last recorded in this clone.
type Ground int

const (
	// Kept is HEAD standing on that commit or on something descended from it, which
	// is every ordinary state including one somebody has committed to by hand.
	Kept Ground = iota

	// Behind is HEAD standing below it: a commit checked out on its own, or a bisect
	// part way through. Nothing has been lost, and what origins wrote is still ahead.
	Behind

	// Lost is HEAD standing neither on it nor after it. An amend, a rebase, a reset
	// onto another line — the commits origins wrote are not in this history any more,
	// which is what every sha written down elsewhere was pointing at.
	Lost
)

func (g Ground) String() string {
	switch g {
	case Behind:
		return "behind"
	case Lost:
		return "lost"
	}
	return "kept"
}

// Standing is where origins stands with a corpus's history.
type Standing struct {
	// Anchor is the newest commit origins wrote, or nil where it wrote none.
	Anchor *Entry

	// Foreign is everything committed on top of the anchor, newest first. Origins
	// wrote none of it, by the definition of the anchor.
	Foreign []Entry

	// Marker is what the local ref says origins last wrote, and Ground is what became
	// of it. The ref is empty in a fresh clone, which is an answer and not a gap.
	Marker string
	Ground Ground

	// Dirty is the corpus files that have been changed and not recorded. It is not a
	// divergence — a commit names its paths, so a dirty tree is never swept into one —
	// but it is the other half of what somebody asking about the state wants to know.
	Dirty []string

	// Problems is commits that could not be read. One of them must not take the whole
	// answer away, so it travels beside the answer.
	Problems error
}

// Settled reports a history origins can account for the whole of.
//
// The dirty tree is deliberately not part of it. Hand-editing a file is the escape
// hatch the format exists to leave open, and a command that names the paths it commits
// cannot sweep the edit up by accident; a commit somebody made is the thing origins
// has no description of.
func (s Standing) Settled() bool { return len(s.Foreign) == 0 && s.Ground == Kept }

// Survey works out where origins stands, asking both of the questions it can ask and
// neither of them twice.
//
// The two have a gap each and cover each other's. The trailers say which commits are
// origins' own work, and travel with the history — they are what still answers in a
// clone that has never run origins. The marker ref says which commit origins last left
// HEAD at, and is the only thing that notices history being rewritten underneath: a
// rebase carries the trailers along with the commits, so every commit still reads as
// origins' own while every sha anybody wrote down has quietly changed.
func (r *Repo) Survey() (Standing, error) {
	anchor, foreign, marked, problems := r.lastWritten()
	standing := Standing{Anchor: anchor, Foreign: foreign, Marker: marked, Problems: problems}

	// a marker on a commit the log read from HEAD is kept, which is where it stands after
	// every change origins makes, and only a marker elsewhere or nowhere is asked after
	var err error
	if marked == "" {
		if standing.Marker, err = r.LastRecorded(); err != nil {
			return Standing{}, err
		}
		if standing.Marker != "" {
			if standing.Ground, err = r.became(standing.Marker); err != nil {
				return Standing{}, err
			}
		}
	}
	if standing.Dirty, err = r.Dirty(); err != nil {
		return Standing{}, err
	}
	return standing, nil
}

// Unsettled is a history origins cannot account for, which Owned refuses. What it says
// is Standing.Describe, and what can be done about it is the caller's to add.
type Unsettled struct{ Standing Standing }

func (u *Unsettled) Error() string { return u.Standing.Describe() }

// Owned opens the repository a corpus's history is kept in, and refuses one whose
// history origins cannot account for.
//
// Everything that reaches for the history comes through here, and everything that
// does not reach for it is untouched. That is the whole of the rule, and it draws the
// line in the one place it can honestly be drawn: what origins says about the past is
// worth exactly as much as its claim to have written the past, so whatever rests on
// that claim is what has to test it.
//
// origins check never opens a repository and so never stops, which is what lets
// git bisect run origins check work at all — a bisect leaves HEAD standing where
// origins never put it, deliberately, for as long as the search takes.
func Owned(dir string) (*Repo, error) {
	repo, err := Open(dir)
	if err != nil {
		return nil, err
	}
	standing, err := repo.Survey()
	if err != nil {
		return nil, err
	}
	if !standing.Settled() {
		return nil, &Unsettled{Standing: standing}
	}
	return repo, nil
}

// became is what has happened to a commit, seen from HEAD.
//
// A commit the repository no longer holds is Lost rather than an error: it is gone,
// which is the answer, and a gc after a rebase is the ordinary way for it to happen.
func (r *Repo) became(sha string) (Ground, error) {
	if _, err := r.git("rev-parse", "--verify", "-q", sha+"^{commit}"); err != nil {
		return Lost, nil
	}
	ahead, err := r.asks("merge-base", "--is-ancestor", sha, "HEAD")
	if err != nil {
		return Kept, err
	}
	if ahead {
		return Kept, nil
	}
	behind, err := r.asks("merge-base", "--is-ancestor", "HEAD", sha)
	if err != nil {
		return Kept, err
	}
	if behind {
		return Behind, nil
	}
	return Lost, nil
}

// Shared reports whether a commit has left this machine, which is the one checkable
// fact that decides whether history may be rewritten.
//
// Reachability from a remote-tracking ref is the test. It answers for what this clone
// has seen of the remote rather than for what the remote holds now, which is the
// honest limit of a question asked without the network: a commit pushed from somewhere
// else and never fetched here cannot be known about, and a commit this clone pushed is
// recorded in the ref it pushed to.
func (r *Repo) Shared(sha string) (bool, error) {
	out, err := r.git("for-each-ref", "--format=%(refname)", "--contains", sha, "refs/remotes")
	return out != "", err
}

// Dirty lists the corpus files that differ from what was last recorded, named the way
// the corpus names them and sorted.
//
// It is one status, which refreshes the index before it compares: without that git
// reports a file whose timestamp has moved as changed, having compared the stat
// information and not the bytes, and on a platform that rewrites line endings on
// checkout that is most of the corpus. A file differing from HEAD in the index or in
// the working tree is dirty, and so is one git has never seen; where nothing has been
// committed there is no HEAD to differ from, and only the files never seen are.
func (r *Repo) Dirty() ([]string, error) {
	prefix, err := r.Prefix()
	if err != nil {
		return nil, err
	}
	out, err := r.output("status", "--porcelain=v2", "-z", "--branch", "--untracked-files=all", "--no-renames",
		"--", ".")
	if err != nil {
		return nil, err
	}
	var dirty []string
	unborn := false
	for _, entry := range strings.Split(string(out), "\x00") {
		var path string
		switch kind, rest, _ := strings.Cut(entry, " "); kind {
		case "#":
			unborn = unborn || rest == "branch.oid (initial)"
		case "?":
			path = rest
		case "1":
			// XY sub mH mI mW hH hI path
			if fields := strings.SplitN(rest, " ", 8); len(fields) == 8 && !unborn {
				path = fields[7]
			}
		case "u":
			// XY sub m1 m2 m3 mW h1 h2 h3 path
			if fields := strings.SplitN(rest, " ", 10); len(fields) == 10 && !unborn {
				path = fields[9]
			}
		}
		if path != "" {
			dirty = append(dirty, strings.TrimPrefix(path, prefix))
		}
	}
	slices.Sort(dirty)
	return slices.Compact(dirty), nil
}

func lines(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// Describe says what is standing between origins and the history it expected, in the
// words somebody reading a refusal needs. It says nothing about a settled history,
// which is what leaves the caller free to ask before it has decided to complain.
func (s Standing) Describe() string {
	var said []string
	switch s.Ground {
	case Behind:
		said = append(said, fmt.Sprintf("this is an older state of the corpus: origins last wrote %s, "+
			"which is ahead of where HEAD is now", short(s.Marker)))
	case Lost:
		said = append(said, fmt.Sprintf("the history has been rewritten: origins last wrote %s, "+
			"which is no longer in it", short(s.Marker)))
	}
	if len(s.Foreign) > 0 {
		said = append(said, unaccounted(s))
		for _, entry := range s.Foreign[:min(len(s.Foreign), shown)] {
			said = append(said, fmt.Sprintf("  %s  %s  (%s)", short(entry.Sha), entry.Subject, entry.Author))
		}
		if rest := len(s.Foreign) - shown; rest > 0 {
			said = append(said, fmt.Sprintf("  and %d more", rest))
		}
	}
	return strings.Join(said, "\n")
}

// shown is how many of the commits to name. Every one of them is in the report a
// caller can ask for; a corpus somebody has kept in git for ten years without origins
// is a perfectly ordinary thing to be told about, and not a thing to be told about
// four hundred times.
const shown = 10

// unaccounted is the line above the list: how many commits origins cannot answer for,
// and where they stand. A history origins wrote no part of is not commits sitting on
// top of anything, there being nothing of its own underneath them.
func unaccounted(s Standing) string {
	switch {
	case s.Anchor == nil && len(s.Foreign) == 1:
		return "the one commit in this history is not origins' own work:"
	case s.Anchor == nil:
		return fmt.Sprintf("origins wrote no part of this history, and all %d of its commits "+
			"are somebody else's work:", len(s.Foreign))
	case len(s.Foreign) == 1:
		return "a commit origins did not write sits on top of the last one it did:"
	}
	return fmt.Sprintf("%d commits origins did not write sit on top of the last one it did:",
		len(s.Foreign))
}

// short is a sha as a reader quotes one back. The commands print shas the same way,
// and a refusal written here is read beside them.
func short(sha string) string {
	if len(sha) < 10 {
		return sha
	}
	return sha[:10]
}
