package main

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/origins-project/origins/internal/api"
)

// The commands that read a corpus's history, which api.Log and api.Whence say more
// about.

// recall is what every command that only reads history takes.
type recall struct {
	root   string
	limit  int
	asJSON bool
}

func recallFlags(name string, args []string, stdout, stderr io.Writer) (*recall, []string, int) {
	c := &recall{}
	flags := newFlags(name, stderr)
	flags.StringVar(&c.root, "C", ".", "the `corpus` to read the history of")
	flags.IntVar(&c.limit, "n", 0, "the newest `n` entries")
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

// told writes out what the history said.
func (c *recall) told(report *api.Journal, err error, stdout, stderr io.Writer) int {
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Journal) {
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
		for _, read := range report.Entries {
			recite(stdout, read)
		}
	}, func(report *api.Journal) bool { return len(report.Problems) > 0 })
}

// some is as many of the entities a commit names as belong on one line.
//
// A commit is usually about one or two, and the ones about fifty are the ones that
// had to be: adopting a corpus somebody had been keeping by hand, or an import. All
// of them are in the trailers, which is where the searching is done, and all of them
// are in the JSON; what a reader scanning a log wants to see is that there are fifty.
func some(entities []string) []string {
	const most = 4
	if len(entities) <= most {
		return entities
	}
	return append(slices.Clone(entities[:most]), fmt.Sprintf("and %d more", len(entities)-most))
}

// recite writes one commit out: the line a reader scans, then what the trailers say.
// Nothing is repeated between the two, so a history reads down the left edge and the
// detail is there for the entry that turns out to matter.
func recite(stdout io.Writer, read api.Entry) {
	const indent = "            " // a short sha and the two spaces after it

	fmt.Fprintf(stdout, "%s  %s  %s\n", short(read.Commit), read.When.Format(time.DateOnly), read.Subject)

	did := read.Op
	if read.Foreign {
		did = "not origins"
	}
	fmt.Fprintf(stdout, "%s%s  (%s)\n",
		indent, strings.Join(append([]string{did}, some(read.Subjects)...), "  "), read.Author)

	var clauses []string
	if len(read.Events) > 0 {
		clauses = append(clauses, "touching "+strings.Join(read.Events, ", "))
	}
	if len(read.Sources) > 0 {
		clauses = append(clauses, "from "+strings.Join(read.Sources, ", "))
	}
	if read.Hypothesis != "" {
		clauses = append(clauses, "by "+read.Hypothesis)
	}
	if read.ProposedBy != "" {
		clauses = append(clauses, "proposed by "+read.ProposedBy)
	}
	if read.Corrects != "" {
		clauses = append(clauses, "correcting "+short(read.Corrects))
	}
	if len(clauses) > 0 {
		fmt.Fprintf(stdout, "%s%s\n", indent, strings.Join(clauses, ", "))
	}
}

// research is the history of the corpus, or of one entity in it.
func research(args []string, stdout, stderr io.Writer) int {
	c, rest, code := recallFlags("log", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) > 1 {
		return misuse(stderr, "log reads the history of one entity at a time")
	}
	report, err := api.Log(c.root, strings.Join(rest, ""), c.limit)
	return c.told(report, err, stdout, stderr)
}

// whence is every conclusion ever drawn from a source.
func whence(args []string, stdout, stderr io.Writer) int {
	c, rest, code := recallFlags("whence", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) != 1 {
		return misuse(stderr, "that is not the shape of it: origins whence firenze-civil-deaths-1901")
	}
	report, err := api.Whence(c.root, rest[0], c.limit)
	return c.told(report, err, stdout, stderr)
}
