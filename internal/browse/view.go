package browse

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// Options is what a server serves besides the corpus as it stands: a view it is locked
// to, so that a copy served to somebody else cannot be switched back, and hypotheses
// supposed true. Either makes the server read-only, what it serves being a copy.
//
// Passphrase is what a server off the loopback asks for before anything else, and
// Notice what an export's pages show below the bar on every page.
type Options struct {
	Hiding     model.Hiding
	With       []model.Slug
	Passphrase string
	Notice     string
}

// ViewParam is the query parameter a read asks for a hidden view with, once for each of
// HidePrivate and HideLiving it wants, and WithParam the one it asks for a hypothesis
// supposed true with, once for each.
const (
	ViewParam   = "view"
	HidePrivate = "hide-private"
	HideLiving  = "hide-living"
	WithParam   = "with"
)

// supposed is the corpus with the hypotheses of --with in it, built again whenever the
// corpus is read again, and how each of them was read.
type supposed struct {
	trial *ops.Trial
	read  []api.Supposition
}

// views is each supposition and each hidden view of the corpus asked for since it was
// last read, made once, a supposition that could not be built kept as the error it
// gave. Reads make them holding mu only to read, so they have a lock of their own.
type views struct {
	mu       sync.Mutex
	made     map[viewed]*store.Corpus
	supposed map[string]asked
}

// viewed is a view of a supposition, the hypotheses joined by a space.
type viewed struct {
	with   string
	hiding model.Hiding
}

// asked is the corpus with the hypotheses a read asked for in it, or why it could not be.
type asked struct {
	supposed
	err error
}

// forget takes away every view made and every supposition a read asked for, the corpus
// they were made from having been read again.
func (v *views) forget() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	var err error
	for _, one := range v.supposed {
		if one.trial != nil {
			err = errors.Join(err, one.trial.Close())
		}
	}
	v.made, v.supposed = nil, nil
	return err
}

// suppose builds the supposition afresh over the corpus just read, closing the one it
// replaces. A supposition that cannot be built is a problem the corpus is served with,
// as it stands and read-only.
func (s *Server) suppose() {
	if s.supposed.trial != nil {
		_ = s.supposed.trial.Close()
		s.supposed = supposed{}
	}
	if len(s.options.With) == 0 || s.problems != nil {
		return
	}
	trial, read, err := api.Supposed(s.corpus, s.options.With)
	if err != nil {
		s.problems = err
		return
	}
	s.supposed = supposed{trial: trial, read: read}
}

// Close takes away the copies a supposition and a draft were built in.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := errors.Join(s.drafted.corpus.Close(), s.views.forget())
	s.drafted = drafted{}
	if s.supposed.trial != nil {
		err = errors.Join(err, s.supposed.trial.Close())
		s.supposed = supposed{}
	}
	return err
}

// base is the corpus every view is made from: the one supposed, where there is one.
func (s *Server) base() *store.Corpus {
	if s.supposed.trial != nil {
		return s.supposed.trial.Corpus
	}
	return s.corpus
}

// hiding is the view a read asks for, together with the one the server is locked to.
func (s *Server) hiding(r *http.Request) (model.Hiding, error) {
	hiding := s.options.Hiding
	for _, asked := range r.URL.Query()[ViewParam] {
		for _, one := range strings.Split(asked, ",") {
			switch one {
			case HidePrivate:
				hiding.Private = true
			case HideLiving:
				hiding.Living = true
			default:
				return hiding, &askedError{"a view hides " + HidePrivate + ", " + HideLiving +
					" or both, and not " + one}
			}
		}
	}
	return hiding, nil
}

// with is the hypotheses a read asks to have supposed, in the order it names them.
func with(r *http.Request) []model.Slug {
	var named []model.Slug
	for _, asked := range r.URL.Query()[WithParam] {
		for _, one := range strings.Split(asked, ",") {
			if slug := model.Slug(strings.TrimSpace(one)); slug != "" && !slices.Contains(named, slug) {
				named = append(named, slug)
			}
		}
	}
	return named
}

