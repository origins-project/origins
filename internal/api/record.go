package api

import (
	"fmt"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// Record is one record as the pages and an agent are given it.
//
// Its fields are its frontmatter as the emitter writes it, store.Fields, so that the
// files, the answers and whoever reads either read one shape, and the model takes no
// JSON of its own. Beside them is what it takes to change the record: Seen pins a
// change to the record as it was shown, and Edits names every path into it, the empty
// ones included, with the operations that can be carried out there. Sorted puts each
// of its events in order, keyed by its path, where a date or a sort date places it.
type Record struct {
	Ref      string            `json:"ref"`
	Fields   map[string]any    `json:"fields"`
	Seen     string            `json:"seen"`
	File     string            `json:"file"`
	Attached []string          `json:"attached"`
	Edits    []Editable        `json:"edits"`
	Sorted   map[string]string `json:"sorted"`
}

// Editable is one path into a record and the operations that apply to it.
type Editable struct {
	Path string   `json:"path"`
	Ops  []string `json:"ops"`
}

// The operations on a path: anything is set and unset, and a list is added to and
// removed from as well.
var (
	onValue = []string{model.Set{}.Op(), model.Unset{}.Op()}
	onList  = []string{model.Set{}.Op(), model.Unset{}.Op(), model.Add{}.Op(), model.Remove{}.Op()}
)

// Read is the record a corpus holds under a name, written kind/slug, as a bare slug,
// or project for the manifest.
func Read(root, entity string) (*Record, error) {
	corpus, problems := store.Load(root)
	if problems != nil {
		return nil, unreadable(problems, "origins will not show a record of a corpus it could not read whole")
	}
	ref := model.TheProject
	if entity != model.TheProject.String() {
		var err error
		if ref, err = Entity(corpus, entity); err != nil {
			return nil, err
		}
	}
	return RecordOf(corpus, ref)
}

// RecordOf is a record of a corpus already read.
func RecordOf(corpus *store.Corpus, ref model.EntityRef) (*Record, error) {
	var (
		entity   any
		file     = store.ProjectFile
		attached = []string{}
	)
	if ref == model.TheProject {
		entity = &corpus.Project
	} else {
		entity = corpus.Entity(ref.Slug)
		if entity == nil || model.RefOf(entity).Kind != ref.Kind {
			return nil, fmt.Errorf("there is no %s in %s", ref, corpus.Root)
		}
		file, _ = corpus.Path(ref.Slug)
		attached = append(attached, corpus.Beside(ref.Slug)...)
	}
	fields, err := store.Fields(entity)
	if err != nil {
		return nil, err
	}
	record := &Record{
		Ref:      ref.String(),
		Fields:   fields,
		Seen:     ops.Seen(corpus, ref),
		File:     file,
		Attached: attached,
		Edits:    []Editable{},
		Sorted:   sorted(entity),
	}
	for _, place := range store.Places(entity) {
		applies := onValue
		if place.List {
			applies = onList
		}
		record.Edits = append(record.Edits, Editable{Path: place.Path, Ops: applies})
	}
	return record, nil
}

// sorted is where each event of an entity goes in order, by its path.
func sorted(entity any) map[string]string {
	var events []model.Event
	switch e := entity.(type) {
	case *model.Person:
		events = e.Events
	case *model.Family:
		events = e.Events
	case *model.Note:
		events = e.Events
	}
	keys := map[string]string{}
	for i, event := range events {
		if key, ok := event.Placing().SortKey(); ok {
			keys[fmt.Sprintf("events[%d]", i)] = key
		}
	}
	return keys
}
