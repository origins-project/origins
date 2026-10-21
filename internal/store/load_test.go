package store

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

func write(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const projectFile = `project:
  name: Rossi family
  description: Descendants of Bartolomeo Rossi
  main-person: rossi-giovanni
origins:
  format-version: 1
`

// corpus writes a project file plus whatever the caller lists, and loads it.
func corpus(t *testing.T, files map[string]string) (*Corpus, error) {
	t.Helper()
	root := t.TempDir()
	write(t, root, ProjectFile, projectFile)
	for path, content := range files {
		write(t, root, path, content)
	}
	return Load(root)
}

func loaded(t *testing.T, files map[string]string) *Corpus {
	t.Helper()
	c, err := corpus(t, files)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return c
}

func TestLoadProject(t *testing.T) {
	c := loaded(t, nil)
	if c.Project.Name != "Rossi family" || c.Project.MainPerson != "rossi-giovanni" {
		t.Errorf("got %+v", c.Project)
	}
	if c.Project.FormatVersion != 1 {
		t.Errorf("format version = %d", c.Project.FormatVersion)
	}
}

// A bookmark is a kind's directory and a slug, a researcher has a name and cites
// nothing, and a tag is named.
func TestTheProjectRefusesWhatItCannotMean(t *testing.T) {
	for _, bad := range []string{
		"bookmarks: [person/rossi-giovanni]",
		"bookmarks: [rossi-giovanni]",
		"bookmarks: [people/Rossi Giovanni]",
		"researchers: [Anna Rossi]",
		"researchers: [{note: Compiled it}]",
		"researchers: [{name: Anna Rossi, addresses: [{city: Firenze, sources: [census-1871]}]}]",
		"researchers: [{name: Anna Rossi, email: anna@example.org}]",
		"tags: {To do: {colour: red}}",
		"tags: {'': {color: red}}",
	} {
		manifest := "project:\n  name: x\n  " + bad + "\norigins:\n  format-version: 1\n"
		if _, err := decodeProject([]byte(manifest)); err == nil {
			t.Errorf("%s: read", bad)
		}
	}
}

func TestLoadPerson(t *testing.T) {
	c := loaded(t, map[string]string{
		"people/rossi-giovanni/README.md": `---
name:
  given: Giovanni Battista
  call: Giovanni
  surname: Rossi
  sources: [parish-san-lorenzo-baptisms#1834-042 OPD]
also-known-as:
  - Gio. Rossi
  - given: Giovanni
    surname: Rosi
    type: aka
sex:
  value: M
  sources: [parish-san-lorenzo-baptisms#1834-042 OPD]
events:
  - type: birth
    date: 12 MAR 1834
    place: borgo-san-lorenzo
    sources: [parish-san-lorenzo-baptisms#1834-042]
  - type: death
    date: 3 NOV 1901
sources: [censimento-1861-firenze#f088]
ids:
  familysearch: XXXX-YYY
  wikitree: 1834
---

He appears as "Gio. Rossi" in the 1861 census.
`,
	})

	giovanni, ok := c.People["rossi-giovanni"]
	if !ok {
		t.Fatal("rossi-giovanni did not load")
	}
	name := giovanni.Name
	if name.Given != "Giovanni Battista" || name.Call != "Giovanni" || name.PrimarySurname().Name != "Rossi" ||
		len(name.Sources) != 1 || name.Sources[0].String() != "parish-san-lorenzo-baptisms#1834-042 OPD" {
		t.Errorf("got name %+v", name)
	}
	if giovanni.Sex.Value != model.Male || len(giovanni.Sex.Sources) != 1 {
		t.Errorf("sex = %+v", giovanni.Sex)
	}

	if len(giovanni.Sources) != 1 || giovanni.Sources[0].String() != "censimento-1861-firenze#f088" {
		t.Errorf("sources = %+v", giovanni.Sources)
	}
	// an identifier that looks like a number is text, as every other value is
	if !reflect.DeepEqual(giovanni.IDs, model.IDs{"familysearch": model.IdentifiersOf("XXXX-YYY"), "wikitree": model.IdentifiersOf("1834")}) {
		t.Errorf("ids = %+v", giovanni.IDs)
	}

	if len(giovanni.AlsoKnownAs) != 2 || giovanni.AlsoKnownAs[0].Text != "Gio. Rossi" ||
		giovanni.AlsoKnownAs[1].PrimarySurname().Name != "Rosi" || giovanni.AlsoKnownAs[1].Type != "aka" {
		t.Errorf("also-known-as = %+v", giovanni.AlsoKnownAs)
	}
	if len(giovanni.Events) != 2 {
		t.Fatalf("events = %+v", giovanni.Events)
	}
	birth := giovanni.Events[0]
	if birth.Type != "birth" || birth.Date.String() != "12 MAR 1834" || birth.Place != "borgo-san-lorenzo" {
		t.Errorf("birth = %+v", birth)
	}
	if len(birth.Sources) != 1 || birth.Sources[0].String() != "parish-san-lorenzo-baptisms#1834-042" {
		t.Errorf("birth sources = %+v", birth.Sources)
	}
	if !strings.HasPrefix(giovanni.Body, `He appears as "Gio. Rossi"`) {
		t.Errorf("body = %q", giovanni.Body)
	}
}

// A family lists its children in the order given, each a slug or a mapping, and a
// relation is one word or a word for each partner.
func TestLoadChildren(t *testing.T) {
	c := loaded(t, map[string]string{
		"families/galli-ines--rossi-pietro/README.md": `---
partners: [galli-ines, rossi-pietro]
children:
  - rossi-giovanni-1902
  - id: rossi-elena
    relation:
      rossi-pietro: adoptive
      galli-ines: step
  - id: unknown
    relation: natural
    status: challenged
    note: a third child the census counts
    sources: [parish-san-lorenzo-baptisms#1834-042]
  - name: Gemma Nocentini
    status: disproven
---
`,
	})
	cited := []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}}
	want := []model.Child{
		{ID: "rossi-giovanni-1902"},
		{ID: "rossi-elena", Relation: model.Relationship{Partners: []model.PartnerRelation{
			{Partner: "rossi-pietro", Relation: model.Adoptive}, {Partner: "galli-ines", Relation: model.Step},
		}}},
		{ID: model.Unknown, Relation: model.Relationship{Word: "natural"}, Status: model.Challenged,
			Note: "a third child the census counts", Sources: cited},
		{Name: model.PersonName{Text: "Gemma Nocentini"}, Status: model.Disproven},
	}
	if got := c.Families["galli-ines--rossi-pietro"].Children; !reflect.DeepEqual(got, want) {
		t.Errorf("children = %+v, want %+v", got, want)
	}
}

