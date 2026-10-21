package main

import (
	"flag"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/browse"
	"github.com/origins-project/origins/internal/chart"
	"github.com/origins-project/origins/internal/model"
)

// exporting hands the corpus to somebody who does not use origins, as a hidden view of it
// that hides nothing unless asked to.
func exporting(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && asksHelp(args[0]) {
		printHelp(stdout, helpOf("export"), nil)
		return exitHelped
	}
	if len(args) == 0 {
		return misuse(stderr, "export what? origins export gedcom --out <file.gdz>, html --out <dir>, or chart <person>")
	}
	switch args[0] {
	case "gedcom":
		return exportGEDCOM(args[1:], stdout, stderr)
	case "html":
		return exportHTML(args[1:], stdout, stderr)
	case "chart":
		return exportChart(args[1:], stdout, stderr)
	}
	return misuse(stderr, "origins exports gedcom, html and a chart, and not %s", args[0])
}

// hidingFlags are the views an export is made as, as browse serves them.
func hidingFlags(flags *flag.FlagSet, hiding *model.Hiding, public *bool) {
	flags.BoolVar(&hiding.Private, "hide-private", false, "leave out what is private")
	flags.BoolVar(&hiding.Living, "hide-living", false, "leave out whoever is taken as living")
	flags.BoolVar(public, "public", false, "the same as --hide-private and --hide-living together")
}

func exportGEDCOM(args []string, stdout, stderr io.Writer) int {
	var (
		asked  api.GEDCOMAsked
		public bool
	)
	q, code := queryFlags("export gedcom", args, 0, "export gedcom --out <file.gdz>", stdout, stderr,
		func(flags *flag.FlagSet) {
			flags.StringVar(&asked.Out, "out", "", "the GEDZIP `file` to write")
			hidingFlags(flags, &asked.Hiding, &public)
		})
	if code != exitOK {
		return code
	}
	if asked.Out == "" {
		return misuse(stderr, "a GEDZIP is written to a file, and --out names it")
	}
	if public {
		asked.Hiding = model.Hiding{Private: true, Living: true}
	}
	asked.With = q.with
	report, err := api.ExportGEDCOM(q.root, asked)
	return answered(q, report, err, func(r *api.GEDCOMWritten) []string { return r.Problems },
		func(r *api.GEDCOMWritten) []api.Supposition { return r.Read }, stdout, stderr, func(report *api.GEDCOMWritten) {
			fmt.Fprintf(stdout, "wrote %s: %s, %s\n", report.Out, counts(report.Records),
				api.Tally(report.Pictures, "picture", "pictures"))
			if len(report.Extensions) > 0 {
				fmt.Fprintf(stdout, "extensions: %s\n", counts(report.Extensions))
			}
			if len(report.Kept) > 0 {
				fmt.Fprintf(stdout, "kept as an import read them: %s\n", counts(report.Kept))
			}
			if len(report.Omitted) > 0 {
				fmt.Fprintf(stdout, "left out, GEDCOM having nowhere for them: %s\n", counts(report.Omitted))
			}
			for _, path := range report.Missing {
				fmt.Fprintf(stdout, "not there to put in: %s\n", path)
			}
		})
}

func exportHTML(args []string, stdout, stderr io.Writer) int {
	var (
		out     string
		options browse.Options
		public  bool
	)
	q, code := queryFlags("export html", args, 0, "export html --out <dir>", stdout, stderr,
		func(flags *flag.FlagSet) {
			flags.StringVar(&out, "out", "", "the `directory` to write the pages into, not there or empty")
			flags.StringVar(&options.Notice, "notice", "", "a `text` shown below the bar on every page")
			hidingFlags(flags, &options.Hiding, &public)
		})
	if code != exitOK {
		return code
	}
	if out == "" {
		return misuse(stderr, "the pages are written into a directory, and --out names it")
	}
	if public {
		options.Hiding = model.Hiding{Private: true, Living: true}
	}
	options.With = q.with
	report, err := browse.Export(q.root, out, options)
	return answered(q, report, err, func(r *browse.Exported) []string { return r.Problems },
		func(r *browse.Exported) []api.Supposition { return r.Read }, stdout, stderr, func(report *browse.Exported) {
			fmt.Fprintf(stdout, "wrote %s: %s, %s\n", report.Out, counts(report.Records),
				api.Tally(report.Files, "file", "files"))
			fmt.Fprintf(stdout, "open %s in a browser\n", filepath.Join(report.Out, "index.html"))
		})
}

// counts is a count by name, in the order of the names, a count of the records kept in
// a kind's directory saying one of the kind where there is one.
func counts(by map[string]int) string {
	var out []string
	for _, name := range slices.Sorted(maps.Keys(by)) {
		if kind := model.KindIn(name); kind != "" {
			out = append(out, api.Tally(by[name], string(kind), name))
		} else {
			out = append(out, fmt.Sprintf("%d %s", by[name], name))
		}
	}
	return strings.Join(out, ", ")
}

func exportChart(args []string, stdout, stderr io.Writer) int {
	var (
		asked  api.ChartAsked
		shape  string
		public bool
	)
	q, code := queryFlags("export chart", args, 1, "export chart --shape <shape> --out <file> <person>",
		stdout, stderr,
		func(flags *flag.FlagSet) {
			flags.StringVar(&shape, "shape", string(chart.Ancestors), "the chart's `shape`")
			flags.IntVar(&asked.Distance, "distance", chart.DefaultDistance, "how many `steps` the chart reaches")
			flags.StringVar(&asked.To, "to", "", "the `person` a path runs to")
			flags.StringVar(&asked.Out, "out", "", "the `file` to draw it to, as svg, pdf or png by its extension")
			hidingFlags(flags, &asked.Hiding, &public)
		})
	if code != exitOK {
		return code
	}
	if asked.Out == "" {
		return misuse(stderr, "a chart is drawn to a file, and --out names it")
	}
	if public {
		asked.Hiding = model.Hiding{Private: true, Living: true}
	}
	asked.Person, asked.Shape, asked.With = q.args[0], chart.Shape(shape), q.with
	report, err := api.ExportChart(q.root, asked)
	return answered(q, report, err, func(r *api.ChartDrawn) []string { return r.Problems },
		func(r *api.ChartDrawn) []api.Supposition { return r.Read }, stdout, stderr, func(report *api.ChartDrawn) {
			people := fmt.Sprintf("%d people", report.People)
			if report.People == 1 {
				people = "1 person"
			}
			fmt.Fprintf(stdout, "drew the %s chart of %s to %s: %s\n",
				report.Shape, report.Person.Name, report.Out, people)
			if report.Truncated {
				fmt.Fprintf(stdout, "it stopped at %d people\n", report.Limit)
			}
			if report.Undated > 0 {
				fmt.Fprintf(stdout, "%d left off for want of a date\n", report.Undated)
			}
			switch path := report.Path; {
			case path == nil:
			case path.Related && path.To != report.Person.Slug:
				fmt.Fprintf(stdout, "%s is the %s of %s\n", report.Person.Name, path.Relation, report.To.Name)
			case !path.Related:
				fmt.Fprintf(stdout, "%s and %s have no ancestor in common in this corpus\n",
					report.Person.Name, report.To.Name)
			}
		})
}
