package history

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// diverged is the fixture pushed to a bare repository, with a collaborator's commit
// pushed on top of it there and one of this clone's own on top of it here, each
// changing the file named.
func diverged(t *testing.T, there, here string) (*Repo, *Repo) {
	t.Helper()
	repo := gitFixture(t)
	bare := shared(t, repo)
	if err := repo.AddRemote("origin", bare); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Push("origin", quiet()); err != nil {
		t.Fatal(err)
	}
	other := elsewhere(t, bare)
	appendTo(t, other, there, "Changed there.\n")
	commitAll(t, other, "There", "Origins-Tool: origins test\nOrigins-Op: body.set")
	if _, err := other.git("push", "-q"); err != nil {
		t.Fatal(err)
	}
	appendTo(t, repo, here, "Changed here.\n")
	commitAll(t, repo, "Here", "Origins-Tool: origins test\nOrigins-Op: body.set")
	if err := repo.Fetch("origin", quiet()); err != nil {
		t.Fatal(err)
	}
	return repo, other
}

func appendTo(t *testing.T, repo *Repo, relative, text string) {
	t.Helper()
	path := filepath.Join(repo.dir, filepath.FromSlash(relative))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, text...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commitAll(t *testing.T, repo *Repo, subject, trailers string) {
	t.Helper()
	if _, err := repo.git("commit", "-q", "-a", "-m", subject+"\n\n"+trailers); err != nil {
		t.Fatal(err)
	}
}

func TestTheIncomingBranchIsTheUpstreamAndWhatIsAboveItIsForeign(t *testing.T) {
	t.Parallel()
	repo, other := diverged(t, "people/verdier-anna/README.md", "places/kumasi/README.md")
	ref, sha, err := repo.Incoming("origin")
	if err != nil {
		t.Fatal(err)
	}
	if theirs, _ := other.Head(); ref != "origin/main" || sha != theirs {
		t.Errorf("the incoming branch is %s at %s", ref, sha)
	}
	if foreign, err := repo.Above(ref); err != nil || len(foreign) != 0 {
		t.Errorf("above the upstream: %v (%v)", foreign, err)
	}
	if n, err := repo.Count(ref); n != 1 || err != nil {
		t.Errorf("the upstream has %d commits HEAD does not (%v)", n, err)
	}

	appendTo(t, other, "people/verdier-anna/README.md", "By hand.\n")
	commitAll(t, other, "By hand", "Signed-off-by: Paolo Verdi <paolo@example.invalid>")
	if _, err := other.git("push", "-q"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Fetch("origin", quiet()); err != nil {
		t.Fatal(err)
	}
	foreign, err := repo.Above(ref)
	if err != nil || len(foreign) != 1 || foreign[0].Subject != "By hand" {
		t.Errorf("above the upstream: %v (%v)", foreign, err)
	}
	if _, _, err := repo.Incoming("nowhere"); err == nil {
		t.Error("a remote with no such branch gave one")
	}
}

func TestAMergeIsConcludedAsOneOfOriginsOwnCommits(t *testing.T) {
	t.Parallel()
	repo, _ := diverged(t, "people/verdier-anna/README.md", "places/kumasi/README.md")
	head, _ := repo.Head()
	conflicts, err := repo.Merge("origin/main")
	if err != nil || conflicts != nil {
		t.Fatalf("the merge: %v (%v)", conflicts, err)
	}
	if merging, _ := repo.Merging(); !merging {
		t.Fatal("no merge is part way through")
	}
	sha, err := repo.Conclude(Commit{Subject: "Pull origin/main", Tool: "origins test", Op: Pulled})
	if err != nil {
		t.Fatal(err)
	}
	if parents := strings.Fields(must(t, repo, "log", "-1", "--format=%P")); len(parents) != 2 || parents[0] != head {
		t.Errorf("the merge commit has parents %v", parents)
	}
	if marker, _ := repo.LastRecorded(); marker != sha {
		t.Errorf("the marker is at %s, not %s", marker, sha)
	}
	if standing, err := repo.Survey(); err != nil || !standing.Settled() || len(standing.Dirty) > 0 {
		t.Errorf("after the merge the history stands as %s (%v)", standing.Describe(), err)
	}
}

func TestAMergeOfLinesChangedOnBothSidesIsAbandoned(t *testing.T) {
	t.Parallel()
	repo, _ := diverged(t, "people/verdier-anna/README.md", "people/verdier-anna/README.md")
	head, _ := repo.Head()
	conflicts, err := repo.Merge("origin/main")
	if err != nil || !slices.Equal(conflicts, []string{"people/verdier-anna/README.md"}) {
		t.Errorf("the merge: %v (%v)", conflicts, err)
	}
	if merging, _ := repo.Merging(); merging {
		t.Error("the merge is still part way through")
	}
	if now, _ := repo.Head(); now != head {
		t.Error("HEAD moved")
	}
	if dirty, _ := repo.Dirty(); len(dirty) > 0 {
		t.Errorf("the abandoned merge left %v", dirty)
	}
}

func TestAFastForwardMarksWhereItArrives(t *testing.T) {
	t.Parallel()
	repo, other := diverged(t, "people/verdier-anna/README.md", "places/kumasi/README.md")
	if _, err := repo.git("reset", "-q", "--hard", "HEAD~1"); err != nil {
		t.Fatal(err)
	}
	sha, err := repo.FastForward("origin/main")
	if err != nil {
		t.Fatal(err)
	}
	if theirs, _ := other.Head(); sha != theirs {
		t.Errorf("fast-forwarded to %s, not %s", sha, theirs)
	}
	if marker, _ := repo.LastRecorded(); marker != sha {
		t.Errorf("the marker is at %s", marker)
	}
}

// Outside asks of the rest of a repository a corpus is kept inside, and of nothing
// where the corpus is the whole of it.
func TestOutsideNamesWhatIsStagedOrModifiedBesideTheCorpus(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	if outside, err := repo.Outside(); err != nil || outside != nil {
		t.Errorf("the corpus is the repository, and outside it is %v (%v)", outside, err)
	}
	if err := os.WriteFile(filepath.Join(repo.dir, "notes.txt"), []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	must(t, repo, "add", "notes.txt")
	must(t, repo, "commit", "-q", "-m", "Notes")

	tree := &Repo{dir: filepath.Join(repo.dir, "people")}
	if outside, err := tree.Outside(); err != nil || outside != nil {
		t.Errorf("before anything changed, outside is %v (%v)", outside, err)
	}
	appendTo(t, repo, "notes.txt", "more\n")
	appendTo(t, repo, "people/verdier-anna/README.md", "Inside.\n")
	if outside, err := tree.Outside(); err != nil || !slices.Equal(outside, []string{"notes.txt"}) {
		t.Errorf("outside is %v (%v)", outside, err)
	}
}

func must(t *testing.T, repo *Repo, args ...string) string {
	t.Helper()
	out, err := repo.git(args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
