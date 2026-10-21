package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/history"
)

// Sharing a corpus with another clone of it. A remote is git's configuration and a
// fetch moves no branch, so neither opens the history to ask about the past, neither
// asks whether origins can account for it, and neither takes the lock: fetching is how
// this clone comes to know which of its commits have left it, which is something
// recover itself asks. A push hands the history on, and so asks.

// Remotes is every remote a clone knows, and where the current branch stands against
// its upstream as this clone last fetched it.
type Remotes struct {
	Remotes  []Remote `json:"remotes"`
	Branch   string   `json:"branch"`
	Upstream string   `json:"upstream"`
	Ahead    int      `json:"ahead"`
	Behind   int      `json:"behind"`
}

// Remote is a remote by its name and the URL it is fetched from.
type Remote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// AddRemote names a remote and the URL it is fetched from, and says what remotes there
// then are.
func AddRemote(root, name, url string) (*Remotes, error) {
	return remoting(root, func(repo *history.Repo) error { return repo.AddRemote(name, url) })
}

// RemoveRemote forgets a remote, and says what remotes there then are.
func RemoveRemote(root, name string) (*Remotes, error) {
	return remoting(root, func(repo *history.Repo) error { return repo.RemoveRemote(name) })
}

// ListRemotes says what remotes there are.
func ListRemotes(root string) (*Remotes, error) {
	return remoting(root, func(*history.Repo) error { return nil })
}

func remoting(root string, change func(*history.Repo) error) (*Remotes, error) {
	repo, err := history.Open(root)
	if err != nil {
		return nil, err
	}
	if err := change(repo); err != nil {
		return nil, err
	}
	known, err := repo.Remotes()
	if err != nil {
		return nil, err
	}
	tracking, err := repo.Tracks()
	if err != nil {
		return nil, err
	}
	report := &Remotes{
		Remotes: []Remote{}, Branch: tracking.Branch, Upstream: tracking.Upstream,
		Ahead: tracking.Ahead, Behind: tracking.Behind,
	}
	for _, one := range known {
		report.Remotes = append(report.Remotes, Remote{Name: one.Name, URL: one.URL})
	}
	return report, nil
}

// Standing is where the current branch stands against its upstream, after a fetch.
type Standing struct {
	Fetched  []string `json:"fetched"`
	Branch   string   `json:"branch"`
	Upstream string   `json:"upstream"`
	Ahead    int      `json:"ahead"`
	Behind   int      `json:"behind"`
}

// Fetch brings in what the remote named holds, or every remote where none is, and
// says how far the branch and its upstream have each gone without the other. Anything
// git has to ask is asked through terminal.
func Fetch(root, named string, terminal history.Terminal) (*Standing, error) {
	repo, err := history.Open(root)
	if err != nil {
		return nil, err
	}
	known, err := repo.Remotes()
	if err != nil {
		return nil, err
	}
	report := &Standing{Fetched: []string{}}
	if named != "" {
		report.Fetched = append(report.Fetched, named)
	} else {
		for _, one := range known {
			report.Fetched = append(report.Fetched, one.Name)
		}
	}
	if len(report.Fetched) == 0 {
		return nil, errors.New("there is no remote to fetch from: origins remote add origin <url> names one")
	}
	if err := repo.Fetch(named, terminal); err != nil {
		return nil, err
	}
	tracking, err := repo.Tracks()
	if err != nil {
		return nil, err
	}
	report.Branch, report.Upstream = tracking.Branch, tracking.Upstream
	report.Ahead, report.Behind = tracking.Ahead, tracking.Behind
	return report, nil
}

// toward is the remote a change reaches, as history.Toward chooses it, with a refusal
// that says what to do where it cannot choose.
func toward(repo *history.Repo, named, command string) (string, error) {
	remote, err := repo.Toward(named)
	var (
		none history.NoRemote
		many history.ManyRemotes
	)
	switch {
	case errors.As(err, &none):
		return "", fmt.Errorf("there is no remote to %s: origins remote add origin <url> names one", command)
	case errors.As(err, &many):
		return "", fmt.Errorf("%v: origins %s %s says which", err, command, many.Names[0])
	}
	return remote, err
}

