package draw

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/tdewolff/canvas"
	ctext "github.com/tdewolff/canvas/text"

	"github.com/origins-project/origins/internal/chart"
)

// recording keeps the text drawn on it.
type recording struct {
	texts []*canvas.Text
}

func (r *recording) Size() (float64, float64)                             { return 0, 0 }
func (r *recording) RenderPath(*canvas.Path, canvas.Style, canvas.Matrix) {}
func (r *recording) RenderImage(image.Image, canvas.Matrix)               {}
func (r *recording) RenderText(t *canvas.Text, _ canvas.Matrix)           { r.texts = append(r.texts, t) }

var scripts = []struct {
	name, lang string
	rtl        bool
}{
	{"Giovanni Rossi", "it", false},
	{"Ελένη Παπαδοπούλου", "el", false},
	{"Анна Иванова", "ru", false},
	{"زينب الحلبي", "ar", true},
	{"پروین رحیمی", "fa", true},
	{"מרים כהן", "he", true},
	{"陳家蘭", "zh-Hant", false},
	{"佐藤由美子", "ja", false},
	{"さとう ゆみこ", "ja", false},
	{"박서연", "ko", false},
	{"रमेश विठ्ठल", "mr", false},
}

func TestEveryScriptIsDrawnInGlyphsItHas(t *testing.T) {
	c := chart.Chart{Subject: "p0", Shape: chart.Relatives, Box: chart.Standard}
	cards := map[string]Card{}
	for i, s := range scripts {
		slug := "p" + string(rune('a'+i))
		y := 16 + float64(i)*(chart.Standard.Height+chart.Standard.RowGap)
		c.People = append(c.People, chart.Placed{Slug: slug, Point: chart.Point{X: 16, Y: y}})
		cards[slug] = Card{Name: s.name, Lang: s.lang, Years: "1900 – 1980"}
	}
	c.Width = chart.Standard.Width + 32
	c.Height = float64(len(scripts)) * (chart.Standard.Height + chart.Standard.RowGap)

	writing.Lock()
	drawn, err := drawing(c, cards)
	writing.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	r := &recording{}
	drawn.RenderTo(r)

	for _, s := range scripts {
		found := false
		for _, txt := range r.texts {
			if !strings.Contains(txt.String(), s.name) {
				continue
			}
			found = true
			txt.WalkSpans(func(_, _ float64, span canvas.TextSpan) {
				for _, g := range span.Glyphs {
					if g.ID == 0 {
						t.Errorf("%s (%s) has no glyph for %q in %s", s.name, s.lang, span.Text, span.Face.Font.Name())
					}
				}
				if rtl := span.Direction == ctext.RightToLeft; rtl != s.rtl && strings.TrimSpace(span.Text) != "" {
					t.Errorf("%s (%s) is written right to left %v, and should be %v", s.name, s.lang, rtl, s.rtl)
				}
			})
		}
		if !found {
			t.Errorf("%s (%s) is not drawn", s.name, s.lang)
		}
	}

	for _, f := range Formats {
		var out bytes.Buffer
		if err := Write(&out, c, cards, f); err != nil {
			t.Errorf("%s: %v", f, err)
		}
		if f == SVG && strings.Contains(out.String(), "Noto") {
			t.Errorf("the SVG embeds a fallback, which a viewer would shape again")
		}
	}
}

func TestAnIdeographIsWrittenAsItsRegionWritesIt(t *testing.T) {
	glyph := func(lang, ideograph string) uint16 {
		writing.Lock()
		defer writing.Unlock()
		var id uint16
		text(textSM, ink, canvas.FontRegular, lang, ideograph, canvas.Left).WalkSpans(func(_, _ float64, span canvas.TextSpan) {
			id = span.Glyphs[0].ID
		})
		return id
	}
	if ja, hans, hant := glyph("ja", "骨"), glyph("zh-Hans", "骨"), glyph("zh-Hant", "骨"); ja == hans || hans == hant || ja == hant {
		t.Errorf("骨 is not one glyph each in Japanese (%d), simplified (%d) and traditional (%d) Chinese", ja, hans, hant)
	}
	if ja, ko := glyph("ja", "誤"), glyph("ko", "誤"); ja == ko {
		t.Errorf("誤 is the same glyph in Japanese and Korean (%d)", ja)
	}
}
