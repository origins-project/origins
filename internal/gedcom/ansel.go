package gedcom

import (
	"fmt"
	"strings"
)

// asANSEL decodes ANSEL, ANSI/NISO Z39.47, which agrees with ASCII below 0x80 and
// spells the rest as spacing characters and as diacritics written before the character
// they go on. Unicode writes a combining mark after its base, so the marks are held
// until their base arrives and written after it, in the order the file gave them;
// composing them into precomposed letters is left to the NFC the parse applies to
// every encoding.
//
// A byte the table does not hold is refused, as is a diacritic with nothing after it
// to go on: a line end, a control character or the end of the file.
func asANSEL(raw []byte) (string, error) {
	var text strings.Builder
	text.Grow(len(raw))
	var marks []rune
	pending := -1
	for i, b := range raw {
		var r rune
		switch {
		case b < 0x80:
			r = rune(b)
		case anselSpacing[b] != 0:
			r = anselSpacing[b]
		case anselMarks[b] != 0:
			if pending < 0 {
				pending = i
			}
			marks = append(marks, anselMarks[b])
			continue
		default:
			return "", fmt.Errorf("this file declares ANSEL and byte %d is 0x%02X, "+
				"which has no agreed reading: re-export it as UTF-8", i, b)
		}
		if pending >= 0 {
			if r < 0x20 || r == 0x7F {
				return "", danglingMark(pending)
			}
			text.WriteRune(r)
			for _, m := range marks {
				text.WriteRune(m)
			}
			marks, pending = marks[:0], -1
			continue
		}
		text.WriteRune(r)
	}
	if pending >= 0 {
		return "", danglingMark(pending)
	}
	return text.String(), nil
}

func danglingMark(at int) error {
	return fmt.Errorf("this file declares ANSEL and the diacritic at byte %d has no "+
		"character after it to go on: re-export it as UTF-8", at)
}

// The table holds the bytes on which the Library of Congress's MARC-8 Extended Latin
// table and Gramps' ANSEL reader agree, and GEDCOM 5.5.1's Appendix C does not
// contradict; 0xCF, ß, is GEDCOM's own and Gramps', where MARC-8 puts ß at 0xC7 and
// leaves 0xCF unassigned. The ligature and double tilde halves, 0xEB and 0xEC, 0xFA and
// 0xFB, are left out because the sources map them differently, and so are the bytes
// only one of them assigns.
var anselSpacing = [256]rune{
	0xA1: 'Ł', 0xA2: 'Ø', 0xA3: 'Đ', 0xA4: 'Þ', 0xA5: 'Æ', 0xA6: 'Œ', 0xA7: 'ʹ',
	0xA8: '·', 0xA9: '♭', 0xAA: '®', 0xAB: '±', 0xAC: 'Ơ', 0xAD: 'Ư', 0xAE: 'ʼ',
	0xB0: 'ʻ', 0xB1: 'ł', 0xB2: 'ø', 0xB3: 'đ', 0xB4: 'þ', 0xB5: 'æ', 0xB6: 'œ',
	0xB7: 'ʺ', 0xB8: 'ı', 0xB9: '£', 0xBA: 'ð', 0xBC: 'ơ', 0xBD: 'ư',
	0xC0: '°', 0xC1: 'ℓ', 0xC2: '℗', 0xC3: '©', 0xC4: '♯', 0xC5: '¿', 0xC6: '¡',
	0xC7: 'ß', 0xC8: '€', 0xCF: 'ß',
}

var anselMarks = [256]rune{
	0xE0: '̉', // hook above
	0xE1: '̀', // grave
	0xE2: '́', // acute
	0xE3: '̂', // circumflex
	0xE4: '̃', // tilde
	0xE5: '̄', // macron
	0xE6: '̆', // breve
	0xE7: '̇', // dot above
	0xE8: '̈', // diaeresis
	0xE9: '̌', // caron
	0xEA: '̊', // ring above
	0xED: '̕', // comma above right
	0xEE: '̋', // double acute
	0xEF: '̐', // candrabindu
	0xF0: '̧', // cedilla
	0xF1: '̨', // ogonek
	0xF2: '̣', // dot below
	0xF3: '̤', // diaeresis below
	0xF4: '̥', // ring below
	0xF5: '̳', // double low line
	0xF6: '̲', // low line
	0xF7: '̦', // comma below
	0xF8: '̜', // left half ring below
	0xF9: '̮', // breve below
	0xFE: '̓', // comma above
}
