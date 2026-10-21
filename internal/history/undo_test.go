package history

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// undoing takes back a commit the way a caller does: what to reverse, the files it
// reached, and a commit of origins' own to record the reversal as.
func undoing(t *testing.T, repo *Repo, e Entry) (string, error) {
	t.Helper()
	paths, err := repo.Touched(e.Sha)
	if err != nil {
		t.Fatal(err)
	}
	taking := Reversing(e)
	taking.Tool = "origins 0.2.0"
	return repo.Undo(e.Sha, paths, taking)
}

// holds is what a file of the corpus says, or nothing where there is no such file.
func holds(t *testing.T, repo *Repo, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repo.dir, filepath.FromSlash(relative)))
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// deeper is the same repository seen from a directory inside it: a corpus kept among
// other things rather than alone at the top of a repository of its own.
func deeper(t *testing.T, repo *Repo, relative string) *Repo {
	t.Helper()
	dir := filepath.Join(repo.dir, filepath.FromSlash(relative))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return &Repo{dir: dir}
}

// An undo puts the corpus back and leaves the history longer, not shorter. The commit
// it took back is still there to be read, because it is part of the research: somebody
// concluded that and then stopped concluding it, and both halves are the record.
func TestUndoingAChangePutsTheCorpusBackAndSaysSo(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	added := change("He died", "events.add", person("rossi-giovanni"))
	added.Events = []string{"death"}
	changing(t, repo, died, added)

	entries := logging(t, repo, Query{})
	took, err := undoing(t, repo, entries[0])
	if err != nil {
		t.Fatal(err)
	}
	if holds(t, repo, blamed) != born {
		t.Errorf("the file reads\n%s\nwant\n%s", holds(t, repo, blamed), born)
	}

	after := logging(t, repo, Query{})
	if len(after) != 3 {
		t.Fatalf("%d commits, not 3: %v", len(after), subjects(after))
	}
	undo := after[0]
	if undo.Sha != took {
		t.Errorf("the undo came back as %s, want %s", undo.Sha, took)
	}
	if undo.Subject != `Undo "He died"` {
		t.Errorf("the undo reads %q", undo.Subject)
	}
	if undo.Op != "history.undo" {
		t.Errorf("the undo is filed under %q", undo.Op)
	}
	if undo.Corrects != entries[0].Sha {
		t.Errorf("the undo corrects %q, want %s", undo.Corrects, entries[0].Sha)
	}
	if !reflect.DeepEqual(written(undo.Entities), []string{"person/rossi-giovanni"}) {
		t.Errorf("the undo is about %v", written(undo.Entities))
	}
	if !reflect.DeepEqual(undo.Events, []string{"death"}) {
		t.Errorf("the undo touches %v, want the death it took away", undo.Events)
	}
}

// An undo that read as somebody else's work would have the next command report
// perfectly good history as having moved underneath origins. It is a commit origins
// made, and it has to be marked like every other one.
func TestAnUndoIsOriginsOwnWork(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	changing(t, repo, died, change("He died", "events.add", person("rossi-giovanni")))

	took, err := undoing(t, repo, logging(t, repo, Query{})[0])
	if err != nil {
		t.Fatal(err)
	}
	if recorded, err := repo.LastRecorded(); err != nil || recorded != took {
		t.Errorf("the marker reads %q after an undo of %s, %v", recorded, took, err)
	}
	if undo := logging(t, repo, Query{})[0]; undo.Foreign() {
		t.Error("the undo reads as a commit origins did not write")
	}
}

// A file the reverted commit brought into the corpus goes away again, which is the
// case that cannot be staged the way every other commit is: the revert has already
// taken the file out of the index, and there is nothing left there to add.
func TestUndoingTheCommitThatCreatedAFileTakesTheFileAway(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/bianchi-maria/README.md", "---\nname: Maria\n---\n")
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	if _, err := repo.Commit(change("She was born", "person.create", person("bianchi-maria")),
		[]string{"people/bianchi-maria/README.md"}); err != nil {
		t.Fatal(err)
	}

	if _, err := undoing(t, repo, logging(t, repo, Query{})[0]); err != nil {
		t.Fatal(err)
	}
	if left := holds(t, repo, "people/bianchi-maria/README.md"); left != "" {
		t.Errorf("the file is still there, holding\n%s", left)
	}
	if status, err := repo.git("status", "--short"); err != nil || status != "" {
		t.Errorf("the corpus was left holding %q, %v", status, err)
	}
}

