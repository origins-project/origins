package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	out, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimRight(string(out), "\r\n")
}

func read(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func trailer(t *testing.T, root, key string) string {
	t.Helper()
	return strings.TrimSpace(git(t, root, "log", "-1", "--format=%(trailers:key="+key+",valueonly)"))
}

// theRossis is a corpus small enough to read in one look and complete enough to change:
// a person with an event, and the place the event happened in for the event to point at.
func theRossis() map[string]string {
	return map[string]string{
		"origins.yaml": sound,
		"people/rossi-giovanni/README.md": "---\nname:\n  given: Giovanni\n  surname: Rossi\nsex: M\n" +
			"events:\n  - type: birth\n    date: 1834\n    place: firenze\n---\n",
		"places/firenze/README.md": "---\nname: Firenze\ntype: city\n---\n",
	}
}

// twoGiovannis is one man recorded twice, which is the corpus a merge is for: a full
// record and a sparse one, with a photograph filed beside the sparse one and a
// portrait naming it. Whether the two are one man is the question origins cannot
// answer; carrying the photograph over once somebody has answered it is the part it
// can.
func twoGiovannis() map[string]string {
	files := theRossis()
	files["people/rossi-giovanni-1902/README.md"] = "---\nname:\n  given: Giovanni\n  surname: Rossi\n" +
		"portrait: photos/passaporto.jpg\nevents:\n  - type: death\n    date: 1902\n---\n"
	files["people/rossi-giovanni-1902/photos/passaporto.jpg"] = "a scan of a photograph"
	return files
}

// inARepo is a corpus versioned and committed once, as origins' own work.
//
// The first commit says origins made it, because every command stops on a history
// origins cannot account for and a corpus somebody committed by hand is one of those.
// byHand is the same corpus without that, for the tests about being stopped.
func inARepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := neverCommitted(t, files)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", strings.Join([]string{
		"The corpus as it stood", "", "Origins-Tool: origins " + api.Version, "Origins-Op: manual.adopt",
	}, "\n"))
	git(t, root, "update-ref", "refs/origins/head", "HEAD")
	return root
}

// byHand is a corpus in git that origins has had nothing to do with, which is where
// every researcher with a corpus already in git starts.
func byHand(t *testing.T, files map[string]string) string {
	t.Helper()
	root := neverCommitted(t, files)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "The corpus as it stood")
	return root
}

// neverCommitted is a corpus in a repository with no commits in it at all, which is
// the other thing a fresh start looks like.
func neverCommitted(t *testing.T, files map[string]string) string {
	t.Helper()
	root := corpusDir(t, files)
	git(t, root, "init", "-q")
	git(t, root, "config", "core.autocrlf", "false")
	return root
}

