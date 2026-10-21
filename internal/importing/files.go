package importing

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Finder says where on disk a file an export names is, or that it is nowhere to be
// found. An export names its pictures and may not contain them, so a picture comes in
// only where the file it names is still to hand.
type Finder func(name string) (path string, found bool)

// Beside finds what an export names where its own references lead: at the path it gives,
// or relative to dir, the directory the export is in, which is how GEDCOM 7.0 reads one
// and how an export carried to another machine with its pictures beside it still reads.
func Beside(dir string) Finder {
	return func(name string) (string, bool) {
		name = strings.TrimSpace(name)
		if address, err := url.Parse(name); err == nil && address.Scheme == "file" {
			name = address.Path
			// file:///C:/photos names a Windows path, which has no slash in front
			if len(name) > 2 && name[0] == '/' && name[2] == ':' {
				name = name[1:]
			}
		}
		var candidates []string
		if filepath.IsAbs(name) {
			candidates = append(candidates, name)
		}
		candidates = append(candidates, filepath.Join(dir, filepath.FromSlash(strings.ReplaceAll(name, `\`, "/"))))
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
				return candidate, true
			}
		}
		return "", false
	}
}

// Among finds what an export names as Beside does, in each of dirs in turn, the first
// to have it answering.
func Among(dirs []string) Finder {
	return func(name string) (string, bool) {
		for _, dir := range dirs {
			if found, ok := Beside(dir)(name); ok {
				return found, true
			}
		}
		return "", false
	}
}

// Attachment is a file an export names that was found, and the record it is attached
// to: File is its path relative to the record's directory, and From where it was found.
type Attachment struct {
	Record model.EntityRef
	File   string
	From   string
}

// Attacher attaches the files an export names to the records they belong to, and keeps
// the list of what it attached for the caller to copy in.
type Attacher struct {
	found       map[string]string // a record and an attached file's path in it, to where it was found
	attachments []Attachment
}

// NewAttacher is an attacher that has attached nothing yet.
func NewAttacher() *Attacher { return &Attacher{found: map[string]string{}} }

// Attach attaches the file found on disk to a record, under folder, and reports whether
// it is new to the record: one file named twice by the same record is attached once.
// Two different files of the same name are told apart by a number, as slugs are.
func (a *Attacher) Attach(entity any, folder, found string) (string, bool) {
	ref := model.RefOf(entity)
	base := filepath.Base(found)
	extension := path.Ext(base)
	for candidate := 1; ; candidate++ {
		name := path.Join(folder, base)
		if candidate > 1 {
			name = path.Join(folder, strings.TrimSuffix(base, extension)+"-"+strconv.Itoa(candidate)+extension)
		}
		key := path.Join(ref.String(), name)
		switch existing, taken := a.found[key]; {
		case taken && existing == found:
			return name, false
		case taken:
			continue
		}
		a.found[key] = found
		a.attachments = append(a.attachments, Attachment{Record: ref, File: name, From: found})
		files := model.Files(entity)
		*files = append(*files, name)
		slices.Sort(*files)
		return name, true
	}
}

// Found is where on disk the file a record has attached as name was found.
func (a *Attacher) Found(record model.EntityRef, name string) string {
	return a.found[path.Join(record.String(), name)]
}

// List is everything attached, in the order it was.
func (a *Attacher) List() []Attachment { return a.attachments }

// MediaOf is the list a record keeps its media in, for adding to.
func MediaOf(entity any) *[]model.Media {
	switch e := entity.(type) {
	case *model.Person:
		return &e.Media
	case *model.Family:
		return &e.Media
	case *model.Place:
		return &e.Media
	case *model.Source:
		return &e.Media
	case *model.Repository:
		return &e.Media
	case *model.Note:
		return &e.Media
	}
	panic(fmt.Sprintf("%T keeps no media", entity))
}
