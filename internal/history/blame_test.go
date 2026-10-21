package history

import (
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// blamed is the file every test here reads the history of, since blame is asked about
// one file at a time.
const blamed = "people/rossi-giovanni/README.md"

// changing records a file as it now stands, so each commit accounts for the lines it
// brought with it.
func changing(t *testing.T, repo *Repo, content string, c Commit) string {
	t.Helper()
	write(t, repo, blamed, content)
	sha, err := repo.Commit(c, []string{blamed})
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

func blaming(t *testing.T, repo *Repo) []Blamed {
	t.Helper()
	lines, err := repo.Blame(blamed)
	if err != nil {
		t.Fatal(err)
	}
	return lines
}

const born = `---
name: Giovanni
surname: Rossi
---
`

const died = `---
name: Giovanni
surname: Rossi
events:
  - type: death
---
`

// Canonical serialisation puts one fact on one line so that blame answers at the
// granularity of a fact. What blame has to bring back with each line is the whole of
// what the commit said, because the question behind when did this appear is always
// what it rested on.
func TestEachLineComesBackWithTheCommitThatWroteIt(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	first := changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))

	added := change("Add to events on person/rossi-giovanni", "events.add", person("rossi-giovanni"))
	added.Events = []string{"death"}
	added.Sources = []model.CitationRef{{Source: "firenze-civil-deaths-1901", Anchor: "1103-17"}}
	second := changing(t, repo, died, added)

	lines := blaming(t, repo)
	if len(lines) != 6 {
		t.Fatalf("%d lines came back, not 6", len(lines))
	}
	for i, want := range []struct {
		text string
		sha  string
	}{
		{"---", first},
		{"name: Giovanni", first},
		{"surname: Rossi", first},
		{"events:", second},
		{"  - type: death", second},
		{"---", first},
	} {
		got := lines[i]
		switch {
		case got.Line != i+1:
			t.Errorf("the line at %d is numbered %d", i+1, got.Line)
		case got.Text != want.text:
			t.Errorf("line %d is %q, want %q", i+1, got.Text, want.text)
		case got.Sha != want.sha:
			t.Errorf("line %q is %s, want %s", got.Text, short(got.Sha), short(want.sha))
		}
	}

	events := lines[3]
	if events.Op != "events.add" || events.Subject != "Add to events on person/rossi-giovanni" {
		t.Errorf("the commit behind %q is %q, %q", events.Text, events.Op, events.Subject)
	}
	if got := len(events.Sources); got != 1 || events.Sources[0].Anchor != "1103-17" {
		t.Errorf("the line rests on %v", events.Sources)
	}
	if events.Uncommitted() || events.Foreign() {
		t.Error("a line origins committed reads as somebody else's")
	}
}

// A corpus somebody has edited by hand is the case origins has to be able to see, not
// the case it can assume away. A line nothing accounts for must say so rather than be
// attributed to whatever it replaced.
func TestALineNobodyHasCommittedSaysSo(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	write(t, repo, blamed, strings.Replace(born, "surname: Rossi", "surname: Rossi di Firenze", 1))

	lines := blaming(t, repo)
	if len(lines) != 4 {
		t.Fatalf("%d lines came back, not 4", len(lines))
	}
	for _, line := range lines {
		if got := line.Uncommitted(); got != (line.Text == "surname: Rossi di Firenze") {
			t.Errorf("%q reads as uncommitted: %v", line.Text, got)
		}
	}
}

func TestALineOfACommitOriginsDidNotWriteIsForeign(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	byHand(t, repo, blamed, "Whatever I felt like writing")

	lines := blaming(t, repo)
	if len(lines) != 1 {
		t.Fatalf("%d lines came back, not 1", len(lines))
	}
	if !lines[0].Foreign() || lines[0].Uncommitted() {
		t.Errorf("%q reads as origins' own work", lines[0].Text)
	}
}

// One commit nobody can parse must not take away which line changed when, any more
// than it takes away the log it appears in.
func TestATrailerNobodyCanReadStillLeavesTheLines(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	byHand(t, repo, blamed, "Something somebody edited\n\nOrigins-Tool: origins 0.2.0\n"+
		"Origins-Subject: whoever that was")

	lines, err := repo.Blame(blamed)
	if err == nil {
		t.Fatal("a trailer that is not an entity reference was read without complaint")
	}
	if !strings.Contains(err.Error(), "Origins-Subject") {
		t.Errorf("the complaint does not name the trailer: %v", err)
	}
	if len(lines) != 4 {
		t.Fatalf("%d lines came back, not 4", len(lines))
	}
	if lines[0].Tool != "origins 0.2.0" {
		t.Errorf("the rest of the commit did not come back: %+v", lines[0].Entry)
	}
}

// Blaming a file git has never been shown is a question with no answer, and saying so
// is better than an empty one that looks like a file nobody has ever changed.
func TestAFileGitHasNeverSeenIsRefused(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	changing(t, repo, born, change("He was born", "person.create", person("rossi-giovanni")))
	write(t, repo, "people/bianchi-maria/README.md", born)

	_, err := repo.Blame("people/bianchi-maria/README.md")
	if err == nil {
		t.Fatal("a file nobody has committed was blamed anyway")
	}
	if !strings.Contains(err.Error(), "bianchi-maria") {
		t.Errorf("the complaint does not name the file: %v", err)
	}
}
