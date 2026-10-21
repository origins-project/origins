// Command origins reads, checks and changes a genealogy corpus.
//
// Each command parses its flags, puts the question to internal/api, and prints what
// comes back, as JSON or for a reader. What the answer is lives there, so that the
// pages and an agent are given the same one.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
)

// Exit codes are kept narrow enough for git bisect run: nothing wrong, something wrong
// with the corpus, or something wrong with the way origins was asked.
const (
	exitOK       = 0
	exitFindings = 1
	exitMisuse   = 2

	// exitHelped is a command that has printed its help and done nothing else. run
	// exits 0 for it.
	exitHelped = -1
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if code := dispatch(args, stdin, stdout, stderr); code != exitHelped {
		return code
	}
	return exitOK
}

func dispatch(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitMisuse
	}
	command, rest := args[0], args[1:]
	if change, ours := changes[command]; ours {
		return change(rest, stdout, stderr)
	}
	switch command {
	case "init":
		return initialise(rest, stdout, stderr)
	case "load":
		return load(rest, stdout, stderr)
	case "check":
		return validate(rest, stdout, stderr)
	case "fmt":
		return canonicalise(rest, stdout, stderr)
	case "import":
		return ingest(rest, stdin, stdout, stderr)
	case "ancestors":
		return ancestors(rest, stdout, stderr)
	case "descendants":
		return descendants(rest, stdout, stderr)
	case "kinship":
		return kinship(rest, stdout, stderr)
	case "todo":
		return todo(rest, stdout, stderr)
	case "index":
		return indexed(rest, stdout, stderr)
	case "log":
		return research(rest, stdout, stderr)
	case "whence":
		return whence(rest, stdout, stderr)
	case "blame":
		return blame(rest, stdout, stderr)
	case "why":
		return why(rest, stdout, stderr)
	case "hypothesis":
		return hypothesis(rest, stdout, stderr)
	case "undo":
		return undo(rest, stdout, stderr)
	case "snapshot":
		return snapshot(rest, stdout, stderr)
	case "adopt":
		return adopt(rest, stdout, stderr)
	case "recover":
		return recovering(rest, stdout, stderr)
	case "remote":
		return remoteCommand(rest, stdout, stderr)
	case "fetch":
		return fetch(rest, stdin, stdout, stderr)
	case "push":
		return push(rest, stdin, stdout, stderr)
	case "pull":
		return pull(rest, stdin, stdout, stderr)
	case "clone":
		return clone(rest, stdin, stdout, stderr)
	case "schema":
		return describe(rest, stdout, stderr)
	case "browse":
		return serve(rest, stdout, stderr)
	case "export":
		return exporting(rest, stdout, stderr)
	case "mcp":
		return serveAgent(rest, stdout, stderr)
	case "version":
		return versioned(rest, stdout, stderr)
	case "help":
		return helpCommand(rest, stdin, stdout, stderr)
	case "-h", "--help":
		usage(stdout)
		return exitOK
	default:
		fmt.Fprintf(stderr, "origins: unknown command %q\n\n", command)
		usage(stderr)
		return exitMisuse
	}
}

func misuse(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "origins: "+format+"\n", args...)
	return exitMisuse
}

// answer writes what a question put to internal/api came back with, and is how every
// command ends. A refusal is said in place of an answer. An answer is written as JSON,
// or by text for a reader, with whatever went wrong beside it said first, and found
// says whether it is a finding, which exits 1. A corpus written and not recorded is
// said after the answer, and a lock left behind last of all, whatever happened, since
// the command's own work is done and that is about the next origins, which will refuse
// until somebody removes the file.
func answer[T any](report *T, err error, asJSON bool, stdout, stderr io.Writer,
	text func(*T), found func(*T) bool) int {
	locks, err := api.Without[*history.Unreleased](err)
	defer func() {
		for _, lock := range locks {
			fmt.Fprintf(stderr, "origins: %v\n", lock)
		}
	}()
	if report == nil {
		return refused(err, stderr)
	}
	unrecorded, err := api.Without[*api.Unrecorded](err)
	for _, part := range parts(err) {
		fmt.Fprintf(stderr, "origins: %v\n", part)
	}
	if asJSON {
		if err := writeJSON(stdout, report); err != nil {
			return misuse(stderr, "%v", err)
		}
	} else if text != nil {
		text(report)
	}
	switch {
	case len(unrecorded) > 0:
		return misuse(stderr, "%v", unrecorded[0])
	case found != nil && found(report):
		return exitFindings
	}
	return exitOK
}

