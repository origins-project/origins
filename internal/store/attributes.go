package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"

	"github.com/origins-project/origins/internal/model"
)

// AttributesFile is where a corpus tells git what its files are.
const AttributesFile = ".gitattributes"

// declared is what the corpus tells git: the manifest and every record are text, and
// their lines end in LF on every machine that checks them out.
//
// It names what origins writes and nothing else. A photograph marked as text would be
// corrupted on the way out, and a note somebody keeps beside the corpus is theirs to
// declare, so the patterns are anchored to the corpus root and reach only the markdown
// under the seven directories records are kept in.
var declared = func() []string {
	out := []string{"/" + ProjectFile + " text eol=lf"}
	for _, kind := range model.Kinds {
		out = append(out, "/"+kind.Dir()+"/**/*.md text eol=lf")
	}
	return out
}()

// DeclareEndings makes sure the corpus at root tells git its files are LF, says
// whether anything had to be written for that, and hands back the way to put the file
// back as it was.
//
// A corpus is byte-stable or its diffs are noise, and git on Windows converts every
// line ending to CRLF when it checks a file out unless it is told otherwise — a clone
// made there would hold a corpus that is canonical nowhere. Origins turns that off for
// every checkout it makes itself; this is what reaches the ones it does not.
//
// The declaration lives in the corpus directory rather than at the top of the
// repository, so a corpus kept among somebody's notes changes nothing above it and
// carries its declaration along wherever it is copied. A file already there is added
// to and never rewritten: the lines it lacks go at the end, where a later line
// overrides an earlier one, and whatever else it says stays as it was.
func DeclareEndings(root string) (written bool, undo func() error, err error) {
	path := filepath.Join(root, AttributesFile)
	was, err := os.ReadFile(path)
	existed := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, nil, err
	}

	var present []string
	for _, line := range bytes.Split(was, []byte("\n")) {
		present = append(present, string(bytes.TrimSpace(line)))
	}
	var missing []string
	for _, line := range declared {
		if !slices.Contains(present, line) {
			missing = append(missing, line)
		}
	}
	if len(missing) == 0 {
		return false, func() error { return nil }, nil
	}

	now := slices.Clone(was)
	if len(now) > 0 && now[len(now)-1] != '\n' {
		now = append(now, '\n')
	}
	for _, line := range missing {
		now = append(append(now, line...), '\n')
	}
	if err := os.WriteFile(path, now, 0o644); err != nil {
		return false, nil, err
	}
	return true, func() error {
		if existed {
			return os.WriteFile(path, was, 0o644)
		}
		return os.Remove(path)
	}, nil
}
