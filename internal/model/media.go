package model

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
)

// Media says what one of a record's own files is: what kind of picture, a title, when
// and where it was made, what says so, and who is in it. A file with nothing to say
// about it needs no entry. Region is the part of the picture the record is about where
// that is not a person: a house in a street, one entry on a page.
//
// File is a path from the record's directory, so an entry whose path leads into another
// record's refers to a picture that record keeps, and says only what its own reference
// says: Region, Attributes, Note, Sources and Private. A path that is not relative, or
// leads out of the corpus, is the one a picture still to be found was last known by.
//
// The entries are a list rather than a mapping keyed by the file, because a file's path
// is written with dots and a field path is divided at them.
type Media struct {
	Private    bool // left out of a hidden view
	File       string
	Type       string // the medium, an open word: photo, tombstone, newspaper
	Title      string
	Date       Date
	Place      Slug
	Region     *Region
	Depicts    []Depicted
	Attributes []Attribute
	Note       string
	Sources    []CitationRef
	Tags       []string
	IDs        IDs
}

// Qualified reports whether the entry says more than which file it is.
func (m Media) Qualified() bool {
	return m.Private || m.Type != "" || m.Title != "" || !m.Date.IsZero() || m.Place != "" ||
		m.Region != nil || len(m.Depicts) > 0 || len(m.Attributes) > 0 || m.Note != "" ||
		len(m.Sources) > 0 || len(m.Tags) > 0 || len(m.IDs) > 0
}

// Refers reports whether the entry refers to a picture another record keeps, its path
// leading out of the directory of the record it is written in and staying in the
// corpus. Every record's directory is as deep as every other's, so that is a question
// about the path alone.
func (m Media) Refers() bool {
	if !relative(m.File) {
		return false
	}
	up := 0
	for rest := path.Clean(m.File); rest == ".." || strings.HasPrefix(rest, "../"); up++ {
		rest = strings.TrimPrefix(strings.TrimPrefix(rest, ".."), "/")
	}
	return up == 1 || up == 2
}

// Says reports whether the entry says what only the record keeping the picture says:
// what it is, what it is called, when and where it was made, how a service knows it
// and who is in it.
func (m Media) Says() bool {
	return m.Type != "" || m.Title != "" || !m.Date.IsZero() || m.Place != "" ||
		len(m.IDs) > 0 || len(m.Depicts) > 0
}

