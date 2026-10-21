package history

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// TestMain points git at configuration of the package's own for every test in it, so
// that nothing a test asserts depends on how the machine running it is set up. It is
// done once here rather than by each test, because Go will not run a test that changes
// the environment beside any other; a test that needs git configured some other way
// sets it itself and runs alone.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "origins-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "no-such-global-config"))
	os.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(dir, "no-such-system-config"))
	// git packs no test's repository after a commit, none living long enough to need
	// it, and commits as nobody it has not been told of, rather than guessing at an
	// identity from the machine, which on some machines it finds
	os.Setenv("GIT_CONFIG_COUNT", "2")
	os.Setenv("GIT_CONFIG_KEY_0", "maintenance.auto")
	os.Setenv("GIT_CONFIG_VALUE_0", "false")
	os.Setenv("GIT_CONFIG_KEY_1", "user.useConfigOnly")
	os.Setenv("GIT_CONFIG_VALUE_1", "true")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// gitFixture is a copy of testgit, the fixture corpus built through origins from
// nothing, with the history a researcher would have made getting there. A test is free
// to change the copy, and it has an identity of its own for the commits it makes.
//
// It is generated rather than committed and takes a couple of minutes to build, so this
// skips when it is not there. Run tools/mkgitfixture.sh to produce it.
func gitFixture(t *testing.T) *Repo {
	t.Helper()
	built := filepath.Join("..", "..", "testgit")
	if _, err := os.Stat(built); err != nil {
		t.Skip("no git fixture; run tools/mkgitfixture.sh")
	}
	dir := t.TempDir()

	work := filepath.Join(dir, "corpus")
	if err := os.CopyFS(work, os.DirFS(built)); err != nil {
		t.Fatal(err)
	}
	repo := &Repo{dir: work}
	for _, command := range [][]string{
		{"config", "user.name", "Rosa Bianchi"},
		{"config", "user.email", "rosa@example.invalid"},
	} {
		if _, err := repo.git(command...); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

// Every commit in the fixture was made by an origins command, so there is nothing in
// its history for recover to resolve and nothing between it and the next change.
func TestTheGitFixtureIsOriginsOwnThroughout(t *testing.T) {
	t.Parallel()
	standing, err := gitFixture(t).Survey()
	if err != nil {
		t.Fatal(err)
	}
	if !standing.Settled() || standing.Problems != nil || len(standing.Dirty) > 0 {
		t.Errorf("the fixture stands as %s", standing.Describe())
	}
}

// Iván and Rachel's family was created under the name the grandson had before his
// grandfather was found and took it, and the rename that carried the family to its
// present name is the only thread back to where it began.
func TestTheGitFixtureLogFollowsAFamilyAcrossItsPartnersRename(t *testing.T) {
	t.Parallel()
	entries, err := gitFixture(t).Log(Query{Entities: []model.EntityRef{
		{Kind: model.KindFamily, Slug: "ershov-ivan-1885--modiano-rachel"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var did []string
	for _, entry := range entries {
		did = append(did, entry.Op)
	}
	want := []string{"person.rename", "events.add", "family.create"}
	if !slices.Equal(did, want) {
		t.Errorf("the family's log is %q, want %q", did, want)
	}
}

// The grandson was created as ershov-ivan and renamed to make room for his
// grandfather, who was then created under the name. Each has a log of his own, and
// nothing about one of them is in the other's.
func TestTheGitFixtureKeepsTheTwoIvansApart(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	for _, one := range []struct {
		asking model.Slug
		want   []string
	}{
		{"ershov-ivan", []string{
			"body.set", "events.add", "events.add", "events.add", "events.add", "person.create",
		}},
		{"ershov-ivan-1885", []string{
			"body.set", "same-as.add", "person.rename",
			"events.add", "events.add", "events.add", "person.create",
		}},
	} {
		entries, err := repo.Log(Query{Entities: []model.EntityRef{person(one.asking)}})
		if err != nil {
			t.Fatal(err)
		}
		var did []string
		for _, entry := range entries {
			did = append(did, entry.Op)
		}
		if !slices.Equal(did, one.want) {
			t.Errorf("asked after %s: got %q, want %q", one.asking, did, one.want)
		}
	}
}
