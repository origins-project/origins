package ops

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimRight(string(out), "\r\n")
}

// inARepo is the corpus of onDisk, versioned and already committed once, with an
// identity of its own.
func inARepo(t *testing.T) (*store.Corpus, *history.Repo) {
	t.Helper()
	corpus := onDisk(t)

	git(t, corpus.Root, "init", "-q")
	git(t, corpus.Root, "config", "user.name", "Rosa Bianchi")
	git(t, corpus.Root, "config", "user.email", "rosa@example.invalid")
	git(t, corpus.Root, "add", "-A")
	git(t, corpus.Root, "commit", "-q", "-m", "The corpus as it stood")

	repo, err := history.Open(corpus.Root)
	if err != nil {
		t.Fatal(err)
	}
	return corpus, repo
}

func anEntry() history.Commit {
	return history.Commit{
		Subject:  "Record what the Mugello registers say",
		Tool:     "origins 0.2.0",
		Op:       "person.edit",
		Entities: []model.EntityRef{{Kind: model.KindPerson, Slug: "rossi-giovanni"}},
	}
}

// One operation is one commit, however many files it turns out to touch. A change
// spread over two commits is a corpus that was briefly in a state nobody chose, and
// bisecting over it finds the wrong commit.
func TestATransactionRecordsItselfAsOneCommit(t *testing.T) {
	t.Parallel()
	corpus, repo := inARepo(t)

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Body = "Born in the Mugello."
	tx.Corpus().Places["firenze"] = &model.Place{Slug: "firenze", Name: model.PlaceName{Text: "Firenze"}, Type: "city"}

	change, sha, err := tx.Commit(repo, anEntry())
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Fatalf("sha = %q", sha)
	}
	if got := change.Paths(); !slices.Equal(got, []string{
		"people/rossi-giovanni/README.md", "places/firenze/README.md",
	}) {
		t.Fatalf("touched %v", got)
	}

	if count := git(t, corpus.Root, "rev-list", "--count", "HEAD"); count != "2" {
		t.Errorf("the corpus has %s commits", count)
	}
	files := git(t, corpus.Root, "show", "--name-only", "--format=", "HEAD")
	if files != "people/rossi-giovanni/README.md\nplaces/firenze/README.md" {
		t.Errorf("the commit contains:\n%s", files)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
	if op := git(t, corpus.Root, "log", "-1", "--format=%(trailers:key=Origins-Op,valueonly)"); op != "person.edit" {
		t.Errorf("the operation came back as %q", op)
	}
}

// A corpus changed on disk and not recorded is the state the whole scheme exists to
// keep a researcher out of, and it is the one that looks healthiest from inside the
// files. So a commit git will not make takes the write back with it — the working tree
// and the index both.
func TestAChangeGitRefusesComesBackOffTheDisk(t *testing.T) {
	t.Parallel()
	corpus, repo := inARepo(t)
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")
	git(t, corpus.Root, "config", "--unset", "user.name")
	git(t, corpus.Root, "config", "--unset", "user.email")

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Body = "A change git will not accept."
	tx.Corpus().Places["firenze"] = &model.Place{Slug: "firenze", Name: model.PlaceName{Text: "Firenze"}, Type: "city"}

	if _, _, err := tx.Commit(repo, anEntry()); err == nil {
		t.Fatal("the commit reported success")
	}

	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the change was left on disk:\n%s", after)
	}
	if count := git(t, corpus.Root, "rev-list", "--count", "HEAD"); count != "1" {
		t.Errorf("the corpus has %s commits", count)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("something was left behind:\n%s", left)
	}
}

// A mutation that works out to what is already recorded is not a failure, and not a
// commit either. An empty commit would say a change was made when none was.
func TestATransactionThatChangesNothingRecordsNothing(t *testing.T) {
	t.Parallel()
	corpus, repo := inARepo(t)

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Name = model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")} // what it already said

	change, sha, err := tx.Commit(repo, anEntry())
	if err != nil {
		t.Fatal(err)
	}
	if !change.Empty() || sha != "" {
		t.Errorf("recorded %v as %q", change.Paths(), sha)
	}
	if count := git(t, corpus.Root, "rev-list", "--count", "HEAD"); count != "1" {
		t.Errorf("the corpus has %s commits", count)
	}
}

// A rename is one state and not two, so the file that goes and the file that arrives
// are in the same commit — which git then reads as the rename it is, so log --follow
// tracks the person across it. Two commits would leave the corpus briefly holding
// neither name, and would break the follow.
func TestAnEntityLeavingAndArrivingIsOneCommit(t *testing.T) {
	t.Parallel()
	corpus, repo := inARepo(t)

	tx := Begin(corpus)
	giovanni := tx.Corpus().People["rossi-giovanni"]
	delete(tx.Corpus().People, "rossi-giovanni")
	giovanni.Slug = "rossi-giovanni-1834"
	tx.Corpus().People["rossi-giovanni-1834"] = giovanni
	tx.Corpus().Project.MainPerson = "rossi-giovanni-1834"

	change, _, err := tx.Commit(repo, history.Commit{
		Subject: "Distinguish Giovanni from his grandson", Tool: "origins 0.2.0", Op: "person.rename",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []Move{{From: "people/rossi-giovanni", To: "people/rossi-giovanni-1834"}}
	if got := change.Moved(); !slices.Equal(got, want) {
		t.Fatalf("moved %v, want %v", got, want)
	}
	if got := change.Removed(); len(got) != 0 {
		t.Fatalf("the file was removed as well as carried: %v", got)
	}

	files := git(t, corpus.Root, "show", "--name-status", "--format=", "HEAD")
	shows := "M\torigins.yaml\nR100\tpeople/rossi-giovanni/README.md\tpeople/rossi-giovanni-1834/README.md"
	if files != shows {
		t.Errorf("the commit contains:\n%s\nwant:\n%s", files, shows)
	}
}

// The corpus of inARepo was committed by hand before origins ever touched it, which is
// how most corpora will start: the marker is absent until origins writes something of
// its own, and from then on it names what origins wrote.
func TestATransactionMarksTheCommitAsOriginsOwn(t *testing.T) {
	t.Parallel()
	corpus, repo := inARepo(t)
	if recorded, err := repo.LastRecorded(); err != nil || recorded != "" {
		t.Fatalf("a corpus origins has not written to came back as %q, %v", recorded, err)
	}

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Body = "Born in the Mugello."

	_, sha, err := tx.Commit(repo, anEntry())
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := repo.LastRecorded()
	if err != nil {
		t.Fatal(err)
	}
	if recorded != sha {
		t.Errorf("origins recorded %s and marked %q", sha, recorded)
	}
	if head := git(t, corpus.Root, "rev-parse", "HEAD"); recorded != head {
		t.Errorf("the marker reads %q and HEAD is %s", recorded, head)
	}
}
