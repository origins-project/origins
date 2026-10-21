package api

import (
	"fmt"
	"time"

	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// The researcher's questions: who somebody descends from, who descends from them, how
// two people are related, and what is still to be found out.
//
// Each reads the corpus and never the repository. They are questions about what the
// corpus says, which is as true of a corpus in no repository as of one with a long
// history, and they take with because what the corpus would say with a hypothesis in
// it is the same question asked of a different corpus.

// querying is what every one of them works from. The caller closes it, which takes
// away the copy a supposition was built in.
type querying struct {
	corpus   *store.Corpus
	read     []Supposition
	problems []string
	warning  error
	close    func()
}

func asking(root string, with []model.Slug) (*querying, error) {
	corpus, problems, warning := opened(root)
	q := &querying{corpus: corpus, problems: Flatten(problems), warning: warning, close: func() {}}
	trial, err := supposing(corpus, with)
	if err != nil {
		return nil, err
	}
	if trial != nil {
		q.corpus, q.read, q.close = trial.Corpus, suppositions(trial.Read), func() { _ = trial.Close() }
	}
	return q, nil
}

// person is a name that has to be somebody.
func (q *querying) person(written string) (model.Slug, error) {
	ref, err := Entity(q.corpus, written)
	if err != nil {
		return "", err
	}
	if ref.Kind != model.KindPerson {
		return "", fmt.Errorf("%s is a %s, and only a person has relatives", ref.Slug, ref.Kind)
	}
	return ref.Slug, nil
}

// Relative is somebody an answer names.
type Relative struct {
	Slug string `json:"slug"`
	Name string `json:"name"`

	// Sex is what the records say of them, and nothing for a parent named and never
	// recorded, whom a path can still run through.
	Sex model.Sex `json:"-"`
}

func (q *querying) relative(slug model.Slug) Relative {
	one := Relative{Slug: string(slug), Name: string(slug)}
	if person := q.corpus.People[slug]; person != nil {
		if !person.Name.IsZero() {
			one.Name = person.Name.String()
		}
		one.Sex = person.Sex.Value
	}
	return one
}

// Generation is one generation of an ancestors or descendants answer, by what it is
// called.
type Generation struct {
	Called string     `json:"called"`
	People []Relative `json:"people"`
}

// Lineage is a person's ancestors or descendants, a generation at a time.
type Lineage struct {
	Person      Relative      `json:"person"`
	Generations []Generation  `json:"generations"`
	Read        []Supposition `json:"read,omitempty"`
	Problems    []string      `json:"problems"`
}

// Ancestors lists who a person descends from, a generation at a time, the nearest
// first and no further than depth where it is more than nought.
func Ancestors(root string, with []model.Slug, person string, depth int) (*Lineage, error) {
	return descent(root, with, person, depth, true)
}

// Descendants lists who descends from a person, as Ancestors does.
func Descendants(root string, with []model.Slug, person string, depth int) (*Lineage, error) {
	return descent(root, with, person, depth, false)
}

// descent lists a person's line a generation at a time, the nearest first. Somebody
// reached two ways, as a cousin marriage makes an ancestor, is listed once, in the
// nearer generation.
func descent(root string, with []model.Slug, person string, depth int, upward bool) (*Lineage, error) {
	if depth < 0 {
		return nil, fmt.Errorf("--depth counts generations, and %d is not a number of them", depth)
	}
	q, err := asking(root, with)
	if err != nil {
		return nil, err
	}
	defer q.close()
	subject, err := q.person(person)
	if err != nil {
		return nil, err
	}

	found := q.corpus.Descendants(subject)
	if upward {
		found = q.corpus.Ancestors(subject)
	}
	if depth > 0 && len(found) > depth {
		found = found[:depth]
	}
	report := &Lineage{
		Person:      q.relative(subject),
		Generations: []Generation{},
		Read:        q.read,
		Problems:    q.problems,
	}
	for i, slugs := range found {
		one := Generation{Called: model.Generation(i+1, upward)}
		for _, slug := range slugs {
			one.People = append(one.People, q.relative(slug))
		}
		report.Generations = append(report.Generations, one)
	}
	return report, q.warning
}

// Step is one person on the way between two relatives, and what they are to the one
// before them.
type Step struct {
	Relative
	Is string `json:"is,omitempty"`
}

// Kin is what one person is to another, and the people it runs through.
type Kin struct {
	From           Relative      `json:"from"`
	To             Relative      `json:"to"`
	Related        bool          `json:"related"`
	Relation       string        `json:"relation,omitempty"`
	CommonAncestor string        `json:"common-ancestor,omitempty"`
	Up             int           `json:"up"`
	Down           int           `json:"down"`
	Path           []Step        `json:"path"`
	Read           []Supposition `json:"read,omitempty"`
	Problems       []string      `json:"problems"`
}

// Kinship names what the first person is to the second, and the people it runs
// through: up from the first to the nearest ancestor the two share, and down from there
// to the second. Two people with no ancestor in common here is an answer and not a
// failure, since a tree assembled from records mostly consists of them.
func Kinship(root string, with []model.Slug, first, second string) (*Kin, error) {
	q, err := asking(root, with)
	if err != nil {
		return nil, err
	}
	defer q.close()
	return q.kinship(first, second)
}

// KinshipOf is Kinship asked of a corpus already read.
func KinshipOf(corpus *store.Corpus, first, second string) (*Kin, error) {
	q := &querying{corpus: corpus, problems: []string{}, close: func() {}}
	return q.kinship(first, second)
}

func (q *querying) kinship(first, second string) (*Kin, error) {
	from, err := q.person(first)
	if err != nil {
		return nil, err
	}
	to, err := q.person(second)
	if err != nil {
		return nil, err
	}

	report := &Kin{
		From: q.relative(from), To: q.relative(to), Path: []Step{},
		Read: q.read, Problems: q.problems,
	}
	found, related := q.corpus.Kinship(from, to)
	if related {
		report.Related = true
		report.Relation = found.Name(report.From.Sex)
		report.CommonAncestor = string(found.CommonAncestor)
		report.Up, report.Down = found.Up, found.Down
		for i, slug := range found.Path {
			one := Step{Relative: q.relative(slug)}
			if i > 0 {
				one.Is = q.link(found.Path[i-1], slug, i <= found.Up)
			}
			report.Path = append(report.Path, one)
		}
	}
	return report, q.warning
}

// link is what one person on a path is to the one before: their parent going up, their
// child coming down, and the kind of parent where it is not a birth.
func (q *querying) link(before, after model.Slug, upward bool) string {
	child, parent := after, before
	distance := model.Kinship{Up: 1}
	if upward {
		child, parent = before, after
		distance = model.Kinship{Down: 1}
	}
	word := distance.Name(q.relative(after).Sex)
	for _, ref := range q.corpus.ParentsOf(child) {
		if ref.Slug == parent && ref.Relation.Kin() != model.Birth {
			return string(ref.Relation) + " " + word
		}
	}
	return word
}

// ToDo is the research still to do.
type ToDo struct {
	Todo     []check.Finding `json:"todo"`
	Read     []Supposition   `json:"read,omitempty"`
	Problems []string        `json:"problems"`
}

// Todo lists the research still to do. None of it is wrong: it is the answer to the
// question, and a corpus being researched has one.
func Todo(root string, with []model.Slug) (*ToDo, error) {
	q, err := asking(root, with)
	if err != nil {
		return nil, err
	}
	defer q.close()
	report := &ToDo{
		Todo:     check.Research(q.corpus.Corpus, today()),
		Read:     q.read,
		Problems: q.problems,
	}
	if report.Todo == nil {
		report.Todo = []check.Finding{}
	}
	return report, q.warning
}

// today is the day a question is asked on, which is what decides how long ago a birth
// was.
func today() model.Date { return model.DateOf(time.Now()) }
