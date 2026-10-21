package chart

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
)

// Geometry is the size of a box and the space around it.
type Geometry struct {
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	ColumnGap float64 `json:"column-gap"`
	RowGap    float64 `json:"row-gap"`
	Margin    float64 `json:"margin"`
}

// Standard is the geometry the pages and the files draw at.
var Standard = Geometry{Width: 208, Height: 50, ColumnGap: 56, RowGap: 12, Margin: 16}

// Point is a place on the chart, in its own pixels.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Line is an SVG path joining two boxes, and the tie it draws.
type Line struct {
	Path string `json:"path"`
	Tie  Tie    `json:"tie"`
}

// Drawing is a walk laid out: where each box goes, the lines between them and the
// size of the whole.
type Drawing struct {
	// Boxes is the top left corner of each box, a slot's by its key.
	Boxes  map[string]Point
	Lines  []Line
	Width  float64
	Height float64
}

const passes = 4

type edge struct {
	a, b string
	tie  Tie
}

// Orientation is which way the generations run: in columns, oldest on the right, or in
// rows, oldest on top.
type Orientation int

const (
	Columns Orientation = iota
	Rows
)

// Arrange is where every box of a walk goes. One person is drawn once however many ways
// the walk reached them: the columns, or the rows, come from the generations, and the
// order along each from wanting each box near its relations.
func Arrange(walked Walked, g Geometry, o Orientation) Drawing {
	nodes := slices.Clone(walked.Order)
	generation := maps.Clone(walked.Generation)
	var edges []edge
	for _, e := range walked.Edges {
		edges = append(edges, edge{e.A, e.B, e.Tie})
	}
	for _, s := range walked.Slots {
		nodes = append(nodes, s.Key)
		generation[s.Key] = s.Generation
		edges = append(edges, edge{s.Key, s.Child, GapTie})
	}
	columns := map[int][]string{}
	for _, slug := range nodes {
		columns[generation[slug]] = append(columns[generation[slug]], slug)
	}
	generations := slices.Sorted(maps.Keys(columns))
	adjacent := map[string][]string{}
	for _, e := range edges {
		adjacent[e.a] = append(adjacent[e.a], e.b)
		adjacent[e.b] = append(adjacent[e.b], e.a)
	}
	order(generation, columns, generations, adjacent)

	oldest, youngest := generations[0], generations[len(generations)-1]
	along := g.Height + g.RowGap
	if o == Rows {
		along = g.Width + g.RowGap
	}
	middles := place(walked.Subject, generation, columns, generations, adjacent, along)
	lowest := middles[walked.Subject]
	highest := lowest
	for _, at := range middles {
		lowest = min(lowest, at)
		highest = max(highest, at)
	}

	// each box, by where its line meets it: the middle of its left edge in columns and
	// of its top edge in rows
	ends := map[string]Point{}
	boxes := map[string]Point{}
	var drawing Drawing
	if o == Rows {
		shift := g.Margin + g.Width/2 - lowest
		y := func(gen int) float64 { return g.Margin + float64(gen-oldest)*(g.Height+g.ColumnGap) }
		for _, slug := range nodes {
			at := Point{middles[slug] + shift, y(generation[slug])}
			ends[slug] = at
			boxes[slug] = Point{at.X - g.Width/2, at.Y}
		}
		drawing.Width = highest - lowest + g.Width + g.Margin*2
		drawing.Height = y(youngest) + g.Height + g.Margin
	} else {
		shift := g.Margin + g.Height/2 - lowest
		x := func(gen int) float64 { return g.Margin + float64(youngest-gen)*(g.Width+g.ColumnGap) }
		for _, slug := range nodes {
			at := Point{x(generation[slug]), middles[slug] + shift}
			ends[slug] = at
			boxes[slug] = Point{at.X, at.Y - g.Height/2}
		}
		drawing.Width = x(oldest) + g.Width + g.Margin
		drawing.Height = highest - lowest + g.Height + g.Margin*2
	}
	drawing.Boxes = boxes
	drawing.Lines = make([]Line, 0, len(edges))
	for _, e := range edges {
		path := connector(ends[e.a], ends[e.b], g)
		if o == Rows {
			path = descender(ends[e.a], ends[e.b], g)
		}
		drawing.Lines = append(drawing.Lines, Line{path, e.tie})
	}
	return drawing
}

