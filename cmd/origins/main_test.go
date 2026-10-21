package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"image"
	"image/png"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func corpusDir(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func invoke(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errs bytes.Buffer
	code = run(args, strings.NewReader(""), &out, &errs)
	return code, out.String(), errs.String()
}

const sound = `project:
  name: Rossi family
  main-person: rossi-giovanni
origins:
  format-version: 1
`

func TestCheckExitsZeroOnACleanCorpus(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, map[string]string{
		"origins.yaml":                    sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\n---\n",
	})
	code, stdout, stderr := invoke(t, "check", root)
	if code != exitOK {
		t.Errorf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "0 error(s)") {
		t.Errorf("stdout = %q", stdout)
	}
}

func TestCheckExitsOneOnAContradiction(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, map[string]string{
		"origins.yaml": sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\n" +
			"sources:\n  - nowhere#p-1\n---\n",
	})
	code, stdout, _ := invoke(t, "check", root)
	if code != exitFindings {
		t.Errorf("exit %d, want %d", code, exitFindings)
	}
	if !strings.Contains(stdout, "dangling-source") {
		t.Errorf("stdout = %q", stdout)
	}
}

// A scan dropped among the records and never filed fails the run, and a change is
// refused until it is filed beside the record it belongs to or taken out from among
// the records.
func TestAFileThatBelongsToNoRecordIsRefused(t *testing.T) {
	t.Parallel()
	root := inARepo(t, map[string]string{
		"origins.yaml":                    sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\n---\n",
		"people/scan-0042.jpg":            "",
	})
	const refused = "people/scan-0042.jpg: is neither a record nor beside one"
	code, _, stderr := invoke(t, "check", root)
	if code != exitFindings {
		t.Errorf("check exit %d, want %d", code, exitFindings)
	}
	if !strings.Contains(stderr, refused) {
		t.Errorf("check stderr = %q", stderr)
	}
	code, _, stderr = invoke(t, "set", "-C", root, "person/rossi-giovanni", "name", "Giovanni")
	if code == exitOK {
		t.Error("a change was made to a corpus holding a file that belongs to no record")
	}
	if !strings.Contains(stderr, refused) {
		t.Errorf("set stderr = %q", stderr)
	}
}

// A corpus written by a later origins is read, and the reader is told what it is being
// read as; it is not changed, since every change re-emits what it touches and would put
// it back into the earlier format.
func TestACorpusInALaterFormatIsReadAndNotChanged(t *testing.T) {
	t.Parallel()
	files := theRossis()
	files["origins.yaml"] = strings.Replace(sound, "format-version: 1", "format-version: 2", 1)
	root := inARepo(t, files)
	before := read(t, root, "people/rossi-giovanni/README.md")

	code, stdout, stderr := invoke(t, "check", root)
	if code != exitOK {
		t.Errorf("check exited %d\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "format-version 2, which is later than the 1") {
		t.Errorf("check did not say what it read the corpus as:\n%s", stderr)
	}

	code, stdout, stderr = invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossini")
	if code != exitMisuse {
		t.Errorf("set exited %d, want %d\n%s%s", code, exitMisuse, stdout, stderr)
	}
	if !strings.Contains(stderr, "an origins that writes format-version 2 is needed") {
		t.Errorf("the refusal reads:\n%s", stderr)
	}
	if after := read(t, root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the file was rewritten:\n%s", after)
	}
}

func TestCheckExitsZeroWhenOnlyWarningsRemain(t *testing.T) {
	t.Parallel()
	// a corpus can be entirely correct and still be full of research still to do
	root := corpusDir(t, map[string]string{
		"origins.yaml": sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\n" +
			"events:\n  - type: birth\n    date: 1834\n---\n",
	})
	code, stdout, _ := invoke(t, "check", root)
	if code != exitOK {
		t.Errorf("exit %d, want %d\n%s", code, exitOK, stdout)
	}
	if !strings.Contains(stdout, "unsourced-event") {
		t.Errorf("stdout = %q", stdout)
	}

	code, stdout, _ = invoke(t, "check", "--errors", root)
	if code != exitOK || strings.Contains(stdout, "unsourced-event") {
		t.Errorf("--errors should leave warnings out: exit %d, stdout %q", code, stdout)
	}
}

func TestCheckJSON(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, map[string]string{
		"origins.yaml": sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\n" +
			"sources:\n  - nowhere#p-1\n---\n",
	})
	code, stdout, _ := invoke(t, "check", "--json", root)
	if code != exitFindings {
		t.Errorf("exit %d", code)
	}

	var report api.Validation
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if report.Errors != 1 || len(report.Findings) != 1 {
		t.Fatalf("got %+v", report)
	}
	finding := report.Findings[0]
	if finding.Rule != "dangling-source" || finding.Entity != "person/rossi-giovanni" || finding.Path != "sources[0]" {
		t.Errorf("got %+v", finding)
	}
	if finding.Message == "" || finding.Severity != "error" {
		t.Errorf("got %+v", finding)
	}
}

