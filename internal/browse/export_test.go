package browse

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// application is an application as Vite writes one, its page, its script and its
// WebAssembly, enough for an export to be made of.
var application = fstest.MapFS{
	"index.html":          {Data: []byte(`<!doctype html><html><head><script defer id="origins" src="./assets/index.js"></script></head><body></body></html>`)},
	"assets/index.js":     {Data: []byte("console.log('origins')\n")},
	"assets/chart-1.wasm": {Data: []byte("\x00asm\x01\x00\x00\x00")},
}

// The export of the fixture is the application with its page marked as an export's, an
// answer beside it for every read its pages make of a server that takes no change and
// keeps no history, every file a record keeps where the pages ask for it, and the
// WebAssembly as a script.
func TestAnExportIsTheApplicationWithEveryAnswerBesideIt(t *testing.T) {
	t.Parallel()
	root := fixture(t)
	out := filepath.Join(t.TempDir(), "site")
	report, err := export(root, out, Options{}, application)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}

	page, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil || !strings.Contains(string(page), `<meta name="`+ExportMeta+`" content=""></head>`) {
		t.Errorf("the page is %q: %v", page, err)
	}
	if script, err := os.ReadFile(filepath.Join(out, "assets", "index.js")); err != nil || string(script) != "console.log('origins')\n" {
		t.Errorf("the script is %q: %v", script, err)
	}

	var session Session
	readAnswer(t, out, "/api/session", &session)
	if session.Writes || session.Live || session.History || session.View == nil || session.View.Private || session.View.Living {
		t.Errorf("the session says %+v", session)
	}
	records := 0
	for _, kind := range model.Kinds {
		for _, slug := range loaded.SlugsOf(kind) {
			var page struct {
				Record struct {
					Ref string `json:"ref"`
				} `json:"record"`
			}
			readAnswer(t, out, "/api/records/"+string(kind)+"/"+string(slug), &page)
			if page.Record.Ref != string(kind)+"/"+string(slug) {
				t.Errorf("%s/%s answers %q", kind, slug, page.Record.Ref)
			}
			records++
			if report.Records[kind.Dir()] == 0 {
				t.Errorf("the report counts no %s", kind.Dir())
			}
		}
	}
	if report.Answers != records+5 {
		t.Errorf("wrote %d answers, for %d records", report.Answers, records)
	}

	files := 0
	for _, kind := range model.Kinds {
		for _, slug := range loaded.SlugsOf(kind) {
			for _, name := range loaded.Beside(slug) {
				files++
				want, err := os.ReadFile(filepath.Join(root, kind.Dir(), string(slug), filepath.FromSlash(name)))
				if err != nil {
					t.Fatal(err)
				}
				got, err := os.ReadFile(filepath.Join(out, "files", string(kind), string(slug), filepath.FromSlash(name)))
				if err != nil || string(got) != string(want) {
					t.Errorf("%s/%s/%s was written as %d bytes: %v", kind, slug, name, len(got), err)
				}
			}
		}
	}
	if files == 0 || report.Files != files {
		t.Errorf("wrote %d files of %d", report.Files, files)
	}

	var wasm string
	readAnswer(t, out, ChartPath, &wasm)
	if wasm != "AGFzbQEAAAA=" {
		t.Errorf("the WebAssembly is %q", wasm)
	}
	_ = filepath.WalkDir(out, func(name string, entry fs.DirEntry, err error) error {
		if err == nil && filepath.Ext(name) == ".wasm" {
			t.Errorf("%s is written, which a page opened from a folder cannot fetch", name)
		}
		return err
	})
}

// An export's session says what it was made from: when, the commit the corpus stood on
// and whether its files differed from it, the origins that made it, its view, the
// hypotheses it supposes and its notice; a corpus without a history stood on no commit.
func TestAnExportSaysWhatItWasMadeFrom(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	head := git(t, root, "rev-parse", "HEAD")
	exported := func(root string, options Options) Session {
		t.Helper()
		out := filepath.Join(t.TempDir(), "site")
		if _, err := export(root, out, options, application); err != nil {
			t.Fatal(err)
		}
		var session Session
		readAnswer(t, out, "/api/session", &session)
		if session.Export == nil {
			t.Fatalf("the session says nothing of the export: %+v", session)
		}
		return session
	}

	before := time.Now().Add(-time.Second)
	session := exported(root, Options{Notice: "A demo."})
	made := *session.Export
	if made.At.Before(before) || made.At.After(time.Now()) {
		t.Errorf("made at %s", made.At)
	}
	if made.Commit != head || made.Changed || made.Version != api.Version || made.Notice != "A demo." {
		t.Errorf("made from %+v, where HEAD is %s and origins %s", made, head, api.Version)
	}

	person := filepath.Join(root, "people", "al-halabi-1889", "README.md")
	written, err := os.ReadFile(person)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(person, append(written, "\nMore.\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	session = exported(root, Options{Hiding: model.Hiding{Living: true}, With: []model.Slug{"louisa-born-1844"}})
	if made := *session.Export; made.Commit != head || !made.Changed || made.Notice != "" {
		t.Errorf("made from %+v, with a file changed since %s", made, head)
	}
	if session.View == nil || session.View.Private || !session.View.Living {
		t.Errorf("made as %+v", session.View)
	}
	if len(session.With) != 1 || session.With[0].Hypothesis != "louisa-born-1844" {
		t.Errorf("supposing %+v", session.With)
	}

	if made := *exported(fixture(t), Options{}).Export; made.Commit != "" || made.Changed {
		t.Errorf("a corpus without a history made from %+v", made)
	}
}

func TestAnExportIsWrittenWhereNothingIs(t *testing.T) {
	t.Parallel()
	root, dir := fixture(t), t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kept"), []byte("kept"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := export(root, dir, Options{}, application); err == nil {
		t.Error("an export was written over a directory holding something")
	}
	if kept, err := os.ReadFile(filepath.Join(dir, "kept")); err != nil || string(kept) != "kept" {
		t.Errorf("the directory holds %q: %v", kept, err)
	}
	out := filepath.Join(dir, "site")
	if _, err := export(root, out, Options{}, nil); err == nil || !strings.Contains(err.Error(), "npm --prefix web") {
		t.Errorf("an origins without the application exported: %v", err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a refused export left %s: %v", out, err)
	}
	if _, err := export(root, out, Options{With: []model.Slug{"no-such-hypothesis"}}, application); err == nil {
		t.Error("an export supposed a hypothesis that is not there")
	}
}

// readAnswer reads the answer an export wrote for a route, as the page's client does.
func readAnswer(t *testing.T, out, route string, into any) {
	t.Helper()
	script, err := os.ReadFile(filepath.Join(out, "data", filepath.FromSlash(route)+".js"))
	if err != nil {
		t.Fatal(err)
	}
	quoted, _ := json.Marshal(route)
	text, ok := strings.CutPrefix(string(script), AnswerFunction+"("+string(quoted)+",")
	if !ok {
		t.Fatalf("%s is %.80q", route, script)
	}
	text, ok = strings.CutSuffix(text, ");\n")
	if !ok {
		t.Fatalf("%s ends %q", route, script[max(0, len(script)-20):])
	}
	if err := json.Unmarshal([]byte(text), into); err != nil {
		t.Fatalf("%s: %v", route, err)
	}
}
