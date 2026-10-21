package main

import (
	"github.com/origins-project/origins/internal/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// edited writes a file of the corpus and leaves the change where it is, unrecorded.
func edited(t *testing.T, root, relative, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const giovanni = "---\nname:\n  given: Giovanni\n  surname: Rossi\nsex: M\n" +
	"events:\n  - type: birth\n    date: 1834\n    place: firenze\n---\n"

// The rule, and the reason for everything else in this file. Origins owns the history
// of a corpus, so a commit it cannot account for stops every command that reads or
// writes one — because blame, why, whence and a hypothesis's base all rest on the
// claim that origins knows what each commit was for, and none of them would report
// the doubt. They would answer, and answer wrongly.
func TestACommitOriginsDidNotWriteStopsEveryCommandThatReadsTheHistory(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
	git(t, root, "commit", "-q", "-am", "Firenze is a comune")

	for _, command := range [][]string{
		{"set", "-C", root, "person/rossi-giovanni", "name.surname", "Rossi di Firenze"},
		{"unset", "-C", root, "person/rossi-giovanni", "sex"},
		{"create", "-C", root, "person/rossi-annunziata"},
		{"rename", "-C", root, "person/rossi-giovanni", "rossi-giovanni-battista"},
		{"delete", "-C", root, "place/firenze"},
		{"log", "-C", root},
		{"whence", "-C", root, "firenze-civil-deaths-1901"},
		{"blame", "-C", root, "person/rossi-giovanni"},
		{"why", "-C", root, "rossi-giovanni", "name.surname"},
		{"hypothesis", "status", "-C", root},
		{"undo", "-C", root},
		{"snapshot", "-C", root},
		{"adopt", "-C", root},
	} {
		code, _, stderr := invoke(t, command...)
		if code != exitMisuse {
			t.Errorf("origins %s: exit %d, want a refusal", strings.Join(command[:1], " "), code)
			continue
		}
		if !strings.Contains(stderr, "Firenze is a comune") || !strings.Contains(stderr, "origins recover") {
			t.Errorf("origins %s is refused with:\n%s", command[0], stderr)
		}
	}
}

// The other half of the rule, and what keeps it from being a nuisance. A command that
// never asks about the history is never stopped by it — and check is the one that
// matters, because git bisect run origins check leaves HEAD standing where origins
// never put it, deliberately, for as long as the search takes.
func TestACommandThatNeverAsksAboutTheHistoryIsNotStopped(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	git(t, root, "commit", "-q", "--allow-empty", "-m", "Something I committed myself")

	for _, command := range [][]string{
		{"check", root},
		{"load", root},
		{"fmt", "--list", root},
	} {
		if code, _, stderr := invoke(t, command...); code != exitOK {
			t.Errorf("origins %s: exit %d, want it to run: %s", command[0], code, stderr)
		}
	}
}

// Editing a file is the escape hatch the format exists to leave open, and adopting is
// what closes the loop on it: the edit becomes one commit that says it was made by
// hand and names who it was about.
func TestAdoptRecordsAHandEditAsOneCommitNamingWhoItWasAbout(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	edited(t, root, "people/rossi-giovanni/README.md",
		strings.Replace(giovanni, "surname: Rossi", "surname: Rossi di Firenze", 1))

	report := answering[api.Adoption](t, "adopt", "-C", root)
	switch {
	case report.Commit == "":
		t.Fatal("nothing was recorded")
	case len(report.Files) != 1 || report.Files[0] != "people/rossi-giovanni/README.md":
		t.Errorf("the change covers %q", report.Files)
	case len(report.Subjects) != 1 || report.Subjects[0] != "person/rossi-giovanni":
		t.Errorf("the commit is about %q, want the person whose file changed", report.Subjects)
	}
	if got := trailer(t, root, "Origins-Op"); got != "manual.adopt" {
		t.Errorf("the commit is filed under %q, want manual.adopt", got)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("something was left unrecorded:\n%s", left)
	}
	if code, _, _ := invoke(t, "log", "-C", root); code != exitOK {
		t.Error("the commands are still stopped after the edit was adopted")
	}
}

// A hand edit is re-emitted before it is recorded, so that the next real change to
// that file produces a diff of the change and not of the change plus somebody's
// spacing. What is committed is the canonical form, and the file on disk is left
// holding it.
func TestAdoptReEmitsWhatWasNotWrittenAsOriginsWritesIt(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	edited(t, root, "people/rossi-giovanni/README.md",
		"---\nname:\n  given:    Giovanni\n  surname:   'Rossi'\nsex: M\n"+
			"events:\n  - {type: birth, date: 1834, place: firenze}\n---\n")

	report := answering[api.Adoption](t, "adopt", "-C", root)
	if len(report.Emitted) != 1 || report.Emitted[0] != "people/rossi-giovanni/README.md" {
		t.Fatalf("the files origins wrote again are %q", report.Emitted)
	}
	if now := read(t, root, "people/rossi-giovanni/README.md"); now != giovanni {
		t.Errorf("the file reads\n%s\nwant\n%s", now, giovanni)
	}
	if code, _, stderr := invoke(t, "fmt", "--list", root); code != exitOK {
		t.Errorf("what was adopted is still not canonical: %s", stderr)
	}
}

// The escape hatch cannot bypass validation, which is the whole reason adopting is a
// command rather than an instruction to run git commit.
//
// Judging it takes the state the edit was made from, and that is the last commit
// rather than the files in front of origins — they already contain the change, so a
// transaction asking what they contradict now would find every contradiction the edit
// introduced already present and pass the lot.
func TestAdoptWillNotRecordAContradictionSomebodyTypedIn(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	before := git(t, root, "rev-parse", "HEAD")
	edited(t, root, "people/rossi-giovanni/README.md",
		strings.Replace(giovanni, "    place: firenze", "    place: nowhere-at-all", 1))

	code, _, stderr := invoke(t, "adopt", "-C", root)
	if code != exitFindings {
		t.Fatalf("exit %d, want the findings code: %s", code, stderr)
	}
	if !strings.Contains(stderr, "nowhere-at-all") {
		t.Errorf("the refusal does not name what is wrong:\n%s", stderr)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != before {
		t.Error("a contradiction was recorded anyway")
	}
}

// A contradiction that was already there is not the adopter's to answer for. A corpus
// with errors in it is most of what a researcher works on, and refusing to record
// anything until they are all gone would leave no way to work through them.
func TestAdoptLeavesAContradictionThatWasAlreadyThereAlone(t *testing.T) {
	t.Parallel()
	files := theRossis()
	files["people/rossi-giovanni/README.md"] = strings.Replace(giovanni,
		"    place: firenze", "    place: nowhere-at-all", 1)
	root := inARepo(t, files)
	edited(t, root, "people/rossi-giovanni/README.md",
		strings.Replace(files["people/rossi-giovanni/README.md"], "name: Giovanni", "name: Giovanni Battista", 1))

	if code, _, stderr := invoke(t, "adopt", "-C", root); code != exitOK {
		t.Fatalf("exit %d, want the edit recorded: %s", code, stderr)
	}
	if got := trailer(t, root, "Origins-Op"); got != "manual.adopt" {
		t.Errorf("the commit is filed under %q", got)
	}
}

// A dry run says what would happen and leaves the directory as it found it, including
// the file it would have written again.
func TestADryRunOfAdoptRecordsNothingAndWritesNothing(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	loose := "---\nname:\n  given:    Giovanni\n  surname: Rossi\nsex: M\n" +
		"events:\n  - type: birth\n    date: 1834\n    place: firenze\n---\n"
	edited(t, root, "people/rossi-giovanni/README.md", loose)
	before := git(t, root, "rev-parse", "HEAD")

	code, stdout, stderr := invoke(t, "adopt", "-C", root, "--dry-run")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "would record people/rossi-giovanni/README.md") {
		t.Errorf("the dry run says:\n%s", stdout)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != before {
		t.Error("a dry run made a commit")
	}
	if now := read(t, root, "people/rossi-giovanni/README.md"); now != loose {
		t.Error("a dry run wrote the file again")
	}
}

// Nothing to adopt is an answer and not a failure.
func TestAdoptOnACorpusNobodyHasTouchedSaysSo(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, stdout, stderr := invoke(t, "adopt", "-C", root)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "nothing in this corpus has been changed by hand") {
		t.Errorf("origins says %q", stdout)
	}
}

// recover on its own inspects and advises. It never acts, because which resolution is
// right turns on what the commits were for, and that is the one part of the question
// origins is in no position to answer.
func TestRecoverSaysWhatItFoundAndWhatItWouldDo(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	before := git(t, root, "rev-parse", "HEAD")
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
	git(t, root, "commit", "-q", "-am", "Firenze is a comune")

	code, stdout, stderr := invoke(t, "recover", "-C", root)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, want := range []string{
		"Firenze is a comune", "not left this machine", "--rewrite", "--adopt", "--reset",
		"what origins would do",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the report does not mention %q:\n%s", want, stdout)
		}
	}
	if now := git(t, root, "rev-parse", "HEAD"); now == before {
		t.Error("the foreign commit is gone, and recover was asked to do nothing")
	}
}

// The recommendation turns on one checkable fact. A commit somebody else has is one
// that must not be rewritten, because a rewrite hands them a history that no longer
// joins onto theirs.
func TestRecoverRecommendsAdoptingWhatHasLeftTheMachine(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
	git(t, root, "commit", "-q", "-am", "Firenze is a comune")
	git(t, root, "update-ref", "refs/remotes/origin/master", "HEAD")

	report := answering[api.Resolution](t, "recover", "-C", root)
	if !report.Shared {
		t.Error("a commit a remote has reads as one that has not left this machine")
	}
	if report.Advises != "--adopt" {
		t.Errorf("origins would take %s, want --adopt for a shared commit", report.Advises)
	}

	code, _, stderr := invoke(t, "recover", "-C", root, "--rewrite")
	if code != exitMisuse {
		t.Fatalf("exit %d, want rewriting a shared commit refused", code)
	}
	if !strings.Contains(stderr, "left this machine") {
		t.Errorf("the refusal reads:\n%s", stderr)
	}
}

// Adopting resolves the divergence without touching a commit of it: what origins
// writes is a commit of its own saying it has read the others and answers for them.
// The note is the local half, for whoever reads the old commit later.
func TestRecoverAdoptEndsTheRefusalAndNotesWhatItRead(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
	git(t, root, "commit", "-q", "-am", "Firenze is a comune")
	foreign := git(t, root, "rev-parse", "HEAD")

	code, stdout, stderr := invoke(t, "recover", "-C", root, "--adopt")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "Firenze is a comune") {
		t.Errorf("the report does not say what was adopted:\n%s", stdout)
	}
	if got := trailer(t, root, "Origins-Op"); got != "manual.adopt" {
		t.Errorf("the commit is filed under %q, want manual.adopt", got)
	}
	if note := git(t, root, "notes", "show", foreign); !strings.Contains(note, "Reviewed by origins") {
		t.Errorf("the commit that was read carries the note %q", note)
	}
	if still := git(t, root, "cat-file", "-p", foreign); !strings.Contains(still, "Firenze is a comune") {
		t.Error("the commit that was adopted is not there any more")
	}
	if code, _, stderr := invoke(t, "log", "-C", root); code != exitOK {
		t.Errorf("the commands are still stopped after adopting: %s", stderr)
	}
}

