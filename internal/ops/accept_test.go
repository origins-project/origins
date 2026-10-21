package ops

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// proposing writes a hypothesis into the corpus and records it, which is where one
// has to be before it can be settled. A hypothesis is an entity like any other, and
// what settling it changes is its own file.
func proposing(t *testing.T, root string, h *model.Hypothesis) *store.Corpus {
	t.Helper()
	tx := Begin(reloaded(t, root))
	tx.Corpus().Hypotheses[h.Slug] = h
	writing(t, tx)
	committed(t, root, "The question, written down")
	return reloaded(t, root)
}

// landing records a decision, with the test's own failure for one origins would not
// reach and for one it could not write down.
func landing(t *testing.T, decision *Decision, reached error) (*Change, string) {
	t.Helper()
	if reached != nil {
		t.Fatal(reached)
	}
	decision.Entry.Tool = "origins 0.2.0"
	change, sha, err := decision.Land()
	if err != nil {
		t.Fatal(err)
	}
	return change, sha
}

func trailer(t *testing.T, root, key string) string {
	t.Helper()
	return git(t, root, "log", "-1", "--format=%(trailers:key="+key+",valueonly)")
}

func stands(t *testing.T, root string, slug model.Slug) model.Status {
	t.Helper()
	held, ours := reloaded(t, root).Hypotheses[slug]
	if !ours {
		t.Fatalf("the corpus no longer has %s", slug)
	}
	return held.Status
}

// What accepting one comes to: the corpus says what the hypothesis proposed, the
// hypothesis says it was accepted, and one commit records both — naming the
// hypothesis, so that a blame on the line reaches the argument behind it.
func TestAcceptingAHypothesisTakesItIntoTheCorpusAndMarksItAccepted(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, aCorrection(base))

	decision, err := Accept(repo, corpus, "giovanni-died-1902")
	_, sha := landing(t, decision, err)

	if len(sha) != 40 {
		t.Fatalf("sha = %q", sha)
	}
	after := reloaded(t, corpus.Root)
	if got := after.People["rossi-giovanni"].Events[1].Date.String(); got != "1902" {
		t.Errorf("the death is %s, want 1902", got)
	}
	if got := after.Hypotheses["giovanni-died-1902"].Status; got != model.Accepted {
		t.Errorf("the hypothesis stands at %s, want accepted", got)
	}
	if count := git(t, corpus.Root, "rev-list", "--count", "HEAD"); count != "4" {
		t.Errorf("accepting it took %s commits in all", count)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("something was left unrecorded:\n%s", left)
	}
	for key, want := range map[string]string{
		"Origins-Hypothesis": "giovanni-died-1902",
		"Origins-Op":         "hypothesis.accept",
		"Origins-Event":      "death",
	} {
		if got := trailer(t, corpus.Root, key); got != want {
			t.Errorf("%s came back as %q, want %q", key, got, want)
		}
	}
}

// The whole of what base pinning is for, at the one moment it settles something
// irreversible. A correction to events[1] written before anybody recorded the
// baptism that now sits there has to reach the death, and has to reach it in the
// corpus as it now stands, baptism and all.
func TestAcceptingCarriesTheEditsOutWhereTheyWereWritten(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, aCorrection(base))

	later := Begin(reloaded(t, corpus.Root))
	giovanni := later.Corpus().People["rossi-giovanni"]
	giovanni.Events = slices.Insert(giovanni.Events, 1,
		model.Event{Type: "baptism", Date: on(t, "1834")})
	writing(t, later)
	committed(t, corpus.Root, "The baptism turns up")

	decision, err := Accept(repo, reloaded(t, corpus.Root), "giovanni-died-1902")
	landing(t, decision, err)

	events := reloaded(t, corpus.Root).People["rossi-giovanni"].Events
	if len(events) != 3 || events[1].Type != "baptism" {
		t.Fatalf("the corpus is not as it stood: %v", events)
	}
	if got := events[2].Date.String(); got != "1902" {
		t.Errorf("the death is %s, want 1902 — the edit landed where it was not written", got)
	}
	if got := events[1].Date.String(); got != "1834" {
		t.Errorf("the baptism is %s, want 1834 — the edit landed on it", got)
	}
}

