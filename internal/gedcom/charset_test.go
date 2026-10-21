package gedcom

import (
	"strings"
	"testing"
	"unicode/utf16"
)

func header(charset, name string) string {
	return "0 HEAD\n1 CHAR " + charset + "\n0 @I1@ INDI\n1 NAME " + name + "\n0 TRLR\n"
}

func TestDecodingTheEncodingsRealExportsUse(t *testing.T) {
	utf16le := func(s string) []byte {
		raw := []byte{0xFF, 0xFE}
		for _, unit := range utf16.Encode([]rune(s)) {
			raw = append(raw, byte(unit), byte(unit>>8))
		}
		return raw
	}
	utf16be := func(s string) []byte {
		raw := []byte{0xFE, 0xFF}
		for _, unit := range utf16.Encode([]rune(s)) {
			raw = append(raw, byte(unit>>8), byte(unit))
		}
		return raw
	}

	for _, c := range []struct {
		what string
		raw  []byte
	}{
		{what: "UTF-8 declared", raw: []byte(header("UTF-8", "José /Cambiasù/"))},
		{what: "UTF-8 with a byte order mark", raw: append([]byte{0xEF, 0xBB, 0xBF},
			header("UTF-8", "José /Cambiasù/")...)},
		{what: "UTF-16 little endian", raw: utf16le(header("UNICODE", "José /Cambiasù/"))},
		{what: "UTF-16 big endian", raw: utf16be(header("UNICODE", "José /Cambiasù/"))},
		// The bytes Windows-1252 actually uses for é and ù, which are not UTF-8 at all.
		{what: "Windows-1252 declared as ANSI",
			raw: []byte(header("ANSI", "Jos\xE9 /Cambias\xF9/"))},
		// Code page 437, which the DOS-era exporters call IBMPC, puts them elsewhere again.
		{what: "code page 437 declared as IBMPC",
			raw: []byte(header("IBMPC", "Jos\x82 /Cambias\x97/"))},
	} {
		text, err := decode(c.raw)
		if err != nil {
			t.Errorf("%s: %v", c.what, err)
			continue
		}
		if !strings.Contains(text, "José") {
			t.Errorf("%s: decoded to %q", c.what, text)
		}
	}
}

// The line this package draws, and the reason it is drawn here rather than further out.
// Repairing structure loses nothing; guessing at an encoding spells every accented
// surname in the file wrong, quietly, in a corpus meant to outlive the tool. A refusal
// is recoverable by re-exporting. A corrupted import is not recoverable at all.
func TestAnEncodingOriginsCannotReadIsRefusedRatherThanGuessed(t *testing.T) {
	for _, c := range []struct {
		what string
		raw  []byte
	}{
		{what: "ANSEL with a byte no table agrees on",
			raw: append([]byte(header("ANSEL", "Jos")), 0xEB, 'e', '\n')},
		{what: "an encoding origins has never heard of",
			raw: []byte(header("MACINTOSH", "Jose"))},
		{what: "UNICODE with no byte order mark to say which",
			raw: []byte(header("UNICODE", "Jose"))},
		{what: "bytes that are not UTF-8 and do not say what they are",
			raw: append([]byte("0 HEAD\n0 @I1@ INDI\n1 NAME Jos"), 0xE9, '\n')},
	} {
		if _, err := decode(c.raw); err == nil {
			t.Errorf("%s was accepted", c.what)
		}
	}
}
