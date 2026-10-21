package gedcom

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"maps"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// Chooser, Finder and Attachment are the shared importer's, which Convert takes and
// hands back.
type (
	Chooser    = importing.Chooser
	Finder     = importing.Finder
	Attachment = importing.Attachment
)

// Beside finds what an export names at the path it gives or relative to dir, the
// directory the export is in.
func Beside(dir string) Finder { return importing.Beside(dir) }

// kept is where a picture GEDCOM shares between records as an OBJE record is kept: by
// the first record to name it, in its own media or an event's, as the entries listed in
// entries, one for each of its files.
type kept struct {
	entity  any // the record, which may not be in the corpus yet
	record  model.EntityRef
	event   int // which of the record's events holds it, -1 for the record itself
	entries []int
	photo   bool
}

// list is the list of pictures the entries are in.
func (k *kept) list() *[]model.Media {
	if k.event < 0 {
		return importing.MediaOf(k.entity)
	}
	return &(*eventsOf(k.entity))[k.event].Media
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

// filePath is where a FILE says its file is: in 7.0 a URI reference, a path with no
// scheme escaped as a URI's is, and in 5.5.1 a path as the exporter's computer wrote it.
func (c *converter) filePath(file *Node) string {
	path := strings.TrimSpace(file.Text())
	if !c.seven {
		return path
	}
	if address, err := url.Parse(path); err == nil && address.Scheme == "" {
		if unescaped, err := url.PathUnescape(path); err == nil {
			return unescaped
		}
	}
	return path
}

// readMedia reads the OBJE a record names into its media, and what each picture shows of
// it where the picture is of a person.
func (c *converter) readMedia(record *Node, entity any) {
	c.readPictures(record, entity, -1, importing.MediaOf(entity), "")
}

// readPictures reads the OBJE under node into list, the pictures of entity or of its event
// at event, whose paths start at.
//
// A picture is a file attached to one record, and a GEDCOM shares one among several by
// pointing each at the same OBJE record. The first record to name it keeps it; every
// other person naming a photograph is somebody in it, which is what naming it says, and
// so is the keeper once anybody else is; and every other record, or event, has an entry
// with the path to it, which holds only what its own OBJE says, the region its CROP
// marks.
func (c *converter) readPictures(node *Node, entity any, event int, list *[]model.Media, at string) {
	ref := model.RefOf(entity)
	for _, obje := range node.All("OBJE") {
		if voided(obje.Value) {
			continue
		}
		object, xref := obje, pointer(obje.Value)
		if xref != "" {
			if object = c.xref[xref]; object == nil {
				c.notes.Add("a record names picture %s, which is not in the file", xref)
				continue
			}
			c.referred[xref] = true
		}
		shared, ok := c.pictures[xref]
		switch {
		case !ok && xref != "" && c.keptElsewhere(object, xref, ref):
			// what this record says of the picture is read once its keeper has read it,
			// an entry referring to it holding its place in the list meanwhile
			obje.Mark()
			waiter := &kept{entity: entity, record: ref, event: event}
			slot := -1
			if !(c.photograph(object) && ref.Kind == model.KindPerson && event < 0) {
				*list = append(*list, model.Media{})
				slot = len(*list) - 1
			}
			c.waiting[xref] = append(c.waiting[xref], waiting{obje, object, waiter, at, slot})
		case !ok || xref == "":
			if picture := c.keep(obje, object, entity, list, at, true); picture != nil && xref != "" {
				picture.event = event
				c.pictures[xref] = picture
				for _, w := range c.waiting[xref] {
					c.share(picture, w.obje, w.at, w.path, w.slot)
				}
				delete(c.waiting, xref)
			}
		default:
			c.share(shared, obje, &kept{entity: entity, record: ref, event: event}, at, -1)
		}
	}
}

// share reads what obje says of a picture shared kept by another record or another
// event, from the record or event named by at, whose paths start at: whoever is in a
// photograph where at is a person, and otherwise an entry referring to it, which holds
// only the region its CROP marks, put at slot in the list where one was kept for it.
func (c *converter) share(shared *kept, obje *Node, at *kept, path string, slot int) {
	ref := at.record
	obje.Mark()
	switch {
	case shared.record == ref && shared.event == at.event:
		return
	case shared.photo && ref.Kind == model.KindPerson && at.event < 0 && slot < 0:
		c.depict(shared, ref.Slug, obje)
		return
	}
	list := at.list()
	for _, i := range shared.entries {
		keeper := (*shared.list())[i]
		entry := model.Media{File: keeper.File}
		if model.Locate(shared.record, keeper.File).Record == shared.record && shared.record != ref {
			entry.File = "../../" + shared.record.Kind.Dir() + "/" + string(shared.record.Slug) + "/" + keeper.File
		}
		c.cropped(&entry.Region, &entry.Attributes, obje, c.files.Found(shared.record, keeper.File))
		index := slot
		if slot >= 0 {
			(*list)[slot], slot = entry, -1
		} else {
			*list = append(*list, entry)
			index = len(*list) - 1
		}
		c.home(obje, ref.Slug, fmt.Sprintf("%smedia[%d].note", path, index))
	}
}

// keptElsewhere says whether origins wrote that a picture is kept by a record other than
// ref, one that names it and is still to be read; GEDCOM itself saying nothing of who
// keeps a picture, the first record naming it is taken to otherwise.
func (c *converter) keptElsewhere(object *Node, xref string, ref model.EntityRef) bool {
	line := c.take(object, "_KEEPER")
	if line == nil {
		return false
	}
	keeper := pointer(line.Value)
	if slug, ok := c.slugOf(keeper); ok && slug == ref.Slug {
		return false
	}
	var names func(*Node) bool
	names = func(n *Node) bool {
		if n.Tag == "OBJE" && pointer(n.Value) == xref {
			return true
		}
		return slices.ContainsFunc(n.Children, names)
	}
	record := c.xref[keeper]
	if record == nil || !names(record) {
		c.notes.Add("a picture is said to be kept by %s, which does not name it", keeper)
		return false
	}
	return true
}

// photograph says whether a picture is a photograph, as every file of it is.
func (c *converter) photograph(object *Node) bool {
	for _, file := range object.All("FILE") {
		if c.filePath(file) == "" {
			continue
		}
		if _, photo := c.mediumOf(file); !photo {
			return false
		}
	}
	return true
}

// keep reads a picture into list, an entry for each of its files, and says where it is
// kept, or nil where nothing in it was read. obje is the line naming it and object what
// it names, the same line where the picture is written in place. A file that is not to
// hand is an entry all the same, its file the path the export gave, unless a TRAN of it
// is, another rendering of the same picture; where the file is found, a TRAN is a copy of
// what is there already. A file named twice by one record is one entry, save among a
// citation's scans, each citing of an entry naming its own.
//
// A picture can cite the source a scan of which it is, and a file that makes the two
// point at each other is one 7.0 forbids and says to read safely: a picture being read
// already is not read again from inside itself.
func (c *converter) keep(obje, object *Node, entity any, list *[]model.Media, at string, once bool) *kept {
	if c.reading[object] {
		return nil
	}
	c.reading[object] = true
	defer delete(c.reading, object)
	ref := model.RefOf(entity)
	title := firstText(obje, "TITL")
	sources := c.citations(object, ref.Slug, "media")
	note := strings.Join(c.noteTexts(object), "\n\n")
	if object != obje {
		if more := c.noteTexts(obje); len(more) > 0 {
			note = strings.Join(append([]string{note}, more...), "\n\n")
			note = strings.TrimPrefix(note, "\n\n")
		}
	}
	private := c.private(object)
	ids := c.ids(object)
	date, read := readDateNode(c.take(object, "_DATE"))
	if !read {
		c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
	}
	var place model.Slug
	if at := c.take(object, "_PLACE"); at != nil {
		place = c.place(at.First("PLAC"))
	}
	attributes := c.attributes(object, ref.Slug)
	tags := c.tagged(object)
	c.take(object, "_KEEPER") // which entity is reading it says who keeps it now
	picture := &kept{entity: entity, record: ref, event: -1, photo: true}
	for _, file := range object.All("FILE") {
		path := c.filePath(file)
		if path == "" {
			continue
		}
		kind, photo := c.mediumOf(file)
		found, ok := c.find(path)
		for _, tran := range file.All("TRAN") {
			if !ok {
				if found, ok = c.find(tran.Text()); !ok {
					continue
				}
			}
			tran.Mark().Take("FORM")
		}
		file.Mark()
		entry := model.Media{Private: private, Type: kind, Title: title, Date: date, Place: place,
			Attributes: slices.Clone(attributes), Note: note, Sources: slices.Clone(sources),
			Tags: slices.Clone(tags), IDs: maps.Clone(ids)}
		if ok {
			folder := "photos"
			switch {
			case ref.Kind == model.KindSource:
				folder = "scans"
			case !photo:
				folder = "documents"
			}
			if c.mine {
				folder = c.folderOf(path, folder)
			}
			name, fresh := c.files.Attach(entity, folder, found)
			if !fresh && once {
				continue
			}
			entry.File = name
		} else {
			c.notes.Add("a picture the export names was not found beside it, and is kept as the path it gave")
			entry.File = path
		}
		var region *model.Region
		c.cropped(&region, &entry.Attributes, obje, found)
		if region != nil && ref.Kind == model.KindPerson && photo && at == "" {
			entry.Depicts = []model.Depicted{{Person: ref.Slug, Region: region}}
		} else {
			entry.Region = region
		}
		if entry.Title == "" {
			entry.Title = firstText(file, "TITL")
		}
		if entry.Title == "" {
			entry.Title = firstText(object, "TITL")
		}
		*list = append(*list, entry)
		picture.entries = append(picture.entries, len(*list)-1)
		picture.photo = picture.photo && photo
		// a citation's scan is homed by the citation, whose anchor is not named yet
		var home string
		if at != "images" {
			home = fmt.Sprintf("%smedia[%d].note", at, len(*list)-1)
		}
		c.home(obje, ref.Slug, home)
		c.home(object, ref.Slug, home)
	}
	if len(picture.entries) == 0 {
		return nil
	}
	obje.Mark()
	object.Mark()
	return picture
}

// cropped reads the CROP under the line naming a picture as the region it marks, where
// the picture it is of was found at found and its size can be read, and as an attribute
// of the entry in the file's words where it cannot. The fractions origins writes where
// the picture's size was not known, or its pixels too few to say them, are the region
// over any CROP beside them.
func (c *converter) cropped(region **model.Region, attributes *[]model.Attribute, obje *Node, found string) {
	crop := obje.First("CROP")
	if *region = c.fractions(obje); *region != nil || crop == nil {
		if *region != nil && crop != nil {
			crop.markAll()
		}
		return
	}
	if found != "" {
		if *region = c.region(crop, found); *region != nil {
			return
		}
	}
	var words []string
	for _, line := range crop.Children {
		if value := collapse(line.Value); value != "" {
			words = append(words, line.Tag+" "+value)
		}
	}
	if len(words) == 0 {
		return
	}
	crop.markAll()
	*attributes = append(*attributes, model.Attribute{Type: "crop", Value: strings.Join(words, ", ")})
}

// folderOf is the folder of its record's directory a file origins wrote is in, which the
// path it wrote names after the record's kind and slug, or otherwise where none is named.
func (c *converter) folderOf(file, otherwise string) string {
	parts := strings.SplitN(file, "/", 3)
	if len(parts) < 3 || !slices.ContainsFunc(model.Kinds, func(k model.Kind) bool { return k.Dir() == parts[0] }) {
		return otherwise
	}
	if i := strings.LastIndex(parts[2], "/"); i >= 0 {
		return parts[2][:i]
	}
	return ""
}

// waiting is a record naming a picture before the record origins said keeps it: the
// line naming it, what it names, the record or event naming it and where its paths
// start, and the place kept in its list for the entry referring to it, or -1.
type waiting struct {
	obje, object *Node
	at           *kept
	path         string
	slot         int
}

// unwaited reads what every record naming a picture said of it where the record said to
// keep it was never read, as though nobody had said so: the first of them keeps it and
// the others share it, each place kept in a list for an entry taken out again.
func (c *converter) unwaited() {
	var slots []waiting
	for _, xref := range slices.Sorted(maps.Keys(c.waiting)) {
		for _, w := range c.waiting[xref] {
			if w.slot >= 0 {
				slots = append(slots, w)
			}
		}
	}
	// the places kept go from the last, so that each is where it was kept
	slices.SortFunc(slots, func(a, b waiting) int { return b.slot - a.slot })
	for _, w := range slots {
		list := w.at.list()
		*list = slices.Delete(*list, w.slot, w.slot+1)
	}
	for _, xref := range slices.Sorted(maps.Keys(c.waiting)) {
		waiters := c.waiting[xref]
		c.notes.Add("picture %s is said to be kept by a record that was not read, and is kept by the first record naming it", xref)
		var shared *kept
		for _, w := range waiters {
			if shared == nil {
				if shared = c.keep(w.obje, w.object, w.at.entity, w.at.list(), w.path, true); shared != nil {
					shared.event = w.at.event
					c.pictures[xref] = shared
				}
				continue
			}
			c.share(shared, w.obje, w.at, w.path, -1)
		}
	}
	c.waiting = nil
}

// scans reads the OBJE under a citation as its images, kept by the source: under its
// scans, or as the path the export gave where the file is not to hand.
func (c *converter) scans(sour *Node, source model.Slug) []model.Media {
	var images []model.Media
	entity := c.corpus.Sources[source]
	for _, obje := range sour.All("OBJE") {
		if voided(obje.Value) {
			continue
		}
		object := obje
		if xref := pointer(obje.Value); xref != "" {
			if object = c.xref[xref]; object == nil {
				c.notes.Add("a record names picture %s, which is not in the file", xref)
				continue
			}
			c.referred[xref] = true
		}
		c.keep(obje, object, entity, &images, "images", false)
	}
	return images
}

// unreferred makes every OBJE record nothing points at a note keeping the picture,
// titled as the picture is or with its file's name, since a picture is kept by a record
// and this one has no other.
func (c *converter) unreferred(records []*Node) {
	for _, record := range records {
		if record.Tag != "OBJE" || record.Xref == "" || c.referred[record.Xref] {
			continue
		}
		heading := collapse(record.First("TITL").Text())
		for _, file := range record.All("FILE") {
			if heading != "" {
				break
			}
			if heading = collapse(file.First("TITL").Text()); heading == "" {
				name := c.filePath(file)
				heading = name[strings.LastIndexAny(name, `/\`)+1:]
			}
		}
		if heading == "" {
			heading = title(record)
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
		c.citer = slug
		c.home(record, slug, "body")
		c.keep(record, record, note, &note.Media, "", true)
	}
}

// depict adds somebody to the photograph kept by shared, and its keeper beside them where
// the keeper is a person not yet said to be in it.
func (c *converter) depict(shared *kept, person model.Slug, obje *Node) {
	list := shared.list()
	for _, i := range shared.entries {
		entry := &(*list)[i]
		if shared.record.Kind == model.KindPerson && !depicts(*entry, shared.record.Slug) {
			entry.Depicts = append([]model.Depicted{{Person: shared.record.Slug}}, entry.Depicts...)
		}
		if depicts(*entry, person) {
			continue
		}
		depicted := model.Depicted{Person: person}
		c.cropped(&depicted.Region, &depicted.Attributes, obje, c.files.Found(shared.record, entry.File))
		entry.Depicts = append(entry.Depicts, depicted)
	}
}

func depicts(entry model.Media, person model.Slug) bool {
	return slices.ContainsFunc(entry.Depicts, func(d model.Depicted) bool { return d.Person == person })
}

// media are the words for GEDCOM's media, where they are not the word in lower case.
var media = map[string]string{"FILM": "microfilm", "FICHE": "microfiche"}

// medium reads a MEDI in origins' words, OTHER as its PHRASE says, marking what it read.
// One it cannot read is left unread, and so kept as written.
func medium(medi *Node) string {
	word := strings.ToUpper(collapse(medi.Text()))
	if word == "" {
		return ""
	}
	if word == "OTHER" {
		if phrase := collapse(medi.First("PHRASE").Text()); phrase != "" {
			medi.Mark().Take("PHRASE")
			return phrase
		}
	}
	medi.Mark()
	if spelled, ok := media[word]; ok {
		return spelled
	}
	return strings.ToLower(collapse(medi.Text()))
}

// mediumOf reads what a FILE says it is, as 7.0's MEDI and 5.5.1's TYPE under FORM say,
// and whether that is a photograph, as a picture saying nothing about itself is taken
// to be.
func (c *converter) mediumOf(file *Node) (string, bool) {
	form := file.Take("FORM")
	medi := form.First("MEDI")
	if medi == nil {
		medi = form.First("TYPE")
	}
	kind := medium(medi)
	return kind, kind == "" || kind == "photo"
}

// region is the part of a picture a 7.0 CROP marks, which it gives in pixels and the
// model as fractions of the picture, so that the region survives the picture being
// scaled. Turning one into the other takes the picture's size, which is read from the
// file. A CROP that gives a region is marked read, and any other is kept as written.
func (c *converter) region(crop *Node, found string) *model.Region {
	width, height, ok := measure(found)
	if !ok {
		return nil
	}
	pixels := func(tag string, otherwise int) (int, bool) {
		text := strings.TrimSpace(crop.First(tag).Text())
		if text == "" {
			return otherwise, true
		}
		n, err := strconv.Atoi(text)
		return n, err == nil && n >= 0
	}
	top, topRead := pixels("TOP", 0)
	left, leftRead := pixels("LEFT", 0)
	tall, tallRead := pixels("HEIGHT", height-top)
	wide, wideRead := pixels("WIDTH", width-left)
	if !topRead || !leftRead || !tallRead || !wideRead ||
		tall == 0 || wide == 0 || left+wide > width || top+tall > height {
		return nil
	}
	crop.markAll()
	return &model.Region{
		Left: fractionOf(left, width), Top: fractionOf(top, height),
		Right: fractionOf(left+wide, width), Bottom: fractionOf(top+tall, height),
	}
}

// measure reads the size of a picture in pixels.
func measure(found string) (width, height int, ok bool) {
	file, err := os.Open(found)
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()
	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width == 0 || config.Height == 0 {
		return 0, 0, false
	}
	return config.Width, config.Height, true
}
