package history

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/origins-project/origins/internal/model"
)

// Entry is a commit read back out of a corpus's history: what git records about it,
// and what origins wrote into it.
//
// The trailers decode into the same Commit the writer fills in, so what origins can
// say about a change and what it can be asked about one are a single vocabulary.
// Neither side can gain a trailer without the other.
type Entry struct {
	Sha    string
	When   time.Time
	Author string
	Commit
}

// Foreign reports a commit origins did not write. The trailer says so rather than the
// marker ref, which is what still answers after a clone, a rebase, or a ref somebody
// deleted.
func (e Entry) Foreign() bool { return e.Tool == "" }

// Query narrows a log to the commits that mention something.
//
// Everything it names is an alternative and not a further condition: the question a
// caller has is what touched any of these, and a commit that touched two of them is
// still one commit. A query naming nothing is the whole history.
type Query struct {
	// Entities are the entities to read about, found by their Origins-Subject.
	Entities []model.EntityRef

	// Sources are sources to read about: a commit resting on any citation in one of
	// them answers. Citations narrows that to one citation each, however it was
	// graded where it was cited.
	Sources   []model.Slug
	Citations []model.CitationRef

	// Limit keeps the newest this many entries, or all of them at zero.
	Limit int
}

// Read reads back the one commit a sha names, which may be written short.
func (r *Repo) Read(sha string) (Entry, error) {
	if !abbreviated.MatchString(sha) {
		return Entry{}, fmt.Errorf("%q is not a commit's sha", sha)
	}
	entries, err := r.read(Query{}, "--max-count=1", sha)
	if len(entries) == 0 {
		return Entry{}, errors.Join(fmt.Errorf("this repository has no commit %s", sha), err)
	}
	return entries[0], err
}

var abbreviated = regexp.MustCompile(`^[0-9a-f]{4,64}$`)

// Log reads commits back, newest first.
//
// A commit whose trailers origins cannot read is reported and the rest are returned,
// the way a corpus with one unreadable file still loads. History is read in order to
// answer a question, and one commit nobody can parse must not take the answer away.
func (r *Repo) Log(q Query) ([]Entry, error) {
	asked := q.Entities
	names := map[model.EntityRef]bool{}
	for _, entity := range asked {
		names[entity] = true
	}
	for {
		entries, problems := r.read(q)
		if !widen(entries, names) {
			if len(asked) > 0 {
				entries = holders(entries, asked, q)
			}
			// the limit is on the answer, so it is applied once every name the
			// entity has answered to is known and not to each search along the way
			if q.Limit > 0 && len(entries) > q.Limit {
				entries = entries[:q.Limit]
			}
			return entries, problems
		}
		q.Entities = slices.SortedFunc(maps.Keys(names), func(a, b model.EntityRef) int {
			return strings.Compare(a.String(), b.String())
		})
	}
}

// The vocabulary's words for the changes that decide which record a name belongs to,
// taken from the vocabulary rather than spelled again here.
var (
	creating  = model.Create{}.Op()
	deleting  = model.Delete{}.Op()
	renaming  = model.Rename{}.Op()
	merging   = model.Merge{}.Op()
	splitting = model.Split{}.Op()
)

// widen adds to names every other name the same entities have answered to, and reports
// whether it found any.
//
// An entity's history is the history of all of its names. A rename names the one that
// was and the one it became in a single commit, so a search that finds either finds
// both, and the search runs again until no further name turns up. Nothing else joins
// them: the slug is the identity, and a renamed entity has simply had more than one.
// A merge names the record that goes and the one it goes into the same way, and is
// followed the same way, the two of them having turned out to be one entity whose
// history was being kept twice. So is a name given up in any other commit, which says
// so in a trailer of its own and not in its subjects.
//
// Within one rename or merge, the names of a kind are the same entity before and
// after. Names of different kinds are not, because renaming a partner renames the
// family they are in, and the family is somebody else with a history of their own.
//
// What it finds is every commit that could be about the entity and not only those that
// are, since a name freed by a rename can be given to somebody else and a search by
// name finds both of them. holders is what tells the two apart.
func widen(entries []Entry, names map[model.EntityRef]bool) bool {
	grown := false
	for _, entry := range entries {
		for _, pair := range entry.Renamed {
			if names[pair.From] != names[pair.To] {
				names[pair.From], names[pair.To], grown = true, true, true
			}
		}
		if verb(entry.Op) != renaming && verb(entry.Op) != merging {
			continue
		}
		for _, ref := range paired(entry) {
			if !names[ref] {
				continue
			}
			for _, same := range paired(entry) {
				if same.Kind == ref.Kind && !names[same] {
					names[same], grown = true, true
				}
			}
		}
	}
	return grown
}

