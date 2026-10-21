package model

import (
	"maps"
	"slices"
)

// Cited hands every citation an entity makes to a function that may change it, wherever
// in the record it is written. It is how a renamed source is followed into everything
// citing it and how a source finds what cites it, and a field added to the model that
// holds a citation belongs in citationLists, which it walks.
func Cited(entity any, each func(*CitationRef)) {
	citationLists(entity, func(refs *[]CitationRef) {
		for i := range *refs {
			each(&(*refs)[i])
		}
	})
}

// citationLists hands every list of citations an entity holds to a function that may
// replace it, so that a citation can be taken out as well as changed.
func citationLists(entity any, each func(*[]CitationRef)) {
	attributes := func(list []Attribute) {
		for i := range list {
			each(&list[i].Sources)
		}
	}
	identifiers := func(ids IDs) {
		for _, service := range slices.Sorted(maps.Keys(ids)) {
			for i := range ids[service] {
				each(&ids[service][i].Sources)
			}
		}
	}
	events := func(list []Event) {
		for i := range list {
			for j := range list[i].With {
				each(&list[i].With[j].Name.Sources)
				attributes(list[i].With[j].Attributes)
				each(&list[i].With[j].Sources)
			}
			each(&list[i].Address.Sources)
			each(&list[i].Age.Sources)
			each(&list[i].Cause.Sources)
			attributes(list[i].Attributes)
			each(&list[i].Sources)
			identifiers(list[i].IDs)
		}
	}
	nonEvents := func(list []NonEvent) {
		for i := range list {
			each(&list[i].Sources)
		}
	}
	addresses := func(list []Address) {
		for i := range list {
			each(&list[i].Sources)
		}
	}
	picture := func(m *Media) {
		for j := range m.Depicts {
			attributes(m.Depicts[j].Attributes)
		}
		attributes(m.Attributes)
		each(&m.Sources)
		identifiers(m.IDs)
	}
	switch v := entity.(type) {
	case *Person:
		each(&v.Name.Sources)
		for i := range v.AlsoKnownAs {
			each(&v.AlsoKnownAs[i].Sources)
		}
		each(&v.Sex.Sources)
		events(v.Events)
		nonEvents(v.Not)
		each(&v.Sources)
		identifiers(v.IDs)
	case *Family:
		for i := range v.Partners {
			each(&v.Partners[i].Name.Sources)
		}
		for i := range v.Children {
			each(&v.Children[i].Name.Sources)
			each(&v.Children[i].Sources)
		}
		events(v.Events)
		nonEvents(v.Not)
		each(&v.Sources)
		identifiers(v.IDs)
	case *Place:
		each(&v.Name.Sources)
		for i := range v.AlsoKnownAs {
			each(&v.AlsoKnownAs[i].Sources)
		}
		for i := range v.Within {
			each(&v.Within[i].Sources)
		}
		addresses(v.Addresses)
		each(&v.Sources)
		identifiers(v.IDs)
	case *Source:
		attributes(v.Attributes)
		// a citation is held in the map by value, and its lists in arrays the copy
		// shares with it
		for _, anchor := range slices.Sorted(maps.Keys(v.Citations)) {
			with := v.Citations[anchor].With
			for j := range with {
				each(&with[j].Name.Sources)
				attributes(with[j].Attributes)
			}
			attributes(v.Citations[anchor].Attributes)
			identifiers(v.Citations[anchor].IDs)
		}
		v.Scans(func(_ string, m *Media) { picture(m) })
		identifiers(v.IDs)
	case *Repository:
		addresses(v.Addresses)
		identifiers(v.IDs)
	case *Note:
		events(v.Events)
		each(&v.Sources)
		identifiers(v.IDs)
	case *Hypothesis:
		each(&v.Supports)
		each(&v.Contradicts)
	}
	Pictures(entity, func(_ string, m *Media) { picture(m) })
	if same := SameAsOf(entity); same != nil {
		for i := range *same {
			each(&(*same)[i].Sources)
		}
	}
}
