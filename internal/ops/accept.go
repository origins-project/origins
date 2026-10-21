package ops

import (
	"errors"
	"fmt"
	"slices"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Accepting a hypothesis is the corpus changing its mind, and it is the only change
// origins makes that was written somewhere else and at some other time. Everything
// else a researcher types is about the corpus in front of them; this was written
// against a state that may be two years old, and what puts it in the right place is
// that it is carried out there and brought forward, which is the road --with
// rehearses.
//
// Rejecting one comes to almost nothing, and that is the point. The corpus is always
// the standing conclusion, so a proposal that fails leaves it exactly as it was and
// only the hypothesis changes — staying on record, with the argument and with the
// records that sank it.

// Decision is a hypothesis settled one way or the other and not yet recorded.
//
// Accepting one puts it in the files before anything has been recorded, a patch
// having nothing but files to be applied to, and it is the one place origins writes
// a corpus it has not read. What lands there is read straight back, re-emitted the
// way origins writes, and held to the contradictions it introduces before any of it
// becomes a commit; where any of that refuses, the files go back to the last state
// recorded. Discard is that, and it is also how a rehearsal ends.
type Decision struct {
	Tx    *Tx            // the change, judged against the corpus as it stood before it
	Entry history.Commit // and what the commit recording it will say

	Slug   model.Slug
	Status model.Status // where the hypothesis now stands
	Base   string       // where its edits were carried out, for one accepted
	Files  []string     // and what they changed, as the corpus names them

	repo    *history.Repo
	carried bool // whether it arrived as a patch, which is what is left to put back
}

// Land writes the change and records it as one commit.
//
// What recording it means differs between the two answers. A rejection is a change
// origins is about to make, and it names the one file it makes it in as every other
// change does. An acceptance is already in the files and origins wrote almost none
// of it, so what to record cannot be worked out from what origins wrote: the answer
// is the corpus, which holds the acceptance and nothing besides, this having refused
// to begin until there was nothing else outstanding in it.
func (d *Decision) Land() (*Change, string, error) {
	if d.carried {
		return d.Tx.Adopt(d.repo, d.Entry)
	}
	return d.Tx.Commit(d.repo, d.Entry)
}

// Discard puts the corpus back to the last state recorded, which for an acceptance
// is where it stood before the hypothesis was carried into it. A rejection has
// written nothing at all, and there is nothing to take back.
func (d *Decision) Discard() error {
	if !d.carried {
		return nil
	}
	return d.repo.Restore()
}

// Unrestored is a corpus a hypothesis was carried into and could not be taken back out
// of, which leaves it in neither the state recorded nor the one proposed.
type Unrestored struct{ Err error }

func (u *Unrestored) Error() string { return fmt.Sprintf("the corpus is in neither state: %v", u.Err) }

func (u *Unrestored) Unwrap() error { return u.Err }

// Settle records the decision, or for a dry run works out what it would record, and
// wherever nothing is recorded puts the corpus back where it was.
//
// A rehearsal ends the same way a refusal does. Accepting one writes the hypothesis
// into the files before anything is recorded, so saying what would happen means doing
// it and then undoing it, and the undoing is the same call either way.
func (d *Decision) Settle(dryRun bool) (*Change, string, error) {
	var (
		change *Change
		sha    string
		err    error
	)
	if dryRun {
		change, err = d.Tx.Stage()
	} else {
		change, sha, err = d.Land()
	}
	if (err != nil && sha == "") || dryRun {
		if put := d.Discard(); put != nil {
			return nil, "", errors.Join(err, &Unrestored{Err: put})
		}
	}
	if err != nil && sha == "" {
		return nil, "", err
	}
	return change, sha, err
}

// Decide settles a hypothesis in the corpus at dir, accepting it or rejecting it as
// answer says, and records that as one commit.
//
// What comes back besides the decision is what Settle hands back. A Drifted is the
// hypothesis and the corpus disagreeing about a fact, which is an answer about the two
// of them rather than a mistake in the asking.
func Decide(dir string, slug model.Slug, answer model.Status, asked Asked) (
	decision *Decision, change *Change, sha string, err error) {
	repo, corpus, release, err := opened(dir, asked.Command)
	if err != nil {
		return nil, nil, "", err
	}
	defer func() { err = released(err, release) }()

	if err = asked.holds(corpus, model.EntityRef{Kind: model.KindHypothesis, Slug: slug}); err != nil {
		return nil, nil, "", err
	}
	switch answer {
	case model.Accepted:
		decision, err = Accept(repo, corpus, slug)
	case model.Rejected:
		decision, err = Reject(repo, corpus, slug)
	default:
		err = fmt.Errorf("a hypothesis is settled by accepting or rejecting it, and %s is neither", answer)
	}
	if err != nil {
		return nil, nil, "", err
	}
	asked.describe(&decision.Entry)
	change, sha, err = decision.Settle(asked.DryRun)
	return decision, change, sha, err
}

// Accept takes a hypothesis into the corpus as what it now concludes.
//
// The edits are carried out at the commit the hypothesis pins and the result is
// brought forward by git's three-way merge, which is the whole of what base pinning
// was for: a positional path means there what whoever wrote it meant, and a fact
// somebody has changed since comes back as a conflict rather than as a correction
// quietly applied to the wrong line.
//
// A hypothesis with no state to be read at is refused rather than read against the
// corpus, which is where accepting parts company with supposing. Both readings
// produce a corpus; only one of them is a change to the standing conclusion, and the
// second-best answer is worth having to a question and not to that.
//
// The corpus has to hold nothing that is not recorded. Carrying a patch writes over
// the files and putting it back writes over them again, so an unrecorded edit is the
// work most easily lost here; and what the acceptance amounts to is read off what
// the corpus differs from the last commit in afterwards, which says the hypothesis
// alone only where it said nothing else to begin with.
func Accept(repo *history.Repo, corpus *store.Corpus, slug model.Slug) (*Decision, error) {
	h, err := proposed(corpus, slug)
	if err != nil {
		return nil, err
	}
	if h.Status == model.Accepted {
		return nil, fmt.Errorf("%s has already been accepted, and the corpus holds what it proposed", slug)
	}
	switch dirty, err := repo.Dirty(); {
	case err != nil:
		return nil, err
	case len(dirty) > 0:
		return nil, fmt.Errorf("%s, and accepting a hypothesis writes over the files: "+
			"origins adopt records what is there", outstanding(dirty))
	}
	pinned, err := Pin(repo, h, corpus)
	if err != nil {
		return nil, err
	}
	conflicts, err := repo.Carry(pinned.Patch)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%s could not be carried from %s: %w", slug, pinned.Base, err),
			repo.Restore())
	}
	if len(conflicts) > 0 {
		return nil, errors.Join(&Drifted{Slug: slug, Base: pinned.Base, Files: conflicts}, repo.Restore())
	}
	decision, err := accepted(repo, corpus, h, pinned)
	if err != nil {
		return nil, errors.Join(err, repo.Restore())
	}
	return decision, nil
}

