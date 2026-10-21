package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// THIRD_PARTY_NOTICES is what the build is made of, so that a dependency added, or one
// whose version changed, is a licence in the file or a test that fails.
func TestTheNoticesAreTheBuild(t *testing.T) {
	root, err := top()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "internal/browse/app/dist/.vite/license.json")); err != nil {
		t.Skip("the application is not built")
	}
	written, err := notices(root)
	if err != nil {
		t.Fatal(err)
	}
	there, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(there, written) {
		t.Errorf("%s is not what the build is made of: run go run ./tools/notices", file)
	}
}
