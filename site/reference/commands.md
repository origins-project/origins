<!-- Written by go test ./cmd/origins -run TestTheSitesGeneratedPages -update from cmd/origins/help.go. -->

# Commands

This page shows the help that Origins prints for each command. You can see the
same text in a terminal with `origins help <command>` or `origins <command> -h`.

## origins help

```text
Origins keeps a family tree as text files in a git repository.

Usage: origins <command> [arguments]

  init         start a new corpus
  load         read a corpus and count its records
  check        look for contradictions in a corpus
  fmt          rewrite a corpus in canonical form
  import       make a corpus from a GEDCOM or Gramps file

  set          put a value at a path in a record
  unset        remove the value at a path
  add          add an entry to a list
  remove       take an entry off a list
  create       add a new record
  rename       give a record a new slug
  delete       remove a record
  merge        make two records into one
  split        make one record into several

  ancestors    list a person's ancestors
  descendants  list a person's descendants
  kinship      say how two people are related
  todo         list the research still to do
  index        list every record and how they are linked

  log          list the changes Origins recorded
  whence       list the changes that cite a source
  blame        say when each fact of a record was recorded
  why          follow one fact back to its argument
  hypothesis   propose, accept or reject a hypothesis
  undo         take back the last change
  snapshot     name the current state, or list the names
  adopt        record changes made to the files by hand
  recover      deal with commits Origins did not write

  remote       add, remove or list remotes
  fetch        bring in what a remote holds
  push         send the branch and its snapshots to a remote
  pull         take in what another clone recorded
  clone        copy a corpus somebody shared

  browse       open the corpus in a browser
  mcp          serve the corpus to an AI agent
  export       write the corpus as GEDCOM, pages or a chart
  schema       print the JSON Schema of a record
  version      print the versions of origins and git
  help         list the commands, or explain one

An entity is written kind/slug, such as person/rossi-bartolomeo. Most commands
read the corpus in the working directory unless you give another with -C or as
the last argument. Run origins help <command> to see how to use a command.
```

## origins init

Start a new corpus.

```text
usage: origins init [flags] [dir]

init starts a corpus in dir, or in the working directory if you don't give
one. It writes origins.yaml and records it as the first commit. If the
directory is not inside a git repository, init makes one there.

init records only origins.yaml. If the directory already has files in it,
init leaves them alone, and you can record them afterwards with origins adopt.

Example:
  origins init --name "Rossi family" rossi

Flags:
  --json       report as JSON on stdout
  --name name  the project's name, in place of the directory's
```

## origins load

Read a corpus and count its records.

```text
usage: origins load [flags] [dir]

load reads a corpus and prints its name, its description and how many
records of each kind it holds. If some files cannot be read, load lists the
problems and exits 1.

Example:
  origins load rossi

Flags:
  --json  report as JSON on stdout
```

## origins check

Look for contradictions in a corpus.

```text
usage: origins check [flags] [dir]

check reads a corpus and lists the contradictions it finds, as errors and
warnings. It exits 1 if it finds an error and 0 if it finds only warnings, so
you can use it with git bisect run.

With --with, check answers as though the hypothesis you name were true. origins
applies the hypothesis's edits at the commit the hypothesis is pinned to, and
brings them forward to the current corpus with a three-way merge. If that
commit is not in the history, Origins applies the edits to the corpus as it is
now, and check tells you which of the two it did. If somebody has since
changed what the hypothesis changes, check reports a conflict and exits 1.
Give --with more than once to ask whether several hypotheses can be true
together.

Example:
  origins check --with louisa-born-1844

Flags:
  --errors           leave out warnings
  --json             report as JSON on stdout
  --with hypothesis  answer as though this hypothesis were true
```

## origins fmt

Rewrite a corpus in canonical form.

