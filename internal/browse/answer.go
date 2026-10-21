package browse

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/ops"
)

// Refusal is what a request is answered with when it is not answered: what kind of
// refusal it is, for the page to decide how to show it, and the words the command line
// would have said.
type Refusal struct {
	Kind     string          `json:"kind"`
	Error    string          `json:"error"`
	Problems []string        `json:"problems,omitempty"`
	Findings []check.Finding `json:"findings,omitempty"`
}

// answer writes a report as JSON.
func answer(w http.ResponseWriter, status int, report any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(report)
}

// answered writes what a question put to internal/api came back with: the report where
// there is one, the error said beside it being about something after the answer, and
// the refusal where there is none.
func answered[T any](w http.ResponseWriter, report *T, err error) {
	if report != nil {
		answer(w, http.StatusOK, report)
		return
	}
	refused(w, err)
}

// refused answers with a refusal, in the kind ops and history give it.
func refused(w http.ResponseWriter, err error) {
	status, refusal := refusalOf(err)
	answer(w, status, refusal)
}

func refusalOf(err error) (int, Refusal) {
	refusal := Refusal{Kind: "refused", Error: err.Error()}
	var (
		unreadable    *api.Unreadable
		opsUnreadable *ops.Unreadable
		stale         *ops.Stale
		contradiction *ops.Contradiction
		drifted       *ops.Drifted
		locked        *history.Locked
		unsettled     *history.Unsettled
		missing       *missingError
		undrafted     ops.Undrafted
		asked         *askedError
		readOnly      *readOnlyError
		hidden        *hiddenError
		unreached     *unreachedError
	)
	switch {
	case errors.As(err, &unreached):
		refusal.Kind = "unreached"
		return http.StatusBadGateway, refusal
	case errors.As(err, &readOnly):
		refusal.Kind = "read-only"
		return http.StatusForbidden, refusal
	case errors.As(err, &hidden):
		refusal.Kind = "hidden"
		return http.StatusForbidden, refusal
	case errors.As(err, &missing), errors.As(err, &undrafted):
		refusal.Kind = "missing"
		return http.StatusNotFound, refusal
	case errors.As(err, &asked):
		refusal.Kind = "asked"
		return http.StatusBadRequest, refusal
	case errors.As(err, &unreadable):
		refusal.Kind, refusal.Error, refusal.Problems = "unreadable", unreadable.Refusal, unreadable.Problems
	case errors.As(err, &opsUnreadable):
		refusal.Kind, refusal.Problems = "unreadable", api.Flatten(opsUnreadable.Problems)
	case errors.As(err, &stale):
		refusal.Kind = "stale"
	case errors.As(err, &contradiction):
		refusal.Kind, refusal.Findings = "contradiction", contradiction.Findings
	case errors.As(err, &drifted):
		refusal.Kind = "drifted"
	case errors.As(err, &locked):
		refusal.Kind = "lock"
	case errors.As(err, &unsettled):
		refusal.Kind = "unsettled"
	}
	return http.StatusConflict, refusal
}

// missingError is a request for something the corpus does not hold.
type missingError struct{ what string }

func (m *missingError) Error() string { return m.what }

// askedError is a request that does not say what it asks.
type askedError struct{ what string }

func (a *askedError) Error() string { return a.what }

func (s *Server) unknown(w http.ResponseWriter, r *http.Request) {
	refused(w, &missingError{"there is no " + r.Method + " " + r.URL.Path})
}