// supposedFor is the corpus a read is answered from before any view, and how each
// hypothesis was read into it: the base, or the base with the hypotheses the read asks
// for supposed too, built once until the corpus is read again.
func (s *Server) supposedFor(r *http.Request) (*store.Corpus, []api.Supposition, string, error) {
	named := with(r)
	if len(named) == 0 {
		return s.base(), s.supposed.read, "", nil
	}
	if s.problems != nil {
		return nil, nil, "", s.problems
	}
	all := slices.Clone(s.options.With)
	for _, slug := range named {
		if !slices.Contains(all, slug) {
			all = append(all, slug)
		}
	}
	key := strings.Join(slugStrings(all), " ")
	s.views.mu.Lock()
	defer s.views.mu.Unlock()
	one, ok := s.views.supposed[key]
	if !ok {
		trial, read, err := api.Supposed(s.corpus, all)
		one = asked{supposed{trial: trial, read: read}, err}
		if s.views.supposed == nil {
			s.views.supposed = map[string]asked{}
		}
		s.views.supposed[key] = one
	}
	if one.err != nil {
		return nil, nil, key, one.err
	}
	return one.trial.Corpus, one.read, key, nil
}

func slugStrings(slugs []model.Slug) []string {
	out := make([]string, len(slugs))
	for i, slug := range slugs {
		out[i] = string(slug)
	}
	return out
}

// served is the corpus a read is answered from: the base, with whatever the read asks
// to have supposed, as the view the read asks for shows it, or the corpus with the
// draft in it where the read asks for that.
func (s *Server) served(r *http.Request) (*store.Corpus, model.Hiding, error) {
	hiding, err := s.hiding(r)
	if err != nil {
		return nil, hiding, err
	}
	if r.URL.Query().Has(DraftParam) && r.URL.Query().Has(WithParam) {
		return nil, hiding, &askedError{"a draft is the corpus being changed, and is not shown with a hypothesis supposed"}
	}
	if drafted, asked, err := s.draftedFor(r, hiding); asked {
		return drafted, hiding, err
	}
	base, _, key, err := s.supposedFor(r)
	if err != nil || !hiding.Any() {
		return base, hiding, err
	}
	s.views.mu.Lock()
	defer s.views.mu.Unlock()
	if s.views.made == nil {
		s.views.made = map[viewed]*store.Corpus{}
	}
	view, ok := s.views.made[viewed{key, hiding}]
	if !ok {
		view = base.Viewed(model.Hide(base.Corpus, hiding, s.today()))
		s.views.made[viewed{key, hiding}] = view
	}
	return view, hiding, nil
}

// readOnly says why the server takes no change, or nil where it takes one.
func (s *Server) readOnly(r *http.Request) error {
	switch hiding, err := s.hiding(r); {
	case err != nil:
		return err
	case hiding.Any():
		return &readOnlyError{"a hidden view is read-only: what it shows is a copy, without " + hiding.String()}
	case len(s.options.With) > 0 || r.URL.Query().Has(WithParam):
		return &readOnlyError{"a corpus served with hypotheses supposed true is read-only: " +
			"what it shows is a copy, and accepting a hypothesis is what writes it into the corpus"}
	}
	return nil
}

// readOnlyError is a change asked of a server serving a copy.
type readOnlyError struct{ why string }

func (e *readOnlyError) Error() string { return e.why }

// hiddenError is a question a hidden view cannot answer without showing what it hides.
type hiddenError struct{ why string }

func (e *hiddenError) Error() string { return e.why }

// unhidden is nil where a read may be answered from the history, which is the corpus
// whole: never under a hidden view.
func (s *Server) unhidden(r *http.Request) error {
	switch hiding, err := s.hiding(r); {
	case err != nil:
		return err
	case hiding.Any():
		return &hiddenError{"the history is the corpus whole, and a view without " + hiding.String() +
			" does not show it"}
	}
	return nil
}
