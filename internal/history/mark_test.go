package history

import (
	"path/filepath"
	"testing"
)

// The ref is the whole of origins' memory of its own work. A commit that did not move
// it would have the next command report perfectly good history as somebody else's.
func TestACommitMarksWhatOriginsLastWrote(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	paths := []string{"people/rossi-giovanni/README.md"}

	// a repository origins has never written to has nothing to say, and saying so is
	// not a failure: it is the fresh clone, and the corpus somebody started by hand
	if recorded, err := repo.LastRecorded(); err != nil || recorded != "" {
		t.Fatalf("an untouched repository came back as %q, %v", recorded, err)
	}

	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	first, err := repo.Commit(anEntry(), paths)
	if err != nil {
		t.Fatal(err)
	}
	if recorded, err := repo.LastRecorded(); err != nil || recorded != first {
		t.Fatalf("origins wrote %s and marked %q, %v", first, recorded, err)
	}

	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n\nBorn in the Mugello.\n")
	second, err := repo.Commit(anEntry(), paths)
	if err != nil {
		t.Fatal(err)
	}
	if recorded, err := repo.LastRecorded(); err != nil || recorded != second {
		t.Fatalf("the marker stayed at %q rather than following to %s, %v", recorded, second, err)
	}
}

// A commit git would not make is not something origins wrote, so the marker still has
// to name the last thing it did write. Moving it would report the commit before this
// one as work somebody else had done.
func TestACommitGitRefusesLeavesTheMarkerWhereItWas(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	paths := []string{"people/rossi-giovanni/README.md"}
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	sha, err := repo.Commit(anEntry(), paths)
	if err != nil {
		t.Fatal(err)
	}

	for _, unset := range []string{"user.name", "user.email"} {
		if _, err := repo.git("config", "--unset", unset); err != nil {
			t.Fatal(err)
		}
	}
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n\nA change git will refuse.\n")
	if _, err := repo.Commit(anEntry(), paths); err == nil {
		t.Fatal("the commit reported success")
	}

	if recorded, err := repo.LastRecorded(); err != nil || recorded != sha {
		t.Errorf("the marker reads %q, want %s, %v", recorded, sha, err)
	}
}

// The ref is one clone's memory of its own work, and a clone that inherited another
// machine's marker would take an ordinary pull for history that had moved underneath
// it. Nothing in the code arranges that: it comes of the ref living outside refs/heads
// and refs/tags, which git's default refspecs carry in neither direction. So what goes
// out here is every branch and every tag rather than the one branch: a marker kept as
// either would survive pushing a single branch, and those are the two shapes this has
// to rule out.
func TestTheMarkerStaysOnTheMachineThatWroteIt(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	branch, err := repo.git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		t.Fatal(err)
	}

	elsewhere := filepath.Dir(repo.dir)
	shared := &Repo{dir: filepath.Join(elsewhere, "shared.git")}
	clone := &Repo{dir: filepath.Join(elsewhere, "clone")}
	for _, command := range [][]string{
		{"init", "-q", "--bare", shared.dir},
		{"remote", "add", "origin", shared.dir},
		{"push", "-q", "--all", "origin"},
		{"push", "-q", "--tags", "origin"},
		{"clone", "-q", shared.dir, clone.dir},
	} {
		if _, err := repo.git(command...); err != nil {
			t.Fatal(err)
		}
	}

	refs, err := shared.git("for-each-ref", "--format=%(refname)")
	if err != nil {
		t.Fatal(err)
	}
	if refs != "refs/heads/"+branch {
		t.Errorf("pushing everything put these refs on the remote:\n%s", refs)
	}
	for _, other := range []struct {
		what string
		repo *Repo
	}{
		{"pushed to", shared},
		{"cloned into", clone},
	} {
		recorded, err := other.repo.LastRecorded()
		if err != nil {
			t.Fatal(err)
		}
		if recorded != "" {
			t.Errorf("the repository %s carries a marker of %q", other.what, recorded)
		}
	}
}
