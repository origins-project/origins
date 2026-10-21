// Notices writes THIRD_PARTY_NOTICES, the licence of everything a release of origins
// carries that origins' authors did not write, from what the build is made of:
//
//	npm --prefix web run build && go run ./tools/notices
//
// That is Go and every module linked into origins on any of the platforms a release is
// built for, or into the chart's WebAssembly; every package Vite bundled into the
// application, as the build lists them in .vite/license.json, and every dependency of
// the application besides, which is how the faces it bundles through its stylesheet
// are found; and the faces the charts are drawn in. A module or package with no licence
// file beside it is refused rather than left out, unless chosen says where its licence
// is stated.
//
// With -check it writes nothing, and fails where the file differs from what it would
// write.
package main

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// file is where the notices are written, from the top of the repository.
const file = "THIRD_PARTY_NOTICES"

// platforms are the ones a release is built for, as .goreleaser.yaml names them.
var platforms = []string{
	"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64", "windows/arm64",
}

// faces are the fonts the charts are drawn in, embedded by internal/chart/draw beside
// the licences they came with.
var faces = []notice{
	{name: "Inter", licences: []string{"internal/chart/draw/fonts/LICENSE.txt"}},
	{name: "Noto Sans Arabic, Hebrew, Devanagari and CJK", licences: []string{"internal/chart/draw/fonts/NOTO-LICENSE.txt"}},
}

// licenceFile is a file at the top of a module or package that says how it may be
// used.
var licenceFile = regexp.MustCompile(`(?i)^(licen[cs]e|copying|notice|patents)([.-].*)?$`)

// freetypeCredit is the acknowledgement the FreeType License asks of whoever uses code
// under it, in the words the licence suggests.
const freetypeCredit = "Portions of this software are copyright © 2010 The FreeType Project\n" +
	"(www.freetype.org). All rights reserved."

// choice is a module whose licence is not one file at its top: the files that state
// it, from the module's directory, or the opening comment of one of its source files
// with the FreeType License from another module.
type choice struct {
	files  []string
	header string
	from   string
}

// chosen are the modules derived from Freetype-Go, which each offer the FreeType
// License or the GPL, version 2 or later. Origins takes them under the FreeType
// License. scanx states it only at the top of its source.
var chosen = map[string]choice{
	"github.com/golang/freetype":        {files: []string{"LICENSE", "licenses/ftl.txt"}},
	"github.com/BurntSushi/freetype-go": {files: []string{"LICENSE", "licenses/ftl.txt"}},
	"github.com/srwiley/scanx":          {header: "scan.go", from: "github.com/golang/freetype"},
}

// notice is one entry, whose licence is read from the files in licences, or is text.
type notice struct {
	name, version string
	credit        string
	licences      []string
	text          string
}

func main() {
	check := flag.Bool("check", false, "fail where "+file+" differs from what would be written, and write nothing")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, "notices:", err)
		os.Exit(1)
	}
}

func run(check bool) error {
	root, err := top()
	if err != nil {
		return err
	}
	written, err := notices(root)
	if err != nil {
		return err
	}
	path := filepath.Join(root, file)
	if !check {
		return os.WriteFile(path, written, 0o644)
	}
	there, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(there, written) {
		return fmt.Errorf("%s is not what the build is made of: run go run ./tools/notices", file)
	}
	return nil
}

// notices is the file, whole.
func notices(root string) ([]byte, error) {
	golang, err := goNotices(root)
	if err != nil {
		return nil, err
	}
	npm, err := npmNotices(root)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("Origins includes the software and the fonts listed below, which other people wrote.\n" +
		"Each entry gives the name and the version, and the licence it is distributed under.\n")
	for _, part := range []struct {
		heading string
		notices []notice
	}{
		{"Go, and the Go modules built into origins and its charts", golang},
		{"The packages bundled into the application", npm},
		{"The fonts the charts are drawn in", faces},
	} {
		fmt.Fprintf(&b, "\n\n%s\n%s\n", part.heading, strings.Repeat("=", len(part.heading)))
		for _, n := range part.notices {
			text, err := n.read(root)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&b, "\n\n%s\n%s\n\n", n.title(), strings.Repeat("-", len(n.title())))
			if n.credit != "" {
				fmt.Fprintf(&b, "%s\n\n", n.credit)
			}
			fmt.Fprintf(&b, "%s\n", text)
		}
	}
	return b.Bytes(), nil
}

func (n notice) title() string {
	if n.version == "" {
		return n.name
	}
	return n.name + " " + n.version
}

