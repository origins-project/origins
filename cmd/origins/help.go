package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// A help is what origins help says about one command.
type help struct {
	name        string
	summary     string
	usage       []string
	description string
	example     string
}

// helps holds every command's help, in the order origins help lists them, with a blank
// line between each group.
var helps = [][]help{
	{
		{
			name:    "init",
			summary: "start a new corpus",
			usage:   []string{"origins init [flags] [dir]"},
			description: `init starts a corpus in dir, or in the working directory if you don't give
one. It writes origins.yaml and records it as the first commit. If the
directory is not inside a git repository, init makes one there.

init records only origins.yaml. If the directory already has files in it,
init leaves them alone, and you can record them afterwards with origins adopt.`,
			example: `origins init --name "Rossi family" rossi`,
		},
		{
			name:    "load",
			summary: "read a corpus and count its records",
			usage:   []string{"origins load [flags] [dir]"},
			description: `load reads a corpus and prints its name, its description and how many
records of each kind it holds. If some files cannot be read, load lists the
problems and exits 1.`,
			example: "origins load rossi",
		},
		{
			name:    "check",
			summary: "look for contradictions in a corpus",
			usage:   []string{"origins check [flags] [dir]"},
			description: `check reads a corpus and lists the contradictions it finds, as errors and
warnings. It exits 1 if it finds an error and 0 if it finds only warnings, so
you can use it with git bisect run.

With --with, check answers as though the hypothesis you name were true. origins
applies the hypothesis's edits at the commit the hypothesis is pinned to, and
brings them forward to the current corpus with a three-way merge. If that
commit is not in the history, Origins applies the edits to the corpus as it is
now, and check tells you which of the two it did. If somebody has since
changed what the hypothesis changes, check reports a conflict and exits 1.
Give --with more than once to ask whether several hypotheses can be true
together.`,
			example: "origins check --with louisa-born-1844",
		},
		{
			name:    "fmt",
			summary: "rewrite a corpus in canonical form",
			usage:   []string{"origins fmt [flags] [dir]"},
			description: `fmt rewrites the files of a corpus in the form Origins writes them, and
prints the files it changed. You can use it after editing files by hand. fmt
does not rewrite a file it cannot read.

With --list, fmt writes nothing. It prints the files it would change and exits
1 if there are any, so you can run it from a git hook or a build.`,
			example: "origins fmt --list",
		},
		{
			name:    "import",
			summary: "make a corpus from a GEDCOM or Gramps file",
			usage: []string{
				"origins import gedcom [flags] <file>",
				"origins import gramps [flags] <file>",
			},
			description: `import reads a GEDCOM file, or a Gramps file or package, and writes it as a
new corpus in one commit. If the new corpus is not inside a git repository,
import makes one. Origins keeps anything it has no field for in a fenced block
in the record, so the import does not lose any data.

When two records would get the same slug, Origins adds something to tell them
apart. With --interactive, it asks you to choose the slug instead.

Run origins check on the new corpus after the import. To see the flags of
each format, run origins import gedcom -h or origins import gramps -h.`,
			example: "origins import gedcom --into rossi rossi.ged",
		},
	},
	{
		{
			name:    "set",
			summary: "put a value at a path in a record",
			usage:   []string{"origins set [flags] <entity> <path> <value>"},
			description: `set puts a value at a path in a record, and Origins records the change as
one commit. You write an entity as kind/slug, such as person/rossi-bartolomeo,
and a path the way the record's fields spell it, such as name.surname,
events[0].date or citations.1103-17.text.

A field that holds text takes the value as you typed it. Any other field reads
it as YAML, so you can give it a mapping or a list.

You write the manifest, origins.yaml, as project, with the paths under its
project: key. You write a tag as tag/<word>.`,
			example: "origins set person/rossi-bartolomeo name.surname Rossi",
		},
		{
			name:    "unset",
			summary: "remove the value at a path",
			usage:   []string{"origins unset [flags] <entity> <path>"},
			description: `unset removes the value at a path in a record. origins help set says how to
write the entity and the path.`,
			example: "origins unset person/rossi-bartolomeo events[0].place",
		},
		{
			name:    "add",
			summary: "add an entry to a list",
			usage:   []string{"origins add [flags] <entity> <list> <entry>"},
			description: `add puts one more entry at the end of a list in a record. origins help set
says how to write the entity, the path and the value.`,
			example: "origins add person/rossi-bartolomeo events '{type: burial, date: 1799}'",
		},
		{
			name:        "remove",
			summary:     "take an entry off a list",
			usage:       []string{"origins remove [flags] <entity> <list> <entry>"},
			description: `remove takes an entry off a list in a record.`,
			example:     "origins remove family/lang-georg--rossi-caterina children lang-joseph",
		},
		{
			name:    "create",
			summary: "add a new record",
			usage:   []string{"origins create [flags] <entity> [path=value]..."},
			description: `create adds a new record to the corpus, with the fields you give as
path=value. You can also create a record with no fields at all, for a person
you only know existed.`,
			example: "origins create place/borgo-san-lorenzo name='Borgo San Lorenzo' type=comune",
		},
		{
			name:    "rename",
			summary: "give a record a new slug",
			usage:   []string{"origins rename [flags] <entity> <slug>"},
			description: `rename gives a record a new slug. Origins moves the record's directory and
rewrites every reference to it.

To rename a tag, write it as tag/<word>. Origins rewrites the word on every
record that carries it and moves its style. If you rename a tag to a word that
is already in use, the two tags become one.`,
			example: "origins rename person/rossi-bartolomeo rossi-bartolomeo-1751",
		},
		{
			name:        "delete",
			summary:     "remove a record",
			usage:       []string{"origins delete [flags] <entity>"},
			description: `delete removes a record from the corpus.`,
			example:     "origins delete hypothesis/unrecorded-child-1845",
		},
		{
			name:    "merge",
			summary: "make two records into one",
			usage:   []string{"origins merge [flags] <entity> <slug>"},
			description: `merge is for two records that describe the same thing. You name the record
to remove first and the record to keep second. Origins moves what was
attached to the first record to the second, and rewrites every reference to
the first so that it points to the second.`,
			example: "origins merge person/rossi-bartolomeo-1751 rossi-bartolomeo",
		},
		{
			name:    "split",
			summary: "make one record into several",
			usage:   []string{"origins split [flags] <entity> <slug> <slug>..."},
			description: `split is for a record that turns out to describe more than one person. The
first slug you give keeps everything the record had, including the
references to it. The other records start empty, and you move facts to them
with set, unset, add and remove.`,
			example: "origins split person/rossi-bartolomeo rossi-bartolomeo-1751 rossi-bartolomeo-1775",
		},
	},
	{
		{
			name:    "ancestors",
			summary: "list a person's ancestors",
			usage:   []string{"origins ancestors [flags] <person>"},
			description: `ancestors lists a person's ancestors one generation at a time, the parents
first. --depth stops the list after that many generations.

ancestors reads the corpus and not its history. With --with, it answers as
though the hypothesis were true, as origins check does.`,
			example: "origins ancestors --depth 3 verdier-jean-baptiste",
		},
		{
			name:    "descendants",
			summary: "list a person's descendants",
			usage:   []string{"origins descendants [flags] <person>"},
			description: `descendants lists a person's descendants one generation at a time, the
children first. --depth stops the list after that many generations.

descendants reads the corpus and not its history. With --with, it answers as
though the hypothesis were true, as origins check does.`,
			example: "origins descendants rossi-bartolomeo",
		},
		{
			name:    "kinship",
			summary: "say how two people are related",
			usage:   []string{"origins kinship [flags] <person> <person>"},
			description: `kinship says what the first person is to the second, and through which
people they are related.`,
			example: "origins kinship rossi-bartolomeo verdier-jean-baptiste",
		},
		{
			name:    "todo",
			summary: "list the research still to do",
			usage:   []string{"origins todo [flags]"},
			description: `todo lists the research that is still to do: events that no citation
supports, scans that no citation transcribes, open hypotheses, and people
born more than 110 years ago with no death recorded. None of these is an
error, so todo exits 0 however long the list is.`,
			example: "origins todo",
		},
		{
			name:    "index",
			summary: "list every record and how they are linked",
			usage:   []string{"origins index [flags]"},
			description: `index lists every record with its name and the references that link it to
other records, without the prose. Without --json, it prints only how many
records of each kind there are. The pages draw their lists and charts from
its JSON.`,
			example: "origins index --json",
		},
	},
	{
		{
			name:    "log",
			summary: "list the changes Origins recorded",
			usage:   []string{"origins log [flags] [<entity>]"},
			description: `log lists the changes Origins has recorded in the corpus, the newest first,
with the records each change touched. If you name an entity, log lists only
the changes to it.`,
			example: "origins log -n 5 person/rossi-bartolomeo",
		},
		{
			name:    "whence",
			summary: "list the changes that cite a source",
			usage:   []string{"origins whence [flags] <source>"},
			description: `whence lists every change that cites the source you name, so you can see
each conclusion you drew from it.`,
			example: "origins whence census-1861-england",
		},
		{
			name:    "blame",
			summary: "say when each fact of a record was recorded",
			usage:   []string{"origins blame [flags] <entity> [<field>]"},
			description: `blame lists the facts of a record, each with the commit that recorded it.
Name a field to see only that field.`,
			example: "origins blame person/rossi-bartolomeo events",
		},
		{
			name:    "why",
			summary: "follow one fact back to its argument",
			usage:   []string{"origins why [flags] <entity> <path>"},
			description: `why shows what a record says at one path, the commit that recorded it, and
the hypothesis that commit accepted, if there was one, with its evidence and
its argument.`,
			example: "origins why person/rossi-bartolomeo events[0].date",
		},
		{
			name:    "hypothesis",
			summary: "propose, accept or reject a hypothesis",
			usage: []string{
				"origins hypothesis propose [flags] <name>",
				"origins hypothesis accept [flags] <name>",
				"origins hypothesis reject [flags] <name>",
				"origins hypothesis status [flags] [<name>]",
			},
			description: `A hypothesis is a set of edits to the corpus that you have not accepted yet,
with the evidence and the argument for it.

propose records one. Edit the files as though the hypothesis were true and
run origins hypothesis propose. Origins states how the files differ from the
last commit as edits, records the hypothesis pinned to that commit, and puts
the files back as they were. It refuses a change it cannot state in order,
such as an entry put in before the end of a list. It also refuses if a file
that is not a record has unrecorded changes, because putting the files back
would lose them. With --from, Origins records a hypothesis file that you
wrote yourself and changes no other file. --proposed-by names whoever
proposed the hypothesis, if it was not you.

accept applies the edits in the same way as check --with, and records the
result as one commit that names the hypothesis. The corpus must have no
unrecorded changes when you start, and accept refuses a hypothesis whose
commit is not in the history. reject changes no record. It only records that
you considered the hypothesis and decided against it.

status lists every hypothesis and applies each accepted one again. If that
changes nothing, the corpus still holds the conclusion. If it would change
something, status names the files that somebody has changed since and exits 1.

Each of the four has its own flags. To see them, run origins hypothesis
accept -h, for example.`,
			example: "origins hypothesis accept unrecorded-child-1845",
		},
		{
			name:    "undo",
			summary: "take back the last change",
			usage:   []string{"origins undo [flags]"},
			description: `undo takes back the last change that Origins recorded, in a new commit. It
doesn't run check afterwards.`,
			example: "origins undo --dry-run",
		},
		{
			name:    "snapshot",
			summary: "name the current state, or list the names",
			usage:   []string{"origins snapshot [flags] [<name>]"},
			description: `snapshot gives a name to the current state of the corpus, as a git tag, so
that you can refer to it later. Without a name, it lists the snapshots there
are. origins push sends the snapshots with the branch.`,
			example: "origins snapshot -m 'the tree as sent to zia Maria' sent-to-zia-maria",
		},
		{
			name:    "adopt",
			summary: "record changes made to the files by hand",
			usage:   []string{"origins adopt [flags]"},
			description: `adopt records the changes you made to the files by hand as one commit.
Origins rewrites each changed file in canonical form first, and refuses the
changes if they contradict the rest of the corpus.`,
			example: "origins adopt -m 'transcribe the 1861 census'",
		},
		{
			name:    "recover",
			summary: "deal with commits Origins did not write",
			usage:   []string{"origins recover [flags]"},
			description: `Origins stops every command that reads or writes the history if it finds a
commit it did not write, until you run origins recover.

Without a flag, recover says what it found and does nothing else. Then you
choose what to do with those commits. --adopt keeps them and records that
Origins has read them. --rewrite replaces them with commits that describe
their changes. --reset removes them and leaves their changes in the working
tree.`,
			example: "origins recover --adopt",
		},
	},
	{
		{
			name:    "remote",
			summary: "add, remove or list remotes",
			usage: []string{
				"origins remote add [flags] <name> <url>",
				"origins remote remove [flags] <name>",
				"origins remote list [flags]",
			},
			description: `remote manages the git remotes that you share the corpus through. list
prints each remote's name and the URL it fetches from. git handles the
authentication, with an SSH key or a credential helper, and Origins stores
nothing of it.`,
			example: "origins remote add origin git@example.org:rossi.git",
		},
		{
			name:    "fetch",
			summary: "bring in what a remote holds",
			usage:   []string{"origins fetch [flags] [<remote>]"},
			description: `fetch brings in what a remote holds without moving any branch. Then it says
how many commits the branch's upstream has that the branch does not, and the
other way round. If you don't name a remote, fetch fetches from all of them.`,
			example: "origins fetch",
		},
		{
			name:    "push",
			summary: "send the branch and its snapshots to a remote",
			usage:   []string{"origins push [flags] [<remote>]"},
			description: `push sends the current branch and every snapshot on it. It sends them to the
remote you name, to the remote the branch follows, or to the only remote
there is. After the first push, the branch follows the remote it was pushed
to.

push refuses to send commits that Origins did not write. If the remote has
commits that you don't have, run origins pull first.`,
			example: "origins push origin",
		},
		{
			name:    "pull",
			summary: "take in what another clone recorded",
			usage:   []string{"origins pull [flags] [<remote>]"},
			description: `pull takes in the changes that another clone recorded and pushed. It chooses
the remote in the same way as push. It does not start if anything in the
repository is staged or modified.

pull only takes in commits that Origins wrote. If the remote has a commit
that Origins did not write, pull refuses before it takes anything in.
Whoever made that commit can run origins recover --adopt and push again.

If only the remote has moved on, pull fast-forwards the branch. If both
clones have new commits, git merges them. pull refuses the merge if both
sides changed the same lines, or if the merged corpus contradicts itself.
Otherwise it records the merge as one commit. When pull refuses, it leaves
the corpus as it was.`,
			example: "origins pull",
		},
		{
			name:    "clone",
			summary: "copy a corpus somebody shared",
			usage:   []string{"origins clone [flags] <url> [<dir>]"},
			description: `clone copies a corpus from a git URL and checks out every file exactly as it
was committed, whatever git is configured to do with line endings. Then it
checks the copy: whether it has a corpus at its root, what origins check
finds in it, and whether Origins can account for its history.

If any of these checks fails, clone keeps the copy, says what it found and
what to run, and exits 1.`,
			example: "origins clone git@example.org:rossi.git rossi",
		},
	},
	{
		{
			name:    "browse",
			summary: "open the corpus in a browser",
			usage:   []string{"origins browse [flags] [dir]"},
			description: `browse serves the corpus to a browser at 127.0.0.1:8100 and opens it there.
In the browser you can read the corpus and change it.

With --listen, browse serves on the address you give and doesn't open a
browser. On an address other than the loopback, browse asks every visitor for
a passphrase. It reads the passphrase from ORIGINS_PASSPHRASE or from the
file --passphrase-file names, and doesn't start without one.

With --with, --hide-private, --hide-living or --public, browse serves the
corpus read-only. If some files of the corpus cannot be read, browse serves
the rest read-only and shows the problems.`,
			example: "origins browse rossi",
		},
		{
			name:    "mcp",
			summary: "serve the corpus to an AI agent",
			usage:   []string{"origins mcp [flags]"},
			description: `mcp serves the corpus to an AI agent over the Model Context Protocol on
standard input and output. The agent's client starts it.

The agent can ask index, read, check, todo, ancestors, descendants, kinship,
log, whence, blame, why, hypothesis_status and schema, and each answers with
the JSON that the command's --json gives. The agent can also propose a
hypothesis, in the same way as origins hypothesis propose --from. It cannot
change the corpus in any other way. Origins records the name you give with
--as as the one who proposed the hypothesis. Without --as, it records the
name the client gives itself.`,
			example: "origins mcp -C rossi --as assistant",
		},
		{
			name:    "export",
			summary: "write the corpus as GEDCOM, pages or a chart",
			usage: []string{
				"origins export gedcom [flags] --out <file.gdz>",
				"origins export html [flags] --out <dir>",
				"origins export chart [flags] --out <file> <person>",
			},
			description: `export writes the corpus for somebody who doesn't use origins.

export gedcom writes GEDCOM 7.0 in a GEDZIP file, with the pictures, and
reports what GEDCOM has no place for. export html writes the pages into a
directory that must be empty or not exist yet, and you can open them from
there without a server. Their Info page says when and from which commit you
made them, and --notice shows a line of text below the bar on every page.
export chart draws one chart of a person as SVG, PDF or PNG, according to
the file's extension. The shapes are ancestors,
descendants, both, relatives, ancestor-tree, descendant-tree, fan, path and
timeline. --to names the person a path chart runs to.

Each export takes --hide-private, --hide-living and --public to leave out
what is private or whoever is taken to be living, and --with to export the
corpus as though a hypothesis were true. To see the flags of one kind of
export, run origins export chart -h, for example.`,
			example: "origins export chart --shape descendant-tree --out rossi.pdf rossi-bartolomeo",
		},
		{
			name:    "schema",
			summary: "print the JSON Schema of a record",
			usage:   []string{"origins schema [<kind>]"},
			description: `schema prints the JSON Schema of one kind of record: person, family, place,
source, repository, note, hypothesis, or project for origins.yaml. Without a
kind, it prints all of them keyed by kind, with common holding what they
share. It doesn't need a corpus.`,
			example: "origins schema person",
		},
		{
			name:    "version",
			summary: "print the versions of origins and git",
			usage:   []string{"origins version [--json]"},
			description: `version prints the version of origins, the commit it was built from, the
version of Go that built it, and which git it runs and where that git is. A
build that isn't a release says dev. Origins needs git for every command that
reads or writes the history, and version says so if it can't find one.`,
			example: "origins version",
		},
		{
			name:    "help",
			summary: "list the commands, or explain one",
			usage:   []string{"origins help [<command>]"},
			description: `help lists the commands. If you name a command, help prints its usage, an
example and its flags, as origins <command> -h does.`,
			example: "origins help check",
		},
	},
}

