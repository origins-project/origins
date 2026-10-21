package ops

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// onDisk lays a small corpus down and reads it back, which is the state every
// transaction starts from: a model, and a file each entity came out of.
func onDisk(t *testing.T) *store.Corpus {
	t.Helper()
	corpus := model.NewCorpus()
	corpus.Project = model.Project{
		Name: "Rossi family", MainPerson: "rossi-giovanni", FormatVersion: 1,
	}
	corpus.People["rossi-giovanni"] = &model.Person{
		Slug: "rossi-giovanni", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
	}
	corpus.People["bianchi-maria"] = &model.Person{
		Slug: "bianchi-maria", Name: model.PersonName{Given: "Maria", Surname: model.SurnameOf("Bianchi")}, Sex: model.StatedSex{Value: model.Female},
	}

	root := filepath.Join(t.TempDir(), "corpus")
	if err := store.Write(root, corpus); err != nil {
		t.Fatal(err)
	}
	loaded, problems := store.Load(root)
	if problems != nil {
		t.Fatalf("what was written did not read back:\n%v", problems)
	}
	return loaded
}

func read(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// Every write re-emits what it touches, so a change to a corpus in a format later than
// this origins writes would put it back into the earlier one without a word. It is
// refused before anything is emitted, naming both formats.
func TestACorpusInALaterFormatIsNotChanged(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	corpus.Project.FormatVersion = model.FormatVersion + 1
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Body = "Born in the Mugello."
	_, err := tx.Stage()
	if err == nil {
		t.Fatal("a corpus in a later format was staged")
	}
	for _, said := range []string{"format-version 2", "the 1 this origins"} {
		if !strings.Contains(err.Error(), said) {
			t.Errorf("the refusal does not say %q: %v", said, err)
		}
	}
	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the file was rewritten:\n%s", after)
	}
}

// Canonical serialisation is what makes git blame answer at the level of a fact, and it
// only holds if every file origins touches is left in the same form. A transaction is
// therefore not allowed to rewrite the files it did not change either — a change to one
// person that restated fifty others would bury the one line that matters.
func TestAChangeWritesOnlyTheFileItChanged(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	before := read(t, corpus.Root, "people/bianchi-maria/README.md")

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Body = "Born in the Mugello."

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if got := change.Written(); !slices.Equal(got, []string{"people/rossi-giovanni/README.md"}) {
		t.Fatalf("staged %v", got)
	}
	if got := change.Removed(); len(got) != 0 {
		t.Fatalf("would remove %v", got)
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}

	if after := read(t, corpus.Root, "people/bianchi-maria/README.md"); after != before {
		t.Errorf("an untouched person was rewritten:\n%s", after)
	}
	reloaded, problems := store.Load(corpus.Root)
	if problems != nil {
		t.Fatal(problems)
	}
	if body := reloaded.People["rossi-giovanni"].Body; body != "Born in the Mugello." {
		t.Errorf("body came back as %q", body)
	}
}

// A mutation that works out to what is already there is not a failure, but it is not
// history either, and a commit recording it would say a change was made when none was.
func TestAChangeThatAltersNothingIsEmpty(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Name = model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")} // what it already said

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if !change.Empty() {
		t.Fatalf("staged %v and %v", change.Written(), change.Removed())
	}
}

func TestANewEntityLandsWhereNewEntitiesGo(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	tx := Begin(corpus)
	tx.Corpus().Places["firenze"] = &model.Place{Slug: "firenze", Name: model.PlaceName{Text: "Firenze"}, Type: "city"}

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if got := change.Written(); !slices.Equal(got, []string{"places/firenze/README.md"}) {
		t.Fatalf("staged %v", got)
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}

	reloaded, problems := store.Load(corpus.Root)
	if problems != nil {
		t.Fatal(problems)
	}
	if place := reloaded.Places["firenze"]; place == nil || place.Name.Text != "Firenze" {
		t.Errorf("the new place did not read back: %+v", place)
	}
}

// An entity the model no longer holds has a file that no longer describes anything.
// Leaving it there would mean the next load found an entity origins thought it had
// removed.
func TestARemovedEntityTakesItsFileWithIt(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	tx := Begin(corpus)
	delete(tx.Corpus().People, "bianchi-maria")

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if got := change.Removed(); !slices.Equal(got, []string{"people/bianchi-maria/README.md"}) {
		t.Fatalf("would remove %v", got)
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(corpus.Root, "people", "bianchi-maria", "README.md")); !os.IsNotExist(err) {
		t.Errorf("the file is still there: %v", err)
	}
}

// Origins owns the format, so it must never be the thing that breaks it. The refusal
// happens before anything is written, because a corpus that has to be repaired after
// the fact is one a researcher has to notice first.
func TestAChangeThatWouldContradictTheRecordsIsRefused(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Sources = []model.CitationRef{{Source: "nobody-register", Anchor: "p-1"}}

	change, err := tx.Stage()
	if change != nil {
		t.Fatalf("staged %v", change.Written())
	}
	var contradiction *Contradiction
	if !errors.As(err, &contradiction) {
		t.Fatalf("refused with %v", err)
	}
	if len(contradiction.Findings) != 1 || contradiction.Findings[0].Rule != "dangling-source" {
		t.Errorf("found %+v", contradiction.Findings)
	}
	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the refused change was written anyway:\n%s", after)
	}
}

