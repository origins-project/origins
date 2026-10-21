package history

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/model"
)

func person(slug model.Slug) model.EntityRef {
	return model.EntityRef{Kind: model.KindPerson, Slug: slug}
}

// change is a commit about some entities, carrying the three parts no commit can do
// without and nothing else.
func change(subject, op string, entities ...model.EntityRef) Commit {
	return Commit{Subject: subject, Tool: "origins 0.2.0", Op: op, Entities: entities}
}

func record(t *testing.T, repo *Repo, c Commit) string {
	t.Helper()
	write(t, repo, "corpus.md", c.Subject+"\n")
	sha, err := repo.Commit(c, []string{"corpus.md"})
	if err != nil {
		t.Fatal(err)
	}
	return sha
}

// byHand is a commit origins had nothing to do with.
func byHand(t *testing.T, repo *Repo, relative, message string) {
	t.Helper()
	write(t, repo, relative, message+"\n")
	for _, command := range [][]string{
		{"add", "--", relative},
		{"commit", "-q", "-m", message, "--", relative},
	} {
		if _, err := repo.git(command...); err != nil {
			t.Fatal(err)
		}
	}
}

func subjects(entries []Entry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Subject)
	}
	return out
}

func logging(t *testing.T, repo *Repo, q Query) []Entry {
	t.Helper()
	entries, err := repo.Log(q)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

// The writer and the reader share one description of a commit, so anything origins can
// record it can be asked about afterwards. A trailer that went out and did not come
// back is history that looks complete and is not.
func TestACommitReadsBackAsItWasWritten(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	wrote := anEntry()
	sha := record(t, repo, wrote)

	entries := logging(t, repo, Query{})
	if len(entries) != 1 {
		t.Fatalf("%d commits came back, not 1", len(entries))
	}
	read := entries[0]
	if read.Sha != sha {
		t.Errorf("read %s, committed %s", read.Sha, sha)
	}
	if !reflect.DeepEqual(read.Commit, wrote) {
		t.Errorf("read back\n\t%+v\nwrote\n\t%+v", read.Commit, wrote)
	}
	if read.Author != "Rosa Bianchi" {
		t.Errorf("authored by %q", read.Author)
	}
	if since := time.Since(read.When); since < 0 || since > time.Hour {
		t.Errorf("committed %v ago, which is not just now", since)
	}
	if read.Foreign() {
		t.Error("a commit origins wrote reads as foreign")
	}
}

func TestABodyReadsBackBetweenTheSubjectAndTheTrailers(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	wrote := anEntry()
	wrote.Body = "Read from rossi.ged, written by Gramps 5.2.\n\nExtension tags:\n_FREL https://example.org/frel"
	record(t, repo, wrote)

	entries := logging(t, repo, Query{})
	if len(entries) != 1 {
		t.Fatalf("%d commits came back, not 1", len(entries))
	}
	if !reflect.DeepEqual(entries[0].Commit, wrote) {
		t.Errorf("read back\n\t%+v\nwrote\n\t%+v", entries[0].Commit, wrote)
	}
}

func TestTheWholeHistoryComesBackNewestFirst(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	var made []string
	for _, subject := range []string{"First", "Second", "Third"} {
		made = append(made, record(t, repo, change(subject, "person.create", person("rossi-giovanni"))))
	}

	entries := logging(t, repo, Query{})
	got := subjects(entries)
	want := []string{"Third", "Second", "First"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for i, read := range entries {
		if was := made[len(made)-1-i]; read.Sha != was {
			t.Errorf("%q came back as %s, committed as %s", read.Subject, read.Sha, was)
		}
	}
}

// A slug is a whole name. Asking after one must not answer with another that merely
// begins the same way, which is exactly what a discriminated slug looks like.
func TestALogAboutOneEntityLeavesTheRestOut(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("She was born", "person.create", person("bianchi-maria")))
	record(t, repo, change("His grandson too", "person.create", person("rossi-giovanni-1834")))
	record(t, repo, change("He died", "events.add", person("rossi-giovanni")))

	got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person("rossi-giovanni")}}))
	want := []string{"He died", "He was born"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Renaming is the one thing that breaks a history in two, since the slug is the
// identity and there is no second name underneath it. A rename names both ends, so
// following them back is what makes a person's history the whole of it.
func TestAnEntityIsFoundUnderEveryNameItHasHad(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("A grandson of the same name",
		"person.rename", person("rossi-giovanni"), person("rossi-giovanni-1834")))
	record(t, repo, change("He died", "events.add", person("rossi-giovanni-1834")))
	record(t, repo, change("The family called him that",
		"person.rename", person("rossi-giovanni-1834"), person("rossi-giovanni-il-vecchio")))
	record(t, repo, change("Somebody else entirely", "person.create", person("bianchi-maria")))

	want := []string{
		"The family called him that",
		"He died",
		"A grandson of the same name",
		"He was born",
	}
	for _, name := range []model.Slug{"rossi-giovanni", "rossi-giovanni-1834", "rossi-giovanni-il-vecchio"} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person(name)}}))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("asked after %s: got %q, want %q", name, got, want)
		}
	}
}

