package ops

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/check"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Tx is one change to a corpus, from before it is made until it lands.
//
// Begin takes the corpus as it stands, the caller mutates the model through Corpus, and
// Stage works out which files that leaves differing from the ones on disk. Nothing
// reaches the directory until the whole change is known to be good, and what Stage
// returns then either lands entire or leaves no trace.
//
// Begin expects a corpus that loaded without problems. A file origins could not parse
// is not in the model, so writing the model back over it would record a state that does
// not describe the directory.
type Tx struct {
	corpus *store.Corpus
	before map[check.Finding]bool
	taken  map[model.Slug]model.Slug // records folded into other records, and into which
}

// Begin opens a transaction, which costs one validation pass: what the corpus already
// contradicts has to be known before the change is made, because afterwards there is no
// way back to it.
func Begin(corpus *store.Corpus) *Tx { return from(corpus, corpus.Corpus) }

// Adopting opens a transaction over a corpus that has already been changed — edited by
// hand, or carried in by commits origins did not write — and judges it against the
// state it was changed from rather than against itself.
//
// Begin cannot do it, and the reason is the same one that makes it right everywhere
// else: it reads what the corpus already contradicts from the corpus in front of it.
// Where the change is still to be made that is the state before the change, and where
// the change is already in the files it is the state after, so every contradiction the
// change introduced would count as one that was always there and nothing would ever be
// refused. The state before is the last one recorded, and the only way to it is to read
// the files as they were.
func Adopting(corpus *store.Corpus, before *model.Corpus) *Tx { return from(corpus, before) }

func from(corpus *store.Corpus, before *model.Corpus) *Tx {
	tx := &Tx{
		corpus: corpus,
		before: map[check.Finding]bool{},
		taken:  map[model.Slug]model.Slug{},
	}
	for _, finding := range check.Run(before) {
		if finding.Severity == check.Error {
			tx.before[finding] = true
		}
	}
	return tx
}

// Corpus is the model to mutate. What it holds when Stage is called is what gets
// written.
func (tx *Tx) Corpus() *model.Corpus { return tx.corpus.Corpus }

// Perform carries out an edit against the transaction's corpus and describes it, and
// is the only way an edit is carried out: done on the model alone, there would be
// nowhere to put the one part of the answer a corpus cannot hold.
//
// A record folded into another is indistinguishable afterwards from a record deleted,
// so the transaction is told which two they were and carries what was attached to the
// one after the other. This is not a record of which entities the change touched,
// which is the bookkeeping the write path exists to avoid: it is about a name the
// corpus no longer has, and it is kept only until the change is staged.
func (tx *Tx) Perform(edit model.Edit) (history.Commit, error) {
	entry, taken, err := performed(tx.Corpus(), edit)
	for _, one := range taken {
		tx.taken[one.From] = one.Into
	}
	return entry, err
}

// Stage emits the corpus, works out which files differ from those on disk, and refuses
// a change that would make the corpus contradict itself.
//
// The test is for contradictions this change introduces, not for a corpus that is
// already wrong. An error already in the tree is often the very thing being worked on,
// and a transaction that refused to run until the corpus was perfect would leave no way
// to make it so.
func (tx *Tx) Stage() (*Change, error) {
	change, err := tx.staged()
	if err != nil {
		return nil, err
	}
	if introduced := tx.introduced(); len(introduced) > 0 {
		return nil, &Contradiction{Findings: introduced}
	}
	return change, nil
}

