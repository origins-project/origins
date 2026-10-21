// Package mcp serves a corpus to an agent over the Model Context Protocol.
//
// It is a transport over internal/api, as the command line and the pages are: every
// tool is a command that reads, answering with the report its --json gives, and one
// that writes, propose, which records a hypothesis stated whole and nothing else. An
// agent proposes; accepting, rejecting and every other change are the researcher's,
// and there is no tool for them.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/model"
)

// New is a server answering about the corpus at root. Whoever it proposes a
// hypothesis for is as, or where that is empty, the name the client gave itself.
func New(root, as string) *sdk.Server {
	server := sdk.NewServer(&sdk.Implementation{Name: "origins", Version: api.Version}, nil)

	reads(server, "index", "Every record with what it is called and what ties it to the others, without the prose",
		func(in Supposing) (*api.Index, error) { return api.IndexOf(root, in.supposed()) })
	reads(server, "read", "One record as its file's frontmatter says it, with the paths that can be edited",
		func(in Naming) (*api.Record, error) { return api.Read(root, in.Entity) })
	reads(server, "check", "The findings of check, with the hypotheses named supposed true",
		func(in Checking) (*api.Validation, error) { return api.Check(root, in.supposed(), in.ErrorsOnly) })
	reads(server, "todo", "The research still to do",
		func(in Supposing) (*api.ToDo, error) { return api.Todo(root, in.supposed()) })
	reads(server, "ancestors", "Whom a person descends from, a generation at a time",
		func(in Descent) (*api.Lineage, error) {
			return api.Ancestors(root, in.supposed(), in.Person, in.Depth)
		})
	reads(server, "descendants", "Who descends from a person, a generation at a time",
		func(in Descent) (*api.Lineage, error) {
			return api.Descendants(root, in.supposed(), in.Person, in.Depth)
		})
	reads(server, "kinship", "What the first person is to the second, and whom it runs through",
		func(in Kin) (*api.Kin, error) { return api.Kinship(root, in.supposed(), in.First, in.Second) })
	reads(server, "log", "What has been recorded, about the corpus or one record in it",
		func(in Logging) (*api.Journal, error) { return api.Log(root, in.Entity, in.Limit) })
	reads(server, "whence", "Every conclusion drawn from a source or a citation in it",
		func(in Whence) (*api.Journal, error) { return api.Whence(root, in.Source, in.Limit) })
	reads(server, "blame", "When each fact of a record appeared, and on what",
		func(in Blaming) (*api.Provenance, error) { return api.Blame(root, in.Entity, in.Field) })
	reads(server, "why", "One fact, back to the change that wrote it and the argument for it",
		func(in Tracing) (*api.Reason, error) { return api.Why(root, in.Entity, in.Path) })
	reads(server, "hypothesis_status", "Every hypothesis, or the one named, and whether the corpus still holds "+
		"each one accepted",
		func(in Standing) (*[]api.Holding, error) {
			holdings, err := api.Status(root, hypothesis(in.Hypothesis))
			switch {
			case holdings != nil:
				return &holdings, err
			case err != nil:
				return nil, err
			}
			return &[]api.Holding{}, nil
		})
	reads(server, "schema", "The JSON Schema of a kind of record, of the manifest as project, or of every one",
		func(in Describing) (*json.RawMessage, error) {
			described, err := api.Schema(in.Kind)
			if err != nil {
				return nil, err
			}
			return &described, nil
		})

	sdk.AddTool(server, &sdk.Tool{
		Name:        "propose",
		Description: "Record a hypothesis stated whole, its file's frontmatter and body, as an open hypothesis",
		Annotations: &sdk.ToolAnnotations{DestructiveHint: new(false), OpenWorldHint: new(false)},
	}, func(_ context.Context, request *sdk.CallToolRequest, in Proposing) (*sdk.CallToolResult, any, error) {
		slug, err := model.ParseName(model.KindHypothesis, in.Slug)
		if err != nil {
			return nil, nil, err
		}
		by := proposer(as, request.Session)
		if by == "" {
			return nil, nil, errors.New("nobody is named to have proposed it: origins mcp --as names who")
		}
		asking := api.Asking{Root: root, Command: "mcp propose", DryRun: in.DryRun}
		return answered(api.ProposeStated(asking, slug, []byte(in.Hypothesis), by))
	})
	return server
}

// proposer is who a hypothesis is proposed for: as, or the client's own name.
func proposer(as string, session *sdk.ServerSession) string {
	if as = strings.TrimSpace(as); as != "" {
		return as
	}
	if session == nil {
		return ""
	}
	if params := session.InitializeParams(); params != nil && params.ClientInfo != nil {
		return strings.TrimSpace(params.ClientInfo.Name)
	}
	return ""
}

