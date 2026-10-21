package gedcom

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/origins-project/origins/internal/roundtrip"
	"github.com/origins-project/origins/internal/store"
)

// The fixture written as GEDCOM and read back is the fixture again, record by record,
// in everything but what the report counts as left out.
func TestTheFixtureComesBackFromItsExport(t *testing.T) {
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(filepath.Join(root, store.ProjectFile)); err != nil {
		t.Skip("the fixture is not built: run tools/mkfixture.sh")
	}
	corpus, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	_, nodes, report := export(t, corpus.Corpus, Options{Version: "test"})
	read, _, _ := Convert(nodes, nil, Beside(root))
	differences, err := roundtrip.Compare(corpus.Corpus, read)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range differences {
		if !roundtrip.Counted(d, report.Omitted) {
			t.Error(d)
		}
	}
}
