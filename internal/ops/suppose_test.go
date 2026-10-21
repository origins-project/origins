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

// supposing puts the hypotheses in front of the corpus and asks the question. They
// are not written down first: where a hypothesis is kept is store's business and
// settled elsewhere, and what supposing one does with it is what is asked here.
func supposing(t *testing.T, corpus *store.Corpus, wondered ...*model.Hypothesis) *Trial {
	t.Helper()
	named := make([]model.Slug, len(wondered))
	for i, h := range wondered {
		corpus.Hypotheses[h.Slug] = h
		named[i] = h.Slug
	}
	trial, err := Suppose(corpus, named)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = trial.Close() })
	return trial
}

// aRenaming is a second reading of the same record, touching a different line of it,
// which is what two hypotheses at once have to be to be worth asking about.
func aRenaming(base string) *model.Hypothesis {
	return &model.Hypothesis{
		Slug:   "giovanni-was-a-rossini",
		Status: model.Open,
		Base:   base,
		Edits: []model.Edit{model.Set{
			Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
			Path:   "name.surname",
			Value:  "Rossini",
		}},
	}
}

func died(t *testing.T, trial *Trial) string {
	t.Helper()
	events := trial.Corpus.People["rossi-giovanni"].Events
	for _, event := range events {
		if event.Type == "death" {
			return event.Date.String()
		}
	}
	t.Fatalf("the trial corpus has no death among %v", events)
	return ""
}

// What --with is for, and the corpus it is asked about stays exactly where it was.
func TestSupposingAHypothesisPutsItInTheCorpusAndSaysWhereItWasRead(t *testing.T) {
	t.Parallel()
	corpus, _, base := aLifeInTwoEvents(t)
	where, _ := corpus.Path("rossi-giovanni")
	before := read(t, corpus.Root, where)

	trial := supposing(t, reloaded(t, corpus.Root), aCorrection(base))

	if got := died(t, trial); got != "1902" {
		t.Errorf("the death is %s in the trial corpus, want 1902", got)
	}
	if got := trial.Read[0]; !got.Pinned() || got.Base != base {
		t.Errorf("%v, want it read at %s", got, base)
	}
	if after := read(t, corpus.Root, where); after != before {
		t.Errorf("the corpus itself was changed:\n%s", after)
	}
}

// The reason the question goes through the base at all. A path is positional, so an
// edit written when the death was events[1] has to reach the death and not the baptism
// that has turned up since — and it has to arrive in the corpus as it now stands,
// baptism and all, because whether the tree holds together today is the whole question.
func TestAHypothesisIsCarriedOutWhereItWasWrittenAndBroughtForwardFromThere(t *testing.T) {
	t.Parallel()
	corpus, _, base := aLifeInTwoEvents(t)

	later := Begin(reloaded(t, corpus.Root))
	giovanni := later.Corpus().People["rossi-giovanni"]
	giovanni.Events = slices.Insert(giovanni.Events, 1,
		model.Event{Type: "baptism", Date: on(t, "1834")})
	writing(t, later)
	committed(t, corpus.Root, "The baptism turns up")

	trial := supposing(t, reloaded(t, corpus.Root), aCorrection(base))

	events := trial.Corpus.People["rossi-giovanni"].Events
	if len(events) != 3 || events[1].Type != "baptism" {
		t.Fatalf("the trial corpus is not the corpus as it now stands: %v", events)
	}
	if got := died(t, trial); got != "1902" {
		t.Errorf("the death is %s, want 1902 — the edit landed where it was not written", got)
	}
}

// Two readings of two facts hold together, and asking for both at once is how a
// researcher finds out. Each is carried onto the result of the one before it.
func TestTwoHypothesesAreBothCarriedOntoTheCorpus(t *testing.T) {
	t.Parallel()
	corpus, _, base := aLifeInTwoEvents(t)

	trial := supposing(t, reloaded(t, corpus.Root), aCorrection(base), aRenaming(base))

	if got := died(t, trial); got != "1902" {
		t.Errorf("the death is %s, want 1902", got)
	}
	if got := trial.Corpus.People["rossi-giovanni"].Name.PrimarySurname().Name; got != "Rossini" {
		t.Errorf("the surname is %s, want Rossini", got)
	}
	for _, read := range trial.Read {
		if !read.Pinned() {
			t.Errorf("%v, want it read at %s", read, base)
		}
	}
}

// Two readings of one fact are what the question was about, and the answer is that
// they cannot both be held. It names the file, because that is where a researcher
// goes to look at the two of them.
func TestAHypothesisSomebodyHasOverwrittenComesBackAsAConflict(t *testing.T) {
	t.Parallel()
	corpus, _, base := aLifeInTwoEvents(t)

	since := Begin(reloaded(t, corpus.Root))
	since.Corpus().People["rossi-giovanni"].Events[1].Date = on(t, "1903")
	writing(t, since)
	committed(t, corpus.Root, "The stone says 1903")

	held := reloaded(t, corpus.Root)
	held.Hypotheses["giovanni-died-1902"] = aCorrection(base)
	_, err := Suppose(held, []model.Slug{"giovanni-died-1902"})

	var drifted *Drifted
	if !errors.As(err, &drifted) {
		t.Fatalf("supposing a hypothesis nobody can hold any more gave %v", err)
	}
	where, _ := held.Path("rossi-giovanni")
	if !slices.Contains(drifted.Files, where) {
		t.Errorf("the conflict is in %v, want it to name %s", drifted.Files, where)
	}
}

