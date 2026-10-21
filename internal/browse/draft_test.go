package browse

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
)

// drafting takes one edit into the draft as a page would, pinned to the record as the
// page was shown it: with the draft in it, where there is a draft.
func (c *client) drafting(edit map[string]any, status int) api.Drafting {
	c.t.Helper()
	entity, _ := edit["entity"].(string)
	seen := c.seen(entity)
	if read[*api.Drafting](c, c.get("/api/draft"), http.StatusOK) != nil {
		seen = c.seen(entity + "?" + DraftParam)
	}
	return read[api.Drafting](c, c.post("/api/draft/edit", Change{Edit: raw(c.t, edit), Seen: seen}), status)
}

// drafted is a record as the corpus with the draft in it shows it.
func (c *client) drafted(ref string) Page {
	c.t.Helper()
	return read[Page](c, c.get("/api/records/"+ref+"?"+DraftParam), http.StatusOK)
}

func tagged(page Page, tag string) bool {
	tags, _ := page.Record.Fields["tags"].([]any)
	return slices.Contains(tags, any(tag))
}

var threeRecords = []map[string]any{
	{"op": "create", "entity": "person/test-bozza",
		"value": map[string]any{"name": map[string]any{"given": "Bozza", "surname": "Test"}}},
	{"op": "add", "entity": "person/chen-yuying", "path": "tags", "value": "drafted"},
	{"op": "add", "entity": "place/kumasi", "path": "tags", "value": "drafted"},
	{"op": "set", "entity": "person/test-bozza", "path": "sex", "value": "M"},
}

func TestADraftIsKeptAsAHypothesis(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	before := head(t, root)

	if none := read[*api.Drafting](c, c.get("/api/draft"), http.StatusOK); none != nil {
		t.Errorf("no draft was shown as %+v", none)
	}
	var shown api.Drafting
	for _, edit := range threeRecords {
		shown = c.drafting(edit, http.StatusOK)
	}
	if shown.Base != before || len(shown.Edits) != 4 || len(shown.Stated) != 4 ||
		!slices.Equal(shown.Records, []string{"person/test-bozza", "person/chen-yuying", "place/kumasi"}) {
		t.Errorf("the draft is shown as %+v", shown)
	}
	if head(t, root) != before || git(t, root, "status", "--porcelain") != "" {
		t.Fatal("drafting changed the corpus or its history")
	}
	if c.get("/api/records/person/test-bozza").Code != http.StatusNotFound {
		t.Error("a record drafted is served without the draft being asked for")
	}
	if bozza := c.drafted("person/test-bozza"); bozza.Record.Fields["sex"] != "M" {
		t.Errorf("the drafted record is %v", bozza.Record.Fields)
	}
	if !tagged(c.drafted("person/chen-yuying"), "drafted") || !tagged(c.drafted("place/kumasi"), "drafted") {
		t.Error("an edit drafted is not in the corpus the draft is read from")
	}

	popped := read[api.Drafting](c, c.post("/api/draft/pop", struct{}{}), http.StatusOK)
	if len(popped.Edits) != 3 || c.drafted("person/test-bozza").Record.Fields["sex"] != nil {
		t.Errorf("popping left %+v", popped)
	}
	c.drafting(threeRecords[3], http.StatusOK)
	stale := Change{Edit: raw(t, map[string]any{"op": "set", "entity": "person/chen-yuying", "path": "sex", "value": "X"}),
		Seen: c.seen("person/chen-yuying")}
	if refusal := read[Refusal](c, c.post("/api/draft/edit", stale), http.StatusConflict); refusal.Kind != "stale" {
		t.Errorf("an edit pinned to the record without the draft was refused as %s", refusal.Kind)
	}

	want := map[string]string{}
	for _, ref := range []string{"person/test-bozza", "person/chen-yuying", "place/kumasi"} {
		want[ref] = c.drafted(ref).Record.Seen
	}
	kept := read[api.Proposal](c, c.post("/api/draft/keep", Kept{
		Slug: "teresa-bozza", Question: "Is Bozza Test somebody?", Argument: "Drafted from the pages.",
		Confidence: "possible",
	}), http.StatusOK)
	if kept.Commit != head(t, root) || kept.Base != before || len(kept.Edits) != 4 {
		t.Errorf("keeping the draft answered %+v", kept)
	}
	if n := git(t, root, "rev-list", "--count", before+"..HEAD"); n != "1" {
		t.Errorf("keeping the draft made %s commits", n)
	}
	if files := git(t, root, "diff", "--name-only", before, "HEAD"); files != "hypotheses/teresa-bozza/README.md" {
		t.Errorf("keeping the draft wrote %s", files)
	}
	if git(t, root, "status", "--porcelain") != "" {
		t.Error("keeping the draft left something unrecorded")
	}
	if read[*api.Drafting](c, c.get("/api/draft"), http.StatusOK) != nil {
		t.Error("the draft was kept and is still there")
	}
	written := c.get("/api/records/hypothesis/teresa-bozza").Body.String()
	for _, said := range []string{`"# Is Bozza Test somebody?\n\nDrafted from the pages."`, `"possible"`, before} {
		if !strings.Contains(written, said) {
			t.Errorf("the hypothesis kept does not say %s:\n%s", said, written)
		}
	}

	supposing := servingWith(t, root, Options{With: []model.Slug{"teresa-bozza"}})
	for ref, seen := range want {
		if got := read[Page](supposing, supposing.get("/api/records/"+ref), http.StatusOK).Record.Seen; got != seen {
			t.Errorf("%s supposing the hypothesis kept is not %s as the draft showed it", ref, ref)
		}
	}
}