// staged is the file half of it, which is Stage for a caller that wants to be told
// what the change comes to rather than whether it is allowed.
//
// A corpus in a later format than this origins writes is refused before anything is
// emitted, since everything emitted would be in the earlier one.
func (tx *Tx) staged() (*Change, error) {
	if err := store.Later(tx.corpus.Project); err != nil {
		return nil, fmt.Errorf("%w, and changing it would write it back in the earlier format: "+
			"an origins that writes format-version %d is needed", err, tx.corpus.Project.FormatVersion)
	}
	var (
		corpus   = tx.corpus
		change   = &Change{root: corpus.Root, write: map[string][]byte{}, move: map[string]string{}}
		problems []error
	)

	// The bytes are compared where they are and written where they belong, which are
	// two places for an entity that has been renamed. Comparing at the new name would
	// find nothing there and rewrite every file a rename touched.
	type staging struct {
		now, belongs string
		emit         func() ([]byte, error)
		canonical    []byte
		err          error
	}
	var files []staging

	// origins.yaml is only re-emitted when it read cleanly, as fmt does it. A project
	// that failed to decode is left zero-valued, and writing that out would delete the
	// header rather than record a change to it.
	if !corpus.Project.IsZero() {
		files = append(files, staging{now: store.ProjectFile, belongs: store.ProjectFile,
			emit: func() ([]byte, error) { return store.EmitProject(corpus.Project) }})
	}

	for _, slug := range corpus.Corpus.AllSlugs() {
		entity := corpus.Entity(slug)
		now, belongs, err := corpus.Place(entity, slug)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if now == "" {
			now = belongs // nothing has placed it, so it is new and there is nothing to compare
		} else if now != belongs {
			// the entity has been renamed, and its whole directory goes with it, the
			// photographs beside it included. Leaving them behind would be a rename
			// that quietly loses the evidence.
			change.move[store.Location(now)] = store.Location(belongs)
		}
		files = append(files, staging{now: now, belongs: belongs,
			emit: func() ([]byte, error) { return store.Emit(entity) }})
	}

	store.Each(len(files), func(i int) {
		file := &files[i]
		canonical, err := file.emit()
		if err != nil {
			file.err = fmt.Errorf("%s: %w", file.belongs, err)
			return
		}
		current, err := os.ReadFile(filepath.Join(corpus.Root, filepath.FromSlash(file.now)))
		switch {
		case err == nil && bytes.Equal(current, canonical):
			// already what it should be
		case err != nil && !os.IsNotExist(err):
			file.err = fmt.Errorf("%s: %w", file.now, err)
		default:
			file.canonical = canonical
		}
	})
	for _, file := range files {
		switch {
		case file.err != nil:
			problems = append(problems, file.err)
		case file.canonical != nil:
			change.write[file.belongs] = file.canonical
		}
	}

	// What was attached to a record taken into another goes where the record went. It
	// is the one thing the comparison above cannot find, the corpus having nothing left
	// in it to say that the two were ever one change, and it is why the transaction was
	// told.
	for _, gone := range slices.Sorted(maps.Keys(tx.taken)) {
		was, kept := corpus.Path(gone)
		files := corpus.Beside(gone)
		if !kept || len(files) == 0 {
			continue
		}
		into := corpus.Entity(tx.taken[gone])
		if into == nil {
			problems = append(problems,
				fmt.Errorf("%s was merged into %s, which is not in the corpus", gone, tx.taken[gone]))
			continue
		}
		_, belongs, err := corpus.Place(into, tx.taken[gone])
		if err != nil {
			problems = append(problems, err)
			continue
		}
		for _, file := range files {
			change.move[path.Join(store.Location(was), file)] =
				path.Join(store.Location(belongs), file)
		}
	}

	// A slug the corpus has a file for and the model no longer holds has gone, and its
	// file goes with it. Only the file: a record taken into another has had what was
	// attached to it carried off just above, and one simply deleted leaves its
	// photographs where they are, since deleting a record says nothing about the
	// evidence somebody filed beside it. A slug the model gave up because the entity
	// now answers to another one is a rename, and its file has already been carried
	// off rather than deleted.
	for _, slug := range corpus.Slugs() {
		if corpus.Entity(slug) != nil {
			continue
		}
		relative, ok := corpus.Path(slug)
		if ok && !moving(change, relative) {
			change.remove = append(change.remove, relative)
		}
	}
	slices.Sort(change.remove)

	if len(problems) > 0 {
		return nil, errors.Join(problems...)
	}
	return change, nil
}

// moving reports whether a file has already been carried off by a move, either as
// itself or inside the directory it sits in.
func moving(change *Change, relative string) bool {
	_, whole := change.move[relative]
	_, inside := change.move[store.Location(relative)]
	return whole || inside
}

// introduced lists the errors the corpus has now and did not have when the transaction
// opened.
func (tx *Tx) introduced() []check.Finding {
	var out []check.Finding
	for _, finding := range check.Run(tx.corpus.Corpus) {
		if finding.Severity == check.Error && !tx.before[finding] {
			out = append(out, finding)
		}
	}
	return out
}

