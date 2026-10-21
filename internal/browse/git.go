package browse

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/history"
)

// Sharing the corpus from a page is the commands that share it, run where nobody is at
// a terminal: git is told so, and a remote it cannot reach without asking somebody is
// refused with what git said and the command that asks, to be run once in a terminal.

// Remoting is a remote as a page names one.
type Remoting struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Reaching is the remote a fetch, pull or push is for, or "" for the one the command
// would choose.
type Reaching struct {
	Remote string `json:"remote"`
}

// Recovering is which of the ways back a recover takes, as the command's flag names
// it, or "" to be told what there is to resolve.
type Recovering struct {
	Way string `json:"way"`
}

// unreachedError is a remote git could not reach without asking somebody.
type unreachedError struct {
	said, command string
	err           error
}

func (e *unreachedError) Error() string {
	said := strings.TrimSpace(e.said)
	if said == "" {
		said = e.err.Error()
	}
	return said + "\n\nNobody is at a terminal here to answer git. Run `" + strings.TrimSpace(e.command) +
		"` once in a terminal on this machine, where git can ask; a key an SSH agent holds, " +
		"or a password a credential helper keeps, is used here without asking."
}

func (e *unreachedError) Unwrap() error { return e.err }

// reaching runs a command that reaches another repository, keeping what git said for
// the refusal where it did not get there.
func reaching[T any](command string, run func(history.Terminal) (*T, error)) (*T, error) {
	var said bytes.Buffer
	report, err := run(history.Unattended(&said))
	var unreached *history.Unreached
	if errors.As(err, &unreached) {
		return report, &unreachedError{said: said.String(), command: command, err: err}
	}
	return report, err
}

func (s *Server) remotes(w http.ResponseWriter, r *http.Request) {
	if err := s.unhidden(r); err != nil {
		refused(w, err)
		return
	}
	report, err := api.ListRemotes(s.root)
	answered(w, report, err)
}

func (s *Server) addRemote(w http.ResponseWriter, r *http.Request) {
	var asked Remoting
	if !s.sharing(w, r, &asked) {
		return
	}
	report, err := api.AddRemote(s.root, asked.Name, asked.URL)
	answered(w, report, err)
}

func (s *Server) removeRemote(w http.ResponseWriter, r *http.Request) {
	if !s.sharing(w, r, nil) {
		return
	}
	report, err := api.RemoveRemote(s.root, r.PathValue("name"))
	answered(w, report, err)
}

func (s *Server) fetch(w http.ResponseWriter, r *http.Request) {
	var asked Reaching
	if !s.sharing(w, r, &asked) {
		return
	}
	report, err := reaching("origins fetch "+asked.Remote, func(t history.Terminal) (*api.Standing, error) {
		return api.Fetch(s.root, asked.Remote, t)
	})
	answered(w, report, err)
}

func (s *Server) push(w http.ResponseWriter, r *http.Request) {
	var asked Reaching
	if !s.sharing(w, r, &asked) {
		return
	}
	report, err := reaching("origins push "+asked.Remote, func(t history.Terminal) (*api.Sent, error) {
		return api.Push(s.root, asked.Remote, t)
	})
	answered(w, report, err)
}

// pull and recover change the corpus, so they hold it as a write does and the pages
// are told where it moved.
func (s *Server) pull(w http.ResponseWriter, r *http.Request) {
	var asked Reaching
	if !s.sharing(w, r, &asked) {
		return
	}
	s.writing(func() {
		report, err := reaching("origins pull "+asked.Remote, func(t history.Terminal) (*api.Taken, error) {
			return api.Pull(s.root, asked.Remote, t)
		})
		s.wrote()
		answered(w, report, err)
	})
}

func (s *Server) recover(w http.ResponseWriter, r *http.Request) {
	var asked Recovering
	if !s.sharing(w, r, &asked) {
		return
	}
	switch asked.Way {
	case "", api.ByAdopting, api.ByRewriting, api.ByResetting:
	default:
		refused(w, &askedError{"a recovery takes " + api.ByAdopting + ", " + api.ByRewriting +
			" or " + api.ByResetting + ", or nothing to be told what there is to resolve, and not " + asked.Way})
		return
	}
	s.writing(func() {
		report, err := api.Recover(s.root, asked.Way)
		s.wrote()
		answered(w, report, err)
	})
}

// sharing reads what a request to share the corpus asks into into, where there is
// anything to read, answering the request itself where it cannot be done: a copy is
// not shared, nor is anything changed from one.
func (s *Server) sharing(w http.ResponseWriter, r *http.Request, into any) bool {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return false
	}
	return into == nil || decoded(w, r, into)
}
