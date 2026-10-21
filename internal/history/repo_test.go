package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A family is named by both of its partners, so two slugs at their full length join
// into a file name of a hundred and thirty characters, and a corpus kept a few
// directories down crosses Windows' old path limit somewhere in the middle of the tree.
// Go writes such a file without noticing and git will not index it unless it is told
// to, which would leave origins having written a corpus it cannot record.
func TestAPathLongerThanWindowsOnceAllowedIsStillRecorded(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	const name = "families/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
		"--bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb.md"
	long := name
	for len(filepath.Join(repo.dir, filepath.FromSlash(long))) < 300 {
		long = "deeper/" + long
	}
	write(t, repo, long, "---\nname: a family of two long names\n---\n")

	sha, err := repo.Commit(anEntry(), []string{"."})
	if err != nil {
		t.Fatalf("a path of %d characters was not recorded: %v",
			len(filepath.Join(repo.dir, filepath.FromSlash(long))), err)
	}
	held, err := repo.git("show", "--name-only", "--format=", sha)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(held, long) {
		t.Errorf("the commit holds %q, want the file with the long path", held)
	}
}

// What git said is the answer, and what origins asked is the context. A commit carries
// its whole message as one argument, trailers and all, so reproducing the command
// verbatim buries git's own explanation under it — and in the case that matters most,
// an identity git has not been told, the explanation is the part naming the two
// commands to run.
func TestARefusedCommitSaysWhatGitSaidAndNotTheWholeMessageBack(t *testing.T) {
	repo := aRepo(t)
	write(t, repo, "corpus.md", "He was born\n")
	t.Setenv("GIT_COMMITTER_DATE", "not a date at all")

	_, err := repo.Commit(anEntry(), []string{"corpus.md"})
	if err == nil {
		t.Fatal("a commit was made with a date git cannot read")
	}
	if !strings.Contains(err.Error(), "invalid date format") {
		t.Errorf("git's own explanation is missing: %v", err)
	}
	if strings.Contains(err.Error(), "Origins-Tool") {
		t.Errorf("the whole commit message is quoted back: %v", err)
	}
}

// A corpus written where there is no repository gets one. Nothing else can be done for
// it: its first commit has to go somewhere, and origins owns the history of a corpus
// from the moment there is a corpus to have one.
func TestStartMakesARepositoryWhereThereIsNone(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	work := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}

	repo, made, err := Start(work)
	if err != nil {
		t.Fatal(err)
	}
	if !made {
		t.Error("Start said it had found a repository where there was none")
	}
	if _, err := os.Stat(filepath.Join(work, ".git")); err != nil {
		t.Errorf("no repository was made: %v", err)
	}
	if _, err := repo.Root(); err != nil {
		t.Errorf("the repository Start made cannot be read: %v", err)
	}
}

// A corpus written inside a repository somebody already keeps is recorded in that one,
// so a tree can live beside the notes, the photographs and the README it belongs with.
// A repository nested inside another is invisible to it, and a history started there
// would be lost the first time anybody cloned the outer one.
func TestStartUsesTheRepositoryACorpusIsAlreadyInside(t *testing.T) {
	t.Parallel()
	outer := aRepo(t)
	inside := deeper(t, outer, "trees/nonna")

	repo, made, err := Start(inside.dir)
	if err != nil {
		t.Fatal(err)
	}
	if made {
		t.Error("Start made a repository inside the one that was already there")
	}
	if _, err := os.Stat(filepath.Join(inside.dir, ".git")); err == nil {
		t.Error("a repository was made inside the one that was already there")
	}
	root, err := repo.Root()
	if err != nil {
		t.Fatal(err)
	}
	was, err := outer.Root()
	if err != nil {
		t.Fatal(err)
	}
	if root != was {
		t.Errorf("the corpus is recorded in %q, want the repository already there at %q", root, was)
	}
}
