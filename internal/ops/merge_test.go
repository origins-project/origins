package ops

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// twiceOver is one couple recorded twice over: a fuller Giovanni and a sparser one
// carried in from somewhere else, each married to the same Maria, with a child
// declaring the sparser one as a parent. It is what a duplicate looks like after an
// import, and the shape every merge worth making has.
//
// Each of the two families records something the other does not, so that which of them
// went where can be told after a merge has left them under one couple's names.
func twiceOver(t *testing.T) *model.Corpus {
	t.Helper()
	corpus := aCouple(t)
	corpus.Families["bianchi-maria--rossi-giovanni"].Events = []model.Event{
		{Type: "divorce", Date: mustDate(t, "1870")},
	}
	corpus.People["rossi-giovanni-1902"] = &model.Person{
		Slug: "rossi-giovanni-1902", Name: model.PersonName{Surname: model.SurnameOf("Rossi")},
		Events: []model.Event{{Type: "death", Date: mustDate(t, "1902")}},
	}
	corpus.People["rossi-pietro"] = &model.Person{
		Slug: "rossi-pietro", Name: model.PersonName{Given: "Pietro", Surname: model.SurnameOf("Rossi")},
	}
	corpus.Families["bianchi-maria--rossi-giovanni-1902"] = &model.Family{
		Slug:     "bianchi-maria--rossi-giovanni-1902",
		Partners: model.PartnersOf("rossi-giovanni-1902", "bianchi-maria"),
		Children: model.ChildrenOf("rossi-pietro"),
		Events:   []model.Event{{Type: "marriage", Date: mustDate(t, "1861")}},
	}
	return corpus
}

// fatherOf is the partner other than Maria in the family listing Pietro.
func fatherOf(corpus *model.Corpus, child model.Slug) model.Slug {
	for _, parent := range corpus.ParentsOf(child) {
		if parent.Slug != "bianchi-maria" {
			return parent.Slug
		}
	}
	return ""
}

func mustDate(t *testing.T, s string) model.Date {
	t.Helper()
	d, err := model.ParseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// The record that goes leaves nothing behind: what it said is in the one that stays,
// and everything that named it names that one now.
func TestAMergeLeavesOneRecordWithBothOfThemInIt(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	apply(t, corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})

	giovanni := corpus.People["rossi-giovanni"]
	switch {
	case corpus.People["rossi-giovanni-1902"] != nil:
		t.Error("the record merged away is still in the corpus")
	case giovanni.Name.String() != "Giovanni Rossi":
		t.Errorf("the surviving record's own name became %q", giovanni.Name)
	case len(giovanni.Events) != 1 || giovanni.Events[0].Type != "death":
		t.Errorf("the events read %+v", giovanni.Events)
	case fatherOf(corpus, "rossi-pietro") != "rossi-giovanni":
		t.Errorf("a family still lists a child of the record that went: %+v",
			corpus.ParentsOf("rossi-pietro"))
	}
}

// That the two records were one is what the merge decided, and a claim either made of
// the other goes with it; a claim about anybody else stays, naming the one that stays.
func TestAMergeSettlesTheClaimThatTheTwoWereOne(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	corpus.People["rossi-giovanni"].SameAs = []model.SameAs{{ID: "rossi-giovanni-1902", Note: "The same man."}}
	corpus.People["rossi-giovanni-1902"].SameAs = []model.SameAs{{ID: "rossi-giovanni"}, {Name: "Giovanni Rossi of Boston"}}
	corpus.People["rossi-pietro"].SameAs = []model.SameAs{{ID: "rossi-giovanni-1902"}}
	apply(t, corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})
	if got := corpus.People["rossi-giovanni"].SameAs; !reflect.DeepEqual(got, []model.SameAs{{Name: "Giovanni Rossi of Boston"}}) {
		t.Errorf("the record that stays claims %+v", got)
	}
	if got := corpus.People["rossi-pietro"].SameAs; !reflect.DeepEqual(got, []model.SameAs{{ID: "rossi-giovanni"}}) {
		t.Errorf("another record's claim reads %+v", got)
	}
}

// A link to the record that goes leads to the one that stays.
func TestAMergeRewritesALinkInProse(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	corpus.People["rossi-pietro"].Body = "Son of [Giovanni](../../people/rossi-giovanni-1902/README.md).\n"
	apply(t, corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})
	if body := corpus.People["rossi-pietro"].Body; body != "Son of [Giovanni](../../people/rossi-giovanni/README.md).\n" {
		t.Errorf("the body reads %q", body)
	}
}