// paired is the names a rename or a merge gives in pairs, the name that was and the
// name it became. A tag renamed names its two words and then the records carrying it,
// which keep their names.
func paired(e Entry) []model.EntityRef {
	if len(e.Entities) > 2 && e.Entities[0].Kind == model.KindTag {
		return e.Entities[:2]
	}
	return e.Entities
}

// verb is the edit's own word in an operation, which is written <noun>.<verb>.
func verb(op string) string { return op[strings.LastIndex(op, ".")+1:] }

// holders keeps the entries that are about the records the names asked after stand
// for, and the ones found only because somebody else once went by one of those names
// are left out.
//
// A name belongs to one record at a time: from the commit that gives it — a create, a
// rename to it, a split making it — to the commit that takes it away — a rename from
// it, a delete, a merge into something else. Asked after, a name means whoever answers
// to it now, or whoever answered to it last where nobody does. The grandson renamed to
// make room for his grandfather has a history under the name that stops at the
// rename, and his grandfather's begins at the create that gave it to him.
//
// A merge says that two records were one person all along, so from then on they are
// one record with the history of both, and asking after either name — the one that
// stayed, or the one that went where nobody has been given it since — is asking after
// the whole of it.
//
// An entry citing a source the query also names is kept whoever it is about, everything
// a query names being an alternative.
func holders(entries []Entry, asked []model.EntityRef, q Query) []Entry {
	w := walking()
	about := make([]map[int]bool, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		about[i] = w.read(entries[i])
	}
	wanted := map[int]bool{}
	for _, name := range asked {
		if record := w.last[name]; record != 0 {
			wanted[w.one(record)] = true
		}
	}
	var kept []Entry
	for i, entry := range entries {
		keep := cites(entry, q)
		for record := range about[i] {
			keep = keep || wanted[w.one(record)]
		}
		if keep {
			kept = append(kept, entry)
		}
	}
	return kept
}

// Fate is what has become of a record since some commit.
type Fate struct {
	Now    model.EntityRef // the name it goes by now, or that of the record it went into
	Merged bool            // and whether it went into it
}

// Gone reports a record nothing goes by any more: deleted, or merged into one that was.
func (f Fate) Gone() bool { return f.Now == model.EntityRef{} }

// Since says what has become of the records that went by the given names at a commit:
// what each is called now, or what it was merged into, or that it is gone.
//
// It is the walk a log takes, begun at that commit with every one of the names already
// held, so the grandson renamed to make room for his grandfather is followed to his new
// name and not confused with the man who took his old one. Every commit since is read,
// and one whose trailers cannot be read is an error rather than something to step over,
// since the answer is a claim about where a record is and a commit left out could be
// the one that moved it.
func (r *Repo) Since(at string, names []model.EntityRef) (map[model.EntityRef]Fate, error) {
	entries, err := r.read(Query{}, at+"..HEAD")
	if err != nil {
		return nil, err
	}
	w := walking()
	records := map[model.EntityRef]int{}
	for _, name := range names {
		records[name] = w.of(name)
	}
	for i := len(entries) - 1; i >= 0; i-- {
		w.read(entries[i])
	}

	held := slices.SortedFunc(maps.Keys(w.now), func(a, b model.EntityRef) int {
		return strings.Compare(a.String(), b.String())
	})
	fates := make(map[model.EntityRef]Fate, len(records))
	for name, record := range records {
		one := w.one(record)
		for _, now := range held {
			if w.now[now] == one {
				fates[name] = Fate{Now: now, Merged: one != record}
				break
			}
		}
	}
	return fates, nil
}

