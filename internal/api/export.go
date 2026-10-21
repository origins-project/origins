package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/origins-project/origins/internal/chart"
	"github.com/origins-project/origins/internal/chart/draw"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// An export is the corpus handed to somebody who does not use origins, always as a
// hidden view of it: model.Hide of the corpus, as it reads with the hypotheses of with
// in it, hiding nothing where nothing is asked to be hidden.

// ChartAsked is a chart asked to be drawn to a file: its shape around the person, as far
// as distance, the person a path runs to, and the file, whose extension says what it is
// drawn as.
type ChartAsked struct {
	Person   string
	Shape    chart.Shape
	Distance int
	To       string
	Out      string
	Hiding   model.Hiding
	With     []model.Slug
}

// ChartDrawn is what was drawn: who, in which shape and how far, how many people, and
// whether the walk stopped at its limit; a timeline's people left off for want of a
// date, and a path's relation, as the pages show them beside the chart.
type ChartDrawn struct {
	Out       string         `json:"out"`
	Format    draw.Format    `json:"format"`
	Shape     chart.Shape    `json:"shape"`
	Person    Relative       `json:"person"`
	To        *Relative      `json:"to,omitempty"`
	Distance  int            `json:"distance,omitempty"`
	People    int            `json:"people"`
	Truncated bool           `json:"truncated"`
	Limit     int            `json:"limit"`
	Undated   int            `json:"undated,omitempty"`
	Path      *chart.Related `json:"path,omitempty"`
	Read      []Supposition  `json:"read,omitempty"`
	Problems  []string       `json:"problems"`
}

// ExportChart draws a chart of the hidden view to a file, laid out by internal/chart
// as the pages lay it out, each person as the index shows them.
func ExportChart(root string, asked ChartAsked) (*ChartDrawn, error) {
	format, ok := draw.FormatOf(asked.Out)
	switch {
	case !slices.Contains(chart.Shapes, asked.Shape):
		return nil, fmt.Errorf("a chart's shape is one of %s, and not %q", shapeList(), asked.Shape)
	case asked.Shape == chart.Path && asked.To == "":
		return nil, fmt.Errorf("a path runs to somebody, and --to names them")
	case asked.Shape != chart.Path && asked.To != "":
		return nil, fmt.Errorf("only a path runs to somebody, and a chart of %s does not", asked.Shape)
	case asked.Shape != chart.Path && (asked.Distance < 1 || asked.Distance > chart.MaxDistance):
		return nil, fmt.Errorf("a chart reaches 1 to %d steps, and not %d", chart.MaxDistance, asked.Distance)
	case !ok:
		return nil, fmt.Errorf("a chart is drawn as %s, by the extension of the file, and %s has none of them",
			formatList(), asked.Out)
	}

	q, err := asking(root, asked.With)
	if err != nil {
		return nil, err
	}
	defer q.close()
	view := q.corpus.Viewed(model.Hide(q.corpus.Corpus, asked.Hiding, today()))
	shown := &querying{corpus: view, read: q.read, problems: q.problems}
	subject, err := hiddenPerson(q, shown, asked.Person, asked.Hiding)
	if err != nil {
		return nil, err
	}
	var to model.Slug
	if asked.To != "" {
		if to, err = hiddenPerson(q, shown, asked.To, asked.Hiding); err != nil {
			return nil, err
		}
	}

	index := indexOf(view.Corpus, today())
	var graph chart.Index
	if data, err := json.Marshal(index); err != nil {
		return nil, err
	} else if err := json.Unmarshal(data, &graph); err != nil {
		return nil, err
	}
	laid, _ := chart.Draw(chart.KinOf(graph), chart.Ask{
		Subject:  string(subject),
		Shape:    asked.Shape,
		Distance: asked.Distance,
		To:       string(to),
		Year:     time.Now().Year(),
	})

	people := map[string]IndexedPerson{}
	for _, p := range index.People {
		people[strings.TrimPrefix(p.Ref, "person/")] = p
	}
	cards := map[string]draw.Card{}
	for _, p := range laid.People {
		cards[p.Slug] = cardOf(view, p.Slug, people[p.Slug])
	}
	var out bytes.Buffer
	if err := draw.Write(&out, laid, cards, format); err != nil {
		return nil, err
	}
	if err := os.WriteFile(asked.Out, out.Bytes(), 0o644); err != nil {
		return nil, err
	}

	report := &ChartDrawn{
		Out:       asked.Out,
		Format:    format,
		Shape:     asked.Shape,
		Person:    shown.relative(subject),
		People:    len(laid.People),
		Truncated: laid.Truncated,
		Limit:     laid.Limit,
		Undated:   laid.Undated,
		Path:      laid.Path,
		Read:      q.read,
		Problems:  q.problems,
	}
	if asked.Shape != chart.Path {
		report.Distance = asked.Distance
	} else {
		other := shown.relative(to)
		report.To = &other
	}
	return report, q.warning
}

