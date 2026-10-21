package ops

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// Change is the set of file writes a staged transaction amounts to: the bytes each
// changed file is to be given, the files that go, and the ones that only change their
// name. It holds the whole change in memory, which is what lets Apply be all or
// nothing.
type Change struct {
	root   string
	write  map[string][]byte
	move   map[string]string // from, to
	remove []string
}

// Move is one thing carried from one name to another: an entity renamed, taking with
// it every photograph and scan attached to it, or one of those files going after a
// record that has been merged away.
//
// A move rather than a write and a removal, because origins does not hold the bytes of
// what is attached to an entity and has no business reading a hundred megabytes of
// scans in order to put them down again unchanged.
type Move struct{ From, To string }

// Moved names what the change carries from one name to another, in order of where it
// came from.
func (c *Change) Moved() []Move {
	out := make([]Move, 0, len(c.move))
	for from, to := range c.move {
		out = append(out, Move{From: from, To: to})
	}
	slices.SortFunc(out, func(a, b Move) int { return strings.Compare(a.From, b.From) })
	return out
}

// Written names the files the change writes, relative to the corpus root and sorted,
// which is what a caller reports and a test reads.
func (c *Change) Written() []string {
	out := make([]string, 0, len(c.write))
	for relative := range c.write {
		out = append(out, relative)
	}
	slices.Sort(out)
	return out
}

// Removed names the files the change takes out, the same way.
func (c *Change) Removed() []string { return slices.Clone(c.remove) }

