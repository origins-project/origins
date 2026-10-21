package history

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Repo is the git repository a corpus lives in.
//
// Commands run with the corpus directory as their working directory rather than the
// repository root, so a corpus kept in a subdirectory of a larger repository works
// without anything here having to know that it is.
type Repo struct {
	dir string

	// where the repository is, which git is asked once: its top, the corpus's place
	// under it, and the git directory
	locating sync.Once
	top      string
	prefix   string
	gitDir   string
	located  error
}

// Open confirms a corpus directory is somewhere origins can record history. Origins
// owns the history of a corpus, and a directory that is not versioned has none to own.
func Open(dir string) (*Repo, error) {
	repo, err := available(dir)
	if err != nil {
		return nil, err
	}
	if _, err := repo.Root(); err != nil {
		return nil, fmt.Errorf("origins records a corpus's history in git: %w", err)
	}
	return repo, nil
}

// Start opens the repository a corpus that has just been written will be recorded in,
// making one where there is none, and says which of the two it did.
//
// A corpus written inside a repository somebody already keeps is recorded in that one,
// which is what lets a tree live beside the notes, the photographs and the README it
// belongs with. Only a corpus in no repository at all gets one of its own: a repository
// nested inside another is invisible to it, so a history started there would be lost
// the first time anyone cloned the outer one.
func Start(dir string) (repo *Repo, made bool, err error) {
	if repo, err := Open(dir); err == nil {
		return repo, false, nil
	}
	repo, err = available(dir)
	if err != nil {
		return nil, false, err
	}
	if _, err := repo.git("init", "-q"); err != nil {
		return nil, false, err
	}
	return repo, true, nil
}

// Git is where the git origins runs is and the version it says it is, as
// "2.51.0.windows.1", or an error a reader can be shown after "git" where none is on
// the path or the one there will not answer.
func Git() (path, version string, err error) {
	path, err = exec.LookPath("git")
	if err != nil {
		return "", "", errors.New("not found on the path")
	}
	out, err := exec.Command(path, "version").Output()
	if err != nil {
		return path, "", fmt.Errorf("%s did not say its version: %w", path, err)
	}
	said := strings.TrimSpace(string(out))
	return path, strings.TrimPrefix(said, "git version "), nil
}

func available(dir string) (*Repo, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("origins keeps a corpus's history in git, and git is not on the path: %w", err)
	}
	return &Repo{dir: dir}, nil
}

// GitPath is where a file of origins' own is kept for this repository, absolute: in
// the git directory of the working tree the corpus is in, beside the lock, so never a
// file of the corpus. It is where git rev-parse --git-path puts a name git has no use
// for itself.
func (r *Repo) GitPath(name string) (string, error) {
	r.locate()
	return filepath.Join(r.gitDir, name), r.located
}

// Root is the top of the repository the corpus is in, which is not always the corpus
// directory.
func (r *Repo) Root() (string, error) {
	r.locate()
	return r.top, r.located
}

// Prefix is where the corpus is under the top of the repository, as git names a path:
// with forward slashes and a trailing one, and empty where the corpus is the top.
func (r *Repo) Prefix() (string, error) {
	r.locate()
	return r.prefix, r.located
}

// locate asks git where the repository is, once for as long as the Repo is kept: a
// repository does not move under a command, and every command asks.
func (r *Repo) locate() {
	r.locating.Do(func() {
		out, err := r.output("rev-parse", "--show-toplevel", "--show-prefix", "--absolute-git-dir")
		if err != nil {
			r.located = err
			return
		}
		// one line each, the prefix an empty one where the corpus is the top
		where := strings.SplitN(strings.TrimRight(string(out), "\n"), "\n", 3)
		if len(where) != 3 {
			r.located = fmt.Errorf("git rev-parse could not say where the repository is: %q", out)
			return
		}
		r.top, r.prefix, r.gitDir = where[0], where[1], filepath.FromSlash(where[2])
	})
}

