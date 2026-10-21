package chart

import (
	"bytes"
	"encoding/json"
	"flag"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata")

// The fixture's index, as internal/api writes it for the pages.
func fixture(t *testing.T) Index {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "api", "testdata", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index Index
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	return index
}

// The person the fixture's project names as its main one, whom every path runs to.
func main(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "api", "testdata", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Project struct {
			MainPerson string `json:"main-person"`
		} `json:"project"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	return index.Project.MainPerson
}

type snapshot struct {
	Subject   string   `json:"subject"`
	Shape     Shape    `json:"shape"`
	Distance  int      `json:"distance"`
	Width     float64  `json:"width"`
	Height    float64  `json:"height"`
	Truncated bool     `json:"truncated"`
	Boxes     [][]any  `json:"boxes"`
	Gaps      [][]any  `json:"gaps"`
	Edges     [][]any  `json:"edges"`
	Sectors   [][]any  `json:"sectors,omitempty"`
	Bars      [][]any  `json:"bars,omitempty"`
	Ticks     []int    `json:"ticks,omitempty"`
	Undated   int      `json:"undated,omitempty"`
	Path      *Related `json:"path,omitempty"`
}

func hundredths(n float64) float64 { return math.Round(n*100) / 100 }

var inline = regexp.MustCompile(`\[\s+([^\[\]{}]*?)\s+\]`)
var commas = regexp.MustCompile(`,\s+`)

// Every chart of every person of the fixture, at two distances, a path to the main
// person, written out one person and one tie to a line, so that a change meant to move
// one is a diff somebody reads.
func TestTheChartsOfTheFixture(t *testing.T) {
	t.Parallel()
	k := KinOf(fixture(t))
	to := main(t)
	var charts []snapshot
	for _, subject := range k.People() {
		for _, shape := range Shapes {
			distances := []int{2, 5}
			if shape == Path {
				distances = []int{0}
			}
			for _, distance := range distances {
				c, _ := Draw(k, Ask{Subject: subject, Shape: shape, Distance: distance, Gaps: true, To: to, Year: 2026})
				s := snapshot{
					Subject: subject, Shape: shape, Distance: distance,
					Width: c.Width, Height: c.Height, Truncated: c.Truncated,
					Boxes: [][]any{}, Gaps: [][]any{}, Edges: [][]any{},
					Undated: c.Undated, Path: c.Path,
				}
				for _, p := range c.People {
					if p.Sector != nil {
						s.Sectors = append(s.Sectors, []any{p.Slug, p.Sector.Ring, hundredths(p.Sector.Start), hundredths(p.Sector.End)})
					}
					if p.Bar != nil {
						s.Bars = append(s.Bars, []any{p.Slug, hundredths(p.Bar.From), hundredths(p.Bar.To), p.Bar.Born, p.Bar.Ended})
					}
				}
				for _, tick := range c.Ticks {
					s.Ticks = append(s.Ticks, tick.Year)
				}
				for _, p := range c.People {
					s.Boxes = append(s.Boxes, []any{p.Slug, p.Generation, p.X, p.Y})
				}
				for _, g := range c.Slots {
					var family, partner any
					if g.Family != "" {
						family = g.Family
					}
					if g.Partner != nil {
						partner = *g.Partner
					}
					s.Gaps = append(s.Gaps, []any{g.Key, g.Generation, g.X, g.Y, family, partner})
				}
				for _, e := range c.Edges {
					s.Edges = append(s.Edges, []any{e.A, e.B, e.Tie})
				}
				charts = append(charts, s)
			}
		}
	}
	text, err := json.MarshalIndent(charts, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	text = inline.ReplaceAllFunc(text, func(m []byte) []byte {
		inner := inline.FindSubmatch(m)[1]
		return append(append([]byte("["), commas.ReplaceAll(inner, []byte(", "))...), ']')
	})
	text = append(text, '\n')
	path := filepath.Join("testdata", "charts.json")
	if *update {
		if err := os.WriteFile(path, text, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(text, want) {
		t.Errorf("the charts are not %s; run go test ./internal/chart -update and read the diff", path)
	}
}

func person(slug string) Person { return Person{Ref: "person/" + slug} }

type child struct {
	person, relation, status string
}

func family(slug string, partners []string, children ...child) Family {
	f := Family{Ref: "family/" + slug}
	for _, p := range partners {
		f.Partners = append(f.Partners, Partner{Person: p})
	}
	for _, c := range children {
		one := Child{Person: c.person, Status: c.status}
		for _, p := range partners {
			one.Relations = append(one.Relations, Relation{Partner: p, Relation: c.relation})
		}
		f.Children = append(f.Children, one)
	}
	return f
}

func corpus(people []string, families ...Family) *Kin {
	index := Index{Families: families}
	for _, p := range people {
		index.People = append(index.People, person(p))
	}
	return KinOf(index)
}

func walk(k *Kin, subject string, shape Shape, distance int) Walked {
	reach, _ := ReachOf(shape)
	return Walk(k, subject, reach, distance, MaxPeople)
}

// three generations deep, with an adoption in the middle
func line() *Kin {
	return corpus([]string{"one", "two", "three"},
		family("one", []string{"one"}, child{person: "two"}),
		family("two", []string{"two"}, child{person: "three", relation: "adoptive"}))
}

func expect[T any](t *testing.T, got, want T) {
	t.Helper()
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(want)
	if !bytes.Equal(g, w) {
		t.Errorf("got %s, want %s", g, w)
	}
}

func TestTheWalkStopsAtTheDistance(t *testing.T) {
	t.Parallel()
	expect(t, walk(line(), "three", Ancestors, 1).Order, []string{"three", "two"})
	expect(t, walk(line(), "three", Ancestors, 2).Order, []string{"three", "two", "one"})
}

func TestTheWalkLooksOnlyItsOwnWay(t *testing.T) {
	t.Parallel()
	expect(t, walk(line(), "one", Ancestors, 5).Order, []string{"one"})
	expect(t, walk(line(), "one", Descendants, 5).Order, []string{"one", "two", "three"})
	expect(t, walk(line(), "three", Descendants, 5).Order, []string{"three"})
}

func TestOnlyAllRelativesTurn(t *testing.T) {
	t.Parallel()
	hourglass := corpus([]string{"parent", "self", "sibling", "child"},
		family("parent", []string{"parent"}, child{person: "self"}, child{person: "sibling"}),
		family("self", []string{"self"}, child{person: "child"}))
	both := slices.Sorted(slices.Values(walk(hourglass, "self", Both, 5).Order))
	expect(t, both, []string{"child", "parent", "self"})
	if !slices.Contains(walk(hourglass, "self", Relatives, 5).Order, "sibling") {
		t.Error("all relatives do not reach the sibling")
	}
}

func TestOnlyAllRelativesReachThroughAPartner(t *testing.T) {
	t.Parallel()
	married := corpus([]string{"self", "spouse", "inlaw"},
		family("inlaw", []string{"inlaw"}, child{person: "spouse"}),
		family("self--spouse", []string{"self", "spouse"}))
	walked := walk(married, "self", Relatives, 5)
	expect(t, walked.Order, []string{"self", "spouse", "inlaw"})
	if !slices.Contains(walked.Edges, Kinship{"self", "spouse", PartnerTie}) {
		t.Errorf("no partnership among %v", walked.Edges)
	}
	expect(t, walk(married, "self", Ancestors, 5).Order, []string{"self"})
}

func TestARelationNotByBirthIsTiedDifferently(t *testing.T) {
	t.Parallel()
	for _, w := range []Walked{walk(line(), "three", Ancestors, 5), walk(line(), "one", Descendants, 5)} {
		if !slices.Contains(w.Edges, Kinship{"two", "three", OtherTie}) {
			t.Errorf("no adoption among %v", w.Edges)
		}
	}
}

func TestTheWalkFollowsNoLinkDisprovenNoRelationNoneAndNobodyWithoutARecord(t *testing.T) {
	t.Parallel()
	k := corpus([]string{"mother", "child", "stranger"},
		family("mother--unknown", []string{"mother", "unknown"}, child{person: "child"}, child{person: "stranger", status: "disproven"}),
		family("stranger", []string{"stranger"}, child{person: "child", relation: "none"}))
	expect(t, walk(k, "child", Ancestors, 5).Order, []string{"child", "mother"})
	expect(t, walk(k, "mother", Descendants, 5).Order, []string{"mother", "child"})
}

func TestTheWalkSurvivesACircleOfAncestry(t *testing.T) {
	t.Parallel()
	circle := corpus([]string{"one", "two"},
		family("one", []string{"one"}, child{person: "two"}),
		family("two", []string{"two"}, child{person: "one"}))
	expect(t, walk(circle, "one", Ancestors, 5).Order, []string{"one", "two"})
	expect(t, walk(circle, "one", Relatives, 5).Order, []string{"one", "two"})
}

func TestAGapIsLeftForEachParentUnknown(t *testing.T) {
	t.Parallel()
	k := corpus([]string{"mother", "child", "alone", "adopted"},
		family("mother--unknown", []string{"mother", "unknown"}, child{person: "child"}),
		family("alone--none", []string{"alone", "none"}, child{person: "adopted", relation: "adoptive"}))
	gaps := func(subject string) []Gap {
		var out []Gap
		for _, s := range walk(k, subject, Ancestors, 5).Slots {
			out = append(out, s.Gap)
		}
		return out
	}
	one := 1
	expect(t, gaps("child"), []Gap{{Child: "child", Family: "mother--unknown", Partner: &one}, {Child: "mother"}, {Child: "mother"}})
	expect(t, gaps("adopted"), []Gap{{Child: "alone"}, {Child: "alone"}})
}

func TestGapsAreLeftOnlyLookingUpWithinTheDistance(t *testing.T) {
	t.Parallel()
	slots := func(subject string, shape Shape, distance int) []string {
		out := []string{}
		for _, s := range walk(line(), subject, shape, distance).Slots {
			out = append(out, s.Child)
		}
		return out
	}
	expect(t, slots("three", Ancestors, 5), []string{"one", "one"})
	expect(t, slots("three", Ancestors, 2), []string{})
	expect(t, slots("two", Both, 5), []string{"one", "one"})
	expect(t, slots("one", Descendants, 5), []string{})
	expect(t, slots("one", Both, 5), []string{"one", "one"})
	expect(t, slots("three", Relatives, 5), []string{})
}

func TestTheWalkStopsAtTheLimitAndSaysSo(t *testing.T) {
	t.Parallel()
	reach, _ := ReachOf(Descendants)
	walked := Walk(line(), "one", reach, 5, 2)
	expect(t, walked.Order, []string{"one", "two"})
	if !walked.Truncated {
		t.Error("not truncated")
	}
}

func TestTheOldestAreOnTheRight(t *testing.T) {
	t.Parallel()
	d := Arrange(walk(line(), "three", Ancestors, 5), Standard, Columns)
	if !(d.Boxes["three"].X < d.Boxes["two"].X && d.Boxes["two"].X < d.Boxes["one"].X) {
		t.Errorf("columns %v", d.Boxes)
	}
}

func TestSomebodyAloneIsOneBox(t *testing.T) {
	t.Parallel()
	c, _ := Draw(line(), Ask{Subject: "three", Shape: Descendants, Distance: 5, Gaps: true})
	expect(t, len(c.People)+len(c.Slots), 1)
	expect(t, c.Lines, []Line{})
}

func TestAGapIsInTheColumnOfTheParents(t *testing.T) {
	t.Parallel()
	c, _ := Draw(line(), Ask{Subject: "one", Shape: Ancestors, Distance: 5, Gaps: true})
	var keys, ties []string
	for _, s := range c.Slots {
		keys = append(keys, s.Key)
	}
	for _, l := range c.Lines {
		ties = append(ties, string(l.Tie))
	}
	expect(t, keys, []string{"one 0", "one 1"})
	if c.Slots[0].X <= c.People[0].X {
		t.Errorf("the gap is at %v, the child at %v", c.Slots[0].Point, c.People[0].Point)
	}
	expect(t, ties, []string{"gap", "gap"})
	none, _ := Draw(line(), Ask{Subject: "one", Shape: Ancestors, Distance: 5})
	expect(t, len(none.Slots), 0)
}

func TestAShapeThereIsNotIsRefused(t *testing.T) {
	t.Parallel()
	if _, ok := Draw(line(), Ask{Subject: "one", Shape: Shape("pedigree"), Distance: 5, Gaps: true}); ok {
		t.Error("drew a shape there is not")
	}
}

func TestATreeUpwardHasTheOldestOnTop(t *testing.T) {
	t.Parallel()
	c, _ := Draw(line(), Ask{Subject: "three", Shape: AncestorTree, Distance: 5})
	y := map[string]float64{}
	for _, p := range c.People {
		y[p.Slug] = p.Y
	}
	if !(y["one"] < y["two"] && y["two"] < y["three"]) {
		t.Errorf("rows %v", y)
	}
	down, _ := Draw(line(), Ask{Subject: "one", Shape: DescendantTree, Distance: 5})
	if down.People[0].Slug != "one" || down.People[0].Y >= down.People[2].Y {
		t.Errorf("the descendants of one are %v", down.People)
	}
}

func TestAFanHalvesEachRingBetweenTheParents(t *testing.T) {
	t.Parallel()
	k := corpus([]string{"self", "father", "mother", "grandfather"},
		family("father--mother", []string{"father", "mother"}, child{person: "self"}),
		family("grandfather", []string{"grandfather"}, child{person: "father"}))
	c, _ := Draw(k, Ask{Subject: "self", Shape: Fan, Distance: 5})
	sectors := map[string][2]float64{}
	for _, p := range c.People {
		if p.Sector != nil {
			sectors[p.Slug] = [2]float64{p.Sector.Start, p.Sector.End}
		}
	}
	expect(t, sectors, map[string][2]float64{"father": {0, 90}, "mother": {90, 180}, "grandfather": {0, 45}})
	expect(t, len(c.Lines), 0)
}

func TestAPathNamesTheRelationAsKinshipDoes(t *testing.T) {
	t.Parallel()
	k := corpus([]string{"parent", "self", "sibling", "nephew", "stranger"},
		family("parent", []string{"parent"}, child{person: "self"}, child{person: "sibling"}),
		family("sibling", []string{"sibling"}, child{person: "nephew", relation: "adoptive"}))
	c, _ := Draw(k, Ask{Subject: "nephew", Shape: Path, To: "self"})
	var order []string
	for _, p := range c.People {
		order = append(order, p.Slug)
	}
	expect(t, order, []string{"nephew", "sibling", "parent", "self"})
	expect(t, c.Path, &Related{To: "self", Related: true, Relation: "niece or nephew", CommonAncestor: "parent"})
	expect(t, c.Edges, []Kinship{{"sibling", "nephew", OtherTie}, {"parent", "sibling", BirthTie}, {"parent", "self", BirthTie}})

	none, _ := Draw(k, Ask{Subject: "self", Shape: Path, To: "stranger"})
	expect(t, none.Path, &Related{To: "stranger"})
	expect(t, len(none.People), 2)
	alone, _ := Draw(k, Ask{Subject: "self", Shape: Path})
	expect(t, len(alone.People), 1)
}

func TestATimelineLeavesOffAndCountsALifeWithNoDates(t *testing.T) {
	t.Parallel()
	index := Index{
		People: []Person{
			{Ref: "person/old", Birth: &Dated{Sort: "1800-01-01"}, Death: &Dated{Sort: "1850-07-01"}},
			{Ref: "person/young", Birth: &Dated{Sort: "1990-01-01"}, Living: true},
			{Ref: "person/lost", Birth: &Dated{Sort: "1830-01-01"}},
			{Ref: "person/blank"},
		},
		Families: []Family{family("old", []string{"old"}, child{person: "lost"}, child{person: "blank"}), family("lost", []string{"lost"}, child{person: "young"})},
	}
	c, _ := Draw(KinOf(index), Ask{Subject: "old", Shape: Timeline, Distance: 5, Year: 2026})
	var rows []string
	for _, p := range c.People {
		rows = append(rows, p.Slug)
	}
	expect(t, rows, []string{"old", "lost", "young"})
	expect(t, c.Undated, 1)
	bars := map[string]Bar{}
	for _, p := range c.People {
		bars[p.Slug] = *p.Bar
	}
	if b := bars["young"]; b.To != 2026 || !b.Ended {
		t.Errorf("a life still going is %+v", b)
	}
	if b := bars["lost"]; b.Ended || b.To != 1830+StandardYears.Tail {
		t.Errorf("a life with no end is %+v", b)
	}
	expect(t, c.Ticks[0].Year, 1800)
	expect(t, c.Ticks[len(c.Ticks)-1].Year, 2030)
}