// Paths is every file the change touches, which is what a commit recording it is
// limited to. Both ends of a move are named, since git has to be shown the file
// leaving one and arriving at the other before it can call it a rename.
func (c *Change) Paths() []string {
	out := append(c.Written(), c.remove...)
	for _, moved := range c.Moved() {
		out = append(out, moved.From, moved.To)
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Empty reports a change that would alter nothing. Mutations that work out to the state
// already on disk are not failures, but they are not history either.
func (c *Change) Empty() bool {
	return len(c.write) == 0 && len(c.move) == 0 && len(c.remove) == 0
}

// Apply writes the change, or leaves the directory as it found it.
//
// Every file is read before it is touched, so that a write failing partway through —
// a full disk, a file someone has open, a permission — can be undone. A corpus left
// half in one state and half in another is the outcome a transaction exists to prevent,
// and it is the one a researcher would have no way to detect.
func (c *Change) Apply() error {
	_, err := c.apply()
	return err
}

// apply is Apply, handing back what it would take to undo the write. Recording a
// change can still fail after the files are on disk, and then the write has to come
// back off them.
func (c *Change) apply() (*restore, error) {
	undo := &restore{root: c.root, prior: map[string][]byte{}}

	// Moves first, because what is written afterwards is written where the move put
	// it: an entity renamed and edited in the same breath is one file at the new name
	// and not two at both.
	for _, moved := range c.ordered() {
		if err := undo.mkdir(moved.To); err != nil {
			return nil, undo.rollback(err)
		}
		if err := os.Rename(undo.path(moved.From), undo.path(moved.To)); err != nil {
			return nil, undo.rollback(fmt.Errorf("%s to %s: %w", moved.From, moved.To, err))
		}
		undo.moved = append(undo.moved, moved)
	}

	for _, relative := range c.Written() {
		if err := undo.remember(relative); err != nil {
			return nil, undo.rollback(err)
		}
		if err := undo.mkdir(relative); err != nil {
			return nil, undo.rollback(err)
		}
		if err := os.WriteFile(undo.path(relative), c.write[relative], 0o644); err != nil {
			return nil, undo.rollback(fmt.Errorf("%s: %w", relative, err))
		}
	}

	for _, relative := range c.remove {
		if err := undo.remember(relative); err != nil {
			return nil, undo.rollback(err)
		}
		if err := os.Remove(undo.path(relative)); err != nil && !os.IsNotExist(err) {
			return nil, undo.rollback(fmt.Errorf("%s: %w", relative, err))
		}
	}

	// A directory left holding nothing goes with what was in it. A folder still named
	// after a record the corpus no longer has is a trace of it that nothing accounts
	// for, whether the record was deleted or folded into another and its photographs
	// carried off after it. Deepest first, so a directory is asked about only once
	// everything under it has gone.
	emptied := slices.Clone(c.remove)
	for _, moved := range c.Moved() {
		emptied = append(emptied, moved.From)
	}
	slices.SortFunc(emptied, func(a, b string) int { return len(b) - len(a) })
	for _, relative := range emptied {
		undo.prune(relative)
	}
	return undo, nil
}

// ordered is the moves in an order they can be carried out in, one at a time.
//
// A move onto a name another move has still to vacate puts one file over the other, and
// os.Rename replaces what it finds without a word, so a record renamed away and another
// renamed onto the name it left would end as one file holding the second of them. Each
// move waits until nothing is left to go from where it is going. Where every move left
// is waiting on another the names are going round in a circle — two records that
// swapped them — and one of the circle is carried aside to a name nothing uses, which
// empties the place the rest are waiting for, and goes on from there afterwards. What
// is carried out is what a failure then carries back, in reverse, so the detour is
// undone along with everything else.
func (c *Change) ordered() []Move {
	pending := c.Moved()
	named := slices.Clone(pending)
	out := make([]Move, 0, len(pending))
	for len(pending) > 0 {
		if i := slices.IndexFunc(pending, func(m Move) bool { return waiting(pending, m) < 0 }); i >= 0 {
			out = append(out, pending[i])
			pending = slices.Delete(pending, i, i+1)
			continue
		}
		// every move is waiting, so following what each waits on comes round again,
		// and the one it comes round to is in the circle
		i, seen := 0, map[int]bool{}
		for ; !seen[i]; i = waiting(pending, pending[i]) {
			seen[i] = true
		}
		aside := c.aside(pending[i].From, named)
		out = append(out, Move{From: pending[i].From, To: aside})
		named = append(named, Move{From: pending[i].From, To: aside})
		pending[i].From = aside
	}
	return out
}

// waiting is the index of a move still to go from where one is going, or -1.
func waiting(pending []Move, m Move) int {
	return slices.IndexFunc(pending, func(other Move) bool {
		return other != m && overlaps(m.To, other.From)
	})
}

// aside is a name beside a file or directory that nothing is kept under and no move
// names.
func (c *Change) aside(relative string, named []Move) string {
	for n := 1; ; n++ {
		candidate := fmt.Sprintf("%s.moving-%d", relative, n)
		if slices.ContainsFunc(named, func(m Move) bool {
			return overlaps(candidate, m.From) || overlaps(candidate, m.To)
		}) {
			continue
		}
		if _, err := os.Lstat(filepath.Join(c.root, filepath.FromSlash(candidate))); os.IsNotExist(err) {
			return candidate
		}
	}
}

// overlaps reports whether two paths name one place, or one of them holds the other: a
// file arriving inside a directory still to be carried off would be carried off with it.
func overlaps(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// restore is what the directory held before the change started, kept so that a failure
// can put it back.
type restore struct {
	root    string
	prior   map[string][]byte // files that existed, and what was in them
	absent  []string          // files that did not, to be taken away again
	made    []string          // directories that did not, likewise
	emptied []string          // directories that did and were left holding nothing
	moved   []Move            // what has been carried to a new name, to be carried back
}

func (r *restore) path(relative string) string {
	return filepath.Join(r.root, filepath.FromSlash(relative))
}

func (r *restore) remember(relative string) error {
	if _, seen := r.prior[relative]; seen || slices.Contains(r.absent, relative) {
		return nil
	}
	content, err := os.ReadFile(r.path(relative))
	switch {
	case os.IsNotExist(err):
		r.absent = append(r.absent, relative)
	case err != nil:
		return fmt.Errorf("%s: %w", relative, err)
	default:
		r.prior[relative] = content
	}
	return nil
}

// mkdir makes the directories a file needs, noting the ones that were not already
// there. An empty folder left behind is a trace, and a transaction that rolled back is
// supposed to leave none.
func (r *restore) mkdir(relative string) error {
	dir := filepath.Dir(r.path(relative))
	for d := dir; ; {
		if _, err := os.Stat(d); err == nil {
			break
		}
		r.made = append(r.made, d)
		parent := filepath.Dir(d)
		if parent == d {
			break
		}
		d = parent
	}
	return os.MkdirAll(dir, 0o755)
}

// prune takes away the directories that carrying a file off or removing it left empty,
// climbing until it reaches one that still holds something.
func (r *restore) prune(relative string) {
	for dir := path.Dir(relative); dir != "." && dir != "/"; dir = path.Dir(dir) {
		if os.Remove(r.path(dir)) != nil {
			return
		}
		r.emptied = append(r.emptied, dir)
	}
}

// rollback returns the cause it was given, joined with anything that went wrong putting
// the directory back — which the caller needs to hear about above all, since it means
// the corpus is in neither state.
func (r *restore) rollback(cause error) error {
	problems := []error{cause}
	// the emptied directories first: everything below is putting things back into them
	for _, dir := range r.emptied {
		if err := os.MkdirAll(r.path(dir), 0o755); err != nil {
			problems = append(problems, fmt.Errorf("could not make %s again: %w", dir, err))
		}
	}
	for relative, content := range r.prior {
		if err := os.WriteFile(r.path(relative), content, 0o644); err != nil {
			problems = append(problems, fmt.Errorf("could not put %s back: %w", relative, err))
		}
	}
	for _, relative := range r.absent {
		if err := os.Remove(r.path(relative)); err != nil && !os.IsNotExist(err) {
			problems = append(problems, fmt.Errorf("could not take %s away again: %w", relative, err))
		}
	}
	// after the files, so that what was written into a moved directory is put back
	// before the directory goes home under its old name
	for i := len(r.moved) - 1; i >= 0; i-- {
		moved := r.moved[i]
		if err := os.Rename(r.path(moved.To), r.path(moved.From)); err != nil {
			problems = append(problems,
				fmt.Errorf("could not call %s %s again: %w", moved.To, moved.From, err))
		}
	}
	// deepest first, and a directory someone else has put something in stays
	made := slices.Clone(r.made)
	slices.SortFunc(made, func(a, b string) int { return len(b) - len(a) })
	for _, dir := range made {
		_ = os.Remove(dir)
	}
	return errors.Join(problems...)
}
