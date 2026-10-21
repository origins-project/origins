// Package gramps reads the XML Gramps writes and maps it onto the model.
//
// Like gedcom, it is an import format and nothing of it escapes into the rest of
// origins. It shares with gedcom only what internal/importing holds, so nothing
// GEDCOM-shaped reaches a tree read from Gramps' own file.
package gramps

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// The versions of Gramps XML origins reads: what Gramps 5.1 writes, to what Gramps 6.1
// does. Structures moved between the older ones, and a current Gramps reads every one
// of them and writes it out again as current.
var (
	oldest = version{1, 7, 1}
	newest = version{1, 7, 2}
)

// Export is a Gramps file opened for reading.
//
// Media are the directories a relative path in the XML may be relative to, in the order
// they are looked in: in a package, the directory it was extracted into; otherwise the
// one the header's mediapath names, where it can be made out, and the one the file is
// in. A package's directory is removed by Close.
type Export struct {
	Root    *Element
	Version string
	Newer   bool
	Media   []string

	extracted string
}

// Close removes what Open extracted. It is safe to call on any Export.
func (x *Export) Close() error {
	if x == nil || x.extracted == "" {
		return nil
	}
	return os.RemoveAll(x.extracted)
}

// Open reads a Gramps export: plain XML, gzipped XML, or a package of gzipped XML and
// the media beside it. Which it is is told by what is in it rather than by its name,
// since packages are renamed as they are passed around.
//
// A file in a version older than origins reads is refused, naming the version; a newer
// one is read, and Newer says so.
func Open(path string) (*Export, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	content, err := ungzipped(file)
	if err != nil {
		return nil, err
	}
	head, _ := content.Peek(512)
	switch {
	case tarred(head):
		return openPackage(content)
	case xmlish(head):
		root, err := readXML(content)
		if err != nil {
			return nil, err
		}
		dir := filepath.Dir(path)
		var media []string
		if declared, ok := expand(strings.TrimSpace(root.First("header").First("mediapath").Text())); ok && declared != "" {
			if !filepath.IsAbs(declared) {
				declared = filepath.Join(dir, declared)
			}
			media = append(media, filepath.Clean(declared))
		}
		if !slices.Contains(media, dir) {
			media = append(media, dir)
		}
		return versioned(root, media, "")
	}
	return nil, errors.New("this is not a Gramps export: it is neither Gramps XML, gzipped or not, nor a Gramps package")
}

// expand fills in the {NAME} variables of a mediapath from the environment, as Gramps
// does, {{ and }} standing for the braces themselves. Gramps names its own installation
// as {GRAMPS_RESOURCES}, which is set where Gramps runs and seldom anywhere else, and a
// path naming a variable that is not set names no directory origins can find.
func expand(path string) (string, bool) {
	var out strings.Builder
	for {
		brace := strings.IndexAny(path, "{}")
		if brace < 0 {
			out.WriteString(path)
			return out.String(), true
		}
		out.WriteString(path[:brace])
		path = path[brace:]
		if strings.HasPrefix(path, "{{") || strings.HasPrefix(path, "}}") {
			out.WriteByte(path[0])
			path = path[2:]
			continue
		}
		end := strings.IndexByte(path, '}')
		if path[0] == '}' || end < 0 {
			return "", false
		}
		value, set := os.LookupEnv(path[1:end])
		if !set {
			return "", false
		}
		out.WriteString(value)
		path = path[end+1:]
	}
}

// openPackage extracts a package into a temporary directory and reads the data.gramps
// in it. Everything is extracted before the XML is read, because the XML may come last
// in the archive and the files it names have to be to hand when it is.
func openPackage(archive io.Reader) (*Export, error) {
	dir, err := os.MkdirTemp("", "origins-gramps-")
	if err != nil {
		return nil, err
	}
	export, err := func() (*Export, error) {
		if err := extract(archive, dir); err != nil {
			return nil, err
		}
		data, err := os.Open(filepath.Join(dir, "data.gramps"))
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("this Gramps package has no data.gramps in it")
		}
		if err != nil {
			return nil, err
		}
		defer data.Close()
		content, err := ungzipped(data)
		if err != nil {
			return nil, err
		}
		if head, _ := content.Peek(512); !xmlish(head) {
			return nil, errors.New("the data.gramps in this package is not Gramps XML")
		}
		root, err := readXML(content)
		if err != nil {
			return nil, err
		}
		return versioned(root, []string{dir}, dir)
	}()
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	return export, nil
}

