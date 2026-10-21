package ops

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
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

// The fixture corpus is generated rather than committed, so this skips when it is not
// there. Run tools/mkfixture.sh to produce it.
func fixture(t *testing.T) *store.Corpus {
	t.Helper()
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	corpus, err := store.Load(root)
	if err != nil {
		t.Fatalf("the fixture should load without a single problem:\n%v", err)
	}
	return corpus
}

// The broadest test of the diff there is: fifty-three entities, every awkward case the
// fixture carries, and a transaction that was asked to change none of them. Anything it
// stages is a disagreement between what the reader understood and what the emitter
// would write back, which would turn every future commit into a corpus-wide rewrite.
//
// It stages rather than applies. The fixture is a directory on disk like any other, and
// a test has no business writing to it.
func TestATransactionThatChangesNothingStagesNothing(t *testing.T) {
	t.Parallel()
	change, err := Begin(fixture(t)).Stage()
	if err != nil {
		t.Fatal(err)
	}
	if !change.Empty() {
		t.Errorf("would write %v and remove %v", change.Written(), change.Removed())
	}
}

// One fact changed in a corpus of fifty-three entities is one file changed.
func TestOneChangeToTheFixtureTouchesOneFile(t *testing.T) {
	t.Parallel()
	tx := Begin(fixture(t))
	tx.Corpus().People["verdier-louisa"].Body = "The census ages were reported at the door."

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"people/verdier-louisa/README.md"}
	if got := change.Written(); !slices.Equal(got, want) {
		t.Errorf("staged %v, want %v", got, want)
	}
}

// gitFixture is a copy of testgit, the fixture corpus built through origins from
// nothing, with the history a researcher would have made getting there. A test is free
// to change the copy, and it has an identity of its own for the commits it makes.
//
// It is generated rather than committed and takes a couple of minutes to build, so this
// skips when it is not there. Run tools/mkgitfixture.sh to produce it.
func gitFixture(t *testing.T) (*store.Corpus, *history.Repo) {
	t.Helper()
	built := filepath.Join("..", "..", "testgit")
	if _, err := os.Stat(built); err != nil {
		t.Skip("no git fixture; run tools/mkgitfixture.sh")
	}
	dir := t.TempDir()

	root := filepath.Join(dir, "corpus")
	if err := os.CopyFS(root, os.DirFS(built)); err != nil {
		t.Fatal(err)
	}
	git(t, root, "config", "user.name", "Rosa Bianchi")
	git(t, root, "config", "user.email", "rosa@example.invalid")

	repo, err := history.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return reloaded(t, root), repo
}

// The fixture's hypotheses pin a commit well behind the last one, with the arguments
// for most of the corpus written since. Louisa's is one of them, so her hypothesis
// is carried onto a file that has changed since it was written.
func TestTheGitFixturesHypothesesAreCarriedAcrossEverythingSince(t *testing.T) {
	t.Parallel()
	corpus, _ := gitFixture(t)
	where, _ := corpus.Path("verdier-louisa")
	before := read(t, corpus.Root, where)

	trial, err := Suppose(corpus, []model.Slug{"louisa-born-1844", "unrecorded-child-1845"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = trial.Close() })

	for _, reading := range trial.Read {
		if base := corpus.Hypotheses[reading.Slug].Base; !reading.Pinned() || reading.Base != base {
			t.Errorf("%v, want it read at %s", reading, base)
		}
	}
	louisa := trial.Corpus.People["verdier-louisa"]
	if got := louisa.Events[0].Date.String(); got != "1844" {
		t.Errorf("Louisa was born in %s in the trial corpus, want 1844", got)
	}
	if louisa.Body != corpus.People["verdier-louisa"].Body {
		t.Error("the argument written since the base did not survive carrying the date across")
	}
	if _, ok := trial.Corpus.People["verdier-unknown-1845"]; !ok {
		t.Error("the child the other hypothesis proposed is not in the trial corpus")
	}
	if after := read(t, corpus.Root, where); after != before {
		t.Errorf("the corpus itself was changed:\n%s", after)
	}
}

// Accepting one is the same road ending in the corpus rather than in a copy.
func TestAHypothesisInTheGitFixtureIsAcceptedAcrossEverythingSince(t *testing.T) {
	t.Parallel()
	corpus, repo := gitFixture(t)
	decision, err := Accept(repo, corpus, "unrecorded-child-1845")
	landing(t, decision, err)

	after := reloaded(t, corpus.Root)
	if _, ok := after.People["verdier-unknown-1845"]; !ok {
		t.Fatal("the child the hypothesis proposed is not in the corpus")
	}
	if children := after.Children("verdier-louis"); !slices.Contains(children, "verdier-unknown-1845") {
		t.Errorf("Louis's children are %v", children)
	}
	if got := after.Hypotheses["unrecorded-child-1845"].Status; got != model.Accepted {
		t.Errorf("the hypothesis stands at %s, want accepted", got)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("something was left unrecorded:\n%s", left)
	}
}
