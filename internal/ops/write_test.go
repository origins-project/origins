package ops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

func asking(command string) Asked {
	return Asked{Tool: "origins 0.2.0", Command: command}
}

func herDeath() model.Edit {
	return model.Set{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "lang-anna-maria"},
		Path:   "events[2].date",
		Value:  store.Typed("15 JUL 1878"),
	}
}

// free asserts that nothing holds the lock, by taking it and giving it up again.
func free(t *testing.T, repo *history.Repo) {
	t.Helper()
	release, err := repo.Lock("origins test")
	if err != nil {
		t.Fatalf("the lock was not given up: %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
}

// Carry is the whole of what a command does short of printing, so a change carried is
// one commit saying how it was asked for, and the lock is free again afterwards.
func TestACarriedEditIsOneCommitSayingHowItWasAskedFor(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	before := git(t, corpus.Root, "rev-parse", "HEAD")

	asked := asking("origins set")
	asked.Subject = "Correct the day of Anna Maria's death"
	asked.Sources = []model.CitationRef{{Source: "census-1861-england", Anchor: "f088"}}
	entry, change, sha, err := Carry(corpus.Root, herDeath(), asked)
	if err != nil {
		t.Fatal(err)
	}
	if got := git(t, corpus.Root, "rev-parse", "HEAD^"); got != before || sha == "" {
		t.Errorf("the change is %q on top of %s, want one commit on %s", sha, got, before)
	}
	if entry.Subject != asked.Subject || entry.Tool != asked.Tool {
		t.Errorf("the entry says %q from %q", entry.Subject, entry.Tool)
	}
	if got := trailer(t, corpus.Root, "Origins-Tool"); got != asked.Tool {
		t.Errorf("Origins-Tool is %q", got)
	}
	if got := trailer(t, corpus.Root, "Origins-Source"); got != "census-1861-england#f088" {
		t.Errorf("Origins-Source is %q", got)
	}
	if written := change.Written(); len(written) != 1 || written[0] != "people/lang-anna-maria/README.md" {
		t.Errorf("the change wrote %v", written)
	}
	free(t, repo)
}

// A dry run works out the change and writes none of it, and records nothing.
func TestADryRunCarriesNothing(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	head := git(t, corpus.Root, "rev-parse", "HEAD")

	asked := asking("origins set")
	asked.DryRun = true
	_, change, sha, err := Carry(corpus.Root, herDeath(), asked)
	if err != nil {
		t.Fatal(err)
	}
	if sha != "" || change.Empty() {
		t.Errorf("a dry run came back %q with an empty change %t", sha, change.Empty())
	}
	if got := git(t, corpus.Root, "rev-parse", "HEAD"); got != head {
		t.Error("a dry run recorded something")
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("a dry run wrote:\n%s", left)
	}
	free(t, repo)
}

// Another writer holding the lock is refused before the corpus is read, and the
// refusal leaves the lock as that writer's.
func TestNothingIsCarriedWhileAnotherWriterHoldsTheLock(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	release, err := repo.Lock("origins import")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = release() }()

	_, _, _, err = Carry(corpus.Root, herDeath(), asking("origins set"))
	var locked *history.Locked
	if !errors.As(err, &locked) || locked.Command != "origins import" {
		t.Fatalf("carrying under another writer's lock came back %v", err)
	}
	if _, err := Undo(corpus.Root, asking("origins undo")); !errors.As(err, &locked) {
		t.Errorf("undoing under another writer's lock came back %v", err)
	}
	if _, _, _, err := Decide(corpus.Root, "unrecorded-child-1845", model.Rejected,
		asking("origins hypothesis reject")); !errors.As(err, &locked) {
		t.Errorf("deciding under another writer's lock came back %v", err)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("something was written:\n%s", left)
	}
	if _, err := os.Stat(locked.Path); err != nil {
		t.Errorf("a refused writer took the lock away: %v", err)
	}
}

// A corpus that did not read whole is refused with what it could not read, and the
// lock taken to read it is given up again.
func TestACorpusThatDidNotReadWholeIsNotCarriedInto(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	broken := filepath.Join(corpus.Root, "places", "kumasi", "README.md")
	if err := os.WriteFile(broken, []byte("---\nname: [\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, _, err := Carry(corpus.Root, herDeath(), asking("origins set"))
	var unreadable *Unreadable
	if !errors.As(err, &unreadable) || unreadable.Problems == nil {
		t.Fatalf("carrying into a corpus that did not read came back %v", err)
	}
	free(t, repo)
}

// Undo takes back what Carry did, as a commit of its own that says which one.
func TestUndoTakesBackTheLastChange(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	where := "people/lang-anna-maria/README.md"
	was := read(t, corpus.Root, where)
	_, _, sha, err := Carry(corpus.Root, herDeath(), asking("origins set"))
	if err != nil {
		t.Fatal(err)
	}

	taken, err := Undo(corpus.Root, asking("origins undo"))
	if err != nil {
		t.Fatal(err)
	}
	if taken.Undid.Sha != sha || taken.Sha == "" || taken.Entry.Tool != "origins 0.2.0" {
		t.Errorf("the undo took back %s as %q from %q, want %s", taken.Undid.Sha, taken.Sha,
			taken.Entry.Tool, sha)
	}
	if got := trailer(t, corpus.Root, "Origins-Corrects"); got != sha {
		t.Errorf("Origins-Corrects is %q, want %s", got, sha)
	}
	if now := read(t, corpus.Root, where); now != was {
		t.Errorf("the undo left:\n%s", now)
	}
	free(t, repo)
}

// Settling a hypothesis is one call, whichever way it goes. A rehearsal of accepting
// one writes it into the files and so has to put them back.
func TestAHypothesisIsSettledInOneCall(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	head := git(t, corpus.Root, "rev-parse", "HEAD")

	rehearsed := asking("origins hypothesis accept")
	rehearsed.DryRun = true
	decision, change, sha, err := Decide(corpus.Root, "unrecorded-child-1845", model.Accepted, rehearsed)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != model.Accepted || sha != "" || change == nil {
		t.Errorf("the rehearsal came back %s, %q", decision.Status, sha)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" ||
		git(t, corpus.Root, "rev-parse", "HEAD") != head {
		t.Errorf("the rehearsal was not put back:\n%s", left)
	}

	decision, _, sha, err = Decide(corpus.Root, "unrecorded-child-1845", model.Rejected,
		asking("origins hypothesis reject"))
	if err != nil {
		t.Fatal(err)
	}
	if decision.Status != model.Rejected || sha == "" {
		t.Errorf("the rejection came back %s, %q", decision.Status, sha)
	}
	if got := trailer(t, corpus.Root, "Origins-Hypothesis"); got != "unrecorded-child-1845" {
		t.Errorf("Origins-Hypothesis is %q", got)
	}
	if got := reloaded(t, corpus.Root).Hypotheses["unrecorded-child-1845"].Status; got != model.Rejected {
		t.Errorf("the hypothesis stands at %s", got)
	}
	free(t, repo)
}

// A change asked for from a picture of the corpus is pinned to what the picture showed:
// the record it addresses, a name that was free, or the commit an undo would take back.
// Each is refused once the corpus has moved on from it, and nothing is written.
func TestAChangeAskedForFromWhatWasShownIsRefusedOnceItHasChanged(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	annaMaria := model.EntityRef{Kind: model.KindPerson, Slug: "lang-anna-maria"}
	child := model.EntityRef{Kind: model.KindHypothesis, Slug: "unrecorded-child-1845"}
	shown, question := Seen(corpus, annaMaria), Seen(corpus, child)
	last, _, _ := repo.LastWritten()
	if Seen(corpus, model.EntityRef{Kind: model.KindPlace, Slug: "lang-anna-maria"}) != Absent {
		t.Error("a person is seen as a place of the same name")
	}

	if _, _, _, err := Carry(corpus.Root, model.Set{Entity: annaMaria, Path: "name.surname",
		Value: store.Typed("Rosi")}, asking("origins set")); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := Carry(corpus.Root, model.Set{Entity: child, Path: "confidence",
		Value: store.Typed("probable")}, asking("origins set")); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := Carry(corpus.Root, model.Create{Entity: model.EntityRef{Kind: model.KindPlace,
		Slug: "pontassieve"}}, asking("origins create")); err != nil {
		t.Fatal(err)
	}
	head := git(t, corpus.Root, "rev-parse", "HEAD")

	pinned := func(seen string) Asked {
		asked := asking("origins import")
		asked.Seen = seen
		return asked
	}
	var stale *Stale
	_, _, _, err := Carry(corpus.Root, herDeath(), pinned(shown))
	if !errors.As(err, &stale) {
		t.Errorf("a change to a record changed since it was shown: %v", err)
	}
	_, _, _, err = Decide(corpus.Root, child.Slug, model.Rejected, pinned(question))
	if !errors.As(err, &stale) {
		t.Errorf("rejecting a hypothesis changed since it was shown: %v", err)
	}
	_, _, _, err = Carry(corpus.Root, model.Create{Entity: model.EntityRef{Kind: model.KindPlace,
		Slug: "pontassieve"}}, pinned(Absent))
	if !errors.As(err, &stale) {
		t.Errorf("a create under a name taken since it was shown to be free: %v", err)
	}
	_, err = Undo(corpus.Root, pinned(last.Sha))
	if !errors.As(err, &stale) {
		t.Errorf("an undo of a change that is no longer the last: %v", err)
	}
	if got := git(t, corpus.Root, "rev-parse", "HEAD"); got != head {
		t.Errorf("HEAD moved from %s to %s", head, got)
	}
	if dirty := git(t, corpus.Root, "status", "--porcelain"); dirty != "" {
		t.Errorf("the tree is not clean:\n%s", dirty)
	}
	free(t, repo)

	now, err := store.Load(corpus.Root)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := Carry(corpus.Root, herDeath(), pinned(Seen(now, annaMaria))); err != nil {
		t.Errorf("a change to the record as it is now: %v", err)
	}
}