// order settles the order within each column by moving every box towards the average
// place of its relations in the columns beside it, a few sweeps each way: the
// barycentre heuristic, which stops most lines crossing.
func order(generation map[string]int, columns map[int][]string, generations []int, adjacent map[string][]string) {
	index := map[string]int{}
	for _, gen := range generations {
		for i, slug := range columns[gen] {
			index[slug] = i
		}
	}
	sweep := func(gens []int) {
		for _, gen := range gens {
			column := columns[gen]
			weight := map[string]float64{}
			for _, slug := range column {
				total, count := 0.0, 0
				for _, other := range adjacent[slug] {
					if generation[other] == gen {
						continue
					}
					total += float64(index[other])
					count++
				}
				if count == 0 {
					weight[slug] = float64(index[slug])
				} else {
					weight[slug] = total / float64(count)
				}
			}
			slices.SortStableFunc(column, func(a, b string) int { return cmp.Compare(weight[a], weight[b]) })
			for i, slug := range column {
				index[slug] = i
			}
		}
	}
	backward := slices.Clone(generations)
	slices.Reverse(backward)
	for range passes {
		sweep(generations)
		sweep(backward)
	}
}

// place is where along its column or row the middle of every box goes: the columns are
// placed outwards from the subject's, each box wanting the average place of its
// relations already placed, and each column opened out, in its order, until no two
// boxes overlap.
func place(subject string, generation map[string]int, columns map[int][]string, generations []int, adjacent map[string][]string, slot float64) map[string]float64 {
	placed := map[string]float64{}
	column := func(gen int) {
		slugs := columns[gen]
		want := make([]float64, len(slugs))
		for i, slug := range slugs {
			total, count := 0.0, 0
			for _, other := range adjacent[slug] {
				if y, ok := placed[other]; ok && generation[other] != gen {
					total += y
					count++
				}
			}
			if count == 0 {
				want[i] = float64(i) * slot
			} else {
				want[i] = total / float64(count)
			}
		}
		for i := 1; i < len(want); i++ {
			want[i] = max(want[i], want[i-1]+slot)
		}
		for i := len(want) - 2; i >= 0; i-- {
			want[i] = min(want[i], want[i+1]-slot)
		}
		for i, slug := range slugs {
			placed[slug] = want[i]
		}
	}
	start := max(slices.Index(generations, generation[subject]), 0)
	column(generations[start])
	for offset := 1; offset < len(generations); offset++ {
		if start-offset >= 0 {
			column(generations[start-offset])
		}
		if start+offset < len(generations) {
			column(generations[start+offset])
		}
	}
	return placed
}

func fixed(n float64) string { return strconv.FormatFloat(n, 'f', 1, 64) }

// connector is a flat-sided curve between two boxes, given the middles of their left
// edges. Partners share a column, so their line loops out to the left rather than
// running through the boxes between them.
func connector(from, to Point, g Geometry) string {
	if from.X == to.X {
		lo, hi := from, to
		if from.Y > to.Y {
			lo, hi = to, from
		}
		bulge := lo.X - g.ColumnGap/3
		return fmt.Sprintf("M %s %s C %s %s, %s %s, %s %s",
			fixed(lo.X), fixed(lo.Y), fixed(bulge), fixed(lo.Y), fixed(bulge), fixed(hi.Y), fixed(hi.X), fixed(hi.Y))
	}
	left, right := from, to
	if from.X > to.X {
		left, right = to, from
	}
	x1 := left.X + g.Width
	x2 := right.X
	bend := (x2 - x1) / 2
	return fmt.Sprintf("M %s %s C %s %s, %s %s, %s %s",
		fixed(x1), fixed(left.Y), fixed(x1+bend), fixed(left.Y), fixed(x2-bend), fixed(right.Y), fixed(x2), fixed(right.Y))
}

// descender is the same curve between two boxes in rows, given the middles of their top
// edges, running from the bottom of the upper to the top of the lower. Partners share a
// row, so their line loops out above it.
func descender(from, to Point, g Geometry) string {
	if from.Y == to.Y {
		left, right := from, to
		if from.X > to.X {
			left, right = to, from
		}
		bulge := left.Y - g.ColumnGap/3
		return fmt.Sprintf("M %s %s C %s %s, %s %s, %s %s",
			fixed(left.X), fixed(left.Y), fixed(left.X), fixed(bulge), fixed(right.X), fixed(bulge), fixed(right.X), fixed(right.Y))
	}
	upper, lower := from, to
	if from.Y > to.Y {
		upper, lower = to, from
	}
	y1 := upper.Y + g.Height
	y2 := lower.Y
	bend := (y2 - y1) / 2
	return fmt.Sprintf("M %s %s C %s %s, %s %s, %s %s",
		fixed(upper.X), fixed(y1), fixed(upper.X), fixed(y1+bend), fixed(lower.X), fixed(y2-bend), fixed(lower.X), fixed(y2))
}