// Undoing is an ordinary change to the corpus, so it is itself the last thing origins
// did and itself undoable. That is the only way back from an undo, since nothing here
// rewrites history.
func TestUndoingAnUndoPutsTheChangeBack(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	changing(t, repo, died, change("He died", "events.add", person("rossi-giovanni")))

	if _, err := undoing(t, repo, logging(t, repo, Query{})[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := undoing(t, repo, logging(t, repo, Query{})[0]); err != nil {
		t.Fatal(err)
	}
	if holds(t, repo, blamed) != died {
		t.Errorf("the file reads\n%s\nwant the death back", holds(t, repo, blamed))
	}
}

// Git for Windows writes CRLF into every file it checks out unless it is told not to,
// and a repository with nothing declaring its line endings tells it nothing. What git
// writes for origins is still the bytes that were committed, whether it is putting a
// file back in the corpus or checking a commit out to be read.
func TestAFileGitWritesIsTheBytesThatWereCommitted(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	if _, err := repo.git("config", "core.autocrlf", "true"); err != nil {
		t.Fatal(err)
	}
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	changing(t, repo, died, change("He died", "events.add", person("rossi-giovanni")))
	first := logging(t, repo, Query{})[1].Sha

	if _, err := undoing(t, repo, logging(t, repo, Query{})[0]); err != nil {
		t.Fatal(err)
	}
	if got := holds(t, repo, blamed); got != born {
		t.Errorf("the undo left\n%q\nwant\n%q", got, born)
	}

	dir := filepath.Join(t.TempDir(), "then")
	within, err := repo.AsOf(first, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Discard(dir)
	if got, err := os.ReadFile(filepath.Join(within, filepath.FromSlash(blamed))); err != nil || string(got) != born {
		t.Errorf("the checkout holds\n%q\nwant\n%q, %v", got, born, err)
	}
}

// A file somebody is part way through editing is the one thing an undo must not walk
// over: the edit is work nobody else has a copy of, and the commit being taken back
// is in the history either way.
func TestAFileSomebodyIsEditingByHandStopsAnUndo(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	changing(t, repo, died, change("He died", "events.add", person("rossi-giovanni")))

	const halfWritten = "---\nname: Giovanni\nsurname: Rossi\nevents:\n  - type: death\n    date: 14 MAR\n"
	write(t, repo, blamed, halfWritten)
	_, err := undoing(t, repo, logging(t, repo, Query{})[0])
	if err == nil {
		t.Fatal("the undo reported success")
	}
	// git cleans up after a revert it would not start, so there is nothing left to
	// abandon and saying so would bury the reason underneath a second complaint
	if strings.Contains(err.Error(), "no cherry-pick or revert in progress") {
		t.Errorf("the refusal reads:\n%v", err)
	}
	if !strings.Contains(err.Error(), blamed) {
		t.Errorf("the refusal does not name the file in the way:\n%v", err)
	}
	if holds(t, repo, blamed) != halfWritten {
		t.Errorf("the hand-written file now reads\n%s", holds(t, repo, blamed))
	}
	if entries := logging(t, repo, Query{}); len(entries) != 2 {
		t.Errorf("%d commits, not 2: %v", len(entries), subjects(entries))
	}
}

// A reversal sitting in the working tree with no commit to account for it is the state
// the whole scheme exists to keep a researcher out of, and a refused commit is how it
// would happen. Taking it back out must leave everything else alone, because a
// researcher with one file open is the ordinary case and not an exception.
func TestACommitGitWillNotMakeLeavesNothingOfTheUndoBehind(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	changing(t, repo, died, change("He died", "events.add", person("rossi-giovanni")))
	write(t, repo, "people/bianchi-maria/README.md", "---\nname: Maria\n---\n")
	if _, err := repo.Commit(change("She was born", "person.create", person("bianchi-maria")),
		[]string{"people/bianchi-maria/README.md"}); err != nil {
		t.Fatal(err)
	}

	const beingEdited = "---\nname: Maria\nsurname: Bianchi\n---\n"
	write(t, repo, "people/bianchi-maria/README.md", beingEdited)
	for _, unset := range []string{"user.name", "user.email"} {
		if _, err := repo.git("config", "--unset", unset); err != nil {
			t.Fatal(err)
		}
	}

	recorded := logging(t, repo, Query{})[1]
	if recorded.Subject != "He died" {
		t.Fatalf("the test is undoing %q", recorded.Subject)
	}
	if _, err := undoing(t, repo, recorded); err == nil {
		t.Fatal("the undo reported success")
	}
	if holds(t, repo, blamed) != died {
		t.Errorf("the reversal was left in the working tree:\n%s", holds(t, repo, blamed))
	}
	if holds(t, repo, "people/bianchi-maria/README.md") != beingEdited {
		t.Errorf("the file being edited beside it now reads\n%s",
			holds(t, repo, "people/bianchi-maria/README.md"))
	}
	if status, err := repo.git("status", "--short"); err != nil || status != " M people/bianchi-maria/README.md" {
		t.Errorf("the corpus was left holding %q, %v", status, err)
	}
}

// Origins owns a corpus's history, so a commit it did not write is what it has to
// find before it can act on the one it did. The trailer is what says which is which,
// and it answers in a clone where the marker ref never went.
func TestTheLastCommitOriginsWroteIsTheOneUnderAnyForeignOnes(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	byHand(t, repo, "notes.md", "A thought I had")
	byHand(t, repo, "more-notes.md", "Another one")

	wrote, since, err := repo.LastWritten()
	if err != nil {
		t.Fatal(err)
	}
	if wrote == nil {
		t.Fatal("origins found none of its own work")
	}
	if wrote.Subject != "He was born" {
		t.Errorf("origins last wrote %q", wrote.Subject)
	}
	if want := []string{"Another one", "A thought I had"}; !reflect.DeepEqual(subjects(since), want) {
		t.Errorf("the commits on top of it are %v, want %v", subjects(since), want)
	}
}

// A history origins wrote no part of is an ordinary answer and not a failure: it is
// the corpus somebody started by hand and the repository origins has only just been
// pointed at.
func TestAHistoryOriginsHasNotWrittenHasNothingToUndo(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	byHand(t, repo, "notes.md", "A thought I had")

	wrote, since, err := repo.LastWritten()
	if err != nil {
		t.Fatal(err)
	}
	if wrote != nil {
		t.Errorf("origins claims to have written %q", wrote.Subject)
	}
	if !reflect.DeepEqual(subjects(since), []string{"A thought I had"}) {
		t.Errorf("the history reads %v", subjects(since))
	}
}

// Everything origins handles is named relative to the corpus, and a corpus is not
// always the top of its repository. A path relative to the repository would be looked
// for in the wrong place by everything that went on to use it.
func TestTheFilesACommitTouchedAreNamedAsTheCorpusNamesThem(t *testing.T) {
	t.Parallel()
	corpus := deeper(t, aRepo(t), "papers")
	changing(t, corpus, born, change("He was born", "person.create", person("rossi-giovanni")))
	sha := changing(t, corpus, died, change("He died", "events.add", person("rossi-giovanni")))

	touched, err := corpus.Touched(sha)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{blamed}; !reflect.DeepEqual(touched, want) {
		t.Errorf("the commit touched %v, want %v", touched, want)
	}
	if _, err := undoing(t, corpus, logging(t, corpus, Query{})[0]); err != nil {
		t.Fatal(err)
	}
	if holds(t, corpus, blamed) != born {
		t.Errorf("the file reads\n%s", holds(t, corpus, blamed))
	}
}

// The first commit of a history is one somebody can want back like any other — an
// import thought better of is the obvious case — and it is the one commit git will
// not describe unless asked to, having nothing to compare it against.
func TestTheFirstCommitOfAHistoryCanBeTakenBack(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))

	if _, err := undoing(t, repo, logging(t, repo, Query{})[0]); err != nil {
		t.Fatal(err)
	}
	if left := holds(t, repo, blamed); left != "" {
		t.Errorf("the file is still there, holding\n%s", left)
	}
	if status, err := repo.git("status", "--short"); err != nil || status != "" {
		t.Errorf("the corpus was left holding %q, %v", status, err)
	}
}

// written spells out what a commit is about, so a test can say what it expects in the
// same words the trailer does.
func written(refs []model.EntityRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.String())
	}
	return out
}
