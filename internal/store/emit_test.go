package store

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// The three guarantees canonical serialisation means, from DESIGN.md. Everything else
// in this file is a case one of them is easy to break on.

// Deterministic: the bytes are a pure function of the model. The hazard this actually
// catches is map iteration, which Go randomises deliberately so that the mistake fails
// on the second run rather than years later on someone else's machine.
func TestEmittingIsDeterministic(t *testing.T) {
	c := fixture(t)
	for _, slug := range c.Slugs() {
		first, err := Emit(c.Entity(slug))
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		for range 8 {
			again, err := Emit(c.Entity(slug))
			if err != nil {
				t.Fatalf("%s: %v", slug, err)
			}
			if !bytes.Equal(first, again) {
				t.Fatalf("%s emitted differently on a later pass:\n%s\n---\n%s", slug, first, again)
			}
		}
	}
}

// The same corpus read twice must emit the same bytes, however the walk happened to
// visit it. This is the whole-corpus form of the test above, and the one that would
// catch a slug map reaching the output.
func TestEmittingDoesNotDependOnTheOrderOfTheWalk(t *testing.T) {
	first, second := fixture(t), fixture(t)
	if !slices.Equal(first.Slugs(), second.Slugs()) {
		t.Fatal("the corpus held different entities on the second load")
	}
	for _, slug := range first.Slugs() {
		a, err := Emit(first.Entity(slug))
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		b, err := Emit(second.Entity(slug))
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if !bytes.Equal(a, b) {
			t.Errorf("%s emitted differently from a second load:\n%s\n---\n%s", slug, a, b)
		}
	}
}

// Idempotent: emitting is a normalisation, so a second pass has nothing left to do.
// Lossless: what comes back out of the parser is what went in. The two are tested
// together because the round trip is how the second emission is produced.
func TestEmittingIsIdempotentAndLosesNothing(t *testing.T) {
	c := fixture(t)
	for _, slug := range c.Slugs() {
		entity := c.Entity(slug)
		file, err := Emit(entity)
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		reread := reparse(t, slug, entity, file)
		if !reflect.DeepEqual(withoutFiles(entity), reread) {
			t.Errorf("%s did not survive the round trip:\n%s\ngot %+v\nwant %+v", slug, file, reread, entity)
			continue
		}
		again, err := Emit(reread)
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		if !bytes.Equal(file, again) {
			t.Errorf("%s emitted differently the second time:\n%s\n---\n%s", slug, file, again)
		}
	}
}

// withoutFiles copies an entity with its attachments cleared. Attachments are files
// beside the entity rather than frontmatter, so a round trip through the emitter cannot
// carry them and the comparison has to set them aside.
func withoutFiles(entity any) any {
	switch e := entity.(type) {
	case *model.Person:
		copied := *e
		copied.Files = nil
		return &copied
	case *model.Family:
		copied := *e
		copied.Files = nil
		return &copied
	case *model.Place:
		copied := *e
		copied.Files = nil
		return &copied
	case *model.Source:
		copied := *e
		copied.Files = nil
		return &copied
	case *model.Hypothesis:
		copied := *e
		copied.Files = nil
		return &copied
	}
	return entity
}

// reparse reads emitted bytes back as the same kind of entity they came from.
func reparse(t *testing.T, slug model.Slug, like any, file []byte) any {
	t.Helper()
	frontmatter, body, err := Split(file)
	if err != nil {
		t.Fatalf("%s: %v\n%s", slug, err, file)
	}
	var entity any
	switch like.(type) {
	case *model.Person:
		entity, err = decodePerson(slug, frontmatter, string(body))
	case *model.Family:
		entity, err = decodeFamily(slug, frontmatter, string(body))
	case *model.Place:
		entity, err = decodePlace(slug, frontmatter, string(body))
	case *model.Source:
		entity, err = decodeSource(slug, frontmatter, string(body))
	case *model.Repository:
		entity, err = decodeRepository(slug, frontmatter, string(body))
	case *model.Note:
		entity, err = decodeNote(slug, frontmatter, string(body))
	case *model.Hypothesis:
		entity, err = decodeHypothesis(slug, frontmatter, string(body))
	default:
		t.Fatalf("%s: nothing reads a %T back", slug, like)
	}
	if err != nil {
		t.Fatalf("%s did not parse after being emitted: %v\n%s", slug, err, file)
	}
	return entity
}

