package api

import (
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/origins-project/origins/internal/history"
)

// The release build sets these with
// -ldflags "-X github.com/origins-project/origins/internal/api.release=0.1.0 -X ...api.commit=<sha>".
// Any other build leaves them empty and is described by what Go recorded in it.
var release, commit string

// Version is the version of origins: the release it was built as, or dev.
var Version = versionOf(release, buildInfo())

// About is what origins version reports: the build, and the git it would run.
type About struct {
	Version string `json:"version"`
	// Commit is the commit it was built from, where that is known.
	Commit string `json:"commit,omitempty"`
	// Modified is a build from a working tree with changes no commit holds.
	Modified bool   `json:"modified,omitempty"`
	Go       string `json:"go"`
	// Git is the git found on the path, nil where there is none.
	Git *Git `json:"git"`
	// GitProblem is why no git was found, for a reader.
	GitProblem string `json:"-"`
}

// Git is where a git is and the version it says it is.
type Git struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// Describe is what origins is and the git it would run.
func Describe() *About {
	about := &About{Version: Version, Go: runtime.Version()}
	about.Commit, about.Modified = commitOf(commit, buildInfo())
	if path, version, err := history.Git(); err != nil {
		about.GitProblem = err.Error()
	} else {
		about.Git = &Git{Path: path, Version: version}
	}
	return about
}

func buildInfo() *debug.BuildInfo {
	info, _ := debug.ReadBuildInfo()
	return info
}

// released is a version a module is tagged with, v and all.
var released = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`)

// pseudo is the version Go makes up for a commit no tag names, which looks released.
var pseudo = regexp.MustCompile(`[0-9]{14}-[0-9a-f]{12}$`)

// versionOf is the version a release build was given, or else the tag Go recorded
// when it built a module at one: from go install at a version, or from a checkout
// standing on a tag with nothing changed. Anything else is dev.
func versionOf(given string, info *debug.BuildInfo) string {
	if given != "" {
		return strings.TrimPrefix(given, "v")
	}
	if info == nil {
		return "dev"
	}
	v := info.Main.Version
	if !released.MatchString(v) || pseudo.MatchString(v) {
		return "dev"
	}
	return strings.TrimPrefix(v, "v")
}

// commitOf is the commit a release build was given, or the one Go recorded from the
// checkout it built in, and whether that checkout had changes.
func commitOf(given string, info *debug.BuildInfo) (sha string, modified bool) {
	if given != "" || info == nil {
		return given, false
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			sha = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	return sha, modified
}
