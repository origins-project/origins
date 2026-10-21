package gramps

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// gramps is a small export in the given version, its header naming mediapath where that
// is not empty.
func gramps(version, mediapath string) string {
	header := ""
	if mediapath != "" {
		header = "\n    <mediapath>" + mediapath + "</mediapath>"
	}
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE database PUBLIC "-//Gramps//DTD Gramps XML ` + version + `//EN"
"http://gramps-project.org/xml/` + version + `/grampsxml.dtd">
<database xmlns="http://gramps-project.org/xml/` + version + `/">
  <header>
    <created date="2026-04-28" version="6.0.0"/>` + header + `
  </header>
  <people>
    <person handle="_a" id="I0001">
      <gender>F</gender>
      <name type="Birth Name">
        <first>Annunziata</first>
        <surname>Rossi</surname>
      </name>
    </person>
    <person handle="_b" id="I0002">
      <gender>M</gender>
    </person>
  </people>
  <notes>
    <note handle="_n" id="N0001" type="General">
      <text>  Two lines,
with &amp; kept as written.  </text>
    </note>
  </notes>
</database>
`
}

func gzipped(t *testing.T, content []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := gzip.NewWriter(&out)
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// entry is one member of a package: a file with content, or a header as given.
type entry struct {
	header  tar.Header
	content string
}

func file(name, content string) entry {
	return entry{tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))}, content}
}

// packaged is a gzipped tar of entries, as Gramps writes a package.
func packaged(t *testing.T, entries ...entry) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := tar.NewWriter(&archive)
	for _, e := range entries {
		header := e.header
		if err := writer.WriteHeader(&header); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return gzipped(t, archive.Bytes())
}

func written(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func opened(t *testing.T, path string) *Export {
	t.Helper()
	export, err := Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", filepath.Base(path), err)
	}
	t.Cleanup(func() { export.Close() })
	return export
}

// refused opens path expecting a refusal that says each of words.
func refused(t *testing.T, path string, words ...string) {
	t.Helper()
	export, err := Open(path)
	if err == nil {
		export.Close()
		t.Fatalf("%s was read, want it refused", filepath.Base(path))
	}
	for _, word := range words {
		if !strings.Contains(err.Error(), word) {
			t.Errorf("refusal %q does not say %q", err, word)
		}
	}
}

// readsTheTree checks that export holds the tree gramps writes.
func readsTheTree(t *testing.T, export *Export) {
	t.Helper()
	people := export.Root.First("people").All("person")
	if len(people) != 2 {
		t.Fatalf("got %d people, want 2", len(people))
	}
	if got := people[0].Attr("id"); got != "I0001" {
		t.Errorf("first person's id = %q", got)
	}
	if got := people[0].First("name").First("first").Text(); got != "Annunziata" {
		t.Errorf("first person's first name = %q", got)
	}
	if got := people[0].First("name").Attr("type"); got != "Birth Name" {
		t.Errorf("first person's name type = %q", got)
	}
}

func TestPlainXMLIsRead(t *testing.T) {
	dir, pictures := t.TempDir(), t.TempDir()
	export := opened(t, written(t, dir, "tree.gramps", []byte(gramps("1.7.2", pictures))))

	readsTheTree(t, export)
	if export.Version != "1.7.2" || export.Newer {
		t.Errorf("version = %q, newer = %v", export.Version, export.Newer)
	}
	if want := []string{pictures, dir}; !slices.Equal(export.Media, want) {
		t.Errorf("media = %q, want %q", export.Media, want)
	}
}

func TestGzippedXMLIsRead(t *testing.T) {
	dir := t.TempDir()
	export := opened(t, written(t, dir, "tree.gramps", gzipped(t, []byte(gramps("1.7.1", "pictures")))))

	readsTheTree(t, export)
	if export.Version != "1.7.1" {
		t.Errorf("version = %q", export.Version)
	}
	if want := []string{filepath.Join(dir, "pictures"), dir}; !slices.Equal(export.Media, want) {
		t.Errorf("media = %q, want %q", export.Media, want)
	}
}

// A header naming no media directory leaves the one the file is in.
func TestWithoutAMediapathTheMediaAreBesideTheFile(t *testing.T) {
	dir := t.TempDir()
	export := opened(t, written(t, dir, "tree.gramps", []byte(gramps("1.7.2", ""))))
	if want := []string{dir}; !slices.Equal(export.Media, want) {
		t.Errorf("media = %q, want %q", export.Media, want)
	}
}

// A mediapath's variables are filled in from the environment, and one naming a variable
// that is not set, as Gramps' own examples name its installation, names no directory.
func TestAMediapathsVariablesAreFilledIn(t *testing.T) {
	dir, resources := t.TempDir(), t.TempDir()
	t.Setenv("GRAMPS_RESOURCES", resources)
	export := opened(t, written(t, dir, "set.gramps", []byte(gramps("1.7.2", "{GRAMPS_RESOURCES}/doc/gramps/example/gramps"))))
	if want := []string{filepath.Join(resources, "doc", "gramps", "example", "gramps"), dir}; !slices.Equal(export.Media, want) {
		t.Errorf("media = %q, want %q", export.Media, want)
	}

	export = opened(t, written(t, dir, "braces.gramps", []byte(gramps("1.7.2", "{{pictures}}"))))
	if want := []string{filepath.Join(dir, "{pictures}"), dir}; !slices.Equal(export.Media, want) {
		t.Errorf("media = %q, want %q", export.Media, want)
	}

	os.Unsetenv("GRAMPS_RESOURCES")
	for _, mediapath := range []string{"{GRAMPS_RESOURCES}/doc/gramps/example/gramps", "{unclosed", "stray}"} {
		export := opened(t, written(t, dir, "unset.gramps", []byte(gramps("1.7.2", mediapath))))
		if want := []string{dir}; !slices.Equal(export.Media, want) {
			t.Errorf("%s: media = %q, want %q", mediapath, export.Media, want)
		}
	}
}

// A package is told by its content: this one is called .zip, as one of the published
// samples is, and its XML comes after the picture it names.
func TestAPackageIsExtractedAndRemovedAgain(t *testing.T) {
	path := written(t, t.TempDir(), "tree.zip", packaged(t,
		entry{tar.Header{Name: "GrampsMedia/", Typeflag: tar.TypeDir, Mode: 0o755}, ""},
		file("GrampsMedia/portrait.jpg", "a picture"),
		file("data.gramps", string(gzipped(t, []byte(gramps("1.7.2", "/somewhere/else"))))),
	))
	export, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	readsTheTree(t, export)
	if len(export.Media) != 1 {
		t.Fatalf("media = %q, want the directory the package was extracted into", export.Media)
	}
	extracted := export.Media[0]
	picture, err := os.ReadFile(filepath.Join(extracted, "GrampsMedia", "portrait.jpg"))
	if err != nil || string(picture) != "a picture" {
		t.Errorf("the picture reads %q, %v", picture, err)
	}

	if err := export.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(extracted); !os.IsNotExist(err) {
		t.Errorf("%s is still there after Close: %v", extracted, err)
	}
}

// A package whose XML is not itself gzipped is read as well.
func TestAPackageMayHoldPlainXML(t *testing.T) {
	path := written(t, t.TempDir(), "tree.gpkg", packaged(t, file("data.gramps", gramps("1.7.1", ""))))
	readsTheTree(t, opened(t, path))
}

// Every refusal of a package leaves nothing extracted behind, and nothing written where
// an entry's name would have put it.
func TestAnUnsafePackageIsRefused(t *testing.T) {
	temporary := t.TempDir()
	t.Setenv("TMPDIR", temporary)
	t.Setenv("TMP", temporary)
	t.Setenv("TEMP", temporary)
	data := file("data.gramps", gramps("1.7.2", ""))

	for _, test := range []struct {
		name  string
		entry entry
		says  string
	}{
		{"climbing out", file("../escaped.jpg", "x"), "outside"},
		{"climbing out from inside", file("GrampsMedia/../../escaped.jpg", "x"), "outside"},
		{"absolute", file("/escaped.jpg", "x"), "outside"},
		{"a symbolic link", entry{tar.Header{Name: "link.jpg", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o777}, ""}, "neither a file nor a directory"},
		{"a hard link", entry{tar.Header{Name: "link.jpg", Typeflag: tar.TypeLink, Linkname: "data.gramps", Mode: 0o644}, ""}, "neither a file nor a directory"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			refused(t, written(t, dir, "tree.gpkg", packaged(t, data, test.entry)), test.entry.header.Name, test.says)

			if _, err := os.Stat(filepath.Join(dir, "escaped.jpg")); !os.IsNotExist(err) {
				t.Errorf("an entry was written outside the package: %v", err)
			}
			if left, _ := filepath.Glob(filepath.Join(temporary, "origins-gramps-*")); len(left) > 0 {
				t.Errorf("left behind in the temporary directory: %q", left)
			}
		})
	}
}

func TestAnOlderVersionIsRefusedNamingIt(t *testing.T) {
	dir := t.TempDir()
	refused(t, written(t, dir, "old.gramps", []byte(gramps("1.6.0", ""))), "1.6.0", "1.7.1", "export it again")

	unversioned := strings.Replace(gramps("1.7.2", ""), ` xmlns="http://gramps-project.org/xml/1.7.2/"`, "", 1)
	refused(t, written(t, dir, "older.gramps", []byte(unversioned)), "no version", "export it again")

	garbled := strings.Replace(gramps("1.7.2", ""), `xmlns="http://gramps-project.org/xml/1.7.2/"`,
		`xmlns="http://gramps-project.org/xml/one.seven/"`, 1)
	refused(t, written(t, dir, "garbled.gramps", []byte(garbled)), "one.seven")
}

func TestANewerVersionIsReadAndNamed(t *testing.T) {
	export := opened(t, written(t, t.TempDir(), "new.gramps", []byte(gramps("1.8.0", ""))))
	readsTheTree(t, export)
	if export.Version != "1.8.0" || !export.Newer {
		t.Errorf("version = %q, newer = %v; want 1.8.0, newer", export.Version, export.Newer)
	}
}

func TestWhatIsNoneOfTheThreeIsRefused(t *testing.T) {
	dir := t.TempDir()
	ged := []byte("0 HEAD\n1 GEDC\n2 VERS 5.5.1\n0 TRLR\n")

	refused(t, written(t, dir, "tree.ged", ged), "not a Gramps export")
	refused(t, written(t, dir, "tree.ged.gz", gzipped(t, ged)), "not a Gramps export")
	refused(t, written(t, dir, "broken.gramps", []byte{0x1f, 0x8b, 0, 0}), "not a Gramps export")
	refused(t, written(t, dir, "page.gramps", []byte("<html><body/></html>")), "<html>")
	refused(t, written(t, dir, "cut.gramps", []byte(gramps("1.7.2", "")[:300])), "not Gramps XML")
	refused(t, written(t, dir, "empty.gpkg", packaged(t, file("portrait.jpg", "a picture"))), "no data.gramps")
	refused(t, written(t, dir, "wrapped.gpkg", packaged(t,
		file("data.gramps", string(packaged(t, file("data.gramps", gramps("1.7.2", "")))))),
	), "not Gramps XML")
}

// A note's text is kept exactly as written, and an element with children keeps none of
// the whitespace laid out between them.
func TestTextIsKeptAsWritten(t *testing.T) {
	export := opened(t, written(t, t.TempDir(), "tree.gramps", []byte(gramps("1.7.2", ""))))
	note := export.Root.First("notes").First("note")
	if got, want := note.First("text").Text(), "  Two lines,\nwith & kept as written.  "; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if note.Text() != "" || export.Root.Text() != "" {
		t.Errorf("an element with children kept text: %q, %q", note.Text(), export.Root.Text())
	}
}

func TestAnAbsentElementAnswersForItself(t *testing.T) {
	var absent *Element
	if absent.First("x").First("y").Text() != "" || absent.Attr("id") != "" || absent.All("z") != nil {
		t.Error("an absent element answered something")
	}
}

// An import is recorded as one commit, which says what Gramps wrote the export it read.
func TestTheHeaderSaysWhatExportWasRead(t *testing.T) {
	export := opened(t, written(t, t.TempDir(), "tree.gramps", []byte(gramps("1.7.2", ""))))
	want := "Read from tree.gramps, which Gramps 6.0.0 exported on 2026-04-28 as Gramps XML 1.7.2."
	if got := Describe(export, "tree.gramps"); got != want {
		t.Errorf("described as %q, want %q", got, want)
	}
	if got := Describe(&Export{Root: element(t, "<database/>")}, ""); got != "Read from an export, which Gramps exported." {
		t.Errorf("an export with no header was described as %q", got)
	}
}
