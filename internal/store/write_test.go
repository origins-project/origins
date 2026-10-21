package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

func aCorpus() *model.Corpus {
	corpus := model.NewCorpus()
	corpus.Project = model.Project{
		Name: "Rossi family", MainPerson: "rossi-giovanni", Lang: "it", Copyright: "© 2024 Anna Rossi",
		Researchers: []model.Researcher{
			{Name: "Anna Rossi", Links: []model.Link{{URL: "mailto:anna@example.org"}},
				Addresses: []model.Address{{Lines: "Via dei Servi 12\nFirenze"}}},
			{Name: "Luca Bianchi", Note: "Compiled the Bianchi branch, 1998–2004."},
		},
		Bookmarks:     []model.EntityRef{{Kind: model.KindPerson, Slug: "rossi-giovanni"}},
		Tags:          map[string]model.TagStyle{"To do": {Color: "#d33", Priority: "1"}, "Verified": {}},
		FormatVersion: 1,
	}
	corpus.People["rossi-giovanni"] = &model.Person{
		Slug: "rossi-giovanni", Name: model.PersonName{Nickname: "Nancy", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
		Events: []model.Event{{
			Type:    "birth",
			Date:    mustDate("12 MAR 1834"),
			Place:   "firenze",
			Sources: []model.CitationRef{{Source: "register", Anchor: "1834-042"}},
		}},
		Body: "The argument for why this Giovanni is that Giovanni.",
	}
	corpus.People["rossi-antonio"] = &model.Person{
		Slug: "rossi-antonio", Name: model.PersonName{Given: "Antonio", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
	}
	corpus.Families["bianchi-maria--rossi-giovanni"] = &model.Family{
		Slug: "bianchi-maria--rossi-giovanni", Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"),
	}
	corpus.People["bianchi-maria"] = &model.Person{
		Slug: "bianchi-maria", Name: model.PersonName{Given: "Maria", Surname: model.SurnameOf("Bianchi")}, Sex: model.StatedSex{Value: model.Female},
	}
	corpus.Places["firenze"] = &model.Place{
		Slug: "firenze", Name: model.PlaceName{Text: "Firenze"}, Type: "city",
		Coordinates: &model.Coordinates{Latitude: 43.7696, Longitude: 11.2558},
	}
	corpus.Sources["register"] = &model.Source{
		Slug: "register", Title: "Parish register",
		Citations: map[string]model.Citation{"1834-042": {Text: "baptised 14 March 1834"}},
	}
	return corpus
}

func mustDate(s string) model.Date {
	d, err := model.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

// A corpus written and read back is the same corpus. This is the round trip the import
// path depends on, and it runs through the real layout rather than a buffer, because
// where a file is put is part of being able to find it again.
func TestWritingACorpusAndReadingItBack(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	if err := Write(root, aCorpus()); err != nil {
		t.Fatal(err)
	}

	loaded, problems := Load(root)
	if problems != nil {
		t.Fatalf("what was written did not read back:\n%v", problems)
	}

	original := aCorpus()
	if !reflect.DeepEqual(loaded.Project, original.Project) {
		t.Errorf("project = %+v", loaded.Project)
	}
	if len(loaded.People) != 3 || len(loaded.Families) != 1 ||
		len(loaded.Places) != 1 || len(loaded.Sources) != 1 {
		t.Fatalf("counts are wrong: %v %v %v %v", loaded.PeopleSlugs(),
			loaded.FamilySlugs(), loaded.PlaceSlugs(), loaded.SourceSlugs())
	}

	for _, slug := range original.PeopleSlugs() {
		want, err := Emit(original.People[slug])
		if err != nil {
			t.Fatal(err)
		}
		got, err := Emit(loaded.People[slug])
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(want) {
			t.Errorf("%s came back as:\n%s\nwant:\n%s", slug, got, want)
		}
	}
}

// Origins writes canonically wherever it writes, so a corpus that has just been written
// has nothing for fmt to do. Anything else would mean the import path and the format
// path disagree about the same model.
func TestWhatIsWrittenIsAlreadyCanonical(t *testing.T) {
	root := filepath.Join(t.TempDir(), "tree")
	if err := Write(root, aCorpus()); err != nil {
		t.Fatal(err)
	}
	corpus, problems := Load(root)
	if problems != nil {
		t.Fatal(problems)
	}
	changed, err := Format(corpus, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(changed) > 0 {
		t.Errorf("fmt would rewrite %v", changed)
	}
}

// Merging two trees means deciding that this Giovanni Rossi is that one, which is a
// research judgement and not a file operation. Refusing is what keeps an import from
// quietly making it.
func TestWritingRefusesADirectoryThatIsNotEmpty(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "origins.yaml"), []byte("project:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Write(root, aCorpus()); err == nil {
		t.Fatal("writing into an existing corpus was allowed")
	}
	content, err := os.ReadFile(filepath.Join(root, "origins.yaml"))
	if err != nil || string(content) != "project:\n" {
		t.Errorf("the existing file was touched: %q %v", content, err)
	}
}

// A record is kept in a directory of its own whether or not anything is attached to it
// yet, so the first photograph attached to it moves nothing.
func TestWhereAnEntityIsKeptDoesNotDependOnWhatIsAttachedToIt(t *testing.T) {
	alone := &model.Person{Slug: "rossi-giovanni"}
	carrying := &model.Person{Slug: "rossi-giovanni", Files: []string{"photos/ritratto.jpg"}}
	for _, p := range []*model.Person{alone, carrying} {
		if where, err := PathOf(p); err != nil || where != "people/rossi-giovanni/README.md" {
			t.Errorf("with %d files beside him he goes to %q: %v", len(p.Files), where, err)
		}
	}
}
