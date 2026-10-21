package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// theRossisMarried is theRossis with the family a rename has to carry, so that what
// reads history back is asked about the whole of what a rename does.
func theRossisMarried() map[string]string {
	files := theRossis()
	files["people/bianchi-maria/README.md"] = "---\nname:\n  given: Maria\n  surname: Bianchi\nsex: F\n---\n"
	files["families/bianchi-maria--rossi-giovanni/README.md"] =
		"---\npartners:\n  - rossi-giovanni\n  - bianchi-maria\n---\n"
	return files
}

// answering runs a command that reports on a corpus and reads back what it said.
// The flag goes in after the subcommand, because flags come first.
func answering[T any](t *testing.T, args ...string) T {
	t.Helper()
	var report T
	code, stdout, stderr := invoke(t, append([]string{args[0], "--json"}, args[1:]...)...)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	return report
}

func reading(t *testing.T, args ...string) api.Journal {
	t.Helper()
	report := answering[api.Journal](t, args...)
	if len(report.Problems) > 0 {
		t.Fatalf("reading the history reported %v", report.Problems)
	}
	return report
}

func operations(report api.Journal) []string {
	out := make([]string, 0, len(report.Entries))
	for _, one := range report.Entries {
		out = append(out, one.Op)
	}
	return out
}

// What origins adds to git log is the decoding. A commit says in trailers what it did
// and what the fact rests on, and reading that back is the whole point of having
// written it.
func TestTheLogSaysWhatEachCommitDid(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, _, stderr := invoke(t, "add", "-C", root,
		"--source", "firenze-civil-deaths-1901#1103-17",
		"person/rossi-giovanni", "events", "{type: death, date: 1901, place: firenze}")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}

	report := reading(t, "log", "-C", root, "person/rossi-giovanni")
	if len(report.Entries) != 1 {
		t.Fatalf("%d entries, not 1: %v", len(report.Entries), operations(report))
	}
	read := report.Entries[0]
	for what, got := range map[string]string{
		"op":       read.Op,
		"tool":     read.Tool,
		"author":   read.Author,
		"subject":  read.Subject,
		"subjects": strings.Join(read.Subjects, " "),
		"events":   strings.Join(read.Events, " "),
		"sources":  strings.Join(read.Sources, " "),
	} {
		want := map[string]string{
			"op":       "events.add",
			"tool":     "origins " + api.Version,
			"author":   "Rosa Bianchi",
			"subject":  "Add to events on person/rossi-giovanni",
			"subjects": "person/rossi-giovanni",
			"events":   "death",
			"sources":  "firenze-civil-deaths-1901#1103-17",
		}[what]
		if got != want {
			t.Errorf("%s is %q, want %q", what, got, want)
		}
	}
	if read.Commit == "" {
		t.Error("the entry names no commit")
	}

	_, stdout, _ := invoke(t, "log", "-C", root, "person/rossi-giovanni")
	for _, want := range []string{
		"Add to events on person/rossi-giovanni",
		"events.add  person/rossi-giovanni  (Rosa Bianchi)",
		"touching death, from firenze-civil-deaths-1901#1103-17",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the log does not say %q:\n%s", want, stdout)
		}
	}
}

// A person's history is the history of every name they have had, and of nobody else.
// A rename carries the family along, so one commit names four entities of two kinds,
// and neither kind may answer for the other.
func TestALogFollowsAPersonAndTheirFamilyAcrossARename(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossisMarried())
	for _, change := range [][]string{
		{"set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze"},
		{"rename", "-C", root, "person/rossi-giovanni", "rossi-giovanni-1834"},
	} {
		if code, _, stderr := invoke(t, change...); code != exitOK {
			t.Fatalf("%s: exit %d: %s", strings.Join(change, " "), code, stderr)
		}
	}

	for _, one := range []struct {
		asking string
		want   []string
	}{
		{"person/rossi-giovanni-1834", []string{"person.rename", "name.set"}},
		{"person/rossi-giovanni", []string{"person.rename", "name.set"}},
		{"family/bianchi-maria--rossi-giovanni-1834", []string{"person.rename"}},
		{"family/bianchi-maria--rossi-giovanni", []string{"person.rename"}},
		{"person/bianchi-maria", nil},
	} {
		got := operations(reading(t, "log", "-C", root, one.asking))
		if strings.Join(got, " ") != strings.Join(one.want, " ") {
			t.Errorf("asked after %s: got %v, want %v", one.asking, got, one.want)
		}
	}
}

