package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// nowhere is a directory with nothing in it and no repository under it, which is where
// a corpus that does not exist yet begins.
func nowhere(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// Starting a corpus has to leave one that every other command will work on, which
// means a repository, a manifest, and a commit origins will recognise as its own.
// Changing it straight afterwards is the whole of the claim: no git by hand, no
// recover, nothing between having no corpus and having one that can be worked on.
func TestStartingACorpusLeavesOneThatCanBeChanged(t *testing.T) {
	t.Parallel()
	root := filepath.Join(nowhere(t), "rossi")

	code, stdout, stderr := invoke(t, "init", "--name", "Rossi family", root)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if got := read(t, root, "origins.yaml"); !strings.Contains(got, "name: Rossi family") ||
		!strings.Contains(got, "format-version: 1") {
		t.Errorf("origins.yaml = %q", got)
	}
	if got := trailer(t, root, "Origins-Op"); got != "corpus.init" {
		t.Errorf("Origins-Op = %q", got)
	}
	if got := git(t, root, "status", "--porcelain"); got != "" {
		t.Errorf("the corpus should be recorded whole, and git says %q", got)
	}

	code, stdout, stderr = invoke(t, "create", "-C", root, "person/rossi-giovanni", "name=Giovanni Rossi")
	if code != exitOK {
		t.Fatalf("a corpus just started should take a change: exit %d\nstdout: %s\nstderr: %s",
			code, stdout, stderr)
	}
}

// The reason the manifest cannot wait for the first change to carry it in.
//
// A commit names its paths, so a change made before the manifest was recorded would
// record a corpus with no manifest in it — and reading the last commit back is how a
// change made by hand is judged and how a hypothesis is taken in. Every one of those
// would fail from then on, with nothing to say why and no way back.
func TestTheManifestIsRecordedSoLaterCommitsCanBeJudgedAgainstIt(t *testing.T) {
	t.Parallel()
	root := filepath.Join(nowhere(t), "rossi")
	if code, _, stderr := invoke(t, "init", root); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if code, _, stderr := invoke(t, "create", "-C", root, "person/rossi-giovanni",
		"name=Giovanni Rossi"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}

	// a change nobody made through origins, which is judged against the last commit
	hand := "---\nname:\n  given: Giovanni\n  surname: Rossi\n---\n"
	if err := os.WriteFile(filepath.Join(root, "people", "rossi-giovanni", "README.md"), []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := invoke(t, "adopt", "-C", root)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
}

// A corpus started inside a repository somebody already keeps is recorded in that one,
// beside the notes and photographs it belongs with, and the commits that were there
// before are not origins' business: the corpus did not exist when they were made, so
// there is nothing for the test of ownership to be about and nothing to recover.
func TestACorpusStartedInsideARepositoryJoinsIt(t *testing.T) {
	t.Parallel()
	outside := nowhere(t)
	git(t, outside, "init", "-q")
	if err := os.WriteFile(filepath.Join(outside, "NOTES.md"), []byte("what the aunts said\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, outside, "add", "-A")
	git(t, outside, "commit", "-q", "-m", "My notes")

	root := filepath.Join(outside, "tree")
	code, stdout, stderr := invoke(t, "init", "--name", "Rossi family", root)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if strings.Contains(stdout, "a new git repository") {
		t.Errorf("a repository was already here, and init reported making one:\n%s", stdout)
	}
	got := git(t, root, "rev-parse", "--show-toplevel")
	recorded, errGot := os.Stat(got)
	want, errWant := os.Stat(outside)
	if errGot != nil || errWant != nil || !os.SameFile(recorded, want) {
		t.Errorf("recorded in %q, want the repository at %q", got, outside)
	}

	// the foreign commit below is not something to recover from, because it is not
	// about this corpus
	if code, stdout, stderr = invoke(t, "create", "-C", root, "person/rossi-giovanni",
		"name=Giovanni Rossi"); code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if got := read(t, outside, "NOTES.md"); got != "what the aunts said\n" {
		t.Errorf("the notes were disturbed: %q", got)
	}
}

// Starting a corpus where there is one is refused rather than written over. The
// manifest is the one thing that says a directory is a corpus at all, so it is also
// the one thing init reads before it writes anything.
func TestStartingACorpusThatIsAlreadyOneIsRefused(t *testing.T) {
	t.Parallel()
	root := filepath.Join(nowhere(t), "rossi")
	if code, _, stderr := invoke(t, "init", root); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	before := read(t, root, "origins.yaml")

	code, _, stderr := invoke(t, "init", "--name", "Something else", root)
	if code != exitMisuse {
		t.Errorf("exit %d, want %d", code, exitMisuse)
	}
	if !strings.Contains(stderr, "is a corpus already") {
		t.Errorf("stderr = %q", stderr)
	}
	if after := read(t, root, "origins.yaml"); after != before {
		t.Errorf("the manifest was written over:\n%s", after)
	}
}

func TestACorpusTakesTheDirectorysNameWhenNobodyNamesIt(t *testing.T) {
	t.Parallel()
	root := filepath.Join(nowhere(t), "verdi-tree")
	if code, _, stderr := invoke(t, "init", root); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got := read(t, root, "origins.yaml"); !strings.Contains(got, "name: verdi-tree") {
		t.Errorf("origins.yaml = %q", got)
	}
}

// The ordinary way it is run: in the directory, with nothing said about which. The name
// still has to be the directory's own, which means resolving where "." is rather than
// taking the last part of it.
func TestACorpusStartedInTheDirectoryItIsRunInTakesThatName(t *testing.T) {
	root := filepath.Join(nowhere(t), "verdi-tree")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	if code, _, stderr := invoke(t, "init"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got := read(t, root, "origins.yaml"); !strings.Contains(got, "name: verdi-tree") {
		t.Errorf("origins.yaml = %q", got)
	}
}

// Init records the manifest and nothing else, so a directory somebody has already been
// writing files into keeps them, unrecorded, and is told so. Recording them is what
// adopt is for, and it works from here because the commit init made can be read back.
func TestStartingACorpusSaysWhatItFoundAndDidNotRecord(t *testing.T) {
	t.Parallel()
	root := filepath.Join(nowhere(t), "rossi")
	if err := os.MkdirAll(filepath.Join(root, "people", "rossi-giovanni"), 0o755); err != nil {
		t.Fatal(err)
	}
	hand := "---\nname:\n  given: Giovanni\n  surname: Rossi\n---\n"
	if err := os.WriteFile(filepath.Join(root, "people", "rossi-giovanni", "README.md"), []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := invoke(t, "init", "--name", "Rossi family", root)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "people/rossi-giovanni/README.md") || !strings.Contains(stdout, "origins adopt") {
		t.Errorf("stdout = %q", stdout)
	}
	if got := git(t, root, "log", "--format=%s", "--name-only", "-1"); strings.Contains(got, "rossi-giovanni") {
		t.Errorf("init recorded more than the manifest:\n%s", got)
	}

	if code, stdout, stderr = invoke(t, "adopt", "-C", root); code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
}

// A commit git refuses takes the manifest back with it, and the declaration of line
// endings with it. A manifest with no history behind it is the state this command
// exists to prevent, and it is also one init would refuse to start again, so leaving
// it would wedge the directory for good.
func TestACommitGitRefusesLeavesNoManifestBehind(t *testing.T) {
	outside := nowhere(t)
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(outside, "no-such-global-config"))
	root := filepath.Join(outside, "rossi")

	code, stdout, stderr := invoke(t, "init", "--name", "Rossi family", root)
	if code != exitMisuse {
		t.Fatalf("exit %d, want %d\nstdout: %s\nstderr: %s", code, exitMisuse, stdout, stderr)
	}
	for _, left := range []string{"origins.yaml", ".gitattributes"} {
		if _, err := os.Stat(filepath.Join(root, left)); !os.IsNotExist(err) {
			t.Errorf("%s is still there: %v", left, err)
		}
	}
	if !strings.Contains(stderr, "nothing has been kept") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestStartingACorpusReportsItAsJSON(t *testing.T) {
	t.Parallel()
	root := filepath.Join(nowhere(t), "rossi")
	code, stdout, stderr := invoke(t, "init", "--json", "--name", "Rossi family", root)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var report api.Begun
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if report.Name != "Rossi family" || report.Subject != "Start the Rossi family corpus" {
		t.Errorf("report = %+v", report)
	}
	if !report.Started {
		t.Error("a repository was made here, and the report says it was not")
	}
	if report.Commit != git(t, root, "rev-parse", "HEAD") {
		t.Errorf("commit = %q", report.Commit)
	}
	if report.Waiting == nil {
		t.Error("waiting should be an empty list rather than absent")
	}
}

func TestStartingIsRefusedBeforeItStarts(t *testing.T) {
	t.Parallel()
	outside := nowhere(t)
	one, other := filepath.Join(outside, "one"), filepath.Join(outside, "other")
	for _, refusal := range []struct {
		what string
		args []string
		says string
	}{
		{"two corpora at once", []string{"init", one, other}, "one corpus at a time"},
		{"a flag after the directory", []string{"init", one, "--name", "x"}, "flags come first"},
	} {
		t.Run(refusal.what, func(t *testing.T) {
			code, _, stderr := invoke(t, refusal.args...)
			if code != exitMisuse {
				t.Errorf("exit %d, want %d", code, exitMisuse)
			}
			if !strings.Contains(stderr, refusal.says) {
				t.Errorf("stderr = %q", stderr)
			}
		})
	}
	// refused before anything was written, which is the half a message cannot say
	if _, err := os.Stat(one); !os.IsNotExist(err) {
		t.Errorf("a refused init made the directory anyway: %v", err)
	}
}