// accepted is everything after the patch has landed, which is everything that can
// still refuse: a corpus the merge left unreadable, a hypothesis whose own record the
// edits took away, a git that will not answer for what changed.
func accepted(repo *history.Repo, corpus *store.Corpus, h *model.Hypothesis, pinned *Pinned) (
	*Decision, error) {
	now, problems := store.Load(corpus.Root)
	if problems != nil {
		return nil, fmt.Errorf("%s leaves a corpus origins cannot read: %w", h.Slug, problems)
	}
	tx := Adopting(now, corpus.Corpus)
	settled, err := tx.Perform(marking(h.Slug, model.Accepted))
	if err != nil {
		return nil, err
	}
	changed, err := repo.Dirty()
	if err != nil {
		return nil, err
	}
	return &Decision{
		Tx: tx,
		Entry: history.Commit{
			Subject:    "Accept " + string(h.Slug),
			Op:         "hypothesis.accept",
			Entities:   also(slices.Clone(pinned.Entities), settled.Entities...),
			Renamed:    pinned.Renamed,
			Events:     pinned.Events,
			Sources:    h.Supports,
			Hypothesis: h.Slug,
			ProposedBy: h.ProposedBy,
		},
		Slug:    h.Slug,
		Status:  model.Accepted,
		Base:    pinned.Base,
		Files:   changed,
		repo:    repo,
		carried: true,
	}, nil
}

// Reject closes a hypothesis, and writes nothing back.
//
// There is nothing to write back. The corpus is always the standing conclusion and a
// hypothesis exists only to propose something else, so one that fails leaves the
// corpus exactly as it was; what the commit records is that the question was put and
// answered. It rests on what the hypothesis contradicts — the records that say the
// corpus was right all along, which is what whence has to reach when one of them
// turns out to be unreliable.
//
// A hypothesis already accepted is refused. The corpus holds what it proposed, and
// calling it rejected now would leave the argument and the records disagreeing with
// nothing to say which of them is wrong. Taking the change back is what undo is for,
// and it has to happen first.
func Reject(repo *history.Repo, corpus *store.Corpus, slug model.Slug) (*Decision, error) {
	h, err := proposed(corpus, slug)
	if err != nil {
		return nil, err
	}
	if h.Status == model.Accepted {
		return nil, fmt.Errorf("%s has been accepted and the corpus holds what it proposed, so "+
			"rejecting it now would leave the two disagreeing: origins undo takes a change back", slug)
	}
	tx := Begin(corpus)
	settled, err := tx.Perform(marking(slug, model.Rejected))
	if err != nil {
		return nil, err
	}
	return &Decision{
		Tx: tx,
		Entry: history.Commit{
			Subject:    "Reject " + string(slug),
			Op:         "hypothesis.reject",
			Entities:   settled.Entities,
			Sources:    h.Contradicts,
			Hypothesis: slug,
			ProposedBy: h.ProposedBy,
		},
		Slug:   slug,
		Status: model.Rejected,
		repo:   repo,
	}, nil
}

// marking is the edit that puts a hypothesis's new status in its file: the whole of
// what rejecting one amounts to, and the last thing accepting one does. It goes
// through the vocabulary like every other change, a status being a field and there
// being one way to set a field.
func marking(slug model.Slug, status model.Status) model.Edit {
	return model.Set{
		Entity: model.EntityRef{Kind: model.KindHypothesis, Slug: slug},
		Path:   "status",
		Value:  store.Typed(string(status)),
	}
}

func proposed(corpus *store.Corpus, slug model.Slug) (*model.Hypothesis, error) {
	h, ours := corpus.Hypotheses[slug]
	if !ours {
		return nil, fmt.Errorf("this corpus has no hypothesis called %s", slug)
	}
	return h, nil
}

// outstanding is what stands between a corpus and a hypothesis being accepted into
// it. One file is worth naming and a dozen are not.
func outstanding(files []string) string {
	if len(files) == 1 {
		return files[0] + " has been changed and not recorded"
	}
	return fmt.Sprintf("%d files in this corpus have been changed and not recorded", len(files))
}
