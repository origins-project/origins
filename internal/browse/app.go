package browse

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// app is the application the pages are drawn by: the built one where
// app/dist holds it, and otherwise the placeholder saying how to build it, so that
// origins builds and its tests run without Node.
//
//go:embed app
var app embed.FS

// built is the application as Vite wrote it, or nil where it was not built.
var built = func() fs.FS {
	dist, err := fs.Sub(app, "app/dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(dist, "index.html"); err != nil {
		return nil
	}
	return dist
}()

// app serves a file of the application where one is asked for, and its page
// everywhere else, the application's router deciding what a route shows.
func (s *Server) app(w http.ResponseWriter, r *http.Request) {
	s.serve(w, r, s.page)
}

// serve serves a file of the application, or the page drawn where none is asked for.
func (s *Server) serve(w http.ResponseWriter, r *http.Request, drawn func() ([]byte, error)) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		s.unknown(w, r)
		return
	}
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if built != nil && name != "" && name != "index.html" {
		if info, err := fs.Stat(built, name); err == nil && !info.IsDir() {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			if !servePacked(w, r, built, name) {
				http.ServeFileFS(w, r, built, name)
			}
			return
		}
	}
	page, err := drawn()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(page)
}

// page is the application's page with this server's token in it, which is how the
// application comes to hold what a write has to carry.
func (s *Server) page() ([]byte, error) {
	return s.drawn(built)
}

// unsigned is the application's page without the token, for a browser that has not
// given the passphrase.
func (s *Server) unsigned() ([]byte, error) {
	return written(built)
}

// written is the page of the application dist holds, or the placeholder's where it is
// nil. The application names its files relative to its page, as it must opened from a
// folder, and the page is served at every route, so it names them from the root here.
func written(dist fs.FS) ([]byte, error) {
	if dist == nil {
		return app.ReadFile("app/placeholder.html")
	}
	page, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		return nil, err
	}
	return bytes.ReplaceAll(page, []byte(`="./`), []byte(`="/`)), nil
}

// drawn is the page of the application dist holds, or the placeholder's where it is
// nil, with this server's token in it.
func (s *Server) drawn(dist fs.FS) ([]byte, error) {
	written, err := written(dist)
	if err != nil {
		return nil, err
	}
	meta := []byte(`<meta name="` + TokenMeta + `" content="` + s.token + `">`)
	return bytes.Replace(written, []byte("</head>"), append(meta, "</head>"...), 1), nil
}
