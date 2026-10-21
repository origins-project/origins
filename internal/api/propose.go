package api

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// Proposal is what proposing a hypothesis came to.
type Proposal struct {
	Hypothesis string   `json:"hypothesis"`
	Base       string   `json:"base"`
	ProposedBy string   `json:"proposed-by,omitempty"`
	Commit     string   `json:"commit"`
	Subject    string   `json:"subject"`
	Edits      []string `json:"edits"`
	Restored   []string `json:"restored"`
}

// Propose writes a hypothesis out of the corpus as it has been edited, proposed by
// whoever proposedBy names, or by the researcher where it names nobody.
//
// The files are edited as though the hypothesis were true, and origins states the
// difference from the last commit in the vocabulary, pins the hypothesis there, and
// puts the files back as that commit had them: the corpus is the standing conclusion,
// and a hypothesis exists only to propose something else. What goes back is what the
// hypothesis now holds, so nothing may be unrecorded that it does not hold — a
// photograph, a note beside the corpus — or putting the files back would lose it.
func Propose(asking Asking, slug model.Slug, proposedBy string) (report *Proposal, err error) {
	if proposedBy, err = proposer(proposedBy); err != nil {
		return nil, err
	}
	repo, release, err := writing(asking.Root, asking.Command)
	if err != nil {
		return nil, err
	}
	defer func() { err = joined(err, release()) }()
	corpus, problems := store.Load(asking.Root)
	if problems != nil {
		return nil, unreadable(problems, "origins will not propose from a corpus it could not read whole")
	}
	head, err := repo.Head()
	switch {
	case err != nil:
		return nil, err
	case head == "":
		return nil, errors.New("nothing has been recorded yet, so there is nothing for a hypothesis to depart from")
	}
	dirty, err := repo.Dirty()
	switch {
	case err != nil:
		return nil, err
	case len(dirty) == 0:
		return nil, fmt.Errorf("nothing in the corpus has been changed: edit the files as though %s "+
			"were true, then propose it", slug)
	}
	was, done, err := recordedAt(repo, head)
	defer done()
	if err != nil {
		return nil, err
	}
	if corpus.Taken(slug) || was.Taken(slug) {
		return nil, fmt.Errorf("the corpus already has something called %s", slug)
	}
	if path, ok := unstatable(dirty, corpus, was); !ok {
		return nil, fmt.Errorf("%s has been changed, and a hypothesis states changes to records and "+
			"the manifest and nothing else: origins adopt records it, or it can be put back", path)
	}

	edits, err := ops.Propose(was.Corpus, corpus.Corpus)
	if err != nil {
		return nil, fmt.Errorf("%v; origins will not propose what it cannot state", err)
	}
	if len(edits) == 0 {
		return nil, errors.New("the files differ from the last commit only in how they are written, " +
			"so there is nothing to propose: origins adopt records the spelling")
	}

	h := &model.Hypothesis{
		Slug:       slug,
		Status:     model.Open,
		Base:       head,
		Supports:   asking.Sources,
		Edits:      edits,
		ProposedBy: proposedBy,
		Body:       "# " + string(slug),
	}
	entry := history.Commit{
		Subject:    "Propose " + string(slug),
		Tool:       Tool,
		Op:         "hypothesis.propose",
		Entities:   []model.EntityRef{{Kind: model.KindHypothesis, Slug: slug}},
		Sources:    asking.Sources,
		Hypothesis: slug,
		ProposedBy: proposedBy,
	}
	if asking.Message != "" {
		entry.Subject = asking.Message
	}
	report = &Proposal{
		Hypothesis: string(slug),
		Base:       head,
		ProposedBy: proposedBy,
		Subject:    entry.Subject,
		Restored:   dirty,
	}
	for _, edit := range edits {
		report.Edits = append(report.Edits, stating(edit))
	}
	if asking.DryRun {
		return report, nil
	}
	sha, err := proposing(corpus.Root, repo, h, entry, dirty)
	if !landed(sha, err) {
		return nil, err
	}
	report.Commit = sha
	return report, err
}

