package browse

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const passphrase = "a passphrase only the researcher knows"

// abroad is a server off the loopback, behind a proxy answering to tree.example.
type abroad struct {
	t       *testing.T
	server  *Server
	handler http.Handler
	host    string
	cookie  *http.Cookie
}

func servingAbroad(t *testing.T) *abroad {
	t.Helper()
	c := servingWith(t, fixture(t), Options{Passphrase: passphrase})
	return &abroad{t: t, server: c.server, handler: c.server.Handler("0.0.0.0:8100"), host: "tree.example"}
}

// ask sends a request as the browser at the far side of an HTTPS proxy would, with the
// session where there is one.
func (a *abroad) ask(method, target string, body any, header map[string]string) *httptest.ResponseRecorder {
	a.t.Helper()
	var written []byte
	if body != nil {
		written, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(written))
	request.Host = a.host
	request.Header.Set("X-Forwarded-Proto", "https")
	if method != http.MethodGet {
		request.Header.Set("Origin", "https://"+a.host)
		request.Header.Set("Content-Type", "application/json")
	}
	if a.cookie != nil {
		request.AddCookie(a.cookie)
	}
	for name, value := range header {
		if value == "" {
			request.Header.Del(name)
		} else {
			request.Header.Set(name, value)
		}
	}
	recorder := httptest.NewRecorder()
	a.handler.ServeHTTP(recorder, request)
	return recorder
}

func refusalIn(t *testing.T, response *httptest.ResponseRecorder, status int, kind string) {
	t.Helper()
	var refusal Refusal
	if response.Code != status || json.Unmarshal(response.Body.Bytes(), &refusal) != nil || refusal.Kind != kind {
		t.Errorf("answered %d, want %d and a refusal of kind %s: %s", response.Code, status, kind, response.Body)
	}
}

func TestAServerAbroadAsksForItsPassphrase(t *testing.T) {
	t.Parallel()
	a := servingAbroad(t)

	refusalIn(t, a.ask(http.MethodGet, "/api/index", nil, nil), http.StatusUnauthorized, "login")
	refusalIn(t, a.ask(http.MethodGet, "/files/people/verdier-jean-baptiste/portrait.jpg", nil, nil), http.StatusUnauthorized, "login")
	unsigned, err := written(built)
	if err != nil {
		t.Fatal(err)
	}
	if page := a.ask(http.MethodGet, "/people/verdier-jean-baptiste", nil, nil); page.Code != http.StatusOK ||
		page.Body.String() != string(unsigned) || strings.Contains(page.Body.String(), a.server.token) {
		t.Errorf("a page asked for without a session is not the application without the token: %d\n%s", page.Code, page.Body)
	}

	refusalIn(t, a.ask(http.MethodPost, "/api/login", Login{Passphrase: passphrase}, map[string]string{"Origin": "https://elsewhere.example"}),
		http.StatusForbidden, "forbidden")
	time.Sleep(time.Second)
	refusalIn(t, a.ask(http.MethodPost, "/api/login", Login{Passphrase: "a guess"}, nil), http.StatusUnauthorized, "login")
	refusalIn(t, a.ask(http.MethodPost, "/api/login", Login{Passphrase: passphrase}, nil), http.StatusTooManyRequests, "login")
	time.Sleep(time.Second)
	given := a.ask(http.MethodPost, "/api/login", Login{Passphrase: passphrase}, nil)
	if given.Code != http.StatusOK {
		t.Fatalf("the passphrase was answered %d: %s", given.Code, given.Body)
	}
	cookies := given.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "origins-session" || !cookies[0].HttpOnly ||
		cookies[0].SameSite != http.SameSiteStrictMode || !cookies[0].Secure {
		t.Fatalf("the session is %+v", cookies)
	}
	a.cookie = cookies[0]

	if index := a.ask(http.MethodGet, "/api/index", nil, nil); index.Code != http.StatusOK {
		t.Errorf("a read with the session was answered %d", index.Code)
	}
	if page := a.ask(http.MethodGet, "/people/verdier-jean-baptiste", nil, nil); !strings.Contains(page.Body.String(), a.server.token) {
		t.Error("a page asked for with a session is not the application")
	}
	refusalIn(t, a.ask(http.MethodPost, "/api/render", Rendering{Entity: "person/verdier-jean-baptiste", Prose: "*a*"}, nil),
		http.StatusForbidden, "forbidden")
	if rendered := a.ask(http.MethodPost, "/api/render", Rendering{Entity: "person/verdier-jean-baptiste", Prose: "*a*"},
		map[string]string{TokenHeader: a.server.token}); rendered.Code != http.StatusOK {
		t.Errorf("a change with the session and the token was answered %d: %s", rendered.Code, rendered.Body)
	}
	refusalIn(t, a.ask(http.MethodPost, "/api/render", Rendering{Entity: "person/verdier-jean-baptiste", Prose: "*a*"},
		map[string]string{TokenHeader: a.server.token, "Origin": "http://tree.example"}), http.StatusForbidden, "forbidden")

	a.cookie = &http.Cookie{Name: "origins-session", Value: strings.Repeat("0", 64)}
	refusalIn(t, a.ask(http.MethodGet, "/api/index", nil, nil), http.StatusUnauthorized, "login")
}

