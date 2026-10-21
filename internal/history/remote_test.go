package history

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// shared is a bare repository cloned from the fixture, beside it in the test's own
// directory and reached by its path, so that nothing a test sends leaves the machine.
func shared(t *testing.T, repo *Repo) string {
	t.Helper()
	bare := filepath.Join(filepath.Dir(repo.dir), "shared.git")
	if _, err := repo.git("clone", "-q", "--bare", ".", bare); err != nil {
		t.Fatal(err)
	}
	return bare
}

// elsewhere is another clone of a shared repository, with an identity of its own, for
// a collaborator's commits to be made in.
func elsewhere(t *testing.T, bare string) *Repo {
	t.Helper()
	dir := filepath.Join(filepath.Dir(bare), "elsewhere")
	command := exec.Command("git", "clone", "-q", bare, dir)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	other := &Repo{dir: dir}
	for _, command := range [][]string{
		{"config", "user.name", "Paolo Verdi"},
		{"config", "user.email", "paolo@example.invalid"},
	} {
		if _, err := other.git(command...); err != nil {
			t.Fatal(err)
		}
	}
	return other
}

func quiet() Terminal { return Terminal{In: strings.NewReader(""), Err: &strings.Builder{}} }

func TestARemoteIsAddedListedAndRemoved(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	bare := shared(t, repo)

	if remotes, err := repo.Remotes(); err != nil || len(remotes) != 0 {
		t.Fatalf("the fixture has remotes %v (%v)", remotes, err)
	}
	if err := repo.AddRemote("origin", bare); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddRemote("zia-maria", "https://example.invalid/zia maria.git"); err != nil {
		t.Fatal(err)
	}
	remotes, err := repo.Remotes()
	if err != nil {
		t.Fatal(err)
	}
	want := []Remote{{"origin", bare}, {"zia-maria", "https://example.invalid/zia maria.git"}}
	if !slices.Equal(remotes, want) {
		t.Errorf("the remotes are %v, want %v", remotes, want)
	}

	if err := repo.AddRemote("origin", bare); err == nil {
		t.Error("a second remote of the same name was added")
	}
	if err := repo.RemoveRemote("zia-maria"); err != nil {
		t.Fatal(err)
	}
	if remotes, _ := repo.Remotes(); !slices.Equal(remotes, want[:1]) {
		t.Errorf("after removing one the remotes are %v", remotes)
	}
	if err := repo.RemoveRemote("zia-maria"); err == nil {
		t.Error("a remote that is not there was removed")
	}
}

