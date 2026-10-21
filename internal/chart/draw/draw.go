// Package draw draws a chart laid out by internal/chart to a file, as SVG, PDF or PNG,
// with tdewolff/canvas: the boxes, the lines, the names, the years and the portraits as
// the pages draw them, in Inter and in the colours of the pages' light theme, with Noto
// for the scripts Inter has no glyphs for.
//
// The chart is drawn once, in its own pixels, and each format is that drawing written
// out: an SVG at its size in pixels, a PDF at the size a browser prints those pixels,
// and a PNG at twice their resolution, less for a chart too big for that.
package draw

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/pdf"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	"github.com/tdewolff/canvas/renderers/svg"
	xdraw "golang.org/x/image/draw"

	"github.com/origins-project/origins/internal/chart"
)

// Format is what a chart is written as.
type Format string

const (
	SVG Format = "svg"
	PDF Format = "pdf"
	PNG Format = "png"
)

// Formats is every format a chart can be written as.
var Formats = []Format{SVG, PDF, PNG}

// FormatOf is the format a file's extension names, and false for one there is not.
func FormatOf(file string) (Format, bool) {
	f := Format(strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), "."))
	switch f {
	case SVG, PDF, PNG:
		return f, true
	}
	return "", false
}

// Card is somebody as a box shows them: their name and the language it is in, the
// years they lived, their recorded sex, the initials drawn where there is no picture,
// and the picture, cut to the part of it that is them where Region marks one, as
// fractions left, top, right and bottom.
type Card struct {
	Name     string
	Lang     string
	Years    string
	Sex      string
	Initials string
	Picture  image.Image
	Region   []float64
}

// The colours of a person's sex, a background and an ink, as the pages give them.
var tones = map[string][2]color.RGBA{
	"M": {male, maleInk},
	"F": {female, femaleInk},
	"":  {unknown, unknownInk},
}

// The sizes of the pages' type and their boxes, in pixels.
const (
	textXS  = 11.84
	textSM  = 12.8
	avatar  = 36
	space   = 8
	radius  = 10
	stroke  = 1.5
	picture = 2 // pixels of a portrait to each pixel of the chart

	// pixel is a pixel in millimetres, as a browser prints one, and largest the most
	// pixels a PNG is given along either side.
	pixel   = 25.4 / 96
	largest = 16384
)

func tone(sex string) [2]color.RGBA {
	if t, ok := tones[sex]; ok {
		return t
	}
	return tones[""]
}

// Write draws the chart in the format asked for, each person by their card, the
// subject's box marked as the pages mark it.
func Write(w io.Writer, c chart.Chart, cards map[string]Card, f Format) error {
	writing.Lock()
	defer writing.Unlock()
	drawn, err := drawing(c, cards)
	if err != nil {
		return err
	}
	switch f {
	case SVG:
		r := svg.New(w, c.Width, c.Height, &svg.Options{EmbedFonts: true, SubsetFonts: true, SizeUnits: "px"})
		drawn.RenderTo(outlining{r})
		return r.Close()
	case PDF:
		r := pdf.New(w, c.Width*pixel, c.Height*pixel, nil)
		drawn.RenderViewTo(r, canvas.Identity.Scale(pixel, pixel))
		return r.Close()
	case PNG:
		scale := min(picture, largest/max(c.Width, c.Height, 1))
		return png.Encode(w, rasterizer.Draw(drawn, canvas.DPMM(scale), canvas.DefaultColorSpace))
	}
	return fmt.Errorf("a chart is drawn as svg, pdf or png, and not as %s", f)
}

