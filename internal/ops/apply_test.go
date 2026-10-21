package ops

import (
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/model"
)

func ref(kind model.Kind, slug model.Slug) model.EntityRef {
	return model.EntityRef{Kind: kind, Slug: slug}
}

// aCorpus is three entities of three kinds: enough for an edit to land on, to name the
// wrong kind of, and to collide with.
func aCorpus(t *testing.T) *model.Corpus {
	t.Helper()
	born, err := model.ParseDate("1834")
	if err != nil {
		t.Fatal(err)
	}
	corpus := model.NewCorpus()
	corpus.People["rossi-giovanni"] = &model.Person{
		Slug: "rossi-giovanni", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
		Events: []model.Event{{Type: "birth", Date: born, Place: "firenze"}},
	}
	corpus.Places["firenze"] = &model.Place{Slug: "firenze", Name: model.PlaceName{Text: "Firenze"}, Type: "city"}
	corpus.Sources["firenze-civil-deaths-1901"] = &model.Source{
		Slug: "firenze-civil-deaths-1901", Title: "Registri di morte, Firenze, 1901",
	}
	return corpus
}

func apply(t *testing.T, corpus *model.Corpus, edit model.Edit) {
	t.Helper()
	if _, err := applied(corpus, edit); err != nil {
		t.Fatalf("%s: %v", edit.Op(), err)
	}
}

// The six operations origins can carry out, each doing the thing its name says. Two of
// them are entity-level and four reach inside a record, and all six take the same road
// into the corpus, which is what lets a hypothesis and a typed command be the same
// thing underneath.
func TestEachOperationDoesWhatItSays(t *testing.T) {
	t.Parallel()
	corpus := aCorpus(t)
	giovanni := ref(model.KindPerson, "rossi-giovanni")

	apply(t, corpus, model.Set{Entity: giovanni, Path: "name.surname", Value: "Rossi Tempesti"})
	apply(t, corpus, model.Unset{Entity: giovanni, Path: "events[0].place"})
	apply(t, corpus, model.Add{Entity: giovanni, Path: "sources", Value: "firenze-civil-deaths-1901#p-1"})
	apply(t, corpus, model.Add{Entity: giovanni, Path: "sources", Value: "firenze-civil-deaths-1901#p-2"})
	apply(t, corpus, model.Remove{Entity: giovanni, Path: "sources", Value: "firenze-civil-deaths-1901#p-1"})
	apply(t, corpus, model.Create{
		Entity: ref(model.KindPlace, "borgo-san-lorenzo"),
		Value: map[string]any{
			"name": "Borgo San Lorenzo", "type": "comune", "within": []any{"firenze"},
		},
	})
	apply(t, corpus, model.Delete{Entity: ref(model.KindSource, "firenze-civil-deaths-1901")})

	person := corpus.People["rossi-giovanni"]
	borgo := corpus.Places["borgo-san-lorenzo"]
	switch {
	case person.Name.PrimarySurname().Name != "Rossi Tempesti":
		t.Errorf("set left the surname as %+v", person.Name.Surname)
	case person.Events[0].Place != "":
		t.Errorf("unset left the place as %q", person.Events[0].Place)
	case person.Events[0].Date.String() != "1834":
		t.Errorf("unsetting the place took the date with it: %q", person.Events[0].Date)
	case len(person.Sources) != 1 || person.Sources[0].Anchor != "p-2":
		t.Errorf("add and remove left the citations as %+v", person.Sources)
	case borgo == nil:
		t.Error("create wrote nothing")
	case borgo.Name.Text != "Borgo San Lorenzo" || len(borgo.Within) != 1 || borgo.Within[0].Place != "firenze":
		t.Errorf("create wrote %+v", borgo)
	case corpus.Sources["firenze-civil-deaths-1901"] != nil:
		t.Error("delete left the source behind")
	}
}

// A slug names one thing across all five kinds, so a reference could find its entity by
// slug alone. The kind is checked anyway: it is the writer saying what they believe
// they are editing, and an edit that means to change a person and would change a place
// is a mistake about the corpus rather than a shortcut.
func TestAnEditNamingTheWrongKindIsRefused(t *testing.T) {
	t.Parallel()
	corpus := aCorpus(t)
	_, err := applied(corpus, model.Set{
		Entity: ref(model.KindPerson, "firenze"), Path: "name", Value: "Fiorenza",
	})
	if err == nil {
		t.Fatal("a place was edited as a person")
	}
	if !strings.Contains(err.Error(), "place") {
		t.Errorf("the error does not say what firenze is: %v", err)
	}
	if corpus.Places["firenze"].Name.Text != "Firenze" {
		t.Errorf("the place was changed anyway: %+v", corpus.Places["firenze"])
	}
}

