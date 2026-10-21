package gedcom

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// decode turns the bytes of a GEDCOM file into text.
//
// The encoding is declared inside the file it encodes, which is only workable because
// every encoding GEDCOM permits agrees with ASCII over the header — except the UTF-16
// ones, which announce themselves with a byte order mark. So the mark decides first and
// the declaration decides after.
//
// Nothing here guesses. An encoding origins cannot decode is refused, with the name the
// file gave, because an import that silently mangles every accented surname is worse
// than one that does not happen.
func decode(raw []byte) (string, error) {
	switch {
	case hasPrefix(raw, 0xEF, 0xBB, 0xBF):
		return asUTF8(raw[3:])
	case hasPrefix(raw, 0xFF, 0xFE):
		return asUTF16(raw[2:], false), nil
	case hasPrefix(raw, 0xFE, 0xFF):
		return asUTF16(raw[2:], true), nil
	}

	switch charset := declared(raw); charset {
	case "", "UTF-8", "UTF8", "ASCII", "ANSEL/UTF-8":
		return asUTF8(raw)
	case "ANSI", "WINDOWS-1252", "CP1252", "IBM WINDOWS":
		return viaCharmap(charmap.Windows1252, raw, charset)
	case "IBMPC", "IBM PC", "CP437":
		// IBMPC is what the DOS-era exporters call code page 437. Its European sibling
		// 850 goes by a different name and spells the accented letters at the same
		// places anyway, so the only bytes a mix-up could reach are the ones for drawing
		// boxes, which no name contains.
		return viaCharmap(charmap.CodePage437, raw, charset)
	case "ANSEL":
		return asANSEL(raw)
	case "UNICODE":
		return "", errors.New("this file declares UNICODE but carries no byte order mark, " +
			"so there is no way to tell which: re-export it as UTF-8")
	default:
		return "", fmt.Errorf("unknown character set %q: re-export the file as UTF-8", charset)
	}
}

func viaCharmap(m *charmap.Charmap, raw []byte, charset string) (string, error) {
	text, err := m.NewDecoder().Bytes(raw)
	if err != nil {
		return "", fmt.Errorf("this file declares %s and does not decode as it: %w", charset, err)
	}
	return string(text), nil
}

func hasPrefix(raw []byte, prefix ...byte) bool {
	if len(raw) < len(prefix) {
		return false
	}
	for i, b := range prefix {
		if raw[i] != b {
			return false
		}
	}
	return true
}

// declared reads the header's CHAR line without decoding anything, which is sound for
// every encoding that reaches this point: all of them agree with ASCII here.
func declared(raw []byte) string {
	head := raw
	if len(head) > 4096 {
		head = head[:4096]
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(head), "\r\n", "\n"), "\n") {
		if rest, found := strings.CutPrefix(strings.TrimSpace(line), "1 CHAR "); found {
			return strings.ToUpper(strings.TrimSpace(rest))
		}
	}
	return ""
}

func asUTF8(raw []byte) (string, error) {
	if !utf8.Valid(raw) {
		return "", errors.New("this file is not valid UTF-8 and does not say what it is: " +
			"re-export it as UTF-8")
	}
	return string(raw), nil
}

func asUTF16(raw []byte, bigEndian bool) string {
	units := make([]uint16, 0, len(raw)/2)
	for i := 0; i+1 < len(raw); i += 2 {
		lo, hi := raw[i], raw[i+1]
		if bigEndian {
			lo, hi = hi, lo
		}
		units = append(units, uint16(hi)<<8|uint16(lo))
	}
	return string(utf16.Decode(units))
}