func drawing(c chart.Chart, cards map[string]Card) (*canvas.Canvas, error) {
	if _, err := inter.load(); err != nil {
		return nil, err
	}
	drawn := canvas.New(c.Width, c.Height)
	ctx := canvas.NewContext(drawn)
	ctx.SetCoordSystem(canvas.CartesianIV)
	ctx.SetFillColor(surface)
	ctx.DrawPath(0, 0, canvas.Rectangle(c.Width, c.Height))
	ctx.SetFillColor(canvas.Transparent)

	if len(c.Ticks) > 0 {
		axis := c.Height
		for _, p := range c.People {
			axis = min(axis, p.Y)
		}
		for _, t := range c.Ticks {
			ctx.SetStrokeColor(line)
			ctx.SetStrokeWidth(1)
			tick := &canvas.Path{}
			tick.MoveTo(t.X, axis)
			tick.LineTo(t.X, c.Height)
			ctx.DrawPath(0, 0, tick)
			write(ctx, t.X, axis-6, text(textXS, muted, canvas.FontRegular, "", fmt.Sprint(t.Year), canvas.Center))
		}
	}

	for _, l := range c.Lines {
		path, err := canvas.ParseSVGPath(l.Path)
		if err != nil {
			return nil, err
		}
		ctx.SetStrokeWidth(stroke)
		ctx.SetStrokeColor(lineStrong)
		ctx.SetDashes(0)
		switch l.Tie {
		case chart.OtherTie:
			ctx.SetDashes(0, 5, 4)
		case chart.PartnerTie:
			ctx.SetStrokeColor(line)
		case chart.GapTie:
			ctx.SetStrokeColor(line)
			ctx.SetDashes(0, 5, 4)
		}
		ctx.DrawPath(0, 0, path)
	}
	ctx.SetDashes(0)

	for _, p := range c.People {
		card := cards[p.Slug]
		if card.Name == "" {
			card.Name = p.Slug
		}
		subject := p.Slug == c.Subject
		switch {
		case p.Sector != nil:
			if err := sector(ctx, p.Sector, card); err != nil {
				return nil, err
			}
		case p.Bar != nil:
			bar(ctx, p, card, subject)
		default:
			box(ctx, c.Box, p.Point, card, subject)
		}
	}
	return drawn, nil
}

// box is a person's box: the colour of their sex, the subject's outlined, their
// portrait or initials in a circle at the left, and their name and years beside it.
func box(ctx *canvas.Context, g chart.Geometry, at chart.Point, card Card, subject bool) {
	t := tone(card.Sex)
	ctx.SetFillColor(t[0])
	ctx.SetStrokeColor(canvas.Transparent)
	ctx.DrawPath(at.X, at.Y, canvas.RoundedRectangle(g.Width, g.Height, radius))
	if subject {
		ctx.SetFillColor(canvas.Transparent)
		ctx.SetStrokeColor(accent)
		ctx.SetStrokeWidth(2)
		ctx.DrawPath(at.X+1, at.Y+1, canvas.RoundedRectangle(g.Width-2, g.Height-2, radius-1))
	}

	cx, cy := at.X+space+avatar/2, at.Y+g.Height/2
	if pic := portrait(card.Picture, card.Region); pic != nil {
		ctx.DrawImage(cx-avatar/2, cy-avatar/2, pic, canvas.DPMM(picture))
	} else {
		ctx.SetFillColor(t[0])
		ctx.SetStrokeColor(canvas.Transparent)
		ctx.DrawPath(cx, cy, canvas.Circle(avatar/2))
		write(ctx, cx, cy+textSM*0.36, text(textSM, t[1], canvas.FontMedium, card.Lang, card.Initials, canvas.Center))
	}

	x := at.X + space + avatar + space
	room := at.X + g.Width - space - x
	name := fit(textSM, ink, canvas.FontSemiBold, card.Lang, card.Name, room, canvas.Left)
	if card.Years == "" {
		write(ctx, x, cy+textSM*0.36, name)
		return
	}
	write(ctx, x, cy-4, name)
	write(ctx, x, cy+12, fit(textXS, muted, canvas.FontRegular, "", card.Years, room, canvas.Left))
}

// sector is a person of a fan: their piece of the ring in the colour of their sex, and
// their name and years turned as the layout turns them, cut to fit within it.
func sector(ctx *canvas.Context, s *chart.Sector, card Card) error {
	path, err := canvas.ParseSVGPath(s.Path)
	if err != nil {
		return err
	}
	ctx.SetFillColor(tone(card.Sex)[0])
	ctx.SetStrokeColor(surface)
	ctx.SetStrokeWidth(2)
	ctx.DrawPath(0, 0, path)

	// the words run along the ring where the layout turned them a quarter from the
	// middle of the sector, and across it otherwise
	rings := chart.StandardRings
	room := rings.Depth - space*2
	if s.Turn == (s.Start+s.End)/2-90 {
		middle := rings.Inner + (float64(s.Ring)-0.5)*rings.Depth
		room = middle*(s.End-s.Start)*math.Pi/180 - space*2
	}

	name := fit(textXS, ink, canvas.FontSemiBold, card.Lang, card.Name, room, canvas.Center)
	ctx.Push()
	ctx.Translate(s.Label.X, s.Label.Y)
	ctx.Rotate(s.Turn)
	if card.Years == "" {
		write(ctx, 0, textXS*0.35, name)
	} else {
		write(ctx, 0, -textXS*0.2, name)
		write(ctx, 0, textXS*1.05, fit(textXS, muted, canvas.FontRegular, "", card.Years, room, canvas.Center))
	}
	ctx.Pop()
	return nil
}