```text
usage: origins fmt [flags] [dir]

fmt rewrites the files of a corpus in the form Origins writes them, and
prints the files it changed. You can use it after editing files by hand. fmt
does not rewrite a file it cannot read.

With --list, fmt writes nothing. It prints the files it would change and exits
1 if there are any, so you can run it from a git hook or a build.

Example:
  origins fmt --list

Flags:
  --json  report as JSON on stdout
  --list  report what would change without writing
```

## origins import

Make a corpus from a GEDCOM or Gramps file.

```text
usage: origins import gedcom [flags] <file>
       origins import gramps [flags] <file>

import reads a GEDCOM file, or a Gramps file or package, and writes it as a
new corpus in one commit. If the new corpus is not inside a git repository,
import makes one. Origins keeps anything it has no field for in a fenced block
in the record, so the import does not lose any data.

When two records would get the same slug, Origins adds something to tell them
apart. With --interactive, it asks you to choose the slug instead.

Run origins check on the new corpus after the import. To see the flags of
each format, run origins import gedcom -h or origins import gramps -h.

Example:
  origins import gedcom --into rossi rossi.ged
```

### origins import gedcom

```text
Flags:
  --interactive     choose the slug whenever two entities would share one
  --into directory  the directory to create the corpus in
  --json            report as JSON on stdout
  --name name       the project's name
```

### origins import gramps

```text
Flags:
  --interactive      choose the slug whenever two entities would share one
  --into directory   the directory to create the corpus in
  --json             report as JSON on stdout
  --media directory  another directory the pictures may be in, looked in after the ones the file names
  --name name        the project's name
```

## origins set

Put a value at a path in a record.

```text
usage: origins set [flags] <entity> <path> <value>

set puts a value at a path in a record, and Origins records the change as
one commit. You write an entity as kind/slug, such as person/rossi-bartolomeo,
and a path the way the record's fields spell it, such as name.surname,
events[0].date or citations.1103-17.text.

A field that holds text takes the value as you typed it. Any other field reads
it as YAML, so you can give it a mapping or a list.

You write the manifest, origins.yaml, as project, with the paths under its
project: key. You write a tag as tag/<word>.

Example:
  origins set person/rossi-bartolomeo name.surname Rossi

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins unset

Remove the value at a path.

```text
usage: origins unset [flags] <entity> <path>

unset removes the value at a path in a record. origins help set says how to
write the entity and the path.

Example:
  origins unset person/rossi-bartolomeo events[0].place

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins add

Add an entry to a list.

```text
usage: origins add [flags] <entity> <list> <entry>

add puts one more entry at the end of a list in a record. origins help set
says how to write the entity, the path and the value.

Example:
  origins add person/rossi-bartolomeo events '{type: burial, date: 1799}'

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins remove

Take an entry off a list.

```text
usage: origins remove [flags] <entity> <list> <entry>

remove takes an entry off a list in a record.

Example:
  origins remove family/lang-georg--rossi-caterina children lang-joseph

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins create

Add a new record.

```text
usage: origins create [flags] <entity> [path=value]...

create adds a new record to the corpus, with the fields you give as
path=value. You can also create a record with no fields at all, for a person
you only know existed.

Example:
  origins create place/borgo-san-lorenzo name='Borgo San Lorenzo' type=comune

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins rename

Give a record a new slug.

```text
usage: origins rename [flags] <entity> <slug>

rename gives a record a new slug. Origins moves the record's directory and
rewrites every reference to it.

To rename a tag, write it as tag/<word>. Origins rewrites the word on every
record that carries it and moves its style. If you rename a tag to a word that
is already in use, the two tags become one.

Example:
  origins rename person/rossi-bartolomeo rossi-bartolomeo-1751

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins delete

Remove a record.

```text
usage: origins delete [flags] <entity>

delete removes a record from the corpus.

Example:
  origins delete hypothesis/unrecorded-child-1845

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins merge

Make two records into one.

```text
usage: origins merge [flags] <entity> <slug>

