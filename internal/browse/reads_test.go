package browse

import (
	"bytes"
	"image/jpeg"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
)

func TestSessionSaysWhatIsServed(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	session := read[Session](c, c.get("/api/session"), http.StatusOK)
	if !session.Writes || session.View != nil || len(session.With) != 0 || len(session.Problems) != 0 {
		t.Errorf("session = %+v, want a whole corpus served as it is, writes taken", session)
	}
	if session.Project == "" {
		t.Error("the session names no project")
	}
}

func TestTheIndexListsEveryRecord(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	index := read[api.Index](c, c.get("/api/index"), http.StatusOK)
	counts := index.Counts()
	if counts.People != 46 || counts.Families != 25 || counts.Places != 27 || counts.Sources != 17 ||
		counts.Repositories != 11 || counts.Notes != 2 || counts.Hypotheses != 2 {
		t.Errorf("the index counts %+v", counts)
	}
}

func TestEveryRecordIsServedWithWhatIsAroundIt(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	corpus := c.server.corpus
	for _, kind := range model.Kinds {
		for _, slug := range corpus.SlugsOf(kind) {
			ref := model.EntityRef{Kind: kind, Slug: slug}
			page := read[Page](c, c.get("/api/records/"+ref.String()), http.StatusOK)
			if page.Record.Ref != ref.String() || page.Record.HTML == nil || page.Around == nil {
				t.Errorf("%s served as %+v", ref, page.Record.Record)
			}
			for _, near := range page.Around {
				if near.Ref == ref.String() {
					t.Errorf("%s is around itself", ref)
				}
			}
		}
	}
}

func TestAPageHoldsTheRecordsOneStepOut(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	page := read[Page](c, c.get("/api/records/person/verdier-jean-baptiste"), http.StatusOK)
	around := map[string]bool{}
	for _, near := range page.Around {
		around[near.Ref] = true
	}
	// a family he is a partner in refers to him, and a source his prose and citations name
	for _, want := range []string{"family/papadopoulou-maria--verdier-jean-baptiste", "source/french-church-london-baptisms"} {
		if !around[want] {
			t.Errorf("verdier-jean-baptiste's page holds %v, and not %s", around, want)
		}
	}
}

func TestProseIsRenderedWithItsLinksRouted(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	page := read[Page](c, c.get("/api/records/person/verdier-jean-baptiste"), http.StatusOK)
	body := page.Record.HTML["body"]
	for _, want := range []string{`href="/person/lang-georg"`, `href="/source/french-church-london-baptisms#1834-042"`} {
		if !strings.Contains(body, want) {
			t.Errorf("the body has no %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, "README.md") {
		t.Errorf("the body still links to a file of the corpus:\n%s", body)
	}

	source := read[Page](c, c.get("/api/records/source/parish-san-lorenzo-baptisms"), http.StatusOK)
	if !strings.Contains(source.Record.HTML["body"], "<u>") {
		t.Errorf("the source's <u> was not kept:\n%s", source.Record.HTML["body"])
	}
}

func TestProseRunsNoScript(t *testing.T) {
	t.Parallel()
	m := newMarkdown()
	owner := model.EntityRef{Kind: model.KindPerson, Slug: "verdier-jean-baptiste"}
	for _, written := range []string{
		"<script>alert(1)</script>",
		"text <span onclick=\"alert(1)\">here</span>",
		"<img src=x onerror=alert(1)>",
		"<u onmouseover='alert(1)'>under</u>",
	} {
		rendered := m.render(owner, written)
		if strings.Contains(rendered, "<script") || strings.Contains(rendered, "<span on") ||
			strings.Contains(rendered, "<img") || strings.Contains(rendered, "<u on") {
			t.Errorf("%q rendered as %q", written, rendered)
		}
	}
	for _, written := range []string{"a <sup>2</sup>", `<span style="color: red">red</span>`, "<mark>this</mark>"} {
		if rendered := m.render(owner, written); strings.Contains(rendered, "&lt;") {
			t.Errorf("%q rendered as %q, its tag escaped", written, rendered)
		}
	}
}

func TestProseLinksToFilesBesideTheRecord(t *testing.T) {
	t.Parallel()
	m := newMarkdown()
	owner := model.EntityRef{Kind: model.KindPerson, Slug: "verdier-jean-baptiste"}
	rendered := m.render(owner, "![portrait](photos/1890-portrait.jpg) and [elsewhere](https://example.org/a)")
	for _, want := range []string{`src="/files/person/verdier-jean-baptiste/photos/1890-portrait.jpg"`, `href="https://example.org/a"`} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered without %s:\n%s", want, rendered)
		}
	}
	if heading := m.render(owner, "# Notes"); !strings.Contains(heading, "<h2") {
		t.Errorf("a heading in prose rendered as %q, not a level down", heading)
	}
}

