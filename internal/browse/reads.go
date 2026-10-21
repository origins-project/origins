package browse

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Session is what the application asks first: what it is serving, whether it may
// change it, whether a server is behind it at all, which an export, its answers written
// down beforehand, has not, whether it may ask of the history, and what an export was
// made from.
type Session struct {
	Project  string            `json:"project"`
	Writes   bool              `json:"writes"`
	Live     bool              `json:"live"`
	History  bool              `json:"history"`
	View     *Hiding           `json:"view"`
	With     []api.Supposition `json:"with"`
	Problems []string          `json:"problems"`
	Export   *ExportedFrom     `json:"export"`
}

// ExportedFrom is what an export was made from: when, by the clock of the machine that
// made it, the commit the corpus stood on and whether its files differed from it, the
// commit empty for a corpus without a history, the origins that made it, and the notice
// its pages show.
type ExportedFrom struct {
	At      time.Time `json:"at"`
	Commit  string    `json:"commit"`
	Changed bool      `json:"changed"`
	Version string    `json:"version"`
	Notice  string    `json:"notice"`
}

// Hiding is what a view leaves out.
type Hiding struct {
	Private bool `json:"private"`
	Living  bool `json:"living"`
}

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	session := Session{
		Project:  s.corpus.Project.Name,
		Writes:   s.writes(),
		With:     []api.Supposition{},
		Problems: api.Flatten(s.problems),
	}
	session.Live, session.History = s.asks()
	// an export is locked to the view it was made as, since its pages read no other
	if locked := s.options.Hiding; locked.Any() || s.exported {
		session.View = &Hiding{Private: locked.Private, Living: locked.Living}
	}
	session.With = append(session.With, s.supposed.read...)
	if s.exported {
		session.Export = &s.made
	}
	answer(w, http.StatusOK, session)
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	c, _, err := s.served(r)
	if err != nil {
		refused(w, err)
		return
	}
	_, read, _, _ := s.supposedFor(r)
	index := api.IndexOfCorpus(c.Corpus, s.today())
	index.Read = read
	index.Problems = api.Flatten(s.problems)
	answer(w, http.StatusOK, index)
}

// Shown is a record as a page is given it: the record, with each piece of its prose
// rendered as HTML under its path.
type Shown struct {
	*api.Record
	HTML map[string]string `json:"html"`

	// Written is the manifest as origins writes it, for the project alone.
	Written string `json:"written,omitempty"`
}

// Page is a record and every record it refers to or that refers to it, one step out,
// which is what a page about it draws besides the record itself.
type Page struct {
	Record Shown   `json:"record"`
	Around []Shown `json:"around"`
}

func (s *Server) record(w http.ResponseWriter, r *http.Request) {
	c, _, err := s.served(r)
	if err != nil {
		refused(w, err)
		return
	}
	ref, err := model.ParseEntityRef(r.PathValue("kind") + "/" + r.PathValue("slug"))
	if err != nil || ref.Kind == model.KindProject || ref.Kind == model.KindTag ||
		c.KindOf(ref.Slug) != ref.Kind {
		refused(w, &missingError{"there is no " + r.PathValue("kind") + "/" + r.PathValue("slug") +
			" in " + c.Root})
		return
	}
	shown, err := s.shown(c, ref)
	if err != nil {
		refused(w, err)
		return
	}
	page := Page{Record: shown, Around: []Shown{}}
	for _, near := range around(c.Corpus, ref) {
		one, err := s.shown(c, near)
		if err != nil {
			refused(w, err)
			return
		}
		page.Around = append(page.Around, one)
	}
	answer(w, http.StatusOK, page)
}

func (s *Server) project(w http.ResponseWriter, r *http.Request) {
	c, _, err := s.served(r)
	if err != nil {
		refused(w, err)
		return
	}
	shown, err := s.shown(c, model.TheProject)
	if err != nil {
		refused(w, err)
		return
	}
	answer(w, http.StatusOK, Page{Record: shown, Around: []Shown{}})
}

// shown is a record of the corpus served, with its prose rendered.
func (s *Server) shown(c *store.Corpus, ref model.EntityRef) (Shown, error) {
	record, err := api.RecordOf(c, ref)
	if err != nil {
		return Shown{}, err
	}
	shown := Shown{Record: record, HTML: map[string]string{}}
	if ref == model.TheProject {
		written, err := store.EmitProject(c.Project)
		shown.Written = string(written)
		return shown, err
	}
	model.Prose(c.Entity(ref.Slug), func(path string, prose *string) {
		if rendered := s.markdown.render(ref, *prose); rendered != "" {
			shown.HTML[path] = rendered
		}
	})
	return shown, nil
}

