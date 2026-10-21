package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/origins-project/origins/internal/api"
)

// The commands that write to a corpus's history rather than reading it: one takes a
// change back out of it, the other gives a state of the corpus a name.

// undo takes back the last change origins made, which ops.Undo says more about.
// Nothing is validated afterwards, and origins check is one word away.
func undo(args []string, stdout, stderr io.Writer) int {
	var (
		root, message  string
		dryRun, asJSON bool
	)
	flags := newFlags("undo", stderr)
	flags.StringVar(&root, "C", ".", "the `corpus` to take a change back in")
	flags.StringVar(&message, "m", "", "the `subject` line, in place of the one origins writes")
	flags.BoolVar(&dryRun, "dry-run", false, "say what would be taken back without taking it")
	flags.BoolVar(&asJSON, "json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	if flags.NArg() != 0 {
		return misuse(stderr, "undo takes back the last change and nothing else, so it takes no arguments")
	}
	report, err := api.Undo(api.Asking{Root: root, Command: "undo", Message: message, DryRun: dryRun})
	return undone(report, err, dryRun, asJSON, stdout, stderr)
}

// undone says what came of taking a change back.
func undone(report *api.Reversal, err error, dryRun, asJSON bool, stdout, stderr io.Writer) int {
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Reversal) {
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
		recite(stdout, report.Undid)
		if dryRun {
			fmt.Fprintln(stdout, "would be taken back")
		} else {
			fmt.Fprintf(stdout, "taken back as %s\n", short(report.Commit))
		}
		for _, file := range report.Files {
			fmt.Fprintf(stdout, "  %s\n", file)
		}
	}, nil)
}

// snapshot names a state of the corpus, or says which states have been named, which
// api.Snapshot says more about.
func snapshot(args []string, stdout, stderr io.Writer) int {
	var (
		root, message string
		asJSON        bool
	)
	flags := newFlags("snapshot", stderr)
	flags.StringVar(&root, "C", ".", "the `corpus` to name a state of")
	flags.StringVar(&message, "m", "", "a `message` saying what the state is named for")
	flags.BoolVar(&asJSON, "json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	rest := flags.Args()
	if code := flagsFirst("snapshot", flags, rest, stderr); code != exitOK {
		return code
	}
	if len(rest) > 1 {
		return misuse(stderr, "that is not the shape of it: origins snapshot sent-to-zia-maria")
	}
	report, err := api.Snapshot(root, strings.Join(rest, ""), message)
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Snapshots) {
		catalogue(stdout, report.Snapshots)
	}, nil)
}

// catalogue writes the snapshots out a row each, in the order a log's entries start:
// the commit, the day, and then what a reader is actually looking for.
func catalogue(stdout io.Writer, snapshots []api.Named) {
	width := 0
	for _, taken := range snapshots {
		width = max(width, len(taken.Name))
	}
	for _, taken := range snapshots {
		row := fmt.Sprintf("%s  %s  %-*s  %s",
			short(taken.Commit), taken.When.Format(time.DateOnly), width, taken.Name, taken.Message)
		fmt.Fprintln(stdout, strings.TrimRight(row, " "))
	}
}
