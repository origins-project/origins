package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/api"
)

// A release build reports the version and commit it is given, and any other build
// reports dev, or the tag it was built at with nothing changed, and the commit Go
// recorded from the checkout. Both are real builds of this command, since what Go
// records is what is under test.
func TestABuildSaysWhatItIs(t *testing.T) {
	t.Parallel()
	head := strings.TrimSpace(gitIn(t, "rev-parse", "HEAD"))
	var tags []string
	for tag := range strings.FieldsSeq(gitIn(t, "tag", "--points-at", "HEAD")) {
		if regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+`).MatchString(tag) {
			tags = append(tags, strings.TrimPrefix(tag, "v"))
		}
	}

	t.Run("released", func(t *testing.T) {
		t.Parallel()
		about := built(t, "-X github.com/origins-project/origins/internal/api.release=0.2.0 "+
			"-X github.com/origins-project/origins/internal/api.commit=0123456789abcdef")
		if about.Version != "0.2.0" || about.Commit != "0123456789abcdef" || about.Modified {
			t.Errorf("%+v", about)
		}
	})
	t.Run("not released", func(t *testing.T) {
		t.Parallel()
		about := built(t, "")
		atTag := !about.Modified && slices.Contains(tags, about.Version)
		if about.Version != "dev" && !atTag || about.Commit != head {
			t.Errorf("%+v, at %s tagged %v", about, head, tags)
		}
	})
}

// built is what origins version --json says of a build made with the linker flags
// given.
func built(t *testing.T, ldflags string) api.About {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "origins.exe")
	build := exec.Command("go", "build", "-ldflags", ldflags, "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, err := exec.Command(binary, "version", "--json").Output()
	if err != nil {
		t.Fatalf("origins version --json: %v", err)
	}
	var about api.About
	if err := json.Unmarshal(out, &about); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if about.Go == "" || about.Git == nil || about.Git.Version == "" {
		t.Errorf("no Go or git in %s", out)
	}
	return about
}

func gitIn(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return string(out)
}
