package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// twoClones is the fixture pushed to a bare repository, and a collaborator's clone of
// that, each with an origins of its own.
func twoClones(t *testing.T) (root, other string) {
	t.Helper()
	root = gitFixture(t)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	changed(t, "push", "-C", root)
	return root, collaborator(t, bare, "paolo")
}

func pulling(t *testing.T, root string) api.Taken {
	t.Helper()
	var report api.Taken
	code, stdout, stderr := invoke(t, "pull", "--json", "-C", root)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	return report
}

// settledAfter holds that every command works on a corpus after a pull without
// recover: the history is origins' own throughout and the marker stands on HEAD.
func settledAfter(t *testing.T, root string) {
	t.Helper()
	if marker := git(t, root, "rev-parse", "refs/origins/head"); marker != git(t, root, "rev-parse", "HEAD") {
		t.Errorf("the marker is at %s, HEAD at %s", marker, git(t, root, "rev-parse", "HEAD"))
	}
	if code, stdout, stderr := invoke(t, "recover", "-C", root); code != exitOK || !strings.Contains(stdout, "nothing") {
		t.Errorf("recover after the pull: exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	changed(t, "log", "-C", root, "-n", "1")
	changed(t, "add", "-C", root, "person/verdier-anna", "tags", "pulled")
}

func TestPullFastForwardsOverWhatAnotherCloneRecorded(t *testing.T) {
	t.Parallel()
	root, other := twoClones(t)
	changed(t, "set", "-C", other, "person/verdier-anna", "body", "Anna, as her nephew remembered her.")
	changed(t, "push", "-C", other)

	report := pulling(t, root)
	if report.Took != "fast-forward" || report.Commits != 1 || report.Upstream != "origin/main" {
		t.Errorf("the pull reports %+v", report)
	}
	if head := git(t, root, "rev-parse", "HEAD"); head != git(t, other, "rev-parse", "HEAD") || report.Commit != head {
		t.Errorf("HEAD is %s after the pull", head)
	}
	if !strings.Contains(read(t, root, "people/verdier-anna/README.md"), "her nephew") {
		t.Error("the change did not arrive")
	}
	settledAfter(t, root)

	if again := pulling(t, root); again.Took != "nothing" {
		t.Errorf("a second pull reports %+v", again)
	}
}

// Two clones changing different records are joined in a merge commit of origins' own,
// which log reads like any other, and which the other clone then fast-forwards to.
func TestPullRecordsAMergeOfTwoClonesThatEachWentOn(t *testing.T) {
	t.Parallel()
	root, other := twoClones(t)
	changed(t, "set", "-C", other, "person/verdier-anna", "body", "Anna, as her nephew remembered her.")
	changed(t, "push", "-C", other)
	changed(t, "set", "-C", root, "place/kumasi", "body", "Where Kwame was born in 1950.")

	report := pulling(t, root)
	if report.Took != "merge" || report.Commits != 1 {
		t.Errorf("the pull reports %+v", report)
	}
	if parents := strings.Fields(git(t, root, "log", "-1", "--format=%P")); len(parents) != 2 {
		t.Errorf("the merge has parents %v", parents)
	}
	last := reading(t, "log", "-C", root, "-n", "1").Entries[0]
	if last.Op != "remote.pull" || last.Subject != "Pull origin/main" ||
		strings.Join(last.Subjects, " ") != "person/verdier-anna" {
		t.Errorf("the merge reads back as %+v", last)
	}
	if !strings.Contains(read(t, root, "people/verdier-anna/README.md"), "her nephew") ||
		!strings.Contains(read(t, root, "places/kumasi/README.md"), "in 1950") {
		t.Error("the merge lost one side")
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("the merge left\n%s", status)
	}
	if ops := operations(reading(t, "log", "-C", root, "person/verdier-anna")); ops[0] != "remote.pull" || ops[1] != "body.set" {
		t.Errorf("the record's log after the merge is %q", ops)
	}
	if code, _, stderr := invoke(t, "undo", "-C", root); code != exitMisuse || !strings.Contains(stderr, "pull") {
		t.Errorf("undoing the pull: exit %d, %s", code, stderr)
	}
	settledAfter(t, root)

	changed(t, "push", "-C", root)
	if back := pulling(t, other); back.Took != "fast-forward" {
		t.Errorf("the other clone's pull reports %+v", back)
	}
	settledAfter(t, other)
}

func TestPullIsRefusedOverLinesBothClonesChanged(t *testing.T) {
	t.Parallel()
	root, other := twoClones(t)
	changed(t, "set", "-C", other, "person/verdier-anna", "body", "Anna, as her nephew remembered her.")
	changed(t, "push", "-C", other)
	changed(t, "set", "-C", root, "person/verdier-anna", "body", "Anna, as her niece remembered her.")
	head := git(t, root, "rev-parse", "HEAD")

	code, _, stderr := invoke(t, "pull", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "people/verdier-anna/README.md") {
		t.Errorf("exit %d: %s", code, stderr)
	}
	unmoved(t, root, head)
	if !strings.Contains(read(t, root, "people/verdier-anna/README.md"), "her niece") {
		t.Error("the corpus is not as it was")
	}
}

// Two changes each sound on their own: one clone takes out a place nothing names, the
// other has somebody born there.
func TestPullIsRefusedForAContradictionTheMergeIntroduces(t *testing.T) {
	t.Parallel()
	files := theRossis()
	files["places/prato/README.md"] = "---\nname: Prato\ntype: city\n---\n"
	root := inARepo(t, files)
	bare := bareFrom(t, root)
	changed(t, "remote", "add", "-C", root, "origin", bare)
	changed(t, "push", "-C", root)
	other := collaborator(t, bare, "paolo")
	changed(t, "delete", "-C", other, "place/prato")
	changed(t, "push", "-C", other)
	changed(t, "set", "-C", root, "person/rossi-giovanni", "events[0].place", "prato")
	head := git(t, root, "rev-parse", "HEAD")

	code, _, stderr := invoke(t, "pull", "-C", root)
	if code != exitFindings || !strings.Contains(stderr, "prato") || !strings.Contains(stderr, "nothing has been taken in") {
		t.Errorf("exit %d: %s", code, stderr)
	}
	unmoved(t, root, head)
	if _, err := os.Stat(filepath.Join(root, "places", "prato", "README.md")); err != nil {
		t.Error("the place the merge deleted is still gone")
	}
}

// A commit git made alone is refused before anything is taken in, and is taken in by
// hand with git merge and recover --adopt, the merge git made among what is adopted.
func TestPullIsRefusedOverACommitOriginsDidNotWrite(t *testing.T) {
	t.Parallel()
	root, other := twoClones(t)
	edited(t, other, "people/verdier-anna/README.md",
		strings.Replace(read(t, other, "people/verdier-anna/README.md"), "---\n", "---\ntags:\n  - by-hand\n", 1))
	git(t, other, "commit", "-q", "-am", "Tag Anna")
	git(t, other, "push", "-q")
	changed(t, "set", "-C", root, "place/kumasi", "body", "Where Kwame was born in 1950.")
	head := git(t, root, "rev-parse", "HEAD")

	code, _, stderr := invoke(t, "pull", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "Tag Anna") || !strings.Contains(stderr, "recover --adopt") {
		t.Errorf("exit %d: %s", code, stderr)
	}
	unmoved(t, root, head)

	git(t, root, "merge", "-q", "--no-edit", "origin/main")
	if code, _, _ := invoke(t, "log", "-C", root); code != exitMisuse {
		t.Errorf("the log over git's merge exited %d", code)
	}
	if code, stdout, stderr := invoke(t, "recover", "--adopt", "-C", root); code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	settledAfter(t, root)
	changed(t, "push", "-C", root)
}

func TestPullIsRefusedOverAnUnrecordedEditInTheCorpus(t *testing.T) {
	t.Parallel()
	root, _ := twoClones(t)
	edited(t, root, "people/verdier-anna/README.md", read(t, root, "people/verdier-anna/README.md")+"Edited.\n")
	code, _, stderr := invoke(t, "pull", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "people/verdier-anna/README.md") ||
		!strings.Contains(stderr, "origins adopt") {
		t.Errorf("exit %d: %s", code, stderr)
	}
}

// A corpus kept inside a larger repository is pulled with the rest of it, so a file
// staged beside the corpus would be swept into the merge commit.
func TestPullIsRefusedOverAFileStagedOutsideTheCorpus(t *testing.T) {
	t.Parallel()
	files := map[string]string{"notes.txt": "what to ask zia Maria\n"}
	for path, content := range theRossis() {
		files["tree/"+path] = content
	}
	outer := inARepo(t, files)
	root := filepath.Join(outer, "tree")
	edited(t, outer, "notes.txt", "what to ask zia Maria, and when\n")
	git(t, outer, "add", "notes.txt")

	code, _, stderr := invoke(t, "pull", "-C", root)
	if code != exitMisuse || !strings.Contains(stderr, "notes.txt") || !strings.Contains(stderr, "outside the corpus") {
		t.Errorf("exit %d: %s", code, stderr)
	}
}

// unmoved holds a repository to where it stood: the same HEAD, nothing staged or
// modified, and no merge part way through.
func unmoved(t *testing.T, root, head string) {
	t.Helper()
	if now := git(t, root, "rev-parse", "HEAD"); now != head {
		t.Errorf("HEAD moved from %s to %s", head, now)
	}
	if status := git(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("the refusal left\n%s", status)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(git(t, root, "rev-parse", "--git-path", "MERGE_HEAD")))); err == nil {
		t.Error("a merge is left part way through")
	}
}
