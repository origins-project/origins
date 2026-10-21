package main

import (
	"fmt"
	"io"

	"github.com/origins-project/origins/internal/api"
)

// versioned prints what origins is and the git it would run, which a report of a fault
// starts from.
func versioned(args []string, stdout, stderr io.Writer) int {
	flags := newFlags("version", stderr)
	asJSON := flags.Bool("json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	if flags.NArg() > 0 {
		return misuse(stderr, "version takes no arguments")
	}
	return answer(api.Describe(), nil, *asJSON, stdout, stderr, func(about *api.About) {
		fmt.Fprintf(stdout, "origins %s\n", about.Version)
		switch {
		case about.Commit == "":
		case about.Modified:
			fmt.Fprintf(stdout, "commit  %s, with changes no commit holds\n", about.Commit)
		default:
			fmt.Fprintf(stdout, "commit  %s\n", about.Commit)
		}
		fmt.Fprintf(stdout, "go      %s\n", about.Go)
		if about.Git == nil {
			fmt.Fprintf(stdout, "git     %s\n", about.GitProblem)
		} else {
			fmt.Fprintf(stdout, "git     %s, at %s\n", about.Git.Version, about.Git.Path)
		}
	}, nil)
}
