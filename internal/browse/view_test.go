package browse

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// hidings is each view a read can ask for, with the parameter it asks with.
var hidings = []struct {
	asked  string
	hiding model.Hiding
}{
	{HidePrivate, model.Hiding{Private: true}},
	{HideLiving, model.Hiding{Living: true}},
	{HidePrivate + "," + HideLiving, model.Hiding{Private: true, Living: true}},
}

// hiddenFrom says nothing the view leaves out of the corpus at root can be reached
// through what a page is served: not a record, not a file kept beside one, and not the
// index. query is what each read carries to ask for the view.
func hiddenFrom(t *testing.T, c *client, root string, hiding model.Hiding, query string) {
	t.Helper()
	whole, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	view := model.Hide(whole.Corpus, hiding, testDay)
	index := c.get("/api/index" + query)
	if index.Code != http.StatusOK {
		t.Fatalf("the index was answered %d: %s", index.Code, index.Body)
	}
	listed := index.Body.String()

	gone, kept := 0, 0
	for _, slug := range whole.AllSlugs() {
		kind := whole.KindOf(slug)
		ref := model.EntityRef{Kind: kind, Slug: slug}
		if kind == model.KindHypothesis {
			continue
		}
		files := slices.Clone(*model.Files(whole.Entity(slug)))
		if view.KindOf(slug) != kind {
			gone++
			if response := c.get("/api/records/" + ref.String() + query); response.Code != http.StatusNotFound {
				t.Errorf("%s is hidden and was answered %d", ref, response.Code)
			}
			if strings.Contains(listed, `"`+string(slug)+`"`) || strings.Contains(listed, `/`+string(slug)+`"`) {
				t.Errorf("%s is hidden and the index names it", ref)
			}
		} else {
			shown := *model.Files(view.Entity(slug))
			files = slices.DeleteFunc(files, func(file string) bool { return slices.Contains(shown, file) })
			if response := c.get("/api/records/" + ref.String() + query); response.Code != http.StatusOK {
				t.Errorf("%s is in the view and was answered %d", ref, response.Code)
			}
		}
		for _, file := range files {
			kept++
			if response := c.get("/files/" + ref.String() + "/" + file + query); response.Code != http.StatusNotFound {
				t.Errorf("%s/%s is hidden and was served", ref, file)
			}
		}
	}
	if gone == 0 {
		t.Errorf("the view hiding %s hid no record, so nothing was tested", hiding)
	}
	if hiding.Private && kept == 0 {
		t.Errorf("the view hiding %s hid no file, so nothing was tested", hiding)
	}

	for _, slug := range view.PeopleSlugs() {
		if strings.HasPrefix(string(slug), "private-") {
			placeholder := read[Page](c, c.get("/api/records/person/"+string(slug)+query), http.StatusOK)
			if written, _ := json.Marshal(placeholder.Record.Fields); !strings.Contains(string(written), model.PlaceholderName) {
				t.Errorf("the placeholder %s is not named %s: %s", slug, model.PlaceholderName, placeholder.Record.Fields)
			}
			return
		}
	}
	t.Errorf("the view hiding %s has no placeholder", hiding)
}

func TestAHiddenViewIsAskedFor(t *testing.T) {
	t.Parallel()
	root := fixture(t)
	c := serving(t, root)
	for _, view := range hidings {
		hiddenFrom(t, c, root, view.hiding, "?"+ViewParam+"="+view.asked)
	}
	session := read[Session](c, c.get("/api/session"), http.StatusOK)
	if session.View != nil || !session.Writes {
		t.Errorf("a server locked to no view says %+v", session)
	}
}

func TestAServerIsLockedToAView(t *testing.T) {
	t.Parallel()
	for _, view := range hidings {
		root := fixture(t)
		c := servingWith(t, root, Options{Hiding: view.hiding})
		hiddenFrom(t, c, root, view.hiding, "")
		session := read[Session](c, c.get("/api/session"), http.StatusOK)
		if session.Writes || session.View == nil ||
			session.View.Private != view.hiding.Private || session.View.Living != view.hiding.Living {
			t.Errorf("a server locked to hide %s says %+v", view.hiding, session)
		}
	}
}

// readOnly says every kind of change is refused as one a copy cannot take, and that
// nothing was written.
func readOnly(t *testing.T, c *client, root, query string) {
	t.Helper()
	before := head(t, root)
	edit := raw(t, map[string]any{"op": "add", "entity": "person/verdier-jean-baptiste", "path": "tags", "value": "checked"})
	for target, body := range map[string]any{
		"/api/edit": Change{Edit: edit, Seen: c.seen("person/verdier-jean-baptiste")},
		"/api/undo": Settling{},
		"/api/hypotheses/unrecorded-child-1845/accept": Settling{},
		"/api/hypotheses/unrecorded-child-1845/reject": Settling{},
	} {
		refusal := read[Refusal](c, c.post(target+query, body), http.StatusForbidden)
		if refusal.Kind != "read-only" {
			t.Errorf("%s%s was refused as %s: %s", target, query, refusal.Kind, refusal.Error)
		}
	}
	if head(t, root) != before {
		t.Errorf("a copy wrote to the corpus")
	}
}

