package check

import (
	"fmt"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Severity separates what is wrong from what is worth a look. Only an error means the
// corpus contradicts itself; a warning is research still to do, and a corpus full of
// them can be perfectly correct.
type Severity string

// The two severities a finding can have.
const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

// Finding is one thing the validator noticed. Rule is a stable identifier so that a
// finding can be filtered, counted or suppressed by something other than its wording.
type Finding struct {
	Rule     string   `json:"rule"`
	Severity Severity `json:"severity"`
	Entity   string   `json:"entity"`
	Path     string   `json:"path,omitempty"`
	Message  string   `json:"message"`
}

// rule is one pass over the corpus. Keeping them separate and listing them here means
// the set of checks is readable in one place.
type rule func(*model.Corpus, *report)

var rules = []rule{
	danglingReferences,
	sameRecords,
	proseLinks,
	familyShape,
	familyChildren,
	birthParents,
	reservedSlugs,
	missingImages,
	impossibleRegions,
	unplacedCoordinates,
	untranscribedScans,
	attachments,
	surnameSlugs,
	surnameLists,
	unsourcedEvents,
	circularAncestry,
	duplicateCandidates,
	personDates,
	parentDates,
	familyDates,
	ruledOut,
	hypothesisEvidence,
}

// Run applies every rule and returns what they found, in a stable order. Nothing here
// touches the disk: the corpus is already in memory, so a validator run against a
// hypothetical corpus costs no more than one against the real one.
func Run(c *model.Corpus) []Finding {
	var r report
	for _, apply := range rules {
		apply(c, &r)
	}
	return r.sorted()
}

// Errors counts the findings that mean the corpus contradicts itself, which is what
// decides the exit status.
func Errors(findings []Finding) int {
	n := 0
	for _, f := range findings {
		if f.Severity == Error {
			n++
		}
	}
	return n
}

func cmp(pairs ...string) int {
	for i := 0; i+1 < len(pairs); i += 2 {
		if c := strings.Compare(pairs[i], pairs[i+1]); c != 0 {
			return c
		}
	}
	return 0
}

type report struct {
	findings []Finding
}

func (r *report) note(severity Severity, rule, entity, path, format string, args ...any) {
	r.findings = append(r.findings, Finding{
		Rule:     rule,
		Severity: severity,
		Entity:   entity,
		Path:     path,
		Message:  fmt.Sprintf(format, args...),
	})
}

func (r *report) fail(rule, entity, path, format string, args ...any) {
	r.note(Error, rule, entity, path, format, args...)
}

func (r *report) warn(rule, entity, path, format string, args ...any) {
	r.note(Warning, rule, entity, path, format, args...)
}

func (r *report) sorted() []Finding {
	slices.SortFunc(r.findings, func(a, b Finding) int {
		return cmp(a.Entity, b.Entity, a.Rule, b.Rule, a.Path, b.Path, a.Message, b.Message)
	})
	return r.findings
}

func ref(kind model.Kind, s model.Slug) string {
	return model.EntityRef{Kind: kind, Slug: s}.String()
}

func at(field string, index int) string {
	return fmt.Sprintf("%s[%d]", field, index)
}
