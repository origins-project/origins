package store

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

const readme = "README.md"

// ProjectFile is the manifest, at the root of the corpus.
const ProjectFile = "origins.yaml"

// PathOf is where an entity is kept: <kind>/<slug>/README.md, relative to the corpus
// root and forward-slashed, as every path a corpus records is.
//
// It is the only place an entity can be. A directory from the start, so that the first
// photograph attached to a record is a file added beside it rather than the record
// moving into a directory it has only just earned, and one level down, so that where a
// record is kept follows from what it is and what it is called and from nothing else.
func PathOf(entity any) (string, error) {
	dir, slug, err := kind(entity)
	if err != nil {
		return "", err
	}
	return pathOf(entity, dir, slug)
}

func pathOf(entity any, dir string, slug model.Slug) (string, error) {
	if slug == "" {
		return "", fmt.Errorf("%T has no slug, so there is nothing to name its file", entity)
	}
	return path.Join(dir, string(slug), readme), nil
}

// kind is the directory an entity's kind is kept in, and the entity's slug.
func kind(entity any) (string, model.Slug, error) {
	ref := model.RefOf(entity)
	if ref.Kind == "" {
		return "", "", fmt.Errorf("%T is not an entity", entity)
	}
	return ref.Kind.Dir(), ref.Slug, nil
}

// MadeOf reports whether the file at rel, relative to root and forward-slashed, is one
// the corpus is made of: the manifest, or a file in an entity's directory, the README
// holding the entity or anything attached to it. A name beginning with a dot anywhere
// along the way is not, being by long convention a file nobody put there to be read,
// which is the rule Load keeps as well.
//
// It answers from the path and from which directories are entities, so it can be asked
// about a file git has not been given yet, which is the file it is asked about.
func MadeOf(root, rel string) bool {
	if rel == ProjectFile {
		return true
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 3 || model.KindIn(parts[0]) == "" {
		return false
	}
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return false
		}
	}
	info, err := os.Stat(filepath.Join(root, parts[0], parts[1], readme))
	return err == nil && !info.IsDir()
}

// entry is one entity as it sits on disk.
type entry struct {
	slug  model.Slug
	path  string   // the entity's README, relative to the corpus root and forward-slashed
	files []string // attachments, relative to the entity directory, sorted
}

// entries finds every entity kept in dir, one of the seven kind directories, relative
// to root. An entity is a directory directly inside it holding a README.md, and
// everything else in that directory is attached to it.
//
// A missing directory is not an error. A corpus with no hypotheses yet is an ordinary
// corpus.
//
// A kind directory holds entities and nothing else, so every other file in dir is a
// problem rather than something to read or to step over. A markdown file is a record
// out of place, and the problem names where it belongs: reading it from where it is
// would give a corpus a second way to keep a record. Any other file, a README
// describing the kind directory itself among them, belongs to no record, and a file no
// record answers for is one no change made through origins would ever account for. A
// name beginning with a dot is left out, being by long convention a file nobody put
// there to be read.
func entries(root, dir string) (found []entry, problems []error) {
	children, err := os.ReadDir(filepath.Join(root, dir))
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, []error{err}
	}

	misplaced := func(relative string) error {
		if path.Ext(relative) != ".md" || relative == path.Join(dir, readme) {
			return fmt.Errorf("%s: is neither a record nor beside one", relative)
		}
		name := strings.TrimSuffix(path.Base(relative), ".md")
		if path.Base(relative) == readme {
			name = path.Base(path.Dir(relative))
		}
		return fmt.Errorf("%s: a record is kept as %s", relative, path.Join(dir, name, readme))
	}

	type looked struct {
		entry    entry
		found    bool
		problems []error
	}
	children = slices.DeleteFunc(children, func(child fs.DirEntry) bool { return strings.HasPrefix(child.Name(), ".") })
	all := make([]looked, len(children))
	Each(len(children), func(i int) {
		child, one := children[i], &all[i]
		loose := func(relative string) { one.problems = append(one.problems, misplaced(relative)) }
		relative := path.Join(dir, child.Name())
		at := filepath.Join(root, filepath.FromSlash(relative))
		if !child.IsDir() {
			loose(relative)
			return
		}
		if info, err := os.Stat(filepath.Join(at, readme)); err != nil || info.IsDir() {
			if err := scatter(at, relative, loose); err != nil {
				one.problems = append(one.problems, err)
			}
			return
		}
		files, err := attachments(at)
		if err != nil {
			one.problems = append(one.problems, err)
			return
		}
		if e, err := newEntry(child.Name(), relative, files); err != nil {
			one.problems = append(one.problems, err)
		} else {
			one.entry, one.found = e, true
		}
	})
	for _, one := range all {
		problems = append(problems, one.problems...)
		if one.found {
			found = append(found, one.entry)
		}
	}
	slices.SortFunc(found, func(a, b entry) int { return strings.Compare(string(a.slug), string(b.slug)) })
	return found, problems
}

// scatter hands each file under a directory that is not an entity to loose, by its
// path relative to the corpus root, leaving out any name beginning with a dot.
func scatter(at, relative string, loose func(string)) error {
	return filepath.WalkDir(at, func(p string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case p != at && strings.HasPrefix(d.Name(), "."):
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		case d.IsDir():
			return nil
		}
		inside, err := filepath.Rel(at, p)
		if err != nil {
			return err
		}
		loose(path.Join(relative, filepath.ToSlash(inside)))
		return nil
	})
}

func newEntry(name, dir string, files []string) (entry, error) {
	s, err := model.ParseSlug(name)
	if err != nil {
		return entry{}, fmt.Errorf("%s: %w", dir, err)
	}
	return entry{slug: s, path: path.Join(dir, readme), files: files}, nil
}

// attachments lists every file under an entity's directory except the README that holds
// the entity itself and any name beginning with a dot, with forward slashes whatever the platform, so that the paths read
// the same on disk as they do in a citation's image field.
func attachments(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case path != dir && strings.HasPrefix(d.Name(), "."):
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		case d.IsDir():
			return nil
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if relative == readme {
			return nil
		}
		files = append(files, relative)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(files)
	return files, nil
}