// reads adds a tool that changes nothing, answering what ask answers.
func reads[In, Out any](server *sdk.Server, name, description string, ask func(In) (*Out, error)) {
	sdk.AddTool(server, &sdk.Tool{
		Name:        name,
		Description: description,
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: new(false)},
	}, func(_ context.Context, _ *sdk.CallToolRequest, in In) (*sdk.CallToolResult, any, error) {
		return answered(ask(in))
	})
}

// answered is a report as structured content, with whatever went wrong after it said
// beside it, or the refusal there is in its place.
func answered[T any](report *T, err error) (*sdk.CallToolResult, any, error) {
	if report == nil {
		if err == nil {
			err = errors.New("there is no answer")
		}
		return nil, nil, refusal(err)
	}
	if err == nil {
		return nil, report, nil
	}
	written, marshalled := json.Marshal(report)
	if marshalled != nil {
		return nil, nil, marshalled
	}
	return &sdk.CallToolResult{Content: []sdk.Content{
		&sdk.TextContent{Text: string(written)},
		&sdk.TextContent{Text: said(err)},
	}}, report, nil
}

// refusal is err in the words the command line says it in.
func refusal(err error) error {
	var unsettled *history.Unsettled
	if errors.As(err, &unsettled) {
		return errors.New(unsettled.Error() + "\n\norigins recover says what can be done about it")
	}
	return errors.New(said(err))
}

// said is every part of err on a line of its own, the problems of a corpus that did
// not read whole before the refusal they led to.
func said(err error) string {
	var lines []string
	unreadable, rest := api.Without[*api.Unreadable](err)
	for _, one := range unreadable {
		lines = append(lines, one.Problems...)
		lines = append(lines, one.Refusal)
	}
	if rest != nil {
		lines = append(lines, rest.Error())
	}
	return strings.Join(lines, "\n")
}

// Supposing is a question put as though the hypotheses named were true.
type Supposing struct {
	With []string `json:"with,omitempty" jsonschema:"hypotheses to suppose true"`
}

func (s Supposing) supposed() []model.Slug {
	out := make([]model.Slug, len(s.With))
	for i, one := range s.With {
		out[i] = hypothesis(one)
	}
	return out
}

// hypothesis is a hypothesis's slug, written bare or as hypothesis/slug.
func hypothesis(name string) model.Slug {
	return model.Slug(strings.TrimPrefix(name, string(model.KindHypothesis)+"/"))
}

// Naming is what `read` asks: one record, or the manifest.
type Naming struct {
	Entity string `json:"entity" jsonschema:"kind/slug, or project"`
}

// Checking is what `check` asks.
type Checking struct {
	Supposing
	ErrorsOnly bool `json:"errors-only,omitempty" jsonschema:"leave out warnings"`
}

// Descent is what `ancestors` and `descendants` ask.
type Descent struct {
	Supposing
	Person string `json:"person" jsonschema:"a person's slug"`
	Depth  int    `json:"depth,omitempty" jsonschema:"the generations to go, all of them where nought"`
}

// Kin is what `kinship` asks: how two people are related.
type Kin struct {
	Supposing
	First  string `json:"first" jsonschema:"a person's slug"`
	Second string `json:"second" jsonschema:"a person's slug"`
}

// Logging is what `log` asks.
type Logging struct {
	Entity string `json:"entity,omitempty" jsonschema:"kind/slug, the whole corpus where empty"`
	Limit  int    `json:"limit,omitempty" jsonschema:"the newest entries to give, all of them where nought"`
}

// Whence is what `whence` asks: what was concluded from a source.
type Whence struct {
	Source string `json:"source" jsonschema:"a source's slug, or source#anchor for one citation"`
	Limit  int    `json:"limit,omitempty" jsonschema:"the newest entries to give, all of them where nought"`
}

// Blaming is what `blame` asks.
type Blaming struct {
	Entity string `json:"entity" jsonschema:"kind/slug"`
	Field  string `json:"field,omitempty" jsonschema:"one field, every one where empty"`
}

// Tracing is what `why` asks: the argument behind one field.
type Tracing struct {
	Entity string `json:"entity" jsonschema:"kind/slug"`
	Path   string `json:"path" jsonschema:"a field's path, events[0].date"`
}

// Standing is what `hypothesis_status` asks.
type Standing struct {
	Hypothesis string `json:"hypothesis,omitempty" jsonschema:"a hypothesis's slug, every one where empty"`
}

// Describing is what `schema` asks.
type Describing struct {
	Kind string `json:"kind,omitempty" jsonschema:"a kind, project or common, every one where empty"`
}

// Proposing is what `propose` asks: a hypothesis stated whole.
type Proposing struct {
	Slug       string `json:"slug" jsonschema:"the hypothesis's slug"`
	Hypothesis string `json:"hypothesis" jsonschema:"the hypothesis's file, YAML frontmatter and markdown body"`
	DryRun     bool   `json:"dry-run,omitempty" jsonschema:"say what would be recorded, and record nothing"`
}