// ProposeStated records a hypothesis stated whole, its file's frontmatter and body in
// document, as the open hypothesis slug, proposed by whoever proposedBy names, or by
// the researcher where it names nobody.
//
// It is the proposal made without the files: nothing is written but the hypothesis's
// own file. Who proposed it is the transport's to say and not the document's, and the
// document is pinned to the last commit where it names none. What it states has to
// carry onto the corpus as check --with carries it, since a hypothesis that cannot be
// supposed puts a question nobody can answer; one that drifted or conflicts comes back
// as the ops.Drifted it is.
func ProposeStated(asking Asking, slug model.Slug, document []byte, proposedBy string) (report *Proposal, err error) {
	if proposedBy, err = proposer(proposedBy); err != nil {
		return nil, err
	}
	h, err := store.ParseHypothesis(slug, document)
	if err != nil {
		return nil, fmt.Errorf("the hypothesis does not read: %w", err)
	}
	switch {
	case h.Status != "" && h.Status != model.Open:
		return nil, fmt.Errorf("the hypothesis says it is %s, and one is proposed open: "+
			"accepting or rejecting it is the researcher's", h.Status)
	case h.ProposedBy != "":
		return nil, errors.New("the hypothesis says who proposed it, which whoever records it writes " +
			"and the hypothesis does not state")
	case len(h.Edits) == 0:
		return nil, fmt.Errorf("%s states no edits, so there is nothing it would change", slug)
	}

	repo, release, err := writing(asking.Root, asking.Command)
	if err != nil {
		return nil, err
	}
	defer func() { err = joined(err, release()) }()
	corpus, problems := store.Load(asking.Root)
	if problems != nil {
		return nil, unreadable(problems, "origins will not propose to a corpus it could not read whole")
	}
	head, err := repo.Head()
	switch {
	case err != nil:
		return nil, err
	case head == "":
		return nil, errors.New("nothing has been recorded yet, so there is no commit to pin a hypothesis to")
	}
	if corpus.Taken(slug) {
		return nil, fmt.Errorf("the corpus already has something called %s", slug)
	}
	if h.Base == "" {
		h.Base = head
	} else if in, err := inHistory(repo, head, h.Base); err != nil {
		return nil, err
	} else if !in {
		return nil, fmt.Errorf("%s is no commit in this corpus's history, and a hypothesis is pinned to one",
			h.Base)
	}
	h.Status = model.Open
	h.ProposedBy = proposedBy
	if h.Supports == nil {
		h.Supports = asking.Sources
	}

	with := corpus.Copy()
	with.Hypotheses[slug] = h
	trial, _, err := Supposed(with, []model.Slug{slug})
	if err != nil {
		return nil, err
	}
	if err := trial.Close(); err != nil {
		return nil, err
	}

	entry := history.Commit{
		Subject:    "Propose " + string(slug),
		Tool:       Tool,
		Op:         "hypothesis.propose",
		Entities:   []model.EntityRef{{Kind: model.KindHypothesis, Slug: slug}},
		Sources:    h.Supports,
		Hypothesis: slug,
		ProposedBy: proposedBy,
	}
	if asking.Message != "" {
		entry.Subject = asking.Message
	}
	report = &Proposal{
		Hypothesis: string(slug),
		Base:       h.Base,
		ProposedBy: proposedBy,
		Subject:    entry.Subject,
		Restored:   []string{},
	}
	for _, edit := range h.Edits {
		report.Edits = append(report.Edits, stating(edit))
	}
	tx := ops.Begin(corpus)
	tx.Corpus().Hypotheses[slug] = h
	if asking.DryRun {
		if _, err := tx.Stage(); err != nil {
			return nil, err
		}
		return report, nil
	}
	_, sha, err := tx.Commit(repo, entry)
	if !landed(sha, err) {
		return nil, err
	}
	report.Commit = sha
	return report, err
}

// proposer is who proposedBy names, on the one line a trailer is.
func proposer(proposedBy string) (string, error) {
	proposedBy = strings.TrimSpace(proposedBy)
	if strings.ContainsAny(proposedBy, "\n\r") {
		return "", fmt.Errorf("%q covers more than one line, and whoever proposed a hypothesis "+
			"is named on one", proposedBy)
	}
	return proposedBy, nil
}

// inHistory is whether base is a commit the history at head leads from.
func inHistory(repo *history.Repo, head, base string) (bool, error) {
	known, err := repo.Has(base)
	if err != nil || !known {
		return false, err
	}
	return repo.Descends(head, base)
}

// unstatable finds a changed file that is no record's, on either side of the change,
// nor the manifest.
func unstatable(dirty []string, now, was *store.Corpus) (string, bool) {
	records := map[string]bool{store.ProjectFile: true}
	for _, side := range []*store.Corpus{now, was} {
		for _, slug := range side.Slugs() {
			if path, ok := side.Path(slug); ok {
				records[path] = true
			}
		}
	}
	for _, path := range dirty {
		if !records[path] {
			return path, false
		}
	}
	return "", true
}

// proposing puts the files back as the last commit had them and records the
// hypothesis there. The files are put back first, so that the commit is made on the
// corpus it is pinned to, and where the commit is refused they are put back as they
// were edited: they are the only copy of the proposal there is until it is recorded.
func proposing(root string, repo *history.Repo, h *model.Hypothesis, entry history.Commit,
	dirty []string) (string, error) {
	edited, err := readEdited(root, dirty)
	if err != nil {
		return "", err
	}
	unwind := func(reason error) (string, error) {
		if put := edited.putBack(root); put != nil {
			return "", errors.Join(fmt.Errorf("the edited files could not be put back: %w", put), reason)
		}
		return "", reason
	}
	if err := repo.Restore(); err != nil {
		return unwind(err)
	}
	recorded, problems := store.Load(root)
	if problems != nil {
		return unwind(fmt.Errorf("the corpus as last recorded will not read: %w", problems))
	}
	tx := ops.Begin(recorded)
	tx.Corpus().Hypotheses[h.Slug] = h
	_, sha, err := tx.Commit(repo, entry)
	if err != nil && sha == "" {
		return unwind(err)
	}
	return sha, err
}

// asEdited is the files a proposal was made from, as they were edited, with nil for one
// the edit deleted.
type asEdited map[string][]byte

func readEdited(root string, paths []string) (asEdited, error) {
	out := asEdited{}
	for _, path := range paths {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		switch {
		case errors.Is(err, fs.ErrNotExist):
			out[path] = nil
		case err != nil:
			return nil, err
		default:
			out[path] = content
		}
	}
	return out, nil
}

func (k asEdited) putBack(root string) error {
	var errs []error
	for path, content := range k {
		full := filepath.Join(root, filepath.FromSlash(path))
		if content == nil {
			if err := os.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, err)
			}
			_ = os.Remove(filepath.Dir(full)) // a directory the record was alone in
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			errs = append(errs, err)
			continue
		}
		errs = append(errs, os.WriteFile(full, content, 0o644))
	}
	return errors.Join(errs...)
}

// stating is an edit on one line, as the reader of a proposal checks it.
func stating(edit model.Edit) string {
	subjects := edit.Subjects()
	parts := []string{edit.Op(), subjects[0].String()}
	switch e := edit.(type) {
	case model.Rename:
		parts = append(parts, "to", string(e.To))
	default:
		if field := edit.Field(); field != "" {
			parts = append(parts, field)
		}
	}
	return strings.Join(parts, " ")
}