// A change lands as one commit describing itself in terms origins can read back, which
// is the whole of why origins does the committing. The trailers are read through git's
// own parser rather than out of the message text, because that is how every command
// that asks history a question will read them.
func TestAChangeIsOneCommitSayingWhatItDid(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())

	code, stdout, stderr := invoke(t, "add", "-C", root,
		"--source", "firenze-civil-deaths-1901#1103-17",
		"person/rossi-giovanni", "events", "{type: death, date: 14 MAR 1901, place: firenze}")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	if count := git(t, root, "rev-list", "--count", "HEAD"); count != "2" {
		t.Errorf("the corpus has %s commits", count)
	}
	for key, want := range map[string]string{
		"Origins-Tool":    "origins " + api.Version,
		"Origins-Op":      "events.add",
		"Origins-Subject": "person/rossi-giovanni",
		"Origins-Event":   "death",
		"Origins-Source":  "firenze-civil-deaths-1901#1103-17",
	} {
		if got := trailer(t, root, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if written := read(t, root, "people/rossi-giovanni/README.md"); !strings.Contains(written, "date: 14 MAR 1901") {
		t.Errorf("the person reads:\n%s", written)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}

// The events trailer is read on the side of the change the event is on: an event added
// is in the corpus only afterwards, and one taken out only before. Both are recorded,
// because a reader asking history for every death has no other way to find either.
func TestAnEventIsNamedWhicheverWayItWent(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	if code, _, stderr := invoke(t, "unset", "-C", root, "person/rossi-giovanni", "events[0]"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got := trailer(t, root, "Origins-Event"); got != "birth" {
		t.Errorf("Origins-Event = %q after taking a birth out", got)
	}
	if got := trailer(t, root, "Origins-Op"); got != "events.unset" {
		t.Errorf("Origins-Op = %q", got)
	}
}

// A value typed at a command is read as the field it is going into, and nothing else
// can decide it: the title of a register has a colon in it and is not a mapping, and a
// pair of coordinates is a pair of coordinates and not a sentence.
func TestAValueIsReadAsTheFieldItGoesInto(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())

	code, _, stderr := invoke(t, "create", "-C", root, "source/firenze-civil-deaths-1901",
		"title=Stato civile: registri di morte, 1901", "type=civil")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	written := read(t, root, "sources/firenze-civil-deaths-1901/README.md")
	if !strings.Contains(written, "Stato civile: registri di morte, 1901") {
		t.Errorf("the source reads:\n%s", written)
	}

	if code, _, stderr := invoke(t, "set", "-C", root, "place/firenze",
		"coordinates", "43.7696, 11.2558"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if written := read(t, root, "places/firenze/README.md"); !strings.Contains(written, "43.7696, 11.2558") {
		t.Errorf("the place reads:\n%s", written)
	}

	// what a file may not say, a command may not say either
	for _, refused := range [][]string{
		{"set", "-C", root, "person/rossi-giovanni", "sex", "male"},
		{"set", "-C", root, "person/rossi-giovanni", "birthplace", "firenze"},
		{"add", "-C", root, "person/rossi-giovanni", "events", "{type: death, parish: San Lorenzo}"},
	} {
		if code, _, _ := invoke(t, refused...); code != exitMisuse {
			t.Errorf("%q exited %d", strings.Join(refused, " "), code)
		}
	}
}

// The six commands that need nothing but a corpus: four that reach inside a record and
// two that are about a whole one, every one of them a commit. Rename, merge and split
// each want a corpus arranged for them and have a test apiece below.
func TestEachCommandDoesWhatItSays(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	for _, run := range [][]string{
		{"set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi Tempesti"},
		{"unset", "-C", root, "person/rossi-giovanni", "events[0].place"},
		{"add", "-C", root, "person/rossi-giovanni", "also-known-as", "Nino"},
		{"remove", "-C", root, "person/rossi-giovanni", "also-known-as", "Nino"},
		{"create", "-C", root, "place/borgo-san-lorenzo", "name=Borgo San Lorenzo", "type=comune"},
		{"delete", "-C", root, "place/borgo-san-lorenzo"},
	} {
		code, stdout, stderr := invoke(t, run...)
		if code != exitOK {
			t.Fatalf("origins %s: exit %d\nstdout: %s\nstderr: %s",
				strings.Join(run[3:], " "), code, stdout, stderr)
		}
	}

	person := read(t, root, "people/rossi-giovanni/README.md")
	switch {
	case !strings.Contains(person, "surname: Rossi Tempesti"):
		t.Errorf("set left the person as:\n%s", person)
	case strings.Contains(person, "place: firenze"):
		t.Errorf("unset left the place behind:\n%s", person)
	case !strings.Contains(person, "date: 1834"):
		t.Errorf("unsetting the place took the date with it:\n%s", person)
	case strings.Contains(person, "Nino"):
		t.Errorf("remove left the name behind:\n%s", person)
	}
	if _, err := os.Stat(filepath.Join(root, "places", "borgo-san-lorenzo", "README.md")); !os.IsNotExist(err) {
		t.Errorf("delete left the place behind: %v", err)
	}
	if count := git(t, root, "rev-list", "--count", "HEAD"); count != "7" {
		t.Errorf("six changes came to %s commits", count)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}

// Every operation origins can carry out has a command, and every command is one of the
// operations. The commands are the edit vocabulary and nothing besides, so a tenth
// command cannot appear without a tenth word to go with it, and the two lists come
// apart the moment either one of them moves.
func TestThereIsACommandForEachOperationAndNoOther(t *testing.T) {
	t.Parallel()
	commands, operations := slices.Sorted(maps.Keys(changes)), model.EditOps()
	if !slices.Equal(commands, operations) {
		t.Fatalf("the commands are %v and the vocabulary is %v", commands, operations)
	}
	for _, op := range operations {
		if _, _, stderr := invoke(t, op); strings.Contains(stderr, "unknown command") {
			t.Errorf("origins %s is in the table and run does not reach it", op)
		}
	}
}

// A rename moves what the entity occupies and rewrites everything pointing at it, in
// one commit, which is what lets git read it as the rename it is. Two commits would
// leave the corpus briefly holding neither name, and log --follow would stop there.
func TestARenameIsOneCommitGitReadsAsARename(t *testing.T) {
	t.Parallel()
	root := inARepo(t, map[string]string{
		"origins.yaml":                              sound,
		"people/rossi-giovanni/README.md":           theRossis()["people/rossi-giovanni/README.md"],
		"people/rossi-giovanni/photos/portrait.jpg": "a scan of a photograph",
		"places/firenze/README.md":                  theRossis()["places/firenze/README.md"],
	})

	code, stdout, stderr := invoke(t, "rename", "-C", root, "person/rossi-giovanni", "rossi-giovanni-1834")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for key, want := range map[string]string{
		"Origins-Op":      "person.rename",
		"Origins-Subject": "person/rossi-giovanni\nperson/rossi-giovanni-1834",
	} {
		if got := trailer(t, root, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	files := git(t, root, "show", "--name-status", "--format=", "HEAD")
	shows := "M\torigins.yaml\n" +
		"R100\tpeople/rossi-giovanni/README.md\tpeople/rossi-giovanni-1834/README.md\n" +
		"R100\tpeople/rossi-giovanni/photos/portrait.jpg\tpeople/rossi-giovanni-1834/photos/portrait.jpg"
	if files != shows {
		t.Errorf("the commit contains:\n%s\nwant:\n%s", files, shows)
	}

	// what the whole arrangement is for: the photograph is still his, before and after
	followed := git(t, root, "log", "--follow", "--format=%s", "--",
		"people/rossi-giovanni-1834/photos/portrait.jpg")
	if strings.Count(followed, "\n") != 1 {
		t.Errorf("following the photograph back reaches:\n%s", followed)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}

// A merge is a rename whose new name is already taken. What the record that goes had
// is in the record that stays, the photograph filed beside it is filed beside that one
// now, and the name it was kept under is nowhere in the corpus.
func TestAMergeLeavesOneRecordHoldingBoth(t *testing.T) {
	t.Parallel()
	root := inARepo(t, twoGiovannis())

	code, stdout, stderr := invoke(t, "merge", "-C", root,
		"person/rossi-giovanni-1902", "rossi-giovanni")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for key, want := range map[string]string{
		"Origins-Op":      "person.merge",
		"Origins-Subject": "person/rossi-giovanni-1902\nperson/rossi-giovanni",
	} {
		if got := trailer(t, root, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	stays := read(t, root, "people/rossi-giovanni/README.md")
	switch {
	case !strings.Contains(stays, "date: 1834"):
		t.Errorf("the record that stays lost its own birth:\n%s", stays)
	case !strings.Contains(stays, "date: 1902"):
		t.Errorf("the record that goes did not arrive:\n%s", stays)
	case !strings.Contains(stays, "portrait: photos/passaporto.jpg"):
		t.Errorf("the portrait did not come across:\n%s", stays)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-giovanni-1902")); !os.IsNotExist(err) {
		t.Errorf("people/rossi-giovanni-1902 is still there: %v", err)
	}

	// the photograph was carried rather than written out again, so git reads it as the
	// one photograph it is and a reader can follow it back past the merge
	files := git(t, root, "show", "--name-status", "--format=", "HEAD")
	moved := "R100\tpeople/rossi-giovanni-1902/photos/passaporto.jpg\t" +
		"people/rossi-giovanni/photos/passaporto.jpg"
	if !strings.Contains(files, moved) {
		t.Errorf("the commit contains:\n%s\nwant a line reading:\n%s", files, moved)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}

// A split makes the records and divides nothing between them. The first named keeps
// everything the record had, references to it included, and the rest arrive holding
// nothing at all: which fact belongs to which of them is the judgement that made the
// split worth proposing, and origins has nothing to decide it with.
func TestASplitMakesTheRecordsAndDividesNothing(t *testing.T) {
	t.Parallel()
	files := theRossis()
	files["people/rossi-maria/README.md"] = "---\nname:\n  given: Maria\n  surname: Rossi\nsex: F\n---\n"
	files["families/rossi-giovanni--unknown/README.md"] = "---\npartners: [rossi-giovanni, unknown]\n" +
		"children: [rossi-maria]\n---\n"
	root := inARepo(t, files)

	code, stdout, stderr := invoke(t, "split", "-C", root, "person/rossi-giovanni",
		"rossi-giovanni-1834", "rossi-giovanni-1867")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for key, want := range map[string]string{
		"Origins-Op": "person.split",
		"Origins-Subject": "person/rossi-giovanni\nperson/rossi-giovanni-1834\n" +
			"person/rossi-giovanni-1867",
	} {
		if got := trailer(t, root, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	keeps := read(t, root, "people/rossi-giovanni-1834/README.md")
	if !strings.Contains(keeps, "date: 1834") || !strings.Contains(keeps, "surname: Rossi") {
		t.Errorf("the first named did not keep what the record had:\n%s", keeps)
	}
	if empty := read(t, root, "people/rossi-giovanni-1867/README.md"); strings.Contains(empty, "Rossi") {
		t.Errorf("the second arrived holding something:\n%s", empty)
	}
	// a reference follows the record that kept everything, that being where the record
	// went; whether the daughter is this Giovanni's is the next question, not this one
	if family := read(t, root, "families/rossi-giovanni-1834--unknown/README.md"); !strings.Contains(family, "[rossi-giovanni-1834, unknown]") {
		t.Errorf("the daughter's family names:\n%s", family)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-giovanni", "README.md")); !os.IsNotExist(err) {
		t.Errorf("the record that was split is still there: %v", err)
	}

	// too few names is answered by spelling the command out: a record split into one
	// record is a rename, and into none is nothing at all
	for _, refused := range [][]string{
		{"split", "-C", root, "person/rossi-maria", "rossi-maria-1860"},
		{"split", "-C", root, "person/rossi-maria"},
		{"split", "-C", root},
	} {
		code, _, stderr := invoke(t, refused...)
		if code != exitMisuse || !strings.Contains(stderr, "that is not the shape of it") {
			t.Errorf("origins %s exited %d: %s", strings.Join(refused, " "), code, stderr)
		}
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}

// The entity says which kind of record is being changed, and every name after it is a
// name of that kind. Writing kind/slug twice is the mistake worth answering, kind/slug
// being how a record is named everywhere else, so it is said rather than left to come
// back as a slug with a slash in it.
func TestTheKindIsSaidOnce(t *testing.T) {
	t.Parallel()
	root := inARepo(t, twoGiovannis())
	for _, refused := range [][]string{
		{"merge", "-C", root, "person/rossi-giovanni-1902", "person/rossi-giovanni"},
		{"rename", "-C", root, "person/rossi-giovanni", "person/rossi-giovanni-1834"},
		{"split", "-C", root, "person/rossi-giovanni", "person/rossi-giovanni-1834", "rossi-giovanni-1867"},
	} {
		shown := strings.Join(refused[3:], " ")
		code, _, stderr := invoke(t, refused...)
		if code != exitMisuse {
			t.Errorf("origins %s %s exited %d", refused[0], shown, code)
		}
		if !strings.Contains(stderr, "the kind is said once") {
			t.Errorf("origins %s %s said %q", refused[0], shown, stderr)
		}
	}
}

// A change the corpus could not survive is refused whole: no commit, no file, and a
// tree as clean as it was found. The corpus disagreeing with itself is a finding about
// the records rather than a mistake in how origins was asked, so it exits 1.
func TestAChangeThatWouldContradictTheCorpusIsRefused(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	before := git(t, root, "rev-parse", "HEAD")

	code, _, stderr := invoke(t, "delete", "-C", root, "place/firenze")
	if code != exitFindings {
		t.Fatalf("exit %d, want %d\nstderr: %s", code, exitFindings, stderr)
	}
	if !strings.Contains(stderr, "dangling-place") {
		t.Errorf("stderr = %q", stderr)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != before {
		t.Errorf("the refused change was committed as %s", now)
	}
	if _, err := os.Stat(filepath.Join(root, "places", "firenze", "README.md")); err != nil {
		t.Errorf("the place went anyway: %v", err)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}

// A dry run is the question without the answer: it says what the change would come to,
// and the corpus and its history are exactly as they were.
func TestADryRunChangesNothing(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	before := git(t, root, "rev-parse", "HEAD")

	code, stdout, stderr := invoke(t, "set", "-C", root, "--dry-run", "--json",
		"person/rossi-giovanni", "name.surname", "Rossi Tempesti")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	var report api.Recorded
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.Commit != "" {
		t.Errorf("a dry run reported commit %q", report.Commit)
	}
	if len(report.Written) != 1 || report.Written[0] != "people/rossi-giovanni/README.md" {
		t.Errorf("would write %v", report.Written)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != before {
		t.Errorf("a dry run committed %s", now)
	}
	if written := read(t, root, "people/rossi-giovanni/README.md"); !strings.Contains(written, "surname: Rossi\n") {
		t.Errorf("a dry run wrote:\n%s", written)
	}
}

// A file origins could not read is not in the model, so writing the model back over the
// directory would record a corpus that never held what the directory holds — and it
// would look, ever after, exactly like a change somebody meant to make.
func TestACorpusThatDidNotReadWholeIsNotWrittenTo(t *testing.T) {
	t.Parallel()
	files := theRossis()
	files["people/rossi-antonio/README.md"] = "---\nname: Antonio\nborn-in: firenze\n---\n"
	root := inARepo(t, files)
	before := git(t, root, "rev-parse", "HEAD")

	code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi Tempesti")
	if code != exitMisuse {
		t.Fatalf("exit %d, want %d", code, exitMisuse)
	}
	if !strings.Contains(stderr, "born-in") {
		t.Errorf("the refusal does not say which file it could not read: %q", stderr)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != before {
		t.Errorf("it committed %s anyway", now)
	}
}

// Origins owns the history of a corpus, and a directory that is not versioned has none
// to own. The refusal comes before anything is written rather than after.
func TestACorpusOutsideGitIsNotChanged(t *testing.T) {
	t.Parallel()
	root := corpusDir(t, theRossis())
	code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi Tempesti")
	if code != exitMisuse {
		t.Fatalf("exit %d, want %d", code, exitMisuse)
	}
	if !strings.Contains(stderr, "git") {
		t.Errorf("the refusal does not mention git: %q", stderr)
	}
	if written := read(t, root, "people/rossi-giovanni/README.md"); !strings.Contains(written, "surname: Rossi\n") {
		t.Errorf("it wrote anyway:\n%s", written)
	}
}

// Parsing stops at the first argument that is not a flag, so a flag written after the
// entity is read as part of the edit. Saying which flag it was is the difference
// between a tool that looks broken and one that looks strict.
func TestAFlagAfterTheEntitySaysSo(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi", "--json")
	if code != exitMisuse {
		t.Fatalf("exit %d, want %d", code, exitMisuse)
	}
	if !strings.Contains(stderr, "--json") || !strings.Contains(stderr, "flags come first") {
		t.Errorf("stderr = %q", stderr)
	}
}
