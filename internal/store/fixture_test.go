package store

import (
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// The fixture corpus is generated rather than committed, so this skips when it is not
// there. Run tools/mkfixture.sh to produce it.
func fixture(t *testing.T) *Corpus {
	t.Helper()
	root := filepath.Join("..", "..", "test")
	if _, err := os.Stat(root); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	c, err := Load(root)
	if err != nil {
		t.Fatalf("the fixture should load without a single problem:\n%v", err)
	}
	return c
}

func TestFixtureLoads(t *testing.T) {
	c := fixture(t)
	for _, count := range []struct {
		what string
		got  int
		want int
	}{
		{"people", len(c.People), 46},
		{"families", len(c.Families), 25},
		{"places", len(c.Places), 27},
		{"sources", len(c.Sources), 17},
		{"repositories", len(c.Repositories), 11},
		{"hypotheses", len(c.Hypotheses), 2},
	} {
		if count.got != count.want {
			t.Errorf("%s = %d, want %d", count.what, count.got, count.want)
		}
	}
	if c.Project.Name != "Marchena Sólveigarson family" || c.Project.MainPerson != "verdier-jean-baptiste" {
		t.Errorf("project = %+v", c.Project)
	}
}

func TestFixtureDerivesJeanBaptistesFamily(t *testing.T) {
	c := fixture(t)

	assertSlugs := func(what string, got, want []model.Slug) {
		t.Helper()
		if !slices.Equal(got, want) {
			t.Errorf("%s = %q, want %q", what, got, want)
		}
	}

	assertSlugs("families", c.FamiliesOf("verdier-jean-baptiste"),
		[]model.Slug{"papadopoulou-maria--verdier-jean-baptiste"})
	assertSlugs("children", c.Children("verdier-jean-baptiste"),
		[]model.Slug{"verdier-anna", "verdier-pauline", "verdier-pierre"})

	full, half := c.Siblings("verdier-jean-baptiste")
	assertSlugs("full siblings", full, []model.Slug{"verdier-louisa"})
	assertSlugs("half siblings", half, []model.Slug{"ershov-michael"})

	// the grandson named for his half-brother's father, and the reason the grandson's
	// slug needed a discriminator
	if k, ok := c.Kinship("ershov-ivan", "ershov-ivan-1885"); !ok || k.Up != 0 || k.Down != 2 {
		t.Errorf("kinship = %+v, %v", k, ok)
	}
}

func TestFixtureKeepsTheAwkwardCases(t *testing.T) {
	c := fixture(t)

	// a gravestone giving a day and a month, and a remembered month with no year
	floating := 0
	for _, slug := range c.PeopleSlugs() {
		for _, event := range c.People[slug].Events {
			if !event.Date.IsZero() && !event.Date.Anchored() {
				floating++
			}
		}
	}
	if floating < 2 {
		t.Errorf("the fixture should carry floating dates, found %d", floating)
	}

	if sex := c.People["al-halabi-1889"].Sex; sex.Value != model.SexUnknown || sex.Qualified() {
		t.Errorf("an infant who died on the day of birth has no recorded sex, got %v", sex)
	}

	// two photographs, so which one is the portrait has to be said rather than guessed
	jean := c.People["verdier-jean-baptiste"]
	if jean.Portrait != (model.Image{File: "photos/1890-portrait.jpg"}) {
		t.Errorf("portrait = %+v", jean.Portrait)
	}
	if !slices.Contains(jean.Files, "photos/1875-carte-de-visite.jpg") {
		t.Errorf("the declared portrait should not be the only photograph: %v", jean.Files)
	}

	// four parents of three relation kinds, in two families, one of them a mother
	// whose partner nobody knows
	parents := c.ParentsOf("galvao-helena")
	kinds := map[model.Relation]int{}
	for _, parent := range parents {
		kinds[parent.Relation]++
	}
	if len(parents) != 4 || kinds[model.Birth] != 2 || kinds[model.Adoptive] != 1 || kinds[model.Step] != 1 {
		t.Errorf("parents = %+v", parents)
	}

	// an event, another name and a picture marked private, and the private person
	// named in an event and in a picture that are not
	if !c.People["marchena-pablo"].Events[1].Private || !c.People["galvao-claudia"].AlsoKnownAs[0].Private ||
		!c.People["marchena-marco"].Media[0].Private {
		t.Error("the fixture should mark an event, a name and a picture private")
	}
	witnessed := c.Families["marchena-marco--solveigarson-stefan"].Events[0]
	christening := c.People["marchena-lucas"].Media[0]
	if witnessed.Private || witnessed.With[0].Person != "sato-yumiko" ||
		christening.Private || christening.Depicts[1].Person != "sato-yumiko" {
		t.Errorf("Yumiko should witness a family and be in a picture, neither private: %+v, %+v", witnessed, christening)
	}

	// both levels of the edit vocabulary
	if _, ok := c.Hypotheses["louisa-born-1844"].Edits[0].(model.Set); !ok {
		t.Errorf("edit is %T", c.Hypotheses["louisa-born-1844"].Edits[0])
	}
	if _, ok := c.Hypotheses["unrecorded-child-1845"].Edits[0].(model.Create); !ok {
		t.Errorf("edit is %T", c.Hypotheses["unrecorded-child-1845"].Edits[0])
	}

	attached := 0
	for _, slug := range c.PeopleSlugs() {
		attached += len(c.People[slug].Files)
	}
	for _, slug := range c.SourceSlugs() {
		attached += len(c.Sources[slug].Files)
	}
	if attached != 10 {
		t.Errorf("attachments = %d, want 10", attached)
	}
}

func TestFixtureLoadsTheSameWayTwice(t *testing.T) {
	first, second := fixture(t), fixture(t)
	for _, slug := range first.PeopleSlugs() {
		a, b := first.People[slug], second.People[slug]
		if a.Body != b.Body || len(a.Events) != len(b.Events) {
			t.Errorf("%q loaded differently on the second pass", slug)
		}
	}
	if !slices.Equal(first.PeopleSlugs(), second.PeopleSlugs()) {
		t.Error("the order of people depended on the walk")
	}
}

// testgit is the fixture again, built through origins from nothing, and a test that
// needs a history reaches for it where one that needs only a corpus reaches for test.
// That is only sound while the two are one corpus, so they are held to each other file
// for file. The one thing that differs is where a hypothesis was written: test is in no
// repository, so its bases are illustrative, and testgit's name commits it has.
func TestTheGitFixtureIsTheFixture(t *testing.T) {
	built := filepath.Join("..", "..", "testgit")
	if _, err := os.Stat(built); err != nil {
		t.Skip("no git fixture; run tools/mkgitfixture.sh")
	}
	written, rebuilt := contents(t, filepath.Join("..", "..", "test")), contents(t, built)

	for _, name := range slices.Sorted(maps.Keys(written)) {
		again, ok := rebuilt[name]
		switch {
		case !ok:
			t.Errorf("%s is in test and not in testgit", name)
		case again != written[name]:
			t.Errorf("%s differs between test and testgit", name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(rebuilt)) {
		if _, ok := written[name]; !ok {
			t.Errorf("%s is in testgit and not in test", name)
		}
	}
	if t.Failed() {
		t.Log("regenerate both with tools/mkgitfixture.sh, which runs tools/mkfixture.sh first")
	}
}

var based = regexp.MustCompile(`(?m)^base: .*\n`)

// contents is every file of a corpus by the name the corpus gives it, with the base a
// hypothesis pins taken out and the repository around it left alone.
func contents(t *testing.T, root string) map[string]string {
	t.Helper()
	if _, err := os.Stat(root); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	found := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if strings.HasPrefix(name, "hypotheses/") {
			content = based.ReplaceAll(content, nil)
		}
		found[name] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return found
}
