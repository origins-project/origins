package chart

import (
	"fmt"
	"math"
)

// Sector is where a person of a fan is drawn: a piece of a half ring, its angles in
// degrees clockwise from the left, the outline to draw, and where the words go and how
// far they are turned, along the ring where there is room and across it where there is
// not.
type Sector struct {
	Ring  int     `json:"ring"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Path  string  `json:"path"`
	Label Point   `json:"label"`
	Turn  float64 `json:"turn"`
}

// Rings is the size of a fan: the half disc the subject's box stands in, and how deep
// each ring around it is.
type Rings struct {
	Inner float64 `json:"inner"`
	Depth float64 `json:"depth"`
}

// StandardRings is the fan the pages and the files draw.
var StandardRings = Rings{Inner: 124, Depth: 110}

// fan lays the ancestors of a walk out in half rings, a generation to a ring, the
// subject in a box at the middle. Each person's half ring is shared equally between
// their parents, never fewer than two shares so that a parent missing is a gap, and
// a person the walk reached twice is drawn where it reached them first, their other
// place left empty.
func fan(walked Walked, k *Kin, g Geometry, r Rings) (map[string]Point, map[string]*Sector, float64, float64) {
	type span struct{ start, end float64 }
	spans := map[string]span{walked.Subject: {0, 180}}
	for _, slug := range walked.Order {
		s, ok := spans[slug]
		if !ok {
			continue
		}
		var parents []string
		for _, p := range k.Parents(slug) {
			if gen, ok := walked.Generation[p.Slug]; ok && gen == walked.Generation[slug]-1 {
				parents = append(parents, p.Slug)
			}
		}
		share := (s.end - s.start) / float64(max(2, len(parents)))
		for i, p := range parents {
			if _, placed := spans[p]; !placed {
				spans[p] = span{s.start + float64(i)*share, s.start + float64(i+1)*share}
			}
		}
	}
	rings := 1
	for slug := range spans {
		rings = max(rings, -walked.Generation[slug])
	}
	outer := r.Inner + float64(rings)*r.Depth
	centre := Point{g.Margin + outer, g.Margin + outer}

	boxes := map[string]Point{}
	sectors := map[string]*Sector{}
	for _, slug := range walked.Order {
		s, ok := spans[slug]
		if !ok {
			continue
		}
		ring := -walked.Generation[slug]
		if ring == 0 {
			boxes[slug] = Point{centre.X - g.Width/2, centre.Y - g.Height - g.RowGap}
			continue
		}
		inner := r.Inner + float64(ring-1)*r.Depth
		mid := inner + r.Depth/2
		middle := (s.start + s.end) / 2
		label := at(centre, mid, middle)
		turn := middle - 90
		if mid*radians(s.end-s.start) < r.Depth*1.2 {
			turn = middle
			if middle > 90 {
				turn = middle - 180
			}
		}
		sectors[slug] = &Sector{
			Ring:  ring,
			Start: s.start,
			End:   s.end,
			Path:  sector(centre, inner, inner+r.Depth, s.start, s.end),
			Label: label,
			Turn:  turn,
		}
		boxes[slug] = label
	}
	return boxes, sectors, centre.X * 2, centre.Y + g.Margin
}

func radians(degrees float64) float64 { return degrees * math.Pi / 180 }

// at is the point at a distance from the centre, at an angle clockwise from the left.
func at(centre Point, distance, angle float64) Point {
	θ := radians(180 + angle)
	return Point{round(centre.X + distance*math.Cos(θ)), round(centre.Y + distance*math.Sin(θ))}
}

func round(n float64) float64 { return math.Round(n*10) / 10 }

// sector is the outline of a piece of a half ring: out along the start, round the outer
// edge, back along the end and round the inner edge.
func sector(centre Point, inner, outer, start, end float64) string {
	a, b := at(centre, inner, start), at(centre, outer, start)
	c, d := at(centre, outer, end), at(centre, inner, end)
	large := 0
	if end-start > 180 {
		large = 1
	}
	return fmt.Sprintf("M %s %s L %s %s A %s %s 0 %d 1 %s %s L %s %s A %s %s 0 %d 0 %s %s Z",
		fixed(a.X), fixed(a.Y), fixed(b.X), fixed(b.Y),
		fixed(outer), fixed(outer), large, fixed(c.X), fixed(c.Y),
		fixed(d.X), fixed(d.Y),
		fixed(inner), fixed(inner), large, fixed(a.X), fixed(a.Y))
}
