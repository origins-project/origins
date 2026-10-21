package ops

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Supposing a hypothesis true is the one question a corpus cannot be asked any other
// way. It is always in exactly one state, that being what lets anything traverse it,
// so the only place an alternative can be put to it is somewhere else — and putting
// it somewhere else is what makes doubt executable rather than merely filed.
//
// What comes out is the corpus in front of the researcher with the hypothesis in it,
// built in a copy made for the purpose. Not the corpus as it stood at the base: the
// question is whether the tree holds together now, and an answer about a state from
// two years ago answers nobody. Nor the edits re-resolved against today, which is the
// one thing base pinning exists to stop. They are carried out where they were
// written, and git's three-way merge brings the result forward — the same road
// accepting the hypothesis will take, so what --with shows is what accepting would
// do.

// Reading is a hypothesis and the state its edits were carried out at.
//
// It is reported every time rather than only when something went wrong. The two
// readings are different answers to the same question and the corpus that comes out
// of them looks identical, so a researcher who is not told cannot tell.
type Reading struct {
	Slug model.Slug
	Base string // the commit it was read at, as the hypothesis spells it
	Why  string // and where there was no state to read it at, the clause saying why
}

// Pinned says whether the hypothesis was read where it was written.
func (r Reading) Pinned() bool { return r.Base != "" }

func (r Reading) String() string {
	if r.Pinned() {
		return fmt.Sprintf("%s read at %s", r.Slug, r.Base)
	}
	return fmt.Sprintf("%s read against the corpus in front of you, because %s", r.Slug, r.Why)
}

// Drifted is a hypothesis git could not carry onto the corpus as it now stands,
// somebody having changed since what the hypothesis changes.
//
// It is neither a fault in the hypothesis nor a finding about the corpus. It is the
// two of them disagreeing about one fact, which is an answer to the question --with
// puts and not a failure to arrive at one — so the only honest thing to do with it is
// to say where the disagreement is and stop.
type Drifted struct {
	Slug  model.Slug
	Base  string
	Files []string                            // where somebody has changed since what it changes
	Gone  []model.EntityRef                   // and the records it changes that the corpus no longer has
	Into  map[model.EntityRef]model.EntityRef // and of those, the ones merged into another
}

func (d *Drifted) Error() string {
	return fmt.Sprintf("%s was written against %s and no longer applies: %s",
		d.Slug, d.Base, strings.Join(d.Reasons(), "; "))
}

// Reasons says where the hypothesis and the corpus part company, a clause for each.
func (d *Drifted) Reasons() []string {
	var (
		why   []string
		names []string
	)
	for _, gone := range d.Gone {
		if into, merged := d.Into[gone]; merged {
			why = append(why, fmt.Sprintf("it changes %s, which has since been merged into %s", gone, into))
		} else {
			names = append(names, gone.String())
		}
	}
	if len(names) > 0 {
		why = append(why, "it changes "+strings.Join(names, ", ")+
			", and nothing in this corpus goes by that name any more")
	}
	if len(d.Files) > 0 {
		why = append(why, "somebody has changed what it changes, in "+strings.Join(d.Files, ", "))
	}
	return why
}

// Trial is a corpus with hypotheses in it, and how each of them came to be there.
//
// It is a corpus on disk in a directory of its own, because a hypothesis crosses from
// its base to now as a patch and a patch is applied to files. Close takes the
// directory away, and nothing in the corpus the researcher keeps is written to at any
// point.
type Trial struct {
	Corpus *store.Corpus
	Read   []Reading

	dir  string        // everything this trial made, to be taken away whole
	tree string        // the checkout the copy was made in, where there is one
	root string        // the copy itself
	repo *history.Repo // where the hypotheses are read back from
	at   *history.Repo // the same repository, pointed at the copy
	why  string        // and where there is none, what is to be said instead
}

// Close takes away the copy the trial was built in and the checkout it read the base
// from.
func (t *Trial) Close() error {
	var discarded error
	if t.repo != nil && t.tree != "" {
		discarded = t.repo.Discard(t.tree)
	}
	return errors.Join(discarded, os.RemoveAll(t.dir))
}

// Suppose builds the corpus that would stand if the named hypotheses were true, in
// the order they are named.
//
// Several at once is the compatibility question: two hypotheses that each hold on
// their own and cannot both be true are two readings of one fact, and the second one
// carried onto the first is where that shows.
func Suppose(corpus *store.Corpus, named []model.Slug) (*Trial, error) {
	chosen := make([]*model.Hypothesis, len(named))
	for i, slug := range named {
		held, err := proposed(corpus, slug)
		if err != nil {
			return nil, err
		}
		chosen[i] = held
	}
	trial, err := copied(corpus)
	if err != nil {
		return nil, err
	}
	for _, h := range chosen {
		read, err := trial.take(h)
		if err != nil {
			return nil, errors.Join(err, trial.Close())
		}
		trial.Read = append(trial.Read, read)
	}
	loaded, problems := store.Load(trial.root)
	if problems != nil {
		return nil, errors.Join(problems, trial.Close())
	}
	trial.Corpus = loaded
	return trial, nil
}