// Has reports whether a commit is one this repository can reach, which is a question
// worth asking before anything is done that would fail for a dozen reasons and be
// able to name only the last of them.
//
// Anything a commit can be named by answers: a sha of any length, a tag, a branch. It
// is the commit the caller wants, so a name that reaches something else reaches
// nothing.
//
// Asked quietly, because git otherwise treats a name it cannot resolve as a failure
// to answer rather than as the answer no, and the two have to be told apart here as
// everywhere else.
func (r *Repo) Has(sha string) (bool, error) {
	return r.asks("rev-parse", "--verify", "--quiet", sha+"^{commit}")
}

// git runs one command and hands back its standard output, with the trailing newline
// taken off because every caller would otherwise take it off itself.
//
// What git wrote to standard error becomes the error. An exit status on its own is a
// failure nobody can act on, and git's own wording is better than a paraphrase of it:
// the case that matters most in practice, an identity git has not been told, comes back
// as the four lines telling the reader which two commands to run. Nothing here checks
// for that beforehand, because the check would only replace a good message with a
// shorter one.
func (r *Repo) git(args ...string) (string, error) {
	out, err := r.output(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// output is the same command with nothing taken off the end of what it wrote. A patch
// is bytes rather than an answer, and one that has lost its last newline is one git
// will not apply.
func (r *Repo) output(args ...string) ([]byte, error) {
	command := r.command(args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		said := strings.TrimSpace(stderr.String())
		if said == "" {
			said = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", attempted(args), said)
	}
	return stdout.Bytes(), nil
}

// command is one git command, run in the corpus directory with three of git's defaults
// turned off. None of them can carry what a corpus is, and turning them off here rather
// than at each call is what keeps a call from being made without them.
//
// Paths come back as the bytes they are. Git's default is to escape anything outside
// ASCII and wrap the result in quotes, which turns a scan filed under an accented name
// into a path that names no file — one origins would go on to print, and to hand back
// to git as a pathspec matching nothing. Entity files are named by slugs and have no
// accents to lose, but what is attached to an entity keeps the name it arrived with.
//
// A path longer than Windows' old limit is still a path. A family is named by both of
// its partners, so two slugs at their full length join into a file name of a hundred
// and thirty characters, and a corpus kept a few directories down then crosses the line
// at some point in the middle of the tree. Go writes such a file without noticing; git
// refuses to index it, which would leave origins having written a corpus it cannot
// record. The setting does nothing anywhere else.
//
// And a file git writes is the bytes that were committed. Git for Windows turns every
// line ending into CRLF on the way out by default, so an undo would put files back that
// are no longer what origins wrote: fmt would report them, and the next change to
// anything would write them again and name them among what it changed, git seeing no
// difference at all. The corpus declares its files LF in its own .gitattributes, which
// is what reaches a clone origins did not make; this reaches the commits made before
// the declaration was, and a corpus somebody has taken it out of.
func (r *Repo) command(args ...string) *exec.Cmd {
	command := exec.Command("git", append([]string{
		"-c", "core.quotePath=false",
		"-c", "core.longpaths=true",
		"-c", "core.autocrlf=false",
	}, args...)...)
	command.Dir = r.dir
	return command
}

// attempted is the command as a reader can take it in at a glance. A commit carries its
// whole message as one argument, trailers and all, and reproducing that verbatim buries
// git's own explanation underneath it — which in the case that matters most, an identity
// git has not been told, is the part naming the two commands to run.
func attempted(args []string) string {
	shown := make([]string, len(args))
	for i, arg := range args {
		if first, _, more := strings.Cut(arg, "\n"); more {
			arg = first + " ..."
		}
		shown[i] = arg
	}
	return strings.Join(shown, " ")
}

// asks runs a command whose answer is its exit status rather than its output, and says
// which of the two answers it gave.
//
// git reserves 1 for no and anything above it for could not tell, so a repository that
// cannot be read comes back as an error rather than as a confident no. The difference
// matters for every question here: an unreadable history and a history that has not
// moved are opposite answers, and only one of them means it is safe to go on.
func (r *Repo) asks(args ...string) (bool, error) {
	command := r.command(args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	err := command.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return true, nil
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return false, nil
	}
	said := strings.TrimSpace(stderr.String())
	if said == "" {
		said = err.Error()
	}
	return false, fmt.Errorf("git %s: %s", strings.Join(args, " "), said)
}
