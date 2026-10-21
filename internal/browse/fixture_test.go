package browse

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/model"
)

// TestMain points git at configuration of the package's own for every test in it, so
// that nothing a test asserts depends on how the machine running it is set up.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "origins-test")
	if err != nil {
		panic(err)
	}
	config := filepath.Join(dir, "git-config")
	identity := "[user]\n\tname = Rosa Bianchi\n\temail = rosa@example.invalid\n" +
		"[core]\n\tautocrlf = false\n"
	if err := os.WriteFile(config, []byte(identity), 0o644); err != nil {
		panic(err)
	}
	os.Setenv("GIT_CONFIG_GLOBAL", config)
	os.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(dir, "no-such-system-config"))
	// git packs no test's repository after a commit, none living long enough to need
	// it, and commits as nobody it has not been told of, rather than guessing at an
	// identity from the machine, which on some machines it finds
	os.Setenv("GIT_CONFIG_COUNT", "2")
	os.Setenv("GIT_CONFIG_KEY_0", "maintenance.auto")
	os.Setenv("GIT_CONFIG_VALUE_0", "false")
	os.Setenv("GIT_CONFIG_KEY_1", "user.useConfigOnly")
	os.Setenv("GIT_CONFIG_VALUE_1", "true")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// fixture is a copy of the fixture corpus, which is generated rather than committed,
// so this skips where it is not there. Run tools/mkfixture.sh to produce it.
func fixture(t *testing.T) string {
	t.Helper()
	built := filepath.Join("..", "..", "test")
	if _, err := os.Stat(built); err != nil {
		t.Skip("no fixture corpus; run tools/mkfixture.sh")
	}
	root := filepath.Join(t.TempDir(), "corpus")
	if err := os.CopyFS(root, os.DirFS(built)); err != nil {
		t.Fatal(err)
	}
	return root
}

// gitFixture is a copy of testgit, the fixture corpus built through origins from
// nothing, with its history. It skips where it has not been built; run
// tools/mkgitfixture.sh to produce it.
func gitFixture(t *testing.T) string {
	t.Helper()
	built := filepath.Join("..", "..", "testgit")
	if _, err := os.Stat(built); err != nil {
		t.Skip("no git fixture; run tools/mkgitfixture.sh")
	}
	root := filepath.Join(t.TempDir(), "corpus")
	if err := os.CopyFS(root, os.DirFS(built)); err != nil {
		t.Fatal(err)
	}
	return root
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// address is the address a test server says it is bound to.
const address = "127.0.0.1:8100"

// testDay is the day a test server decides who is living as of.
var testDay = model.DateOf(time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC))

// serving is a server over the corpus at root, answering as though bound to address.
func serving(t *testing.T, root string) *client {
	t.Helper()
	return servingWith(t, root, Options{})
}

// servingWith is the same, serving as options say.
func servingWith(t *testing.T, root string, options Options) *client {
	t.Helper()
	s := New(root, options)
	t.Cleanup(func() { _ = s.Close() })
	s.today = func() model.Date { return testDay }
	s.found = s.findingsDigest()
	return &client{t: t, server: s, handler: s.Handler(address)}
}

// client asks a server what a page would, through its guard.
type client struct {
	t       *testing.T
	server  *Server
	handler http.Handler
}

// do sends a request as a page this server drew would, and hands back the response.
func (c *client) do(method, target string, body io.Reader, header map[string]string) *httptest.ResponseRecorder {
	c.t.Helper()
	request := httptest.NewRequest(method, target, body)
	request.Host = address
	if method != http.MethodGet && method != http.MethodHead {
		request.Header.Set("Origin", "http://"+address)
		request.Header.Set(TokenHeader, c.server.token)
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range header {
		if value == "" {
			request.Header.Del(name)
		} else {
			request.Header.Set(name, value)
		}
	}
	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, request)
	return recorder
}

// newRequest is a request addressed to host rather than to the server's own address.
func newRequest(method, target, host string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.Host = host
	return request
}

// record is how the server answers request, as it was made.
func record(c *client, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, request)
	return recorder
}

func (c *client) get(target string) *httptest.ResponseRecorder {
	c.t.Helper()
	return c.do(http.MethodGet, target, nil, nil)
}

// read is what a request asking for a T was answered with, failing the test where it
// was not answered with status.
func read[T any](c *client, response *httptest.ResponseRecorder, status int) T {
	c.t.Helper()
	var out T
	if response.Code != status {
		c.t.Fatalf("answered %d, want %d:\n%s", response.Code, status, response.Body)
	}
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		c.t.Fatalf("%v:\n%s", err, response.Body)
	}
	return out
}