func TestLoadRejectsAChildThatIsNobody(t *testing.T) {
	for _, child := range []string{
		"none",
		"{id: rossi-elena, name: Elena Rossi}",
		"{relation: adoptive}",
		"{id: rossi-elena, status: likely}",
		"{id: rossi-elena, relation: {}}",
	} {
		_, err := corpus(t, map[string]string{
			"families/galli-ines--rossi-pietro/README.md": "---\npartners: [galli-ines, rossi-pietro]\nchildren: [" + child + "]\n---\n",
		})
		if err == nil {
			t.Errorf("%s read as a child", child)
		}
	}
}

func TestLoadDatedAlias(t *testing.T) {
	c := loaded(t, map[string]string{
		"people/bianchi-maria/README.md": `---
name:
  given: Maria
  surname: Bianchi
also-known-as:
  - Maria Bianca
  - given: Maria
    surname: Rossi
    type: married
    from: 1858
sex: F
---
`,
	})
	maria := c.People["bianchi-maria"]
	if len(maria.AlsoKnownAs) != 2 {
		t.Fatalf("got %+v", maria.AlsoKnownAs)
	}
	// an undated variant is a spelling to match on, a dated one is a name she bore
	if maria.AlsoKnownAs[0].Qualified() {
		t.Errorf("a clerk's misspelling should carry no dates: %+v", maria.AlsoKnownAs[0])
	}
	if got := maria.AlsoKnownAs[1]; !got.Qualified() || got.Type != "married" || got.From.String() != "1858" {
		t.Errorf("the married name should be dated: %+v", got)
	}
}

func TestLoadPlaceWithinBothWays(t *testing.T) {
	c := loaded(t, map[string]string{
		"places/firenze/README.md": `---
name: Firenze
also-known-as: [Florence, Florenz]
type: city
within: toscana
coordinates: 43.7696, 11.2558
---
`,
		"places/pian-del-mugnone/README.md": `---
name: Pian del Mugnone
type: hamlet
within:
  - place: granducato-di-toscana
    until: 1860
  - place: toscana
    from: 1861
until: 1890
---
`,
	})

	firenze := c.Places["firenze"]
	if len(firenze.Within) != 1 || firenze.Within[0].Place != "toscana" || firenze.Within[0].Qualified() {
		t.Errorf("a scalar within should read as one unqualified entry: %+v", firenze.Within)
	}
	if firenze.Coordinates == nil || firenze.Coordinates.Latitude != 43.7696 || firenze.Coordinates.Longitude != 11.2558 {
		t.Errorf("coordinates = %+v", firenze.Coordinates)
	}

	if firenze.Name.Text != "Firenze" || firenze.Name.Qualified() {
		t.Errorf("a scalar name should read as the name alone: %+v", firenze.Name)
	}

	pian := c.Places["pian-del-mugnone"]
	if len(pian.Within) != 2 {
		t.Fatalf("got %+v", pian.Within)
	}
	if pian.Within[0].Until.String() != "1860" || pian.Within[1].From.String() != "1861" {
		t.Errorf("jurisdictions = %+v", pian.Within)
	}
	if pian.Until.String() != "1890" {
		t.Errorf("a place that ceased to exist should carry until: %q", pian.Until)
	}
}