func TestLoadJSON(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, map[string]string{
		"origins.yaml":                    sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\n---\n",
		"places/firenze/README.md":        "---\nname: Firenze\n---\n",
	})
	code, stdout, _ := invoke(t, "load", "--json", root)
	if code != exitOK {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
	var report api.Summary
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if report.Counts.People != 1 || report.Counts.Places != 1 {
		t.Errorf("got %+v", report.Counts)
	}
	if report.Project.Name != "Rossi family" || len(report.Problems) != 0 {
		t.Errorf("got %+v", report)
	}
}

func TestMalformedCorpusIsReportedNotIgnored(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, map[string]string{
		"origins.yaml":                    sound,
		"people/rossi-giovanni/README.md": "---\nname: Giovanni\nnickname: Gio\n---\n",
	})
	for _, command := range []string{"load", "check"} {
		code, _, stderr := invoke(t, command, root)
		if code != exitFindings {
			t.Errorf("%s exited %d", command, code)
		}
		if !strings.Contains(stderr, "nickname") {
			t.Errorf("%s stderr = %q", command, stderr)
		}
	}
}

func TestMisuse(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{},
		{"fly"},
		{"check", "one", "two"},
		{"load", "--nonsense"},
	} {
		if code, _, _ := invoke(t, args...); code != exitMisuse {
			t.Errorf("%v exited %d, want %d", args, code, exitMisuse)
		}
	}
}

func TestVersionAndHelp(t *testing.T) {
	t.Parallel()
	if code, stdout, _ := invoke(t, "version"); code != exitOK || !strings.HasPrefix(stdout, "origins "+api.Version+"\n") {
		t.Errorf("exit %d, stdout %q", code, stdout)
	}
	if code, stdout, _ := invoke(t, "help"); code != exitOK || !strings.Contains(stdout, helpOf("check").summary) {
		t.Errorf("exit %d, stdout %q", code, stdout)
	}
}

// schema prints the schema of one kind, or every one by name, and names the ones there
// are when asked for one there is not.
func TestSchemaDescribesAKindOrEvery(t *testing.T) {
	t.Parallel()
	code, stdout, _ := invoke(t, "schema", "person")
	var person map[string]any
	if err := json.Unmarshal([]byte(stdout), &person); code != exitOK || err != nil || person["$id"] != "person.json" {
		t.Errorf("schema person: exit %d, %v, %.80q", code, err, stdout)
	}
	code, stdout, _ = invoke(t, "schema")
	var every map[string]map[string]any
	if err := json.Unmarshal([]byte(stdout), &every); code != exitOK || err != nil || every["project"]["$id"] != "project.json" || len(every) != 9 {
		t.Errorf("schema: exit %d, %v, %d schemas", code, err, len(every))
	}
	if code, _, stderr := invoke(t, "schema", "people"); code != exitMisuse || !strings.Contains(stderr, "person") {
		t.Errorf("schema people: exit %d, %q", code, stderr)
	}
}

func TestFmtRewritesACorpusAndThenHasNothingToSay(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, map[string]string{
		"origins.yaml": sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\n" +
			"events:\n  - type: christening\n    with: [{person: rossi-antonio, role: godfather}]\n---\n",
		"people/rossi-antonio/README.md": "---\nname: Antonio Rossi\nsex: M\n---\n",
	})

	code, stdout, stderr := invoke(t, "fmt", root)
	if code != exitOK {
		t.Fatalf("fmt exited %d: %s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "people/rossi-giovanni/README.md") {
		t.Errorf("fmt did not report the file it rewrote: %q", stdout)
	}

	code, stdout, _ = invoke(t, "fmt", root)
	if code != exitOK || stdout != "" {
		t.Errorf("a formatted corpus still had %d changes: %q", code, stdout)
	}
}

