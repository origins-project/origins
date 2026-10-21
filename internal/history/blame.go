package history

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Blamed is one line of a file and the commit that last changed it.
//
// The commit is the whole of what a log reads back, trailers and all, because what
// blame is asked is not when a line changed but what the change was: the source it
// rests on, the hypothesis that decided it. The porcelain does not carry those, so
// they are read out of the commits it names.
type Blamed struct {
	Line int
	Text string
	Entry
}

// unrecorded is the sha git gives a line that is in the working tree and in no commit.
const unrecorded = "0000000000000000000000000000000000000000"

// Uncommitted reports a line nothing accounts for: somebody edited the file and origins
// has not been told. It is the first thing to ask, since the rest of the entry then
// describes a commit that does not exist.
func (b Blamed) Uncommitted() bool { return b.Sha == unrecorded }

// Blame reads which commit last changed each line of a file, in file order.
//
// The file is named relative to the corpus, and it is the working tree that is read,
// so a line somebody has just typed comes back uncommitted rather than attributed to
// whatever it replaced.
//
// A commit whose trailers origins cannot read is reported and the lines still come
// back, as they do for a log: which line changed when is answerable whether or not one
// of the commits behind it describes itself in terms origins knows.
func (r *Repo) Blame(file string) ([]Blamed, error) {
	out, err := r.git("blame", "--porcelain", "--", file)
	if err != nil {
		return nil, err
	}
	lines, err := attributed(out)
	if err != nil {
		return nil, err
	}

	named := map[string]bool{}
	for _, line := range lines {
		if !line.Uncommitted() {
			named[line.Sha] = true
		}
	}
	behind, problems := r.commits(slices.Sorted(maps.Keys(named)))
	for i, line := range lines {
		if entry, ok := behind[line.Sha]; ok {
			lines[i].Entry = entry
		}
	}
	return lines, problems
}

// header opens a commit's account of one line of the file: the sha, where the line was
// and where it is now. Between it and the line's own content git writes the commit's
// details, which are read past — the same commit is described once however many of its
// lines survive, and the trailers are not there at all.
var header = regexp.MustCompile(`^([0-9a-f]{40}) [0-9]+ ([0-9]+)`)

// attributed reads the porcelain: a header, then anything git has to say, then the
// line itself behind a tab. The tab is what makes this unambiguous, since a line of a
// corpus can otherwise say whatever it likes.
func attributed(out string) ([]Blamed, error) {
	var (
		lines []Blamed
		at    Blamed
	)
	for _, line := range strings.Split(out, "\n") {
		if text, is := strings.CutPrefix(line, "\t"); is {
			if at.Line == 0 {
				return nil, errors.New("git blame gave a line of the file before saying whose it was")
			}
			at.Text = text
			lines, at = append(lines, at), Blamed{}
			continue
		}
		if found := header.FindStringSubmatch(line); found != nil {
			number, err := strconv.Atoi(found[2])
			if err != nil {
				return nil, fmt.Errorf("git blame put a line at %q: %w", found[2], err)
			}
			at = Blamed{Line: number, Entry: Entry{Sha: found[1]}}
		}
	}
	return lines, nil
}

// commits reads the commits named and nothing else, which is how blame comes by what a
// commit said about itself.
func (r *Repo) commits(shas []string) (map[string]Entry, error) {
	if len(shas) == 0 {
		return nil, nil
	}
	args := append(append([]string{"log", "--no-walk=unsorted", "--no-show-signature"},
		entryShape...), shas...)
	out, err := r.git(args...)
	if err != nil {
		return nil, err
	}
	entries, _, problems := decode(out)
	behind := make(map[string]Entry, len(entries))
	for _, entry := range entries {
		behind[entry.Sha] = entry
	}
	return behind, problems
}