func TestLoadSourceCitations(t *testing.T) {
	c := loaded(t, map[string]string{
		"sources/parish-san-lorenzo-baptisms/README.md": `---
title: Parish of San Lorenzo — Baptismal Register 1820–1850
short-title: S. Lorenzo, Battesimi 4
type: parish-register
covers:
  - events:
      - baptism
    date: FROM 1820 TO 1850
    place: firenze
repositories:
  - repository: archivio-diocesano-di-firenze
    call-number: ADF/SL/B/4
  - repository: familysearch
    call-number: Film 1903581
    medium: microfilm
  - archivio-di-stato-di-firenze
citations:
  1834-042:
    event: baptism
    page: c. 88r, n. 112
    date: 14 MAR 1834
    text: >
      Giovanni Battista, son of Antonio Rossi and Lucia Verdi,
      baptised 14 March 1834.
    with:
      - person: rossi-giovanni-battista
        role: child
      - person: rossi-antonio
        role: father
    images:
      - file: scans/1834-042.jpg
        region: [0.08, 0.41, 0.93, 0.62]
      - scans/1834-042.jpg
---
`,
		"sources/parish-san-lorenzo-baptisms/scans/1834-042.jpg": "",
	})

	source := c.Sources["parish-san-lorenzo-baptisms"]
	if source.ShortTitle != "S. Lorenzo, Battesimi 4" {
		t.Errorf("short title = %q", source.ShortTitle)
	}
	period, _ := model.ParseDate("FROM 1820 TO 1850")
	covers := []model.Coverage{{Events: []string{"baptism"}, Date: period, Place: "firenze"}}
	if !reflect.DeepEqual(source.Covers, covers) {
		t.Errorf("covers = %+v", source.Covers)
	}
	held := []model.Holding{
		{Repository: "archivio-diocesano-di-firenze", CallNumber: "ADF/SL/B/4"},
		{Repository: "familysearch", CallNumber: "Film 1903581", Medium: "microfilm"},
		{Repository: "archivio-di-stato-di-firenze"},
	}
	if !reflect.DeepEqual(source.Repositories, held) {
		t.Errorf("repositories = %+v", source.Repositories)
	}
	citation, ok := source.Citations["1834-042"]
	if !ok {
		t.Fatalf("citations = %+v", source.Citations)
	}
	if !strings.Contains(citation.Text, "baptised 14 March 1834") {
		t.Errorf("text = %q", citation.Text)
	}
	if citation.Page != "c. 88r, n. 112" || citation.Date.String() != "14 MAR 1834" {
		t.Errorf("page = %q, date = %v", citation.Page, citation.Date)
	}
	if citation.Event != "baptism" {
		t.Errorf("event = %q", citation.Event)
	}
	with := []model.Named{{Person: "rossi-giovanni-battista", Role: "child"}, {Person: "rossi-antonio", Role: "father"}}
	if !reflect.DeepEqual(citation.With, with) {
		t.Errorf("with = %+v", citation.With)
	}
	want := []model.Media{
		{File: "scans/1834-042.jpg", Region: &model.Region{Left: 0.08, Top: 0.41, Right: 0.93, Bottom: 0.62}},
		{File: "scans/1834-042.jpg"},
	}
	if !reflect.DeepEqual(citation.Images, want) {
		t.Errorf("images = %+v", citation.Images)
	}
	if !slices.Equal(source.Files, []string{"scans/1834-042.jpg"}) {
		t.Errorf("attachments = %q", source.Files)
	}
}

func TestLoadRepository(t *testing.T) {
	c := loaded(t, map[string]string{
		"repositories/archivio-diocesano-di-firenze/README.md": `---
name: Archivio Diocesano di Firenze
type: archive
addresses:
  - |
    Piazza San Giovanni
    Firenze
links: [https://www.example.org/archivio]
---

Open Tuesday and Thursday mornings.
`,
	})
	got := c.Repositories["archivio-diocesano-di-firenze"]
	want := &model.Repository{
		Slug:      "archivio-diocesano-di-firenze",
		Name:      "Archivio Diocesano di Firenze",
		Type:      "archive",
		Addresses: []model.Address{{Lines: "Piazza San Giovanni\nFirenze\n"}},
		Links:     model.LinksOf("https://www.example.org/archivio"),
		Body:      "\nOpen Tuesday and Thursday mornings.\n",
	}
	if got == nil || got.Body == "" {
		t.Fatalf("repositories = %+v", c.Repositories)
	}
	got.Body, want.Body = strings.TrimSpace(got.Body), strings.TrimSpace(want.Body)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("repository = %+v", got)
	}
}

