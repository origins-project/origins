package browse

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/api"
)

// sharedPair is the git fixture with a bare repository beside it as its origin, and a
// collaborator's clone of that, each served by a server of its own.
func sharedPair(t *testing.T) (mine, theirs *client) {
	t.Helper()
	root := gitFixture(t)
	bare := filepath.Join(filepath.Dir(root), "shared.git")
	git(t, root, "clone", "-q", "--bare", ".", bare)
	mine = serving(t, root)
	remotes := read[api.Remotes](mine, mine.post("/api/remotes", Remoting{Name: "origin", URL: bare}), http.StatusOK)
	if len(remotes.Remotes) != 1 || remotes.Remotes[0].URL != bare {
		t.Fatalf("the remotes are %+v", remotes)
	}
	read[api.Standing](mine, mine.post("/api/fetch", Reaching{}), http.StatusOK)
	git(t, root, "branch", "-q", "--set-upstream-to", "origin/main")

	other := filepath.Join(filepath.Dir(root), "paolo")
	git(t, filepath.Dir(root), "clone", "-q", bare, other)
	git(t, other, "config", "user.name", "Paolo Verdi")
	git(t, other, "config", "user.email", "paolo@example.invalid")
	return mine, serving(t, other)
}

func TestAPageSharesTheCorpus(t *testing.T) {
	t.Parallel()
	mine, theirs := sharedPair(t)

	theirs.change(map[string]any{"op": "set", "entity": "person/verdier-jean-baptiste", "path": "sex", "value": "F"}, http.StatusOK)
	sent := read[api.Sent](theirs, theirs.post("/api/push", Reaching{}), http.StatusOK)
	if sent.Remote != "origin" || sent.Commit != head(t, theirs.server.root) {
		t.Errorf("the push reports %+v", sent)
	}

	standing := read[api.Standing](mine, mine.post("/api/fetch", Reaching{Remote: "origin"}), http.StatusOK)
	if standing.Behind != 1 || standing.Ahead != 0 {
		t.Errorf("the fetch reports %+v", standing)
	}
	taken := read[api.Taken](mine, mine.post("/api/pull", Reaching{}), http.StatusOK)
	if taken.Took != "fast-forward" || taken.Commit != sent.Commit || head(t, mine.server.root) != sent.Commit {
		t.Errorf("the pull reports %+v", taken)
	}
	if page := read[Page](mine, mine.get("/api/records/person/verdier-jean-baptiste"), http.StatusOK); page.Record.Seen == "" ||
		!strings.Contains(string(raw(t, page.Record.Fields)), `"F"`) {
		t.Error("the corpus served was not read again after the pull")
	}

	if remotes := read[api.Remotes](mine, mine.get("/api/remotes"), http.StatusOK); remotes.Branch != "main" ||
		remotes.Upstream != "origin/main" || remotes.Ahead != 0 || remotes.Behind != 0 {
		t.Errorf("the remotes say the branch stands as %+v after the pull", remotes)
	}
	read[api.Remotes](mine, mine.do(http.MethodDelete, "/api/remotes/origin", nil, nil), http.StatusOK)
	if remotes := read[api.Remotes](mine, mine.get("/api/remotes"), http.StatusOK); len(remotes.Remotes) != 0 {
		t.Errorf("the remotes are %+v after the one there was went", remotes)
	}
}

func TestARemoteAskingForAPasswordIsNotWaitedOn(t *testing.T) {
	t.Parallel()
	asking := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="tree"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(asking.Close)
	c := serving(t, gitFixture(t))
	read[api.Remotes](c, c.post("/api/remotes", Remoting{Name: "zia-maria", URL: asking.URL + "/rossi.git"}), http.StatusOK)

	began := time.Now()
	refusal := read[Refusal](c, c.post("/api/fetch", Reaching{Remote: "zia-maria"}), http.StatusBadGateway)
	if refusal.Kind != "unreached" || !strings.Contains(refusal.Error, "origins fetch zia-maria") {
		t.Errorf("the refusal is %+v", refusal)
	}
	if took := time.Since(began); took > 30*time.Second {
		t.Errorf("the fetch took %v", took)
	}
}

func TestACopyIsNotShared(t *testing.T) {
	t.Parallel()
	c := serving(t, gitFixture(t))
	if refusal := read[Refusal](c, c.post("/api/push?view=hide-living", Reaching{}), http.StatusForbidden); refusal.Kind != "read-only" {
		t.Errorf("a push under a view is refused as %+v", refusal)
	}
	if refusal := read[Refusal](c, c.get("/api/remotes?view=hide-private"), http.StatusForbidden); refusal.Kind != "hidden" {
		t.Errorf("the remotes under a view are refused as %+v", refusal)
	}
}