// refused says why there is no answer.
//
// A corpus that contradicts itself, and a hypothesis that no longer applies to it, are
// findings about the records rather than mistakes in how origins was asked, and exit
// 1. Everything else exits 2, a corpus that could not be put back as it was above all,
// which is said after whatever it was being put back from, since that is the state it
// has been left in.
func refused(err error, stderr io.Writer) int {
	if err == nil {
		return exitOK
	}
	var unsettled *history.Unsettled
	if errors.As(err, &unsettled) {
		return misuse(stderr, "%v\n\norigins recover says what can be done about it", unsettled)
	}
	unreadable, rest := api.Without[*api.Unreadable](err)
	for _, one := range unreadable {
		for _, problem := range one.Problems {
			fmt.Fprintln(stderr, problem)
		}
		fmt.Fprintf(stderr, "origins: %v\n", one)
	}
	contradictions, rest := api.Without[*ops.Contradiction](rest)
	for _, one := range contradictions {
		fmt.Fprintln(stderr, one)
	}
	var (
		contradiction *ops.Contradiction
		drifted       *ops.Drifted
	)
	if len(contradictions) == 0 && errors.As(rest, &contradiction) {
		fmt.Fprintln(stderr, rest)
		return exitFindings
	}
	found := len(contradictions) > 0 || errors.As(rest, &drifted)
	unrestored, rest := api.Without[*ops.Unrestored](rest)
	for _, part := range parts(rest) {
		fmt.Fprintf(stderr, "origins: %v\n", part)
	}
	switch {
	case len(unrestored) > 0:
		return misuse(stderr, "%v", unrestored[0])
	case found:
		return exitFindings
	}
	return exitMisuse
}

// parts is an error a line at a time, where it is several joined.
func parts(err error) []error {
	if err == nil {
		return nil
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return []error{err}
	}
	var out []error
	for _, one := range joined.Unwrap() {
		out = append(out, parts(one)...)
	}
	return out
}

// flagsFirst catches a flag written after the positional arguments. Parsing stops at
// the first argument that is not a flag, so one written after them is read as part of
// the command, and the failure that follows is about the wrong thing entirely.
//
// A value may perfectly well start with a dash — a longitude west of Greenwich does —
// so only a name the command actually has is worth saying anything about.
func flagsFirst(name string, flags *flag.FlagSet, rest []string, stderr io.Writer) int {
	for _, arg := range rest {
		written, _, _ := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if strings.HasPrefix(arg, "-") && flags.Lookup(written) != nil {
			return misuse(stderr, "%s is a flag, and flags come first: origins %s %s ...", arg, name, arg)
		}
	}
	return exitOK
}

// corpusFlags handles what every command that reads a corpus takes: a directory and
// --json.
func corpusFlags(name string, args []string, stdout, stderr io.Writer, extra func(*flag.FlagSet)) (
	root string, asJSON bool, code int,
) {
	flags := newFlags(name, stderr)
	json := flags.Bool("json", false, "report as JSON on stdout")
	if extra != nil {
		extra(flags)
	}
	if code := parse(flags, args, stdout); code != exitOK {
		return "", false, code
	}

	root = "."
	switch flags.NArg() {
	case 0:
	case 1:
		root = flags.Arg(0)
	default:
		return "", false, misuse(stderr, "%s reads one corpus at a time", name)
	}
	return root, *json, exitOK
}