// A path into the directory of the record that goes leads into the one that stays, and
// one the record that stays held into its own is the path beside it.
func TestAMergeFollowsAPathIntoTheRecordKeepingAPicture(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	corpus.People["rossi-giovanni"].Media = []model.Media{{File: "photos/1858.jpg"}}
	corpus.People["rossi-giovanni-1902"].Media = []model.Media{{File: "photos/1902.jpg"}}
	corpus.People["rossi-giovanni-1902"].Events[0].Media = []model.Media{
		{File: "../../people/rossi-giovanni/photos/1858.jpg", Note: "Taken at the wedding."}}
	corpus.People["rossi-pietro"].Media = []model.Media{{File: "../../people/rossi-giovanni-1902/photos/1902.jpg"}}
	apply(t, corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})

	if got := corpus.People["rossi-pietro"].Media[0].File; got != "../../people/rossi-giovanni/photos/1902.jpg" {
		t.Errorf("Pietro's picture is %q", got)
	}
	if got := corpus.People["rossi-giovanni"].Events[0].Media[0].File; got != "photos/1858.jpg" {
		t.Errorf("the death's picture is %q", got)
	}
}

// An archive typed in twice is one archive, and the sources kept in the one that goes
// are kept in the one that stays.
func TestAMergeOfTwoRepositoriesMovesWhatIsKeptInThem(t *testing.T) {
	t.Parallel()
	corpus := model.NewCorpus()
	corpus.Repositories["archivio-diocesano"] = &model.Repository{Slug: "archivio-diocesano",
		Name: "Archivio Diocesano di Firenze"}
	corpus.Repositories["adf"] = &model.Repository{Slug: "adf",
		Links: model.LinksOf("https://www.example.org/archivio")}
	corpus.Sources["parish-baptisms"] = &model.Source{Slug: "parish-baptisms", Repositories: []model.Holding{{Repository: "adf"}}}
	apply(t, corpus, model.Merge{
		From: ref(model.KindRepository, "adf"),
		Into: ref(model.KindRepository, "archivio-diocesano"),
	})

	kept := corpus.Repositories["archivio-diocesano"]
	switch {
	case corpus.Repositories["adf"] != nil:
		t.Error("the record merged away is still in the corpus")
	case kept.Name != "Archivio Diocesano di Firenze" || !slices.Equal(kept.Links, model.LinksOf("https://www.example.org/archivio")):
		t.Errorf("the repository that stays reads %+v", kept)
	case corpus.Sources["parish-baptisms"].Repositories[0].Repository != "archivio-diocesano":
		t.Errorf("the source is kept in %+v", corpus.Sources["parish-baptisms"].Repositories)
	}
}

// A family is named by its partners, so merging one of them can leave two families with
// one name. One couple's marriage recorded twice and a couple who married twice look the
// same from the names, so the two stay two and the one renamed is told apart by what it
// says itself; merging them is the researcher's to do.
func TestMergingAPartnerKeepsTwoFamiliesOfOneCoupleTwo(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	apply(t, corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})

	if got := corpus.FamilySlugs(); !slices.Equal(got,
		[]model.Slug{"bianchi-maria--rossi-giovanni", "bianchi-maria--rossi-giovanni-1861"}) {
		t.Fatalf("the families are %v", got)
	}
	for slug, kind := range map[model.Slug]string{
		"bianchi-maria--rossi-giovanni":      "divorce",
		"bianchi-maria--rossi-giovanni-1861": "marriage",
	} {
		family := corpus.Families[slug]
		if len(family.Events) != 1 || family.Events[0].Type != kind {
			t.Errorf("%s records %+v", slug, family.Events)
		}
		if !reflect.DeepEqual(family.Partners, model.PartnersOf("rossi-giovanni", "bianchi-maria")) &&
			!reflect.DeepEqual(family.Partners, model.PartnersOf("bianchi-maria", "rossi-giovanni")) {
			t.Errorf("%s's partners read %v", slug, family.Partners)
		}
	}
}

// Where the name a family takes is nobody else's, it is carried to it, exactly as a
// rename would carry it.
func TestMergingAPartnerRenamesAFamilyThatCollidesWithNothing(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	delete(corpus.Families, "bianchi-maria--rossi-giovanni")
	apply(t, corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})

	family := corpus.Families["bianchi-maria--rossi-giovanni"]
	if family == nil {
		t.Fatalf("the family is called %v", corpus.FamilySlugs())
	}
	if len(family.Events) != 1 {
		t.Errorf("the family lost what it recorded: %+v", family.Events)
	}
}

