package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/browse"
	"github.com/origins-project/origins/internal/model"
)

// defaultAddress is where browse listens unless told otherwise.
const defaultAddress = "127.0.0.1:8100"

// open is how browse shows the researcher the address it serves, which a test replaces.
var open = browse.Open

// served is where browse is listening, for whatever reads the output.
type served struct {
	URL string `json:"url"`
}

// serve runs the pages. A corpus that did not read whole is served all the same, with
// its problems said and nothing offered to change: a corpus with one malformed file is
// exactly when looking at the rest of it helps.
func serve(args []string, stdout, stderr io.Writer) int {
	var (
		listen     string
		noOpen     bool
		public     bool
		passphrase string
		options    browse.Options
	)
	root, asJSON, code := corpusFlags("browse", args, stdout, stderr, func(flags *flag.FlagSet) {
		flags.StringVar(&listen, "listen", "", "the `address` to serve on, "+defaultAddress+" where none is given")
		flags.BoolVar(&noOpen, "no-open", false, "do not open a browser")
		flags.StringVar(&passphrase, "passphrase-file", "", "the `file` holding the passphrase asked for off the loopback, "+
			"in place of "+passphraseVariable)
		flags.BoolVar(&options.Hiding.Private, "hide-private", false, "serve the corpus without what is private, read-only")
		flags.BoolVar(&options.Hiding.Living, "hide-living", false, "serve the corpus without whoever is taken as living, read-only")
		flags.BoolVar(&public, "public", false, "the same as --hide-private and --hide-living together")
		flags.Var((*several)(&options.With), "with", "serve the corpus as though this `hypothesis` were true, read-only")
	})
	if code != exitOK {
		return code
	}
	if public {
		options.Hiding = model.Hiding{Private: true, Living: true}
	}
	address := listen
	if address == "" {
		address = defaultAddress
	}
	if !browse.Loopback(address) {
		given, err := passphraseOf(passphrase)
		if err != nil {
			return misuse(stderr, "%v", err)
		}
		if given == "" {
			return misuse(stderr, "origins serves %s off the loopback only behind a passphrase: set %s "+
				"or give --passphrase-file", root, passphraseVariable)
		}
		options.Passphrase = given
	}

	server := browse.New(root, options)
	defer server.Close()
	for _, problem := range api.Flatten(server.Problems()) {
		fmt.Fprintln(stderr, problem)
	}
	if len(options.With) > 0 && server.Problems() != nil {
		return exitFindings
	}
	listener, err := browse.Listen(address)
	if err != nil {
		return misuse(stderr, "%v", err)
	}
	url := "http://" + listener.Addr().String() + "/"
	if asJSON {
		if err := writeJSON(stdout, served{URL: url}); err != nil {
			return misuse(stderr, "%v", err)
		}
	} else if server.Problems() != nil {
		fmt.Fprintf(stdout, "%s — serving %s, read-only until the problems above are fixed\n", url, root)
	} else if len(options.With) > 0 {
		fmt.Fprintf(stdout, "%s — serving %s with %s supposed true, read-only\n", url, root, (*several)(&options.With))
	} else if options.Hiding.Any() {
		fmt.Fprintf(stdout, "%s — serving %s without %s, read-only\n", url, root, options.Hiding)
	} else {
		fmt.Fprintf(stdout, "%s — serving %s\n", url, root)
	}
	if opens(listen, noOpen) {
		if err := open(url); err != nil {
			fmt.Fprintf(stderr, "origins: could not open a browser: %v\n", err)
		}
	}
	if err := server.Serve(listener); err != nil {
		return misuse(stderr, "%v", err)
	}
	return exitOK
}

// passphraseVariable is where the passphrase is given where no file is.
const passphraseVariable = "ORIGINS_PASSPHRASE"

// passphraseOf is the passphrase a server off the loopback asks for: the file's first
// line where one is named, the environment's otherwise.
func passphraseOf(file string) (string, error) {
	if file == "" {
		return os.Getenv(passphraseVariable), nil
	}
	written, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	first, _, _ := strings.Cut(string(written), "\n")
	return strings.TrimRight(first, "\r"), nil
}

// opens says whether browse opens the address it serves in a browser: unless told not
// to, and never where it was told where to listen, the browser being somebody else's.
func opens(listen string, noOpen bool) bool {
	return !noOpen && listen == ""
}
