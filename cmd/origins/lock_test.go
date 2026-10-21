package main

import (
	"bytes"
	"errors"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// occupied takes the lock on a corpus as another origins would, and hands back what
// gives it up.
func occupied(t *testing.T, root, command string) func() error {
	t.Helper()
	repo, err := history.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	release, err := repo.Lock(command)
	if err != nil {
		t.Fatal(err)
	}
	return release
}

// Every command that writes the corpus or its history is refused while another origins
// is writing it, whichever of them it is, and is refused before it has written
// anything: not a file, not a commit, not a tag, and not the lock it was refused over.
func TestEveryWriterIsRefusedWhileAnotherIsWriting(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	export := exportFile(t)
	tree := grampsFile(t, "rossi.gramps", []byte(grampsTree), true)
	release := occupied(t, root, "origins import")
	defer release()
	lock := filepath.Join(root, filepath.FromSlash(git(t, root, "rev-parse", "--git-path", "origins.lock")))
	held, err := os.ReadFile(lock)
	if err != nil {
		t.Fatal(err)
	}

	state := func() string {
		return strings.Join([]string{
			git(t, root, "rev-parse", "HEAD"),
			git(t, root, "for-each-ref"),
			git(t, root, "status", "--porcelain", "--untracked-files=all", "--", ".", ":!imported*"),
		}, "\n")
	}
	before := state()

	for _, run := range [][]string{
		{"set", "-C", root, "person/verdier-anna", "name.surname", "Verdier Galvão"},
		{"unset", "-C", root, "person/verdier-anna", "body"},
		{"add", "-C", root, "person/verdier-anna", "also-known-as", "Annina"},
		{"remove", "-C", root, "person/verdier-anna", "also-known-as", "Annina"},
		{"create", "-C", root, "place/vicchio", "name=Vicchio"},
		{"delete", "-C", root, "place/kumasi"},
		{"rename", "-C", root, "person/verdier-anna", "verdier-anna-1860"},
		{"merge", "-C", root, "person/ershov-ivan-1885", "ershov-ivan"},
		{"split", "-C", root, "person/verdier-anna", "verdier-anna", "verdier-anna-2"},
		{"undo", "-C", root},
		{"adopt", "-C", root},
		{"recover", "--adopt", "-C", root},
		{"hypothesis", "accept", "-C", root, "unrecorded-child-1845"},
		{"hypothesis", "reject", "-C", root, "unrecorded-child-1845"},
		{"hypothesis", "propose", "-C", root, "anna-born-in-prato"},
		{"snapshot", "-C", root, "sent-to-zia-maria"},
		{"pull", "-C", root},
		{"init", filepath.Join(root, "started")},
		{"import", "gedcom", "--into", filepath.Join(root, "imported"), export},
		{"import", "gramps", "--into", filepath.Join(root, "imported-gramps"), tree},
	} {
		code, stdout, stderr := invoke(t, run...)
		if code != exitMisuse || !strings.Contains(stderr, "origins.lock") {
			t.Errorf("origins %s: exit %d\nstdout: %s\nstderr: %s", run[0], code, stdout, stderr)
		}
		if now := state(); now != before {
			t.Errorf("origins %s wrote while another origins held the lock:\n%s\nnow\n%s",
				run[0], before, now)
			before = now
		}
		if still, err := os.ReadFile(lock); err != nil || string(still) != string(held) {
			t.Fatalf("origins %s took away a lock it did not hold: %q, %v", run[0], still, err)
		}
	}

	// the manifest init wrote before it could open the repository goes back off the disk
	if _, err := os.Stat(filepath.Join(root, "started", "origins.yaml")); !os.IsNotExist(err) {
		t.Errorf("init left a manifest behind with no history: %v", err)
	}
}

// A command gives the lock up on every way out of it, the ones that refuse included,
// since a lock left behind is every command after it refused until somebody removes it
// by hand.
func TestTheLockIsGivenUpOnEveryWayOut(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)

	for _, want := range []struct {
		code int
		run  []string
	}{
		{exitOK, []string{"set", "-C", root, "person/verdier-anna", "name.surname", "Verdier Galvão"}},
		{exitOK, []string{"set", "-C", root, "--dry-run", "person/verdier-anna", "name.surname", "Verdier"}},
		{exitFindings, []string{"delete", "-C", root, "place/firenze"}},
		{exitMisuse, []string{"set", "-C", root, "person/verdier-anna", "no-such-field", "x"}},
		{exitOK, []string{"undo", "-C", root}},
		{exitOK, []string{"adopt", "-C", root}},
		{exitMisuse, []string{"recover", "--adopt", "-C", root}},
		{exitOK, []string{"hypothesis", "accept", "-C", root, "--dry-run", "unrecorded-child-1845"}},
		{exitMisuse, []string{"hypothesis", "accept", "-C", root, "nobody-proposed-this"}},
		{exitMisuse, []string{"hypothesis", "propose", "-C", root, "anna-born-in-prato"}},
		{exitOK, []string{"snapshot", "-C", root, "sent-to-zia-maria"}},
		{exitMisuse, []string{"pull", "-C", root}},
	} {
		code, stdout, stderr := invoke(t, want.run...)
		if code != want.code {
			t.Errorf("origins %s: exit %d, want %d\nstdout: %s\nstderr: %s",
				want.run[0], code, want.code, stdout, stderr)
		}
		repo, err := history.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		release, err := repo.Lock("origins test")
		if err != nil {
			t.Fatalf("origins %s kept the lock: %v", want.run[0], err)
		}
		if err := release(); err != nil {
			t.Fatal(err)
		}
	}
}

