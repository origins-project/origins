package ops

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// The test that holds the rewriter to the validator. Every entity in the fixture is
// renamed in turn, and the corpus that results has to be as sound as the one it came
// from: a reference field added to the model and taught to check but not to the
// rewriter leaves a dangling reference here, in the kind it was added to, the first
// time this runs.
//
// Families are left out because a family is not free to be called anything. It is named
// by its partners, and renaming one is a contradiction the transaction refuses, which
// is a test of its own further down.
func TestARenameLeavesNothingPointingAtTheOldName(t *testing.T) {
	t.Parallel()
	loaded := fixture(t)
	if errors := wrong(check.Run(loaded.Corpus)); len(errors) > 0 {
		t.Fatalf("the fixture is not sound to begin with:\n%s", strings.Join(errors, "\n"))
	}

	for _, slug := range loaded.Slugs() {
		kind := loaded.Corpus.KindOf(slug)
		if kind == model.KindFamily {
			continue
		}
		corpus := loaded.Clone()
		if _, err := applied(corpus, model.Rename{Entity: ref(kind, slug), To: slug + "-renamed"}); err != nil {
			t.Fatalf("%s/%s: %v", kind, slug, err)
		}
		if errors := wrong(check.Run(corpus)); len(errors) > 0 {
			t.Errorf("renaming %s/%s left:\n%s", kind, slug, strings.Join(errors, "\n"))
		}
	}
}

// A picture can be kept by any kind of record, so what it cites and who is in it are
// followed into a source and a repository as well as into a person.
func TestARenameFollowsAPictureWhereverItIsKept(t *testing.T) {
	t.Parallel()
	corpus := fixture(t).Clone()
	album := []model.CitationRef{{Source: "london-marriages-1858", Anchor: "p112"}}
	for _, slug := range corpus.RepositorySlugs()[:1] {
		corpus.Repositories[slug].Media = []model.Media{{File: "facade.jpg", Place: "firenze",
			Depicts: []model.Depicted{{Person: "papadopoulou-maria"}}, Sources: album}}
	}
	for _, slug := range corpus.SourceSlugs()[:1] {
		corpus.Sources[slug].Media = []model.Media{{File: "cover.jpg", Sources: slices.Clone(album)}}
	}
	for _, rename := range []model.Rename{
		{Entity: ref(model.KindSource, "london-marriages-1858"), To: "london-st-pancras-marriages-1858"},
		{Entity: ref(model.KindPerson, "papadopoulou-maria"), To: "papadopoulou-maria-1840"},
		{Entity: ref(model.KindPlace, "firenze"), To: "comune-di-firenze"},
	} {
		if _, err := applied(corpus, rename); err != nil {
			t.Fatalf("%s: %v", rename.Entity, err)
		}
	}

	repository := corpus.Repositories[corpus.RepositorySlugs()[0]].Media[0]
	if repository.Sources[0].Source != "london-st-pancras-marriages-1858" || repository.Depicts[0].Person != "papadopoulou-maria-1840" ||
		repository.Place != "comune-di-firenze" {
		t.Errorf("the repository's picture still says %+v", repository)
	}
	if got := corpus.Sources[corpus.SourceSlugs()[0]].Media[0].Sources[0].Source; got != "london-st-pancras-marriages-1858" {
		t.Errorf("the source's picture cites %q", got)
	}
}

