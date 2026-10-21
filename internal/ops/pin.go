package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Base pinning is what keeps a hypothesis saying next year what it said when it was
// written.
//
// A field path is positional. events[1] is the death today and is the marriage the
// moment somebody finds a record of one that happened earlier, so an edit resolved
// against a later corpus lands wherever its path points by then — quietly, and
// looking exactly like an edit that did what it said. Origins therefore does not
// resolve them there. It reconstructs the state the hypothesis pinned, carries the
// edits out where their paths still mean what whoever wrote them meant, and turns the
// result into a patch. What crosses from then to now is text with context around it,
// so anything that moved underneath comes out as a conflict rather than as an answer
// nobody checked.

// Unpinned is a hypothesis with no earlier state to be read at.
//
// There are three ways to arrive at one and they are the whole of the list: a
// hypothesis pinning no commit, one pinning a commit this repository does not have,
// and one whose commit holds no corpus origins can read. None of the three is a fault
// in the hypothesis — they are states a repository is allowed to be in — so what to
// make of one is the caller's to decide rather than this package's. Accepting refuses;
// supposing reads the edits against the corpus in front of it and says so.
type Unpinned struct {
	Slug model.Slug
	Base string // as the hypothesis spells it, and empty where it spells none
	Why  string // the clause that goes after "because"
	err  error
}

func (u *Unpinned) Error() string {
	said := fmt.Sprintf("%s cannot be read at the state it was written against, because %s",
		u.Slug, u.Why)
	if u.err == nil {
		return said
	}
	return said + ": " + u.err.Error()
}

func (u *Unpinned) Unwrap() error { return u.err }

// Pinned is what a hypothesis comes to at the commit it was written against.
//
// What the edits were about is settled here along with what they do, because the base
// is the one state where a positional path means what whoever wrote it meant. Worked
// out afterwards, against a corpus the change has already landed in, events[0] is
// whatever event stands first by then and the commit accepting a correction to a
// death would say it touched a baptism.
type Pinned struct {
	Base     string             // the commit its edits were carried out at, as the hypothesis spells it
	Files    []string           // what they touch there, where the corpus they go onto keeps it
	Patch    []byte             // and what they make of it
	Entities []model.EntityRef  // the records they are about, the renamed and the deleted included
	Renamed  []history.Renaming // the names they give up, in the order they give them up
	Events   []string           // and the kinds of event they reach, where they reach any
}

// Pin reconstructs the state a hypothesis's edits were written against, carries them
// out there, and hands back the patch they amount to.
//
// Nothing here judges. Whether the corpus still holds together once the hypothesis is
// in it is the question origins check --with exists to put, and a hypothesis that
// breaks something is that question's answer rather than a failure to arrive at one;
// refusing here would leave a researcher with a hypothesis origins declines to show
// them. What is refused is a hypothesis nothing can be made of at all, and there are
// two kinds. A hypothesis with no earlier state to be read at comes back as an
// Unpinned, which is a fact about the repository and not about the hypothesis, and
// which the caller is free to answer some other way; a hypothesis whose edits will not
// carry out even where they were written is wrong wherever it is read, and comes back
// as an ordinary error.
//
// The base is brought forward to the names its records go by now before anything is
// carried out there. A slug is not a fixed address, so a record renamed since is the
// same record with the same facts; but a patch names files, and git follows no file to
// where it has gone since. Carried out at the base brought on that way, the edits name
// every record where onto keeps it, a record being kept wherever its name says. What
// the patch has to find in onto is then there as it was at the base, and where it is
// not — a record deleted or merged since, one renamed where the history does not say
// so, a name somebody has taken since — the answer is a Drifted naming it before git
// is asked anything.
//
// onto is read and not touched, and nor is anything else in the corpus. The base is
// checked out somewhere else, and goes again whatever happens.
func Pin(repo *history.Repo, h *model.Hypothesis, onto *store.Corpus) (*Pinned, error) {
	if h.Base == "" {
		return nil, &Unpinned{Slug: h.Slug, Why: "it pins no commit"}
	}
	switch reachable, err := repo.Has(h.Base); {
	case err != nil:
		return nil, err
	case !reachable:
		return nil, &Unpinned{Slug: h.Slug, Base: h.Base,
			Why: "this repository does not have " + h.Base}
	}
	dir, err := os.MkdirTemp("", "origins-")
	if err != nil {
		return nil, err
	}
	where := filepath.Join(dir, "base")
	defer func() {
		_ = repo.Discard(where)
		_ = os.RemoveAll(dir)
	}()

	root, err := repo.AsOf(h.Base, where)
	if err != nil {
		return nil, fmt.Errorf("%s was written against %s, which origins could not read back: %w",
			h.Slug, h.Base, err)
	}
	base, problems := store.Load(root)
	if problems != nil {
		return nil, &Unpinned{Slug: h.Slug, Base: h.Base,
			Why: h.Base + " holds no corpus origins can read", err: problems}
	}

	at, err := history.Open(root)
	if err != nil {
		return nil, err
	}
	var named []model.EntityRef
	for _, r := range records(base) {
		named = append(named, r.EntityRef)
	}
	fates, err := repo.Since(h.Base, named)
	if err != nil {
		return nil, fmt.Errorf("%s: what has become since %s of the records it held: %w", h.Slug, h.Base, err)
	}
	edits, drift := brought(h, fates)
	if drift != nil {
		return nil, drift
	}
	if base, err = forward(base, fates); err != nil {
		return nil, fmt.Errorf("%s: %s under the names its records go by now: %w", h.Slug, h.Base, err)
	}
	moves := followed(base.Corpus, fates)
	for _, edit := range h.Edits {
		for _, given := range gives(edit) {
			delete(moves, given.Slug)
		}
	}

	from, err := at.Tree()
	if err != nil {
		return nil, err
	}

	held := records(base)
	pinned := &Pinned{Base: h.Base}
	tx := Begin(base)
	for i, edit := range edits {
		if edit == nil {
			continue
		}
		entry, err := tx.Perform(edit)
		if err != nil {
			return nil, fmt.Errorf("%s, edit %d of %d (%s): %w", h.Slug, i+1, len(edits), edit.Op(), err)
		}
		pinned.Entities = also(pinned.Entities, entry.Entities...)
		pinned.Renamed = append(pinned.Renamed, handed(edit, entry)...)
		pinned.Events = also(pinned.Events, entry.Events...)
	}
	// what the edits wrote by value names records as they were called at the base
	references(tx.Corpus(), func(slug *model.Slug) {
		if now, moving := moves[*slug]; moving {
			*slug = now
		}
	})
	change, err := tx.staged()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", h.Slug, err)
	}
	carried, drift := drifted(h, held, base.Root, onto, change)
	if drift != nil {
		return nil, drift
	}
	if err := change.Apply(); err != nil {
		return nil, fmt.Errorf("%s: %w", h.Slug, err)
	}
	pinned.Files = carried
	if pinned.Patch, err = at.Patch(from, pinned.Files); err != nil {
		return nil, fmt.Errorf("%s: %w", h.Slug, err)
	}
	return pinned, nil
}