// Sent is what a push sent where.
type Sent struct {
	Remote   string `json:"remote"`
	Branch   string `json:"branch"`
	Commit   string `json:"commit"`
	Upstream bool   `json:"upstream"`
}

// Push sends the branch to a remote with every snapshot on it.
//
// It opens the history as everything that rests on it does, since pushing a history
// origins cannot account for would hand the doubt to every clone of it. A dirty tree
// does not stop it: a push sends commits, and nothing unrecorded is in one.
func Push(root, named string, terminal history.Terminal) (*Sent, error) {
	repo, err := owned(root)
	if err != nil {
		return nil, err
	}
	remote, err := toward(repo, named, "push")
	if err != nil {
		return nil, err
	}
	pushed, err := repo.Push(remote, terminal)
	var overtaken *history.Overtaken
	if errors.As(err, &overtaken) {
		return nil, fmt.Errorf("%w: origins pull takes them in, and a push after it sends what is here", err)
	}
	if err != nil {
		return nil, err
	}
	return &Sent{Remote: pushed.Remote, Branch: pushed.Branch, Commit: pushed.Sha, Upstream: pushed.Upstream}, nil
}

// Cloning is where a clone is, and what was found in it.
type Cloning struct {
	Dir      string   `json:"dir"`
	Corpus   bool     `json:"corpus"`
	Problems []string `json:"problems"`
	Errors   int      `json:"errors"`
	Warnings int      `json:"warnings"`
	Settled  bool     `json:"settled"`
	Foreign  []Entry  `json:"foreign"`
	Marker   string   `json:"marker"`

	// Standing is where the history stands, in a sentence.
	Standing string `json:"-"`
}

// Found is whether a clone answered badly: no corpus, one that does not read whole or
// contradicts itself, or a history origins cannot account for.
func (c *Cloning) Found() bool {
	return !c.Corpus || len(c.Problems) > 0 || c.Errors > 0 || !c.Settled
}

// Clone copies a corpus somebody shared into dir, or a directory named after the URL
// where dir is empty, and asks of the copy what every command asks of a corpus:
// whether there is one, what check finds in it, and whether origins can account for
// its history. Where it can, the marker is set to HEAD, the newest commit being one
// origins wrote.
//
// A copy that answers badly is kept, the report saying what was found. It is the
// researcher's copy of what was shared, and taking it away would hide the finding
// along with it.
func Clone(url, dir string, terminal history.Terminal) (report *Cloning, err error) {
	if dir == "" {
		dir = history.Cloned(url)
	}
	if dir == "" {
		return nil, fmt.Errorf("origins cannot tell what to call a clone of %s: name the directory after it", url)
	}
	repo, err := history.Clone(url, dir, terminal)
	if err != nil {
		return nil, err
	}
	report = &Cloning{Dir: dir, Problems: []string{}, Foreign: []Entry{}}
	var warning error
	if _, err := os.Stat(filepath.Join(dir, "origins.yaml")); err == nil {
		report.Corpus = true
		corpus, problems, later := opened(dir)
		warning = later
		report.Problems = Flatten(problems)
		findings := check.Run(corpus.Corpus)
		report.Errors = check.Errors(findings)
		report.Warnings = len(findings) - report.Errors
	}
	standing, err := repo.Survey()
	if err != nil {
		return nil, err
	}
	report.Standing = standing.Describe()
	for _, one := range standing.Foreign {
		report.Foreign = append(report.Foreign, described(one))
	}
	if report.Settled = standing.Settled(); report.Settled {
		if report.Marker, err = repo.Head(); err != nil {
			return nil, err
		}
		if report.Marker != "" {
			if err := repo.Mark(report.Marker); err != nil {
				return nil, err
			}
		}
	}
	return report, warning
}
