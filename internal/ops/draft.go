package ops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// A draft is supposing done a change at a time: edits made from the pages and kept
// rather than recorded, until they are kept as one hypothesis or thrown away. It is a
// hypothesis in all but name and record — a commit and the edits pinned to it — so it
// is carried onto the corpus as a hypothesis is, and keeping it writes those edits into
// one without restating anything.
//
// It is kept in git's own directory beside the lock and never in the corpus, so it is
// in no commit and no clone, and a server started again takes up where it was.

// DraftFile is the name of the file a draft is kept in, in git's directory.
const DraftFile = "origins-draft.yaml"

// draftSlug is what a draft is called where a hypothesis would be named.
const draftSlug model.Slug = "the draft"

// Draft is the commit a draft was begun on, or brought forward to, and its edits in
// the order they were made.
type Draft struct {
	Base  string
	Edits []model.Edit
}

// Undrafted is a draft asked about where none has been begun.
type Undrafted struct{}

func (Undrafted) Error() string { return "no draft has been begun" }

// ReadDraft is the draft kept for the corpus at dir, or Undrafted.
func ReadDraft(dir string) (*Draft, error) {
	repo, err := history.Open(dir)
	if err != nil {
		return nil, err
	}
	return readDraft(repo)
}

func readDraft(repo *history.Repo) (*Draft, error) {
	at, err := repo.GitPath(DraftFile)
	if err != nil {
		return nil, err
	}
	written, err := os.ReadFile(at)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, Undrafted{}
	case err != nil:
		return nil, err
	}
	base, edits, err := store.ParseDraft(written)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", at, err)
	}
	return &Draft{Base: base, Edits: edits}, nil
}

// write keeps the draft, or takes it away where it has no edits left.
func (d *Draft) write(repo *history.Repo) error {
	at, err := repo.GitPath(DraftFile)
	if err != nil {
		return err
	}
	if len(d.Edits) == 0 {
		return dropped(at)
	}
	written, err := store.EmitDraft(d.Base, d.Edits)
	if err != nil {
		return err
	}
	return os.WriteFile(at, written, 0o644)
}

