package main

import (
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// theRossisArguing is theRossis with the hypothesis a fact can be decided by, so the
// chain from a fact to the argument behind it has all three of its links.
func theRossisArguing() map[string]string {
	files := theRossis()
	files["hypotheses/giovanni-died-1901/README.md"] = "---\nstatus: accepted\nbase: 8f78553\n" +
		"confidence: probable\nproposed-by: Rosa Bianchi\n" +
		"supports: [firenze-civil-deaths-1901#1103-17]\n" +
		"contradicts: [cimitero-san-lorenzo-gravestones#42]\n" +
		"edits:\n  - op: set\n    entity: person/rossi-giovanni\n" +
		"    path: events[0].date\n    value: 1901\n---\n\n" +
		"The gravestone is worn and the 9 could be an 8. The register is legible.\n"
	return files
}

// squeezed collapses the columns, so that a test can say what a row holds without
// saying how wide the longest path in that particular corpus happened to be.
func squeezed(out string) string { return strings.Join(strings.Fields(out), " ") }

func fields(report api.Provenance) map[string]api.Attribution {
	found := map[string]api.Attribution{}
	for _, one := range report.Fields {
		found[one.Path] = one
	}
	return found
}

// hand edits a corpus the way somebody with a text editor would, which is how the one
// trailer nothing writes yet gets written at all.
func hand(t *testing.T, root, relative, was, now string, message ...string) {
	t.Helper()
	file := filepath.Join(root, filepath.FromSlash(relative))
	content := read(t, root, relative)
	if !strings.Contains(content, was) {
		t.Fatalf("%s does not say %q", relative, was)
	}
	if err := os.WriteFile(file, []byte(strings.Replace(content, was, now, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if len(message) > 0 {
		git(t, root, append(append([]string{"commit", "-q"}, message...), "--", relative)...)
	}
}

// Canonical serialisation is what makes a line of a corpus a fact somebody asserted.
// Naming that fact is the whole of what origins adds to git blame, and the name has to
// be the one the rest of origins uses.
func TestBlameNamesEachFactAndWhatItRestsOn(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	for _, change := range [][]string{
		{"set", "-C", root, "--source", "firenze-civil-deaths-1901#1103-17",
			"person/rossi-giovanni", "name.surname", "Rossi di Firenze"},
		{"add", "-C", root, "--source", "cimitero-san-lorenzo-gravestones#42",
			"person/rossi-giovanni", "events", "{type: death, date: 1901}"},
	} {
		if code, _, stderr := invoke(t, change...); code != exitOK {
			t.Fatalf("%s: exit %d: %s", strings.Join(change, " "), code, stderr)
		}
	}

	report := answering[api.Provenance](t, "blame", "-C", root, "person/rossi-giovanni")
	var named []string
	for _, one := range report.Fields {
		named = append(named, one.Path)
	}
	want := "name name.given name.surname sex events events[0].type events[0].date events[0].place " +
		"events[1].type events[1].date"
	if strings.Join(named, " ") != want {
		t.Errorf("the file reads as %q, want %q", strings.Join(named, " "), want)
	}

	found := fields(report)
	for path, want := range map[string]struct {
		op     string
		source string
	}{
		"name":           {"manual.adopt", ""},
		"name.surname":   {"name.set", "firenze-civil-deaths-1901#1103-17"},
		"events[0].type": {"manual.adopt", ""},
		"events[1].type": {"events.add", "cimitero-san-lorenzo-gravestones#42"},
		"events[1].date": {"events.add", "cimitero-san-lorenzo-gravestones#42"},
	} {
		one, ok := found[path]
		if !ok {
			t.Errorf("%s was not found in the file at all", path)
			continue
		}
		if one.Commit == nil {
			t.Errorf("%s is the work of no commit", path)
			continue
		}
		if one.Commit.Op != want.op || strings.Join(one.Commit.Sources, " ") != want.source {
			t.Errorf("%s is %q from %q, want %q from %q",
				path, one.Commit.Op, one.Commit.Sources, want.op, want.source)
		}
	}
	if len(found["name.surname"].Lines) != 1 || found["name.surname"].Lines[0] != "  surname: Rossi di Firenze" {
		t.Errorf("the surname reads %q", found["name.surname"].Lines)
	}

	_, stdout, _ := invoke(t, "blame", "-C", root, "person/rossi-giovanni")
	for _, want := range []string{
		"name.surname from firenze-civil-deaths-1901#1103-17",
		"name",
	} {
		if !strings.Contains(squeezed(stdout), want) {
			t.Errorf("the blame does not say %q:\n%s", want, stdout)
		}
	}
}

// A field asked about answers with everything inside it, since events is how somebody
// asks about an event without knowing which index it ended up at.
func TestBlameOfOneFieldLeavesTheRestOut(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	report := answering[api.Provenance](t, "blame", "-C", root, "person/rossi-giovanni", "events")

	var paths []string
	for _, one := range report.Fields {
		paths = append(paths, one.Path)
	}
	want := "events events[0].type events[0].date events[0].place"
	if strings.Join(paths, " ") != want {
		t.Errorf("got %q, want %q", strings.Join(paths, " "), want)
	}
}

// A renamed person keeps the history of the facts about them. The file moved, and
// blame follows it, so nothing about the record begins the day it was renamed.
func TestBlameFollowsAFactPastARename(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	for _, change := range [][]string{
		{"set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze"},
		{"rename", "-C", root, "person/rossi-giovanni", "rossi-giovanni-1834"},
	} {
		if code, _, stderr := invoke(t, change...); code != exitOK {
			t.Fatalf("%s: exit %d: %s", strings.Join(change, " "), code, stderr)
		}
	}

	found := fields(answering[api.Provenance](t, "blame", "-C", root, "person/rossi-giovanni-1834"))
	one, ok := found["name.surname"]
	if !ok || one.Commit == nil {
		t.Fatalf("the surname came back as %+v", one)
	}
	if one.Commit.Op != "name.set" {
		t.Errorf("the surname is the work of %q, want the change that set it", one.Commit.Op)
	}
}

// A corpus somebody edited by hand is the case origins has to be able to see. The fact
// is in the file and in no commit, and saying which commit it came from would be
// naming the one it replaced.
func TestAFactNobodyHasCommittedSaysSo(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	hand(t, root, "people/rossi-giovanni/README.md", "surname: Rossi", "surname: Rossi di Firenze")

	found := fields(answering[api.Provenance](t, "blame", "-C", root, "person/rossi-giovanni"))
	if found["surname"].Commit != nil {
		t.Errorf("a line nobody committed is the work of %+v", found["surname"].Commit)
	}
	if found["name"].Commit == nil {
		t.Error("a line that was committed reads as though it was not")
	}

	_, stdout, _ := invoke(t, "blame", "-C", root, "person/rossi-giovanni")
	if want := "surname not committed"; !strings.Contains(squeezed(stdout), want) {
		t.Errorf("the blame does not say %q:\n%s", want, stdout)
	}
}

// The whole point of the trailers: a fact leads to the commit that set it, the commit
// names the hypothesis that decided it, and the hypothesis holds the evidence, the
// alternative it was weighed against and what would overturn it.
func TestWhyReachesTheArgumentBehindAFact(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossisArguing())
	hand(t, root, "people/rossi-giovanni/README.md", "date: 1834", "date: 1901",
		"-m", "Set date on person/rossi-giovanni",
		"-m", "Origins-Tool: origins "+api.Version+"\nOrigins-Op: events.set\n"+
			"Origins-Subject: person/rossi-giovanni\nOrigins-Event: birth\n"+
			"Origins-Source: firenze-civil-deaths-1901#1103-17\n"+
			"Origins-Hypothesis: giovanni-died-1901")

	report := answering[api.Reason](t, "why", "-C", root, "rossi-giovanni", "events[0].date")
	switch {
	case report.Field == nil:
		t.Fatal("the fact was not found")
	case report.Field.Lines[0] != "    date: 1901":
		t.Errorf("the fact reads %q", report.Field.Lines)
	case report.Field.Commit == nil:
		t.Fatal("the fact is the work of no commit")
	case report.Field.Commit.Hypothesis != "giovanni-died-1901":
		t.Errorf("the commit was decided by %q", report.Field.Commit.Hypothesis)
	case report.Field.Commit.Date == "":
		t.Error("the commit is undated")
	}
	if report.Hypothesis == nil {
		t.Fatal("the hypothesis the commit names was not read back")
	}
	for what, got := range map[string]string{
		"status":      report.Hypothesis.Status,
		"confidence":  report.Hypothesis.Confidence,
		"supports":    strings.Join(report.Hypothesis.Supports, " "),
		"contradicts": strings.Join(report.Hypothesis.Contradicts, " "),
	} {
		want := map[string]string{
			"status":      "accepted",
			"confidence":  "probable",
			"supports":    "firenze-civil-deaths-1901#1103-17",
			"contradicts": "cimitero-san-lorenzo-gravestones#42",
		}[what]
		if got != want {
			t.Errorf("the hypothesis %s is %q, want %q", what, got, want)
		}
	}
	if !strings.Contains(report.Hypothesis.Body, "could be an 8") {
		t.Errorf("the argument itself did not come back: %q", report.Hypothesis.Body)
	}

	_, stdout, _ := invoke(t, "why", "-C", root, "rossi-giovanni", "events[0].date")
	for _, want := range []string{
		"person/rossi-giovanni events[0].date",
		"date: 1901",
		"by giovanni-died-1901",
		"hypothesis/giovanni-died-1901  accepted, probable, Rosa Bianchi",
		"against cimitero-san-lorenzo-gravestones#42",
		"could be an 8",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the chain does not say %q:\n%s", want, stdout)
		}
	}
}

// A path the file writes nothing at is a question with an answer, and the answer is
// that nothing is written there.
func TestAskingAfterAFieldTheFileDoesNotWrite(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	report := answering[api.Reason](t, "why", "-C", root, "person/rossi-giovanni", "portrait")
	if report.Field != nil {
		t.Errorf("a field the file does not write came back as %+v", report.Field)
	}

	code, stdout, _ := invoke(t, "why", "-C", root, "person/rossi-giovanni", "portrait")
	if code != exitOK {
		t.Errorf("exit %d", code)
	}
	if want := "writes nothing at portrait"; !strings.Contains(stdout, want) {
		t.Errorf("the answer is %q", stdout)
	}
}

// A slug is unique across the kinds, so a bare one is enough here — but a kind written
// out has to be the right one, or blaming the wrong entity's file would answer rather
// than fail.
func TestAnEntityOfAnotherKindIsRefused(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, _, stderr := invoke(t, "blame", "-C", root, "place/rossi-giovanni")
	if code != exitMisuse {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "is a person") {
		t.Errorf("the complaint is %q", stderr)
	}
}

// A value written over several lines is still one fact, and it stands as of its last
// change. The body is the long value in the format — it carries the argument for why
// one Giovanni is that Giovanni — and somebody coming back to correct a sentence of it
// has changed the whole of what it says.
func TestAFactWrittenOverSeveralLinesReadsAsOfItsLastChange(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	const closing = "    place: firenze\n---\n"
	trailers := []string{"-m", "Origins-Tool: origins " + api.Version + "\nOrigins-Op: body.set\n" +
		"Origins-Subject: person/rossi-giovanni"}
	hand(t, root, "people/rossi-giovanni/README.md", closing,
		closing+"\nHe was born in Firenze.\n\nThe register is legible.\n",
		append([]string{"--date", "2020-03-04T05:06:07", "-m", "Write down what I know"}, trailers...)...)
	hand(t, root, "people/rossi-giovanni/README.md",
		"He was born in Firenze.", "He was born in Borgo San Lorenzo.",
		append([]string{"--date", "2021-03-04T05:06:07", "-m", "Correct the place"}, trailers...)...)
	corrected := git(t, root, "rev-parse", "HEAD")

	body := fields(answering[api.Provenance](t, "blame", "-C", root, "person/rossi-giovanni"))["body"]
	if len(body.Lines) != 3 {
		t.Fatalf("the body came back as %q", body.Lines)
	}
	if body.Commit == nil || body.Commit.Commit != corrected {
		t.Errorf("the body is the work of %+v, want the commit that last changed it", body.Commit)
	}
}