// helpOf is the help of the command name, or nil where there is no such command.
func helpOf(name string) *help {
	for _, group := range helps {
		for i := range group {
			if group[i].name == name {
				return &group[i]
			}
		}
	}
	return nil
}

// asksHelp says whether arg is a request for help, as the flag package reads one.
func asksHelp(arg string) bool {
	switch arg {
	case "-h", "--h", "-help", "--help":
		return true
	}
	return false
}

// usage lists the commands with a line each.
func usage(w io.Writer) {
	fmt.Fprint(w, "Origins keeps a family tree as text files in a git repository.\n\n")
	fmt.Fprint(w, "Usage: origins <command> [arguments]\n")
	width := 0
	for _, group := range helps {
		for _, one := range group {
			width = max(width, len(one.name))
		}
	}
	for _, group := range helps {
		fmt.Fprintln(w)
		for _, one := range group {
			fmt.Fprintf(w, "  %-*s  %s\n", width, one.name, one.summary)
		}
	}
	fmt.Fprint(w, `
An entity is written kind/slug, such as person/rossi-bartolomeo. Most commands
read the corpus in the working directory unless you give another with -C or as
the last argument. Run origins help <command> to see how to use a command.
`)
}

// helpCommand lists the commands, or explains the one named, by asking it for -h so
// that the flags it lists are the ones it takes.
func helpCommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch {
	case len(args) == 0:
		usage(stdout)
		return exitOK
	case len(args) > 1:
		return misuse(stderr, "help explains one command at a time")
	case args[0] == "help" || asksHelp(args[0]):
		printHelp(stdout, helpOf("help"), nil)
		return exitOK
	case helpOf(args[0]) == nil:
		return misuse(stderr, "there is no command %q: origins help lists them", args[0])
	}
	return run([]string{args[0], "-h"}, stdin, stdout, stderr)
}