merge is for two records that describe the same thing. You name the record
to remove first and the record to keep second. Origins moves what was
attached to the first record to the second, and rewrites every reference to
the first so that it points to the second.

Example:
  origins merge person/rossi-bartolomeo-1751 rossi-bartolomeo

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins split

Make one record into several.

```text
usage: origins split [flags] <entity> <slug> <slug>...

split is for a record that turns out to describe more than one person. The
first slug you give keeps everything the record had, including the
references to it. The other records start empty, and you move facts to them
with set, unset, add and remove.

Example:
  origins split person/rossi-bartolomeo rossi-bartolomeo-1751 rossi-bartolomeo-1775

Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

## origins ancestors

List a person's ancestors.

```text
usage: origins ancestors [flags] <person>

ancestors lists a person's ancestors one generation at a time, the parents
first. --depth stops the list after that many generations.

ancestors reads the corpus and not its history. With --with, it answers as
though the hypothesis were true, as origins check does.

Example:
  origins ancestors --depth 3 verdier-jean-baptiste

Flags:
  -C corpus          the corpus to ask (default .)
  --depth n          stop after n generations
  --json             report as JSON on stdout
  --with hypothesis  answer as though this hypothesis were true
```

## origins descendants

List a person's descendants.

```text
usage: origins descendants [flags] <person>

descendants lists a person's descendants one generation at a time, the
children first. --depth stops the list after that many generations.

descendants reads the corpus and not its history. With --with, it answers as
though the hypothesis were true, as origins check does.

Example:
  origins descendants rossi-bartolomeo

Flags:
  -C corpus          the corpus to ask (default .)
  --depth n          stop after n generations
  --json             report as JSON on stdout
  --with hypothesis  answer as though this hypothesis were true
```

## origins kinship

Say how two people are related.

```text
usage: origins kinship [flags] <person> <person>

kinship says what the first person is to the second, and through which
people they are related.

Example:
  origins kinship rossi-bartolomeo verdier-jean-baptiste

Flags:
  -C corpus          the corpus to ask (default .)
  --json             report as JSON on stdout
  --with hypothesis  answer as though this hypothesis were true
```

## origins todo

List the research still to do.

```text
usage: origins todo [flags]

todo lists the research that is still to do: events that no citation
supports, scans that no citation transcribes, open hypotheses, and people
born more than 110 years ago with no death recorded. None of these is an
error, so todo exits 0 however long the list is.

Example:
  origins todo

Flags:
  -C corpus          the corpus to ask (default .)
  --json             report as JSON on stdout
  --with hypothesis  answer as though this hypothesis were true
```

## origins index

List every record and how they are linked.

```text
usage: origins index [flags]

index lists every record with its name and the references that link it to
other records, without the prose. Without --json, it prints only how many
records of each kind there are. The pages draw their lists and charts from
its JSON.

Example:
  origins index --json

Flags:
  -C corpus          the corpus to ask (default .)
  --json             report as JSON on stdout
  --with hypothesis  answer as though this hypothesis were true
```

## origins log

List the changes Origins recorded.

```text
usage: origins log [flags] [<entity>]

log lists the changes Origins has recorded in the corpus, the newest first,
with the records each change touched. If you name an entity, log lists only
the changes to it.

Example:
  origins log -n 5 person/rossi-bartolomeo

Flags:
  -C corpus  the corpus to read the history of (default .)
  --json     report as JSON on stdout
  -n n       the newest n entries
```

## origins whence

List the changes that cite a source.

```text
usage: origins whence [flags] <source>

whence lists every change that cites the source you name, so you can see
each conclusion you drew from it.

Example:
  origins whence census-1861-england

Flags:
  -C corpus  the corpus to read the history of (default .)
  --json     report as JSON on stdout
  -n n       the newest n entries
```

## origins blame

Say when each fact of a record was recorded.

```text
usage: origins blame [flags] <entity> [<field>]

blame lists the facts of a record, each with the commit that recorded it.
Name a field to see only that field.

