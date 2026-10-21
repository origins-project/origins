package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// A hidden view of the fixture, written out, names nothing it hides, and refers to
// nothing it does not hold. What is looked for is every slug the view no longer has and
// the words only what it hides would put there.
func TestTheHiddenFixtureNamesNothingItHides(t *testing.T) {
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	loaded, problems := store.Load(root)
	if problems != nil {
		t.Fatalf("the fixture should load cleanly:\n%v", problems)
	}
	corpus := loaded.Corpus
	today := model.DateOf(time.Date(2026, time.September, 28, 12, 0, 0, 0, time.UTC))

	private := []string{
		"Yumiko", "佐藤", "Tōno", "岩手", "koseki", // a private person, and where she came from
		"La Paz",                   // a private event
		"until: 16 FEB 1983",       // a private other name
		"At home, with",            // a private picture
		"Kinkerstraat",             // a private place
		"Recorded interview",       // a private source
		"Ingrid de Vries's papers", // and the private repository it is kept in
		"The boatman of",           // a private note, and the link to it
	}
	for _, v := range []struct {
		hiding model.Hiding
		words  []string
	}{
		{model.Hiding{Private: true}, private},
		{model.Hiding{Living: true}, []string{"Marco", "Sasha", "Lucas"}},
		{model.Hiding{Private: true, Living: true}, append(private, "Marco", "Sasha", "Lucas")},
	} {
		hidden := model.Hide(corpus, v.hiding, today)

		var written bytes.Buffer
		project, err := store.EmitProject(hidden.Project)
		if err != nil {
			t.Fatal(err)
		}
		written.Write(project)
		for _, slug := range hidden.AllSlugs() {
			file, err := store.Emit(hidden.Entity(slug))
			if err != nil {
				t.Fatalf("%+v: %s: %v", v.hiding, slug, err)
			}
			written.Write(file)
		}

		for _, slug := range corpus.AllSlugs() {
			if hidden.Entity(slug) == nil && bytes.Contains(written.Bytes(), []byte(slug)) {
				t.Errorf("%+v: %s is not in the view and is named in it", v.hiding, slug)
			}
		}
		for _, word := range v.words {
			if bytes.Contains(written.Bytes(), []byte(word)) {
				t.Errorf("%+v: %q is in the view", v.hiding, word)
			}
		}

		// two placeholders are alike because the view made them so, which is nothing
		// for a researcher to look into
		for _, f := range Run(hidden) {
			if f.Severity == Error || f.Rule == "duplicate-person" && strings.Contains(f.Message, "private-") {
				t.Errorf("%+v: %s %s: %s (%s)", v.hiding, f.Entity, f.Path, f.Message, f.Rule)
			}
		}
	}
}
