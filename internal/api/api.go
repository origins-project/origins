// Package api is what origins answers, apart from how the answer is carried.
//
// Every question a command puts and every change it makes is a function here that
// takes what was asked and hands back a report, the type --json writes out. The
// command line parses its flags, calls one of them and prints what comes back, as text
// or as JSON, and the pages and an agent are other ways of carrying the same answers,
// so that what origins says does not depend on who asked.
//
// A function hands back a report, an error, or both. A report with no error is the
// answer. An error with no report is a refusal, and the types below and the ones ops
// and history define say which. A report with an error beside it is work that was done
// with something going wrong afterwards — a lock not given up, a commit made and not
// marked — which is said beside the answer and does not take it back.
package api

import (
	"errors"
	"fmt"
	"strings"

	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/ops"
	"github.com/origins-project/origins/internal/store"
)

// Tool is what a commit origins makes says made it.
var Tool = "origins " + Version

// Unreadable is a refusal to go on with a corpus that did not read whole, whose
// problems are the thing to put right and are said before the refusal.
type Unreadable struct {
	Problems []string
	Refusal  string
}

func (u *Unreadable) Error() string { return u.Refusal }

// Unrecorded is a corpus written and not recorded, which is the one way an import half
// works: the report stands, and this is said after it.
type Unrecorded struct {
	Root string
	Err  error
}

func (u *Unrecorded) Error() string {
	return fmt.Sprintf("%s is written, and origins could not record it: %v\n"+
		"  put that right, and origins adopt will record what is there", u.Root, u.Err)
}

func (u *Unrecorded) Unwrap() error { return u.Err }

// unreadable is a corpus that did not read whole, refused for what origins would not
// do to it.
func unreadable(problems error, refusal string) *Unreadable {
	return &Unreadable{Problems: Flatten(problems), Refusal: refusal}
}

// refusing puts an ops.Unreadable in err in the words of what origins would not do,
// leaving whatever else is in err beside it.
func refusing(err error, doing string) error {
	taken, rest := Without[*ops.Unreadable](err)
	if len(taken) == 0 {
		return err
	}
	return errors.Join(unreadable(taken[0].Problems, "origins will not "+doing+" it could not read whole"), rest)
}

// Flatten turns the collected load problems into one message per problem, however
// deeply they were joined on the way up.
func Flatten(err error) []string {
	out := []string{}
	var walk func(error)
	walk = func(err error) {
		if err == nil {
			return
		}
		var joined interface{ Unwrap() []error }
		if errors.As(err, &joined) {
			for _, one := range joined.Unwrap() {
				walk(one)
			}
			return
		}
		out = append(out, err.Error())
	}
	walk(err)
	return out
}

// Without takes every part of err that is a T out of it, handing those back apart from
// whatever else went wrong.
func Without[T error](err error) (taken []T, rest error) {
	if err == nil {
		return nil, nil
	}
	if one, ok := err.(T); ok {
		return []T{one}, nil
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return nil, err
	}
	var left []error
	for _, part := range joined.Unwrap() {
		more, still := Without[T](part)
		taken = append(taken, more...)
		if still != nil {
			left = append(left, still)
		}
	}
	if len(left) == 1 {
		return taken, left[0]
	}
	return taken, errors.Join(left...)
}

// Short is a sha as a reader quotes one back.
func Short(sha string) string {
	if len(sha) < 10 {
		return sha
	}
	return sha[:10]
}

// opened loads a corpus to be read, with a warning where it was written by a later
// origins.
func opened(root string) (corpus *store.Corpus, problems, warning error) {
	corpus, problems = store.Load(root)
	if err := store.Later(corpus.Project); err != nil {
		warning = fmt.Errorf("%w, and reads this corpus as though it were in its own", err)
	}
	return corpus, problems, warning
}

// written spells out references the way the corpus and the trailers both spell them.
func written[T fmt.Stringer](refs []T) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.String())
	}
	return out
}

// Entity is the entity a question was asked about, written kind/slug as everywhere
// else or as a bare slug. A slug is unique within a corpus, so the kind is a check
// rather than a lookup — and it is worth making, since blaming another entity's file
// would answer instead of failing.
func Entity(corpus *store.Corpus, written string) (model.EntityRef, error) {
	var ref model.EntityRef
	if strings.Contains(written, "/") {
		parsed, err := model.ParseEntityRef(written)
		if err != nil {
			return ref, err
		}
		ref = parsed
	} else {
		slug, err := model.ParseSlug(written)
		if err != nil {
			return ref, err
		}
		ref.Slug = slug
	}

	is := corpus.KindOf(ref.Slug)
	switch {
	case is == "":
		return ref, fmt.Errorf("there is nothing called %s in %s", ref.Slug, corpus.Root)
	case ref.Kind == "":
		ref.Kind = is
	case ref.Kind != is:
		return ref, fmt.Errorf("%s is a %s", ref.Slug, is)
	}
	return ref, nil
}

// owned opens the repository a corpus's history is kept in, refusing one whose history
// origins cannot account for. Every question about the history comes through here or
// through the ops that do, and history.Owned says why.
func owned(dir string) (*history.Repo, error) {
	return history.Owned(dir)
}

// writing opens the repository as owned does and takes the lock on it, handing back
// what gives the lock up, which the caller defers and joins into what it returns.
//
// A function that writes reads the corpus after this and never before. A corpus read
// before the lock is one the writer that held it may since have changed, and writing a
// change over it would take that writer's change back without a word.
func writing(dir, command string) (*history.Repo, func() error, error) {
	repo, err := owned(dir)
	if err != nil {
		return nil, nothing, err
	}
	release, err := repo.Lock("origins " + command)
	if err != nil {
		return nil, nothing, err
	}
	return repo, release, nil
}

func nothing() error { return nil }

// joined is err with what giving up a lock said beside it.
func joined(err, released error) error {
	if released == nil {
		return err
	}
	return errors.Join(err, released)
}
