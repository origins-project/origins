package api

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/origins-project/origins/internal/gedcom"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// GEDCOMAsked is the corpus asked to be written as a GEDZIP: the file, and the view of
// the corpus it is written from.
type GEDCOMAsked struct {
	Out    string
	Hiding model.Hiding
	With   []model.Slug
}

// GEDCOMWritten is what went into the GEDZIP: the records by tag, the extension tags by
// tag, the lines an import kept and written again as they were, what 7.0 and the
// extensions had nowhere for by the path of its field, the pictures put beside it, and
// those a FILE names that were not there to put in.
type GEDCOMWritten struct {
	Out        string         `json:"out"`
	Hiding     model.Hiding   `json:"hiding"`
	Records    map[string]int `json:"records"`
	Extensions map[string]int `json:"extensions"`
	Kept       map[string]int `json:"kept"`
	Omitted    map[string]int `json:"omitted"`
	Pictures   int            `json:"pictures"`
	Missing    []string       `json:"missing"`
	Read       []Supposition  `json:"read,omitempty"`
	Problems   []string       `json:"problems"`
}

// gedzipged is the name of the dataset in a GEDZIP, which the specification fixes.
const gedzipged = "gedcom.ged"

// ExportGEDCOM writes the hidden view as GEDCOM 7.0 in a GEDZIP: gedcom.ged, and every
// picture it names at the path into the zip its FILE gives, so that one file carries
// the tree and its pictures. A region of a picture is written in the pixels of the
// picture put beside it. The zip is written beside the file it replaces and renamed
// over it, so an export that fails leaves what was there.
func ExportGEDCOM(root string, asked GEDCOMAsked) (*GEDCOMWritten, error) {
	if asked.Out == "" {
		return nil, errors.New("a GEDZIP is written to a file, and --out names it")
	}
	q, err := asking(root, asked.With)
	if err != nil {
		return nil, err
	}
	defer q.close()
	view := q.corpus.Viewed(model.Hide(q.corpus.Corpus, asked.Hiding, today()))

	var dataset bytes.Buffer
	exported, err := gedcom.Write(&dataset, view.Corpus, gedcom.Options{
		Version: Version,
		Size: func(path string) (int, int, bool) {
			file, ok := onDisk(view, path)
			if !ok {
				return 0, 0, false
			}
			open, err := os.Open(file)
			if err != nil {
				return 0, 0, false
			}
			defer open.Close()
			config, _, err := image.DecodeConfig(open)
			return config.Width, config.Height, err == nil
		},
	})
	if err != nil {
		return nil, err
	}

	report := &GEDCOMWritten{
		Out:        asked.Out,
		Hiding:     asked.Hiding,
		Records:    exported.Records,
		Extensions: exported.Extensions,
		Kept:       exported.Kept,
		Omitted:    exported.Omitted,
		Missing:    []string{},
		Read:       q.read,
		Problems:   q.problems,
	}
	written, err := os.CreateTemp(filepath.Dir(asked.Out), "."+filepath.Base(asked.Out)+".*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(written.Name())
	zipped := zip.NewWriter(written)
	put := func(name string, from io.Reader) error {
		entry, err := zipped.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate})
		if err == nil {
			_, err = io.Copy(entry, from)
		}
		return err
	}
	err = put(gedzipged, &dataset)
	for _, path := range exported.Files {
		if err != nil {
			break
		}
		file, ok := onDisk(view, path)
		if !ok {
			report.Missing = append(report.Missing, path)
			continue
		}
		var open *os.File
		if open, err = os.Open(file); err != nil {
			break
		}
		err = put(path, open)
		open.Close()
		report.Pictures++
	}
	err = errors.Join(err, zipped.Close(), written.Close())
	if err != nil {
		return nil, err
	}
	if err := os.Rename(written.Name(), asked.Out); err != nil {
		return nil, err
	}
	return report, q.warning
}

// onDisk is the file a path from the root of the corpus leads to, where it is a file the
// view keeps beside one of its records.
func onDisk(view *store.Corpus, path string) (string, bool) {
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 3 || model.KindIn(parts[0]) == "" {
		return "", false
	}
	file, err := view.Attachment(model.Slug(parts[1]), parts[2])
	if err != nil {
		return "", false
	}
	if info, err := os.Stat(file); err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return file, true
}

// unzipped is a GEDZIP taken apart into a directory of its own, for its dataset to be
// read with its pictures beside it as its FILEs lead to them: the dataset's path, and
// what removes the directory again. An entry that would land outside the directory is
// refused, as is a zip with no gedcom.ged.
func unzipped(file string) (string, func() error, error) {
	archive, err := zip.OpenReader(file)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", file, err)
	}
	defer archive.Close()
	dir, err := os.MkdirTemp("", "origins-gedzip-")
	if err != nil {
		return "", nil, err
	}
	done := func() error { return os.RemoveAll(dir) }
	found := false
	for _, entry := range archive.File {
		if !filepath.IsLocal(entry.Name) {
			_ = done()
			return "", nil, fmt.Errorf("%s: %q is not a path within the zip", file, entry.Name)
		}
		if strings.HasSuffix(entry.Name, "/") {
			continue
		}
		found = found || entry.Name == gedzipged
		if err := extract(entry, filepath.Join(dir, filepath.FromSlash(entry.Name))); err != nil {
			_ = done()
			return "", nil, fmt.Errorf("%s: %w", file, err)
		}
	}
	if !found {
		_ = done()
		return "", nil, fmt.Errorf("%s is a zip with no %s in it, which a GEDZIP is", file, gedzipged)
	}
	return filepath.Join(dir, gedzipged), done, nil
}

func extract(entry *zip.File, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	from, err := entry.Open()
	if err != nil {
		return err
	}
	defer from.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, from)
	return errors.Join(err, out.Close())
}

// isZip says whether a file starts as a zip does, whatever it is called.
func isZip(file string) bool {
	open, err := os.Open(file)
	if err != nil {
		return false
	}
	defer open.Close()
	magic := make([]byte, 4)
	_, err = io.ReadFull(open, magic)
	return err == nil && string(magic) == "PK\x03\x04"
}