// A corpus with an error in it is often exactly the corpus being worked on, and the
// work is fixing it. A transaction answers for the contradictions it introduces and not
// for the ones it found.
func TestAnErrorAlreadyInTheCorpusDoesNotBlockAnUnrelatedChange(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	corpus.People["bianchi-maria"].Sources = []model.CitationRef{{Source: "nobody-register", Anchor: "p-1"}}
	if _, err := store.Format(corpus, true); err != nil {
		t.Fatal(err)
	}

	reloaded, problems := store.Load(corpus.Root)
	if problems != nil {
		t.Fatal(problems)
	}
	tx := Begin(reloaded)
	tx.Corpus().People["rossi-giovanni"].Body = "Unrelated to Maria's missing father."

	change, err := tx.Stage()
	if err != nil {
		t.Fatalf("an unrelated change was refused for someone else's error: %v", err)
	}
	if got := change.Written(); !slices.Equal(got, []string{"people/rossi-giovanni/README.md"}) {
		t.Fatalf("staged %v", got)
	}
}

// Half a change is the one outcome a transaction exists to prevent, and the one a
// researcher has no way to spot: the files that did land look perfectly correct.
func TestAFailedWritePutsBackWhatItFound(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	before := read(t, corpus.Root, "people/bianchi-maria/README.md")

	tx := Begin(corpus)
	tx.Corpus().People["bianchi-maria"].Body = "Written first, and then taken back."
	tx.Corpus().People["rossi-giovanni"].Body = "The write that fails."
	tx.Corpus().Hypotheses["maria-born-1846"] = &model.Hypothesis{
		Slug: "maria-born-1846", Status: "open", Base: "0000000", Confidence: "possible",
		Edits: []model.Edit{model.Set{
			Entity: model.EntityRef{Kind: model.KindPerson, Slug: "bianchi-maria"},
			Path:   "events[0].date", Value: "1846",
		}},
	}

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if got := change.Written(); !slices.Equal(got, []string{
		"hypotheses/maria-born-1846/README.md", "people/bianchi-maria/README.md", "people/rossi-giovanni/README.md",
	}) {
		t.Fatalf("staged %v", got)
	}

	// The last of the three becomes unwritable once the first two have landed, which is
	// a disk filling up or a file someone has open, arranged so a test can rely on it.
	giovanni := filepath.Join(corpus.Root, "people", "rossi-giovanni", "README.md")
	if err := os.Remove(giovanni); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(giovanni, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := change.Apply(); err == nil {
		t.Fatal("the write reported success")
	}
	if after := read(t, corpus.Root, "people/bianchi-maria/README.md"); after != before {
		t.Errorf("the first file was left in its new state:\n%s", after)
	}
	if _, err := os.Stat(filepath.Join(corpus.Root, "hypotheses")); !os.IsNotExist(err) {
		t.Errorf("the directory the change created is still there: %v", err)
	}
}

// Whatever a transaction writes, it writes canonically, so there is never a commit that
// fmt would immediately want to rewrite.
func TestWhatATransactionWritesIsAlreadyCanonical(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Body = "Born in the Mugello."
	tx.Corpus().Places["firenze"] = &model.Place{Slug: "firenze", Name: model.PlaceName{Text: "Firenze"}, Type: "city"}

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}

	reloaded, problems := store.Load(corpus.Root)
	if problems != nil {
		t.Fatal(problems)
	}
	changed, err := store.Format(reloaded, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) > 0 {
		t.Errorf("fmt would rewrite %v", changed)
	}
}

// A transaction expects a corpus that loaded cleanly, and this is what stands behind
// that expectation at the one point where breaking it would destroy something. An
// origins.yaml that did not decode leaves the project zero-valued, and emitting a zero
// project would not record a change to the header — it would delete it.
func TestAProjectThatDidNotDecodeIsNotWrittenOver(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	header := "project:\n  name: Rossi family\n  this-field-is-not-a-field: true\n"
	if err := os.WriteFile(filepath.Join(corpus.Root, "origins.yaml"), []byte(header), 0o644); err != nil {
		t.Fatal(err)
	}
	reloaded, problems := store.Load(corpus.Root)
	if problems == nil {
		t.Fatal("the unknown field should have been reported")
	}

	tx := Begin(reloaded)
	tx.Corpus().People["rossi-giovanni"].Body = "A change made while the header is unreadable."

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(change.Written(), "origins.yaml") {
		t.Fatalf("staged %v", change.Written())
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}
	if after := read(t, corpus.Root, "origins.yaml"); after != header {
		t.Errorf("the header was rewritten:\n%s", after)
	}
}
