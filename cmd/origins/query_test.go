package main

import (
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// theFixture is the fixture corpus, read where it was generated. The questions only
// read, so it is never copied.
func theFixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	return root
}

func slugsOf(path []api.Step) []string {
	out := make([]string, 0, len(path))
	for _, one := range path {
		out = append(out, one.Slug)
	}
	return out
}

// Lawrence's mother and Pauline's father were sister and brother, which is the whole of
// being first cousins, and the path says so a person at a time.
func TestKinshipNamesTheFixturesCousins(t *testing.T) {
	t.Parallel()
	root := theFixture(t)

	report := answering[api.Kin](t, "kinship", "-C", root, "ershov-lawrence", "person/verdier-pauline")
	if !report.Related || report.Relation != "first cousin" || report.CommonAncestor != "lang-anna-maria" {
		t.Errorf("got %+v", report)
	}
	want := []string{"ershov-lawrence", "verdier-louisa", "lang-anna-maria", "verdier-jean-baptiste", "verdier-pauline"}
	if got := slugsOf(report.Path); !slices.Equal(got, want) {
		t.Errorf("path %q, want %q", got, want)
	}

	code, stdout, stderr := invoke(t, "kinship", "-C", root, "ershov-lawrence", "verdier-pauline")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, says := range []string{
		"Lawrence Henry Ershov is the first cousin of Pauline Verdier",
		"verdier-louisa Louisa Verdier his mother",
		"lang-anna-maria Anna Maria Lang her mother",
		"verdier-jean-baptiste Jean Baptiste Verdier her son",
		"verdier-pauline Pauline Verdier his daughter",
	} {
		if !strings.Contains(squeezed(stdout), says) {
			t.Errorf("stdout does not say %q:\n%s", says, stdout)
		}
	}

	report = answering[api.Kin](t, "kinship", "-C", root, "ershov-lawrence", "galvao-joao")
	if report.Relation != "first cousin once removed" {
		t.Errorf("Lawrence is the %s of João", report.Relation)
	}
}

// Helena was brought into João's family, Henrique's stepdaughter and Pauline's adoptive
// daughter, which makes them brother and sister, and a path through a parent who is not
// one by birth says what kind of parent they are.
func TestKinshipSaysWhereItRunsThroughAnAdoption(t *testing.T) {
	t.Parallel()
	root := theFixture(t)
	report := answering[api.Kin](t, "kinship", "-C", root, "galvao-joao", "galvao-helena")
	if report.Relation != "brother" {
		t.Errorf("relation %q", report.Relation)
	}
	last := report.Path[len(report.Path)-1]
	if last.Slug != "galvao-helena" || last.Is != "step daughter" {
		t.Errorf("the path ends at %+v", last)
	}
}

// Two people with nobody in common is an answer, and most pairs in a real tree are.
func TestKinshipBetweenStrangersIsAnAnswer(t *testing.T) {
	t.Parallel()
	root := theFixture(t)
	code, stdout, stderr := invoke(t, "kinship", "-C", root, "verdier-jean-baptiste", "solveigarson-stefan")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "no ancestor in common") {
		t.Errorf("stdout = %q", stdout)
	}
	if report := answering[api.Kin](t, "kinship", "-C", root, "verdier-jean-baptiste", "solveigarson-stefan"); report.Related {
		t.Errorf("got %+v", report)
	}
}

func TestAncestorsAreListedAGenerationAtATime(t *testing.T) {
	t.Parallel()
	root := theFixture(t)
	report := answering[api.Lineage](t, "ancestors", "-C", root, "--depth", "2", "galvao-joao")
	if len(report.Generations) != 2 {
		t.Fatalf("got %d generations, asked for 2: %+v", len(report.Generations), report.Generations)
	}
	for i, want := range []struct {
		called string
		people []string
	}{
		{"parents", []string{"galvao-henrique", "verdier-pauline"}},
		{"grandparents", []string{"papadopoulou-maria", "verdier-jean-baptiste"}},
	} {
		got := report.Generations[i]
		var slugs []string
		for _, one := range got.People {
			slugs = append(slugs, one.Slug)
		}
		if got.Called != want.called || !slices.Equal(slugs, want.people) {
			t.Errorf("generation %d is %s %q, want %s %q", i+1, got.Called, slugs, want.called, want.people)
		}
	}

	all := answering[api.Lineage](t, "ancestors", "-C", root, "galvao-joao")
	if len(all.Generations) != 6 || all.Generations[5].Called != "great-great-great-great-grandparents" {
		t.Errorf("without --depth, got %+v", all.Generations)
	}
}