// A rename frees the name it leaves, and whoever is given it next is somebody else.
// Following the rename from that name would hand each of them the other's history.
func TestANameGivenToSomebodyElseIsTheirsFromThen(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("The grandson was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("He died in 1975", "events.add", person("rossi-giovanni")))
	record(t, repo, change("Named for his year", "person.rename",
		person("rossi-giovanni"), person("rossi-giovanni-1902")))
	record(t, repo, change("His grandfather was found", "person.create", person("rossi-giovanni")))
	record(t, repo, change("The grandfather married", "events.add", person("rossi-giovanni")))
	record(t, repo, change("The grandson married", "events.add", person("rossi-giovanni-1902")))

	for _, one := range []struct {
		asking model.Slug
		want   []string
	}{
		{"rossi-giovanni", []string{"The grandfather married", "His grandfather was found"}},
		{"rossi-giovanni-1902", []string{
			"The grandson married", "Named for his year", "He died in 1975", "The grandson was born",
		}},
	} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person(one.asking)}}))
		if !reflect.DeepEqual(got, one.want) {
			t.Errorf("asked after %s: got %q, want %q", one.asking, got, one.want)
		}
	}
}

func TestANameDeletedAndGivenAgainIsSomebodyNew(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("A child nobody could find", "person.create", person("rossi-infante")))
	record(t, repo, change("Never there", "person.delete", person("rossi-infante")))
	record(t, repo, change("The register had her after all", "person.create", person("rossi-infante")))

	got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person("rossi-infante")}}))
	if want := []string{"The register had her after all"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// An undo puts every name back where it stood, so a rename taken back gives the old
// name back to the record that had it, and a create taken back leaves the name free for
// somebody else.
func TestAnUndoGivesANameBackToWhoeverHadIt(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	undoing := func(subject, sha string, entities ...model.EntityRef) Commit {
		c := change(subject, reversal, entities...)
		c.Corrects = sha
		return c
	}
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	renamed := record(t, repo, change("Too soon", "person.rename",
		person("rossi-giovanni"), person("rossi-giovanni-1902")))
	record(t, repo, undoing("Not yet", renamed, person("rossi-giovanni"), person("rossi-giovanni-1902")))
	record(t, repo, change("He died", "events.add", person("rossi-giovanni")))
	created := record(t, repo, change("A cousin", "person.create", person("rossi-marco")))
	record(t, repo, undoing("No cousin", created, person("rossi-marco")))
	record(t, repo, change("Another cousin", "person.create", person("rossi-marco")))

	for _, one := range []struct {
		asking model.Slug
		want   []string
	}{
		{"rossi-giovanni", []string{"He died", "Not yet", "Too soon", "He was born"}},
		{"rossi-giovanni-1902", []string{"He died", "Not yet", "Too soon", "He was born"}},
		{"rossi-marco", []string{"Another cousin"}},
	} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person(one.asking)}}))
		if !reflect.DeepEqual(got, one.want) {
			t.Errorf("asked after %s: got %q, want %q", one.asking, got, one.want)
		}
	}
}