// The golden files are one entity of each kind carrying the cases the format is awkward
// about, so that a change to field order, quoting or folding shows up as a diff a
// reader can judge rather than as a passing property test.
func TestGoldenFiles(t *testing.T) {
	for name, entity := range golden() {
		path := filepath.Join("testdata", name)
		got, err := Emit(entity)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if *update {
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v; run go test ./internal/store -update", err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s:\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
		}
	}
}

// Whatever the golden files hold has to be readable, or they are recording a bug.
func TestGoldenFilesRoundTrip(t *testing.T) {
	for name, entity := range golden() {
		file, err := Emit(entity)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		slug := model.Slug(strings.TrimSuffix(name, ".md"))
		if reread := reparse(t, slug, entity, file); !reflect.DeepEqual(entity, reread) {
			t.Errorf("%s did not survive the round trip:\n%s\ngot %+v", name, file, reread)
		}
	}
}

func golden() map[string]any {
	year := func(y int) model.Date { return model.Date{Year: &y} }
	date := func(s string) model.Date {
		d, err := model.ParseDate(s)
		if err != nil {
			panic(err)
		}
		return d
	}
	return map[string]any{
		"person.md": &model.Person{
			Private: true,
			Living:  model.LivingFalse,
			Slug:    "person",
			Name: model.PersonName{Prefix: "Dott.ssa", Given: "Anna Maria", Call: "Anna", Nickname: "Nancy",
				SurnamePrefix: "O'", Surname: model.SurnameOf("Brien"), Suffix: "jr.",
				Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042",
					Grade: model.Grade{Source: 'O', Information: 'P', Evidence: 'D'}}}},
			AlsoKnownAs: []model.PersonName{
				{Text: "Nancy O'Brien"},
				{Text: "1834"},
				{Private: true, Given: "Nancy", Surname: model.SurnameOf("Rossi"), Type: "married", From: year(1858),
					Note: "Taken on her marriage, and kept after his death."},
			},
			Sex: model.StatedSex{Value: model.Female,
				Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}},
			Portrait: model.Image{File: "photos/1858-group.jpg", Region: &model.Region{Left: 0.4, Top: 0.1, Right: 0.6, Bottom: 0.5}},
			Events: []model.Event{
				{Type: "birth", Date: year(1834), Place: "borgo-san-lorenzo",
					Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}},
				{Type: "emigration", Date: date("JULIAN 3 SEP 1860"), Tags: []string{"Verified"}},
				{Private: true, Type: "residence", Date: date("HEBREW 17 IYR 5621")},
				{Type: "death", Note: "the gravestone gives a day and a month: 3 marzo, and no year"},
				{Type: "baptism", Date: date("14 MAR 1834"), With: []model.Named{
					{Person: "verdi-giacomo", Role: "godfather", Age: "40"},
					{Name: model.PersonName{Text: "Maria Neri"}, Role: "godmother"},
					{Name: model.PersonName{Given: "Pietro", Surname: model.SurnameOf("Landi")}, Role: "witness", Age: "> 60y",
						Note:    "The register calls him the sexton, which may be why he was there.",
						Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}},
				}},
				{Type: "occupation", Value: "mezzadro", Date: date("FROM 1855 TO 1880")},
				{Type: "title", Value: "Contessa di Montalto"},
				{Type: "burial", Age: model.StatedAge{Value: "67y 3m"}, Cause: model.StatedCause{Value: "polmonite"}, Address: model.Address{Lines: "Cimitero di Trespiano"}},
				{Type: "residence", Place: "firenze", Date: date("1861"), Address: model.Address{
					Lines: "Via de' Ginori 12\nFirenze", Street: "Via de' Ginori 12", City: "Firenze",
					Sources: []model.CitationRef{{Source: "censimento-1861-firenze", Anchor: "f088"}},
				}},
				{Type: "occupation", Value: "fotografo", Links: []model.Link{{URL: "tel:055 213456"}}},
			},
			Media: []model.Media{
				{File: "photos/1858-group.jpg", Title: "The family: Nancy, her mother and a cousin", Date: date("ABT 1858"),
					Place: "firenze", Depicts: []model.Depicted{
						{Person: "person", Region: &model.Region{Left: 0.4, Top: 0.1, Right: 0.6, Bottom: 0.5}},
						{Person: "a-mother", Note: "Standing at the back, though the album says seated."},
					},
					Note:    "*Research:* the photographer's stamp on the back is a studio in via Tornabuoni.",
					Sources: []model.CitationRef{{Source: "family-album", Anchor: "p3"}},
					Tags:    []string{"Scanned", "To do"}},
				{Private: true, File: "photos/1890 portrait.jpg", Depicts: []model.Depicted{{Person: "person"}}},
			},
			Sources: []model.CitationRef{{Source: "censimento-1861-firenze", Anchor: "f088",
				Grade: model.Grade{Source: 'O', Information: 'S'}}},
			Tags: []string{"O'Brien branch", "To do: the 1871 census"},
			SameAs: []model.SameAs{
				{ID: "obrien-anna-maria"},
				{ID: "obrien-nancy", Note: "The same baptism, entered from the diocesan copy.",
					Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}},
				{Name: "Nancy O'Brien of Boston", Note: "A cousin's tree has her emigrating."},
			},
			Links: []model.Link{
				{URL: "https://www.wikitree.com/wiki/OBrien-0"},
				{URL: "https://www.familysearch.org/tree/person/XXXX-YYY", Type: "web-search", Title: "Tree entry",
					Note: "Merged with a duplicate in 2019."},
			},
			IDs: model.IDs{
				"wikitree": model.IdentifiersOf("OBrien-0"),
				"familysearch": {{ID: "XXXX-YYY"}, {ID: "XXXX-XXX", Note: "The duplicate merged into this one.",
					Sources: []model.CitationRef{{Source: "censimento-1861-firenze", Anchor: "f088"}}}},
				"geni":   model.IdentifiersOf("6000000012345678901"),
				"gramps": model.IdentifiersOf("I0012", "I0345"),
			},
			Body: "The argument for why this Nancy is that Nancy.",
		},
		"family.md": &model.Family{
			Private: true,
			Slug:    "family",
			Partners: []model.Partner{
				{Person: "a-partner", Note: "Her second union, after the first was dissolved."},
				{Name: model.PersonName{Given: "Davide", Surname: model.SurnameOf("Sanna"),
					Sources: []model.CitationRef{{Source: "firenze-unioni-civili-2018", Anchor: "112"}}}},
			},
			Type: model.CivilUnion,
			Children: []model.Child{
				{ID: "a-child", Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}},
				{ID: "a-ward", Relation: model.Relationship{Word: "natural"}, Status: model.Challenged},
				{ID: model.Unknown, Note: "the census counts a third child"},
				{Name: model.PersonName{Given: "Gemma", Surname: model.SurnameOf("Nocentini")}, Relation: model.Relationship{Partners: []model.PartnerRelation{
					{Partner: "a-partner", Relation: model.Adoptive}, {Partner: model.Unknown, Relation: model.NoRelation},
				}}, Status: model.Disproven},
				{ID: "another-child"},
			},
			Events: []model.Event{{Type: "civil-union", Date: date("19 MAY 2018"), Place: "firenze", With: []model.Named{
				{Person: "a-partner", Role: "partner", Age: "34"},
				{Person: "another-partner", Role: "partner", Age: "31"},
			}}},
			Sources: []model.CitationRef{{Source: "firenze-unioni-civili-2018", Anchor: "112"}},
			IDs:     model.IDs{"wikidata": model.IdentifiersOf("Q1")},
		},
		"place.md": &model.Place{
			Private: true,
			Slug:    "place",
			Name:    model.PlaceName{Text: "Firenze", Lang: "it", Note: "The name the city goes by in Tuscany."},
			AlsoKnownAs: []model.PlaceName{
				{Text: "Florence", Lang: "en"},
				{Text: "Florenz", Lang: "de"},
				{Text: "Florentsiya", Lang: "ru-Latn", Type: "romanized"},
				{Text: "Fiorenza", Until: year(1500), Note: "Villani's spelling.",
					Sources: []model.CitationRef{{Source: "villani-cronica", Anchor: "i-38"}}},
			},
			Type: "city",
			Within: []model.PlaceRef{
				{Place: "granducato-di-toscana", Until: year(1860), Note: "Annexed by plebiscite in March 1860."},
				{Place: "toscana", From: year(1861),
					Sources: []model.CitationRef{{Source: "censimento-1861-firenze", Anchor: "f088"}}},
			},
			Coordinates: &model.Coordinates{Latitude: 43.7696, Longitude: 11.2558},
			Addresses:   []model.Address{{Lines: "Palazzo Vecchio\nPiazza della Signoria\n50122 Firenze\n"}},
			From:        date("59 BCE"),
			Links:       []model.Link{{URL: "https://www.comune.fi.it"}},
			Sources:     []model.CitationRef{{Source: "villani-cronica", Anchor: "i-38"}},
			IDs:         model.IDs{"geonames": model.IdentifiersOf("3176959"), "wikidata": model.IdentifiersOf("Q2044")},
		},
		"source.md": &model.Source{
			Private:     true,
			Slug:        "source",
			Title:       "Parish of San Lorenzo — Baptismal Register 1820-1850",
			ShortTitle:  "S. Lorenzo, Battesimi 4",
			Type:        "parish-register",
			Author:      "Diocese of Fiesole",
			Publication: "Borgo San Lorenzo: parish of San Lorenzo, 1820",
			Covers: []model.Coverage{
				{Events: []string{"baptism", "birth"}, Date: date("FROM 1820 TO 1850"), Place: "firenze"},
				{Place: "fiesole", Note: "Two baptisms of children of the parish born there."},
			},
			Repositories: []model.Holding{
				{Repository: "archivio-diocesano-di-firenze", CallNumber: "ADF/SL/B/4"},
				{Repository: "familysearch", CallNumber: "Film 1903581", Medium: "microfilm",
					Note: "Filmed in 1987, the last leaves out of focus."},
				{Repository: "archivio-di-stato-di-firenze"},
				{CallNumber: "B 4/1834"},
			},
			Text: "Liber baptizatorum ecclesiae Sancti Laurentii.\n",
			Citations: map[string]model.Citation{
				"1834-042": {
					Event: "baptism",
					Page:  "c. 88r, n. 112",
					Date:  date("14 MAR 1834"),
					Text:  "Giovanni Battista, figlio di Antonio Rossi e Lucia Verdi, nato il 12 marzo 1834 e battezzato il 14 detto.\n",
					With: []model.Named{
						{Person: "rossi-giovanni-battista", Role: "child"},
						{Person: "rossi-antonio", Role: "father", Age: "31y"},
						{Name: model.PersonName{Text: "Lucia Verdi"}, Role: "mother", Note: "Written Lucia del Verde."},
					},
					Images: []model.Media{
						{File: "scans/1834-042.jpg", Region: &model.Region{Left: 0.08, Top: 0.41, Right: 0.93, Bottom: 0.62}},
						{File: "scans/1834-042 verso.jpg"},
					},
					Links: model.LinksOf("https://www.example.org/registers/san-lorenzo/88r"),
					Note:  "The margin adds the godparents' parish, half legible.\n\n- [ ] Read it under a lamp.\n",
				},
				"1836-011": {Text: "Maria Anna, daughter of the same, baptised 2 February 1836.", Tags: []string{"To do"}},
				"0042":     {Text: "An anchor that a parser would read as a number.\n\nAnd a transcription in two paragraphs."},
			},
			Links: model.LinksOf("https://www.example.org/registers/san-lorenzo"),
			IDs:   model.IDs{"familysearch": model.IdentifiersOf("true")},
		},
		"repository.md": &model.Repository{
			Private: true,
			Slug:    "repository",
			Name:    "Archivio Diocesano di Firenze",
			Type:    "archive",
			Addresses: []model.Address{
				{Lines: "Piazza San Giovanni 3\n50129 Firenze\n", Street: "Piazza San Giovanni 3", City: "Firenze",
					Postal: "50129", Country: "Italia", Links: []model.Link{
						{URL: "tel:055 213456"}, {URL: "tel:055 213457", Type: "fax"},
					}, Note: "The entrance is on the side of the baptistery."},
				{Lines: "Via dei Servi 15\nFirenze\n", Date: date("TO 1998")},
				{Lines: "Casella postale 12, Firenze"},
			},
			Links: model.LinksOf("https://www.example.org/archivio", "mailto:archivio@example.org"),
			Media: []model.Media{{File: "facade.jpg", Title: "Piazza San Giovanni", Date: date("2018")}},
			Body:  "Open Tuesday and Thursday mornings.",
			IDs:   model.IDs{"wikidata": model.IdentifiersOf("Q1")},
		},
		"note.md": &model.Note{
			Private: true,
			Slug:    "note",
			Title:   "Vicchio, the 1861 marriages",
			Events: []model.Event{{Type: "marriage", Date: date("12 FEB 1861"), Place: "vicchio",
				With: []model.Named{{Person: "landi-pietro", Role: "witness"}}, Tags: []string{"To do"}}},
			Media:   []model.Media{{File: "letter-to-the-parish.jpg", Title: "The letter as sent"}},
			Links:   model.LinksOf("https://www.example.org/vicchio/matrimoni"),
			Sources: []model.CitationRef{{Source: "vicchio-matrimoni-1861", Anchor: "12"}},
			Tags:    []string{"Research log"},
			SameAs:  []model.SameAs{{ID: "vicchio-1861"}},
			IDs:     model.IDs{"gramps": model.IdentifiersOf("N0007")},
			Body:    "*Research:* Pietro Landi witnessed this wedding.\n\n- [ ] Ask the parish for the banns of January 1861.",
		},
		"hypothesis.md": &model.Hypothesis{
			Slug:        "hypothesis",
			Status:      model.Open,
			Base:        "8f78553",
			Confidence:  model.Possible,
			ProposedBy:  "agent:1",
			Supports:    []model.CitationRef{{Source: "censimento-1861-firenze", Anchor: "f214"}},
			Contradicts: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1842-007"}},
			Edits: []model.Edit{
				model.Set{Entity: ref(model.KindPerson, "rossi-annunziata"), Path: "events[0].date", Value: "1844"},
				model.Set{Entity: ref(model.KindPerson, "rossi-annunziata"), Path: "body",
					Value: "# Annunziata Rossi\n\nBorn in 1844, if the census is right.\n"},
				model.Unset{Entity: ref(model.KindPerson, "rossi-annunziata"), Path: "surname"},
				model.Add{Entity: ref(model.KindFamily, "a--b"), Path: "partners", Value: "c"},
				model.Remove{Entity: ref(model.KindFamily, "a--b"), Path: "partners", Value: "c"},
				model.Create{Entity: ref(model.KindPerson, "rossi-unknown-1845"), Value: map[string]any{
					"name": "Unknown Rossi",
					"sex":  "U",
					"events": []any{map[string]any{"type": "birth", "date": "ABT 1845",
						"note": "no record of the birth\nonly a gap of five years"}},
				}},
				model.Delete{Entity: ref(model.KindPerson, "a-duplicate")},
				model.Rename{Entity: ref(model.KindPerson, "a-duplicate"), To: "rossi-giovanni-1902"},
				model.Merge{From: ref(model.KindPerson, "a-duplicate"), Into: ref(model.KindPerson, "rossi-giovanni")},
				model.Split{Entity: ref(model.KindPerson, "two-men"), Into: []model.EntityRef{
					ref(model.KindPerson, "one-man"), ref(model.KindPerson, "another-man"),
				}},
			},
		},
	}
}

