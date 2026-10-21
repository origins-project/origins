package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/roundtrip"
	"github.com/origins-project/origins/internal/store"
)

func TestExportChartDrawsToTheFileNamed(t *testing.T) {
	t.Parallel()
	root, out := theFixture(t), filepath.Join(t.TempDir(), "pauline.pdf")
	code, stdout, stderr := invoke(t, "export", "chart", "-C", root, "--shape", "path", "--to", "ershov-lawrence",
		"--out", out, "verdier-pauline")
	if code != exitOK {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "drew the path chart of Pauline Verdier to "+out) ||
		!strings.Contains(stdout, " is the ") {
		t.Errorf("said %q", stdout)
	}
	if data, err := os.ReadFile(out); err != nil || !strings.HasPrefix(string(data), "%PDF-") {
		t.Errorf("wrote %.20q: %v", data, err)
	}
}

// A count of one record is of one person, family or note, however the count is kept.
func TestACountOfOneIsOfOneRecord(t *testing.T) {
	t.Parallel()
	if got, want := counts(map[string]int{"people": 1, "families": 2, "notes": 1, "INDI": 1}),
		"1 INDI, 2 families, 1 note, 1 person"; got != want {
		t.Errorf("counts said %q, want %q", got, want)
	}
	if got, want := countedAll(api.Counts{People: 1, Notes: 1, Hypotheses: 1}),
		"1 person, 0 families, 0 places, 0 sources, 0 repositories, 1 note, 1 hypothesis"; got != want {
		t.Errorf("countedAll said %q, want %q", got, want)
	}
}

func TestExportChartRefusesWhatItCannotDraw(t *testing.T) {
	t.Parallel()
	root, dir := theFixture(t), t.TempDir()
	out := filepath.Join(dir, "chart.svg")
	for _, args := range [][]string{
		{"export"},
		{"export", "gedcom"},
		{"export", "chart", "-C", root, "verdier-pauline"},
		{"export", "chart", "-C", root, "--out", out},
		{"export", "chart", "-C", root, "--shape", "pedigree", "--out", out, "verdier-pauline"},
		{"export", "chart", "-C", root, "--distance", "13", "--out", out, "verdier-pauline"},
		{"export", "chart", "-C", root, "--out", filepath.Join(dir, "chart.gif"), "verdier-pauline"},
		{"export", "chart", "-C", root, "--shape", "path", "--out", out, "verdier-pauline"},
		{"export", "chart", "-C", root, "--public", "--out", out, "marchena-solveigarson-sasha"},
	} {
		if code, _, stderr := invoke(t, args...); code == exitOK || stderr == "" {
			t.Errorf("%q: exit %d, stderr %q", args, code, stderr)
		}
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a refusal wrote the file: %v", err)
	}
}

// The git fixture written as a GEDZIP and imported into an empty directory is the
// fixture again, record by record and matched by what each says, in everything but what
// the export's report counts as left out; and written as a public view, the zip holds
// that view and nothing it hides.
func TestTheGitFixtureComesBackFromItsGEDZIP(t *testing.T) {
	t.Parallel()
	root, dir := gitFixture(t), t.TempDir()
	corpus, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, public := range []bool{false, true} {
		written := corpus.Corpus
		args := []string{"export", "gedcom", "--json", "-C", root}
		name := "whole"
		if public {
			written = model.Hide(corpus.Corpus, model.Hiding{Private: true, Living: true}, model.DateOf(time.Now()))
			args, name = append(args, "--public"), "public"
		}
		out := filepath.Join(dir, name+".gdz")
		code, stdout, stderr := invoke(t, append(args, "--out", out)...)
		if code != exitOK {
			t.Fatalf("%s: exit %d\n%s", name, code, stderr)
		}
		var report api.GEDCOMWritten
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatalf("%s: %v\n%s", name, err, stdout)
		}

		back := filepath.Join(dir, name)
		if code, _, stderr := invoke(t, "import", "gedcom", "--into", back, out); code != exitOK {
			t.Fatalf("%s: import: exit %d\n%s", name, code, stderr)
		}
		read, err := store.Load(back)
		if err != nil {
			t.Fatal(err)
		}
		differences, err := roundtrip.Compare(written, read.Corpus)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range differences {
			if !roundtrip.Counted(d, report.Omitted) {
				t.Errorf("%s: %s", name, d)
			}
		}

		archive, err := zip.OpenReader(out)
		if err != nil {
			t.Fatal(err)
		}
		var pictures []string
		var dataset string
		for _, entry := range archive.File {
			if entry.Name == "gedcom.ged" {
				open, _ := entry.Open()
				data, _ := io.ReadAll(open)
				open.Close()
				dataset = string(data)
				continue
			}
			pictures = append(pictures, entry.Name)
		}
		archive.Close()
		files := attached(written)
		for _, picture := range pictures {
			if !slices.Contains(files, picture) {
				t.Errorf("%s: the zip holds %s, which the corpus does not", name, picture)
			}
		}
		if len(pictures)+report.Omitted["files"] != len(files) || report.Pictures != len(pictures) ||
			len(report.Missing) != 0 {
			t.Errorf("%s: the zip holds %q, and the corpus %q; reported %d, missing %q",
				name, pictures, files, report.Pictures, report.Missing)
		}
		if !public {
			continue
		}
		if len(files) >= len(attached(corpus.Corpus)) {
			t.Errorf("the public view keeps every picture: %q", files)
		}
		hidden := 0
		for _, slug := range corpus.PeopleSlugs() {
			if _, ok := written.People[slug]; !ok {
				hidden++
			}
		}
		for _, said := range unsaid(t, corpus.Corpus, written) {
			if strings.Contains(dataset, said) {
				t.Errorf("the public view does not say %q, and the zip does", said)
			}
		}
		if hidden == 0 {
			t.Error("the public view hides nobody, so says nothing of hiding")
		}
	}
}