// A merge says two records were one man all along, so his history is the history of
// both, asked after by the name that stayed or by the one that went.
func TestAMergedRecordHasTheHistoryOfBoth(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("A second record of him", "person.create", person("rossi-giovanni-bis")))
	record(t, repo, change("The second has his death", "events.add", person("rossi-giovanni-bis")))
	record(t, repo, change("The first has his marriage", "events.add", person("rossi-giovanni")))
	record(t, repo, change("They were one man", "person.merge",
		person("rossi-giovanni-bis"), person("rossi-giovanni")))
	record(t, repo, change("He emigrated", "events.add", person("rossi-giovanni")))
	record(t, repo, change("Somebody else entirely", "person.create", person("bianchi-maria")))

	want := []string{
		"He emigrated",
		"They were one man",
		"The first has his marriage",
		"The second has his death",
		"A second record of him",
		"He was born",
	}
	for _, name := range []model.Slug{"rossi-giovanni", "rossi-giovanni-bis"} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person(name)}}))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("asked after %s: got %q, want %q", name, got, want)
		}
	}
}

// A merge taken back leaves two records again, each with the history it had before.
func TestAnUndoneMergeLeavesTwoHistories(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("Another Giovanni", "person.create", person("rossi-giovanni-bis")))
	record(t, repo, change("The other one died", "events.add", person("rossi-giovanni-bis")))
	merged := record(t, repo, change("Too soon", "person.merge",
		person("rossi-giovanni-bis"), person("rossi-giovanni")))
	undo := change("Not the same man", reversal, person("rossi-giovanni-bis"), person("rossi-giovanni"))
	undo.Corrects = merged
	record(t, repo, undo)
	record(t, repo, change("He married", "events.add", person("rossi-giovanni")))

	for _, one := range []struct {
		asking model.Slug
		want   []string
	}{
		{"rossi-giovanni", []string{"He married", "Not the same man", "Too soon", "He was born"}},
		{"rossi-giovanni-bis", []string{
			"Not the same man", "Too soon", "The other one died", "Another Giovanni",
		}},
	} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person(one.asking)}}))
		if !reflect.DeepEqual(got, one.want) {
			t.Errorf("asked after %s: got %q, want %q", one.asking, got, one.want)
		}
	}
}

// Merging a partner merges the families that become one, and the family that stays has
// the history of both marriages that turned out to be one.
func TestAFamilyMergedWithItsPartnerHasTheHistoryOfBoth(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	family := func(slug model.Slug) model.EntityRef {
		return model.EntityRef{Kind: model.KindFamily, Slug: slug}
	}
	record(t, repo, change("They married", "family.create", family("bianchi-maria--rossi-giovanni")))
	record(t, repo, change("The other record has a wedding", "family.create",
		family("bianchi-maria--rossi-giovanni-bis")))
	record(t, repo, change("And a child", "children.add", family("bianchi-maria--rossi-giovanni-bis")))
	record(t, repo, change("One man, one marriage", "person.merge",
		person("rossi-giovanni-bis"), person("rossi-giovanni"),
		family("bianchi-maria--rossi-giovanni-bis"), family("bianchi-maria--rossi-giovanni")))

	want := []string{"One man, one marriage", "And a child", "The other record has a wedding", "They married"}
	for _, asking := range []model.Slug{"bianchi-maria--rossi-giovanni", "bianchi-maria--rossi-giovanni-bis"} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{family(asking)}}))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("asked after %s: got %q, want %q", asking, got, want)
		}
	}
}