// Two records married to each other are two people. Whatever else that family records,
// it records them as separate enough to have married, and a merge that went ahead would
// leave somebody married to themselves.
func TestTwoPartnersOfOneFamilyAreNotOnePerson(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	_, err := applied(corpus, model.Merge{
		From: ref(model.KindPerson, "bianchi-maria"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})
	if err == nil {
		t.Fatal("a couple were merged into one person")
	}
	if !strings.Contains(err.Error(), "bianchi-maria--rossi-giovanni") {
		t.Errorf("the refusal does not say which family: %v", err)
	}
	if corpus.People["bianchi-maria"] == nil {
		t.Error("the refused merge took a record with it")
	}
}

// A family renamed because one of its partners turned out to be somebody else is a
// family nothing else will ever record the renaming of, so the commit names both ends
// of it as a rename does.
func TestAMergeNamesTheFamiliesItRenamed(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	entry, _, err := performed(corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]bool{}
	for _, one := range entry.Entities {
		named[one.String()] = true
	}
	for _, want := range []string{
		"person/rossi-giovanni-1902", "person/rossi-giovanni",
		"family/bianchi-maria--rossi-giovanni-1902", "family/bianchi-maria--rossi-giovanni-1861",
	} {
		if !named[want] {
			t.Errorf("the commit does not name %s: %v", want, entry.Entities)
		}
	}
}

// A merge that cannot be made leaves the corpus as it was, including the references
// that would have been rewritten had it gone through.
func TestAMergeOfRecordsThatDisagreeChangesNothing(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	corpus.People["rossi-giovanni-1902"].Name = model.PersonName{Given: "Giovanni Battista", Surname: model.SurnameOf("Rossi")}

	_, err := applied(corpus, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})
	if err == nil {
		t.Fatal("two records that name the man differently were merged")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("the refusal does not say which field: %v", err)
	}
	switch {
	case corpus.People["rossi-giovanni-1902"] == nil:
		t.Error("the record that was to go went anyway")
	case fatherOf(corpus, "rossi-pietro") != "rossi-giovanni-1902":
		t.Error("a reference was rewritten by a merge that did not happen")
	case len(corpus.People["rossi-giovanni"].Events) != 0:
		t.Errorf("the surviving record took something: %+v", corpus.People["rossi-giovanni"].Events)
	}
}

// A place and a person are not two records of one thing, and neither is a record and
// itself.
func TestAMergeOfTwoDifferentThingsIsRefused(t *testing.T) {
	t.Parallel()
	giovanni := ref(model.KindPerson, "rossi-giovanni")
	for _, one := range []struct {
		what string
		edit model.Merge
	}{
		{"a person into a place", model.Merge{From: giovanni, Into: ref(model.KindPlace, "firenze")}},
		{"a record into itself", model.Merge{From: giovanni, Into: giovanni}},
		{"a record that is not there", model.Merge{
			From: ref(model.KindPerson, "rossi-nobody"), Into: giovanni}},
	} {
		if _, err := applied(aCorpus(t), one.edit); err == nil {
			t.Errorf("%s was merged", one.what)
		}
	}
}

// The first record named keeps everything the one being split had, references to it
// included, and the rest arrive holding nothing. Copying the record into each of them
// would have the corpus assert that two people were born on the same day in the same
// place to the same parents, which nobody stated.
func TestASplitKeepsOneRecordAndInventsNothingForTheOthers(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	apply(t, corpus, model.Split{
		Entity: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: []model.EntityRef{
			ref(model.KindPerson, "rossi-giovanni-padre"),
			ref(model.KindPerson, "rossi-giovanni-figlio"),
		},
	})

	padre, figlio := corpus.People["rossi-giovanni-padre"], corpus.People["rossi-giovanni-figlio"]
	switch {
	case corpus.People["rossi-giovanni-1902"] != nil:
		t.Error("the record that was split is still there under its old name")
	case padre == nil || figlio == nil:
		t.Fatalf("the corpus holds %v", corpus.PeopleSlugs())
	case len(padre.Events) != 1:
		t.Errorf("the first record named did not keep what there was: %+v", padre.Events)
	case !reflect.DeepEqual(figlio, &model.Person{Slug: "rossi-giovanni-figlio"}):
		t.Errorf("the second record was invented rather than left empty: %+v", figlio)
	case fatherOf(corpus, "rossi-pietro") != "rossi-giovanni-padre":
		t.Errorf("the child was left with %v", corpus.ParentsOf("rossi-pietro"))
	case corpus.Families["bianchi-maria--rossi-giovanni-padre"] == nil:
		t.Errorf("the family was not carried with the partner: %v", corpus.FamilySlugs())
	}
}

