package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// aRepo is an empty repository with an identity of its own, so what a test asserts
// about authorship is what the test set and not what the machine happens to say.
func aRepo(t *testing.T) *Repo {
	t.Helper()
	dir := t.TempDir()

	work := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	repo := &Repo{dir: work}
	for _, command := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "Rosa Bianchi"},
		{"config", "user.email", "rosa@example.invalid"},
	} {
		if _, err := repo.git(command...); err != nil {
			t.Fatal(err)
		}
	}
	return repo
}

func write(t *testing.T, repo *Repo, relative, content string) {
	t.Helper()
	path := filepath.Join(repo.dir, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func anEntry() Commit {
	return Commit{
		Subject: "Add death event to Giovanni Rossi",
		Tool:    "origins 0.2.0",
		Op:      "event.add",
		Entities: []model.EntityRef{
			{Kind: model.KindPerson, Slug: "rossi-giovanni"},
			{Kind: model.KindSource, Slug: "firenze-civil-deaths-1901"},
		},
		Renamed: []Renaming{{
			From: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni-bis"},
			To:   model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		}},
		Events:     []string{"death"},
		Sources:    []model.CitationRef{{Source: "firenze-civil-deaths-1901", Anchor: "1103-17"}},
		Corrects:   "0123456789abcdef0123456789abcdef01234567",
		Hypothesis: "giovanni-died-1901",
		ProposedBy: "transcriber",
	}
}

// The trailers have to be trailers as git understands them, not lines that merely look
// like them, because everything downstream reads them back through git: log --grep for
// whence, and the trailer itself for why.
func TestACommitCarriesEveryTrailer(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")

	sha, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Fatalf("sha = %q", sha)
	}

	message, err := repo.git("log", "-1", "--format=%B")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimRight(anEntry().Message(), "\n")
	if strings.TrimRight(message, "\n") != want {
		t.Errorf("message came back as:\n%s\nwant:\n%s", message, want)
	}

	// git's own parse of the block, which is the reason for choosing trailers
	subjects, err := repo.git("log", "-1", "--format=%(trailers:key=Origins-Subject,valueonly,separator=%x20)")
	if err != nil {
		t.Fatal(err)
	}
	if subjects != "person/rossi-giovanni source/firenze-civil-deaths-1901" {
		t.Errorf("git read the subjects as %q", subjects)
	}
	event, err := repo.git("log", "-1", "--format=%(trailers:key=Origins-Event,valueonly)")
	if err != nil {
		t.Fatal(err)
	}
	if event != "death" {
		t.Errorf("git read the event as %q", event)
	}
}

// The trailers come out in the order they are documented in, so two commits recording
// the same kind of change read the same way and a diff of two messages is about what
// changed rather than about where the lines landed.
func TestTheTrailersKeepTheirDocumentedOrder(t *testing.T) {
	t.Parallel()
	want := strings.Join([]string{
		"Add death event to Giovanni Rossi",
		"",
		"Origins-Tool: origins 0.2.0",
		"Origins-Op: event.add",
		"Origins-Subject: person/rossi-giovanni",
		"Origins-Subject: source/firenze-civil-deaths-1901",
		"Origins-Renamed: person/rossi-giovanni-bis person/rossi-giovanni",
		"Origins-Event: death",
		"Origins-Source: firenze-civil-deaths-1901#1103-17",
		"Origins-Corrects: 0123456789abcdef0123456789abcdef01234567",
		"Origins-Hypothesis: giovanni-died-1901",
		"Origins-Proposed-By: transcriber",
	}, "\n") + "\n"
	if got := anEntry().Message(); got != want {
		t.Errorf("message =\n%s\nwant:\n%s", got, want)
	}
}

// Bulk work is one commit and not thousands, and a dirty tree beside a change is never
// swept into it. Both come from naming the paths rather than committing the index.
func TestACommitTakesOnlyTheFilesItIsGiven(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	write(t, repo, "people/bianchi-maria/README.md", "---\nname: Maria Bianchi\n---\n")
	if _, err := repo.git("add", "--", "people/bianchi-maria/README.md"); err != nil {
		t.Fatal(err) // staged by someone else, and none of this change's business
	}

	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	files, err := repo.git("show", "--name-only", "--format=", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if files != "people/rossi-giovanni/README.md" {
		t.Errorf("the commit contains %q", files)
	}
	left, err := repo.git("status", "--porcelain")
	if err != nil {
		t.Fatal(err)
	}
	if left != "A  people/bianchi-maria/README.md" {
		t.Errorf("the other file came out of the index as %q", left)
	}
}

// An import names every record it wrote, and a message naming five thousand is longer
// than Linux lets one argument be, so it has to reach git some other way.
func TestACommitNamingThousandsOfEntitiesIsMade(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	entry := Commit{Subject: "Import a large tree", Tool: "origins 0.2.0", Op: "corpus.import"}
	for i := range 5000 {
		entry.Entities = append(entry.Entities, model.EntityRef{
			Kind: model.KindPerson, Slug: model.Slug(fmt.Sprintf("rossi-giovanni-%04d", i))})
	}
	if len(entry.Message()) <= 128<<10 {
		t.Fatalf("the message is %d bytes, which one argument could carry", len(entry.Message()))
	}

	if _, err := repo.Commit(entry, []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}
	subjects, err := repo.git("log", "-1", "--format=%(trailers:key=Origins-Subject,valueonly)")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(subjects, "\n") + 1; got != 5000 {
		t.Errorf("the commit names %d entities, want 5000", got)
	}
}

// A shared corpus records who asserted what, so the commit is the user's and not the
// tool's. Nothing here passes an author: git is left to say who it is configured as.
func TestACommitIsAuthoredAsTheUser(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	if _, err := repo.Commit(anEntry(), []string{"people/rossi-giovanni/README.md"}); err != nil {
		t.Fatal(err)
	}

	who, err := repo.git("log", "-1", "--format=%an <%ae>")
	if err != nil {
		t.Fatal(err)
	}
	if who != "Rosa Bianchi <rosa@example.invalid>" {
		t.Errorf("authored as %q", who)
	}
}

// Every optional trailer left out, and the block still has to read back cleanly.
func TestACommitWithNothingOptionalStillCarriesToolAndOp(t *testing.T) {
	t.Parallel()
	entry := Commit{Subject: "Correct a surname", Tool: "origins 0.2.0", Op: "person.edit"}
	want := "Correct a surname\n\nOrigins-Tool: origins 0.2.0\nOrigins-Op: person.edit\n"
	if got := entry.Message(); got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}

// History that looks complete and is not is worse than history that is missing, since
// only the second kind gets noticed.
func TestACommitOriginsCouldNotReadBackIsRefused(t *testing.T) {
	t.Parallel()
	repo := aRepo(t)
	write(t, repo, "people/rossi-giovanni/README.md", "---\nname: Giovanni Rossi\n---\n")
	paths := []string{"people/rossi-giovanni/README.md"}

	for _, refused := range []struct {
		why   string
		entry Commit
	}{
		{"no subject", Commit{Tool: "origins 0.2.0", Op: "person.add"}},
		{"no tool", Commit{Subject: "Add Giovanni", Op: "person.add"}},
		{"no operation", Commit{Subject: "Add Giovanni", Tool: "origins 0.2.0"}},
		{"a subject over two lines", Commit{
			Subject: "Add Giovanni\nand everyone else", Tool: "origins 0.2.0", Op: "person.add",
		}},
		{"a person's name given up to a place", Commit{
			Subject: "Adopt people/firenze/README.md", Tool: "origins 0.2.0", Op: "manual.adopt",
			Renamed: []Renaming{{
				From: model.EntityRef{Kind: model.KindPerson, Slug: "firenze"},
				To:   model.EntityRef{Kind: model.KindPlace, Slug: "firenze"},
			}},
		}},
	} {
		if _, err := repo.Commit(refused.entry, paths); err == nil {
			t.Errorf("a commit with %s was allowed", refused.why)
		}
	}
	if _, err := repo.Commit(anEntry(), nil); err == nil {
		t.Error("a commit with no files in it was allowed")
	}
}

func TestOpeningSomewhereThatIsNotARepositoryFails(t *testing.T) {
	t.Parallel()
	if _, err := Open(t.TempDir()); err == nil {
		t.Fatal("a directory outside any repository was opened")
	}
}

// A rule somebody wrote for other reasons — every PDF ignored, say — would have git
// leave a scan out of a commit that says the corpus was recorded, and the scan would be
// on this disk and in no clone. What the corpus is not made of is left to the rules:
// a note beside it, or a file with a dot in front of its name.
func TestAFileTheCorpusIsMadeOfIsNeverLeftOutOfACommit(t *testing.T) {
	t.Parallel()
	for name, commit := range map[string]func(*Repo) (string, error){
		"record": func(r *Repo) (string, error) { return r.Record(anEntry()) },
		"commit": func(r *Repo) (string, error) { return r.Commit(anEntry(), []string{"."}) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			repo := aRepo(t)
			write(t, repo, ".gitignore", "*.pdf\nnotes.txt\n.DS_Store\n")
			write(t, repo, "origins.yaml", "project:\n  name: rossi\norigins:\n  format-version: 1\n")
			write(t, repo, "people/rossi-giovanni/README.md", aRecord())
			write(t, repo, "people/rossi-giovanni/photos/atto.pdf", "a scan")
			write(t, repo, "people/rossi-giovanni/.DS_Store", "finder's")
			write(t, repo, "notes.txt", "somebody's own")

			_, err := commit(repo)
			if err == nil || !strings.Contains(err.Error(), "people/rossi-giovanni/photos/atto.pdf, by .gitignore:1:*.pdf") {
				t.Fatalf("the ignored scan was answered with %v", err)
			}
			if strings.Contains(err.Error(), "notes.txt") || strings.Contains(err.Error(), ".DS_Store") {
				t.Errorf("a file the corpus is not made of was refused: %v", err)
			}
			if _, err := repo.git("rev-parse", "--verify", "-q", "HEAD"); err == nil {
				t.Error("a commit was made all the same")
			}

			// a file git has been given is one no rule can take out of a commit
			if _, err := repo.git("add", "--force", "--", "people/rossi-giovanni/photos/atto.pdf"); err != nil {
				t.Fatal(err)
			}
			sha, err := commit(repo)
			if err != nil {
				t.Fatalf("the corpus was refused once the scan was given to git: %v", err)
			}
			held, err := repo.git("show", "--name-only", "--format=", sha)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(held, "atto.pdf") || strings.Contains(held, "notes.txt") {
				t.Errorf("the commit holds\n%s", held)
			}
		})
	}
}
