package main

import (
	"github.com/origins-project/origins/internal/api"
	"slices"
	"strings"
	"testing"
)

// changed runs one change against a corpus and fails the test if origins would not
// make it, since every test here needs something of origins' own to take back.
func changed(t *testing.T, args ...string) {
	t.Helper()
	if code, stdout, stderr := invoke(t, args...); code != exitOK {
		t.Fatalf("%s: exit %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, stdout, stderr)
	}
}

// An undo puts the corpus back and the history forward. Both commits are still there
// to be read afterwards, because a conclusion drawn and then withdrawn is two things
// that happened and not none.
func TestUndoTakesBackTheLastChangeAndSaysWhatItWas(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	changed(t, "set", "-C", root, "--source", "firenze-civil-deaths-1901#1103-17",
		"person/rossi-giovanni", "name.surname", "Rossi di Firenze")

	report := answering[api.Reversal](t, "undo", "-C", root)
	if report.Commit == "" {
		t.Fatal("the undo recorded no commit")
	}
	if report.Subject != `Undo "Set name.surname on person/rossi-giovanni"` {
		t.Errorf("the undo reads %q", report.Subject)
	}
	if report.Undid.Op != "name.set" {
		t.Errorf("the undo took back %q", report.Undid.Op)
	}
	if want := []string{"people/rossi-giovanni/README.md"}; !slices.Equal(report.Files, want) {
		t.Errorf("the undo touched %v, want %v", report.Files, want)
	}

	if !strings.Contains(read(t, root, "people/rossi-giovanni/README.md"), "surname: Rossi\n") {
		t.Errorf("the corpus was not put back:\n%s", read(t, root, "people/rossi-giovanni/README.md"))
	}
	if count := git(t, root, "rev-list", "--count", "HEAD"); count != "3" {
		t.Errorf("the corpus has %s commits, want the change and its undo on top of the base", count)
	}
	if status := git(t, root, "status", "--short"); status != "" {
		t.Errorf("the corpus was left holding %q", status)
	}

	// the undo is a change like any other, so it reads back in the same terms
	log := reading(t, "log", "-C", root, "person/rossi-giovanni")
	if len(log.Entries) != 2 {
		t.Fatalf("%d entries, not 2: %v", len(log.Entries), operations(log))
	}
	if log.Entries[0].Op != "history.undo" {
		t.Errorf("the undo is filed under %q", log.Entries[0].Op)
	}
	if log.Entries[0].Corrects != report.Undid.Commit {
		t.Errorf("the undo corrects %q, want %s", log.Entries[0].Corrects, report.Undid.Commit)
	}
}

// What an undo puts back rests on nothing. The citation belonged to the conclusion
// that was withdrawn, and carrying it forward would have blame report the restored
// value as resting on the very reading that had just been abandoned.
func TestAFactPutBackByAnUndoRestsOnNothing(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	changed(t, "set", "-C", root, "--source", "firenze-civil-deaths-1901#1103-17",
		"person/rossi-giovanni", "name.surname", "Rossi di Firenze")
	changed(t, "undo", "-C", root)

	report := answering[api.Provenance](t, "blame", "-C", root, "person/rossi-giovanni", "name.surname")
	found := fields(report)
	surname, named := found["name.surname"]
	if !named {
		t.Fatalf("blame says nothing about the surname: %v", report.Fields)
	}
	if len(surname.Commit.Sources) != 0 {
		t.Errorf("the restored surname is reported as resting on %v", surname.Commit.Sources)
	}
	if surname.Commit.Op != "history.undo" {
		t.Errorf("the surname is the work of %q", surname.Commit.Op)
	}
	if surname.Commit.Corrects == "" {
		t.Error("the undo does not say which commit it took back")
	}
}

// A rename carries a file rather than writing a new one and deleting the old, and
// taking one back has to carry it home the same way. A file left behind under the old
// name would be a person in the corpus twice.
func TestUndoingARenameCarriesTheFileBack(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	changed(t, "rename", "-C", root, "person/rossi-giovanni", "rossi-giovanni-1834")
	changed(t, "undo", "-C", root)

	if read(t, root, "people/rossi-giovanni/README.md") == "" {
		t.Error("the file did not come back under the name it had")
	}
	if left := git(t, root, "status", "--short"); left != "" {
		t.Errorf("the corpus was left holding %q", left)
	}
	if listed := git(t, root, "ls-files", "people"); listed != "people/rossi-giovanni/README.md" {
		t.Errorf("git has these files: %q", listed)
	}
}

// Origins owns a corpus's history. A commit it did not write sitting on top of the
// one it did is history that moved underneath it, and reverting beneath that would be
// building on ground origins has no description of.
func TestUndoWillNotReachUnderACommitOriginsDidNotWrite(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	changed(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze")
	hand(t, root, "places/firenze/README.md", "type: city", "type: comune", "-m", "Firenze is a comune")

	code, _, stderr := invoke(t, "undo", "-C", root)
	if code != exitMisuse {
		t.Fatalf("exit %d, want a refusal", code)
	}
	if !strings.Contains(stderr, "Firenze is a comune") {
		t.Errorf("the refusal does not say what is in the way:\n%s", stderr)
	}
	if count := git(t, root, "rev-list", "--count", "HEAD"); count != "3" {
		t.Errorf("the corpus has %s commits, so something was undone anyway", count)
	}
}

// A history origins wrote no part of has nothing of origins' own to take back, which
// is the corpus somebody started by hand and the repository origins has only just
// been pointed at.
func TestUndoNeedsSomethingOriginsWroteToTakeBack(t *testing.T) {
	t.Parallel()
	root := neverCommitted(t, theRossis())

	code, _, stderr := invoke(t, "undo", "-C", root)
	if code != exitMisuse {
		t.Fatalf("exit %d, want a refusal", code)
	}
	if !strings.Contains(stderr, "recorded nothing") {
		t.Errorf("the refusal reads:\n%s", stderr)
	}
}

// Being told what would be taken back is the point of asking before doing it, so the
// answer has to name the commit and leave the corpus exactly as it was.
func TestADryRunSaysWhatWouldBeTakenBackAndTakesNothing(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	changed(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze")
	was := read(t, root, "people/rossi-giovanni/README.md")

	report := answering[api.Reversal](t, "undo", "-C", root, "--dry-run")
	if report.Commit != "" {
		t.Errorf("a dry run recorded %s", report.Commit)
	}
	if report.Undid.Op != "name.set" {
		t.Errorf("a dry run would take back %q", report.Undid.Op)
	}
	if read(t, root, "people/rossi-giovanni/README.md") != was {
		t.Error("a dry run changed the corpus")
	}
	if count := git(t, root, "rev-list", "--count", "HEAD"); count != "2" {
		t.Errorf("the corpus has %s commits after a dry run", count)
	}
}

// A snapshot names the corpus as it stands, and asking later which states have been
// named is the other half of taking one: what was sent to a relative is no use if
// nothing says what was in it.
func TestASnapshotNamesAStateAndCanBeListedAgain(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	changed(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze")

	taken := answering[api.Snapshots](t, "snapshot", "-C", root, "-m", "sent before the Arezzo trip", "for-zia-maria")
	if len(taken.Snapshots) != 1 {
		t.Fatalf("%d snapshots came back from taking one", len(taken.Snapshots))
	}
	if taken.Snapshots[0].Commit != git(t, root, "rev-parse", "HEAD") {
		t.Errorf("the snapshot names %s, not the commit at the head", taken.Snapshots[0].Commit)
	}

	listed := answering[api.Snapshots](t, "snapshot", "-C", root)
	if len(listed.Snapshots) != 1 {
		t.Fatalf("%d snapshots came back", len(listed.Snapshots))
	}
	if listed.Snapshots[0].Name != "for-zia-maria" {
		t.Errorf("the snapshot is called %q", listed.Snapshots[0].Name)
	}
	if listed.Snapshots[0].Message != "sent before the Arezzo trip" {
		t.Errorf("the snapshot says %q", listed.Snapshots[0].Message)
	}

	// a tag is a tag, so git sees the same thing origins does
	if tags := git(t, root, "tag", "--list"); tags != "for-zia-maria" {
		t.Errorf("git lists the tags as %q", tags)
	}
}

// The name is what says which state was shared, so two states cannot have the same
// one. Git refuses it and says so, and origins has nothing to add.
func TestASnapshotNameAlreadyTakenIsRefused(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	changed(t, "snapshot", "-C", root, "for-zia-maria")

	changed(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze")
	code, _, stderr := invoke(t, "snapshot", "-C", root, "for-zia-maria")
	if code != exitMisuse {
		t.Fatalf("exit %d, want a refusal", code)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("the refusal reads:\n%s", stderr)
	}
}

// A message says what a state is being named for, so it has no meaning without the
// name. Listing with one would quietly do nothing with it.
func TestSayingWhatASnapshotIsForNeedsASnapshotToSayItOf(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())

	code, _, stderr := invoke(t, "snapshot", "-C", root, "-m", "sent before the Arezzo trip")
	if code != exitMisuse {
		t.Fatalf("exit %d, want a refusal", code)
	}
	if !strings.Contains(stderr, "needs a name") {
		t.Errorf("the refusal reads:\n%s", stderr)
	}
}

// A corpus nobody has named a state of has none, and saying so is not a failure.
func TestACorpusWithNoSnapshotsListsNone(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())

	if listed := answering[api.Snapshots](t, "snapshot", "-C", root); len(listed.Snapshots) != 0 {
		t.Errorf("%d snapshots came back", len(listed.Snapshots))
	}
}