// Rewriting replaces each commit with one that says what it is, keeping the sequence
// somebody made their changes in — that sequence is the shape of how they thought,
// and squashing it away would be origins deciding it knew better.
func TestRecoverRewriteMakesEachCommitOriginsOwn(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
	git(t, root, "commit", "-q", "-am", "Firenze is a comune")
	edited(t, root, "people/rossi-giovanni/README.md",
		strings.Replace(giovanni, "surname: Rossi", "surname: Rossi di Firenze", 1))
	git(t, root, "commit", "-q", "-am", "He signed himself Rossi di Firenze")

	code, stdout, stderr := invoke(t, "recover", "-C", root, "--rewrite")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "2 commits rewritten") {
		t.Errorf("the report reads:\n%s", stdout)
	}

	journal := reading(t, "log", "-C", root)
	if len(journal.Entries) != 3 {
		t.Fatalf("the history has %d commits in it, not 3", len(journal.Entries))
	}
	for _, was := range journal.Entries[:2] {
		if was.Tool == "" {
			t.Errorf("%q still reads as somebody else's work", was.Subject)
		}
	}
	if journal.Entries[0].Subject != "He signed himself Rossi di Firenze" {
		t.Errorf("the newest commit is %q, want the last one that was made", journal.Entries[0].Subject)
	}
	if len(journal.Entries[0].Subjects) != 1 || journal.Entries[0].Subjects[0] != "person/rossi-giovanni" {
		t.Errorf("the rewritten commit is about %q", journal.Entries[0].Subjects)
	}
	if now := read(t, root, "people/rossi-giovanni/README.md"); !strings.Contains(now, "Rossi di Firenze") {
		t.Errorf("the corpus reads\n%s\nwant what the commits said", now)
	}
}