func TestACopyTakesNoChange(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	for _, view := range hidings {
		readOnly(t, c, root, "?"+ViewParam+"="+view.asked)
	}
	for _, view := range hidings {
		readOnly(t, servingWith(t, root, Options{Hiding: view.hiding}), root, "")
	}
	readOnly(t, servingWith(t, root, Options{With: []model.Slug{"unrecorded-child-1845"}}), root, "")
	readOnly(t, c, root, "?"+WithParam+"=unrecorded-child-1845")
}

func TestAHypothesisIsServedSupposed(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := servingWith(t, root, Options{With: []model.Slug{"unrecorded-child-1845"}})
	session := read[Session](c, c.get("/api/session"), http.StatusOK)
	if session.Writes || len(session.With) != 1 || session.With[0].Hypothesis != "unrecorded-child-1845" ||
		session.With[0].Base == "" {
		t.Errorf("a server supposing unrecorded-child-1845 says %+v", session)
	}
	index := read[api.Index](c, c.get("/api/index"), http.StatusOK)
	if len(index.Read) != 1 {
		t.Errorf("the index says it was read with %+v", index.Read)
	}
	if c.get("/api/records/person/verdier-unknown-1845").Code != http.StatusOK {
		t.Error("the person the hypothesis creates is not served")
	}
	if whole, err := store.Load(root); err != nil || whole.KindOf("verdier-unknown-1845") != "" {
		t.Errorf("the corpus itself was changed: %v", err)
	}

	hidden := read[Page](c, c.get("/api/records/person/verdier-unknown-1845?"+ViewParam+"="+HideLiving), http.StatusOK)
	if hidden.Record.Ref != "person/verdier-unknown-1845" {
		t.Errorf("a supposition is not viewed as hidden: %s", hidden.Record.Ref)
	}
}

func TestAReadAsksForAHypothesisSupposed(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	query := "?" + WithParam + "=unrecorded-child-1845"
	if c.get("/api/records/person/verdier-unknown-1845").Code != http.StatusNotFound {
		t.Error("the person the hypothesis creates is served without it supposed")
	}
	if c.get("/api/records/person/verdier-unknown-1845"+query).Code != http.StatusOK {
		t.Error("the person the hypothesis creates is not served with it supposed")
	}
	family := read[Page](c, c.get("/api/records/family/lang-anna-maria--verdier-louis"+query), http.StatusOK)
	if written, _ := json.Marshal(family.Record.Fields); !strings.Contains(string(written), "verdier-unknown-1845") {
		t.Errorf("the family the hypothesis adds a child to does not have it: %s", written)
	}
	index := read[api.Index](c, c.get("/api/index"+query), http.StatusOK)
	if len(index.Read) != 1 || index.Read[0].Hypothesis != "unrecorded-child-1845" {
		t.Errorf("the index says it was read with %+v", index.Read)
	}
	if index := read[api.Index](c, c.get("/api/index"), http.StatusOK); len(index.Read) != 0 {
		t.Errorf("the index asked for without a hypothesis says it was read with %+v", index.Read)
	}
	hidden := read[Page](c, c.get("/api/records/person/verdier-unknown-1845"+query+"&"+ViewParam+"="+HideLiving), http.StatusOK)
	if hidden.Record.Ref != "person/verdier-unknown-1845" {
		t.Errorf("a supposition is not viewed as hidden: %s", hidden.Record.Ref)
	}
	if session := read[Session](c, c.get("/api/session"), http.StatusOK); !session.Writes || len(session.With) != 0 {
		t.Errorf("a server asked for a supposition by one read says %+v", session)
	}
	if code := c.get("/api/index?" + WithParam + "=nobody-proposed-this").Code; code == http.StatusOK {
		t.Error("a hypothesis that is not there was supposed")
	}
	if refusal := read[Refusal](c, c.get("/api/index"+query+"&"+DraftParam), http.StatusBadRequest); refusal.Kind != "asked" {
		t.Errorf("a draft asked for with a supposition was refused as %s", refusal.Kind)
	}
	if whole, err := store.Load(root); err != nil || whole.KindOf("verdier-unknown-1845") != "" {
		t.Errorf("the corpus itself was changed: %v", err)
	}
}

func TestAHiddenViewHasNoHistory(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	c := serving(t, root)
	for _, target := range []string{
		"/api/log?", "/api/whence?source=parish-san-lorenzo-baptisms&",
		"/api/blame?entity=person/verdier-jean-baptiste&", "/api/why?entity=person/verdier-jean-baptiste&path=sex&",
	} {
		refusal := read[Refusal](c, c.get(target+ViewParam+"="+HidePrivate), http.StatusForbidden)
		if refusal.Kind != "hidden" {
			t.Errorf("%s under a view was refused as %s", target, refusal.Kind)
		}
	}
	locked := servingWith(t, root, Options{Hiding: model.Hiding{Living: true}})
	if refusal := read[Refusal](locked, locked.get("/api/log"), http.StatusForbidden); refusal.Kind != "hidden" {
		t.Errorf("the log of a server locked to a view was refused as %s", refusal.Kind)
	}
	if refusal := read[Refusal](c, c.get("/api/index?"+ViewParam+"=everything"), http.StatusBadRequest); refusal.Kind != "asked" {
		t.Errorf("a view nobody knows was refused as %s", refusal.Kind)
	}
}
