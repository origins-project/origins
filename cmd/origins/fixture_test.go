package main

import (
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain points git at configuration of the package's own for every test in it, so
// that nothing a test asserts depends on how the machine running it is set up. It is
// done once here rather than by each test, because Go will not run a test that changes
// the environment beside any other; a test that needs git configured some other way
// sets it itself and runs alone. It carries an identity, because init and import make
// the repository and commit in one go, with no moment in between for a test to set
// one on the repository.
func TestMain(m *testing.M) {
	if os.Getenv(asOrigins) != "" {
		main()
	}
	dir, err := os.MkdirTemp("", "origins-test")
	if err != nil {
		panic(err)
	}
	config := filepath.Join(dir, "git-config")
	identity := "[user]\n\tname = Rosa Bianchi\n\temail = rosa@example.invalid\n" +
		"[core]\n\tautocrlf = false\n"
	if err := os.WriteFile(config, []byte(identity), 0o644); err != nil {
		panic(err)
	}
	os.Setenv("GIT_CONFIG_GLOBAL", config)
	os.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(dir, "no-such-system-config"))
	// git packs no test's repository after a commit, none living long enough to need
	// it, and commits as nobody it has not been told of, rather than guessing at an
	// identity from the machine, which on some machines it finds
	os.Setenv("GIT_CONFIG_COUNT", "2")
	os.Setenv("GIT_CONFIG_KEY_0", "maintenance.auto")
	os.Setenv("GIT_CONFIG_VALUE_0", "false")
	os.Setenv("GIT_CONFIG_KEY_1", "user.useConfigOnly")
	os.Setenv("GIT_CONFIG_VALUE_1", "true")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// gitFixture is a copy of testgit, the fixture corpus built through origins from
// nothing, with the history a researcher would have made getting there. A test is free
// to change the copy, which commits as TestMain's configuration says.
//
// It is generated rather than committed and takes a couple of minutes to build, so this
// skips when it is not there. Run tools/mkgitfixture.sh to produce it.
func gitFixture(t *testing.T) string {
	t.Helper()
	built := filepath.Join("..", "..", "testgit")
	if _, err := os.Stat(built); err != nil {
		t.Skip("no git fixture; run tools/mkgitfixture.sh")
	}
	dir := t.TempDir()

	root := filepath.Join(dir, "corpus")
	if err := os.CopyFS(root, os.DirFS(built)); err != nil {
		t.Fatal(err)
	}
	return root
}

// The younger Iván was renamed to make room for his grandfather after his birth and
// death had been written, which moved his file and stated nothing about him. Blame
// follows the file back across the move, so each fact is still the work of the command
// that stated it.
func TestBlameOnTheGitFixtureReachesPastAFileBeingMoved(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	report := answering[api.Provenance](t, "blame", "-C", root, "person/ershov-ivan-1885")
	if report.File != "people/ershov-ivan-1885/README.md" {
		t.Fatalf("blamed %s", report.File)
	}

	found := fields(report)
	for path, want := range map[string]struct {
		op     string
		source string
	}{
		"name":            {"person.create", ""},
		"events[0].date":  {"events.add", ""},
		"events[1].value": {"events.add", ""},
		"events[2].date":  {"events.add", ""},
		"body":            {"body.set", ""},
	} {
		one, ok := found[path]
		if !ok || one.Commit == nil {
			t.Errorf("%s is the work of no commit", path)
			continue
		}
		if one.Commit.Op != want.op || strings.Join(one.Commit.Sources, " ") != want.source {
			t.Errorf("%s is %q from %q, want %q from %q",
				path, one.Commit.Op, one.Commit.Sources, want.op, want.source)
		}
	}
	for _, one := range report.Fields {
		if one.Commit != nil && one.Commit.Op == "person.rename" {
			t.Errorf("%s is credited to %q, which moved the file", one.Path, one.Commit.Subject)
		}
	}
}