// cites reports whether an entry answers for a source or a citation the query names.
func cites(e Entry, q Query) bool {
	for _, cited := range e.Sources {
		if slices.Contains(q.Sources, cited.Source) {
			return true
		}
		for _, citation := range q.Citations {
			if citation.Ungraded() == cited.Ungraded() {
				return true
			}
		}
	}
	return false
}

// walking starts a walk that has read nothing.
func walking() *walk {
	return &walk{
		now:     map[model.EntityRef]int{},
		last:    map[model.EntityRef]int{},
		joined:  map[int]int{},
		changes: map[string]map[model.EntityRef]int{},
		joins:   map[string]map[int]int{},
		about:   map[int]bool{},
	}
}

// walk reads commits oldest first and numbers the records their names stand for.
// Nothing in a commit says which record a name meant at the time; the order of the
// commits does, and the number is how the answer is carried from one to the next.
type walk struct {
	now     map[model.EntityRef]int            // the record each name stands for, where it stands for one
	last    map[model.EntityRef]int            // and the last one each stood for
	joined  map[int]int                        // the record each record was merged into, where it was
	changes map[string]map[model.EntityRef]int // what each commit gave or took, and from whom
	joins   map[string]map[int]int             // and what it merged, and into what before
	count   int

	about   map[int]bool            // the records the commit being read is about
	changed map[model.EntityRef]int // and what it has given or taken so far
	merged  map[int]int             // and what it has merged so far
}

// read works out what one commit did to the names in it and reports the records it
// is about: those whose names it gave or took, and those it only touched.
//
// Renames and merges name what they carry in pairs, the name that was and the name it
// became, which is the only way one commit renaming a partner and the families named
// after them says which family became which. Any other commit that gave a name up says
// so in Renamed, in the same pairs. A split names the record, then the name it keeps
// going by, then those it makes. An undo takes back what the commit it corrects gave,
// took and merged, which is exact because an undo only ever takes back the newest
// commit.
func (w *walk) read(e Entry) map[int]bool {
	w.about, w.changed, w.merged = map[int]bool{}, map[model.EntityRef]int{}, map[int]int{}
	names := e.Entities
	switch earlier, undoing := w.changes[e.Corrects]; {
	case e.Op == reversal && undoing:
		for name, was := range earlier {
			if was == 0 {
				w.take(name)
			} else {
				w.give(name, was)
			}
		}
		for record, was := range w.joins[e.Corrects] {
			w.join(record, was)
		}
		for _, name := range names {
			if _, restored := earlier[name]; !restored {
				w.of(name)
			}
		}
	case verb(e.Op) == creating:
		for _, name := range names {
			w.give(name, w.fresh())
		}
	case verb(e.Op) == deleting:
		for _, name := range names {
			w.of(name)
			w.take(name)
		}
	case verb(e.Op) == renaming:
		pairs := paired(e)
		for i := 0; i+1 < len(pairs); i += 2 {
			record := w.of(pairs[i])
			w.take(pairs[i])
			w.give(pairs[i+1], record)
		}
		for _, name := range names[len(pairs):] {
			w.of(name)
		}
	case verb(e.Op) == merging:
		// the first pair is the two records being made one, and after it come the
		// families named after the one that goes, handed on as any name is
		for i := 0; i+1 < len(names); i += 2 {
			if i == 0 {
				w.of(names[i+1])
			}
			w.hand(names[i], names[i+1])
		}
	case verb(e.Op) == splitting && len(names) > 1:
		record := w.of(names[0])
		if names[1] != names[0] {
			w.take(names[0])
			w.give(names[1], record)
		}
		for _, name := range names[2:] {
			w.give(name, w.fresh())
		}
	default:
		// a commit filed under anything else can still have given names up, and
		// says which in a trailer of its own
		for _, pair := range e.Renamed {
			w.hand(pair.From, pair.To)
		}
		for _, name := range names {
			if _, handed := w.changed[name]; !handed {
				w.of(name)
			}
		}
	}
	w.changes[e.Sha], w.joins[e.Sha] = w.changed, w.merged
	return w.about
}