func TestACommitOriginsDidNotWriteIsForeign(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	byHand(t, repo, "corpus.md", "Fixed a typo")

	entries := logging(t, repo, Query{})
	if len(entries) != 2 {
		t.Fatalf("%d commits came back, not 2", len(entries))
	}
	if !entries[0].Foreign() {
		t.Errorf("%q reads as origins' own work", entries[0].Subject)
	}
	if entries[1].Foreign() {
		t.Errorf("%q reads as foreign", entries[1].Subject)
	}
}

// What rests on a source is the question worth asking when a source turns out to be
// wrong, so it is asked of the source and answered by every citation in it.
func TestASourceAnswersForEveryCitationInIt(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	cited := func(subject string, source model.Slug, anchor string, grade model.Grade) Commit {
		c := change(subject, "events.add", person("rossi-giovanni"))
		c.Sources = []model.CitationRef{{Source: source, Anchor: anchor, Grade: grade}}
		return c
	}
	record(t, repo, cited("From the register", "firenze-deaths", "1103-17", model.Grade{}))
	record(t, repo, cited("From the same register", "firenze-deaths", "1104-02", model.Grade{}))
	record(t, repo, cited("From another register altogether", "firenze-deaths-1901", "88", model.Grade{}))
	record(t, repo, cited("From the same entry, graded", "firenze-deaths", "1103-17", model.Grade{Source: 'O', Information: 'S', Evidence: 'I'}))
	record(t, repo, cited("From an entry beginning the same", "firenze-deaths", "1103-170", model.Grade{}))

	got := subjects(logging(t, repo, Query{Sources: []model.Slug{"firenze-deaths"}}))
	want := []string{"From an entry beginning the same", "From the same entry, graded",
		"From the same register", "From the register"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the source answered %q, want %q", got, want)
	}

	// an entry is the one entry whatever it was graded as, where it was cited and
	// where it is asked about
	want = []string{"From the same entry, graded", "From the register"}
	for _, grade := range []model.Grade{{}, {Source: 'O', Information: 'P', Evidence: 'D'}} {
		one := []model.CitationRef{{Source: "firenze-deaths", Anchor: "1103-17", Grade: grade}}
		got = subjects(logging(t, repo, Query{Citations: one}))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("the citation graded %q answered %q, want %q", grade, got, want)
		}
	}
}

func TestARepositoryWithNothingCommittedHasNoHistory(t *testing.T) {
	t.Parallel()
	entries, err := aRepo(t).Log(Query{})
	if err != nil {
		t.Fatalf("a repository nobody has committed to is not a failure: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("%d commits came back", len(entries))
	}
}

// One commit nobody can parse must not take away the answer to the question that was
// asked, any more than one malformed file stops a corpus loading.
func TestATrailerNobodyCanReadIsReportedAndTheRestComeBack(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	byHand(t, repo, "corpus.md", "Something somebody edited\n\nOrigins-Tool: origins 0.2.0\n"+
		"Origins-Op: person.create\nOrigins-Subject: whoever that was")

	entries, err := repo.Log(Query{})
	if err == nil {
		t.Fatal("a trailer that is not an entity reference was read without complaint")
	}
	if !strings.Contains(err.Error(), "Origins-Subject") {
		t.Errorf("the complaint does not name the trailer: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("%d commits came back, not 2", len(entries))
	}
	if got := entries[0].Entities; len(got) != 0 {
		t.Errorf("the unreadable reference was kept as %v", got)
	}
	if entries[1].Subject != "He was born" {
		t.Errorf("the readable commit came back as %q", entries[1].Subject)
	}
}

func TestALimitKeepsTheNewest(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	for _, subject := range []string{"First", "Second", "Third"} {
		record(t, repo, change(subject, "person.create", person("rossi-giovanni")))
	}

	got := subjects(logging(t, repo, Query{Limit: 2}))
	want := []string{"Third", "Second"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A rename can name entities of two kinds at once, because a family's name is made of
// its partners and changes when one of them does. The two are not the same entity,
// though, and the family has a history of its own from before the marriage was ever
// written down.
func TestAFamilyRenamedWithItsPartnerKeepsItsOwnHistory(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	family := func(slug model.Slug) model.EntityRef {
		return model.EntityRef{Kind: model.KindFamily, Slug: slug}
	}
	record(t, repo, change("They married", "family.create", family("bianchi-maria--rossi-giovanni")))
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("A grandson of the same name", "person.rename",
		person("rossi-giovanni"), person("rossi-giovanni-1834"),
		family("bianchi-maria--rossi-giovanni"), family("bianchi-maria--rossi-giovanni-1834")))
	record(t, repo, change("She was born", "person.create", person("bianchi-maria")))

	for _, one := range []struct {
		asking model.EntityRef
		want   []string
	}{
		{person("rossi-giovanni-1834"), []string{"A grandson of the same name", "He was born"}},
		{family("bianchi-maria--rossi-giovanni-1834"), []string{"A grandson of the same name", "They married"}},
		{family("bianchi-maria--rossi-giovanni"), []string{"A grandson of the same name", "They married"}},
	} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{one.asking}}))
		if !reflect.DeepEqual(got, one.want) {
			t.Errorf("asked after %s: got %q, want %q", one.asking, got, one.want)
		}
	}
}

