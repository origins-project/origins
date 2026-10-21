package browse

import (
	"net/http"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// A draft is supposing from the pages: each change a page makes into it is taken into a
// hypothesis still being written rather than recorded, and every read that asks for it
// is answered from the corpus with the draft in it, so a change after the first is made
// looking at what the ones before it did. Kept, it is one hypothesis and one commit;
// thrown away, it leaves nothing.

// DraftParam is the query parameter a read asks for the corpus with the draft in it by.
const DraftParam = "draft"

// drafted is the corpus with the draft in it, built again whenever the corpus is read
// again or the draft changes, and the draft as a page is shown it.
type drafted struct {
	corpus *ops.Drafted
	shown  *api.Drafting
	err    error // why the draft could not be read or carried, where it could not
}

// redraft builds the draft afresh over the corpus just read, closing the one it
// replaces. Only a server that takes changes has one, a copy taking none.
func (s *Server) redraft() {
	_ = s.drafted.corpus.Close()
	s.drafted = drafted{}
	if !s.writes() {
		return
	}
	corpus, shown, err := api.Drafted(s.corpus)
	s.drafted = drafted{corpus: corpus, shown: shown, err: err}
}

// draftedFor is the corpus with the draft in it, where a read asks for it.
func (s *Server) draftedFor(r *http.Request, hiding model.Hiding) (*store.Corpus, bool, error) {
	if !r.URL.Query().Has(DraftParam) {
		return nil, false, nil
	}
	switch {
	case hiding.Any():
		return nil, true, &askedError{"a draft is the corpus whole, and is not shown under a view without " +
			hiding.String()}
	case !s.writes():
		return nil, true, &readOnlyError{"a draft is a change in the making, and this server takes none"}
	case s.drafted.err != nil:
		return nil, true, s.drafted.err
	case s.drafted.corpus == nil:
		return nil, true, ops.Undrafted{}
	}
	return s.drafted.corpus.Corpus, true, nil
}

// draft answers the draft as a page is shown it, or null where none has been begun:
// asking whether there is one is a read, and nothing to show is its answer.
func (s *Server) draft(w http.ResponseWriter, r *http.Request) {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return
	}
	switch {
	case s.drafted.shown != nil:
		answer(w, http.StatusOK, s.drafted.shown)
	case s.drafted.err != nil:
		refused(w, s.drafted.err)
	default:
		answer(w, http.StatusOK, nil)
	}
}

func (s *Server) draftEdit(w http.ResponseWriter, r *http.Request) {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return
	}
	var asked Change
	if !decoded(w, r, &asked) {
		return
	}
	switch {
	case len(asked.Edit) == 0:
		refused(w, &askedError{"a change says the edit it makes, as edit"})
		return
	case asked.Dry || asked.Message != "" || len(asked.Sources) > 0:
		refused(w, &askedError{"a change to a draft records nothing, so it says nothing a commit would"})
		return
	}
	edit, err := store.ParseEdit(asked.Edit)
	if err != nil {
		refused(w, &askedError{err.Error()})
		return
	}
	asking, _ := s.asking(asked.Seen, nil, "", false)
	s.writing(func() {
		report, err := api.DraftEdit(asking, edit)
		s.wrote()
		answered(w, report, err)
	})
}

func (s *Server) draftPop(w http.ResponseWriter, r *http.Request) {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return
	}
	asking, _ := s.asking("", nil, "", false)
	s.writing(func() {
		report, err := api.DraftPop(asking)
		s.wrote()
		answered(w, report, err)
	})
}

func (s *Server) draftDiscard(w http.ResponseWriter, r *http.Request) {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return
	}
	asking, _ := s.asking("", nil, "", false)
	s.writing(func() {
		err := api.DraftDiscard(asking)
		s.wrote()
		if err != nil {
			refused(w, err)
			return
		}
		answer(w, http.StatusOK, struct{}{})
	})
}

// Kept is how a page asks for the draft to be kept as a hypothesis.
type Kept struct {
	Slug        string   `json:"slug"`
	Question    string   `json:"question"`
	Argument    string   `json:"argument"`
	Confidence  string   `json:"confidence"`
	Supports    []string `json:"supports"`
	Contradicts []string `json:"contradicts"`
	Message     string   `json:"message"`
	Dry         bool     `json:"dry"`
}

func (s *Server) draftKeep(w http.ResponseWriter, r *http.Request) {
	if err := s.readOnly(r); err != nil {
		refused(w, err)
		return
	}
	var asked Kept
	if !decoded(w, r, &asked) {
		return
	}
	keeping, err := asked.keeping()
	if err != nil {
		refused(w, err)
		return
	}
	asking, _ := s.asking("", nil, asked.Message, asked.Dry)
	s.writing(func() {
		report, err := api.Keep(asking, keeping)
		s.wrote()
		answered(w, report, err)
	})
}

func (k Kept) keeping() (api.Keeping, error) {
	slug, err := model.ParseSlug(k.Slug)
	if err != nil {
		return api.Keeping{}, &askedError{err.Error()}
	}
	keeping := api.Keeping{Slug: slug, Question: k.Question, Argument: k.Argument}
	if k.Confidence != "" {
		if keeping.Confidence, err = model.ParseConfidence(k.Confidence); err != nil {
			return api.Keeping{}, &askedError{err.Error()}
		}
	}
	for _, list := range []struct {
		written []string
		into    *[]model.CitationRef
	}{{k.Supports, &keeping.Supports}, {k.Contradicts, &keeping.Contradicts}} {
		for _, written := range list.written {
			ref, err := model.ParseCitationRef(written)
			if err != nil {
				return api.Keeping{}, &askedError{err.Error()}
			}
			*list.into = append(*list.into, ref)
		}
	}
	return keeping, nil
}
