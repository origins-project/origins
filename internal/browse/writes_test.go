package browse

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/ops"
)

// post sends what a page would, as JSON.
func (c *client) post(target string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	written, err := json.Marshal(body)
	if err != nil {
		c.t.Fatal(err)
	}
	return c.do(http.MethodPost, target, bytes.NewReader(written), nil)
}

// seen is a record's seen as a page would be shown it, or ops.Absent where there is
// none of it.
func (c *client) seen(ref string) string {
	c.t.Helper()
	response := c.get("/api/records/" + ref)
	if response.Code == http.StatusNotFound {
		return ops.Absent
	}
	return read[Page](c, response, http.StatusOK).Record.Seen
}

// change makes one edit as a page would, pinned to the record it is about as the page
// was shown it.
func (c *client) change(edit map[string]any, status int) Made {
	c.t.Helper()
	entity, _ := edit["entity"].(string)
	if entity == "" {
		entity, _ = edit["from"].(string)
	}
	return read[Made](c, c.post("/api/edit", Change{Edit: raw(c.t, edit), Seen: c.seen(entity)}), status)
}

func raw(t *testing.T, edit any) json.RawMessage {
	t.Helper()
	written, err := json.Marshal(edit)
	if err != nil {
		t.Fatal(err)
	}
	return written
}

func head(t *testing.T, root string) string {
	t.Helper()
	return git(t, root, "rev-parse", "HEAD")
}

func TestEveryEditIsMadeFromAPage(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	before := head(t, root)

	steps := []struct {
		edit   map[string]any
		stands []string
	}{
		{map[string]any{"op": "create", "entity": "person/test-alpha",
			"value": map[string]any{"name": map[string]any{"given": "Alpha", "surname": "Test"}}},
			[]string{"person/test-alpha"}},
		{map[string]any{"op": "set", "entity": "person/test-alpha", "path": "sex", "value": "M"},
			[]string{"person/test-alpha"}},
		{map[string]any{"op": "add", "entity": "person/test-alpha", "path": "tags", "value": "checked"},
			[]string{"person/test-alpha"}},
		{map[string]any{"op": "remove", "entity": "person/test-alpha", "path": "tags", "value": "checked"},
			[]string{"person/test-alpha"}},
		{map[string]any{"op": "unset", "entity": "person/test-alpha", "path": "sex"},
			[]string{"person/test-alpha"}},
		{map[string]any{"op": "rename", "entity": "person/test-alpha", "to": "test-beta"},
			[]string{"person/test-beta"}},
		{map[string]any{"op": "create", "entity": "person/test-gamma",
			"value": map[string]any{"name": map[string]any{"surname": "Test"}}},
			[]string{"person/test-gamma"}},
		{map[string]any{"op": "merge", "from": "person/test-gamma", "into": "person/test-beta"},
			[]string{"person/test-beta"}},
		{map[string]any{"op": "split", "entity": "person/test-beta", "into": []string{"person/test-one", "person/test-two"}},
			[]string{"person/test-one", "person/test-two"}},
		{map[string]any{"op": "delete", "entity": "person/test-two"},
			[]string{}},
		{map[string]any{"op": "set", "entity": "family/chen-yuying--unknown", "path": "partners[1]", "value": "test-one"},
			[]string{"family/chen-yuying--test-one"}},
	}
	for _, step := range steps {
		made := c.change(step.edit, http.StatusOK)
		if made.Commit == "" || made.Empty {
			t.Fatalf("%v made no commit: %+v", step.edit, made)
		}
		if !slices.Equal(made.Stands, step.stands) {
			t.Errorf("after %v the page goes to %v, want %v", step.edit, made.Stands, step.stands)
		}
		if made.Commit != head(t, root) {
			t.Errorf("%v answered %s, and HEAD is %s", step.edit, made.Commit, head(t, root))
		}
	}

	journal := read[api.Journal](c, c.get("/api/log?n="+strconv.Itoa(len(steps))), http.StatusOK)
	var done []string
	for _, entry := range slices.Backward(journal.Entries) {
		done = append(done, entry.Op)
	}
	want := []string{"person.create", "sex.set", "tags.add", "tags.remove", "sex.unset",
		"person.rename", "person.create", "person.merge", "person.split", "person.delete", "partners.set"}
	if !slices.Equal(done, want) {
		t.Errorf("the history says %v, want %v", done, want)
	}
	if got := git(t, root, "rev-list", "--count", before+"..HEAD"); got != strconv.Itoa(len(steps)) {
		t.Errorf("%s commits since the fixture, want %d", got, len(steps))
	}
	if c.seen("person/test-one") == ops.Absent {
		t.Error("the server was not reading the corpus as it was written")
	}
}

