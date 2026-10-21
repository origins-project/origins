package store

import (
	"fmt"
	"maps"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/model"
)

// A draft is a hypothesis still being written: the commit it was begun on and its
// edits in order, spelled as a hypothesis spells them, so that keeping it is writing
// the same edits into a record and nothing has to be translated on the way.

type draftDoc struct {
	Base  string    `yaml:"base"`
	Edits []editDoc `yaml:"edits"`
}

// EmitDraft writes a draft as the YAML kept beside the repository.
func EmitDraft(base string, edits []model.Edit) ([]byte, error) {
	var e emitter
	e.field("base", base)
	e.sequence("edits", editItems(edits))
	if e.err != nil {
		return nil, e.err
	}
	return []byte(e.b.String()), nil
}

// EditFields is a list of edits as a hypothesis spells them, each the mapping its file
// writes, read back off what EmitDraft writes.
func EditFields(edits []model.Edit) ([]any, error) {
	written, err := EmitDraft("-", edits)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Edits []any `yaml:"edits"`
	}
	if err := yaml.Unmarshal(written, &doc); err != nil {
		return nil, err
	}
	if doc.Edits == nil {
		doc.Edits = []any{}
	}
	return doc.Edits, nil
}

// ParseDraft reads what EmitDraft wrote.
func ParseDraft(written []byte) (base string, edits []model.Edit, err error) {
	var doc draftDoc
	if err := unmarshal(written, &doc); err != nil {
		return "", nil, err
	}
	if strings.TrimSpace(doc.Base) == "" {
		return "", nil, fmt.Errorf("a draft names the commit it was begun on")
	}
	for i, e := range doc.Edits {
		edit, err := e.edit()
		if err != nil {
			return "", nil, fmt.Errorf("edits[%d]: %w", i, err)
		}
		edits = append(edits, edit)
	}
	return doc.Base, edits, nil
}

// Copy is the corpus with a model of its own, to be changed in memory while the corpus
// stays as it was, each record kept where its file is on disk until Refile is asked
// where that is now.
func (c *Corpus) Copy() *Corpus {
	copied := &Corpus{
		Corpus: c.Corpus.Clone(),
		Root:   c.Root,
		paths:  maps.Clone(c.paths),
		kept:   map[any]string{},
		files:  maps.Clone(c.files),
	}
	for slug, at := range c.paths {
		if entity := copied.Entity(slug); entity != nil {
			copied.kept[entity] = at
		}
	}
	return copied
}

// Refile settles where each record of a model changed in memory is kept: where its
// file still is, under whatever name the record now goes by, so that a record renamed
// is read and its pictures served from where they are, and a record made has no file
// yet.
func (c *Corpus) Refile() {
	paths := map[model.Slug]string{}
	for _, slug := range c.AllSlugs() {
		if at, ok := c.kept[c.Entity(slug)]; ok {
			paths[slug] = at
		}
	}
	c.paths = paths
}
