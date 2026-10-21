package chart

// Ask is a chart asked for: its shape around the subject, as far as distance, with a
// slot for each parent missing where gaps asks for them, the person a path runs to,
// and the year a life still going runs to on a timeline.
type Ask struct {
	Subject  string `json:"subject"`
	Shape    Shape  `json:"shape"`
	Distance int    `json:"distance"`
	Gaps     bool   `json:"gaps"`
	To       string `json:"to,omitempty"`
	Year     int    `json:"year,omitempty"`
}

// Chart is a chart laid out, as the pages and the files draw it: each person in the
// order the walk found them and each slot after them, each at the top left corner of
// its box, and the lines between them. A person of a fan is a sector, and of a
// timeline a bar, at the corner of the row.
type Chart struct {
	Subject   string    `json:"subject"`
	Shape     Shape     `json:"shape"`
	Distance  int       `json:"distance"`
	Box       Geometry  `json:"box"`
	Width     float64   `json:"width"`
	Height    float64   `json:"height"`
	Truncated bool      `json:"truncated"`
	Limit     int       `json:"limit"`
	People    []Placed  `json:"people"`
	Slots     []Gapped  `json:"slots"`
	Edges     []Kinship `json:"edges"`
	Lines     []Line    `json:"lines"`

	// Ticks and Undated are a timeline's: the years marked on its axis, and how many
	// people it left off for want of a date.
	Ticks   []Tick `json:"ticks,omitempty"`
	Undated int    `json:"undated,omitempty"`

	// Path is a relationship path's.
	Path *Related `json:"path,omitempty"`
}

// Placed is a person's box where the layout put it, with the ring sector of a fan or
// the bar of a timeline.
type Placed struct {
	Slug       string `json:"slug"`
	Generation int    `json:"generation"`
	Point
	Sector *Sector `json:"sector,omitempty"`
	Bar    *Bar    `json:"bar,omitempty"`
}

// Gapped is a slot's box where the layout put it.
type Gapped struct {
	Slot
	Point
}

// Draw lays out the chart asked for, and false for a shape there is not.
func Draw(k *Kin, ask Ask) (Chart, bool) {
	var walked Walked
	var related *Related
	if ask.Shape == Path {
		walked, related = path(k, ask.Subject, ask.To)
	} else {
		reach, ok := ReachOf(ask.Shape)
		if !ok {
			return Chart{}, false
		}
		walked = Walk(k, ask.Subject, reach, ask.Distance, MaxPeople)
	}
	if !ask.Gaps || ask.Shape == Fan {
		walked.Slots = nil
	}
	c := Chart{
		Subject:   ask.Subject,
		Shape:     ask.Shape,
		Distance:  ask.Distance,
		Box:       Standard,
		Truncated: walked.Truncated,
		Limit:     MaxPeople,
		People:    make([]Placed, 0, len(walked.Order)),
		Slots:     make([]Gapped, 0, len(walked.Slots)),
		Edges:     walked.Edges,
		Lines:     []Line{},
		Path:      related,
	}
	if c.Edges == nil {
		c.Edges = []Kinship{}
	}
	place := func(boxes map[string]Point) {
		for _, slug := range walked.Order {
			if at, ok := boxes[slug]; ok {
				c.People = append(c.People, Placed{Slug: slug, Generation: walked.Generation[slug], Point: at})
			}
		}
	}
	switch ask.Shape {
	case Fan:
		boxes, sectors, width, height := fan(walked, k, Standard, StandardRings)
		place(boxes)
		for i := range c.People {
			c.People[i].Sector = sectors[c.People[i].Slug]
		}
		c.Width, c.Height = width, height
	case Timeline:
		boxes, bars, ticks, dated, undated, width, height := timeline(walked, k, Standard, StandardYears, ask.Year)
		for _, slug := range dated {
			c.People = append(c.People, Placed{Slug: slug, Generation: walked.Generation[slug], Point: boxes[slug], Bar: bars[slug]})
		}
		c.Ticks, c.Undated = ticks, undated
		c.Width, c.Height = width, height
	default:
		o := Columns
		if ask.Shape == AncestorTree || ask.Shape == DescendantTree {
			o = Rows
		}
		drawing := Arrange(walked, Standard, o)
		place(drawing.Boxes)
		for _, s := range walked.Slots {
			c.Slots = append(c.Slots, Gapped{s, drawing.Boxes[s.Key]})
		}
		c.Lines = drawing.Lines
		c.Width, c.Height = drawing.Width, drawing.Height
	}
	return c, true
}