func TestTheProjectIsServed(t *testing.T) {
	t.Parallel()
	root := fixture(t)
	c := serving(t, root)
	page := read[Page](c, c.get("/api/records/project"), http.StatusOK)
	if page.Record.Ref != "project" {
		t.Errorf("the project is served as %q", page.Record.Ref)
	}
	if written, err := os.ReadFile(filepath.Join(root, "origins.yaml")); err != nil || page.Record.Written != string(written) {
		t.Errorf("the manifest is served written as\n%s\nand not as it is on disk (%v)", page.Record.Written, err)
	}
}

func TestARecordNotThereIsMissing(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	for _, target := range []string{
		"/api/records/person/nobody-at-all",
		"/api/records/family/verdier-jean-baptiste",
		"/api/records/tag/x",
		"/api/records/elephant/verdier-jean-baptiste",
		"/api/nothing-here",
	} {
		refusal := read[Refusal](c, c.get(target), http.StatusNotFound)
		if refusal.Kind != "missing" {
			t.Errorf("%s refused as %q", target, refusal.Kind)
		}
	}
}

func TestFindingsAreServed(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	findings := read[api.Findings](c, c.get("/api/findings"), http.StatusOK)
	if findings.Errors != 0 || findings.Warnings == 0 || len(findings.Todo) == 0 {
		t.Errorf("findings = %d errors, %d warnings, %d left to do", findings.Errors, findings.Warnings, len(findings.Todo))
	}
}

func TestKinshipIsServed(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	kin := read[api.Kin](c, c.get("/api/kinship?from=verdier-jean-baptiste&to=rossi-caterina"), http.StatusOK)
	if !kin.Related {
		t.Errorf("kinship = %+v", kin)
	}
	read[Refusal](c, c.get("/api/kinship?from=verdier-jean-baptiste"), http.StatusBadRequest)
}

