package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// Taken is what a pull took in from where, and how.
type Taken struct {
	Remote   string   `json:"remote"`
	Upstream string   `json:"upstream"`
	Took     string   `json:"took"` // nothing, fast-forward or merge
	Commits  int      `json:"commits"`
	Commit   string   `json:"commit"`
	Subjects []string `json:"subjects"`
}

// Pull takes in what another clone recorded and pushed, from the remote named or the
// one history.Toward chooses.
//
// Every commit it takes in has to be one an origins transaction wrote, because a pull
// that merged somebody's hand-made commits would bury them beneath a commit of origins'
// own, where nothing would ever find them again. Where the two clones have each gone
// on, the merge is judged as an adoption is, against the corpus as HEAD left it, since
// two lines each sound on their own can give one person two births.
func Pull(root, named string, terminal history.Terminal) (report *Taken, err error) {
	repo, release, err := writing(root, "pull")
	if err != nil {
		return nil, err
	}
	defer func() { err = joined(err, release()) }()
	if err := unpulled(repo); err != nil {
		return nil, err
	}
	remote, err := toward(repo, named, "pull")
	if err != nil {
		return nil, err
	}
	if err := repo.Fetch(remote, terminal); err != nil {
		return nil, err
	}
	upstream, there, err := repo.Incoming(remote)
	if err != nil {
		return nil, err
	}
	foreign, err := repo.Above(upstream)
	if err != nil {
		return nil, err
	}
	if len(foreign) > 0 {
		return nil, errors.New(unvouched(upstream, foreign))
	}

	report = &Taken{Remote: remote, Upstream: upstream, Took: "nothing", Subjects: []string{}}
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	if report.Commits, err = repo.Count(there); err != nil && head != "" {
		return nil, err
	}
	level := head == there
	if !level && head != "" {
		if level, err = repo.Descends(head, there); err != nil {
			return nil, err
		}
	}
	ahead := head == ""
	if !level && !ahead {
		if ahead, err = repo.Descends(there, head); err != nil {
			return nil, err
		}
	}

	switch {
	case level:
		report.Commits = 0
	case ahead:
		report.Took = "fast-forward"
		if report.Commit, err = repo.FastForward(upstream); err != nil {
			return nil, err
		}
	default:
		report.Took = "merge"
		var subjects []string
		report.Commit, subjects, err = merging(repo, root, head, upstream)
		if report.Commit == "" {
			return nil, err
		}
		report.Subjects = append(report.Subjects, subjects...)
		return report, err
	}
	return report, nil
}

// unpulled refuses to begin a pull while anything in the repository is staged or
// modified. A merge writes over the working tree and its commit takes whatever is
// staged, so what was unrecorded would be lost or swept into the merge.
func unpulled(repo *history.Repo) error {
	if merging, err := repo.Merging(); err != nil {
		return err
	} else if merging {
		return errors.New("git is part way through a merge here: git merge --continue or git merge --abort settles it")
	}
	dirty, err := repo.Dirty()
	if err != nil {
		return err
	}
	outside, err := repo.Outside()
	if err != nil {
		return err
	}
	if len(dirty) == 0 && len(outside) == 0 {
		return nil
	}
	var said []string
	if len(dirty) > 0 {
		said = append(said, "these files in the corpus have been changed and not recorded, "+
			"which origins adopt records:")
		for _, file := range dirty {
			said = append(said, "  "+file)
		}
	}
	if len(outside) > 0 {
		said = append(said, "these files outside the corpus are staged or modified, "+
			"and are to be committed or put away first:")
		for _, file := range outside {
			said = append(said, "  "+file)
		}
	}
	return fmt.Errorf("origins will not pull while anything is unrecorded, a merge writing over "+
		"the working tree:\n%s", strings.Join(said, "\n"))
}

// unvouched is the refusal of an upstream with commits origins did not write on top
// of the last one it did, and the two ways to take them in.
func unvouched(upstream string, foreign []history.Entry) string {
	said := []string{fmt.Sprintf("%s holds %s origins did not write on top of the last one it did, "+
		"and nothing has been taken in:", upstream, Tally(len(foreign), "commit", "commits"))}
	for _, entry := range foreign[:min(len(foreign), 10)] {
		said = append(said, fmt.Sprintf("  %s  %s  (%s)", Short(entry.Sha), entry.Subject, entry.Author))
	}
	if len(foreign) > 10 {
		said = append(said, fmt.Sprintf("  and %d more", len(foreign)-10))
	}
	said = append(said, "",
		"whoever made them is the one to vouch for them, with origins recover --adopt in their clone "+
			"and origins push; or they are taken in here with git merge "+upstream+
			", origins recover --adopt and origins push")
	return strings.Join(said, "\n")
}

// merging merges an upstream the branch has diverged from, judges the result against
// the corpus as HEAD left it, and records it as the merge commit, or leaves the
// repository as it was.
func merging(repo *history.Repo, root, head, upstream string) (string, []string, error) {
	conflicts, err := repo.Merge(upstream)
	if err != nil {
		return "", nil, err
	}
	if len(conflicts) > 0 {
		said := make([]string, len(conflicts))
		for i, file := range conflicts {
			said[i] = "  " + file
		}
		return "", nil, fmt.Errorf("this clone and %s changed the same lines, and nothing has been taken in:\n%s\n\n"+
			"git merge %s, settling them, and origins recover after it take them in by hand",
			upstream, strings.Join(said, "\n"), upstream)
	}
	abort := func(err error) (string, []string, error) {
		return "", nil, joined(err, repo.AbortMerge())
	}

	corpus, problems := store.Load(root)
	if problems != nil {
		return abort(unreadable(problems, fmt.Sprintf("the merge with %s left a corpus origins could not "+
			"read whole, and nothing has been taken in", upstream)))
	}
	before, done, err := recordedAt(repo, head)
	defer done()
	if err != nil {
		return abort(err)
	}
	changed, err := repo.Dirty()
	if err != nil {
		return abort(err)
	}
	entry := history.Commit{
		Subject:  "Pull " + upstream,
		Tool:     Tool,
		Op:       history.Pulled,
		Entities: ops.Subjects(corpus, changed),
		Renamed:  ops.Renamed(before.Corpus, corpus.Corpus),
	}
	_, sha, err := ops.Adopting(corpus, before.Corpus).Conclude(repo, entry)
	if !landed(sha, err) {
		return abort(errors.Join(err, fmt.Errorf("nothing has been taken in from %s", upstream)))
	}
	return sha, written(entry.Entities), err
}
