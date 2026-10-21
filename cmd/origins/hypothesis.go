package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
)

// One command with two answers, because they are one act, which api.Decide says more
// about.

// hypothesis settles one, one way or the other, or proposes one or says where each
// stands.
func hypothesis(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && asksHelp(args[0]) {
		printHelp(stdout, helpOf("hypothesis"), nil)
		return exitHelped
	}
	if len(args) == 0 {
		return misuse(stderr, "that is not the shape of it: origins hypothesis "+
			"propose|accept|reject|status annunziata-born-1844")
	}
	answer, rest := args[0], args[1:]
	switch answer {
	case "status":
		return hypothesisStatus(rest, stdout, stderr)
	case "propose":
		return propose(rest, stdout, stderr)
	case "accept", "reject":
	default:
		return misuse(stderr, "origins can propose a hypothesis, accept or reject one, or say where "+
			"each stands, and %q is none of those", answer)
	}
	c, more, code := changeFlags("hypothesis "+answer, rest, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(more) != 1 {
		return misuse(stderr, "that is not the shape of it: origins hypothesis %s "+
			"annunziata-born-1844", answer)
	}
	slug, code := bare(more[0], model.KindHypothesis, stderr)
	if code != exitOK {
		return code
	}

	status := model.Accepted
	if answer == "reject" {
		status = model.Rejected
	}
	report, err := api.Decide(c.asking(), slug, status)
	return settled(c, report, err, stdout, stderr)
}

// settled says what came of settling a hypothesis.
func settled(c *change, report *api.Decided, err error, stdout, stderr io.Writer) int {
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Decided) {
		if report.Empty {
			fmt.Fprintln(stdout, "the corpus already says that, so nothing was written")
			return
		}
		lead := "  "
		if c.dryRun {
			fmt.Fprintln(stdout, report.Subject)
			lead = "  would record "
		} else {
			fmt.Fprintf(stdout, "%s  %s\n", short(report.Commit), report.Subject)
		}
		if report.Base != "" {
			fmt.Fprintf(stdout, "  carried out at %s, where it was written\n", short(report.Base))
		}
		for _, path := range report.Files {
			fmt.Fprintf(stdout, "%s%s\n", lead, path)
		}
	}, nil)
}

// hypothesisStatus lists every hypothesis with where it stands, which api.Status says
// more about. It exits 1 where the corpus has left a conclusion behind, as check does
// for a contradiction.
func hypothesisStatus(args []string, stdout, stderr io.Writer) int {
	var (
		root   string
		asJSON bool
	)
	flags := newFlags("hypothesis status", stderr)
	flags.StringVar(&root, "C", ".", "the `corpus` whose hypotheses to look at")
	flags.BoolVar(&asJSON, "json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	if flags.NArg() > 1 {
		return misuse(stderr, "that is not the shape of it: origins hypothesis status [annunziata-born-1844]")
	}
	var named model.Slug
	if flags.NArg() == 1 {
		slug, code := bare(flags.Arg(0), model.KindHypothesis, stderr)
		if code != exitOK {
			return code
		}
		named = slug
	}

	holdings, err := api.Status(root, named)
	var report *[]api.Holding
	if holdings != nil {
		report = &holdings
	}
	return answer(report, err, asJSON, stdout, stderr, func(report *[]api.Holding) {
		width := 0
		for _, one := range *report {
			width = max(width, len(one.Hypothesis))
		}
		for _, one := range *report {
			fmt.Fprintf(stdout, "%-*s  %s\n", width, one.Hypothesis, said(one))
		}
	}, func(report *[]api.Holding) bool {
		for _, one := range *report {
			if one.Holds != nil && !*one.Holds {
				return true
			}
		}
		return false
	})
}

func said(one api.Holding) string {
	switch {
	case one.Why != "":
		return one.Status + ", and cannot be read where it was written, because " + one.Why
	case one.Holds == nil:
		return one.Status
	case *one.Holds:
		return one.Status + ", and the corpus still holds it"
	}
	return one.Status + ", and the corpus no longer holds it: " + strings.Join(one.Reasons, "; ")
}