func TestDescendantsOfSomebodyWithNone(t *testing.T) {
	t.Parallel()
	root := theFixture(t)
	code, stdout, stderr := invoke(t, "descendants", "-C", root, "marchena-solveigarson-sasha")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "marchena-solveigarson-sasha has no descendants in this corpus") {
		t.Errorf("stdout = %q", stdout)
	}

	report := answering[api.Lineage](t, "descendants", "-C", root, "--depth", "1", "lang-anna-maria")
	if len(report.Generations) != 1 || report.Generations[0].Called != "children" ||
		len(report.Generations[0].People) != 3 {
		t.Errorf("got %+v", report.Generations)
	}
}

// Each of the four kinds of research the fixture has something of, and nothing that
// check would call an error, or anybody still living.
func TestTodoListsTheResearchInTheFixture(t *testing.T) {
	t.Parallel()
	root := theFixture(t)
	report := answering[api.ToDo](t, "todo", "-C", root)

	listed := map[string]bool{}
	for _, f := range report.Todo {
		listed[f.Rule+" "+f.Entity] = true
		if f.Severity != "warning" {
			t.Errorf("%s %s is listed as %s", f.Rule, f.Entity, f.Severity)
		}
	}
	for _, want := range []string{
		"open-hypothesis hypothesis/unrecorded-child-1845",
		"unrecorded-death person/ershov-lawrence",
		"unrecorded-death person/verdier-anna",
		"unsourced-event person/verdier-pauline",
		"task person/verdier-jean-baptiste",
	} {
		if !listed[want] {
			t.Errorf("todo does not list %s", want)
		}
	}
	for _, not := range []string{
		"open-hypothesis hypothesis/louisa-born-1844",
		"unrecorded-death person/solveigarson-stefan",
		"unrecorded-death person/verdier-jean-baptiste",
	} {
		if listed[not] {
			t.Errorf("todo lists %s", not)
		}
	}

	code, stdout, _ := invoke(t, "todo", "-C", root)
	if code != exitOK || !strings.Contains(stdout, "things to do") {
		t.Errorf("exit %d\n%s", code, stdout)
	}
}

// index lists every record for the pages, and says for a reader how many of each kind
// it lists, which is what the corpus holds.
func TestIndexListsEveryRecordAndCountsThem(t *testing.T) {
	t.Parallel()
	root := theFixture(t)
	report := answering[api.Index](t, "index", "-C", root)
	code, stdout, stderr := invoke(t, "index", "-C", root)
	if code != exitOK || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if want := countedAll(report.Counts()) + "\n"; stdout != want {
		t.Errorf("index says %q, and its JSON counts %q", stdout, want)
	}
	if report.Counts().People != 46 || report.Project.MainPerson != "verdier-jean-baptiste" {
		t.Errorf("the fixture's index is %+v", report.Counts())
	}
}

// --with answers as though the hypothesis were true: the father nobody has a record of
// is somebody once it is supposed, and the corpus still has no record of him after.
func TestTheQuestionsTakeAHypothesis(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	put(t, root, "hypotheses/there-was-an-antonio/README.md", thereWasAnAntonio(git(t, root, "rev-parse", "HEAD")))

	before := answering[api.Lineage](t, "ancestors", "-C", root, "rossi-giovanni")
	if got := before.Generations[0].People[0]; got.Name != "rossi-antonio" {
		t.Errorf("without the hypothesis, the father is %+v", got)
	}
	after := answering[api.Lineage](t, "ancestors", "-C", root, "--with", "there-was-an-antonio", "rossi-giovanni")
	if got := after.Generations[0].People[0]; got.Name != "Antonio Rossi" {
		t.Errorf("with it, the father is %+v", got)
	}
	if len(after.Read) != 1 || after.Read[0].Hypothesis != "there-was-an-antonio" {
		t.Errorf("read %+v", after.Read)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-antonio", "README.md")); !os.IsNotExist(err) {
		t.Errorf("supposing wrote the hypothesis into the corpus: %v", err)
	}
}

func TestTheQuestionsRefuseWhatTheyCannotAnswer(t *testing.T) {
	t.Parallel()
	root := theFixture(t)
	for _, args := range [][]string{
		{"kinship", "-C", root, "verdier-jean-baptiste"},
		{"kinship", "-C", root, "verdier-jean-baptiste", "place/firenze"},
		{"ancestors", "-C", root, "firenze"},
		{"ancestors", "-C", root, "nobody-at-all"},
		{"ancestors", "-C", root, "--depth", "-1", "verdier-jean-baptiste"},
		{"descendants", "-C", root},
		{"todo", "-C", root, "verdier-jean-baptiste"},
		{"index", "-C", root, "verdier-jean-baptiste"},
		{"ancestors", "-C", root, "verdier-jean-baptiste", "--depth", "2"},
	} {
		if code, _, stderr := invoke(t, args...); code != exitMisuse || stderr == "" {
			t.Errorf("%q: exit %d, stderr %q", args, code, stderr)
		}
	}
}