func TestARestartedServerTakesUpTheDraft(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	c.drafting(threeRecords[1], http.StatusOK)

	again := serving(t, root)
	shown := read[api.Drafting](again, again.get("/api/draft"), http.StatusOK)
	if len(shown.Edits) != 1 || !tagged(again.drafted("person/chen-yuying"), "drafted") {
		t.Errorf("a server started again shows the draft as %+v", shown)
	}
	if again.do(http.MethodDelete, "/api/draft", nil, nil).Code != http.StatusOK {
		t.Fatal("the draft was not thrown away")
	}
	c.server.mu.Lock()
	c.server.settle(c.server.state())
	c.server.mu.Unlock()
	if refusal := read[Refusal](c, c.get("/api/records/person/chen-yuying?"+DraftParam), http.StatusNotFound); refusal.Kind != "missing" {
		t.Errorf("a read asking for a draft another server threw away was refused as %s", refusal.Kind)
	}
	if refusal := read[Refusal](c, c.do(http.MethodDelete, "/api/draft", nil, nil), http.StatusNotFound); refusal.Kind != "missing" {
		t.Errorf("throwing away no draft was refused as %s", refusal.Kind)
	}
}

func TestADraftIsCarriedOverACommitMadeMeanwhile(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	before := head(t, root)
	c.drafting(threeRecords[1], http.StatusOK)

	c.change(map[string]any{"op": "add", "entity": "person/verdier-jean-baptiste", "path": "tags", "value": "meanwhile"}, http.StatusOK)
	shown := read[api.Drafting](c, c.get("/api/draft"), http.StatusOK)
	if shown.Base != before || shown.Since != 1 || len(shown.Drifted) != 0 {
		t.Errorf("a draft behind the last commit is shown as %+v", shown)
	}
	if !tagged(c.drafted("person/chen-yuying"), "drafted") || !tagged(c.drafted("person/verdier-jean-baptiste"), "meanwhile") {
		t.Error("the corpus with the draft in it is not the corpus as it stands with the draft in it")
	}

	shown = c.drafting(threeRecords[2], http.StatusOK)
	if shown.Base != head(t, root) || shown.Since != 0 || len(shown.Edits) != 2 {
		t.Errorf("a draft edited after a commit is shown as %+v", shown)
	}
	if !tagged(c.drafted("person/chen-yuying"), "drafted") || !tagged(c.drafted("place/kumasi"), "drafted") {
		t.Error("a draft brought forward lost an edit")
	}
}

func TestADraftThatDriftedSaysWhat(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	c.drafting(map[string]any{"op": "set", "entity": "person/chen-yuying", "path": "sex", "value": "M"}, http.StatusOK)
	c.change(map[string]any{"op": "set", "entity": "person/chen-yuying", "path": "sex", "value": "X"}, http.StatusOK)

	shown := read[api.Drafting](c, c.get("/api/draft"), http.StatusOK)
	if len(shown.Drifted) == 0 || len(shown.Edits) != 1 {
		t.Errorf("a draft that drifted is shown as %+v", shown)
	}
	if refusal := read[Refusal](c, c.get("/api/records/person/chen-yuying?"+DraftParam), http.StatusConflict); refusal.Kind != "drifted" {
		t.Errorf("a read asking for a draft that drifted was refused as %s", refusal.Kind)
	}
	refusal := read[Refusal](c, c.post("/api/draft/keep", Kept{Slug: "drifted", Question: "Was she?"}), http.StatusConflict)
	if refusal.Kind != "drifted" {
		t.Errorf("keeping a draft that drifted was refused as %s", refusal.Kind)
	}
	if c.do(http.MethodDelete, "/api/draft", nil, nil).Code != http.StatusOK {
		t.Error("a draft that drifted cannot be thrown away")
	}
}

func TestACopyTakesNoDraft(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	c.drafting(threeRecords[1], http.StatusOK)

	viewed := "/api/records/person/chen-yuying?" + DraftParam + "&" + ViewParam + "=" + HidePrivate
	if refusal := read[Refusal](c, c.get(viewed), http.StatusBadRequest); refusal.Kind != "asked" {
		t.Errorf("a draft under a view was refused as %s", refusal.Kind)
	}
	supposing := servingWith(t, root, Options{With: []model.Slug{"unrecorded-child-1845"}})
	if refusal := read[Refusal](supposing, supposing.get("/api/draft"), http.StatusForbidden); refusal.Kind != "read-only" {
		t.Errorf("the draft of a server supposing was refused as %s", refusal.Kind)
	}
	edit := Change{Edit: raw(t, threeRecords[2])}
	if refusal := read[Refusal](supposing, supposing.post("/api/draft/edit", edit), http.StatusForbidden); refusal.Kind != "read-only" {
		t.Errorf("drafting on a server supposing was refused as %s", refusal.Kind)
	}
}
