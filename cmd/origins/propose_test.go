package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/api"
)

const (
	louisaFile = "people/verdier-louisa/README.md"
	familyFile = "families/lang-anna-maria--verdier-louis/README.md"
)

// editing changes the fixture as though a hypothesis were true — Louisa born in
// 1844 on the census's word alone, dead in 1901, and with a brother nobody recorded —
// and puts the files in the form origins writes, returning them as they now read.
func editing(t *testing.T, root string) map[string]string {
	t.Helper()
	was := says(t, root, louisaFile)
	now := strings.Replace(was, "    date: 1842\n", "    date: 1844\n", 1)
	now = strings.Replace(now, "      - french-church-london-baptisms#1842-007\n", "", 1)
	now = strings.Replace(now, "\n---\n\n", "\n  - type: death\n    date: 1901\n---\n\n", 1)
	if now == was {
		t.Fatal("the fixture no longer reads the way this test edits it")
	}
	put(t, root, louisaFile, now)
	put(t, root, "people/verdier-baby-1845/README.md",
		"---\nname: Baby Verdier\nsex: M\n---\n")
	was = says(t, root, familyFile)
	if now = strings.Replace(was, "  - verdier-louisa\n", "  - verdier-louisa\n  - verdier-baby-1845\n", 1); now == was {
		t.Fatal("the fixture's family no longer reads the way this test edits it")
	}
	put(t, root, familyFile, now)
	if code, stdout, stderr := invoke(t, "fmt", root); code != exitOK {
		t.Fatalf("fmt exited %d\n%s%s", code, stdout, stderr)
	}
	return map[string]string{
		louisaFile:                           says(t, root, louisaFile),
		"people/verdier-baby-1845/README.md": says(t, root, "people/verdier-baby-1845/README.md"),
		familyFile:                           says(t, root, familyFile),
	}
}

// The whole of what proposing is for, end to end: the edits are taken out of the files
// and into a hypothesis pinned to the last commit, the corpus is left as that commit
// had it, and supposing the hypothesis gives back exactly what was written.
func TestAProposalGivesBackTheFilesItWasMadeFrom(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	base := git(t, root, "rev-parse", "HEAD")
	recorded := says(t, root, louisaFile)
	edited := editing(t, root)

	code, stdout, stderr := invoke(t, "hypothesis", "propose", "-C", root,
		"--source", "census-1861-england#f214", "a-brother-in-1845")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, said := range []string{
		"Propose a-brother-in-1845",
		"pinned to " + short(base),
		"create person/verdier-baby-1845",
		"set person/verdier-louisa events[0].date",
		"remove person/verdier-louisa events[0].sources",
		"add person/verdier-louisa events",
	} {
		if !strings.Contains(stdout, said) {
			t.Errorf("stdout does not say %q:\n%s", said, stdout)
		}
	}

	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("something was left unrecorded:\n%s", left)
	}
	if says(t, root, louisaFile) != recorded {
		t.Error("Louisa's file was not put back as the last commit had it")
	}
	hypothesis := says(t, root, "hypotheses/a-brother-in-1845/README.md")
	for _, said := range []string{"status: open", "base: " + base, "supports: [census-1861-england#f214]"} {
		if !strings.Contains(hypothesis, said) {
			t.Errorf("the hypothesis does not say %q:\n%s", said, hypothesis)
		}
	}
	if got := git(t, root, "log", "-1", "--format=%(trailers:key=Origins-Hypothesis,valueonly)"); got !=
		"a-brother-in-1845" {
		t.Errorf("the commit names %q as the hypothesis", got)
	}

	if code, stdout, _ := invoke(t, "check", "--with", "a-brother-in-1845", root); code != exitOK {
		t.Errorf("check --with exited %d\n%s", code, stdout)
	}
	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "a-brother-in-1845"); code != exitOK {
		t.Fatalf("accepting exited %d\n%s%s", code, stdout, stderr)
	}
	for path, want := range edited {
		if got := says(t, root, path); got != want {
			t.Errorf("%s after accepting:\n%s\nwant, as it was edited:\n%s", path, got, want)
		}
	}
}