// handing is a commit filed under something other than a rename that gave names up all
// the same, as an adoption of a file renamed by hand does.
func handing(subject, op string, renamed []Renaming, entities ...model.EntityRef) Commit {
	c := change(subject, op, entities...)
	c.Renamed = renamed
	return c
}

// A rename made in the files and adopted is followed as a rename is, and the name it
// leaves is free for somebody else exactly as the command's would be.
func TestANameGivenUpInAnyCommitIsFollowed(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("The grandson was born", "person.create", person("rossi-giovanni")))
	record(t, repo, handing("Adopt people/rossi-giovanni-1902/README.md, edited by hand", "manual.adopt",
		[]Renaming{{From: person("rossi-giovanni"), To: person("rossi-giovanni-1902")}},
		person("rossi-giovanni-1902")))
	record(t, repo, change("The grandson married", "events.add", person("rossi-giovanni-1902")))
	record(t, repo, change("His grandfather was found", "person.create", person("rossi-giovanni")))

	for _, one := range []struct {
		asking model.Slug
		want   []string
	}{
		{"rossi-giovanni", []string{"His grandfather was found"}},
		{"rossi-giovanni-1902", []string{
			"The grandson married", "Adopt people/rossi-giovanni-1902/README.md, edited by hand", "The grandson was born",
		}},
	} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person(one.asking)}}))
		if !reflect.DeepEqual(got, one.want) {
			t.Errorf("asked after %s: got %q, want %q", one.asking, got, one.want)
		}
	}
}

// A name given up to one already held makes the two records one, which is what an
// accepted hypothesis that merges says, and an undo of it makes them two again.
func TestANameGivenUpToOneHeldIsAMerge(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("A second record of him", "person.create", person("rossi-giovanni-bis")))
	accepted := record(t, repo, handing("Accept one-man", "hypothesis.accept",
		[]Renaming{{From: person("rossi-giovanni-bis"), To: person("rossi-giovanni")}},
		person("rossi-giovanni-bis"), person("rossi-giovanni")))
	record(t, repo, change("He emigrated", "events.add", person("rossi-giovanni")))

	want := []string{"He emigrated", "Accept one-man", "A second record of him", "He was born"}
	for _, name := range []model.Slug{"rossi-giovanni", "rossi-giovanni-bis"} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person(name)}}))
		if !reflect.DeepEqual(got, want) {
			t.Errorf("asked after %s: got %q, want %q", name, got, want)
		}
	}

	undo := change("Two men after all", reversal, person("rossi-giovanni-bis"), person("rossi-giovanni"))
	undo.Corrects = accepted
	record(t, repo, undo)
	got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{person("rossi-giovanni-bis")}}))
	want = []string{"Two men after all", "Accept one-man", "A second record of him"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after the undo: got %q, want %q", got, want)
	}
}

