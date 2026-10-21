package ops

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

func on(t *testing.T, text string) model.Date {
	t.Helper()
	date, err := model.ParseDate(text)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

// writing puts a change on the disk without asking whether it is allowed, which is
// what setting a corpus up is: the state a test starts from is a given and not a
// change anybody is making.
func writing(t *testing.T, tx *Tx) {
	t.Helper()
	change, err := tx.staged()
	if err != nil {
		t.Fatal(err)
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}
}

func committed(t *testing.T, root, subject string) string {
	t.Helper()
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", subject)
	return git(t, root, "rev-parse", "HEAD")
}

func reloaded(t *testing.T, root string) *store.Corpus {
	t.Helper()
	corpus, problems := store.Load(root)
	if problems != nil {
		t.Fatal(problems)
	}
	return corpus
}

// aLifeInTwoEvents is a man born and dead and nothing in between, recorded, with the
// commit to pin a hypothesis to.
func aLifeInTwoEvents(t *testing.T) (*store.Corpus, *history.Repo, string) {
	t.Helper()
	corpus, repo := inARepo(t)
	tx := Begin(corpus)
	tx.Corpus().People["rossi-giovanni"].Events = []model.Event{
		{Type: "birth", Date: on(t, "1834")},
		{Type: "death", Date: on(t, "1901")},
	}
	writing(t, tx)
	return corpus, repo, committed(t, corpus.Root, "What the registers say")
}

// aCorrection is the hypothesis those two events are there for: the death was in 1902,
// and events[1] is where the death was when somebody wrote that down.
func aCorrection(base string) *model.Hypothesis {
	return &model.Hypothesis{
		Slug:   "giovanni-died-1902",
		Status: model.Open,
		Base:   base,
		Edits: []model.Edit{model.Set{
			Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
			Path:   "events[1].date",
			Value:  "1902",
		}},
	}
}

// pinning is Pin with the test's own failure for a hypothesis that would not pin.
func pinning(t *testing.T, onto *store.Corpus, repo *history.Repo, h *model.Hypothesis) *Pinned {
	t.Helper()
	pinned, err := Pin(repo, h, onto)
	if err != nil {
		t.Fatal(err)
	}
	return pinned
}

// changed reads the lines a patch puts in and takes out, which is what a test about a
// patch is actually asking after. Everything else in it is context, and context is
// what the three-way merge reads rather than anybody here.
func changed(patch []byte) (added, taken []string) {
	for _, line := range strings.Split(string(patch), "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
		case strings.HasPrefix(line, "+"):
			added = append(added, strings.TrimSpace(line[1:]))
		case strings.HasPrefix(line, "-"):
			taken = append(taken, strings.TrimSpace(line[1:]))
		}
	}
	return added, taken
}

// The whole of what base pinning is for. A path is positional, so events[1] is the
// death on the day the hypothesis is written and the baptism on the day somebody finds
// the baptism; an edit read against the corpus as it now stands would move the wrong
// date and look exactly like an edit that worked.
func TestAnEditIsCarriedOutWhereItWasWrittenAndNotWhereTheCorpusHasGot(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)

	later := Begin(reloaded(t, corpus.Root))
	giovanni := later.Corpus().People["rossi-giovanni"]
	giovanni.Events = slices.Insert(giovanni.Events, 1,
		model.Event{Type: "baptism", Date: on(t, "1834")})
	writing(t, later)
	committed(t, corpus.Root, "The baptism, from the parish register")

	pinned := pinning(t, corpus, repo, aCorrection(base))
	if pinned.Base != base {
		t.Errorf("pinned to %s, not %s", pinned.Base, base)
	}
	if want := []string{"people/rossi-giovanni/README.md"}; !slices.Equal(pinned.Files, want) {
		t.Errorf("touches %v, want %v", pinned.Files, want)
	}
	added, taken := changed(pinned.Patch)
	if !slices.Equal(added, []string{"date: 1902"}) || !slices.Equal(taken, []string{"date: 1901"}) {
		t.Errorf("the patch puts in %v and takes out %v, and the death was in 1901:\n%s",
			added, taken, pinned.Patch)
	}
}

// The base is what the commit said. A corpus somebody is halfway through editing is
// not a state any hypothesis was written against, and reading the edits there would
// make what a hypothesis says depend on what is open in somebody's editor.
func TestTheBaseIsWhatTheCommitSaidAndNotWhatTheFilesSay(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)

	uncommitted := Begin(reloaded(t, corpus.Root))
	uncommitted.Corpus().People["rossi-giovanni"].Events[1].Date = on(t, "1899")
	writing(t, uncommitted)

	added, taken := changed(pinning(t, corpus, repo, aCorrection(base)).Patch)
	if !slices.Equal(added, []string{"date: 1902"}) || !slices.Equal(taken, []string{"date: 1901"}) {
		t.Errorf("the patch puts in %v and takes out %v, and the commit said 1901", added, taken)
	}
}

