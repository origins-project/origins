package history

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// One writer holds the corpus at a time, and a second is told who has it rather than
// made to wait: waiting would have it carry on over a corpus it read before the first
// one changed it.
func TestASecondWriterIsRefusedAndTheFirstCanGiveItUp(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)

	release, err := repo.Lock("origins import")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.Lock("origins set")
	var locked *Locked
	if !errors.As(err, &locked) {
		t.Fatalf("a second lock came back %v", err)
	}
	if locked.Pid != strconv.Itoa(os.Getpid()) || locked.Command != "origins import" {
		t.Errorf("the refusal names process %q running %q", locked.Pid, locked.Command)
	}
	for _, said := range []string{locked.Path, "origins import", "removed"} {
		if !strings.Contains(err.Error(), said) {
			t.Errorf("the refusal does not say %q: %v", said, err)
		}
	}

	// a refusal is not the holder, and gives up nothing
	if _, err := os.Stat(locked.Path); err != nil {
		t.Fatalf("the refused writer took the lock away: %v", err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	again, err := repo.Lock("origins set")
	if err != nil {
		t.Fatalf("the lock was given up and still refused: %v", err)
	}
	if err := again(); err != nil {
		t.Fatal(err)
	}
}

// The lock lives in the git directory, where git's own locks are, so it is never a
// change to the corpus: nothing a commit could pick up, and nothing store would refuse
// as a file beside no record.
func TestTheLockIsNeverPartOfTheCorpus(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	release, err := repo.Lock("origins set")
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	if _, err := os.Stat(filepath.Join(repo.dir, ".git", lockFile)); err != nil {
		t.Errorf("the lock is not in the git directory: %v", err)
	}
	if dirty, err := repo.Dirty(); err != nil || len(dirty) != 0 {
		t.Errorf("holding the lock left the corpus dirty: %v, %v", dirty, err)
	}
}

// A corpus kept a few directories down a repository of notes and photographs locks the
// repository's git directory, and not some directory of its own git would never read.
func TestACorpusInsideALargerRepositoryLocksThatRepository(t *testing.T) {
	t.Parallel()
	outer := aRepo(t)
	corpus := filepath.Join(outer.dir, "family", "rossi")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	inner := &Repo{dir: corpus}

	release, err := inner.Lock("origins set")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if _, err := os.Stat(filepath.Join(outer.dir, ".git", lockFile)); err != nil {
		t.Errorf("the lock is not in the repository's git directory: %v", err)
	}
	var locked *Locked
	if _, err := outer.Lock("origins adopt"); !errors.As(err, &locked) {
		t.Errorf("the repository was locked from below and taken again from the top: %v", err)
	}
}

// A lock somebody made by hand, or one caught between being made and being written,
// says nothing of who holds it, and is refused just the same.
func TestALockThatSaysNothingIsStillALock(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	path := filepath.Join(repo.dir, ".git", lockFile)
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := repo.Lock("origins set")
	var locked *Locked
	if !errors.As(err, &locked) {
		t.Fatalf("an empty lock came back %v", err)
	}
	if !strings.Contains(err.Error(), "another origins") {
		t.Errorf("the refusal claims to know who holds it: %v", err)
	}
}

// A lock that cannot be given up says so as an Unreleased, naming the file, so that
// whoever held it can pass the warning on without counting its own work as failed.
func TestALockThatCannotBeGivenUpSaysWhichFile(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	release, err := repo.Lock("origins set")
	if err != nil {
		t.Fatal(err)
	}
	// named as git names the git directory, which on Windows may be the long form of a
	// temporary directory the test was given in its short one
	path, err := repo.GitPath(lockFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "kept"), 0o755); err != nil {
		t.Fatal(err)
	}

	err = release()
	var unreleased *Unreleased
	if !errors.As(err, &unreleased) || unreleased.Path != path {
		t.Fatalf("a lock that could not be removed came back %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("the warning does not name the file: %v", err)
	}
}