// A link in somebody's words is a reference like any other, and goes where the record
// does, while a link to anybody else is left as it was written.
func TestARenameRewritesALinkInProse(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	corpus.People["bianchi-maria"].Body = "Wife of [Giovanni](../../people/rossi-giovanni/README.md#birth),\n" +
		"not of [his cousin](../../people/rossi-giovanni-1830/README.md).\n"
	corpus.Families["bianchi-maria--rossi-giovanni"].Events = []model.Event{{Type: "marriage",
		Note: "![the groom](../../people/rossi-giovanni/photos/1858.jpg) on the day."}}
	apply(t, corpus, model.Rename{Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834"})

	if body := corpus.People["bianchi-maria"].Body; body != "Wife of [Giovanni](../../people/rossi-giovanni-1834/README.md#birth),\n"+
		"not of [his cousin](../../people/rossi-giovanni-1830/README.md).\n" {
		t.Errorf("the body reads %q", body)
	}
	if note := corpus.Families["bianchi-maria--rossi-giovanni-1834"].Events[0].Note; note !=
		"![the groom](../../people/rossi-giovanni-1834/photos/1858.jpg) on the day." {
		t.Errorf("the family's note reads %q", note)
	}
}

// A picture another record keeps is referred to by its path, which leads into the
// directory of the record keeping it and so goes where that record does.
func TestARenameFollowsAPathIntoTheRecordKeepingAPicture(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	corpus.People["rossi-giovanni"].Media = []model.Media{{File: "photos/1875.jpg", Type: "photo"}}
	corpus.Families["bianchi-maria--rossi-giovanni"].Media = []model.Media{
		{File: "../../people/rossi-giovanni/photos/1875.jpg", Note: "In the album."},
		{File: "../../people/rossi-giovanni-1830/photos/1875.jpg"},
	}
	apply(t, corpus, model.Rename{Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834"})

	media := corpus.Families["bianchi-maria--rossi-giovanni-1834"].Media
	if media[0].File != "../../people/rossi-giovanni-1834/photos/1875.jpg" ||
		media[1].File != "../../people/rossi-giovanni-1830/photos/1875.jpg" {
		t.Errorf("the family's pictures are %+v", media)
	}
}

func wrong(findings []check.Finding) []string {
	var out []string
	for _, finding := range findings {
		if finding.Severity == check.Error {
			out = append(out, "  "+strings.TrimSpace(finding.Entity+" "+finding.Path)+
				": "+finding.Message+" ("+finding.Rule+")")
		}
	}
	return out
}

// A family is named by the two people in it, so renaming one of them renames the family
// as well. Leaving it behind would be a rename that ends with the corpus contradicting
// itself, which is to say a rename origins could not make at all.
func TestRenamingAPartnerRenamesTheFamilyNamedAfterThem(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	if _, err := applied(corpus, model.Rename{
		Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834",
	}); err != nil {
		t.Fatal(err)
	}

	family, there := corpus.Families["bianchi-maria--rossi-giovanni-1834"]
	if !there {
		t.Fatalf("the families are %v", corpus.FamilySlugs())
	}
	if family.Slug != "bianchi-maria--rossi-giovanni-1834" {
		t.Errorf("the family calls itself %q", family.Slug)
	}
	// the slug is the identity, so an entity filed under a name it does not answer to
	// is two answers to one question
	if giovanni := corpus.People["rossi-giovanni-1834"]; giovanni.Slug != "rossi-giovanni-1834" {
		t.Errorf("the person calls himself %q", giovanni.Slug)
	}
	if partners := family.Partners; !slices.ContainsFunc(partners, func(p model.Partner) bool { return p.Person == "rossi-giovanni-1834" }) {
		t.Errorf("the partners are %v", partners)
	}
	if errors := wrong(check.Run(corpus)); len(errors) > 0 {
		t.Errorf("the corpus is left saying:\n%s", strings.Join(errors, "\n"))
	}
}

// A couple who married twice are two families, and renaming one of the partners renames
// both, the second keeping what told it apart from the first.
func TestRenamingAPartnerKeepsTwoFamiliesOfOneCoupleApart(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	corpus.Families["bianchi-maria--rossi-giovanni-1870"] = &model.Family{
		Slug: "bianchi-maria--rossi-giovanni-1870", Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"),
		Events: []model.Event{{Type: "marriage", Date: mustDate(t, "1870")}},
	}
	apply(t, corpus, model.Rename{Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834"})

	if got := corpus.FamilySlugs(); !slices.Equal(got, []model.Slug{
		"bianchi-maria--rossi-giovanni-1834", "bianchi-maria--rossi-giovanni-1834-1870"}) {
		t.Errorf("the families are %v", got)
	}
	if errors := wrong(check.Run(corpus)); len(errors) > 0 {
		t.Errorf("the corpus is left saying:\n%s", strings.Join(errors, "\n"))
	}
}

// Finding who a partner nobody knew was is putting their slug in the place of unknown,
// and the family is named by who they were. The commit names the family at both of its
// names, as a rename does, it being the only place that ever will.
func TestPuttingSomebodyInThePlaceOfUnknownRenamesTheFamily(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	delete(corpus.Families, "bianchi-maria--rossi-giovanni")
	corpus.Families["bianchi-maria--unknown-1858"] = &model.Family{
		Slug: "bianchi-maria--unknown-1858", Partners: model.PartnersOf("bianchi-maria", model.Unknown),
	}
	entry, _, err := performed(corpus, model.Set{
		Entity: ref(model.KindFamily, "bianchi-maria--unknown-1858"), Path: "partners[1]", Value: "rossi-giovanni",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := corpus.FamilySlugs(); !slices.Equal(got, []model.Slug{"bianchi-maria--rossi-giovanni-1858"}) {
		t.Fatalf("the families are %v", got)
	}
	was, now := ref(model.KindFamily, "bianchi-maria--unknown-1858"), ref(model.KindFamily, "bianchi-maria--rossi-giovanni-1858")
	if !slices.Equal(entry.Entities, []model.EntityRef{was, now}) {
		t.Errorf("the commit names %v", entry.Entities)
	}
	if len(entry.Renamed) != 1 || entry.Renamed[0].From != was || entry.Renamed[0].To != now {
		t.Errorf("the commit says the names given up were %v", entry.Renamed)
	}
	if errors := wrong(check.Run(corpus)); len(errors) > 0 {
		t.Errorf("the corpus is left saying:\n%s", strings.Join(errors, "\n"))
	}

	// a family nobody named after its partners keeps the name it has, being somebody's
	// to put right
	corpus.Families["the-stockdales"] = &model.Family{
		Slug: "the-stockdales", Partners: model.PartnersOf("bianchi-maria", model.Unknown),
	}
	apply(t, corpus, model.Set{Entity: ref(model.KindFamily, "the-stockdales"), Path: "partners[1].person", Value: "rossi-giovanni"})
	if corpus.Families["the-stockdales"] == nil {
		t.Errorf("a family its partners did not name was renamed: %v", corpus.FamilySlugs())
	}
}

// Somebody recorded as Unknown is somebody, and is told apart from the word as a
// collision would be.
func TestARecordIsNotCalledByEitherWord(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	_, err := applied(corpus, model.Create{Entity: ref(model.KindPerson, model.Unknown),
		Value: map[string]any{"events": []any{map[string]any{"type": "birth", "date": "1834"}}}})
	if err == nil || !strings.Contains(err.Error(), "unknown-1834") {
		t.Errorf("a person called unknown was %v", err)
	}
}

// A hypothesis is pinned to the commit its edits were written against, and origins
// reconstructs that state rather than reapplying the edits against a later one. A slug
// inside one names the corpus as it stood, so rewriting it would make the hypothesis
// say something nobody proposed.
func TestARenameLeavesWhatAHypothesisProposedAlone(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	corpus.Hypotheses["giovanni-born-1834"] = &model.Hypothesis{
		Slug: "giovanni-born-1834", Status: model.Open, Base: "0123456789abcdef",
		Edits: []model.Edit{model.Set{
			Entity: ref(model.KindPerson, "rossi-giovanni"), Path: "name.surname", Value: "Rossi Tempesti",
		}},
	}
	if _, err := applied(corpus, model.Rename{
		Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834",
	}); err != nil {
		t.Fatal(err)
	}
	edit := corpus.Hypotheses["giovanni-born-1834"].Edits[0]
	if got := edit.Subjects()[0].Slug; got != "rossi-giovanni" {
		t.Errorf("the hypothesis now proposes something about %q", got)
	}
}

// What a rename will not do, each refused before anything is touched. A slug names one
// thing across all five kinds at once, so a name already in use is never free.
func TestARenameThatCannotBeMadeIsRefused(t *testing.T) {
	t.Parallel()
	giovanni := ref(model.KindPerson, "rossi-giovanni")
	for _, one := range []struct {
		what string
		edit model.Rename
	}{
		{"a name already in use", model.Rename{Entity: giovanni, To: "bianchi-maria"}},
		{"the word for a partner nobody knows", model.Rename{Entity: giovanni, To: model.Unknown}},
		{"the word for no partner at all", model.Rename{Entity: giovanni, To: model.None}},
		{"a name that is not a slug", model.Rename{Entity: giovanni, To: "Rossi Giovanni"}},
		{"the name it has", model.Rename{Entity: giovanni, To: "rossi-giovanni"}},
		{"a person that is not there", model.Rename{
			Entity: ref(model.KindPerson, "rossi-antonio"), To: "rossi-antonio-1800"}},
		{"the wrong kind", model.Rename{
			Entity: ref(model.KindPlace, "rossi-giovanni"), To: "firenze"}},
	} {
		corpus := aCouple(t)
		if _, err := applied(corpus, one.edit); err == nil {
			t.Errorf("%s was allowed", one.what)
		}
		if _, still := corpus.People["rossi-giovanni"]; !still {
			t.Errorf("%s left the corpus holding %v", one.what, corpus.PeopleSlugs())
		}
	}
}

// A rename is not an invitation to rearrange the corpus. The entity keeps its place in
// the tree and takes its new name there, and everything attached to it goes along: a
// photograph left in a directory named after somebody who is no longer called that is
// a photograph nobody will find again.
func TestARenamedEntityTakesWhatIsAttachedToIt(t *testing.T) {
	t.Parallel()
	corpus := withAPhotograph(t)

	tx := Begin(corpus)
	if _, err := applied(tx.Corpus(), model.Rename{
		Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834",
	}); err != nil {
		t.Fatal(err)
	}
	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}

	want := []Move{{From: "people/rossi-giovanni", To: "people/rossi-giovanni-1834"}}
	if got := change.Moved(); !slices.Equal(got, want) {
		t.Fatalf("moved %v, want %v", got, want)
	}
	// only the project header, which names him as the person the corpus is about
	if got := change.Written(); !slices.Equal(got, []string{store.ProjectFile}) {
		t.Errorf("wrote %v", got)
	}
	if got := change.Removed(); len(got) != 0 {
		t.Errorf("removed %v, when the file was carried rather than deleted", got)
	}

	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{
		"people/rossi-giovanni-1834/README.md",
		"people/rossi-giovanni-1834/photos/1890-portrait.jpg",
	} {
		if _, err := os.Stat(filepath.Join(corpus.Root, filepath.FromSlash(relative))); err != nil {
			t.Errorf("%s: %v", relative, err)
		}
	}
	if _, err := os.Stat(filepath.Join(corpus.Root, "people", "rossi-giovanni")); err == nil {
		t.Error("the old directory is still there")
	}
}

// A file that is both carried to a new name and rewritten has to be carried first.
// Writing it where it is going and then moving the old one on top of it would leave
// two of the same entity, or one and an error, depending on the order the filesystem
// happened to take them in.
func TestARenamedFileIsWrittenWhereTheRenamePutIt(t *testing.T) {
	t.Parallel()
	corpus := aCoupleOnDisk(t)

	tx := Begin(corpus)
	if _, err := applied(tx.Corpus(), model.Rename{
		Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834",
	}); err != nil {
		t.Fatal(err)
	}
	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}
	if err := change.Apply(); err != nil {
		t.Fatal(err)
	}

	family := read(t, corpus.Root, "families/bianchi-maria--rossi-giovanni-1834/README.md")
	if !strings.Contains(family, "rossi-giovanni-1834") {
		t.Errorf("the family reads:\n%s", family)
	}
	for _, relative := range []string{
		"people/rossi-giovanni/README.md", "families/bianchi-maria--rossi-giovanni/README.md",
	} {
		if _, err := os.Stat(filepath.Join(corpus.Root, filepath.FromSlash(relative))); err == nil {
			t.Errorf("%s is still there under the old name", relative)
		}
	}
}

// A rename that cannot be finished puts every name back. A corpus left with the person
// moved and the family not is one nothing would load, and the researcher would have no
// way of telling which half had happened.
func TestARenameThatFailsPartWayPutsTheNamesBack(t *testing.T) {
	t.Parallel()
	corpus := aCoupleOnDisk(t)

	tx := Begin(corpus)
	if _, err := applied(tx.Corpus(), model.Rename{
		Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834",
	}); err != nil {
		t.Fatal(err)
	}
	change, err := tx.Stage()
	if err != nil {
		t.Fatal(err)
	}

	// the header becomes unwritable once the files have been carried, which is a disk
	// filling up or a file someone has open, arranged so a test can rely on it
	header := filepath.Join(corpus.Root, store.ProjectFile)
	if err := os.Remove(header); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(header, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := change.Apply(); err == nil {
		t.Fatal("the rename reported success")
	}
	for _, relative := range []string{
		"people/rossi-giovanni/README.md", "families/bianchi-maria--rossi-giovanni/README.md",
	} {
		if _, err := os.Stat(filepath.Join(corpus.Root, filepath.FromSlash(relative))); err != nil {
			t.Errorf("%s did not come back: %v", relative, err)
		}
	}
}

// A name one record gives up and another takes in the same change is two files moved,
// one onto the place the other is leaving, and the one leaving has to go first or the
// one arriving lands on top of it. Two records swapping names are the same thing in a
// circle, where neither can go first until one of them is out of the way.
func TestNamesPassedOnInOneChangeLoseNobody(t *testing.T) {
	t.Parallel()
	giovanni := ref(model.KindPerson, "rossi-giovanni")
	maria := ref(model.KindPerson, "bianchi-maria")
	for _, one := range []struct {
		what  string
		edits []model.Edit
		want  map[string]string // file, and whose it has to be
	}{
		{"along a chain", []model.Edit{
			model.Rename{Entity: giovanni, To: "rossi-giovanni-1834"},
			model.Rename{Entity: maria, To: "rossi-giovanni"},
		}, map[string]string{
			"people/rossi-giovanni-1834/README.md": "given: Giovanni",
			"people/rossi-giovanni/README.md":      "given: Maria",
		}},
		{"round a circle", []model.Edit{
			model.Rename{Entity: giovanni, To: "rossi-giovanni-1834"},
			model.Rename{Entity: maria, To: "rossi-giovanni"},
			model.Rename{Entity: ref(model.KindPerson, "rossi-giovanni-1834"), To: "bianchi-maria"},
		}, map[string]string{
			"people/bianchi-maria/README.md":  "given: Giovanni",
			"people/rossi-giovanni/README.md": "given: Maria",
		}},
	} {
		t.Run(one.what, func(t *testing.T) {
			t.Parallel()
			corpus := onDisk(t)
			change := passing(t, corpus, one.edits)
			if err := change.Apply(); err != nil {
				t.Fatal(err)
			}
			for relative, whose := range one.want {
				if got := read(t, corpus.Root, relative); !strings.Contains(got, whose) {
					t.Errorf("%s reads:\n%s", relative, got)
				}
			}
			if left := names(t, corpus.Root, "people"); len(left) != len(one.want) {
				t.Errorf("people/ holds %v", left)
			}
		})
	}
}

// A swap that cannot be finished puts both names back, the detour one of them took
// included.
func TestASwapThatFailsPartWayPutsBothNamesBack(t *testing.T) {
	t.Parallel()
	corpus := onDisk(t)
	change := passing(t, corpus, []model.Edit{
		model.Rename{Entity: ref(model.KindPerson, "rossi-giovanni"), To: "rossi-giovanni-1834"},
		model.Rename{Entity: ref(model.KindPerson, "bianchi-maria"), To: "rossi-giovanni"},
		model.Rename{Entity: ref(model.KindPerson, "rossi-giovanni-1834"), To: "bianchi-maria"},
	})

	// the header, which names him, becomes unwritable once the files have been carried
	header := filepath.Join(corpus.Root, store.ProjectFile)
	if err := os.Remove(header); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(header, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := change.Apply(); err == nil {
		t.Fatal("the swap reported success")
	}
	for relative, whose := range map[string]string{
		"people/rossi-giovanni/README.md": "given: Giovanni",
		"people/bianchi-maria/README.md":  "given: Maria",
	} {
		if got := read(t, corpus.Root, relative); !strings.Contains(got, whose) {
			t.Errorf("%s reads:\n%s", relative, got)
		}
	}
	if left := names(t, corpus.Root, "people"); len(left) != 2 {
		t.Errorf("people/ holds %v", left)
	}
}

// passing carries the edits out in one transaction and stages the result.
func passing(t *testing.T, corpus *store.Corpus, edits []model.Edit) *Change {
	t.Helper()
	tx := Begin(corpus)
	for _, edit := range edits {
		if _, err := tx.Perform(edit); err != nil {
			t.Fatal(err)
		}
	}
	change, err := tx.staged()
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func names(t *testing.T, root, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, entry := range entries {
		out = append(out, entry.Name())
	}
	return out
}

// aCouple is two people and the family naming them both, which is the smallest corpus a
// rename can be read in: the person, the reference, and the name derived from it.
func aCouple(t *testing.T) *model.Corpus {
	t.Helper()
	corpus := model.NewCorpus()
	corpus.Project = model.Project{Name: "Rossi family", MainPerson: "rossi-giovanni", FormatVersion: 1}
	corpus.People["rossi-giovanni"] = &model.Person{
		Slug: "rossi-giovanni", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
	}
	corpus.People["bianchi-maria"] = &model.Person{
		Slug: "bianchi-maria", Name: model.PersonName{Given: "Maria", Surname: model.SurnameOf("Bianchi")}, Sex: model.StatedSex{Value: model.Female},
	}
	corpus.Families["bianchi-maria--rossi-giovanni"] = &model.Family{
		Slug: "bianchi-maria--rossi-giovanni", Partners: model.PartnersOf("rossi-giovanni", "bianchi-maria"),
	}
	return corpus
}

// aCoupleOnDisk is that corpus written out and read back, which is the state every
// transaction starts from.
func aCoupleOnDisk(t *testing.T) *store.Corpus {
	t.Helper()
	root := filepath.Join(t.TempDir(), "corpus")
	if err := store.Write(root, aCouple(t)); err != nil {
		t.Fatal(err)
	}
	loaded, problems := store.Load(root)
	if problems != nil {
		t.Fatalf("what was written did not read back:\n%v", problems)
	}
	return loaded
}

// withAPhotograph is the corpus of onDisk with one person kept as a directory, which
// is the form an entity takes as soon as anything is attached to it.
func withAPhotograph(t *testing.T) *store.Corpus {
	t.Helper()
	corpus := onDisk(t)
	root := corpus.Root
	dir := filepath.Join(root, "people", "rossi-giovanni")
	if err := os.MkdirAll(filepath.Join(dir, "photos"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(
		filepath.Join(root, "people", "rossi-giovanni", "README.md"),
		filepath.Join(dir, "README.md"),
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "photos", "1890-portrait.jpg"), []byte("a scan"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, problems := store.Load(root)
	if problems != nil {
		t.Fatalf("what was written did not read back:\n%v", problems)
	}
	return loaded
}

// The commit is the only place that will ever say a family was renamed. Its name is
// made of its partners rather than chosen, so it changes when one of them does, and a
// renaming nothing recorded is a history that stops dead at the moment the name did.
func TestARenameNamesTheFamiliesItCarries(t *testing.T) {
	t.Parallel()
	corpus := aCouple(t)
	corpus.People["verdi-lucia"] = &model.Person{Slug: "verdi-lucia", Name: model.PersonName{Given: "Lucia", Surname: model.SurnameOf("Verdi")}}

	for _, one := range []struct {
		renaming model.Slug
		to       model.Slug
		want     []model.EntityRef
	}{
		{"rossi-giovanni", "rossi-giovanni-1834", []model.EntityRef{
			{Kind: model.KindPerson, Slug: "rossi-giovanni"},
			{Kind: model.KindPerson, Slug: "rossi-giovanni-1834"},
			{Kind: model.KindFamily, Slug: "bianchi-maria--rossi-giovanni"},
			{Kind: model.KindFamily, Slug: "bianchi-maria--rossi-giovanni-1834"},
		}},
		// nobody was ever married to her, so there is nothing to carry
		{"verdi-lucia", "verdi-lucia-1810", []model.EntityRef{
			{Kind: model.KindPerson, Slug: "verdi-lucia"},
			{Kind: model.KindPerson, Slug: "verdi-lucia-1810"},
		}},
	} {
		entry, _, err := performed(corpus.Clone(), model.Rename{
			Entity: model.EntityRef{Kind: model.KindPerson, Slug: one.renaming},
			To:     one.to,
		})
		if err != nil {
			t.Fatalf("%s: %v", one.renaming, err)
		}
		if !slices.Equal(entry.Entities, one.want) {
			t.Errorf("renaming %s is about %v, want %v", one.renaming, entry.Entities, one.want)
		}
	}
}