// printHelp writes a command's help, and the flags it takes where they are known.
func printHelp(w io.Writer, h *help, flags *flag.FlagSet) {
	for i, line := range h.usage {
		lead := "usage: "
		if i > 0 {
			lead = "       "
		}
		fmt.Fprintf(w, "%s%s\n", lead, line)
	}
	fmt.Fprintf(w, "\n%s\n\nExample:\n  %s\n", h.description, h.example)
	if flags == nil {
		return
	}
	var listed bool
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	flags.VisitAll(func(f *flag.Flag) {
		if !listed {
			fmt.Fprint(table, "\nFlags:\n")
			listed = true
		}
		dashes := "--"
		if len(f.Name) == 1 {
			dashes = "-"
		}
		name, said := flag.UnquoteUsage(f)
		if name != "" {
			name = " " + name
		}
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" && f.DefValue != "[]" {
			said += fmt.Sprintf(" (default %s)", f.DefValue)
		}
		fmt.Fprintf(table, "  %s%s%s\t%s\n", dashes, f.Name, name, said)
	})
	table.Flush()
}

// newFlags is the flag set of the command called name, whose -h parse answers.
func newFlags(name string, stderr io.Writer) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	return flags
}

// parse reads a command's flags. Asked for help, it writes the command's help and the
// flags it takes, and answers exitHelped. A flag it does not know is followed by the
// command's usage.
func parse(flags *flag.FlagSet, args []string, stdout io.Writer) int {
	err := flags.Parse(args)
	if err == nil {
		return exitOK
	}
	command, _, _ := strings.Cut(flags.Name(), " ")
	h := helpOf(command)
	if errors.Is(err, flag.ErrHelp) {
		printHelp(stdout, h, flags)
		return exitHelped
	}
	for _, line := range h.usage {
		fmt.Fprintf(flags.Output(), "usage: %s\n", line)
	}
	return exitMisuse
}