// A proposal somebody else made says so in the file, in the commit and in the log, and
// the commit accepting it says so again, since its author is whoever accepted it.
func TestAProposalSaysWhoProposedIt(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	editing(t, root)

	code, stdout, stderr := invoke(t, "hypothesis", "propose", "-C", root,
		"--proposed-by", "transcriber", "a-brother-in-1845")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "proposed by transcriber") {
		t.Errorf("stdout does not say who proposed it:\n%s", stdout)
	}
	if hypothesis := says(t, root, "hypotheses/a-brother-in-1845/README.md"); !strings.Contains(
		hypothesis, "proposed-by: transcriber\n") {
		t.Errorf("the hypothesis does not say who proposed it:\n%s", hypothesis)
	}
	proposer := func() string {
		return git(t, root, "log", "-1", "--format=%(trailers:key=Origins-Proposed-By,valueonly)")
	}
	if got := proposer(); got != "transcriber" {
		t.Errorf("the proposal's commit names %q as its proposer", got)
	}
	if code, stdout, _ := invoke(t, "log", "-C", root, "hypothesis/a-brother-in-1845"); code != exitOK ||
		!strings.Contains(stdout, "by a-brother-in-1845, proposed by transcriber") {
		t.Errorf("log exited %d and says:\n%s", code, stdout)
	}

	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "a-brother-in-1845"); code != exitOK {
		t.Fatalf("accepting exited %d\n%s%s", code, stdout, stderr)
	}
	if got := proposer(); got != "transcriber" {
		t.Errorf("the acceptance names %q as the proposer", got)
	}
}

func TestProposingSaysWhatItWouldStateWithoutStatingIt(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	base := git(t, root, "rev-parse", "HEAD")
	edited := editing(t, root)

	code, stdout, stderr := invoke(t, "hypothesis", "propose", "--dry-run", "-C", root, "a-brother-in-1845")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "would state create person/verdier-baby-1845") {
		t.Errorf("stdout does not say what it would state:\n%s", stdout)
	}
	if head := git(t, root, "rev-parse", "HEAD"); head != base {
		t.Error("a rehearsal made a commit")
	}
	for path, want := range edited {
		if says(t, root, path) != want {
			t.Errorf("a rehearsal changed %s", path)
		}
	}
}

// Putting the files back takes away whatever is unrecorded in the corpus, so anything
// the hypothesis would not hold stops it before it starts.
func TestProposingIsRefusedWhereItWouldLoseSomething(t *testing.T) {
	t.Parallel()
	for name, change := range map[string]func(t *testing.T, root string){
		"a note beside the records": func(t *testing.T, root string) {
			put(t, root, "notes.txt", "ask the archive about 1845\n")
		},
		"an entry put in before the end": func(t *testing.T, root string) {
			put(t, root, louisaFile, strings.Replace(says(t, root, louisaFile),
				"events:\n", "events:\n  - type: baptism\n    date: 19 MAR 1842\n", 1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := gitFixture(t)
			base := git(t, root, "rev-parse", "HEAD")
			edited := editing(t, root)
			change(t, root)
			edited[louisaFile] = says(t, root, louisaFile)

			code, stdout, stderr := invoke(t, "hypothesis", "propose", "-C", root, "a-brother-in-1845")
			if code != exitMisuse {
				t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
			}
			if head := git(t, root, "rev-parse", "HEAD"); head != base {
				t.Error("a refused proposal made a commit")
			}
			for path, want := range edited {
				if says(t, root, path) != want {
					t.Errorf("a refused proposal changed %s", path)
				}
			}
		})
	}
}

func TestProposingNothingIsRefused(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	code, _, stderr := invoke(t, "hypothesis", "propose", "-C", root, "a-brother-in-1845")
	if code != exitMisuse || !strings.Contains(stderr, "as though a-brother-in-1845 were true") {
		t.Errorf("exit %d\n%s", code, stderr)
	}
}

// The manifest is edited as any file is and proposed as edits to project, which every
// road a hypothesis takes carries as it carries an edit to a record.
func TestAProposalReachesTheManifest(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	recorded := says(t, root, "origins.yaml")
	edited := strings.Replace(recorded, "main-person: verdier-jean-baptiste\n", "main-person: verdier-louisa\n", 1)
	edited = strings.Replace(edited, "  bookmarks:\n", "    - name: Rosa Bianchi\n  bookmarks:\n", 1)
	if edited == recorded {
		t.Fatal("the fixture's manifest no longer reads the way this test edits it")
	}
	put(t, root, "origins.yaml", edited)

	code, stdout, stderr := invoke(t, "hypothesis", "propose", "-C", root, "louisa-at-the-centre")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, said := range []string{"set project main-person", "add project researchers"} {
		if !strings.Contains(stdout, said) {
			t.Errorf("stdout does not say %q:\n%s", said, stdout)
		}
	}
	if says(t, root, "origins.yaml") != recorded {
		t.Error("the manifest was not put back as the last commit had it")
	}

	if code, stdout, _ := invoke(t, "check", "--with", "louisa-at-the-centre", root); code != exitOK {
		t.Errorf("check --with exited %d\n%s", code, stdout)
	}
	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "louisa-at-the-centre"); code != exitOK {
		t.Fatalf("accepting exited %d\n%s%s", code, stdout, stderr)
	}
	if got := says(t, root, "origins.yaml"); got != edited {
		t.Errorf("the manifest after accepting:\n%s\nwant, as it was edited:\n%s", got, edited)
	}
	code, stdout, stderr = invoke(t, "hypothesis", "status", "-C", root, "louisa-at-the-centre")
	if code != exitOK || !strings.Contains(stdout, "accepted, and the corpus still holds it") {
		t.Errorf("status exited %d\n%s%s", code, stdout, stderr)
	}
}