// around is every record ref refers to and every record referring to it, a hypothesis
// referring to whatever its edits are about, in the order the kinds and the slugs give.
func around(c *model.Corpus, ref model.EntityRef) []model.EntityRef {
	found := map[model.EntityRef]bool{}
	near := func(slug model.Slug) {
		if kind := c.KindOf(slug); kind != "" && slug != ref.Slug {
			found[model.EntityRef{Kind: kind, Slug: slug}] = true
		}
	}
	entity := c.Entity(ref.Slug)
	model.References(entity, func(slug *model.Slug) { near(*slug) })
	if h, ok := entity.(*model.Hypothesis); ok {
		for _, concerned := range h.Concerns() {
			near(concerned.Slug)
		}
	}
	for _, slug := range c.AllSlugs() {
		if slug == ref.Slug {
			continue
		}
		other := c.Entity(slug)
		refers := false
		model.References(other, func(to *model.Slug) { refers = refers || *to == ref.Slug })
		if h, ok := other.(*model.Hypothesis); ok {
			refers = refers || slices.Contains(h.Concerns(), ref)
		}
		if refers {
			near(slug)
		}
	}
	out := make([]model.EntityRef, 0, len(found))
	for _, kind := range model.Kinds {
		for _, slug := range c.SlugsOf(kind) {
			if one := (model.EntityRef{Kind: kind, Slug: slug}); found[one] {
				out = append(out, one)
			}
		}
	}
	return out
}

func (s *Server) findings(w http.ResponseWriter, r *http.Request) {
	c, _, err := s.served(r)
	if err != nil {
		refused(w, err)
		return
	}
	answer(w, http.StatusOK, api.FindingsOf(c.Corpus, s.problems, s.today()))
}

func (s *Server) log(w http.ResponseWriter, r *http.Request) {
	if err := s.unhidden(r); err != nil {
		refused(w, err)
		return
	}
	limit, err := limited(r)
	if err != nil {
		refused(w, err)
		return
	}
	report, err := api.Log(s.root, r.URL.Query().Get("entity"), limit)
	answered(w, report, err)
}

func (s *Server) whence(w http.ResponseWriter, r *http.Request) {
	if err := s.unhidden(r); err != nil {
		refused(w, err)
		return
	}
	limit, err := limited(r)
	if err != nil {
		refused(w, err)
		return
	}
	source := r.URL.Query().Get("source")
	if source == "" {
		refused(w, &askedError{"whence names a source, as ?source=<slug> or <slug>#<anchor>"})
		return
	}
	report, err := api.Whence(s.root, source, limit)
	answered(w, report, err)
}

func (s *Server) blame(w http.ResponseWriter, r *http.Request) {
	if err := s.unhidden(r); err != nil {
		refused(w, err)
		return
	}
	entity := r.URL.Query().Get("entity")
	if entity == "" {
		refused(w, &askedError{"blame names an entity, as ?entity=<kind/slug>"})
		return
	}
	report, err := api.Blame(s.root, entity, r.URL.Query().Get("field"))
	answered(w, report, err)
}

func (s *Server) why(w http.ResponseWriter, r *http.Request) {
	if err := s.unhidden(r); err != nil {
		refused(w, err)
		return
	}
	entity, path := r.URL.Query().Get("entity"), r.URL.Query().Get("path")
	if entity == "" || path == "" {
		refused(w, &askedError{"why names an entity and a path, as ?entity=<kind/slug>&path=<path>"})
		return
	}
	report, err := api.Why(s.root, entity, path)
	answered(w, report, err)
}

func (s *Server) kinship(w http.ResponseWriter, r *http.Request) {
	from, to := r.URL.Query().Get("from"), r.URL.Query().Get("to")
	if from == "" || to == "" {
		refused(w, &askedError{"kinship names two people, as ?from=<slug>&to=<slug>"})
		return
	}
	c, _, err := s.served(r)
	if err != nil {
		refused(w, err)
		return
	}
	report, err := api.KinshipOf(c, from, to)
	answered(w, report, err)
}

func (s *Server) schema(w http.ResponseWriter, r *http.Request) {
	written, err := api.Schema(r.PathValue("name"))
	if err != nil {
		refused(w, &missingError{err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/schema+json")
	_, _ = w.Write(json.RawMessage(written))
}

func (s *Server) date(w http.ResponseWriter, r *http.Request) {
	read, err := api.DateOf(r.URL.Query().Get("text"))
	if err != nil {
		refused(w, &askedError{err.Error()})
		return
	}
	answer(w, http.StatusOK, read)
}

// limited is how many entries a question about the history asks for, every one of
// them where it says nothing.
func limited(r *http.Request) (int, error) {
	written := r.URL.Query().Get("n")
	if written == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(written)
	if err != nil || n < 0 {
		return 0, &askedError{"n is how many entries to give, a number, and not " + strconv.Quote(written)}
	}
	return n, nil
}