// A hypothesis somebody has overtaken is why acceptance is a rebase rather than a
// replay, and the corpus has to come out of it exactly as it went in: the patch
// reaches the files before anything can be known about whether it applies.
func TestAHypothesisSomebodyHasOvertakenIsRefusedAndTheCorpusIsPutBack(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, aCorrection(base))

	otherwise := Begin(reloaded(t, corpus.Root))
	otherwise.Corpus().People["rossi-giovanni"].Events[1].Date = on(t, "1903")
	writing(t, otherwise)
	committed(t, corpus.Root, "The parish register says 1903")
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	_, err := Accept(repo, reloaded(t, corpus.Root), "giovanni-died-1902")
	var drifted *Drifted
	if !errors.As(err, &drifted) {
		t.Fatalf("accepting an overtaken hypothesis came back as %v", err)
	}
	if !slices.Equal(drifted.Files, []string{"people/rossi-giovanni/README.md"}) {
		t.Errorf("the conflict is in %v", drifted.Files)
	}
	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the conflict was left in the corpus:\n%s", after)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("the corpus was not put back:\n%s", left)
	}
	if got := stands(t, corpus.Root, "giovanni-died-1902"); got != model.Open {
		t.Errorf("the hypothesis stands at %s, want open", got)
	}
}

// Where supposing falls back, accepting refuses. Both readings produce a corpus and
// only one of them is a change to the standing conclusion, so the second-best answer
// is worth having to a question and not to that.
func TestAcceptingRefusesAHypothesisWithNoStateToBeReadAt(t *testing.T) {
	t.Parallel()
	for _, this := range []struct {
		named string
		base  string
	}{
		{"it pins no commit", ""},
		{"a commit nobody has", strings.Repeat("0", 40)},
	} {
		t.Run(this.named, func(t *testing.T) {
			corpus, repo, base := aLifeInTwoEvents(t)
			h := aCorrection(base)
			h.Base = this.base
			corpus = proposing(t, corpus.Root, h)

			_, err := Accept(repo, corpus, h.Slug)
			var elsewhere *Unpinned
			if !errors.As(err, &elsewhere) {
				t.Fatalf("accepting it came back as %v", err)
			}
			if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
				t.Errorf("the corpus was written to anyway:\n%s", left)
			}
		})
	}
}

// Carrying a patch writes over the files and putting it back writes over them again,
// so an unrecorded edit is the work most easily lost here. It is also what would make
// the acceptance unreadable afterwards, the commit being everything the corpus
// differs from the last one in.
func TestAcceptingRefusesUntilWhatIsInTheCorpusHasBeenRecorded(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, aCorrection(base))

	byHand := Begin(reloaded(t, corpus.Root))
	byHand.Corpus().People["bianchi-maria"].Body = "Her name is spelled Bianco in the register."
	writing(t, byHand)
	before := read(t, corpus.Root, "people/bianchi-maria/README.md")

	_, err := Accept(repo, reloaded(t, corpus.Root), "giovanni-died-1902")
	if err == nil || !strings.Contains(err.Error(), "origins adopt") {
		t.Fatalf("accepting into an unrecorded corpus came back as %v", err)
	}
	if after := read(t, corpus.Root, "people/bianchi-maria/README.md"); after != before {
		t.Errorf("the unrecorded edit was written over:\n%s", after)
	}
}