// --list is a question, and a corpus that is not canonical is a finding, so it can be
// asked in a hook or a build without being allowed to change anything.
func TestFmtListReportsWithoutWriting(t *testing.T) {
	t.Parallel()
	scruffy := "---\nname:\n  given: Giovanni\n  surname: Rossi\nevents:\n  - type: christening\n" +
		"    with: [{person: rossi-antonio, role: godfather}]\n---\n"
	root := corpusDir(t, map[string]string{
		"origins.yaml":                    sound,
		"people/rossi-giovanni/README.md": scruffy,
		"people/rossi-antonio/README.md":  "---\nname: Antonio Rossi\nsex: M\n---\n",
	})

	code, stdout, stderr := invoke(t, "fmt", "--list", root)
	if code != exitFindings {
		t.Errorf("fmt --list exited %d on a corpus that is not canonical: %s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "people/rossi-giovanni/README.md") {
		t.Errorf("fmt --list did not name the file: %q", stdout)
	}
	left, err := os.ReadFile(filepath.Join(root, "people", "rossi-giovanni", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(left) != scruffy {
		t.Errorf("fmt --list rewrote the file:\n%s", left)
	}

	invoke(t, "fmt", root)
	if code, stdout, _ := invoke(t, "fmt", "--list", root); code != exitOK || stdout != "" {
		t.Errorf("fmt --list exited %d with %q on a formatted corpus", code, stdout)
	}
}

func TestFmtJSON(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, map[string]string{
		"origins.yaml":                    sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\nalso-known-as:\n  - Gio. Rossi\n---\n",
	})

	code, stdout, _ := invoke(t, "fmt", "--json", "--list", root)
	if code != exitFindings {
		t.Fatalf("fmt --json --list exited %d: %s", code, stdout)
	}
	var report struct {
		Changed  []string `json:"changed"`
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v in %q", err, stdout)
	}
	if len(report.Changed) != 1 || report.Changed[0] != "people/rossi-giovanni/README.md" {
		t.Errorf("changed = %v", report.Changed)
	}
	if len(report.Problems) != 0 {
		t.Errorf("problems = %v", report.Problems)
	}
}

func invokeWith(t *testing.T, stdin string, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errs bytes.Buffer
	code = run(args, strings.NewReader(stdin), &out, &errs)
	return code, out.String(), errs.String()
}

// A small export carrying the cases that cost something to get wrong: a place hierarchy
// restated at two events, two people who want the same slug, a source cited by page, a
// single parent, and a date GEDCOM can only express as a phrase.
const export = `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Giovanni Battista /Rossi/
1 SEX M
1 BIRT
2 DATE 12 MAR 1834
2 PLAC Borgo San Lorenzo, Firenze, Toscana
2 SOUR @S1@
3 PAGE 1834-042
3 DATA
4 TEXT Giovanni Battista, son of Antonio Rossi, baptised 14 March 1834.
1 FAMS @F1@
0 @I2@ INDI
1 NAME Maria /Bianchi/
1 SEX F
1 FAMS @F1@
0 @I3@ INDI
1 NAME Giovanni Battista /Rossi/
1 SEX M
1 BIRT
2 DATE 1889
2 PLAC Firenze, Toscana
1 FAMC @F1@
0 @I4@ INDI
1 NAME Anna /Rossi/
1 SEX F
1 BIRT
2 DATE (3 MAR)
1 FAMC @F1@
0 @F1@ FAM
1 HUSB @I1@
1 WIFE @I2@
1 CHIL @I3@
1 CHIL @I4@
1 MARR
2 DATE ABT 1858
2 PLAC Firenze, Toscana
0 @S1@ SOUR
1 TITL Parish of San Lorenzo, Baptismal Register
0 TRLR
`

// An export that disagrees with itself, which is what most of them do somewhere.
const muddled = `0 HEAD
1 CHAR UTF-8
0 @I1@ INDI
1 NAME Pietro /Rossi/
1 BIRT
2 DATE 1900
1 DEAT
2 DATE 1850
0 TRLR
`

func exportFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rossi.ged")
	if err := os.WriteFile(path, []byte(export), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// freshCorpus is where an import will write: a directory that does not exist yet, in no
// repository.
func freshCorpus(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "rossi")
}

// A tree that arrives from somewhere else has to leave a corpus origins can go on
// working with. An import that produced a corpus check rejected would be an import
// whose output nobody could use.
func TestImportProducesACorpusCheckAccepts(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	if code, _, errs := invoke(t, "import", "gedcom", "--into", root, exportFile(t)); code != 0 {
		t.Fatalf("import exited %d: %s", code, errs)
	}

	code, stdout, stderr := invoke(t, "check", root)
	if code != 0 {
		t.Fatalf("check exited %d:\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "0 error(s)") {
		t.Errorf("check said:\n%s", stdout)
	}
}

// Origins is the sole writer of the format wherever it writes, so there is no path by
// which bytes from another tool reach a corpus unread — an import is parsed onto the
// model and emitted, exactly like every other entry point.
func TestAnImportedCorpusIsAlreadyCanonical(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	invoke(t, "import", "gedcom", "--into", root, exportFile(t))

	code, stdout, stderr := invoke(t, "fmt", "--list", root)
	if code != 0 || stdout != "" {
		t.Errorf("fmt --list exited %d and wanted to rewrite:\n%s%s", code, stdout, stderr)
	}
}

func TestImportMapsTheAwkwardCases(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	if code, _, errs := invoke(t, "import", "gedcom", "--into", root, exportFile(t)); code != 0 {
		t.Fatalf("import exited %d: %s", code, errs)
	}
	read := func(path string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}

	// The place is stated at three events and becomes three entities linked by within.
	if got := read("places/borgo-san-lorenzo/README.md"); !strings.Contains(got, "within: firenze") {
		t.Errorf("borgo-san-lorenzo:\n%s", got)
	}
	// A yearless date recovered out of a phrase, which GEDCOM cannot state at all.
	if got := read("people/rossi-anna/README.md"); !strings.Contains(got, "date: 3 MAR") {
		t.Errorf("rossi-anna:\n%s", got)
	}
	// The second Giovanni Battista takes a discriminator worth reading.
	if got := read("people/rossi-giovanni-battista-1889/README.md"); !strings.Contains(got, "date: 1889") {
		t.Errorf("the discriminated slug:\n%s", got)
	}
	// The transcription lives once, in the source, and the event points at it.
	source := read("sources/parish-of-san-lorenzo-baptismal-register/README.md")
	if !strings.Contains(source, "1834-042:") || !strings.Contains(source, "baptised 14 March 1834") {
		t.Errorf("source:\n%s", source)
	}
}

// A picture the export names comes in where the file is beside it, copied to the record
// that keeps it and recorded with everything else, and the commit counts the archives
// the sources are kept in.
func TestAnImportBringsInThePicturesBesideIt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "photos"), 0o755); err != nil {
		t.Fatal(err)
	}
	photo := filepath.Join(dir, "photos", "giovanni.png")
	file, err := os.Create(photo)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image.NewGray(image.Rect(0, 0, 40, 60))); err != nil {
		t.Fatal(err)
	}
	file.Close()
	ged := filepath.Join(dir, "rossi.ged")
	if err := os.WriteFile(ged, []byte(`0 HEAD
1 GEDC
2 VERS 7.0
0 @R1@ REPO
1 NAME Archivio Diocesano di Firenze
0 @S1@ SOUR
1 TITL Register
1 REPO @R1@
0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 OBJE
2 FILE photos/giovanni.png
2 TITL Giovanni in 1890
0 TRLR
`), 0o644); err != nil {
		t.Fatal(err)
	}

	root := freshCorpus(t)
	code, stdout, stderr := invoke(t, "import", "gedcom", "--into", root, ged)
	if code != 0 {
		t.Fatalf("import exited %d: %s", code, stderr)
	}
	copied, err := os.ReadFile(filepath.Join(root, "people", "rossi-giovanni", "photos", "giovanni.png"))
	original, _ := os.ReadFile(photo)
	if err != nil || string(copied) != string(original) {
		t.Errorf("the picture was not copied beside the record: %v", err)
	}
	if !strings.Contains(stdout, "1 repository") ||
		!strings.Contains(git(t, root, "log", "--format=%s"), "1 person, 0 families, 0 places, 1 source, 1 repository") {
		t.Errorf("the repository was not counted:\n%s", stdout)
	}
	if got := git(t, root, "status", "--short"); got != "" {
		t.Errorf("the import left work unrecorded:\n%s", got)
	}
	if code, out, _ := invoke(t, "check", root); code != 0 || !strings.Contains(out, "0 error(s)") {
		t.Errorf("check exited %d:\n%s", code, out)
	}
}