// drifted is what the edits touch that onto does not hold as the base held it, and
// what is left to carry once what onto already has is taken out. The patch reads every
// file it changes at the base and has to find it again at the far end, and has to find
// nothing where it makes one, so a file there at one end and not the other is a change
// git would refuse with a message about its index.
//
// Two of those are the change having happened already, and there is nothing to carry of
// either. A file the edits take away that onto does not have is one. A record the edits
// make that onto already has is the other, provided every line the edits would write
// is in it and in that order: a record is written one fact to a line, so that is every
// fact the edits would have made it with still standing, and what somebody has added
// to it since is theirs to add. Where one of those facts has gone or been changed, the
// record is one somebody has changed since.
//
// Where a whole record has gone — renamed by a commit that does not say so, which is
// the one road to a new name the history cannot follow — the record is named rather
// than the file, that being what whoever wrote the hypothesis was writing about.
func drifted(h *model.Hypothesis, held []record, base string, onto *store.Corpus, change *Change) (
	[]string, *Drifted) {
	var (
		drift   Drifted
		carried []string
	)
	for _, relative := range change.Paths() {
		then, now := exists(base, relative), exists(onto.Root, relative)
		switch made, making := change.write[relative]; {
		case then == now:
		case !then && making && holds(onto.Root, relative, made):
			continue
		case then && slices.Contains(change.remove, relative):
			continue
		default:
			if gone, ok := owner(held, relative); then && ok && onto.KindOf(gone.Slug) != gone.Kind {
				drift.Gone = also(drift.Gone, gone)
			} else {
				drift.Files = append(drift.Files, relative)
			}
			continue
		}
		carried = append(carried, relative)
	}
	if drift.Gone == nil && drift.Files == nil {
		return carried, nil
	}
	drift.Slug, drift.Base = h.Slug, h.Base
	return nil, &drift
}

// holds reports whether a file has in it every line of what would have been written
// there, in the same order.
func holds(root, relative string, made []byte) bool {
	found, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		return false
	}
	have := strings.Split(string(found), "\n")
	for _, line := range strings.Split(string(made), "\n") {
		at := slices.Index(have, line)
		if at < 0 {
			return false
		}
		have = have[at+1:]
	}
	return true
}

func exists(root, relative string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
	return err == nil
}

// record is a record of the corpus and what it occupies there. They are taken before
// the edits are carried out, which is the only time a record the edits take away can
// still be asked what it was.
type record struct {
	model.EntityRef
	at string
}

func records(corpus *store.Corpus) []record {
	var held []record
	for _, slug := range corpus.Slugs() {
		kept, _ := corpus.Path(slug)
		kind := corpus.Corpus.KindOf(slug)
		held = append(held, record{model.EntityRef{Kind: kind, Slug: slug}, store.Location(kept)})
	}
	return held
}

// owner is the record a file belongs to: the record kept in it, or the one it is
// attached to.
func owner(held []record, relative string) (model.EntityRef, bool) {
	for _, r := range held {
		if relative == r.at || strings.HasPrefix(relative, r.at+"/") {
			return r.EntityRef, true
		}
	}
	return model.EntityRef{}, false
}

// also adds what is not already there, keeping the order the edits named things in.
// A hypothesis correcting two dates on one person is about one person, and a commit
// naming them twice says nothing it did not say once.
func also[T comparable](have []T, more ...T) []T {
	for _, one := range more {
		if !slices.Contains(have, one) {
			have = append(have, one)
		}
	}
	return have
}
