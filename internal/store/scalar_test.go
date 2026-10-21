package store

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// awkward is every value the quoting rules exist for, and a few they should leave
// alone. Each one is put through the parser twice below, once as a field of its own and
// once inside a flow sequence, because the two contexts end a plain scalar differently.
var awkward = []string{
	"", " ", "  x  ", "x ", " x",
	"1834", "1750/51", "12 MAR 1834", "ABT 1834", "BET 1830 AND 1835",
	"0x10", "1_0", "12:30", "2001-12-14", "2001-12-14 21:59:43", ".inf", "-.inf", ".nan",
	"true", "True", "TRUE", "false", "yes", "no", "on", "off", "y", "n",
	"null", "Null", "NULL", "~",
	`"Nancy" OBrien`, "O'Brien", "d'Angelo", "'quoted'",
	"- dashed", "-dash", "?query", "mid ? mark", "Q5?", ":colon", "key: value",
	"trailing:", "a: b: c",
	"hash # comment", "#hash", "a#b", "&anchor", "*alias", "!tag", "|pipe", ">fold",
	"%directive", "@at", "`backtick`", "{brace}", "[bracket]", "a, b", "a,b",
	"tab\there", "line\nbreak", "carriage\rreturn", "bell\a",
	"next" + string(rune(0x85)) + "line", "line" + string(rune(0x2028)) + "separator",
	"paragraph" + string(rune(0x2029)) + "separator", string(rune(0xfeff)) + "marked",
	"byte" + string(rune(0xfeff)) + "mark", "escape" + string(rune(0x9b)) + "sequence",
	"not" + string(rune(0xfffe)) + "a character",
	"José", "Müller", "北京", "Åse",
	"a very ordinary name", "borgo-san-lorenzo", "ADF/SL/B/4",
	"parish-san-lorenzo-baptisms#1834-042", "43.7696, 11.2558",
}

// TestQuotingSurvivesTheParser is the test the quoting rules answer to. Working out
// from the YAML specification which values need quoting is guesswork; putting each one
// through the parser origins reads with is not.
func TestQuotingSurvivesTheParser(t *testing.T) {
	for _, value := range awkward {
		var field struct {
			V string `yaml:"v"`
		}
		if err := yaml.Unmarshal([]byte("v: "+quote(value)+"\n"), &field); err != nil {
			t.Errorf("%q emitted as %s: %v", value, quote(value), err)
			continue
		}
		if field.V != value {
			t.Errorf("%q emitted as %s and came back as %q", value, quote(value), field.V)
		}

		var list struct {
			V []string `yaml:"v"`
		}
		flow := "v: [" + quoteFlow(value) + "]\n"
		if err := yaml.Unmarshal([]byte(flow), &list); err != nil {
			t.Errorf("%q emitted in a list as %s: %v", value, quoteFlow(value), err)
			continue
		}
		if len(list.V) != 1 || list.V[0] != value {
			t.Errorf("%q emitted in a list as %s and came back as %q", value, quoteFlow(value), list.V)
		}
	}
}

// Prose is written as a block where it can be and quoted where it cannot, and either
// way it has to come back as it went: each awkward value alone, as a line among others,
// and long enough to fold.
func TestProseSurvivesTheParser(t *testing.T) {
	for _, value := range awkward {
		for _, text := range []string{
			value, value + "\n", "first line\n" + value + "\nlast line",
			strings.Repeat("a word ", 20) + value + strings.Repeat(" and more", 10),
		} {
			head, lines := proseText(text)
			written := "v: " + head + "\n"
			for _, line := range lines {
				written += indentStep + line + "\n"
			}
			var field struct {
				V string `yaml:"v"`
			}
			if err := yaml.Unmarshal([]byte(written), &field); err != nil {
				t.Errorf("%q written as %q: %v", text, written, err)
			} else if text != "" && field.V != text {
				t.Errorf("%q written as %q came back as %q", text, written, field.V)
			}
		}
	}
}

// Quoting has to be decided by the content and nothing else, or two corpora holding the
// same thing would not emit the same bytes.
func TestQuotingIsDecidedByTheContentAlone(t *testing.T) {
	for _, value := range awkward {
		if first, second := quote(value), quote(value); first != second {
			t.Errorf("%q quoted as %s and then as %s", value, first, second)
		}
	}
}

// The point of quoting only where it is needed: the format stays readable. A corpus
// where every year is in quotes is a worse corpus.
func TestOrdinaryValuesAreLeftAlone(t *testing.T) {
	for _, value := range []string{
		"Giovanni Battista Rossi", "borgo-san-lorenzo", "1834", "12 MAR 1834",
		"ABT 1834", "1750/51", "BET 1830 AND 1835", "parish-register",
		"ADF/SL/B/4", "43.7696, 11.2558", "photos/1890-portrait.jpg",
		"parish-san-lorenzo-baptisms#1834-042", "person/rossi-annunziata",
		"events[0].date", "Archivio Diocesano di Firenze", "José",
	} {
		if got := quote(value); got != value {
			t.Errorf("%q did not need quoting but got %s", value, got)
		}
	}
}

// Single quotes are the readable way to carry a value that merely starts with the wrong
// character, and the reason the emitter has three tiers rather than one.
func TestQuotingReachesForDoubleQuotesOnlyWhenItMust(t *testing.T) {
	for value, want := range map[string]string{
		`"Nancy" OBrien`: `'"Nancy" OBrien'`,
		"O'Brien":        "O'Brien",
		"'quoted'":       "'''quoted'''",
		"- dashed":       "'- dashed'",
		"key: value":     "'key: value'",
		"null":           "'null'",
		"line\nbreak":    `"line\nbreak"`,
		" x":             `" x"`,
	} {
		if got := quote(value); got != want {
			t.Errorf("quote(%q) = %s, want %s", value, got, want)
		}
	}
}

// Nothing the emitter writes ends in a space, because a trailing space is invisible in
// a diff and survives a round trip only by accident.
func TestNothingIsWrittenWithTrailingSpace(t *testing.T) {
	c := fixture(t)
	for _, slug := range c.Slugs() {
		file, err := Emit(c.Entity(slug))
		if err != nil {
			t.Fatalf("%s: %v", slug, err)
		}
		for i, line := range strings.Split(string(file), "\n") {
			if strings.TrimRight(line, " \t") != line {
				t.Errorf("%s line %d ends in whitespace: %q", slug, i+1, line)
			}
		}
	}
}