// Contradiction is a change refused because it would leave the records disagreeing with
// each other. It names what it found, since a caller that cannot say what is wrong
// leaves the researcher to guess.
type Contradiction struct {
	Findings []check.Finding
}

func (c *Contradiction) Error() string {
	lines := make([]string, 0, len(c.Findings)+1)
	lines = append(lines, "the change would make the corpus contradict itself:")
	for _, f := range c.Findings {
		where := strings.TrimSpace(f.Entity + " " + f.Path)
		lines = append(lines, fmt.Sprintf("  %s: %s (%s)", where, f.Message, f.Rule))
	}
	return strings.Join(lines, "\n")
}

// Subjects names the entities kept in the given files, so that a change nobody
// described can still be found by the people it was about.
//
// It asks the corpus where each entity is kept rather than reading one out of a path,
// so that only a file an entity was actually read from names it. A file naming no
// entity — a photograph, a scan, something the corpus does not read — simply names
// none, and a file somebody deleted by hand names none either, its entity having left
// the model with it.
func Subjects(corpus *store.Corpus, paths []string) []model.EntityRef {
	changed := map[string]bool{}
	for _, path := range paths {
		changed[path] = true
	}
	var refs []model.EntityRef
	for _, slug := range corpus.Corpus.AllSlugs() {
		where, ok := corpus.Path(slug)
		if !ok || !changed[where] {
			continue
		}
		if kind := corpus.KindOf(slug); kind != "" {
			refs = append(refs, model.EntityRef{Kind: kind, Slug: slug})
		}
	}
	return refs
}

// Commit stages the change, writes it, and records it as one commit — or leaves the
// directory as it found it.
//
// A commit git refuses, for a hook or an identity it has not been told, takes the
// files back with it. A corpus changed on disk and not recorded is the state the whole
// scheme exists to keep a researcher out of, and it is one that looks perfectly
// healthy from inside the files.
//
// A change that amounts to nothing is not an error and not a commit either: the sha
// comes back empty, and the caller can say so.
//
// An error with a sha beside it is the other way round — the commit was made and
// something after it was not — so the change stays on the disk it is now recorded on,
// and the caller reports what went wrong rather than undoing a commit.
func (tx *Tx) Commit(repo *history.Repo, entry history.Commit) (*Change, string, error) {
	change, err := tx.Stage()
	if err != nil {
		return nil, "", err
	}
	if change.Empty() {
		return change, "", nil
	}
	return tx.land(repo, entry, change, func() (string, error) {
		return repo.Commit(entry, change.Paths())
	})
}

// Adopt records a change origins did not make, along with whatever re-emitting it
// canonically took.
//
// What it commits is the corpus rather than the files origins wrote, because origins
// wrote almost none of them: the change is already in the directory and the staging
// pass finds only the places where the hand that made it did not spell something the
// way the emitter does. Those are written first all the same, so that the next real
// edit to the file produces a diff of the change and not of the change plus a
// reformatting.
//
// A corpus needing no normalising is still recorded. Commit treats a change that
// amounts to nothing as no commit at all, which is right for an edit that turned out
// to be what the corpus already said; here the commit is the point, being what says
// this state has been read and accounted for.
func (tx *Tx) Adopt(repo *history.Repo, entry history.Commit) (*Change, string, error) {
	change, err := tx.Stage()
	if err != nil {
		return nil, "", err
	}
	return tx.land(repo, entry, change, func() (string, error) { return repo.Record(entry) })
}

// Conclude records a merge git has made in the files and the index, along with
// whatever re-emitting it canonically took, as the merge commit. It is Adopt for the one
// state in which git will commit nothing but the whole index.
func (tx *Tx) Conclude(repo *history.Repo, entry history.Commit) (*Change, string, error) {
	change, err := tx.Stage()
	if err != nil {
		return nil, "", err
	}
	return tx.land(repo, entry, change, func() (string, error) { return repo.Conclude(entry) })
}

// land writes a staged change and records it, taking the write back off the disk if
// the recording is refused outright.
func (tx *Tx) land(repo *history.Repo, entry history.Commit, change *Change,
	record func() (string, error)) (*Change, string, error) {
	undo, err := change.apply()
	if err != nil {
		return nil, "", err
	}
	sha, err := record()
	if err != nil && sha == "" {
		return nil, "", undo.rollback(err)
	}
	return change, sha, err
}
