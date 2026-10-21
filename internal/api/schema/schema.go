// Package schema is a JSON Schema of each kind of record and of the manifest, spelled
// as the corpus spells them, for whoever writes a record without origins to hand: a
// page sending an edit's value, an agent proposing one, an editor completing a field.
//
// It describes and does not decide. What the reader refuses is decided by the reader,
// check does not validate against it, and the tests in internal/api hold the two
// together: everything the emitter writes is valid, every path the schema names is one
// a path can reach, and a file the reader refuses for its shape is refused here too.
package schema

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"
)

//go:embed *.json
var files embed.FS

// Common is the schema the others take their shared definitions from.
const Common = "common"

// Names is every schema there is, by the word origins schema takes: the seven kinds in
// the singular, project for the manifest, and common.
func Names() []string {
	entries, _ := fs.ReadDir(files, ".")
	var out []string
	for _, e := range entries {
		out = append(out, strings.TrimSuffix(e.Name(), ".json"))
	}
	slices.Sort(out)
	return out
}

// Of is the schema of one name, as its file is written.
func Of(name string) ([]byte, error) {
	if !slices.Contains(Names(), name) {
		return nil, fmt.Errorf("there is no schema of %q; there is one of %s", name, strings.Join(Names(), ", "))
	}
	return files.ReadFile(name + ".json")
}
