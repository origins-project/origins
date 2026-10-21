package main

import (
	"fmt"
	"io"

	"github.com/origins-project/origins/internal/api"
)

// initialise starts a corpus and records that it started, which api.Init says more
// about.
func initialise(args []string, stdout, stderr io.Writer) int {
	flags := newFlags("init", stderr)
	name := flags.String("name", "", "the project's `name`, in place of the directory's")
	asJSON := flags.Bool("json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	rest := flags.Args()
	if code := flagsFirst("init", flags, rest, stderr); code != exitOK {
		return code
	}
	if len(rest) > 1 {
		return misuse(stderr, "init starts one corpus at a time")
	}
	root := "."
	if len(rest) == 1 {
		root = rest[0]
	}
	report, err := api.Init(root, *name)
	return answer(report, err, *asJSON, stdout, stderr, func(report *api.Begun) {
		fmt.Fprintf(stdout, "%s  %s\n", short(report.Commit), report.Subject)
		if report.Started {
			fmt.Fprintln(stdout, "  recorded in a new git repository")
		} else {
			fmt.Fprintf(stdout, "  recorded in the repository at %s\n", report.History)
		}
		for _, path := range report.Files {
			fmt.Fprintf(stdout, "  %s\n", path)
		}
		if len(report.Waiting) > 0 {
			fmt.Fprintln(stdout, "\nthere was more here than the manifest, and origins has recorded none of it:")
			for _, path := range report.Waiting {
				fmt.Fprintf(stdout, "  %s\n", path)
			}
			fmt.Fprintln(stdout, "\nnow run: origins adopt")
		}
	}, nil)
}
