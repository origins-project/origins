package history

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func surveying(t *testing.T, repo *Repo) Standing {
	t.Helper()
	standing, err := repo.Survey()
	if err != nil {
		t.Fatal(err)
	}
	return standing
}

// A history origins wrote the whole of is the state every other command assumes, and
// the survey has to be able to say so plainly. Everything else in this file is a way
// for it to stop being true.
func TestAHistoryOriginsWroteIsSettled(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	last := record(t, repo, change("He died", "events.add", person("rossi-giovanni")))

	standing := surveying(t, repo)
	switch {
	case !standing.Settled():
		t.Errorf("a history origins wrote reads as unsettled: %s", standing.Describe())
	case standing.Anchor == nil || standing.Anchor.Sha != last:
		t.Errorf("the anchor is %+v, want the last commit origins made", standing.Anchor)
	case standing.Ground != Kept:
		t.Errorf("the ground is %s, want kept", standing.Ground)
	}
	if standing.Describe() != "" {
		t.Errorf("a settled history has something to complain about: %q", standing.Describe())
	}
}

// A repository nobody has committed to is settled: there is no history to disagree
// with, which is not the same as a history that disagrees.
func TestAHistoryWithNoCommitsInItIsSettled(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	standing := surveying(t, repo)
	if !standing.Settled() {
		t.Errorf("an empty repository reads as unsettled: %s", standing.Describe())
	}
	if standing.Anchor != nil {
		t.Errorf("an empty repository has an anchor: %+v", standing.Anchor)
	}
}

// The case the whole file is for. The commit is named, and so is the person who made
// it, because a refusal that says only that something is wrong is one nobody can act
// on.
func TestACommitOriginsDidNotWriteIsNotSettled(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	anchor := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	byHand(t, repo, "corpus.md", "Fix the birth year")

	standing := surveying(t, repo)
	switch {
	case standing.Settled():
		t.Fatal("a commit origins did not write reads as settled")
	case len(standing.Foreign) != 1:
		t.Fatalf("%d commits read as foreign, not 1", len(standing.Foreign))
	case standing.Foreign[0].Subject != "Fix the birth year":
		t.Errorf("the foreign commit is %q", standing.Foreign[0].Subject)
	case standing.Anchor == nil || standing.Anchor.Sha != anchor:
		t.Errorf("the anchor is %+v, want the commit under the foreign one", standing.Anchor)
	}
	for _, want := range []string{"Fix the birth year", "Rosa Bianchi", "sits on top"} {
		if !strings.Contains(standing.Describe(), want) {
			t.Errorf("the description does not mention %q:\n%s", want, standing.Describe())
		}
	}
}

// Owned is the survey put as a refusal, and what it refuses with is the standing, so
// that whoever is refused can be told what the refusal was about and what to do.
func TestOwnedRefusesAHistoryOriginsCannotAccountFor(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	if _, err := Owned(repo.dir); err != nil {
		t.Fatalf("a history origins wrote was refused: %v", err)
	}

	byHand(t, repo, "corpus.md", "Fix the birth year")
	_, err := Owned(repo.dir)
	var unsettled *Unsettled
	if !errors.As(err, &unsettled) {
		t.Fatalf("a commit origins did not write came back %v", err)
	}
	if len(unsettled.Standing.Foreign) != 1 || err.Error() != unsettled.Standing.Describe() {
		t.Errorf("the refusal carries %+v and says %q", unsettled.Standing, err)
	}
}

// A corpus somebody has been keeping in git without origins is a history origins
// wrote no part of. There is no anchor to be found, and every commit in it is
// somebody else's work — which is a thing to be resolved and not an error.
func TestAHistoryOriginsWroteNoPartOfHasNoAnchor(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	byHand(t, repo, "corpus.md", "The corpus as it stood")

	standing := surveying(t, repo)
	switch {
	case standing.Settled():
		t.Error("a history origins wrote no part of reads as settled")
	case standing.Anchor != nil:
		t.Errorf("there is an anchor in a history origins never wrote: %+v", standing.Anchor)
	case len(standing.Foreign) != 1:
		t.Errorf("%d commits read as foreign, not 1", len(standing.Foreign))
	}
}