// A hypothesis proposing what the corpus already said is not a failure to pin. It is a
// hypothesis somebody has been overtaken on, and the patch saying nothing is how that
// comes out.
func TestAHypothesisTheCorpusAlreadyAgreesWithChangesNothing(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)

	h := aCorrection(base)
	h.Edits = []model.Edit{model.Set{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		Path:   "events[1].date",
		Value:  "1901",
	}}

	pinned := pinning(t, corpus, repo, h)
	if len(pinned.Patch) != 0 || len(pinned.Files) != 0 {
		t.Errorf("touches %v:\n%s", pinned.Files, pinned.Patch)
	}
}

// Pinning describes and does not judge. Whether a corpus still holds together with a
// hypothesis in it is the question --with exists to put, and a hypothesis that breaks
// something is that question's answer rather than a failure to arrive at one; refusing
// to pin it would leave a researcher with a hypothesis origins declines to show them.
func TestAHypothesisThatWouldContradictTheRecordsStillPins(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)

	h := aCorrection(base)
	h.Slug = "giovanni-was-a-son-of-the-parish"
	h.Edits = []model.Edit{model.Add{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		Path:   "sources",
		Value:  "nobody-register#p-1",
	}}

	added, _ := changed(pinning(t, corpus, repo, h).Patch)
	if !slices.Equal(added, []string{"sources: [nobody-register#p-1]"}) {
		t.Errorf("the patch puts in %v, and a citation of no source is a finding and not a refusal", added)
	}
}

