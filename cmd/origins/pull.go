package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/history"
)

// pull takes in what another clone recorded and pushed, which api.Pull says more about.
func pull(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	c, rest, code := sharingFlags("pull", "the `corpus` to pull into", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) > 1 {
		return misuse(stderr, "that is not the shape of it: origins pull, or origins pull origin")
	}
	report, err := api.Pull(c.root, strings.Join(rest, ""), history.Terminal{In: stdin, Err: stderr})
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Taken) {
		switch report.Took {
		case "nothing":
			fmt.Fprintf(stdout, "%s holds nothing this clone does not\n", report.Upstream)
		case "fast-forward":
			fmt.Fprintf(stdout, "took in %s from %s, at %s\n",
				tally(report.Commits, "commit", "commits"), report.Upstream, short(report.Commit))
		default:
			fmt.Fprintf(stdout, "%s  Pull %s, joining %s\n",
				short(report.Commit), report.Upstream, tally(report.Commits, "commit", "commits"))
			for _, subject := range some(report.Subjects) {
				fmt.Fprintf(stdout, "  %s\n", subject)
			}
		}
	}, nil)
}
