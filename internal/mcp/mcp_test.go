package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/origins-project/origins/internal/api"
)

// TestMain points git at configuration of the package's own, so that nothing a test
// asserts depends on how the machine running it is set up.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "origins-test")
	if err != nil {
		panic(err)
	}
	config := filepath.Join(dir, "git-config")
	identity := "[user]\n\tname = Rosa Bianchi\n\temail = rosa@example.invalid\n" +
		"[core]\n\tautocrlf = false\n"
	if err := os.WriteFile(config, []byte(identity), 0o644); err != nil {
		panic(err)
	}
	os.Setenv("GIT_CONFIG_GLOBAL", config)
	os.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(dir, "no-such-system-config"))
	// git packs no test's repository after a commit, none living long enough to need
	// it, and commits as nobody it has not been told of, rather than guessing at an
	// identity from the machine, which on some machines it finds
	os.Setenv("GIT_CONFIG_COUNT", "2")
	os.Setenv("GIT_CONFIG_KEY_0", "maintenance.auto")
	os.Setenv("GIT_CONFIG_VALUE_0", "false")
	os.Setenv("GIT_CONFIG_KEY_1", "user.useConfigOnly")
	os.Setenv("GIT_CONFIG_VALUE_1", "true")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// gitFixture is a copy of testgit, which tools/mkgitfixture.sh builds; a test skips
// where it has not been.
func gitFixture(t *testing.T) string {
	t.Helper()
	built := filepath.Join("..", "..", "testgit")
	if _, err := os.Stat(built); err != nil {
		t.Skip("no git fixture; run tools/mkgitfixture.sh")
	}
	root := filepath.Join(t.TempDir(), "corpus")
	if err := os.CopyFS(root, os.DirFS(built)); err != nil {
		t.Fatal(err)
	}
	return root
}

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

// connected is a client calling itself client connected to a server on root, started
// as as.
func connected(t *testing.T, root, as, client string) *sdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverSide, clientSide := sdk.NewInMemoryTransports()
	served, err := New(root, as).Connect(ctx, serverSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { served.Close() })
	session, err := sdk.NewClient(&sdk.Implementation{Name: client, Version: "1"}, nil).
		Connect(ctx, clientSide, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// calling calls a tool and hands back what it answered.
func calling(t *testing.T, session *sdk.ClientSession, tool string, arguments map[string]any) *sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: tool, Arguments: arguments})
	if err != nil {
		t.Fatalf("%s: %v", tool, err)
	}
	return result
}

// answer calls a tool that has to answer, and reads the answer as a T.
func answer[T any](t *testing.T, session *sdk.ClientSession, tool string, arguments map[string]any) T {
	t.Helper()
	result := calling(t, session, tool, arguments)
	if result.IsError {
		t.Fatalf("%s refused: %s", tool, text(t, result))
	}
	var report T
	written, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(written, &report); err != nil {
		t.Fatalf("%s: %v\n%s", tool, err, written)
	}
	return report
}

// text is the text a tool answered with.
func text(t *testing.T, result *sdk.CallToolResult) string {
	t.Helper()
	var out []string
	for _, content := range result.Content {
		if one, ok := content.(*sdk.TextContent); ok {
			out = append(out, one.Text)
		}
	}
	return strings.Join(out, "\n")
}

// refused calls a tool that has to refuse, and hands back the words it refused in.
func refused(t *testing.T, session *sdk.ClientSession, tool string, arguments map[string]any) string {
	t.Helper()
	result := calling(t, session, tool, arguments)
	if !result.IsError {
		t.Fatalf("%s answered where it should have refused: %s", tool, text(t, result))
	}
	return text(t, result)
}

// louisa1844 is a hypothesis stated whole: Louisa born in 1844, on the
// census's word.
const louisa1844 = "---\n%sconfidence: possible\nsupports: [census-1861-england#f214]\n" +
	"edits:\n  - op: set\n    entity: person/verdier-louisa\n    path: events[0].date\n    value: \"1844\"\n" +
	"---\n\n# Was Louisa born in 1844?\n\nThe census gives her age as 17.\n"

// The server offers the commands that read and propose, and nothing else that writes.
func TestTheToolsAreTheReadsAndPropose(t *testing.T) {
	t.Parallel()
	session := connected(t, t.TempDir(), "", "agent")
	var names []string
	for tool, err := range session.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, tool.Name)
		if reads := tool.Annotations != nil && tool.Annotations.ReadOnlyHint; reads == (tool.Name == "propose") {
			t.Errorf("%s says it reads only: %v", tool.Name, reads)
		}
	}
	slices.Sort(names)
	want := []string{"ancestors", "blame", "check", "descendants", "hypothesis_status", "index", "kinship",
		"log", "propose", "read", "schema", "todo", "whence", "why"}
	if !slices.Equal(names, want) {
		t.Errorf("the tools are %v, want %v", names, want)
	}
}