// A region is four numbers and nothing else, and a number that is no fraction of
// anything is refused where it is read. Whether the four lie within the picture is
// check's to say.
func TestARegionIsFourNumbers(t *testing.T) {
	for _, region := range []string{"[0.1, 0.2, 0.3]", "0.1, 0.2, 0.3, 0.4", "[0.1, 0.2, 0.3, top]", "[0.1, 0.2, 0.3, .nan]"} {
		_, err := corpus(t, map[string]string{"sources/register/README.md": "---\ncitations:\n  a:\n" +
			"    images:\n      - file: scans/a.jpg\n        region: " + region + "\n---\n"})
		if err == nil || !strings.Contains(err.Error(), "four fractions") {
			t.Errorf("region %s: %v", region, err)
		}
	}
}

// A media entry says what one of the record's files is, so it names one, as its path or
// with what is said of it, and somebody in a picture is a person's slug, with the part
// of the picture that is them or not.
func TestMediaIsReadAsItIsWritten(t *testing.T) {
	c := loaded(t, map[string]string{
		"families/bianchi-maria--rossi-giovanni/README.md": `---
partners: [bianchi-maria, rossi-giovanni]
events:
  - type: marriage
    media:
      - file: ../../people/rossi-giovanni/photos/1858.jpg
        region: [0, 0.5, 1, 1]
        note: The church steps.
media:
  - file: photos/1858-wedding-group.jpg
    type: photo
    title: The wedding party
    date: 21 APR 1858
    place: firenze
    region: [0, 0, 1, 0.9]
    depicts:
      - person: bianchi-maria
        region: [0.41, 0.12, 0.58, 0.47]
      - rossi-giovanni
    sources: [family-album#p3]
    ids:
      flickr: '5123'
  - photos/1858-verso.jpg
---
`,
		"families/bianchi-maria--rossi-giovanni/photos/1858-wedding-group.jpg": "",
		"families/bianchi-maria--rossi-giovanni/photos/1858-verso.jpg":         "",
		"people/bianchi-maria/README.md":                                       "---\nportrait:\n  file: photos/group.jpg\n  region: [0.1, 0.2, 0.3, 0.4]\n---\n",
		"people/bianchi-maria/photos/group.jpg":                                "",
	})
	want := []model.Media{{
		File: "photos/1858-wedding-group.jpg", Type: "photo", Title: "The wedding party", Date: mustDate("21 APR 1858"),
		Place: "firenze", Region: &model.Region{Right: 1, Bottom: 0.9},
		Depicts: []model.Depicted{
			{Person: "bianchi-maria", Region: &model.Region{Left: 0.41, Top: 0.12, Right: 0.58, Bottom: 0.47}},
			{Person: "rossi-giovanni"},
		},
		Sources: []model.CitationRef{{Source: "family-album", Anchor: "p3"}},
		IDs:     model.IDs{"flickr": {{ID: "5123"}}},
	}, {File: "photos/1858-verso.jpg"}}
	family := c.Families["bianchi-maria--rossi-giovanni"]
	if !reflect.DeepEqual(family.Media, want) {
		t.Errorf("media = %+v", family.Media)
	}
	steps := []model.Media{{File: "../../people/rossi-giovanni/photos/1858.jpg",
		Region: &model.Region{Top: 0.5, Right: 1, Bottom: 1}, Note: "The church steps."}}
	if got := family.Events[0].Media; !reflect.DeepEqual(got, steps) {
		t.Errorf("the marriage's media = %+v", got)
	}
	portrait := model.Image{File: "photos/group.jpg", Region: &model.Region{Left: 0.1, Top: 0.2, Right: 0.3, Bottom: 0.4}}
	if got := c.People["bianchi-maria"].Portrait; !reflect.DeepEqual(got, portrait) {
		t.Errorf("portrait = %+v", got)
	}

	for media, refusal := range map[string]string{
		"  - title: The wedding party\n": "names the file",
		"  - [photos/1858.jpg]\n":        "a path, or a mapping with file",
		"  - ''\n":                       "names the file",
		"  - file: a.jpg\n    caption: The wedding party\n":                       `unknown field "caption"`,
		"  - file: a.jpg\n    depicts: [Maria Bianchi]\n":                         "slug",
		"  - file: a.jpg\n    depicts:\n      - person: a\n        age: 18\n":     `unknown field "age"`,
		"  - file: a.jpg\n    depicts:\n      - person: a\n        region: [1]\n": "four fractions",
		"  - file: ../../people/a/a.jpg\n    title: The wedding party\n":          "says only its own",
		"  - file: ../b/a.jpg\n    depicts: [a]\n":                                "says only its own",
	} {
		_, err := corpus(t, map[string]string{"repositories/archivio/README.md": "---\nmedia:\n" + media + "---\n"})
		if err == nil || !strings.Contains(err.Error(), refusal) {
			t.Errorf("%q: %v, want it refused for %q", media, err, refusal)
		}
	}
}