Example:
  origins blame person/rossi-bartolomeo events

Flags:
  -C corpus  the corpus the entity is in (default .)
  --json     report as JSON on stdout
```

## origins why

Follow one fact back to its argument.

```text
usage: origins why [flags] <entity> <path>

why shows what a record says at one path, the commit that recorded it, and
the hypothesis that commit accepted, if there was one, with its evidence and
its argument.

Example:
  origins why person/rossi-bartolomeo events[0].date

Flags:
  -C corpus  the corpus the entity is in (default .)
  --json     report as JSON on stdout
```

## origins hypothesis

Propose, accept or reject a hypothesis.

```text
usage: origins hypothesis propose [flags] <name>
       origins hypothesis accept [flags] <name>
       origins hypothesis reject [flags] <name>
       origins hypothesis status [flags] [<name>]

A hypothesis is a set of edits to the corpus that you have not accepted yet,
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
accept -h, for example.

Example:
  origins hypothesis accept unrecorded-child-1845
```

### origins hypothesis propose

```text
Flags:
  -C corpus           the corpus to change (default .)
  --dry-run           say what would change without changing it
  --from file         a hypothesis's file stating it whole, in place of the edited corpus
  --json              report as JSON on stdout
  -m subject          the subject line, in place of the one origins writes
  --proposed-by name  the name of whoever proposed it, where it was not the researcher
  --source citation   a citation the change rests on, repeatable
```

### origins hypothesis accept

```text
Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

### origins hypothesis reject

```text
Flags:
  -C corpus          the corpus to change (default .)
  --dry-run          say what would change without changing it
  --json             report as JSON on stdout
  -m subject         the subject line, in place of the one origins writes
  --source citation  a citation the change rests on, repeatable
```

### origins hypothesis status

```text
Flags:
  -C corpus  the corpus whose hypotheses to look at (default .)
  --json     report as JSON on stdout
```

## origins undo

Take back the last change.

```text
usage: origins undo [flags]

undo takes back the last change that Origins recorded, in a new commit. It
doesn't run check afterwards.

Example:
  origins undo --dry-run

Flags:
  -C corpus   the corpus to take a change back in (default .)
  --dry-run   say what would be taken back without taking it
  --json      report as JSON on stdout
  -m subject  the subject line, in place of the one origins writes
```

## origins snapshot

Name the current state, or list the names.

```text
usage: origins snapshot [flags] [<name>]

snapshot gives a name to the current state of the corpus, as a git tag, so
that you can refer to it later. Without a name, it lists the snapshots there
are. origins push sends the snapshots with the branch.

Example:
  origins snapshot -m 'the tree as sent to zia Maria' sent-to-zia-maria

Flags:
  -C corpus   the corpus to name a state of (default .)
  --json      report as JSON on stdout
  -m message  a message saying what the state is named for
```

## origins adopt

Record changes made to the files by hand.

```text
usage: origins adopt [flags]

adopt records the changes you made to the files by hand as one commit.
Origins rewrites each changed file in canonical form first, and refuses the
changes if they contradict the rest of the corpus.

Example:
  origins adopt -m 'transcribe the 1861 census'

Flags:
  -C corpus   the corpus to record (default .)
  --dry-run   say what would be recorded without recording it
  --json      report as JSON on stdout
  -m subject  the subject line, in place of the one origins writes
```

## origins recover

Deal with commits Origins did not write.

```text
usage: origins recover [flags]

Origins stops every command that reads or writes the history if it finds a
commit it did not write, until you run origins recover.

Without a flag, recover says what it found and does nothing else. Then you
choose what to do with those commits. --adopt keeps them and records that
Origins has read them. --rewrite replaces them with commits that describe
their changes. --reset removes them and leaves their changes in the working
tree.

Example:
  origins recover --adopt

Flags:
  -C corpus  the corpus whose history to look at (default .)
  --adopt    keep the commits as they are, and record having read them
  --json     report as JSON on stdout
  --reset    take them off, leaving what they said in the working tree
  --rewrite  replace them with commits that say what they are
```

