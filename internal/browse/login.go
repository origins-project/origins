package browse

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Off the loopback the server is reachable by whoever can reach the machine, so it
// asks first who is there. There is one researcher and one passphrase, given when the
// server is started; whoever gives it gets a session cookie, kept in memory until the
// server stops, and every request but the login page's asks for it. HTTPS is a reverse
// proxy's, which says so in X-Forwarded-Proto.

// SessionCookie is the cookie a session is carried in, named for the port the browser
// asked at where it named one: a browser keeps one set of cookies for a host whatever
// the port, and two servers on one machine would otherwise each take the other's
// session for a stranger's.
func SessionCookie(r *http.Request) string {
	if _, port, err := net.SplitHostPort(r.Host); err == nil && port != "" {
		return "origins-session-" + port
	}
	return "origins-session"
}

// Login is what the login page sends.
type Login struct {
	Passphrase string `json:"passphrase"`
}

// sessions is every session given since the server started, and when a passphrase was
// last tried, so that it is tried at most once a second.
type sessions struct {
	mu    sync.Mutex
	given map[string]bool
	tried time.Time
}

func (s *sessions) holds(r *http.Request) bool {
	cookie, err := r.Cookie(SessionCookie(r))
	if err != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.given[cookie.Value]
}

func (s *sessions) give() string {
	id := make([]byte, 32)
	_, _ = rand.Read(id)
	session := hex.EncodeToString(id)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.given == nil {
		s.given = map[string]bool{}
	}
	s.given[session] = true
	return session
}

// try says whether a passphrase may be tried now, and counts this as the try.
func (s *sessions) try(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Sub(s.tried) < time.Second {
		return false
	}
	s.tried = now
	return true
}

// secure says whether the request reached the researcher's browser as HTTPS, at this
// server or at a proxy in front of it.
func secure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// scheme is the scheme the researcher's browser used, which its Origin carries.
func scheme(r *http.Request) string {
	if secure(r) {
		return "https"
	}
	return "http"
}

// login gives a session to whoever gives the passphrase. Both sides are hashed before
// they are compared, so that the comparison takes as long whatever was given.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.sessions.try(time.Now()) {
		answer(w, http.StatusTooManyRequests, Refusal{Kind: "login",
			Error: "A passphrase is tried once a second: try again."})
		return
	}
	var asked Login
	if !decoded(w, r, &asked) {
		return
	}
	given, kept := sha256.Sum256([]byte(asked.Passphrase)), sha256.Sum256([]byte(s.options.Passphrase))
	if subtle.ConstantTimeCompare(given[:], kept[:]) != 1 {
		answer(w, http.StatusUnauthorized, Refusal{Kind: "login", Error: "That is not this server's passphrase."})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookie(r),
		Value:    s.sessions.give(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure(r),
	})
	answer(w, http.StatusOK, struct{}{})
}

// loginPage is what a browser without a session is shown wherever it asks for a page:
// the application, without the token, which asks the passphrase when the server
// refuses it the session, and the application's own files, which hold nothing of the
// corpus.
func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	s.serve(w, r, s.unsigned)
}