// A browser keeps one set of cookies for a host whatever the port, so each of two
// servers on one machine keeps its session under a name of its own.
func TestTwoServersOnOneHostKeepTheirOwnSessions(t *testing.T) {
	t.Parallel()
	first, second := servingAbroad(t), servingAbroad(t)
	first.host, second.host = "tree.example:8100", "tree.example:8101"
	var jar []*http.Cookie
	for _, a := range []*abroad{first, second} {
		given := a.ask(http.MethodPost, "/api/login", Login{Passphrase: passphrase}, nil)
		cookies := given.Result().Cookies()
		if given.Code != http.StatusOK || len(cookies) != 1 {
			t.Fatalf("the passphrase was answered %d with %+v", given.Code, cookies)
		}
		jar = append(jar, cookies[0])
	}
	if jar[0].Name == jar[1].Name {
		t.Fatalf("both servers keep their session as %s", jar[0].Name)
	}
	for _, a := range []*abroad{first, second} {
		var cookie []string
		for _, c := range jar {
			cookie = append(cookie, c.Name+"="+c.Value)
		}
		if index := a.ask(http.MethodGet, "/api/index", nil, map[string]string{"Cookie": strings.Join(cookie, "; ")}); index.Code != http.StatusOK {
			t.Errorf("a read at %s with both sessions was answered %d", a.host, index.Code)
		}
	}
}

func TestASessionOverPlainHTTPIsNotSecure(t *testing.T) {
	t.Parallel()
	a := servingAbroad(t)
	given := a.ask(http.MethodPost, "/api/login", Login{Passphrase: passphrase},
		map[string]string{"X-Forwarded-Proto": "", "Origin": "http://tree.example"})
	cookies := given.Result().Cookies()
	if given.Code != http.StatusOK || len(cookies) != 1 || cookies[0].Secure {
		t.Errorf("a session given over plain HTTP is %d %+v", given.Code, cookies)
	}
}

// Whoever can reach the server can open a connection to it, and one that never finishes
// saying what it asks is closed rather than held open for good.
func TestARequestThatNeverSaysWhatItIsIsClosed(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	c.server.patience = 100 * time.Millisecond
	listener, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() { _ = c.server.Serve(listener) }()

	connection, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if _, err := io.WriteString(connection, "GET /api/index HTTP/1.1\r\nHost: "); err != nil {
		t.Fatal(err)
	}
	_ = connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err = io.ReadAll(connection)
	var timeout net.Error
	if errors.As(err, &timeout) && timeout.Timeout() {
		t.Error("a request that never finished its headers was still open after five seconds")
	}
}
