package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cloned(t *testing.T, url, dir string) (int, api.Cloning, string) {
	t.Helper()
	var report api.Cloning
	code, stdout, stderr := invoke(t, "clone", "--json", url, dir)
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("exit %d, %v\nstdout: %s\nstderr: %s", code, err, stdout, stderr)
	}
	return code, report, stderr
}

// A clone of a corpus origins wrote throughout is settled at once, the marker set where
// the history stops, so that a change made in it needs nothing run first.
func TestCloneOfACorpusSettlesItsHistory(t *testing.T) {
	t.Parallel()
	bare := bareFrom(t, gitFixture(t))
	dir := filepath.Join(filepath.Dir(bare), "rossi")

	code, report, stderr := cloned(t, bare, dir)
	head := git(t, dir, "rev-parse", "HEAD")
	if code != exitOK || !report.Corpus || report.Errors != 0 || !report.Settled || report.Marker != head {
		t.Fatalf("exit %d, %+v\n%s", code, report, stderr)
	}
	if marker := git(t, dir, "rev-parse", "refs/origins/head"); marker != head {
		t.Errorf("the marker is at %s, HEAD at %s", marker, head)
	}
	changed(t, "set", "-C", dir, "person/verdier-anna", "body", "Anna, as her nephew remembered her.")
	if upstream := git(t, dir, "rev-parse", "--abbrev-ref", "main@{upstream}"); upstream != "origin/main" {
		t.Errorf("the clone's main follows %s", upstream)
	}
}

// A commit origins did not write is kept with the rest, and recover is named.
func TestCloneOverACommitOriginsDidNotWriteIsKeptAndPointsAtRecover(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	edited(t, root, "notes.txt", "a list of what to ask zia Maria\n")
	git(t, root, "add", "notes.txt")
	git(t, root, "commit", "-q", "-m", "What to ask")
	bare := bareFrom(t, root)
	dir := filepath.Join(filepath.Dir(bare), "rossi")

	code, report, _ := cloned(t, bare, dir)
	if code != exitFindings || !report.Corpus || report.Settled || len(report.Foreign) != 1 ||
		report.Foreign[0].Subject != "What to ask" || report.Marker != "" {
		t.Errorf("exit %d, %+v", code, report)
	}
	if marker := git(t, dir, "for-each-ref", "refs/origins"); marker != "" {
		t.Errorf("the marker was set: %s", marker)
	}
	code, stdout, _ := invoke(t, "clone", bare, filepath.Join(filepath.Dir(bare), "again"))
	if code != exitFindings || !strings.Contains(stdout, "origins recover -C") || !strings.Contains(stdout, "What to ask") {
		t.Errorf("exit %d: %s", code, stdout)
	}
}

func TestCloneOfARepositoryWithNoCorpusIsKeptAndSaysSo(t *testing.T) {
	t.Parallel()
	plain := t.TempDir()
	git(t, plain, "init", "-q", "-b", "main")
	edited(t, plain, "README.md", "Not a family tree.\n")
	git(t, plain, "add", "README.md")
	git(t, plain, "commit", "-q", "-m", "Begin")
	dir := filepath.Join(t.TempDir(), "plain")

	code, stdout, _ := invoke(t, "clone", plain, dir)
	if code != exitFindings || !strings.Contains(stdout, "no corpus") {
		t.Errorf("exit %d: %s", code, stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err != nil {
		t.Error("the clone was not kept")
	}
}

func TestCloneOfARepositoryThatIsNotThereMakesNothing(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "no-such.git")
	dir := filepath.Join(t.TempDir(), "rossi")
	code, _, stderr := invoke(t, "clone", missing, dir)
	if code != exitMisuse || !strings.Contains(stderr, "did not finish") {
		t.Errorf("exit %d: %s", code, stderr)
	}
	if _, err := os.Stat(dir); err == nil {
		t.Error("the failed clone left a directory")
	}
}

// Git for Windows checks every file out with CRLF unless told otherwise, which would
// leave a fresh clone with every file not as origins writes it.
func TestCloneChecksFilesOutAsTheyWereCommitted(t *testing.T) {
	root := inARepo(t, theRossis())
	bare := bareFrom(t, root)
	windows(t)
	dir := filepath.Join(filepath.Dir(bare), "rossi")

	code, stdout, stderr := invoke(t, "clone", bare, dir)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if strings.Contains(read(t, dir, "origins.yaml"), "\r") {
		t.Error("the manifest was checked out with CRLF")
	}
	if code, stdout, _ := invoke(t, "fmt", "--list", "--json", dir); code != exitOK || !strings.Contains(stdout, `"changed": []`) {
		t.Errorf("fmt finds the clone not canonical: exit %d, %s", code, stdout)
	}
}