// A record that keeps the name it had is split by being left where it is, which is the
// same rule said of the same record: the first one named keeps what there was.
func TestASplitCanLeaveTheRecordUnderTheNameItHad(t *testing.T) {
	t.Parallel()
	corpus := twiceOver(t)
	giovanni := ref(model.KindPerson, "rossi-giovanni-1902")
	apply(t, corpus, model.Split{
		Entity: giovanni,
		Into:   []model.EntityRef{giovanni, ref(model.KindPerson, "rossi-giovanni-figlio")},
	})
	switch {
	case len(corpus.People["rossi-giovanni-1902"].Events) != 1:
		t.Error("the record that kept its name lost what it had")
	case corpus.People["rossi-giovanni-figlio"] == nil:
		t.Errorf("the corpus holds %v", corpus.PeopleSlugs())
	}
}

// A split into fewer than two records is no split, and one into a name already taken is
// a merge nobody asked for.
func TestASplitThatCannotBeMadeIsRefused(t *testing.T) {
	t.Parallel()
	giovanni := ref(model.KindPerson, "rossi-giovanni")
	for _, one := range []struct {
		what string
		edit model.Split
	}{
		{"into one record", model.Split{Entity: giovanni,
			Into: []model.EntityRef{ref(model.KindPerson, "rossi-giovanni-padre")}}},
		{"into a name already taken", model.Split{Entity: giovanni, Into: []model.EntityRef{
			ref(model.KindPerson, "rossi-giovanni-padre"), ref(model.KindPerson, "firenze")}}},
		{"into one name twice", model.Split{Entity: giovanni, Into: []model.EntityRef{
			ref(model.KindPerson, "rossi-giovanni-padre"),
			ref(model.KindPerson, "rossi-giovanni-padre")}}},
		{"into another kind", model.Split{Entity: giovanni, Into: []model.EntityRef{
			ref(model.KindPerson, "rossi-giovanni-padre"),
			ref(model.KindPlace, "borgo-san-lorenzo")}}},
	} {
		corpus := aCorpus(t)
		if _, err := applied(corpus, one.edit); err == nil {
			t.Errorf("a split %s was carried out", one.what)
		}
		if corpus.People["rossi-giovanni"] == nil {
			t.Errorf("a split %s took the record with it", one.what)
		}
	}
}

// twiceOverOnDisk is that corpus written out and read back, which is the state every
// transaction starts from.
func twiceOverOnDisk(t *testing.T) *store.Corpus {
	t.Helper()
	root := filepath.Join(t.TempDir(), "corpus")
	if err := store.Write(root, twiceOver(t)); err != nil {
		t.Fatal(err)
	}
	loaded, problems := store.Load(root)
	if problems != nil {
		t.Fatalf("what was written did not read back:\n%v", problems)
	}
	return loaded
}

