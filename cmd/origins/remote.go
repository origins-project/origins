package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/history"
)

// The commands that share a corpus with another clone of it, which api.Fetch,
// api.Push and api.Clone say more about.

// sharing is what the commands that reach a remote take.
type sharing struct {
	root   string
	asJSON bool
}

func sharingFlags(name, corpus string, args []string, stdout, stderr io.Writer) (*sharing, []string, int) {
	c := &sharing{}
	flags := newFlags(name, stderr)
	flags.StringVar(&c.root, "C", ".", corpus)
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

// remoteCommand adds a remote, removes one, or lists them.
func remoteCommand(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && asksHelp(args[0]) {
		printHelp(stdout, helpOf("remote"), nil)
		return exitHelped
	}
	shape := "that is not the shape of it: origins remote add origin git@example.org:rossi.git, " +
		"origins remote remove origin, or origins remote list"
	if len(args) == 0 {
		return misuse(stderr, "%s", shape)
	}
	verb := args[0]
	c, rest, code := sharingFlags("remote "+verb, "the `corpus` whose remotes these are", args[1:], stdout, stderr)
	if code != exitOK {
		return code
	}
	var wants int
	switch verb {
	case "add":
		wants = 2
	case "remove":
		wants = 1
	case "list":
		wants = 0
	default:
		return misuse(stderr, "%s", shape)
	}
	if len(rest) != wants {
		return misuse(stderr, "%s", shape)
	}

	var (
		report *api.Remotes
		err    error
	)
	switch verb {
	case "add":
		report, err = api.AddRemote(c.root, rest[0], rest[1])
	case "remove":
		report, err = api.RemoveRemote(c.root, rest[0])
	default:
		report, err = api.ListRemotes(c.root)
	}
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Remotes) {
		switch verb {
		case "add":
			fmt.Fprintf(stdout, "added %s  %s\n", rest[0], rest[1])
		case "remove":
			fmt.Fprintf(stdout, "removed %s\n", rest[0])
		default:
			width := 0
			for _, one := range report.Remotes {
				width = max(width, len(one.Name))
			}
			for _, one := range report.Remotes {
				fmt.Fprintf(stdout, "%-*s  %s\n", width, one.Name, one.URL)
			}
		}
	}, nil)
}

// fetch brings in what a remote holds, or every remote, and says how far the branch
// and its upstream have each gone without the other.
func fetch(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	c, rest, code := sharingFlags("fetch", "the `corpus` to fetch into", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) > 1 {
		return misuse(stderr, "that is not the shape of it: origins fetch, or origins fetch origin")
	}
	report, err := api.Fetch(c.root, strings.Join(rest, ""), history.Terminal{In: stdin, Err: stderr})
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Standing) {
		switch {
		case report.Branch == "":
			fmt.Fprintln(stdout, "HEAD is on no branch")
		case report.Upstream == "":
			fmt.Fprintf(stdout, "%s follows no upstream\n", report.Branch)
		case report.Ahead == 0 && report.Behind == 0:
			fmt.Fprintf(stdout, "%s is level with %s\n", report.Branch, report.Upstream)
		default:
			fmt.Fprintf(stdout, "%s is %s behind and %s ahead of %s\n", report.Branch,
				tally(report.Behind, "commit", "commits"), tally(report.Ahead, "commit", "commits"),
				report.Upstream)
		}
	}, nil)
}

// push sends the branch to a remote with every snapshot on it.
func push(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	c, rest, code := sharingFlags("push", "the `corpus` to push", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) > 1 {
		return misuse(stderr, "that is not the shape of it: origins push, or origins push origin")
	}
	report, err := api.Push(c.root, strings.Join(rest, ""), history.Terminal{In: stdin, Err: stderr})
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Sent) {
		fmt.Fprintf(stdout, "pushed %s to %s at %s\n", report.Branch, report.Remote, short(report.Commit))
		if report.Upstream {
			fmt.Fprintf(stdout, "%s now follows %s/%s\n", report.Branch, report.Remote, report.Branch)
		}
	}, nil)
}

// clone copies a corpus somebody shared, and exits 1 saying what was found and what to
// run where the copy answers badly.
func clone(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var asJSON bool
	flags := newFlags("clone", stderr)
	flags.BoolVar(&asJSON, "json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	rest := flags.Args()
	if code := flagsFirst("clone", flags, rest, stderr); code != exitOK {
		return code
	}
	if len(rest) < 1 || len(rest) > 2 {
		return misuse(stderr, "that is not the shape of it: origins clone git@example.org:rossi.git, "+
			"or origins clone git@example.org:rossi.git rossi")
	}
	report, err := api.Clone(rest[0], strings.Join(rest[1:], ""), history.Terminal{In: stdin, Err: stderr})
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Cloning) {
		fmt.Fprintf(stdout, "cloned %s into %s\n", rest[0], report.Dir)
		if !report.Corpus {
			fmt.Fprintf(stdout, "%s has no origins.yaml at its root, so it is no corpus, and is kept as git cloned it\n",
				report.Dir)
		}
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
		if report.Errors > 0 {
			fmt.Fprintf(stdout, "check finds %d error(s) and %d warning(s), which origins check %s lists\n",
				report.Errors, report.Warnings, report.Dir)
		}
		if !report.Settled {
			fmt.Fprintf(stdout, "%s\norigins recover -C %s says what can be done about it\n",
				report.Standing, report.Dir)
		}
	}, (*api.Cloning).Found)
}