// What has become of a record since a commit is a walk begun there, so a record renamed
// once by command and once by hand is found under the name it has now, one merged away
// is found in the record it went into, and one deleted is gone.
func TestSinceSaysWhatBecameOfEachRecord(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	record(t, repo, change("The grandson was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("A second record of her", "person.create", person("bianchi-maria-bis")))
	record(t, repo, change("She was born", "person.create", person("bianchi-maria")))
	record(t, repo, change("A child nobody could find", "person.create", person("rossi-infante")))
	base := record(t, repo, change("Nobody changed", "person.create", person("conti-luigi")))

	record(t, repo, change("Named for his year", "person.rename",
		person("rossi-giovanni"), person("rossi-giovanni-1902")))
	record(t, repo, change("His grandfather was found", "person.create", person("rossi-giovanni")))
	record(t, repo, handing("Adopt a file renamed by hand", "manual.adopt",
		[]Renaming{{From: person("rossi-giovanni-1902"), To: person("rossi-giovanni-il-giovane")}}))
	record(t, repo, handing("Accept one-woman", "hypothesis.accept",
		[]Renaming{{From: person("bianchi-maria-bis"), To: person("bianchi-maria")}}))
	record(t, repo, change("Never there", "person.delete", person("rossi-infante")))

	fates, err := repo.Since(base, []model.EntityRef{
		person("rossi-giovanni"), person("bianchi-maria-bis"), person("bianchi-maria"),
		person("rossi-infante"), person("conti-luigi"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[model.EntityRef]Fate{
		person("rossi-giovanni"):    {Now: person("rossi-giovanni-il-giovane")},
		person("bianchi-maria-bis"): {Now: person("bianchi-maria"), Merged: true},
		person("bianchi-maria"):     {Now: person("bianchi-maria")},
		person("conti-luigi"):       {Now: person("conti-luigi")},
	}
	if !reflect.DeepEqual(fates, want) {
		t.Errorf("got %v, want %v", fates, want)
	}
	if !fates[person("rossi-infante")].Gone() {
		t.Error("a record deleted since is not gone")
	}
}

// A tag renamed names its two words and then every record carrying it, which are
// touched and renamed to nothing, so none of them answers for another's history.
func TestATagRenamedRenamesNoRecord(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	tag := func(word string) model.EntityRef { return model.EntityRef{Kind: model.KindTag, Slug: model.Slug(word)} }
	base := record(t, repo, change("He was born", "person.create", person("rossi-giovanni")))
	record(t, repo, change("She was born", "person.create", person("bianchi-maria")))
	record(t, repo, change("Rename tag/To do to Done", "tags.rename",
		tag("To do"), tag("Done"), person("bianchi-maria"), person("rossi-giovanni"), model.TheProject))

	for _, one := range []struct {
		asking model.EntityRef
		want   []string
	}{
		{person("rossi-giovanni"), []string{"Rename tag/To do to Done", "He was born"}},
		{person("bianchi-maria"), []string{"Rename tag/To do to Done", "She was born"}},
		{tag("Done"), []string{"Rename tag/To do to Done"}},
		{tag("To do"), []string{"Rename tag/To do to Done"}},
	} {
		got := subjects(logging(t, repo, Query{Entities: []model.EntityRef{one.asking}}))
		if !reflect.DeepEqual(got, one.want) {
			t.Errorf("asked after %s: got %q, want %q", one.asking, got, one.want)
		}
	}
	fates, err := repo.Since(base, []model.EntityRef{person("rossi-giovanni")})
	if err != nil {
		t.Fatal(err)
	}
	if now := fates[person("rossi-giovanni")].Now; now != person("rossi-giovanni") {
		t.Errorf("he goes by %v since a tag was renamed", now)
	}
}
