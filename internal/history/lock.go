package history

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// lockFile is the name of the lock inside the git directory. Kept there, it is never a
// file under a kind directory, never beside a record and never committed, and every
// checkout of the repository has the one lock for the one working tree it writes.
const lockFile = "origins.lock"

// Locked is a lock somebody else holds: the file, and who wrote it as far as the file
// says.
type Locked struct {
	Path    string
	Pid     string
	Command string
}

func (l *Locked) Error() string {
	holder := "another origins"
	if l.Pid != "" {
		holder = fmt.Sprintf("process %s", l.Pid)
		if l.Command != "" {
			holder += fmt.Sprintf(" (%s)", l.Command)
		}
	}
	return fmt.Sprintf("%s is writing this corpus: %s is held, and if no origins is running "+
		"it is left over and can be removed", holder, l.Path)
}

// Lock takes the one right to write the corpus and hands back what gives it up.
//
// Two writers at once is a server left running and a command typed beside it, and git's
// own index.lock stands between them only for the commit, not for the files written
// before it: each would read the corpus, the second would write its change over a state
// the first had already moved on from, and the history would record the first change
// taken back by a commit that says it made a different one. A writer takes this before
// it reads the corpus and gives it up once it has recorded what it wrote.
//
// A lock already there is refused and never waited for, the way git refuses over
// index.lock. A writer waiting would carry on over a corpus it read before the one it
// waited for had changed it, and one that crashed leaves its lock behind for somebody
// to see: the refusal says which process wrote it and that the file can go if none is
// running.
func (r *Repo) Lock(command string) (release func() error, err error) {
	path, err := r.GitPath(lockFile)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return nil, held(path)
	}
	if err != nil {
		return nil, fmt.Errorf("origins could not take the lock on this corpus: %w", err)
	}
	_, err = fmt.Fprintf(file, "%d\n%s\n", os.Getpid(), command)
	if closed := file.Close(); err == nil {
		err = closed
	}
	if err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("origins could not take the lock on this corpus: %w", err)
	}
	return func() error {
		if err := os.Remove(path); err != nil {
			return &Unreleased{Path: path, Err: err}
		}
		return nil
	}, nil
}

// Unreleased is a lock its holder could not give up. Whatever the holder did under it
// is done, and this is about the next origins, which will refuse until the file is
// removed — a warning to pass on, and not a reason to count the work as failed.
type Unreleased struct {
	Path string
	Err  error
}

func (u *Unreleased) Error() string {
	return fmt.Sprintf("the lock on this corpus was not given up, and the next origins "+
		"will refuse until %s is removed: %v", u.Path, u.Err)
}

func (u *Unreleased) Unwrap() error { return u.Err }

// held reads what the holder wrote into the lock. A lock just taken may not have been
// written yet, and one somebody made by hand says nothing, so what it lacks is left out
// of the refusal rather than refusing any differently.
func held(path string) *Locked {
	locked := &Locked{Path: path}
	content, err := os.ReadFile(path)
	if err != nil {
		return locked
	}
	pid, command, _ := strings.Cut(strings.TrimRight(string(content), "\r\n"), "\n")
	locked.Pid, locked.Command = strings.TrimSpace(pid), strings.TrimSpace(command)
	return locked
}
