package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// The questions that follow a fact back to the argument behind it.
//
// These load the corpus, where log and whence do not. What they are asked about is a
// fact as the corpus has it now — which line of which file it is written on — so the
// answer depends on what the corpus says today and not only on what was once done to
// it. It is also the one question that needs both halves of the scheme at once: git
// says which commit last wrote a line, and origins says which fact that line is.

// fact is one field of an entity, the lines its file writes it on, and the commit it
// is the work of.
type fact struct {
	span    store.Span
	lines   []history.Blamed
	wrote   history.Entry
	settled bool
}

// traced is everything both questions work from.
type traced struct {
	corpus   *store.Corpus
	ref      model.EntityRef
	file     string
	facts    []fact
	problems []string
}

// trace reads an entity's file, works out which fact each line of it is, and asks git
// which commit last wrote each line.
func trace(root, entity string) (*traced, error) {
	corpus, problems := store.Load(root)
	t := &traced{corpus: corpus, problems: Flatten(problems)}

	spanned := store.Spans
	if entity == model.TheProject.String() {
		t.ref, t.file, spanned = model.TheProject, store.ProjectFile, store.ManifestSpans
	} else {
		ref, err := Entity(corpus, entity)
		if err != nil {
			return nil, err
		}
		t.ref = ref
		t.file, _ = corpus.Path(ref.Slug)
	}

	content, err := os.ReadFile(filepath.Join(corpus.Root, filepath.FromSlash(t.file)))
	if err != nil {
		return nil, err
	}
	spans, err := spanned(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", t.file, err)
	}

	repo, err := owned(corpus.Root)
	if err != nil {
		return nil, err
	}
	lines, err := repo.Blame(t.file)
	if len(lines) == 0 && err != nil {
		// git refusing outright leaves nothing to report; a trailer nobody can read
		// leaves the lines, and the lines are the answer
		return nil, err
	}
	t.problems = append(t.problems, Flatten(err)...)
	t.facts = gather(spans, lines)
	return t, nil
}

// gather puts each line of a file beside the fact it is written on, so that a value
// folded over four lines is one fact and the lines the fences take are none.
func gather(spans []store.Span, lines []history.Blamed) []fact {
	var facts []fact
	where := map[string]int{}
	for _, line := range lines {
		span, ok := store.At(spans, line.Line)
		if !ok {
			continue
		}
		at, seen := where[span.Path]
		if !seen {
			at = len(facts)
			where[span.Path], facts = at, append(facts, fact{span: span})
		}
		facts[at].lines = append(facts[at].lines, line)
	}
	for i, one := range facts {
		facts[i].wrote, facts[i].settled = latest(one.lines)
	}
	return facts
}

// latest is the commit a fact is the work of: the newest behind its lines, since a
// value written over several of them stands as of its last change. A line no commit
// accounts for is newer than any commit, and the fact is then nobody's work yet.
func latest(lines []history.Blamed) (history.Entry, bool) {
	var (
		newest  history.Entry
		settled bool
	)
	for _, line := range lines {
		if line.Uncommitted() {
			return history.Entry{}, false
		}
		if !settled || line.When.After(newest.When) {
			newest, settled = line.Entry, true
		}
	}
	return newest, settled
}

// Attribution is one fact. A field nobody has committed has no commit, which is not
// the same as one whose commit says nothing.
type Attribution struct {
	Path   string   `json:"path"`
	From   int      `json:"from"`
	To     int      `json:"to"`
	Lines  []string `json:"lines"`
	Commit *Entry   `json:"commit"`
}

func attributed(one fact) Attribution {
	said := Attribution{Path: one.span.Path, From: one.span.From, To: one.span.To}
	for _, line := range one.lines {
		said.Lines = append(said.Lines, line.Text)
	}
	if one.settled {
		wrote := described(one.wrote)
		said.Commit = &wrote
	}
	return said
}

// Provenance is when each fact about an entity appeared, and what it rests on.
type Provenance struct {
	Entity   string        `json:"entity"`
	File     string        `json:"file"`
	Fields   []Attribution `json:"fields"`
	Problems []string      `json:"problems"`
}

// Blame is when each fact about an entity appeared, and what it rests on, or only the
// facts at and under field where it is given. It is git blame with the lines named:
// canonical serialisation puts one fact on one line, and naming that fact is what
// origins can add.
func Blame(root, entity, field string) (*Provenance, error) {
	t, err := trace(root, entity)
	if err != nil {
		return nil, err
	}
	report := &Provenance{
		Entity: t.ref.String(), File: t.file,
		Fields: []Attribution{}, Problems: t.problems,
	}
	for _, one := range t.facts {
		if field == "" || under(one.span.Path, field) {
			report.Fields = append(report.Fields, attributed(one))
		}
	}
	return report, nil
}

// under reports whether a path is the field asked about or something inside it, so
// that asking after events answers with events[0].date as well.
func under(path, field string) bool {
	return path == field ||
		strings.HasPrefix(path, field+".") || strings.HasPrefix(path, field+"[")
}

// Argument is the hypothesis a commit says decided a fact: how it stood, what it was
// weighed against, and the prose that says what would overturn it.
type Argument struct {
	Slug        string   `json:"slug"`
	Status      string   `json:"status"`
	Confidence  string   `json:"confidence"`
	ProposedBy  string   `json:"proposed-by"`
	Supports    []string `json:"supports"`
	Contradicts []string `json:"contradicts"`
	Body        string   `json:"body"`
}

// Reason is the chain from one fact to the argument behind it.
type Reason struct {
	Entity     string       `json:"entity"`
	File       string       `json:"file"`
	Field      *Attribution `json:"field"`
	Hypothesis *Argument    `json:"hypothesis"`
	Problems   []string     `json:"problems"`
}

// Why is the whole chain from a fact to the argument behind it: what the corpus says,
// the commit that put it there, and the hypothesis that decided it, which is where the
// evidence and the rejected alternative are written down.
func Why(root, entity, path string) (*Reason, error) {
	t, err := trace(root, entity)
	if err != nil {
		return nil, err
	}
	report := &Reason{Entity: t.ref.String(), File: t.file, Problems: t.problems}
	for _, one := range t.facts {
		if one.span.Path != path {
			continue
		}
		said := attributed(one)
		report.Field = &said
		if one.settled && one.wrote.Hypothesis != "" {
			report.Hypothesis = weighed(t.corpus, one.wrote.Hypothesis)
		}
		break
	}
	return report, nil
}

// weighed is the hypothesis as the corpus holds it. One accepted and then deleted is
// gone, and the commit naming it is all that is left — which the commit's own line
// already says, so there is nothing further to report.
func weighed(corpus *store.Corpus, slug model.Slug) *Argument {
	held := corpus.Hypotheses[slug]
	if held == nil {
		return nil
	}
	return &Argument{
		Slug:        string(slug),
		Status:      string(held.Status),
		Confidence:  string(held.Confidence),
		ProposedBy:  held.ProposedBy,
		Supports:    written(held.Supports),
		Contradicts: written(held.Contradicts),
		Body:        held.Body,
	}
}
