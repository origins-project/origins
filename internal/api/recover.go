package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// The two answers for a corpus whose history has stopped being origins' own: one
// records a change somebody made in the files, the other says what has become of the
// commits and offers the three ways back.
//
// Everything else stops until the history is accounted for, and these two are the
// only ones that can still be asked. That is not a special case so much as the shape
// of the problem — a question about the past cannot also be the one that works out
// what the past is.

// Adoption is what adopting a change amounts to.
type Adoption struct {
	Commit   string   `json:"commit"`
	Subject  string   `json:"subject"`
	Files    []string `json:"files"`
	Emitted  []string `json:"emitted"`
	Subjects []string `json:"subjects"`
}

// Adopt records a change somebody made in the files rather than through origins, and
// hands back nothing at all where nothing has been.
//
// Editing a file is the escape hatch the format exists to leave open, and this is what
// closes the loop on it: the edit is read, re-emitted the way origins writes, judged
// against the state it was made from, and recorded as one commit saying it was made by
// hand. What it will not do is let a contradiction into the corpus, or leave the next
// real edit to that file producing a diff of the change and somebody's formatting at
// once.
func Adopt(asking Asking) (report *Adoption, err error) {
	repo, release, err := writing(asking.Root, "adopt")
	if err != nil {
		return nil, err
	}
	defer func() { err = joined(err, release()) }()
	corpus, problems := store.Load(asking.Root)
	if problems != nil {
		return nil, unreadable(problems, "origins will not record a corpus it could not read whole")
	}
	dirty, err := repo.Dirty()
	if err != nil || len(dirty) == 0 {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	before, done, err := asRecorded(repo, head, corpus.Corpus)
	defer done()
	if err != nil {
		return nil, err
	}

	entry := history.Commit{
		Subject:  adopted(dirty),
		Tool:     Tool,
		Op:       "manual.adopt",
		Entities: ops.Subjects(corpus, dirty),
		Renamed:  ops.Renamed(before, corpus.Corpus),
	}
	if asking.Message != "" {
		entry.Subject = asking.Message
	}
	staged, sha, err := record(ops.Adopting(corpus, before), repo, entry, asking.DryRun)
	if !landed(sha, err) {
		return nil, err
	}
	return &Adoption{
		Commit:   sha,
		Subject:  entry.Subject,
		Files:    dirty,
		Emitted:  staged.Written(),
		Subjects: written(entry.Entities),
	}, err
}

// record stages a change that is already in the files, writes whatever re-emitting it
// takes, and commits the lot. It is the shared middle of adopting and recovering by
// adopting, which differ in what they are recording and not in what recording it means.
func record(tx *ops.Tx, repo *history.Repo, entry history.Commit, dryRun bool) (*ops.Change, string, error) {
	if dryRun {
		staged, err := tx.Stage()
		return staged, "", err
	}
	return tx.Adopt(repo, entry)
}

// adopted is the subject line for a change nobody described. One file is worth naming
// and a dozen are not, which is the rule the rest of the output follows too.
func adopted(files []string) string {
	if len(files) == 1 {
		return "Adopt " + files[0] + ", edited by hand"
	}
	return fmt.Sprintf("Adopt %d files edited by hand", len(files))
}

// asRecorded reads a corpus back as some commit left it, and hands back the way to put
// the checkout away again.
//
// Where origins wrote no commit there is no earlier state, and the corpus in front of
// it is the truthful answer: origins found every contradiction in it and introduced
// none of them, so none of them is this change's to answer for. An empty corpus in
// their place would count a tree somebody has kept in git for ten years as having gone
// wrong in the moment origins first read it, and refuse the only way in — leaving the
// researcher it was written for with nothing to do but put right by hand, unaided, what
// origins is refusing to help them look at.
func asRecorded(repo *history.Repo, sha string, now *model.Corpus) (*model.Corpus, func(), error) {
	if sha == "" {
		return now, func() {}, nil
	}
	was, done, err := recordedAt(repo, sha)
	if err != nil {
		return nil, done, err
	}
	return was.Corpus, done, nil
}

// recordedAt reads a corpus back as a commit left it, where each record was kept as
// well as what it said, and hands back the way to put the checkout away again.
func recordedAt(repo *history.Repo, sha string) (*store.Corpus, func(), error) {
	nothing := func() {}
	dir, err := os.MkdirTemp("", "origins-")
	if err != nil {
		return nil, nothing, err
	}
	where := filepath.Join(dir, "recorded")
	done := func() {
		_ = repo.Discard(where)
		_ = os.RemoveAll(dir)
	}
	if where, err = repo.AsOf(sha, where); err != nil {
		done()
		return nil, nothing, err
	}
	was, problems := store.Load(where)
	if problems != nil {
		return nil, done, fmt.Errorf("%s is a commit whose corpus origins cannot read, so there is "+
			"nothing to judge a change against: %w", Short(sha), problems)
	}
	return was, done, nil
}

// Resolution is where origins stands with a history and what came of saying so.
type Resolution struct {
	Settled  bool     `json:"settled"`
	Ground   string   `json:"ground"`
	Anchor   string   `json:"anchor"`
	Foreign  []Entry  `json:"foreign"`
	Dirty    []string `json:"dirty"`
	Shared   bool     `json:"shared"`
	Advises  string   `json:"advises"`
	Commit   string   `json:"commit"`
	Problems []string `json:"problems"`

	// What a reader is told besides: where the history stands in a sentence, the
	// subject of the commit a resolution made and of the one origins last wrote, why
	// the ways are offered in their order and the ways themselves, which way was
	// taken, the commits a rewrite made, newest last, and the files a resolution
	// re-emitted.
	Standing      string   `json:"-"`
	Subject       string   `json:"-"`
	AnchorSubject string   `json:"-"`
	Because       string   `json:"-"`
	Ways          []Way    `json:"-"`
	Took          string   `json:"-"`
	Made          []string `json:"-"`
	Emitted       []string `json:"-"`
}

// Way is one of the three resolutions, as it is offered.
type Way struct{ Flag, What string }

// The three ways back, as Recover is told which to take.
const (
	ByAdopting  = "--adopt"
	ByRewriting = "--rewrite"
	ByResetting = "--reset"
)

// Recover says what has become of a corpus's history, and on being told which of the
// three ways back to take, takes it.
//
// It never picks for itself. Which resolution is right turns on what the commits are
// for and who else has seen them, and origins knows the second of those and not the
// first; what it can do is say which one it would take and why, which is the whole of
// the advice it has to give.
func Recover(root, way string) (report *Resolution, err error) {
	repo, err := history.Open(root)
	if err != nil {
		return nil, err
	}
	// a resolution writes, and saying what there is to resolve only reads
	if way != "" {
		release, err := repo.Lock("origins recover " + way)
		if err != nil {
			return nil, err
		}
		defer func() { err = joined(err, release()) }()
	}
	standing, err := repo.Survey()
	if err != nil {
		return nil, err
	}
	if way != "" && standing.Settled() {
		return nil, fmt.Errorf("origins wrote the whole of this history, so there is nothing for %s to resolve", way)
	}
	switch way {
	case ByAdopting:
		return keeping(repo, standing, root)
	case ByRewriting:
		return rewriting(repo, standing, root)
	case ByResetting:
		return unrecording(repo, standing)
	case "":
		return surveyed(repo, standing)
	}
	return nil, fmt.Errorf("%s is not a way to recover a history", way)
}

// ways is what can be done about a history, the one origins would take first, and the
// reason that is the order.
//
// The reason is always a fact and never a preference. Whether the commits have left
// this machine decides whether they may be rewritten, because rewriting one somebody
// else has is handing them a history that no longer joins onto theirs; whether origins
// wrote any of this history at all decides whether there is anywhere to rewrite onto.
// What the commits were for is the part origins cannot know, and the part it leaves.
func ways(s history.Standing, shared bool) (string, []Way) {
	var (
		keep      = Way{ByAdopting, "keep them as they are, and record that you have read them"}
		rewrite   = Way{ByRewriting, "replace them with commits that say what they are"}
		reset     = Way{ByResetting, "take them off, and leave what they said in the working tree"}
		have, are = "these commits have", "they"
	)
	if len(s.Foreign) == 1 {
		have, are = "this commit has", "it"
	}
	switch {
	case s.Anchor == nil:
		return "there is nothing of origins' own to rewrite onto and nothing to take it back to",
			[]Way{keep}
	case len(s.Foreign) == 0:
		// nothing is foreign, so the newest commit is origins' own and there is
		// nowhere for a reset to go: what moved was the record of where it had got
		// to, and moving it back is the whole of the repair
		return "every commit here is origins' own work, and what moved underneath it was only " +
			"this clone's record of where it had got to", []Way{{ByAdopting, "accept the history as it now stands"}}
	case shared:
		return fmt.Sprintf("%s left this machine, so %s must not be rewritten", have, are),
			[]Way{keep, reset}
	}
	return fmt.Sprintf("%s not left this machine, so %s can be rewritten", have, are),
		[]Way{rewrite, keep, reset}
}

// shareable asks whether the commits to be resolved have left this machine.
//
// The oldest is the one worth asking about, and asking about it is enough. A commit
// reachable from a remote-tracking ref has every one of its ancestors reachable too,
// so if the oldest has not been shared then nothing built on it has been either.
func shareable(repo *history.Repo, s history.Standing) (bool, error) {
	if len(s.Foreign) == 0 {
		return false, nil
	}
	return repo.Shared(s.Foreign[len(s.Foreign)-1].Sha)
}

func surveyed(repo *history.Repo, s history.Standing) (*Resolution, error) {
	shared, err := shareable(repo, s)
	if err != nil {
		return nil, err
	}
	report := describing(s, shared)
	report.Because, report.Ways = ways(s, shared)
	if !s.Settled() {
		report.Advises = report.Ways[0].Flag
	}
	return report, nil
}

func describing(s history.Standing, shared bool) *Resolution {
	report := &Resolution{
		Settled:  s.Settled(),
		Ground:   s.Ground.String(),
		Foreign:  []Entry{},
		Dirty:    s.Dirty,
		Shared:   shared,
		Problems: Flatten(s.Problems),
		Standing: s.Describe(),
	}
	if s.Anchor != nil {
		report.Anchor, report.AnchorSubject = s.Anchor.Sha, s.Anchor.Subject
	}
	for _, was := range s.Foreign {
		report.Foreign = append(report.Foreign, described(was))
	}
	if report.Dirty == nil {
		report.Dirty = []string{}
	}
	return report
}

// keeping records that origins has read a history it did not write and answers for it,
// without touching a commit of it.
//
// The commit it appends is what resolves the divergence, and it is appended even when
// nothing needed changing. That is the point of it: the trailer saying origins stands
// behind this state travels with the history, where a note stays on the machine that
// wrote it, and a clone somebody made this morning has to be able to tell that the
// work was reviewed.
func keeping(repo *history.Repo, s history.Standing, root string) (*Resolution, error) {
	if len(s.Dirty) > 0 {
		return nil, errors.New("the corpus also has changes nobody has recorded, and adopting would make " +
			"the two one commit: origins adopt takes those first")
	}
	corpus, problems := store.Load(root)
	if problems != nil {
		return nil, unreadable(problems, "origins will not adopt a corpus it could not read whole: "+
			"put the files right, or take the commits off with origins recover --reset")
	}
	// nothing foreign to account for means every commit is already origins' own and
	// there is nothing to say about them that they do not say themselves. What went
	// wrong is that this clone lost track of where it had got to, and pointing it at
	// HEAD again is the repair; a commit recording that would say nothing at all.
	if len(s.Foreign) == 0 {
		head, err := repo.Head()
		if err != nil {
			return nil, err
		}
		if err := repo.Mark(head); err != nil {
			return nil, err
		}
		report := describing(s, false)
		report.Settled, report.Commit, report.Took = true, head, ByAdopting
		return report, nil
	}

	anchor := ""
	if s.Anchor != nil {
		anchor = s.Anchor.Sha
	}
	before, done, err := asRecorded(repo, anchor, corpus.Corpus)
	defer done()
	if err != nil {
		return nil, err
	}

	touched, err := covered(repo, s.Foreign)
	if err != nil {
		return nil, err
	}
	entry := history.Commit{
		Subject:  kept(s.Foreign),
		Tool:     Tool,
		Op:       "manual.adopt",
		Entities: ops.Subjects(corpus, touched),
		Renamed:  ops.Renamed(before, corpus.Corpus),
	}

	// a history origins did not write is one nothing told how to check the corpus out,
	// and the commit taking it on is where the corpus starts saying so
	_, undeclare, err := store.DeclareEndings(corpus.Root)
	if err != nil {
		return nil, err
	}
	staged, sha, err := record(ops.Adopting(corpus, before), repo, entry, false)
	if !landed(sha, err) {
		if put := undeclare(); put != nil {
			err = errors.Join(err, fmt.Errorf("%s could not be put back as it was: %w", store.AttributesFile, put))
		}
		return nil, err
	}

	// the note is the local half, and it is written last: a note against a commit that
	// nothing was appended for would claim a review that is not in the history
	noted := fmt.Sprintf("Reviewed by origins and adopted in %s", Short(sha))
	for _, was := range s.Foreign {
		if note := repo.Note(was.Sha, noted); note != nil {
			err = joined(err, note)
			break
		}
	}

	report := describing(s, true)
	report.Commit, report.Settled, report.Took = sha, true, ByAdopting
	report.Subject = entry.Subject
	report.Emitted = staged.Written()
	return report, err
}

func kept(foreign []history.Entry) string {
	if len(foreign) == 1 {
		return "Adopt " + Short(foreign[0].Sha) + ", a commit origins did not write"
	}
	return fmt.Sprintf("Adopt %d commits origins did not write", len(foreign))
}

// covered is every file the given commits touched, named as the corpus names them.
func covered(repo *history.Repo, foreign []history.Entry) ([]string, error) {
	var all []string
	for _, was := range foreign {
		touched, err := repo.Touched(was.Sha)
		if err != nil {
			return nil, err
		}
		all = append(all, touched...)
	}
	slices.Sort(all)
	return slices.Compact(all), nil
}

// rewriting replaces the commits origins did not write with commits that say what they
// are, each of them keeping its author, its date and its place in the sequence.
func rewriting(repo *history.Repo, s history.Standing, root string) (*Resolution, error) {
	switch {
	case s.Anchor == nil:
		return nil, errors.New("origins wrote no part of this history, so there is nothing to rewrite onto")
	case len(s.Foreign) == 0:
		return nil, errors.New("there are no commits here for origins to rewrite")
	}
	shared, err := shareable(repo, s)
	if err != nil {
		return nil, err
	}
	if shared {
		return nil, errors.New("rewriting what has left this machine would hand everyone who has it a " +
			"history that no longer joins onto theirs: origins recover --adopt keeps it as it is " +
			"and records that you have read it")
	}
	corpus, problems := store.Load(root)
	if problems != nil {
		return nil, unreadable(problems, "origins will not rewrite a corpus it could not read whole: "+
			"put the files right, or take the commits off with origins recover --reset")
	}
	before, done, err := asRecorded(repo, s.Anchor.Sha, corpus.Corpus)
	defer done()
	if err != nil {
		return nil, err
	}

	// each replayed state is judged against the one before it, which is the same rule
	// every other change is held to, applied down the sequence as it is rebuilt
	running := before
	made, err := repo.Rewrite(s.Anchor.Sha, s.Foreign, func(was history.Entry) (history.Commit, error) {
		now, problems := store.Load(corpus.Root)
		if problems != nil {
			return history.Commit{}, fmt.Errorf("%s left a corpus origins cannot read: %w",
				Short(was.Sha), problems)
		}
		change, err := ops.Adopting(now, running).Stage()
		if err != nil {
			return history.Commit{}, fmt.Errorf("%s: %w", Short(was.Sha), err)
		}
		if err := change.Apply(); err != nil {
			return history.Commit{}, err
		}
		renamed := ops.Renamed(running, now.Corpus)
		running = now.Corpus

		touched, err := repo.Touched(was.Sha)
		if err != nil {
			return history.Commit{}, err
		}
		return history.Commit{
			Subject:  was.Subject,
			Tool:     Tool,
			Op:       "manual.adopt",
			Entities: ops.Subjects(now, touched),
			Renamed:  renamed,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	report := describing(s, shared)
	report.Settled, report.Took, report.Made = true, ByRewriting, made
	if len(made) > 0 {
		report.Commit = made[len(made)-1]
	}
	return report, nil
}

// unrecording takes the commits off the branch and leaves everything they said in the
// working tree.
//
// It is the resolution that decides nothing. What was committed becomes a change
// somebody has made and not yet recorded, which is a state origins already knows what
// to do with, and the question of what those commits were for is handed back to the
// person who is in a position to answer it.
func unrecording(repo *history.Repo, s history.Standing) (*Resolution, error) {
	if s.Anchor == nil {
		return nil, errors.New("origins wrote no part of this history, so there is nothing to take it back to")
	}
	if err := repo.Rewind(s.Anchor.Sha); err != nil {
		return nil, err
	}
	if err := repo.Mark(s.Anchor.Sha); err != nil {
		return nil, err
	}
	report := describing(s, false)
	report.Commit, report.Settled, report.Took = s.Anchor.Sha, true, ByResetting
	return report, nil
}
