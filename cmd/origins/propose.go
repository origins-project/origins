package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
)

// propose writes a hypothesis out of the corpus as it has been edited, which
// api.Propose says more about, or records one stated whole in a file, which
// api.ProposeStated does.
func propose(args []string, stdout, stderr io.Writer) int {
	var proposedBy, from string
	c, more, code := changeFlags("hypothesis propose", args, stdout, stderr, func(flags *flag.FlagSet) {
		flags.StringVar(&proposedBy, "proposed-by", "", "the `name` of whoever proposed it, where it was not the researcher")
		flags.StringVar(&from, "from", "", "a hypothesis's `file` stating it whole, in place of the edited corpus")
	})
	if code != exitOK {
		return code
	}
	if len(more) != 1 {
		return misuse(stderr, "that is not the shape of it: origins hypothesis propose annunziata-born-1844")
	}
	slug, code := bare(more[0], model.KindHypothesis, stderr)
	if code != exitOK {
		return code
	}
	var report *api.Proposal
	var err error
	if from == "" {
		report, err = api.Propose(c.asking(), slug, proposedBy)
	} else {
		document, read := os.ReadFile(from)
		if read != nil {
			return misuse(stderr, "%v", read)
		}
		report, err = api.ProposeStated(c.asking(), slug, document, proposedBy)
	}
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Proposal) {
		lead := "  "
		if c.dryRun {
			fmt.Fprintln(stdout, report.Subject)
			lead = "  would state "
		} else {
			fmt.Fprintf(stdout, "%s  %s\n", short(report.Commit), report.Subject)
		}
		fmt.Fprintf(stdout, "  pinned to %s\n", short(report.Base))
		if report.ProposedBy != "" {
			fmt.Fprintf(stdout, "  proposed by %s\n", report.ProposedBy)
		}
		for _, edit := range report.Edits {
			fmt.Fprintf(stdout, "%s%s\n", lead, edit)
		}
		if !c.dryRun && len(report.Restored) > 0 {
			fmt.Fprintf(stdout, "  the edited files are back as %s had them\n", short(report.Base))
		}
	}, nil)
}