func TestTheSchemaIsServed(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	for _, target := range []string{"/api/schema", "/api/schema/person"} {
		response := c.get(target)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"$schema"`) {
			t.Errorf("%s answered %d:\n%.200s", target, response.Code, response.Body)
		}
	}
	read[Refusal](c, c.get("/api/schema/elephant"), http.StatusNotFound)
}

func TestADateIsRead(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	got := read[api.DateRead](c, c.get("/api/date?text="+url.QueryEscape("abt  12 mar 1834 (so the priest wrote)")), http.StatusOK)
	if want := (api.DateRead{Date: "ABT 12 MAR 1834 (so the priest wrote)", Anchored: true, Sort: "1834-03-12"}); got != want {
		t.Errorf("read %+v, want %+v", got, want)
	}
	if got := read[api.DateRead](c, c.get("/api/date?text=3+MAR"), http.StatusOK); got.Anchored || got.Sort != "" {
		t.Errorf("a floating date read %+v", got)
	}
	for _, bad := range []string{"", "31+FEB+1834", "BET+1850+AND+1840"} {
		read[Refusal](c, c.get("/api/date?text="+bad), http.StatusBadRequest)
	}
}

func TestTheHistoryIsServed(t *testing.T) {
	t.Parallel()
	c := serving(t, gitFixture(t))
	for _, target := range []string{
		"/api/log?n=3",
		"/api/log?entity=person/verdier-jean-baptiste",
		"/api/whence?source=parish-san-lorenzo-baptisms",
		"/api/blame?entity=person/verdier-jean-baptiste",
		"/api/why?entity=person/verdier-jean-baptiste&path=name",
	} {
		if response := c.get(target); response.Code != http.StatusOK {
			t.Errorf("%s answered %d:\n%s", target, response.Code, response.Body)
		}
	}
	read[Refusal](c, c.get("/api/log?n=many"), http.StatusBadRequest)
	read[Refusal](c, c.get("/api/whence"), http.StatusBadRequest)
}

func TestAttachedFilesAreServed(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	whole := c.get("/files/person/verdier-jean-baptiste/photos/1890-portrait.jpg")
	if whole.Code != http.StatusOK || whole.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("the portrait answered %d, %s", whole.Code, whole.Header().Get("Content-Type"))
	}
	part := c.get("/files/family/papadopoulou-maria--verdier-jean-baptiste/photos/1858-wedding-group.jpg?region=0.41,0.12,0.58,0.47")
	if part.Code != http.StatusOK {
		t.Fatalf("the region answered %d", part.Code)
	}
	cut, err := jpeg.Decode(bytes.NewReader(part.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	wholeGroup := c.get("/files/family/papadopoulou-maria--verdier-jean-baptiste/photos/1858-wedding-group.jpg")
	group, err := jpeg.Decode(bytes.NewReader(wholeGroup.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if cut.Bounds().Dx() >= group.Bounds().Dx() || cut.Bounds().Dy() >= group.Bounds().Dy() {
		t.Errorf("the region is %v of a picture %v", cut.Bounds(), group.Bounds())
	}

	for _, target := range []string{
		"/files/person/verdier-jean-baptiste/photos/no-such.jpg",
		"/files/person/verdier-jean-baptiste/README.md",
		"/files/person/verdier-jean-baptiste/../../origins.yaml",
		"/files/person/verdier-jean-baptiste/..%2f..%2forigins.yaml",
		"/files/family/verdier-jean-baptiste/photos/1890-portrait.jpg",
	} {
		if response := c.get(target); response.Code == http.StatusOK {
			t.Errorf("%s was served", target)
		}
	}
}

func TestAnUnreadableCorpusIsServedReadOnly(t *testing.T) {
	t.Parallel()
	root := fixture(t)
	broken := filepath.Join(root, "people", "verdier-jean-baptiste", "README.md")
	if err := os.WriteFile(broken, []byte("---\nname: [unclosed\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := serving(t, root)
	session := read[Session](c, c.get("/api/session"), http.StatusOK)
	if session.Writes || len(session.Problems) == 0 {
		t.Errorf("session = %+v, want problems and no writes", session)
	}
	index := read[api.Index](c, c.get("/api/index"), http.StatusOK)
	if len(index.Problems) == 0 || len(index.People) == 0 {
		t.Errorf("the index of an unreadable corpus has %d people and problems %v", len(index.People), index.Problems)
	}
}

func TestTheGuardTurnsAwayOtherPages(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	for name, header := range map[string]map[string]string{
		"no origin":    {"Origin": ""},
		"other origin": {"Origin": "http://evil.example"},
		"no token":     {TokenHeader: ""},
		"wrong token":  {TokenHeader: "0123"},
	} {
		refusal := read[Refusal](c, c.do(http.MethodPost, "/api/nothing", strings.NewReader("{}"), header), http.StatusForbidden)
		if refusal.Kind != "forbidden" {
			t.Errorf("%s: refused as %q", name, refusal.Kind)
		}
	}
	if response := c.do(http.MethodPost, "/api/nothing", strings.NewReader("{}"), nil); response.Code != http.StatusNotFound {
		t.Errorf("a post from the server's own page answered %d, not past the guard", response.Code)
	}

	for _, host := range []string{"evil.example", "127.0.0.1:9999", ""} {
		if recorder := record(c, newRequest(http.MethodGet, "/api/session", host)); recorder.Code != http.StatusForbidden {
			t.Errorf("a request to %q answered %d", host, recorder.Code)
		}
	}
	if recorder := record(c, newRequest(http.MethodGet, "/api/session", "localhost:8100")); recorder.Code != http.StatusOK {
		t.Errorf("a request to localhost answered %d", recorder.Code)
	}
}

func TestThePageCarriesTheToken(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	meta := `<meta name="` + TokenMeta + `" content="` + c.server.token + `">`
	for _, target := range []string{"/", "/person/verdier-jean-baptiste"} {
		response := c.get(target)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), meta) {
			t.Errorf("%s answered %d:\n%s", target, response.Code, response.Body)
		}
	}

	// the placeholder, which is served where the application was not built, and the
	// application where it was
	placeholder, err := c.server.drawn(nil)
	if err != nil || !strings.Contains(string(placeholder), meta) ||
		!strings.Contains(string(placeholder), "npm --prefix web run build") {
		t.Errorf("the placeholder (%v):\n%s", err, placeholder)
	}
	if built != nil {
		application, err := c.server.drawn(built)
		if err != nil || !strings.Contains(string(application), meta) ||
			!strings.Contains(string(application), `<div id="app">`) {
			t.Errorf("the application (%v):\n%s", err, application)
		}
		assets, _ := fs.ReadDir(built, "assets")
		if len(assets) == 0 {
			t.Fatal("the application was built with no assets")
		}
		response := c.get("/assets/" + assets[0].Name())
		if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Cache-Control"), "immutable") {
			t.Errorf("an asset answered %d, %q", response.Code, response.Header().Get("Cache-Control"))
		}
	}
}