func TestLoadAttachments(t *testing.T) {
	c := loaded(t, map[string]string{
		"people/rossi-giovanni/README.md":                            "---\nname: Giovanni\n---\n",
		"people/rossi-giovanni/photos/1890-portrait.jpg":             "",
		"people/rossi-giovanni/documents/1901-death-certificate.jpg": "",
		"people/rossi-giovanni/.DS_Store":                            "",
		"people/rossi-giovanni/photos/.thumbnails/1890-portrait.jpg": "",
		"places/firenze/README.md":                                   "---\nname: Firenze\n---\n",
	})
	want := []string{"documents/1901-death-certificate.jpg", "photos/1890-portrait.jpg"}
	if got := c.People["rossi-giovanni"].Files; !slices.Equal(got, want) {
		t.Errorf("attachments = %q, want %q", got, want)
	}
	if got := c.Places["firenze"].Files; got != nil {
		t.Errorf("a record with nothing beside it has no attachments, got %q", got)
	}
}

// A place's own name is dated where the place went by another before it, and a name
// says how it was put into another script. Coordinates a record gave that are not two
// angles are kept as its words.
func TestLoadPlaceNamesAndCoordinatesAsWords(t *testing.T) {
	c := loaded(t, map[string]string{
		"places/ivano-frankivsk/README.md": `---
name:
  name: Ivano-Frankivsk
  lang: uk-Latn
  from: 1962
also-known-as:
  - name: Stanislav
    from: 1939
    until: 1962
  - name: Stanisławów
    lang: pl
    until: 1939
  - name: Ivano-Frankovsk
    lang: ru-Latn
    type: romanized
coordinates: (N48.55, E24.43)
---
`,
	})
	place := c.Places["ivano-frankivsk"]
	if got := place.Name; got.Text != "Ivano-Frankivsk" || got.Lang != "uk-Latn" || got.From.String() != "1962" {
		t.Errorf("name = %+v", got)
	}
	if got := place.AlsoKnownAs[2]; got.Type != "romanized" || got.Lang != "ru-Latn" {
		t.Errorf("the romanized name = %+v", got)
	}
	if at := place.Coordinates; at == nil || at.Placed() || at.Wording != "N48.55, E24.43" {
		t.Errorf("coordinates = %+v", at)
	}
}

// A file where records are kept that is neither a record nor beside one is refused by
// name, and the records around it still load. What is beside a record is its
// attachment, a name beginning with a dot is nobody's to read, and a file outside the
// five directories is the researcher's own business.
func TestLoadRefusesAFileThatBelongsToNoRecord(t *testing.T) {
	c, err := corpus(t, map[string]string{
		"people/rossi-giovanni/README.md":            "---\nname: Giovanni\n---\n",
		"people/rossi-giovanni/photos/1890.jpg":      "",
		"people/rossi/scan-unfiled.jpg":              "",
		"people/README.md":                           "the Rossis, filed by surname\n",
		"people/.gitkeep":                            "",
		"sources/registers/notes.txt":                "",
		"hypotheses/.DS_Store":                       "",
		"notes.txt":                                  "",
		"research/people/rossi-giovanni-letters.txt": "",
	})
	if err == nil {
		t.Fatal("files belonging to no record loaded without a word")
	}
	var got []string
	for _, problem := range strings.Split(err.Error(), "\n") {
		got = append(got, strings.TrimSuffix(problem, ": is neither a record nor beside one"))
	}
	slices.Sort(got)
	want := []string{"people/README.md", "people/rossi/scan-unfiled.jpg", "sources/registers/notes.txt"}
	if !slices.Equal(got, want) {
		t.Errorf("refused %q, want %q", got, want)
	}
	if _, ok := c.People["rossi-giovanni"]; !ok {
		t.Error("a file belonging to no record stopped the record beside it loading")
	}
}