// copied puts the corpus somewhere it can be changed, and settles what can be done to
// it there.
//
// A checkout of the repository wherever there is one, because carrying a patch from
// the commit it was made at is git's three-way merge and that needs the objects the
// patch names — a directory outside a repository has none of them. Where there is no
// repository there is a plain copy and nothing else, and every hypothesis is then
// read against the corpus rather than at its base.
func copied(corpus *store.Corpus) (*Trial, error) {
	dir, err := os.MkdirTemp("", "origins-")
	if err != nil {
		return nil, err
	}
	trial := &Trial{dir: dir, root: filepath.Join(dir, "corpus")}
	if err := trial.stand(corpus); err != nil {
		return nil, errors.Join(err, trial.Close())
	}
	// a corpus git has never been told about is not in the checkout: one nobody has
	// recorded yet, and one an ignore rule keeps out, are both directories the copy
	// has to make for itself
	if err := os.MkdirAll(trial.root, 0o777); err != nil {
		return nil, errors.Join(err, trial.Close())
	}
	if err := mirror(corpus.Root, trial.root); err != nil {
		return nil, errors.Join(err, trial.Close())
	}
	if trial.repo != nil {
		if trial.at, err = history.Open(trial.root); err != nil {
			return nil, errors.Join(err, trial.Close())
		}
	}
	return trial, nil
}

// stand settles where the copy is going to be made, which is the same question as
// what can be done to it once it is there.
func (t *Trial) stand(corpus *store.Corpus) error {
	repo, absent := history.Open(corpus.Root)
	if absent != nil {
		t.why = "this corpus is in no repository"
		return nil
	}
	head, err := repo.Head()
	if err != nil {
		return err
	}
	if head == "" {
		t.why = "nothing in this corpus has been recorded yet"
		return nil
	}
	t.tree = filepath.Join(t.dir, "tree")
	if t.root, err = repo.AsOf(head, t.tree); err != nil {
		return err
	}
	t.repo = repo
	return nil
}

// take reads one hypothesis into the copy, at its base where there is a state to read
// it at and against the copy itself where there is not.
func (t *Trial) take(h *model.Hypothesis) (Reading, error) {
	pinned, err := t.pin(h)
	var elsewhere *Unpinned
	switch {
	case errors.As(err, &elsewhere):
		if err := t.byHand(h); err != nil {
			return Reading{}, err
		}
		return Reading{Slug: h.Slug, Why: elsewhere.Why}, nil
	case err != nil:
		return Reading{}, err
	}
	conflicts, err := t.at.Carry(pinned.Patch)
	if err != nil {
		return Reading{}, fmt.Errorf("%s could not be carried from %s: %w", h.Slug, pinned.Base, err)
	}
	if len(conflicts) > 0 {
		return Reading{}, &Drifted{Slug: h.Slug, Base: pinned.Base, Files: conflicts}
	}
	return Reading{Slug: h.Slug, Base: pinned.Base}, nil
}

func (t *Trial) pin(h *model.Hypothesis) (*Pinned, error) {
	if t.repo == nil {
		return nil, &Unpinned{Slug: h.Slug, Base: h.Base, Why: t.why}
	}
	// what will not load here will not load once every hypothesis is in either, and
	// is reported then
	onto, _ := store.Load(t.root)
	return Pin(t.repo, h, onto)
}

// byHand carries a hypothesis out against the corpus in front of the researcher,
// which is the only other state there is when the one it was written against cannot
// be reconstructed.
//
// This is the reading base pinning exists to avoid: a path is positional, so an edit
// whose subject has moved since lands on whatever its path points at today, looking
// exactly like an edit that did what it said. Nothing here can tell whether anything
// has moved, which is why what was done is reported rather than quietly done.
//
// The edits are staged without the contradiction test every other way into a corpus
// runs, for the reason pinning stages them the same way: whether the corpus holds
// together with the hypothesis in it is the question being put, and refusing to build
// the corpus that answers it would be declining to answer.
func (t *Trial) byHand(h *model.Hypothesis) error {
	corpus, problems := store.Load(t.root)
	if problems != nil {
		return problems
	}
	tx := Begin(corpus)
	for i, edit := range h.Edits {
		if _, err := tx.Perform(edit); err != nil {
			return fmt.Errorf("%s, edit %d of %d (%s): %w", h.Slug, i+1, len(h.Edits), edit.Op(), err)
		}
	}
	change, err := tx.staged()
	if err != nil {
		return fmt.Errorf("%s: %w", h.Slug, err)
	}
	return change.Apply()
}

const gitDir = ".git"

// mirror makes one directory hold exactly what another holds.
//
// Exactly rather than nearly: what --with answers about has to be the corpus in front
// of the researcher and not the last state of it anybody recorded, or check and check
// --with would be answering about two different corpora with only one of them saying
// so. A file somebody has edited without recording it, and one they have only just
// written, are both part of what the corpus says today.
//
// Git's own directory is left where it is at both ends. A corpus kept at the root of
// its repository has one beside it, and a checkout without it is no longer a
// checkout.
func mirror(from, to string) error {
	replaced, err := os.ReadDir(to)
	if err != nil {
		return err
	}
	for _, entry := range replaced {
		if entry.Name() == gitDir {
			continue
		}
		if err := os.RemoveAll(filepath.Join(to, entry.Name())); err != nil {
			return err
		}
	}
	return filepath.WalkDir(from, func(path string, entry fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case entry.Name() == gitDir && entry.IsDir():
			return fs.SkipDir
		case entry.Name() == gitDir:
			return nil
		}
		relative, err := filepath.Rel(from, path)
		if err != nil || relative == "." {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(to, relative), 0o777)
		}
		return duplicate(path, filepath.Join(to, relative))
	})
}

func duplicate(from, to string) error {
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(to)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
