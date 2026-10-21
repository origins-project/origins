package main

import (
	"encoding/json"
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// asking writes a hypothesis into the corpus and has origins record it, which is
// where one has to be before it can be settled: accepting one changes its file, and
// it refuses to begin while anything in the corpus is unrecorded.
func asking(t *testing.T, root, name, content string) {
	t.Helper()
	put(t, root, "hypotheses/"+name+"/README.md", content)
	if code, stdout, stderr := invoke(t, "adopt", "-C", root, "-m", "Write the question down"); code != exitOK {
		t.Fatalf("recording the question exited %d\n%s%s", code, stdout, stderr)
	}
}

func says(t *testing.T, root, relative string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// The whole of what accepting one is for, end to end: the corpus contradicts itself,
// the hypothesis answers for it, and afterwards the corpus holds the answer, the
// hypothesis says it was accepted, and check has nothing left to report.
func TestAcceptingAHypothesisTakesItIntoTheCorpus(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio(base))

	code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "there-was-an-antonio")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, said := range []string{"Accept there-was-an-antonio", "carried out at " + short(base),
		"people/rossi-antonio/README.md"} {
		if !strings.Contains(stdout, said) {
			t.Errorf("stdout does not say %q:\n%s", said, stdout)
		}
	}

	if code, out, _ := invoke(t, "check", root); code != exitOK {
		t.Errorf("check exited %d after the hypothesis was accepted\n%s", code, out)
	}
	if !strings.Contains(says(t, root, "hypotheses/there-was-an-antonio/README.md"), "status: accepted") {
		t.Error("the hypothesis does not say it was accepted")
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("something was left unrecorded:\n%s", left)
	}
	if got := git(t, root, "log", "-1", "--format=%(trailers:key=Origins-Hypothesis,valueonly)"); got !=
		"there-was-an-antonio" {
		t.Errorf("the commit names %q as the hypothesis that decided it", got)
	}
}

// A rehearsal says what accepting would come to and leaves nothing behind, which for
// this one command means carrying the patch in and taking it straight back out.
func TestAcceptingSaysWhatItWouldDoWithoutDoingIt(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio(base))
	before := git(t, root, "rev-parse", "HEAD")

	code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "--dry-run",
		"there-was-an-antonio")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "would record people/rossi-antonio/README.md") {
		t.Errorf("the rehearsal does not say what it would write:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-antonio", "README.md")); !os.IsNotExist(err) {
		t.Errorf("the rehearsal wrote the record into the corpus: %v", err)
	}
	if !strings.Contains(says(t, root, "hypotheses/there-was-an-antonio/README.md"), "status: open") {
		t.Error("the rehearsal settled the hypothesis")
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != before {
		t.Error("the rehearsal recorded a commit")
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the corpus was not put back:\n%s", left)
	}
}

// A hypothesis somebody has overtaken exits the way a contradiction does, and the
// corpus comes out of the attempt exactly as it went in.
func TestAcceptingAHypothesisThatNoLongerAppliesReportsTheConflict(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "he-was-a-rossini", heWasARossini(base))
	if code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rosso"); code != exitOK {
		t.Fatalf("setting the surname exited %d: %s", code, stderr)
	}
	before := says(t, root, "people/rossi-giovanni/README.md")

	code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "he-was-a-rossini")
	if code != exitFindings {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "no longer applies") {
		t.Errorf("stderr does not say the hypothesis was overtaken:\n%s", stderr)
	}
	if after := says(t, root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("the conflict was left in the corpus:\n%s", after)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the corpus was not put back:\n%s", left)
	}
}

