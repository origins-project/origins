package store

import (
	"fmt"
	"strings"
)

// Quoting is decided by the content alone, in three tiers: plain where YAML reads the
// text back unchanged, single-quoted where the only problem is a character that would
// start something, and double-quoted where a character has to be escaped. Nothing here
// consults a setting or a context, because the emitted bytes have to be a pure function
// of the model.

// indicators start a YAML construct rather than a value when they open a scalar.
const indicators = "-?:,[]{}#&*!|>'\"%@`"

// flowIndicators end a plain scalar inside [ ] as well. The question mark is here for
// the whole of a scalar and not only its first character, because the parser origins
// reads with treats one anywhere inside a flow sequence as the start of a key.
const flowIndicators = ",[]{}?"

func quote(s string) string { return quoted(s, false) }

// quoteFlow quotes for the inside of a flow sequence, where a comma or a bracket closes
// the scalar rather than belonging to it.
func quoteFlow(s string) string { return quoted(s, true) }

func quoted(s string, flow bool) string {
	if plain(s, flow) {
		return s
	}
	if singleQuotable(s) {
		return "'" + strings.ReplaceAll(s, "'", "''") + "'"
	}
	return doubleQuote(s)
}

func plain(s string, flow bool) bool {
	if s == "" {
		return false
	}
	if strings.TrimSpace(s) != s {
		return false
	}
	if strings.ContainsFunc(s, unprintable) {
		return false
	}
	if strings.ContainsAny(s[:1], indicators) {
		return false
	}
	if flow && strings.ContainsAny(s, flowIndicators) {
		return false
	}
	// ": " and " #" open a mapping value and a comment wherever they appear.
	if strings.Contains(s, ": ") || strings.HasSuffix(s, ":") || strings.Contains(s, " #") {
		return false
	}
	return !null(s)
}

// singleQuotable reports whether ' ' quoting carries the text as it stands. Single
// quotes escape nothing but themselves, so anything needing an escape — a line break, a
// control character, whitespace at either end that would be easy to lose — is left to
// the double-quoted form.
func singleQuotable(s string) bool {
	if strings.TrimSpace(s) != s {
		return false
	}
	return !strings.ContainsFunc(s, unprintable)
}

func doubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if unprintable(r) {
				fmt.Fprintf(&b, `\u%04X`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// unprintable reports whether a character has to be escaped to be written at all:
// whatever YAML will not read as written, and the characters it reads as a line break,
// which would end a scalar rather than belong to it.
func unprintable(r rune) bool {
	return r < 0x20 || r == 0x7f || 0x80 <= r && r <= 0x9f || r == 0x2028 || r == 0x2029 ||
		r == 0xfffe || r == 0xffff
}

// null reports whether YAML would read the text as nothing at all. It is the one
// resolution that loses the text it came from: yaml gives a string field the characters
// it found for a number, a boolean or a date, so a person named 1834 or true needs no
// defending, and a person named Null does.
func null(s string) bool {
	switch s {
	case "null", "Null", "NULL", "~":
		return true
	}
	return false
}