// A record is kept in one place only, and a record anywhere else is refused by name
// rather than read from where it is, with where it belongs. The records that are where
// they belong still load.
func TestLoadRefusesARecordOutOfPlace(t *testing.T) {
	c, err := corpus(t, map[string]string{
		"people/rossi-anna/README.md":            "---\nname: Anna\n---\n",
		"people/rossi-giovanni.md":               "---\nname: Giovanni\n---\n",
		"people/bianchi/bianchi-maria/README.md": "---\nname: Maria\n---\n",
		"places/toscana/firenze.md":              "---\nname: Firenze\n---\n",
	})
	if err == nil {
		t.Fatal("records out of place loaded without a word")
	}
	for _, want := range []string{
		"people/rossi-giovanni.md: a record is kept as people/rossi-giovanni/README.md",
		"people/bianchi/bianchi-maria/README.md: a record is kept as people/bianchi-maria/README.md",
		"places/toscana/firenze.md: a record is kept as places/firenze/README.md",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q:\n%v", want, err)
		}
	}
	if len(c.People) != 1 || c.People["rossi-anna"] == nil || len(c.Places) != 0 {
		t.Errorf("loaded %d people and %d places, want rossi-anna alone", len(c.People), len(c.Places))
	}
}

func TestLoadHypothesis(t *testing.T) {
	c := loaded(t, map[string]string{
		"hypotheses/annunziata-born-1844/README.md": `---
status: rejected
base: 8f78553
confidence: possible
supports: [censimento-1861-firenze#f214]
contradicts: [parish-san-lorenzo-baptisms#1842-007]
edits:
  - op: set
    entity: person/rossi-annunziata
    path: events[0].date
    value: 1844
---

# Was Annunziata born in 1844 rather than 1842?

I rejected this.
`,
		"hypotheses/unrecorded-child-1845/README.md": `---
status: open
base: 8f78553
confidence: possible
supports: [parish-san-lorenzo-baptisms#1842-007]
edits:
  - op: create
    entity: person/rossi-unknown-1845
    value:
      name: Unknown Rossi
      surname: Rossi
      sex: U
---
`,
	})

	rejected := c.Hypotheses["annunziata-born-1844"]
	if rejected.Status != model.Rejected || rejected.Base != "8f78553" || rejected.Confidence != model.Possible {
		t.Errorf("got %+v", rejected)
	}
	if len(rejected.Edits) != 1 {
		t.Fatalf("edits = %+v", rejected.Edits)
	}
	set, ok := rejected.Edits[0].(model.Set)
	if !ok {
		t.Fatalf("edit is %T", rejected.Edits[0])
	}
	if set.Entity.String() != "person/rossi-annunziata" || set.Path != "events[0].date" {
		t.Errorf("got %+v", set)
	}
	if len(rejected.Supports) != 1 || len(rejected.Contradicts) != 1 {
		t.Errorf("supports %+v contradicts %+v", rejected.Supports, rejected.Contradicts)
	}
	if !strings.Contains(rejected.Body, "I rejected this.") {
		t.Errorf("body = %q", rejected.Body)
	}

	open := c.Hypotheses["unrecorded-child-1845"]
	create, ok := open.Edits[0].(model.Create)
	if !ok {
		t.Fatalf("edit is %T", open.Edits[0])
	}
	if create.Value["surname"] != "Rossi" {
		t.Errorf("created fields = %+v", create.Value)
	}
}

