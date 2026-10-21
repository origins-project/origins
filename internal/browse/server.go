// Package browse serves a corpus to a browser: a JSON API over internal/api, the files
// attached to records, and the application that draws the pages from both.
//
// The corpus is read when the server starts and again after every write, under a lock
// every request holds to read and every write holds to write, so that no answer is
// given from a corpus half read again. A server that may write turns away every
// request that is not from a page of its own, which is what guard is for.
package browse

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Server serves one corpus.
type Server struct {
	root    string
	options Options
	token   string // asked of every write, so that a page this server did not draw cannot make one
	today   func() model.Date

	// exported is a server answering an HTML export, whose pages can neither change the
	// corpus nor ask anything of its history, there being no server behind them, and made
	// is what it was made from
	exported bool
	made     ExportedFrom

	// mu is held to read for the whole of a request and to write for the whole of a
	// write, so that nothing is answered from a corpus half read again.
	mu       sync.RWMutex
	corpus   *store.Corpus
	problems error // why the corpus did not read whole, and so why nothing may be written
	supposed supposed
	drafted  drafted
	views    views
	looked   look          // where the corpus on disk stood when it was last read
	found    string        // a digest of the findings the pages were last told of
	settled  atomic.Uint64 // how many writes of the server's own have been settled

	// looking is held for a look at the disk, which the watcher takes holding nothing else
	looking sync.Mutex
	repo    *history.Repo // the repository the corpus is in, kept once found for the look

	// patience is how long a request may take to say what it is, so that nobody can
	// hold the server's connections open by never finishing one
	patience time.Duration

	pages    pages
	sessions sessions
	markdown *markdown
	mux      *http.ServeMux
}

// New serves the corpus at root, as options say.
func New(root string, options Options) *Server {
	token := make([]byte, 32)
	_, _ = rand.Read(token)
	s := &Server{
		root:     root,
		options:  options,
		token:    hex.EncodeToString(token),
		today:    func() model.Date { return model.DateOf(time.Now()) },
		patience: 10 * time.Second,
		markdown: newMarkdown(),
	}
	s.load()
	s.looked, s.found = s.state(), s.findingsDigest()
	s.mux = s.routes()
	return s
}

func (s *Server) load() {
	s.corpus, s.problems = store.Load(s.root)
	s.suppose()
	s.redraft()
	_ = s.views.forget()
}

// Problems is why the corpus did not read whole, or nil where it did.
func (s *Server) Problems() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.problems
}

// writes says whether the server takes a change. A corpus that did not read whole is
// one every change would have to refuse, a copy is not the corpus, and an export has
// nothing to write to.
func (s *Server) writes() bool {
	return s.problems == nil && !s.options.Hiding.Any() && len(s.options.With) == 0 && !s.exported
}

// asks says whether a server is behind the pages, to tell them that the corpus moved
// and answer what no answer written down beforehand holds, a hypothesis supposed among
// it, and whether they may ask of the history, which a hidden view is refused.
func (s *Server) asks() (live, history bool) {
	return !s.exported, !s.options.Hiding.Any() && !s.exported
}

func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/session", s.session)
	mux.HandleFunc("GET /api/index", s.index)
	mux.HandleFunc("GET /api/records/project", s.project)
	mux.HandleFunc("GET /api/records/{kind}/{slug}", s.record)
	mux.HandleFunc("GET /api/findings", s.findings)
	mux.HandleFunc("GET /api/log", s.log)
	mux.HandleFunc("GET /api/whence", s.whence)
	mux.HandleFunc("GET /api/blame", s.blame)
	mux.HandleFunc("GET /api/why", s.why)
	mux.HandleFunc("GET /api/kinship", s.kinship)
	mux.HandleFunc("GET /api/schema", s.schema)
	mux.HandleFunc("GET /api/schema/{name}", s.schema)
	mux.HandleFunc("GET /api/date", s.date)
	mux.HandleFunc("GET /api/events", s.events)
	mux.HandleFunc("POST /api/edit", s.edit)
	mux.HandleFunc("POST /api/undo", s.undo)
	mux.HandleFunc("POST /api/hypotheses/{slug}/accept", s.decide(model.Accepted))
	mux.HandleFunc("POST /api/hypotheses/{slug}/reject", s.decide(model.Rejected))
	mux.HandleFunc("POST /api/render", s.render)
	mux.HandleFunc("GET /api/draft", s.draft)
	mux.HandleFunc("DELETE /api/draft", s.draftDiscard)
	mux.HandleFunc("POST /api/draft/edit", s.draftEdit)
	mux.HandleFunc("POST /api/draft/pop", s.draftPop)
	mux.HandleFunc("POST /api/draft/keep", s.draftKeep)
	mux.HandleFunc("GET /api/remotes", s.remotes)
	mux.HandleFunc("POST /api/remotes", s.addRemote)
	mux.HandleFunc("DELETE /api/remotes/{name}", s.removeRemote)
	mux.HandleFunc("POST /api/fetch", s.fetch)
	mux.HandleFunc("POST /api/pull", s.pull)
	mux.HandleFunc("POST /api/push", s.push)
	mux.HandleFunc("POST /api/recover", s.recover)
	mux.HandleFunc("/api/", s.unknown)
	mux.HandleFunc("GET /files/{kind}/{slug}/{path...}", s.file)
	mux.HandleFunc("/", s.app)
	return mux
}

// Handler answers requests addressed to address, the host and port the server is bound
// to, and to localhost at that port, compressing what it answers where that is off the
// loopback.
func (s *Server) Handler(address string) http.Handler {
	handler := s.guard(address, s.mux)
	if !Loopback(address) {
		return compressed(handler)
	}
	return handler
}

// Listen binds addr, so that a port of 0 resolves to a real one and a browser can be
// opened at an address already answering before Serve is called.
func Listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

// Serve answers on listener until it is closed, watching the corpus for a change made
// from outside all the while.
func (s *Server) Serve(listener net.Listener) error {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go s.Watch(ctx, Every)
	server := &http.Server{Handler: s.Handler(listener.Addr().String()), ReadHeaderTimeout: s.patience}
	return server.Serve(listener)
}

// Loopback says whether addr is an address on this machine alone.
func Loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Open asks the desktop to show a URL. A failure is reported rather than fatal: the
// server is running either way, and the address is on screen.
func Open(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	return command.Start()
}
