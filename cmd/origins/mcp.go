package main

import (
	"context"
	"io"
	"os"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/origins-project/origins/internal/mcp"
)

// serveAgent serves the corpus to an agent over stdio until the client goes, which
// internal/mcp says more about.
func serveAgent(args []string, stdout, stderr io.Writer) int {
	var root, as string
	flags := newFlags("mcp", stderr)
	flags.StringVar(&root, "C", ".", "the `corpus` to serve")
	flags.StringVar(&as, "as", "", "the `name` a hypothesis is proposed by, in place of the client's own")
	if code := parse(flags, args, stdout); code != exitOK {
		return code
	}
	if flags.NArg() > 0 {
		return misuse(stderr, "that is not the shape of it: origins mcp [-C dir] [--as name]")
	}
	if strings.ContainsAny(as, "\n\r") {
		return misuse(stderr, "--as names whoever proposes on one line")
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return misuse(stderr, "%s is no directory to serve", root)
	}
	if err := mcp.New(root, as).Run(context.Background(), &sdk.StdioTransport{}); err != nil {
		return misuse(stderr, "%v", err)
	}
	return exitOK
}