// When a source turns out to be unreliable, what rests on it is what has to be looked
// at again — which is a question about the whole source, or about the one entry in it
// that was misread.
func TestWhenceFindsEveryConclusionDrawnFromASource(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	for _, change := range [][]string{
		{"set", "-C", root, "--source", "firenze-civil-deaths-1901#1103-17",
			"person/rossi-giovanni", "name.surname", "Rossi di Firenze"},
		{"set", "-C", root, "--source", "firenze-civil-deaths-1901#1104-02",
			"person/rossi-giovanni", "name", "Giovanni Battista"},
		{"set", "-C", root, "--source", "cimitero-san-lorenzo-gravestones#42",
			"person/rossi-giovanni", "sex", "U"},
		{"set", "-C", root, "--source", "firenze-civil-deaths-1901#1103-17 OSI",
			"person/rossi-giovanni", "name", "Giovanni"},
	} {
		if code, _, stderr := invoke(t, change...); code != exitOK {
			t.Fatalf("%s: exit %d: %s", strings.Join(change, " "), code, stderr)
		}
	}

	if got := operations(reading(t, "whence", "-C", root, "firenze-civil-deaths-1901")); len(got) != 3 {
		t.Errorf("the source answered for %v, want three changes", got)
	}
	// the entry is asked about whatever grade it was cited with, and the trailer
	// carries the grade it was
	for _, asking := range []string{"firenze-civil-deaths-1901#1103-17", "firenze-civil-deaths-1901#1103-17 OPD"} {
		report := reading(t, "whence", "-C", root, asking)
		if len(report.Entries) != 2 {
			t.Fatalf("%s answered for %d changes, want two", asking, len(report.Entries))
		}
		if got := report.Entries[1].Subject; got != "Set name.surname on person/rossi-giovanni" {
			t.Errorf("%s answered with %q", asking, got)
		}
		if got := report.Entries[0].Sources; len(got) != 1 || got[0] != "firenze-civil-deaths-1901#1103-17 OSI" {
			t.Errorf("the graded change was recorded as resting on %q", got)
		}
	}
	if got := operations(reading(t, "whence", "-C", root, "cimitero-san-lorenzo-gravestones")); len(got) != 1 {
		t.Errorf("the gravestone answered for %v, want one change", got)
	}
}

// The trap the commands that read a corpus share with the ones that change it: a flag
// written after the entity has stopped being a flag by the time the parser reaches it.
func TestAFlagAfterTheEntityIsAnsweredAsAFlag(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())

	code, _, stderr := invoke(t, "log", "-C", root, "person/rossi-giovanni", "--json")
	if code != exitMisuse {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "--json") || !strings.Contains(stderr, "flags come first") {
		t.Errorf("the complaint is %q", stderr)
	}
}

// The commit a corpus was put under version control with is the researcher's own work,
// and the log has to say so rather than pass it off as something origins recorded.
//
// Adopting it is what lets the log be asked at all, and adopting it does not make it
// origins' own: what origins wrote is the commit saying it had read the other one.
func TestTheLogSaysWhichCommitsAreNotOrigins(t *testing.T) {
	t.Parallel()
	root := byHand(t, theRossis())
	if code, _, stderr := invoke(t, "recover", "-C", root, "--adopt"); code != exitOK {
		t.Fatalf("adopting: exit %d: %s", code, stderr)
	}
	code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}

	_, stdout, _ := invoke(t, "log", "-C", root)
	if want := "not origins  (Rosa Bianchi)"; !strings.Contains(stdout, want) {
		t.Errorf("the log does not say %q in %q", want, stdout)
	}
	if got := strings.Count(stdout, "not origins"); got != 1 {
		t.Errorf("%d commits read as foreign, not 1: %q", got, stdout)
	}
}

// A name changed where the commit is not a rename is followed all the same, so a file
// renamed by hand and adopted keeps the history it had before, asked after by either
// name.
func TestALogFollowsARecordRenamedByHandAndAdopted(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	if code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze"); code != exitOK {
		t.Fatalf("set: exit %d: %s", code, stderr)
	}
	if err := os.Rename(filepath.Join(root, "people", "rossi-giovanni"),
		filepath.Join(root, "people", "rossi-giovanni-1834")); err != nil {
		t.Fatal(err)
	}
	edited(t, root, "origins.yaml", strings.Replace(read(t, root, "origins.yaml"),
		"rossi-giovanni", "rossi-giovanni-1834", 1))
	if code, _, stderr := invoke(t, "adopt", "-C", root); code != exitOK {
		t.Fatalf("adopt: exit %d: %s", code, stderr)
	}
	if got := trailer(t, root, "Origins-Renamed"); got != "person/rossi-giovanni person/rossi-giovanni-1834" {
		t.Errorf("the adoption says the names it changed were %q", got)
	}

	for _, asking := range []string{"person/rossi-giovanni-1834", "person/rossi-giovanni"} {
		got := operations(reading(t, "log", "-C", root, asking))
		if want := []string{"manual.adopt", "name.set"}; strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("asked after %s: got %v, want %v", asking, got, want)
		}
	}
}

// A hypothesis accepted that renames a record is a rename all the same, and the commit
// accepting it says so, the family carried with the person included.
func TestALogFollowsARecordRenamedByAnAcceptedHypothesis(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossisMarried())
	if code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze"); code != exitOK {
		t.Fatalf("set: exit %d: %s", code, stderr)
	}
	asking(t, root, "the-elder", "---\nstatus: open\nbase: "+git(t, root, "rev-parse", "HEAD")+
		"\nconfidence: possible\nedits:\n  - op: rename\n    entity: person/rossi-giovanni\n"+
		"    to: rossi-giovanni-1834\n---\n\n# Which Giovanni?\n\nThe elder, by his birth.\n")
	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "the-elder"); code != exitOK {
		t.Fatalf("accept: exit %d\n%s%s", code, stdout, stderr)
	}

	for _, one := range []struct {
		asking string
		want   []string
	}{
		{"person/rossi-giovanni-1834", []string{"hypothesis.accept", "name.set"}},
		{"person/rossi-giovanni", []string{"hypothesis.accept", "name.set"}},
		{"family/bianchi-maria--rossi-giovanni", []string{"hypothesis.accept"}},
		{"family/bianchi-maria--rossi-giovanni-1834", []string{"hypothesis.accept"}},
	} {
		got := operations(reading(t, "log", "-C", root, one.asking))
		if strings.Join(got, " ") != strings.Join(one.want, " ") {
			t.Errorf("asked after %s: got %v, want %v", one.asking, got, one.want)
		}
	}
}