// A change to the corpus is held to the contradictions it introduces wherever it
// came from, and this one came from a patch that is already in the files — so the
// refusal has to take them back as well.
func TestAHypothesisThatWouldMakeTheCorpusContradictItselfIsRefusedAndPutBack(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, &model.Hypothesis{
		Slug:   "giovanni-was-a-ricci",
		Status: model.Open,
		Base:   base,
		Edits: []model.Edit{model.Set{
			Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
			Path:   "sources",
			Value:  []any{"ricci-register#p-1"},
		}},
	})
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	decision, err := Accept(repo, corpus, "giovanni-was-a-ricci")
	if err != nil {
		t.Fatal(err)
	}
	decision.Entry.Tool = "origins 0.2.0"
	_, _, err = decision.Land()
	var contradiction *Contradiction
	if !errors.As(err, &contradiction) {
		t.Fatalf("a citation of no source came back as %v", err)
	}
	if err := decision.Discard(); err != nil {
		t.Fatal(err)
	}
	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the refused change was left in the corpus:\n%s", after)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("the corpus was not put back:\n%s", left)
	}
}

// A rehearsal is the whole of accepting and then none of it, which is the only
// honest way to say what accepting would do: the patch has to reach the files before
// anything can be said about what it leaves.
func TestARehearsalOfAcceptingLeavesTheCorpusAsItWas(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, aCorrection(base))
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	decision, err := Accept(repo, corpus, "giovanni-died-1902")
	if err != nil {
		t.Fatal(err)
	}
	change, err := decision.Tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(decision.Files, "people/rossi-giovanni/README.md") {
		t.Errorf("the rehearsal says it would change %v", decision.Files)
	}
	if got := change.Written(); !slices.Contains(got, "hypotheses/giovanni-died-1902/README.md") {
		t.Errorf("the hypothesis would not be marked: %v", got)
	}
	if err := decision.Discard(); err != nil {
		t.Fatal(err)
	}

	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the rehearsal was left in the corpus:\n%s", after)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("the corpus was not put back:\n%s", left)
	}
	if count := git(t, corpus.Root, "rev-list", "--count", "HEAD"); count != "3" {
		t.Errorf("the rehearsal recorded something: %s commits", count)
	}
}

// A hypothesis that creates a record is the case where the patch carries a whole
// file rather than a line of one, and where the commit has to name a record that was
// in no corpus when the question was asked.
func TestAcceptingAHypothesisThatCreatesARecordBringsTheRecordIn(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, &model.Hypothesis{
		Slug:   "there-was-an-antonio",
		Status: model.Open,
		Base:   base,
		Edits: []model.Edit{model.Create{
			Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-antonio"},
			Value:  map[string]any{"name": "Antonio Rossi", "sex": "M"},
		}},
	})

	decision, err := Accept(repo, corpus, "there-was-an-antonio")
	_, sha := landing(t, decision, err)

	if reloaded(t, corpus.Root).People["rossi-antonio"] == nil {
		t.Fatal("the record the hypothesis creates is not in the corpus")
	}
	files := git(t, corpus.Root, "show", "--name-only", "--format=", sha)
	for _, want := range []string{"people/rossi-antonio/README.md", "hypotheses/there-was-an-antonio/README.md"} {
		if !strings.Contains(files, want) {
			t.Errorf("the commit does not carry %s:\n%s", want, files)
		}
	}
	if got := trailer(t, corpus.Root, "Origins-Subject"); !strings.Contains(got, "person/rossi-antonio") {
		t.Errorf("the commit is about %q", got)
	}
}

// A hypothesis whose edits were already true where they were written proposes
// nothing, and settling it is settling it: the question was put and answered, and
// the only thing left to record is which way.
func TestAcceptingAHypothesisThatChangesNothingStillSettlesIt(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, &model.Hypothesis{
		Slug:   "he-was-always-a-rossi",
		Status: model.Open,
		Base:   base,
		Edits: []model.Edit{model.Set{
			Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
			Path:   "name.surname",
			Value:  "Rossi", // what the record said all along
		}},
	})
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	decision, err := Accept(repo, corpus, "he-was-always-a-rossi")
	change, _ := landing(t, decision, err)

	if got := change.Written(); !slices.Equal(got, []string{"hypotheses/he-was-always-a-rossi/README.md"}) {
		t.Errorf("accepting it wrote %v", got)
	}
	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the record was changed:\n%s", after)
	}
	if got := stands(t, corpus.Root, "he-was-always-a-rossi"); got != model.Accepted {
		t.Errorf("the hypothesis stands at %s, want accepted", got)
	}
}

