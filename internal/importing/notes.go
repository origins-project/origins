// Package importing holds what every importer needs whatever format it reads: slugs
// handed out without collision, a count of what could not be carried, and the files an
// export names found on disk.
//
// It imports the model and nothing else of origins, so that an importer for one format
// shares nothing with an importer for another but this.
package importing

import (
	"fmt"
	"slices"
)

// Notes collects what an import could not carry.
//
// It counts rather than lists, because a file with four thousand unsourced events must
// produce one line and not four thousand: a report nobody can read is the same as no
// report. The count is the point — "430 events cite no source" is a research to-do list,
// and an import that said nothing would look like one that lost nothing.
type Notes struct {
	counts map[string]int
}

// NewNotes is a report with nothing in it yet.
func NewNotes() *Notes { return &Notes{counts: map[string]int{}} }

// Add counts one observation, the line it is reported as written as fmt.Sprintf writes
// it.
func (n *Notes) Add(format string, args ...any) {
	n.counts[fmt.Sprintf(format, args...)]++
}

// List returns one line per distinct observation, in sorted order so that importing the
// same file twice reports the same thing in the same order.
func (n *Notes) List() []string {
	out := make([]string, 0, len(n.counts))
	for message, count := range n.counts {
		if count > 1 {
			message = fmt.Sprintf("%s (%d times)", message, count)
		}
		out = append(out, message)
	}
	slices.Sort(out)
	return out
}