## origins remote

Add, remove or list remotes.

```text
usage: origins remote add [flags] <name> <url>
       origins remote remove [flags] <name>
       origins remote list [flags]

remote manages the git remotes that you share the corpus through. list
prints each remote's name and the URL it fetches from. git handles the
authentication, with an SSH key or a credential helper, and Origins stores
nothing of it.

Example:
  origins remote add origin git@example.org:rossi.git
```

### origins remote add

```text
Flags:
  -C corpus  the corpus whose remotes these are (default .)
  --json     report as JSON on stdout
```

### origins remote remove

```text
Flags:
  -C corpus  the corpus whose remotes these are (default .)
  --json     report as JSON on stdout
```

### origins remote list

```text
Flags:
  -C corpus  the corpus whose remotes these are (default .)
  --json     report as JSON on stdout
```

## origins fetch

Bring in what a remote holds.

```text
usage: origins fetch [flags] [<remote>]

fetch brings in what a remote holds without moving any branch. Then it says
how many commits the branch's upstream has that the branch does not, and the
other way round. If you don't name a remote, fetch fetches from all of them.

Example:
  origins fetch

Flags:
  -C corpus  the corpus to fetch into (default .)
  --json     report as JSON on stdout
```

## origins push

Send the branch and its snapshots to a remote.

```text
usage: origins push [flags] [<remote>]

push sends the current branch and every snapshot on it. It sends them to the
remote you name, to the remote the branch follows, or to the only remote
there is. After the first push, the branch follows the remote it was pushed
to.

push refuses to send commits that Origins did not write. If the remote has
commits that you don't have, run origins pull first.

Example:
  origins push origin

Flags:
  -C corpus  the corpus to push (default .)
  --json     report as JSON on stdout
```

## origins pull

Take in what another clone recorded.

```text
usage: origins pull [flags] [<remote>]

pull takes in the changes that another clone recorded and pushed. It chooses
the remote in the same way as push. It does not start if anything in the
repository is staged or modified.

pull only takes in commits that Origins wrote. If the remote has a commit
that Origins did not write, pull refuses before it takes anything in.
Whoever made that commit can run origins recover --adopt and push again.

If only the remote has moved on, pull fast-forwards the branch. If both
clones have new commits, git merges them. pull refuses the merge if both
sides changed the same lines, or if the merged corpus contradicts itself.
Otherwise it records the merge as one commit. When pull refuses, it leaves
the corpus as it was.

Example:
  origins pull

Flags:
  -C corpus  the corpus to pull into (default .)
  --json     report as JSON on stdout
```

## origins clone

Copy a corpus somebody shared.

```text
usage: origins clone [flags] <url> [<dir>]

clone copies a corpus from a git URL and checks out every file exactly as it
was committed, whatever git is configured to do with line endings. Then it
checks the copy: whether it has a corpus at its root, what origins check
finds in it, and whether Origins can account for its history.

If any of these checks fails, clone keeps the copy, says what it found and
what to run, and exits 1.

Example:
  origins clone git@example.org:rossi.git rossi

Flags:
  --json  report as JSON on stdout
```

## origins browse

Open the corpus in a browser.

```text
usage: origins browse [flags] [dir]

browse serves the corpus to a browser at 127.0.0.1:8100 and opens it there.
In the browser you can read the corpus and change it.

With --listen, browse serves on the address you give and doesn't open a
browser. On an address other than the loopback, browse asks every visitor for
a passphrase. It reads the passphrase from ORIGINS_PASSPHRASE or from the
file --passphrase-file names, and doesn't start without one.

With --with, --hide-private, --hide-living or --public, browse serves the
corpus read-only. If some files of the corpus cannot be read, browse serves
the rest read-only and shows the problems.

Example:
  origins browse rossi

Flags:
  --hide-living           serve the corpus without whoever is taken as living, read-only
  --hide-private          serve the corpus without what is private, read-only
  --json                  report as JSON on stdout
  --listen address        the address to serve on, 127.0.0.1:8100 where none is given
  --no-open               do not open a browser
  --passphrase-file file  the file holding the passphrase asked for off the loopback, in place of ORIGINS_PASSPHRASE
  --public                the same as --hide-private and --hide-living together
  --with hypothesis       serve the corpus as though this hypothesis were true, read-only
```