func TestLoadCollectsProblemsWithoutLosingTheRest(t *testing.T) {
	c, err := corpus(t, map[string]string{
		"people/rossi-giovanni/README.md": "---\nname: Giovanni\n---\n",
		"people/broken/README.md":         "---\nname: [unterminated\n---\n",
		"people/no-fence/README.md":       "nothing here\n",
		"people/bad-date/README.md":       "---\nevents:\n  - type: birth\n    date: 30 FEB 1834\n---\n",
	})
	if err == nil {
		t.Fatal("three malformed files should have been reported")
	}
	if _, ok := c.People["rossi-giovanni"]; !ok {
		t.Error("a malformed file stopped the rest of the corpus from loading")
	}

	report := err.Error()
	for _, want := range []string{"broken", "no-fence", "bad-date"} {
		if !strings.Contains(report, want) {
			t.Errorf("the report does not name %q:\n%s", want, report)
		}
	}
	if !strings.Contains(report, "FEB has no day 30") {
		t.Errorf("the report does not say what is wrong with the date:\n%s", report)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	_, err := corpus(t, map[string]string{
		"people/rossi-giovanni/README.md": "---\nname: Giovanni\nnickname: Gio\n---\n",
	})
	if err == nil || !strings.Contains(err.Error(), `unknown field "nickname"`) {
		t.Errorf("an unrecognised field would be dropped on the next write, so it must be reported: %v", err)
	}
}

func TestLoadCountsLinesFromTheTopOfTheFile(t *testing.T) {
	// the reader has the whole file open, fence and all, so a line number that counts
	// from the first line of the frontmatter sends them to the wrong place
	_, err := corpus(t, map[string]string{
		"people/rossi-anna/README.md":  "---\nevents:\n  - type: birth\n    date: 30 FEB 1834\n---\n",
		"people/rossi-paolo/README.md": "---\nname: Paolo\nnickname: Paolino\n---\n",
	})
	if err == nil {
		t.Fatal("both files are malformed")
	}
	report := err.Error()
	if !strings.Contains(report, "line 4: parse date") {
		t.Errorf("the date is on line 4 of its file:\n%s", report)
	}
	if !strings.Contains(report, `line 3: unknown field "nickname"`) {
		t.Errorf("the unknown field is on line 3 of its file:\n%s", report)
	}
}

// A flag is a word the field accepts, not a YAML boolean, and false is the field left
// out rather than a second way of writing it.
func TestAFlagIsTheWordTrue(t *testing.T) {
	c := loaded(t, map[string]string{"families/a--b/README.md": "---\nprivate: true\npartners: [a, b]\n---\n"})
	if !c.Families["a--b"].Private {
		t.Error("a family marked private read as not private")
	}
	for _, word := range []string{"yes", "on", "1", "false", "True"} {
		_, err := corpus(t, map[string]string{"places/firenze/README.md": "---\nprivate: " + word + "\nname: Firenze\n---\n"})
		if err == nil || !strings.Contains(err.Error(), "line 2: \""+word+"\" is not a flag") {
			t.Errorf("private: %s read, or was refused for the wrong reason: %v", word, err)
		}
	}
}

// An event, another name and a picture are marked in their own mapping. The name a
// person is filed under and a name an event gives stand for somebody, and are hidden
// with whoever or whatever they stand for, so neither takes a mark of its own.
func TestAnEntryIsMarkedPrivate(t *testing.T) {
	c := loaded(t, map[string]string{
		"people/rossi-anna/README.md": "---\nname: Anna Rossi\nalso-known-as:\n  - Annetta\n  - private: true\n    given: Nina\n" +
			"events:\n  - private: true\n    type: residence\nmedia:\n  - private: true\n    file: photos/a.jpg\n---\n",
		"people/rossi-anna/photos/a.jpg": "",
	})
	anna := c.People["rossi-anna"]
	if anna.AlsoKnownAs[0].Private || !anna.AlsoKnownAs[1].Private || !anna.Events[0].Private || !anna.Media[0].Private {
		t.Errorf("also-known-as = %+v, events = %+v, media = %+v", anna.AlsoKnownAs, anna.Events, anna.Media)
	}

	for what, file := range map[string]string{
		"the name a person is filed under": "---\nname:\n  private: true\n  given: Anna\n---\n",
		"a name an event gives": "---\nevents:\n  - type: baptism\n    with:\n      - name:\n          private: true\n" +
			"          given: Pietro\n        role: godfather\n---\n",
	} {
		_, err := corpus(t, map[string]string{"people/rossi-anna/README.md": file})
		if err == nil || !strings.Contains(err.Error(), "only a name in also-known-as is private") {
			t.Errorf("%s was marked private: %v", what, err)
		}
	}
	if _, err := corpus(t, map[string]string{"people/rossi-anna/README.md": "---\nalso-known-as:\n  - private: true\n---\n"}); err == nil ||
		!strings.Contains(err.Error(), "a name has its text or at least one of") {
		t.Errorf("a name that is only a mark was read: %v", err)
	}
}

// living is the word true or false, or left out for the rule to decide, and YAML's other
// booleans are refused as a flag's are.
func TestLivingIsTrueOrFalse(t *testing.T) {
	c := loaded(t, map[string]string{
		"people/a/README.md": "---\nliving: true\n---\n",
		"people/b/README.md": "---\nliving: false\n---\n",
		"people/c/README.md": "---\nname: C\n---\n",
	})
	for slug, want := range map[model.Slug]model.Living{"a": model.LivingTrue, "b": model.LivingFalse, "c": model.LivingPresumed} {
		if got := c.People[slug].Living; got != want {
			t.Errorf("%s reads living %q, want %q", slug, got, want)
		}
	}
	for _, word := range []string{"yes", "no", "1", "True"} {
		_, err := corpus(t, map[string]string{"people/a/README.md": "---\nliving: " + word + "\n---\n"})
		if err == nil || !strings.Contains(err.Error(), "line 2: living \""+word+"\": expected true or false") {
			t.Errorf("living: %s read, or was refused for the wrong reason: %v", word, err)
		}
	}
}

func TestLoadRejectsAnUnknownQualifier(t *testing.T) {
	_, err := corpus(t, map[string]string{
		"families/moretti-giulia--unknown/README.md": "---\npartners: [moretti-giulia, unknown]\nchildren:\n  - id: rossi-elena\n    kind: birth\n---\n",
	})
	if err == nil || !strings.Contains(err.Error(), "kind") {
		t.Errorf("got %v", err)
	}
}

func TestLoadWithoutAnOptionalDirectory(t *testing.T) {
	// a corpus with no hypotheses yet is an ordinary corpus
	c := loaded(t, map[string]string{
		"people/rossi-giovanni/README.md": "---\nname: Giovanni\n---\n",
	})
	if len(c.Hypotheses) != 0 {
		t.Errorf("got %d hypotheses", len(c.Hypotheses))
	}
}

func TestLoadReportsAMissingProjectFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "people/rossi-giovanni/README.md", "---\nname: Giovanni\n---\n")
	c, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), ProjectFile) {
		t.Errorf("got %v", err)
	}
	if _, ok := c.People["rossi-giovanni"]; !ok {
		t.Error("a missing project file should not stop the entities loading")
	}
}

