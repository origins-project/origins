package browse

import (
	"crypto/subtle"
	"net"
	"net/http"
	"slices"
	"strings"
)

// TokenHeader is the header a write carries the server's token in. The application
// reads the token from the meta named TokenMeta in the page it was served in.
const (
	TokenHeader = "X-Origins-Token"
	TokenMeta   = "origins-token"
)

// guard is what a request passes before any handler sees it.
//
// Listening on the loopback keeps other machines out, but not other pages: any site
// open in the researcher's browser can have it send a request here. Three checks turn
// those away. The Host has to be this server's, on every request, or a page that
// points its own name at 127.0.0.1 could read the whole tree as though it were its
// own. And anything but a read has to come from one of this server's pages, which the
// browser says in Origin, and carry the token that only those pages hold.
//
// Off the loopback the Host is whatever name a proxy is reached by, and is not held to
// anything; the session cookie, which SameSite keeps every other site's page from
// sending, is what is asked instead, of every request but the login page's.
func (s *Server) guard(address string, next http.Handler) http.Handler {
	if !Loopback(address) {
		return s.signedIn(next)
	}
	hosts := []string{address}
	if _, port, err := net.SplitHostPort(address); err == nil {
		hosts = append(hosts, net.JoinHostPort("localhost", port))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !slices.ContainsFunc(hosts, func(h string) bool { return strings.EqualFold(h, r.Host) }) {
			refuse(w, "This server answers to "+strings.Join(hosts, " and ")+
				", and the request was addressed to "+addressed(r.Host)+".")
			return
		}
		s.pass(w, r, "http://"+r.Host, next)
	})
}

// signedIn is the guard off the loopback.
func (s *Server) signedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		own := scheme(r) + "://" + r.Host
		if r.URL.Path == "/api/login" && r.Method == http.MethodPost {
			if from(w, r, own) {
				s.login(w, r)
			}
			return
		}
		if !s.sessions.holds(r) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/files/"):
				answer(w, http.StatusUnauthorized, Refusal{Kind: "login",
					Error: "This server asks for its passphrase first."})
			case r.Method == http.MethodGet || r.Method == http.MethodHead:
				s.loginPage(w, r)
			default:
				s.unknown(w, r)
			}
			return
		}
		s.pass(w, r, own, next)
	})
}

// pass is what both guards ask once they know the request is for this server: a read
// holds the corpus for its length, and anything else comes from one of this server's
// pages, at own, with its token.
func (s *Server) pass(w http.ResponseWriter, r *http.Request, own string, next http.Handler) {
	if r.Method == http.MethodGet && r.URL.Path == "/api/events" {
		// a socket is open for as long as the page is, and reads nothing of the corpus
		next.ServeHTTP(w, r)
		return
	}
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		s.mu.RLock()
		defer s.mu.RUnlock()
		next.ServeHTTP(w, r)
		return
	}
	if !from(w, r, own) {
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get(TokenHeader)), []byte(s.token)) != 1 {
		refuse(w, "The request did not carry this server's token. A page drawn before the "+
			"server was last started carries an old one: load the page again.")
		return
	}
	next.ServeHTTP(w, r)
}

// from says whether a request came from a page at own, refusing it where not.
func from(w http.ResponseWriter, r *http.Request, own string) bool {
	switch origin := r.Header.Get("Origin"); {
	case origin == "":
		refuse(w, "The request did not say which page it came from, and a change is "+
			"taken only from a page this server drew.")
		return false
	case !strings.EqualFold(origin, own):
		refuse(w, "The request came from "+origin+", and a change is taken only from "+
			"a page this server drew, at "+own+".")
		return false
	}
	return true
}

// refuse answers a request the guard turned away, naming nothing in the corpus, since
// whoever asked is not somebody it answers.
func refuse(w http.ResponseWriter, why string) {
	answer(w, http.StatusForbidden, Refusal{Kind: "forbidden", Error: why})
}

func addressed(host string) string {
	if host == "" {
		return "no host at all"
	}
	return host
}
