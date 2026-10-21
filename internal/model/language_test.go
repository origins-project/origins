package model

import "testing"

func TestALanguageIsReadAsATag(t *testing.T) {
	for in, want := range map[string]Language{
		"":      "",
		"en":    "en",
		" de ":  "de",
		"EN":    "en",
		"en-gb": "en-GB",
		"en_GB": "en-GB",
		"grc":   "grc",
	} {
		got, err := ParseLanguage(in)
		if err != nil || got != want {
			t.Errorf("ParseLanguage(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"english", "xx", "e", "en GB"} {
		if got, err := ParseLanguage(in); err == nil {
			t.Errorf("ParseLanguage(%q) = %q, want an error", in, got)
		}
	}
}
