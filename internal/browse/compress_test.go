package browse

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func gunzipped(t *testing.T, body []byte) []byte {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAnAnswerAbroadIsCompressed(t *testing.T) {
	t.Parallel()
	a := servingAbroad(t)
	given := a.ask(http.MethodPost, "/api/login", Login{Passphrase: passphrase}, nil)
	if given.Code != http.StatusOK {
		t.Fatalf("the passphrase was answered %d: %s", given.Code, given.Body)
	}
	a.cookie = given.Result().Cookies()[0]

	plain := a.ask(http.MethodGet, "/api/index", nil, nil)
	if plain.Header().Get("Content-Encoding") != "" {
		t.Fatal("an answer to a browser not taking gzip was compressed")
	}
	zipped := a.ask(http.MethodGet, "/api/index", nil, map[string]string{"Accept-Encoding": "gzip, br"})
	if zipped.Code != http.StatusOK || zipped.Header().Get("Content-Encoding") != "gzip" ||
		zipped.Header().Get("Vary") != "Accept-Encoding" {
		t.Fatalf("the index was answered %d with %v", zipped.Code, zipped.Header())
	}
	if !bytes.Equal(gunzipped(t, zipped.Body.Bytes()), plain.Body.Bytes()) {
		t.Error("the compressed index is not the index")
	}
	if zipped.Body.Len()*2 > plain.Body.Len() {
		t.Errorf("the index of %d bytes was compressed to %d", plain.Body.Len(), zipped.Body.Len())
	}
	if file := a.ask(http.MethodGet, "/files/people/verdier-jean-baptiste/portrait.jpg", nil,
		map[string]string{"Accept-Encoding": "gzip"}); file.Header().Get("Content-Encoding") != "" {
		t.Error("a picture was compressed")
	}
	if built == nil {
		return
	}
	scripts, _ := fs.Glob(built, "assets/*.js")
	for _, name := range scripts {
		script, _ := fs.ReadFile(built, name)
		zipped := a.ask(http.MethodGet, "/"+name, nil, map[string]string{"Accept-Encoding": "gzip"})
		if zipped.Header().Get("Content-Encoding") != "gzip" || !bytes.Equal(gunzipped(t, zipped.Body.Bytes()), script) {
			t.Errorf("%s was answered %d with %v", name, zipped.Code, zipped.Header())
		}
	}
}

func TestAnAnswerOnTheLoopbackIsNotCompressed(t *testing.T) {
	t.Parallel()
	c := serving(t, fixture(t))
	if index := c.do(http.MethodGet, "/api/index", nil, map[string]string{"Accept-Encoding": "gzip"}); index.Code != http.StatusOK ||
		index.Header().Get("Content-Encoding") != "" {
		t.Errorf("the index on the loopback was answered %d with %v", index.Code, index.Header())
	}
}

func TestAFileOfTheApplicationIsCompressedAbroad(t *testing.T) {
	t.Parallel()
	script := bytes.Repeat([]byte("console.log('a page');\n"), 1000)
	dist := fstest.MapFS{
		"assets/packed-test.js":    {Data: script},
		"assets/packed-test.woff2": {Data: []byte("a font")},
	}
	ask := func(name string, abroad bool) (*httptest.ResponseRecorder, bool) {
		request := httptest.NewRequest(http.MethodGet, "/"+name, nil)
		recorder := httptest.NewRecorder()
		var w http.ResponseWriter = recorder
		if abroad {
			w = &gzipWriter{ResponseWriter: recorder}
		}
		return recorder, servePacked(w, request, dist, name)
	}

	for range 2 {
		zipped, packed := ask("assets/packed-test.js", true)
		if !packed || zipped.Code != http.StatusOK || zipped.Header().Get("Content-Encoding") != "gzip" ||
			zipped.Header().Get("Content-Type") != mime.TypeByExtension(".js") ||
			zipped.Header().Get("Vary") != "Accept-Encoding" {
			t.Fatalf("the script was answered %d with %v", zipped.Code, zipped.Header())
		}
		if !bytes.Equal(gunzipped(t, zipped.Body.Bytes()), script) {
			t.Error("the compressed script is not the script")
		}
	}
	if _, packed := ask("assets/packed-test.js", false); packed {
		t.Error("the script was compressed on the loopback")
	}
	if _, packed := ask("assets/packed-test.woff2", true); packed {
		t.Error("a font was compressed")
	}
}

func TestWhatABrowserTakesIsRead(t *testing.T) {
	t.Parallel()
	for accepts, want := range map[string]bool{
		"":                     false,
		"br":                   false,
		"gzip":                 true,
		"br, gzip, deflate":    true,
		"gzip;q=0.5":           true,
		"gzip; q=0":            false,
		"*":                    true,
		"identity, *;q=0":      false,
		"deflate, gzip ;q=1.0": true,
	} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept-Encoding", accepts)
		if got := gzipped(request); got != want {
			t.Errorf("%q takes gzip %v, want %v", accepts, got, want)
		}
	}
}