func TestADryRunWritesNothing(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	before := head(t, root)
	made := read[Made](c, c.post("/api/edit", Change{
		Edit: raw(t, map[string]any{"op": "rename", "entity": "person/verdier-jean-baptiste", "to": "rossi-giovanni-battista"}),
		Seen: c.seen("person/verdier-jean-baptiste"),
		Dry:  true,
	}), http.StatusOK)
	if made.Commit != "" || len(made.Moved) == 0 || len(made.Written) == 0 {
		t.Errorf("the dry run said %+v", made.Recorded)
	}
	if head(t, root) != before || c.seen("person/verdier-jean-baptiste") == ops.Absent {
		t.Error("the dry run wrote")
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("the dry run left the files changed:\n%s", status)
	}
}

func TestAChangeIsTakenBackFromAPage(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	before := head(t, root)
	made := c.change(map[string]any{"op": "set", "entity": "person/verdier-jean-baptiste", "path": "sex", "value": "F"}, http.StatusOK)

	read[Refusal](c, c.post("/api/undo", Settling{Seen: before}), http.StatusConflict)
	dry := read[api.Reversal](c, c.post("/api/undo", Settling{Seen: made.Commit, Dry: true}), http.StatusOK)
	if dry.Commit != "" || head(t, root) != made.Commit {
		t.Fatalf("the dry run of an undo wrote %s", dry.Commit)
	}
	undone := read[api.Reversal](c, c.post("/api/undo", Settling{Seen: made.Commit}), http.StatusOK)
	if undone.Undid.Commit != made.Commit || undone.Commit != head(t, root) {
		t.Errorf("the undo answered %+v", undone)
	}
	page := read[Page](c, c.get("/api/records/person/verdier-jean-baptiste"), http.StatusOK)
	if page.Record.Fields["sex"] != "M" {
		t.Errorf("after the undo the record says sex %v", page.Record.Fields["sex"])
	}
}

func TestAHypothesisIsSettledFromAPage(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	before := head(t, root)
	dry := read[api.Decided](c, c.post("/api/hypotheses/unrecorded-child-1845/accept", Settling{Dry: true}), http.StatusOK)
	if dry.Commit != "" || head(t, root) != before {
		t.Fatalf("the dry run of an accept wrote %s", dry.Commit)
	}
	accepted := read[api.Decided](c, c.post("/api/hypotheses/unrecorded-child-1845/accept", Settling{}), http.StatusOK)
	if accepted.Status != "accepted" || accepted.Commit != head(t, root) {
		t.Errorf("the accept answered %+v", accepted)
	}
	if c.seen("person/verdier-unknown-1845") == ops.Absent {
		t.Error("the person the hypothesis creates is not served after it was accepted")
	}
	journal := read[api.Journal](c, c.get("/api/log?n=1"), http.StatusOK)
	if len(journal.Entries) != 1 || journal.Entries[0].Hypothesis != "unrecorded-child-1845" {
		t.Errorf("the history says %+v", journal.Entries)
	}
	read[Refusal](c, c.post("/api/hypotheses/no-such-thing/reject", Settling{}), http.StatusConflict)
}

func TestAStaleChangeWritesNothing(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	before := head(t, root)
	shown := c.seen("person/verdier-jean-baptiste")
	c.change(map[string]any{"op": "set", "entity": "person/verdier-jean-baptiste", "path": "sex", "value": "F"}, http.StatusOK)
	changed := head(t, root)

	refusal := read[Refusal](c, c.post("/api/edit", Change{
		Edit: raw(t, map[string]any{"op": "unset", "entity": "person/verdier-jean-baptiste", "path": "portrait"}),
		Seen: shown,
	}), http.StatusConflict)
	if refusal.Kind != "stale" {
		t.Errorf("refused as %q: %s", refusal.Kind, refusal.Error)
	}
	if head(t, root) != changed || changed == before {
		t.Error("a stale change was recorded")
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("a stale change left the files changed:\n%s", status)
	}
	read[Refusal](c, c.post("/api/edit", Change{
		Edit: raw(t, map[string]any{"op": "create", "entity": "person/verdier-jean-baptiste", "value": map[string]any{}}),
		Seen: ops.Absent,
	}), http.StatusConflict)
}

