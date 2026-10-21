package ops

import (
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// accepting proposes a hypothesis and accepts it, which is where one has to be before
// the corpus can be asked whether it still holds it.
func accepting(t *testing.T, root string, repo *history.Repo, h *model.Hypothesis) *store.Corpus {
	t.Helper()
	decision, err := Accept(repo, proposing(t, root, h), h.Slug)
	landing(t, decision, err)
	return reloaded(t, root)
}

func reflecting(t *testing.T, corpus *store.Corpus, slug model.Slug) *Reflection {
	t.Helper()
	reflection, err := Reflected(corpus, slug)
	if err != nil {
		t.Fatal(err)
	}
	return reflection
}

// A conclusion nobody has touched since it was accepted is one the corpus still holds,
// and one whose fact somebody changed afterwards is one it holds no longer, the file
// named.
func TestTheCorpusHoldsAnAcceptedHypothesisUntilItsFactIsChanged(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = accepting(t, corpus.Root, repo, aCorrection(base))

	if held := reflecting(t, corpus, "giovanni-died-1902"); !held.Holds || held.Drift != nil || held.Why != "" {
		t.Fatalf("straight after accepting it: %+v", held)
	}

	since(t, corpus, repo, model.Set{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		Path:   "events[1].date",
		Value:  store.Typed("1903"),
	})
	held := reflecting(t, reloaded(t, corpus.Root), "giovanni-died-1902")
	switch {
	case held.Holds:
		t.Fatal("the death was changed after it was accepted and the corpus still holds it")
	case held.Drift == nil || !slices.Equal(held.Drift.Files, []string{"people/rossi-giovanni/README.md"}):
		t.Errorf("the change is reported as %+v", held.Drift)
	}
}

// Whatever an accepted hypothesis did is what the corpus holds straight afterwards,
// whichever of the operations it did it with: a rename, a merge and a deletion the
// corpus has had are nothing left to carry, and a record it made still has everything
// it was made with.
func TestTheCorpusHoldsWhateverAnAcceptedHypothesisDid(t *testing.T) {
	t.Parallel()
	maria := model.EntityRef{Kind: model.KindPerson, Slug: "bianchi-maria"}
	for _, one := range []struct {
		op   string
		edit model.Edit
	}{
		{"create", model.Create{Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-antonio"},
			Value: map[string]any{"name": map[string]any{"given": "Antonio"}, "sex": "M"}}},
		{"delete", model.Delete{Entity: maria}},
		{"rename", model.Rename{Entity: maria, To: "bianchi-maria-1840"}},
		{"merge", model.Merge{From: model.EntityRef{Kind: model.KindPerson, Slug: "bianchi-maria-bis"},
			Into: maria}},
		{"split", model.Split{Entity: maria, Into: []model.EntityRef{
			maria, {Kind: model.KindPerson, Slug: "bianchi-maria-2"},
		}}},
	} {
		t.Run(one.op, func(t *testing.T) {
			t.Parallel()
			corpus, repo, _ := aLifeInTwoEvents(t)
			since(t, corpus, repo, model.Create{Entity: model.EntityRef{Kind: model.KindPerson, Slug: "bianchi-maria-bis"}})
			h := aCorrection(git(t, corpus.Root, "rev-parse", "HEAD"))
			h.Edits = []model.Edit{one.edit}
			corpus = accepting(t, corpus.Root, repo, h)

			if held := reflecting(t, corpus, h.Slug); !held.Holds {
				t.Errorf("straight after accepting it: %+v", held.Drift)
			}
		})
	}
}

// A record an accepted hypothesis made goes on being added to, and what is added is no
// reason to say the corpus has left the hypothesis behind. A fact it made the record
// with being changed is.
func TestARecordAHypothesisMadeIsHeldWhileWhatItWasMadeWithStands(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	antonio := model.EntityRef{Kind: model.KindPerson, Slug: "rossi-antonio"}
	h := aCorrection(base)
	h.Edits = []model.Edit{model.Create{Entity: antonio, Value: map[string]any{"name": map[string]any{"given": "Antonio"}, "sex": "M"}}}
	corpus = accepting(t, corpus.Root, repo, h)

	since(t, corpus, repo, model.Set{Entity: antonio, Path: "name.surname", Value: store.Typed("Rossi")})
	if held := reflecting(t, reloaded(t, corpus.Root), h.Slug); !held.Holds {
		t.Errorf("a surname added to him since: %+v", held.Drift)
	}

	since(t, corpus, repo, model.Set{Entity: antonio, Path: "sex", Value: store.Typed("F")})
	held := reflecting(t, reloaded(t, corpus.Root), h.Slug)
	if held.Holds || held.Drift == nil || !slices.Equal(held.Drift.Files, []string{"people/rossi-antonio/README.md"}) {
		t.Errorf("the sex it gave him changed since: %+v", held)
	}
}

// Where there is no state to read a hypothesis at there is nothing to say about whether
// the corpus holds it, and why is said instead.
func TestAnAcceptedHypothesisWithNoStateToBeReadAtSaysWhy(t *testing.T) {
	t.Parallel()
	corpus, _, _ := aLifeInTwoEvents(t)
	h := aCorrection("")
	h.Status = model.Accepted
	corpus = proposing(t, corpus.Root, h)

	held := reflecting(t, corpus, h.Slug)
	if held.Holds || held.Drift != nil || held.Why != "it pins no commit" {
		t.Errorf("got %+v", held)
	}
}

func TestOnlyAnAcceptedHypothesisIsAskedWhetherItIsHeld(t *testing.T) {
	t.Parallel()
	corpus, _, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, aCorrection(base))
	if _, err := Reflected(corpus, "giovanni-died-1902"); err == nil {
		t.Error("an open hypothesis was asked whether the corpus holds it")
	}
}