// extract writes a tar archive's files into dir. An entry whose name would land
// outside dir is refused, and so is a link, which could lead anywhere.
func extract(archive io.Reader, dir string) error {
	entries := tar.NewReader(archive)
	for {
		header, err := entries.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading the Gramps package: %w", err)
		}
		if header.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		name := filepath.FromSlash(header.Name)
		if !filepath.IsLocal(name) {
			return fmt.Errorf("this Gramps package has an entry %q, which would be written outside it", header.Name)
		}
		target := filepath.Join(dir, name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := write(target, entries); err != nil {
				return err
			}
		default:
			return fmt.Errorf("this Gramps package has an entry %q that is neither a file nor a directory", header.Name)
		}
	}
}

func write(target string, content io.Reader) error {
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, content); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// ungzipped is what r holds, decompressed where it is gzipped.
func ungzipped(r io.Reader) (*bufio.Reader, error) {
	buffered := bufio.NewReader(r)
	if magic, _ := buffered.Peek(2); !bytes.Equal(magic, []byte{0x1f, 0x8b}) {
		return buffered, nil
	}
	unzipped, err := gzip.NewReader(buffered)
	if err != nil {
		return nil, fmt.Errorf("this is not a Gramps export: %w", err)
	}
	return bufio.NewReader(unzipped), nil
}

// tarred reports whether head begins a tar archive, by the magic every POSIX and GNU tar
// header carries.
func tarred(head []byte) bool {
	return len(head) >= 262 && string(head[257:262]) == "ustar"
}

// xmlish reports whether head begins an XML document.
func xmlish(head []byte) bool {
	head = bytes.TrimPrefix(head, []byte("\xef\xbb\xbf"))
	return bytes.HasPrefix(bytes.TrimLeft(head, " \t\r\n"), []byte("<"))
}

// readXML reads a Gramps XML document, whose root is <database>.
func readXML(r io.Reader) (*Element, error) {
	root, err := parse(r)
	if err != nil {
		return nil, fmt.Errorf("this is not Gramps XML: %w", err)
	}
	if root.Name != "database" {
		return nil, fmt.Errorf("this is not Gramps XML: its root is <%s>, where Gramps writes <database>", root.Name)
	}
	return root, nil
}

// versioned checks the version a document's namespace states, which Gramps writes as
// http://gramps-project.org/xml/1.7.2/, and makes the Export.
func versioned(root *Element, media []string, extracted string) (*Export, error) {
	namespace := root.Attr("xmlns")
	parts := strings.Split(namespace, "/")
	if len(parts) < 5 || parts[2] != "gramps-project.org" {
		return nil, fmt.Errorf("this Gramps XML states no version, as Gramps wrote it before 1.0; "+
			"origins reads %s and later, so open it in a current Gramps and export it again", oldest)
	}
	stated := parts[4]
	v, ok := parseVersion(stated)
	if !ok {
		return nil, fmt.Errorf("this Gramps XML states a version origins cannot make out, %q", stated)
	}
	if v.less(oldest) {
		return nil, fmt.Errorf("this is Gramps XML %s, and origins reads %s and later: "+
			"open it in a current Gramps and export it again", stated, oldest)
	}
	return &Export{
		Root:      root,
		Version:   stated,
		Newer:     newest.less(v),
		Media:     media,
		extracted: extracted,
	}, nil
}

// version is a Gramps XML version, major, minor and patch.
type version [3]int

// parseVersion reads a version of up to three numbers, missing ones being zero, as
// Gramps does.
func parseVersion(s string) (version, bool) {
	var v version
	numbers := strings.Split(s, ".")
	if len(numbers) > len(v) {
		return v, false
	}
	for i, number := range numbers {
		n, err := strconv.Atoi(number)
		if err != nil || n < 0 {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

func (v version) less(than version) bool { return slices.Compare(v[:], than[:]) < 0 }

func (v version) String() string { return fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2]) }

// Describe says where an export came from, as its header gives it, for the commit that
// records the import: the version of Gramps that wrote it, when, and the version of
// Gramps XML it is in. It marks what it reads of the header, and file is the name the
// export was read under.
func Describe(export *Export, file string) string {
	read := "Read from an export"
	if file != "" {
		read = "Read from " + file
	}
	header := export.Root.First("header").Mark()
	header.Take("mediapath")
	created := header.Take("created")
	var b strings.Builder
	b.WriteString(read + ", which Gramps")
	if version := strings.TrimSpace(created.Use("version")); version != "" {
		b.WriteString(" " + version)
	}
	b.WriteString(" exported")
	if date := strings.TrimSpace(created.Use("date")); date != "" {
		b.WriteString(" on " + date)
	}
	if export.Version != "" {
		b.WriteString(" as Gramps XML " + export.Version)
	}
	b.WriteString(".")
	return b.String()
}