// One slug names one thing, and it names it across the kinds at once, because that is
// what lets a reference be a bare slug with nothing to say which map to look in.
func TestCreatingOverSomethingThatExistsIsRefused(t *testing.T) {
	t.Parallel()
	corpus := aCorpus(t)
	for _, taken := range []model.EntityRef{
		ref(model.KindPlace, "firenze"),
		ref(model.KindPerson, "firenze"),
	} {
		if _, err := applied(corpus, model.Create{Entity: taken}); err == nil {
			t.Errorf("%s was created over a place of that name", taken)
		}
	}
	if corpus.Places["firenze"].Name.Text != "Firenze" {
		t.Errorf("the place was overwritten: %+v", corpus.Places["firenze"])
	}
}

// A second Ivan Ershov is an ordinary thing to find, so a name already taken is
// answered with names that are free, made out of what the create says about him and
// never including one somebody else already has: the fixture's grandson holds
// ershov-ivan-1885, so an Ivan born that year is offered the place instead.
func TestCreatingOverATakenNameOffersNamesThatAreFree(t *testing.T) {
	t.Parallel()
	corpus := fixture(t).Corpus
	for _, c := range []struct {
		create model.Create
		want   string
	}{
		{model.Create{Entity: ref(model.KindPerson, "ershov-ivan"), Value: map[string]any{
			"name": "Ivan",
			"events": []any{
				map[string]any{"type": "baptism", "date": "1880", "place": "hamburg"},
				map[string]any{"type": "birth", "date": "1885", "place": "buenos-aires"},
				map[string]any{"type": "death", "date": "1971"},
			},
		}}, "ershov-ivan-buenos-aires, ershov-ivan-1971, ershov-ivan-2"},
		{model.Create{Entity: ref(model.KindPlace, "firenze"), Value: map[string]any{
			"name": "Firenze", "within": []any{"toscana"},
		}}, "firenze-toscana, firenze-2"},
		{model.Create{Entity: ref(model.KindPerson, "firenze")}, "firenze-2"},
	} {
		_, err := applied(corpus, c.create)
		if err == nil || !strings.HasSuffix(err.Error(), "names still free: "+c.want) {
			t.Errorf("creating %s said %v, want the names %s", c.create.Entity, err, c.want)
		}
	}
}

// A create writes its fields through the same resolution a set goes through, so there
// is nothing it can put in a corpus that a set could not — and a create that cannot
// finish leaves nothing behind, since a half-made person is worse than none.
func TestACreateThatCannotFinishWritesNothing(t *testing.T) {
	t.Parallel()
	corpus := aCorpus(t)
	for _, refused := range []struct {
		why    string
		fields map[string]any
	}{
		{"a field nobody knows", map[string]any{"name": "Antonio", "birthplace": "firenze"}},
		{"a value the corpus could not hold", map[string]any{"name": "Antonio", "sex": "male"}},
	} {
		_, err := applied(corpus, model.Create{
			Entity: ref(model.KindPerson, "rossi-antonio"), Value: refused.fields,
		})
		if err == nil {
			t.Errorf("a person with %s was created", refused.why)
		}
		if corpus.People["rossi-antonio"] != nil {
			t.Fatalf("%s left a person behind: %+v", refused.why, corpus.People["rossi-antonio"])
		}
	}
}

// Apply answers for what the edit says and not for whether the corpus still holds
// together afterwards. Deleting a place that events happened in leaves references
// pointing at nothing, and the transaction is where that is caught — once, for every
// way there is of introducing it, rather than in each operation that might.
func TestApplyLeavesTheContradictionsToTheTransaction(t *testing.T) {
	t.Parallel()
	tx := Begin(fixture(t))
	apply(t, tx.Corpus(), model.Delete{Entity: ref(model.KindPlace, "firenze")})

	_, err := tx.Stage()
	contradiction, refused := err.(*Contradiction)
	if !refused {
		t.Fatalf("staging a corpus with a place deleted out from under it gave %v", err)
	}
	for _, finding := range contradiction.Findings {
		if finding.Rule == "dangling-place" && finding.Severity == check.Error {
			return
		}
	}
	t.Errorf("nothing was reported as a dangling place:\n%v", contradiction)
}