// louisa1844 is a hypothesis stated whole, as an agent would write it without
// touching the corpus: Louisa born in 1844, on the census's word.
const louisa1844 = "---\n%sconfidence: possible\nsupports: [census-1861-england#f214]\n" +
	"edits:\n  - op: set\n    entity: person/verdier-louisa\n    path: events[0].date\n    value: \"%s\"\n" +
	"---\n\n# Was Louisa born in 1844?\n\nThe census gives her age as 17.\n"

// statement writes a hypothesis's file outside the corpus and hands back where.
func statement(t *testing.T, frontmatter, date string) string {
	t.Helper()
	dir := t.TempDir()
	put(t, dir, "stated.md", fmt.Sprintf(louisa1844, frontmatter, date))
	return filepath.Join(dir, "stated.md")
}

// A hypothesis stated whole is recorded as it was stated, pinned to the last commit,
// with nothing else in the corpus touched, and accepting it lands what it stated.
func TestAHypothesisStatedWholeIsProposedAsItStands(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	base := git(t, root, "rev-parse", "HEAD")
	recorded := says(t, root, louisaFile)

	code, stdout, stderr := invoke(t, "hypothesis", "propose", "-C", root, "--proposed-by", "transcriber",
		"--from", statement(t, "", "1844"), "louisa-in-1844")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, said := range []string{"Propose louisa-in-1844", "pinned to " + short(base),
		"proposed by transcriber", "set person/verdier-louisa events[0].date"} {
		if !strings.Contains(stdout, said) {
			t.Errorf("stdout does not say %q:\n%s", said, stdout)
		}
	}
	if strings.Contains(stdout, "back as") {
		t.Errorf("stdout says files were put back, and none were edited:\n%s", stdout)
	}
	if touched := git(t, root, "show", "--name-only", "--format=", "HEAD"); touched !=
		"hypotheses/louisa-in-1844/README.md" {
		t.Errorf("the commit touched:\n%s", touched)
	}
	if says(t, root, louisaFile) != recorded {
		t.Error("Louisa's file was changed by proposing")
	}
	hypothesis := says(t, root, "hypotheses/louisa-in-1844/README.md")
	for _, said := range []string{"status: open\n", "base: " + base + "\n", "proposed-by: transcriber\n",
		"# Was Louisa born in 1844?"} {
		if !strings.Contains(hypothesis, said) {
			t.Errorf("the hypothesis does not say %q:\n%s", said, hypothesis)
		}
	}
	for key, want := range map[string]string{
		"Origins-Op": "hypothesis.propose", "Origins-Proposed-By": "transcriber",
		"Origins-Source": "census-1861-england#f214",
	} {
		if got := git(t, root, "log", "-1", "--format=%(trailers:key="+key+",valueonly)"); got != want {
			t.Errorf("%s is %q, want %q", key, got, want)
		}
	}

	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "louisa-in-1844"); code != exitOK {
		t.Fatalf("accepting exited %d\n%s%s", code, stdout, stderr)
	}
	if got := says(t, root, louisaFile); !strings.Contains(got, "    date: 1844\n") {
		t.Errorf("Louisa's file after accepting:\n%s", got)
	}
}