func relative(file string) bool {
	return file != "" && !scheme.MatchString(file) && !strings.HasPrefix(file, "/") &&
		!strings.HasPrefix(file, `\`)
}

// Location is where the file a media entry names is kept: beside Record, at File,
// where the path leads into a record's directory, and nowhere where it leads into the
// corpus but into none. Abroad is a path that is not relative or leads out of the
// corpus, the one a picture still to be found was last known by.
type Location struct {
	Record EntityRef
	File   string
	Abroad bool

	// slug is which of the path's segments names Record, -1 where none does because
	// the path stays within the directory of the record it is written in.
	slug int
}

// Locate works out where the file a media entry written on the record from names is
// kept.
func Locate(from EntityRef, file string) Location {
	if !relative(file) {
		return Location{Abroad: true, slug: -1}
	}
	stack, escaped := walk(from, strings.Split(file, "/"))
	if escaped {
		return Location{Abroad: true, slug: -1}
	}
	if len(stack) < 3 || KindIn(stack[0].name) == "" {
		return Location{slug: -1}
	}
	var names []string
	for _, s := range stack[2:] {
		names = append(names, s.name)
	}
	return Location{Record: EntityRef{Kind: KindIn(stack[0].name), Slug: Slug(stack[1].name)},
		File: strings.Join(names, "/"), slug: stack[1].written}
}

// MediaLists hands every list of pictures a record holds to a function, with the path
// to it, in the order the record is written: each event's, then the record's own. A
// citation's scans are not among them, Scans handing those out.
func MediaLists(entity any, each func(at string, list []Media)) {
	events := func(list []Event) {
		for i := range list {
			each(fmt.Sprintf("events[%d].media", i), list[i].Media)
		}
	}
	switch e := entity.(type) {
	case *Person:
		events(e.Events)
		each("media", e.Media)
	case *Family:
		events(e.Events)
		each("media", e.Media)
	case *Place:
		each("media", e.Media)
	case *Source:
		each("media", e.Media)
	case *Repository:
		each("media", e.Media)
	case *Note:
		events(e.Events)
		each("media", e.Media)
	}
}

// Pictures hands every media entry MediaLists finds to a function that may change it,
// with the path to it.
func Pictures(entity any, each func(at string, m *Media)) {
	MediaLists(entity, func(at string, list []Media) {
		for i := range list {
			each(fmt.Sprintf("%s[%d]", at, i), &list[i])
		}
	})
}

// Keeps reports whether a record has an entry of its own for a file beside it, in its
// own media or an event's.
func Keeps(entity any, file string) bool {
	from, kept := RefOf(entity), false
	Pictures(entity, func(_ string, m *Media) {
		if !m.Refers() {
			at := Locate(from, m.File)
			kept = kept || at.Record == from && at.File == file
		}
	})
	return kept
}

// Depicted is somebody a picture shows, and the part of it they are where that is
// marked.
type Depicted struct {
	Person     Slug
	Region     *Region
	Attributes []Attribute
	Note       string
}

// Qualified reports whether the entry says more than who it is.
func (d Depicted) Qualified() bool {
	return d.Region != nil || len(d.Attributes) > 0 || d.Note != ""
}

// Scans is every scan a source's citations keep, by anchor in sorted order and then in
// the order each citation lists them, with the path to each.
func (s *Source) Scans(each func(at string, m *Media)) {
	for _, anchor := range slices.Sorted(maps.Keys(s.Citations)) {
		// a citation is held in the map by value, and its images in a list the copy
		// shares with it
		images := s.Citations[anchor].Images
		for i := range images {
			each(fmt.Sprintf("citations.%s.images[%d]", anchor, i), &images[i])
		}
	}
}

// Depiction is a picture somebody is in: the record keeping it, the path to its media
// entry and which of that entry's depicts, the file, and the part of it they are.
type Depiction struct {
	Record   EntityRef
	Media    string
	Depicted int
	File     string
	Region   *Region
}

// Depicting returns every picture that depicts person, wherever it is kept, in the
// order of the record's slug and then of where in the record it is written.
func (c *Corpus) Depicting(person Slug) []Depiction {
	return c.depictions(func(p Slug) bool { return p == person })[person]
}

// depictions walks every picture once and hands back, for each person wanted, the
// pictures depicting them in the order Depicting gives.
func (c *Corpus) depictions(wanted func(Slug) bool) map[Slug][]Depiction {
	out := map[Slug][]Depiction{}
	for _, slug := range c.AllSlugs() {
		entity := c.Entity(slug)
		Pictures(entity, func(at string, media *Media) {
			for j, depicted := range media.Depicts {
				if wanted(depicted.Person) {
					out[depicted.Person] = append(out[depicted.Person], Depiction{Record: RefOf(entity), Media: at,
						Depicted: j, File: media.File, Region: depicted.Region})
				}
			}
		})
	}
	return out
}

func cloneMedia(list []Media) []Media {
	out := slices.Clone(list)
	for i := range out {
		out[i].Region = cloneRegion(list[i].Region)
		out[i].Depicts = slices.Clone(list[i].Depicts)
		for j, depicted := range out[i].Depicts {
			out[i].Depicts[j].Region = cloneRegion(depicted.Region)
			out[i].Depicts[j].Attributes = cloneAttributes(depicted.Attributes)
		}
		out[i].Attributes = cloneAttributes(list[i].Attributes)
		out[i].Sources = slices.Clone(list[i].Sources)
		out[i].Tags = slices.Clone(list[i].Tags)
		out[i].IDs = cloneIDs(list[i].IDs)
	}
	return out
}

func cloneRegion(r *Region) *Region {
	if r == nil {
		return nil
	}
	copied := *r
	return &copied
}

// Portrait is the picture a person is shown by: a file kept beside Record, and the part
// of it that is them where that is marked.
type Portrait struct {
	Record EntityRef
	File   string
	Region *Region
}

// PortraitOf chooses the picture a person is shown by, wherever they are shown: the
// record's own portrait, else the first image under photos/, else the first picture
// anywhere that depicts them, cropped to them. A file that is not beside the record
// keeping it is passed over, there being nothing to show.
func (c *Corpus) PortraitOf(person Slug) (Portrait, bool) {
	p, ok := c.People[person]
	if !ok {
		return Portrait{}, false
	}
	return c.portrait(person, p, func() []Depiction { return c.Depicting(person) })
}

// Portraits is PortraitOf for everybody who has a portrait, looking through the
// pictures once rather than once a person.
func (c *Corpus) Portraits() map[Slug]Portrait {
	var depicted map[Slug][]Depiction
	out := map[Slug]Portrait{}
	for slug, p := range c.People {
		found, ok := c.portrait(slug, p, func() []Depiction {
			if depicted == nil {
				depicted = c.depictions(func(Slug) bool { return true })
			}
			return depicted[slug]
		})
		if ok {
			out[slug] = found
		}
	}
	return out
}

func (c *Corpus) portrait(person Slug, p *Person, depicting func() []Depiction) (Portrait, bool) {
	self := EntityRef{Kind: KindPerson, Slug: person}
	if p.Portrait.File != "" && slices.Contains(p.Files, p.Portrait.File) {
		return Portrait{Record: self, File: p.Portrait.File, Region: cloneRegion(p.Portrait.Region)}, true
	}
	for _, file := range p.Files {
		if strings.HasPrefix(file, "photos/") && Picture(file) {
			return Portrait{Record: self, File: file}, true
		}
	}
	for _, d := range depicting() {
		at := Locate(d.Record, d.File)
		if at.Abroad || at.Record.Slug == "" || !Picture(at.File) {
			continue
		}
		if files := Files(c.Entity(at.Record.Slug)); files == nil || !slices.Contains(*files, at.File) {
			continue
		}
		return Portrait{Record: at.Record, File: at.File, Region: cloneRegion(d.Region)}, true
	}
	return Portrait{}, false
}

// Picture reports whether a file is an image, by what its name ends in.
func Picture(file string) bool {
	switch strings.ToLower(path.Ext(file)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".bmp", ".tif", ".tiff", ".heic", ".svg":
		return true
	}
	return false
}
