package history

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// aRecord is long enough for a three-way merge to have something to work with, and
// spelled the way the corpus spells one so that what the tests do to it reads as
// research rather than as line noise.
func aRecord(fields ...string) string {
	return "---\n" + strings.Join(fields, "\n") + "\n---\n\nBorn at Vicchio.\n"
}

var asRecorded = []string{
	"name: Giovanni Rossi",
	"surname: Rossi",
	"given: Giovanni",
	"sex: M",
	"birth: 1834",
	"death: 1901",
	"birthplace: vicchio",
}

func onDisk(t *testing.T, repo *Repo, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repo.dir, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func carrying(t *testing.T, repo *Repo, patch string) []string {
	t.Helper()
	conflicts, err := repo.Carry([]byte(patch))
	if err != nil {
		t.Fatal(err)
	}
	return conflicts
}

// writtenThen is the patch a change makes at the state the repository stands on, and
// the repository put back to that state afterwards. It is how a hypothesis reaches a
// corpus that has since moved: written then, carried now.
func writtenThen(t *testing.T, repo *Repo, relative, content string) string {
	t.Helper()
	standing, err := repo.git("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	write(t, repo, relative, content)
	patch := patching(t, repo, relative)
	if _, err := repo.git("reset", "--hard", "-q", standing); err != nil {
		t.Fatal(err)
	}
	return patch
}

// The whole of what carrying a patch is for. A correction written against one state of
// a record reaches the record as it now stands, and what somebody changed in between
// is still there afterwards — which is what makes a hypothesis worth keeping for as
// long as the question stays open.
func TestAPatchWrittenAtOneStateReachesTheRecordAsItNowStands(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	corrected := slices.Clone(asRecorded)
	corrected[5] = "death: 1902"
	patch := writtenThen(t, repo, "people/rossi-giovanni/README.md", aRecord(corrected...))

	moved := slices.Clone(asRecorded)
	moved[0] = "name: Giovanni Battista Rossi"
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(moved...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	if conflicts := carrying(t, repo, patch); len(conflicts) > 0 {
		t.Fatalf("two changes to two facts conflicted: %v", conflicts)
	}
	now := onDisk(t, repo, "people/rossi-giovanni/README.md")
	for _, says := range []string{"death: 1902", "name: Giovanni Battista Rossi"} {
		if !strings.Contains(now, says) {
			t.Errorf("the record does not say %q:\n%s", says, now)
		}
	}
}

// A patch names the corpus and git resolves a path against the top of the working
// tree, so a corpus kept a few directories down needs the prefix put back. The failure
// this guards is the quiet one: git matches nothing, says it skipped the patch, and
// exits nought, which reads as a hypothesis that changed the record to exactly what it
// already said.
func TestAPatchReachesACorpusKeptBelowTheRepositoryRoot(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	corpus := &Repo{dir: filepath.Join(repo.dir, "family")}
	write(t, corpus, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := corpus.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	corrected := slices.Clone(asRecorded)
	corrected[5] = "death: 1902"
	patch := writtenThen(t, corpus, "people/rossi-giovanni/README.md", aRecord(corrected...))
	if strings.Contains(patch, "family/") {
		t.Fatalf("the patch names the repository rather than the corpus:\n%s", patch)
	}

	if conflicts := carrying(t, corpus, patch); len(conflicts) > 0 {
		t.Fatalf("carrying a patch onto the state it was made at conflicted: %v", conflicts)
	}
	if now := onDisk(t, corpus, "people/rossi-giovanni/README.md"); !strings.Contains(now, "death: 1902") {
		t.Errorf("the patch reached nothing:\n%s", now)
	}
}

// Two readings of one fact are the answer to the question, not a failure to arrive at
// one, so a conflict is returned rather than raised and it names the file it is in.
func TestTwoChangesToOneFactComeBackAsAConflictAndNotAnError(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	corrected := slices.Clone(asRecorded)
	corrected[5] = "death: 1902"
	patch := writtenThen(t, repo, "people/rossi-giovanni/README.md", aRecord(corrected...))

	otherwise := slices.Clone(asRecorded)
	otherwise[5] = "death: 1903"
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(otherwise...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	conflicts := carrying(t, repo, patch)
	if want := []string{"people/rossi-giovanni/README.md"}; !slices.Equal(conflicts, want) {
		t.Errorf("conflicted in %v, want %v", conflicts, want)
	}
}

// A three-way merge takes the side it is merging onto out of the index and refuses a
// file the index and the working tree disagree about. A corpus with changes nobody has
// recorded is exactly that disagreement, and it is the ordinary state of a corpus
// somebody is working in, so everything is staged before the patch goes near it.
func TestAPatchIsCarriedOntoChangesNobodyHasRecorded(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	corrected := slices.Clone(asRecorded)
	corrected[5] = "death: 1902"
	patch := writtenThen(t, repo, "people/rossi-giovanni/README.md", aRecord(corrected...))

	byHand := slices.Clone(asRecorded)
	byHand[0] = "name: Giovanni Battista Rossi"
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(byHand...))

	if conflicts := carrying(t, repo, patch); len(conflicts) > 0 {
		t.Fatalf("an unrecorded change to another line conflicted: %v", conflicts)
	}
	now := onDisk(t, repo, "people/rossi-giovanni/README.md")
	for _, says := range []string{"death: 1902", "name: Giovanni Battista Rossi"} {
		if !strings.Contains(now, says) {
			t.Errorf("the record does not say %q:\n%s", says, now)
		}
	}
}

// A hypothesis whose edits were already true where they were written makes a patch
// with nothing in it, which git reads as input it cannot make sense of at all rather
// than as a change that comes to nothing.
func TestAPatchWithNothingInItCarriesNothing(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	before := onDisk(t, repo, "people/rossi-giovanni/README.md")

	if conflicts := carrying(t, repo, ""); len(conflicts) > 0 {
		t.Errorf("carrying nothing conflicted: %v", conflicts)
	}
	if after := onDisk(t, repo, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("carrying nothing changed the record:\n%s", after)
	}
}

// A file git has never seen is the ordinary case for a hypothesis that creates a
// record, and it is invisible to a three-way merge until it has been added.
func TestAPatchIsCarriedOntoAFileGitHasNotSeen(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	patch := writtenThen(t, repo, "people/rossi-annunziata/README.md",
		aRecord("name: Annunziata Rossi", "sex: F"))

	write(t, repo, "places/vicchio/README.md", aRecord("name: Vicchio"))
	if conflicts := carrying(t, repo, patch); len(conflicts) > 0 {
		t.Fatalf("a record nobody else touched conflicted: %v", conflicts)
	}
	if now := onDisk(t, repo, "people/rossi-annunziata/README.md"); !strings.Contains(now, "Annunziata") {
		t.Errorf("the record the hypothesis creates is not there:\n%s", now)
	}
	if now := onDisk(t, repo, "places/vicchio/README.md"); !strings.Contains(now, "Vicchio") {
		t.Errorf("an unrecorded file beside it was written over:\n%s", now)
	}
}