// Rejecting one writes nothing back: the corpus is the standing conclusion and
// already holds the answer, so all that changes is the hypothesis.
func TestRejectingAHypothesisChangesNothingButTheHypothesis(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "he-was-a-rossini", heWasARossini(base))
	before := says(t, root, "people/rossi-giovanni/README.md")

	code, stdout, stderr := invoke(t, "hypothesis", "reject", "-C", root, "he-was-a-rossini")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Reject he-was-a-rossini") {
		t.Errorf("stdout does not say what was recorded:\n%s", stdout)
	}
	if after := says(t, root, "people/rossi-giovanni/README.md"); after != before {
		t.Errorf("rejecting it changed the record:\n%s", after)
	}
	if !strings.Contains(says(t, root, "hypotheses/he-was-a-rossini/README.md"), "status: rejected") {
		t.Error("the hypothesis does not say it was rejected")
	}
	files := git(t, root, "show", "--name-only", "--format=", "HEAD")
	if files != "hypotheses/he-was-a-rossini/README.md" {
		t.Errorf("the commit carries:\n%s", files)
	}
}

// Carrying a patch writes over the files and putting it back writes over them again,
// so the one thing that must not be in the corpus when this starts is work nobody
// has recorded.
func TestAcceptingIsRefusedWhileTheCorpusHasChangesNobodyHasRecorded(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio(base))
	put(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")

	code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "there-was-an-antonio")
	if code != exitMisuse {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "origins adopt") {
		t.Errorf("stderr does not say what to do about it:\n%s", stderr)
	}
	if !strings.Contains(says(t, root, "places/firenze/README.md"), "comune") {
		t.Error("the unrecorded edit was written over")
	}
}

// Where --with falls back to the corpus in front of you, accepting refuses: a change
// to the standing conclusion is worth more care than a question about one.
func TestAcceptingAHypothesisWithNoStateToBeReadAtIsRefused(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio("8f78553"))

	if code, stdout, _ := invoke(t, "check", "--with", "there-was-an-antonio", root); code != exitOK {
		t.Fatalf("check --with exited %d, so the two do not differ here\n%s", code, stdout)
	}
	code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "there-was-an-antonio")
	if code != exitMisuse {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stderr, "cannot be read at the state it was written against") {
		t.Errorf("stderr does not say why:\n%s", stderr)
	}
}

func TestSettlingAHypothesisIsRefusedBeforeItStarts(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio(base))

	for named, this := range map[string]struct {
		args []string
		says string
	}{
		"no answer":       {[]string{"hypothesis"}, "that is not the shape of it"},
		"another answer":  {[]string{"hypothesis", "consider", "there-was-an-antonio"}, "is none of those"},
		"no name":         {[]string{"hypothesis", "accept", "-C", root}, "that is not the shape of it"},
		"the kind twice":  {[]string{"hypothesis", "accept", "-C", root, "hypothesis/there-was-an-antonio"}, "said once"},
		"nobody wrote it": {[]string{"hypothesis", "reject", "-C", root, "no-such-question"}, "no hypothesis called"},
	} {
		t.Run(named, func(t *testing.T) {
			code, stdout, stderr := invoke(t, this.args...)
			if code != exitMisuse {
				t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
			}
			if !strings.Contains(stderr, this.says) {
				t.Errorf("stderr does not say %q:\n%s", this.says, stderr)
			}
			if left := git(t, root, "status", "--porcelain"); left != "" {
				t.Errorf("the corpus was written to anyway:\n%s", left)
			}
		})
	}
}

func TestAcceptingReportsWhatItDidAsJSON(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio(base))

	code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "--json",
		"there-was-an-antonio")
	if code != exitOK {
		t.Fatalf("exit %d\nstderr: %s", code, stderr)
	}
	var report api.Decided
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	switch {
	case report.Hypothesis != "there-was-an-antonio":
		t.Errorf("the report is about %q", report.Hypothesis)
	case report.Status != "accepted":
		t.Errorf("the hypothesis stands at %q", report.Status)
	case report.Base != base:
		t.Errorf("it says it was carried out at %q, want %s", report.Base, base)
	case len(report.Commit) != 40:
		t.Errorf("the commit is %q", report.Commit)
	case !strings.Contains(strings.Join(report.Files, " "), "people/rossi-antonio/README.md"):
		t.Errorf("it changed %v", report.Files)
	case !strings.Contains(strings.Join(report.Subjects, " "), "person/rossi-antonio"):
		t.Errorf("it is about %v", report.Subjects)
	}
}