// On disk a merge is two files becoming one: the record that stays is rewritten with
// what it took in and the record that goes has its file removed, and the family named
// after the one that goes is renamed after the one that stays, beside the family that
// name already belonged to, its child going with it. The child's own record says nothing
// of whose child he is, and is not written.
func TestAMergeWritesTheRecordThatStaysAndRemovesTheOneThatGoes(t *testing.T) {
	t.Parallel()
	tx := Begin(twiceOverOnDisk(t))
	perform(t, tx, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	written := []string{
		"families/bianchi-maria--rossi-giovanni-1861/README.md",
		"people/rossi-giovanni/README.md",
	}
	removed := []string{
		"people/rossi-giovanni-1902/README.md",
	}
	if got := change.Written(); !slices.Equal(got, written) {
		t.Errorf("would write %v, want %v", got, written)
	}
	if got := change.Removed(); !slices.Equal(got, removed) {
		t.Errorf("would remove %v, want %v", got, removed)
	}
}

// The fixture's two Ivans are a grandfather and the grandson named for him, and the
// corpus already says they are two men by giving them two names. A merge asked for over
// a real corpus is refused on that, and refused having changed nothing: over a hundred
// entities later there is still nothing to write.
func TestTheFixturesTwoIvansAreNotMadeOneMan(t *testing.T) {
	t.Parallel()
	tx := Begin(fixture(t))
	_, err := applied(tx.Corpus(), model.Merge{
		From: ref(model.KindPerson, "ershov-ivan-1885"),
		Into: ref(model.KindPerson, "ershov-ivan"),
	})
	if err == nil {
		t.Fatal("a grandfather and his grandson were made one man")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("the refusal does not say what the two records disagree about: %v", err)
	}
	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if !change.Empty() {
		t.Errorf("the refused merge left something to write: %v", change.Written())
	}
}

// twiceOverWithEvidence is that corpus with something attached to each of the records
// a merge would do away with: a photograph the sparser Giovanni's portrait names and a
// scan nothing names at all, and the marriage certificate of the family he is a partner
// in. It is the ordinary shape of a duplicate worth merging, the record carried in
// from somewhere else being the one with the evidence beside it.
func twiceOverWithEvidence(t *testing.T) *store.Corpus {
	t.Helper()
	corpus := twiceOver(t)
	corpus.People["rossi-giovanni-1902"].Portrait = model.Image{File: "photos/1902/passaporto.jpg"}

	root := filepath.Join(t.TempDir(), "corpus")
	if err := store.Write(root, corpus); err != nil {
		t.Fatal(err)
	}
	attach(t, root, "people", "rossi-giovanni-1902",
		"photos/1902/passaporto.jpg", "certificato.pdf")
	attach(t, root, "families", "bianchi-maria--rossi-giovanni-1902", "atto-di-matrimonio.jpg")

	loaded, problems := store.Load(root)
	if problems != nil {
		t.Fatalf("what was written did not read back:\n%v", problems)
	}
	return loaded
}

// attach puts files in an entity's directory, beside the README holding it.
func attach(t *testing.T, root, kind string, slug model.Slug, files ...string) {
	t.Helper()
	dir := filepath.Join(root, kind, string(slug))
	for _, file := range files {
		at := filepath.Join(dir, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func perform(t *testing.T, tx *Tx, edit model.Edit) {
	t.Helper()
	if _, err := tx.Perform(edit); err != nil {
		t.Fatalf("%s: %v", edit.Op(), err)
	}
}

// What was attached to the record that goes is carried after it, both what the
// surviving record now points at and what nothing points at yet, and a family renamed
// takes its whole directory with it.
func TestAMergeCarriesWhatWasAttachedToTheRecordThatGoes(t *testing.T) {
	t.Parallel()
	tx := Begin(twiceOverWithEvidence(t))
	perform(t, tx, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	want := []Move{
		{"families/bianchi-maria--rossi-giovanni-1902", "families/bianchi-maria--rossi-giovanni-1861"},
		{"people/rossi-giovanni-1902/certificato.pdf", "people/rossi-giovanni/certificato.pdf"},
		{"people/rossi-giovanni-1902/photos/1902/passaporto.jpg",
			"people/rossi-giovanni/photos/1902/passaporto.jpg"},
	}
	if got := change.Moved(); !slices.Equal(got, want) {
		t.Errorf("would carry %v, want %v", got, want)
	}
}

// The same merge written out. The photograph is beside the record that stayed, where
// its portrait says it is, the scan nothing points at is beside it too, and the
// directory the two of them came out of is gone: a folder named after a record the
// corpus no longer holds is a trace of it that nothing accounts for.
func TestTheCarriedFilesEndUpBesideTheRecordThatStays(t *testing.T) {
	t.Parallel()
	corpus := twiceOverWithEvidence(t)
	tx := Begin(corpus)
	perform(t, tx, model.Merge{
		From: ref(model.KindPerson, "rossi-giovanni-1902"),
		Into: ref(model.KindPerson, "rossi-giovanni"),
	})

	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}
	for _, file := range []string{
		"people/rossi-giovanni/README.md",
		"people/rossi-giovanni/certificato.pdf",
		"people/rossi-giovanni/photos/1902/passaporto.jpg",
		"families/bianchi-maria--rossi-giovanni-1861/atto-di-matrimonio.jpg",
	} {
		if _, err := os.Stat(filepath.Join(corpus.Root, filepath.FromSlash(file))); err != nil {
			t.Errorf("%v", err)
		}
	}
	for _, dir := range []string{
		"people/rossi-giovanni-1902",
		"families/bianchi-maria--rossi-giovanni-1902",
	} {
		if _, err := os.Stat(filepath.Join(corpus.Root, filepath.FromSlash(dir))); !os.IsNotExist(err) {
			t.Errorf("%s, which nothing is kept in any more, is still there", dir)
		}
	}

	again, problems := store.Load(corpus.Root)
	if problems != nil {
		t.Fatalf("the merged corpus did not read back:\n%v", problems)
	}
	if _, err := again.Attachment("rossi-giovanni", "photos/1902/passaporto.jpg"); err != nil {
		t.Errorf("the portrait it took does not resolve: %v", err)
	}
}
