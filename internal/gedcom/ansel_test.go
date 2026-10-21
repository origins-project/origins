package gedcom

import (
	"strings"
	"testing"
)

// ANSEL agrees with ASCII below 0x80, so a file that declares it and carries no accent
// reads as it is.
func TestAnAnselFileWithinAsciiReads(t *testing.T) {
	text, err := decode([]byte(header("ANSEL", "John /Smith/")))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "John /Smith/") {
		t.Errorf("decoded to %q", text)
	}
}

// Every byte above 0x80, each read as the character it stands for or a diacritic put
// after the letter it goes on, and every byte this list does not hold refused.
func TestEveryAnselByteReadsOrIsRefused(t *testing.T) {
	spacing := map[byte]string{
		0xA1: "Ł", 0xA2: "Ø", 0xA3: "Đ", 0xA4: "Þ", 0xA5: "Æ", 0xA6: "Œ", 0xA7: "ʹ",
		0xA8: "·", 0xA9: "♭", 0xAA: "®", 0xAB: "±", 0xAC: "Ơ", 0xAD: "Ư", 0xAE: "ʼ",
		0xB0: "ʻ", 0xB1: "ł", 0xB2: "ø", 0xB3: "đ", 0xB4: "þ", 0xB5: "æ", 0xB6: "œ",
		0xB7: "ʺ", 0xB8: "ı", 0xB9: "£", 0xBA: "ð", 0xBC: "ơ", 0xBD: "ư",
		0xC0: "°", 0xC1: "ℓ", 0xC2: "℗", 0xC3: "©", 0xC4: "♯", 0xC5: "¿", 0xC6: "¡",
		0xC7: "ß", 0xC8: "€", 0xCF: "ß",
	}
	// Each on an a, composed where Unicode has a letter for it.
	marks := map[byte]string{
		0xE0: "ả", 0xE1: "à", 0xE2: "á", 0xE3: "â", 0xE4: "ã", 0xE5: "ā", 0xE6: "ă",
		0xE7: "ȧ", 0xE8: "ä", 0xE9: "ǎ", 0xEA: "å", 0xED: "a̕", 0xEE: "a̋",
		0xEF: "a̐", 0xF0: "a̧", 0xF1: "ą", 0xF2: "ạ", 0xF3: "a̤",
		0xF4: "ḁ", 0xF5: "a̳", 0xF6: "a̲", 0xF7: "a̦", 0xF8: "a̜",
		0xF9: "a̮", 0xFE: "a̓",
	}
	for b := 0x80; b <= 0xFF; b++ {
		raw := []byte(header("ANSEL", "x"+string([]byte{byte(b), 'a'})))
		nodes, err := Parse(strings.NewReader(string(raw)))
		want, isSpacing := spacing[byte(b)]
		if mark, isMark := marks[byte(b)]; isMark {
			want = mark
		} else if isSpacing {
			want += "a"
		} else {
			if err == nil {
				t.Errorf("0x%02X was read", b)
			}
			continue
		}
		if err != nil {
			t.Errorf("0x%02X: %v", b, err)
			continue
		}
		if got := nodes[1].First("NAME").Text(); got != "x"+want {
			t.Errorf("0x%02X read as %q, want %q", b, got, "x"+want)
		}
	}
}

func TestAnselDiacriticsGoAfterTheirBase(t *testing.T) {
	for _, c := range []struct {
		what string
		raw  string
		want string
	}{
		{what: "a mark before a letter", raw: "Jos\xE2e", want: "José"},
		{what: "two marks stacked, kept in order", raw: "\xE5\xE2a", want: "ā́"},
		{what: "a mark before a space", raw: "\xE8 x", want: " ̈x"},
		{what: "a mark before a digit", raw: "\xE5" + "1", want: "1̄"},
		{what: "a mark before a spacing ANSEL letter", raw: "\xE2\xB8", want: "ı́"},
		{what: "marks on consecutive letters", raw: "\xE9c\xE9s", want: "čš"},
	} {
		got, err := asANSEL([]byte(c.raw))
		if err != nil {
			t.Errorf("%s: %v", c.what, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: read as %q, want %q", c.what, got, c.want)
		}
	}
}

func TestAnAnselDiacriticWithNothingToGoOnIsRefused(t *testing.T) {
	for _, c := range []struct {
		what string
		raw  string
	}{
		{what: "a mark at the end of the file", raw: "Jos\xE2"},
		{what: "a mark before a line end", raw: "Jos\xE2\nx"},
		{what: "a mark before a carriage return", raw: "Jos\xE2\r\nx"},
		{what: "two marks at the end of the file", raw: "Jos\xE2\xE8"},
		{what: "a mark before a byte no table agrees on", raw: "Jos\xE2\xEBe"},
	} {
		if _, err := asANSEL([]byte(c.raw)); err == nil {
			t.Errorf("%s was accepted", c.what)
		}
	}
}

// A whole file, the names and places spelled out byte by byte from the ANSEL table and
// the expected text typed as it is written.
func TestAnAnselFileReadsWhole(t *testing.T) {
	raw := "0 HEAD\r\n1 CHAR ANSEL\r\n" +
		"0 @I1@ INDI\r\n1 NAME Anton\xE2in /Dvo\xE9r\xE2ak/\r\n" +
		"1 BIRT\r\n2 PLAC Nelahozeves, \xE9Cechy\r\n" +
		"0 @I2@ INDI\r\n1 NAME \xA1ucja /Wa\xB1\xF1esa-\xE8Ostr\xE8o\xE7zka/\r\n" +
		"1 RESI\r\n2 PLAC \xA1\xE2od\xE2z\r\n" +
		"0 @I3@ INDI\r\n1 NAME Fran\xF0cois /\xA5lfric \xB2deg\xE3a/\r\n" +
		"1 NOTE \xC3 1993, \xC5qu\xE2e? \xC6s\xE2i! 25\xC0 \xB9 5\r\n" +
		"0 TRLR\r\n"
	nodes, err := Parse(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		node *Node
		want string
	}{
		{nodes[1].First("NAME"), "Antonín /Dvořák/"},
		{nodes[1].First("BIRT").First("PLAC"), "Nelahozeves, Čechy"},
		{nodes[2].First("NAME"), "Łucja /Wałęsa-Öströżka/"},
		{nodes[2].First("RESI").First("PLAC"), "Łódź"},
		{nodes[3].First("NAME"), "François /Ælfric ødegâ/"},
		{nodes[3].First("NOTE"), "© 1993, ¿qué? ¡sí! 25° £ 5"},
	} {
		if got := c.node.Text(); got != c.want {
			t.Errorf("read %q, want %q", got, c.want)
		}
	}
}