// What is attached to a record merged away goes with it, and it has to go with it
// through the patch as well — a photograph left behind in the carrying is one that
// arrives nowhere.
func TestWhatAMergeCarriesIsInThePatch(t *testing.T) {
	t.Parallel()
	corpus, repo := inARepo(t)
	sparse := filepath.Join(corpus.Root, "people", "rossi-giovanni-nonno")
	if err := os.MkdirAll(filepath.Join(sparse, "photos"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"README.md":            "---\nname:\n  given: Giovanni\n  surname: Rossi\nsex: M\n---\n",
		"photos/passaporto.md": "a scan of a photograph",
	} {
		if err := os.WriteFile(filepath.Join(sparse, filepath.FromSlash(name)),
			[]byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := committed(t, corpus.Root, "The same man, written down twice")

	pinned := pinning(t, corpus, repo, &model.Hypothesis{
		Slug:   "the-two-giovannis-are-one",
		Status: model.Open,
		Base:   base,
		Edits: []model.Edit{model.Merge{
			From: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni-nonno"},
			Into: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		}},
	})

	patch := string(pinned.Patch)
	for _, line := range []string{
		"rename from people/rossi-giovanni-nonno/photos/passaporto.md",
		"rename to people/rossi-giovanni/photos/passaporto.md",
	} {
		if !strings.Contains(patch, line) {
			t.Errorf("the patch does not say %q:\n%s", line, patch)
		}
	}
}

// A corpus is the thing, and where in a repository somebody keeps it is incidental. A
// patch naming the repository's paths would be one that only applies to a corpus
// filed the same distance down.
func TestAPatchNamesTheCorpusPathsAndNotTheRepositorys(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	if _, err := os.Stat(filepath.Join(corpus.Root, ".git")); err != nil {
		t.Fatalf("this test wants the corpus to be the repository root: %v", err)
	}

	atTheRoot := pinning(t, corpus, repo, aCorrection(base))
	inside, moved := deeper(t, corpus)
	within := pinning(t, reloaded(t, filepath.Join(corpus.Root, "tree")), inside, aCorrection(moved))

	if string(atTheRoot.Patch) != string(within.Patch) {
		t.Errorf("a corpus a directory down gets a different patch:\n%s\n%s",
			atTheRoot.Patch, within.Patch)
	}
	if !slices.Equal(within.Files, []string{"people/rossi-giovanni/README.md"}) {
		t.Errorf("touches %v", within.Files)
	}
}

// deeper moves a corpus down into a subdirectory of its own repository and hands back
// the repository as origins opens it there, with the commit that put it there.
func deeper(t *testing.T, corpus *store.Corpus) (*history.Repo, string) {
	t.Helper()
	within := filepath.Join(corpus.Root, "tree")
	if err := os.Mkdir(within, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"people", store.ProjectFile} {
		if err := os.Rename(filepath.Join(corpus.Root, name),
			filepath.Join(within, name)); err != nil {
			t.Fatal(err)
		}
	}
	moved := committed(t, corpus.Root, "Keep the corpus in a directory of its own")

	repo, err := history.Open(within)
	if err != nil {
		t.Fatal(err)
	}
	return repo, moved
}

// The reconstruction is a checkout, and a checkout somebody has to know about is one
// git will go on mentioning. What the corpus in front of the caller says is not part
// of the question either, so nothing of it may move.
func TestNothingOfTheReconstructionIsLeftBehind(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	before := read(t, corpus.Root, "people/rossi-giovanni/README.md")

	pinning(t, corpus, repo, aCorrection(base))

	if worktrees := git(t, corpus.Root, "worktree", "list"); strings.Count(worktrees, "\n") != 0 {
		t.Errorf("a checkout is still registered:\n%s", worktrees)
	}
	if dirty := git(t, corpus.Root, "status", "--porcelain"); dirty != "" {
		t.Errorf("the corpus was written to:\n%s", dirty)
	}
	if now := read(t, corpus.Root, "people/rossi-giovanni/README.md"); now != before {
		t.Errorf("the corpus was changed:\n%s", now)
	}
}

// The three ways a hypothesis can be one nothing is to be made of, each answered by
// naming it — a hypothesis is a file somebody wrote, and which file is the first thing
// they need to hear.
func TestAHypothesisThatCannotBeReadWhereItWasWrittenIsRefused(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	unpinned, elsewhere, wrong := aCorrection(base), aCorrection("8f78553"), aCorrection(base)
	unpinned.Base = ""
	wrong.Edits = []model.Edit{model.Set{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-nobody"},
		Path:   "name.surname",
		Value:  "Rossi",
	}}

	for _, refused := range []struct {
		h    *model.Hypothesis
		says string
	}{
		{unpinned, "pins no commit"},
		{elsewhere, "this repository does not have"},
		{wrong, "rossi-nobody"},
	} {
		pinned, err := Pin(repo, refused.h, corpus)
		switch {
		case err == nil:
			t.Errorf("%s pinned to %q: %v", refused.h.Slug, refused.h.Base, pinned)
		case !strings.Contains(err.Error(), refused.says):
			t.Errorf("%v does not say %q", err, refused.says)
		case !strings.Contains(err.Error(), string(refused.h.Slug)):
			t.Errorf("%v does not name the hypothesis", err)
		}
	}
}

// The two kinds of refusal are not the same refusal, and the difference is the whole
// of what lets check --with answer a question accepting would decline. A repository
// without the commit is a fact about the repository; an edit that will not carry out
// where it was written is wrong wherever it is read.
func TestAHypothesisWithNoEarlierStateIsToldApartFromOneThatIsWrong(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	wrong := aCorrection(base)
	wrong.Edits = []model.Edit{model.Set{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-nobody"},
		Path:   "name.surname",
		Value:  "Rossi",
	}}

	for _, refused := range []struct {
		h       *model.Hypothesis
		unknown bool
	}{
		{aCorrection(""), true},
		{aCorrection("8f78553"), true},
		{wrong, false},
	} {
		_, err := Pin(repo, refused.h, corpus)
		var elsewhere *Unpinned
		if errors.As(err, &elsewhere) != refused.unknown {
			t.Errorf("%v: read at an earlier state %v, want %v",
				err, !refused.unknown, refused.unknown)
		}
	}
}

// since changes the corpus with one edit and records it as origins would, trailers
// and all, which is what the history is read by.
func since(t *testing.T, corpus *store.Corpus, repo *history.Repo, edit model.Edit) {
	t.Helper()
	later := Begin(reloaded(t, corpus.Root))
	if _, deleting := edit.(model.Delete); deleting {
		// the project names a main person, and a corpus naming nobody is refused
		later.Corpus().Project.MainPerson = "bianchi-maria"
	}
	entry, err := later.Perform(edit)
	if err != nil {
		t.Fatal(err)
	}
	entry.Tool = "origins 0.2.0"
	if _, _, err := later.Commit(repo, entry); err != nil {
		t.Fatal(err)
	}
}

// A record renamed since is the same record under another name, and the history says
// which. The hypothesis still calls it what it was called when it was written, and
// what it says about it lands on the record as it is called now.
func TestAHypothesisFollowsARecordRenamedSince(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	since(t, corpus, repo, model.Rename{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		To:     "rossi-giovanni-battista",
	})

	pinned := pinning(t, reloaded(t, corpus.Root), repo, aCorrection(base))
	if want := []string{"people/rossi-giovanni-battista/README.md"}; !slices.Equal(pinned.Files, want) {
		t.Errorf("touches %v, want %v", pinned.Files, want)
	}
	added, taken := changed(pinned.Patch)
	if !slices.Equal(added, []string{"date: 1902"}) || !slices.Equal(taken, []string{"date: 1901"}) {
		t.Errorf("the patch puts in %v and takes out %v:\n%s", added, taken, pinned.Patch)
	}
	if want := []model.EntityRef{{Kind: model.KindPerson, Slug: "rossi-giovanni-battista"}}; !slices.Equal(
		pinned.Entities, want) {
		t.Errorf("the edits are about %v, want %v", pinned.Entities, want)
	}
}

// A name an edit writes as a value is the name the record had at the base too, and it
// is followed the same way: the parent a hypothesis gives somebody is the man it meant
// and not whoever has the name now.
func TestANameAnEditWritesIsFollowedToo(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	since(t, corpus, repo, model.Rename{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		To:     "rossi-giovanni-battista",
	})
	h := aCorrection(base)
	h.Edits = []model.Edit{model.Add{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "bianchi-maria"},
		Path:   "events",
		Value:  store.Typed("{type: christening, with: [{person: rossi-giovanni, role: godfather}]}"),
	}}

	added, _ := changed(pinning(t, reloaded(t, corpus.Root), repo, h).Patch)
	if !slices.Contains(added, "- person: rossi-giovanni-battista") {
		t.Errorf("the patch puts in %v, want the godfather under the name he has now", added)
	}
}

// Two records that swapped names since cannot be renamed one after the other, and
// each is still found under the name it has now.
func TestRecordsThatSwappedNamesAreEachFollowed(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	giovanni := model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"}
	maria := model.EntityRef{Kind: model.KindPerson, Slug: "bianchi-maria"}
	since(t, corpus, repo, model.Rename{Entity: giovanni, To: "rossi-aside"})
	since(t, corpus, repo, model.Rename{Entity: maria, To: "rossi-giovanni"})
	since(t, corpus, repo, model.Rename{Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-aside"},
		To: "bianchi-maria"})

	pinned := pinning(t, reloaded(t, corpus.Root), repo, aCorrection(base))
	if want := []string{"people/bianchi-maria/README.md"}; !slices.Equal(pinned.Files, want) {
		t.Errorf("touches %v, want %v", pinned.Files, want)
	}
}

// A record deleted since has nothing an edit could be carried onto, and one merged
// since is part of another whose facts the edit was not written about. Both are drift,
// and a merge names the record it went into, that being where to look.
func TestAHypothesisAboutARecordGoneSinceHasDrifted(t *testing.T) {
	t.Parallel()
	giovanni := model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"}
	for _, one := range []struct {
		why   string
		edit  model.Edit
		names string
	}{
		{"deleted", model.Delete{Entity: giovanni},
			"person/rossi-giovanni, and nothing in this corpus goes by that name any more"},
		{"merged", model.Merge{From: giovanni, Into: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni-2"}},
			"person/rossi-giovanni, which has since been merged into person/rossi-giovanni-2"},
	} {
		t.Run(one.why, func(t *testing.T) {
			t.Parallel()
			corpus, repo, base := aLifeInTwoEvents(t)
			if _, merging := one.edit.(model.Merge); merging {
				since(t, corpus, repo, model.Create{
					Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni-2"}})
			}
			since(t, corpus, repo, one.edit)

			_, err := Pin(repo, aCorrection(base), reloaded(t, corpus.Root))
			var drifted *Drifted
			switch {
			case !errors.As(err, &drifted):
				t.Fatalf("%v, want it to have drifted", err)
			case !slices.Equal(drifted.Gone, []model.EntityRef{giovanni}) || drifted.Files != nil:
				t.Errorf("%v: gone %v, changed in %v", err, drifted.Gone, drifted.Files)
			case !strings.Contains(err.Error(), one.names):
				t.Errorf("%v does not say %q", err, one.names)
			}
		})
	}
}

// A record renamed where the history does not say so — a commit origins did not
// write — cannot be followed, and is still named rather than the file git could not
// find.
func TestAHypothesisAboutARecordRenamedWhereNothingSaysSoHasDrifted(t *testing.T) {
	t.Parallel()
	corpus, repo, base := aLifeInTwoEvents(t)
	giovanni := model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"}
	later := Begin(reloaded(t, corpus.Root))
	if _, err := later.Perform(model.Rename{Entity: giovanni, To: "rossi-giovanni-battista"}); err != nil {
		t.Fatal(err)
	}
	writing(t, later)
	committed(t, corpus.Root, "He was baptised Giovanni Battista")

	_, err := Pin(repo, aCorrection(base), reloaded(t, corpus.Root))
	var drifted *Drifted
	switch {
	case !errors.As(err, &drifted):
		t.Fatalf("%v, want it to have drifted", err)
	case !slices.Equal(drifted.Gone, []model.EntityRef{giovanni}) || drifted.Files != nil:
		t.Errorf("%v: gone %v, changed in %v", err, drifted.Gone, drifted.Files)
	case !strings.Contains(err.Error(), "person/rossi-giovanni,"):
		t.Errorf("%v does not name him", err)
	}
}