// The gap the trailers cannot cover, and the whole reason the marker ref is kept.
//
// An amend carries the message across, so every commit still says origins wrote it and
// the trailer test is perfectly happy. What has changed is every sha: the commit a
// hypothesis was pinned to is not in the history any more, and only the ref noticed.
func TestARewriteIsNoticedThoughEveryCommitStillReadsAsOrigins(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	was := record(t, repo, change("He died", "events.add", person("rossi-giovanni")))
	if _, err := repo.git("commit", "-q", "--amend", "-m", "He died in Firenze\n\n"+
		"Origins-Tool: origins 0.2.0\nOrigins-Op: events.add"); err != nil {
		t.Fatal(err)
	}

	standing := surveying(t, repo)
	switch {
	case len(standing.Foreign) != 0:
		t.Errorf("%d commits read as foreign after an amend, want none", len(standing.Foreign))
	case standing.Ground != Lost:
		t.Errorf("the ground is %s after an amend, want lost", standing.Ground)
	case standing.Settled():
		t.Error("a history that was rewritten underneath origins reads as settled")
	case standing.Marker != was:
		t.Errorf("the marker says %s, want the commit that was amended away", short(standing.Marker))
	}
	if !strings.Contains(standing.Describe(), "rewritten") {
		t.Errorf("the description does not say the history was rewritten:\n%s", standing.Describe())
	}
}

// Standing on an older commit is not the same as having lost the newer ones, and
// saying so is what keeps a bisect from reading as a catastrophe.
func TestAnOlderCommitCheckedOutIsBehindAndNotLost(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	first := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("He died", "events.add", person("rossi-giovanni")))
	if _, err := repo.git("checkout", "--detach", "-q", first); err != nil {
		t.Fatal(err)
	}

	standing := surveying(t, repo)
	if standing.Ground != Behind {
		t.Errorf("the ground is %s at an older commit, want behind", standing.Ground)
	}
	if !strings.Contains(standing.Describe(), "older state") {
		t.Errorf("the description does not say this is an older state:\n%s", standing.Describe())
	}
}

// A dirty working tree is reported and stops nothing. Editing a file is the escape
// hatch the format exists to leave open, and a commit names the paths it records, so
// there is nothing for an unrecorded edit to be swept into.
func TestAChangeNobodyRecordedIsReportedAndIsNoDivergence(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	write(t, repo, "corpus.md", "Something I typed in myself\n")
	write(t, repo, "people/rossi-annunziata/README.md", "Somebody new\n")

	standing := surveying(t, repo)
	if !standing.Settled() {
		t.Errorf("an unrecorded edit reads as a divergence: %s", standing.Describe())
	}
	want := []string{"corpus.md", "people/rossi-annunziata/README.md"}
	if !slices.Equal(standing.Dirty, want) {
		t.Errorf("the unrecorded files are %q, want %q", standing.Dirty, want)
	}
}

// Git compares the stat information before it compares the bytes, so a file whose
// timestamp has moved reads as changed until the index is refreshed. On a platform
// that rewrites line endings on checkout that is most of a corpus, and a tool that
// announced a hundred hand edits nobody made would be one nobody believed.
func TestAFileOnlyTouchedHasNotBeenChanged(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(filepath.Join(repo.dir, "corpus.md"), later, later); err != nil {
		t.Fatal(err)
	}
	if dirty := surveying(t, repo).Dirty; len(dirty) != 0 {
		t.Errorf("a file nobody edited reads as changed: %q", dirty)
	}
}

// Paths come back as the bytes they are. Git escapes anything outside ASCII by
// default, and an escaped path names no file: origins would print it, and hand it
// back to git as a pathspec matching nothing. A slug has no accents in it, but a scan
// filed beside an entity keeps whatever name it arrived with.
func TestAFileWithAnAccentInItsNameIsNamedAsItIs(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	const scan = "people/atto di nascita è.txt"
	write(t, repo, scan, "a scan of the register\n")
	if dirty := surveying(t, repo).Dirty; len(dirty) != 1 || dirty[0] != scan {
		t.Errorf("the file is named %q, want %q", dirty, scan)
	}
	if _, err := repo.Commit(change("A scan", "person.create", person("rossi-giovanni")),
		[]string{scan}); err != nil {
		t.Fatalf("a path origins named could not be committed: %v", err)
	}
	touched, err := repo.Touched("HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if len(touched) != 1 || touched[0] != scan {
		t.Errorf("the commit touched %q, want %q", touched, scan)
	}
}

// Whether a commit has left this machine is the one part of the recovery question
// that is a fact, and it decides whether history may be rewritten.
func TestACommitIsSharedOnceARemoteHasSeenIt(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	shared := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	// a remote-tracking ref is what a clone remembers of a push, and writing one is
	// the whole of what a push leaves behind here
	if _, err := repo.git("update-ref", "refs/remotes/origin/master", shared); err != nil {
		t.Fatal(err)
	}
	mine := record(t, repo, change("He died", "events.add", person("rossi-giovanni")))

	for sha, want := range map[string]bool{shared: true, mine: false} {
		got, err := repo.Shared(sha)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s reads as shared=%v, want %v", short(sha), got, want)
		}
	}
}