// bar is a person of a timeline: their name and years above a bar over their life, an
// end with no date dashed and the subject's outlined.
func bar(ctx *canvas.Context, p chart.Placed, card Card, subject bool) {
	b := p.Bar
	write(ctx, b.X, p.Y+12, text(textXS, ink, canvas.FontSemiBold, card.Lang, card.Name, canvas.Left))
	if card.Years != "" {
		after := width(textXS, canvas.FontSemiBold, card.Lang, card.Name+" ")
		write(ctx, b.X+after, p.Y+12, text(textXS, muted, canvas.FontRegular, "", card.Years, canvas.Left))
	}
	ctx.SetFillColor(tone(card.Sex)[0])
	ctx.SetStrokeColor(lineStrong)
	ctx.SetStrokeWidth(1)
	if !b.Born || !b.Ended {
		ctx.SetDashes(0, 4, 3)
	}
	if subject {
		ctx.SetStrokeColor(accent)
		ctx.SetStrokeWidth(2)
	}
	ctx.DrawPath(b.X, p.Y+17, canvas.RoundedRectangle(b.Width, 10, 3))
	ctx.SetDashes(0)
}

// fit is the words set as text sets them, cut short with an ellipsis where they are
// wider than the room.
func fit(size float64, fill color.Color, style canvas.FontStyle, lang, words string, room float64, align canvas.TextAlign) *canvas.Text {
	if t := text(size, fill, style, lang, words, align); t.Width <= room {
		return t
	}
	letters := []rune(words)
	for len(letters) > 0 {
		letters = letters[:len(letters)-1]
		cut := strings.TrimRight(string(letters), " ") + "…"
		if t := text(size, fill, style, lang, cut, align); t.Width <= room {
			return t
		}
	}
	return text(size, fill, style, lang, "…", align)
}

// portrait is the picture cut to a circle, scaled so that the region covers it and
// centred on the region, or on the picture where there is none, as the pages crop it.
func portrait(img image.Image, region []float64) image.Image {
	if img == nil {
		return nil
	}
	b := img.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	if w == 0 || h == 0 {
		return nil
	}
	left, top, right, bottom := 0.0, 0.0, 1.0, 1.0
	if len(region) == 4 && region[2] > region[0] && region[3] > region[1] {
		left, top, right, bottom = region[0], region[1], region[2], region[3]
	}
	// the square the circle is cut from, as large as the region's shorter side and
	// centred on it
	side := min((right-left)*w, (bottom-top)*h)
	cx, cy := (left+right)/2*w, (top+bottom)/2*h
	from := image.Rect(int(cx-side/2), int(cy-side/2), int(cx+side/2), int(cy+side/2)).Add(b.Min)

	size := avatar * picture
	square := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(square, square.Bounds(), img, from, xdraw.Src, nil)
	circle := image.NewRGBA(square.Bounds())
	xdraw.DrawMask(circle, circle.Bounds(), square, image.Point{}, disc(size), image.Point{}, xdraw.Over)
	return circle
}

// disc is a circle filling a square of the size, its edge smoothed over a pixel.
func disc(size int) *image.Alpha {
	out := image.NewAlpha(image.Rect(0, 0, size, size))
	r := float64(size) / 2
	for y := range size {
		for x := range size {
			d := math.Hypot(float64(x)+0.5-r, float64(y)+0.5-r)
			out.SetAlpha(x, y, color.Alpha{uint8(255 * min(1, max(0, r-d+0.5)))})
		}
	}
	return out
}

// Decode reads a picture a portrait is drawn from, and nil for a file that is not one
// it can read.
func Decode(data []byte) image.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}
