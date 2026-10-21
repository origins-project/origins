package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
)

// The questions about a corpus's history do not load the corpus.
//
// History outlives the model. A person deleted last year still has one, a person
// renamed has one under every name, and a slug the corpus has never held is a
// perfectly good thing to ask about. Loading first would make the answer depend on
// what the corpus happens to say today, which is the one thing history is asked
// instead of.

// Entry is one commit as the report gives it: the trailers spelled out, since decoding
// them is what origins adds to git log.
type Entry struct {
	Commit     string   `json:"commit"`
	Date       string   `json:"date"`
	Author     string   `json:"author"`
	Subject    string   `json:"subject"`
	Tool       string   `json:"tool"`
	Op         string   `json:"op"`
	Subjects   []string `json:"subjects"`
	Renamed    []string `json:"renamed"`
	Events     []string `json:"events"`
	Sources    []string `json:"sources"`
	Corrects   string   `json:"corrects"`
	Hypothesis string   `json:"hypothesis"`
	ProposedBy string   `json:"proposed-by"`

	// When is the date as a time, and Foreign whether origins did not write the commit.
	When    time.Time `json:"-"`
	Foreign bool      `json:"-"`
}

// described is one commit as the report gives it.
func described(read history.Entry) Entry {
	return Entry{
		Commit:     read.Sha,
		Date:       read.When.Format(time.RFC3339),
		Author:     read.Author,
		Subject:    read.Subject,
		Tool:       read.Tool,
		Op:         read.Op,
		Subjects:   written(read.Entities),
		Renamed:    written(read.Renamed),
		Events:     append([]string{}, read.Events...),
		Sources:    written(read.Sources),
		Corrects:   read.Corrects,
		Hypothesis: string(read.Hypothesis),
		ProposedBy: read.ProposedBy,
		When:       read.When,
		Foreign:    read.Foreign(),
	}
}

// Journal is what the history says.
type Journal struct {
	Entries  []Entry  `json:"entries"`
	Problems []string `json:"problems"`
}

// Log is the history of the corpus, or of one entity in it, decoded from the trailers
// origins wrote, the newest limit entries of it where limit is more than nought.
func Log(root string, entity string, limit int) (*Journal, error) {
	var q history.Query
	if entity != "" {
		ref, err := model.ParseEntityRef(entity)
		if err != nil {
			return nil, err
		}
		q.Entities = []model.EntityRef{ref}
	}
	return ask(root, q, limit)
}

// Whence is every conclusion ever drawn from a source. It is the question to ask when
// a source turns out to be unreliable, since what rests on it is what has to be looked
// at again.
func Whence(root string, source string, limit int) (*Journal, error) {
	var q history.Query
	// A source is asked about whole. Naming a citation in it narrows the question to
	// the one entry, which is what a register with a single mistranscribed page needs.
	if strings.Contains(source, "#") {
		citation, err := model.ParseCitationRef(source)
		if err != nil {
			return nil, err
		}
		q.Citations = []model.CitationRef{citation}
	} else {
		slug, err := model.ParseSlug(source)
		if err != nil {
			return nil, err
		}
		q.Sources = []model.Slug{slug}
	}
	return ask(root, q, limit)
}

// ask puts a question to the history and reports what comes back.
func ask(root string, q history.Query, limit int) (*Journal, error) {
	repo, err := owned(root)
	if err != nil {
		return nil, err
	}
	q.Limit = limit
	entries, problems := repo.Log(q)
	report := &Journal{Entries: []Entry{}, Problems: Flatten(problems)}
	for _, read := range entries {
		report.Entries = append(report.Entries, described(read))
	}
	return report, nil
}

// Snapshots is the snapshots taken, or the one just taken.
type Snapshots struct {
	Snapshots []Named `json:"snapshots"`
}

// Named is one snapshot.
type Named struct {
	Name    string `json:"name"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Message string `json:"message"`

	When time.Time `json:"-"`
}

// Snapshot names the state of the corpus name, saying message of it, or where name is
// empty says which states have been named.
//
// A snapshot is a tag and not a record origins keeps to itself, so every tag is one.
// What was sent to a relative has to be something they can be pointed at with git
// alone, and a tag somebody made by hand is the same statement about the same commit.
func Snapshot(root, name, message string) (report *Snapshots, err error) {
	repo, err := owned(root)
	if err != nil {
		return nil, err
	}
	var listed []history.Snapshot
	if name != "" {
		release, err := repo.Lock("origins snapshot")
		if err != nil {
			return nil, err
		}
		defer func() { err = joined(err, release()) }()
		taken, err := repo.Take(name, message)
		if err != nil {
			return nil, err
		}
		listed = []history.Snapshot{taken}
	} else {
		if message != "" {
			return nil, fmt.Errorf("-m says what a state is being named for, so it needs a name to go with")
		}
		if listed, err = repo.Snapshots(); err != nil {
			return nil, err
		}
	}

	report = &Snapshots{Snapshots: []Named{}}
	for _, taken := range listed {
		report.Snapshots = append(report.Snapshots, Named{
			Name:    taken.Name,
			Commit:  taken.Sha,
			Date:    taken.When.Format(time.RFC3339),
			Message: taken.Message,
			When:    taken.When,
		})
	}
	return report, nil
}
