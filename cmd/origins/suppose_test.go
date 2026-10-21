package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aMissingFather is theRossis with a family whose father nobody has a record of, which
// check calls an error and a hypothesis can answer.
func aMissingFather() map[string]string {
	files := theRossis()
	files["families/rossi-antonio--unknown/README.md"] = "---\npartners: [rossi-antonio, unknown]\n" +
		"children: [rossi-giovanni]\n---\n"
	return files
}

// thereWasAnAntonio is the record that would answer for him, as a hypothesis pinning
// whatever commit it is handed.
func thereWasAnAntonio(base string) string {
	return "---\nstatus: open\nbase: " + base + "\nconfidence: possible\n" +
		"edits:\n  - op: create\n    entity: person/rossi-antonio\n" +
		"    value:\n      name:\n        given: Antonio\n        surname: Rossi\n      sex: M\n" +
		"---\n\n# Was there an Antonio?\n\nGiovanni's record names a father.\n"
}

// heWasARossini is a second reading of the same corpus, touching a different record,
// so that two at once is a question about two facts rather than one.
func heWasARossini(base string) string {
	return "---\nstatus: open\nbase: " + base + "\nconfidence: possible\n" +
		"edits:\n  - op: set\n    entity: person/rossi-giovanni\n" +
		"    path: name.surname\n    value: Rossini\n" +
		"---\n\n# Was the family called Rossini?\n\nThe 1861 return spells it so.\n"
}

func put(t *testing.T, root, relative, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The whole of what --with is for, end to end: the corpus contradicts itself, the
// hypothesis says why it need not, and check answers the question without anything
// being recorded or the corpus being touched.
func TestCheckWithAHypothesisAnswersAsThoughItWereTrue(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	put(t, root, "hypotheses/there-was-an-antonio/README.md", thereWasAnAntonio(base))

	if code, stdout, _ := invoke(t, "check", root); code != exitFindings {
		t.Fatalf("check exited %d on a corpus with a dangling partner\n%s", code, stdout)
	}

	code, stdout, stderr := invoke(t, "check", "--with", "there-was-an-antonio", root)
	if code != exitOK {
		t.Errorf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if strings.Contains(stdout, "dangling-partner") {
		t.Errorf("the hypothesis did not reach the corpus:\n%s", stdout)
	}
	if want := "there-was-an-antonio read at " + base; !strings.Contains(stdout, want) {
		t.Errorf("stdout does not say %q:\n%s", want, stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-antonio", "README.md")); !os.IsNotExist(err) {
		t.Errorf("supposing a hypothesis wrote it into the corpus: %v", err)
	}
}

// Which of the two readings it was is said every time, because the corpus that comes
// out of them is the same corpus either way and a researcher cannot otherwise tell.
func TestCheckWithSaysWhenAHypothesisWasReadAgainstTheCorpus(t *testing.T) {
	t.Parallel()
	files := aMissingFather()
	files["hypotheses/there-was-an-antonio/README.md"] = thereWasAnAntonio("8f78553")
	root := corpusDir(t, files)

	code, stdout, stderr := invoke(t, "check", "--with", "there-was-an-antonio", root)
	if code != exitOK {
		t.Errorf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, says := range []string{
		"there-was-an-antonio read against the corpus in front of you",
		"this corpus is in no repository",
	} {
		if !strings.Contains(stdout, says) {
			t.Errorf("stdout does not say %q:\n%s", says, stdout)
		}
	}
}

// Whether two of them can both be true is the question several at once puts, and each
// has to say for itself where it was read.
func TestCheckTakesMoreThanOneHypothesis(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	put(t, root, "hypotheses/there-was-an-antonio/README.md", thereWasAnAntonio(base))
	put(t, root, "hypotheses/he-was-a-rossini/README.md", heWasARossini(base))

	code, stdout, stderr := invoke(t, "check", "--json",
		"--with", "there-was-an-antonio", "--with", "he-was-a-rossini", root)
	if code != exitOK {
		t.Errorf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var report api.Validation
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if len(report.Read) != 2 {
		t.Fatalf("read %v, want both hypotheses", report.Read)
	}
	for i, want := range []string{"there-was-an-antonio", "he-was-a-rossini"} {
		if report.Read[i].Hypothesis != want || report.Read[i].Base != base {
			t.Errorf("read %v, want %s at %s", report.Read[i], want, base)
		}
	}
}

// A name nobody wrote a hypothesis under is a mistake in the asking, which is the
// other side of the line the exit codes draw.
func TestCheckWithAHypothesisTheCorpusDoesNotHaveIsMisuse(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, aMissingFather())

	code, _, stderr := invoke(t, "check", "--with", "there-was-an-antonio", root)
	if code != exitMisuse {
		t.Errorf("exit %d, want %d", code, exitMisuse)
	}
	if !strings.Contains(stderr, "there-was-an-antonio") {
		t.Errorf("stderr does not name the hypothesis: %q", stderr)
	}
}

// A hypothesis somebody has overtaken exits the way a contradiction does, and says
// which file the two readings are in.
func TestCheckWithAHypothesisThatNoLongerAppliesReportsTheConflict(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	base := git(t, root, "rev-parse", "HEAD")
	put(t, root, "hypotheses/he-was-a-rossini/README.md", heWasARossini(base))

	if code, _, stderr := invoke(t, "set", "-C", root,
		"person/rossi-giovanni", "name.surname", "Rossetti"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}

	code, _, stderr := invoke(t, "check", "--with", "he-was-a-rossini", root)
	if code != exitFindings {
		t.Errorf("exit %d, want %d: %s", code, exitFindings, stderr)
	}
	for _, says := range []string{"he-was-a-rossini", "no longer applies", "rossi-giovanni/README.md"} {
		if !strings.Contains(stderr, says) {
			t.Errorf("stderr does not say %q: %q", says, stderr)
		}
	}
}