func TestProposingAStatedHypothesisDryWritesNothing(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	base := git(t, root, "rev-parse", "HEAD")
	code, stdout, stderr := invoke(t, "hypothesis", "propose", "-C", root, "--dry-run", "--json",
		"--from", statement(t, "", "1844"), "louisa-in-1844")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	var report api.Proposal
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if report.Base != base || report.Commit != "" || len(report.Edits) != 1 {
		t.Errorf("the rehearsal reported %+v", report)
	}
	if git(t, root, "rev-parse", "HEAD") != base || git(t, root, "status", "--porcelain") != "" {
		t.Error("a rehearsal changed the corpus")
	}
}

// What would make a stated hypothesis anything but an open question the corpus can be
// asked is refused, and nothing is written.
func TestProposingAStatedHypothesisIsRefusedWhereItCouldNotBeSupposed(t *testing.T) {
	t.Parallel()
	recording := func(t *testing.T, root string, args ...string) string {
		t.Helper()
		if code, stdout, stderr := invoke(t, append(args[:1:1], append([]string{"-C", root}, args[1:]...)...)...); code != exitOK {
			t.Fatalf("%v exited %d\n%s%s", args, code, stdout, stderr)
		}
		return git(t, root, "rev-parse", "HEAD")
	}
	for _, refusal := range []struct {
		name, slug, saying string
		stated             func(t *testing.T, root string) string
	}{
		{"settled", "louisa-in-1844", "says it is accepted", func(t *testing.T, root string) string {
			return statement(t, "status: accepted\n", "1844")
		}},
		{"its proposer named", "louisa-in-1844", "says who proposed it", func(t *testing.T, root string) string {
			return statement(t, "proposed-by: transcriber\n", "1844")
		}},
		{"a base outside the history", "louisa-in-1844", "no commit in this corpus's history",
			func(t *testing.T, root string) string {
				return statement(t, "base: 0123456789abcdef0123456789abcdef01234567\n", "1844")
			}},
		{"a slug taken", "unrecorded-child-1845", "already has something called", func(t *testing.T, root string) string {
			return statement(t, "", "1844")
		}},
		{"no edits", "louisa-in-1844", "states no edits", func(t *testing.T, root string) string {
			dir := t.TempDir()
			put(t, dir, "stated.md", "---\nconfidence: possible\n---\n\n# Nothing?\n")
			return filepath.Join(dir, "stated.md")
		}},
		{"a conflict", "louisa-in-1844", "no longer applies", func(t *testing.T, root string) string {
			base := recording(t, root, "set", "person/verdier-louisa", "events[0].date", "1843")
			recording(t, root, "set", "person/verdier-louisa", "events[0].date", "1845")
			return statement(t, "base: "+base+"\n", "1844")
		}},
		{"a record gone", "louisa-in-1844", "no longer applies", func(t *testing.T, root string) string {
			base := recording(t, root, "create", "person/doe-jane", "name=Jane Doe")
			recording(t, root, "delete", "person/doe-jane")
			dir := t.TempDir()
			put(t, dir, "stated.md", "---\nbase: "+base+"\nedits:\n  - op: set\n    entity: person/doe-jane\n"+
				"    path: sex\n    value: F\n---\n\n# Was she a daughter?\n")
			return filepath.Join(dir, "stated.md")
		}},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			t.Parallel()
			root := gitFixture(t)
			from := refusal.stated(t, root)
			head := git(t, root, "rev-parse", "HEAD")
			code, stdout, stderr := invoke(t, "hypothesis", "propose", "-C", root, "--from", from, refusal.slug)
			if code == exitOK || !strings.Contains(stderr, refusal.saying) {
				t.Errorf("exit %d, and stderr does not say %q\nstdout: %s\nstderr: %s",
					code, refusal.saying, stdout, stderr)
			}
			if git(t, root, "rev-parse", "HEAD") != head || git(t, root, "status", "--porcelain") != "" {
				t.Error("a refused proposal changed the corpus")
			}
		})
	}
}