func TestAttachmentResolvesOnlyWhatIsAttached(t *testing.T) {
	c := loaded(t, map[string]string{
		"people/rossi-giovanni/README.md":           "---\nname: Giovanni\n---\n",
		"people/rossi-giovanni/photos/portrait.jpg": "",
		"people/bianchi-maria/README.md":            "---\nname: Maria\n---\n",
	})

	path, err := c.Attachment("rossi-giovanni", "photos/portrait.jpg")
	if err != nil {
		t.Fatalf("the portrait should resolve: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(path), "people/rossi-giovanni/photos/portrait.jpg") {
		t.Errorf("resolved to %s", path)
	}

	for _, name := range []string{
		"../bianchi-maria/README.md", // climbing out of the entity
		"../../origins.yaml",         // climbing out of the corpus
		"README.md",                  // the entity's own file is not an attachment
		"photos/missing.jpg",         // nothing of that name was found
	} {
		if _, err := c.Attachment("rossi-giovanni", name); err == nil {
			t.Errorf("%q should not resolve", name)
		}
	}
	if _, err := c.Attachment("nobody", "photos/portrait.jpg"); err == nil {
		t.Error("an unknown entity should not resolve")
	}
}

// A link beside a record leads wherever it was made to, so it is not served as the file
// it is named like, and a corpus somebody else wrote reaches nothing outside itself.
func TestAttachmentRefusesALink(t *testing.T) {
	root := t.TempDir()
	write(t, root, ProjectFile, projectFile)
	write(t, root, "people/rossi-giovanni/README.md", "---\nname: Giovanni\n---\n")
	write(t, root, "elsewhere/secret.txt", "")
	link := filepath.Join(root, "people", "rossi-giovanni", "portrait.jpg")
	if err := os.Symlink(filepath.Join(root, "elsewhere", "secret.txt"), link); err != nil {
		t.Skipf("no symbolic links here: %v", err)
	}
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Attachment("rossi-giovanni", "portrait.jpg"); err == nil {
		t.Error("a link beside a record resolved")
	}
}

// What the corpus is made of is asked about a file before git has it, so it is
// answered from the path and the directories around it: every record and everything
// beside one, and neither a stray file, nor a README where no record is, nor a note
// beside the corpus, nor a name with a dot in front of it.
func TestMadeOfIsTheRecordsAndWhatIsBesideThem(t *testing.T) {
	root := t.TempDir()
	made := map[string]bool{
		"origins.yaml":                          true,
		"people/rossi-anna/README.md":           true,
		"people/rossi-giovanni/README.md":       true,
		"people/rossi-giovanni/photos/atto.pdf": true,
		"people/rossi-giovanni/.DS_Store":       false,
		"people/rossi/rossi-giovanni/README.md": false,
		"people/rossi-paolo.md":                 false,
		"people/README.md":                      false,
		"people/rossi/scan.jpg":                 false,
		"sources/archivio/.cache/index":         false,
		"notes.txt":                             false,
		"drafts/rossi-giovanni.md":              false,
	}
	for file := range made {
		write(t, root, file, "")
	}
	for file, want := range made {
		if got := MadeOf(root, file); got != want {
			t.Errorf("MadeOf(%s) = %v, want %v", file, got, want)
		}
	}
}

// A record with nothing in it is legal, being what somebody known only to have existed
// looks like, and what origins writes for one has to read back as one.
func TestAnEmptyRecordReadsBack(t *testing.T) {
	empty := map[string]any{
		"people/nobody":     &model.Person{},
		"families/a--b":     &model.Family{},
		"places/nowhere":    &model.Place{},
		"sources/unwritten": &model.Source{},
		"hypotheses/idle":   &model.Hypothesis{},
	}
	files := map[string]string{}
	for dir, entity := range empty {
		written, err := Emit(entity)
		if err != nil {
			t.Fatal(err)
		}
		files[dir+"/README.md"] = string(written)
	}
	loaded, err := corpus(t, files)
	if err != nil {
		t.Fatal(err)
	}
	for dir := range empty {
		if slug := model.Slug(filepath.Base(dir)); loaded.Entity(slug) == nil {
			t.Errorf("%s did not read back", dir)
		}
	}
}
