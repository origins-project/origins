package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// bareFrom is a bare repository cloned from a corpus, beside it in the test's own
// directory and reached by its path, so that nothing a test sends leaves the machine.
func bareFrom(t *testing.T, root string) string {
	t.Helper()
	bare := filepath.Join(filepath.Dir(root), "shared.git")
	git(t, root, "clone", "-q", "--bare", ".", bare)
	return bare
}

// collaborator is another clone of a shared repository, with an identity of its own.
func collaborator(t *testing.T, bare, name string) string {
	t.Helper()
	dir := filepath.Join(filepath.Dir(bare), name)
	git(t, filepath.Dir(bare), "clone", "-q", bare, dir)
	git(t, dir, "config", "user.name", "Paolo Verdi")
	git(t, dir, "config", "user.email", "paolo@example.invalid")
	return dir
}

// remoting runs one of the remote commands and reads back what it said.
func remoting(t *testing.T, args ...string) api.Remotes {
	t.Helper()
	var report api.Remotes
	code, stdout, stderr := invoke(t, append([]string{"remote", args[0], "--json"}, args[1:]...)...)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	return report
}

func TestRemotesAreAddedListedAndRemoved(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	bare := bareFrom(t, root)

	code, stdout, stderr := invoke(t, "remote", "add", "-C", root, "origin", bare)
	if code != exitOK || !strings.Contains(stdout, "added origin") {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	added := remoting(t, "add", "-C", root, "zia-maria", "https://example.invalid/rossi.git")
	want := []api.Remote{{Name: "origin", URL: bare}, {Name: "zia-maria", URL: "https://example.invalid/rossi.git"}}
	if !slices.Equal(added.Remotes, want) {
		t.Errorf("after adding two the remotes are %v", added.Remotes)
	}
	code, stdout, _ = invoke(t, "remote", "list", "-C", root)
	if code != exitOK || squeezed(stdout) != squeezed("origin "+bare+" zia-maria https://example.invalid/rossi.git") {
		t.Errorf("the list reads %q", stdout)
	}

	if code, _, stderr := invoke(t, "remote", "add", "-C", root, "origin", bare); code != exitMisuse ||
		!strings.Contains(stderr, "origin") {
		t.Errorf("a second origin: exit %d, %s", code, stderr)
	}
	left := remoting(t, "remove", "-C", root, "zia-maria")
	if !slices.Equal(left.Remotes, want[:1]) {
		t.Errorf("after removing one the remotes are %v", left.Remotes)
	}
	if code, _, _ := invoke(t, "remote", "rename", "-C", root, "origin", "upstream"); code != exitMisuse {
		t.Errorf("remote rename exited %d", code)
	}
	if code, _, _ := invoke(t, "remote", "add", "-C", root, "origin"); code != exitMisuse {
		t.Errorf("remote add without a URL exited %d", code)
	}
}

// A fetch reports for the branch, and leaves the corpus and its history as they were.
func TestFetchSaysHowFarTheBranchAndItsUpstreamHaveGone(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)

	report := answering[api.Standing](t, "fetch", "-C", root)
	if !slices.Equal(report.Fetched, []string{"origin"}) || report.Branch != "main" || report.Upstream != "" {
		t.Errorf("before an upstream is set the fetch reports %+v", report)
	}
	git(t, root, "branch", "--set-upstream-to", "origin/main")

	other := collaborator(t, bare, "paolo")
	git(t, other, "commit", "-q", "--allow-empty", "-m", "one")
	git(t, other, "push", "-q")
	changed(t, "set", "-C", root, "person/verdier-anna", "body", "Anna, as her niece remembered her.")
	head := git(t, root, "rev-parse", "HEAD")

	code, stdout, stderr := invoke(t, "fetch", "-C", root, "origin")
	if code != exitOK || strings.TrimSpace(stdout) != "main is 1 commit behind and 1 commit ahead of origin/main" {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if git(t, root, "rev-parse", "HEAD") != head {
		t.Error("the fetch moved HEAD")
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("the fetch left the tree as\n%s", status)
	}
	if code, _, stderr := invoke(t, "log", "-C", root, "-n", "1"); code != exitOK {
		t.Errorf("the log after a fetch: %s", stderr)
	}
}

// What fetch asks of the history is nothing, so a history recover would have to settle
// first is fetched into all the same: fetching is how recover learns what has left.
func TestFetchNeedsNoHistoryOriginsCanAccountFor(t *testing.T) {
	t.Parallel()
	root := byHand(t, theRossis())
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	if code, _, stderr := invoke(t, "fetch", "-C", root); code != exitOK {
		t.Errorf("exit %d: %s", code, stderr)
	}
}

func TestFetchWithNoRemoteSaysHowToNameOne(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, _, stderr := invoke(t, "fetch", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "origins remote add") {
		t.Errorf("exit %d: %s", code, stderr)
	}
}

func TestFetchFromARemoteThatIsNotThereIsRefused(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	missing := filepath.Join(t.TempDir(), "no-such.git")
	changed(t, "remote", "add", "-C", root, "origin", missing)
	code, _, stderr := invoke(t, "fetch", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "did not finish") {
		t.Errorf("exit %d: %s", code, stderr)
	}
	if _, err := os.Stat(missing); err == nil {
		t.Error("the fetch made the remote")
	}
}

// A change goes with its trailers, a snapshot goes with the branch, and the marker of
// what this clone wrote stays here.
func TestPushSendsTheChangesAndTheSnapshotsAndNotTheMarker(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	changed(t, "set", "-C", root, "person/verdier-anna", "body", "Anna, as her niece remembered her.")
	changed(t, "snapshot", "-C", root, "-m", "what was sent to zia Maria", "sent-to-zia-maria")

	report := answering[api.Sent](t, "push", "-C", root)
	head := git(t, root, "rev-parse", "HEAD")
	if report.Remote != "origin" || report.Branch != "main" || report.Commit != head || !report.Upstream {
		t.Errorf("the push reports %+v", report)
	}
	if there := git(t, bare, "rev-parse", "main"); there != head {
		t.Errorf("the remote's main is %s, want %s", there, head)
	}
	if op := strings.TrimSpace(git(t, bare, "log", "-1", "--format=%(trailers:key=Origins-Op,valueonly)")); op != "body.set" {
		t.Errorf("the remote's last commit says it did %q", op)
	}
	if tag := git(t, bare, "for-each-ref", "--format=%(objecttype) %(contents:subject)", "refs/tags/sent-to-zia-maria"); tag != "tag what was sent to zia Maria" {
		t.Errorf("the snapshot is there as %q", tag)
	}
	if marker := git(t, bare, "for-each-ref", "refs/origins"); marker != "" {
		t.Errorf("the marker went too: %s", marker)
	}
	if upstream := git(t, root, "rev-parse", "--abbrev-ref", "main@{upstream}"); upstream != "origin/main" {
		t.Errorf("main follows %s", upstream)
	}

	code, stdout, stderr := invoke(t, "push", "-C", root)
	if code != exitOK || strings.Contains(stdout, "now follows") {
		t.Errorf("a second push: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
}

// A dirty tree is nothing a push sends, so it does not stop one.
func TestPushGoesAheadBesideAnUnrecordedEdit(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	edited(t, root, "notes.txt", "a list of what to ask zia Maria\n")
	changed(t, "push", "-C", root)
}

func TestPushIsRefusedOverACommitOriginsDidNotWrite(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	before := git(t, bare, "rev-parse", "main")
	edited(t, root, "notes.txt", "a list of what to ask zia Maria\n")
	git(t, root, "add", "notes.txt")
	git(t, root, "commit", "-q", "-m", "What to ask")

	code, _, stderr := invoke(t, "push", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "origins recover") {
		t.Errorf("exit %d: %s", code, stderr)
	}
	if after := git(t, bare, "rev-parse", "main"); after != before {
		t.Error("the foreign commit was pushed")
	}
}

func TestPushIsRefusedAsBehindAndPointsAtPull(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	changed(t, "push", "-C", root)

	other := collaborator(t, bare, "paolo")
	changed(t, "set", "-C", other, "person/verdier-anna", "body", "Anna, as her nephew remembered her.")
	changed(t, "push", "-C", other)
	changed(t, "set", "-C", root, "person/verdier-anna", "body", "Anna, as her niece remembered her.")

	code, _, stderr := invoke(t, "push", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "origins pull") {
		t.Errorf("exit %d: %s", code, stderr)
	}
}

func TestPushToOneOfSeveralRemotesHasToBeTold(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	changed(t, "remote", "add", "-C", root, "backup", bare)

	code, _, stderr := invoke(t, "push", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "backup, origin") {
		t.Errorf("exit %d: %s", code, stderr)
	}
	changed(t, "push", "-C", root, "backup")
	changed(t, "push", "-C", root)
}

func TestPushWithNoRemoteSaysHowToNameOne(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, _, stderr := invoke(t, "push", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "origins remote add") {
		t.Errorf("exit %d: %s", code, stderr)
	}
}
