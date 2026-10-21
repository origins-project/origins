package chart

import (
	"math"
	"slices"
	"strconv"
	"strings"
)

// Bar is a life on a timeline: from the year of the birth to the year of the death, in
// years with their fraction, and where it is drawn. An end the index gives no date for
// is drawn as a tail of Tail years and said to be open; a life still going runs to the
// year the chart is drawn in.
type Bar struct {
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Born  bool    `json:"born"`
	Ended bool    `json:"ended"`
	X     float64 `json:"x"`
	Width float64 `json:"width"`
}

// Tick is a year marked on a timeline's axis.
type Tick struct {
	Year int     `json:"year"`
	X    float64 `json:"x"`
}

// Years is the size of a timeline: how wide a year is, how tall a row, the space above
// the rows for the axis, how far a year can be from the next marked, how long an open
// end is drawn, and the room left on the right for the words of the latest lives.
type Years struct {
	Year   float64 `json:"year"`
	Row    float64 `json:"row"`
	Axis   float64 `json:"axis"`
	Every  int     `json:"every"`
	Tail   float64 `json:"tail"`
	Beyond float64 `json:"beyond"`
}

// StandardYears is the timeline the pages and the files draw.
var StandardYears = Years{Year: 6, Row: 32, Axis: 24, Every: 10, Tail: 10, Beyond: 200}

// year is the year a sort key falls in, with its fraction.
func year(d *Dated) (float64, bool) {
	if d == nil || d.Sort == "" {
		return 0, false
	}
	parts := strings.Split(d.Sort, "-")
	if len(parts) < 3 {
		return 0, false
	}
	y, err := strconv.Atoi(strings.Join(parts[:len(parts)-2], "-"))
	m, err2 := strconv.Atoi(parts[len(parts)-2])
	day, err3 := strconv.Atoi(parts[len(parts)-1])
	if err != nil || err2 != nil || err3 != nil {
		return 0, false
	}
	return float64(y) + float64(m-1)/12 + float64(day-1)/365, true
}

// timeline is a bar for everybody of the walk the index dates the birth or the death
// of, a row each in the order their lives began, and how many it left off for want of
// either.
func timeline(walked Walked, k *Kin, g Geometry, y Years, now int) (map[string]Point, map[string]*Bar, []Tick, []string, int, float64, float64) {
	bars := map[string]*Bar{}
	var dated []string
	undated := 0
	for _, slug := range walked.Order {
		p := k.persons[slug]
		from, born := year(p.Birth)
		to, ended := year(p.Death)
		switch {
		case !born && !ended:
			undated++
			continue
		case !born:
			from = to - y.Tail
		case !ended && p.Living:
			to = max(from, float64(now))
		case !ended:
			to = from + y.Tail
		}
		bars[slug] = &Bar{From: from, To: to, Born: born, Ended: ended || p.Living}
		dated = append(dated, slug)
	}
	slices.SortStableFunc(dated, func(a, b string) int {
		switch {
		case bars[a].From < bars[b].From:
			return -1
		case bars[a].From > bars[b].From:
			return 1
		}
		return 0
	})

	boxes := map[string]Point{}
	if len(dated) == 0 {
		return boxes, bars, []Tick{}, dated, undated, g.Margin * 2, g.Margin*2 + y.Axis
	}
	first, last := math.Inf(1), math.Inf(-1)
	for _, b := range bars {
		first = min(first, b.From)
		last = max(last, b.To)
	}
	every := float64(y.Every)
	start := math.Floor(first/every) * every
	end := math.Ceil(last/every) * every
	x := func(year float64) float64 { return round(g.Margin + (year-start)*y.Year) }
	var ticks []Tick
	for at := start; at <= end; at += every {
		ticks = append(ticks, Tick{int(at), x(at)})
	}
	for i, slug := range dated {
		b := bars[slug]
		b.X = x(b.From)
		b.Width = max(round((b.To-b.From)*y.Year), 2)
		boxes[slug] = Point{b.X, g.Margin + y.Axis + float64(i)*y.Row}
	}
	return boxes, bars, ticks, dated, undated, x(end) + y.Beyond + g.Margin, g.Margin*2 + y.Axis + float64(len(dated))*y.Row
}
