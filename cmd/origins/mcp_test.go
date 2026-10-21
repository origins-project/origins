package main

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/origins-project/origins/internal/api"
)

// asOrigins set in the environment makes the test binary origins itself, so that a
// test can start it as a client would.
const asOrigins = "ORIGINS_TEST_AS_ORIGINS"

// origins mcp speaks the protocol on its own stdin and stdout, to a client that starts
// it as a process.
func TestMCPServesOverStdio(t *testing.T) {
	t.Parallel()
	command := exec.Command(os.Args[0], "mcp", "-C", t.TempDir(), "--as", "research-agent")
	command.Env = append(os.Environ(), asOrigins+"=1")
	ctx := context.Background()
	session, err := sdk.NewClient(&sdk.Implementation{Name: "agent", Version: "1"}, nil).
		Connect(ctx, &sdk.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if served := session.InitializeResult().ServerInfo; served.Name != "origins" || served.Version != api.Version {
		t.Errorf("the server says it is %s %s", served.Name, served.Version)
	}
	var names []string
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, tool.Name)
	}
	if !slices.Contains(names, "propose") || len(names) != 14 {
		t.Errorf("the tools are %v", names)
	}
}
