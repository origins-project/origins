package history

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// standing is what git makes of the corpus: empty where nothing has been changed
// since the last commit, and the porcelain lines otherwise.
func standing(t *testing.T, repo *Repo) string {
	t.Helper()
	out, err := repo.git("status", "--porcelain", "--", ".")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func restoring(t *testing.T, repo *Repo) {
	t.Helper()
	if err := repo.Restore(); err != nil {
		t.Fatal(err)
	}
}

// proposing is the patch a change to several records makes at the state the
// repository stands on, with the repository put back to that state afterwards. It is
// writtenThen for a hypothesis that touches more than one record, which is what one
// creating a record always does.
func proposing(t *testing.T, repo *Repo, written map[string]string) string {
	t.Helper()
	was, err := repo.git("rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	paths := make([]string, 0, len(written))
	for relative, content := range written {
		write(t, repo, relative, content)
		paths = append(paths, relative)
	}
	slices.Sort(paths)
	patch := patching(t, repo, paths...)
	for _, command := range [][]string{{"reset", "--hard", "-q", was}, {"clean", "-fdq"}} {
		if _, err := repo.git(command...); err != nil {
			t.Fatal(err)
		}
	}
	return patch
}

// The state accepting a hypothesis has to be able to get out of. A patch that no
// longer applies leaves two readings of one fact in the file and a half-merged entry
// in the index, and a record the hypothesis creates arrives whether the rest of it
// conflicted or not.
func TestACorpusGoesBackToWhatWasRecordedAfterAPatchThatWouldNotApply(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	corrected := slices.Clone(asRecorded)
	corrected[5] = "death: 1902"
	patch := proposing(t, repo, map[string]string{
		"people/rossi-giovanni/README.md":   aRecord(corrected...),
		"people/rossi-annunziata/README.md": aRecord("name: Annunziata Rossi", "sex: F"),
	})

	otherwise := slices.Clone(asRecorded)
	otherwise[5] = "death: 1903"
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(otherwise...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	recorded := onDisk(t, repo, "people/rossi-giovanni/README.md")

	if conflicts := carrying(t, repo, patch); len(conflicts) == 0 {
		t.Fatal("two readings of the death did not conflict, so there is nothing to restore from")
	}
	restoring(t, repo)

	if got := standing(t, repo); got != "" {
		t.Errorf("the corpus is not back to what was recorded:\n%s", got)
	}
	if got := onDisk(t, repo, "people/rossi-giovanni/README.md"); got != recorded {
		t.Errorf("the record still holds the conflict:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(repo.dir, "people", "rossi-annunziata", "README.md")); !os.IsNotExist(err) {
		t.Errorf("the record the patch created is still there: %v", err)
	}
}

// A patch that applied is undone the same way, which is what a rehearsal of accepting
// rests on: everything happens, and then none of it has.
func TestACorpusGoesBackToWhatWasRecordedAfterAPatchThatDidApply(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	recorded := onDisk(t, repo, "people/rossi-giovanni/README.md")

	corrected := slices.Clone(asRecorded)
	corrected[5] = "death: 1902"
	patch := writtenThen(t, repo, "people/rossi-giovanni/README.md", aRecord(corrected...))

	if conflicts := carrying(t, repo, patch); len(conflicts) > 0 {
		t.Fatalf("the patch did not apply: %v", conflicts)
	}
	restoring(t, repo)

	if got := standing(t, repo); got != "" {
		t.Errorf("the corpus is not back to what was recorded:\n%s", got)
	}
	if got := onDisk(t, repo, "people/rossi-giovanni/README.md"); got != recorded {
		t.Errorf("the correction is still in the record:\n%s", got)
	}
}

// A corpus is often a directory inside a repository somebody keeps their notes and
// photographs in, and none of that is origins' to put back. Every command here names
// the corpus and nothing above it, and this is the one where getting that wrong would
// throw away somebody's unrecorded work.
func TestRestoringLeavesTheRestOfTheRepositoryAlone(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	corpus := &Repo{dir: filepath.Join(repo.dir, "family")}
	write(t, repo, "README.md", "What is in this repository\n")
	write(t, corpus, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{"README.md", "family"}); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "README.md", "Notes I have not finished\n")
	write(t, repo, "letters/1861.txt", "Caro Giovanni,\n")

	restoring(t, corpus)

	if got := onDisk(t, repo, "README.md"); got != "Notes I have not finished\n" {
		t.Errorf("the note above the corpus was put back:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(repo.dir, "letters", "1861.txt")); err != nil {
		t.Errorf("a file beside the corpus was taken away: %v", err)
	}
}

// What an ignore rule keeps out of the history is not origins' to take away. A
// researcher's scratch directory beside a corpus is the ordinary case, and putting a
// patch back is no occasion to go through it.
func TestRestoringLeavesWhatAnIgnoreRuleKeepsOut(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, ".gitignore", "scratch/\n")
	write(t, repo, "people/rossi-giovanni/README.md", aRecord(asRecorded...))
	if _, err := repo.Commit(anEntry(), []string{".gitignore", "people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "scratch/notes.txt", "the 1861 census, page 214\n")

	restoring(t, repo)

	if _, err := os.Stat(filepath.Join(repo.dir, "scratch", "notes.txt")); err != nil {
		t.Errorf("the scratch notes were taken away: %v", err)
	}
}
