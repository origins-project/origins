package main

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/api"
)

// The two commands for a corpus whose history has stopped being origins' own, which
// api.Adopt and api.Recover say more about.

// adopt records a change somebody made in the files rather than through origins.
func adopt(args []string, stdout, stderr io.Writer) int {
	var (
		root, message  string
		dryRun, asJSON bool
	)
	flags := newFlags("adopt", stderr)
	flags.StringVar(&root, "C", ".", "the `corpus` to record")
	flags.StringVar(&message, "m", "", "the `subject` line, in place of the one origins writes")
	flags.BoolVar(&dryRun, "dry-run", false, "say what would be recorded without recording it")
	flags.BoolVar(&asJSON, "json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	if flags.NArg() != 0 {
		return misuse(stderr, "adopt records whatever has been changed in the corpus, so it takes no arguments")
	}

	report, err := api.Adopt(api.Asking{Root: root, Command: "adopt", Message: message, DryRun: dryRun})
	if report == nil && err == nil {
		fmt.Fprintln(stdout, "nothing in this corpus has been changed by hand")
		return exitOK
	}
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Adoption) {
		if dryRun {
			fmt.Fprintln(stdout, report.Subject)
		} else {
			fmt.Fprintf(stdout, "%s  %s\n", short(report.Commit), report.Subject)
		}
		listing(stdout, report.Files, report.Emitted, dryRun)
	}, nil)
}

// listing writes out the files a change covers, marking the ones origins had to write
// again because the hand that made them did not spell everything the way the emitter
// does.
func listing(stdout io.Writer, files, emitted []string, dryRun bool) {
	rewritten := map[string]bool{}
	for _, path := range emitted {
		rewritten[path] = true
	}
	all := slices.Concat(files, emitted)
	slices.Sort(all)
	for _, path := range slices.Compact(all) {
		lead := "  "
		if dryRun {
			lead = "  would record "
		}
		if rewritten[path] {
			fmt.Fprintf(stdout, "%s%s, re-emitted as origins writes it\n", lead, path)
			continue
		}
		fmt.Fprintf(stdout, "%s%s\n", lead, path)
	}
}

// recovering says what has become of a corpus's history, and on being told which of
// the three ways back to take, takes it.
func recovering(args []string, stdout, stderr io.Writer) int {
	var (
		root                        string
		asJSON                      bool
		byAdopt, byRewrite, byReset bool
		flags                       = newFlags("recover", stderr)
	)
	flags.StringVar(&root, "C", ".", "the `corpus` whose history to look at")
	flags.BoolVar(&byAdopt, "adopt", false, "keep the commits as they are, and record having read them")
	flags.BoolVar(&byRewrite, "rewrite", false, "replace them with commits that say what they are")
	flags.BoolVar(&byReset, "reset", false, "take them off, leaving what they said in the working tree")
	flags.BoolVar(&asJSON, "json", false, "report as JSON on stdout")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	if flags.NArg() != 0 {
		return misuse(stderr, "recover works on the whole history, so it takes no arguments")
	}
	var chosen []string
	for flag, taken := range map[string]bool{api.ByAdopting: byAdopt, api.ByRewriting: byRewrite, api.ByResetting: byReset} {
		if taken {
			chosen = append(chosen, flag)
		}
	}
	slices.Sort(chosen)
	if len(chosen) > 1 {
		return misuse(stderr, "%s are different answers to the same question, so origins takes one of them",
			strings.Join(chosen, " and "))
	}

	report, err := api.Recover(root, strings.Join(chosen, ""))
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Resolution) {
		switch report.Took {
		case api.ByAdopting:
			kept(stdout, report)
		case api.ByRewriting:
			rewritten(stdout, report)
		case api.ByResetting:
			reset(stdout, report)
		default:
			surveyed(stdout, stderr, report)
		}
	}, nil)
}

func surveyed(stdout, stderr io.Writer, report *api.Resolution) {
	for _, problem := range report.Problems {
		fmt.Fprintln(stderr, problem)
	}
	if report.Settled {
		fmt.Fprintln(stdout, "origins wrote this history, so there is nothing to recover")
		unrecorded(stdout, report.Dirty)
		return
	}
	fmt.Fprintln(stdout, report.Standing)
	fmt.Fprintf(stdout, "\n%s:\n", report.Because)
	for i, w := range report.Ways {
		recommended := ""
		if i == 0 {
			recommended = "   <- what origins would do"
		}
		fmt.Fprintf(stdout, "  origins recover %-9s  %s%s\n", w.Flag, w.What, recommended)
	}
	unrecorded(stdout, report.Dirty)
}

// unrecorded mentions a dirty working tree wherever the state of the history is being
// reported. It is not a divergence and stops nothing, but somebody asking what shape
// the corpus is in wants to hear about it in the same breath.
func unrecorded(stdout io.Writer, dirty []string) {
	switch len(dirty) {
	case 0:
	case 1:
		fmt.Fprintf(stdout, "\n%s has been changed and not recorded; origins adopt records it\n", dirty[0])
	default:
		fmt.Fprintf(stdout, "\n%d files have been changed and not recorded; origins adopt records them\n",
			len(dirty))
	}
}

func kept(stdout io.Writer, report *api.Resolution) {
	if report.Subject == "" {
		fmt.Fprintf(stdout, "the history as it now stands is accounted for, down to %s\n", short(report.Commit))
		return
	}
	fmt.Fprintf(stdout, "%s  %s\n", short(report.Commit), report.Subject)
	for _, was := range report.Foreign {
		fmt.Fprintf(stdout, "  %s  %s  (%s)\n", short(was.Commit), was.Subject, was.Author)
	}
	if len(report.Emitted) > 0 {
		fmt.Fprintln(stdout, "re-emitted as origins writes them:")
		for _, path := range report.Emitted {
			fmt.Fprintf(stdout, "  %s\n", path)
		}
	}
}

func rewritten(stdout io.Writer, report *api.Resolution) {
	written := "%d commits rewritten as origins' own:\n"
	if len(report.Made) == 1 {
		written = "%d commit rewritten as origins' own:\n"
	}
	fmt.Fprintf(stdout, written, len(report.Made))
	for i, sha := range report.Made {
		was := report.Foreign[len(report.Foreign)-1-i]
		fmt.Fprintf(stdout, "  %s  was %s  %s\n", short(sha), short(was.Commit), was.Subject)
	}
}

func reset(stdout io.Writer, report *api.Resolution) {
	taken := fmt.Sprintf("%d commits taken off", len(report.Foreign))
	if len(report.Foreign) == 1 {
		taken = "one commit taken off"
	}
	fmt.Fprintf(stdout, "%s, back to %s  %s\n", taken, short(report.Anchor), report.AnchorSubject)
	fmt.Fprintln(stdout, "what they said is in the working tree and recorded nowhere; "+
		"origins adopt records it")
}