// The three ways a hypothesis has no earlier state to be read at. None of them is a
// reason to decline the question — the corpus in front of the researcher is the only
// other state there is, and reading it there is an answer as long as it is said.
func TestAHypothesisWithNoEarlierStateIsReadAgainstTheCorpusAndSaysSo(t *testing.T) {
	t.Parallel()
	inNoRepository := func(t *testing.T) (*store.Corpus, string) {
		corpus := onDisk(t)
		tx := Begin(corpus)
		tx.Corpus().People["rossi-giovanni"].Events = []model.Event{
			{Type: "birth", Date: on(t, "1834")},
			{Type: "death", Date: on(t, "1901")},
		}
		writing(t, tx)
		return corpus, "8f78553"
	}
	pinningNothing := func(t *testing.T) (*store.Corpus, string) {
		corpus, _, _ := aLifeInTwoEvents(t)
		return corpus, ""
	}
	pinningACommitNobodyHas := func(t *testing.T) (*store.Corpus, string) {
		corpus, _, _ := aLifeInTwoEvents(t)
		return corpus, "8f785531f5e6b6de26c60b2f3a0e5f6cc0e5bc11"
	}

	for _, unreadable := range []struct {
		what string
		set  func(*testing.T) (*store.Corpus, string)
		says string
	}{
		{"a corpus in no repository", inNoRepository, "in no repository"},
		{"a hypothesis pinning no commit", pinningNothing, "pins no commit"},
		{"a base nobody has", pinningACommitNobodyHas, "does not have"},
		{"a base holding no corpus", olderThanTheCorpus, "holds no corpus"},
	} {
		t.Run(unreadable.what, func(t *testing.T) {
			corpus, base := unreadable.set(t)
			trial := supposing(t, reloaded(t, corpus.Root), aCorrection(base))

			read := trial.Read[0]
			if read.Pinned() {
				t.Fatalf("%v, want it read against the corpus", read)
			}
			if !strings.Contains(read.Why, unreadable.says) {
				t.Errorf("%v does not say %q", read, unreadable.says)
			}
			if got := died(t, trial); got != "1902" {
				t.Errorf("the death is %s, want 1902", got)
			}
		})
	}
}

// olderThanTheCorpus is the case the fixture is in: a repository somebody was already
// keeping before there was a corpus in it, so the commit a hypothesis names is a real
// commit whose tree has nothing origins can read.
func olderThanTheCorpus(t *testing.T) (*store.Corpus, string) {
	t.Helper()
	corpus := onDisk(t)

	git(t, corpus.Root, "init", "-q")
	git(t, corpus.Root, "config", "user.name", "Rosa Bianchi")
	git(t, corpus.Root, "config", "user.email", "rosa@example.invalid")

	notes := filepath.Join(corpus.Root, "to-do.md")
	if err := os.WriteFile(notes, []byte("Write to the parish at Vicchio.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, corpus.Root, "add", "--", "to-do.md")
	git(t, corpus.Root, "commit", "-q", "-m", "Before there was a corpus")
	before := git(t, corpus.Root, "rev-parse", "HEAD")

	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Events = []model.Event{
		{Type: "birth", Date: on(t, "1834")},
		{Type: "death", Date: on(t, "1901")},
	}
	writing(t, tx)
	committed(t, corpus.Root, "The corpus as it stood")
	return corpus, before
}

// check --with has to differ from check by the hypothesis and by nothing else, or the
// two would be answering about different corpora with only one of them saying so. A
// file somebody has edited and not recorded is part of what the corpus says today.
func TestWhatIsSupposedIsTheCorpusInFrontOfYouAndNotTheLastStateRecorded(t *testing.T) {
	t.Parallel()
	corpus, _, base := aLifeInTwoEvents(t)

	byHand := Begin(reloaded(t, corpus.Root))
	byHand.Corpus().People["bianchi-maria"].Body = "The register has her at Borgo.\n"
	writing(t, byHand)

	trial := supposing(t, reloaded(t, corpus.Root), aCorrection(base))

	if got := trial.Corpus.People["bianchi-maria"].Body; !strings.Contains(got, "Borgo") {
		t.Errorf("an unrecorded change is not in what was supposed: %q", got)
	}
	if got := died(t, trial); got != "1902" {
		t.Errorf("the death is %s, want 1902", got)
	}
}

// A trial is a checkout and a copy of a corpus, and both go when it does. Leaving
// either behind would have a researcher's disk fill up with corpora nobody asked for.
func TestClosingATrialLeavesNothingBehind(t *testing.T) {
	t.Parallel()
	corpus, _, base := aLifeInTwoEvents(t)

	held := reloaded(t, corpus.Root)
	held.Hypotheses["giovanni-died-1902"] = aCorrection(base)
	trial, err := Suppose(held, []model.Slug{"giovanni-died-1902"})
	if err != nil {
		t.Fatal(err)
	}
	if err := trial.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(trial.dir); !os.IsNotExist(err) {
		t.Errorf("%s is still there: %v", trial.dir, err)
	}
	listed := git(t, corpus.Root, "worktree", "list")
	if strings.Contains(listed, filepath.Base(trial.dir)) {
		t.Errorf("git still lists the checkout:\n%s", listed)
	}
}

// A name nobody wrote a hypothesis under is a mistake in the asking, and saying which
// name is the whole of what the reader needs.
func TestSupposingAHypothesisTheCorpusDoesNotHaveIsRefused(t *testing.T) {
	t.Parallel()
	corpus, _, _ := aLifeInTwoEvents(t)

	_, err := Suppose(reloaded(t, corpus.Root), []model.Slug{"giovanni-died-1902"})
	if err == nil {
		t.Fatal("a hypothesis nobody wrote was supposed anyway")
	}
	if !strings.Contains(err.Error(), "giovanni-died-1902") {
		t.Errorf("%v does not name the hypothesis", err)
	}
}
