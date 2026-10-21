package gramps

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// kept is where a media object is kept: by the first record to name it, as one entry of
// that record's media or of one of its events'. self says the record named it by a
// reference of its own, rather than through an event it holds, which is what makes a
// person keeping a photograph somebody in it.
type kept struct {
	entity any // the record, which may not be in the corpus yet
	record model.EntityRef
	event  int // which of the record's events holds it, -1 for the record itself
	entry  int
	file   string
	photo  bool
	self   bool
}

// list is the list of pictures the entry is in.
func (k *kept) list() *[]model.Media {
	if k.event < 0 {
		return importing.MediaOf(k.entity)
	}
	return &(*eventsOf(k.entity))[k.event].Media
}

// note is the path of the note on the entry a picture is.
func (k *kept) note() string { return mediaPath(k.event, k.entry) + ".note" }

// path is the picture's file as a record refers to it: as it is named where the record
// is the one keeping it, and otherwise by its path into the keeper's directory, a file
// that is not there being named as the export named it.
func (k *kept) path(from model.EntityRef) string {
	if from == k.record || model.Locate(k.record, k.file).Record != k.record {
		return k.file
	}
	return "../../" + k.record.Kind.Dir() + "/" + string(k.record.Slug) + "/" + k.file
}

// mediaPath is the path of an entry of a record's media, or of its event's at event.
func mediaPath(event, entry int) string {
	if event < 0 {
		return fmt.Sprintf("media[%d]", entry)
	}
	return fmt.Sprintf("events[%d].media[%d]", event, entry)
}

// eventsOf is the list a record keeps its events in.
func eventsOf(entity any) *[]model.Event {
	switch e := entity.(type) {
	case *model.Person:
		return &e.Events
	case *model.Family:
		return &e.Events
	case *model.Note:
		return &e.Events
	}
	panic(fmt.Sprintf("%T holds no events", entity))
}

// pictures reads the media an element names into list, the media of entity or of its
// event at event.
//
// A media object is kept once, by the first record to name it. Every other person
// naming a photograph is somebody in it, which is what naming it says, and so is the
// person keeping it once anybody else is. A picture named through an event is kept in
// the event of the first record holding it, and says nothing about who is in it. Every
// other record, and every other holder of the event, has an entry with the path to it,
// carrying what its own reference says.
func (c *converter) pictures(e *Element, entity any, list *[]model.Media, event int) {
	owner := model.RefOf(entity)
	depicting := owner.Kind == model.KindPerson && event < 0
	for _, ref := range e.All("objref") {
		object := c.object(ref, "object")
		if object == nil {
			c.notes.Add("a picture was referred to that is not in the file")
			continue
		}
		shared, ok := c.kept[object.Attr("handle")]
		switch {
		case !ok:
			c.keep(ref, object, entity, list, event, depicting)
		case shared.record == owner && shared.event == event:
			ref.Mark()
			c.home(ref, owner.Slug, shared.note())
		case shared.photo && depicting:
			ref.Mark()
			at := c.depict(shared, owner.Slug, ref)
			c.home(ref, shared.record.Slug, fmt.Sprintf("%s.depicts[%d].note", mediaPath(shared.event, shared.entry), at))
		default:
			ref.Mark()
			entry := model.Media{File: shared.path(owner), Region: c.region(ref)}
			c.reference(ref, &entry)
			*list = append(*list, entry)
			c.home(ref, owner.Slug, mediaPath(event, len(*list)-1)+".note")
		}
	}
}

// reference adds to an entry what a reference to a picture says of it: its privacy, its
// attributes, its notes and its citations.
func (c *converter) reference(ref *Element, entry *model.Media) {
	entry.Private = entry.Private || private(ref)
	for _, attribute := range ref.All("attribute") {
		entry.Attributes = append(entry.Attributes, c.said(attribute, &entry.Private))
	}
	if notes := c.prose(ref); len(notes) > 0 {
		entry.Note = strings.TrimPrefix(entry.Note+"\n\n"+strings.Join(notes, "\n\n"), "\n\n")
	}
	entry.Sources = append(entry.Sources, c.citations(ref)...)
}

// keep makes a media object an entry of list, attaching its file where it is found and
// naming it by the path the export gave where it is not. The reference naming it may be
// nil.
func (c *converter) keep(ref, object *Element, entity any, list *[]model.Media, event int, depicting bool) *kept {
	object.Mark()
	file := object.Take("file")
	src := strings.TrimSpace(file.Use("src"))
	file.Use("checksum")
	photo := strings.HasPrefix(strings.ToLower(file.Use("mime")), "image/")
	owner := model.RefOf(entity)
	entry := model.Media{
		Private: private(object),
		Title:   collapse(file.Use("description")),
		Date:    c.date(object),
		IDs:     c.grampsID(object),
		Note:    strings.Join(c.prose(object), "\n\n"),
		Sources: c.citations(object),
		Tags:    c.tags(object),
	}
	for _, attribute := range object.All("attribute") {
		entry.Attributes = append(entry.Attributes, c.said(attribute, &entry.Private))
	}
	if found, ok := c.find(src); ok {
		folder := "photos"
		switch {
		case owner.Kind == model.KindSource:
			folder = "scans"
		case !photo:
			folder = "documents"
		}
		entry.File, _ = c.files.Attach(entity, folder, found)
	} else {
		c.notes.Add("a picture the export names was not found where Gramps would look for it, and is kept as the path it gave")
		entry.File = src
	}
	if ref != nil {
		ref.Mark()
		region := c.region(ref)
		if depicting && photo && region != nil {
			entry.Depicts = []model.Depicted{{Person: owner.Slug, Region: region}}
		} else {
			entry.Region = region
		}
		c.reference(ref, &entry)
	}
	*list = append(*list, entry)
	picture := &kept{entity: entity, record: owner, event: event, entry: len(*list) - 1, file: entry.File, photo: photo, self: depicting}
	c.kept[object.Attr("handle")] = picture
	c.home(object, owner.Slug, picture.note())
	c.home(ref, owner.Slug, picture.note())
	return picture
}

