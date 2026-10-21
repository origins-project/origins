package browse

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// compressed gzips every JSON answer and the application's files to a browser that
// takes it. It is put in front of a server off the loopback only: there the answer
// crosses a network and is worth the time compressing it takes, and on the loopback it
// is not.
func compressed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" || !gzipped(r) {
			next.ServeHTTP(w, r)
			return
		}
		g := &gzipWriter{ResponseWriter: w}
		defer g.close()
		next.ServeHTTP(g, r)
	})
}

// gzipWriter compresses what is written through it where the answer is JSON, decided
// when the header is written.
type gzipWriter struct {
	http.ResponseWriter
	decided bool
	zip     *gzip.Writer
}

func (g *gzipWriter) WriteHeader(status int) {
	if !g.decided {
		g.decided = true
		if h := g.Header(); strings.HasPrefix(h.Get("Content-Type"), "application/json") && h.Get("Content-Encoding") == "" &&
			status != http.StatusNoContent && status != http.StatusNotModified {
			h.Add("Vary", "Accept-Encoding")
			h.Set("Content-Encoding", "gzip")
			h.Del("Content-Length")
			g.zip, _ = gzip.NewWriterLevel(g.ResponseWriter, gzip.BestSpeed)
		}
	}
	g.ResponseWriter.WriteHeader(status)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.decided {
		g.WriteHeader(http.StatusOK)
	}
	if g.zip != nil {
		return g.zip.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipWriter) close() {
	if g.zip != nil {
		_ = g.zip.Close()
	}
}

func (g *gzipWriter) Unwrap() http.ResponseWriter { return g.ResponseWriter }

// gzipped says whether the browser takes an answer in gzip.
func gzipped(r *http.Request) bool {
	for _, coding := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		name, params, _ := strings.Cut(coding, ";")
		if name = strings.TrimSpace(name); name != "gzip" && name != "*" {
			continue
		}
		if q, ok := strings.CutPrefix(strings.ReplaceAll(params, " ", ""), "q="); ok {
			if weight, err := strconv.ParseFloat(q, 64); err == nil && weight == 0 {
				return false
			}
		}
		return true
	}
	return false
}

// packable are the kinds of the application's files worth compressing; a font is
// compressed already.
var packable = map[string]bool{".js": true, ".css": true, ".wasm": true, ".svg": true, ".json": true}

// packs holds each of the application's files compressed, made the first time one is
// asked for: the files are embedded and never change, so each is compressed once.
var packs sync.Map

type pack struct {
	once sync.Once
	body []byte
	err  error
}

// servePacked serves a file of the application compressed where the answer is being
// compressed, as it is only off the loopback, and the file's kind is worth it, saying
// whether it did.
func servePacked(w http.ResponseWriter, r *http.Request, dist fs.FS, name string) bool {
	g, compressing := w.(*gzipWriter)
	extension := path.Ext(name)
	if !compressing || !packable[extension] {
		return false
	}
	held, _ := packs.LoadOrStore(name, &pack{})
	p := held.(*pack)
	p.once.Do(func() { p.body, p.err = packed(dist, name) })
	if p.err != nil {
		return false
	}
	h := g.Header()
	h.Set("Content-Type", mime.TypeByExtension(extension))
	h.Set("Content-Encoding", "gzip")
	h.Add("Vary", "Accept-Encoding")
	http.ServeContent(g.ResponseWriter, r, name, time.Time{}, bytes.NewReader(p.body))
	return true
}

func packed(dist fs.FS, name string) ([]byte, error) {
	file, err := dist.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var body bytes.Buffer
	zip := gzip.NewWriter(&body)
	if _, err := io.Copy(zip, file); err != nil {
		return nil, err
	}
	if err := zip.Close(); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}
