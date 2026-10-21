package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// windows is nowhere on a machine configured the way Git for Windows configures one,
// converting every line ending to CRLF when it checks a file out.
func windows(t *testing.T) string {
	t.Helper()
	outside := t.TempDir()
	config := filepath.Join(outside, "git-config")
	identity := "[user]\n\tname = Rosa Bianchi\n\temail = rosa@example.invalid\n" +
		"[core]\n\tautocrlf = true\n"
	if err := os.WriteFile(config, []byte(identity), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(outside, "no-such-system-config"))
	return outside
}

func canonical(t *testing.T, root, after string) {
	t.Helper()
	if code, stdout, stderr := invoke(t, "fmt", "--list", root); code != exitOK || stdout != "" {
		t.Errorf("after %s, fmt --list exited %d naming\n%s%s", after, code, stdout, stderr)
	}
}

// An undo has git put the files back, and in a corpus that declares nothing about its
// line endings only origins can see to it that they come back as they were committed.
func TestAnUndoLeavesTheCorpusCanonicalWhereGitWouldWriteCRLF(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	git(t, root, "config", "core.autocrlf", "true")

	for _, args := range [][]string{
		{"set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze"},
		{"undo", "-C", root},
	} {
		if code, stdout, stderr := invoke(t, args...); code != exitOK {
			t.Fatalf("%s exited %d\n%s%s", args[0], code, stdout, stderr)
		}
	}
	canonical(t, root, "an undo")
}

// A clone is git writing every file with origins nowhere near it, and what reaches it
// is the declaration the corpus was started with.
func TestAClonedCorpusIsCanonicalWhereGitWouldWriteCRLF(t *testing.T) {
	outside := windows(t)
	root := filepath.Join(outside, "rossi")
	for _, args := range [][]string{
		{"init", root},
		{"create", "-C", root, "person/rossi-giovanni", "name=Giovanni Rossi"},
	} {
		if code, stdout, stderr := invoke(t, args...); code != exitOK {
			t.Fatalf("%s exited %d\n%s%s", args[0], code, stdout, stderr)
		}
	}

	clone := filepath.Join(outside, "clone")
	git(t, outside, "clone", "-q", root, clone)
	canonical(t, clone, "a clone")
}

// Accepting a hypothesis has git check out the commit it pins and carry the result onto
// the corpus, and the corpus it leaves has to be canonical all the same.
func TestAnAcceptedHypothesisLeavesTheCorpusCanonicalWhereGitWouldWriteCRLF(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	git(t, root, "config", "core.autocrlf", "true")
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio(git(t, root, "rev-parse", "HEAD")))

	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "there-was-an-antonio"); code != exitOK {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	canonical(t, root, "an accepted hypothesis")
}

// Every way a corpus comes to be origins' own records the declaration in the commit
// that makes it so: starting one, importing one, and taking on one somebody kept in git
// without it. Recording it later would leave every commit before that checking out
// wrongly on the next machine.
func TestEveryWayACorpusIsTakenOnDeclaresItsLineEndings(t *testing.T) {
	t.Parallel()
	recorded := func(t *testing.T, root string) {
		t.Helper()
		if touched := git(t, root, "show", "--name-only", "--format=", "HEAD"); !strings.Contains(touched, ".gitattributes") {
			t.Errorf("the commit records\n%s\nand no declaration", touched)
		}
		if left := git(t, root, "status", "--porcelain"); left != "" {
			t.Errorf("something was left unrecorded:\n%s", left)
		}
	}

	t.Run("init", func(t *testing.T) {
		root := filepath.Join(nowhere(t), "rossi")
		code, stdout, stderr := invoke(t, "init", root)
		if code != exitOK {
			t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
		}
		if !strings.Contains(stdout, "  .gitattributes\n") {
			t.Errorf("init does not say it recorded the declaration:\n%s", stdout)
		}
		recorded(t, root)
	})
	t.Run("import", func(t *testing.T) {
		root := freshCorpus(t)
		if code, stdout, stderr := invoke(t, "import", "gedcom", "--into", root, exportFile(t)); code != exitOK {
			t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
		}
		recorded(t, root)
	})
	t.Run("recover --adopt", func(t *testing.T) {
		root := inARepo(t, theRossis())
		edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
		git(t, root, "commit", "-q", "-am", "Firenze is a comune")
		if code, stdout, stderr := invoke(t, "recover", "-C", root, "--adopt"); code != exitOK {
			t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
		}
		recorded(t, root)
	})
}

// The declaration names what origins writes and not a byte more: a photograph marked
// as text would be corrupted on the way out, and a file somebody keeps beside the
// records is theirs to declare.
func TestTheDeclarationReachesTheRecordsAndNothingElse(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	paths := append(strings.Split(git(t, root, "ls-files"), "\n"),
		"README.md", "research/people/rossi-giovanni/README.md", "research/origins.yaml")
	answered := git(t, root, append([]string{"check-attr", "eol", "--"}, paths...)...)

	records := 0
	for _, line := range strings.Split(answered, "\n") {
		path, value, _ := strings.Cut(line, ": eol: ")
		kind, _, _ := strings.Cut(path, "/")
		record := path == "origins.yaml" || strings.HasSuffix(path, ".md") &&
			strings.Contains(" people families places sources repositories notes hypotheses ", " "+kind+" ")
		if record {
			records++
		}
		if record != (value == "lf") {
			t.Errorf("%s is declared %s", path, value)
		}
	}
	if records < 50 {
		t.Errorf("only %d records were asked about", records)
	}
}