// aLockLeftBehind is what giving up the lock says when the file could not be removed.
func aLockLeftBehind() *history.Unreleased {
	return &history.Unreleased{Path: "corpus/.git/origins.lock", Err: errors.New("it is in use")}
}

func herDeathOn(date string) model.Edit {
	return model.Set{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "lang-anna-maria"},
		Path:   "events[2].date",
		Value:  store.Typed(date),
	}
}

// saidLast asserts that the lock left behind is the last thing said, as it was when the
// command gave the lock up on its way out.
func saidLast(t *testing.T, stderr string) {
	t.Helper()
	lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
	if last := lines[len(lines)-1]; last != "origins: "+aLockLeftBehind().Error() {
		t.Errorf("the lock left behind is not the last thing said:\n%s", stderr)
	}
}

// A lock left behind is a warning about the next origins and not a failure of this
// one. A dry run and a change that comes to nothing have no commit to show the work was
// done, and are reported all the same.
func TestALockLeftBehindDoesNotUndoTheReport(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	for _, tc := range []struct {
		name   string
		dryRun bool
		date   string
		want   string
	}{
		{"a dry run", true, "15 JUL 1878", "would write people/lang-anna-maria/README.md"},
		{"a change to nothing", false, "14 JUL 1878", "the corpus already says that"},
	} {
		c := &change{name: "set", root: root, dryRun: tc.dryRun}
		report, err := api.Carry(c.asking(), herDeathOn(tc.date))
		if err != nil || report.Commit != "" {
			t.Fatalf("%s: came back %+v, %v", tc.name, report, err)
		}
		var stdout, stderr bytes.Buffer
		code := c.reported(report, aLockLeftBehind(), &stdout, &stderr)
		if code != exitOK || !strings.Contains(stdout.String(), tc.want) {
			t.Errorf("%s: exit %d\nstdout: %s\nstderr: %s", tc.name, code, &stdout, &stderr)
		}
		saidLast(t, stderr.String())
	}
}

// Settling a hypothesis and taking a change back are reported through a lock left
// behind the same way.
func TestALockLeftBehindDoesNotUndoARehearsal(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)

	c := &change{name: "hypothesis accept", root: root, dryRun: true}
	decision, err := api.Decide(c.asking(), "unrecorded-child-1845", model.Accepted)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := settled(c, decision, aLockLeftBehind(), &stdout, &stderr)
	if code != exitOK || !strings.Contains(stdout.String(), "Accept unrecorded-child-1845") {
		t.Errorf("accept: exit %d\nstdout: %s\nstderr: %s", code, &stdout, &stderr)
	}
	saidLast(t, stderr.String())

	taken, err := api.Undo(api.Asking{Root: root, Command: "undo", DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = undone(taken, aLockLeftBehind(), true, false, &stdout, &stderr)
	if code != exitOK || !strings.Contains(stdout.String(), "would be taken back") {
		t.Errorf("undo: exit %d\nstdout: %s\nstderr: %s", code, &stdout, &stderr)
	}
	saidLast(t, stderr.String())
}

// A refusal is still a refusal with a lock left behind beside it, and says both.
func TestALockLeftBehindIsSaidBesideARefusal(t *testing.T) {
	t.Parallel()
	c := &change{name: "set", root: "corpus"}
	unreadable := &api.Unreadable{
		Problems: []string{"places/prato/README.md: not a record"},
		Refusal:  "origins will not change a corpus it could not read whole",
	}
	var stdout, stderr bytes.Buffer
	code := c.reported(nil, errors.Join(unreadable, aLockLeftBehind()), &stdout, &stderr)
	want := "places/prato/README.md: not a record\n" +
		"origins: origins will not change a corpus it could not read whole\n" +
		"origins: " + aLockLeftBehind().Error() + "\n"
	if code != exitMisuse || stderr.String() != want {
		t.Errorf("exit %d, stderr:\n%s\nwant:\n%s", code, &stderr, want)
	}
}

// A corpus a hypothesis could not be taken back out of is said after what it was being
// taken back from, each in its own words, and the lock left behind after both.
func TestACorpusLeftInNeitherStateIsSaidAfterWhatLeftItThere(t *testing.T) {
	t.Parallel()
	c := &change{name: "hypothesis accept", root: "corpus"}
	contradiction := &ops.Contradiction{Findings: []check.Finding{{
		Rule: "death-before-birth", Entity: "person/rossi-antonio", Message: "died before he was born",
	}}}
	stuck := &ops.Unrestored{Err: errors.New("git would not restore the index")}
	err := errors.Join(errors.Join(contradiction, stuck), aLockLeftBehind())

	var stdout, stderr bytes.Buffer
	code := settled(c, nil, err, &stdout, &stderr)
	want := contradiction.Error() + "\n" +
		"origins: " + stuck.Error() + "\n" +
		"origins: " + aLockLeftBehind().Error() + "\n"
	if code != exitMisuse || stderr.String() != want {
		t.Errorf("exit %d, stderr:\n%s\nwant:\n%s", code, &stderr, want)
	}
}
