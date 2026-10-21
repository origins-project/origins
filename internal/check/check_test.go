package check

import (
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

func build(entities ...any) *model.Corpus {
	c := model.NewCorpus()
	for _, e := range entities {
		ref := model.RefOf(e)
		if ref.Kind == "" {
			panic("build: not an entity")
		}
		c.Put(ref.Slug, e)
	}
	return c
}

// childOf is a family of parent and somebody nobody knows, listing children.
func childOf(parent model.Slug, children ...model.Slug) *model.Family {
	return &model.Family{
		Slug:     parent + "--unknown",
		Partners: model.PartnersOf(parent, model.Unknown),
		Children: model.ChildrenOf(children...),
	}
}

func on(s string) model.Date {
	d, err := model.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func born(date string) model.Event { return model.Event{Type: "birth", Date: on(date)} }
func died(date string) model.Event { return model.Event{Type: "death", Date: on(date)} }

func ruleIDs(findings []Finding, severity Severity) []string {
	var out []string
	for _, f := range findings {
		if f.Severity == severity && !slices.Contains(out, f.Rule) {
			out = append(out, f.Rule)
		}
	}
	slices.Sort(out)
	return out
}

// fires asserts that a rule reports, which is the other half of every must-not-flag
// case: a rule that never speaks passes those trivially.
func fires(t *testing.T, rule string, c *model.Corpus) Finding {
	t.Helper()
	for _, f := range Run(c) {
		if f.Rule == rule {
			return f
		}
	}
	t.Fatalf("%s did not report; findings were %v", rule, Run(c))
	return Finding{}
}

func silent(t *testing.T, rule string, c *model.Corpus) {
	t.Helper()
	for _, f := range Run(c) {
		if f.Rule == rule {
			t.Fatalf("%s should not have reported: %s %s %s", rule, f.Entity, f.Path, f.Message)
		}
	}
}