// hiddenPerson is the person a chart is asked of, who has to be somebody the view
// shows.
func hiddenPerson(whole, shown *querying, written string, hiding model.Hiding) (model.Slug, error) {
	slug, err := shown.person(written)
	if err == nil {
		return slug, nil
	}
	if _, missing := whole.person(written); missing == nil {
		return "", fmt.Errorf("%s is hidden from a view without %s", written, hiding)
	}
	return "", err
}

// cardOf is somebody as the pages' cards show them: the name the index gives, the
// years of their birth and death, and the portrait it chooses, read from beside the
// record keeping it.
func cardOf(corpus *store.Corpus, slug string, p IndexedPerson) draw.Card {
	card := draw.Card{Name: p.Name, Sex: p.Sex, Years: years(p)}
	if card.Name == "" {
		card.Name = slug
	}
	if person := corpus.People[model.Slug(slug)]; person != nil {
		card.Lang = string(person.Name.Lang)
	}
	card.Initials = initials(card.Name)
	if p.Portrait == nil {
		return card
	}
	from, err := model.ParseEntityRef(p.Portrait.Record)
	if err != nil {
		return card
	}
	at := model.Locate(from, p.Portrait.File)
	file, err := corpus.Attachment(at.Record.Slug, at.File)
	if err != nil {
		return card
	}
	if data, err := os.ReadFile(file); err == nil {
		card.Picture, card.Region = draw.Decode(data), p.Portrait.Region
	}
	return card
}

var leadingYear = regexp.MustCompile(`^-?\d+`)

// years is the years somebody lived, as a card beside their name gives them.
func years(p IndexedPerson) string {
	year := func(d *Dated) string {
		if d == nil {
			return ""
		}
		found := leadingYear.FindString(d.Sort)
		if found == "" {
			return ""
		}
		n, _ := strconv.Atoi(found)
		return strconv.Itoa(n)
	}
	born, died := year(p.Birth), year(p.Death)
	if born == "" && died == "" {
		return ""
	}
	return born + " – " + died
}

// initials are the first letters of the first and last words of a name that begin with
// a capital, or with a letter of a script that has none, past an article joined to the
// word, as in d'Annunzio, al-Halabi and الحلبي, drawn where there is no picture, as the
// pages draw them. A name in ideographs or Hangul gives its first character, the
// surname, and a name with no capital its first letter.
func initials(name string) string {
	if r, _ := utf8.DecodeRuneInString(strings.TrimSpace(name)); unicode.In(r, unicode.Han, unicode.Hangul) {
		return string(r)
	}
	var letters []rune
	for _, w := range strings.Fields(name) {
		if r, ok := initial(w); ok {
			letters = append(letters, r)
		}
	}
	if len(letters) > 1 {
		letters = []rune{letters[0], letters[len(letters)-1]}
	}
	if len(letters) == 0 {
		if i := strings.IndexFunc(name, unicode.IsLetter); i >= 0 {
			r, _ := utf8.DecodeRuneInString(name[i:])
			letters = []rune{unicode.ToUpper(r)}
		}
	}
	return string(letters)
}

var joinedArticle = regexp.MustCompile(`^\p{Ll}+['’-](.+)`)

func initial(word string) (rune, bool) {
	capital := func(w string) bool {
		r, _ := utf8.DecodeRuneInString(w)
		return unicode.In(r, unicode.Lu, unicode.Lt, unicode.Lo)
	}
	if m := joinedArticle.FindStringSubmatch(word); m != nil && capital(m[1]) {
		word = m[1]
	} else if rest, ok := strings.CutPrefix(word, "ال"); ok && utf8.RuneCountInString(rest) >= 2 {
		word = rest
	}
	r, _ := utf8.DecodeRuneInString(word)
	return r, capital(word)
}

func shapeList() string {
	names := make([]string, len(chart.Shapes))
	for i, s := range chart.Shapes {
		names[i] = string(s)
	}
	return strings.Join(names, ", ")
}

func formatList() string {
	names := make([]string, len(draw.Formats))
	for i, f := range draw.Formats {
		names[i] = string(f)
	}
	return strings.Join(names, ", ")
}