// describe prints the JSON Schema of one kind of record, or every one by name, which is
// JSON whoever asks, and so takes no --json.
func describe(args []string, stdout, stderr io.Writer) int {
	flags := newFlags("schema", stderr)
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	name := ""
	switch flags.NArg() {
	case 0:
	case 1:
		name = flags.Arg(0)
	default:
		return misuse(stderr, "schema describes one kind at a time, or every kind")
	}
	read, err := api.Schema(name)
	if err != nil {
		return misuse(stderr, "%v", err)
	}
	stdout.Write(read)
	if len(read) > 0 && read[len(read)-1] != '\n' {
		fmt.Fprintln(stdout)
	}
	return exitOK
}

// countedAll is how many records of each kind there are, every kind said.
func countedAll(c api.Counts) string {
	return strings.Join([]string{
		tally(c.People, "person", "people"),
		tally(c.Families, "family", "families"),
		tally(c.Places, "place", "places"),
		tally(c.Sources, "source", "sources"),
		tally(c.Repositories, "repository", "repositories"),
		tally(c.Notes, "note", "notes"),
		tally(c.Hypotheses, "hypothesis", "hypotheses"),
	}, ", ")
}

func load(args []string, stdout, stderr io.Writer) int {
	root, asJSON, code := corpusFlags("load", args, stdout, stderr, nil)
	if code != exitOK {
		return code
	}
	report, err := api.Load(root)
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Summary) {
		if report.Project.Name != "" {
			fmt.Fprintln(stdout, report.Project.Name)
		}
		if report.Project.Description != "" {
			fmt.Fprintln(stdout, report.Project.Description)
		}
		fmt.Fprintln(stdout, countedAll(report.Counts))
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
	}, func(report *api.Summary) bool { return len(report.Problems) > 0 })
}

func validate(args []string, stdout, stderr io.Writer) int {
	var (
		errorsOnly bool
		with       several
	)
	root, asJSON, code := corpusFlags("check", args, stdout, stderr, func(flags *flag.FlagSet) {
		flags.BoolVar(&errorsOnly, "errors", false, "leave out warnings")
		flags.Var(&with, "with", "answer as though this `hypothesis` were true")
	})
	if code != exitOK {
		return code
	}
	report, err := api.Check(root, with, errorsOnly)
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Validation) {
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
		for _, r := range report.Read {
			fmt.Fprintln(stdout, r)
		}
		for _, f := range report.Findings {
			// a corpus-wide finding names no entity, so it is addressed by path alone
			where := strings.TrimSpace(f.Entity + " " + f.Path)
			fmt.Fprintf(stdout, "%s: %s: %s (%s)\n", f.Severity, where, f.Message, f.Rule)
		}
		fmt.Fprintf(stdout, "%d error(s), %d warning(s)\n", report.Errors, report.Warnings)
	}, func(report *api.Validation) bool { return len(report.Problems) > 0 || report.Errors > 0 })
}