// What an agent is there for, end to end: it reads the index and a record, proposes a
// hypothesis, and asks check about its own proposal, nobody in between.
func TestAnAgentReadsProposesAndChecksItsOwnProposal(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	base := git(t, root, "rev-parse", "HEAD")
	session := connected(t, root, "", "transcriber")

	index := answer[api.Index](t, session, "index", nil)
	if !slices.ContainsFunc(index.People, func(p api.IndexedPerson) bool { return p.Ref == "person/verdier-louisa" }) {
		t.Fatal("the index has no Louisa")
	}
	record := answer[api.Record](t, session, "read", map[string]any{"entity": "person/verdier-louisa"})
	if record.File != "people/verdier-louisa/README.md" {
		t.Fatalf("read %s", record.File)
	}

	proposal := answer[api.Proposal](t, session, "propose", map[string]any{
		"slug": "louisa-in-1844", "hypothesis": fmt.Sprintf(louisa1844, ""),
	})
	if proposal.Base != base || proposal.ProposedBy != "transcriber" || proposal.Commit == "" {
		t.Errorf("the proposal is %+v", proposal)
	}
	if touched := git(t, root, "show", "--name-only", "--format=", proposal.Commit); touched !=
		"hypotheses/louisa-in-1844/README.md" {
		t.Errorf("the proposal touched %q", touched)
	}
	if trailer := git(t, root, "show", "-s", "--format=%(trailers:key=Origins-Proposed-By,valueonly)",
		proposal.Commit); trailer != "transcriber" {
		t.Errorf("the commit says it was proposed by %q", trailer)
	}

	checked := answer[api.Validation](t, session, "check", map[string]any{"with": []string{"louisa-in-1844"}})
	if len(checked.Read) != 1 || checked.Read[0].Hypothesis != "louisa-in-1844" {
		t.Errorf("check read %+v", checked.Read)
	}
	stated := answer[api.Record](t, session, "read", map[string]any{"entity": "hypothesis/louisa-in-1844"})
	if stated.Fields["proposed-by"] != "transcriber" {
		t.Errorf("the hypothesis says it was proposed by %v", stated.Fields["proposed-by"])
	}
}

// The server's --as names the proposer, whatever the client calls itself.
func TestTheServersIdentityNamesTheProposer(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	session := connected(t, root, "research-agent", "transcriber")
	proposal := answer[api.Proposal](t, session, "propose", map[string]any{
		"slug": "louisa-in-1844", "hypothesis": fmt.Sprintf(louisa1844, ""),
	})
	if proposal.ProposedBy != "research-agent" {
		t.Errorf("proposed by %q", proposal.ProposedBy)
	}
}

// An agent never names itself: neither beside the hypothesis nor in it.
func TestAnAgentCannotSayWhoProposed(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	head := git(t, root, "rev-parse", "HEAD")
	session := connected(t, root, "", "transcriber")

	refused(t, session, "propose", map[string]any{
		"slug": "louisa-in-1844", "hypothesis": fmt.Sprintf(louisa1844, ""), "proposed-by": "researcher",
	})
	words := refused(t, session, "propose", map[string]any{
		"slug": "louisa-in-1844", "hypothesis": fmt.Sprintf(louisa1844, "proposed-by: researcher\n"),
	})
	if !strings.Contains(words, "says who proposed it") {
		t.Errorf("refused with %q", words)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != head {
		t.Error("a refused proposal was recorded")
	}
}

// A refusal is in the words the command uses.
func TestARefusalIsTheCommands(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	session := connected(t, root, "", "agent")
	_, err := api.Read(root, "person/nobody-at-all")
	if err == nil {
		t.Fatal("api read somebody who is not there")
	}
	if words := refused(t, session, "read", map[string]any{"entity": "person/nobody-at-all"}); words != err.Error() {
		t.Errorf("refused with %q, the command with %q", words, err)
	}
}

// Every question answers on the fixture, as its --json does.
func TestEveryReadAnswers(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	session := connected(t, root, "", "agent")
	for tool, arguments := range map[string]map[string]any{
		"todo":              nil,
		"ancestors":         {"person": "verdier-louisa", "depth": 2},
		"descendants":       {"person": "lang-anna-maria"},
		"kinship":           {"first": "verdier-louisa", "second": "lang-anna-maria"},
		"log":               {"entity": "person/verdier-louisa", "limit": 2},
		"whence":            {"source": "census-1861-england"},
		"blame":             {"entity": "person/verdier-louisa"},
		"why":               {"entity": "person/verdier-louisa", "path": "events[0].date"},
		"hypothesis_status": nil,
		"schema":            {"kind": "person"},
	} {
		result := calling(t, session, tool, arguments)
		if result.IsError {
			t.Errorf("%s refused: %s", tool, text(t, result))
			continue
		}
		if result.StructuredContent == nil {
			t.Errorf("%s answered with no content", tool)
		}
	}
}
