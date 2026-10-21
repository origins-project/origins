package api

import (
	"bytes"
	"encoding/xml"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/chart"
	"github.com/origins-project/origins/internal/chart/draw"
	"github.com/origins-project/origins/internal/model"
)

// asked is a chart of somebody in the fixture, a path running to their cousin.
func asked(shape chart.Shape, out string) ChartAsked {
	a := ChartAsked{Person: "verdier-pauline", Shape: shape, Distance: chart.MaxDistance, Out: out}
	if shape == chart.Path {
		a.Distance, a.To = 0, "ershov-lawrence"
	}
	return a
}

// readBack says what a file is from its bytes, and fails where it is not what its
// extension says it is.
func readBack(t *testing.T, file string, format draw.Format) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	switch format {
	case draw.SVG:
		decoder := xml.NewDecoder(bytes.NewReader(data))
		for {
			token, err := decoder.Token()
			if err != nil {
				t.Fatalf("%s holds no svg element: %v", file, err)
			}
			if start, ok := token.(xml.StartElement); ok {
				if start.Name.Local != "svg" {
					t.Errorf("%s starts with <%s>", file, start.Name.Local)
				}
				break
			}
		}
		for {
			if _, err := decoder.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s is not XML: %v", file, err)
			}
		}
	case draw.PDF:
		if !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.HasSuffix(bytes.TrimSpace(data), []byte("%%EOF")) {
			t.Errorf("%s is not a PDF", file)
		}
	case draw.PNG:
		picture, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s is not a PNG: %v", file, err)
		}
		if b := picture.Bounds(); b.Dx() == 0 || b.Dy() == 0 {
			t.Errorf("%s is %v", file, b)
		}
	}
}

// drawnText is every run of text an SVG draws.
func drawnText(t *testing.T, file string) []string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	decoder := xml.NewDecoder(bytes.NewReader(data))
	inside := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return out
		} else if err != nil {
			t.Fatal(err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			if token.Name.Local == "tspan" {
				inside++
			}
		case xml.EndElement:
			if token.Name.Local == "tspan" {
				inside--
			}
		case xml.CharData:
			if inside > 0 {
				out = append(out, string(token))
			}
		}
	}
}

func TestEveryChartIsDrawnToEveryFormat(t *testing.T) {
	t.Parallel()
	fixture(t)
	root, dir := filepath.Join("..", "..", "test"), t.TempDir()
	for _, shape := range chart.Shapes {
		for _, format := range draw.Formats {
			out := filepath.Join(dir, string(shape)+"."+string(format))
			report, err := ExportChart(root, asked(shape, out))
			if err != nil {
				t.Fatalf("%s as %s: %v", shape, format, err)
			}
			if report.People < 2 || report.Format != format || report.Shape != shape {
				t.Errorf("%s as %s: %+v", shape, format, report)
			}
			readBack(t, out, format)
		}
	}
}

// A public chart is drawn from the hidden view, so nobody it hides is named in it,
// and the placeholders standing for them are.
func TestAPublicChartNamesNobodyItHides(t *testing.T) {
	t.Parallel()
	whole := fixture(t)
	public := model.Hiding{Private: true, Living: true}
	hidden := whole.Viewed(model.Hide(whole.Corpus, public, today()))
	shown := map[string]bool{}
	for _, p := range indexOf(hidden.Corpus, today()).People {
		shown[p.Name] = true
	}
	var gone []string
	for _, p := range indexOf(whole.Corpus, today()).People {
		if !shown[p.Name] {
			gone = append(gone, p.Name)
		}
	}
	if len(gone) == 0 {
		t.Fatal("the fixture hides nobody from a public view")
	}

	root, dir := filepath.Join("..", "..", "test"), t.TempDir()
	placeholders := 0
	for _, shape := range chart.Shapes {
		if shape == chart.Path {
			continue
		}
		out := filepath.Join(dir, string(shape)+".svg")
		a := asked(shape, out)
		a.Person, a.Hiding = "rossi-bartolomeo", public
		if _, err := ExportChart(root, a); err != nil {
			t.Fatalf("%s: %v", shape, err)
		}
		for _, text := range drawnText(t, out) {
			for _, name := range gone {
				if strings.Contains(text, name) {
					t.Errorf("the public %s chart names %s", shape, name)
				}
			}
			if text == "Private" {
				placeholders++
			}
		}
	}
	if placeholders == 0 {
		t.Error("no public chart drew anybody in the place of whoever it hides")
	}
}

func TestAChartRefusesWhatItCannotDraw(t *testing.T) {
	t.Parallel()
	fixture(t)
	root, dir := filepath.Join("..", "..", "test"), t.TempDir()
	out := filepath.Join(dir, "chart.svg")
	for name, a := range map[string]ChartAsked{
		"a shape it does not know": {Person: "verdier-pauline", Shape: "pedigree", Distance: 5, Out: out},
		"no distance":              {Person: "verdier-pauline", Shape: chart.Ancestors, Out: out},
		"too far": {Person: "verdier-pauline", Shape: chart.Ancestors, Distance: chart.MaxDistance + 1,
			Out: out},
		"an extension it does not know": {Person: "verdier-pauline", Shape: chart.Ancestors, Distance: 5,
			Out: filepath.Join(dir, "chart.jpg")},
		"a path to nobody": {Person: "verdier-pauline", Shape: chart.Path, Out: out},
		"a tree to somebody": {Person: "verdier-pauline", Shape: chart.Ancestors, Distance: 5,
			To: "ershov-lawrence", Out: out},
		"a place": {Person: "firenze", Shape: chart.Ancestors, Distance: 5, Out: out},
		"somebody hidden": {Person: "marchena-solveigarson-sasha", Shape: chart.Ancestors, Distance: 5, Out: out,
			Hiding: model.Hiding{Living: true}},
	} {
		if _, err := ExportChart(root, a); err == nil {
			t.Errorf("drew a chart of %s", name)
		}
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("a refusal wrote the file: %v", err)
	}
}

func TestInitialsAreThePages(t *testing.T) {
	for name, want := range map[string]string{
		"Giovanni Battista de Rossi": "GR",
		"Maria":                      "M",
		"زينب الحلبي":                "زح",
		"מרים כהן":                   "מכ",
		"रमेश विठ्ठल":                "रव",
		"陳家蘭":                        "陳",
		"박서연":                        "박",
		"山田 太郎":                      "山",
		"Gabriele d'Annunzio":        "GA",
		"Zaynab al-Halabi":           "ZH",
		"Maria dell'Acqua":           "MA",
		"Seán O'Brien":               "SO",
		"maria rossi":                "M",
		"":                           "",
	} {
		if got := initials(name); got != want {
			t.Errorf("initials(%q) = %q, and the pages make %q", name, got, want)
		}
	}
}