// A fetch moves no branch, so where the branch stands afterwards is a count on each
// side: what a collaborator pushed, and what was recorded here and not yet sent.
func TestAFetchCountsWhatEachSideHasThatTheOtherDoesNot(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	bare := shared(t, repo)
	if err := repo.AddRemote("origin", bare); err != nil {
		t.Fatal(err)
	}
	if err := repo.Fetch("", quiet()); err != nil {
		t.Fatal(err)
	}
	tracking, err := repo.Tracks()
	if err != nil {
		t.Fatal(err)
	}
	if tracking != (Tracking{Branch: "main"}) {
		t.Errorf("before an upstream is set the branch stands as %+v", tracking)
	}
	if _, err := repo.git("branch", "--set-upstream-to", "origin/main"); err != nil {
		t.Fatal(err)
	}

	other := elsewhere(t, bare)
	for _, message := range []string{"one", "two"} {
		if _, err := other.git("commit", "-q", "--allow-empty", "-m", message); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := other.git("push", "-q"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.git("commit", "-q", "--allow-empty", "-m", "here"); err != nil {
		t.Fatal(err)
	}

	head, _ := repo.Head()
	if err := repo.Fetch("origin", quiet()); err != nil {
		t.Fatal(err)
	}
	if after, _ := repo.Head(); after != head {
		t.Errorf("the fetch moved HEAD from %s to %s", head, after)
	}
	tracking, err = repo.Tracks()
	if err != nil {
		t.Fatal(err)
	}
	want := Tracking{Branch: "main", Upstream: "origin/main", Ahead: 1, Behind: 2}
	if tracking != want {
		t.Errorf("the branch stands as %+v, want %+v", tracking, want)
	}
}

// A remote is named by whoever asks, a page among them, and a name is never an option:
// one that read as `--upload-pack` would run whatever it went on to say.
func TestARemoteNamedLikeAnOptionIsOnlyAName(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	bare := shared(t, repo)
	if err := repo.AddRemote("origin", bare); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.git("push", "-q", "--set-upstream", "origin", "main"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Fetch("--dry-run", quiet()); err == nil {
		t.Error("a fetch from --dry-run was taken as a fetch from origin")
	}
	if _, err := repo.Push("--dry-run", quiet()); err == nil {
		t.Error("a push to --dry-run was taken as a push to origin")
	}
	if err := repo.AddRemote("--tags", bare); err != nil {
		t.Fatal(err)
	}
	if remotes, _ := repo.Remotes(); !slices.Contains(remotes, Remote{"--tags", bare}) {
		t.Errorf("after adding --tags the remotes are %v", remotes)
	}
	if err := repo.RemoveRemote("--tags"); err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveRemote("--verbose"); err == nil {
		t.Error("a remote named --verbose was removed")
	}
	if _, err := repo.Take("--list", ""); err == nil {
		t.Error("a snapshot named --list was taken")
	}
}

// What git says when it cannot reach a remote is said to the terminal, and the error
// names the command.
func TestAFetchFromNowhereIsUnreached(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	if err := repo.AddRemote("origin", filepath.Join(t.TempDir(), "no-such.git")); err != nil {
		t.Fatal(err)
	}
	var said strings.Builder
	err := repo.Fetch("origin", Terminal{In: strings.NewReader(""), Err: &said})
	unreached, ok := err.(*Unreached)
	if !ok {
		t.Fatalf("fetching from nowhere returned %v", err)
	}
	if !slices.Equal(unreached.Args, []string{"fetch", "--", "origin"}) {
		t.Errorf("the error names git %v", unreached.Args)
	}
	if said.Len() == 0 {
		t.Error("git said nothing to the terminal")
	}
}

// The remote is the one named, the one the branch follows, or the only one there is.
func TestTowardChoosesTheRemoteAsSaidFollowedOrAlone(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	bare := shared(t, repo)
	if _, err := repo.Toward(""); !errors.As(err, &NoRemote{}) {
		t.Errorf("with no remote Toward returned %v", err)
	}
	if err := repo.AddRemote("origin", bare); err != nil {
		t.Fatal(err)
	}
	if remote, err := repo.Toward(""); remote != "origin" || err != nil {
		t.Errorf("with one remote Toward chose %q (%v)", remote, err)
	}
	if err := repo.AddRemote("backup", bare); err != nil {
		t.Fatal(err)
	}
	var many ManyRemotes
	if _, err := repo.Toward(""); !errors.As(err, &many) || !slices.Equal(many.Names, []string{"backup", "origin"}) {
		t.Errorf("with two remotes Toward returned %v", err)
	}
	if remote, _ := repo.Toward("backup"); remote != "backup" {
		t.Errorf("asked for backup, Toward chose %q", remote)
	}
	pushed, err := repo.Push("backup", quiet())
	if err != nil {
		t.Fatal(err)
	}
	if !pushed.Upstream {
		t.Error("the first push set no upstream")
	}
	if remote, _ := repo.Toward(""); remote != "backup" {
		t.Errorf("following backup, Toward chose %q", remote)
	}
	if pushed, err := repo.Push("origin", quiet()); err != nil || pushed.Upstream {
		t.Errorf("pushing to the remote not followed: %+v (%v)", pushed, err)
	}
	if remote, _ := repo.Toward(""); remote != "backup" {
		t.Errorf("after pushing elsewhere, Toward chose %q", remote)
	}
}

func TestAPushTheRemoteHasOvertakenIsSaidSo(t *testing.T) {
	t.Parallel()
	repo := gitFixture(t)
	bare := shared(t, repo)
	if err := repo.AddRemote("origin", bare); err != nil {
		t.Fatal(err)
	}
	other := elsewhere(t, bare)
	if _, err := other.git("commit", "-q", "--allow-empty", "-m", "there"); err != nil {
		t.Fatal(err)
	}
	if _, err := other.git("push", "-q"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.git("commit", "-q", "--allow-empty", "-m", "here"); err != nil {
		t.Fatal(err)
	}
	_, err := repo.Push("origin", quiet())
	var overtaken *Overtaken
	if !errors.As(err, &overtaken) || overtaken.Branch != "main" {
		t.Errorf("the push returned %v", err)
	}
}

func TestClonedNamesTheDirectoryAsGitWould(t *testing.T) {
	t.Parallel()
	for url, want := range map[string]string{
		"git@example.org:rossi.git":            "rossi",
		"https://example.org/family/rossi":     "rossi",
		"https://example.org/family/rossi/":    "rossi",
		"https://example.org/rossi/.git":       "rossi",
		`D:\trees\rossi.git`:                   "rossi",
		"/srv/git/rossi.git/":                  "rossi",
		"ssh://git@example.org/~paolo/bianchi": "bianchi",
	} {
		if got := Cloned(url); got != want {
			t.Errorf("Cloned(%q) = %q, want %q", url, got, want)
		}
	}
}
