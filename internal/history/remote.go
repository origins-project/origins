package history

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// A corpus is shared through git's remotes, and nothing about sharing it is origins'
// own: the remotes are git's configuration, the transport is git's, and so is asking
// for a key or a password. What origins adds is running it with the settings every
// other command carries, and saying afterwards where the branch stands.

// Remote is a repository the corpus is shared with, by the name this clone knows it by
// and the URL it fetches from.
type Remote struct {
	Name string
	URL  string
}

// Remotes are the remotes this clone knows, in the order git lists them.
func (r *Repo) Remotes() ([]Remote, error) {
	out, err := r.git("remote", "-v")
	if err != nil {
		return nil, err
	}
	var remotes []Remote
	for _, line := range lines(out) {
		name, rest, _ := strings.Cut(line, "\t")
		url, fetches := strings.CutSuffix(rest, " (fetch)")
		if fetches {
			remotes = append(remotes, Remote{Name: name, URL: url})
		}
	}
	return remotes, nil
}

// AddRemote gives this clone a remote, which is configuration and reaches nothing: the
// URL is first used by the fetch or push after it.
func (r *Repo) AddRemote(name, url string) error {
	_, err := r.git("remote", "add", "--", name, url)
	return err
}

// RemoveRemote forgets a remote, and with it the branches this clone last saw there.
func (r *Repo) RemoveRemote(name string) error {
	_, err := r.git("remote", "remove", "--", name)
	return err
}

// Terminal is where git asks for what it needs to reach a remote and says how it is
// getting on: the researcher's own input and error stream, handed to git as they are.
//
// SSH asks for a passphrase and a credential helper for a password, and either may
// ask on the input git was given rather than the console. What git writes as it goes
// is progress and refusals worded for a person, and it writes them only where it sees
// a person to read them, so a stream that is the console is passed along untouched
// rather than copied, which would hide the console from it.
//
// A terminal nobody is at, a server's, is one Unattended makes: git is told there is
// nobody to ask, and what it says is kept for whoever asked over the network.
type Terminal struct {
	In  io.Reader
	Err io.Writer
	Env []string // set for git besides the environment origins runs in
}

// Unattended is a terminal nobody is at, writing what git says to said. git's own
// prompt is turned off, and so is SSH's and the Git Credential Manager's, so that a
// remote wanting a password or a passphrase is refused at once rather than waited on
// forever; a key an SSH agent holds and a password a credential helper keeps still
// reach it, neither asking anybody.
func Unattended(said io.Writer) Terminal {
	ssh := "ssh"
	if own := os.Getenv("GIT_SSH_COMMAND"); own != "" {
		ssh = own
	}
	return Terminal{In: strings.NewReader(""), Err: said, Env: []string{
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=never",
		"GIT_SSH_COMMAND=" + ssh + " -o BatchMode=yes",
	}}
}

// Unreached is a command across the network that git did not finish. What it said is
// already on the terminal, so this names the command and how it ended and nothing more.
type Unreached struct {
	Args []string
	Err  error
}

func (u *Unreached) Error() string {
	return fmt.Sprintf("git %s did not finish: %v", strings.Join(u.Args, " "), u.Err)
}

func (u *Unreached) Unwrap() error { return u.Err }

// across runs a command that reaches another repository, with the three settings every
// command carries. Its standard output, where git writes anything meant for a program,
// is handed back; everything else is between git and the person at the terminal.
func (r *Repo) across(term Terminal, args ...string) (string, error) {
	command := r.command(args...)
	var stdout bytes.Buffer
	command.Stdin, command.Stdout, command.Stderr = term.In, &stdout, term.Err
	if term.Env != nil {
		command.Env = append(os.Environ(), term.Env...)
	}
	if err := command.Run(); err != nil {
		return stdout.String(), &Unreached{Args: args, Err: err}
	}
	return stdout.String(), nil
}

// Fetch brings what a remote holds into this clone's remote-tracking branches, or what
// every remote holds where none is named. It moves no branch of this clone's, so the
// corpus and its history are exactly as they were; what changes is what this clone
// knows has left it, which is what Shared asks.
func (r *Repo) Fetch(remote string, term Terminal) error {
	args := []string{"fetch", "--all"}
	if remote != "" {
		args = []string{"fetch", "--", remote}
	}
	_, err := r.across(term, args...)
	return err
}

// Tracking is where the current branch stands against its upstream, as of the last
// fetch.
type Tracking struct {
	// Branch is the branch HEAD is on, or "" where HEAD is detached.
	Branch string

	// Upstream is the remote-tracking branch it follows, or "" where it follows none.
	Upstream string

	// Ahead is how many commits the branch has that its upstream does not, and Behind
	// how many the upstream has that the branch does not.
	Ahead, Behind int
}

// Tracks says where the current branch stands against its upstream. A detached HEAD
// and a branch following nothing are ordinary answers, the fields left empty.
func (r *Repo) Tracks() (Tracking, error) {
	var t Tracking
	branch, err := r.git("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return t, nil
	}
	t.Branch = branch
	upstream, err := r.git("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return t, nil
	}
	t.Upstream = upstream
	if head, _ := r.Head(); head == "" {
		return t, nil
	}
	counts, err := r.git("rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err != nil {
		return t, err
	}
	ahead, behind, _ := strings.Cut(counts, "\t")
	if t.Ahead, err = strconv.Atoi(ahead); err != nil {
		return t, errors.New("git counted the commits on each side as " + strconv.Quote(counts))
	}
	if t.Behind, err = strconv.Atoi(behind); err != nil {
		return t, errors.New("git counted the commits on each side as " + strconv.Quote(counts))
	}
	return t, nil
}

