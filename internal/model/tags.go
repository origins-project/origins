package model

import (
	"fmt"
	"maps"
	"slices"
)

// Tags hands every list of tags an entity holds to a function that may replace it, with
// the field path it is at: the record's own, each event's, each citation's and each
// picture's, a citation's scans among them. It is how a tag renamed is followed into
// every record carrying it, and a field added to the model that holds tags belongs in
// it.
func Tags(entity any, each func(path string, tags *[]string)) {
	events := func(list []Event) {
		for i := range list {
			each(fmt.Sprintf("events[%d].tags", i), &list[i].Tags)
		}
	}
	switch v := entity.(type) {
	case *Person:
		each("tags", &v.Tags)
		events(v.Events)
	case *Family:
		each("tags", &v.Tags)
		events(v.Events)
	case *Place:
		each("tags", &v.Tags)
	case *Source:
		each("tags", &v.Tags)
		for _, anchor := range slices.Sorted(maps.Keys(v.Citations)) {
			// a citation is held in the map by value
			citation := v.Citations[anchor]
			each("citations."+anchor+".tags", &citation.Tags)
			v.Citations[anchor] = citation
		}
		v.Scans(func(at string, m *Media) { each(at+".tags", &m.Tags) })
	case *Repository:
		each("tags", &v.Tags)
	case *Note:
		each("tags", &v.Tags)
		events(v.Events)
	}
	Pictures(entity, func(at string, m *Media) { each(at+".tags", &m.Tags) })
}
