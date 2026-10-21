package browse

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
)

// Exported is what an HTML export wrote: where, as what view, the records each page of
// which it wrote, the files beside them, and every answer it wrote as data.
type Exported struct {
	Out      string            `json:"out"`
	Hiding   model.Hiding      `json:"hiding"`
	Records  map[string]int    `json:"records"`
	Files    int               `json:"files"`
	Answers  int               `json:"answers"`
	Read     []api.Supposition `json:"read,omitempty"`
	Problems []string          `json:"problems"`
}

// AnswerFunction is what each data script of an export calls with the path it answers
// and the answer, the client keeping it for the read that asked.
const AnswerFunction = "originsAnswer"

// ExportMeta is the meta an export's page holds, saying that its answers are scripts
// beside it.
const ExportMeta = "origins-export"

// ChartPath is the path the chart's WebAssembly is answered under in an export, as a
// data script holding it in base64.
const ChartPath = "/chart.wasm"

// Export writes the corpus at root as options say into out, a directory that is not
// there or is empty: the application as built, with every answer its pages ask for
// written beside it as a script, the chart's WebAssembly as one more and every file a
// record keeps, so that the pages opened from the folder read what a server would
// have answered them. Each answer is the one the server's own handler gives, asked of
// a server that takes no change and keeps no history, since the folder can do neither.
func Export(root, out string, options Options) (*Exported, error) {
	return export(root, out, options, built)
}

func export(root, out string, options Options, dist fs.FS) (*Exported, error) {
	if dist == nil {
		return nil, errors.New("this origins was built without the application, which an HTML export is: " +
			"npm --prefix web ci && npm --prefix web run build, then go build again")
	}
	if entries, err := os.ReadDir(out); err == nil && len(entries) > 0 {
		return nil, fmt.Errorf("%s is not empty, and an export is written into a directory of its own", out)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	s := New(root, options)
	s.exported = true
	s.made = made(root, options.Notice)
	defer s.Close()
	if s.problems != nil && len(options.With) > 0 {
		return nil, s.problems
	}

	parent := filepath.Dir(filepath.Clean(out))
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return nil, err
	}
	written, err := os.MkdirTemp(parent, "."+filepath.Base(out)+".*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(written)

	c, _, err := s.served(httptest.NewRequest(http.MethodGet, "/api/index", nil))
	if err != nil {
		return nil, err
	}
	report := &Exported{
		Out:      out,
		Hiding:   options.Hiding,
		Records:  map[string]int{},
		Read:     s.supposed.read,
		Problems: api.Flatten(s.problems),
	}
	asked := []string{"/api/session", "/api/index", "/api/findings", "/api/records/project", "/api/schema"}
	for _, kind := range model.Kinds {
		for _, slug := range c.SlugsOf(kind) {
			asked = append(asked, "/api/records/"+string(kind)+"/"+string(slug))
			report.Records[kind.Dir()]++
			for _, name := range c.Beside(slug) {
				route := "/files/" + string(kind) + "/" + string(slug) + "/" + name
				if err := s.exportFile(written, route); err != nil {
					return nil, err
				}
				report.Files++
			}
		}
	}
	for _, route := range asked {
		answer, err := s.asked(route)
		if err != nil {
			return nil, err
		}
		if err := writeAnswer(written, route, answer); err != nil {
			return nil, err
		}
		report.Answers++
	}
	if err := exportApplication(written, dist); err != nil {
		return nil, err
	}
	if err := os.Remove(out); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err := os.Rename(written, out); err != nil {
		return nil, err
	}
	return report, nil
}

// made is what an export of the corpus at root is made from now. A corpus whose
// history cannot be read is made from no commit, as one with nothing committed is.
func made(root, notice string) ExportedFrom {
	m := ExportedFrom{At: time.Now().Truncate(time.Second), Version: api.Version, Notice: notice}
	repo, err := history.Open(root)
	if err != nil {
		return m
	}
	if m.Commit, _ = repo.Head(); m.Commit != "" {
		dirty, err := repo.Dirty()
		m.Changed = err != nil || len(dirty) > 0
	}
	return m
}

// asked is the server's answer to a read, as its own handler gives it.
func (s *Server) asked(route string) (json.RawMessage, error) {
	recorder := httptest.NewRecorder()
	s.mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, route, nil))
	if recorder.Code != http.StatusOK {
		return nil, fmt.Errorf("%s was answered %d: %s", route, recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
	return json.RawMessage(recorder.Body.Bytes()), nil
}

// exportFile writes a file a record keeps where the pages ask for it, under files/.
func (s *Server) exportFile(into, route string) error {
	recorder := httptest.NewRecorder()
	steps := strings.Split(route, "/")
	for i, step := range steps {
		steps[i] = url.PathEscape(step)
	}
	s.mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, strings.Join(steps, "/"), nil))
	if recorder.Code != http.StatusOK {
		return fmt.Errorf("%s was answered %d: %s", route, recorder.Code, strings.TrimSpace(recorder.Body.String()))
	}
	return writeFile(into, strings.TrimPrefix(route, "/"), recorder.Body.Bytes())
}

// writeAnswer writes an answer as the script handing it to the client, at data/ and the
// route the client asks it under.
func writeAnswer(into, route string, answer json.RawMessage) error {
	quoted, err := json.Marshal(route)
	if err != nil {
		return err
	}
	script := AnswerFunction + "(" + string(quoted) + "," + strings.TrimSpace(string(answer)) + ");\n"
	return writeFile(into, "data"+route+".js", []byte(script))
}

// exportApplication copies the application as Vite wrote it, its page marked as an
// export's for the client to read the data scripts rather than ask a server, and its
// WebAssembly written as a data script instead, since a page opened from a folder cannot
// fetch the file.
func exportApplication(into string, dist fs.FS) error {
	return fs.WalkDir(dist, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		read, err := fs.ReadFile(dist, name)
		switch {
		case err != nil:
			return err
		case name == "index.html":
			meta := `<meta name="` + ExportMeta + `" content="">`
			read = bytes.Replace(read, []byte("</head>"), []byte(meta+"</head>"), 1)
		case path.Ext(name) == ".wasm":
			script := AnswerFunction + `("` + ChartPath + `","` + base64.StdEncoding.EncodeToString(read) + "\");\n"
			return writeFile(into, "data"+ChartPath+".js", []byte(script))
		}
		return writeFile(into, name, read)
	})
}

func writeFile(into, name string, content []byte) error {
	at := filepath.Join(into, filepath.FromSlash(name))
	if !strings.HasPrefix(at, filepath.Clean(into)+string(filepath.Separator)) {
		return fmt.Errorf("%s would be written outside the export", name)
	}
	if err := os.MkdirAll(filepath.Dir(at), 0o755); err != nil {
		return err
	}
	return os.WriteFile(at, content, 0o644)
}
