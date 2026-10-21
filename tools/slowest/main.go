// Slowest prints the events of go test -json as go test would have printed them
// without -json, the output of a test kept only where it failed, and the ten slowest
// tests after them, so that a run says where its time went:
//
//	go test -json ./... | go run ./tools/slowest
//
// A test is timed as go test times it, from when it runs to when it and its subtests
// are done. Whether the run passed is go test's to say, through the pipe's status.
package main

import (
	"bufio"
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

type event struct {
	Action  string
	Package string
	Test    string
	Elapsed float64
	Output  string
}

type timed struct {
	name    string
	elapsed float64
}

const shown = 10

func main() {
	var tests []timed
	held := map[string][]string{}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 1<<20), 1<<26)
	out := bufio.NewWriter(os.Stdout)
	for in.Scan() {
		line := in.Bytes()
		var e event
		if json.Unmarshal(line, &e) != nil {
			// what go test printed that is not an event, such as a build failing
			out.Write(line)
			out.WriteByte('\n')
			continue
		}
		test := e.Package + " " + e.Test
		switch {
		case e.Test == "" && (e.Action == "output" || e.Action == "build-output"):
			out.WriteString(e.Output)
		case e.Action == "output":
			held[test] = append(held[test], e.Output)
		case e.Action == "fail" && e.Test != "":
			for _, said := range held[test] {
				out.WriteString(said)
			}
			fallthrough
		case e.Action == "pass" || e.Action == "skip":
			delete(held, test)
			if e.Action != "skip" && e.Test != "" && !strings.Contains(e.Test, "/") {
				tests = append(tests, timed{test, e.Elapsed})
			}
		}
		out.Flush()
	}
	slices.SortFunc(tests, func(a, b timed) int { return cmp.Compare(b.elapsed, a.elapsed) })
	fmt.Fprintf(out, "\nthe %d slowest tests:\n", min(shown, len(tests)))
	for _, t := range tests[:min(shown, len(tests))] {
		fmt.Fprintf(out, "%8.1fs  %s\n", t.elapsed, t.name)
	}
	out.Flush()
	if err := in.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
