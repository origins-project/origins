package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/origins-project/origins/internal/model"
)

// Load reads a corpus from a directory.
//
// Every problem names the file it came from and is collected rather than raised: one
// malformed file must not hide the other fifty-four, and a corpus that mostly loads is
// far more useful than no corpus at all. The returned corpus holds everything that read
// cleanly whether or not the error is nil, so a caller that wants to press on can.
func Load(root string) (*Corpus, error) {
	corpus := &Corpus{
		Corpus: model.NewCorpus(),
		Root:   root,
		paths:  map[model.Slug]string{},
		kept:   map[any]string{},
		files:  map[model.Slug][]string{},
	}
	var problems []error

	project, err := os.ReadFile(filepath.Join(root, ProjectFile))
	switch {
	case err != nil:
		problems = append(problems, fmt.Errorf("%s: %w", ProjectFile, err))
	default:
		if corpus.Project, err = decodeProject(project); err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", ProjectFile, err))
		}
	}

	problems = append(problems, read(corpus, model.KindPerson.Dir(), decodePerson)...)
	problems = append(problems, read(corpus, model.KindFamily.Dir(), decodeFamily)...)
	problems = append(problems, read(corpus, model.KindPlace.Dir(), decodePlace)...)
	problems = append(problems, read(corpus, model.KindSource.Dir(), decodeSource)...)
	problems = append(problems, read(corpus, model.KindRepository.Dir(), decodeRepository)...)
	problems = append(problems, read(corpus, model.KindNote.Dir(), decodeNote)...)
	problems = append(problems, read(corpus, model.KindHypothesis.Dir(), decodeHypothesis)...)

	return corpus, errors.Join(problems...)
}

func read[T any](
	corpus *Corpus, dir string,
	decode func(model.Slug, []byte, string) (*T, error),
) []error {
	found, problems := entries(corpus.Root, dir)

	decoded := make([]*T, len(found))
	failed := make([]error, len(found))
	Each(len(found), func(i int) {
		decoded[i], failed[i] = readOne(corpus.Root, found[i], decode)
	})
	for i, e := range found {
		if failed[i] != nil {
			problems = append(problems, failed[i])
			continue
		}
		entity := decoded[i]
		*model.Files(entity) = e.files
		corpus.Put(e.slug, entity)
		corpus.paths[e.slug] = e.path
		corpus.kept[entity] = e.path
		corpus.files[e.slug] = e.files
	}
	return problems
}

func readOne[T any](root string, e entry, decode func(model.Slug, []byte, string) (*T, error)) (*T, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.path)))
	if err != nil {
		return nil, err
	}
	frontmatter, body, err := Split(content)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", e.path, err)
	}
	entity, err := decode(e.slug, frontmatter, string(body))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", e.path, err)
	}
	return entity, nil
}

// Each calls do with every index below n, as many at once as there are processors to
// run them. A corpus is thousands of small files, and the time to read one is mostly
// waiting on the file system, which a second read can do at the same time.
func Each(n int, do func(int)) {
	var (
		next atomic.Int64
		all  sync.WaitGroup
	)
	for range min(n, runtime.GOMAXPROCS(0)) {
		all.Go(func() {
			for i := int(next.Add(1) - 1); i < n; i = int(next.Add(1) - 1) {
				do(i)
			}
		})
	}
	all.Wait()
}
