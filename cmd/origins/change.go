package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// The commands that change a corpus are the edit vocabulary and nothing besides. There
// is no origins add-person taking a flag per field: the same nine operations serve a
// person, a place and a source, they are the operations a hypothesis is written in, and
// a tenth command cannot appear without a tenth word to go with it.
//
// Everything goes through the one applier and lands as one commit, so a change typed
// here and a change accepted from a hypothesis reach the corpus by the same road and
// are recorded in the same terms.

// changes is the family of them, which is how run reaches one and how a test holds
// the two lists to each other. A command that is not one of the operations and an
// operation with no command are the same mistake, and neither can be made quietly.
var changes = map[string]func(args []string, stdout, stderr io.Writer) int{
	"set":    setValue,
	"unset":  unsetValue,
	"add":    addValue,
	"remove": removeValue,
	"create": createEntity,
	"delete": deleteEntity,
	"rename": renameEntity,
	"merge":  mergeEntity,
	"split":  splitEntity,
}

// change is what every mutation command takes besides its own arguments.
//
// The corpus is a flag rather than a trailing directory, as it is for the commands that
// only read one, because here the positional arguments are the edit itself.
type change struct {
	name    string
	root    string
	message string
	sources citations
	dryRun  bool
	asJSON  bool
}

// citations collects a repeatable --source, each read the way the corpus reads one, so
// a reference that could not be written into a file cannot be written into a trailer.
type citations []model.CitationRef

func (c *citations) String() string {
	out := make([]string, 0, len(*c))
	for _, ref := range *c {
		out = append(out, ref.String())
	}
	return strings.Join(out, " ")
}

func (c *citations) Set(value string) error {
	ref, err := model.ParseCitationRef(value)
	if err != nil {
		return err
	}
	*c = append(*c, ref)
	return nil
}

func changeFlags(name string, args []string, stdout, stderr io.Writer, more ...func(*flag.FlagSet)) (*change, []string, int) {
	c := &change{name: name}
	flags := newFlags(name, stderr)
	flags.StringVar(&c.root, "C", ".", "the `corpus` to change")
	flags.StringVar(&c.message, "m", "", "the `subject` line, in place of the one origins writes")
	flags.Var(&c.sources, "source", "a `citation` the change rests on, repeatable")
	flags.BoolVar(&c.dryRun, "dry-run", false, "say what would change without changing it")
	flags.BoolVar(&c.asJSON, "json", false, "report as JSON on stdout")
	for _, add := range more {
		add(flags)
	}
	if code := parse(flags, args, stdout); code != exitOK {
		return nil, nil, code
	}
	rest := flags.Args()
	if code := flagsFirst(name, flags, rest, stderr); code != exitOK {
		return nil, nil, code
	}
	return c, rest, exitOK
}

// asking is how the change was asked for, as internal/api takes it.
func (c *change) asking() api.Asking {
	return api.Asking{
		Root:    c.root,
		Command: c.name,
		Message: c.message,
		Sources: c.sources,
		DryRun:  c.dryRun,
	}
}

// carry runs one edit against a corpus and records it as a commit.
func (c *change) carry(edit model.Edit, stdout, stderr io.Writer) int {
	report, err := api.Carry(c.asking(), edit)
	return c.reported(report, err, stdout, stderr)
}

// reported says what came of carrying an edit.
func (c *change) reported(report *api.Recorded, err error, stdout, stderr io.Writer) int {
	return answer(report, err, c.asJSON, stdout, stderr, func(report *api.Recorded) {
		if report.Empty {
			fmt.Fprintln(stdout, "the corpus already says that, so nothing was written")
			return
		}
		carried, wrote, gone := "  %s is now %s\n", "  %s\n", "  %s (gone)\n"
		if c.dryRun {
			fmt.Fprintf(stdout, "%s\n", report.Subject)
			carried, wrote, gone = "  would move %s to %s\n", "  would write %s\n", "  would remove %s\n"
		} else {
			fmt.Fprintf(stdout, "%s %s\n", short(report.Commit), report.Subject)
		}
		for _, moved := range report.Moved {
			fmt.Fprintf(stdout, carried, moved.From, moved.To)
		}
		for _, path := range report.Written {
			fmt.Fprintf(stdout, wrote, path)
		}
		for _, path := range report.Removed {
			fmt.Fprintf(stdout, gone, path)
		}
	}, nil)
}

// entityAnd reads the arguments every mutation command starts with: the entity, and
// however many more the operation needs. A count that is wrong is answered by spelling
// the command out, since that is the answer.
func entityAnd(rest []string, want int, shows string, stderr io.Writer) (model.EntityRef, []string, int) {
	if len(rest) != want+1 {
		return model.EntityRef{}, nil, misuse(stderr, "that is not the shape of it: origins %s", shows)
	}
	ref, err := model.ParseEntityRef(rest[0])
	if err != nil {
		return model.EntityRef{}, nil, misuse(stderr, "%v", err)
	}
	return ref, rest[1:], exitOK
}