func TestImportJSON(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	code, stdout, stderr := invoke(t, "import", "gedcom", "--json", "--into", root, exportFile(t))
	if code != 0 {
		t.Fatalf("exited %d: %s", code, stderr)
	}
	var report api.Imported
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v in:\n%s", err, stdout)
	}
	if report.Counts.People != 4 || report.Counts.Families != 1 || report.Counts.Places != 3 {
		t.Errorf("counts = %+v", report.Counts)
	}
	if report.Corpus != root {
		t.Errorf("corpus = %q", report.Corpus)
	}
	if len(report.Notes) == 0 {
		t.Error("an import with unsourced events reported nothing")
	}
	if report.Commit != git(t, root, "rev-parse", "HEAD") {
		t.Errorf("commit = %q", report.Commit)
	}
	if !report.Started || report.History == "" {
		t.Errorf("the repository origins made is reported as %+v", report)
	}
}

// Collisions are where a human who knows the family beats any rule, so --interactive
// puts the proposals up and takes an answer. An empty answer takes the first, which is
// what makes holding down return equivalent to not asking at all.
func TestImportAsksWhenItIsToldTo(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	code, _, stderr := invokeWith(t, "rossi-giovanni-battista-il-giovane\n",
		"import", "gedcom", "--interactive", "--into", root, exportFile(t))
	if code != 0 {
		t.Fatalf("exited %d: %s", code, stderr)
	}
	if !strings.Contains(stderr, "Giovanni Battista Rossi already has that name") {
		t.Errorf("nothing was asked:\n%s", stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-giovanni-battista-il-giovane", "README.md")); err != nil {
		t.Errorf("the answer was not used: %v", err)
	}
}

// Bulk work is one commit and not thousands, and a tree arriving from elsewhere is the
// bulkiest work origins does. The commit has to say origins made it, or the very next
// command would stop on a history it could not account for — a corpus that refused to
// be used the moment it was imported.
func TestAnImportIsOneCommitOriginsAnswersFor(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	if code, out, errs := invoke(t, "import", "gedcom", "--into", root, exportFile(t)); code != 0 {
		t.Fatalf("import exited %d:\n%s%s", code, out, errs)
	}

	if got := git(t, root, "log", "--format=%s"); strings.Count(got, "\n") != 0 {
		t.Errorf("the import is more than one commit:\n%s", got)
	}
	if got := trailer(t, root, "Origins-Op"); got != "gedcom.import" {
		t.Errorf("Origins-Op = %q", got)
	}
	if got := trailer(t, root, "Origins-Tool"); got != "origins "+api.Version {
		t.Errorf("Origins-Tool = %q", got)
	}
	if got := git(t, root, "status", "--short"); got != "" {
		t.Errorf("the import left work unrecorded:\n%s", got)
	}
	if got := git(t, root, "for-each-ref", "--format=%(objectname)", "refs/origins/head"); got == "" {
		t.Error("the commit was not marked as origins' own")
	}

	// the proof that it is a history origins answers for: a command that reads one
	// stops dead on a history it did not write, and this one does not stop
	if code, _, errs := invoke(t, "log", "-C", root); code != exitOK {
		t.Errorf("log exited %d on a freshly imported corpus: %s", code, errs)
	}
}

// The import names no subjects, and what that would have bought is bought anyway.
// Everywhere else the trailer says which few entities a change was about; here the
// answer is all of them, which is the corpus written out one line at a time — and the
// question it would answer is one blame answers from the file it is asked about.
func TestAnImportNamesNoSubjectsAndBlameStillReachesIt(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	if code, out, errs := invoke(t, "import", "gedcom", "--into", root, exportFile(t)); code != 0 {
		t.Fatalf("import exited %d:\n%s%s", code, out, errs)
	}

	if got := trailer(t, root, "Origins-Subject"); got != "" {
		t.Errorf("the import named the corpus one entity at a time: %q", got)
	}
	code, stdout, stderr := invoke(t, "blame", "-C", root, "rossi-anna")
	if code != exitOK {
		t.Fatalf("blame exited %d: %s", code, stderr)
	}
	if sha := git(t, root, "rev-parse", "HEAD"); !strings.Contains(stdout, short(sha)) {
		t.Errorf("blame does not reach the import at %s:\n%s", short(sha), stdout)
	}
}

// Every command that opens a repository stops on a history origins did not write, and
// import is where that rule shows what it is about. A commit made before the corpus
// existed is not a commit about the corpus, so there is nothing for the test of
// ownership to be about — and no way through it either, since recovering a corpus that
// is not there yet is not something anybody could be asked to do.
func TestAnImportIsNotStoppedByAHistoryOriginsDidNotWrite(t *testing.T) {
	t.Parallel()
	outer := byHand(t, map[string]string{"README.md": "the notes I keep\n"})
	root := filepath.Join(outer, "trees", "nonna")

	if code, out, errs := invoke(t, "import", "gedcom", "--into", root, exportFile(t)); code != exitOK {
		t.Fatalf("import exited %d beneath a history origins did not write:\n%s%s", code, out, errs)
	}
	// and what comes out is usable straight away: the anchor is the import, and the
	// commits below it are about something else entirely
	if code, _, errs := invoke(t, "log", "-C", root); code != exitOK {
		t.Errorf("log exited %d on the imported corpus: %s", code, errs)
	}
}

// A tree written inside a repository somebody already keeps is recorded in that one,
// and takes nothing else in the repository with it. A commit names its paths, so the
// draft sitting unfinished beside it is still sitting there afterwards.
func TestAnImportIntoARepositoryIsRecordedThereAndTakesNothingElseWithIt(t *testing.T) {
	t.Parallel()
	outer := byHand(t, map[string]string{"README.md": "the notes I keep\n"})
	if err := os.WriteFile(filepath.Join(outer, "draft.md"), []byte("half written\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(outer, "trees", "nonna")

	if code, out, errs := invoke(t, "import", "gedcom", "--into", root, exportFile(t)); code != 0 {
		t.Fatalf("import exited %d:\n%s%s", code, out, errs)
	}

	if _, err := os.Stat(filepath.Join(root, ".git")); err == nil {
		t.Error("a repository was made inside the one already there")
	}
	if got := git(t, outer, "log", "--format=%s"); strings.Count(got, "\n") != 1 {
		t.Errorf("the history of the repository reads:\n%s", got)
	}
	if got := git(t, outer, "show", "--name-only", "--format=", "HEAD"); slices.Contains(strings.Split(got, "\n"), "README.md") {
		t.Errorf("the import commit reached outside the corpus:\n%s", got)
	}
	if got := git(t, outer, "status", "--short"); got != "?? draft.md" {
		t.Errorf("what was beside the corpus is now %q, want it left alone", got)
	}
}

// An import is the one write path with no earlier state to be judged against, and it
// judges nothing. What a real export contradicts is the export's, stated by whoever
// kept the tree; a tool that refused to carry a file in until the file agreed with
// itself would refuse every file anybody actually has. check is what says so, after.
func TestAnImportCarriesInWhateverTheExportContradicts(t *testing.T) {
	t.Parallel()
	root := freshCorpus(t)
	file := filepath.Join(t.TempDir(), "muddled.ged")
	if err := os.WriteFile(file, []byte(muddled), 0o644); err != nil {
		t.Fatal(err)
	}

	if code, out, errs := invoke(t, "import", "gedcom", "--into", root, file); code != exitOK {
		t.Fatalf("import exited %d on an export that contradicts itself:\n%s%s", code, out, errs)
	}
	code, stdout, _ := invoke(t, "check", root)
	if code != exitFindings || !strings.Contains(stdout, "death-before-birth") {
		t.Errorf("check exited %d and said:\n%s", code, stdout)
	}
}

// The corpus is the part that took the work, and git refusing a commit is the part
// that takes a minute to put right. The import says both things happened and leaves
// the tree where it is.
func TestAnImportThatCannotBeRecordedKeepsTheCorpusAndSaysSo(t *testing.T) {
	root := freshCorpus(t)
	t.Setenv("GIT_COMMITTER_DATE", "not a date at all")

	code, stdout, stderr := invoke(t, "import", "gedcom", "--into", root, exportFile(t))
	if code != exitMisuse {
		t.Errorf("import exited %d, want %d", code, exitMisuse)
	}
	if !strings.Contains(stderr, "could not record it") || !strings.Contains(stderr, "origins adopt") {
		t.Errorf("stderr = %q", stderr)
	}
	if !strings.Contains(stdout, "4 people") {
		t.Errorf("the import did not report what it read:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-anna", "README.md")); err != nil {
		t.Errorf("the corpus was taken back off the disk: %v", err)
	}
	if got := git(t, root, "status", "--short", "--untracked-files=no"); got != "" {
		t.Errorf("the refused commit left work staged:\n%s", got)
	}
}

// A directory already holding something is refused, since writing into it would mean
// merging two trees. Where origins chose the directory the reader may not know it did,
// so the refusal says where the name came from, and either way it says how to choose
// another — nothing having been read from the file first.
func TestAnImportRefusedADirectorySaysHowToChooseAnother(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	export := filepath.Join(dir, "tiny.ged")
	if err := os.WriteFile(export, []byte("not a GEDCOM file at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("tiny", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join("tiny", "notes.txt"), []byte("mine\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		args []string
		says []string
	}{
		{[]string{"import", "gedcom", "tiny.ged"}, []string{"tiny is not empty", "tiny.ged", "--into"}},
		{[]string{"import", "gedcom", "--into", "tiny", "tiny.ged"}, []string{"tiny is not empty", "--into"}},
	} {
		code, _, stderr := invoke(t, c.args...)
		if code != exitMisuse {
			t.Errorf("%v exited %d, want %d", c.args, code, exitMisuse)
		}
		for _, said := range c.says {
			if !strings.Contains(stderr, said) {
				t.Errorf("%v does not say %q:\n%s", c.args, said, stderr)
			}
		}
	}
	if got := read(t, dir, "tiny/notes.txt"); got != "mine\n" {
		t.Errorf("the directory was disturbed: %q", got)
	}
}

func TestImportRefusesWhatItCannotDo(t *testing.T) {
	t.Parallel()
	existing := corpusDir(t, map[string]string{"origins.yaml": sound})
	for _, c := range []struct {
		what string
		args []string
	}{
		{what: "a format it does not read", args: []string{"import", "csv", exportFile(t)}},
		{what: "no format at all", args: []string{"import"}},
		{what: "two files at once", args: []string{"import", "gedcom", "a.ged", "b.ged"}},
		{what: "a corpus that already exists",
			args: []string{"import", "gedcom", "--into", existing, exportFile(t)}},
	} {
		if code, _, _ := invoke(t, c.args...); code != exitMisuse {
			t.Errorf("%s exited %d, want %d", c.what, code, exitMisuse)
		}
	}
}

// A small Gramps tree: a couple and their son, the father's baptism cited to a register
// page, and a photograph of the son the file names relative to where it was written.
const grampsTree = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE database PUBLIC "-//Gramps//DTD Gramps XML 1.7.2//EN"
"http://gramps-project.org/xml/1.7.2/grampsxml.dtd">
<database xmlns="http://gramps-project.org/xml/1.7.2/">
  <header>
    <created date="2026-04-28" version="6.0.1"/>
  </header>
  <events>
    <event handle="_e1" change="1" id="E0001">
      <type>Baptism</type>
      <dateval val="1834-03-14"/>
      <citationref hlink="_c1"/>
    </event>
    <event handle="_e2" change="1" id="E0002">
      <type>Marriage</type>
      <dateval val="1858" type="about"/>
    </event>
  </events>
  <people>
    <person handle="_p1" change="1" id="I0001">
      <gender>M</gender>
      <name type="Birth Name">
        <first>Giovanni Battista</first>
        <surname>Rossi</surname>
      </name>
      <eventref hlink="_e1" role="Primary"/>
      <parentin hlink="_f1"/>
    </person>
    <person handle="_p2" change="1" id="I0002">
      <gender>F</gender>
      <name type="Birth Name">
        <first>Maria</first>
        <surname>Bianchi</surname>
      </name>
      <parentin hlink="_f1"/>
    </person>
    <person handle="_p3" change="1" id="I0003" >
      <gender>M</gender>
      <name type="Birth Name">
        <first>Giovanni Battista</first>
        <surname>Rossi</surname>
      </name>
      <objref hlink="_o1"/>
      <childof hlink="_f1"/>
    </person>
  </people>
  <families>
    <family handle="_f1" change="1" id="F0001">
      <rel type="Married"/>
      <father hlink="_p1"/>
      <mother hlink="_p2"/>
      <eventref hlink="_e2" role="Family"/>
      <childref hlink="_p3"/>
    </family>
  </families>
  <citations>
    <citation handle="_c1" change="1" id="C0001">
      <page>n. 42</page>
      <confidence>2</confidence>
      <sourceref hlink="_s1"/>
    </citation>
  </citations>
  <sources>
    <source handle="_s1" change="1" id="S0001">
      <stitle>Battesimi di San Lorenzo</stitle>
    </source>
  </sources>
  <objects>
    <object handle="_o1" change="1" id="O0001">
      <file src="photos/giovanni.png" mime="image/png" description="Giovanni in 1890"/>
    </object>
  </objects>
</database>
`

// grampsFile writes content into a directory of its own under name, with a photograph
// at photos/giovanni.png beside it where pictured is true, and hands back its path.
func grampsFile(t *testing.T, name string, content []byte, pictured bool) string {
	t.Helper()
	dir := t.TempDir()
	if pictured {
		writePNG(t, filepath.Join(dir, "photos", "giovanni.png"))
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, image.NewGray(image.Rect(0, 0, 40, 60))); err != nil {
		t.Fatal(err)
	}
}

// gzipTar is a Gramps package: the XML gzipped as data.gramps, beside the files given,
// the whole a gzipped tar.
func gzipTar(t *testing.T, xml string, files map[string][]byte) []byte {
	t.Helper()
	var data bytes.Buffer
	zipped := gzip.NewWriter(&data)
	zipped.Write([]byte(xml))
	zipped.Close()
	var out bytes.Buffer
	outer := gzip.NewWriter(&out)
	archive := tar.NewWriter(outer)
	add := func(name string, content []byte) {
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		archive.Write(content)
	}
	// Gramps writes the media first and the XML last
	for _, name := range slices.Sorted(maps.Keys(files)) {
		add(name, files[name])
	}
	add("data.gramps", data.Bytes())
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	outer.Close()
	return out.Bytes()
}

// A tree kept in Gramps comes in as a GEDCOM export does: one commit origins answers
// for, saying which Gramps wrote the file, a corpus check accepts and fmt leaves alone,
// and the photograph beside the file copied in beside the record keeping it.
func TestAGrampsImportIsOneCommitOriginsAnswersFor(t *testing.T) {
	t.Parallel()
	file := grampsFile(t, "rossi.gramps", []byte(grampsTree), true)
	root := freshCorpus(t)
	code, stdout, stderr := invoke(t, "import", "gramps", "--into", root, file)
	if code != exitOK {
		t.Fatalf("import exited %d:\n%s%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "3 people, 1 family, 0 places, 1 source, 0 repositories") {
		t.Errorf("the report reads:\n%s", stdout)
	}

	if got := git(t, root, "log", "--format=%s"); strings.Count(got, "\n") != 0 ||
		got != "Import rossi.gramps: 3 people, 1 family, 0 places, 1 source, 0 repositories" {
		t.Errorf("the history reads:\n%s", got)
	}
	if got := trailer(t, root, "Origins-Op"); got != "gramps.import" {
		t.Errorf("Origins-Op = %q", got)
	}
	if body := git(t, root, "log", "-1", "--format=%b"); !strings.Contains(body,
		"Read from rossi.gramps, which Gramps 6.0.1 exported on 2026-04-28 as Gramps XML 1.7.2.") {
		t.Errorf("the commit does not say what it read:\n%s", body)
	}
	if got := git(t, root, "status", "--short"); got != "" {
		t.Errorf("the import left work unrecorded:\n%s", got)
	}

	copied, err := os.ReadFile(filepath.Join(root, "people", "rossi-giovanni-battista-2", "photos", "giovanni.png"))
	original, _ := os.ReadFile(filepath.Join(filepath.Dir(file), "photos", "giovanni.png"))
	if err != nil || !bytes.Equal(copied, original) {
		t.Errorf("the photograph was not copied beside the record: %v", err)
	}
	if got := read(t, root, "sources/battesimi-di-san-lorenzo/README.md"); !strings.Contains(got, "n-42") {
		t.Errorf("the citation is not an anchor of its source:\n%s", got)
	}
	if got := read(t, root, "people/rossi-giovanni-battista/README.md"); !strings.Contains(got, "gramps: I0001") {
		t.Errorf("the Gramps ID was not kept:\n%s", got)
	}

	if code, out, _ := invoke(t, "check", root); code != exitOK || !strings.Contains(out, "0 error(s)") {
		t.Errorf("check exited %d:\n%s", code, out)
	}
	if code, out, errs := invoke(t, "fmt", "--list", root); code != exitOK || out != "" {
		t.Errorf("fmt --list exited %d and wanted to rewrite:\n%s%s", code, out, errs)
	}
	if code, _, errs := invoke(t, "log", "-C", root); code != exitOK {
		t.Errorf("log exited %d on a freshly imported corpus: %s", code, errs)
	}
}

// What is in a file says what it is, and its name does not: a package passed around
// under another name is still a package, its pictures copied out of it before the
// directory it was opened into is removed, and a gzipped file reads as a plain one.
func TestAGrampsImportReadsAFileByWhatIsInIt(t *testing.T) {
	t.Parallel()
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewGray(image.Rect(0, 0, 40, 60))); err != nil {
		t.Fatal(err)
	}
	var zipped bytes.Buffer
	writer := gzip.NewWriter(&zipped)
	writer.Write([]byte(grampsTree))
	writer.Close()

	for _, c := range []struct {
		name     string
		content  []byte
		pictured bool
	}{
		{"rossi.zip", gzipTar(t, grampsTree, map[string][]byte{"photos/giovanni.png": photo.Bytes()}), false},
		{"rossi.gramps", zipped.Bytes(), true},
	} {
		root := freshCorpus(t)
		file := grampsFile(t, c.name, c.content, c.pictured)
		if code, out, errs := invoke(t, "import", "gramps", "--into", root, file); code != exitOK {
			t.Fatalf("%s: import exited %d:\n%s%s", c.name, code, out, errs)
		}
		got, err := os.ReadFile(filepath.Join(root, "people", "rossi-giovanni-battista-2", "photos", "giovanni.png"))
		if err != nil || !bytes.Equal(got, photo.Bytes()) {
			t.Errorf("%s: the photograph was not copied in: %v", c.name, err)
		}
		if got := git(t, root, "status", "--short"); got != "" {
			t.Errorf("%s: the import left work unrecorded:\n%s", c.name, got)
		}
	}
}

// A header names a directory on the machine that wrote it, so --media names one more,
// looked in once the file's own have not had the picture; a picture found nowhere is
// an entry all the same, and reported.
func TestAGrampsImportLooksForPicturesWhereItIsTold(t *testing.T) {
	t.Parallel()
	file := grampsFile(t, "rossi.gramps", []byte(grampsTree), false)

	root := freshCorpus(t)
	code, stdout, stderr := invoke(t, "import", "gramps", "--into", root, file)
	if code != exitOK {
		t.Fatalf("import exited %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "was not found where Gramps would look for it") {
		t.Errorf("the picture not found is not reported:\n%s", stdout)
	}
	if got := read(t, root, "people/rossi-giovanni-battista-2/README.md"); !strings.Contains(got, "photos/giovanni.png") {
		t.Errorf("the picture not found is not an entry:\n%s", got)
	}

	media := t.TempDir()
	writePNG(t, filepath.Join(media, "photos", "giovanni.png"))
	root = freshCorpus(t)
	if code, _, stderr := invoke(t, "import", "gramps", "--media", media, "--into", root, file); code != exitOK {
		t.Fatalf("import --media exited %d: %s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-giovanni-battista-2", "photos", "giovanni.png")); err != nil {
		t.Errorf("the picture in --media was not copied in: %v", err)
	}
}

func TestAGrampsImportReportsAsJSONAndAsksWhenToldTo(t *testing.T) {
	t.Parallel()
	file := grampsFile(t, "rossi.gramps", []byte(grampsTree), true)
	root := freshCorpus(t)
	code, stdout, stderr := invokeWith(t, "rossi-giovanni-battista-il-giovane\n",
		"import", "gramps", "--json", "--interactive", "--name", "Casa Rossi", "--into", root, file)
	if code != exitOK {
		t.Fatalf("exited %d: %s", code, stderr)
	}
	var report api.Imported
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v in:\n%s", err, stdout)
	}
	if report.Counts.People != 3 || report.Counts.Families != 1 || report.Counts.Sources != 1 {
		t.Errorf("counts = %+v", report.Counts)
	}
	if report.Commit != git(t, root, "rev-parse", "HEAD") || !report.Started {
		t.Errorf("the report = %+v", report)
	}
	if !strings.Contains(stderr, "already has that name") {
		t.Errorf("nothing was asked:\n%s", stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-giovanni-battista-il-giovane", "README.md")); err != nil {
		t.Errorf("the answer was not used: %v", err)
	}
	if got := read(t, root, "origins.yaml"); !strings.Contains(got, "name: Casa Rossi") {
		t.Errorf("the manifest:\n%s", got)
	}
}

// What cannot be read is refused before anything is written: a file Gramps did not
// write, one in a version older than origins reads, and a package with an entry that
// would land outside the directory it is opened into.
func TestAGrampsImportRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()
	older := strings.ReplaceAll(grampsTree, "1.7.2", "1.5.0")
	escaping := gzipTar(t, grampsTree, map[string][]byte{"../outside.png": []byte("x")})
	for _, c := range []struct {
		what    string
		args    func(root string) []string
		message string
	}{
		{"a file Gramps did not write", func(root string) []string {
			return []string{"import", "gramps", "--into", root, grampsFile(t, "x.gramps", []byte(export), false)}
		}, "not a Gramps export"},
		{"an older version", func(root string) []string {
			return []string{"import", "gramps", "--into", root, grampsFile(t, "old.gramps", []byte(older), false)}
		}, "Gramps XML 1.5.0"},
		{"an entry outside the package", func(root string) []string {
			return []string{"import", "gramps", "--into", root, grampsFile(t, "bad.gpkg", escaping, false)}
		}, "outside"},
		{"a file that is not there", func(root string) []string {
			return []string{"import", "gramps", "--into", root, filepath.Join(t.TempDir(), "none.gramps")}
		}, "none.gramps"},
		{"--media that is no directory", func(root string) []string {
			return []string{"import", "gramps", "--media", filepath.Join(t.TempDir(), "none"), "--into", root,
				grampsFile(t, "rossi.gramps", []byte(grampsTree), false)}
		}, "is not a directory"},
		{"--media on a GEDCOM", func(root string) []string {
			return []string{"import", "gedcom", "--media", t.TempDir(), "--into", root, exportFile(t)}
		}, "-media"},
		{"two files at once", func(root string) []string {
			return []string{"import", "gramps", "--into", root, "a.gramps", "b.gramps"}
		}, "one file at a time"},
	} {
		root := freshCorpus(t)
		code, _, stderr := invoke(t, c.args(root)...)
		if code != exitMisuse || !strings.Contains(stderr, c.message) {
			t.Errorf("%s: exited %d, saying %q", c.what, code, stderr)
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Errorf("%s: something was written: %v", c.what, err)
		}
	}
}