func ref(kind model.Kind, slug model.Slug) model.EntityRef {
	return model.EntityRef{Kind: kind, Slug: slug}
}

// The edit vocabulary is closed, and an operation with no shape to write would emit an
// edit missing its own fields. The golden hypothesis is where the whole vocabulary is
// exercised, so this holds it to the vocabulary itself.
func TestEveryEditOperationIsWritten(t *testing.T) {
	written := map[string]bool{}
	for _, edit := range golden()["hypothesis.md"].(*model.Hypothesis).Edits {
		written[edit.Op()] = true
	}
	for _, op := range model.EditOps() {
		if !written[op] {
			t.Errorf("no edit of op %q is emitted anywhere, so nothing proves it can be", op)
		}
	}
}

func TestEmitRefusesWhatIsNotAnEntity(t *testing.T) {
	if _, err := Emit(model.Project{}); err == nil {
		t.Error("emitting a project as an entity should fail")
	}
}

// Folding is only safe where a line break can be turned back into the single space it
// replaced, so these are the texts where it must decline to fold, or fold and still
// come back unchanged.
func TestFoldingNeverChangesTheText(t *testing.T) {
	for _, text := range []string{
		strings.Repeat("word ", 40) + "end",
		"two  spaces " + strings.Repeat("and more words ", 8) + "in it",
		strings.Repeat("x", 120) + " then a break " + strings.Repeat("y", 120),
		" leading space " + strings.Repeat("and words ", 12),
		strings.Repeat("trailing space ", 12) + "and then one ",
		"a line\nand another\n" + strings.Repeat("and a long one ", 8),
		"paragraph one\n\nparagraph two\n",
		"ending in blank lines\n\n\n",
		"\n",
	} {
		source := &model.Source{Slug: "s", Citations: map[string]model.Citation{"a": {Text: text}}}
		file, err := Emit(source)
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		reread := reparse(t, "s", source, file).(*model.Source)
		if got := reread.Citations["a"].Text; got != text {
			t.Errorf("text %q came back as %q\n%s", text, got, file)
		}
	}
}

// Folding earns its complexity by keeping a transcription readable in the file, so
// ordinary prose has to come out in lines a reader can follow.
func TestFoldedProseWrapsToReadableLines(t *testing.T) {
	file, err := Emit(&model.Source{Slug: "s", Citations: map[string]model.Citation{
		"a": {Text: strings.Repeat("some ordinary words of transcription ", 8) + "and an end."},
	}})
	if err != nil {
		t.Fatal(err)
	}
	folded := 0
	for _, line := range strings.Split(string(file), "\n") {
		if len(line) > lineWidth+len("      ") {
			t.Errorf("a line ran to %d characters: %q", len(line), line)
		}
		if strings.HasPrefix(line, "      ") {
			folded++
		}
	}
	if folded < 4 {
		t.Errorf("a transcription of 300 characters folded into %d lines\n%s", folded, file)
	}
}
