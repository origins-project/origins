package browse

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/store"
)

// page is a page listening to a server over its socket, as the application does.
type page struct {
	t      *testing.T
	client *client
	conn   *websocket.Conn
	events chan Event
}

// listening is a server over the corpus at root on a real port, watching the corpus
// often, and a page connected to it.
func listening(t *testing.T, root string) *page {
	t.Helper()
	c := serving(t, root)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	served := httptest.NewUnstartedServer(c.server.Handler(listener.Addr().String()))
	served.Listener.Close()
	served.Listener = listener
	served.Start()
	t.Cleanup(served.Close)
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	go c.server.Watch(ctx, 50*time.Millisecond)

	conn, _, err := websocket.Dial(ctx, "ws://"+listener.Addr().String()+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	p := &page{t: t, client: c, conn: conn, events: make(chan Event, 64)}
	go func() {
		for {
			var event Event
			if err := wsjson.Read(ctx, conn, &event); err != nil {
				close(p.events)
				return
			}
			p.events <- event
		}
	}()
	return p
}

// moved waits for the page to be told the corpus moved, and hands back where to.
func (p *page) moved() Event {
	p.t.Helper()
	return p.told(Moved)
}

// told waits for the page to be told an event of the type given.
func (p *page) told(kind string) Event {
	p.t.Helper()
	deadline := time.After(10 * time.Second)
	for {
		select {
		case event, open := <-p.events:
			if !open {
				p.t.Fatal("the socket closed")
			}
			if event.Type == kind {
				return event
			}
		case <-deadline:
			p.t.Fatalf("the page was never told %s", kind)
		}
	}
}

// movedTo waits for the page to be told the corpus moved to commit. A command writes
// its files before it commits them, and a watcher looking in between tells the page of
// both, as it should.
func (p *page) movedTo(commit string) {
	p.t.Helper()
	for p.moved().Head != commit {
	}
}

// still says the page is told of no further move, nor of the draft, for a while, which
// is as long as the watcher takes to look several times.
func (p *page) still() {
	p.t.Helper()
	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case event := <-p.events:
			if event.Type == Moved || event.Type == Draft {
				p.t.Fatalf("the page was told again: %+v", event)
			}
		case <-deadline:
			return
		}
	}
}

func TestACommandsChangeReachesAPage(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	p := listening(t, root)
	edit, err := store.ParseEdit([]byte(`{"op": "add", "entity": "person/verdier-pierre", "path": "tags", "value": "checked"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := api.Carry(api.Asking{Root: root, Command: "add"}, edit); err != nil {
		t.Fatal(err)
	}
	p.movedTo(head(t, root))
	if read[Page](p.client, p.client.get("/api/records/person/verdier-pierre"), http.StatusOK).Record.Seen == "" {
		t.Error("the record has no seen")
	}
	p.still()
}

func TestAHandEditReachesAPage(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	p := listening(t, root)
	before := head(t, root)
	file := filepath.Join(root, "people", "verdier-pierre", "README.md")
	written, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, append(written, "\nWritten by hand.\n"...), 0o644); err != nil {
		t.Fatal(err)
	}
	if event := p.moved(); event.Head != before {
		t.Errorf("the page was told the corpus is at %s, and nothing was committed after %s", event.Head, before)
	}
	p.client.server.mu.RLock()
	body := p.client.server.corpus.People["verdier-pierre"].Body
	p.client.server.mu.RUnlock()
	if !strings.Contains(body, "Written by hand.") {
		t.Errorf("the corpus was not read again: the body is %q", body)
	}
}

func TestAPagesOwnChangeReachesItOnce(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	p := listening(t, root)
	made := p.client.change(map[string]any{"op": "add", "entity": "person/verdier-pierre", "path": "tags", "value": "checked"}, http.StatusOK)
	if event := p.moved(); event.Head != made.Commit {
		t.Errorf("the page was told the corpus is at %s, and the change made %s", event.Head, made.Commit)
	}
	p.still()
}

func TestGitRefreshingItsIndexMovesNothing(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	p := listening(t, root)
	index := filepath.Join(root, ".git", "index")
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(index, later, later); err != nil {
		t.Fatal(err)
	}
	p.still()
}

func TestADraftReachesAPageOnce(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	p := listening(t, root)
	p.client.drafting(threeRecords[1], http.StatusOK)
	p.told(Draft)
	p.still()
	if p.client.do(http.MethodDelete, "/api/draft", nil, nil).Code != http.StatusOK {
		t.Fatal("the draft was not thrown away")
	}
	p.told(Draft)
	p.still()
}

func TestAPageFromElsewhereIsNotListenedTo(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	request := newRequest(http.MethodGet, "/api/events", address)
	request.Header.Set("Connection", "Upgrade")
	request.Header.Set("Upgrade", "websocket")
	request.Header.Set("Sec-WebSocket-Version", "13")
	request.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	request.Header.Set("Origin", "https://elsewhere.example")
	if response := record(c, request); response.Code != http.StatusForbidden {
		t.Errorf("a socket opened from another site was answered %d", response.Code)
	}
}
