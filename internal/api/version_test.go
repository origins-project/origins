package api

import (
	"runtime/debug"
	"testing"
)

func TestTheVersionIsTheReleaseOrDev(t *testing.T) {
	t.Parallel()
	built := func(version string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Path: "github.com/origins-project/origins", Version: version}}
	}
	for _, c := range []struct {
		name, given string
		info        *debug.BuildInfo
		want        string
	}{
		{"given by the release build", "0.2.0", built("(devel)"), "0.2.0"},
		{"given with its v", "v0.2.0", nil, "0.2.0"},
		{"installed at a tag", "", built("v0.2.0"), "0.2.0"},
		{"a release candidate", "", built("v0.3.0-rc.1"), "0.3.0-rc.1"},
		{"built after a tag", "", built("v0.2.1-0.20261019120000-55159be0a1b2"), "dev"},
		{"built where no tag is", "", built("v0.0.0-20261019120000-55159be0a1b2"), "dev"},
		{"built at a tag with changes", "", built("v0.2.0+dirty"), "dev"},
		{"built where no checkout is", "", built("(devel)"), "dev"},
		{"built with nothing recorded", "", nil, "dev"},
	} {
		if got := versionOf(c.given, c.info); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTheCommitIsGivenOrRecorded(t *testing.T) {
	t.Parallel()
	info := &debug.BuildInfo{Settings: []debug.BuildSetting{
		{Key: "vcs", Value: "git"},
		{Key: "vcs.revision", Value: "55159be"},
		{Key: "vcs.modified", Value: "true"},
	}}
	if sha, modified := commitOf("", info); sha != "55159be" || !modified {
		t.Errorf("recorded: %q, %v", sha, modified)
	}
	if sha, modified := commitOf("2094a85", info); sha != "2094a85" || modified {
		t.Errorf("given: %q, %v", sha, modified)
	}
	if sha, modified := commitOf("", nil); sha != "" || modified {
		t.Errorf("nothing recorded: %q, %v", sha, modified)
	}
}
