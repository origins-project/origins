package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
)

// The researcher's questions, which api.Ancestors, api.Kinship and api.Todo say more
// about. Each reads the corpus and never the repository, and takes --with.

// querying is what every one of them takes.
type querying struct {
	root   string
	asJSON bool
	with   several
	args   []string
}

// queryFlags parses what the questions take, and checks the question has the right
// number of names in it.
func queryFlags(name string, args []string, names int, shape string, stdout, stderr io.Writer,
	extra func(*flag.FlagSet)) (*querying, int) {
	q := &querying{}
	flags := newFlags(name, stderr)
	flags.StringVar(&q.root, "C", ".", "the `corpus` to ask")
	flags.BoolVar(&q.asJSON, "json", false, "report as JSON on stdout")
	flags.Var(&q.with, "with", "answer as though this `hypothesis` were true")
	if extra != nil {
		extra(flags)
	}
	if code := parse(flags, args, stdout); code != exitOK {
		return nil, code
	}
	q.args = flags.Args()
	if code := flagsFirst(name, flags, q.args, stderr); code != exitOK {
		return nil, code
	}
	if len(q.args) != names {
		return nil, misuse(stderr, "that is not the shape of it: origins %s", shape)
	}
	return q, exitOK
}

// answered writes a report as JSON, or has text write it for a reader, putting what
// could not be loaded and how each hypothesis was read around it either way.
func answered[T any](q *querying, report *T, err error, problems func(*T) []string, read func(*T) []api.Supposition,
	stdout, stderr io.Writer, text func(*T)) int {
	return answer(report, err, q.asJSON, stdout, stderr, func(report *T) {
		for _, problem := range problems(report) {
			fmt.Fprintln(stderr, problem)
		}
		for _, r := range read(report) {
			fmt.Fprintln(stdout, r)
		}
		text(report)
	}, func(report *T) bool { return len(problems(report)) > 0 })
}

func ancestors(args []string, stdout, stderr io.Writer) int {
	return descent("ancestors", true, args, stdout, stderr)
}

func descendants(args []string, stdout, stderr io.Writer) int {
	return descent("descendants", false, args, stdout, stderr)
}

// descent lists a person's line a generation at a time, the nearest first.
func descent(name string, upward bool, args []string, stdout, stderr io.Writer) int {
	var depth int
	q, code := queryFlags(name, args, 1, name+" [--depth n] <person>", stdout, stderr, func(flags *flag.FlagSet) {
		flags.IntVar(&depth, "depth", 0, "stop after `n` generations")
	})
	if code != exitOK {
		return code
	}
	asked := api.Descendants
	if upward {
		asked = api.Ancestors
	}
	report, err := asked(q.root, q.with, q.args[0], depth)
	return answered(q, report, err, func(r *api.Lineage) []string { return r.Problems },
		func(r *api.Lineage) []api.Supposition { return r.Read }, stdout, stderr, func(report *api.Lineage) {
			if len(report.Generations) == 0 {
				fmt.Fprintf(stdout, "%s has no %s in this corpus\n", report.Person.Slug, name)
				return
			}
			width := 0
			for _, one := range report.Generations {
				for _, person := range one.People {
					width = max(width, len(person.Slug))
				}
			}
			for _, one := range report.Generations {
				fmt.Fprintln(stdout, one.Called)
				for _, person := range one.People {
					fmt.Fprintf(stdout, "  %-*s  %s\n", width, person.Slug, person.Name)
				}
			}
		})
}

// kinship names what the first person is to the second, and the people it runs
// through.
func kinship(args []string, stdout, stderr io.Writer) int {
	q, code := queryFlags("kinship", args, 2, "kinship <person> <person>", stdout, stderr, nil)
	if code != exitOK {
		return code
	}
	report, err := api.Kinship(q.root, q.with, q.args[0], q.args[1])
	return answered(q, report, err, func(r *api.Kin) []string { return r.Problems },
		func(r *api.Kin) []api.Supposition { return r.Read }, stdout, stderr, func(report *api.Kin) {
			switch {
			case !report.Related:
				fmt.Fprintf(stdout, "%s and %s have no ancestor in common in this corpus\n",
					report.From.Name, report.To.Name)
				return
			case report.From.Slug == report.To.Slug:
				fmt.Fprintf(stdout, "%s is one person\n", report.From.Name)
				return
			}
			fmt.Fprintf(stdout, "%s is the %s of %s\n\n", report.From.Name, report.Relation, report.To.Name)
			slugs, names := 0, 0
			for _, one := range report.Path {
				slugs, names = max(slugs, len(one.Slug)), max(names, len(one.Name))
			}
			for i, one := range report.Path {
				said := ""
				if i > 0 {
					said = possessive(report.Path[i-1].Sex) + " " + one.Is
				}
				row := fmt.Sprintf("  %-*s  %-*s  %s", slugs, one.Slug, names, one.Name, said)
				fmt.Fprintln(stdout, strings.TrimRight(row, " "))
			}
		})
}

func possessive(sex model.Sex) string {
	switch sex {
	case model.Male:
		return "his"
	case model.Female:
		return "her"
	}
	return "their"
}

// todo lists the research still to do. None of it is wrong, so a long list exits
// nought: it is the answer to the question, and a corpus being researched has one.
func todo(args []string, stdout, stderr io.Writer) int {
	q, code := queryFlags("todo", args, 0, "todo", stdout, stderr, nil)
	if code != exitOK {
		return code
	}
	report, err := api.Todo(q.root, q.with)
	return answered(q, report, err, func(r *api.ToDo) []string { return r.Problems },
		func(r *api.ToDo) []api.Supposition { return r.Read }, stdout, stderr, func(report *api.ToDo) {
			for _, f := range report.Todo {
				where := strings.TrimSpace(f.Entity + " " + f.Path)
				fmt.Fprintf(stdout, "%s: %s (%s)\n", where, f.Message, f.Rule)
			}
			fmt.Fprintln(stdout, tally(len(report.Todo), "thing to do", "things to do"))
		})
}

func indexed(args []string, stdout, stderr io.Writer) int {
	q, code := queryFlags("index", args, 0, "index", stdout, stderr, nil)
	if code != exitOK {
		return code
	}
	report, err := api.IndexOf(q.root, q.with)
	return answered(q, report, err, func(r *api.Index) []string { return r.Problems },
		func(r *api.Index) []api.Supposition { return r.Read }, stdout, stderr, func(report *api.Index) {
			fmt.Fprintln(stdout, countedAll(report.Counts()))
		})
}