## origins mcp

Serve the corpus to an AI agent.

```text
usage: origins mcp [flags]

mcp serves the corpus to an AI agent over the Model Context Protocol on
standard input and output. The agent's client starts it.

The agent can ask index, read, check, todo, ancestors, descendants, kinship,
log, whence, blame, why, hypothesis_status and schema, and each answers with
the JSON that the command's --json gives. The agent can also propose a
hypothesis, in the same way as origins hypothesis propose --from. It cannot
change the corpus in any other way. Origins records the name you give with
--as as the one who proposed the hypothesis. Without --as, it records the
name the client gives itself.

Example:
  origins mcp -C rossi --as assistant

Flags:
  -C corpus  the corpus to serve (default .)
  --as name  the name a hypothesis is proposed by, in place of the client's own
```

## origins export

Write the corpus as GEDCOM, pages or a chart.

```text
usage: origins export gedcom [flags] --out <file.gdz>
       origins export html [flags] --out <dir>
       origins export chart [flags] --out <file> <person>

export writes the corpus for somebody who doesn't use origins.

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
export, run origins export chart -h, for example.

Example:
  origins export chart --shape descendant-tree --out rossi.pdf rossi-bartolomeo
```

### origins export gedcom

```text
Flags:
  -C corpus          the corpus to ask (default .)
  --hide-living      leave out whoever is taken as living
  --hide-private     leave out what is private
  --json             report as JSON on stdout
  --out file         the GEDZIP file to write
  --public           the same as --hide-private and --hide-living together
  --with hypothesis  answer as though this hypothesis were true
```

### origins export html

```text
Flags:
  -C corpus          the corpus to ask (default .)
  --hide-living      leave out whoever is taken as living
  --hide-private     leave out what is private
  --json             report as JSON on stdout
  --notice text      a text shown below the bar on every page
  --out directory    the directory to write the pages into, not there or empty
  --public           the same as --hide-private and --hide-living together
  --with hypothesis  answer as though this hypothesis were true
```

### origins export chart

```text
Flags:
  -C corpus          the corpus to ask (default .)
  --distance steps   how many steps the chart reaches (default 5)
  --hide-living      leave out whoever is taken as living
  --hide-private     leave out what is private
  --json             report as JSON on stdout
  --out file         the file to draw it to, as svg, pdf or png by its extension
  --public           the same as --hide-private and --hide-living together
  --shape shape      the chart's shape (default ancestors)
  --to person        the person a path runs to
  --with hypothesis  answer as though this hypothesis were true
```

## origins schema

Print the JSON Schema of a record.

```text
usage: origins schema [<kind>]

schema prints the JSON Schema of one kind of record: person, family, place,
source, repository, note, hypothesis, or project for origins.yaml. Without a
kind, it prints all of them keyed by kind, with common holding what they
share. It doesn't need a corpus.

Example:
  origins schema person
```

## origins version

Print the versions of origins and git.

```text
usage: origins version [--json]

version prints the version of origins, the commit it was built from, the
version of Go that built it, and which git it runs and where that git is. A
build that isn't a release says dev. Origins needs git for every command that
reads or writes the history, and version says so if it can't find one.

Example:
  origins version

Flags:
  --json  report as JSON on stdout
```

## origins help

List the commands, or explain one.

```text
usage: origins help [<command>]

help lists the commands. If you name a command, help prints its usage, an
example and its flags, as origins <command> -h does.

Example:
  origins help check
```
