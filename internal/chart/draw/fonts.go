package draw

import (
	_ "embed"
	"image/color"
	"sync"
	"unicode"

	"github.com/tdewolff/canvas"
)

var (
	//go:embed fonts/Inter-Regular.ttf
	regular []byte
	//go:embed fonts/Inter-Medium.ttf
	medium []byte
	//go:embed fonts/Inter-SemiBold.ttf
	semibold []byte

	//go:embed fonts/NotoSansArabic-Regular.ttf
	arabicRegular []byte
	//go:embed fonts/NotoSansArabic-Medium.ttf
	arabicMedium []byte
	//go:embed fonts/NotoSansArabic-SemiBold.ttf
	arabicSemiBold []byte
	//go:embed fonts/NotoSansHebrew-Regular.ttf
	hebrewRegular []byte
	//go:embed fonts/NotoSansHebrew-Medium.ttf
	hebrewMedium []byte
	//go:embed fonts/NotoSansHebrew-SemiBold.ttf
	hebrewSemiBold []byte
	//go:embed fonts/NotoSansDevanagari-Regular.ttf
	devanagariRegular []byte
	//go:embed fonts/NotoSansDevanagari-Medium.ttf
	devanagariMedium []byte
	//go:embed fonts/NotoSansDevanagari-SemiBold.ttf
	devanagariSemiBold []byte

	// the ideographs, kana and Hangul of the whole of East Asia, the forms of a region
	// chosen by the language a name is in, as Japan, Korea, Taiwan or mainland China
	// writes them
	//go:embed fonts/NotoSansCJK-Regular.ttf
	cjkRegular []byte
	//go:embed fonts/NotoSansCJK-Medium.ttf
	cjkMedium []byte
)

// A family is loaded the first time anything is written in it, and kept.
type family struct {
	name   string
	fonts  []loaded
	once   sync.Once
	family *canvas.FontFamily
	first  *canvas.Font
	err    error
}

type loaded struct {
	data  []byte
	style canvas.FontStyle
}

func (f *family) load() (*canvas.FontFamily, error) {
	f.once.Do(func() {
		f.family = canvas.NewFontFamily(f.name)
		for _, one := range f.fonts {
			if f.err = f.family.LoadFont(one.data, 0, one.style); f.err != nil {
				return
			}
		}
		f.first = f.family.Face(1).Font
	})
	return f.family, f.err
}

// has is whether the family has a glyph for the rune.
func (f *family) has(r rune) bool {
	_, err := f.load()
	return err == nil && f.first.GlyphIndex(r) != 0
}

// Inter is the face of everything, and each fallback is Noto for what Inter has no
// glyphs for, in the weights the chart writes in, the ideographs' semibold being their
// medium.
var (
	inter     = &family{name: "Inter", fonts: []loaded{{regular, canvas.FontRegular}, {medium, canvas.FontMedium}, {semibold, canvas.FontSemiBold}}}
	fallbacks = []*family{
		{name: "Noto Sans Arabic", fonts: []loaded{{arabicRegular, canvas.FontRegular}, {arabicMedium, canvas.FontMedium}, {arabicSemiBold, canvas.FontSemiBold}}},
		{name: "Noto Sans Hebrew", fonts: []loaded{{hebrewRegular, canvas.FontRegular}, {hebrewMedium, canvas.FontMedium}, {hebrewSemiBold, canvas.FontSemiBold}}},
		{name: "Noto Sans Devanagari", fonts: []loaded{{devanagariRegular, canvas.FontRegular}, {devanagariMedium, canvas.FontMedium}, {devanagariSemiBold, canvas.FontSemiBold}}},
		{name: "Noto Sans CJK", fonts: []loaded{{cjkRegular, canvas.FontRegular}, {cjkMedium, canvas.FontMedium}}},
	}
)

// writing is held while a chart is drawn, since a font shapes text with state of its
// own.
var writing sync.Mutex

// run is words written in one family.
type run struct {
	family *family
	words  []rune
}

// runs cuts the words where the family that has their glyphs changes, Inter first and
// then each fallback in turn. A space, a digit or a mark stays in the run it is in
// where that family has it, so that the words around it are shaped together.
func runs(words string) ([]run, error) {
	chain := append([]*family{inter}, fallbacks...)
	if _, err := inter.load(); err != nil {
		return nil, err
	}

	var out []run
	for _, r := range words {
		if n := len(out); n > 0 {
			last := &out[n-1]
			joins := unicode.In(r, unicode.Inherited, unicode.Mn, unicode.Me, unicode.Cf)
			if joins || unicode.Is(unicode.Common, r) && last.family.has(r) {
				last.words = append(last.words, r)
				continue
			}
		}
		chosen := inter
		for _, f := range chain {
			if f.has(r) {
				chosen = f
				break
			}
		}
		if n := len(out); n > 0 && out[n-1].family == chosen {
			out[n-1].words = append(out[n-1].words, r)
			continue
		}
		out = append(out, run{chosen, []rune{r}})
	}
	return out, nil
}

// text is the words set in a size, a colour and a weight, each run in the family that
// has its glyphs, shaped, and laid in the order its script reads, right to left
// where it does, as one line aligned on the point it is drawn at. Its baseline is at
// that point.
func text(size float64, fill color.Color, style canvas.FontStyle, lang, words string, align canvas.TextAlign) *canvas.Text {
	pieces, err := runs(words)
	if err != nil || len(pieces) == 0 {
		pieces = []run{{inter, nil}}
	}
	face := func(f *family) *canvas.FontFace {
		fam, _ := f.load()
		// a face is sized in points, and the drawing is in pixels
		face := fam.Face(size/25.4*72, fill, style)
		// thickening a weight it lacks breaks joined script apart, so the nearest one
		// stands in
		face.FauxBold = 0
		face.Language = lang
		return face
	}
	rich := canvas.NewRichText(face(pieces[0].family))
	for _, p := range pieces {
		rich.WriteFace(face(p.family), string(p.words))
	}
	return rich.ToText(0, 0, align, canvas.Top, nil)
}

func width(size float64, style canvas.FontStyle, lang, words string) float64 {
	return text(size, ink, style, lang, words, canvas.Left).Width
}

// outlining writes text set in Inter as text, and text set in any fallback as its
// outlines: an SVG is shaped again by whoever opens it, from a font cut down to the
// glyphs its characters name, without the joined, conjunct and regional forms the
// chart was shaped with.
type outlining struct {
	canvas.Renderer
}

func (o outlining) RenderText(t *canvas.Text, m canvas.Matrix) {
	for _, f := range t.Fonts() {
		if !inter.owns(f) {
			t.RenderTo(o.Renderer, m, 0)
			return
		}
	}
	o.Renderer.RenderText(t, m)
}

// owns is whether the font is one of the family's.
func (f *family) owns(font *canvas.Font) bool {
	fam, err := f.load()
	if err != nil {
		return false
	}
	for _, one := range f.fonts {
		if fam.Face(1, one.style).Font == font {
			return true
		}
	}
	return false
}

// write draws the text with its first baseline at the point.
func write(ctx *canvas.Context, x, y float64, t *canvas.Text) {
	baseline := 0.0
	t.WalkLines(func(y float64, _ []canvas.TextSpan) {
		if baseline == 0 {
			baseline = y
		}
	})
	ctx.DrawText(x, y+baseline, t)
}