// An archive a source is kept in cannot go while the source still says so.
func TestARepositoryKeepingASourceIsNotDeleted(t *testing.T) {
	t.Parallel()
	tx := Begin(fixture(t))
	apply(t, tx.Corpus(), model.Delete{Entity: ref(model.KindRepository, "archivio-di-stato-di-firenze")})

	_, err := tx.Stage()
	contradiction, refused := err.(*Contradiction)
	if !refused {
		t.Fatalf("staging a corpus with a repository deleted out from under it gave %v", err)
	}
	for _, finding := range contradiction.Findings {
		if finding.Rule == "dangling-repository" && finding.Path == "repositories[2].repository" {
			return
		}
	}
	t.Errorf("nothing was reported as a dangling repository:\n%v", contradiction)
}

// An edit typed at a corpus is a change to files and a commit, the same as any other,
// which is the whole point of there being one road in.
func TestAnAppliedEditLandsAsOneCommit(t *testing.T) {
	t.Parallel()
	corpus, repo := inARepo(t)
	tx := Begin(corpus)
	apply(t, tx.Corpus(), model.Set{
		Entity: ref(model.KindPerson, "rossi-giovanni"), Path: "name.surname", Value: "Rossi Tempesti",
	})

	change, sha, err := tx.Commit(repo, anEntry())
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Fatalf("sha = %q", sha)
	}
	if got := change.Written(); len(got) != 1 || got[0] != "people/rossi-giovanni/README.md" {
		t.Fatalf("wrote %v", got)
	}
	if written := read(t, corpus.Root, "people/rossi-giovanni/README.md"); !strings.Contains(written, "surname: Rossi Tempesti") {
		t.Errorf("the file says:\n%s", written)
	}
	if left := git(t, corpus.Root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}

// The vocabulary is closed and origins carries out the whole of it. An operation added
// to the model reaches this test before it reaches a corpus: one nothing here tries is
// one nothing says can be done, and Apply answers a registered operation it has no case
// for rather than silently doing nothing.
func TestTheWholeVocabularyIsCarriedOut(t *testing.T) {
	t.Parallel()
	tried := vocabulary()
	for _, op := range model.EditOps() {
		edit, covered := tried[op]
		if !covered {
			t.Errorf("nothing here tries a %q, so nothing says whether origins can do one", op)
			continue
		}
		corpus := aCorpus(t)
		corpus.People["landi-rosa"] = &model.Person{Slug: "landi-rosa", Name: model.PersonName{Given: "Rosa"}}
		corpus.People["rossi-giovannino"] = &model.Person{
			Slug: "rossi-giovannino", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")},
		}
		corpus.People["rossi-giovanni"].Sources = []model.CitationRef{{Source: "firenze-civil-deaths-1901", Anchor: "p-1"}}
		if _, err := applied(corpus, edit); err != nil {
			t.Errorf("%q should have been carried out: %v", op, err)
		}
	}
}

// A bookmark follows its record: renamed with it, folded into the record it is merged
// into, and gone with it.
func TestABookmarkFollowsItsRecord(t *testing.T) {
	t.Parallel()
	corpus := aCorpus(t)
	corpus.Places["florence"] = &model.Place{Slug: "florence"}
	corpus.Project.Bookmarks = []model.EntityRef{
		ref(model.KindPlace, "firenze"),
		ref(model.KindPerson, "rossi-giovanni"),
		ref(model.KindPlace, "florence"),
		ref(model.KindSource, "firenze-civil-deaths-1901"),
	}

	apply(t, corpus, model.Rename{Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-battista"})
	apply(t, corpus, model.Merge{From: ref(model.KindPlace, "florence"), Into: ref(model.KindPlace, "firenze")})
	apply(t, corpus, model.Delete{Entity: ref(model.KindSource, "firenze-civil-deaths-1901")})

	want := []model.EntityRef{ref(model.KindPlace, "firenze"), ref(model.KindPerson, "rossi-giovanni-battista")}
	if !slices.Equal(corpus.Project.Bookmarks, want) {
		t.Errorf("the bookmarks are %v, want %v", corpus.Project.Bookmarks, want)
	}
}