// A hypothesis written before its record was renamed still calls it by the name it had,
// and accepting it lands what it says on the record under the name it has now.
func TestAcceptingFollowsARecordRenamedSinceTheHypothesisWasWritten(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	asking(t, root, "he-was-a-rossini", heWasARossini(git(t, root, "rev-parse", "HEAD")))
	if code, _, stderr := invoke(t, "rename", "-C", root, "person/rossi-giovanni", "rossi-giovanni-1834"); code != exitOK {
		t.Fatalf("rename: exit %d: %s", code, stderr)
	}

	code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "he-was-a-rossini")
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(says(t, root, "people/rossi-giovanni-1834/README.md"), "surname: Rossini") {
		t.Errorf("the record under its new name reads:\n%s", says(t, root, "people/rossi-giovanni-1834/README.md"))
	}
	if _, err := os.Stat(filepath.Join(root, "people", "rossi-giovanni", "README.md")); !os.IsNotExist(err) {
		t.Error("accepting brought the record back under the name it had")
	}
	if got := trailer(t, root, "Origins-Subject"); !strings.Contains(got, "person/rossi-giovanni-1834") {
		t.Errorf("the commit is about %q", got)
	}
}

// Whether the corpus still holds what it accepted is a question only the history can
// answer, and it is answered for every hypothesis at once: an open one is simply open,
// and an accepted one whose fact somebody changed afterwards is one the corpus has left
// behind, with the file named and an exit of 1.
func TestStatusSaysWhetherTheCorpusStillHoldsWhatItAccepted(t *testing.T) {
	t.Parallel()
	root := inARepo(t, aMissingFather())
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "he-was-a-rossini", heWasARossini(base))
	asking(t, root, "there-was-an-antonio", thereWasAnAntonio(base))
	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "he-was-a-rossini"); code != exitOK {
		t.Fatalf("accept: exit %d\n%s%s", code, stdout, stderr)
	}

	code, stdout, stderr := invoke(t, "hypothesis", "status", "-C", root)
	if code != exitOK {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	for _, said := range []string{
		"he-was-a-rossini      accepted, and the corpus still holds it",
		"there-was-an-antonio  open",
	} {
		if !strings.Contains(stdout, said) {
			t.Errorf("status does not say %q:\n%s", said, stdout)
		}
	}

	if code, _, stderr := invoke(t, "set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi"); code != exitOK {
		t.Fatalf("set: exit %d: %s", code, stderr)
	}
	code, stdout, _ = invoke(t, "hypothesis", "status", "-C", root, "he-was-a-rossini")
	if code != exitFindings {
		t.Errorf("exit %d for a conclusion the corpus has left behind", code)
	}
	if want := "he-was-a-rossini  accepted, and the corpus no longer holds it: somebody has changed what " +
		"it changes, in people/rossi-giovanni/README.md\n"; stdout != want {
		t.Errorf("status says:\n%s\nwant:\n%s", stdout, want)
	}

	code, stdout, _ = invoke(t, "hypothesis", "status", "--json", "-C", root)
	var report []api.Holding
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	if code != exitFindings || len(report) != 2 || report[0].Holds == nil || *report[0].Holds ||
		strings.Join(report[0].Changed, " ") != "people/rossi-giovanni/README.md" || report[1].Holds != nil {
		t.Errorf("exit %d, and the report reads %s", code, stdout)
	}
}

func TestStatusOfAHypothesisTheCorpusDoesNotHaveIsRefused(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	if code, _, stderr := invoke(t, "hypothesis", "status", "-C", root, "nobody-asked"); code != exitMisuse ||
		!strings.Contains(stderr, "no hypothesis called nobody-asked") {
		t.Errorf("exit %d: %s", code, stderr)
	}
}
