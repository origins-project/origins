package browse

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// A write is asked for as the command line asks for one and goes the same road,
// internal/api and then ops, so that a page cannot change the corpus in a way a command
// could not or record it in other terms. The only thing added here is what a page needs
// besides: where to go once the record it was on has gone or been renamed.

// command is what the lock says is writing while a page's change is being made.
const command = "browse"

// Change is an edit asked for from a page.
type Change struct {
	Edit    json.RawMessage `json:"edit"`    // one edit, as a hypothesis writes it
	Seen    string          `json:"seen"`    // the record's seen as the page was shown it
	Sources []string        `json:"sources"` // the citations the change rests on
	Message string          `json:"message"` // the subject line, in place of the one origins writes
	Dry     bool            `json:"dry"`     // say what the change comes to and write none of it
}

// Made is what came of a change: what api.Carry says of it, and the records that now
// stand where the ones it was about stood, for the page to go to.
type Made struct {
	*api.Recorded
	Empty    bool     `json:"empty"`
	Stands   []string `json:"stands"`
	Problems []string `json:"problems"`
}

// Settling is how a page asks to take a change back or settle a hypothesis.
type Settling struct {
	Seen    string   `json:"seen"`
	Sources []string `json:"sources"`
	Message string   `json:"message"`
	Dry     bool     `json:"dry"`
}

// Rendering is prose to render for a record, as a page previews what it is writing.
type Rendering struct {
	Entity string `json:"entity"`
	Prose  string `json:"prose"`
}

// Rendered is prose rendered.
type Rendered struct {
	HTML string `json:"html"`
}

func (s *Server) edit(w http.ResponseWriter, r *http.Request) {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return
	}
	var asked Change
	if !decoded(w, r, &asked) {
		return
	}
	if len(asked.Edit) == 0 {
		refused(w, &askedError{"a change says the edit it makes, as edit"})
		return
	}
	edit, err := store.ParseEdit(asked.Edit)
	if err != nil {
		refused(w, &askedError{err.Error()})
		return
	}
	asking, err := s.asking(asked.Seen, asked.Sources, asked.Message, asked.Dry)
	if err != nil {
		refused(w, err)
		return
	}
	s.writing(func() {
		report, err := api.Carry(asking, edit)
		s.wrote()
		if report == nil {
			refused(w, err)
			return
		}
		made := Made{Recorded: report, Empty: report.Empty, Stands: []string{}, Problems: api.Flatten(err)}
		if !asked.Dry {
			for _, ref := range standing(s.corpus, edit, report) {
				made.Stands = append(made.Stands, ref.String())
			}
		}
		answer(w, http.StatusOK, made)
	})
}

func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return
	}
	var asked Settling
	if !decoded(w, r, &asked) {
		return
	}
	asking, err := s.asking(asked.Seen, asked.Sources, asked.Message, asked.Dry)
	if err != nil {
		refused(w, err)
		return
	}
	s.writing(func() {
		report, err := api.Undo(asking)
		s.wrote()
		answered(w, report, err)
	})
}

func (s *Server) decide(status model.Status) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := s.readOnly(r); err != nil {
			refused(w, err)
			return
		}
		slug, err := model.ParseSlug(r.PathValue("slug"))
		if err != nil {
			refused(w, &askedError{err.Error()})
			return
		}
		var asked Settling
		if !decoded(w, r, &asked) {
			return
		}
		asking, err := s.asking(asked.Seen, asked.Sources, asked.Message, asked.Dry)
		if err != nil {
			refused(w, err)
			return
		}
		s.writing(func() {
			report, err := api.Decide(asking, slug, status)
			s.wrote()
			answered(w, report, err)
		})
	}
}

func (s *Server) render(w http.ResponseWriter, r *http.Request) {
	var asked Rendering
	if !decoded(w, r, &asked) {
		return
	}
	owner, err := model.ParseEntityRef(asked.Entity)
	if err != nil {
		refused(w, &askedError{err.Error()})
		return
	}
	answer(w, http.StatusOK, Rendered{HTML: s.markdown.render(owner, asked.Prose)})
}

// writing makes a change under the lock every read waits on, so that no page is
// answered from the corpus while it is being changed and read again.
func (s *Server) writing(change func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	change()
}

// asking is a change asked for from a page, as internal/api takes it.
func (s *Server) asking(seen string, sources []string, message string, dry bool) (api.Asking, error) {
	asking := api.Asking{Root: s.root, Command: command, Message: message, DryRun: dry, Seen: seen}
	for _, written := range sources {
		ref, err := model.ParseCitationRef(written)
		if err != nil {
			return api.Asking{}, &askedError{err.Error()}
		}
		asking.Sources = append(asking.Sources, ref)
	}
	return asking, nil
}

// decoded reads the JSON a write was sent with into into, answering the request itself
// where it is not that. A body left out is an empty one, a write that needs nothing
// said besides what its route says.
func decoded(w http.ResponseWriter, r *http.Request, into any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(into)
	if err == nil || errors.Is(err, io.EOF) {
		return true
	}
	refused(w, &askedError{"the request is not what it says it is: " + err.Error()})
	return false
}

// standing is every record the edit was about that the corpus holds after it, under
// whatever name it holds it: a record renamed is where it went, a record merged is the
// one it went into, and a record deleted stands nowhere.
func standing(c *store.Corpus, edit model.Edit, report *api.Recorded) []model.EntityRef {
	moved := map[string]string{}
	for _, carried := range report.Moved {
		moved[carried.From] = carried.To
	}
	var out []model.EntityRef
	stands := func(ref model.EntityRef) {
		for _, already := range out {
			if already == ref {
				return
			}
		}
		out = append(out, ref)
	}
	for _, ref := range edit.Subjects() {
		switch {
		case ref == model.TheProject:
			stands(ref)
		case ref.Kind == model.KindTag:
			if rename, ok := edit.(model.Rename); ok {
				stands(model.EntityRef{Kind: model.KindTag, Slug: rename.To})
			} else {
				stands(ref)
			}
		case c.KindOf(ref.Slug) == ref.Kind:
			stands(ref)
		default:
			if to, ok := moved[ref.Kind.Dir()+"/"+string(ref.Slug)]; ok {
				dir, slug := path.Split(strings.TrimSuffix(to, "/"))
				kind := model.KindIn(strings.TrimSuffix(dir, "/"))
				if kind != "" && c.KindOf(model.Slug(slug)) == kind {
					stands(model.EntityRef{Kind: kind, Slug: model.Slug(slug)})
				}
			}
		}
	}
	return out
}
