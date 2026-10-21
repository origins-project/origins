package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func patching(t *testing.T, repo *Repo, paths ...string) string {
	t.Helper()
	patch, err := repo.Patch("", paths)
	if err != nil {
		t.Fatal(err)
	}
	return string(patch)
}

// A file git has never seen is the ordinary case here: a hypothesis that creates a
// record is one whole new file, and a diff that only looks at what git is tracking
// would carry an empty patch for it.
func TestAFileGitHasNotSeenIsInThePatch(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "people/rossi-annunziata/README.md", "---\nname: Annunziata Rossi\n---\n")

	patch := patching(t, repo, "people/rossi-annunziata/README.md")
	for _, line := range []string{
		"--- /dev/null",
		"+++ b/people/rossi-annunziata/README.md",
		"+name: Annunziata Rossi",
	} {
		if !strings.Contains(patch, line) {
			t.Errorf("the patch does not say %q:\n%s", line, patch)
		}
	}
}

// Only the paths it is given, as a commit takes only the paths it is given — and in
// both directions, since the index is where the question is put. A patch that swept up
// whatever else was lying about would carry somebody's unrecorded work off to wherever
// it was going, and one that staged it would leave that work looking recorded.
func TestAPatchTakesOnlyThePathsItIsGiven(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	write(t, repo, "notes/to-do.md", "Write to the parish at Vicchio.")
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md", "notes/to-do.md"}); err != nil {
		t.Fatal(err)
	}
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\nsex: M\n---\n")
	write(t, repo, "notes/to-do.md", "Written to. Nothing back yet.")
	write(t, repo, "notes/scrap.md", "Vicchio, or possibly Borgo.")
	if _, err := repo.git("add", "--", "notes/to-do.md"); err != nil {
		t.Fatal(err)
	}

	patch := patching(t, repo, "people/rossi-giovanni/README.md")
	if !strings.Contains(patch, "+sex: M") {
		t.Errorf("the patch does not carry the change it was asked for:\n%s", patch)
	}
	if strings.Contains(patch, "to-do") || strings.Contains(patch, "scrap") {
		t.Errorf("the patch carried off something nobody asked it to:\n%s", patch)
	}
	staged, err := repo.git("diff", "--cached", "--name-only")
	if err != nil {
		t.Fatal(err)
	}
	if want := "notes/to-do.md\npeople/rossi-giovanni/README.md"; staged != want {
		t.Errorf("it staged %q, and was asked about one of them", staged)
	}
}

// A patch is bytes and not an answer. One whose last newline has been taken off for
// tidiness is one git refuses at the far end, with a message about a corrupt patch
// that says nothing about where it was truncated.
func TestAPatchKeepsTheNewlineGitWroteAtTheEndOfIt(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")

	if patch := patching(t, repo, "people/rossi-giovanni/README.md"); !strings.HasSuffix(patch, "\n") {
		t.Errorf("the patch ends %q", patch[max(0, len(patch)-20):])
	}
}

// Nothing to carry is not a patch of nothing. A pathspec of none is how git spells
// everything, so the question has to be refused rather than asked.
func TestAPatchOfNoPathsIsNoPatch(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")

	if patch := patching(t, repo); patch != "" {
		t.Errorf("patched %q", patch)
	}
}

// What a hypothesis pins is a commit, and the answer for a name that reaches anything
// else — a tree, a blob, nothing at all — is the same no.
func TestOnlyACommitAnswersForOne(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	sha, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"})
	if err != nil {
		t.Fatal(err)
	}
	tree, err := repo.git("rev-parse", "HEAD^{tree}")
	if err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]bool{
		sha:       true,
		sha[:7]:   true,
		"HEAD":    true,
		tree:      false,
		"8f78553": false,
	} {
		switch has, err := repo.Has(name); {
		case err != nil:
			t.Errorf("%s: %v", name, err)
		case has != want:
			t.Errorf("Has(%s) = %v", name, has)
		}
	}
}

// A state no commit holds can still be what a patch is made from. A file moved before
// the tree was taken is where the tree has it, so the patch is the edit made after
// and nothing of the move — and taking the tree has left the history where it was.
func TestAPatchFromATreeIsTheChangeSinceTheTree(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-annunziata/README.md", "---\nname: Annunziata Rossi\n---\n")
	head, err := repo.Commit(anEntry(), []string{"people/rossi-annunziata/README.md"})
	if err != nil {
		t.Fatal(err)
	}
	moved := "people/rossi-annunziata-1842/README.md"
	write(t, repo, moved, "---\nname: Annunziata Rossi\n---\n")
	if err := os.Remove(filepath.Join(repo.dir, "people", "rossi-annunziata", "README.md")); err != nil {
		t.Fatal(err)
	}
	tree, err := repo.Tree()
	if err != nil {
		t.Fatal(err)
	}
	write(t, repo, moved, "---\nname: Annunziata Rossi\nsex: F\n---\n")

	patch, err := repo.Patch(tree, []string{moved})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"--- a/" + moved, "+++ b/" + moved, "+sex: F"} {
		if !strings.Contains(string(patch), line) {
			t.Errorf("the patch does not say %q:\n%s", line, patch)
		}
	}
	if now, err := repo.git("rev-parse", "HEAD"); err != nil || now != head {
		t.Errorf("HEAD is %s, and was %s before the tree was taken (%v)", now, head, err)
	}
}