// depict adds somebody to the photograph shared is, with the part of it their reference
// marks and what it says of them, and the person keeping it beside them where the keeper
// named it as a picture of themselves and is not yet said to be in it. It is where in
// the people the photograph depicts the person is. A private reference makes the
// photograph private, the person in it having no flag of their own.
func (c *converter) depict(shared *kept, person model.Slug, ref *Element) int {
	entry := &(*shared.list())[shared.entry]
	if shared.self && !depicts(*entry, shared.record.Slug) {
		entry.Depicts = append([]model.Depicted{{Person: shared.record.Slug}}, entry.Depicts...)
	}
	if at := slices.IndexFunc(entry.Depicts, func(d model.Depicted) bool { return d.Person == person }); at >= 0 {
		return at
	}
	depicted := model.Depicted{Person: person, Region: c.region(ref), Note: strings.Join(c.prose(ref), "\n\n")}
	for _, attribute := range ref.All("attribute") {
		depicted.Attributes = append(depicted.Attributes, c.said(attribute, &entry.Private))
	}
	if private(ref) && !entry.Private {
		entry.Private = true
		c.notes.Add("a private reference to a photograph made the photograph private")
	}
	entry.Depicts = append(entry.Depicts, depicted)
	return len(entry.Depicts) - 1
}

func depicts(entry model.Media, person model.Slug) bool {
	return slices.ContainsFunc(entry.Depicts, func(d model.Depicted) bool { return d.Person == person })
}

// scan reads a picture a citation names, which is a scan kept by the citation's source
// where nothing keeps it yet, and the part of it the entry is in where the reference
// marks one; what is unread in the reference goes to path at in the source.
func (c *converter) scan(ref *Element, source *model.Source, at string) (model.Media, bool) {
	object := c.object(ref, "object")
	if object == nil {
		c.notes.Add("a picture was referred to that is not in the file")
		return model.Media{}, false
	}
	shared, ok := c.kept[object.Attr("handle")]
	if !ok {
		shared = c.keep(nil, object, source, &source.Media, -1, false)
	}
	ref.Mark()
	c.home(ref, source.Slug, at)
	image := model.Media{File: shared.path(model.RefOf(source)), Region: c.region(ref)}
	c.reference(ref, &image)
	return image, true
}

// eventPictures reads the pictures an event names into a record's copy of it, the event
// at index among the record's.
func (c *converter) eventPictures(handle string, entity any, event *model.Event, index int) {
	if e := c.handles[handle]; e != nil {
		c.pictures(e, entity, &event.Media, index)
	}
}

// unreferred makes every media object nothing refers to a note keeping the picture,
// titled with its description or its file's name, since a picture is kept by a record
// and this one has no other.
func (c *converter) unreferred() {
	for _, object := range c.all("objects", "object") {
		if _, ok := c.kept[object.Attr("handle")]; ok {
			continue
		}
		file := object.First("file")
		heading := collapse(file.Attr("description"))
		if heading == "" {
			name := strings.TrimSpace(file.Attr("src"))
			heading = name[strings.LastIndexAny(name, `/\`)+1:]
		}
		if heading == "" {
			heading = title(object)
		}
		base, err := model.Slugify(heading)
		if err != nil {
			base = "picture"
		}
		slug := c.namer.Assign(importing.Shorten(base), nil, heading)
		if slug == "" {
			continue
		}
		note := &model.Note{Slug: slug, Title: heading}
		c.corpus.Notes[slug] = note
		c.home(object, slug, "body")
		c.keep(nil, object, note, &note.Media, -1, false)
	}
}

// corners are the attributes Gramps gives a region in.
var corners = []string{"corner1_x", "corner1_y", "corner2_x", "corner2_y"}

// region is the part of a picture a reference marks, which Gramps gives as two corners
// in percentages of the picture's width and height, and the model as fractions of it.
// A region that cannot be read, or that marks none of the picture, is left unread.
func (c *converter) region(ref *Element) *model.Region {
	e := ref.First("region")
	if e == nil {
		return nil
	}
	var at [4]float64
	for i, name := range corners {
		n, err := strconv.Atoi(strings.TrimSpace(e.Attr(name)))
		if err != nil || n < 0 || n > 100 {
			return nil
		}
		at[i] = float64(n) / 100
	}
	region := &model.Region{
		Left: min(at[0], at[2]), Top: min(at[1], at[3]),
		Right: max(at[0], at[2]), Bottom: max(at[1], at[3]),
	}
	if region.Left == region.Right || region.Top == region.Bottom {
		return nil
	}
	e.Mark()
	for _, name := range corners {
		e.Use(name)
	}
	return region
}