// Resetting reduces the problem to one that is already solved: what the commits said
// becomes a change nobody has recorded, which origins adopt handles from there.
func TestRecoverResetLeavesTheWorkInTheTreeForAdoptToTake(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	anchor := git(t, root, "rev-parse", "HEAD")
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
	git(t, root, "commit", "-q", "-am", "Firenze is a comune")

	code, stdout, stderr := invoke(t, "recover", "-C", root, "--reset")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "origins adopt") {
		t.Errorf("the report does not say what to do next:\n%s", stdout)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != anchor {
		t.Errorf("HEAD is %s, want the last commit origins wrote", now)
	}
	if now := read(t, root, "places/firenze/README.md"); !strings.Contains(now, "comune") {
		t.Errorf("what the commit said is gone from the files:\n%s", now)
	}
	if code, _, stderr := invoke(t, "adopt", "-C", root); code != exitOK {
		t.Fatalf("what was left could not be adopted: exit %d: %s", code, stderr)
	}
	if got := trailer(t, root, "Origins-Op"); got != "manual.adopt" {
		t.Errorf("the commit is filed under %q", got)
	}
}

// A corpus somebody has been keeping in git without origins is where every researcher
// who already has one starts. There is nothing to rewrite onto and nothing to take it
// back to, so adopting is the only way in — and origins says so rather than offering
// three and refusing two.
func TestAHistoryOriginsWroteNoPartOfIsOnlyAdopted(t *testing.T) {
	t.Parallel()
	root := byHand(t, theRossis())

	code, stdout, stderr := invoke(t, "recover", "-C", root)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "nothing of origins' own to rewrite onto") {
		t.Errorf("the report reads:\n%s", stdout)
	}
	for _, offered := range []string{"--rewrite", "--reset"} {
		if strings.Contains(stdout, offered) {
			t.Errorf("%s is offered where there is no commit of origins' own to use it:\n%s",
				offered, stdout)
		}
	}
	if code, _, stderr := invoke(t, "recover", "-C", root, "--adopt"); code != exitOK {
		t.Fatalf("adopting: exit %d: %s", code, stderr)
	}
	if code, _, stderr := invoke(t, "set", "-C", root,
		"person/rossi-giovanni", "name.surname", "Rossi di Firenze"); code != exitOK {
		t.Errorf("the corpus still cannot be changed after adopting: %s", stderr)
	}
}