// hand passes the record one name stands for to another: to the name itself where it
// is free, and into the record holding it where it is not, which is a merge.
func (w *walk) hand(from, to model.EntityRef) {
	record := w.of(from)
	w.take(from)
	if w.now[to] != 0 {
		w.join(w.one(record), w.one(w.of(to)))
	} else {
		w.give(to, record)
	}
}

// one is the record a record has become by being merged, which is itself where it
// has not been.
func (w *walk) one(record int) int {
	for w.joined[record] != 0 {
		record = w.joined[record]
	}
	return record
}

// join merges one record into another, or with nought takes a merge back.
func (w *walk) join(record, into int) {
	if record == into {
		return
	}
	if _, kept := w.merged[record]; !kept {
		w.merged[record] = w.joined[record]
	}
	if into == 0 {
		delete(w.joined, record)
	} else {
		w.joined[record] = into
	}
}

// of is the record a name stands for. A name standing for none is somebody arriving
// by some other road than a create — an import, a file written by hand and adopted —
// and is somebody new; that is not a change the commit made, and an undo does not
// take it back.
func (w *walk) of(name model.EntityRef) int {
	record := w.now[name]
	if record == 0 {
		record = w.fresh()
		w.now[name], w.last[name] = record, record
	}
	w.about[record] = true
	return record
}

func (w *walk) give(name model.EntityRef, record int) {
	w.remember(name)
	w.now[name], w.last[name] = record, record
	w.about[record] = true
}

func (w *walk) take(name model.EntityRef) {
	w.remember(name)
	if record := w.now[name]; record != 0 {
		w.about[record] = true
	}
	delete(w.now, name)
}

// remember keeps what a name stood for before the commit being read first changed it.
func (w *walk) remember(name model.EntityRef) {
	if _, kept := w.changed[name]; !kept {
		w.changed[name] = w.now[name]
	}
}

func (w *walk) fresh() int {
	w.count++
	return w.count
}

// The fields of one commit and the end of it. Git will not carry a NUL in a commit
// message, so no subject can be mistaken for the end of a field; the record separator
// is there for being a character nobody types. The last field is the marker where it
// is on the commit and empty everywhere else, git being asked to decorate a commit with
// that ref and no other.
const (
	entryFormat = "%H%x00%aI%x00%an%x00%s%x00%(trailers:only=true,unfold=true)%x00%b%x00%D%x00%x1e"
	fieldEnd    = "\x00"
	entryEnd    = "\x1e"
)

// entryShape is how a log is asked to write each commit, for decode to read.
var entryShape = []string{"--format=" + entryFormat, "--decorate-refs=" + marker}

func (r *Repo) read(q Query, revisions ...string) ([]Entry, error) {
	entries, _, err := r.reading(q, revisions...)
	return entries, err
}

// reading is read saying also which of the commits it read the marker is on, where it
// is on one of them.
func (r *Repo) reading(q Query, revisions ...string) ([]Entry, string, error) {
	// in the order the commits were made on top of one another, which is the order
	// holders reads them in and which the order of their dates is only in a line
	args := append([]string{"log", "--no-show-signature", "--topo-order", "--extended-regexp"},
		entryShape...)
	for _, pattern := range matching(q) {
		args = append(args, "--grep="+pattern)
	}
	args = append(args, revisions...)
	out, err := r.git(args...)
	if err != nil {
		if _, unborn := r.git("rev-parse", "--verify", "-q", "HEAD"); unborn != nil {
			// a repository nobody has committed to has a history of no commits,
			// which is an answer rather than a failure
			return nil, "", nil
		}
		return nil, "", err
	}
	return decode(out)
}

