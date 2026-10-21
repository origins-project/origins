package history

import (
	"testing"
	"time"
)

func taking(t *testing.T, repo *Repo, name, message string) Snapshot {
	t.Helper()
	taken, err := repo.Take(name, message)
	if err != nil {
		t.Fatal(err)
	}
	return taken
}

func listing(t *testing.T, repo *Repo) []Snapshot {
	t.Helper()
	snapshots, err := repo.Snapshots()
	if err != nil {
		t.Fatal(err)
	}
	return snapshots
}

// A snapshot names the corpus as it stands, and what it is for is the whole point of
// taking one: three years later the question is not which commit it was but what was
// sent to whom.
func TestASnapshotNamesTheCorpusAsItStands(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	sha := changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))

	taken := taking(t, repo, "sent-to-zia-maria", "everything I had before the Arezzo trip")
	if taken.Name != "sent-to-zia-maria" {
		t.Errorf("the snapshot is called %q", taken.Name)
	}
	if taken.Sha != sha {
		t.Errorf("the snapshot names %s, want the commit at the head, %s", taken.Sha, sha)
	}
	if taken.Message != "everything I had before the Arezzo trip" {
		t.Errorf("the snapshot says %q", taken.Message)
	}
	if time.Since(taken.When) > time.Hour {
		t.Errorf("the snapshot was taken at %s", taken.When)
	}
}

// Taking a snapshot is an act with a date of its own. What was shared with a relative
// in March is that state of the corpus as of March, whenever the last change to it
// happened to have been made, and only an annotated tag carries the second date.
func TestASnapshotIsDatedWhenItWasTakenAndNotWhenTheCorpusLastChanged(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, blamed, born)
	for _, command := range [][]string{
		{"add", "--", blamed},
		{"commit", "-q", "-m", "He was born", "--date=2020-03-04T10:00:00+00:00", "--", blamed},
	} {
		if _, err := repo.git(command...); err != nil {
			t.Fatal(err)
		}
	}

	taken := taking(t, repo, "sent-to-zia-maria", "")
	if taken.When.Year() == 2020 {
		t.Errorf("the snapshot is dated %s, which is when the commit was made", taken.When)
	}
}

// Nobody has to say what a state was named for, and origins cannot guess it. An empty
// message is left empty rather than filled with a sentence that repeats the name.
func TestASnapshotNobodySaidAnythingAboutSaysNothing(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))

	if taken := taking(t, repo, "before-arezzo", ""); taken.Message != "" {
		t.Errorf("the snapshot says %q of its own accord", taken.Message)
	}
}

// Snapshots read back newest first, like everything else origins reads out of a
// history.
func TestSnapshotsComeBackNewestFirst(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	taking(t, repo, "before-arezzo", "")
	changing(t, repo, died, change("He died", "events.add", person("rossi-giovanni")))
	taking(t, repo, "after-arezzo", "with the parish register read")

	snapshots := listing(t, repo)
	if len(snapshots) != 2 {
		t.Fatalf("%d snapshots came back, not 2", len(snapshots))
	}
	if snapshots[0].Name != "after-arezzo" || snapshots[1].Name != "before-arezzo" {
		t.Errorf("they came back as %s then %s", snapshots[0].Name, snapshots[1].Name)
	}
	if snapshots[0].Sha == snapshots[1].Sha {
		t.Error("both snapshots name the same commit")
	}
}

// A snapshot is defined as a tag rather than merely kept as one, so a tag made with
// git is the same statement about the same commit and origins has no business hiding
// it. A lightweight tag carries neither a date nor a message of its own, and what it
// does not carry is reported as absent rather than as the commit's.
func TestATagSomebodyMadeWithGitIsASnapshotToo(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	sha := changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	if _, err := repo.git("tag", "sent-by-hand"); err != nil {
		t.Fatal(err)
	}

	snapshots := listing(t, repo)
	if len(snapshots) != 1 {
		t.Fatalf("%d snapshots came back, not 1", len(snapshots))
	}
	if snapshots[0].Name != "sent-by-hand" || snapshots[0].Sha != sha {
		t.Errorf("the snapshot is %s at %s, want sent-by-hand at %s",
			snapshots[0].Name, snapshots[0].Sha, sha)
	}
	if snapshots[0].Message != "" {
		t.Errorf("the snapshot says %q, which is the commit talking", snapshots[0].Message)
	}
}

// A corpus nobody has named a state of has no snapshots, which is an answer and not a
// failure.
func TestACorpusWithNoSnapshotsHasNone(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))

	if snapshots := listing(t, repo); len(snapshots) != 0 {
		t.Errorf("%d snapshots came back", len(snapshots))
	}
}

// Two states cannot share a name, since the point of the name is to say which state
// was shared. Git refuses it and says so, and origins has nothing to add.
func TestASnapshotCannotTakeANameAlreadyTaken(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	taking(t, repo, "before-arezzo", "")

	changing(t, repo, died, change("He died", "events.add", person("rossi-giovanni")))
	if _, err := repo.Take("before-arezzo", ""); err == nil {
		t.Fatal("the second snapshot reported success")
	}
	if snapshots := listing(t, repo); len(snapshots) != 1 {
		t.Errorf("%d snapshots came back, not 1", len(snapshots))
	}
}