// Adopting a history origins wrote no part of cannot be refused for what is in it.
// There is no earlier state to have been better, origins introduced none of it, and a
// tree with a date disagreeing with another date somewhere in it is every tree worth
// adopting. Refusing would close the only way in and leave the researcher to put right
// by hand, unaided, what origins is refusing to help them look at.
func TestAHistoryOriginsWroteNoPartOfIsAdoptedContradictionsAndAll(t *testing.T) {
	t.Parallel()
	files := theRossis()
	files["people/rossi-giovanni/README.md"] = strings.Replace(giovanni,
		"    place: firenze", "    place: nowhere-at-all", 1)
	root := byHand(t, files)

	if code, _, stderr := invoke(t, "recover", "-C", root, "--adopt"); code != exitOK {
		t.Fatalf("exit %d, want the corpus adopted as it stands: %s", code, stderr)
	}
	if got := trailer(t, root, "Origins-Op"); got != "manual.adopt" {
		t.Errorf("the commit is filed under %q", got)
	}
	// and the guard still stands: what is wrong was there before, and adding to it is
	// still refused
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: city\nwithin: atlantis\n---\n")
	if code, _, stderr := invoke(t, "adopt", "-C", root); code != exitFindings {
		t.Errorf("a fresh contradiction exited %d, want it refused: %s", code, stderr)
	}
}

// The same holds where nothing has been committed at all, which is the other shape a
// corpus origins has never recorded comes in: somebody wrote the files, ran git init,
// and stopped there.
func TestACorpusNothingHasBeenCommittedOfIsAdoptedAsItStands(t *testing.T) {
	t.Parallel()
	files := theRossis()
	files["people/rossi-giovanni/README.md"] = strings.Replace(giovanni,
		"    place: firenze", "    place: nowhere-at-all", 1)
	root := neverCommitted(t, files)

	if code, _, stderr := invoke(t, "adopt", "-C", root); code != exitOK {
		t.Fatalf("exit %d, want the corpus recorded as it stands: %s", code, stderr)
	}
	if got := trailer(t, root, "Origins-Op"); got != "manual.adopt" {
		t.Errorf("the commit is filed under %q", got)
	}
}