// NoRemote is a clone that knows no remote to reach.
type NoRemote struct{}

func (NoRemote) Error() string { return "this clone knows no remote" }

// ManyRemotes is a clone that knows several, none of which the branch follows, so that
// which one is meant is the researcher's to say.
type ManyRemotes struct{ Names []string }

func (m ManyRemotes) Error() string {
	return "this clone knows " + strings.Join(m.Names, ", ") + ", and the branch follows none of them"
}

// Toward is the remote a command that reaches one goes to: the one named, the one the
// branch's upstream is on, or the only one there is.
func (r *Repo) Toward(named string) (string, error) {
	if named != "" {
		return named, nil
	}
	if branch, err := r.git("symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		if remote, err := r.git("config", "--get", "branch."+branch+".remote"); err == nil && remote != "" {
			return remote, nil
		}
	}
	known, err := r.Remotes()
	if err != nil {
		return "", err
	}
	switch len(known) {
	case 0:
		return "", NoRemote{}
	case 1:
		return known[0].Name, nil
	}
	names := make([]string, len(known))
	for i, one := range known {
		names[i] = one.Name
	}
	return "", ManyRemotes{Names: names}
}

// Detached is a HEAD on no branch, which has nothing to push or pull.
type Detached struct{}

func (Detached) Error() string { return "HEAD is on no branch" }

// Overtaken is a push the remote turned down because it holds commits this clone does
// not, which only taking them in first can answer.
type Overtaken struct{ Remote, Branch string }

func (b *Overtaken) Error() string {
	return fmt.Sprintf("%s holds commits on %s that this clone does not", b.Remote, b.Branch)
}

// Pushed is what a push sent: the branch, where to, and the commit it now stands on
// there.
type Pushed struct {
	Remote string
	Branch string
	Sha    string

	// Upstream is whether this push made the remote's branch the one this branch
	// follows, which the first push from a clone does.
	Upstream bool
}

// Push sends the current branch to a remote, and every annotated tag reachable from it,
// a snapshot being one: a state named here is a state somebody else may be pointed at.
// refs/origins/head is under neither refs/heads nor refs/tags and so never goes, each
// clone answering for what its own origins wrote.
//
// A branch following nothing yet comes to follow what it was pushed to, under its own
// name; one that follows something on this remote is pushed to what it follows, and
// one pushed to another remote than it follows keeps following what it did.
func (r *Repo) Push(remote string, term Terminal) (Pushed, error) {
	branch, err := r.git("symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil {
		return Pushed{}, Detached{}
	}
	pushed := Pushed{Remote: remote, Branch: branch}
	if pushed.Sha, err = r.Head(); err != nil || pushed.Sha == "" {
		return Pushed{}, fmt.Errorf("nothing has been committed on %s to push", branch)
	}

	args := []string{"push", "--porcelain", "--follow-tags"}
	to := "refs/heads/" + branch
	follows, _ := r.git("config", "--get", "branch."+branch+".remote")
	switch merge, err := r.git("config", "--get", "branch."+branch+".merge"); {
	case err == nil && follows == remote:
		to = merge
	case follows == "":
		args = append(args, "--set-upstream")
		pushed.Upstream = true
	}
	args = append(args, "--", remote, "refs/heads/"+branch+":"+to)

	out, err := r.across(term, args...)
	if err != nil {
		for _, line := range lines(strings.TrimRight(out, "\n")) {
			if strings.HasPrefix(line, "!") &&
				(strings.Contains(line, "(fetch first)") || strings.Contains(line, "(non-fast-forward)")) {
				return Pushed{}, &Overtaken{Remote: remote, Branch: strings.TrimPrefix(to, "refs/heads/")}
			}
		}
		return Pushed{}, err
	}
	return pushed, nil
}

// Clone copies the repository at url into dir, which git makes, with the three
// settings every command carries: core.autocrlf above all, so that every file is
// checked out as it was committed whatever the machine is configured to do.
//
// Nothing is asked of what arrives here. Whether it is a corpus, and whether origins
// can account for its history, are the caller's questions, put to the clone once it is
// there to be asked.
func Clone(url, dir string, term Terminal) (*Repo, error) {
	// run where origins was, which is where dir is named from
	here, err := available("")
	if err != nil {
		return nil, err
	}
	if _, err := here.across(term, "clone", "--", url, dir); err != nil {
		return nil, err
	}
	return &Repo{dir: dir}, nil
}

// Cloned is the directory git would clone a URL into, named for the last part of it
// with any .git taken off, as git names one where it is not told.
func Cloned(url string) string {
	name := strings.TrimRight(url, `/\`)
	name = strings.TrimSuffix(name, "/.git")
	name = strings.TrimRight(name, `/\`)
	if i := strings.LastIndexAny(name, `/\:`); i >= 0 {
		name = name[i+1:]
	}
	return strings.TrimSuffix(name, ".git")
}