// Rejecting one writes nothing back, the corpus already holding the answer, and
// records that the question was put. What it rests on is what the hypothesis
// contradicts, which is how whence reaches an argument that was turned down.
func TestRejectingWritesNothingBackAndRecordsWhatSankIt(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	h := aCorrection(base)
	h.Contradicts = []model.CitationRef{{Source: "parish-san-lorenzo", Anchor: "1901-014"}}
	corpus = proposing(t, corpus.Root, h)
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	decision, err := Reject(repo, corpus, "giovanni-died-1902")
	change, sha := landing(t, decision, err)

	if got := change.Written(); !slices.Equal(got, []string{"hypotheses/giovanni-died-1902/README.md"}) {
		t.Errorf("rejecting it wrote %v", got)
	}
	if after := read(t, corpus.Root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the record was changed:\n%s", after)
	}
	if got := stands(t, corpus.Root, "giovanni-died-1902"); got != model.Rejected {
		t.Errorf("the hypothesis stands at %s, want rejected", got)
	}
	if len(sha) != 40 {
		t.Fatalf("sha = %q", sha)
	}
	for key, want := range map[string]string{
		"Origins-Op":         "hypothesis.reject",
		"Origins-Hypothesis": "giovanni-died-1902",
		"Origins-Source":     "parish-san-lorenzo#1901-014",
	} {
		if got := trailer(t, corpus.Root, key); got != want {
			t.Errorf("%s came back as %q, want %q", key, got, want)
		}
	}
}

// Git's author is whoever settled a hypothesis, so who proposed it is said in a trailer
// of its own, either way it was settled, and only where the researcher did not.
func TestSettlingAMachineProposalSaysWhoProposedIt(t *testing.T) {
	t.Parallel()
	for _, answer := range []model.Status{model.Accepted, model.Rejected} {
		for _, proposer := range []string{"", "transcriber"} {
			corpus, repo, base := aLifeInTwoEvents(t)
			h := aCorrection(base)
			h.ProposedBy = proposer
			corpus = proposing(t, corpus.Root, h)

			settle := Accept
			if answer == model.Rejected {
				settle = Reject
			}
			decision, err := settle(repo, corpus, h.Slug)
			landing(t, decision, err)

			if got := trailer(t, corpus.Root, "Origins-Proposed-By"); got != proposer {
				t.Errorf("%s, proposed by %q: the commit says %q", answer, proposer, got)
			}
		}
	}
}

// Both answers refuse a hypothesis the corpus has already taken in: accepting it
// again would carry the patch onto a corpus that holds it, and rejecting it would
// leave the argument and the records disagreeing with nothing to say which is wrong.
func TestAHypothesisAlreadyAcceptedIsRefusedBothWays(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	corpus = proposing(t, corpus.Root, aCorrection(base))
	decision, err := Accept(repo, corpus, "giovanni-died-1902")
	landing(t, decision, err)

	if _, err := Accept(repo, reloaded(t, corpus.Root), "giovanni-died-1902"); err == nil {
		t.Error("it was accepted a second time")
	}
	if _, err := Reject(repo, reloaded(t, corpus.Root), "giovanni-died-1902"); err == nil {
		t.Error("what the corpus holds was rejected")
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("the corpus was written to anyway:\n%s", left)
	}
}

func TestSettlingAHypothesisTheCorpusDoesNotHaveIsRefused(t *testing.T) {
	t.Parallel()
	corpus, repo, _ := aLifeInTwoEvents(t)
	if _, err := Accept(repo, corpus, "no-such-question"); err == nil {
		t.Error("accepting a hypothesis nobody wrote reported success")
	}
	if _, err := Reject(repo, corpus, "no-such-question"); err == nil {
		t.Error("rejecting a hypothesis nobody wrote reported success")
	}
}
