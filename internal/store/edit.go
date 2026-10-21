package store

import (
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/model"
)

// editDoc is one entry of a hypothesis's edit list, before it is known which operation
// it is. The fields an operation does not use must be absent: an edit that carries a
// path alongside a create is not a create with extra decoration, it is a mistake, and
// silently ignoring it would apply something other than what the file says.
type editDoc struct {
	op     string
	entity string
	path   string
	value  any
	from   string
	into   []string
	to     string
	line   int
}

func (e *editDoc) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: an edit is a mapping", node.Line)
	}
	if err := onlyFields(node, "op", "entity", "path", "value", "from", "into", "to"); err != nil {
		return err
	}

	var doc struct {
		Op     string `yaml:"op"`
		Entity string `yaml:"entity"`
		Path   string `yaml:"path"`
		From   string `yaml:"from"`
		Into   any    `yaml:"into"`
		To     string `yaml:"to"`
	}
	if err := node.Decode(&doc); err != nil {
		return err
	}

	*e = editDoc{
		op:     doc.Op,
		entity: doc.Entity,
		path:   doc.Path,
		from:   doc.From,
		to:     doc.To,
		line:   node.Line,
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != "value" {
			continue
		}
		value, err := literal(node.Content[i+1])
		if err != nil {
			return err
		}
		e.value = value
	}
	switch into := doc.Into.(type) {
	case nil:
	case string:
		e.into = []string{into}
	case []any:
		for _, one := range into {
			text, ok := one.(string)
			if !ok {
				return fmt.Errorf("line %d: into lists entity references", node.Line)
			}
			e.into = append(e.into, text)
		}
	default:
		return fmt.Errorf("line %d: into is an entity reference, or a list of them", node.Line)
	}
	return nil
}

// literal reads an edit's value as it is written. The value stands in for a field of an
// entity, and every field of an entity is text — 1834 is a year and not an integer,
// 1750/51 is a year too — so YAML's resolver is not allowed near it. Without this, a
// hypothesis proposing a date would hold a time.Time that no entity could ever accept.
// Nor is an alias, which origins never writes and which a few lines can nest into more
// text than any machine holds.
func literal(node *yaml.Node) (any, error) {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" {
			return nil, nil
		}
		return node.Value, nil
	case yaml.SequenceNode:
		entries := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			entry, err := literal(child)
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
		return entries, nil
	case yaml.MappingNode:
		fields := make(map[string]any, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			value, err := literal(node.Content[i+1])
			if err != nil {
				return nil, err
			}
			fields[node.Content[i].Value] = value
		}
		return fields, nil
	case yaml.AliasNode:
		return nil, fmt.Errorf("line %d: a value is written out rather than named by an alias", node.Line)
	}
	return nil, fmt.Errorf("line %d: a value is text, a list, or a mapping", node.Line)
}

func (e editDoc) edit() (model.Edit, error) {
	if e.op == "" {
		return nil, fmt.Errorf("line %d: an edit must say which op it is", e.line)
	}
	if _, err := model.NewEdit(e.op); err != nil {
		return nil, fmt.Errorf("line %d: %w", e.line, err)
	}

	switch e.op {
	case "set":
		return build(e, "entity", "path", "value")(func(ref model.EntityRef) model.Edit {
			return model.Set{Entity: ref, Path: e.path, Value: e.value}
		})
	case "unset":
		return build(e, "entity", "path")(func(ref model.EntityRef) model.Edit {
			return model.Unset{Entity: ref, Path: e.path}
		})
	case "add":
		return build(e, "entity", "path", "value")(func(ref model.EntityRef) model.Edit {
			return model.Add{Entity: ref, Path: e.path, Value: e.value}
		})
	case "remove":
		return build(e, "entity", "path", "value")(func(ref model.EntityRef) model.Edit {
			return model.Remove{Entity: ref, Path: e.path, Value: e.value}
		})
	case "create":
		fields, err := mapping(e.value)
		if err != nil {
			return nil, fmt.Errorf("line %d: create %w", e.line, err)
		}
		return build(e, "entity", "value")(func(ref model.EntityRef) model.Edit {
			return model.Create{Entity: ref, Value: fields}
		})
	case "delete":
		return build(e, "entity")(func(ref model.EntityRef) model.Edit {
			return model.Delete{Entity: ref}
		})
	case "rename":
		edit, err := build(e, "entity", "to")(func(ref model.EntityRef) model.Edit {
			return model.Rename{Entity: ref}
		})
		if err != nil {
			return nil, err
		}
		rename := edit.(model.Rename)
		if rename.To, err = model.ParseName(rename.Entity.Kind, e.to); err != nil {
			return nil, fmt.Errorf("line %d: %w", e.line, err)
		}
		return rename, nil
	case "merge":
		if err := e.uses("from", "into"); err != nil {
			return nil, err
		}
		if len(e.into) != 1 {
			return nil, fmt.Errorf("line %d: merge folds into one entity", e.line)
		}
		from, err := model.ParseEntityRef(e.from)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", e.line, err)
		}
		into, err := model.ParseEntityRef(e.into[0])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", e.line, err)
		}
		return model.Merge{From: from, Into: into}, nil
	case "split":
		if err := e.uses("entity", "into"); err != nil {
			return nil, err
		}
		if len(e.into) < 2 {
			return nil, fmt.Errorf("line %d: split needs at least two entities to split into", e.line)
		}
		refs := make([]model.EntityRef, 0, len(e.into))
		for _, one := range e.into {
			ref, err := model.ParseEntityRef(one)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", e.line, err)
			}
			refs = append(refs, ref)
		}
		entity, err := model.ParseEntityRef(e.entity)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", e.line, err)
		}
		return model.Split{Entity: entity, Into: refs}, nil
	}
	return nil, fmt.Errorf("line %d: op %q is registered but not decoded", e.line, e.op)
}

// build checks the fields an operation uses, resolves its entity reference, and hands
// it to the constructor. Every operation but merge is addressed by one entity.
func build(e editDoc, fields ...string) func(func(model.EntityRef) model.Edit) (model.Edit, error) {
	return func(construct func(model.EntityRef) model.Edit) (model.Edit, error) {
		if err := e.uses(fields...); err != nil {
			return nil, err
		}
		ref, err := model.ParseEntityRef(e.entity)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", e.line, err)
		}
		return construct(ref), nil
	}
}

func (e editDoc) uses(fields ...string) error {
	present := []struct {
		name string
		set  bool
	}{
		{"entity", e.entity != ""},
		{"path", e.path != ""},
		{"value", e.value != nil},
		{"from", e.from != ""},
		{"into", len(e.into) > 0},
		{"to", e.to != ""},
	}
	for _, field := range present {
		switch wanted := slices.Contains(fields, field.name); {
		case wanted && !field.set:
			return fmt.Errorf("line %d: op %q needs %s", e.line, e.op, field.name)
		case !wanted && field.set:
			return fmt.Errorf("line %d: op %q takes %s, not %s",
				e.line, e.op, strings.Join(fields, " and "), field.name)
		}
	}
	return nil
}

func mapping(value any) (map[string]any, error) {
	fields, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("takes the new entity's fields as a mapping")
	}
	return fields, nil
}

// ParseEdit reads one edit written as a hypothesis writes an entry of its edit list, in
// YAML or in JSON, which YAML reads as it is. A value is text whichever it was written
// in, a number in JSON included, as every field of an entity is.
func ParseEdit(written []byte) (model.Edit, error) {
	var doc editDoc
	if err := yaml.Unmarshal(written, &doc); err != nil {
		return nil, err
	}
	return doc.edit()
}
