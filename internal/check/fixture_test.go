package check

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/store"
)

// The fixture corpus is generated rather than committed, so this skips when it is not
// there. Run tools/mkfixture.sh to produce it.
func TestFixtureIsClean(t *testing.T) {
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	corpus, problems := store.Load(root)
	if problems != nil {
		t.Fatalf("the fixture should load cleanly:\n%v", problems)
	}

	findings := Run(corpus.Corpus)
	for _, f := range findings {
		if f.Severity == Error {
			t.Errorf("%s %s: %s (%s)", f.Entity, f.Path, f.Message, f.Rule)
		}
	}

	// the fixture is a research corpus with more assertions than citations, which is
	// ordinary, and it keeps one disproven link, one place's coordinates as words, one
	// picture still to be found and two people said to be one to show them; anything
	// else warning is worth knowing about
	warned := ruleIDs(findings, Warning)
	if !slices.Equal(warned, []string{"coordinates-as-words", "disproven-link", "missing-media", "same-as", "unsourced-event"}) {
		t.Errorf("warnings came from %v, expected only coordinates as words, the disproven link, "+
			"the picture still to be found, the claim that two people are one and unsourced events", warned)
	}
}
