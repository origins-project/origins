package store

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Corpus is a loaded corpus together with where on disk each entity came from.
//
// The model holds no paths, because where an entity is kept follows from its kind and
// its name, and no reference is ever a path. Anything that has to reach the files
// themselves — serving a photograph, rewriting an entity — needs this beside the model
// rather than inside it.
type Corpus struct {
	*model.Corpus

	// Root is the directory Load was given, as it was given.
	Root string

	paths map[model.Slug]string   // the entity's README, relative to Root, forward-slashed
	kept  map[any]string          // the same, keyed by the entity rather than by its name
	files map[model.Slug][]string // what was found beside it, relative to its directory
}

// Path is where the README holding the entity sits, relative to Root.
func (c *Corpus) Path(slug model.Slug) (string, bool) {
	p, ok := c.paths[slug]
	return p, ok
}

// Beside is what Load found in the entity's directory, relative to it and sorted.
//
// It still answers for a slug the model no longer holds, which is the reason it is
// kept here as well as on the entity: what becomes of a record's photographs is
// settled after the record itself has gone.
func (c *Corpus) Beside(slug model.Slug) []string { return c.files[slug] }

// Place is where an entity's file is now, and where its name says it belongs.
//
// The two differ when the entity has been renamed. Keying by the entity rather than by
// its slug is what makes that answerable: a slug is not a fixed address, and an entity
// given a new name is the same entity somewhere else in the model. Nothing has to
// record that a rename happened, which is bookkeeping that could drift from what the
// model holds.
//
// An entity origins has not seen before is nowhere yet.
func (c *Corpus) Place(entity any, slug model.Slug) (now, belongs string, err error) {
	dir, _, err := kind(entity)
	if err != nil {
		return "", "", err
	}
	if belongs, err = pathOf(entity, dir, slug); err != nil {
		return "", "", err
	}
	return c.kept[entity], belongs, nil
}

// Location is what an entity kept at relative occupies: its directory, the README
// holding it and everything attached to it. It is what moves when an entity is
// renamed, so that nothing attached to it is left behind.
func Location(relative string) string { return path.Dir(relative) }

// Slugs lists every entity that loaded, of every kind, in sorted order.
func (c *Corpus) Slugs() []model.Slug {
	slugs := make([]model.Slug, 0, len(c.paths))
	for slug := range c.paths {
		slugs = append(slugs, slug)
	}
	slices.Sort(slugs)
	return slugs
}

// Attachment resolves one of an entity's attached files to a path on disk.
//
// name is as it appears in the entity's Files or among a citation's images: relative to the
// entity's directory, forward-slashed. It is checked against the attachments actually
// found on disk rather than merely sanitised, so a name that climbs out of the corpus
// or names a file nobody attached is refused by the same test. A link is refused too,
// since it leads wherever it was made to.
func (c *Corpus) Attachment(slug model.Slug, name string) (string, error) {
	entity, ok := c.paths[slug]
	if !ok {
		return "", fmt.Errorf("no entity %q", slug)
	}
	clean := path.Clean(strings.TrimPrefix(name, "./"))
	if !slices.Contains(c.attachments(slug), clean) {
		return "", fmt.Errorf("%q has no attachment %q", slug, name)
	}
	found := filepath.Join(c.Root, filepath.FromSlash(path.Dir(entity)), filepath.FromSlash(clean))
	info, err := os.Lstat(found)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%q has %q as a link rather than a file", slug, name)
	}
	return found, nil
}

func (c *Corpus) attachments(slug model.Slug) []string {
	if files := model.Files(c.Entity(slug)); files != nil {
		return *files
	}
	return nil
}

// Viewed is the corpus as a view of it shows it: the model the view made, with each
// record still kept where it is on disk and with only the files the view keeps of it,
// so that a record or a file the view left out is one nothing here can lead to.
func (c *Corpus) Viewed(view *model.Corpus) *Corpus {
	viewed := &Corpus{Corpus: view, Root: c.Root, paths: map[model.Slug]string{}, files: map[model.Slug][]string{}}
	for slug, at := range c.paths {
		if kind := view.KindOf(slug); kind != "" && kind == c.KindOf(slug) {
			viewed.paths[slug] = at
			if files := model.Files(view.Entity(slug)); files != nil {
				viewed.files[slug] = slices.Clone(*files)
			}
		}
	}
	return viewed
}