// bare reads a name where a name is wanted rather than a record.
//
// Only one kind is ever in play, and something has already said which: a rename
// changes what something is called and not what it is, a merge across two kinds is
// not two records of one thing, a split makes records of whatever it divides, and a
// hypothesis is the only thing there is to accept or reject. Writing the kind twice
// is the mistake worth answering, kind/slug being how a record is named everywhere
// else.
func bare(text string, kind model.Kind, stderr io.Writer) (model.Slug, int) {
	if name, again := strings.CutPrefix(text, string(kind)+"/"); again {
		return "", misuse(stderr, "the kind is said once: %s, not %s", name, text)
	}
	slug, err := model.ParseName(kind, text)
	if err != nil {
		return "", misuse(stderr, "%v", err)
	}
	return slug, exitOK
}

func setValue(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("set", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	ref, more, code := entityAnd(rest, 2, "set person/rossi-giovanni surname Rossi", stderr)
	if code != exitOK {
		return code
	}
	return c.carry(model.Set{Entity: ref, Path: more[0], Value: store.Typed(more[1])}, stdout, stderr)
}

func unsetValue(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("unset", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	ref, more, code := entityAnd(rest, 1, "unset person/rossi-giovanni events[0].place", stderr)
	if code != exitOK {
		return code
	}
	return c.carry(model.Unset{Entity: ref, Path: more[0]}, stdout, stderr)
}

func addValue(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("add", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	ref, more, code := entityAnd(rest, 2, "add person/rossi-giovanni events '{type: death, date: 14 MAR 1901}'", stderr)
	if code != exitOK {
		return code
	}
	return c.carry(model.Add{Entity: ref, Path: more[0], Value: store.Typed(more[1])}, stdout, stderr)
}

func removeValue(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("remove", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	ref, more, code := entityAnd(rest, 2, "remove family/rossi-antonio--verdi-lucia children rossi-giovanni", stderr)
	if code != exitOK {
		return code
	}
	return c.carry(model.Remove{Entity: ref, Path: more[0], Value: store.Typed(more[1])}, stdout, stderr)
}

// createEntity brings a new entity into the corpus with whatever fields are given.
//
// The fields are paths, so nothing is reachable at creation that is not reachable
// afterwards, and an entity created with nothing at all is perfectly legal — it is what
// a person known only to have existed looks like.
func createEntity(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("create", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) == 0 {
		return misuse(stderr, "that is not the shape of it: origins create "+
			"place/borgo-san-lorenzo name='Borgo San Lorenzo' type=comune")
	}
	ref, err := model.ParseEntityRef(rest[0])
	if err != nil {
		return misuse(stderr, "%v", err)
	}
	fields := map[string]any{}
	for _, pair := range rest[1:] {
		path, value, given := strings.Cut(pair, "=")
		if !given || path == "" {
			return misuse(stderr, "%q is not a field: a field is written path=value", pair)
		}
		fields[path] = store.Typed(value)
	}
	return c.carry(model.Create{Entity: ref, Value: fields}, stdout, stderr)
}

// renameEntity gives an entity a new slug, which moves its file and rewrites every
// reference to it.
func renameEntity(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("rename", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	ref, more, code := entityAnd(rest, 1, "rename person/rossi-giovanni rossi-giovanni-1834", stderr)
	if code != exitOK {
		return code
	}
	to, code := bare(more[0], ref.Kind, stderr)
	if code != exitOK {
		return code
	}
	return c.carry(model.Rename{Entity: ref, To: to}, stdout, stderr)
}

func deleteEntity(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("delete", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	ref, _, code := entityAnd(rest, 0, "delete hypothesis/annunziata-born-1844", stderr)
	if code != exitOK {
		return code
	}
	return c.carry(model.Delete{Entity: ref}, stdout, stderr)
}

// mergeEntity decides that two records describe one thing, and leaves the corpus
// holding the second of them.
//
// It is spelled like a rename because it is one: origins merge person/x y does what
// origins rename person/x y would do if y were not a name the corpus already had. The
// record that goes is named first, as the record that changes is in every one of these
// commands, and what was attached to it goes with it.
func mergeEntity(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("merge", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	ref, more, code := entityAnd(rest, 1, "merge person/rossi-giovanni-1902 rossi-giovanni", stderr)
	if code != exitOK {
		return code
	}
	into, code := bare(more[0], ref.Kind, stderr)
	if code != exitOK {
		return code
	}
	return c.carry(model.Merge{From: ref, Into: model.EntityRef{Kind: ref.Kind, Slug: into}},
		stdout, stderr)
}

// splitEntity makes one record into several, for when it turns out to describe more
// than one person.
//
// The order of the names is the whole of what the command decides: the first keeps
// everything the record had, references included, and the rest arrive holding nothing.
// Moving the facts across is what the four field operations are for, and it is the
// judgement that made the split worth proposing in the first place.
func splitEntity(args []string, stdout, stderr io.Writer) int {
	c, rest, code := changeFlags("split", args, stdout, stderr)
	if code != exitOK {
		return code
	}
	if len(rest) < 3 {
		return misuse(stderr, "that is not the shape of it: origins split "+
			"person/rossi-giovanni rossi-giovanni-1834 rossi-giovanni-1867")
	}
	ref, err := model.ParseEntityRef(rest[0])
	if err != nil {
		return misuse(stderr, "%v", err)
	}
	into := make([]model.EntityRef, 0, len(rest)-1)
	for _, name := range rest[1:] {
		slug, code := bare(name, ref.Kind, stderr)
		if code != exitOK {
			return code
		}
		into = append(into, model.EntityRef{Kind: ref.Kind, Slug: slug})
	}
	return c.carry(model.Split{Entity: ref, Into: into}, stdout, stderr)
}
