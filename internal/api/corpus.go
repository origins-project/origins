package api

import (
	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Summary is what a corpus holds, as load reports it.
type Summary struct {
	Project  ProjectSummary `json:"project"`
	Counts   Counts         `json:"counts"`
	Problems []string       `json:"problems"`
}

// ProjectSummary is what the manifest says of the corpus as a whole.
type ProjectSummary struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	MainPerson    string `json:"main-person"`
	FormatVersion int    `json:"format-version"`
}

// Counts is how many records of each kind a corpus holds.
type Counts struct {
	People       int `json:"people"`
	Families     int `json:"families"`
	Places       int `json:"places"`
	Sources      int `json:"sources"`
	Repositories int `json:"repositories"`
	Notes        int `json:"notes"`
	Hypotheses   int `json:"hypotheses"`
}

func counted(corpus *model.Corpus) Counts {
	return Counts{
		People:       len(corpus.People),
		Families:     len(corpus.Families),
		Places:       len(corpus.Places),
		Sources:      len(corpus.Sources),
		Repositories: len(corpus.Repositories),
		Notes:        len(corpus.Notes),
		Hypotheses:   len(corpus.Hypotheses),
	}
}

// Load reads a corpus and says what is in it.
func Load(root string) (*Summary, error) {
	corpus, problems, warning := opened(root)
	return &Summary{
		Project: ProjectSummary{
			Name:          corpus.Project.Name,
			Description:   corpus.Project.Description,
			MainPerson:    string(corpus.Project.MainPerson),
			FormatVersion: corpus.Project.FormatVersion,
		},
		Counts:   counted(corpus.Corpus),
		Problems: Flatten(problems),
	}, warning
}

// Validation is what check finds.
type Validation struct {
	Problems []string        `json:"problems"`
	Read     []Supposition   `json:"read,omitempty"`
	Findings []check.Finding `json:"findings"`
	Errors   int             `json:"errors"`
	Warnings int             `json:"warnings"`
}

// Check looks for contradictions in a corpus, with the hypotheses named in with in it,
// leaving out warnings where errorsOnly says so.
func Check(root string, with []model.Slug, errorsOnly bool) (*Validation, error) {
	corpus, problems, warning := opened(root)
	trial, err := supposing(corpus, with)
	if err != nil {
		return nil, err
	}
	var read []Supposition
	if trial != nil {
		defer func() { _ = trial.Close() }()
		corpus, read = trial.Corpus, suppositions(trial.Read)
	}

	findings := check.Run(corpus.Corpus)
	errorCount := check.Errors(findings)
	report := &Validation{
		Problems: Flatten(problems),
		Read:     read,
		Findings: findings,
		Errors:   errorCount,
		Warnings: len(findings) - errorCount,
	}
	if errorsOnly {
		kept := make([]check.Finding, 0, errorCount)
		for _, f := range findings {
			if f.Severity == check.Error {
				kept = append(kept, f)
			}
		}
		report.Findings = kept
	}
	if report.Findings == nil {
		report.Findings = []check.Finding{}
	}
	return report, warning
}

// Findings is everything check and todo find, together: what is wrong and what is
// still to do, which a page shows beside whatever each is about.
type Findings struct {
	Findings []check.Finding `json:"findings"`
	Todo     []check.Finding `json:"todo"`
	Errors   int             `json:"errors"`
	Warnings int             `json:"warnings"`
	Problems []string        `json:"problems"`
}

// FindingsOf is what check and todo find in a corpus already read, with the problems
// it was read with, deciding who is living as of today.
func FindingsOf(corpus *model.Corpus, problems error, today model.Date) *Findings {
	findings := check.Run(corpus)
	errorCount := check.Errors(findings)
	report := &Findings{
		Findings: findings,
		Todo:     check.Research(corpus, today),
		Errors:   errorCount,
		Warnings: len(findings) - errorCount,
		Problems: Flatten(problems),
	}
	if report.Findings == nil {
		report.Findings = []check.Finding{}
	}
	if report.Todo == nil {
		report.Todo = []check.Finding{}
	}
	return report
}

// Formatted is what fmt rewrote, or would.
type Formatted struct {
	Changed  []string `json:"changed"`
	Problems []string `json:"problems"`
}

// Fmt re-emits a corpus, or with list only says which files it would rewrite.
//
// Origins is the sole writer of the format and guarantees canonical output, so this is
// what puts a hand-edited file back in step — and what proves, by rewriting nothing,
// that it already is.
func Fmt(root string, list bool) (*Formatted, error) {
	corpus, problems, warning := opened(root)
	changed, failures := store.Format(corpus, !list)
	report := &Formatted{
		Changed:  changed,
		Problems: append(Flatten(problems), Flatten(failures)...),
	}
	if report.Changed == nil {
		report.Changed = []string{}
	}
	return report, warning
}