func TestEveryKindOfRefusalIsSaid(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		kind    string
		prepare func(t *testing.T, root string)
		ask     func(c *client) *httptest.ResponseRecorder
		status  int
	}{
		{"asked", nil, func(c *client) *httptest.ResponseRecorder {
			return c.do(http.MethodPost, "/api/edit", strings.NewReader("{not json"), nil)
		}, http.StatusBadRequest},
		{"asked", nil, func(c *client) *httptest.ResponseRecorder {
			return c.post("/api/edit", Change{Edit: raw(c.t, map[string]any{"op": "promote", "entity": "person/verdier-jean-baptiste"})})
		}, http.StatusBadRequest},
		{"refused", nil, func(c *client) *httptest.ResponseRecorder {
			return c.post("/api/edit", Change{Edit: raw(c.t, map[string]any{
				"op": "set", "entity": "person/verdier-jean-baptiste", "path": "colour", "value": "red"})})
		}, http.StatusConflict},
		{"contradiction", nil, func(c *client) *httptest.ResponseRecorder {
			return c.post("/api/edit", Change{Edit: raw(c.t, map[string]any{
				"op": "set", "entity": "person/verdier-jean-baptiste", "path": "events[0].place", "value": "atlantis"})})
		}, http.StatusConflict},
		{"stale", nil, func(c *client) *httptest.ResponseRecorder {
			return c.post("/api/edit", Change{Seen: "0000", Edit: raw(c.t, map[string]any{
				"op": "set", "entity": "person/verdier-jean-baptiste", "path": "sex", "value": "F"})})
		}, http.StatusConflict},
		{"lock", func(t *testing.T, root string) {
			lock := git(t, root, "rev-parse", "--absolute-git-dir")
			if err := os.WriteFile(filepath.Join(lock, "origins.lock"), []byte("1\norigins set\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, func(c *client) *httptest.ResponseRecorder {
			return c.post("/api/edit", Change{Edit: raw(c.t, map[string]any{
				"op": "set", "entity": "person/verdier-jean-baptiste", "path": "sex", "value": "F"})})
		}, http.StatusConflict},
		{"unreadable", func(t *testing.T, root string) {
			broken := filepath.Join(root, "people", "verdier-pierre", "README.md")
			if err := os.WriteFile(broken, []byte("---\nname: [unclosed\n---\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, func(c *client) *httptest.ResponseRecorder {
			return c.post("/api/edit", Change{Edit: raw(c.t, map[string]any{
				"op": "set", "entity": "person/verdier-jean-baptiste", "path": "sex", "value": "F"})})
		}, http.StatusConflict},
		{"drifted", func(t *testing.T, root string) {
			c := serving(t, root)
			c.change(map[string]any{"op": "delete", "entity": "family/lang-anna-maria--verdier-louis"}, http.StatusOK)
		}, func(c *client) *httptest.ResponseRecorder {
			return c.post("/api/hypotheses/unrecorded-child-1845/accept", Settling{})
		}, http.StatusConflict},
	} {
		t.Run(one.kind, func(t *testing.T) {
			t.Parallel()
			root := gitFixture(t)
			if one.prepare != nil {
				one.prepare(t, root)
			}
			c := serving(t, root)
			before := head(t, root)
			refusal := read[Refusal](c, one.ask(c), one.status)
			if refusal.Kind != one.kind || refusal.Error == "" {
				t.Errorf("refused as %q: %s", refusal.Kind, refusal.Error)
			}
			if one.kind == "contradiction" && len(refusal.Findings) == 0 {
				t.Error("a contradiction came without its findings")
			}
			if one.kind == "unreadable" && len(refusal.Problems) == 0 {
				t.Error("an unreadable corpus came without its problems")
			}
			if head(t, root) != before {
				t.Error("a refused change was recorded")
			}
		})
	}
}

func TestProseIsRenderedForAPreview(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	rendered := read[Rendered](c, c.post("/api/render", Rendering{
		Entity: "person/verdier-jean-baptiste",
		Prose:  "See [his sister](../../people/verdier-louisa/README.md).\n\n<script>x</script>",
	}), http.StatusOK)
	if !strings.Contains(rendered.HTML, `href="/person/verdier-louisa"`) || strings.Contains(rendered.HTML, "<script") {
		t.Errorf("rendered as %s", rendered.HTML)
	}
	read[Refusal](c, c.post("/api/render", Rendering{Entity: "nobody", Prose: "x"}), http.StatusBadRequest)
}
