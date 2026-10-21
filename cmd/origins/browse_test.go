package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowseOpensABrowserOnlyWhereItChoseTheAddress(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		listen string
		noOpen bool
		opens  bool
	}{
		{"", false, true},
		{"", true, false},
		{"127.0.0.1:9000", false, false},
		{"127.0.0.1:9000", true, false},
	} {
		if got := opens(c.listen, c.noOpen); got != c.opens {
			t.Errorf("opens(%q, %v) = %v", c.listen, c.noOpen, got)
		}
	}
}

func TestBrowseServesOffTheLoopbackOnlyBehindAPassphrase(t *testing.T) {
	t.Setenv(passphraseVariable, "")
	code, _, stderr := invoke(t, "browse", "--listen", "0.0.0.0:0", "--no-open", t.TempDir())
	if code != exitMisuse || !strings.Contains(stderr, passphraseVariable) {
		t.Errorf("browse on every interface without a passphrase exited %d:\n%s", code, stderr)
	}
	missing := filepath.Join(t.TempDir(), "passphrase")
	code, _, stderr = invoke(t, "browse", "--listen", "0.0.0.0:0", "--no-open", "--passphrase-file", missing, t.TempDir())
	if code != exitMisuse || !strings.Contains(stderr, "passphrase") {
		t.Errorf("browse with a passphrase file that is not there exited %d:\n%s", code, stderr)
	}
}

func TestBrowseServesNoSuppositionItCannotBuild(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	code, _, stderr := invoke(t, "browse", "--no-open", "--listen", "127.0.0.1:0", "--with", "no-such-hypothesis", root)
	if code == exitOK || !strings.Contains(stderr, "no-such-hypothesis") {
		t.Errorf("browse supposing a hypothesis there is not exited %d:\n%s", code, stderr)
	}
}