// canonicalise re-emits a corpus, which api.Fmt says more about.
func canonicalise(args []string, stdout, stderr io.Writer) int {
	var list bool
	root, asJSON, code := corpusFlags("fmt", args, stdout, stderr, func(flags *flag.FlagSet) {
		flags.BoolVar(&list, "list", false, "report what would change without writing")
	})
	if code != exitOK {
		return code
	}
	report, err := api.Fmt(root, list)
	return answer(report, err, asJSON, stdout, stderr, func(report *api.Formatted) {
		for _, problem := range report.Problems {
			fmt.Fprintln(stderr, problem)
		}
		for _, path := range report.Changed {
			fmt.Fprintln(stdout, path)
		}
	}, func(report *api.Formatted) bool {
		// --list is a question, so a corpus that is not canonical is a finding.
		// Rewriting one is an answer, and leaves nothing to report.
		return len(report.Problems) > 0 || list && len(report.Changed) > 0
	})
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// ingest reads a GEDCOM export or a Gramps file into a new corpus, which api.Import
// says more about.
func ingest(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && asksHelp(args[0]) {
		printHelp(stdout, helpOf("import"), nil)
		return exitHelped
	}
	if len(args) == 0 {
		return misuse(stderr, "import needs a format: origins import gedcom|gramps <file>")
	}
	format, rest := args[0], args[1:]
	if format != "gedcom" && format != "gramps" {
		return misuse(stderr, "cannot import %q: origins reads gedcom and gramps", format)
	}

	flags := newFlags("import "+format, stderr)
	asJSON := flags.Bool("json", false, "report as JSON on stdout")
	into := flags.String("into", "", "the `directory` to create the corpus in")
	name := flags.String("name", "", "the project's `name`")
	ask := flags.Bool("interactive", false, "choose the slug whenever two entities would share one")
	media := new(string)
	if format == "gramps" {
		media = flags.String("media", "", "another `directory` the pictures may be in, "+
			"looked in after the ones the file names")
	}
	if code := parse(flags, rest, stdout); code != exitOK {
		return code
	}
	if flags.NArg() != 1 {
		return misuse(stderr, "import %s reads one file at a time", format)
	}
	asked := api.Importing{Format: format, File: flags.Arg(0), Into: *into, Name: *name, Media: *media}
	if *ask {
		asked.Choose = asker(stdin, stderr)
	}
	report, err := api.Import(asked)
	return answer(report, err, *asJSON, stdout, stderr, func(report *api.Imported) {
		fmt.Fprintf(stdout, "%s: %s\n", report.Corpus, api.CountedIn(report.Counts))
		for _, note := range report.Notes {
			fmt.Fprintf(stdout, "  %s\n", note)
		}
		if report.Commit != "" {
			fmt.Fprintf(stdout, "\n%s  %s\n", short(report.Commit), report.Subject)
			if report.Started {
				fmt.Fprintln(stdout, "  recorded in a new git repository")
			} else {
				fmt.Fprintf(stdout, "  recorded in the repository at %s\n", report.History)
			}
		}
		fmt.Fprintf(stdout, "\nnow run: origins check %s\n", report.Corpus)
	}, nil)
}

// short is a sha as a reader quotes one back.
func short(sha string) string { return api.Short(sha) }

func tally(n int, one, many string) string { return api.Tally(n, one, many) }

// asker settles a slug collision by putting it to the person running the import, which
// is what DESIGN.md asks for: origins proposes discriminators worth reading in three
// years, and a human who knows the family chooses between them or names it better.
//
// An empty answer takes the first proposal, so holding down return imports the file.
func asker(stdin io.Reader, prompt io.Writer) importing.Chooser {
	reader := bufio.NewReader(stdin)
	return func(what string, proposals []model.Slug) model.Slug {
		fmt.Fprintf(prompt, "\n%s already has that name. Choose one:\n", what)
		for i, proposal := range proposals {
			fmt.Fprintf(prompt, "  %d) %s\n", i+1, proposal)
		}
		fmt.Fprint(prompt, "number, or a slug of your own [1]: ")

		line, err := reader.ReadString('\n')
		answer := strings.TrimSpace(line)
		if err != nil && answer == "" || answer == "" {
			return ""
		}
		if n, err := strconv.Atoi(answer); err == nil {
			if n >= 1 && n <= len(proposals) {
				return proposals[n-1]
			}
			fmt.Fprintf(prompt, "  there is no %d; taking %s\n", n, proposals[0])
			return ""
		}
		slug, err := model.ParseSlug(answer)
		if err != nil {
			fmt.Fprintf(prompt, "  %v; taking %s\n", err, proposals[0])
			return ""
		}
		return slug
	}
}