// matching is what a query asks git to look for: one pattern per thing named, which
// git takes as alternatives. A trailer is a whole line, so a pattern is anchored at
// both ends and a name cannot match another that merely begins the same way.
func matching(q Query) []string {
	var patterns []string
	for _, entity := range q.Entities {
		name := regexp.QuoteMeta(entity.String())
		patterns = append(patterns, "^Origins-Subject: "+name+"$",
			"^Origins-Renamed: ([^ ]+ )?"+name+"( [^ ]+)?$")
	}
	for _, source := range q.Sources {
		// a citation is written source#anchor, so the hash is the boundary that
		// keeps one source from answering for another it happens to begin
		patterns = append(patterns, "^Origins-Source: "+regexp.QuoteMeta(string(source))+"#")
	}
	for _, citation := range q.Citations {
		// whatever grade it was cited with, which follows it after a space
		patterns = append(patterns, "^Origins-Source: "+regexp.QuoteMeta(citation.Ungraded().String())+"( |$)")
	}
	return patterns
}

func decode(out string) ([]Entry, string, error) {
	var (
		entries  []Entry
		marked   string
		problems []error
	)
	for _, record := range strings.Split(out, entryEnd) {
		record = strings.TrimLeft(record, "\n")
		if record == "" {
			continue
		}
		fields := strings.Split(record, fieldEnd)
		if len(fields) < 7 {
			problems = append(problems, fmt.Errorf("git described a commit in %d fields, not 7", len(fields)))
			continue
		}
		if fields[6] != "" {
			marked = fields[0]
		}
		entry := Entry{Sha: fields[0], Author: fields[2]}
		entry.Subject = fields[3]

		when, err := time.Parse(time.RFC3339, fields[1])
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", entry.Sha, err))
		}
		entry.When = when
		if err := entry.trailers(fields[4]); err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", entry.Sha, err))
		}
		entry.Body = body(fields[5], fields[4])
		entries = append(entries, entry)
	}
	return entries, marked, errors.Join(problems...)
}

// body is what a commit says between its subject and its trailers: everything after the
// subject but the last paragraph, where git found trailers there.
func body(message, trailers string) string {
	message = strings.TrimSpace(message)
	if strings.TrimSpace(trailers) != "" {
		if i := strings.LastIndex(message, "\n\n"); i >= 0 {
			message = message[:i]
		} else {
			message = ""
		}
	}
	return strings.TrimSpace(message)
}

// trailers reads a commit's trailers back into what the writer put there.
//
// A trailer origins does not know is left alone. Git's own Signed-off-by is nobody's
// business here, and a commit is not less readable for carrying one.
func (e *Entry) trailers(block string) error {
	var problems []error
	for _, line := range strings.Split(block, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		value = strings.TrimSpace(value)

		var err error
		switch key {
		case "Origins-Tool":
			e.Tool = value
		case "Origins-Op":
			e.Op = value
		case "Origins-Subject":
			var ref model.EntityRef
			if ref, err = model.ParseEntityRef(value); err == nil {
				e.Entities = append(e.Entities, ref)
			}
		case "Origins-Renamed":
			var renaming Renaming
			if renaming, err = parseRenaming(value); err == nil {
				e.Renamed = append(e.Renamed, renaming)
			}
		case "Origins-Event":
			e.Events = append(e.Events, value)
		case "Origins-Source":
			var ref model.CitationRef
			if ref, err = model.ParseCitationRef(value); err == nil {
				e.Sources = append(e.Sources, ref)
			}
		case "Origins-Corrects":
			e.Corrects = value
		case "Origins-Hypothesis":
			var slug model.Slug
			if slug, err = model.ParseSlug(value); err == nil {
				e.Hypothesis = slug
			}
		case "Origins-Proposed-By":
			e.ProposedBy = value
		}
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", key, err))
		}
	}
	return errors.Join(problems...)
}
