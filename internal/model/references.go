package model

import (
	"maps"
	"slices"
	"strings"
)

// References hands every slug one entity writes about another record to a function that
// may change it. It is the one place the shape of the graph is written down as something
// to walk rather than something to validate, and a field added to the model that holds
// a reference belongs in it.
//
// A hypothesis's edits are deliberately not walked. They are pinned to the commit they
// were written against and are reconstructed there rather than reapplied to the corpus
// in front of them, so a slug inside one names the corpus as it stood.
//
// What an entity cites is walked by Cited, which is where a field holding a citation is
// listed, and a picture is reached through Pictures, whichever kind keeps it. A
// picture's path into another record's directory names that record, and is written
// again where the function changes the name, as the path beside the record where it
// leads into its own, unless the function clears it, there being no path to nothing.
func References(entity any, each func(*Slug)) {
	events := func(list []Event) {
		for i := range list {
			each(&list[i].Place)
			for j := range list[i].With {
				each(&list[i].With[j].Person)
			}
			each(&list[i].Family)
		}
	}
	switch v := entity.(type) {
	case *Person:
		events(v.Events)
	case *Family:
		for i := range v.Partners {
			each(&v.Partners[i].Person)
		}
		for i := range v.Children {
			each(&v.Children[i].ID)
			for j := range v.Children[i].Relation.Partners {
				each(&v.Children[i].Relation.Partners[j].Partner)
			}
		}
		events(v.Events)
	case *Place:
		for i := range v.Within {
			each(&v.Within[i].Place)
		}
	case *Note:
		events(v.Events)
	}
	if same := SameAsOf(entity); same != nil {
		for i := range *same {
			if (*same)[i].ID != "" {
				each(&(*same)[i].ID)
			}
		}
	}
	picture := func(m *Media) {
		each(&m.Place)
		for j := range m.Depicts {
			each(&m.Depicts[j].Person)
		}
		from := RefOf(entity)
		at := Locate(from, m.File)
		if at.slug < 0 {
			return
		}
		slug := at.Record.Slug
		if each(&slug); slug == at.Record.Slug || slug == "" {
			return
		}
		if at.Record.Kind == from.Kind && slug == from.Slug {
			m.File = at.File
			return
		}
		parts := strings.Split(m.File, "/")
		parts[at.slug] = string(slug)
		m.File = strings.Join(parts, "/")
	}
	switch v := entity.(type) {
	case *Source:
		for i := range v.Covers {
			each(&v.Covers[i].Place)
		}
		for i := range v.Repositories {
			each(&v.Repositories[i].Repository)
		}
		// a citation is held in the map by value, and who it names in a list the copy
		// shares with it
		for _, anchor := range slices.Sorted(maps.Keys(v.Citations)) {
			with := v.Citations[anchor].With
			for j := range with {
				each(&with[j].Person)
			}
		}
		v.Scans(func(_ string, m *Media) { picture(m) })
	}
	Pictures(entity, func(_ string, m *Media) { picture(m) })
	Cited(entity, func(c *CitationRef) { each(&c.Source) })
}