// An amend carries the trailers across, so every commit still says origins wrote it
// and only the marker ref noticed anything. There is nothing to record and nowhere to
// reset to: what moved was this clone's record of where it had got to, and moving it
// back is the whole of the repair.
func TestRecoverAfterARewriteOnlyMovesTheRecordBack(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	if code, _, stderr := invoke(t, "set", "-C", root,
		"person/rossi-giovanni", "name.surname", "Rossi di Firenze"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	was := git(t, root, "rev-parse", "HEAD")
	// the date moves so that the amend produces a different commit: the same tree,
	// parent, message and author in the same second is the same sha
	git(t, root, "commit", "-q", "--amend", "--no-edit", "--date=2021-03-04T05:06:07+00:00")
	amended := git(t, root, "rev-parse", "HEAD")
	if amended == was {
		t.Fatal("the amend left the commit exactly as it was")
	}

	report := answering[api.Resolution](t, "recover", "-C", root)
	switch {
	case report.Settled:
		t.Error("a history that was rewritten underneath origins reads as settled")
	case report.Ground != "lost":
		t.Errorf("the ground is %q, want lost", report.Ground)
	case len(report.Foreign) != 0:
		t.Errorf("%d commits read as foreign after an amend, want none", len(report.Foreign))
	case report.Advises != "--adopt":
		t.Errorf("origins would take %q, want --adopt", report.Advises)
	}
	if code, _, stderr := invoke(t, "recover", "-C", root, "--adopt"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != amended {
		t.Errorf("HEAD is %s, want the amended commit at %s: nothing needed recording",
			short(now), short(amended))
	}
	if now := git(t, root, "rev-parse", "refs/origins/head"); now == was {
		t.Error("the record still points at the commit the amend took away")
	}
	if code, _, stderr := invoke(t, "log", "-C", root); code != exitOK {
		t.Errorf("the commands are still stopped: %s", stderr)
	}
}

// The three are different answers to the same question, so origins takes one.
func TestRecoverTakesOneWayAtATime(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, _, stderr := invoke(t, "recover", "-C", root, "--adopt", "--reset")
	if code != exitMisuse {
		t.Fatalf("exit %d, want a refusal", code)
	}
	if !strings.Contains(stderr, "--adopt and --reset") {
		t.Errorf("the refusal reads:\n%s", stderr)
	}
}

// Nothing to recover is an answer, and asking for a resolution anyway is a mistake
// worth naming rather than a commit worth making.
func TestRecoverOnAHistoryOriginsWroteSaysThereIsNothingToDo(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	code, stdout, stderr := invoke(t, "recover", "-C", root)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "nothing to recover") {
		t.Errorf("the report reads:\n%s", stdout)
	}
	if code, _, stderr := invoke(t, "recover", "-C", root, "--adopt"); code != exitMisuse {
		t.Errorf("exit %d, want a refusal: %s", code, stderr)
	}
}

// An unrecorded edit and commits origins did not write are two different things, and
// recording them as one commit would say they were one change. Adopting the edit
// first is the order, and origins says which way round.
func TestRecoverAdoptWillNotSweepUpAnUnrecordedEdit(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	edited(t, root, "places/firenze/README.md", "---\nname: Firenze\ntype: comune\n---\n")
	git(t, root, "commit", "-q", "-am", "Firenze is a comune")
	edited(t, root, "people/rossi-giovanni/README.md",
		strings.Replace(giovanni, "sex: M", "sex: U", 1))

	code, _, stderr := invoke(t, "recover", "-C", root, "--adopt")
	if code != exitMisuse {
		t.Fatalf("exit %d, want a refusal", code)
	}
	if !strings.Contains(stderr, "origins adopt") {
		t.Errorf("the refusal does not say which way round:\n%s", stderr)
	}
}

// A commit somebody else made that renamed a file is replayed as the rename it was, so
// rewriting a history does not leave the record's own starting again at its new name.
func TestRecoverRewriteSaysWhichNamesACommitChanged(t *testing.T) {
	t.Parallel()
	root := inARepo(t, theRossis())
	git(t, root, "mv", "people/rossi-giovanni", "people/rossi-giovanni-1834")
	edited(t, root, "origins.yaml", strings.Replace(read(t, root, "origins.yaml"),
		"rossi-giovanni", "rossi-giovanni-1834", 1))
	git(t, root, "commit", "-q", "-am", "The elder Giovanni, by his birth")

	if code, _, stderr := invoke(t, "recover", "-C", root, "--rewrite"); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got := trailer(t, root, "Origins-Renamed"); got != "person/rossi-giovanni person/rossi-giovanni-1834" {
		t.Errorf("the rewritten commit says the names it changed were %q", got)
	}
}