func dropped(at string) error {
	if err := os.Remove(at); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// hypothesis is the draft as the hypothesis it would be kept as.
func (d *Draft) hypothesis(slug model.Slug) *model.Hypothesis {
	return &model.Hypothesis{Slug: slug, Status: model.Open, Base: d.Base, Edits: d.Edits}
}

// Drafted is a corpus with a draft in it, and how far the history has gone on since
// the draft's commit.
type Drafted struct {
	Corpus *store.Corpus
	Draft  *Draft
	Since  int

	trial *Trial
}

// Close takes away the copy the draft was carried onto, where it needed one.
func (d *Drafted) Close() error {
	if d == nil || d.trial == nil {
		return nil
	}
	return d.trial.Close()
}

// In is the corpus with the draft in it. While the draft's commit is still the last
// the edits are carried out on a copy in memory, where they mean what they meant when
// they were made; once the history has gone on, the draft is carried onto the corpus
// as --with carries a hypothesis, from its commit, and one somebody has since changed
// the ground under comes back Drifted.
func (d *Draft) In(corpus *store.Corpus) (*Drafted, error) {
	repo, err := history.Open(corpus.Root)
	if err != nil {
		return nil, err
	}
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	if head == d.Base {
		copied, err := d.carried(corpus)
		if err != nil {
			return nil, err
		}
		return &Drafted{Corpus: copied, Draft: d}, nil
	}
	since, err := repo.Beyond(d.Base)
	if err != nil {
		return nil, fmt.Errorf("the draft was begun on %s, which this history does not lead from: %w", d.Base, err)
	}
	trial, err := copied(corpus)
	if err != nil {
		return nil, err
	}
	if _, err := trial.take(d.hypothesis(draftSlug)); err != nil {
		return nil, errors.Join(err, trial.Close())
	}
	loaded, problems := store.Load(trial.root)
	if problems != nil {
		return nil, errors.Join(problems, trial.Close())
	}
	trial.Corpus = loaded
	return &Drafted{Corpus: loaded, Draft: d, Since: since, trial: trial}, nil
}

// carried is the corpus with the draft's edits carried out on a copy in memory.
func (d *Draft) carried(corpus *store.Corpus) (*store.Corpus, error) {
	copied := corpus.Copy()
	for i, edit := range d.Edits {
		if _, _, err := performed(copied.Corpus, edit); err != nil {
			return nil, fmt.Errorf("the draft, edit %d of %d (%s): %w", i+1, len(d.Edits), edit.Op(), err)
		}
	}
	copied.Refile()
	return copied, nil
}

// DraftEdit takes an edit into the draft kept for the corpus at dir, beginning one at
// the last commit where none has been begun. An edit is taken in only once the draft
// with it has been carried out, so that a draft is always one that can be, and it is
// pinned by Seen to the record as the draft showed it.
//
// A draft begun on a commit the history has since gone on from is brought forward to
// the last one first, stated again as the difference it makes there: an edit made now
// was made looking at the corpus as it stands, and its path means what it means there
// and not at the commit the draft was begun on.
func DraftEdit(dir string, edit model.Edit, asked Asked) (draft *Draft, err error) {
	repo, corpus, release, err := opened(dir, asked.Command)
	if err != nil {
		return nil, err
	}
	defer func() { err = released(err, release) }()

	head, err := repo.Head()
	switch {
	case err != nil:
		return nil, err
	case head == "":
		return nil, errors.New("nothing has been recorded yet, so there is nothing for a draft to depart from")
	}
	draft, err = readDraft(repo)
	switch {
	case errors.As(err, new(Undrafted)):
		draft = &Draft{Base: head}
	case err != nil:
		return nil, err
	case draft.Base != head:
		if draft, err = forwarded(draft, corpus, head); err != nil {
			return nil, err
		}
	}
	copied, err := draft.carried(corpus)
	if err != nil {
		return nil, err
	}
	if err := asked.holds(copied, edit.Subjects()[0]); err != nil {
		return nil, err
	}
	if _, _, err := performed(copied.Corpus, edit); err != nil {
		return nil, err
	}
	draft.Edits = append(draft.Edits, edit)
	return draft, draft.write(repo)
}

// forwarded is the draft stated again at head, as the difference it makes to the
// corpus there.
func forwarded(draft *Draft, corpus *store.Corpus, head string) (*Draft, error) {
	drafted, err := draft.In(corpus)
	if err != nil {
		return nil, err
	}
	defer drafted.Close()
	edits, err := Propose(corpus.Corpus, drafted.Corpus.Corpus)
	if err != nil {
		return nil, fmt.Errorf("the draft was begun on %s and cannot be stated again at %s: %w", draft.Base, head, err)
	}
	return &Draft{Base: head, Edits: edits}, nil
}

// DraftPop takes the last edit back out of the draft kept for the corpus at dir, and
// the draft away with it where it was the only one.
func DraftPop(dir, command string) (draft *Draft, err error) {
	repo, release, err := locked(dir, command)
	if err != nil {
		return nil, err
	}
	defer func() { err = released(err, release) }()
	if draft, err = readDraft(repo); err != nil {
		return nil, err
	}
	draft.Edits = draft.Edits[:len(draft.Edits)-1]
	return draft, draft.write(repo)
}

// DraftDiscard throws the draft kept for the corpus at dir away.
func DraftDiscard(dir, command string) (err error) {
	repo, release, err := locked(dir, command)
	if err != nil {
		return err
	}
	defer func() { err = released(err, release) }()
	if _, err := readDraft(repo); err != nil {
		return err
	}
	return (&Draft{}).write(repo)
}

// Keeping is what a draft is kept as besides its edits: the hypothesis's name, the
// question it puts and the argument for it, and what it rests on.
type Keeping struct {
	Slug        model.Slug
	Question    string
	Argument    string
	Confidence  model.Confidence
	Supports    []model.CitationRef
	Contradicts []model.CitationRef
}

// Keep writes the draft kept for the corpus at dir as an open hypothesis pinned to the
// draft's commit, records it as one commit, takes the draft away, and hands back the
// hypothesis it wrote. A draft that no
// longer carries onto the corpus is not kept, since a hypothesis that cannot be
// supposed is no question anybody can put.
func Keep(dir string, keeping Keeping, asked Asked) (entry history.Commit, h *model.Hypothesis, sha string, err error) {
	repo, corpus, release, err := opened(dir, asked.Command)
	if err != nil {
		return history.Commit{}, nil, "", err
	}
	defer func() { err = released(err, release) }()

	draft, err := readDraft(repo)
	if err != nil {
		return history.Commit{}, nil, "", err
	}
	if corpus.Taken(keeping.Slug) {
		return history.Commit{}, nil, "", fmt.Errorf("the corpus already has something called %s", keeping.Slug)
	}
	drafted, err := draft.In(corpus)
	if err != nil {
		return history.Commit{}, nil, "", err
	}
	if err := drafted.Close(); err != nil {
		return history.Commit{}, nil, "", err
	}

	h = draft.hypothesis(keeping.Slug)
	h.Confidence = keeping.Confidence
	h.Supports = keeping.Supports
	h.Contradicts = keeping.Contradicts
	h.Body = "# " + keeping.Question
	if keeping.Argument != "" {
		h.Body += "\n\n" + keeping.Argument
	}
	entry = history.Commit{
		Subject:    "Propose " + string(keeping.Slug),
		Op:         "hypothesis.propose",
		Entities:   []model.EntityRef{{Kind: model.KindHypothesis, Slug: keeping.Slug}},
		Hypothesis: keeping.Slug,
	}
	asked.describe(&entry)
	tx := Begin(corpus)
	tx.Corpus().Hypotheses[keeping.Slug] = h
	if asked.DryRun {
		_, err = tx.Stage()
		return entry, h, "", err
	}
	_, sha, err = tx.Commit(repo, entry)
	if sha == "" {
		return history.Commit{}, nil, "", err
	}
	return entry, h, sha, errors.Join(err, (&Draft{}).write(repo))
}
