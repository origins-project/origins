package store

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// copyFixture puts the fixture corpus somewhere it can be rewritten.
func copyFixture(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "test")
	if _, err := os.Stat(source); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	root := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func format(t *testing.T, root string, write bool) []string {
	t.Helper()
	corpus, problems := Load(root)
	if problems != nil {
		t.Fatalf("the corpus should load cleanly:\n%v", problems)
	}
	changed, err := Format(corpus, write)
	if err != nil {
		t.Fatalf("formatting %s: %v", root, err)
	}
	return changed
}

// The guarantee a corpus is kept under: once formatted, formatting again does nothing.
// Anything else and the next real edit to a file produces a diff carrying both the
// change and a reformatting, which is exactly the fact-level blame the scheme exists to
// provide.
func TestFormattingTwiceChangesNothingTheSecondTime(t *testing.T) {
	root := copyFixture(t)
	format(t, root, true)
	if changed := format(t, root, true); len(changed) > 0 {
		t.Errorf("a second pass rewrote %d files: %v", len(changed), changed)
	}
}

// The fixture is the corpus everything else in this repository is tested against, so it
// is the one that has to be canonical — otherwise every test reads a format origins
// would never write.
func TestTheFixtureIsAlreadyCanonical(t *testing.T) {
	root := copyFixture(t)
	if changed := format(t, root, false); len(changed) > 0 {
		t.Errorf("the fixture is not canonical, %d files differ: %v\n"+
			"regenerate it with tools/mkfixture.sh, or update the script from a formatted copy",
			len(changed), changed)
	}
}

// Reporting without writing has to mean exactly that, because it is what a corpus is
// examined with when the answer must not be to change it.
func TestListingChangesNothingOnDisk(t *testing.T) {
	root := copyFixture(t)
	scruffy := filepath.Join(root, "places", "firenze", "README.md")
	original, err := os.ReadFile(scruffy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scruffy, []byte(strings.Replace(string(original),
		"type: city", "type: city\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	messy, err := os.ReadFile(scruffy)
	if err != nil {
		t.Fatal(err)
	}

	if changed := format(t, root, false); !slices.Contains(changed, "places/firenze/README.md") {
		t.Errorf("a file with a stray blank line was not reported: %v", changed)
	}
	after, err := os.ReadFile(scruffy)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(messy) {
		t.Error("listing rewrote the file it was only asked about")
	}

	if changed := format(t, root, true); !slices.Contains(changed, "places/firenze/README.md") {
		t.Errorf("writing did not rewrite the file: %v", changed)
	}
	if fixed, _ := os.ReadFile(scruffy); string(fixed) != string(original) {
		t.Errorf("the file was not put back in canonical form:\n%s", fixed)
	}
}

// What is canonical is a question about the format, so a corpus in one this origins
// does not know is neither rewritten nor listed as though it were in its own.
func TestACorpusInALaterFormatIsNotFormatted(t *testing.T) {
	root := copyFixture(t)
	manifest := filepath.Join(root, ProjectFile)
	original, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	later := strings.Replace(string(original), "format-version: 1", "format-version: 2\n", 1)
	if err := os.WriteFile(manifest, []byte(later), 0o644); err != nil {
		t.Fatal(err)
	}
	corpus, problems := Load(root)
	if problems != nil {
		t.Fatal(problems)
	}

	for _, write := range []bool{false, true} {
		changed, err := Format(corpus, write)
		if err == nil || !strings.Contains(err.Error(), "format-version 2") {
			t.Errorf("formatting with write=%v said %v", write, err)
		}
		if len(changed) > 0 {
			t.Errorf("formatting with write=%v reported %v", write, changed)
		}
	}
	if after, _ := os.ReadFile(manifest); string(after) != later {
		t.Errorf("the manifest was rewritten:\n%s", after)
	}
}

// A file origins cannot read is a file it must not write. Emitting it would mean
// emitting a model that never held its contents, which is the one way a formatter can
// destroy what it was pointed at.
func TestAFileThatDoesNotLoadIsLeftAlone(t *testing.T) {
	root := copyFixture(t)
	broken := filepath.Join(root, "people", "verdier-anna", "README.md")
	garbage := []byte("---\nname: Anna Rossi\nnot-a-field: nonsense\n---\n")
	if err := os.WriteFile(broken, garbage, 0o644); err != nil {
		t.Fatal(err)
	}

	corpus, problems := Load(root)
	if problems == nil {
		t.Fatal("an unknown field should have been reported")
	}
	changed, err := Format(corpus, true)
	if err != nil {
		t.Fatalf("formatting the rest of the corpus: %v", err)
	}
	if slices.Contains(changed, "people/verdier-anna/README.md") {
		t.Error("the unreadable file was rewritten")
	}
	if left, _ := os.ReadFile(broken); string(left) != string(garbage) {
		t.Errorf("the unreadable file was changed:\n%s", left)
	}
}

// Normalising to NFC at the one point every byte enters the corpus is what stops the
// same name emitting two different ways depending on which editor last saved it.
func TestDecomposedTextIsNormalisedOnTheWayIn(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("origins.yaml", "project:\n  name: x\norigins:\n  format-version: 1\n")
	write("people/a-one/README.md", "---\nname: José Rossi\nsex: M\n---\n")

	corpus, problems := Load(root)
	if problems != nil {
		t.Fatal(problems)
	}
	if got := corpus.People["a-one"].Name.Text; got != "José Rossi" {
		t.Errorf("name = %q, want the composed form", got)
	}
	if changed := format(t, root, true); !slices.Contains(changed, "people/a-one/README.md") {
		t.Errorf("the decomposed file should have been rewritten: %v", changed)
	}
	if again := format(t, root, true); len(again) > 0 {
		t.Errorf("normalising again rewrote %v", again)
	}
}