// unsaid is every line of words the corpus says that its view does not, nor any line
// of the view holds: what an export of the view must not write anywhere.
func unsaid(t *testing.T, corpus, view *model.Corpus) []string {
	t.Helper()
	said := func(c *model.Corpus) []string {
		var lines []string
		var walk func(any)
		walk = func(v any) {
			switch v := v.(type) {
			case string:
				for _, line := range strings.Split(v, "\n") {
					if line = strings.TrimSpace(line); len(line) >= 6 {
						lines = append(lines, line)
					}
				}
			case []any:
				for _, item := range v {
					walk(item)
				}
			case map[string]any:
				for _, item := range v {
					walk(item)
				}
			}
		}
		for _, kind := range model.Kinds {
			for _, slug := range c.SlugsOf(kind) {
				fields, err := store.Fields(c.Entity(slug))
				if err != nil {
					t.Fatal(err)
				}
				walk(string(slug))
				walk(fields)
			}
		}
		project, err := store.Fields(&c.Project)
		if err != nil {
			t.Fatal(err)
		}
		walk(project)
		return lines
	}
	shown := said(view)
	var hidden []string
	for _, line := range said(corpus) {
		if !slices.ContainsFunc(shown, func(s string) bool { return strings.Contains(s, line) }) &&
			!slices.Contains(hidden, line) {
			hidden = append(hidden, line)
		}
	}
	if len(hidden) == 0 {
		t.Error("the view hides no words, so says nothing of hiding")
	}
	return hidden
}

// attached is every file kept beside a record, as the path to it from the root of the
// corpus, in order.
func attached(c *model.Corpus) []string {
	var files []string
	for _, kind := range model.Kinds {
		for _, slug := range c.SlugsOf(kind) {
			if list := model.Files(c.Entity(slug)); list != nil {
				for _, file := range *list {
					files = append(files, kind.Dir()+"/"+string(slug)+"/"+file)
				}
			}
		}
	}
	slices.Sort(files)
	return files
}

// The git fixture written as public pages holds nothing the public view hides: no line
// of its words in any answer but the schema's, which says nothing of the corpus, and no
// file of a record the view leaves out.
func TestThePublicPagesSayNothingTheViewHides(t *testing.T) {
	t.Parallel()
	if _, err := os.Stat(filepath.Join("..", "..", "internal", "browse", "app", "dist", "index.html")); err != nil {
		t.Skip("the application is not built, and the pages are the application")
	}
	root, out := gitFixture(t), filepath.Join(t.TempDir(), "site")
	if code, _, stderr := invoke(t, "export", "html", "-C", root, "--public", "--out", out); code != exitOK {
		t.Fatalf("exit %d\n%s", code, stderr)
	}
	corpus, err := store.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	view := model.Hide(corpus.Corpus, model.Hiding{Private: true, Living: true}, model.DateOf(time.Now()))
	hidden := unsaid(t, corpus.Corpus, view)

	var words []string
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case string:
			words = append(words, v)
		case []any:
			for _, item := range v {
				walk(item)
			}
		case map[string]any:
			for key, item := range v {
				words = append(words, key)
				walk(item)
			}
		}
	}
	answers := 0
	err = filepath.WalkDir(filepath.Join(out, "data"), func(name string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || strings.HasSuffix(name, "chart.wasm.js") ||
			strings.HasSuffix(name, "schema.js") {
			return err
		}
		script, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		text := string(script)
		var answer any
		if err := json.Unmarshal([]byte(text[strings.Index(text, ",")+1:strings.LastIndex(text, ")")]), &answer); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		walk(answer)
		answers++
		return nil
	})
	if err != nil || answers == 0 {
		t.Fatalf("read %d answers: %v", answers, err)
	}
	for _, said := range hidden {
		if slices.ContainsFunc(words, func(w string) bool { return strings.Contains(w, said) }) {
			t.Errorf("the public view does not say %q, and the pages do", said)
		}
	}

	shown := attached(view)
	err = filepath.WalkDir(filepath.Join(out, "files"), func(name string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(filepath.Join(out, "files"), name)
		kind, rest, _ := strings.Cut(filepath.ToSlash(relative), "/")
		file := model.Kind(kind).Dir() + "/" + rest
		if !slices.Contains(shown, file) {
			t.Errorf("the pages hold %s, which the public view does not", file)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
