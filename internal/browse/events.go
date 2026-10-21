package browse

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// Every changes the corpus, whoever makes them: a page of this server, a command typed
// in a terminal, an editor saving a file. A page is told so over a WebSocket and asks
// again for what it shows; the event carries nothing a page could show, so there is one
// way for anything to reach it.

// Event is what a page is told.
type Event struct {
	Type string `json:"type"`           // moved, findings or draft
	Head string `json:"head,omitempty"` // the commit the corpus stands on, where it moved
}

// The events a page is sent.
const (
	Moved    = "moved"
	Findings = "findings"
	Draft    = "draft"
)

// Every is how often the corpus is looked at for a change made from outside.
const Every = 2 * time.Second

// pages is every page listening, each sent what happens on a channel of its own.
type pages struct {
	mu      sync.Mutex
	members map[chan Event]bool
}

// join adds a page, handing back what it is sent on.
func (p *pages) join() chan Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.members == nil {
		p.members = map[chan Event]bool{}
	}
	events := make(chan Event, 16)
	p.members[events] = true
	return events
}

func (p *pages) leave(events chan Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.members[events] {
		delete(p.members, events)
		close(events)
	}
}

// send tells every page. A page that has fallen so far behind that sixteen events are
// waiting for it is let go, and asks for everything again when it reconnects, which is
// what it would have done with the events anyway.
func (p *pages) send(event Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for events := range p.members {
		select {
		case events <- event:
		default:
			delete(p.members, events)
			close(events)
		}
	}
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx := conn.CloseRead(r.Context())
	events := s.pages.join()
	defer s.pages.leave(events)
	for {
		select {
		case <-ctx.Done():
			return
		case event, open := <-events:
			if !open {
				conn.Close(websocket.StatusTryAgainLater, "fell behind")
				return
			}
			sending, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := wsjson.Write(sending, conn, event)
			cancel()
			if err != nil {
				return
			}
		}
	}
}

// Watch looks at the corpus every so often until ctx is done, and reads it again and
// tells every page where it has moved.
func (s *Server) Watch(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.watched()
		}
	}
}

// watched looks at the corpus once and settles what it found. The look reads every
// file's name, size and time, which on a large corpus takes a while, and is taken
// holding nothing, so that no request waits on it; mu is only held to write once the
// look has found a move. A write of the server's own settled while the look was being
// taken leaves it out of date, and it is let go rather than told of as a move back.
func (s *Server) watched() {
	before := s.settled.Load()
	now := s.state()
	s.mu.RLock()
	same := now == s.looked
	s.mu.RUnlock()
	if same {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settled.Load() == before {
		s.settle(now)
	}
}

// look is where the corpus on disk stands: the commit, a digest of what a change to it
// would change, and the same of the draft kept beside it.
type look struct {
	head  string
	files string
	draft string
}

// state is a look at the corpus on disk, taken without reading any of it: HEAD and the
// name, size and time of every file a record is kept in or beside, and of the
// manifest. A digest of those moves whenever any of them does, on every system,
// without a dependency to be told so. Git's index is left out: git rewrites it to
// refresh what it knows of the files, on a status as much as a commit, and nothing a
// page shows is read from it.
func (s *Server) state() look {
	s.looking.Lock()
	defer s.looking.Unlock()
	var at look
	digest := sha256.New()
	if s.repo == nil {
		s.repo, _ = history.Open(s.root)
	}
	if repo := s.repo; repo != nil {
		at.head, _ = repo.Head()
		if draft, err := repo.GitPath(ops.DraftFile); err == nil {
			if info, err := os.Stat(draft); err == nil {
				at.draft = fmt.Sprintf("%d %d", info.Size(), info.ModTime().UnixNano())
			}
		}
	}
	stat := func(path string, info fs.FileInfo) {
		fmt.Fprintf(digest, "%s %d %d\n", filepath.ToSlash(path), info.Size(), info.ModTime().UnixNano())
	}
	if info, err := os.Stat(filepath.Join(s.root, store.ProjectFile)); err == nil {
		stat(store.ProjectFile, info)
	}
	for _, kind := range model.Kinds {
		_ = filepath.WalkDir(filepath.Join(s.root, kind.Dir()), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if info, err := entry.Info(); err == nil {
				stat(path, info)
			}
			return nil
		})
	}
	at.files = hex.EncodeToString(digest.Sum(nil))
	return at
}

// settle reads the corpus again where it has moved since it was last read, and tells
// every page: that it moved, that the draft changed where it did, and that the
// findings changed where they did. Where only the draft moved, only the draft is read
// again. It is called holding mu to write.
func (s *Server) settle(now look) {
	if now == s.looked {
		return
	}
	moved := now.head != s.looked.head || now.files != s.looked.files
	drafted := now.draft != s.looked.draft
	s.looked = now
	if !moved {
		s.redraft()
		s.pages.send(Event{Type: Draft})
		return
	}
	s.load()
	s.pages.send(Event{Type: Moved, Head: now.head})
	if drafted {
		s.pages.send(Event{Type: Draft})
	}
	if findings := s.findingsDigest(); findings != s.found {
		s.found = findings
		s.pages.send(Event{Type: Findings})
	}
}

// wrote is settle after a write of the server's own, to the corpus or to the draft,
// so that the pages are told of it once, as of a change made anywhere else, and the
// watcher finds nothing left to tell. It is called holding mu to write.
func (s *Server) wrote() {
	s.settled.Add(1)
	s.settle(s.state())
}

func (s *Server) findingsDigest() string {
	written, _ := json.Marshal(api.FindingsOf(s.corpus.Corpus, s.problems, s.today()))
	sum := sha256.Sum256(written)
	return hex.EncodeToString(sum[:])
}