// read is the text of the licence, with Windows' line endings and the blank lines
// around it taken off, so that the file is the same wherever it is written.
func (n notice) read(root string) (string, error) {
	texts := []string{n.text}
	if n.text == "" {
		texts = nil
		for _, path := range n.licences {
			if !filepath.IsAbs(path) {
				path = filepath.Join(root, path)
			}
			read, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			texts = append(texts, decoded(read))
		}
	}
	for i, text := range texts {
		texts[i] = strings.Trim(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	}
	return strings.Join(texts, "\n\n"), nil
}

// goNotices is Go and the modules linked into origins or the chart on some platform.
func goNotices(root string) ([]notice, error) {
	goroot, err := output(root, nil, "go", "env", "GOROOT")
	if err != nil {
		return nil, err
	}
	goroot = strings.TrimSpace(goroot)
	version, err := os.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return nil, err
	}
	release, _, _ := strings.Cut(string(version), "\n")
	golang, err := licensed("Go", strings.TrimPrefix(release, "go"), goroot)
	if err != nil {
		return nil, err
	}

	modules := map[string]notice{}
	builds := map[string]string{"js/wasm": "./cmd/chartwasm"}
	for _, platform := range platforms {
		builds[platform] = "./cmd/origins"
	}
	for platform, command := range builds {
		goos, goarch, _ := strings.Cut(platform, "/")
		env := []string{"GOOS=" + goos, "GOARCH=" + goarch, "CGO_ENABLED=0"}
		listed, err := output(root, env, "go", "list", "-deps",
			"-f", "{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}{{end}}", command)
		if err != nil {
			return nil, err
		}
		for line := range strings.SplitSeq(strings.TrimSpace(listed), "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) != 3 || modules[fields[0]].name != "" {
				continue
			}
			module, err := moduleNotice(root, fields[0], fields[1], fields[2])
			if err != nil {
				return nil, err
			}
			modules[fields[0]] = module
		}
	}
	listed := slices.SortedFunc(func(yield func(notice) bool) {
		for _, m := range modules {
			if !yield(m) {
				return
			}
		}
	}, func(a, b notice) int { return cmp.Compare(a.name, b.name) })
	return append([]notice{golang}, listed...), nil
}

// moduleNotice is a Go module's entry, its licence where chosen says or else at its top.
func moduleNotice(root, path, version, dir string) (notice, error) {
	choice, found := chosen[path]
	if !found {
		return licensed(path, version, dir)
	}
	n := notice{name: path, version: version, credit: freetypeCredit}
	for _, file := range choice.files {
		n.licences = append(n.licences, filepath.Join(dir, filepath.FromSlash(file)))
	}
	if choice.header == "" {
		return n, nil
	}
	source, err := os.ReadFile(filepath.Join(dir, choice.header))
	if err != nil {
		return notice{}, err
	}
	var header []string
	for line := range strings.SplitSeq(strings.ReplaceAll(string(source), "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "//") || line == "//_" {
			break
		}
		header = append(header, line)
	}
	other, err := output(root, nil, "go", "list", "-m", "-f", "{{.Dir}}", choice.from)
	if err != nil {
		return notice{}, err
	}
	ftl, err := os.ReadFile(filepath.Join(strings.TrimSpace(other), "licenses", "ftl.txt"))
	if err != nil {
		return notice{}, err
	}
	n.text = strings.Join(header, "\n") + "\n\n" + decoded(ftl)
	return n, nil
}

// decoded is a licence's text, which the FreeType License's file writes in Latin-1.
func decoded(read []byte) string {
	if utf8.Valid(read) {
		return string(read)
	}
	runes := make([]rune, len(read))
	for i, b := range read {
		runes[i] = rune(b)
	}
	return string(runes)
}

// licensed is what is named, with every licence file at the top of dir.
func licensed(name, version, dir string) (notice, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return notice{}, err
	}
	n := notice{name: name, version: version}
	for _, entry := range entries {
		if !entry.IsDir() && licenceFile.MatchString(entry.Name()) {
			n.licences = append(n.licences, filepath.Join(dir, entry.Name()))
		}
	}
	if len(n.licences) == 0 {
		return notice{}, fmt.Errorf("%s %s has no licence file in %s", name, version, dir)
	}
	return n, nil
}

// npmNotices is every package Vite bundled, and every dependency of the application.
func npmNotices(root string) ([]notice, error) {
	read, err := os.ReadFile(filepath.Join(root, "internal/browse/app/dist/.vite/license.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("the application is not built: run npm --prefix web run build")
	} else if err != nil {
		return nil, err
	}
	var bundled []struct {
		Name, Version, Text string
	}
	if err := json.Unmarshal(read, &bundled); err != nil {
		return nil, err
	}
	packages := map[string]notice{}
	for _, p := range bundled {
		if strings.TrimSpace(p.Text) == "" {
			return nil, fmt.Errorf("%s %s has no licence file", p.Name, p.Version)
		}
		packages[p.Name] = notice{name: p.Name, version: p.Version, text: p.Text}
	}

	var application struct{ Dependencies map[string]string }
	if err := readJSON(filepath.Join(root, "web/package.json"), &application); err != nil {
		return nil, err
	}
	for name := range application.Dependencies {
		if _, found := packages[name]; found {
			continue
		}
		dir := filepath.Join(root, "web/node_modules", filepath.FromSlash(name))
		var installed struct{ Version string }
		if err := readJSON(filepath.Join(dir, "package.json"), &installed); err != nil {
			return nil, fmt.Errorf("%w: run npm --prefix web ci", err)
		}
		n, err := licensed(name, installed.Version, dir)
		if err != nil {
			return nil, err
		}
		packages[name] = n
	}
	return slices.SortedFunc(func(yield func(notice) bool) {
		for _, p := range packages {
			if !yield(p) {
				return
			}
		}
	}, func(a, b notice) int { return cmp.Compare(a.name, b.name) }), nil
}

func readJSON(path string, into any) error {
	read, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(read, into)
}

// top is the top of the repository, found from where the program was started.
func top() (string, error) {
	dir, err := output("", nil, "go", "env", "GOMOD")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dir) == "" || strings.TrimSpace(dir) == os.DevNull {
		return "", errors.New("run it inside the repository")
	}
	return filepath.Dir(strings.TrimSpace(dir)), nil
}

func output(dir string, env []string, name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	command.Dir = dir
	command.Env = append(os.Environ(), env...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %v\n%s", name, strings.Join(args, " "), err, stderr.String())
	}
	return string(out), nil
}
