package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/origins-project/origins/internal/api"
)

// The commands that follow a fact back to the argument behind it, which api.Blame and
// api.Why say more about.

// tracing is what both of them take.
type tracing struct {
	root   string
	asJSON bool
}

func tracingFlags(name string, args []string, stdout, stderr io.Writer) (*tracing, []string, int) {
	c := &tracing{}
	flags := newFlags(name, stderr)
	flags.StringVar(&c.root, "C", ".", "the `corpus` the entity is in")
	flags.BoolVar(&c.asJSON, "json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return nil, nil, code
	}
	rest := flags.Args()
	if code := flagsFirst(name, flags, rest, stderr); code != exitOK {
		return nil, nil, code
	}
	return c, rest, exitOK
}

// blame is when each fact about an entity appeared, and what it rests on.
func blame(args []string, stdout, stderr io.Writer) int {
	c, rest, code := tracingFlags("blame", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) == 0 || len(rest) > 2 {
		return misuse(stderr, "that is not the shape of it: origins blame person/rossi-giovanni [events]")
	}
	report, err := api.Blame(c.root, rest[0], strings.Join(rest[1:], ""))
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Provenance) {
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
		if len(report.Fields) == 0 {
			fmt.Fprintf(stdout, "%s writes nothing at %s\n", report.Entity, strings.Join(rest[1:], ""))
		}
		tabulate(stdout, report.Fields)
	}, func(report *api.Provenance) bool { return len(report.Problems) > 0 })
}

// tabulate writes one fact per line, in the order the file writes them, so that the
// history of a record reads down the page as the record itself does.
func tabulate(stdout io.Writer, facts []api.Attribution) {
	width := 0
	for _, one := range facts {
		width = max(width, len(one.Path))
	}
	for _, one := range facts {
		// the day is as much of the time as a reader scanning a record wants, and
		// the whole of it is in the JSON
		sha, when, rests := "-", "-", "not committed"
		if wrote := one.Commit; wrote != nil {
			sha, when, rests = short(wrote.Commit), wrote.When.Format(time.DateOnly), ""
			switch {
			case wrote.Foreign:
				rests = "not origins"
			case len(wrote.Sources) > 0:
				rests = "from " + strings.Join(wrote.Sources, ", ")
			}
		}
		row := fmt.Sprintf("%-10s  %-10s  %-*s  %s", sha, when, width, one.Path, rests)
		fmt.Fprintln(stdout, strings.TrimRight(row, " "))
	}
}

// why is the whole chain from a fact to the argument behind it.
func why(args []string, stdout, stderr io.Writer) int {
	c, rest, code := tracingFlags("why", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) != 2 {
		return misuse(stderr, "that is not the shape of it: origins why person/rossi-annunziata events[0].date")
	}
	report, err := api.Why(c.root, rest[0], rest[1])
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Reason) {
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
		if report.Field == nil {
			fmt.Fprintf(stdout, "%s writes nothing at %s\n", report.Entity, rest[1])
		} else {
			explain(stdout, report)
		}
	}, func(report *api.Reason) bool { return len(report.Problems) > 0 })
}

// explain writes the chain out: what the file says, then the commit that wrote it in
// the same terms a log gives, then the argument it rested on.
func explain(stdout io.Writer, report *api.Reason) {
	said, by := report.Field, report.Hypothesis
	fmt.Fprintf(stdout, "%s %s\n", report.Entity, said.Path)
	for _, line := range said.Lines {
		fmt.Fprintf(stdout, "  %s\n", strings.TrimSpace(line))
	}
	if said.Commit == nil {
		fmt.Fprintln(stdout, "\nnothing has been committed, so nobody has said why yet")
		return
	}
	fmt.Fprintln(stdout)
	recite(stdout, *said.Commit)
	if by == nil {
		return
	}

	const indent = "            "
	fmt.Fprintf(stdout, "\nhypothesis/%s  %s\n", by.Slug,
		strings.Join(nonEmpty(by.Status, by.Confidence, by.ProposedBy), ", "))
	for _, weighing := range []struct {
		what  string
		which []string
	}{
		{"resting on", by.Supports},
		{"against", by.Contradicts},
	} {
		if len(weighing.which) > 0 {
			fmt.Fprintf(stdout, "%s%s %s\n", indent, weighing.what, strings.Join(weighing.which, ", "))
		}
	}
	if by.Body != "" {
		fmt.Fprintf(stdout, "\n%s\n", by.Body)
	}
}

func nonEmpty(values ...string) []string {
	kept := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			kept = append(kept, value)
		}
	}
	return kept
}
