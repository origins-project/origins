package history

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rewriting replays the foreign commits the way a caller does, keeping each subject
// and marking each as work origins now answers for.
func rewriting(t *testing.T, repo *Repo, anchor string, foreign []Entry) []string {
	t.Helper()
	made, err := repo.Rewrite(anchor, foreign, func(was Entry) (Commit, error) {
		return Commit{Subject: was.Subject, Tool: "origins 0.2.0", Op: "manual.adopt"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// A note says something about a commit without touching it, which is the only thing
// that can be said about one that has been shared.
func TestANoteLeavesTheCommitAloneAndKeepsWhatWasThere(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	sha := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	if err := repo.Note(sha, "Checked against the register"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Note(sha, "Reviewed by origins"); err != nil {
		t.Fatal(err)
	}
	notes, err := repo.git("notes", "show", sha)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Checked against the register", "Reviewed by origins"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the note reads %q, want it to keep %q", notes, want)
		}
	}
	if now, err := repo.git("rev-parse", "HEAD"); err != nil || now != sha {
		t.Errorf("the commit is now %s, want it untouched at %s", short(now), short(sha))
	}
}

// Rewinding takes the commits off and keeps every word of what they said, which is
// what makes it the resolution that decides nothing.
func TestRewindingLeavesWhatTheCommitsSaidInTheTree(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	anchor := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	byHand(t, repo, "corpus.md", "Fix the birth year")

	if err := repo.Rewind(anchor); err != nil {
		t.Fatal(err)
	}
	if now, _ := repo.git("rev-parse", "HEAD"); now != anchor {
		t.Errorf("HEAD is %s, want the commit origins wrote", short(now))
	}
	if held := holds(t, repo, "corpus.md"); held != "Fix the birth year\n" {
		t.Errorf("the file reads %q, want what the commit that came off said", held)
	}
	standing := surveying(t, repo)
	if !standing.Settled() {
		t.Errorf("the history is still unsettled after a rewind: %s", standing.Describe())
	}
	if len(standing.Dirty) != 1 || standing.Dirty[0] != "corpus.md" {
		t.Errorf("the unrecorded files are %q, want the one the commit had changed", standing.Dirty)
	}
}

// Asking what the corpus used to say means reading the files it used to have, and a
// checkout of its own is where they can be read without disturbing anything.
func TestACorpusCanBeReadBackAsSomeCommitLeftIt(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	was := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("He died", "events.add", person("rossi-giovanni")))

	where, err := repo.AsOf(was, filepath.Join(t.TempDir(), "recorded"))
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(where, "corpus.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "He was born\n" {
		t.Errorf("the corpus as of that commit reads %q", content)
	}
	if held := holds(t, repo, "corpus.md"); held != "He died\n" {
		t.Errorf("reading an older state changed the corpus itself: %q", held)
	}
	if err := repo.Discard(where); err != nil {
		t.Fatal(err)
	}
	if listed, _ := repo.git("worktree", "list"); strings.Contains(listed, "recorded") {
		t.Errorf("the checkout is still one git knows about:\n%s", listed)
	}
}

// A corpus kept in a subdirectory of a larger repository is read back from the same
// place inside the checkout, which is not the top of it.
func TestACorpusInASubdirectoryIsFoundInTheCheckoutToo(t *testing.T) {
	t.Parallel()
	outer := aRepo(t)
	repo := deeper(t, outer, "corpus")
	sha := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	where, err := repo.AsOf(sha, filepath.Join(t.TempDir(), "recorded"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Discard(where) }()
	if _, err := os.Stat(filepath.Join(where, "corpus.md")); err != nil {
		t.Errorf("the corpus was not found where the checkout keeps it: %v", err)
	}
}

// A path with nothing at it is nothing to discard. The checkouts AsOf makes are thrown
// away in a defer that runs however the reading went, including where it went so badly
// that there was never a checkout to throw away.
func TestDiscardingAPathWithNothingAtItIsNotAFailure(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	if err := repo.Discard(filepath.Join(t.TempDir(), "never-made")); err != nil {
		t.Errorf("discarding a checkout that was never made: %v", err)
	}
}

// A directory git never checked anything out into is refused, since it is what git
// says about a path that is not there either and only the filesystem tells them apart.
func TestDiscardingADirectoryThatIsNoCheckoutIsRefused(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	stranger := filepath.Join(t.TempDir(), "somebody-elses")
	if err := os.MkdirAll(stranger, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := repo.Discard(stranger); err == nil {
		t.Error("a directory that is no checkout of origins' was discarded as though it were one")
	}
}

// A checkout nobody took away, as one left by a process killed part way through, is
// nothing the repository knows about: git lists no worktree of it, and a copy of the
// repository carries nothing of it.
func TestACheckoutLeftBehindLeavesNothingInTheRepository(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	sha := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))

	if _, err := repo.AsOf(sha, filepath.Join(t.TempDir(), "recorded")); err != nil {
		t.Fatal(err)
	}
	if listed, _ := repo.git("worktree", "list"); strings.Contains(listed, "recorded") {
		t.Errorf("the checkout is one git knows about:\n%s", listed)
	}
	if _, err := os.Stat(filepath.Join(repo.dir, ".git", "worktrees")); !os.IsNotExist(err) {
		t.Errorf("the repository's git directory has worktrees in it: %v", err)
	}
}

// A rewrite replaces what the commits are without replacing what they were. Somebody
// wrote that change on that day, which stays true of the commit standing in its place,
// and the sequence they made it in is the shape of how they thought.
func TestARewriteKeepsEachCommitWithItsAuthorAndItsDate(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	anchor := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	for _, subject := range []string{"Write down what I know", "Correct the place"} {
		write(t, repo, "corpus.md", subject+"\n")
		if _, err := repo.git("add", "--", "corpus.md"); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.git("-c", "user.name=Nonna Lucia", "-c", "user.email=lucia@example.invalid",
			"commit", "-q", "--date=2021-03-04T05:06:07+00:00", "-m", subject); err != nil {
			t.Fatal(err)
		}
	}

	before := surveying(t, repo)
	made := rewriting(t, repo, anchor, before.Foreign)
	if len(made) != 2 {
		t.Fatalf("%d commits were made, not 2", len(made))
	}

	after := surveying(t, repo)
	switch {
	case !after.Settled():
		t.Errorf("the history is still unsettled after a rewrite: %s", after.Describe())
	case after.Anchor == nil || after.Anchor.Sha != made[len(made)-1]:
		t.Errorf("the anchor is %+v, want the last commit the rewrite made", after.Anchor)
	}

	entries := logging(t, repo, Query{})
	if got := subjects(entries); len(got) != 3 || got[0] != "Correct the place" ||
		got[1] != "Write down what I know" {
		t.Errorf("the history reads %q, want the commits in the order they were made", got)
	}
	for _, was := range entries[:2] {
		switch {
		case was.Author != "Nonna Lucia":
			t.Errorf("%q is now the work of %q", was.Subject, was.Author)
		case was.When.Year() != 2021:
			t.Errorf("%q is now dated %s", was.Subject, was.When)
		case was.Op != "manual.adopt":
			t.Errorf("%q came back as %q", was.Subject, was.Op)
		}
	}
	if held := holds(t, repo, "corpus.md"); held != "Correct the place\n" {
		t.Errorf("the corpus reads %q, want what the last commit said", held)
	}
}

// Nothing moves until all of it has worked. The replay happens away from the branch,
// so a commit origins cannot describe costs a checkout and not a history.
func TestARewriteThatFailsPartWayLeavesTheBranchWhereItWas(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	anchor := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	byHand(t, repo, "corpus.md", "Write down what I know")
	byHand(t, repo, "corpus.md", "Correct the place")
	was, _ := repo.git("rev-parse", "HEAD")

	standing := surveying(t, repo)
	refused := errors.New("this one cannot be described")
	_, err := repo.Rewrite(anchor, standing.Foreign, func(e Entry) (Commit, error) {
		if e.Subject == "Correct the place" {
			return Commit{}, refused
		}
		return Commit{Subject: e.Subject, Tool: "origins 0.2.0", Op: "manual.adopt"}, nil
	})
	if !errors.Is(err, refused) {
		t.Fatalf("the rewrite reported %v, want the reason it stopped", err)
	}
	if now, _ := repo.git("rev-parse", "HEAD"); now != was {
		t.Errorf("HEAD is %s, want the branch untouched at %s", short(now), short(was))
	}
	if branch, _ := repo.git("symbolic-ref", "--short", "HEAD"); branch == "" {
		t.Error("the repository was left on a detached HEAD")
	}
	if held := holds(t, repo, "corpus.md"); held != "Correct the place\n" {
		t.Errorf("the corpus reads %q, want what it said before the rewrite", held)
	}
}

// A rewrite needs a branch to point at the result, and a detached HEAD is not one.
// Replaying onto nothing would leave the commits reachable from nowhere.
func TestARewriteWillNotStartOnADetachedHead(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	anchor := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	byHand(t, repo, "corpus.md", "Fix the birth year")
	standing := surveying(t, repo)
	if _, err := repo.git("checkout", "--detach", "-q", "HEAD"); err != nil {
		t.Fatal(err)
	}

	_, err := repo.Rewrite(anchor, standing.Foreign, func(Entry) (Commit, error) {
		return Commit{}, errors.New("should never be asked")
	})
	if err == nil || !strings.Contains(err.Error(), "detached") {
		t.Errorf("the rewrite reported %v, want a refusal naming the detached HEAD", err)
	}
}

// Replaying sets the working tree to each commit in turn, so an edit nobody has
// recorded would be written over and gone. It is the one thing a recovery must not
// cost, since an unrecorded edit is the work most easily lost.
//
// The edit is to a file the commits being replayed never touch, which is the case git
// would not catch on its own: checking the anchor out carries a change to such a file
// across quite happily, and only setting the tree to each commit in turn takes it away.
func TestARewriteWillNotWriteOverAnEditNobodyRecorded(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "notes.md", "what I have been reading\n")
	anchor, err := repo.Commit(change("He was born", "person.create", person("rossi-giovanni")),
		[]string{"notes.md"})
	if err != nil {
		t.Fatal(err)
	}
	byHand(t, repo, "corpus.md", "Fix the birth year")
	standing := surveying(t, repo)

	const partway = "Something I am part way through\n"
	write(t, repo, "notes.md", partway)

	_, err = repo.Rewrite(anchor, standing.Foreign, func(Entry) (Commit, error) {
		return Commit{}, errors.New("should never be asked")
	})
	if err == nil || !strings.Contains(err.Error(), "notes.md") {
		t.Errorf("the rewrite reported %v, want a refusal naming the file", err)
	}
	if held := holds(t, repo, "notes.md"); held != partway {
		t.Errorf("the edit reads %q, want it left exactly as it was", held)
	}
}
